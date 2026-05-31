# Task definition reference

A task is one YAML file under the bundle's `tasks/` tree. The filename doesn't
matter; the `name:` field is the URL slug. This doc is the complete schema. For
a guided walkthrough, see [build-your-own-task.md](build-your-own-task.md).

```yaml
name: "category/short-name"      # required — the URL slug
poolTag: "default"               # which window pool to lease from
transports: ["rest"]             # invocation styles (metadata)
timeoutMs: 30000                 # whole-task budget
setupTask: "category/login"      # optional idempotent prelude
input:                           # caller-supplied values
  q: { type: string, required: true, default: "", doc: "…" }
flow:                            # the ordered steps
  - run: goto
    params: { url: "https://…?q={{q}}" }
```

---

## Task-level fields

| Field | Type | Default | Notes |
|---|---|---|---|
| `name` | string | — | **Required.** The slug for `POST /tasks/<name>`. Slashes are allowed and conventional (`category/thing`). Must be unique across the bundle. |
| `poolTag` | string | `default` | Which window pool to lease a Chrome window from. Declared in `tasks/pool.yaml`. → [pools.md](pools.md) |
| `transports` | list | — | Invocation styles: `rest`, `sse`, `websocket`, `async`. Currently metadata surfaced in `GET /tasks`; REST + SSE are served off the same endpoint. → [http-api.md](http-api.md) |
| `input` | map | `{}` | Declared inputs (below). |
| `flow` | list | — | **Required.** The ordered list of step commands. → [actions.md](actions.md) |
| `timeoutMs` | int | `60000` | Whole-task budget. Becomes a context deadline every browser call inherits; a stuck step fails with `context deadline exceeded` once it elapses. |
| `setupTask` | string | — | Names another registered task whose flow runs **in the same window, before** this task's flow. Must be idempotent (a no-op when its post-condition already holds) — e.g. an "ensure logged in" prelude. The caller's inputs pass through to it. A single "Running setup: …" status is emitted. |

---

## The `input:` block

Each entry declares one caller parameter. It does double duty: it validates
incoming requests *and* it's the schema published in `GET /tasks`.

```yaml
input:
  query:      { type: string, required: true,  doc: "What to search for" }
  pageSize:   { type: int,    required: false, default: 25 }
  includeAds: { type: bool,   required: false, default: false, doc: "…" }
```

| Field | Notes |
|---|---|
| `type` | Documentation hint (`string`, `int`, `bool`, …). Not strictly enforced — values are used as-is. |
| `required` | When `true` and the caller supplies no value (and there's no default), the run errors *before any browser action* with `missing required input(s): …`. |
| `default` | Used when the caller omits the key. |
| `doc` | Human description, surfaced in `GET /tasks`. |

**Binding rules** (`bindInputs`):

- A declared input absent from the request takes its `default`.
- A `required` input that resolves to `nil`/`""` (and has no default) is
  collected into the "missing required input(s)" error.
- **Unmodelled inputs pass through.** Keys the caller sends that aren't in the
  `input:` schema are still bound and usable via `{{name}}`. The `input:` block
  is the *documented* contract, not a whitelist.

Reference any bound value in a string param with `{{name}}`; templating also
falls back to environment variables, so `{{API_TOKEN}}` resolves to a secret
declared in `secrets.yaml`. → [templating.md](templating.md), [secrets.md](secrets.md)

---

## The `flow:` block

A list of step commands, run in order. One step:

```yaml
- run: <action>          # the action keyword (required)
  status: "Loading…"     # optional — emitted as an SSE `status` event (templated)
  as: <name>             # optional — where the result is stored
  record: true           # optional — screencast just this step to a debug GIF
  params: { … }          # action-specific parameters (templated)
  do:                    # child steps for block actions (for-each, loop, record, …)
    - run: …
```

| Step field | Notes |
|---|---|
| `run` | **Required.** The action keyword. → [actions.md](actions.md) |
| `status` | A human-readable progress line, emitted as a `status` SSE event (no-op for sync REST). Templated. |
| `as` | Names where the step's result lands — in both the response `data` and the live bindings (so later steps can `{{name}}` it). |
| `record` | `record: true` screencasts *this step alone* to a GIF under `$TMPDIR/webtasks-recordings/`. If the step fails, the error names the file. A debug aid. → [actions.md#record](actions.md#record) |
| `params` | Action parameters. Templated recursively (strings, list items, map values). |
| `do` | Child steps, for the block actions (`for-each`, `loop`, `record`, `capture-network`, `console`). |

---

## A complete example

```yaml
name: "my/github-trending"
poolTag: "default"
transports: ["rest", "sse"]
timeoutMs: 30000

input:
  language: { type: string, required: false, default: "" }
  since:    { type: string, required: false, default: "daily" }

flow:
  - status: "Loading GitHub trending"
    run: goto
    params: { url: "https://github.com/trending/{{language}}?since={{since}}" }

  - run: wait-for
    params: { selector: "article.Box-row", timeoutMs: 15000 }

  - run: extract
    as: repos
    params:
      selector: "article.Box-row"
      repeat: true
      fields:
        slug: { kind: text, selector: "h2 a", transform: trim }
        href: { kind: attr, selector: "h2 a", name: "href" }

  - run: return
    params: { value: "{{repos}}" }
```

See the full annotated cheatsheet in
[build-your-own-task.md §12](build-your-own-task.md#12-cheatsheet).
