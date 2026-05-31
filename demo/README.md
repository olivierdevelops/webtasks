# demo bundle

A collection of **38 tasks** across 11 categories you can run against any live
internet connection. Each one is short enough to read end-to-end and was
designed to exercise a different feature of the engine.

📖 **[Full demo guide with YAML, diagrams & animations → Documentation](https://olivierdevelops.github.io/webtasks/demos/)**

## Run it

```bash
# point the server at this bundle instead of bundle-example/
WEBTASKS_BUNDLE=$(pwd)/demo ./build/webtasks &

# explore
executor list-tasks
executor call basics/title
executor call crawl/hackernews-top
executor call streaming/progress '{}' true
```

To run a single demo against the running server:

```bash
executor call <name> '<json-body>'              # sync JSON
executor call <name> '<json-body>' true         # stream Server-Sent Events
```

## Catalogue

### basics/ — engine primitives
| Task | Demonstrates |
|---|---|
| `basics/title` | minimal flow — `goto` + `extract` on `example.com` |
| `basics/screenshot` | viewport screenshot returned as base64 PNG |
| `basics/save-html` | dumping rendered HTML to a server-side path |
| `basics/inline-js` | running an inline `script:` block with templated args |
| `basics/wait-then-click` | `goto` → `wait-for` → `action(click)` → `extract` |

→ [Basics demo page](https://olivierdevelops.github.io/webtasks/demos/basics/)

### crawl/ — list extraction
| Task | Source | Demonstrates |
|---|---|---|
| `crawl/hackernews-top` | news.ycombinator.com | extracting from a flat table |
| `crawl/github-trending` | github.com/trending | template-driven URL (language, since) |
| `crawl/wikipedia-toc` | en.wikipedia.org | mixing `repeat=false` and `repeat=true` |
| `crawl/trending-papers` | huggingface.co/papers/trending | the canonical smoke test (100 rows) |
| `crawl/quotes-paginated` | quotes.toscrape.com | multi-page scraping pattern |

→ [Crawl demo page](https://olivierdevelops.github.io/webtasks/demos/crawl/)

### search/ — input → URL
| Task | Demonstrates |
|---|---|
| `search/duckduckgo` | URL-encoded `{{q}}` templating |
| `search/hn-search` | Algolia HN JSON API + post-processing in `js` |

### interaction/ — form + scroll
| Task | Demonstrates |
|---|---|
| `interaction/form-fill` | `sendkeys` + form submit on httpbin |
| `interaction/scroll-feed` | `scroll-until-stable` against an infinite-scroll demo |

### streaming/ — progress events
| Task | Demonstrates |
|---|---|
| `streaming/progress` | `status:` lines + `emit-event` over SSE |

→ [Streaming demo page](https://olivierdevelops.github.io/webtasks/demos/streaming/)

### js-modules/ — reusable JS in `scripts/demo/*.js`
| Task | Script used | Demonstrates |
|---|---|---|
| `js-modules/meta-tags` | `demo/get-meta-tags.js` | one task ↔ one named JS module |
| `js-modules/page-stats` | `demo/page-stats.js` | DOM-summary helper |
| `js-modules/all-links` | `demo/all-links.js` | passing args from YAML into a module |

### downloads/ — file handling
| Task | Demonstrates |
|---|---|
| `downloads/grab-image` | `download-each` clicking a link, capturing the PNG |

### rendering/ — PDF, screenshots, MHTML
| Task | Demonstrates |
|---|---|
| `rendering/pdf` | print page to PDF |
| `rendering/snapshot` | MHTML archival snapshot |
| `rendering/fullpage-shot` | full-page screenshot |
| `rendering/html-to-pdf` | HTML string → PDF |
| `rendering/emulate-dark` | device/dark-mode emulation |

→ [Rendering demo page](https://olivierdevelops.github.io/webtasks/demos/rendering/)

### network/ — HAR, cookies, console
| Task | Demonstrates |
|---|---|
| `network/capture` | HAR-style request capture |
| `network/cookies` | read/write cookies |
| `network/console` | browser console capture |
| `network/idle` | wait-for-network-idle |

### backend/ — HTTP + export (no browser)
| Task | Demonstrates |
|---|---|
| `backend/http-get` | outbound GET request |
| `backend/http-post` | outbound POST request |
| `backend/export-csv` | CSV export |
| `backend/export-formats` | CSV / NDJSON / Markdown export |

### control/ — composition & loops
| Task | Demonstrates |
|---|---|
| `control/call` | run another task inline |
| `control/loop` | pagination loop |
| `control/loop-fn` | loop condition from JS module |
| `control/await-js` | wait until JS returns true |
| `control/record-step` | GIF of a single step |

### recording/ — screencast
| Task | Demonstrates |
|---|---|
| `recording/record` | animated GIF (or MP4) of a flow segment |

## Try the SSE transport

The same `POST /tasks/streaming/progress` endpoint switches to Server-Sent
Events when the caller requests `Accept: text/event-stream`. With the helper:

```bash
executor call streaming/progress '{}' true
# event: status
# data: {"text":"Step 1 of 4 — navigate","data":null}
# event: progress
# data: {"text":"navigation complete","data":{"fraction":0.25}}
# …
# event: done
# data: {"ok":true,"data":{"page":{"title":"Example Domain"}}}
```

## Add your own

The `pool.yaml` is set to **three** windows so a few demos can run in
parallel. Edit `demo/tasks/...` freely — the server hot-reloads YAML on
every call, no restart needed. See [../docs/cookbook.md](../docs/cookbook.md)
for the full action vocabulary.
