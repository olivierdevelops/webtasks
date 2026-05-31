package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"webtasks/internal/domain"
	"webtasks/internal/features"
	"webtasks/internal/infra/bundle"
	chromedpx "webtasks/internal/infra/chromedp"
	"webtasks/internal/infra/console"
	"webtasks/internal/infra/goqueryx"
	"webtasks/internal/infra/httpclient"
	"webtasks/internal/infra/httpserver"
	"webtasks/internal/infra/yamlreader"
	"webtasks/internal/io/rest"
	"webtasks/internal/io/staticmounts"
	orchFeatures "webtasks/internal/orchestrator/features"
	orchUC "webtasks/internal/orchestrator/usecases"
)

// Run boots the server and blocks until SIGINT/SIGTERM. `args` are forwarded
// to the secrets loader so `--NAME=value` can satisfy declared secrets.
func Run(cfg Config, args []string) error {
	fmt.Printf("[webtasks] starting on %s:%d\n", cfg.Host, cfg.Port)

	// Bundle (dir or zip) carries every YAML/JS/config file. The code-only
	// binary would never ship one of these.
	b, err := bundle.Open(cfg.BundlePath)
	if err != nil {
		return fmt.Errorf("open bundle %q: %w", cfg.BundlePath, err)
	}
	defer b.Close()
	fmt.Printf("[webtasks] bundle: %s (%s)\n", b.Source(), b.Kind())

	// Resolve declared secrets early so templating sees them. Published as env
	// vars (which the templating layer's `osLookup` falls back to).
	prompter := console.Prompter{}
	resolved, err := LoadSecrets(b, prompter, args)
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}
	if len(resolved) > 0 {
		fmt.Println("[webtasks] secrets resolved:")
		for _, r := range resolved {
			tag := ""
			if r.Sensitive {
				tag = ", value hidden"
			}
			fmt.Printf("  - %s (from %s%s)\n", r.Name, r.Source, tag)
		}
	}

	// Load static-mounts.yaml + pool config from the bundle.
	mounts, _ := loadMounts(b)
	poolSizes, poolProfiles, _ := loadPoolSizes(b)
	if _, hasDefault := poolSizes["default"]; !hasDefault {
		poolSizes["default"] = 1
	}

	// Persistent Chrome profiles (for pools marked `persistent:` in pool.yaml)
	// live here, surviving restarts. Override with WEBTASKS_PROFILE_DIR.
	profileRoot := os.Getenv("WEBTASKS_PROFILE_DIR")
	if profileRoot == "" {
		if home, err := os.UserHomeDir(); err == nil {
			profileRoot = filepath.Join(home, ".webtasks", "profiles")
		}
	}

	// Infra adapters.
	src := chromedpx.NewWindowSource(cfg.Headless, cfg.DownloadDir, profileRoot)
	prim := chromedpx.Primitives{}
	gq := goqueryx.Extractor{}

	// Features.
	tpl := orchFeatures.MakeTemplating()
	browser := orchFeatures.MakeBrowserActions(prim, src)
	extract := orchFeatures.MakeHTMLExtraction(gq)
	registry, err := orchFeatures.MakeTaskRegistry(b, true)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	lease, err := orchFeatures.MakeWindowLease(poolSizes, poolProfiles, src)
	if err != nil {
		return fmt.Errorf("build window lease: %w", err)
	}
	scripts := features.JsScripts{Get: func(name string) (string, bool) {
		if !hasExt(name, ".js") {
			name = name + ".js"
		}
		data, err := b.ReadFile("scripts/" + name)
		if err != nil {
			return "", false
		}
		return string(data), true
	}}

	// Use cases.
	httpc := orchFeatures.MakeHTTPClient(httpclient.Client{})
	run := orchUC.NewRunRegisteredTask(registry, lease, browser, extract, tpl, scripts, httpc)
	list := orchUC.NewListTasks(registry)
	hlth := orchUC.NewHealth(registry, lease)

	fmt.Println("[webtasks] registered tasks:")
	for _, t := range registry.List() {
		fmt.Printf("  - %s (pool=%s)\n", t.Name, t.PoolTag)
	}

	// HTTP server.
	hs := httpserver.New(cfg.Host, cfg.Port)
	uc := rest.UseCases{
		RunTask: func(name string, vals domain.InputValues) (domain.Output, error) {
			return run.Execute(context.Background(), name, vals, features.NoopEvents())
		},
		StreamTask: func(ctx context.Context, name string, vals domain.InputValues, events features.EventPublisher) (domain.Output, error) {
			return run.Execute(ctx, name, vals, events)
		},
		ListTasks: list.Execute,
		Health:    hlth.Execute,
	}
	if err := hs.Start(func(r chi.Router) {
		rest.Register(r, uc)
		staticmounts.Register(r, mounts)
	}); err != nil {
		return err
	}
	fmt.Printf("[webtasks] listening on http://%s:%d\n", cfg.Host, cfg.Port)

	// Block until signal.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("[webtasks] shutting down...")
	_ = hs.Stop(5 * time.Second)
	src.CloseAll()
	return nil
}

func hasExt(name, ext string) bool {
	if len(name) < len(ext) {
		return false
	}
	return name[len(name)-len(ext):] == ext
}

// loadMounts reads <bundle>/static-mounts.yaml. Missing file is fine.
func loadMounts(b *bundle.Root) ([]domain.StaticMount, error) {
	if !b.Exists("static-mounts.yaml") {
		return nil, nil
	}
	data, err := b.ReadFile("static-mounts.yaml")
	if err != nil {
		return nil, err
	}
	var raw struct {
		Mounts []domain.StaticMount `yaml:"mounts"`
	}
	if err := yamlreader.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	// Expand ${ENV} / ${ENV:-default} in `dir`.
	for i := range raw.Mounts {
		raw.Mounts[i].Dir = expandEnv(raw.Mounts[i].Dir)
	}
	return raw.Mounts, nil
}

// loadPoolSizes reads tasks/pool.yaml. It returns each pool's size and, for
// pools marked `persistent: true`, the profile name to back them with.
func loadPoolSizes(b *bundle.Root) (map[string]int, map[string]string, error) {
	sizes := map[string]int{}
	profiles := map[string]string{}
	if !b.Exists("tasks/pool.yaml") {
		return sizes, profiles, nil
	}
	data, err := b.ReadFile("tasks/pool.yaml")
	if err != nil {
		return nil, nil, err
	}
	var raw struct {
		Pools map[string]struct {
			Size       int    `yaml:"size"`
			Persistent bool   `yaml:"persistent"`
			Profile    string `yaml:"profile"`
		} `yaml:"pools"`
	}
	if err := yamlreader.Unmarshal(data, &raw); err != nil {
		return nil, nil, err
	}
	for name, p := range raw.Pools {
		sizes[name] = p.Size
		if p.Persistent {
			prof := p.Profile
			if prof == "" {
				prof = name
			}
			profiles[name] = prof
		}
	}
	return sizes, profiles, nil
}
