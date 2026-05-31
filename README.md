# webtasks

[![Documentation](https://img.shields.io/badge/docs-GitHub%20Pages-indigo)](https://olivierdevelops.github.io/webtasks/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

**Browser automation as a service.** One small binary, a folder of `.webtask`
recipes, and every recipe becomes a typed HTTP endpoint.

📖 **[Full documentation → olivierdevelops.github.io/webtasks](https://olivierdevelops.github.io/webtasks/)**

Built on **chromedp** (Chrome DevTools Protocol). No chromedriver, no Selenium,
no JVM — a single static binary talks to Chrome directly.

---

## Install

```bash
curl -fsSL https://olivierdevelops.github.io/webtasks/install.sh | sh
```

The installer drops the `webtasks` binary on your `PATH` (building from source
if no prebuilt binary fits your platform). You also need **Chrome or Chromium**
installed — chromedp drives an external browser.

→ [Install guide](https://olivierdevelops.github.io/webtasks/install/)

---

## 30-second quick start

```bash
# grab the demo bundle (38 example tasks) and start the server
git clone --depth 1 https://github.com/olivierdevelops/webtasks ~/webtasks
WEBTASKS_BUNDLE=~/webtasks/demo webtasks &

# run a task
curl -s -X POST http://127.0.0.1:8765/tasks/basics/title \
  -H 'Content-Type: application/json' -d '{}' | python3 -m json.tool
```

```json
{ "ok": true, "data": { "page": { "title": "Example Domain" } } }
```

---

## A task is a recipe

Drop this in `tasks/crawl/hackernews-top.webtask` and it becomes
`POST /tasks/crawl/hackernews-top`:

```capy
task "crawl/hackernews-top"
    pool default
    timeout 20000
    transport rest

    goto "https://news.ycombinator.com"
    wait until "tr.athing" timeout 10000

    extract stories from "tr.athing" repeat
        title text ".titleline > a"
        url   attr href on ".titleline > a"
    end
end
```

The server **hot-reloads** recipes on every request — edit, re-call, no restart.

→ [Writing tasks](https://olivierdevelops.github.io/webtasks/writing-tasks/)

---

## What it does

| | |
|---|---|
| REST + SSE | Call tasks synchronously or stream live progress |
| Window pools | Bound concurrency; keep logged-in sessions alive |
| Capture | PDF, screenshots, MHTML, animated GIF / MP4 recording |
| Network | HAR capture, cookies, console, network-idle waits |
| JS modules | Reusable scripts under `scripts/` |
| Secrets | Declared, resolved at startup, never in the recipe |

The [`demo/`](demo/) bundle ships **38 runnable tasks** across 11 categories.
Real-world bundle: [`concio/`](concio/) — a logged-in scrape with secrets,
persistent sessions, and blob capture.

→ [Example catalogue](https://olivierdevelops.github.io/webtasks/demos/)

---

## HTTP surface

| Endpoint | Purpose |
|---|---|
| `GET /health` | Pool status + task count |
| `GET /tasks` | List tasks with input/output schemas |
| `POST /tasks/<name>` | Run a task → JSON response |
| `POST /tasks/<name>` + `Accept: text/event-stream` | Stream progress via SSE |
| `GET <mount>/<path>` | Static file mounts |

→ [HTTP API](https://olivierdevelops.github.io/webtasks/http-api/)

---

## Documentation

| Doc | Link |
|---|---|
| Install | [install](https://olivierdevelops.github.io/webtasks/install/) |
| How it works | [how-it-works](https://olivierdevelops.github.io/webtasks/how-it-works/) |
| Writing tasks | [writing-tasks](https://olivierdevelops.github.io/webtasks/writing-tasks/) |
| Examples (38) | [demos](https://olivierdevelops.github.io/webtasks/demos/) |
| Recipes | [cookbook](https://olivierdevelops.github.io/webtasks/cookbook/) |
| Actions reference | [actions](https://olivierdevelops.github.io/webtasks/actions/) |
| CLI & commands | [cli](https://olivierdevelops.github.io/webtasks/cli/) |
| Deployment | [deploy](https://olivierdevelops.github.io/webtasks/deploy/) |

Build the docs locally:

```bash
pip install -r requirements-docs.txt
mkdocs serve   # → http://127.0.0.1:8000
```

---

## License

**GNU General Public License v3.0** — copyleft. Derivative works must be
GPL-compatible when distributed. See [LICENSE](LICENSE).
