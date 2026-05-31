# Sample 9 — Concio production bundle

**Complexity:** expert · **Features:** secrets, persistent pool, blob capture, task chains

This sample sketches how the [`concio/`](../../concio/) bundle would look as
Capy tasks — not a full port, but the **patterns** that matter at scale.

---

## Bundle layout

```
concio/
├── capy/
│   └── webtasks.capy          # shared grammar (or extends base)
├── tasks/
│   ├── pool.capy              # OR keep pool.yaml
│   └── concio/
│       ├── setup.capy
│       ├── list-chats.capy
│       ├── get-messages.capy
│       └── capture-files.capy
├── scripts/concio/*.js        # unchanged
├── secrets.yaml               # unchanged
└── static-mounts.yaml
```

---

## setup.capy — idempotent login

```capy
task "concio/setup"
    pool concio
    timeout 120s
    transport rest

    goto "https://app.example.com/login"
    js logged_in fn "concio/check-logged-in.js"

    # if not logged_in — library emits conditional via inner DSL
    # (or separate branch tasks; keep setup idempotent)

    js _ fn "concio/navigate-login.js"
    js _ fn "concio/login.js"          # uses {{CONCIO_PASSWORD}} from secrets
    wait-for ".chat-sidebar" timeout 30s
end
```

Secrets still resolve via `secrets.yaml` + env — Capy only authors the flow.

---

## get-messages.capy — scroll + extract

```capy
task "concio/get-messages"
    pool concio
    timeout 300s
    transport rest
    setup concio/setup

    input peerName string required doc "Chat display name"

    js _ fn "concio/open-chat-by-name.js" args ["{{peerName}}"]

    scroll-until-stable ".chat-panel" up stable 1000ms max 50

    js messages fn "concio/dump-chat-panel.js"
    return "{{messages}}"
end
```

---

## capture-files.capy — blob hook pattern

```capy
task "concio/capture-files"
    pool concio
    timeout 600s
    transport rest
    setup concio/setup

    js _ fn "concio/install-download-hook.js"

    capture-network captures:
        js _ fn "concio/click-all-files.js"
        js files fn "concio/poll-captures.js"
    end

    return "{{files}}"
end
```

---

## Why Capy at this scale

| Concern | Without Capy | With Capy |
|---|---|---|
| 10+ tasks sharing login | Copy-paste setup prelude | `setup concio/setup` one line |
| JS module paths | String typos in YAML | `type ModulePath` validation |
| Agent-authored tasks | Can invent dangerous steps | Enum-limited `call` targets |
| Onboarding | Read 780-line actions.md | `capy docs webtasks.capy` |

---

## Orchestration outside webtasks

The Python client [`concio_extract`](../../concio/README.md) still drives
per-chat sweeps. Capy tasks remain **HTTP endpoints** — only the file format
changed.

---

[← Back to index](../index.md) · [AI authoring →](../22-ai-authoring.md)
