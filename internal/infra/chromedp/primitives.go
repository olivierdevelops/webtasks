package chromedp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	cdpcore "github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	cdp "github.com/chromedp/chromedp"
)

// Primitives is a thin wrapper around chromedp actions. Each method takes the
// per-window chromedp.Context and executes one (or a sequence) of actions.
type Primitives struct{}

func (Primitives) Navigate(ctx context.Context, url string) error {
	return cdp.Run(ctx, cdp.Navigate(url))
}

// Click clicks the first visible match of `selector`. CDP's
// Input.dispatchMouseEvent fires isTrusted=true so Vue / framework handlers
// see a real user click (unlike el.click() from JS, which they may ignore).
func (Primitives) Click(ctx context.Context, selector string) error {
	return cdp.Run(ctx, cdp.Click(selector, cdp.NodeVisible))
}

// ClickText finds the first `selector` element whose trimmed textContent
// matches `text` (exact, or substring when mode=="contains"). When `closest`
// is set it retargets to that element's nearest matching ancestor. The chosen
// element is stamped with a temporary attribute and clicked natively
// (isTrusted). This is the YAML-native alternative to a "find element by
// visible name" JS module.
func (Primitives) ClickText(ctx context.Context, selector, text, mode, closest string) error {
	contains := mode == "contains"
	findJS := "(function(){" +
		"var els = document.querySelectorAll(" + quoteJSString(selector) + ");" +
		"var want = " + quoteJSString(text) + ";" +
		"var contains = " + boolJS(contains) + ";" +
		"var closestSel = " + quoteJSString(closest) + ";" +
		"for (var i = 0; i < els.length; i++) {" +
		"  var t = (els[i].textContent || '').trim();" +
		"  if (contains ? t.indexOf(want) >= 0 : t === want) {" +
		"    var target = els[i];" +
		"    if (closestSel) { var c = target.closest(closestSel); if (c) target = c; }" +
		"    document.querySelectorAll('[data-webtasks-click]').forEach(function(e){" +
		"      e.removeAttribute('data-webtasks-click'); });" +
		"    target.setAttribute('data-webtasks-click', '1');" +
		"    target.scrollIntoView({ block: 'center', inline: 'center' });" +
		"    return true;" +
		"  }" +
		"}" +
		"return false;" +
		"})()"
	var found bool
	if err := cdp.Run(ctx, cdp.Evaluate(findJS, &found)); err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("click: no %q element with text %q", selector, text)
	}
	return cdp.Run(ctx, cdp.Click("[data-webtasks-click]", cdp.NodeVisible))
}

func boolJS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// SendKeys focuses an element and types text. Uses keyboard.dispatchKeyEvent.
func (Primitives) SendKeys(ctx context.Context, selector, text string) error {
	return cdp.Run(ctx,
		cdp.Focus(selector, cdp.NodeVisible),
		cdp.SendKeys(selector, text, cdp.NodeVisible),
	)
}

// WaitFor blocks until the selector is in the DOM (`NodeReady`) or timeout.
func (Primitives) WaitFor(ctx context.Context, selector string, timeout time.Duration) error {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return cdp.Run(tctx, cdp.WaitReady(selector, cdp.ByQuery))
}

// Screenshot returns image bytes. `fullPage` captures the whole scrollable
// page (chromedp.FullScreenshot); otherwise the viewport, or a selector if
// one is given. `quality` < 100 with fullPage yields JPEG, else PNG.
func (Primitives) Screenshot(ctx context.Context, selector string, fullPage bool, format string, quality int) ([]byte, error) {
	var buf []byte
	if fullPage {
		q := quality
		if q <= 0 {
			q = 100
		}
		if err := cdp.Run(ctx, cdp.FullScreenshot(&buf, q)); err != nil {
			return nil, err
		}
		return buf, nil
	}
	if selector == "" || selector == "." {
		if err := cdp.Run(ctx, cdp.CaptureScreenshot(&buf)); err != nil {
			return nil, err
		}
		return buf, nil
	}
	if err := cdp.Run(ctx, cdp.Screenshot(selector, &buf, cdp.NodeVisible, cdp.ByQuery)); err != nil {
		return nil, err
	}
	return buf, nil
}

// ExecuteJS evaluates `script` in the page context. The script must return a
// JSON-serialisable value or `undefined`. Extra args are passed via the
// `arguments` array, mirroring the Java engine's contract. When `await` is
// true the script runs inside an async IIFE and a returned Promise is awaited
// (WithAwaitPromise) — needed for self-contained async routines.
func (Primitives) ExecuteJS(ctx context.Context, script string, await bool, args ...any) (any, error) {
	var result any
	var action cdp.Action
	if await {
		wrapped := "(async function(){\n" + injectArgs(args) + script + "\n})()"
		action = cdp.Evaluate(wrapped, &result, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		})
	} else {
		wrapped := "(function(){\n" + injectArgs(args) + script + "\n})()"
		action = cdp.Evaluate(wrapped, &result)
	}
	if err := cdp.Run(ctx, action); err != nil {
		return nil, err
	}
	return result, nil
}

func injectArgs(args []any) string {
	if len(args) == 0 {
		return "var arguments = [];\n"
	}
	var sb strings.Builder
	sb.WriteString("var arguments = ")
	encodeArray(&sb, args)
	sb.WriteString(";\n")
	return sb.String()
}

func encodeArray(sb *strings.Builder, vs []any) {
	sb.WriteByte('[')
	for i, v := range vs {
		if i > 0 {
			sb.WriteByte(',')
		}
		encodeValue(sb, v)
	}
	sb.WriteByte(']')
}

func encodeValue(sb *strings.Builder, v any) {
	switch t := v.(type) {
	case string:
		sb.WriteString(quoteJSString(t))
	case nil:
		sb.WriteString("null")
	default:
		fmt.Fprintf(sb, "%v", t)
	}
}

func quoteJSString(s string) string {
	out := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	).Replace(s)
	return `"` + out + `"`
}

// SaveHTML writes the outerHTML of the matched element (or whole document) to
// `path`, creating parent directories as needed.
func (p Primitives) SaveHTML(ctx context.Context, selector, path string) error {
	html, err := p.GetOuterHTML(ctx, selector)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(html), fs.FileMode(0o644))
}

func (Primitives) GetOuterHTML(ctx context.Context, selector string) (string, error) {
	if selector == "" || selector == "." {
		var html string
		if err := cdp.Run(ctx, cdp.Evaluate("document.documentElement.outerHTML", &html)); err != nil {
			return "", err
		}
		return html, nil
	}
	var html string
	if err := cdp.Run(ctx, cdp.OuterHTML(selector, &html, cdp.ByQuery)); err != nil {
		return "", err
	}
	return html, nil
}

// ScrollUntilStable repeatedly sets `el.scrollTop` to either 0 (up) or
// scrollHeight (down) until scrollHeight stops growing for `stableMs`.
func (Primitives) ScrollUntilStable(ctx context.Context, selector, direction string, stableMs int64, maxIterations int) error {
	target := "0"
	if direction != "up" {
		target = "el.scrollHeight"
	}
	js := `(function() {
		var el = document.querySelector(` + quoteJSString(selector) + `);
		if (!el) return -1;
		el.scrollTop = ` + target + `;
		return el.scrollHeight;
	})()`
	var lastHeight float64 = -1
	lastChange := time.Now()
	stable := time.Duration(stableMs) * time.Millisecond
	for i := 0; ; i++ {
		var h float64
		if err := cdp.Run(ctx, cdp.Evaluate(js, &h)); err != nil {
			return err
		}
		if h < 0 {
			return nil
		}
		if h != lastHeight {
			lastHeight = h
			lastChange = time.Now()
		} else if time.Since(lastChange) >= stable {
			return nil
		}
		if maxIterations > 0 && i+1 >= maxIterations {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// DownloadEach native-clicks each match of `selector` in DOM order, polling
// `downloadDir` after each click for a new file. Returns the paths of newly
// arrived files in click order — an empty string slot means the click went
// through but no download landed within `perFile`.
//
// Before each click we scroll the target into view and hide overlays
// (e.g. Concio's `.to-bottom-chat`) so the click isn't intercepted.
func (Primitives) DownloadEach(ctx context.Context, selector string, downloadDir string, perFile time.Duration) ([]string, error) {
	var nodes []*cdpcore.Node
	if err := cdp.Run(ctx, cdp.Nodes(selector, &nodes, cdp.ByQueryAll)); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return nil, err
	}
	seen := snapshotFiles(downloadDir)
	hideOverlays := `document.querySelectorAll('.to-bottom-chat, .modalRoot.adInvite, [class*="unread-message-tip"]')` +
		`.forEach(o => { o.style.display = 'none'; o.style.pointerEvents = 'none'; });`
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		// Scroll the specific node into view (chromedp.ScrollIntoView works
		// on a single *dom.Node so we don't need a per-node selector) and
		// hide overlays before clicking. MouseClickNode produces an
		// isTrusted=true click at the node's centre — the right semantics
		// for Vue/framework handlers that ignore JS-dispatched events.
		var ignore any
		_ = cdp.Run(ctx, cdp.Evaluate(hideOverlays, &ignore))
		_ = cdp.Run(ctx,
			dom.ScrollIntoViewIfNeeded().WithBackendNodeID(n.BackendNodeID),
			cdp.MouseClickNode(n),
		)
		newFile := pollForNewDownload(downloadDir, seen, perFile)
		if newFile != "" {
			seen[filepath.Base(newFile)] = struct{}{}
		}
		out = append(out, newFile)
		time.Sleep(400 * time.Millisecond)
	}
	return out, nil
}

func snapshotFiles(dir string) map[string]struct{} {
	out := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		out[e.Name()] = struct{}{}
	}
	return out
}

func pollForNewDownload(dir string, seen map[string]struct{}, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				if _, ok := seen[name]; ok {
					continue
				}
				if strings.HasSuffix(name, ".crdownload") || strings.HasSuffix(name, ".tmp") {
					continue
				}
				return filepath.Join(dir, name)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return ""
}

// drainCapturesScript pulls finished entries (bytesB64 != null OR error set)
// out of `window.__webtasks_captures` and leaves pending ones behind.
const drainCapturesScript = `(function() {
	var keep = [], drained = [];
	for (var i = 0; i < (window.__webtasks_captures || []).length; i++) {
		var e = window.__webtasks_captures[i];
		if (e.bytesB64 != null || e.error) drained.push(e); else keep.push(e);
	}
	window.__webtasks_captures = keep;
	return drained;
})()`

// SaveCapturesToDir drains the in-page captures buffer and writes each blob
// to disk using the supplied naming template. The template supports the
// tokens `{id}`, `{name}`, `{mime}`, `{ext}`, `{size}`, `{ts}`. When the
// blob doesn't carry a `name` we fall back to `<id>.<ext>` for the basename.
func (p Primitives) SaveCapturesToDir(ctx context.Context, dir, naming string) ([]CapturedFile, error) {
	if naming == "" {
		naming = "{id}_{name}"
	}
	var raw json.RawMessage
	if err := cdp.Run(ctx, cdp.Evaluate(drainCapturesScript, &raw)); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	out := make([]CapturedFile, 0, len(entries))
	for _, e := range entries {
		b64, _ := e["bytesB64"].(string)
		if b64 == "" {
			continue
		}
		bytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			continue
		}
		base := renderBasename(naming, e)
		target := filepath.Join(dir, base)
		if err := os.WriteFile(target, bytes, fs.FileMode(0o644)); err != nil {
			return out, err
		}
		out = append(out, CapturedFile{
			Path:     target,
			Basename: base,
			Size:     len(bytes),
			Mime:     toString(e["mime"]),
			Name:     toString(e["name"]),
			URL:      toString(e["url"]),
		})
	}
	return out, nil
}

// CapturedFile is what SaveCapturesToDir returns. Mirrors features.CapturedFile
// so the orchestrator wiring can adapt between them.
type CapturedFile struct {
	Path     string
	Basename string
	Size     int
	Mime     string
	Name     string
	URL      string
}

var sanitiseFilename = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

func renderBasename(pattern string, capture map[string]any) string {
	name := toString(capture["name"])
	m := toString(capture["mime"])
	ext := inferExt(m, name)
	id := toString(capture["id"])
	ts := toString(capture["ts"])
	size := toString(capture["size"])
	safeName := name
	if safeName == "" {
		safeName = id + "." + ext
	}
	out := pattern
	for k, v := range map[string]string{
		"{id}": id, "{name}": safeName, "{mime}": m, "{ext}": ext, "{size}": size, "{ts}": ts,
	} {
		out = strings.ReplaceAll(out, k, v)
	}
	return strings.TrimSpace(sanitiseFilename.ReplaceAllString(out, "_"))
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func inferExt(mimeType, name string) string {
	if name != "" {
		if i := strings.LastIndex(name, "."); i >= 0 && i < len(name)-1 {
			return name[i+1:]
		}
	}
	if mimeType == "" {
		return "bin"
	}
	if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
		return strings.TrimPrefix(exts[0], ".")
	}
	return "bin"
}

// ---- Batch A: rendering & capture ----

// PDFOpts mirrors features.PDFOptions; the orchestrator adapts between them
// (infra keeps its own shapes — see CapturedFile for the same pattern).
type PDFOpts struct {
	Landscape, PrintBackground, DisplayHeaderFooter  bool
	Scale, PaperWidth, PaperHeight                   float64
	MarginTop, MarginBottom, MarginLeft, MarginRight float64
	PageRanges, HeaderTemplate, FooterTemplate       string
}

// EmulateOpts mirrors features.EmulateOptions.
type EmulateOpts struct {
	Width, Height     int64
	DeviceScaleFactor float64
	Mobile            bool
	ColorScheme       string
	Reset             bool
}

// ScreencastOpts mirrors features.ScreencastOptions (capture knobs only).
type ScreencastOpts struct {
	FPS           int
	Quality       int
	EveryNthFrame int
	MaxFrames     int
	MaxDurationMs int64
}

// NetCaptureOpts mirrors features.NetCaptureOptions.
type NetCaptureOpts struct {
	IncludeBodies bool
	MaxBodyBytes  int
	URLFilter     string
}

// Cookie mirrors features.Cookie.
type Cookie struct {
	Name, Value, Domain, Path, SameSite, URL string
	Expires                                  float64
	HTTPOnly, Secure                         bool
}

// PrintToPDF renders the current page to PDF via Page.printToPDF.
func (Primitives) PrintToPDF(ctx context.Context, o PDFOpts) ([]byte, error) {
	var buf []byte
	err := cdp.Run(ctx, cdp.ActionFunc(func(ctx context.Context) error {
		p := page.PrintToPDF().
			WithLandscape(o.Landscape).
			WithPrintBackground(o.PrintBackground)
		if o.Scale > 0 {
			p = p.WithScale(o.Scale)
		}
		if o.PaperWidth > 0 {
			p = p.WithPaperWidth(o.PaperWidth)
		}
		if o.PaperHeight > 0 {
			p = p.WithPaperHeight(o.PaperHeight)
		}
		if o.MarginTop > 0 {
			p = p.WithMarginTop(o.MarginTop)
		}
		if o.MarginBottom > 0 {
			p = p.WithMarginBottom(o.MarginBottom)
		}
		if o.MarginLeft > 0 {
			p = p.WithMarginLeft(o.MarginLeft)
		}
		if o.MarginRight > 0 {
			p = p.WithMarginRight(o.MarginRight)
		}
		if o.PageRanges != "" {
			p = p.WithPageRanges(o.PageRanges)
		}
		if o.DisplayHeaderFooter {
			p = p.WithDisplayHeaderFooter(true).
				WithHeaderTemplate(o.HeaderTemplate).
				WithFooterTemplate(o.FooterTemplate)
		}
		data, _, err := p.Do(ctx)
		if err != nil {
			return err
		}
		buf = data
		return nil
	}))
	return buf, err
}

// RenderHTMLToPDF loads an HTML document (as a data: URL) then prints it.
func (p Primitives) RenderHTMLToPDF(ctx context.Context, html string, o PDFOpts) ([]byte, error) {
	url := "data:text/html;charset=utf-8;base64," + base64.StdEncoding.EncodeToString([]byte(html))
	if err := cdp.Run(ctx, cdp.Navigate(url)); err != nil {
		return nil, err
	}
	return p.PrintToPDF(ctx, o)
}

// CaptureSnapshot returns an MHTML single-file archive of the page.
func (Primitives) CaptureSnapshot(ctx context.Context) ([]byte, error) {
	var data string
	err := cdp.Run(ctx, cdp.ActionFunc(func(ctx context.Context) error {
		d, e := page.CaptureSnapshot().Do(ctx)
		data = d
		return e
	}))
	return []byte(data), err
}

// Emulate applies device-metrics and emulated-media overrides.
func (Primitives) Emulate(ctx context.Context, o EmulateOpts) error {
	if o.Reset {
		_ = cdp.Run(ctx, emulation.ClearDeviceMetricsOverride())
		return cdp.Run(ctx, emulation.SetEmulatedMedia())
	}
	var actions []cdp.Action
	if o.Width > 0 && o.Height > 0 {
		dsf := o.DeviceScaleFactor
		if dsf <= 0 {
			dsf = 1
		}
		actions = append(actions, emulation.SetDeviceMetricsOverride(o.Width, o.Height, dsf, o.Mobile))
	}
	if o.ColorScheme != "" {
		actions = append(actions, emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{
			{Name: "prefers-color-scheme", Value: o.ColorScheme},
		}))
	}
	if len(actions) == 0 {
		return nil
	}
	return cdp.Run(ctx, actions...)
}

// ---- Batch B: screen recording ----

// Screencast captures the page as a sequence of PNG frames while `body` (the
// wrapped steps) runs, then returns the frames + their capture times. It
// works by polling CaptureScreenshot from a background goroutine at `FPS` —
// far more reliable in headless Chrome than the CDP screencast event stream.
func (Primitives) Screencast(ctx context.Context, o ScreencastOpts, body func() error) ([][]byte, []time.Time, error) {
	maxFrames := o.MaxFrames
	if maxFrames <= 0 {
		maxFrames = 300
	}
	fps := o.FPS
	if fps <= 0 {
		fps = 5
	}
	interval := time.Second / time.Duration(fps)
	maxDur := time.Duration(o.MaxDurationMs) * time.Millisecond

	var (
		mu     sync.Mutex
		frames [][]byte
		times  []time.Time
	)
	stop := make(chan struct{})
	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if maxDur > 0 && time.Since(start) > maxDur {
					return
				}
				mu.Lock()
				full := len(frames) >= maxFrames
				mu.Unlock()
				if full {
					return
				}
				var buf []byte
				if err := cdp.Run(ctx, cdp.CaptureScreenshot(&buf)); err != nil || len(buf) == 0 {
					continue
				}
				mu.Lock()
				frames = append(frames, buf)
				times = append(times, time.Now())
				mu.Unlock()
			}
		}
	}()
	bodyErr := body()
	close(stop)
	<-done

	mu.Lock()
	defer mu.Unlock()
	fc := make([][]byte, len(frames))
	copy(fc, frames)
	tc := make([]time.Time, len(times))
	copy(tc, times)
	return fc, tc, bodyErr
}

// ---- Batch C: network & session ----

// CaptureNetwork records request/response events while `body` runs and
// returns HAR-ish entries (one map per request, in arrival order).
func (Primitives) CaptureNetwork(ctx context.Context, o NetCaptureOpts, body func() error) ([]map[string]any, error) {
	capCtx, capCancel := context.WithCancel(ctx)
	defer capCancel()

	var mu sync.Mutex
	var wg sync.WaitGroup // tracks async response-body fetches
	entries := map[network.RequestID]map[string]any{}
	var order []network.RequestID

	matches := func(url string) bool {
		return o.URLFilter == "" || strings.Contains(url, o.URLFilter)
	}

	cdp.ListenTarget(capCtx, func(ev any) {
		mu.Lock()
		defer mu.Unlock()
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			if e.Request == nil || !matches(e.Request.URL) {
				return
			}
			entries[e.RequestID] = map[string]any{
				"request": map[string]any{
					"method":  e.Request.Method,
					"url":     e.Request.URL,
					"headers": e.Request.Headers,
				},
			}
			order = append(order, e.RequestID)
		case *network.EventResponseReceived:
			ent, ok := entries[e.RequestID]
			if !ok || e.Response == nil {
				return
			}
			ent["response"] = map[string]any{
				"status":   e.Response.Status,
				"mimeType": e.Response.MimeType,
				"headers":  e.Response.Headers,
			}
		case *network.EventLoadingFinished:
			ent, ok := entries[e.RequestID]
			if !ok {
				return
			}
			ent["encodedDataLength"] = e.EncodedDataLength
			if o.IncludeBodies {
				rid := e.RequestID
				wg.Add(1)
				go func() {
					defer wg.Done()
					var b []byte
					_ = cdp.Run(ctx, cdp.ActionFunc(func(ctx context.Context) error {
						body, err := network.GetResponseBody(rid).Do(ctx)
						b = body
						return err
					}))
					if len(b) > 0 {
						if o.MaxBodyBytes > 0 && len(b) > o.MaxBodyBytes {
							b = b[:o.MaxBodyBytes]
						}
						mu.Lock()
						ent["body"] = string(b)
						mu.Unlock()
					}
				}()
			}
		}
	})

	if err := cdp.Run(capCtx, network.Enable()); err != nil {
		return nil, err
	}
	bodyErr := body()
	_ = cdp.Run(ctx, network.Disable())
	capCancel()  // stop the listener before draining, so no late event mutates entries
	wg.Wait()    // let in-flight response-body fetches finish writing

	mu.Lock()
	defer mu.Unlock()
	out := make([]map[string]any, 0, len(order))
	for _, id := range order {
		out = append(out, entries[id])
	}
	return out, bodyErr
}

// CaptureConsole collects console.* calls while `body` runs.
func (Primitives) CaptureConsole(ctx context.Context, body func() error) ([]map[string]any, error) {
	capCtx, capCancel := context.WithCancel(ctx)
	defer capCancel()

	var mu sync.Mutex
	var logs []map[string]any
	cdp.ListenTarget(capCtx, func(ev any) {
		e, ok := ev.(*runtime.EventConsoleAPICalled)
		if !ok {
			return
		}
		mu.Lock()
		logs = append(logs, map[string]any{
			"level": string(e.Type),
			"text":  consoleArgsText(e.Args),
		})
		mu.Unlock()
	})
	if err := cdp.Run(capCtx, runtime.Enable()); err != nil {
		return nil, err
	}
	bodyErr := body()

	mu.Lock()
	defer mu.Unlock()
	out := make([]map[string]any, len(logs))
	copy(out, logs)
	return out, bodyErr
}

func consoleArgsText(args []*runtime.RemoteObject) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if a == nil {
			continue
		}
		if len(a.Value) > 0 {
			parts = append(parts, strings.Trim(string(a.Value), `"`))
		} else if a.Description != "" {
			parts = append(parts, a.Description)
		} else {
			parts = append(parts, string(a.Type))
		}
	}
	return strings.Join(parts, " ")
}

// WaitForNetworkIdle blocks until in-flight requests stay <= maxInflight for
// idleMs, or timeoutMs elapses.
func (Primitives) WaitForNetworkIdle(ctx context.Context, idleMs, timeoutMs int64, maxInflight int) error {
	capCtx, capCancel := context.WithCancel(ctx)
	defer capCancel()

	var mu sync.Mutex
	inflight := 0
	lastChange := time.Now()
	cdp.ListenTarget(capCtx, func(ev any) {
		mu.Lock()
		defer mu.Unlock()
		switch ev.(type) {
		case *network.EventRequestWillBeSent:
			inflight++
			lastChange = time.Now()
		case *network.EventLoadingFinished:
			inflight--
			lastChange = time.Now()
		case *network.EventLoadingFailed:
			inflight--
			lastChange = time.Now()
		}
	})
	if err := cdp.Run(capCtx, network.Enable()); err != nil {
		return err
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	idle := time.Duration(idleMs) * time.Millisecond
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("wait-for-network-idle: timed out after %dms", timeoutMs)
		}
		mu.Lock()
		quiet := inflight <= maxInflight && time.Since(lastChange) >= idle
		mu.Unlock()
		if quiet {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// GetCookies reads cookies (optionally scoped to `urls`).
func (Primitives) GetCookies(ctx context.Context, urls []string) ([]Cookie, error) {
	var out []Cookie
	err := cdp.Run(ctx, cdp.ActionFunc(func(ctx context.Context) error {
		p := network.GetCookies()
		if len(urls) > 0 {
			p.Urls = urls
		}
		cks, err := p.Do(ctx)
		if err != nil {
			return err
		}
		for _, c := range cks {
			out = append(out, Cookie{
				Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path,
				Expires: c.Expires, HTTPOnly: c.HTTPOnly, Secure: c.Secure,
				SameSite: string(c.SameSite),
			})
		}
		return nil
	}))
	return out, err
}

// SetCookies installs cookies into the browser session.
func (Primitives) SetCookies(ctx context.Context, cookies []Cookie) error {
	params := make([]*network.CookieParam, 0, len(cookies))
	for _, c := range cookies {
		cp := &network.CookieParam{
			Name: c.Name, Value: c.Value, URL: c.URL, Domain: c.Domain,
			Path: c.Path, Secure: c.Secure, HTTPOnly: c.HTTPOnly,
		}
		if c.SameSite != "" {
			cp.SameSite = network.CookieSameSite(c.SameSite)
		}
		if c.Expires > 0 {
			t := cdpcore.TimeSinceEpoch(time.Unix(int64(c.Expires), 0))
			cp.Expires = &t
		}
		params = append(params, cp)
	}
	return cdp.Run(ctx, network.SetCookies(params))
}
