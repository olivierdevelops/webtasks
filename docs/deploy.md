# Deployment

Everything you need to run webtasks in production: how the server is
configured, how the bundle is laid out, how window pools bound concurrency, and
how secrets and static file mounts work.

---

## Run the server

The binary ships **no configuration**. You point it at a **bundle** — a folder
(or `.zip`) of `.webtask` tasks, JS modules, and config — with `WEBTASKS_BUNDLE`:

```bash
WEBTASKS_BUNDLE=/path/to/bundle webtasks
```

The server logs which kind of bundle it opened (`bundle: <path> (dir|zip)`) and
starts on `127.0.0.1:8765` by default.

### Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `WEBTASKS_HOST` | `127.0.0.1` | Bind address. Set `0.0.0.0` to accept remote connections. |
| `WEBTASKS_PORT` | `8765` | HTTP port. |
| `WEBTASKS_BUNDLE` | `./bundle-example` | Path to the bundle — directory or `.zip`/`.jar`. |
| `WEBTASKS_DOWNLOADS_DIR` | `./build/downloads` | Root for per-window download directories. |
| `WEBTASKS_HEADLESS` | `false` | `true` runs Chrome headless. Leave `false` while authoring to watch the browser. |
| `WEBTASKS_PROFILE_DIR` | `~/.webtasks/profiles` | Where persistent Chrome profiles live. |

### Typical configurations

=== "Local authoring"

    Visible browser, in-repo demo bundle:

    ```bash
    WEBTASKS_HEADLESS=false WEBTASKS_BUNDLE=$(pwd)/demo webtasks
    ```

=== "Production / container"

    Headless, shipped zip bundle, all interfaces:

    ```bash
    WEBTASKS_HOST=0.0.0.0 \
    WEBTASKS_PORT=8080 \
    WEBTASKS_HEADLESS=true \
    WEBTASKS_BUNDLE=/opt/app/bundle.zip \
    webtasks
    ```

=== "Persistent session"

    Stable profile across restarts — log in once, stay logged in:

    ```bash
    WEBTASKS_PROFILE_DIR=/var/lib/webtasks/profiles \
    WEBTASKS_HEADLESS=false \
    WEBTASKS_BUNDLE=$(pwd)/concio \
    webtasks
    ```

!!! info "Chrome is required"
    chromedp drives an **external** Chrome/Chromium — the binary does not bundle
    a browser. For containers, `chromedp/headless-shell` is the standard base
    image. `ffmpeg` is only needed for MP4 recording (GIF is pure Go).

---

## The bundle

A bundle is a directory (or `.zip`/`.jar`, read in-place — never extracted):

```
bundle/
├── tasks/
│   └── **/*.webtask        # one task → one HTTP endpoint
├── scripts/
│   └── **/*.js             # JS modules referenced from tasks via fn:
├── pool                    # window-pool sizes (optional)
├── static-mounts           # URL prefix → directory mounts (optional)
└── secrets                 # declared runtime values (optional)
```

| Path | Purpose |
|---|---|
| `tasks/**/*.webtask` | Each file is one task. The task slug becomes `POST /tasks/<slug>`. |
| `scripts/**/*.js` | JS modules resolved by `fn:` (path under `scripts/`, `.js` optional). |
| Pool config | Window-pool sizes per tag (see below). |
| Static mounts | URL-prefix → directory mounts (see below). |
| Secrets | Declared startup secrets (see below). |

Only `tasks/` is required. A minimal bundle is one `.webtask` file.

### Directory or zip

Both forms expose the same interface, so the server is oblivious to which it
got:

```bash
WEBTASKS_BUNDLE=$(pwd)/dist/bundle.zip webtasks   # zip, read in-place
WEBTASKS_BUNDLE=$(pwd)/my-bundle webtasks          # directory
```

### Hot-reload

Tasks are re-read on **every** request — edit a `.webtask` file and immediately
re-call it, no restart. (Pool sizes, mounts, and secrets are read once at
startup; changing those needs a restart. JS modules under `scripts/` hot-reload
per `js` step.)

### Packaging

Ship a portable distribution — the binary plus a zipped bundle — that runs on
any host with Chrome. `webtasks bundle` transpiles your `.webtask` recipes to
YAML and zips them with your scripts and config:

```bash
webtasks bundle ./my-recipes dist/bundle.zip
```

The same binary serves *any* deployment — point `WEBTASKS_BUNDLE` at a different
bundle to change behaviour without rebuilding.

---

## Window pools & sessions

Every task runs inside a leased Chrome **window** drawn from a named **pool**.
Pools bound concurrency, keep logged-in sessions alive, and recover crashed
tabs. A task picks its pool with `pool <tag>` (defaulting to `default`).

### Declaring pools

A pool config in the bundle declares each tag's settings:

| Field | Default | Notes |
|---|---|---|
| `size` | — | Number of Chrome windows pre-allocated. This is the pool's max concurrency. |
| `persistent` | `false` | When `true`, windows use a stable profile that survives restarts. |
| `profile` | the pool tag | Profile name for a persistent pool. |

A `default` pool of `size: 1` is injected automatically, so a minimal bundle
needs no pool config at all.

### Leasing & concurrency

- All of a pool's windows are **pre-allocated at startup** — the first request
  is fast.
- A run leases one window for its **entire duration** (including any `setup`
  prelude) and releases it when done.
- **Parallelism per pool = `size`.** A request beyond `size` waits up to **30 s**
  on a condition variable, then fails with `acquire timeout: <tag>`.
- A window is never shared by two runs at once, so concurrent runs can't
  cross-talk. Successive runs on the *same* window inherit leftover state
  (cookies, localStorage) — exactly what `setup` and persistent profiles exploit.

Live occupancy is at `GET /health` as `{size, free, busy}` per pool.

### Persistent profiles

A pool marked `persistent` backs its window with a stable profile directory
under `WEBTASKS_PROFILE_DIR`. A one-time manual login survives runs *and* server
restarts — invaluable for sites you can't script a login for (2FA, captchas).

- **Persistent pools must be `size: 1`** — two live Chrome processes cannot share
  one profile directory.
- First-run flow: start with `WEBTASKS_HEADLESS=false`, log in manually, then
  restart headless — the session is still there.

### Crash recovery

If a step error signals a dead target (`target detached`, `tab crashed`,
`websocket: close`, …), the engine spawns a fresh window under the same id and
tells the caller the session was reset. Re-run any `setup`/login task before
retrying. For persistent pools, the on-disk profile is intact.

### `setup` preludes

A data task can declare an idempotent prelude that runs in the same leased
window before its own steps:

```capy
task "concio/get-messages"
    pool concio
    setup "concio/setup"          # ensure-logged-in, runs first, same window

    js _ fn "concio/open-chat-by-name" args ["{{peerName}}"]
end
```

The setup task **must be idempotent** — a no-op when already satisfied (e.g.
"if logged in, return immediately"). Pairs naturally with a persistent pool.

---

## Secrets

Tasks reference sensitive values (passwords, API keys) via templating
(`{{CONCIO_PASSWORD}}`) — but the values never live in the task. The bundle
declares *what* secrets the server needs; the server resolves them at startup
from the environment, CLI args, or an interactive prompt, then publishes them as
process env vars so templating can find them.

### Declaring secrets

Each declared secret supports:

| Field | Default | Notes |
|---|---|---|
| `name` | — | The env-var name the value is published under, and the `{{name}}` token tasks use. |
| `description` | — | Shown in the interactive prompt. |
| `required` | `false` | If `true` and unresolved, the server **refuses to start**. |
| `sensitive` | `false` | Read silently when prompting; hidden in the startup audit log. |
| `default` | `""` | Fallback value if no source yields one. |
| `sources` | `["env","arg","prompt"]` | Resolution order. |

### Resolution chain

For each secret, the loader walks `sources` **in order**, taking the first
non-empty value:

| Source | How |
|---|---|
| `env` | `CONCIO_PASSWORD=… webtasks`. |
| `arg` | A launcher flag — `webtasks --CONCIO_PASSWORD=…`. |
| `prompt` | Interactive TTY prompt. Silent when `sensitive`. **Skipped without a terminal** (e.g. CI). |

If nothing yields a value, the `default` is used; a still-missing `required`
secret fails startup.

### Using a secret

Once declared and resolved, reference it like any binding — the env fallback
does the work, no `input` entry needed:

```capy
sendkeys "#password" keys "{{CONCIO_PASSWORD}}"
```

!!! tip "Keep secrets out of shell history"
    Store them in a secret manager and inject at launch (e.g. `sm exec -- webtasks`)
    so the `env` source resolves them.

---

## Static file mounts

The server can map URL prefixes to local directories — for listing and serving
files tasks produce (downloads, captured blobs, generated PDFs). The mount table
is read at startup; **no URLs are hardcoded**.

### Declaring mounts

Each mount supports:

| Field | Default | Notes |
|---|---|---|
| `prefix` | — | URL prefix. Leading `/` added if missing. |
| `dir` | — | Local directory. Supports `${ENV}` and `${ENV:-default}` expansion. |
| `list` | `false` | Register `GET <prefix>` → JSON directory listing. |
| `serve` | `false` | Register `GET <prefix>/<path>` → stream the file. |
| `recursive` | `false` | Whether the listing walks subdirectories. |

### Listing & serving

```bash
curl -s http://127.0.0.1:8765/downloads            # JSON listing (list: true)
curl -s http://127.0.0.1:8765/downloads/report.pdf -o report.pdf   # serve: true
```

```json
{
  "ok": true,
  "mount": "/downloads",
  "dir": "/abs/path/to/build/downloads",
  "count": 1,
  "entries": [
    { "name": "report.pdf", "url": "/downloads/report.pdf", "size": 12345, "mtime": 1716393600000 }
  ]
}
```

### Path-traversal safety

The serve handler is hardened: request paths are cleaned, and any `..` or
attempt to escape the mount root is rejected with **403**. So
`GET /downloads/../../etc/passwd` cannot escape the mounted directory.

### `${ENV}` expansion

`dir` values are expanded at startup so one config works across hosts:

| Form | Resolves to |
|---|---|
| `${NAME}` | the env var, or `""` if unset. |
| `${NAME:-default}` | the env var, or `default` if unset/empty. |

Point a `/downloads` mount at the same `WEBTASKS_DOWNLOADS_DIR` the browser
writes to, and a `download-each` result's path becomes fetchable over HTTP.
