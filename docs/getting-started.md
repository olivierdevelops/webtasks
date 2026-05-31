# Getting started

This guide takes you from zero to a running task in under five minutes.

---

## Prerequisites

| Requirement | Notes |
|---|---|
| **Go 1.22+** | To build the binary |
| **Chrome or Chromium** | chromedp talks to an installed browser — it does not bundle one |
| **Internet** | Demo tasks hit live websites |

For containers, [chromedp/headless-shell](https://hub.docker.com/r/chromedp/headless-shell)
makes a clean base image.

---

## Step 1 — Build

```bash
git clone https://github.com/olivierdevelops/webtasks.git
cd webtasks
go build -o build/webtasks ./cmd/webtasks
```

The result is a single static binary (~17 MB) with no JVM, no chromedriver,
no Selenium server.

---

## Step 2 — Start the server

Point the server at the included demo bundle:

```bash
WEBTASKS_BUNDLE=$(pwd)/demo ./build/webtasks
```

You should see:

```
[webtasks] starting on 127.0.0.1:8765
[webtasks] bundle: /path/to/webtasks/demo (dir)
```

!!! tip "Watch Chrome while authoring"
    Set `WEBTASKS_HEADLESS=false` to open a visible browser window — invaluable
    while learning selectors:

    ```bash
    WEBTASKS_HEADLESS=false WEBTASKS_BUNDLE=$(pwd)/demo ./build/webtasks
    ```

Environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `WEBTASKS_HOST` | `127.0.0.1` | Bind address |
| `WEBTASKS_PORT` | `8765` | HTTP port |
| `WEBTASKS_BUNDLE` | `./bundle-example` | Path to config bundle (dir or `.zip`) |
| `WEBTASKS_HEADLESS` | `true` | Run Chrome headless |

Full list: [Configuration](configuration.md).

---

## Step 3 — Check health

```bash
curl -s http://127.0.0.1:8765/health | python3 -m json.tool
```

```json
{
  "ok": true,
  "pools": { "default": { "size": 3, "available": 3 } },
  "taskCount": 38
}
```

List every registered task:

```bash
curl -s http://127.0.0.1:8765/tasks | python3 -m json.tool
```

Each entry includes the task name, input schema, and transport types.

---

## Step 4 — Run your first task

=== "curl"

    ```bash
    curl -s -X POST http://127.0.0.1:8765/tasks/basics/title \
      -H 'Content-Type: application/json' \
      -d '{}' | python3 -m json.tool
    ```

=== "executor helper"

    ```bash
    executor call basics/title
    ```

=== "Python"

    ```python
    import requests

    resp = requests.post(
        "http://127.0.0.1:8765/tasks/basics/title",
        json={},
    )
    print(resp.json())
    ```

Response:

```json
{
  "ok": true,
  "data": {
    "page": {
      "title": "Example Domain",
      "heading": "Example Domain",
      "body": "This domain is for use in documentation examples …"
    }
  }
}
```

---

## Step 5 — Try more demos

Work through the demo catalogue — each page shows the YAML, the command to
run, and the expected output shape.

| Category | Start here | What you'll learn |
|---|---|---|
| Basics | `basics/title` | `goto` + `extract` |
| Crawl | `crawl/hackernews-top` | List extraction |
| Search | `search/duckduckgo` | Input templating `{{q}}` |
| Interaction | `interaction/form-fill` | `sendkeys` + click |
| Streaming | `streaming/progress` | SSE live events |
| Rendering | `rendering/pdf` | Print to PDF |
| Recording | `recording/record` | Animated GIF capture |

[:octicons-arrow-right-24: Full demo catalogue](demos/index.md)

---

## Step 6 — Stream progress (SSE)

Some tasks support Server-Sent Events for live progress:

```bash
curl -N -X POST http://127.0.0.1:8765/tasks/streaming/progress \
  -H 'Content-Type: application/json' \
  -H 'Accept: text/event-stream' \
  -d '{}'
```

<div class="diagram" markdown="1">

![SSE event stream](assets/sse-stream.svg)

</div>

Or with the executor helper:

```bash
executor call streaming/progress '{}' true
```

---

## Step 7 — Author your own task

Create a new YAML file — the server hot-reloads on every request:

```bash
mkdir -p my-bundle/tasks
cat > my-bundle/tasks/pool.yaml <<'EOF'
pools:
  default: { size: 1 }
EOF

cat > my-bundle/tasks/hello.yaml <<'EOF'
name: "hello"
poolTag: "default"
transports: ["rest"]
timeoutMs: 15000

flow:
  - run: goto
    params: { url: "https://example.com" }
  - run: extract
    as: page
    params:
      selector: "h1"
      repeat: false
      fields:
        title: { kind: text, selector: "." }
EOF

WEBTASKS_BUNDLE=$(pwd)/my-bundle ./build/webtasks &
curl -s -X POST http://127.0.0.1:8765/tasks/hello -H 'Content-Type: application/json' -d '{}'
```

Full walkthrough: [Build your own task](build-your-own-task.md).

---

## Next steps

- [Demo catalogue](demos/index.md) — 38 copy-paste examples
- [Cookbook](cookbook.md) — 12 worked recipes
- [HTTP API](http-api.md) — integrate from any language
- [Bundle packaging](bundle.md) — ship config as a zip
