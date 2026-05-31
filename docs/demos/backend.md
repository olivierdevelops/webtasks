# Backend & export demos

Four tasks for outbound HTTP (no browser) and structured data export.

---

## http-get

Fetch a URL via Go's HTTP client — no Chrome window needed.

```bash
executor call backend/http-get
executor call backend/http-get '{"url":"https://api.github.com/repos/chromedp/chromedp"}'
```

**Concepts:** `http-get`, mixing browser and non-browser steps in one bundle.

Use when you've discovered a JSON API via [Network → capture](network.md).

---

## http-post

POST JSON to an endpoint and return the response.

```bash
executor call backend/http-post
```

=== "Pattern"

    ```yaml
    - run: http-post
      as: response
      params:
        url: "https://httpbin.org/post"
        headers: { "Content-Type": "application/json" }
        body: '{"key": "{{value}}"}'
    ```

**Concepts:** webhook triggers, API integration without browser overhead.

---

## export-csv

Extract data and export as CSV.

```bash
executor call backend/export-csv
```

**Concepts:** `export` action with `format: csv`, server-side file generation.

---

## export-formats

Export the same data as CSV, NDJSON, and Markdown in one task.

```bash
executor call backend/export-formats
```

=== "Export action"

    ```yaml
    - run: export
      params:
        format: "ndjson"    # csv | ndjson | markdown
        path: "/tmp/out.ndjson"
        data: "{{items}}"
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
- [Cookbook §11](../cookbook.md#11-ship-a-deployment-bundle) — ship exports in production
