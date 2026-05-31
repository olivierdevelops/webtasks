# HTTP API

The server exposes a small, fixed set of endpoints. Tasks are *not* hardcoded
in Go — they come from the bundle's `tasks/**/*.yaml` and are surfaced under
`POST /tasks/<name>`. Static mounts add further routes declaratively
(see [static-mounts.md](static-mounts.md)).

Default bind address: `http://127.0.0.1:8765` (override with `WEBTASKS_HOST` /
`WEBTASKS_PORT` — see [configuration.md](configuration.md)).

| Method & path | Purpose |
|---|---|
| `GET /health` | Pool status + registered task count. |
| `GET /tasks` | List registered tasks with their input/output schemas. |
| `POST /tasks/<name>` | Run a task; return the final JSON. |
| `POST /tasks/<name>` + `Accept: text/event-stream` | Run a task, streaming progress events. |
| `GET <mount>` | Static-mount listing (JSON). → [static-mounts.md](static-mounts.md) |
| `GET <mount>/<path>` | Stream a file from a static mount. |

---

## `GET /health`

```bash
curl -s http://127.0.0.1:8765/health
```

```json
{
  "ok": true,
  "taskCount": 14,
  "pools": {
    "default": { "size": 3, "free": 3, "busy": 0 },
    "concio":  { "size": 1, "free": 1, "busy": 0 }
  }
}
```

- `taskCount` — number of registered tasks (re-counted live, since the registry
  hot-reloads).
- `pools` — per-pool `{size, free, busy}` window counts. → [pools.md](pools.md)

---

## `GET /tasks`

Lists every registered task with the metadata other consumers need to call it.

```bash
curl -s http://127.0.0.1:8765/tasks
```

```json
[
  {
    "name": "examples/trending-papers",
    "poolTag": "default",
    "transports": ["rest"],
    "timeoutMs": 60000,
    "input": {
      "q": { "type": "string", "required": false, "default": "go", "doc": "Query" }
    }
  }
]
```

The `input` map is exactly the task's declared `input:` schema — this is the
contract for callers. → [task-definition.md](task-definition.md)

---

## `POST /tasks/<name>`

Run a task. `<name>` is the task's `name:` field (slashes allowed — chi
captures the whole wildcard, leading slash trimmed). The request body is an
optional JSON object of input values:

```bash
curl -s -X POST http://127.0.0.1:8765/tasks/examples/trending-papers \
     -H 'Content-Type: application/json' \
     -d '{"q":"chromedp"}'
```

### Success response

```json
{
  "ok": true,
  "data": { "papers": [ … ] }
}
```

`data` is the task's output map — the accumulation of every step's `as:`
binding. **Unless** the flow ran a `return` action, in which case `data` is that
returned value alone:

```yaml
- run: return
  params: { value: "{{papers}}" }
```

```json
{ "ok": true, "data": [ … ] }
```

### Error response

```json
{
  "ok": false,
  "error": { "code": "EXECUTION_FAILED", "message": "step \"wait-for\": context deadline exceeded" }
}
```

| HTTP status | `error.code` | Meaning |
|---|---|---|
| 400 | `MISSING_NAME` | No task name in the path. |
| 400 | `BAD_BODY` | Request body wasn't valid JSON. |
| 500 | `EXECUTION_FAILED` | The task ran but errored (missing input, selector timeout, browser fault, …). |
| 500 | `NO_FLUSHER` | (SSE only) the response writer can't stream. |

Common `EXECUTION_FAILED` messages:

- `missing required input(s): q` — a `required: true` input wasn't supplied.
- `unknown task: foo/bar` — no such task (check `GET /tasks`).
- `acquire timeout: <pool>` — no free window for 30 s. → [pools.md](pools.md)
- `browser session was reset (tab crashed or detached); re-run pool setup…` —
  Chrome target died; the pool replaced the window. Re-run setup/login.

---

## Server-Sent Events (streaming)

The **same** `POST /tasks/<name>` endpoint switches to SSE when the caller
sends `Accept: text/event-stream`. Every step's `status:` field, and every
`emit-event` action, becomes an event; the run finishes with a terminal `done`
or `error` event.

```bash
curl -N -X POST http://127.0.0.1:8765/tasks/streaming/progress \
     -H 'Accept: text/event-stream' \
     -H 'Content-Type: application/json' \
     -d '{}'
```

```
event: status
data: {"text":"Step 1 of 4 — navigate","data":null}

event: progress
data: {"text":"navigation complete","data":{"fraction":0.25}}

event: done
data: {"ok":true,"data":{"page":{"title":"Example Domain"}}}
```

### Event kinds

| `event:` | Emitted by | `data:` payload |
|---|---|---|
| `status` | a step's `status:` field, or `emit-event` with default kind | `{text, data}` |
| `progress` (or any custom kind) | `emit-event` with `kind:` set | `{text, data}` |
| `recording` | a step with `record: true` | `{text, data:{path, step, ok}}` |
| `error` | the run failed | `{message}` |
| `done` | the run succeeded | `{ok:true, data:…}` (same `data` as sync REST) |

Exactly one terminal event (`done` or `error`) ends the stream.

### Transport details

- Response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`,
  `Connection: keep-alive`, `X-Accel-Buffering: no`.
- A `: ping` comment is sent every 15 s as a heartbeat so proxies don't drop an
  idle connection.
- If the client disconnects, the server stops emitting (the run's context is
  cancelled).

Sync REST callers see none of these events — the engine wires a no-op
publisher, so adding `status:`/`emit-event` to a task is always safe.

---

## Transports declared on a task

A task's `transports:` field (e.g. `["rest"]`, `["rest","sse"]`) documents
which invocation styles it supports. In practice the REST endpoint serves both
sync JSON and SSE off the same path based on the `Accept` header; the field is
metadata surfaced in `GET /tasks`. The domain also defines `websocket` and
`async` transport names for future use. → [task-definition.md](task-definition.md)
