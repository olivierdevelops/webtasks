# Sample 2 — Hacker News crawl

**Complexity:** low · **Features:** list extraction, `wait-for`, transforms

---

## Capy source

```capy
task "crawl/hackernews-top"
    pool default
    timeout 20s
    transport rest

    goto "https://news.ycombinator.com"
    wait-for "tr.athing" timeout 10s

    extract stories from "tr.athing" repeat:
        rank  text ".rank" trim
        title text ".titleline > a"
        url   attr href on ".titleline > a"
        site  text ".sitestr"
    end
end
```

---

## Compare to YAML

See [`demo/tasks/crawl/hackernews-top.yaml`](../../demo/tasks/crawl/hackernews-top.yaml).

The Capy version keeps selectors visually aligned with field names — easier
to scan in code review.

---

## Expected response

```json
{
  "ok": true,
  "data": {
    "stories": [
      { "rank": "1.", "title": "…", "url": "https://…", "site": "…" }
    ]
  }
}
```

---

## Validation

```bash
capy run capy/webtasks.capy tasks/crawl/hackernews-top.capy | diff - demo/tasks/crawl/hackernews-top.yaml
```

During migration, diff should be empty (modulo field ordering).

---

Next: [Sample 3 — Search →](03-search.md)
