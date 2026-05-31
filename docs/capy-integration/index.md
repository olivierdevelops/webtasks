# Capy integration guide

Replace verbose YAML task definitions with a **typed, sandboxed DSL** powered by
[Capy](https://github.com/olivierdevelops/capy) — a configurable transpiler
where **you define the entire grammar** in a `.capy` library file.

This guide is the full integration blueprint for webtasks: architecture,
proposed grammar, Go embedding code, migration path, and **nine worked sample
projects** from a one-line task to a production Concio-style bundle.

---

## Why this exists

webtasks tasks today are YAML files under `tasks/**/*.yaml`. YAML works, but
authors repeatedly hit the same friction:

| YAML pain | Capy answer |
|---|---|
| Indentation errors silently change meaning | Grammar-defined blocks with explicit closers |
| No validation until runtime in Chrome | Library `type` blocks + `capy check` at author time |
| Copy-paste boilerplate (`poolTag`, `transports`, `flow:`) | `task … end` block accumulates metadata once |
| AI agents hallucinate action keywords | Grammar is the boundary — invalid actions fail at transpile |
| 40-line YAML for a 5-step flow | ~12 lines of DSL → expanded YAML by the engine |

Capy does **not** replace the webtasks runtime. It replaces the **authoring
surface** — the files in your bundle — and transpiles them into the same
`domain.TaskDef` the server already executes.

---

## Document map (26 chapters)

| # | Chapter | What you'll learn |
|---|---|---|
| 1 | [Why replace YAML?](01-why-replace-yaml.md) | Cost/benefit, coexistence model |
| 2 | [Capy primer](02-capy-primer.md) | Libraries, functions, context, inner DSL |
| 3 | [YAML ↔ Capy mapping](03-yaml-capy-mapping.md) | Field-by-field equivalence |
| 4 | [Proposed grammar overview](04-proposed-grammar.md) | Design goals and surface syntax |
| 5 | [The `webtasks.capy` library](05-webtasks-library.md) | Full library skeleton |
| 6 | [Action vocabulary in Capy](06-actions-grammar.md) | Every `run:` as a DSL function |
| 7 | [Types and validation](07-types-validation.md) | `ActionName`, `PoolTag`, enums |
| 8 | [Integration architecture](08-architecture.md) | Where Capy sits in VHCO |
| 9 | [Bundle loader changes](09-bundle-loader.md) | `WalkCapy`, hot-reload |
| 10 | [Transpilation pipeline](10-transpilation-pipeline.md) | `.capy` → YAML → `TaskDef` |
| 11 | [Go embedding](11-go-embedding.md) | `capy.NewLibrary`, registry hook |
| 12 | [Migration strategy](12-migration.md) | YAML + Capy in one bundle |
| 13 | [Sample 1 — hello](samples/01-hello.md) | Smallest task |
| 14 | [Sample 2 — HN crawl](samples/02-hackernews.md) | List extraction |
| 15 | [Sample 3 — search](samples/03-search.md) | Inputs + templating |
| 16 | [Sample 4 — form fill](samples/04-form-fill.md) | Interaction |
| 17 | [Sample 5 — SSE](samples/05-streaming.md) | Progress events |
| 18 | [Sample 6 — JS modules](samples/06-js-modules.md) | `fn:` references |
| 19 | [Sample 7 — control flow](samples/07-control-flow.md) | `call`, `loop` |
| 20 | [Sample 8 — rendering](samples/08-rendering.md) | PDF, screenshot |
| 21 | [Sample 9 — Concio bundle](samples/09-concio-bundle.md) | Production complexity |
| 22 | [AI authoring](22-ai-authoring.md) | Token compression, sandboxing |
| 23 | [Editor tooling](23-editor-tooling.md) | Introspect, MCP, playground |
| 24 | [Testing and CI](24-testing-ci.md) | `capy check`, golden files |
| 25 | [Troubleshooting](25-troubleshooting.md) | Common errors |
| 26 | [Roadmap](26-roadmap.md) | Phased rollout |

Reference artifacts live alongside this guide:

- [`grammar/webtasks.capy`](grammar/webtasks.capy) — proposed library (copy-paste ready)
- [`grammar/samples/*.capy`](grammar/samples/) — source scripts for each sample

---

## 30-second preview

**Today (YAML):**

```yaml
name: "basics/title"
poolTag: "default"
transports: ["rest"]
timeoutMs: 15000
flow:
  - run: goto
    params: { url: "https://example.com" }
  - run: extract
    as: page
    params:
      selector: "html"
      repeat: false
      fields:
        title: { kind: text, selector: "title" }
```

**Proposed (Capy DSL):**

```capy
task "basics/title"
    pool default
    timeout 15s
    transport rest

    goto "https://example.com"
    extract page from "html":
        title text "title"
end
```

**Transpiled YAML** (what the server loads — identical to hand-written YAML):

```yaml
name: "basics/title"
poolTag: "default"
transports: ["rest"]
timeoutMs: 15000
flow:
  - run: goto
    params: { url: "https://example.com" }
  - run: extract
    as: page
    params:
      selector: "html"
      repeat: false
      fields:
        title: { kind: text, selector: "title" }
```

---

## External links

- [Capy repository](https://github.com/olivierdevelops/capy)
- [Capy library authoring](https://github.com/olivierdevelops/capy/blob/main/docs/library-authoring.md)
- [Capy embedding in Go](https://github.com/olivierdevelops/capy/blob/main/docs/embedding.md)
- [Capy for LLMs](https://github.com/olivierdevelops/capy/blob/main/docs/CAPY_FOR_LLMS.md)
- [webtasks task definition](../task-definition.md)
- [webtasks actions reference](../actions.md)
