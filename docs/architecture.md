# Architecture

How the server is put together, and what happens end-to-end when a task runs.
For the task-author's view, read [build-your-own-task.md](build-your-own-task.md)
first; this doc is for people extending or operating the engine.

---

## VHCO layering

The codebase follows a strict layering convention (VHCO): six folders inside
`internal/`, nothing else. Each layer may only depend "inward".

```
cmd/webtasks/                  # entry point — just calls orchestrator.Run
internal/
├── domain/                    # pure data shapes (no behaviour)
├── features/                  # capabilities, declared as structs of func values
├── usecases/                  # use-case contracts + the dep shapes they need
├── io/                        # HTTP "views": REST handlers, SSE, static mounts
├── infra/                     # adapters: chromedp, bundle, yaml, goquery, http…
└── orchestrator/              # composition root: wires everything together
```

The defining ideas:

- **A feature is a struct of function values**, not an interface with an
  `*Impl`. `features.BrowserActions` is a record of `Goto func(...)`,
  `Click func(...)`, etc. The struct value *is* the feature. See
  [internal/features/features.go](../internal/features/features.go).
- **A use case declares, locally, the dep shapes it needs** and the contract
  it offers. The wired-up implementation lives under
  `internal/orchestrator/usecases/`.
- **The IO layer declares its own use-case protocols** (`io/rest.UseCases`).
  One orchestrator-built implementation can satisfy several consumers' shapes.
- **Only the orchestrator imports concrete types from every layer.** It is the
  single composition root; everything else stays decoupled.

Adding a capability is always the same six-file recipe — see
[cookbook.md §12](cookbook.md#12-add-a-new-action-to-the-engine).

---

## Components

| Component | Package | Responsibility |
|---|---|---|
| **Bundle** | `infra/bundle` | Opens the config root — a directory or a `.zip`/`.jar` read in-place via `archive/zip` (never extracted). Exposes `ReadFile`, `Exists`, `WalkYAML`. → [bundle.md](bundle.md) |
| **WindowSource** | `infra/chromedp` | Owns the Chrome lifecycle. One pooled window = one chromedp allocator + child context. Handles ephemeral vs persistent profiles, per-window download dirs, and crash replacement. → [pools.md](pools.md) |
| **Primitives** | `infra/chromedp` | Stateless CDP operations: navigate, click, screenshot, print-to-PDF, screencast, network/console capture, cookies, etc. |
| **Extractor** | `infra/goqueryx` | Turns an HTML string + an extract spec into typed JSON, using goquery. |
| **WindowLease** | `orchestrator/features` | A counting pool per `poolTag`. `Acquire`/`Release`/`Status`/`Recover`, goroutine-safe via a `sync.Cond`. |
| **TaskRegistry** | `orchestrator/features` | Loads `tasks/**/*.yaml` from the bundle. Hot-reloads on every call in server mode. |
| **Templating** | `orchestrator/features` | `{{name|or:default}}` substitution with dotted-path lookup and process-env fallback. → [templating.md](templating.md) |
| **Run interpreter** | `orchestrator/usecases` | `RunRegisteredTaskImpl` — the "bytecode interpreter" that walks a task's flow and dispatches each step. |
| **REST/SSE view** | `io/rest` | Decodes requests, calls use cases, encodes JSON or streams SSE. → [http-api.md](http-api.md) |
| **Static mounts** | `io/staticmounts` | Serves user-configured directories at user-configured URL prefixes. → [static-mounts.md](static-mounts.md) |
| **Secrets loader** | `orchestrator` | Resolves declared secrets at startup, publishes them as env vars. → [secrets.md](secrets.md) |
| **HTTP server** | `infra/httpserver` | chi-router host with graceful shutdown. |

---

## Boot sequence

`cmd/webtasks/main.go` calls `orchestrator.Run(cfg, args)`. In order
([app.go](../internal/orchestrator/app.go)):

1. **Open the bundle** (`bundle.Open`) — directory or zip.
2. **Resolve secrets** (`LoadSecrets`) — read `secrets.yaml`, walk each
   secret's source list, export resolved values into the process env so
   templating can see them.
3. **Load `static-mounts.yaml`** and **`tasks/pool.yaml`** from the bundle.
   A `default` pool of size 1 is injected if not declared.
4. **Resolve the persistent-profile root** (`WEBTASKS_PROFILE_DIR`, default
   `~/.webtasks/profiles`).
5. **Build infra adapters** — `WindowSource` (which eagerly spawns every
   pool's Chrome windows), `Primitives`, the goquery extractor.
6. **Build features** — templating, browser actions, extraction, the task
   registry (hot-reload on), the window lease (this is where windows are
   actually allocated), the JS-scripts lookup.
7. **Build use cases** — run-task, list-tasks, health.
8. **Start the HTTP server**, registering the REST routes and the static
   mounts.
9. **Block until SIGINT/SIGTERM**, then graceful-shutdown the server and
   `CloseAll()` every Chrome window.

---

## Request lifecycle of a task run

A `POST /tasks/<name>` request flows through:

```
HTTP handler (io/rest)
   │  decode JSON body → InputValues
   │  branch: Accept: text/event-stream → SSE, else sync REST
   ▼
RunRegisteredTask.Execute (orchestrator/usecases)
   │  1. registry.Get(name)                 → TaskDef (hot-reloaded from YAML)
   │  2. context.WithTimeout(def.Timeout())  → whole-run budget
   │  3. bindInputs(def, vals)                → validate required, apply defaults
   │  4. lease.Acquire(poolTag, 30s)          → lease a Chrome window
   │  5. if def.SetupTask: run its flow first (same window)
   │  6. runFlow(def.Flow)                     → execute each step
   │  7. lease.Release(window)                 (deferred)
   ▼
runCommand → runCommandRaw (the big switch on c.Run)
   │  renderParams(c.Params, bindings)         → resolve {{…}} templating
   │  emit status event if c.Status set
   │  dispatch to a features.BrowserActions.* / control-flow handler
   │  assign(out, bindings, c.As, result)      → record output, expose to later steps
   ▼
Output map  →  resultData() unwraps __result__ if a `return` set it
            →  {"ok": true, "data": …}  (REST)  or  event: done  (SSE)
```

Key behaviours:

- **Per-call deadline.** The task's `timeoutMs` becomes a context deadline that
  every browser call inherits (see `withCtx` in
  [makes.go](../internal/orchestrator/features/makes.go)). A stuck selector or
  hung click fails with `context deadline exceeded` instead of blocking the
  window lease forever.
- **Bindings vs output.** `assign` writes a step's `as:` result into *both* the
  response `Output` map (returned to the caller) and the live `bindings` map
  (so later steps can reference it via `{{name}}`). → [templating.md](templating.md)
- **Nested flows.** `for-each`, `loop`, `record`, `capture-network`, and
  `console` run their `do:` children through the same `runFlow`, with cloned
  bindings for iteration variables.

---

## Error and recovery model

- **`return` action** — raises a sentinel `errStopFlow`. `runFlow` propagates
  it and `Execute` unwraps it as a *successful* early exit. The reserved
  `__result__` binding (set by `return value:`) becomes the sole response
  payload.
- **Fatal browser state** — if a step error contains a marker like
  `target detached`, `tab crashed`, `websocket: close`, or `context canceled`
  (`isFatalBrowserState`), the engine calls `lease.Recover(window)` to replace
  the dead Chrome target with a fresh one. The pool slot stays usable, but the
  caller must re-run any setup task (e.g. login) before retrying. The returned
  error states the session was reset.
- **Setup-task failures** are reported with a `setup "x" failed:` prefix and
  trigger the same recovery path on fatal errors.
- **Ordinary step errors** are wrapped as `step "<run>": <err>` and returned
  as an `EXECUTION_FAILED` HTTP 500 (or an SSE `error` event).

---

## Concurrency model

- Each pool tag has a fixed number of pre-allocated Chrome windows. A task run
  leases one for its entire duration and releases it at the end.
- Parallelism per pool = the pool's `size`. When all windows are busy,
  `Acquire` blocks (on a `sync.Cond`) up to 30 s, then returns
  `acquire timeout: <tag>`.
- Persistent pools must be `size: 1` — two live Chrome processes cannot share
  one profile directory. → [pools.md](pools.md)
- A single window is never used by two runs at once, so tasks need not worry
  about cross-talk in page state — but successive runs on the same window do
  share whatever the previous run left behind (cookies, localStorage), which is
  exactly what `setupTask` and persistent profiles exploit.
