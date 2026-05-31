# Configuration

The server is configured entirely through environment variables and a handful
of launcher CLI flags — no config file beyond the bundle itself. Defaults come
from `Config.FromEnv()` ([config.go](../internal/orchestrator/config.go)).

---

## Environment variables

| Variable | Default | Notes |
|---|---|---|
| `WEBTASKS_HOST` | `127.0.0.1` | Bind address. Set `0.0.0.0` to accept remote connections. |
| `WEBTASKS_PORT` | `8765` | HTTP port. |
| `WEBTASKS_BUNDLE` | `./bundle-example` | Path to the config bundle — a directory or a `.zip`/`.jar`. → [bundle.md](bundle.md) |
| `WEBTASKS_DOWNLOADS_DIR` | `./build/downloads` | Root for per-window download directories (each window gets a sub-dir). |
| `WEBTASKS_HEADLESS` | `false` | `true` runs Chrome headless. Leave `false` while authoring to watch the browser. |
| `WEBTASKS_PROFILE_DIR` | `~/.webtasks/profiles` | Where persistent Chrome profiles live (for `persistent:` pools). → [pools.md](pools.md) |

Notes:

- `WEBTASKS_HEADLESS` is read as the literal string `true` — any other value is
  treated as non-headless.
- `WEBTASKS_PORT` falls back to `8765` if it isn't a valid integer.
- The `executor` commands set sensible defaults of their own (e.g. `server`
  defaults the bundle to `./demo` and headless to `true`); env vars you export
  override those. → [cli.md](cli.md)

---

## Launcher CLI flags

Beyond env vars, the binary accepts `--name=value` flags, consumed by the
**secrets loader** as the `arg` source for declared secrets:

```bash
./webtasks --CONCIO_PASSWORD='secret' --API_KEY='…'
```

Only flags matching a secret declared in `secrets.yaml` (with `arg` in its
`sources`) are meaningful. → [secrets.md](secrets.md)

---

## Typical configurations

**Local authoring** — visible browser, in-repo demo bundle:

```bash
WEBTASKS_HEADLESS=false WEBTASKS_BUNDLE=$(pwd)/demo ./build/webtasks
```

**Production / container** — headless, shipped zip bundle, all interfaces:

```bash
WEBTASKS_HOST=0.0.0.0 \
WEBTASKS_PORT=8080 \
WEBTASKS_HEADLESS=true \
WEBTASKS_BUNDLE=/opt/app/bundle.zip \
WEBTASKS_DOWNLOADS_DIR=/var/lib/webtasks/downloads \
./webtasks
```

**Persistent logged-in session** — stable profile across restarts:

```bash
WEBTASKS_PROFILE_DIR=/var/lib/webtasks/profiles \
WEBTASKS_HEADLESS=false \
WEBTASKS_BUNDLE=$(pwd)/concio \
sm exec -- ./build/webtasks         # secrets injected by sm
```

(Log in once in the visible window, then restart headless — the session
persists. → [pools.md](pools.md))

---

## Requirements

- **Go 1.22+** to build (the module targets `go 1.25.0`).
- **Chrome or Chromium** on the host running the server — chromedp drives an
  external browser and does not bundle one. For containers,
  `chromedp/headless-shell` is the standard base image.
- **`ffmpeg`** on `PATH` only if a task uses `record` with `format: mp4`; GIF
  recording is pure Go and needs nothing extra. → [actions.md#record](actions.md#record)
