# 11. Go embedding

Reference implementation for `infra/capyx` using
[Capy v0.20.0](https://github.com/olivierdevelops/capy) (`github.com/olivierdevelops/capy`).

See also [v0.20 overview](00-v020-overview.md) for the full API surface.

---

## Dependency

```bash
# v0.20.0 API — use @main if @v0.20.0 tag reports a module-path mismatch
go install github.com/olivierdevelops/capy/cmd/capy@main
go get github.com/olivierdevelops/capy@main
```

When the tagged release resolves cleanly:

```bash
go install github.com/olivierdevelops/capy/cmd/capy@v0.20.0
go get github.com/olivierdevelops/capy@v0.20.0
```

Pin explicitly in `go.mod`. Capy is pre-1.0 — check
[CHANGELOG](https://github.com/olivierdevelops/capy/blob/main/CHANGELOG.md) when bumping.

For local Capy development, a `replace` directive in `go.mod` can point at a
checkout (remove before release).

---

## capyx package (v0.20.0)

```go
// internal/infra/capyx/transpiler.go
package capyx

import (
    "fmt"
    "sync"

    "github.com/olivierdevelops/capy"
)

type Transpiler struct {
    lib   *capy.Library
    mu    sync.RWMutex
    cache map[string]map[string]string // source hash → path → contents
}

func NewFromBundle(readFile func(string) ([]byte, error), libRelPath string) (*Transpiler, error) {
    src, err := readFile(libRelPath)
    if err != nil {
        return nil, fmt.Errorf("capy library %q: %w", libRelPath, err)
    }
    lib, err := capy.NewLibrary(string(src))
    if err != nil {
        return nil, fmt.Errorf("compile capy library: %w", err)
    }
    // Default NoOpHost — safe for user/AI-authored task scripts.
    // lib.SetHost(capyinfra.OSHost{}) // trusted libraries only
    return &Transpiler{lib: lib, cache: map[string]map[string]string{}}, nil
}

// Transpile returns all files from RunMulti. For single-output libraries the
// primary string is stored under lib.OutputFile().
func (t *Transpiler) Transpile(taskSource string) (map[string]string, error) {
    t.mu.RLock()
    lib := t.lib
    t.mu.RUnlock()

    primary, files, err := lib.RunMulti(taskSource)
    if err != nil {
        return nil, err
    }
    if files == nil {
        files = map[string]string{}
    }
    if primary != "" {
        if out := lib.OutputFile(); out != "" {
            files[out] = primary
        }
    }
    return files, nil
}

// TaskYAML returns the main task definition bytes (first .yaml in the map, or
// the declared output_file).
func (t *Transpiler) TaskYAML(files map[string]string, lib *capy.Library) ([]byte, error) {
    if out := t.lib.OutputFile(); out != "" {
        if s, ok := files[out]; ok {
            return []byte(s), nil
        }
    }
    for path, content := range files {
        if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
            return []byte(content), nil
        }
    }
    return nil, fmt.Errorf("no YAML output in transpile result")
}

func (t *Transpiler) Reload(librarySrc string) error {
    lib, err := capy.NewLibrary(librarySrc)
    if err != nil {
        return err
    }
    t.mu.Lock()
    t.lib = lib
    t.cache = map[string]map[string]string{}
    t.mu.Unlock()
    return nil
}

func (t *Transpiler) Introspect() []capy.FunctionInfo { return t.lib.Introspect() }
func (t *Transpiler) DocsMarkdown() string            { return capy.RenderLibraryDocs(t.lib) }
func (t *Transpiler) Extension() string               { return t.lib.Extension() }
```

`infra/` imports Capy; only `orchestrator/` wires it into `MakeTaskRegistry`.

---

## Orchestrator wiring

```go
func MakeTaskRegistry(b *bundle.Root, hot bool) (features.TaskRegistry, error) {
    var tr *capyx.Transpiler
    if lp := b.CapyLibraryPath(); lp != "" {
        var err error
        tr, err = capyx.NewFromBundle(b.ReadFile, lp)
        if err != nil {
            return features.TaskRegistry{}, err
        }
    }
    load := func() ([]domain.TaskDef, map[string]domain.TaskDef, error) {
        return loadAllTasks(b, tr) // WalkCapy → Transpile → yaml.Unmarshal
    }
    // ... hot/cold split unchanged
}
```

---

## Host sandboxing

| Host | When to use |
|---|---|
| Default `NoOpHost` | User tasks, AI-generated source, CI |
| `capyinfra.OSHost{}` | First-party `webtasks.capy` that reads version from env |

Never set `OSHost` on the task-script transpile path unless the library source is
fully trusted.

---

## Introspection & docs endpoints (optional)

```go
// GET /capy/introspect  → lib.Introspect() JSON
// GET /capy/docs        → capy.RenderLibraryDocs(lib) text/markdown
```

Same data powers MCP (`capy-mcp`) and the WASM playground.

---

## Testing

```go
func TestTranspileBasicsTitle(t *testing.T) {
    lib, err := capy.NewLibraryFromFile("testdata/webtasks.capy")
    require.NoError(t, err)
    src, _ := os.ReadFile("testdata/basics-title.capy")
    primary, files, err := lib.RunMulti(string(src))
    require.NoError(t, err)
    yaml := primary
    if yaml == "" {
        yaml = files["tasks/basics/title.yaml"]
    }
    var d domain.TaskDef
    require.NoError(t, yamlv3.Unmarshal([]byte(yaml), &d))
    assert.Equal(t, "basics/title", d.Name)
}
```

---

Next: [Migration strategy →](12-migration.md)
