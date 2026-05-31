# Backend & export demos

Four tasks for outbound HTTP (no browser) and structured data export.

---

## http-get

Fetch a URL via Go's HTTP client — no Chrome window needed.

```bash
curl -s -X POST localhost:8765/tasks/backend/http-get -d '{}'
curl -s -X POST localhost:8765/tasks/backend/http-get \
  -d '{"url":"https://api.github.com/repos/chromedp/chromedp"}'
```

**Concepts:** `http-get`, mixing browser and non-browser steps in one bundle.

Use when you've discovered a JSON API via [Network → capture](network.md).

---

## http-post

POST JSON to an endpoint and return the response.

```bash
curl -s -X POST localhost:8765/tasks/backend/http-post -d '{}'
```

=== "Pattern"

    ```capy
    http-post response url "https://httpbin.org/post" body "{\"key\":\"{{value}}\"}"
    ```

**Concepts:** webhook triggers, API integration without browser overhead.

---

## export-csv

Extract data and export as CSV.

```bash
curl -s -X POST localhost:8765/tasks/backend/export-csv -d '{}'
```

**Concepts:** `export csv`, server-side file generation.

---

## export-formats

Export the same data as CSV, NDJSON, and Markdown in one task.

```bash
curl -s -X POST localhost:8765/tasks/backend/export-formats -d '{}'
```

=== "Export step"

    ```capy
    export ndjson path "/tmp/out.ndjson" data "{{items}}"   # csv | ndjson | md
    ```

**Concepts:** multiple export formats, piping extract results to files.

---

## Browser vs backend steps

```mermaid
flowchart LR
    subgraph browser["Needs Chrome"]
        goto
        extract
        click
    end
    subgraph backend["Go HTTP client"]
        http-get
        http-post
        export
    end

    browser -->|"DOM data"| backend
```

| Step type | Pool lease? | Speed |
|---|---|---|
| Browser (`goto`, `extract`, …) | Yes | Slower |
| Backend (`http-get`, `export`) | No* | Fast |

\*Backend steps still run inside a task that may hold a window — structure
tasks to minimize browser time.

---

## What's next?

- [Crawl](crawl.md) — extract data to export
- [Deployment](../deploy.md) — ship exports in production
