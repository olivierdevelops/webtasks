# Search demos

Two tasks showing how caller input drives URLs and how to combine browser
automation with JSON APIs.

---

## duckduckgo

Search DuckDuckGo with a query string from the caller.

```bash
executor call search/duckduckgo '{"q":"chromedp golang"}'
executor call search/duckduckgo '{"q":"webtasks yaml automation"}'
```

=== "Input + templated URL"

    ```yaml
    input:
      q:
        type: string
        required: true
        doc: "Search query"

    flow:
      - run: goto
        params: { url: "https://duckduckgo.com/?q={{q}}" }
      - run: wait-for
        params: { selector: "article[data-testid='result']", timeoutMs: 10000 }
      - run: extract
        as: results
        params:
          selector: "article[data-testid='result']"
          repeat: true
          fields:
            title: { kind: text, selector: "h2" }
            link:  { kind: attr, selector: "a", name: "href" }
    ```

```mermaid
flowchart LR
    Input["Caller JSON\n{\"q\": \"…\"}"]
    Template["URL template\n?q={{q}}"]
    Browser["Chrome navigates"]
    Extract["extract repeat:true"]
    Output["results array"]

    Input --> Template --> Browser --> Extract --> Output
```

**Concepts:** required inputs, URL encoding via templating, result extraction.

!!! note "URL encoding"
    For queries with spaces or special characters, the templating engine passes
    values as-is. Encode in the caller if needed, or use a `js` step to
    `encodeURIComponent()`.

---

## hn-search

Hacker News search via the Algolia JSON API — no browser scraping needed for
the search itself, but uses `js` for post-processing.

```bash
executor call search/hn-search '{"q":"go concurrency"}'
```

This demo shows the hybrid pattern:

1. `goto` an API URL or use `http-get` for pure HTTP
2. `js` to parse and reshape the JSON response
3. `return` structured results

**Concepts:** API-driven tasks, JS post-processing, when *not* to scrape the DOM.

Compare with [Backend → http-get](backend.md#http-get) for outbound HTTP without
a browser window.

---

## Pattern: input → URL → extract

Most search/scrape tasks follow this template:

```yaml
input:
  q: { type: string, required: true }

flow:
  - run: goto
    params: { url: "https://example.com/search?q={{q}}" }
  - run: wait-for
    params: { selector: ".results", timeoutMs: 10000 }
  - run: extract
    as: results
    params:
      selector: ".result-item"
      repeat: true
      fields: { … }
```

Copy this skeleton for any search endpoint.

---

## What's next?

- [Crawl demos](crawl.md) — fixed-URL list extraction
- [Templating reference](../templating.md) — `{{var|or:default}}` fallbacks
