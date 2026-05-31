# Usage guide

You have the `webtasks` binary. **Now what?** This page takes you from a
freshly built binary to running real automations — three different ways,
depending on what you're trying to do.

---

## The mental model

There is **one binary** and it does **three jobs**:

<div class="grid cards" markdown>

-   :material-flash: **Run one recipe**

    `webtasks run file.webtask`

    Execute a single recipe once, print the result, exit. Best for trying
    things, scripts, and cron jobs.

-   :material-server: **Serve a bundle**

    `webtasks serve`

    Turn a folder of recipes into a long-running HTTP API. Best for apps and
    services that call automations on demand.

-   :material-package-variant-closed: **Build a bundle**

    `webtasks bundle ./dir out.zip`

    Package your recipes into one deployable zip. Best for shipping to a
    server.

</div>

Everything below uses these three commands. Run `webtasks help` any time to see
them.

---

## Step 0 — confirm your build

If you built from source you have a binary in the current directory (or
`build/webtasks` if you used the dev command):

```bash
go build -o webtasks ./cmd/webtasks
```

Put it somewhere on your `PATH` so you can call it from anywhere:

```bash
sudo mv webtasks /usr/local/bin/        # or: mv webtasks ~/bin/
webtasks version                        # → webtasks dev
webtasks help                           # full command list
```

!!! warning "You also need Chrome"
    webtasks drives an **installed** Chrome or Chromium through the DevTools
    Protocol — it doesn't ship a browser. Install Google Chrome, Chromium, or
    use the `chromedp/headless-shell` image in containers. (`ffmpeg` is only
    needed for MP4 recording.)

---

## Path A — run a single recipe (start here)

This is the fastest way to see something happen. Grab a demo recipe from the
repo and run it:

```bash
git clone --depth 1 https://github.com/olivierdevelops/webtasks ~/webtasks
webtasks run ~/webtasks/capy/demos/01-basics-title.webtask
```

Chrome launches **headless**, opens `example.com`, reads the page, and prints
JSON to stdout:

```json
{
  "page": {
    "title": "Example Domain",
    "heading": "Example Domain",
    "body": "This domain is for use in documentation examples …"
  }
}
```

Progress messages (`[status] …`) go to **stderr**, so the JSON on stdout stays
clean and pipeable:

```bash
webtasks run ~/webtasks/capy/demos/01-basics-title.webtask 2>/dev/null > result.json
```

### Pass inputs

Recipes can declare inputs. Set them with `--input` (repeatable) or `--json`:

```bash
# one input at a time
webtasks run ~/webtasks/capy/demos/12-search-hn-search.webtask --input q=golang

# a whole JSON object (handy for paths, lists, nested values)
webtasks run ~/webtasks/capy/demos/20-rendering-pdf.webtask \
  --json '{"url":"https://news.ycombinator.com","path":"/tmp/hn.pdf"}'
```

### Watch it happen

Set `WEBTASKS_HEADLESS=false` to open a real, visible Chrome window — invaluable
while you're figuring out selectors:

```bash
WEBTASKS_HEADLESS=false webtasks run ~/webtasks/capy/demos/02-basics-wait-click.webtask
```

!!! note "`.webtask` vs `.yaml`"
    A `.webtask` recipe is transpiled to YAML using the grammar baked into the
    binary — this needs the [`capy`](https://github.com/olivierdevelops/capy)
    CLI on your `PATH`. A `.yaml` task runs with **no extra tools**, so
    `webtasks run demo/tasks/basics/title.yaml` works out of the box.

---

## Path B — serve a bundle over HTTP

When you want automations available on demand (from an app, a backend, a
script), run the server. It loads a **bundle** — a folder of recipes — and
exposes each one as an HTTP endpoint.

```bash
WEBTASKS_BUNDLE=~/webtasks/demo webtasks
```

You'll see it boot:

```
[webtasks] starting on 127.0.0.1:8765
[webtasks] bundle: /home/you/webtasks/demo (dir)
[webtasks] registered tasks:
  - basics/title (pool=default)
  - search/duckduckgo (pool=default)
  …
[webtasks] listening on http://127.0.0.1:8765
```

Leave it running and open a second terminal.

### 1. Check it's healthy

```bash
curl -s http://127.0.0.1:8765/health | python3 -m json.tool
```

```json
{ "ok": true, "pools": { "default": { "size": 3, "available": 3 } }, "taskCount": 38 }
```

### 2. See what you can call

```bash
curl -s http://127.0.0.1:8765/tasks | python3 -m json.tool
```

Each task lists its name and input schema. The endpoint for a task is its name:
`basics/title` → `POST /tasks/basics/title`.

### 3. Call a task

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

Inputs go in the JSON body:

```bash
curl -s -X POST http://127.0.0.1:8765/tasks/search/duckduckgo \
  -H 'Content-Type: application/json' \
  -d '{"q":"golang"}' | python3 -m json.tool
```

### 4. Stream live progress (SSE)

Ask for an event stream and watch a task report progress as it runs:

```bash
curl -N -X POST http://127.0.0.1:8765/tasks/streaming/progress \
  -H 'Accept: text/event-stream' \
  -H 'Content-Type: application/json' -d '{}'
```

!!! tip "Change the address"
    `WEBTASKS_HOST` and `WEBTASKS_PORT` move the server (default
    `127.0.0.1:8765`). Set `WEBTASKS_HOST=0.0.0.0` to accept connections from
    other machines.

→ Full endpoint contract: [HTTP API](http-api.md)

---

## Path C — make your own recipes

### Write a recipe

A recipe is a `.webtask` file. Create `my-recipes/tasks/quotes.webtask`:

```capy
task "quotes"
    pool default
    timeout 20000
    transport rest

    goto "https://quotes.toscrape.com"
    wait until ".quote" timeout 10000

    extract quotes from ".quote" repeat
        text   text ".text"
        author text ".author"
    end
end
```

Run it directly to check it works:

```bash
webtasks run my-recipes/tasks/quotes.webtask
```

Iterate freely — edit the file and re-run. → [Writing tasks](writing-tasks.md)
covers every keyword, and [Actions reference](actions.md) lists every step.

### Bundle it

A **bundle** is the deployable unit. `webtasks bundle` transpiles every
`.webtask` to YAML and zips it together with any `scripts/` and config:

```bash
webtasks bundle ./my-recipes dist/bundle.zip
```

```
[webtasks] bundled 1 recipe(s) + 0 file(s) -> dist/bundle.zip
[webtasks] run it with: WEBTASKS_BUNDLE=dist/bundle.zip webtasks
```

A bundle directory typically looks like:

```
my-recipes/
├── tasks/
│   ├── quotes.webtask        # your recipes
│   └── pool.yaml             # optional: window pool sizes
├── scripts/                  # optional: reusable JS modules
└── static-mounts.yaml        # optional: expose a folder over HTTP
```

### Deploy it

Copy the binary and the zip to any host with Chrome, then:

```bash
WEBTASKS_BUNDLE=/srv/bundle.zip webtasks
```

The same binary serves **any** bundle — swap the zip to change behaviour, no
rebuild. → [Deployment](deploy.md) for pools, secrets, and production config.

---

## Configuration cheat sheet

| Variable | Applies to | Meaning |
|---|---|---|
| `WEBTASKS_BUNDLE` | serve | Bundle directory or `.zip`. |
| `WEBTASKS_HOST` / `WEBTASKS_PORT` | serve | Bind address (default `127.0.0.1:8765`). |
| `WEBTASKS_HEADLESS` | serve, run | `true`/`false`. `run` defaults to headless; the server defaults to visible. |
| `WEBTASKS_DOWNLOADS_DIR` | serve, run | Where downloaded files are written. |
| `WEBTASKS_PROFILE_DIR` | serve, run | Where persistent Chrome profiles live. |

---

## Troubleshooting

??? question "`chrome failed to start` / nothing happens"
    Chrome or Chromium isn't installed or isn't on `PATH`. Install a browser, or
    run with `WEBTASKS_HEADLESS=false` to see what Chrome is doing.

??? question "`capy CLI not found on PATH`"
    You ran `webtasks run` on a `.webtask` file without the `capy` transpiler
    installed. Either install [`capy`](https://github.com/olivierdevelops/capy),
    or run a `.yaml` task (or `webtasks bundle` on a machine that has `capy`,
    then ship the zip — the server never needs `capy`).

??? question "`address already in use`"
    Another process holds port 8765. Pick another: `WEBTASKS_PORT=9000 webtasks`.

??? question "`unknown task: …` when calling the API"
    The name must match the recipe's `task "…"` exactly (it's the URL path after
    `/tasks/`). `curl localhost:8765/tasks` lists the real names.

??? question "Empty extract results"
    The selector matched nothing, or the page hadn't loaded yet. Add a
    `wait until "<selector>"` before the `extract`, and watch with
    `WEBTASKS_HEADLESS=false`.

---

## Where to go next

- **[Browse the examples](demos/index.md)** — 38 copy-paste recipes
- **[Writing tasks](writing-tasks.md)** — the full `.webtask` language
- **[Actions reference](actions.md)** — every available step
- **[How it works](how-it-works.md)** — pools, hot-reload, the request lifecycle
- **[Deployment](deploy.md)** — ship a bundle to production
