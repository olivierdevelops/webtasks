# CLI & commands

Everything you do runs through one binary: **`webtasks`**. It can serve the
HTTP API, run a single recipe from the terminal, or package a folder of recipes
into a deployable bundle.

```bash
webtasks                          # start the HTTP server (bundle from WEBTASKS_BUNDLE)
webtasks serve                    # same as above, explicit
webtasks run <file> [opts]        # run one recipe once and print JSON
webtasks bundle <dir> [out.zip]   # package a directory into a runnable bundle
webtasks version                  # print version
```

---

## `webtasks run` — run one file

The fastest way to try a recipe: point `webtasks run` at a `.webtask` (or a
`.yaml` task) and it executes once, prints the result as JSON to stdout, and
exits. Progress events stream to stderr, so you can pipe the JSON cleanly.

```bash
# run a recipe (Chrome launches headless by default)
webtasks run recipes/title.webtask

# pass inputs
webtasks run recipes/search.webtask --input q=golang
webtasks run recipes/pdf.webtask --json '{"url":"https://news.ycombinator.com","path":"/tmp/hn.pdf"}'

# watch it happen in a real browser window
WEBTASKS_HEADLESS=false webtasks run recipes/title.webtask
```

| Option | Meaning |
|---|---|
| `--input k=v` | Set one input. Repeatable. |
| `--json '{...}'` | Set inputs from a JSON object. |

`.webtask` sources are transpiled with the grammar baked into the binary; this
needs the [`capy`](https://github.com/olivierdevelops/capy) CLI on your PATH. A
`.yaml` task runs with no extra tools. JS modules referenced by a recipe are
loaded from a `scripts/` folder next to the file.

---

## `webtasks bundle` — package a directory

A **bundle** is what the server loads: a directory (or zip) of recipes, JS
modules, and config. `webtasks bundle` zips a source folder, transpiling every
`.webtask` to YAML on the way in and copying everything else verbatim.

```bash
webtasks bundle ./my-recipes                 # -> my-recipes.zip
webtasks bundle ./my-recipes dist/bundle.zip # custom output path
```

Then run the server against it:

```bash
WEBTASKS_BUNDLE=dist/bundle.zip webtasks
```

→ [Deployment](deploy.md) for the full bundle layout and config files.

---

## `webtasks serve` — run the HTTP API

The default. Reads a bundle from `WEBTASKS_BUNDLE` and serves every recipe as an
HTTP endpoint. Call tasks with any HTTP client:

```bash
WEBTASKS_BUNDLE=./my-bundle webtasks &

curl -s localhost:8765/health   | python3 -m json.tool   # pool status + task count
curl -s localhost:8765/tasks    | python3 -m json.tool   # every task + its schema

curl -s -X POST localhost:8765/tasks/basics/title -d '{}'
curl -s -X POST localhost:8765/tasks/search/duckduckgo -d '{"q":"golang"}'

# stream live progress as Server-Sent Events
curl -N -X POST localhost:8765/tasks/streaming/progress \
  -H 'Accept: text/event-stream' -d '{}'
```

→ [HTTP API](http-api.md) for the full contract.

---

## Environment

| Variable | Used by | Meaning |
|---|---|---|
| `WEBTASKS_BUNDLE` | serve | Bundle directory or `.zip`. |
| `WEBTASKS_HOST` / `WEBTASKS_PORT` | serve | Bind address (default `127.0.0.1:8765`). |
| `WEBTASKS_HEADLESS` | serve, run | `true`/`false`. `run` defaults to headless; the server defaults to visible. |
| `WEBTASKS_DOWNLOADS_DIR` | serve, run | Where downloaded files land. |
| `WEBTASKS_PROFILE_DIR` | serve, run | Where persistent Chrome profiles live. |

---

!!! note "Contributing to webtasks itself?"
    The repo also ships a `commands.yaml` of developer shortcuts (build, lint,
    release packaging) run via the `capy`-based command runner. Those are for
    hacking on the engine and are documented in
    [`commands.yaml`](https://github.com/olivierdevelops/webtasks/blob/main/commands.yaml).
    You never need them to *use* webtasks.
