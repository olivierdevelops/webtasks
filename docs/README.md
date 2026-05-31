# webtasks documentation

`webtasks` is a long-running **browser-automation server**: one Go binary plus
a config bundle of YAML "tasks". Each task is a declarative browser flow
(`goto`, `wait-for`, `click`, `extract`, `download-each`, …) that the server
exposes as a typed HTTP endpoint. Callers send JSON inputs and get JSON back;
selectors and Chrome internals stay on the server.

Built on [chromedp](https://github.com/chromedp/chromedp) (Chrome DevTools
Protocol over WebSocket) — no chromedriver, no Selenium server.

---

## Map of the docs

### Start here

- **[../README.md](../README.md)** — project overview, the "why", 30-second
  quick start.
- **[build-your-own-task.md](build-your-own-task.md)** — end-to-end tutorial:
  from "I want to scrape X" to a working `POST /tasks/my-thing` endpoint.
- **[cookbook.md](cookbook.md)** — 12 worked recipes, the condensed action
  reference, templating cheatsheet, and troubleshooting.

### Feature reference (this set — detailed, one subsystem per file)

| Doc | Covers |
|---|---|
| [architecture.md](architecture.md) | VHCO layering, the components, a task's full request lifecycle, the flow interpreter, error/recovery model. |
| [actions.md](actions.md) | **The complete action vocabulary** — every `run:` keyword, all params, defaults, output shapes, and examples. |
| [http-api.md](http-api.md) | Every HTTP endpoint, request/response shapes, the SSE event protocol, error envelope, status codes. |
| [task-definition.md](task-definition.md) | The task YAML schema in full — `name`, `poolTag`, `transports`, `input`, `flow`, `timeoutMs`, `setupTask`, step-level fields. |
| [templating.md](templating.md) | The `{{…}}` engine — lookup order, `or:` fallback, dotted paths, env fallback, single-token raw resolution. |
| [pools.md](pools.md) | Window pools, leasing, concurrency, persistent Chrome profiles, crash recovery, `setupTask` preludes. |
| [bundle.md](bundle.md) | The config bundle — directory vs zip layout, hot-reload, packaging and shipping. |
| [secrets.md](secrets.md) | Declaring secrets, the env/arg/prompt resolution chain, the `sm` workflow. |
| [static-mounts.md](static-mounts.md) | Mapping URL prefixes to directories — listing, serving, `${ENV}` expansion, path-traversal safety. |
| [configuration.md](configuration.md) | Every `WEBTASKS_*` environment variable and launcher CLI flag. |
| [cli.md](cli.md) | The `executor` command reference (`commands.yaml`). |

### Example bundles

- **[../demo/README.md](../demo/README.md)** — 14 runnable demo tasks across 7
  categories, each exercising a different engine feature.
- **[../concio/README.md](../concio/README.md)** — a real-world bundle that
  scrapes a logged-in Concio (Starise IM) account, including encrypted-blob
  capture.

---

## Feature summary

A one-screen index of everything the engine can do. Each links to where it is
documented in detail.

**Browser automation** — navigate, wait, click (CSS or by visible text),
type, scroll-until-stable, run arbitrary JS. → [actions.md](actions.md)

**Data extraction** — CSS-selector field specs producing typed JSON (single
object or repeated list), with transforms. → [actions.md#extract](actions.md#extract)

**Rendering & capture** — PDF, HTML-string-to-PDF, screenshots (viewport /
element / full-page), MHTML snapshots, device emulation. → [actions.md](actions.md)

**Recording** — screencast a flow (or a single step) to an animated GIF (pure
Go) or MP4 (via `ffmpeg`). → [actions.md#record](actions.md#record)

**Network & session** — HAR-style request capture, console capture, wait-for-
network-idle, read/write cookies, outbound HTTP (no browser). → [actions.md](actions.md)

**Downloads** — native click-and-poll downloads, plus a client-side-blob
capture path for apps that decrypt before download. → [actions.md#download-each](actions.md#download-each)

**Control flow & data** — `for-each`, `loop` (JS condition), `set`, `call`,
`return`, `export` (CSV/NDJSON/markdown), `read-file`, `write-files`. → [actions.md](actions.md)

**Transports** — synchronous JSON over REST, and Server-Sent Events for live
progress. → [http-api.md](http-api.md)

**Operational** — window pools with concurrency, persistent login profiles,
crash recovery, declared secrets, static file mounts, hot-reloaded config,
single-binary + zip-bundle deployment. → [pools.md](pools.md),
[secrets.md](secrets.md), [bundle.md](bundle.md)
