# 9. Bundle loader changes

Concrete changes to `internal/infra/bundle` and `MakeTaskRegistry`.

---

## WalkCapy

Mirror `WalkYAML`:

```go
// WalkCapy invokes fn for every *.capy file under dir.
func (r *Root) WalkCapy(dir string, fn func(relPath string, content []byte) error) error {
    if !r.Exists(dir) {
        return nil
    }
    return fs.WalkDir(r.fsys, dir, func(p string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        if !strings.HasSuffix(strings.ToLower(p), ".capy") {
            return nil
        }
        content, err := fs.ReadFile(r.fsys, p)
        if err != nil {
            return err
        }
        return fn(p, content)
    })
}
```

Skip `pool.capy` if you ever move pool config — today `pool.yaml` stays YAML.

---

## Library discovery

```go
func (r *Root) CapyLibraryPath() string {
    if p := os.Getenv("WEBTASKS_CAPY_LIB"); p != "" {
        return p
    }
    if r.Exists("capy/webtasks.capy") {
        return "capy/webtasks.capy"
    }
    return ""
}
```

For zip bundles, library must live inside the archive at `capy/webtasks.capy`.

---

## Registry load order

```go
func loadTasks(b *bundle.Root, lib *capyx.Transpiler) ([]domain.TaskDef, map[string]domain.TaskDef, error) {
    byName := map[string]domain.TaskDef{}
    capySources := map[string]struct{}{}

    // 1. Capy tasks
    err := b.WalkCapy("tasks", func(rel string, src []byte) error {
        yaml, err := lib.Transpile(string(src))
        if err != nil {
            return fmt.Errorf("%s: %w", rel, err)
        }
        var d domain.TaskDef
        if err := yamlreader.Unmarshal([]byte(yaml), &d); err != nil {
            return fmt.Errorf("%s (transpiled): %w", rel, err)
        }
        byName[d.Name] = d
        capySources[d.Name] = struct{}{}
        return nil
    })
    if err != nil {
        return nil, nil, err
    }

    // 2. YAML tasks (skip if name already from Capy)
    err = b.WalkYAML("tasks", func(rel string, content []byte) error {
        if path.Base(rel) == "pool.yaml" {
            return nil
        }
        var d domain.TaskDef
        if err := yamlreader.Unmarshal(content, &d); err != nil {
            return fmt.Errorf("%s: %w", rel, err)
        }
        if _, ok := capySources[d.Name]; ok {
            return nil // Capy wins
        }
        byName[d.Name] = d
        return nil
    })
    // ...
}
```

---

## Hot-reload

When `hot: true`, re-transpile `.capy` on every `Get(name)` — same cost profile
as re-reading YAML. Cache by `(relPath, modTime, contentHash)` inside
`capyx.Transpiler` to avoid re-parsing unchanged files.

---

## Bundle layout (final)

```
my-bundle/
├── capy/
│   └── webtasks.capy
├── tasks/
│   ├── pool.yaml
│   ├── basics/
│   │   └── title.capy
│   └── crawl/
│       └── hackernews-top.capy
├── scripts/
│   └── demo/*.js
├── secrets.yaml
└── static-mounts.yaml
```

Packaging: include `capy/` in `executor bundle` zip output.

---

Next: [Transpilation pipeline →](10-transpilation-pipeline.md)
