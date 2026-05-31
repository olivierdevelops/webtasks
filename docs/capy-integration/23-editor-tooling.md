# 23. Editor tooling (v0.20.0)

Leverage Capy's v0.20 CLI, introspection, MCP, and WASM for webtasks task authoring.

---

## Auto-generated reference

```bash
go install github.com/olivierdevelops/capy/cmd/capy@v0.20.0
capy docs capy/webtasks.capy > docs/capy-integration/DSL_REFERENCE.generated.md
```

Or from Go:

```go
md := capy.RenderLibraryDocs(lib)
```

Regenerate when `webtasks.capy` changes — never maintain a parallel verb list.

---

## Introspection API

```go
for _, fn := range lib.Introspect() {
    // fn.Name, fn.Description, fn.Block, fn.Priority
    for _, a := range fn.Args {
        // a.Kind ("literal"|"capture"), a.Type, a.Default, a.Optional
    }
}
names := lib.FunctionNames()
markers := lib.CommentMarkers()   // e.g. ["#"]
```

Build editor features:

- Autocomplete for `goto`, `extract`, `pool`, …
- Hover docs from `fn.Description` and per-arg docs
- Syntax highlighting using `CommentMarkers()`

Optional webtasks endpoints: `GET /capy/introspect`, `GET /capy/docs`.

---

## CLI dev loop

| Command | Use |
|---|---|
| `capy watch capy/webtasks.capy tasks/foo.capy` | Re-transpile on save (250ms poll) |
| `capy fmt tasks/**/*.capy` | Format task sources |
| `capy fmt … --check` | CI formatting gate |
| `capy check …` | Parse + validate before commit |

After Go integration, `executor server` hot-reloads transpiled tasks like YAML.

---

## WASM playground

Capy ships `cmd/capy-wasm` — same compiler in a webview:

[olivierdevelops.github.io/capy/playground/](https://olivierdevelops.github.io/capy/playground/)

Load `webtasks.capy` + a task script for live YAML preview (cannot diverge from CLI).

---

## MCP server

```bash
go install github.com/olivierdevelops/capy/cmd/capy-mcp@v0.20.0
```

Agents introspect/check/transpile over MCP. See
[mcp.md](https://github.com/olivierdevelops/capy/blob/main/docs/mcp.md).

---

## Standalone transpiler binary

Ship a grammar-specific CLI without requiring Capy on the host:

```bash
capy build capy/webtasks.capy -o webtasks-capy
./webtasks-capy run tasks/basics/title.capy
```

Cross-compile with `GOOS`/`GOARCH` for CI workers.

---

Next: [Testing and CI →](24-testing-ci.md)
