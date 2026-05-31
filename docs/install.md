# Install

Get a running server and your first result in under a minute.

---

## 1. Install the binary

```bash
curl -fsSL https://olivierdevelops.github.io/webtasks/install.sh | sh
```

The installer detects your OS and architecture, downloads the matching
`webtasks` binary, and drops it on your `PATH`. If no prebuilt binary is
available for your platform it builds from source automatically (needs Go and
git).

??? note "Other ways to install"

    === "Pick the install dir"

        ```bash
        WEBTASKS_INSTALL_DIR=$HOME/bin \
          curl -fsSL https://olivierdevelops.github.io/webtasks/install.sh | sh
        ```

    === "A specific version"

        ```bash
        WEBTASKS_VERSION=v0.1.0 \
          curl -fsSL https://olivierdevelops.github.io/webtasks/install.sh | sh
        ```

    === "From source"

        ```bash
        git clone https://github.com/olivierdevelops/webtasks
        cd webtasks
        go build -o webtasks ./cmd/webtasks
        ```

!!! info "One prerequisite: Chrome"
    webtasks drives an **installed** Chrome or Chromium — it doesn't bundle a
    browser. On a server or container, `chromedp/headless-shell` is the standard
    base image. `ffmpeg` is only needed for MP4 recording.

---

## 2. Get a bundle of tasks

A bundle is a folder of `.webtask` recipes the server exposes as endpoints. The
repo ships a **demo bundle** with 38 example tasks — grab it to try things out:

```bash
git clone --depth 1 https://github.com/olivierdevelops/webtasks ~/webtasks
```

---

## 3. Start the server

```bash
WEBTASKS_BUNDLE=~/webtasks/demo webtasks
```

You should see:

```
[webtasks] starting on 127.0.0.1:8765
[webtasks] bundle: /home/you/webtasks/demo (dir)
```

!!! tip "Watch the browser"
    Add `WEBTASKS_HEADLESS=false` to open a visible Chrome window — handy while
    learning selectors.

    ```bash
    WEBTASKS_HEADLESS=false WEBTASKS_BUNDLE=~/webtasks/demo webtasks
    ```

---

## 4. Check it's healthy

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

List every available task:

```bash
curl -s http://127.0.0.1:8765/tasks | python3 -m json.tool
```

---

## 5. Run your first task

=== "curl"

    ```bash
    curl -s -X POST http://127.0.0.1:8765/tasks/basics/title \
      -H 'Content-Type: application/json' -d '{}' | python3 -m json.tool
    ```

=== "Python"

    ```python
    import requests
    r = requests.post("http://127.0.0.1:8765/tasks/basics/title", json={})
    print(r.json())
    ```

=== "JavaScript"

    ```js
    const r = await fetch("http://127.0.0.1:8765/tasks/basics/title", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    });
    console.log(await r.json());
    ```

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

Pass inputs in the JSON body — for example a search query:

```bash
curl -s -X POST http://127.0.0.1:8765/tasks/search/duckduckgo \
  -H 'Content-Type: application/json' \
  -d '{"q":"golang"}' | python3 -m json.tool
```

---

## 6. Stream live progress (SSE)

Some tasks emit progress as they go. Ask for an event stream and watch it
unfold:

```bash
curl -N -X POST http://127.0.0.1:8765/tasks/streaming/progress \
  -H 'Accept: text/event-stream' \
  -H 'Content-Type: application/json' \
  -d '{}'
```

<div class="diagram" markdown="1">

![SSE event stream](assets/sse-stream.svg)

</div>

---

## Next steps

- **[Browse the examples](demos/index.md)** — 38 copy-paste tasks
- **[Write your own task](writing-tasks.md)** — the `.webtask` language
- **[How it works](how-it-works.md)** — pools, hot-reload, the request lifecycle
- **[Deployment](deploy.md)** — ship a bundle to production
