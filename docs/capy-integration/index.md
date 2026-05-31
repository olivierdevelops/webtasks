# Capy integration guide

Replace verbose YAML task definitions with a **typed, sandboxed DSL** powered by
[Capy v0.20.0](https://github.com/olivierdevelops/capy) — a configurable
transpiler where **you define the entire grammar** in a `.capy` library file.

!!! info "Version"
    This guide targets **Capy `v0.20.0`** (`github.com/olivierdevelops/capy`).
    Start with the [v0.20 integration reference](00-v020-overview.md), then follow
    the chapters below for webtasks-specific grammar, samples, and rollout.

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

## Document map (27 chapters)

| # | Chapter | What you'll learn |
|---|---|---|
| 0 | **[Capy v0.20 overview](00-v020-overview.md)** | Latest API, CLI, sandboxing, rollout |
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
| 11 | [Go embedding](11-go-embedding.md) | `RunMulti`, `SetHost`, registry hook |
| 12 | [Migration strategy](12-migration.md) | YAML + Capy in one bundle |
| 13–21 | [Samples](samples/01-hello.md) | Hello → Concio-scale projects |
| 22 | [AI authoring](22-ai-authoring.md) | Token compression, MCP, sandboxing |
| 23 | [Editor tooling](23-editor-tooling.md) | Introspect, fmt, watch, WASM |
| 24 | [Testing and CI](24-testing-ci.md) | `capy check`, golden files |
| 25 | [Troubleshooting](25-troubleshooting.md) | Common errors |
| 26 | [Roadmap](26-roadmap.md) | Phased rollout |

Reference artifacts:

- [`grammar/webtasks.capy`](grammar/webtasks.capy) — proposed library (v0.20.0 syntax)
- [`grammar/samples/*.capy`](grammar/samples/) — runnable examples

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
    timeout 15000
    transport rest

    goto "https://example.com"
    extract page from "html"
        title text "title"
    end
end
```

Transpile with:

```bash
go install github.com/olivierdevelops/capy/cmd/capy@v0.20.0
capy run capy/webtasks.capy tasks/basics/title.capy
```

---

## External links

- [Capy repository](https://github.com/olivierdevelops/capy)
- [Capy CHANGELOG](https://github.com/olivierdevelops/capy/blob/main/CHANGELOG.md)
- [Capy migration guide](https://github.com/olivierdevelops/capy/blob/main/docs/migration-guide.md)
- [Grammar-as-contract](https://github.com/olivierdevelops/capy/blob/main/docs/grammar-as-contract.md)
- [Host capabilities & sandboxing](https://github.com/olivierdevelops/capy/blob/main/docs/host-capabilities.md)
- [AI agents](https://github.com/olivierdevelops/capy/blob/main/docs/ai-agents.md)
- [CLI reference (v0.20)](https://github.com/olivierdevelops/capy/blob/main/docs/cli.md)
- [webtasks task definition](../task-definition.md)
- [webtasks actions reference](../actions.md)
