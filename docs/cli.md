# CLI & commands

There are two ways to drive webtasks. **End users only need the first.**

## 1. The `webtasks` binary + HTTP (what you actually use)

The product is the `webtasks` binary. It takes its configuration from
environment variables and serves the HTTP API — there are no subcommands to
learn. You run the server, then call tasks with `curl` (or any HTTP client).

```bash
# start the server against a bundle
WEBTASKS_BUNDLE=~/webtasks/demo webtasks &

# inspect
curl -s localhost:8765/health        | python3 -m json.tool   # pool status + task count
curl -s localhost:8765/tasks         | python3 -m json.tool   # every task + its schema

# run a task (optional JSON body of inputs)
curl -s -X POST localhost:8765/tasks/basics/title -d '{}'
curl -s -X POST localhost:8765/tasks/search/duckduckgo -d '{"q":"golang"}'

# stream live progress
curl -N -X POST localhost:8765/tasks/streaming/progress \
  -H 'Accept: text/event-stream' -d '{}'
```

That's the whole user-facing surface. → [HTTP API](http-api.md) for the full
contract, [Deployment](deploy.md) for the environment variables.

---

## 2. `executor` — an optional helper for contributors

The repo ships a `commands.yaml` of `executor` shortcuts that wrap the
`go build` / run / package workflows used while **developing webtasks itself**.
They're pure convenience around the binary and `curl` — you never need them to
*use* webtasks, only to hack on it. See
[commands.yaml](https://github.com/olivierdevelops/webtasks/blob/main/commands.yaml)
for the authoritative source.

All commands honour `WEBTASKS_PORT` (default `8765`) when talking to a running
server, and `WEBTASKS_*` env vars when starting one. → [Deployment](deploy.md)

---

## Build & lifecycle

| Command | What it does |
|---|---|
| `executor build` | `go build -o build/webtasks ./cmd/webtasks`. |
| `executor server` | Build, then run the server. Defaults the bundle to `./demo` and `WEBTASKS_HEADLESS=true`; override with env vars. |
| `executor clean` | Remove `build/` and `dist/`. |
| `executor bundle [out]` | Zip `bundle-example/` into `out` (default `dist/bundle.zip`). |
| `executor package` | Build a stripped static binary (`CGO_ENABLED=0 -trimpath -ldflags '-s -w'`) **and** `dist/bundle.zip`. Ship both. → [Deployment](deploy.md) |

```bash
executor build
executor server &
WEBTASKS_HEADLESS=false WEBTASKS_BUNDLE=$(pwd)/my-bundle executor server &
```

---

## Inspect & call

| Command | What it does |
|---|---|
| `executor health` | `GET /health` (pretty-printed). |
| `executor list-tasks` | `GET /tasks` (pretty-printed). |
| `executor call <name> [body] [stream]` | `POST /tasks/<name>` with optional JSON `body` (default `{}`). `stream=true` switches to SSE. Probes `/health` first for a clear "no server" error. |

```bash
executor list-tasks
executor call examples/trending-papers
executor call examples/trending-papers '{"q":"chromedp"}'
executor call examples/trending-papers '{}' true        # SSE stream
```

The third positional arg to `call` is the stream flag — pass `true` to receive
Server-Sent Events instead of waiting for one JSON blob. → [http-api.md](http-api.md)

---

## Concio workflow

The Concio bundle (`./concio`) has its own commands. They expect Concio
credentials in the environment as `CONCIO_USER` / `CONCIO_PASSWORD` (or a vault
that provides them).

| Command | What it does |
|---|---|
| `executor concio-server` | Build + run the server against the `concio/` bundle, wrapped in `sm exec --` so `CONCIO_PASSWORD` resolves. |
| `executor concio-test [task] [out]` | Self-contained: build, boot the server on its own port against `concio/`, wait for readiness, run one Concio task streaming live SSE, then shut down. Leaves nothing running. |
| `executor concio-setup` | `POST concio/setup` — idempotent login. |
| `executor concio-list-chats` | `POST concio/list-chats` — sidebar chats. |
| `executor concio-list-contacts` | `POST concio/list-contacts` — "People" directory. |
| `executor concio-list-groups` | `POST concio/list-groups` — "Groups" directory. |
| `executor concio-get-messages <peer> [maxScrolls] [stream]` | Open one chat, scroll to the start of history, return every message. |
| `executor concio-capture-files <peer> [outDir]` | Install the blob hook, click each file bubble, write decrypted attachments to disk. |
| `executor concio-extract [out]` | Run the full server-side sweep (`concio/extract-all`) over every chat, streaming progress. |

```bash
# one-shot, self-contained test (creds inline):
CONCIO_USER='200 008 6861' CONCIO_PASSWORD='secret' executor concio-test watch

# or with a long-running server + a vault:
sm exec -- executor concio-server &
executor concio-setup
executor concio-get-messages "Nicholas Huang" 0 true
```

See the [concio bundle](https://github.com/olivierdevelops/webtasks/tree/main/concio)
for its structure and the output format it produces.

---

## Notes

- These commands assume a working `go` toolchain for anything that builds. On a
  host without `go`, ship a `package`d binary + bundle and run it directly.
- The `call` family pretty-prints JSON via `python3 -m json.tool`; SSE output is
  streamed raw.
- `executor` is the command runner reading `commands.yaml`; the `{{CWD}}` and
  `{{.arg}}` placeholders in that file are the runner's own templating, not the
  webtasks engine's.
