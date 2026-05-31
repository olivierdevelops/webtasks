// Package features (under orchestrator) builds wired-up feature values by
// closing over infra adapters. Each Make* function returns a fully-wired
// features.* record.
package features

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"webtasks/internal/domain"
	"webtasks/internal/features"
	"webtasks/internal/infra/bundle"
	chromedpx "webtasks/internal/infra/chromedp"
	"webtasks/internal/infra/ffmpeg"
	"webtasks/internal/infra/gifenc"
	"webtasks/internal/infra/goqueryx"
	"webtasks/internal/infra/httpclient"
	"webtasks/internal/infra/yamlreader"
)

// MakeBrowserActions wires chromedp primitives onto BrowserActions. Each
// capability resolves the per-window chromedp context and bounds it by the
// run context's deadline, so a task's `timeoutMs` actually interrupts a
// stuck browser call instead of hanging the window lease forever.
func MakeBrowserActions(p chromedpx.Primitives, src *chromedpx.WindowSource) features.BrowserActions {
	// withCtx returns a chromedp context (carrying the window's target) that
	// also inherits the run context's deadline. Callers must defer cancel().
	withCtx := func(run context.Context, w domain.WindowID) (context.Context, context.CancelFunc, error) {
		base, err := src.Context(string(w))
		if err != nil {
			return nil, nil, err
		}
		if dl, ok := run.Deadline(); ok {
			c, cancel := context.WithDeadline(base, dl)
			return c, cancel, nil
		}
		return base, func() {}, nil
	}
	return features.BrowserActions{
		Goto: func(run context.Context, w domain.WindowID, url string) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.Navigate(ctx, url)
		},
		Click: func(run context.Context, w domain.WindowID, selector string) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.Click(ctx, selector)
		},
		ClickText: func(run context.Context, w domain.WindowID, selector, text, mode, closest string) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.ClickText(ctx, selector, text, mode, closest)
		},
		SendKeys: func(run context.Context, w domain.WindowID, selector, text string) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.SendKeys(ctx, selector, text)
		},
		WaitFor: func(run context.Context, w domain.WindowID, selector string, timeout time.Duration) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.WaitFor(ctx, selector, timeout)
		},
		ScrollUntilStable: func(run context.Context, w domain.WindowID, selector, direction string, stableMs int64, maxIterations int) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.ScrollUntilStable(ctx, selector, direction, stableMs, maxIterations)
		},
		Screenshot: func(run context.Context, w domain.WindowID, opts features.ScreenshotOptions) ([]byte, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			return p.Screenshot(ctx, opts.Selector, opts.FullPage, opts.Format, opts.Quality)
		},
		ExecuteJS: func(run context.Context, w domain.WindowID, script string, await bool, args ...any) (any, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			return p.ExecuteJS(ctx, script, await, args...)
		},
		SaveHTML: func(run context.Context, w domain.WindowID, selector, file string) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.SaveHTML(ctx, selector, file)
		},
		GetOuterHTML: func(run context.Context, w domain.WindowID, selector string) (string, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return "", err
			}
			defer cancel()
			return p.GetOuterHTML(ctx, selector)
		},
		Sleep: time.Sleep,
		DownloadEach: func(run context.Context, w domain.WindowID, selector string, perFile time.Duration) ([]string, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			dir, err := src.DownloadDir(string(w))
			if err != nil {
				return nil, err
			}
			return p.DownloadEach(ctx, selector, dir, perFile)
		},
		SaveCapturesToDir: func(run context.Context, w domain.WindowID, dir, naming string) ([]features.CapturedFile, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			caps, err := p.SaveCapturesToDir(ctx, dir, naming)
			if err != nil {
				return nil, err
			}
			out := make([]features.CapturedFile, 0, len(caps))
			for _, c := range caps {
				out = append(out, features.CapturedFile{
					Path: c.Path, Basename: c.Basename, Size: c.Size,
					Mime: c.Mime, Name: c.Name, URL: c.URL,
				})
			}
			return out, nil
		},

		PrintToPDF: func(run context.Context, w domain.WindowID, opts features.PDFOptions) ([]byte, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			return p.PrintToPDF(ctx, toPDFOpts(opts))
		},
		RenderHTMLToPDF: func(run context.Context, w domain.WindowID, html string, opts features.PDFOptions) ([]byte, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			return p.RenderHTMLToPDF(ctx, html, toPDFOpts(opts))
		},
		CaptureSnapshot: func(run context.Context, w domain.WindowID) ([]byte, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			return p.CaptureSnapshot(ctx)
		},
		Emulate: func(run context.Context, w domain.WindowID, opts features.EmulateOptions) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.Emulate(ctx, chromedpx.EmulateOpts{
				Width: opts.Width, Height: opts.Height,
				DeviceScaleFactor: opts.DeviceScaleFactor, Mobile: opts.Mobile,
				ColorScheme: opts.ColorScheme, Reset: opts.Reset,
			})
		},
		Record: func(run context.Context, w domain.WindowID, opts features.ScreencastOptions, body func() error) (features.RecordResult, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return features.RecordResult{}, err
			}
			defer cancel()
			// bodyErr is the error from the wrapped steps — it is NOT a
			// recording failure. Frames are still encoded and returned so a
			// failing run can be inspected; bodyErr is passed back as the
			// second result for the caller to surface.
			frames, times, bodyErr := p.Screencast(ctx, chromedpx.ScreencastOpts{
				FPS: opts.FPS, Quality: opts.Quality, EveryNthFrame: opts.EveryNthFrame,
				MaxFrames: opts.MaxFrames, MaxDurationMs: opts.MaxDurationMs,
			}, body)
			if len(frames) == 0 {
				if bodyErr != nil {
					return features.RecordResult{}, bodyErr
				}
				return features.RecordResult{}, fmt.Errorf("record: no frames captured")
			}
			fps := opts.FPS
			if fps <= 0 {
				fps = 5
			}
			var durMs int64
			if len(times) > 1 {
				durMs = times[len(times)-1].Sub(times[0]).Milliseconds()
			}
			var data []byte
			var encErr error
			if opts.Format == "mp4" {
				if !ffmpeg.Available() {
					return features.RecordResult{}, fmt.Errorf(
						"mp4 output requires ffmpeg on PATH; use format: gif or install ffmpeg")
				}
				data, encErr = ffmpeg.EncodeMP4(ctx, frames, fps)
			} else {
				data, encErr = gifenc.Encode(frames, frameDelays(times), 100/fps)
			}
			if encErr != nil {
				return features.RecordResult{}, encErr
			}
			return features.RecordResult{Data: data, Frames: len(frames), DurationMs: durMs}, bodyErr
		},
		CaptureNetwork: func(run context.Context, w domain.WindowID, opts features.NetCaptureOptions, body func() error) ([]map[string]any, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			return p.CaptureNetwork(ctx, chromedpx.NetCaptureOpts{
				IncludeBodies: opts.IncludeBodies, MaxBodyBytes: opts.MaxBodyBytes,
				URLFilter: opts.URLFilter,
			}, body)
		},
		CaptureConsole: func(run context.Context, w domain.WindowID, body func() error) ([]map[string]any, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			return p.CaptureConsole(ctx, body)
		},
		GetCookies: func(run context.Context, w domain.WindowID, urls []string) ([]features.Cookie, error) {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return nil, err
			}
			defer cancel()
			cks, err := p.GetCookies(ctx, urls)
			if err != nil {
				return nil, err
			}
			out := make([]features.Cookie, 0, len(cks))
			for _, c := range cks {
				out = append(out, features.Cookie{
					Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path,
					Expires: c.Expires, HTTPOnly: c.HTTPOnly, Secure: c.Secure,
					SameSite: c.SameSite,
				})
			}
			return out, nil
		},
		SetCookies: func(run context.Context, w domain.WindowID, cookies []features.Cookie) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			in := make([]chromedpx.Cookie, 0, len(cookies))
			for _, c := range cookies {
				in = append(in, chromedpx.Cookie{
					Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path,
					SameSite: c.SameSite, URL: c.URL, Expires: c.Expires,
					HTTPOnly: c.HTTPOnly, Secure: c.Secure,
				})
			}
			return p.SetCookies(ctx, in)
		},
		WaitForNetworkIdle: func(run context.Context, w domain.WindowID, idleMs, timeoutMs int64, maxInflight int) error {
			ctx, cancel, err := withCtx(run, w)
			if err != nil {
				return err
			}
			defer cancel()
			return p.WaitForNetworkIdle(ctx, idleMs, timeoutMs, maxInflight)
		},
	}
}

// toPDFOpts adapts the feature-side PDFOptions to the infra-side PDFOpts.
func toPDFOpts(o features.PDFOptions) chromedpx.PDFOpts {
	return chromedpx.PDFOpts{
		Landscape: o.Landscape, PrintBackground: o.PrintBackground,
		DisplayHeaderFooter: o.DisplayHeaderFooter, Scale: o.Scale,
		PaperWidth: o.PaperWidth, PaperHeight: o.PaperHeight,
		MarginTop: o.MarginTop, MarginBottom: o.MarginBottom,
		MarginLeft: o.MarginLeft, MarginRight: o.MarginRight,
		PageRanges: o.PageRanges, HeaderTemplate: o.HeaderTemplate,
		FooterTemplate: o.FooterTemplate,
	}
}

// frameDelays converts screencast frame timestamps into per-frame GIF delays
// (1/100s units), clamped to a sane range.
func frameDelays(times []time.Time) []int {
	if len(times) < 2 {
		return nil
	}
	out := make([]int, len(times))
	for i := 0; i < len(times)-1; i++ {
		d := int(times[i+1].Sub(times[i]).Milliseconds() / 10)
		if d < 2 {
			d = 2
		}
		if d > 500 {
			d = 500
		}
		out[i] = d
	}
	out[len(times)-1] = out[len(times)-2]
	return out
}

// MakeHTTPClient wires the outbound-HTTP infra adapter onto the feature.
func MakeHTTPClient(c httpclient.Client) features.HTTPClient {
	return features.HTTPClient{
		Do: func(ctx context.Context, req features.HTTPRequest) (features.HTTPResponse, error) {
			resp, err := c.Do(ctx, req.Method, req.URL, req.Headers, req.Body,
				req.TimeoutMs, !req.NoRedirect)
			if err != nil {
				return features.HTTPResponse{}, err
			}
			return features.HTTPResponse{
				Status: resp.Status, Headers: resp.Headers, Body: resp.Body,
			}, nil
		},
	}
}

// MakeWindowLease constructs a simple counting pool. `sizes` declares how many
// windows each pool tag may hold; windows are pre-allocated. A pool tag listed
// in `persistent` (tag → profile name) gets stable, restart-surviving Chrome
// profiles instead of ephemeral ones.
func MakeWindowLease(sizes map[string]int, persistent map[string]string, src *chromedpx.WindowSource) (features.WindowLease, error) {
	type pool struct {
		size, busy int
		free       []string
	}
	pools := make(map[string]*pool, len(sizes))
	owners := make(map[string]string)
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	for tag, size := range sizes {
		p := &pool{size: size, free: make([]string, 0, size)}
		for i := 0; i < size; i++ {
			var (
				id  string
				err error
			)
			if prof, ok := persistent[tag]; ok {
				name := prof
				if i > 0 {
					name = fmt.Sprintf("%s-%d", prof, i)
				}
				id, err = src.CreatePersistent(name)
			} else {
				id, err = src.Create()
			}
			if err != nil {
				return features.WindowLease{}, fmt.Errorf("create %s window: %w", tag, err)
			}
			p.free = append(p.free, id)
			owners[id] = tag
		}
		pools[tag] = p
	}

	lease := features.WindowLease{
		Acquire: func(tag domain.PoolTag, timeout time.Duration) (domain.WindowID, error) {
			deadline := time.Now().Add(timeout)
			mu.Lock()
			defer mu.Unlock()
			p, ok := pools[string(tag)]
			if !ok {
				return "", fmt.Errorf("unknown pool: %s", tag)
			}
			for len(p.free) == 0 {
				if time.Now().After(deadline) {
					return "", errors.New("acquire timeout: " + string(tag))
				}
				cond.Wait()
			}
			id := p.free[0]
			p.free = p.free[1:]
			p.busy++
			return domain.WindowID(id), nil
		},
		Release: func(w domain.WindowID) {
			mu.Lock()
			defer mu.Unlock()
			tag, ok := owners[string(w)]
			if !ok {
				return
			}
			p := pools[tag]
			if p.busy > 0 {
				p.busy--
			}
			p.free = append(p.free, string(w))
			cond.Broadcast()
		},
		Status: func() map[string]features.PoolStatus {
			mu.Lock()
			defer mu.Unlock()
			out := make(map[string]features.PoolStatus, len(pools))
			for tag, p := range pools {
				out[tag] = features.PoolStatus{Size: p.size, Free: len(p.free), Busy: p.busy}
			}
			return out
		},
		Recover: func(w domain.WindowID) error {
			return src.Replace(string(w))
		},
	}
	return lease, nil
}

// MakeTaskRegistry loads all yaml files under `tasks/` of the bundle. Returns
// a registry that re-loads on every call (hot-reload) when `hot` is true.
func MakeTaskRegistry(b *bundle.Root, hot bool) (features.TaskRegistry, error) {
	load := func() ([]domain.TaskDef, map[string]domain.TaskDef, error) {
		var defs []domain.TaskDef
		byName := map[string]domain.TaskDef{}
		err := b.WalkYAML("tasks", func(rel string, content []byte) error {
			if path.Base(rel) == "pool.yaml" {
				return nil
			}
			var d domain.TaskDef
			if err := yamlreader.Unmarshal(content, &d); err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			defs = append(defs, d)
			byName[d.Name] = d
			return nil
		})
		return defs, byName, err
	}

	if hot {
		return features.TaskRegistry{
			List: func() []domain.TaskDef {
				defs, _, _ := load()
				return defs
			},
			Get: func(name string) (domain.TaskDef, bool) {
				_, byName, _ := load()
				d, ok := byName[name]
				return d, ok
			},
		}, nil
	}

	defs, byName, err := load()
	if err != nil {
		return features.TaskRegistry{}, err
	}
	return features.TaskRegistry{
		List: func() []domain.TaskDef { return defs },
		Get: func(name string) (domain.TaskDef, bool) {
			d, ok := byName[name]
			return d, ok
		},
	}, nil
}

// MakeHTMLExtraction adapts the goquery extractor onto the features record.
func MakeHTMLExtraction(ex goqueryx.Extractor) features.HTMLExtraction {
	return features.HTMLExtraction{
		ExtractObject: ex.ExtractObject,
		ExtractList:   ex.ExtractList,
	}
}

// MakeTemplating supplies `{{name|or:default}}` substitution with system-env
// fallback (mirrors the Java MakeTemplating).
func MakeTemplating() features.Templating {
	re := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.|:'\- ]+?)\s*\}\}`)
	resolve := func(expr string, vars map[string]any) string {
		key := expr
		fallback := ""
		if i := strings.Index(expr, "|"); i >= 0 {
			key = strings.TrimSpace(expr[:i])
			rest := strings.TrimSpace(expr[i+1:])
			if strings.HasPrefix(rest, "or:") {
				fallback = strings.TrimSpace(strings.TrimPrefix(rest, "or:"))
			}
		}
		// dotted path lookup
		var cur any
		if strings.Contains(key, ".") {
			parts := strings.Split(key, ".")
			cur = vars[parts[0]]
			for i := 1; i < len(parts); i++ {
				m, ok := cur.(map[string]any)
				if !ok {
					cur = nil
					break
				}
				cur = m[parts[i]]
			}
		} else {
			cur = vars[key]
		}
		if cur == nil || cur == "" {
			if envVal := osLookup(key); envVal != "" {
				return envVal
			}
			return fallback
		}
		return toString(cur)
	}
	return features.Templating{
		Substitute: func(template string, vars map[string]any) string {
			if template == "" {
				return ""
			}
			return re.ReplaceAllStringFunc(template, func(match string) string {
				m := re.FindStringSubmatch(match)
				if len(m) < 2 {
					return match
				}
				return resolve(m[1], vars)
			})
		},
	}
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

// osLookup is the env-var fallback used by Templating: when a `{{name}}` token
// doesn't resolve from the per-task vars map, the secrets loader has likely
// stashed it in the process env (see orchestrator/secrets.go).
func osLookup(key string) string { return os.Getenv(key) }
