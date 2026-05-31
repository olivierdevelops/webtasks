// Package features declares capabilities as structs of function values. The
// orchestrator constructs concrete instances by closing over infra adapters.
// No `*Impl` types live here — the struct value IS the feature.
package features

import (
	"context"
	"time"

	"webtasks/internal/domain"
)

// BrowserActions mirrors the Java BrowserActions record. Every operation
// takes a context so chromedp can honour cancellation and per-call timeouts.
type BrowserActions struct {
	Goto  func(ctx context.Context, w domain.WindowID, url string) error
	Click func(ctx context.Context, w domain.WindowID, selector string) error
	// ClickText clicks an element addressed by visible text. It finds the
	// first `selector` whose trimmed textContent matches `text` (`mode` =
	// "exact" default, or "contains"); if `closest` is non-empty it then
	// walks up to that element's nearest `closest` ancestor and clicks that
	// instead (match a label, click its row). All YAML-native — no JS module.
	ClickText         func(ctx context.Context, w domain.WindowID, selector, text, mode, closest string) error
	SendKeys          func(ctx context.Context, w domain.WindowID, selector, text string) error
	WaitFor           func(ctx context.Context, w domain.WindowID, selector string, timeout time.Duration) error
	ScrollUntilStable func(ctx context.Context, w domain.WindowID, selector, direction string, stableMs int64, maxIterations int) error
	Screenshot        func(ctx context.Context, w domain.WindowID, opts ScreenshotOptions) ([]byte, error)
	// ExecuteJS evaluates a script. When `await` is true a returned Promise
	// is awaited (chromedp WithAwaitPromise) so async routines resolve.
	ExecuteJS    func(ctx context.Context, w domain.WindowID, script string, await bool, args ...any) (any, error)
	SaveHTML     func(ctx context.Context, w domain.WindowID, selector, path string) error
	GetOuterHTML func(ctx context.Context, w domain.WindowID, selector string) (string, error)
	Sleep        func(d time.Duration)

	// --- Batch A: rendering & capture ---
	PrintToPDF      func(ctx context.Context, w domain.WindowID, opts PDFOptions) ([]byte, error)
	RenderHTMLToPDF func(ctx context.Context, w domain.WindowID, html string, opts PDFOptions) ([]byte, error)
	CaptureSnapshot func(ctx context.Context, w domain.WindowID) ([]byte, error)
	Emulate         func(ctx context.Context, w domain.WindowID, opts EmulateOptions) error

	// --- Batch B: recording. `body` runs the wrapped steps mid-capture; the
	// closure encodes the captured frames to GIF (or MP4 via ffmpeg). ---
	Record func(ctx context.Context, w domain.WindowID, opts ScreencastOptions, body func() error) (RecordResult, error)

	// --- Batch C: network & session. The capture closures run `body`
	// (the wrapped `do:` steps) with a CDP listener attached. ---
	CaptureNetwork     func(ctx context.Context, w domain.WindowID, opts NetCaptureOptions, body func() error) ([]map[string]any, error)
	CaptureConsole     func(ctx context.Context, w domain.WindowID, body func() error) ([]map[string]any, error)
	GetCookies         func(ctx context.Context, w domain.WindowID, urls []string) ([]Cookie, error)
	SetCookies         func(ctx context.Context, w domain.WindowID, cookies []Cookie) error
	WaitForNetworkIdle func(ctx context.Context, w domain.WindowID, idleMs, timeoutMs int64, maxInflight int) error

	// DownloadEach native-clicks every match of `selector` in DOM order. For
	// each click, polls the per-window download dir for up to `perFile` and
	// returns the path of any new file (empty string slot if it timed out).
	DownloadEach func(ctx context.Context, w domain.WindowID, selector string, perFile time.Duration) ([]string, error)

	// SaveCapturesToDir drains `window.__webtasks_captures` (set up by an
	// in-page URL.createObjectURL hook) and writes each ready blob to
	// `<dir>/<basename>` using the supplied naming template. Pending entries
	// stay in the buffer for a future drain.
	SaveCapturesToDir func(ctx context.Context, w domain.WindowID, dir, naming string) ([]CapturedFile, error)
}

// CapturedFile is what SaveCapturesToDir returns per drained capture.
type CapturedFile struct {
	Path     string `json:"path"`
	Basename string `json:"basename"`
	Size     int    `json:"size"`
	Mime     string `json:"mime,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
}

// PoolStatus is the snapshot the lease publishes via `Status`.
type PoolStatus struct {
	Size int `json:"size"`
	Free int `json:"free"`
	Busy int `json:"busy"`
}

// WindowLease leases windows from named pools. Goroutine-safe by contract.
type WindowLease struct {
	Acquire func(tag domain.PoolTag, timeout time.Duration) (domain.WindowID, error)
	Release func(w domain.WindowID)
	Status  func() map[string]PoolStatus
	Recover func(w domain.WindowID) error // replace the driver behind a crashed window
}

// TaskRegistry hands out task definitions loaded from the bundle.
type TaskRegistry struct {
	List func() []domain.TaskDef
	Get  func(name string) (domain.TaskDef, bool)
}

// HTMLExtraction converts an HTML string + spec into typed JSON.
type HTMLExtraction struct {
	ExtractObject func(html string, spec domain.ExtractSpec) (map[string]any, error)
	ExtractList   func(html string, spec domain.ExtractSpec) ([]map[string]any, error)
}

// Templating substitutes `{{name}}` tokens in a string using the supplied
// variable map, with `{{name|or:default}}` fallback and JVM-style system
// property fallback.
type Templating struct {
	Substitute func(template string, vars map[string]any) string
}

// EventPublisher hands progress events to whatever transport is active.
type EventPublisher struct {
	Emit func(e domain.Event)
}

// Noop returns an EventPublisher that drops every event — used under sync REST.
func NoopEvents() EventPublisher { return EventPublisher{Emit: func(domain.Event) {}} }

// JsScripts looks up named JS modules from the bundle.
type JsScripts struct {
	Get func(name string) (string, bool)
}

// PDFOptions configures page.PrintToPDF (the `pdf` / `html-to-pdf` actions).
// Zero values mean "Chrome default".
type PDFOptions struct {
	Landscape           bool
	PrintBackground     bool
	Scale               float64
	PaperWidth          float64
	PaperHeight         float64
	MarginTop           float64
	MarginBottom        float64
	MarginLeft          float64
	MarginRight         float64
	PageRanges          string
	DisplayHeaderFooter bool
	HeaderTemplate      string
	FooterTemplate      string
}

// ScreenshotOptions configures the `screenshot` action.
type ScreenshotOptions struct {
	Selector string // "" or "." → viewport / full page
	FullPage bool
	Format   string // "png" (default) | "jpeg"
	Quality  int    // jpeg only, 0-100
}

// EmulateOptions configures the `emulate` action (device metrics + media).
type EmulateOptions struct {
	Width             int64
	Height            int64
	DeviceScaleFactor float64
	Mobile            bool
	ColorScheme       string // "light" | "dark" | "no-preference" | ""
	Reset             bool   // clear all overrides
}

// ScreencastOptions configures the `record` action.
type ScreencastOptions struct {
	Format        string // "gif" (default) | "mp4"
	FPS           int
	Quality       int
	EveryNthFrame int
	MaxFrames     int
	MaxDurationMs int64
}

// RecordResult is what Record returns: the encoded media plus metadata.
type RecordResult struct {
	Data       []byte
	Frames     int
	DurationMs int64
}

// NetCaptureOptions configures the `capture-network` action.
type NetCaptureOptions struct {
	IncludeBodies bool
	MaxBodyBytes  int
	URLFilter     string
}

// Cookie is one browser cookie (get-cookies / set-cookies).
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain,omitempty"`
	Path     string  `json:"path,omitempty"`
	Expires  float64 `json:"expires,omitempty"`
	HTTPOnly bool    `json:"httpOnly,omitempty"`
	Secure   bool    `json:"secure,omitempty"`
	SameSite string  `json:"sameSite,omitempty"`
	URL      string  `json:"url,omitempty"`
}

// HTTPRequest / HTTPResponse are the shapes the `http-request` action uses.
type HTTPRequest struct {
	Method     string
	URL        string
	Headers    map[string]string
	Body       []byte
	TimeoutMs  int64
	NoRedirect bool
}

type HTTPResponse struct {
	Status  int
	Headers map[string]string
	Body    string
}

// HTTPClient is the outbound-HTTP capability used by the `http-request`
// action. The orchestrator builds it from the httpclient infra adapter.
type HTTPClient struct {
	Do func(ctx context.Context, req HTTPRequest) (HTTPResponse, error)
}
