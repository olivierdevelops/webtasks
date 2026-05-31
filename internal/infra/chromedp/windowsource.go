// Package chromedp wraps the raw chromedp Allocator/Context lifecycle. One
// pooled window = one chromedp.AllocatorContext + child Context per pool slot.
package chromedp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

// filteredErrorf is the WithErrorf hook for every chromedp browser context.
// It drops the harmless cdproto-version-mismatch warnings (chromedp ships an
// older cdproto than Chrome 145 emits) while letting real errors through.
func filteredErrorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(msg, "could not unmarshal event: unknown ") {
		return
	}
	log.Print(msg)
}

// WindowSource owns the Chrome lifecycle for every window. Concurrent
// safe.
type WindowSource struct {
	mu           sync.Mutex
	headless     bool
	downloads    string
	profileRoot  string
	drivers      map[string]*window
	seq          atomic.Uint64
	parentCtx    context.Context
	parentCancel context.CancelFunc
}

type window struct {
	id            string
	allocCtx      context.Context
	allocCancel   context.CancelFunc
	browserCtx    context.Context
	browserCancel context.CancelFunc
	downloadDir   string
	profileDir    string
}

// NewWindowSource creates a factory. `downloadsRoot` will receive a sub-dir
// per window; `profileRoot` holds persistent Chrome profiles (see
// CreatePersistent). Missing directories are created as needed.
func NewWindowSource(headless bool, downloadsRoot, profileRoot string) *WindowSource {
	parent, cancel := context.WithCancel(context.Background())
	return &WindowSource{
		headless:     headless,
		downloads:    downloadsRoot,
		profileRoot:  profileRoot,
		drivers:      make(map[string]*window),
		parentCtx:    parent,
		parentCancel: cancel,
	}
}

// Create allocates a new window with an ephemeral profile (wiped on restart)
// and returns its id.
func (s *WindowSource) Create() (string, error) {
	return s.create("")
}

// CreatePersistent allocates a window backed by a stable, non-temporary
// profile directory under profileRoot/profile-<name>. A one-time interactive
// login persists across runs and server restarts. Persistent pools should be
// size 1 — two live Chrome processes cannot share one profile directory.
func (s *WindowSource) CreatePersistent(name string) (string, error) {
	return s.create(name)
}

func (s *WindowSource) create(profile string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("w%d", s.seq.Add(1))
	dir := filepath.Join(s.downloads, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	profileDir := filepath.Join(os.TempDir(), "webtasks-userdata-"+id)
	if profile != "" {
		root := s.profileRoot
		if root == "" {
			root = filepath.Join(os.TempDir(), "webtasks-profiles")
		}
		profileDir = filepath.Join(root, "profile-"+profile)
		if err := os.MkdirAll(profileDir, 0o755); err != nil {
			return "", err
		}
	}
	w, err := s.spawn(id, dir, profileDir)
	if err != nil {
		return "", err
	}
	s.drivers[id] = w
	return id, nil
}

func (s *WindowSource) spawn(id, dir, profileDir string) (*window, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.WindowSize(1280, 900),
		chromedp.UserDataDir(profileDir),
	)
	if !s.headless {
		opts = append(opts, chromedp.Flag("headless", false))
	} else {
		opts = append(opts, chromedp.Flag("headless", "new"))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(s.parentCtx, opts...)
	// chromedp's bundled cdproto is older than Chrome 145, so it logs benign
	// "could not unmarshal event: unknown <Enum> value" lines for new CDP
	// enum values (PrivateNetworkRequestPolicy, ClientNavigationReason, etc.).
	// Drop those at source so the operator log isn't flooded; still pass
	// genuine errors through.
	browserCtx, browserCancel := chromedp.NewContext(allocCtx,
		chromedp.WithErrorf(filteredErrorf),
		chromedp.WithLogf(func(string, ...any) {}),
		chromedp.WithDebugf(func(string, ...any) {}),
	)

	// Force Chrome to launch + enable downloads to the per-window directory.
	// Headless Chrome blocks downloads unless we tell it explicitly via CDP.
	if err := chromedp.Run(browserCtx, browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
		WithDownloadPath(dir).
		WithEventsEnabled(true),
	); err != nil {
		browserCancel()
		allocCancel()
		return nil, fmt.Errorf("chromedp launch: %w", err)
	}
	w := &window{
		id:            id,
		allocCtx:      allocCtx,
		allocCancel:   allocCancel,
		browserCtx:    browserCtx,
		browserCancel: browserCancel,
		downloadDir:   dir,
		profileDir:    profileDir,
	}
	return w, nil
}

// Context returns the chromedp Context for a window so primitives can drive it.
// The returned context is parented under the browser context so cancellation
// at shutdown propagates correctly.
func (s *WindowSource) Context(id string) (context.Context, error) {
	s.mu.Lock()
	w, ok := s.drivers[id]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown window: %s", id)
	}
	return w.browserCtx, nil
}

// DownloadDir returns the per-window download directory.
func (s *WindowSource) DownloadDir(id string) (string, error) {
	s.mu.Lock()
	w, ok := s.drivers[id]
	s.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("unknown window: %s", id)
	}
	return w.downloadDir, nil
}

// Replace tears down a crashed window and creates a fresh one under the same
// id. Used by the pool's `Recover`.
func (s *WindowSource) Replace(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	profileDir := filepath.Join(os.TempDir(), "webtasks-userdata-"+id)
	old, ok := s.drivers[id]
	if ok {
		if old.profileDir != "" {
			profileDir = old.profileDir
		}
		old.browserCancel()
		old.allocCancel()
		delete(s.drivers, id)
	}
	dir := filepath.Join(s.downloads, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	w, err := s.spawn(id, dir, profileDir)
	if err != nil {
		return err
	}
	s.drivers[id] = w
	return nil
}

// CloseAll shuts every window down. Safe to call multiple times.
func (s *WindowSource) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, w := range s.drivers {
		w.browserCancel()
		w.allocCancel()
		delete(s.drivers, id)
	}
	s.parentCancel()
}

// WaitReady is a tiny helper that ensures the context is still healthy.
func (s *WindowSource) WaitReady(id string, timeout time.Duration) error {
	ctx, err := s.Context(id)
	if err != nil {
		return err
	}
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return chromedp.Run(tctx)
}
