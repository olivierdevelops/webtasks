// Package usecases (under orchestrator) holds wired-up use-case
// implementations. RunRegisteredTaskImpl is the bytecode-interpreter for
// task YAML flows.
package usecases

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"webtasks/internal/domain"
	"webtasks/internal/features"
	"webtasks/internal/usecases/runtask"
)

// errStopFlow is returned by the `return` action to halt the remaining steps.
// runFlow treats it as a successful early exit (Execute unwraps it).
var errStopFlow = errors.New("flow stopped by return")

func filepathBase(p string) string {
	if p == "" {
		return ""
	}
	return filepath.Base(p)
}

// New wires an instance that satisfies the RunRegisteredTask use-case
// contract. It owns no state; the deps it needs are passed as features.
func NewRunRegisteredTask(
	reg features.TaskRegistry,
	lease features.WindowLease,
	browser features.BrowserActions,
	extract features.HTMLExtraction,
	tpl features.Templating,
	scripts features.JsScripts,
	http features.HTTPClient,
) runtask.RunRegisteredTask {
	return &runImpl{
		registry: reg, lease: lease, browser: browser,
		extract: extract, tpl: tpl, scripts: scripts, http: http,
	}
}

type runImpl struct {
	registry features.TaskRegistry
	lease    features.WindowLease
	browser  features.BrowserActions
	extract  features.HTMLExtraction
	tpl      features.Templating
	scripts  features.JsScripts
	http     features.HTTPClient
}

func (r *runImpl) Execute(ctx context.Context, name string, vals domain.InputValues, events features.EventPublisher) (domain.Output, error) {
	def, ok := r.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown task: %s", name)
	}
	// Bound the whole run by the task's declared timeout so a stuck action
	// (a never-appearing selector, a hung click) fails cleanly instead of
	// blocking the window lease forever.
	ctx, cancel := context.WithTimeout(ctx, def.Timeout())
	defer cancel()

	bindings, err := bindInputs(def, vals)
	if err != nil {
		return nil, err
	}
	w, err := r.lease.Acquire(def.PoolTag, 30*time.Second)
	if err != nil {
		return nil, err
	}
	defer r.lease.Release(w)

	out := domain.Output{}

	// Idempotent setup prelude: lets data tasks declare `setupTask: foo/bar`
	// and have that task's flow run (in the same window) first. We pass the
	// caller's inputs through so the setup task's templating sees the same
	// bindings, but only emit a single "running setup" status to avoid
	// drowning the caller's SSE stream in prelude events.
	if def.SetupTask != "" && def.SetupTask != name {
		setupDef, ok := r.registry.Get(def.SetupTask)
		if !ok {
			return nil, fmt.Errorf("setup task %q not found (referenced by %q)", def.SetupTask, name)
		}
		setupBindings, err := bindInputs(setupDef, vals)
		if err != nil {
			return nil, fmt.Errorf("setup %q: %w", def.SetupTask, err)
		}
		events.Emit(domain.Event{Kind: "status", Text: "Running setup: " + def.SetupTask})
		if err := r.runFlow(ctx, w, setupDef.Flow, setupBindings, out, events); err != nil && !errors.Is(err, errStopFlow) {
			if isFatalBrowserState(err) {
				_ = r.lease.Recover(w)
				return out, fmt.Errorf("setup %q failed and the browser session was reset: %w", def.SetupTask, err)
			}
			return out, fmt.Errorf("setup %q failed: %w", def.SetupTask, err)
		}
	}

	if err := r.runFlow(ctx, w, def.Flow, bindings, out, events); err != nil {
		if errors.Is(err, errStopFlow) {
			// `return` action — successful early exit.
			return out, nil
		}
		if isFatalBrowserState(err) {
			// Replace the underlying Chrome target so the next caller gets a
			// fresh window. The pool slot stays usable; caller needs to re-run
			// any setup task (e.g. login) before using the window again.
			_ = r.lease.Recover(w)
			return out, fmt.Errorf("browser session was reset (tab crashed or detached); "+
				"re-run pool setup before retrying. original: %w", err)
		}
		return out, err
	}
	return out, nil
}

// isFatalBrowserState reports whether an executor error indicates the Chrome
// target is gone — at which point retrying against the same window is pointless.
func isFatalBrowserState(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for _, marker := range []string{
		"target detached",
		"target closed",
		"session deleted",
		"tab crashed",
		"websocket: close",
		"page crashed",
		"context canceled",
	} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

func bindInputs(def domain.TaskDef, vals domain.InputValues) (map[string]any, error) {
	out := map[string]any{}
	var missing []string
	for k, f := range def.InputSchema {
		v, ok := vals[k]
		if !ok {
			v = f.DefaultValue
		}
		if (v == nil || v == "") && f.Required {
			missing = append(missing, k)
			continue
		}
		if v != nil {
			out[k] = v
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required input(s): %s", strings.Join(missing, ", "))
	}
	// Carry through unmodelled inputs too.
	for k, v := range vals {
		if _, set := out[k]; !set {
			out[k] = v
		}
	}
	return out, nil
}

func (r *runImpl) runFlow(ctx context.Context, w domain.WindowID, commands []domain.Command, bindings map[string]any, out domain.Output, events features.EventPublisher) error {
	for _, c := range commands {
		if err := r.runCommand(ctx, w, c, bindings, out, events); err != nil {
			if errors.Is(err, errStopFlow) {
				return errStopFlow
			}
			return fmt.Errorf("step %q: %w", c.Run, err)
		}
	}
	return nil
}

// runCommand dispatches one step. When the step carries `record: true` it is
// wrapped in a screencast so the run can be inspected afterwards (a debug aid
// for failing steps); otherwise it runs directly.
func (r *runImpl) runCommand(ctx context.Context, w domain.WindowID, c domain.Command, bindings map[string]any, out domain.Output, events features.EventPublisher) error {
	if !c.Record {
		return r.runCommandRaw(ctx, w, c, bindings, out, events)
	}
	var stepErr error
	res, _ := r.browser.Record(ctx, w,
		features.ScreencastOptions{Format: "gif", FPS: 4, MaxFrames: 200},
		func() error {
			stepErr = r.runCommandRaw(ctx, w, c, bindings, out, events)
			return stepErr
		})
	if len(res.Data) > 0 {
		path := recordingPath(c.Run)
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr == nil {
			if wErr := os.WriteFile(path, res.Data, 0o644); wErr == nil {
				events.Emit(domain.Event{Kind: "recording", Text: "step recording saved",
					Data: map[string]any{"path": path, "step": c.Run, "ok": stepErr == nil}})
				if stepErr != nil {
					return fmt.Errorf("%w (step recording: %s)", stepErr, path)
				}
			}
		}
	}
	return stepErr
}

func (r *runImpl) runCommandRaw(ctx context.Context, w domain.WindowID, c domain.Command, bindings map[string]any, out domain.Output, events features.EventPublisher) error {
	params := r.renderParams(c.Params, bindings)
	if c.Status != "" {
		events.Emit(domain.Event{Kind: "status", Text: r.tpl.Substitute(c.Status, bindings)})
	}
	switch c.Run {
	case "goto":
		return r.browser.Goto(ctx, w, asString(params["url"]))
	case "wait":
		dur := parseDuration(params["duration"])
		r.browser.Sleep(time.Duration(dur) * time.Millisecond)
		return nil
	case "wait-for":
		timeout := time.Duration(longOr(params["timeoutMs"], 10000)) * time.Millisecond
		return r.browser.WaitFor(ctx, w, asString(params["selector"]), timeout)
	case "sendkeys":
		return r.browser.SendKeys(ctx, w, asString(params["selector"]), asString(params["keys"]))
	case "action":
		if asString(params["action"]) != "click" {
			return fmt.Errorf("unsupported action: %v", params["action"])
		}
		// `text:` present → click the element matching selector by visible
		// text (YAML-native; no JS module needed). Otherwise click the
		// first selector match.
		if t := asString(params["text"]); t != "" {
			return r.browser.ClickText(ctx, w,
				asStringOr(params["selector"], "*"), t,
				asStringOr(params["match"], "exact"),
				asString(params["closest"]))
		}
		return r.browser.Click(ctx, w, asString(params["selector"]))
	case "set":
		// Assign a literal or templated value to a binding/output. `value`
		// is whatever the param renders to (string, list, map, number).
		assign(out, bindings, c.As, params["value"])
		return nil
	case "read-file":
		path := asString(params["path"])
		if path == "" {
			return fmt.Errorf("read-file requires `path`")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) && asBool(params["optional"]) {
				assign(out, bindings, c.As, "")
				return nil
			}
			return fmt.Errorf("read-file %s: %w", path, err)
		}
		assign(out, bindings, c.As, string(data))
		return nil
	case "scroll-until-stable":
		return r.browser.ScrollUntilStable(ctx, w,
			asString(params["selector"]),
			asStringOr(params["direction"], "up"),
			longOr(params["stableMs"], 1500),
			intOr(params["maxIterations"], 0),
		)
	case "screenshot":
		data, err := r.browser.Screenshot(ctx, w, features.ScreenshotOptions{
			Selector: asStringOr(params["selector"], "."),
			FullPage: asBool(params["fullPage"]),
			Format:   asString(params["format"]),
			Quality:  intOr(params["quality"], 0),
		})
		if err != nil {
			return err
		}
		return r.emitBytes(out, bindings, c, params, data)
	case "js":
		script, err := r.resolveJS(params)
		if err != nil {
			return err
		}
		args := jsArgs(params["args"])
		result, err := r.browser.ExecuteJS(ctx, w, script, asBool(params["await"]), args...)
		if err != nil {
			return err
		}
		assign(out, bindings, c.As, result)
		return nil
	case "pdf":
		data, err := r.browser.PrintToPDF(ctx, w, pdfOptions(params))
		if err != nil {
			return err
		}
		return r.emitBytes(out, bindings, c, params, data)
	case "snapshot":
		data, err := r.browser.CaptureSnapshot(ctx, w)
		if err != nil {
			return err
		}
		return r.emitBytes(out, bindings, c, params, data)
	case "html-to-pdf":
		html := asString(params["html"])
		if html == "" {
			src := asString(params["file"])
			if src == "" {
				return fmt.Errorf("html-to-pdf requires `html` or `file`")
			}
			b, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("html-to-pdf: read %s: %w", src, err)
			}
			html = string(b)
		}
		if css := asString(params["css"]); css != "" {
			html = "<style>" + css + "</style>\n" + html
		}
		data, err := r.browser.RenderHTMLToPDF(ctx, w, html, pdfOptions(params))
		if err != nil {
			return err
		}
		return r.emitBytes(out, bindings, c, params, data)
	case "emulate":
		return r.browser.Emulate(ctx, w, features.EmulateOptions{
			Width:             longOr(params["width"], 0),
			Height:            longOr(params["height"], 0),
			DeviceScaleFactor: floatOr(params["deviceScaleFactor"], 0),
			Mobile:            asBool(params["mobile"]),
			ColorScheme:       asString(params["colorScheme"]),
			Reset:             asBool(params["reset"]),
		})
	case "record":
		res, bodyErr := r.browser.Record(ctx, w, recordOptions(params), func() error {
			return r.runFlow(ctx, w, c.Children, bindings, out, events)
		})
		path := asString(params["path"])
		// Persist the recording even when the wrapped steps failed — that is
		// exactly the run worth inspecting.
		if len(res.Data) > 0 {
			if path != "" {
				if err := writeFileMk(path, res.Data); err != nil {
					return err
				}
			}
			if c.As != "" {
				m := map[string]any{"frames": res.Frames, "durationMs": res.DurationMs, "size": len(res.Data)}
				if path != "" {
					m["path"] = path
				} else {
					m["bytesB64"] = base64.StdEncoding.EncodeToString(res.Data)
				}
				assign(out, bindings, c.As, m)
			}
		}
		if bodyErr != nil {
			return bodyErr
		}
		if path == "" && c.As == "" {
			return fmt.Errorf("record requires `path` or `as`")
		}
		return nil
	case "capture-network":
		entries, err := r.browser.CaptureNetwork(ctx, w, features.NetCaptureOptions{
			IncludeBodies: asBool(params["includeBodies"]),
			MaxBodyBytes:  intOr(params["maxBodyBytes"], 65536),
			URLFilter:     asString(params["urlFilter"]),
		}, func() error {
			return r.runFlow(ctx, w, c.Children, bindings, out, events)
		})
		if err != nil {
			return err
		}
		har := map[string]any{"entries": entries, "count": len(entries)}
		if path := asString(params["path"]); path != "" {
			js, _ := json.MarshalIndent(har, "", "  ")
			if err := writeFileMk(path, js); err != nil {
				return err
			}
		}
		if c.As != "" {
			assign(out, bindings, c.As, har)
		}
		return nil
	case "console":
		logs, err := r.browser.CaptureConsole(ctx, w, func() error {
			return r.runFlow(ctx, w, c.Children, bindings, out, events)
		})
		if err != nil {
			return err
		}
		assign(out, bindings, c.As, logs)
		return nil
	case "wait-for-network-idle":
		return r.browser.WaitForNetworkIdle(ctx, w,
			longOr(params["idleMs"], 500),
			longOr(params["timeoutMs"], 15000),
			intOr(params["maxInflight"], 0))
	case "get-cookies":
		var urls []string
		if lst, ok := toAnySlice(params["urls"]); ok {
			for _, u := range lst {
				urls = append(urls, asString(u))
			}
		}
		cks, err := r.browser.GetCookies(ctx, w, urls)
		if err != nil {
			return err
		}
		rows := make([]map[string]any, 0, len(cks))
		for _, ck := range cks {
			rows = append(rows, map[string]any{
				"name": ck.Name, "value": ck.Value, "domain": ck.Domain,
				"path": ck.Path, "expires": ck.Expires, "httpOnly": ck.HTTPOnly,
				"secure": ck.Secure, "sameSite": ck.SameSite,
			})
		}
		if path := asString(params["path"]); path != "" {
			js, _ := json.MarshalIndent(rows, "", "  ")
			if err := writeFileMk(path, js); err != nil {
				return err
			}
		}
		if c.As != "" {
			assign(out, bindings, c.As, rows)
		}
		return nil
	case "set-cookies":
		raw, err := r.resolveCookieList(params)
		if err != nil {
			return err
		}
		cookies := make([]features.Cookie, 0, len(raw))
		for _, item := range raw {
			m, ok := toStringMap(item)
			if !ok {
				continue
			}
			cookies = append(cookies, features.Cookie{
				Name: asString(m["name"]), Value: asString(m["value"]),
				Domain: asString(m["domain"]), Path: asString(m["path"]),
				URL: asString(m["url"]), SameSite: asString(m["sameSite"]),
				Expires:  floatOr(m["expires"], 0),
				HTTPOnly: asBool(m["httpOnly"]), Secure: asBool(m["secure"]),
			})
		}
		if err := r.browser.SetCookies(ctx, w, cookies); err != nil {
			return err
		}
		assign(out, bindings, c.As, map[string]any{"count": len(cookies)})
		return nil
	case "http-request":
		return r.runHTTPRequest(ctx, c, params, out, bindings)
	case "export":
		return r.runExport(c, params, out, bindings)
	case "return":
		assign(out, bindings, "__result__", params["value"])
		return errStopFlow
	case "loop":
		return r.runLoop(ctx, w, c, params, bindings, out, events)
	case "call":
		name := asString(params["task"])
		if name == "" {
			return fmt.Errorf("call requires `task`")
		}
		def, ok := r.registry.Get(name)
		if !ok {
			return fmt.Errorf("call: unknown task %q", name)
		}
		// Run the named task's flow in the current window with the current
		// bindings — a way to factor a reusable flow (e.g. a watch-loop that
		// repeatedly invokes a watch task).
		return r.runFlow(ctx, w, def.Flow, bindings, out, events)
	case "save-html":
		return r.browser.SaveHTML(ctx, w, asStringOr(params["selector"], "."), asString(params["path"]))
	case "extract":
		html, err := r.browser.GetOuterHTML(ctx, w, asStringOr(params["from"], "."))
		if err != nil {
			return err
		}
		spec := buildExtractSpec(params)
		if asBool(params["repeat"]) {
			rows, err := r.extract.ExtractList(html, spec)
			if err != nil {
				return err
			}
			assign(out, bindings, c.As, rows)
		} else {
			obj, err := r.extract.ExtractObject(html, spec)
			if err != nil {
				return err
			}
			assign(out, bindings, c.As, obj)
		}
		return nil
	case "emit-event":
		events.Emit(domain.Event{
			Kind: asStringOr(params["kind"], "status"),
			Text: asString(params["text"]),
			Data: asMap(params["data"]),
		})
		return nil
	case "download-each":
		perFile := time.Duration(longOr(params["timeoutPerFileMs"], 30000)) * time.Millisecond
		paths, err := r.browser.DownloadEach(ctx, w, asString(params["selector"]), perFile)
		if err != nil {
			return err
		}
		results := make([]map[string]any, 0, len(paths))
		for _, p := range paths {
			results = append(results, map[string]any{"path": p, "basename": filepathBase(p)})
		}
		assign(out, bindings, c.As, results)
		return nil
	case "save-captures-to-dir":
		dir := asString(params["dir"])
		if dir == "" {
			return fmt.Errorf("save-captures-to-dir requires `dir`")
		}
		naming := asStringOr(params["naming"], "{id}_{name}")
		saved, err := r.browser.SaveCapturesToDir(ctx, w, dir, naming)
		if err != nil {
			return err
		}
		assign(out, bindings, c.As, saved)
		return nil
	case "for-each":
		return r.runForEach(ctx, w, c, params, bindings, out, events)
	case "write-files":
		return r.runWriteFiles(c, params, out, bindings)
	default:
		return fmt.Errorf("unknown command: %s", c.Run)
	}
}

// assign records a command's `as:` result into both the response Output and
// the live bindings map, so later steps can reference it with `{{name}}`.
// No-op when the command declared no `as:`.
func assign(out domain.Output, bindings map[string]any, key string, val any) {
	if key == "" {
		return
	}
	out[key] = val
	bindings[key] = val
}

// runForEach is a generic iteration action. `over` is a list (typically a
// `{{ref}}` to a prior step's output); `as` names the per-item binding.
// Each iteration runs the command's `do:` children with a cloned bindings
// map carrying `<as>` and `<as>_index`. `continueOnError: true` keeps the
// loop going (and emits an error event) when an iteration fails.
func (r *runImpl) runForEach(ctx context.Context, w domain.WindowID, c domain.Command,
	params map[string]any, bindings map[string]any, out domain.Output, events features.EventPublisher) error {

	items, ok := toAnySlice(params["over"])
	if !ok {
		return fmt.Errorf("for-each: `over` did not resolve to a list (got %T)", params["over"])
	}
	itemVar := asStringOr(params["as"], "item")
	continueOnError := asBool(params["continueOnError"])

	for i, item := range items {
		child := cloneBindings(bindings)
		child[itemVar] = item
		child[itemVar+"_index"] = i
		if err := r.runFlow(ctx, w, c.Children, child, out, events); err != nil {
			if errors.Is(err, errStopFlow) {
				return errStopFlow
			}
			if continueOnError {
				events.Emit(domain.Event{Kind: "error",
					Text: fmt.Sprintf("for-each item %d failed: %v", i, err)})
				continue
			}
			return fmt.Errorf("for-each item %d: %w", i, err)
		}
	}
	return nil
}

// runWriteFiles is the generic "store content + create dirs" backend
// function. `files` is a list of {path, content} or {path, bytesB64};
// each is written under `root`, parent directories created as needed.
func (r *runImpl) runWriteFiles(c domain.Command, params map[string]any,
	out domain.Output, bindings map[string]any) error {

	root := asString(params["root"])
	files, ok := toAnySlice(params["files"])
	if !ok {
		return fmt.Errorf("write-files: `files` did not resolve to a list (got %T)", params["files"])
	}
	written := make([]map[string]any, 0, len(files))
	for _, f := range files {
		m, ok := toStringMap(f)
		if !ok {
			continue
		}
		rel := asString(m["path"])
		if rel == "" {
			continue
		}
		full := filepath.Join(root, filepath.Clean("/"+rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("write-files: mkdir %s: %w", filepath.Dir(full), err)
		}
		var data []byte
		if b64, ok := m["bytesB64"].(string); ok && b64 != "" {
			decoded, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return fmt.Errorf("write-files: bad base64 for %s: %w", rel, err)
			}
			data = decoded
		} else {
			data = []byte(asString(m["content"]))
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			return fmt.Errorf("write-files: %s: %w", full, err)
		}
		written = append(written, map[string]any{"path": full, "size": len(data)})
	}
	assign(out, bindings, c.As, map[string]any{"count": len(written), "root": root, "files": written})
	return nil
}

// cloneBindings makes a shallow copy so for-each iteration variables don't
// leak between iterations or back to the parent flow.
func cloneBindings(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src)+2)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// toAnySlice coerces the various concrete slice types the engine produces
// (`[]any` from JS results, `[]map[string]any` from extract) into `[]any`.
func toAnySlice(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []map[string]any:
		out := make([]any, len(t))
		for i, m := range t {
			out[i] = m
		}
		return out, true
	case nil:
		return nil, false
	default:
		return nil, false
	}
}

// toStringMap coerces a JS/JSON object value into map[string]any.
func toStringMap(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[fmt.Sprintf("%v", k)] = val
		}
		return out, true
	default:
		return nil, false
	}
}

func (r *runImpl) resolveJS(p map[string]any) (string, error) {
	fn := asString(p["fn"])
	if fn == "" {
		fn = asString(p["file"])
	}
	if fn != "" {
		body, ok := r.scripts.Get(fn)
		if !ok {
			return "", fmt.Errorf("js module not found: %s", fn)
		}
		return body, nil
	}
	inline := asString(p["script"])
	if inline == "" {
		return "", errors.New("js action requires `fn` or `script`")
	}
	return inline, nil
}

func (r *runImpl) renderParams(params map[string]any, bindings map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = r.renderValue(v, bindings)
	}
	return out
}

// singleTokenRE matches a param value that is *exactly* one `{{ref}}` token.
var singleTokenRE = regexp.MustCompile(`^\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}$`)

func (r *runImpl) renderValue(v any, bindings map[string]any) any {
	switch t := v.(type) {
	case string:
		// A param whose entire value is one `{{ref}}` token resolves to the
		// *raw* bound value — list, map, number, bool — not its stringified
		// form. This is what lets `for-each over: "{{chats}}"` receive the
		// actual list a prior step produced, and `write-files files: "{{x}}"`
		// receive a real slice. Plain text or multi-token strings still
		// substitute normally.
		if m := singleTokenRE.FindStringSubmatch(t); m != nil {
			if val, ok := lookupBinding(m[1], bindings); ok {
				if _, isStr := val.(string); !isStr && val != nil {
					return val
				}
			}
		}
		return r.tpl.Substitute(t, bindings)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = r.renderValue(item, bindings)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = r.renderValue(val, bindings)
		}
		return out
	default:
		return v
	}
}

// lookupBinding resolves a dotted reference (`chat.peerName`) against the
// bindings map. Returns (value, true) only when every path segment exists.
func lookupBinding(ref string, bindings map[string]any) (any, bool) {
	parts := strings.Split(ref, ".")
	var cur any = bindings
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func buildExtractSpec(p map[string]any) domain.ExtractSpec {
	spec := domain.ExtractSpec{
		Selector: asStringOr(p["selector"], "."),
		Repeat:   asBool(p["repeat"]),
		Fields:   map[string]domain.ExtractFieldSpec{},
	}
	if raw, ok := p["fields"].(map[string]any); ok {
		for name, fdef := range raw {
			// The capy transpiler emits each field spec as a single-element
			// list; hand-written YAML uses a bare object. Accept both.
			if lst, ok := fdef.([]any); ok && len(lst) > 0 {
				fdef = lst[0]
			}
			fm, _ := fdef.(map[string]any)
			spec.Fields[name] = domain.ExtractFieldSpec{
				Kind:       asStringOr(fm["kind"], "text"),
				Selector:   asStringOr(fm["selector"], "."),
				AttrName:   asString(fm["name"]),
				Transform:  asString(fm["transform"]),
				ConstValue: fm["value"],
			}
		}
	}
	return spec
}

// --- type helpers ---

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func asStringOr(v any, def string) string {
	s := asString(v)
	if s == "" {
		return def
	}
	return s
}

func asBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func parseDuration(v any) int64 { return longOr(v, 0) }

func longOr(v any, def int64) int64 {
	switch t := v.(type) {
	case nil:
		return def
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case string:
		s := strings.ReplaceAll(strings.TrimSpace(t), "_", "")
		if s == "" {
			return def
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
	}
	return def
}

func intOr(v any, def int) int { return int(longOr(v, int64(def))) }

func jsArgs(v any) []any {
	lst, ok := v.([]any)
	if !ok {
		return nil
	}
	return lst
}

func floatOr(v any, def float64) float64 {
	switch t := v.(type) {
	case nil:
		return def
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return def
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return def
}

// writeFileMk writes data to path, creating parent directories as needed.
func writeFileMk(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// emitBytes routes a binary artifact to disk (`path:`) and/or a base64 binding
// (`as:`). With both, the binding is a {path,size,bytesB64} map.
func (r *runImpl) emitBytes(out domain.Output, bindings map[string]any, c domain.Command, params map[string]any, data []byte) error {
	path := asString(params["path"])
	if path == "" && c.As == "" {
		return fmt.Errorf("%s requires `path` or `as`", c.Run)
	}
	if path != "" {
		if err := writeFileMk(path, data); err != nil {
			return fmt.Errorf("%s: %w", c.Run, err)
		}
	}
	if c.As != "" {
		if path != "" {
			assign(out, bindings, c.As, map[string]any{
				"path": path, "size": len(data),
				"bytesB64": base64.StdEncoding.EncodeToString(data),
			})
		} else {
			assign(out, bindings, c.As, base64.StdEncoding.EncodeToString(data))
		}
	}
	return nil
}

// emitText routes a text artifact to disk (`path:`) and/or a string binding.
func (r *runImpl) emitText(out domain.Output, bindings map[string]any, c domain.Command, params map[string]any, text string) error {
	path := asString(params["path"])
	if path == "" && c.As == "" {
		return fmt.Errorf("%s requires `path` or `as`", c.Run)
	}
	if path != "" {
		if err := writeFileMk(path, []byte(text)); err != nil {
			return fmt.Errorf("%s: %w", c.Run, err)
		}
	}
	if c.As != "" {
		assign(out, bindings, c.As, text)
	}
	return nil
}

var sanitiseStep = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

// recordingPath is where a per-step `record: true` GIF lands.
func recordingPath(step string) string {
	safe := sanitiseStep.ReplaceAllString(step, "_")
	return filepath.Join(os.TempDir(), "webtasks-recordings",
		fmt.Sprintf("%s_%d.gif", safe, time.Now().UnixNano()))
}

// pdfOptions builds PDFOptions from a `pdf` / `html-to-pdf` param map.
func pdfOptions(p map[string]any) features.PDFOptions {
	o := features.PDFOptions{
		Landscape:           asBool(p["landscape"]),
		PrintBackground:     true,
		Scale:               floatOr(p["scale"], 0),
		PaperWidth:          floatOr(p["paperWidth"], 0),
		PaperHeight:         floatOr(p["paperHeight"], 0),
		MarginTop:           floatOr(p["marginTop"], 0),
		MarginBottom:        floatOr(p["marginBottom"], 0),
		MarginLeft:          floatOr(p["marginLeft"], 0),
		MarginRight:         floatOr(p["marginRight"], 0),
		PageRanges:          asString(p["pageRanges"]),
		DisplayHeaderFooter: asBool(p["displayHeaderFooter"]),
		HeaderTemplate:      asString(p["headerTemplate"]),
		FooterTemplate:      asString(p["footerTemplate"]),
	}
	if v, ok := p["printBackground"]; ok {
		o.PrintBackground = asBool(v)
	}
	switch strings.ToLower(asString(p["format"])) {
	case "letter":
		o.PaperWidth, o.PaperHeight = 8.5, 11
	case "legal":
		o.PaperWidth, o.PaperHeight = 8.5, 14
	case "a4":
		o.PaperWidth, o.PaperHeight = 8.27, 11.69
	case "a3":
		o.PaperWidth, o.PaperHeight = 11.69, 16.54
	}
	return o
}

// recordOptions builds ScreencastOptions from a `record` param map.
func recordOptions(p map[string]any) features.ScreencastOptions {
	return features.ScreencastOptions{
		Format:        asStringOr(p["format"], "gif"),
		FPS:           intOr(p["fps"], 5),
		Quality:       intOr(p["quality"], 80),
		EveryNthFrame: intOr(p["everyNthFrame"], 2),
		MaxFrames:     intOr(p["maxFrames"], 300),
		MaxDurationMs: longOr(p["maxDurationMs"], 30000),
	}
}

// resolveCookieList reads the cookie list for `set-cookies` from either the
// `cookies` param (inline) or a JSON file at `path`.
func (r *runImpl) resolveCookieList(params map[string]any) ([]any, error) {
	if lst, ok := toAnySlice(params["cookies"]); ok {
		return lst, nil
	}
	if path := asString(params["path"]); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("set-cookies: read %s: %w", path, err)
		}
		var parsed []any
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("set-cookies: parse %s: %w", path, err)
		}
		return parsed, nil
	}
	return nil, fmt.Errorf("set-cookies requires `cookies` or `path`")
}

// runHTTPRequest performs the `http-request` action.
func (r *runImpl) runHTTPRequest(ctx context.Context, c domain.Command, params map[string]any, out domain.Output, bindings map[string]any) error {
	url := asString(params["url"])
	if url == "" {
		return fmt.Errorf("http-request requires `url`")
	}
	if r.http.Do == nil {
		return fmt.Errorf("http-request: HTTP client not configured")
	}
	headers := map[string]string{}
	if hm, ok := params["headers"].(map[string]any); ok {
		for k, v := range hm {
			headers[k] = asString(v)
		}
	}
	var body []byte
	switch bv := params["body"].(type) {
	case nil:
	case string:
		body = []byte(bv)
	default:
		js, _ := json.Marshal(bv)
		body = js
		if _, has := headers["Content-Type"]; !has {
			headers["Content-Type"] = "application/json"
		}
	}
	resp, err := r.http.Do(ctx, features.HTTPRequest{
		Method:     strings.ToUpper(asStringOr(params["method"], "GET")),
		URL:        url,
		Headers:    headers,
		Body:       body,
		TimeoutMs:  longOr(params["timeoutMs"], 30000),
		NoRedirect: params["followRedirects"] == false,
	})
	if err != nil {
		return err
	}
	result := map[string]any{"status": resp.Status, "headers": resp.Headers, "body": resp.Body}
	var parsed any
	if json.Unmarshal([]byte(resp.Body), &parsed) == nil {
		result["json"] = parsed
	}
	assign(out, bindings, c.As, result)
	return nil
}

// runExport renders a list/map into CSV / NDJSON / markdown-table text.
func (r *runImpl) runExport(c domain.Command, params map[string]any, out domain.Output, bindings map[string]any) error {
	format := strings.ToLower(asStringOr(params["format"], "csv"))
	rows := exportRows(params["data"])
	var cols []string
	if lst, ok := toAnySlice(params["columns"]); ok {
		for _, x := range lst {
			cols = append(cols, asString(x))
		}
	}
	if len(cols) == 0 {
		cols = unionKeys(rows)
	}
	var text string
	var err error
	switch format {
	case "csv":
		text, err = exportCSV(rows, cols)
	case "ndjson":
		text = exportNDJSON(rows)
	case "md-table", "md", "markdown":
		text = exportMarkdownTable(rows, cols)
	default:
		return fmt.Errorf("export: unknown format %q (use csv|ndjson|md-table)", format)
	}
	if err != nil {
		return err
	}
	return r.emitText(out, bindings, c, params, text)
}

// exportRows coerces the `data` param into a slice of maps.
func exportRows(v any) []map[string]any {
	lst, ok := toAnySlice(v)
	if !ok {
		if m, ok := toStringMap(v); ok {
			return []map[string]any{m}
		}
		return nil
	}
	rows := make([]map[string]any, 0, len(lst))
	for _, item := range lst {
		if m, ok := toStringMap(item); ok {
			rows = append(rows, m)
		}
	}
	return rows
}

func unionKeys(rows []map[string]any) []string {
	seen := map[string]bool{}
	var keys []string
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func exportCSV(rows []map[string]any, cols []string) (string, error) {
	var sb strings.Builder
	wr := csv.NewWriter(&sb)
	if err := wr.Write(cols); err != nil {
		return "", err
	}
	for _, row := range rows {
		rec := make([]string, len(cols))
		for i, col := range cols {
			rec[i] = asString(row[col])
		}
		if err := wr.Write(rec); err != nil {
			return "", err
		}
	}
	wr.Flush()
	return sb.String(), wr.Error()
}

func exportNDJSON(rows []map[string]any) string {
	var sb strings.Builder
	for _, row := range rows {
		js, _ := json.Marshal(row)
		sb.Write(js)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func exportMarkdownTable(rows []map[string]any, cols []string) string {
	var sb strings.Builder
	sb.WriteString("| " + strings.Join(cols, " | ") + " |\n")
	seps := make([]string, len(cols))
	for i := range seps {
		seps[i] = "---"
	}
	sb.WriteString("| " + strings.Join(seps, " | ") + " |\n")
	for _, row := range rows {
		cells := make([]string, len(cols))
		for i, col := range cols {
			cells[i] = strings.ReplaceAll(asString(row[col]), "|", "\\|")
		}
		sb.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	return sb.String()
}

// runLoop is the generic while-loop: it evaluates the `while`/`until` JS
// condition each iteration, runs the `do:` children, and pauses between.
func (r *runImpl) runLoop(ctx context.Context, w domain.WindowID, c domain.Command, params map[string]any, bindings map[string]any, out domain.Output, events features.EventPublisher) error {
	whileJS := asString(params["while"])
	untilJS := asString(params["until"])
	// `whileFn` / `untilFn` resolve a JS module from the bundle instead of an
	// inline expression — handy for non-trivial conditions.
	if fn := asString(params["whileFn"]); fn != "" {
		body, ok := r.scripts.Get(fn)
		if !ok {
			return fmt.Errorf("loop: js module not found: %s", fn)
		}
		whileJS = body
	}
	if fn := asString(params["untilFn"]); fn != "" {
		body, ok := r.scripts.Get(fn)
		if !ok {
			return fmt.Errorf("loop: js module not found: %s", fn)
		}
		untilJS = body
	}
	pause := time.Duration(longOr(params["pauseMs"], 1000)) * time.Millisecond
	maxIter := intOr(params["maxIterations"], 1000)
	if maxIter <= 0 {
		maxIter = 1000
	}
	for i := 0; i < maxIter; i++ {
		if untilJS != "" {
			done, err := r.evalBool(ctx, w, untilJS)
			if err != nil {
				return fmt.Errorf("loop `until`: %w", err)
			}
			if done {
				return nil
			}
		}
		if whileJS != "" {
			cont, err := r.evalBool(ctx, w, whileJS)
			if err != nil {
				return fmt.Errorf("loop `while`: %w", err)
			}
			if !cont {
				return nil
			}
		}
		child := cloneBindings(bindings)
		child["loop_index"] = i
		if err := r.runFlow(ctx, w, c.Children, child, out, events); err != nil {
			if errors.Is(err, errStopFlow) {
				return errStopFlow
			}
			return fmt.Errorf("loop iteration %d: %w", i, err)
		}
		if i+1 < maxIter {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(pause):
			}
		}
	}
	return nil
}

// evalBool evaluates a JS expression and reports its truthiness. A plain
// expression (no `return`) is wrapped so `until: "x === 1"` works.
func (r *runImpl) evalBool(ctx context.Context, w domain.WindowID, expr string) (bool, error) {
	script := expr
	if !strings.Contains(expr, "return") {
		script = "return (" + expr + ");"
	}
	v, err := r.browser.ExecuteJS(ctx, w, script, false)
	if err != nil {
		return false, err
	}
	return truthy(v), nil
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case nil:
		return false
	case string:
		return t != "" && t != "false" && t != "0"
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return true
	}
}
