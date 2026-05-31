# Window pools & sessions

Every task runs inside a leased Chrome **window** drawn from a named **pool**.
Pools are how the server bounds concurrency, keeps logged-in sessions alive, and
recovers from crashed tabs.

Configured in the bundle's `tasks/pool.yaml`; implemented by `MakeWindowLease`
([makes.go](../internal/orchestrator/features/makes.go)) over the `WindowSource`
([windowsource.go](../internal/infra/chromedp/windowsource.go)).

---

## Declaring pools

`tasks/pool.yaml`:

```yaml
pools:
  default: { size: 3 }
  concio:  { size: 1, persistent: true, profile: "concio" }
```

| Field | Default | Notes |
|---|---|---|
| `size` | — | Number of Chrome windows pre-allocated for the pool. This is the pool's max concurrency. |
| `persistent` | `false` | When `true`, the pool's windows use a stable Chrome profile that survives restarts (below). |
| `profile` | the pool tag | Profile name for a persistent pool. |

A `default` pool of `size: 1` is injected automatically if you don't declare
one, so a minimal bundle needs no `pool.yaml` at all. A task picks its pool with
`poolTag:` (defaulting to `default`). → [task-definition.md](task-definition.md)

---

## Leasing & concurrency

- All of a pool's windows are **pre-allocated at startup** — the Chrome
  processes spawn during boot, so the first request is fast.
- A task run leases one window for its **entire duration** (including its
  `setupTask` prelude) and releases it when done.
- **Parallelism per pool = `size`.** Two requests against a `size: 3` pool can
  run concurrently on different windows; a fourth waits.
- When no window is free, `Acquire` blocks on a condition variable up to **30
  seconds**, then fails with `acquire timeout: <tag>`. Fix by raising `size` or
  shortening upstream tasks.
- A window is never shared by two runs simultaneously, so page state can't
  cross-talk between concurrent runs. Successive runs on the *same* window do
  inherit leftover state (cookies, localStorage) — which is exactly what
  `setupTask` and persistent profiles rely on.

Live pool occupancy is visible at `GET /health` as `{size, free, busy}` per
pool. → [http-api.md](http-api.md)

---

## Ephemeral vs persistent profiles

**Ephemeral (default).** Each window gets a throwaway Chrome profile, wiped when
the process restarts. Good for stateless scraping.

**Persistent.** A pool marked `persistent: true` backs its window with a stable
profile directory under the profile root (`profile-<name>`). A one-time manual
login persists across runs *and* server restarts — invaluable for sites you
can't script a login for (2FA, captchas).

```yaml
pools:
  myapp: { size: 1, persistent: true, profile: "myapp" }
```

- The profile root is `WEBTASKS_PROFILE_DIR`, defaulting to
  `~/.webtasks/profiles`. → [configuration.md](configuration.md)
- **Persistent pools must be `size: 1`.** Two live Chrome processes cannot share
  one profile directory. (If `size > 1`, extra windows get suffixed profile
  names `profile-<name>-1`, `-2`, … — but you almost never want this.)
- First-run flow: start the server with `WEBTASKS_HEADLESS=false`, log in
  manually in the window, then restart headless — the session is still there.

---

## Crash recovery

If a step error indicates the Chrome target is gone — `target detached`,
`target closed`, `session deleted`, `tab crashed`, `websocket: close`,
`page crashed`, or `context canceled` (`isFatalBrowserState`) — the engine:

1. Calls `lease.Recover(window)`, which tears down the dead target and spawns a
   fresh one under the **same window id** (`WindowSource.Replace`).
2. Returns an error telling the caller the session was reset.

The pool slot stays usable, but a fresh window has lost any prior session
state — so **re-run any setup/login task before retrying**. For persistent
pools, the profile on disk is intact, so a re-login may not even be needed.

---

## `setupTask` preludes

A data task can declare an idempotent prelude that runs in the same leased
window before its own flow:

```yaml
name: "concio/get-messages"
poolTag: "concio"
setupTask: "concio/setup"        # ensure-logged-in, runs first, same window
flow:
  - run: js
    params: { fn: "concio/open-chat-by-name", args: ["{{peerName}}"] }
  …
```

The setup task **must be idempotent** — a no-op when its post-condition already
holds (e.g. "if already logged in, return immediately"). The caller's inputs are
passed through to the setup task's templating. Only a single `Running setup: …`
status is emitted, to avoid drowning an SSE caller in prelude events.

This pairs naturally with a persistent pool: the profile keeps you logged in
across restarts, and `setupTask` re-establishes session state within a run if a
window was recycled.
