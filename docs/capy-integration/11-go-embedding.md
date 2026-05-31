# 11. Go embedding

Reference implementation for `infra/capyx` using
[github.com/olivierdevelops/capy](https://github.com/olivierdevelops/capy).

---

## Install

```bash
go install github.com/olivierdevelops/capy/cmd/capy@main   # @latest may fail on module path; use @main
go get github.com/olivierdevelops/capy@main
```

Pin to a tagged release when integrating (Capy is pre-1.0).

---

## capyx package

```go
// internal/infra/capyx/transpiler.go
package capyx

import (
    "fmt"
    "sync"

    "github.com/olivierdevelops/capy"
)

type Transpiler struct {
    lib *capy.Library
    mu  sync.RWMutex
    cache map[string]string // optional: hash(source) → yaml
}

func NewFromBundle(b bundle.Loader, libRelPath string) (*Transpiler, error) {
    src, err := b.ReadFile(libRelPath)
    if err != nil {
        return nil, fmt.Errorf("capy library %q: %w", libRelPath, err)
    }
    lib, err := capy.NewLibrary(string(src))
    if err != nil {
        return nil, fmt.Errorf("compile capy library: %w", err)
    }
    return &Transpiler{lib: lib, cache: map[string]string{}}, nil
}

func (t *Transpiler) Transpile(taskSource string) (string, error) {
    t.mu.RLock()
    lib := t.lib
    t.mu.RUnlock()
    out, err := lib.Run(taskSource)
    if err != nil {
        return "", err
    }
    return out, nil
}

func (t *Transpiler) Reload(librarySrc string) error {
    lib, err := capy.NewLibrary(librarySrc)
    if err != nil {
        return err
    }
    t.mu.Lock()
    t.lib = lib
    t.cache = map[string]string{}
    t.mu.Unlock()
    return nil
}

func (t *Transpiler) Introspect() []capy.FunctionInfo {
    return t.lib.Introspect()
}
```

---

## Orchestrator wiring

```go
// internal/orchestrator/features/makes.go (sketch)

func MakeTaskRegistry(b *bundle.Root, hot bool) (features.TaskRegistry, error) {
    var tr *capyx.Transpiler
    if lp := b.CapyLibraryPath(); lp != "" {
        var err error
        tr, err = capyx.NewFromBundle(b, lp)
        if err != nil {
            return features.TaskRegistry{}, err
        }
    }
    load := func() ([]domain.TaskDef, map[string]domain.TaskDef, error) {
        return loadAllTasks(b, tr)
    }
    // ... same hot/cold split as today
}
```

---

## Embedded library (no bundle file)

For single-binary demos, embed grammar:

```go
//go:embed capy/webtasks.capy
var webtasksLib string

lib, _ := capy.NewLibrary(webtasksLib)
```

Users still author `.capy` task files in the bundle; only the grammar ships in the binary.

---

## Introspection endpoint (optional)

Expose grammar for editors:

```go
// GET /capy/introspect → JSON FunctionInfo[]
func handleCapyIntrospect(tr *capyx.Transpiler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(tr.Introspect())
    }
}
```

Build VS Code completion from this JSON — same data as Capy playground wasm.

---

## Testing

```go
func TestTranspileBasicsTitle(t *testing.T) {
    lib, err := capy.NewLibraryFromFile("testdata/webtasks.capy")
    require.NoError(t, err)
    src, _ := os.ReadFile("testdata/basics-title.capy")
    out, err := lib.Run(string(src))
    require.NoError(t, err)
    var d domain.TaskDef
    require.NoError(t, yaml.Unmarshal([]byte(out), &d))
    assert.Equal(t, "basics/title", d.Name)
    assert.Len(t, d.Flow, 2)
}
```

---

Next: [Migration strategy →](12-migration.md)
