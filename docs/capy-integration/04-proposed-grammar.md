# 4. Proposed grammar overview

This chapter defines design goals and the **surface syntax** authors would write.
The executable grammar lives in [`grammar/webtasks.capy`](grammar/webtasks.capy).

---

## Design goals

1. **Readable top-to-bottom** — a task reads like a script: metadata, then steps.
2. **Fail before Chrome** — invalid actions, pools, or field kinds error at `capy run`.
3. **YAML-compatible output** — Phase 1 emits YAML the existing server loads unchanged.
4. **AI-friendly** — short source, bounded vocabulary, `capy docs` for reference.
5. **Progressive disclosure** — simple tasks need ~6 lines; complex tasks use blocks.

---

## Top-level shape

One `.capy` file = one task (convention mirrors one YAML file = one task).

```capy
# tasks/basics/title.capy
task "basics/title"
    pool default
    timeout 15s
    transport rest

    goto "https://example.com"
    extract page from "html":
        title   text "title"
        heading text "h1"
    end
end
```

Rules:

- Exactly one `task "…" … end` block per file (enforced by lint rule, not engine).
- Steps appear inside the task block in execution order.
- `#` comments allowed (declared in library `comments` block).

---

## Metadata statements

| Statement | Example | YAML effect |
|---|---|---|
| `pool TAG` | `pool concio` | `poolTag: concio` |
| `timeout DURATION` | `timeout 30s` | `timeoutMs: 30000` |
| `timeout MS` | `timeout 60000` | `timeoutMs: 60000` |
| `transport T` | `transport sse` | appends to `transports` |
| `setup TASK` | `setup concio/setup` | `setupTask: …` |
| `input NAME TYPE …` | see below | adds to `input:` map |

Duration suffixes: `s` (seconds), `m` (minutes), bare integer = milliseconds.

Input declaration forms:

```capy
input q string required doc "Search query"
input language string default "go"
input includeAds bool default false
```

---

## Step statements

### Simple actions (no block body)

```capy
goto "https://example.com"
wait 1s
wait-for "tr.athing" timeout 10s
sendkeys "input[name=q]" keys "{{q}}"
click "button.submit"
js result script:
    return document.title;
end
return "{{page.title}}"
call "basics/title"
emit progress "halfway" data { fraction: 0.5 }
```

### Actions with structured params

```capy
screenshot png as png_b64 selector "body"
pdf doc path "/tmp/out.pdf" format A4
http-get response url "https://api.example.com/data"
```

### Block actions

```capy
extract stories from "tr.athing" repeat:
    rank  text ".rank" trim
    title text ".titleline > a"
end

record clip format gif fps 4:
    scroll-until-stable "body" down stable 800ms max 8
    wait 500
end

capture-network har:
    goto "https://quotes.toscrape.com"
    wait-network-idle 800ms timeout 20s
end

for-each item in "{{pages}}":
    call "crawl/page" input { page: item }
end

loop max 10 while js "return document.querySelector('.next')":
    extract page_items from ".item" repeat:
        name text ".name"
    end
    click ".next"
end
```

---

## Status and debug

```capy
status "Step 2 — waiting for results"
record-step debug:          # record: true on next step only (library sugar)
    wait-for ".results"
end
```

---

## Naming convention

Task slug in `task "category/name"` becomes YAML `name:` — same as today.
File path is irrelevant to the URL; keeping `tasks/crawl/hn.capy` aligned with
`task "crawl/hn"` is recommended for navigation.

---

## Grammar versioning

Ship the library at a fixed path in every bundle:

```
bundle/
├── capy/
│   └── webtasks.capy      # pin version in comment: # webtasks-grammar 0.1.0
├── tasks/
│   └── …/*.capy
```

Server reads `WEBTASKS_CAPY_LIB` (default `./capy/webtasks.capy`). Bundles can
override for experimental grammars — same pattern as custom Chrome profiles.

---

Next: [The `webtasks.capy` library →](05-webtasks-library.md)
