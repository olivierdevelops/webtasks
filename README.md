# webtasks

[![Documentation](https://img.shields.io/badge/docs-GitHub%20Pages-indigo)](https://olivierdevelops.github.io/webtasks/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

**Browser-automation as a service.** One Go binary, a folder of YAML tasks, and
every task becomes a typed HTTP endpoint.

📖 **[Full documentation & demos → olivierdevelops.github.io/webtasks](https://olivierdevelops.github.io/webtasks/)**

Built on **chromedp** (Chrome DevTools Protocol). No chromedriver, no Selenium,
no JVM — a single ~17 MB static binary talks to Chrome directly.

---

## Why this exists

| Problem (Selenium/Java) | webtasks answer |
|---|---|
| JVM + chromedriver version-matched to Chrome on every host | Single Go ELF, ~17 MB |
| Headless download flakiness (WebDriver hacks) | Native CDP `setDownloadBehavior` |
| Config baked into the jar — rebuild for every change | External bundle (dir or zip), hot-reloaded |

---

## 30-second quick start

```bash
git clone https://github.com/olivierdevelops/webtasks.git && cd webtasks
go build -o build/webtasks ./cmd/webtasks
WEBTASKS_BUNDLE=$(pwd)/demo ./build/webtasks &

curl -s -X POST http://127.0.0.1:8765/tasks/basics/title \
  -H 'Content-Type: application/json' -d '{}' | python3 -m json.tool
```

With the `executor` helper (see [commands.yaml](commands.yaml)):

```bash
executor build && executor server &
executor call basics/title
executor call crawl/hackernews-top
executor call streaming/progress '{}' true   # live SSE
```

---

## 38 runnable demos

The [`demo/`](demo/) bundle ships **38 tasks** across 11 categories. Each is a
short YAML file you can read end-to-end.

| Category | Example command | What it shows |
|---|---|---|
| **basics/** | `executor call basics/title` | `goto` + `extract` |
| **basics/** | `executor call basics/screenshot` | viewport PNG (base64) |
| **crawl/** | `executor call crawl/hackernews-top` | list extraction |
| **crawl/** | `executor call crawl/github-trending '{"language":"go"}'` | templated URLs |
| **crawl/** | `executor call crawl/trending-papers` | smoke test (100 papers) |
| **search/** | `executor call search/duckduckgo '{"q":"golang"}'` | input-driven search |
| **interaction/** | `executor call interaction/form-fill` | `sendkeys` + click |
| **interaction/** | `executor call interaction/scroll-feed` | infinite scroll |
| **streaming/** | `executor call streaming/progress '{}' true` | SSE live events |
| **js-modules/** | `executor call js-modules/page-stats` | reusable `scripts/*.js` |
| **downloads/** | `executor call downloads/grab-image` | native download capture |
| **rendering/** | `executor call rendering/pdf` | print to PDF |
| **rendering/** | `executor call rendering/fullpage-shot` | full-page screenshot |
| **network/** | `executor call network/capture` | HAR-style capture |
| **network/** | `executor call network/cookies` | cookie read/write |
| **backend/** | `executor call backend/http-get` | outbound HTTP (no browser) |
| **backend/** | `executor call backend/export-csv` | CSV export |
| **control/** | `executor call control/call` | compose tasks with `call` |
| **control/** | `executor call control/loop` | pagination loops |
| **recording/** | `executor call recording/record` | animated GIF screencast |

→ **[Interactive demo guide with YAML, diagrams & copy-paste commands](https://olivierdevelops.github.io/webtasks/demos/)**

Real-world bundle: [`concio/`](concio/) — logged-in IM scrape with secrets,
blob capture, and persistent sessions.
→ [Concio walkthrough](https://olivierdevelops.github.io/webtasks/demos/concio/)

---

## How it works

```
HTTP client  ──POST /tasks/name──▶  webtasks server  ──▶  Chrome pool  ──▶  websites
                                       │
                                       ▼
                                  config bundle
                               tasks/*.yaml + scripts/*.js
```

A task YAML file:

```yaml
name: "crawl/hackernews-top"
poolTag: "default"
transports: ["rest"]
timeoutMs: 20000

flow:
  - run: goto
    params: { url: "https://news.ycombinator.com" }
  - run: wait-for
    params: { selector: "tr.athing", timeoutMs: 10000 }
  - run: extract
    as: stories
    params:
      selector: "tr.athing"
      repeat: true
      fields:
        title: { kind: text, selector: ".titleline > a" }
        url:   { kind: attr, selector: ".titleline > a", name: "href" }
```

The server **hot-reloads YAML on every request** — edit, re-call, no restart.

---

## HTTP surface

| Endpoint | Purpose |
|---|---|
| `GET /health` | Pool status + task count |
| `GET /tasks` | List tasks with input/output schemas |
| `POST /tasks/<name>` | Run task → JSON response |
| `POST /tasks/<name>` + `Accept: text/event-stream` | Stream progress via SSE |
| `GET <mount>/<path>` | Static file mounts |

---

## Documentation

| Doc | Link |
|---|---|
| **Site home (MkDocs)** | [olivierdevelops.github.io/webtasks](https://olivierdevelops.github.io/webtasks/) |
| Getting started | [docs/getting-started.md](docs/getting-started.md) |
| Demo catalogue (38 tasks) | [docs/demos/](docs/demos/) |
| Build your own task | [docs/build-your-own-task.md](docs/build-your-own-task.md) |
| Cookbook (12 recipes) | [docs/cookbook.md](docs/cookbook.md) |
| Actions reference | [docs/actions.md](docs/actions.md) |
| HTTP API | [docs/http-api.md](docs/http-api.md) |
| Architecture (VHCO) | [docs/architecture.md](docs/architecture.md) |

Build docs locally:

```bash
pip install -r requirements-docs.txt
mkdocs serve   # → http://127.0.0.1:8000
```

---

## The bundle

```
bundle/
├── tasks/
│   ├── pool.yaml              # window pool sizes
│   └── **/*.yaml              # one task per file → one HTTP endpoint
├── scripts/**/*.js            # JS modules (fn: references)
├── static-mounts.yaml         # optional URL → directory mounts
└── secrets.yaml               # optional declared secrets
```

Set `WEBTASKS_BUNDLE=/path/to/bundle` (directory or `.zip` — read in-place).

---

## Requirements

- **Go 1.22+** (to build)
- **Chrome or Chromium** on the server host
- **ffmpeg** (optional — for MP4 recording)

---

## License

**GNU General Public License v3.0** — copyleft, not permissive. Derivative works
must be GPL-compatible when distributed. See [LICENSE](LICENSE) and
[docs/license.md](docs/license.md).
