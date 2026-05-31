# 1. Why replace YAML?

webtasks chose YAML deliberately: human-readable, diff-friendly, no compile
step, hot-reloadable. That choice still holds for **runtime config** (`pool.yaml`,
`secrets.yaml`, `static-mounts.yaml`). This guide proposes Capy only for
**task definitions** — the repetitive, schema-heavy files under `tasks/`.

---

## What goes wrong with raw YAML

### 1. Schema lives in documentation, not in the file

The authoritative schema for a task is spread across
[task-definition.md](../task-definition.md) and [actions.md](../actions.md).
Nothing in the YAML file itself prevents:

```yaml
- run: gotoo          # typo — fails at runtime in Chrome
  params: { urll: "…" }
```

With a Capy library, `gotoo` is not a declared function shape. Transpilation
fails **before** the server starts (or before the task is registered).

### 2. Indentation is the parser

A mis-indented `fields:` block silently nests under the wrong key. YAML linters
help but do not know webtasks semantics.

Capy block functions use explicit closers (`end`) and indentation rules enforced
by the Capy lexer — the same rules as the library manifest.

### 3. Boilerplate dominates small tasks

Open any file in `demo/tasks/basics/`. Roughly half the lines are ceremony:
`name`, `poolTag`, `transports`, `timeoutMs`, `flow:`, `- run:`, `params:`.

The proposed DSL collapses that into a `task … end` block where metadata and
steps share one visual hierarchy.

### 4. AI agents need a bounded language

When an agent writes YAML tasks, it can invent action names, nest `params`
incorrectly, or confuse `repeat: true` with `repeat: "true"`. Capy's core
value for webtasks is **grammar-as-contract**:

> The library is the complete grammar. A task DSL whose `ActionName` is an
> enum **cannot** emit `DROP TABLE`. A webtasks DSL whose actions are
> whitelisted **cannot** invoke undefined `run:` keywords.

See [Capy's AI agents doc](https://github.com/olivierdevelops/capy/blob/main/docs/ai-agents.md).

---

## What Capy is (and is not)

| Capy is | Capy is not |
|---|---|
| A transpiler: source DSL → target text | A replacement for chromedp or the flow interpreter |
| Grammar defined in `webtasks.capy` | A fixed syntax you must accept as-is |
| Embeddable Go library (`go get github.com/olivierdevelops/capy`) | A separate daemon or sidecar |
| Safe — inner DSL does not execute user code | A general-purpose scripting runtime |

The webtasks server continues to run `domain.Command` steps exactly as today.
Capy only changes **how those commands are authored**.

---

## Coexistence model

Phase 1 (recommended): **both formats in one bundle**

```
bundle/
├── capy/
│   └── webtasks.capy       # shared grammar library
├── tasks/
│   ├── pool.yaml           # stays YAML
│   ├── basics/
│   │   ├── title.yaml      # legacy
│   │   └── title.capy      # new — transpiles to equivalent YAML
```

Loader precedence (proposed):

1. Walk `tasks/**/*.capy`
2. Transpile each with `webtasks.capy` → YAML bytes
3. Walk `tasks/**/*.yaml` (skip if `.capy` sibling exists, or merge by `name:`)
4. Unmarshal into `domain.TaskDef`

Phase 2: Capy-only bundles for new projects; YAML remains supported indefinitely.

Phase 3 (optional): transpile directly to JSON matching `TaskDef`, skip YAML.

---

## Cost/benefit summary

| Cost | Benefit |
|---|---|
| Add `github.com/olivierdevelops/capy` dependency | Typed authoring, `capy check` in CI |
| Ship `capy/webtasks.capy` in bundle (~300–600 lines) | 2–3× shorter task sources |
| Extend `MakeTaskRegistry` (~80 lines) | Hot-reload `.capy` same as YAML |
| Authors learn new syntax | `capy docs` auto-generates reference from grammar |
| Capy is pre-1.0 | Library schema stable enough for embedding; pin version in `go.mod` |

---

## When to keep YAML

- **`pool.yaml`, `secrets.yaml`, `static-mounts.yaml`** — operational config,
  not flow logic; YAML + env expansion is fine.
- **One-off experiments** — paste YAML, call endpoint, delete.
- **Teams with heavy yq/Helm investment** — Capy library can emit YAML; keep
  downstream tooling unchanged.

---

Next: [Capy primer →](02-capy-primer.md)
