# Capy v0.20.0 — integration reference for webtasks

> **Version:** targets **Capy `v0.20.0`** (module
> `github.com/olivierdevelops/capy`). Capy is pre-1.0 — the library `.capy`
> schema may change between minor versions; check
> [Capy's CHANGELOG](https://github.com/olivierdevelops/capy/blob/main/CHANGELOG.md)
> when bumping.

[**Capy**](https://github.com/olivierdevelops/capy) is a transpiler engine with
**zero default grammar**. You define a small source language in a `.capy`
*library*, and Capy turns scripts in *your* language into *any* textual output —
YAML, JSON, HTML, Go — one file or many, in a single pass. There is no separate
template language: the library is written in Capy's native syntax, and the
renderer walks the parsed AST directly.

This chapter is the **v0.20.0 integration reference** adapted for webtasks.
For the full webtasks-specific grammar proposal and samples, continue with
[chapter 1](01-why-replace-yaml.md).

---

## Why embed Capy in webtasks

A webtasks bundle is several artifacts that must agree on strings no compiler
checks today:

```
demo/tasks/crawl/hackernews-top.yaml   name + flow steps
demo/scripts/demo/page-stats.js        fn: path referenced from YAML
tasks/pool.yaml                        poolTag must exist here
```

Rename a task slug in YAML but not the `call` target in another file, or typo
an action keyword (`gotoo`), and you only find out when Chrome runs. Capy lets
you declare a task **once** in a domain vocabulary (`task`, `goto`, `extract`)
and project it into the YAML `TaskDef` the server already executes — with
cross-references and action names enforced at transpile time.

> Capy never replaces chromedp or the flow interpreter. It sits *in front of*
> them as a source-generation step.

---

## Embed in Go — the v0.20.0 API

Capy is **pure Go, no CGo**:

```go
import "github.com/olivierdevelops/capy"

lib, err := capy.NewLibrary(librarySrc)        // or NewLibraryFromFile("capy/webtasks.capy")

// Single output (one task YAML)
out, err := lib.Run(scriptSrc)

// Multi-file output (task YAML + sidecars)
primary, files, err := lib.RunMulti(scriptSrc) // map[path]contents
```

| Method | Returns |
|---|---|
| `lib.Extension()` | declared `extension` (e.g. `"yaml"`) |
| `lib.OutputFile()` | optional `output_file` for single-output libraries |
| `lib.FunctionNames()` | sorted function names |
| `lib.Introspect()` | `[]FunctionInfo` — args, types, block kind, docs |
| `lib.CommentMarkers()` | declared line-comment markers |
| `lib.SetHost(h)` | install `domain.Host` for `env` / `arg` / `read_file` |
| `capy.RenderLibraryDocs(lib)` | Markdown reference for your DSL |

`Library` is safe to reuse across many `Run`/`RunMulti` calls; each call gets a
fresh context. `Run` is re-entrant on a fixed library.

### Proposed `infra/capyx` helper (webtasks)

```go
package capyx

import "github.com/olivierdevelops/capy"

// TranspileTask compiles webtasks.capy and runs a task script. Returns generated
// files (path → contents). Default NoOpHost — safe for untrusted/AI source.
func TranspileTask(librarySrc, scriptSrc string) (map[string]string, error) {
    lib, err := capy.NewLibrary(librarySrc)
    if err != nil {
        return nil, err
    }
    // lib.SetHost(capyinfra.OSHost{}) // opt-in only for trusted libraries

    primary, files, err := lib.RunMulti(scriptSrc)
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
```

The orchestrator loads transpiled YAML into `domain.TaskDef` exactly as today.

---

## Library syntax (v0.20.0)

### Header

```capy
extension yaml

comments
    line "#"
end

context
    name       ""
    poolTag    "default"
    transports []
    timeoutMs  60000
    flow       []
end
```

### Functions — your grammar

```capy
function task
    arg literal "task"
    arg capture slug string
    block_closer end
    set context.name slug
end

function goto
    arg literal "goto"
    arg capture url string
    append context.flow { run: "goto", params: { url: url } }
end
```

**Argument directives (v0.20.0)**

| Directive | Meaning |
|---|---|
| `arg literal "x"` | fixed keyword at this position |
| `arg capture NAME TYPE` | typed hole |
| `arg capture NAME TYPE default "v"` | optional trailing arg |
| `block_closer end` | nested body closed by `end` |
| `block_verbatim end` | raw bytes (embedded JS snippets) |

**Capture types:** `string`, `int`, `bool`, `ident`, `dotted_ident`, `word`,
`tail`. String captures arrive quoted — use `${decoded …}` / `unquote` in
output blocks.

**Inner DSL verbs:** `set`, `append`, `prepend`, `if`/`else`, `for`, `while`,
`write`, `error`, `return`.

### Output blocks

```capy
file "tasks/basics/title.yaml"
    write `name: ${toQuoted context.name}
poolTag: ${toQuoted context.poolTag}
timeoutMs: ${context.timeoutMs}
transports: ${toJSON context.transports}
flow: ${toJSONIndent context.flow}
`
end
```

Or a single `file_template` when one output suffices. Template helpers include
`escapeHtml`, `toJSON`, `toJSONIndent`, `decoded`, `unquote`, `indent`,
`pascalCase`, `add`, `join`, and more — see
[Capy templates](https://github.com/olivierdevelops/capy/blob/main/docs/templates.md).

---

## One source → many files

`RunMulti` can emit a task YAML **and** companion files from one script:

```capy
task "js-modules/page-stats"
    pool default
    goto "{{url}}"
    js stats fn "demo/page-stats.js"
end
```

Possible outputs (library-defined):

```
page-stats.capy ──capy──▶ tasks/js-modules/page-stats.yaml
                          scripts/demo/page-stats.js   (from block_verbatim)
```

Shared handler names and `fn:` paths are stamped from one source — they cannot drift.

Scripts may also use **`define NAME … end`** metaprogramming blocks merged into
the library before evaluation. See
[metaprogramming.md](https://github.com/olivierdevelops/capy/blob/main/docs/metaprogramming.md).

---

## Grammar-as-contract

The parser **is** the contract:

1. **No orphan references** — `call "missing/task"` fails if the library
   validates task names against a registry or enum.
2. **No invalid actions** — `gotoo` is not a declared function shape.
3. **Shape agreement** — extract field kinds must match declared `FieldKind` options.

Errors include line/column. Use `capy check` in CI. See
[grammar-as-contract.md](https://github.com/olivierdevelops/capy/blob/main/docs/grammar-as-contract.md).

---

## Host sandboxing

```go
import (
    "github.com/olivierdevelops/capy"
    capyinfra "github.com/olivierdevelops/capy/infra"
)

lib, _ := capy.NewLibrary(src)
// Default: NoOpHost — env/arg/read_file blocked. Safe for AI-generated source.

lib.SetHost(capyinfra.OSHost{}) // opt-in for trusted first-party libraries only
```

webtasks codegen should keep **NoOpHost** when transpiling user/AI task sources.

---

## CLI & tooling (v0.20.0)

```bash
go install github.com/olivierdevelops/capy/cmd/capy@v0.20.0   # or @main if tag module path mismatches
```

| Command | Purpose |
|---|---|
| `capy run <lib> <script>` | transpile (legacy invocation) |
| `capy <lib> <command> [args…]` | library command dispatch |
| `capy check <lib> <script>` | parse + validate (CI gate) |
| `capy docs <lib>` | Markdown DSL reference |
| `capy fmt <files…>` | formatter (`--check` for CI) |
| `capy watch <lib> [args…]` | re-run on file changes |
| `capy lib add <url\|path>` | install library to `CAPY_LIBS` |
| `capy build <lib> [-o out]` | standalone binary with library baked in |

Also: **MCP server** (`cmd/capy-mcp`), **WASM** (`cmd/capy-wasm`) for in-browser
preview. See [cli.md](https://github.com/olivierdevelops/capy/blob/main/docs/cli.md),
[mcp.md](https://github.com/olivierdevelops/capy/blob/main/docs/mcp.md).

### webtasks dev loop

```bash
capy check capy/webtasks.capy
capy run capy/webtasks.capy tasks/basics/title.capy
capy watch capy/webtasks.capy tasks/basics/title.capy   # hot transpile while authoring
```

---

## AI agents

Typical loop (also available over MCP):

```
1. lib.Introspect()     → allowed verbs for this grammar
2. draft title.capy     → ~10 lines of DSL
3. capy check           → parse error? retry with caret message
4. lib.RunMulti(source) → tasks/…/*.yaml (deterministic, NoOpHost)
5. executor call …      → run against live Chrome
```

Token reduction: ~12 lines DSL → ~25 lines YAML → agent never emits YAML directly.
Parser-as-sandbox: tokens outside the grammar never reach the filesystem.

See [ai-agents.md](https://github.com/olivierdevelops/capy/blob/main/docs/ai-agents.md)
and [chapter 22](22-ai-authoring.md).

---

## Versioning

- Pin `github.com/olivierdevelops/capy v0.20.0` in `go.mod` when implementing
  `infra/capyx`.
- Pre-1.0: re-run `capy check` on `webtasks.capy` after every bump.
- Migration: [migration-guide.md](https://github.com/olivierdevelops/capy/blob/main/docs/migration-guide.md).

---

## Suggested rollout for webtasks

1. **Pilot** — `webtasks.capy` + golden YAML diff for `basics/title` (no Go changes).
2. **Contract checks** — enum `PoolTag`, `ActionName`; orphan `call` targets error.
3. **Embed** — `infra/capyx` + extended `MakeTaskRegistry` ([chapter 11](11-go-embedding.md)).
4. **Agent surface** — MCP introspect + check; `GET /capy/introspect` optional.
5. **Demo bundle** — convert `demo/tasks/**/*.yaml` → `.capy`.

Each step is independently shippable; the Chrome runtime never changes.

---

Next: [Why replace YAML? →](01-why-replace-yaml.md)
