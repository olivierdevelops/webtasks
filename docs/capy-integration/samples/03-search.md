# Sample 3 — DuckDuckGo search

**Complexity:** low · **Features:** `input` schema, URL templating

---

## Capy source

```capy
task "search/duckduckgo"
    pool default
    timeout 30s
    transport rest

    input q string required doc "Search query"

    goto "https://duckduckgo.com/?q={{q}}"
    wait-for "article[data-testid='result']" timeout 10s

    extract results from "article[data-testid='result']" repeat:
        title text "h2"
        link  attr href on "a"
    end
end
```

---

## Input binding

Caller:

```bash
executor call search/duckduckgo '{"q":"chromedp golang"}'
```

`{{q}}` in transpiled YAML resolves identically to hand-written tasks.

---

## Optional defaults

```capy
input q string default "webtasks" doc "Search query"
```

Or inline fallback in URL (still YAML templating):

```capy
goto "https://duckduckgo.com/?q={{q|or:webtasks}}"
```

---

Next: [Sample 4 — Form fill →](04-form-fill.md)
