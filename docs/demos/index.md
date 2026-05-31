# Example catalogue

The [`demo/`](https://github.com/olivierdevelops/webtasks/tree/main/demo) bundle
contains **38 runnable tasks** across 11 categories. Each is a self-contained
`.webtask` recipe you can read in under a minute.

---

## Run any example

```bash
# Start the server with the demo bundle
WEBTASKS_BUNDLE=$(pwd)/demo webtasks &

# List everything
curl -s http://127.0.0.1:8765/tasks | python3 -m json.tool

# Run any task by name
curl -s -X POST http://127.0.0.1:8765/tasks/basics/title \
  -H 'Content-Type: application/json' -d '{}'

# Stream live progress with SSE
curl -N -X POST http://127.0.0.1:8765/tasks/streaming/progress \
  -H 'Accept: text/event-stream' -d '{}'
```

The server **hot-reloads** the bundle on every request — edit a recipe and
re-call it, no restart.

---

## Browse by category

<div class="grid cards" markdown>

- :material-rocket-launch-outline:{ .lg .middle } **[Basics](basics.md)** · 5 tasks

    ---

    `title` · `screenshot` · `inline-js` · `save-html` · `wait-then-click`

- :material-spider-web:{ .lg .middle } **[Crawl & scrape](crawl.md)** · 5 tasks

    ---

    `hackernews-top` · `github-trending` · `wikipedia-toc` · `trending-papers` · `quotes-paginated`

- :material-magnify:{ .lg .middle } **[Search](search.md)** · 2 tasks

    ---

    `duckduckgo` · `hn-search`

- :material-cursor-default-click-outline:{ .lg .middle } **[Interaction](interaction.md)** · 2 tasks

    ---

    `form-fill` · `scroll-feed`

- :material-radio-tower:{ .lg .middle } **[Streaming (SSE)](streaming.md)** · 1 task

    ---

    `progress`

- :material-language-javascript:{ .lg .middle } **[JS modules](js-modules.md)** · 3 tasks

    ---

    `meta-tags` · `page-stats` · `all-links`

- :material-download:{ .lg .middle } **[Downloads](downloads.md)** · 1 task

    ---

    `grab-image`

- :material-file-pdf-box:{ .lg .middle } **[Rendering](rendering.md)** · 5 tasks

    ---

    `pdf` · `snapshot` · `fullpage-shot` · `html-to-pdf` · `emulate-dark`

- :material-lan-connect:{ .lg .middle } **[Network](network.md)** · 4 tasks

    ---

    `capture` · `cookies` · `console` · `idle`

- :material-server-network:{ .lg .middle } **[Backend & export](backend.md)** · 4 tasks

    ---

    `http-get` · `http-post` · `export-csv` · `export-formats`

- :material-sitemap-outline:{ .lg .middle } **[Control flow](control.md)** · 5 tasks

    ---

    `call` · `loop` · `loop-fn` · `await-js` · `record-step`

- :material-movie-open-outline:{ .lg .middle } **[Recording](recording.md)** · 1 task

    ---

    `record`

- :material-briefcase-outline:{ .lg .middle } **[Real-world: Concio](concio.md)** · 10+ tasks

    ---

    A production logged-in scrape: secrets, persistent sessions, blob capture.

</div>

---

## Suggested learning path

Follow this order if you're new to webtasks:

1. **[Basics → title](basics.md#title)** — smallest possible task
2. **[Basics → screenshot](basics.md#screenshot)** — capture a PNG
3. **[Crawl → hackernews-top](crawl.md#hackernews-top)** — list extraction
4. **[Search → duckduckgo](search.md#duckduckgo)** — input templating
5. **[Interaction → form-fill](interaction.md#form-fill)** — typing and clicking
6. **[Streaming → progress](streaming.md)** — live SSE events
7. **[Control → call](control.md#call)** — compose tasks
8. **[Recording → record](recording.md)** — GIF screencast

---

## Hot-reload

The server reloads recipes from the bundle on **every request**. Edit any
`.webtask` file under `demo/tasks/`, then immediately re-run — no restart needed.

The demo pool runs **3 concurrent windows** so several tasks can run in parallel.
→ [Pools & sessions](../deploy.md#window-pools-sessions)

---

## Add your own

```bash
cp demo/tasks/basics/title.webtask demo/tasks/basics/my-task.webtask
# edit the task name + steps, then call it
curl -s -X POST localhost:8765/tasks/basics/my-task -d '{}'
```

Learn the language: [Writing tasks](../writing-tasks.md) ·
[Actions reference](../actions.md) · [Recipes](../cookbook.md)
