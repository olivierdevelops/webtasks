# concio bundle

A ready-to-load config bundle for scraping a logged-in Concio (Starise IM)
account into the JSON format that
[rag-ingestion / concio_message_reader](https://github.com/) consumes:

```
<owner>/                         # e.g. 2000086861/
├── chats/<YYYY>/<MM>/<DD>/
│   └── chat_<owner>_<peer>_<ts>_<mo|mt>_<msgIdHi>_<msgIdLo>.json
├── files/                       # decrypted attachment bytes (one file per blob)
└── users_mapping.json
groups_mapping.json
```

The Go server drives Chrome; the Python client in
[clients/concio_extract/extract_messages.py](../clients/concio_extract/extract_messages.py)
orchestrates the per-chat sweeps and writes the artefacts to disk.

## What's in here

```
concio/
├── tasks/
│   ├── pool.yaml                  # concio pool, size 1 (single-session app)
│   └── concio/
│       ├── setup.yaml             # login (idempotent)
│       ├── list-chats.yaml        # sidebar chat list
│       ├── list-contacts.yaml     # contacts directory ("People")
│       ├── list-groups.yaml       # groups directory
│       ├── get-messages.yaml      # open chat + scroll-to-top + extract messages
│       └── capture-files.yaml     # capture encrypted attachments via blob hook
├── scripts/concio/
│   ├── navigate-login.js
│   ├── check-logged-in.js
│   ├── login.js                   # fills the form using {{CONCIO_PASSWORD}}
│   ├── open-chat-by-name.js
│   ├── install-download-hook.js   # patches URL.createObjectURL to capture blobs
│   ├── poll-captures.js
│   ├── click-all-files.js
│   ├── file-bubbles-info.js
│   └── dump-chat-panel.js
├── static-mounts.yaml             # /downloads + /files mounts
└── secrets.yaml                   # CONCIO_PASSWORD (the account password)
```

## Run it

```bash
# 1. Boot the server with this bundle.
WEBTASKS_BUNDLE=$(pwd)/concio executor server &
# (under `sm exec --` so CONCIO_PASSWORD / other secrets resolve from your vault)

# 2. Log in. setup.yaml is idempotent — re-running while already logged in is a no-op.
executor call concio/setup

# 3. Peek at what the sidebar shows.
executor call concio/list-chats

# 4. Pull every chat's full history + mappings + write to .ignore/data/.
executor concio-extract

# Single chat:
executor concio-extract "Nicholas Huang"
```

For attachments, run `capture-files` per chat (the Python client doesn't
yet drive this automatically):

```bash
executor call concio/capture-files '{"peerName":"Nicholas Huang"}'
```

Files land in the directory you pass as `outDir` (default
`build/downloads/captured`) and are visible at
`http://127.0.0.1:8765/files` via the configured static mount.

## How it works end-to-end

1. **`concio/setup`** — navigates to the login page if not already there,
   fills `#loginUserInp` / `#loginPassInp` (password from `{{CONCIO_PASSWORD}}`),
   submits, waits for `.chat-list-inner` to appear. Re-running while
   already authenticated short-circuits.

2. **`concio/list-{chats,contacts,groups}`** — straight `extract` against
   the relevant left-sidebar list. Returns name + last-message preview +
   hidden epoch span (for ordering).

3. **`concio/get-messages`** — JS `open-chat-by-name` dispatches a real
   isTrusted click sequence on the row whose `.name` text matches (Concio
   ignores synthesised JS-only clicks). Then `scroll-until-stable` runs
   the scroll-up loop on `.chats.chat-content-scroll` until Concio stops
   prepending older messages. Finally one `extract` over every `.chatList`
   row capturing both directions (`chat-left-…` / `chat-right-…`), system
   messages, quote replies, and file references.

4. **`concio/capture-files`** — installs a JS hook that monkey-patches
   `URL.createObjectURL` to capture every Blob Concio decrypts in the
   page (`window.__webtasks_captures`). After scrolling history into the
   DOM and clicking each file bubble natively via CDP `MouseClickNode`,
   the `save-captures-to-dir` action drains the buffer and writes each
   blob to `<outDir>/<basename>` server-side.

5. **`clients/concio_extract/extract_messages.py`** — calls (1) → (4) in
   order, parses message dom IDs (`_msgId_<ts>_<peer>_<seq>`), groups by
   date, and writes per-message JSON files in the scanner-compatible
   filename layout. Group chats get synthetic `groupNNN` peer ids per the
   rag-ingestion convention.

## Known limitations

- One Chrome window per Concio account (pool size 1). Concio's session
  cookies + IndexedDB key material are tied to the browser profile.
- `capture-files` reliably captures the **first** download in a session,
  then Concio tends to throttle re-decryption of subsequent files in the
  same chat. The Java predecessor had the same constraint. Workaround:
  reload the SPA between chats (`navigate-login.js` does this if the
  hook isn't present), or run one capture per session.
- Group chat ids use Concio's internal `<userid>G<roomid>` format. The
  Python client synthesises `groupNNN` ids satisfying the pipeline
  scanner's "starts with 'group'" requirement.

## Operate it

```bash
WEBTASKS_BUNDLE=$(pwd)/concio executor server   # start
executor health                                  # pool status
executor list-tasks                              # what's registered
executor call concio/list-chats                  # quick sanity
executor concio-extract                          # full sweep
executor call concio/capture-files '{"peerName":"…"}'
```

`executor concio-extract` is the workflow wrapper; the underlying
HTTP calls are all `POST /tasks/concio/…`.
