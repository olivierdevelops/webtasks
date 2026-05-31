package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"webtasks/internal/domain"
	"webtasks/internal/features"
	"webtasks/internal/infra/capytranspile"
	chromedpx "webtasks/internal/infra/chromedp"
	"webtasks/internal/infra/goqueryx"
	"webtasks/internal/infra/httpclient"
	"webtasks/internal/infra/yamlreader"
	orchFeatures "webtasks/internal/orchestrator/features"
	orchUC "webtasks/internal/orchestrator/usecases"
)

// RunFile executes a single task file once and prints its output as JSON.
// `args` is the slice after the `run` subcommand:
//
//	webtasks run path/to/task.webtask [--input k=v ...] [--json '{...}']
//
// A `.webtask` recipe is transpiled to YAML via the bundled grammar; a `.yaml`
// task is used as-is. Progress events go to stderr so stdout stays clean JSON.
func RunFile(args []string) error {
	file, vals, err := parseRunArgs(args)
	if err != nil {
		return err
	}

	def, err := loadTaskFile(file)
	if err != nil {
		return err
	}
	if def.PoolTag == "" {
		def.PoolTag = "default"
	}

	headless := os.Getenv("WEBTASKS_HEADLESS") != "false"
	downloadDir := getenv("WEBTASKS_DOWNLOADS_DIR", filepath.Join(os.TempDir(), "webtasks-downloads"))
	profileRoot := os.Getenv("WEBTASKS_PROFILE_DIR")
	if profileRoot == "" {
		if home, herr := os.UserHomeDir(); herr == nil {
			profileRoot = filepath.Join(home, ".webtasks", "profiles")
		}
	}

	src := chromedpx.NewWindowSource(headless, downloadDir, profileRoot)
	defer src.CloseAll()

	tpl := orchFeatures.MakeTemplating()
	browser := orchFeatures.MakeBrowserActions(chromedpx.Primitives{}, src)
	extract := orchFeatures.MakeHTMLExtraction(goqueryx.Extractor{})
	httpc := orchFeatures.MakeHTTPClient(httpclient.Client{})

	registry := features.TaskRegistry{
		List: func() []domain.TaskDef { return []domain.TaskDef{def} },
		Get: func(name string) (domain.TaskDef, bool) {
			if name == def.Name {
				return def, true
			}
			return domain.TaskDef{}, false
		},
	}

	lease, err := orchFeatures.MakeWindowLease(map[string]int{string(def.PoolTag): 1}, nil, src)
	if err != nil {
		return fmt.Errorf("build window: %w", err)
	}

	// JS modules resolve from a `scripts/` directory. We look next to the task
	// file and one level up (the bundle root, when the recipe lives in tasks/).
	fileDir := filepath.Dir(file)
	scriptDirs := []string{
		filepath.Join(fileDir, "scripts"),
		filepath.Join(fileDir, "..", "scripts"),
	}
	scripts := features.JsScripts{Get: func(name string) (string, bool) {
		if !strings.HasSuffix(name, ".js") {
			name += ".js"
		}
		for _, dir := range scriptDirs {
			if data, rerr := os.ReadFile(filepath.Join(dir, name)); rerr == nil {
				return string(data), true
			}
		}
		return "", false
	}}

	run := orchUC.NewRunRegisteredTask(registry, lease, browser, extract, tpl, scripts, httpc)

	events := features.EventPublisher{Emit: func(e domain.Event) {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", e.Kind, e.Text)
	}}

	fmt.Fprintf(os.Stderr, "[webtasks] running %s (pool=%s, headless=%v)\n", def.Name, def.PoolTag, headless)
	out, err := run.Execute(context.Background(), def.Name, vals, events)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// loadTaskFile reads a task file, transpiling .webtask recipes to YAML first,
// then unmarshals into a TaskDef.
func loadTaskFile(file string) (domain.TaskDef, error) {
	var def domain.TaskDef
	if _, err := os.Stat(file); err != nil {
		return def, fmt.Errorf("read task %q: %w", file, err)
	}
	var raw []byte
	var err error
	if capytranspile.IsRecipe(file) {
		raw, err = capytranspile.ToYAML(file)
	} else {
		raw, err = os.ReadFile(file)
	}
	if err != nil {
		return def, err
	}
	if err := yamlreader.Unmarshal(raw, &def); err != nil {
		return def, fmt.Errorf("parse %q: %w", file, err)
	}
	if def.Name == "" {
		return def, fmt.Errorf("%q has no task name", file)
	}
	return def, nil
}

// parseRunArgs pulls the file path and inputs out of the `run` args.
func parseRunArgs(args []string) (string, domain.InputValues, error) {
	var file string
	vals := domain.InputValues{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--input" || a == "-i":
			i++
			if i >= len(args) {
				return "", nil, fmt.Errorf("--input needs k=v")
			}
			k, v, ok := strings.Cut(args[i], "=")
			if !ok {
				return "", nil, fmt.Errorf("--input expects k=v, got %q", args[i])
			}
			vals[k] = v
		case a == "--json" || a == "-j":
			i++
			if i >= len(args) {
				return "", nil, fmt.Errorf("--json needs a JSON object")
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(args[i]), &m); err != nil {
				return "", nil, fmt.Errorf("--json: %w", err)
			}
			for k, v := range m {
				vals[k] = v
			}
		case strings.HasPrefix(a, "-"):
			return "", nil, fmt.Errorf("unknown flag %q", a)
		default:
			if file != "" {
				return "", nil, fmt.Errorf("unexpected extra argument %q", a)
			}
			file = a
		}
	}
	if file == "" {
		return "", nil, fmt.Errorf("usage: webtasks run <file.webtask|file.yaml> [--input k=v] [--json '{...}']")
	}
	return file, vals, nil
}
