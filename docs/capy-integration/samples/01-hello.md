# Sample 1 — Hello (basics/title)

**Complexity:** minimal · **Lines of DSL:** 9 · **Engine features:** `goto`, `extract`

---

## Goal

Open example.com, return title and heading — equivalent to
[`demo/tasks/basics/title.yaml`](../../demo/tasks/basics/title.yaml).

---

## Capy source

[`grammar/samples/01-hello.capy`](../grammar/samples/01-hello.capy):

```capy
task "basics/title"
    pool default
    timeout 15s
    transport rest

    status "Visiting example.com"
    goto "https://example.com"

    status "Reading title and heading"
    extract page from "html":
        title   text "title"
        heading text "h1"
        body    text "p"
    end
end
```

---

## Transpiled YAML

```yaml
name: "basics/title"
poolTag: "default"
transports: ["rest"]
timeoutMs: 15000
flow:
  - status: "Visiting example.com"
    run: goto
    params: { url: "https://example.com" }
  - status: "Reading title and heading"
    run: extract
    as: page
    params:
      selector: "html"
      repeat: false
      fields:
        title:   { kind: text, selector: "title" }
        heading: { kind: text, selector: "h1" }
        body:    { kind: text, selector: "p" }
```

---

## Run

```bash
# Transpile
capy run capy/webtasks.capy tasks/basics/title.capy

# Execute (after Go integration or manual yaml drop-in)
executor call basics/title
```

---

## What Capy improved

| YAML | Capy |
|---|---|
| 22 lines | 9 lines |
| Repeated `run:` / `params:` nesting | Flat verbs |
| `kind: text` boilerplate | `text "selector"` sugar |

---

Next: [Sample 2 — Hacker News →](02-hackernews.md)
