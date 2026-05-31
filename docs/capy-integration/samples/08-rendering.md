# Sample 8 — Rendering (PDF + screenshot)

**Complexity:** medium · **Features:** artifacts, paths, base64 output

---

## PDF

```capy
task "rendering/pdf"
    pool default
    timeout 30s
    transport rest

    input url  string default "https://example.com"
    input path string default "/tmp/webtasks-demo/example.pdf"

    goto "{{url}}"
    wait-for "body" timeout 10s
    pdf pdf path "{{path}}" format A4 printBackground true
end
```

---

## Screenshot

```capy
task "basics/screenshot"
    pool default
    timeout 20s
    transport rest

    input url string default "https://example.com"

    goto "{{url}}"
    wait-for "body" timeout 10s
    screenshot png_b64 selector "."
end
```

---

## Full-page + dark mode

```capy
task "rendering/emulate-dark"
    pool default
    timeout 30s
    transport rest

    input url string default "https://example.com"

    goto "{{url}}"
    emulate dark
    screenshot shot selector "html" fullPage true
end
```

Capy groups rendering verbs in one visual cluster — easier than hunting through
YAML keys (`printBackground`, `format`, etc.) without IDE schema support.

---

Next: [Sample 9 — Concio bundle →](09-concio-bundle.md)
