# Sample 7 — Control flow (call + loop)

**Complexity:** high · **Features:** task composition, pagination loop

---

## call — reuse another task

```capy
task "control/call"
    pool default
    timeout 30s
    transport rest

    call "basics/title"
    return "{{page.title}}"
end
```

Equivalent to [`demo/tasks/control/call.yaml`](../../demo/tasks/control/call.yaml).

Restrict `call` targets with `type ConcioTask` or `type TaskName` enums for
sandboxed agent bundles.

---

## loop — paginated crawl

```capy
task "crawl/quotes-paginated"
    pool default
    timeout 60s
    transport rest

    goto "https://quotes.toscrape.com"
    set all []

    loop max 10 while js "return document.querySelector('.next') !== null":
        extract page from "div.quote" repeat:
            text   text ".text"
            author text ".author"
            tags   text ".tag"
        end
        append all "{{page}}"
        click ".next"
        wait-for "div.quote" timeout 5s
    end

    return "{{all}}"
end
```

Note: `set`/`append` here are **DSL statements** transpiling to webtasks `set`
actions — library defines them separately from inner-DSL `set context.*`.

---

## record-step debugging

```capy
record-step debug:
    wait-for ".slow-widget" timeout 30s
end
```

Transpiles to `record: true` on that step — GIF in `$TMPDIR/webtasks-recordings/`.

---

Next: [Sample 8 — Rendering →](08-rendering.md)
