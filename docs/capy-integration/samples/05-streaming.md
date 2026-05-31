# Sample 5 — SSE progress streaming

**Complexity:** medium · **Features:** `status`, `emit-event`, dual transport

---

## Capy source

```capy
task "streaming/progress"
    pool default
    timeout 30s
    transport rest
    transport sse

    status "Step 1 of 4 — navigate"
    goto "https://example.com"
    emit progress "navigation complete" data { fraction: 0.25 }

    status "Step 2 of 4 — pause briefly"
    wait 1s
    emit progress "pause finished" data { fraction: 0.5 }

    status "Step 3 of 4 — read title"
    extract page from "html":
        title text "title"
    end
    emit progress "extracted title" data { fraction: 0.75 }

    status "Step 4 of 4 — done"
    wait 500
end
```

---

## SSE invocation

```bash
executor call streaming/progress '{}' true
curl -N -X POST localhost:8765/tasks/streaming/progress \
  -H 'Accept: text/event-stream' -d '{}'
```

---

## Capy vs YAML

Multiple `transport` lines replace YAML list syntax. `emit progress … data {…}`
is clearer than nested `emit-event` params maps.

---

Next: [Sample 6 — JS modules →](06-js-modules.md)
