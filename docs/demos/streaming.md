# Streaming (SSE) demo

One task that demonstrates live progress via Server-Sent Events.

---

## progress

A four-step flow that emits `status`, `progress`, and `done` events as it runs.

```bash
# Sync REST (waits for final JSON)
curl -s -X POST localhost:8765/tasks/streaming/progress -d '{}'

# SSE stream (live events)
curl -N -X POST localhost:8765/tasks/streaming/progress \
  -H 'Accept: text/event-stream' -d '{}'
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

## Recipe (.webtask)

```capy
task "streaming/progress"
    pool default
    timeout 30000
    transport rest
    transport sse

    status "Step 1 of 4 — navigate"
    goto "https://example.com"
    emit progress "navigation complete" data { fraction: 0.25 }

    status "Step 2 of 4 — pause briefly"
    wait 1000
    emit progress "pause finished" data { fraction: 0.5 }

    # … steps 3–4 …
end
```

**Concepts:**

- Two `transport` lines (`rest` + `sse`) opt the task into streaming
- Every `status "…"` line becomes an SSE `status` event automatically
- `emit progress "…" data {…}` sends custom events with arbitrary JSON
- The final result arrives as `event: done`

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
