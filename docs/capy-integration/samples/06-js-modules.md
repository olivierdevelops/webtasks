# Sample 6 — JS modules

**Complexity:** medium · **Features:** `fn:` module reference, typed module paths

---

## Capy source

```capy
task "js-modules/page-stats"
    pool default
    timeout 20s
    transport rest

    input url string default "https://example.com"

    goto "{{url}}"
    wait-for "body" timeout 10s

    js stats fn "demo/page-stats.js" args []
end
```

Library validates `demo/page-stats.js` against `type ModulePath`.

---

## Inline vs module

=== "Inline (small scripts)"

    ```capy
    js title script:
        return { title: document.title, links: document.links.length };
    end
    ```

=== "Module (reusable)"

    ```capy
    js stats fn "demo/page-stats.js" args ["{{selector}}"]
    ```

Modules stay in `scripts/` — Capy does not inline JS; it only generates YAML
pointing at bundle paths.

---

Next: [Sample 7 — Control flow →](07-control-flow.md)
