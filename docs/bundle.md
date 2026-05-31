# The config bundle

The Go binary contains **no configuration** — no tasks, scripts, or secrets are
compiled in. Every deployment supplies a separate **bundle**: the YAML tasks, JS
modules, and config files the server loads at startup. This is the project's
core deployment idea (the Java predecessor baked configs into a fat jar; every
change meant a rebuild).

Implemented by `infra/bundle` ([bundle.go](../internal/infra/bundle/bundle.go)).

---

## Layout

```
bundle/
├── tasks/
│   ├── pool.yaml                 # optional: window-pool sizes per tag
│   └── **/*.yaml                 # one task definition per file
├── scripts/
│   └── **/*.js                   # JS modules referenced from tasks via fn:
├── static-mounts.yaml            # optional: URL prefix → directory mounts
└── secrets.yaml                  # optional: declared runtime values
```

| Path | Loaded by | Doc |
|---|---|---|
| `tasks/**/*.yaml` (except `pool.yaml`) | The task registry — one `TaskDef` each. | [task-definition.md](task-definition.md) |
| `tasks/pool.yaml` | Window-pool configuration. | [pools.md](pools.md) |
| `scripts/**/*.js` | Resolved by `fn:`/`whileFn:`/`untilFn:` (path under `scripts/`, `.js` optional). | [actions.md#javascript](actions.md#javascript) |
| `static-mounts.yaml` | URL-prefix → directory mounts. | [static-mounts.md](static-mounts.md) |
| `secrets.yaml` | Declared startup secrets. | [secrets.md](secrets.md) |

All files except the task definitions are optional. A minimal bundle is just a
`tasks/` directory with one YAML file.

---

## Directory or zip

`bundle.Open(pathOrZip)` accepts either:

- a **directory** on disk (`os.DirFS`), or
- a **`.zip` / `.jar`** file, opened with `archive/zip` and read **in-place** —
  never extracted to a temp directory.

Both expose the same `fs.FS` interface (`ReadFile`, `Exists`, `WalkYAML`), so
the rest of the server is oblivious to which form it got. The server logs which
kind it opened at startup (`bundle: <path> (dir|zip)`).

The bundle path comes from `WEBTASKS_BUNDLE`, defaulting to `./bundle-example`
for dev convenience. → [configuration.md](configuration.md)

```bash
WEBTASKS_BUNDLE=$(pwd)/dist/bundle.zip ./webtasks      # zip, read in-place
WEBTASKS_BUNDLE=$(pwd)/my-bundle ./webtasks            # directory
```

---

## Hot-reload

The task registry is built with hot-reload **on** in server mode: `tasks/`
is re-walked and re-parsed on **every** `GET /tasks` and every task invocation.
Edit a task YAML and immediately re-run it — no restart needed.

This makes authoring tight: keep the server running, edit the file, call again.
The flip side: a YAML syntax error surfaces only when the task is next listed or
called.

> Hot-reload covers `tasks/` (definitions and pool config is read once at boot).
> Pool sizes, static mounts, and secrets are read at startup — changing those
> needs a restart. JS modules under `scripts/` are read on each `js` step, so
> they hot-reload too.

---

## Packaging & shipping

Build a portable distribution — a static binary plus a zipped bundle — that runs
on any host with Chrome:

```bash
executor bundle                 # → dist/bundle.zip (zips bundle-example/)
executor package                # → dist/webtasks (static ELF) + dist/bundle.zip
```

`executor package` builds with `CGO_ENABLED=0 -trimpath -ldflags '-s -w'` for a
small static binary (~17 MB). Ship the two artifacts together:

```bash
# on the target host (Chrome/Chromium installed):
WEBTASKS_BUNDLE=$(pwd)/bundle.zip ./webtasks
```

The same binary serves *any* deployment — point `WEBTASKS_BUNDLE` at a different
bundle to change behaviour without rebuilding. → [cli.md](cli.md)

Host requirement: Chrome or Chromium on the machine running the server (chromedp
doesn't bundle a browser). For containers, `chromedp/headless-shell` is the
standard base image.
