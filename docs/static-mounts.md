# Static mounts

The server can map URL prefixes to local directories — for listing and serving
files that tasks produce (downloads, captured blobs, generated PDFs). The mount
table is read at startup from the bundle's `static-mounts.yaml`; **no URLs are
hardcoded in Go**.

Implemented by `io/staticmounts` ([routes.go](../internal/io/staticmounts/routes.go)),
loaded by `loadMounts` ([app.go](../internal/orchestrator/app.go)).

---

## Declaring mounts

`<bundle>/static-mounts.yaml`:

```yaml
mounts:
  - prefix: "/downloads"
    dir: "${WEBTASKS_DOWNLOADS_DIR:-build/downloads}"
    list: true
    serve: true
    recursive: true
  - prefix: "/captures"
    dir: "${HOME}/Documents/captures"
    list: true
    serve: false          # discovery only — list, but don't serve bytes
```

| Field | Default | Notes |
|---|---|---|
| `prefix` | — | URL prefix. A leading `/` is added if missing; trailing slashes trimmed. |
| `dir` | — | Local directory. Supports `${ENV}` and `${ENV:-default}` expansion (below). |
| `list` | `false` | Register `GET <prefix>` → JSON directory listing. |
| `serve` | `false` | Register `GET <prefix>/<path>` → stream the file. |
| `recursive` | `false` | Whether the listing walks subdirectories. |

The file is optional. Mounts are registered in declaration order.

---

## Listing: `GET <prefix>`

When `list: true`:

```bash
curl -s http://127.0.0.1:8765/downloads
```

```json
{
  "ok": true,
  "mount": "/downloads",
  "dir": "/abs/path/to/build/downloads",
  "count": 2,
  "entries": [
    { "name": "a.pdf", "url": "/downloads/a.pdf", "size": 12345, "mtime": 1716393600000 }
  ]
}
```

- `recursive: true` walks subdirectories; `name`/`url` are then slash-joined
  relative paths.
- Entries are sorted by name. A missing or non-directory `dir` yields an empty
  `entries` list (not an error).

---

## Serving: `GET <prefix>/<path>`

When `serve: true`, files are streamed with a guessed `Content-Type` (from the
extension, via `mime.TypeByExtension`). Directories and missing files return
404.

```bash
curl -s http://127.0.0.1:8765/downloads/report.pdf -o report.pdf
```

### Path-traversal safety

The serve handler is hardened against escaping the mount root:

- The request path is `filepath.Clean`ed; any `..` prefix or absolute path is
  rejected with **403 forbidden**.
- The resolved target must remain within the mount's absolute directory (a
  prefix check with a path separator), else **403**.

So `GET /downloads/../../etc/passwd` cannot escape the mounted directory.

---

## `${ENV}` expansion

`dir:` values are expanded at startup so one `static-mounts.yaml` works across
hosts without editing:

| Form | Resolves to |
|---|---|
| `${NAME}` | `os.Getenv("NAME")`, or `""` if unset. |
| `${NAME:-default}` | `os.Getenv("NAME")`, or `default` if unset/empty. |

Only `dir:` is expanded (handled by `expandEnv` in
[envexpand.go](../internal/orchestrator/envexpand.go)). Names must match
`[A-Z_][A-Z0-9_]*`.

```yaml
dir: "${WEBTASKS_DOWNLOADS_DIR:-build/downloads}"
```

This pairs with the per-window download directory: point a `/downloads` mount at
the same `WEBTASKS_DOWNLOADS_DIR` the browser writes to, and a
`download-each` result's `path` becomes directly fetchable over HTTP.
→ [actions.md#download-each](actions.md#download-each)
