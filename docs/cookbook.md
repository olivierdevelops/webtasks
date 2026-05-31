# webtasks cookbook

Working recipes for writing tasks, deploying the server, and extending the
action vocabulary. Read the [README](../README.md) first for the big picture.

---

## Table of contents

- [1. Write your first task](#1-write-your-first-task)
- [2. Accept inputs from the caller](#2-accept-inputs-from-the-caller)
- [3. Extract a list of records](#3-extract-a-list-of-records)
- [4. Use a JS module instead of inline `script:`](#4-use-a-js-module-instead-of-inline-script)
- [5. Open a chat by visible name (Concio pattern)](#5-open-a-chat-by-visible-name-concio-pattern)
- [6. Load history by scrolling until stable](#6-load-history-by-scrolling-until-stable)
- [7. Trigger downloads and capture the bytes](#7-trigger-downloads-and-capture-the-bytes)
- [8. Stream progress to the caller (SSE)](#8-stream-progress-to-the-caller-sse)
- [9. Declare secrets and prompt for them](#9-declare-secrets-and-prompt-for-them)
- [10. Mount a directory at a URL](#10-mount-a-directory-at-a-url)
- [11. Ship a deployment bundle](#11-ship-a-deployment-bundle)
- [12. Add a new action to the engine](#12-add-a-new-action-to-the-engine)
- [Action reference](#action-reference)
- [Templating reference](#templating-reference)
- [Troubleshooting](#troubleshooting)

---

## 1. Write your first task

Create `bundle-example/tasks/recipes/title.yaml`:

```yaml
name: "recipes/title"
poolTag: "default"
transports: ["rest"]
timeoutMs: 15000

flow:
  - run: goto
    params: { url: "https://example.com" }
  - run: extract
    as: page
    params:
      selector: "h1"
      repeat: false
      fields:
        title: { kind: text, selector: "." }
```

The server hot-reloads YAML on every request, so no restart needed.

```bash
executor call recipes/title
# → { "ok": true, "data": { "page": { "title": "Example Domain" } } }
```

Anatomy:

- `name` is the URL slug (`POST /tasks/recipes/title`).
- `poolTag` picks which browser pool to lease from (`default` is always
  available; declare more in `tasks/pool.yaml`).
- `flow` is a list of `Command`s; each runs in order.
- `as: page` puts the extract result into the response under `data.page`.

---

## 2. Accept inputs from the caller

Declare an input schema and reference it with `{{name}}` templating.

```yaml
name: "recipes/search"
poolTag: "default"
transports: ["rest"]
timeoutMs: 30000

input:
  q:
    type: string
    required: true
    doc: "Query string"

flow:
  - run: goto
    params: { url: "https://duckduckgo.com/?q={{q}}" }
  - run: wait-for
    params: { selector: "article[data-testid='result']", timeoutMs: 10000 }
  - run: extract
    as: results
    params:
      selector: "article[data-testid='result']"
      repeat: true
      fields:
        title: { kind: text, selector: "h2" }
        link:  { kind: attr, selector: "a", name: "href" }
```

```bash
executor call recipes/search '{"q":"chromedp"}'
```

Default values + the `or:` fallback work too:

```yaml
input:
  q: { type: string, required: false, default: "go" }
# In the flow you can also fall back inside a string:
# url: "https://duckduckgo.com/?q={{q|or:go}}"
```

---

## 3. Extract a list of records

`extract` with `repeat: true` runs the field-spec against each match of
`selector` and returns an array.

```yaml
- run: extract
  as: items
  params:
    selector: "ul.product-list > li"
    repeat: true
    fields:
      name:  { kind: text, selector: ".name" }
      price: { kind: text, selector: ".price", transform: trim }
      sku:   { kind: attr, selector: ".",      name: "data-sku" }
      url:   { kind: attr, selector: "a",      name: "href" }
      tags:
        kind: html
        selector: ".tags"
```

`kind:` values: `text` (default), `attr` (needs `name:`), `html`, `const`
(returns `value:`). `transform:` accepts `int`, `long`, `trim`, `lower`,
`upper`.

---

## 4. Use a JS module instead of inline `script:`

Inline `script:` blocks get unreadable past a few lines. Drop the script
into `bundle/scripts/<path>.js` and reference it by `fn:`.

`bundle/scripts/recipes/click-by-text.js`:

```js
// Click the first descendant of `arguments[1]` whose .textContent matches
// `arguments[0]`. Returns true on success, false otherwise.
const target = arguments[0];
const within = arguments[1] ? document.querySelector(arguments[1]) : document;
if (!within) return false;
const match = Array.from(within.querySelectorAll('*'))
    .find(el => el.children.length === 0 && el.textContent.trim() === target);
if (!match) return false;
const opts = { bubbles: true, cancelable: true, view: window, button: 0 };
match.dispatchEvent(new MouseEvent('mousedown', opts));
match.dispatchEvent(new MouseEvent('mouseup', opts));
match.dispatchEvent(new MouseEvent('click', opts));
return true;
```

YAML:

```yaml
- run: js
  as: clicked
  params:
    fn: "recipes/click-by-text"
    args: ["{{button}}", "form"]
```

`fn:` is the path inside `scripts/` without the `.js`. The dispatched event
trick is useful for non-Concio sites; for sites that require `isTrusted`
(most modern Vue/React apps), use the `action: click` command instead — that
goes through CDP's `Input.dispatchMouseEvent`.

---

## 5. Open a chat by visible name (Concio pattern)

When a CSS selector can't address the element (you only know its visible
text), use a small JS module to find it. For sites that gate handlers on
`isTrusted`, dispatch a real MouseEvent sequence.

`bundle/scripts/concio/open-chat-by-name.js`:

```js
const name = arguments[0];
const rows = document.querySelectorAll('.chat-list-inner');
const match = Array.from(rows).find(r => {
    const n = r.querySelector('.name');
    return n && n.textContent.trim() === name;
});
if (!match) return false;
const target = match.querySelector('.message-panel') || match;
const opts = { bubbles: true, cancelable: true, view: window, button: 0 };
target.dispatchEvent(new MouseEvent('mousedown', opts));
target.dispatchEvent(new MouseEvent('mouseup', opts));
target.dispatchEvent(new MouseEvent('click', opts));
return true;
```

`bundle/tasks/concio/get-messages.yaml`:

```yaml
input:
  peerName: { type: string, required: true }

flow:
  - run: js
    params: { fn: "concio/open-chat-by-name", args: ["{{peerName}}"] }
  - run: wait-for
    params: { selector: ".chats.chat-content-scroll", timeoutMs: 15000 }
  # … extract messages …
```

---

## 6. Load history by scrolling until stable

For infinite-scroll panels (chat history, feeds) use `scroll-until-stable`:

```yaml
- run: scroll-until-stable
  params:
    selector: ".chats.chat-content-scroll"
    direction: up               # `up` (scrollTop=0) or `down` (scrollHeight)
    stableMs: 2500              # exit when scrollHeight hasn't changed for this long
    maxIterations: 0            # 0 = unbounded
```

The action sets `scrollTop` and re-queries `scrollHeight` in a loop. Each
new prepend keeps the loop going; when the site stops loading, the height
stabilises and the action returns.

---

## 7. Trigger downloads and capture the bytes

Two patterns, depending on how the site exposes downloads:

### 7a. Normal HTTP downloads → `download-each`

The browser writes the file to the per-window download directory. The
action clicks each match natively (isTrusted) and polls the directory.

```yaml
- run: download-each
  as: downloads
  params:
    selector: "a.download-link"
    timeoutPerFileMs: 30000
```

Result: `data.downloads = [{ path, basename }, …]`.

### 7b. Client-side decryption → install a JS hook + drain

Sites that decrypt blobs in the browser before triggering download (Concio,
ProtonMail-style apps) bypass the normal download flow. Install a hook on
`URL.createObjectURL`, click each file, then drain the captures.

`bundle/scripts/concio/install-download-hook.js`:

```js
if (window.__webtasks_hookInstalled) return { alreadyInstalled: true };
window.__webtasks_hookInstalled = true;
window.__webtasks_captures = [];
window.__webtasks_nextId = 1;
const origCreate = URL.createObjectURL.bind(URL);
URL.createObjectURL = function(blob) {
    const url = origCreate(blob);
    const id = window.__webtasks_nextId++;
    const entry = { id, url, mime: blob.type || null, size: blob.size, ts: Date.now(),
                    name: null, bytesB64: null };
    window.__webtasks_captures.push(entry);
    blob.arrayBuffer().then(buf => {
        const u8 = new Uint8Array(buf);
        let bin = '', chunk = 0x8000;
        for (let i = 0; i < u8.length; i += chunk)
            bin += String.fromCharCode.apply(null, u8.subarray(i, i + chunk));
        entry.bytesB64 = btoa(bin);
    });
    return url;
};
return { installed: true };
```

Task YAML:

```yaml
flow:
  - run: js
    params: { fn: "concio/install-download-hook" }
  - run: js
    params: { fn: "concio/open-chat-by-name", args: ["{{peerName}}"] }
  - run: wait-for
    params: { selector: ".chats.chat-content-scroll", timeoutMs: 15000 }
  - run: download-each
    params:
      selector: ".chats.chat-content-scroll .chat-right-file-box"
      timeoutPerFileMs: 2500
  - run: wait
    params: { duration: "8_000" }      # let blob.arrayBuffer() finish
  - run: save-captures-to-dir
    as: saved
    params:
      dir: "{{outDir}}"
      naming: "{id}_{name}"            # available tokens: {id} {name} {ext} {mime} {size} {ts}
```

Result: `data.saved = [{ path, basename, size, mime, name, url }, …]`.

The CDP `Browser.setDownloadBehavior` call is already wired into the window
factory, so headless Chrome doesn't block normal downloads either.

---

## 8. Stream progress to the caller (SSE)

The same `POST /tasks/<name>` endpoint switches to server-sent events when
the caller sends `Accept: text/event-stream`. Every `status:` field in the
flow becomes a `status` event; the final response is a `done` event.

```bash
curl -N -X POST http://127.0.0.1:8765/tasks/examples/trending-papers \
     -H 'Accept: text/event-stream' \
     -H 'Content-Type: application/json' \
     -d '{}'

# event: status
# data: {"text":"Visiting Hugging Face trending papers","data":null}
#
# event: status
# data: {"text":"Waiting for page to settle","data":null}
#
# event: done
# data: {"ok":true,"data":{"papers":[…]}}
```

You can also emit custom events with `emit-event`:

```yaml
- run: emit-event
  params:
    kind: "progress"
    text: "Processed {{i}}/{{n}}"
    data: { fraction: 0.5 }
```

---

## 9. Declare secrets and prompt for them

`bundle/secrets.yaml`:

```yaml
secrets:
  - name: CONCIO_PASSWORD
    description: "Concio account password"
    required: true
    sensitive: true              # silent input from TTY
    sources: ["env", "arg", "prompt"]
  - name: API_KEY
    required: false
    sources: ["env", "arg"]
    default: ""
```

Resolution chain walks `sources` in order:

- **env** — `CONCIO_PASSWORD=… ./webtasks`
- **arg** — `./webtasks --CONCIO_PASSWORD=…`
- **prompt** — interactive TTY (silent for sensitive secrets)

Resolved values are exported into the process env, so any task YAML can
reference them via `{{CONCIO_PASSWORD}}`.

The recommended secure setup: keep secrets in [`sm`](https://github.com/) and
launch via `sm exec -- ./webtasks`. The `executor server` command already
does that.

---

## 10. Mount a directory at a URL

`bundle/static-mounts.yaml` declares routes generically — Java/Go code knows
nothing about specific URLs:

```yaml
mounts:
  - prefix: "/downloads"
    dir: "${WEBTASKS_DOWNLOADS_DIR:-build/downloads}"
    list: true                       # GET /downloads -> JSON listing
    serve: true                      # GET /downloads/<file> -> bytes
    recursive: true
  - prefix: "/captures"
    dir: "${HOME}/Documents/captures"
    list: true
    serve: false                     # discovery only, no serving
```

`${ENV}` and `${ENV:-default}` placeholders are expanded at startup. Reorder
or add mounts without touching Go.

---

## 11. Ship a deployment bundle

The Go binary contains no config files. Deploy the universal binary + a
config bundle separately:

```bash
executor package
# → dist/webtasks                 (static ELF, ~17 MB, CGO_ENABLED=0)
# → dist/bundle.zip               (everything from bundle-example/)

# On the target host:
WEBTASKS_BUNDLE=$(pwd)/bundle.zip ./webtasks
```

The same binary supports any deployment — just supply a different bundle.
The zip is read in-place via `archive/zip` (no extraction to disk).

Host requirements: Chrome or Chromium installed (chromedp doesn't bundle a
browser). For containers, `chromedp/headless-shell` is the standard base.

---

## 12. Add a new action to the engine

Suppose you want a `set-cookie` action.

1. Add the capability to `internal/features/features.go`:

   ```go
   SetCookie func(ctx context.Context, w domain.WindowID, name, value, domain string) error
   ```

2. Implement the chromedp primitive in `internal/infra/chromedp/primitives.go`:

   ```go
   func (Primitives) SetCookie(ctx context.Context, name, value, domainStr string) error {
       return cdp.Run(ctx, network.SetCookie(name, value).WithDomain(domainStr))
   }
   ```

3. Wire it in `internal/orchestrator/features/makes.go` (inside
   `MakeBrowserActions`):

   ```go
   SetCookie: func(_ context.Context, w domain.WindowID, name, value, dom string) error {
       ctx, err := withCtx(w); if err != nil { return err }
       return p.SetCookie(ctx, name, value, dom)
   },
   ```

4. Dispatch the new YAML keyword in
   `internal/orchestrator/usecases/runtask_impl.go`:

   ```go
   case "set-cookie":
       return r.browser.SetCookie(ctx, w,
           asString(params["name"]), asString(params["value"]), asString(params["domain"]))
   ```

5. Use it from YAML:

   ```yaml
   - run: set-cookie
     params: { name: "session", value: "{{token}}", domain: ".example.com" }
   ```

Six files, no Java-style ceremony.

---

## Action reference

All actions current as of the live engine. Param keys without a default are
required.

### Navigation & timing

| Action | Params | Notes |
|---|---|---|
| `goto` | `url` | `driver.Navigate` |
| `wait` | `duration` (ms; supports `5_000` underscore form) | `time.Sleep` |
| `wait-for` | `selector`, `timeoutMs?=10000` | Blocks until selector is in DOM |

### Input

| Action | Params | Notes |
|---|---|---|
| `sendkeys` | `selector`, `keys` | Focus + type |
| `action` | `action=click`, `selector`, `text?`, `match?=exact`, `closest?` | Native isTrusted click. With `text:` it clicks the first `selector` whose visible text matches (`match: contains` for substring); `closest:` retargets to that element's nearest matching ancestor (match a label, click its row). |

### Scrolling

| Action | Params | Notes |
|---|---|---|
| `scroll-until-stable` | `selector`, `direction?=up`, `stableMs?=1500`, `maxIterations?=0` | Set `scrollTop` repeatedly until `scrollHeight` stabilises |

### JS (escape hatch for complex logic)

| Action | Params | Notes |
|---|---|---|
| `js` | `fn?` or `file?` or `script?`, `args?`, `await?=false` | Resolution order: `fn` → `file` → `script`. `args` is passed to the page as the `arguments` array. With `await: true` the script runs as an async function and a returned Promise is awaited — needed for self-contained async routines (`await ...`). Prefer YAML actions; reach for `js` only when no action fits. |

### Rendering & capture

| Action | Params | Notes |
|---|---|---|
| `pdf` | `path?`, `as?`, `format?` (A4/Letter/Legal/A3), `landscape?`, `printBackground?=true`, `scale?`, `paperWidth?`/`paperHeight?` (inches), `marginTop/Bottom/Left/Right?`, `pageRanges?`, `displayHeaderFooter?`, `headerTemplate?`, `footerTemplate?` | Renders the current page to PDF. Writes to `path:` and/or returns it base64 in `as:` — at least one required. |
| `screenshot` | `selector?=.`, `fullPage?=false`, `format?=png`, `quality?`, `path?`, `as?` | `fullPage` captures the whole scrollable page. Writes to `path:` and/or returns base64 in `as:`. |
| `snapshot` | `path?`, `as?` | MHTML single-file archive (HTML + CSS + images inlined). |
| `html-to-pdf` | `html?` or `file?` (source), `css?`, plus all `pdf` page options, `path?`/`as?` | Renders an HTML string (or file) to PDF. For Markdown, build the HTML first with a CDN `marked.js` script. |
| `emulate` | `width?`, `height?`, `deviceScaleFactor?=1`, `mobile?=false`, `colorScheme?` (light/dark), `reset?=false` | Device-metrics + emulated-media override. Persists on the window — use `reset: true` to clear. |

### Recording

| Action | Params | Notes |
|---|---|---|
| `record` | `format?=gif` (gif/mp4), `path?`, `as?`, `fps?=5`, `maxFrames?=300`, `maxDurationMs?=30000`, `do:` | Screencasts the page while the `do:` children run, then encodes a GIF (pure Go) or MP4 (needs `ffmpeg` on PATH). A failed run is still recorded. |

Any step can also carry `record: true` to screencast just that one step — a
debug aid: the GIF lands under `$TMPDIR/webtasks-recordings/` and, if the step
fails, the error names the file.

### Network & session

| Action | Params | Notes |
|---|---|---|
| `capture-network` | `path?`, `as?`, `includeBodies?=false`, `maxBodyBytes?=65536`, `urlFilter?`, `do:` | Records request/response entries (HAR-ish) while the `do:` children run. |
| `console` | `as`, `do:` | Collects `console.*` messages while the `do:` children run. |
| `wait-for-network-idle` | `idleMs?=500`, `timeoutMs?=15000`, `maxInflight?=0` | Blocks until in-flight requests stay quiet. |
| `get-cookies` | `urls?`, `path?`, `as?` | Reads cookies; writes JSON to `path:` and/or `as:`. |
| `set-cookies` | `cookies?` or `path?` (JSON file) | Installs cookies — session import. |

### Extraction

| Action | Params | Notes |
|---|---|---|
| `extract` | `selector`, `repeat?=false`, `fields={…}`, `from?=.` | Returns object (or array if `repeat`). See [§3](#3-extract-a-list-of-records). |

### Control flow & values

| Action | Params | Notes |
|---|---|---|
| `for-each` | `over`, `as?=item`, `continueOnError?=false`, `do:` | Iterates a list (`over: "{{ref}}"` resolves to the actual list). Each iteration runs the `do:` children with `{{<as>}}` and `{{<as>_index}}` bound. |
| `loop` | `while?` or `until?` (inline JS expr) / `whileFn?` or `untilFn?` (JS module), `pauseMs?=1000`, `maxIterations?=1000`, `do:` | Generic while-loop: evaluate the JS condition, run the `do:` children, pause, repeat. `loop_index` is bound in the body. `emit-event` inside the body is how live updates reach an SSE caller. |
| `set` | `value` | Assigns the rendered `value` (string/list/map/number) to `as:` — compute paths, constants, accumulate. |
| `call` | `task` | Runs another registered task's flow in the current window with the current bindings — factor a reusable flow (e.g. a watch-loop calling a watch task). |
| `return` | `value` | Sets the task's response payload to `value` alone and stops the remaining steps. Without it the full output map is returned. |
| `http-request` | `url`, `method?=GET`, `headers?`, `body?` (string or map→JSON), `timeoutMs?=30000`, `followRedirects?=true` | Outbound HTTP call (no browser). Result `{status, headers, body, json?}` into `as:`. |
| `export` | `data`, `format` (csv/ndjson/md-table), `columns?`, `path?`, `as?` | Renders a list/map of records into CSV / NDJSON / a markdown table. |

A param whose entire value is a single `{{ref}}` token resolves to the **raw**
bound value (list/map/number), not its stringified form — that's how
`for-each over: "{{chats}}"` and `write-files files: "{{built.files}}"`
receive real structured data.

### Filesystem (generic backend functions)

| Action | Params | Notes |
|---|---|---|
| `write-files` | `root`, `files` | `files` is a list of `{path, content}` or `{path, bytesB64}`; each is written under `root`, parent dirs created. The generic "store content + make dirs". |
| `read-file` | `path`, `optional?=false` | Reads a file into `as:` (string). `optional: true` yields `""` instead of erroring when missing — useful for resume/dedup. |
| `save-html` | `selector?=.`, `path` | Writes `outerHTML` to disk |
| `save-captures-to-dir` | `dir`, `naming?={id}_{name}` | Drains `window.__webtasks_captures`; writes blobs to disk |

### Downloads

| Action | Params | Notes |
|---|---|---|
| `download-each` | `selector`, `timeoutPerFileMs?=30000` | Clicks each match in DOM order; polls per-window download dir |

(`screenshot` and the other media actions are under [Rendering & capture](#rendering--capture).)

### Events

| Action | Params | Notes |
|---|---|---|
| `emit-event` | `kind?=status`, `text`, `data?` | Visible to SSE/WS callers; no-op for sync REST |

### Task- and step-level fields

| Field | Scope | Notes |
|---|---|---|
| `timeoutMs` | task | Whole-task budget; a stuck action fails with `context deadline exceeded` once it elapses. |
| `setupTask` | task | Names another task whose flow runs (same window) before this one — e.g. an idempotent "ensure logged in" prelude. |
| `record` | step | `record: true` on any step screencasts that step to a GIF under `$TMPDIR/webtasks-recordings/` — a debug aid for diagnosing why a step fails. |

### Pool fields (`tasks/pool.yaml`)

| Field | Notes |
|---|---|
| `size` | Number of pre-allocated windows for the pool. |
| `persistent` | When `true`, the pool's windows use a stable Chrome profile (under `WEBTASKS_PROFILE_DIR`, default `~/.webtasks/profiles`) that survives restarts — a one-time manual login persists. Keep persistent pools at `size: 1`. |
| `profile` | Profile name for a persistent pool (defaults to the pool tag). |

---

## Templating reference

`{{name}}` in any string param is replaced at render time.

Lookup order:

1. The task's input bindings (after `bindInputs`)
2. The process env (so resolved secrets propagate)

Supported forms:

```text
{{user}}                       # plain lookup
{{user|or:guest}}              # fallback when empty
{{item.address.city}}          # dotted path into a map value
```

Templating runs recursively over `params`: strings, list items, and map
values are all substituted. Other types pass through.

---

## Troubleshooting

**"unknown task: foo/bar"** — YAML wasn't picked up. Check `executor
list-tasks`. Hot-reload re-reads YAML on every call, so a typo in YAML is
the usual cause.

**"acquire timeout: <pool>"** — no free window in the pool for 30 s.
Increase pool size in `tasks/pool.yaml` or shorten upstream tasks.

**"browser session was reset (tab crashed or detached)"** — chromedp lost
the target. The pool already replaced the window; re-run any required
setup task (e.g. login) before retrying.

**Click does nothing** — the framework may gate on `isTrusted`. Use
`action: click` (which uses CDP `MouseClickNode`) instead of a JS-dispatched
event. For Concio-style sidebars where the click handler lives on an inner
element, use a `js` module that finds the right child and dispatches a
proper MouseEvent sequence — see [§5](#5-open-a-chat-by-visible-name-concio-pattern).

**Downloads don't land in the dir** — Chrome auto-saves files into the
per-window directory once `Browser.setDownloadBehavior` is set (already
wired in `infra/chromedp/windowsource.go`). If the site decrypts client-side
and never triggers a real HTTP download, install a `URL.createObjectURL`
hook and use `save-captures-to-dir` instead — see [§7b](#7b-client-side-decryption--install-a-js-hook--drain).

**Secrets prompt blocks in CI** — set `WEBTASKS_HEADLESS=true` doesn't
affect prompting. Either supply secrets via `--NAME=value` args / env vars
(both come before `prompt` in the default source list), or use
`required: false` with a `default:` so missing values don't block.
