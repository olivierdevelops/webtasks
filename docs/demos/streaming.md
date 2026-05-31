# Streaming (SSE) demo

One task that demonstrates live progress via Server-Sent Events.

---

## progress

A four-step flow that emits `status`, `progress`, and `done` events as it runs.

```bash
# Sync REST (waits for final JSON)
executor call streaming/progress

# SSE stream (live events)
executor call streaming/progress '{}' true
```

=== "curl SSE"

    ```bash
    curl -N -X POST http://127.0.0.1:8765/tasks/streaming/progress \
      -H 'Content-Type: application/json' \
      -H 'Accept: text/event-stream' \
      -d '{}'
    ```

=== "Python SSE"

    ```python
    import requests

    with requests.post(
        "http://127.0.0.1:8765/tasks/streaming/progress",
        json={},
        headers={"Accept": "text/event-stream"},
        stream=True,
    ) as resp:
        for line in resp.iter_lines(decode_unicode=True):
            if line:
                print(line)
    ```

---

## Event stream

<div class="diagram" markdown="1">

![SSE event stream](../assets/sse-stream.svg)

</div>

Sample output:

```
event: status
data: {"text":"Step 1 of 4 — navigate","data":null}

event: progress
data: {"text":"navigation complete","data":{"fraction":0.25}}

event: status
data: {"text":"Step 2 of 4 — pause briefly","data":null}

event: progress
data: {"text":"pause finished","data":{"fraction":0.5}}

event: status
data: {"text":"Step 3 of 4 — read title","data":null}

event: progress
data: {"text":"extracted title","data":{"fraction":0.75}}

event: status
data: {"text":"Step 4 of 4 — done","data":null}

event: done
data: {"ok":true,"data":{"page":{"title":"Example Domain"}}}
```

---

## Task YAML

```yaml
name: "streaming/progress"
poolTag: "default"
transports: ["rest", "sse"]
timeoutMs: 30000

flow:
  - status: "Step 1 of 4 — navigate"
    run: goto
    params: { url: "https://example.com" }

  - run: emit-event
    params:
      kind: "progress"
      text: "navigation complete"
      data: { fraction: 0.25 }

  - status: "Step 2 of 4 — pause briefly"
    run: wait
    params: { duration: "1_000" }

  - run: emit-event
    params:
      kind: "progress"
      text: "pause finished"
      data: { fraction: 0.5 }

  # … steps 3–4 …
```

**Concepts:**

- `transports: ["rest", "sse"]` — opt in to streaming
- Every `status:` line becomes an SSE `status` event automatically
- `emit-event` sends custom `progress` (or any kind) events with arbitrary JSON
- Final result arrives as `event: done`

---

## When to use SSE

| Use SSE when… | Use sync REST when… |
|---|---|
| Long-running tasks (minutes) | Tasks finish in seconds |
| UI shows a progress bar | Batch/cron jobs |
| User waits interactively | Fire-and-forget with polling |

Protocol details: [HTTP API → SSE](../http-api.md)

---

## What's next?

- [HTTP API](../http-api.md) — full SSE protocol
- [Recording](recording.md) — visual progress via GIF
