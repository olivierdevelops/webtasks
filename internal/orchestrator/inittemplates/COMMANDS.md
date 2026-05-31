# webtasks starter — command reference

This folder is a **bundle**: a set of `.webtask` recipes (in `tasks/`) and
config (`tasks/pool.yaml`). The `webtasks` binary can run one recipe at a time,
or serve them all over HTTP.

```
.
├── tasks/
│   ├── hello.webtask    # open a page, read title + heading
│   ├── quotes.webtask   # takes an input, returns a list
│   └── pool.yaml        # window-pool sizes
└── COMMANDS.md          # this file
```

> **Prerequisites:** an installed Chrome/Chromium, and the
> [`capy`](https://github.com/olivierdevelops/capy) CLI to run `.webtask`
> sources (a built/served bundle does not need `capy`).

---

## Run one recipe

```bash
# simplest: run and print JSON
webtasks run tasks/hello.webtask

# pass an input
webtasks run tasks/quotes.webtask --input tag=humor

# pass several / structured inputs as JSON
webtasks run tasks/quotes.webtask --json '{"tag":"life"}'

# watch it in a real browser window (not headless)
WEBTASKS_HEADLESS=false webtasks run tasks/hello.webtask
```

Progress goes to stderr; the JSON result goes to stdout, so you can pipe it:

```bash
webtasks run tasks/hello.webtask 2>/dev/null > result.json
```

---

## Serve everything over HTTP

```bash
# serve this folder; each recipe becomes POST /tasks/<name>
WEBTASKS_BUNDLE=. webtasks
```

Then, from another terminal:

```bash
curl -s localhost:8765/health | python3 -m json.tool        # status + task count
curl -s localhost:8765/tasks  | python3 -m json.tool        # list tasks + schemas

curl -s -X POST localhost:8765/tasks/hello  -d '{}'
curl -s -X POST localhost:8765/tasks/quotes -d '{"tag":"humor"}'

# stream progress as Server-Sent Events
curl -N -X POST localhost:8765/tasks/quotes \
  -H 'Accept: text/event-stream' -d '{"tag":"humor"}'
```

The server **hot-reloads** recipes — edit a `.webtask` file and re-call it, no
restart needed.

---

## Package for deployment

```bash
# transpile every .webtask to YAML and zip the bundle
webtasks bundle . dist/bundle.zip

# run it anywhere (only needs the binary + Chrome; no capy)
WEBTASKS_BUNDLE=dist/bundle.zip webtasks
```

---

## Add your own recipe

1. Create `tasks/my-task.webtask`:

   ```
   task "my-task"
       pool default
       timeout 15000
       transport rest

       goto "https://example.com"
       extract page from "html"
           title text "title"
       end
   end
   ```

2. Try it: `webtasks run tasks/my-task.webtask`
3. It's automatically available as `POST /tasks/my-task` when serving.

Reusable JavaScript modules live in a `scripts/` folder and are called from a
recipe with a `js` step — see the docs below.

---

## Configuration

| Variable | Meaning |
|---|---|
| `WEBTASKS_BUNDLE` | Bundle directory or `.zip` to serve. |
| `WEBTASKS_HOST` / `WEBTASKS_PORT` | Server bind address (default `127.0.0.1:8765`). |
| `WEBTASKS_HEADLESS` | `true`/`false`. `run` defaults to headless. |
| `WEBTASKS_DOWNLOADS_DIR` | Where downloaded files are written. |

## Learn more

- Writing recipes: https://olivierdevelops.github.io/webtasks/writing-tasks/
- Every action: https://olivierdevelops.github.io/webtasks/actions/
- Examples: https://olivierdevelops.github.io/webtasks/demos/
