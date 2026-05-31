# Sample 4 — Form fill (httpbin)

**Complexity:** medium · **Features:** `sendkeys`, `click`, multi-input defaults

---

## Capy source

```capy
task "interaction/form-fill"
    pool default
    timeout 30s
    transport rest

    input custname  string default "Ada Lovelace"
    input custtel   string default "555-0100"
    input custemail string default "ada@example.com"
    input comments  string default "hi from webtasks"

    goto "https://httpbin.org/forms/post"
    wait-for "form input[name='custname']" timeout 10s

    sendkeys "input[name='custname']"  keys "{{custname}}"
    sendkeys "input[name='custtel']"   keys "{{custtel}}"
    sendkeys "input[name='custemail']" keys "{{custemail}}"
    sendkeys "textarea[name='comments']" keys "{{comments}}"

    click "form button"
    wait-for "pre" timeout 10s

    extract echoed from "html":
        body text "pre"
    end
end
```

---

## Why Capy helps

Four similar `sendkeys` blocks in YAML differ only by selector and key name.
In Capy they read as a **uniform sequence** — less vertical noise, easier for
agents to append fields.

---

## Run

```bash
executor call interaction/form-fill
executor call interaction/form-fill '{"custname":"Grace","comments":"test"}'
```

---

Next: [Sample 5 — SSE streaming →](05-streaming.md)
