# Action reference

The complete vocabulary the flow interpreter understands. Every `run:` keyword
is listed here with all of its parameters, defaults, output shape, and an
example. This is the detailed companion to the condensed table in
[cookbook.md](cookbook.md#action-reference).

Each step in a task's `flow:` is one command:

```yaml
- run: <action>            # the keyword (required)
  status: "Human text"     # optional: emitted as an SSE `status` event
  as: <name>               # optional: where the result is stored
  record: true             # optional: screencast just this step to a GIF
  params: { … }            # action-specific parameters
  do:                      # child steps, for the block actions
    - run: …
```

Conventions in the tables below:

- A param written `name?=default` is optional with that default; `name` with no
  `=` is required (or the step errors).
- **"Output"** describes what an `as:` binding receives, and is also what lands
  in the response `data`. Most artifact actions accept `path:` (write to disk),
  `as:` (return inline), or both.
- Param values are templated first — see [templating.md](templating.md).

---

## Table of contents

- [Navigation & timing](#navigation--timing)
- [Input & interaction](#input--interaction)
- [Scrolling](#scrolling)
- [JavaScript](#javascript)
- [Extraction](#extraction)
- [Rendering & capture](#rendering--capture)
- [Recording](#recording)
- [Network & session](#network--session)
- [Downloads & blob capture](#downloads--blob-capture)
- [Control flow](#control-flow)
- [Values & data](#values--data)
- [Filesystem](#filesystem)
- [Events](#events)
- [How results are stored: `as`, `path`, `__result__`](#how-results-are-stored)

---

## Navigation & timing

### `goto`

Navigate the window to a URL.

| Param | Notes |
|---|---|
| `url` | Absolute URL. Templated, so `{{q}}` etc. work. |

```yaml
- run: goto
  params: { url: "https://example.com/?q={{query}}" }
```

### `wait`

Sleep for a fixed duration. Use sparingly — prefer `wait-for` / `wait-for-network-idle`.

| Param | Notes |
|---|---|
| `duration` | Milliseconds. Accepts the underscore form (`5_000`) for readability. |

```yaml
- run: wait
  params: { duration: "8_000" }
```

### `wait-for`

Block until a selector is present in the DOM.

| Param | Notes |
|---|---|
| `selector` | CSS selector to wait for. |
| `timeoutMs?=10000` | Fail with a timeout error if it never appears. |

```yaml
- run: wait-for
  params: { selector: "article h3 a", timeoutMs: 15000 }
```

`wait-for` matches DOM *presence*, not "data loaded". For SPAs that render a
skeleton first, wait for the loaded state (e.g.
`.results .item:not(.loading)`).

---

## Input & interaction

### `sendkeys`

Focus an element and type text into it.

| Param | Notes |
|---|---|
| `selector` | The field to type into. |
| `keys` | The text to type (templated). |

```yaml
- run: sendkeys
  params: { selector: "#search", keys: "{{query}}" }
```

### `action` (click)

A native, trusted click via CDP (`Input.dispatchMouseEvent`) — works on sites
that gate handlers on `isTrusted`, unlike JS-dispatched events. The only
supported `action` value is `click`.

| Param | Notes |
|---|---|
| `action` | Must be `click`. |
| `selector?=*` | Element to click (or to match by text when `text:` is set). |
| `text?` | When set, click the first `selector` whose **visible text** matches. |
| `match?=exact` | `exact` or `contains` (substring match) — only with `text:`. |
| `closest?` | After matching by text, retarget the click to that element's nearest matching ancestor. Pattern: *match a label, click its row.* |

Click the first match of a selector:

```yaml
- run: action
  params: { action: click, selector: "button[type='submit']" }
```

Click by visible text, retargeting to a row ancestor:

```yaml
- run: action
  params:
    action: click
    selector: ".name"
    text: "Nicholas Huang"
    match: exact
    closest: ".chat-row"
```

> When you only know an element by text *and* the site needs a real
> MouseEvent sequence on an inner node, a small `js` module is sometimes
> clearer — see [cookbook.md §5](cookbook.md#5-open-a-chat-by-visible-name-concio-pattern).

---

## Scrolling

### `scroll-until-stable`

Repeatedly scroll a container and stop once its `scrollHeight` stops changing —
the standard way to exhaust infinite-scroll feeds and chat history.

| Param | Notes |
|---|---|
| `selector` | The scrollable container. |
| `direction?=up` | `up` (sets `scrollTop=0`, loads older content) or `down` (sets to `scrollHeight`). |
| `stableMs?=1500` | Exit once the height has been unchanged for this long. |
| `maxIterations?=0` | Cap the number of scroll steps. `0` = unbounded. |

```yaml
- run: scroll-until-stable
  params:
    selector: ".chats.chat-content-scroll"
    direction: up
    stableMs: 2500
    maxIterations: 0
```

---

## JavaScript

### `js`

The escape hatch: evaluate JavaScript in the page. The script runs as the body
of a function; YAML `args:` become its `arguments` array. The return value
becomes the `as:` binding. Prefer dedicated actions; reach for `js` only when
no action fits.

| Param | Notes |
|---|---|
| `fn?` | Name of a JS module under the bundle's `scripts/` (without `.js`). |
| `file?` | Same as `fn` (alias). |
| `script?` | Inline JS source. |
| `args?` | List passed to the page as `arguments`. |
| `await?=false` | When `true`, the script runs as an async function and a returned Promise is awaited (chromedp `WithAwaitPromise`). |

Resolution order is `fn` → `file` → `script`; at least one is required.

```yaml
- run: js
  as: stats
  params:
    fn: "demo/page-stats"
    args: ["{{section}}"]
    await: false
```

Inline form:

```yaml
- run: js
  as: count
  params:
    script: "return document.querySelectorAll('a').length;"
```

See [build-your-own-task.md §7](build-your-own-task.md#7-when-css-isnt-enough--use-a-js-module)
for the module-vs-inline trade-off.

---

## Extraction

### `extract`

Turn rendered HTML into typed JSON via a CSS-selector field spec. The single
most-used action.

| Param | Notes |
|---|---|
| `from?=.` | Selector whose `outerHTML` is the extraction source. `.` = whole document. |
| `selector` | The row(s) to harvest. |
| `repeat?=false` | `false` → one object; `true` → one object per `selector` match (a list). |
| `fields` | Map of `fieldName → field spec` (below). |

**Field spec:**

| Key | Notes |
|---|---|
| `kind?=text` | `text` (trimmed `textContent`), `attr`, `html` (inner HTML), or `const`. |
| `selector?=.` | Selector *relative to the row*. `.` = the row element itself. |
| `name` | Attribute name — required when `kind: attr`. |
| `value` | The literal value — used when `kind: const` (tag every record). |
| `transform?` | `int`, `long`, `trim`, `lower`, or `upper`, applied to the text/attr result. |

```yaml
- run: extract
  as: repos
  params:
    selector: "article.Box-row"
    repeat: true
    fields:
      slug:        { kind: text, selector: "h2 a", transform: trim }
      href:        { kind: attr, selector: "h2 a", name: "href" }
      description: { kind: text, selector: "p" }
      source:      { kind: const, value: "github-trending" }
      summary:     { kind: html, selector: ".desc" }
```

Output: with `repeat: true`, a list of objects; with `repeat: false`, one
object.

---

## Rendering & capture

All four actions accept `path:` (write to disk) and/or `as:` (return base64 /
a `{path,size,bytesB64}` map). At least one is required. See
[How results are stored](#how-results-are-stored).

### `screenshot`

| Param | Notes |
|---|---|
| `selector?=.` | `.` → viewport; a selector → that element only. |
| `fullPage?=false` | Capture the whole scrollable page. |
| `format?=png` | `png` or `jpeg`. |
| `quality?` | JPEG quality 0–100 (jpeg only). |
| `path?`, `as?` | Output sink(s). |

```yaml
- run: screenshot
  as: shot_b64
  params: { fullPage: true, format: png }
```

### `pdf`

Render the current page to PDF (CDP `Page.printToPDF`).

| Param | Notes |
|---|---|
| `format?` | `A4`, `A3`, `Letter`, `Legal` — sets paper size. |
| `landscape?=false` | Orientation. |
| `printBackground?=true` | Print background graphics. |
| `scale?` | Render scale. |
| `paperWidth?`/`paperHeight?` | In inches (override `format`). |
| `marginTop/Bottom/Left/Right?` | In inches. |
| `pageRanges?` | e.g. `"1-3,5"`. |
| `displayHeaderFooter?`, `headerTemplate?`, `footerTemplate?` | Header/footer HTML. |
| `path?`, `as?` | Output sink(s). |

```yaml
- run: pdf
  params: { path: "/tmp/page.pdf", format: A4, printBackground: true }
```

### `html-to-pdf`

Render an HTML string (or file) to PDF — no live page needed. Accepts all the
`pdf` page options above.

| Param | Notes |
|---|---|
| `html?` | Inline HTML source. |
| `file?` | Path to an HTML file (used when `html` is absent). |
| `css?` | CSS injected as a `<style>` prelude. |
| *(plus all `pdf` options)* | |
| `path?`, `as?` | Output sink(s). |

```yaml
- run: html-to-pdf
  params:
    file: "/tmp/report.html"
    css: "body{font-family:sans-serif}"
    format: A4
    path: "/tmp/report.pdf"
```

For Markdown, build the HTML first with a CDN `marked.js` snippet via `js`.

### `snapshot`

Capture an MHTML single-file archive (HTML + CSS + images inlined).

| Param | Notes |
|---|---|
| `path?`, `as?` | Output sink(s). |

```yaml
- run: snapshot
  params: { path: "/tmp/page.mhtml" }
```

### `emulate`

Override device metrics and emulated media. The override **persists on the
window** until reset.

| Param | Notes |
|---|---|
| `width?`, `height?` | Viewport dimensions. |
| `deviceScaleFactor?=1` | DPR. |
| `mobile?=false` | Mobile emulation. |
| `colorScheme?` | `light` / `dark` / `no-preference`. |
| `reset?=false` | Clear all overrides. |

```yaml
- run: emulate
  params: { width: 390, height: 844, mobile: true, deviceScaleFactor: 3 }
# … later …
- run: emulate
  params: { reset: true }
```

---

## Recording

### `record`

Screencast the page while the `do:` children run, then encode the frames to an
animated GIF (pure Go) or MP4 (needs `ffmpeg` on `PATH`). A run that *fails* is
still encoded and saved — that's exactly the run worth inspecting.

| Param | Notes |
|---|---|
| `format?=gif` | `gif` or `mp4`. |
| `fps?=5` | Frames per second. |
| `quality?=80` | Encoder quality. |
| `everyNthFrame?=2` | Sample every Nth screencast frame. |
| `maxFrames?=300` | Hard cap on frames. |
| `maxDurationMs?=30000` | Hard cap on wall-clock. |
| `path?` | Write the media to disk. |
| `as?` | Bind `{frames, durationMs, size, path|bytesB64}`. |
| `do:` | The steps to record. |

At least one of `path:` / `as:` is required.

```yaml
- run: record
  as: clip
  params:
    format: gif
    fps: 4
    path: "/tmp/run.gif"
  do:
    - run: goto
      params: { url: "https://example.com" }
    - run: action
      params: { action: click, selector: "#more" }
```

**Step-level recording.** Any step can carry `record: true` to screencast just
that one step (a GIF at `$TMPDIR/webtasks-recordings/`); if the step fails, the
error names the file. A debug aid, no `do:` needed.

---

## Network & session

### `capture-network`

Record request/response entries (HAR-ish) while the `do:` children run.

| Param | Notes |
|---|---|
| `includeBodies?=false` | Capture response bodies. |
| `maxBodyBytes?=65536` | Truncate bodies past this size. |
| `urlFilter?` | Only record URLs containing this substring. |
| `path?` | Write `{entries, count}` as JSON to disk. |
| `as?` | Bind `{entries, count}`. |
| `do:` | Steps during which traffic is captured. |

```yaml
- run: capture-network
  as: har
  params: { includeBodies: true, urlFilter: "/api/" }
  do:
    - run: goto
      params: { url: "https://example.com/app" }
```

### `console`

Collect `console.*` messages while the `do:` children run.

| Param | Notes |
|---|---|
| `as` | The collected log list. |
| `do:` | Steps during which logs are captured. |

```yaml
- run: console
  as: logs
  do:
    - run: goto
      params: { url: "https://example.com" }
```

### `wait-for-network-idle`

Block until in-flight requests stay quiet.

| Param | Notes |
|---|---|
| `idleMs?=500` | Required quiet window. |
| `timeoutMs?=15000` | Give up after this long. |
| `maxInflight?=0` | Treat ≤ this many requests as "idle". |

```yaml
- run: wait-for-network-idle
  params: { idleMs: 800, timeoutMs: 20000 }
```

### `get-cookies`

Read cookies.

| Param | Notes |
|---|---|
| `urls?` | List of URLs to scope to (omit for all). |
| `path?` | Write the cookie list as JSON to disk. |
| `as?` | Bind the cookie list (`name, value, domain, path, expires, httpOnly, secure, sameSite`). |

### `set-cookies`

Install cookies — useful for importing a session.

| Param | Notes |
|---|---|
| `cookies?` | Inline list of cookie objects. |
| `path?` | Read the cookie list from a JSON file (used when `cookies` is absent). |

Bind result: `{count}`. Each cookie supports `name, value, domain, path, url,
sameSite, expires, httpOnly, secure`.

```yaml
- run: set-cookies
  params:
    cookies:
      - { name: "session", value: "{{token}}", domain: ".example.com", path: "/" }
```

### `http-request`

An outbound HTTP call with **no browser** involved.

| Param | Notes |
|---|---|
| `url` | Target URL. |
| `method?=GET` | Upper-cased. |
| `headers?` | Map of header → value. |
| `body?` | String, or a map/list (marshalled to JSON; `Content-Type: application/json` added unless set). |
| `timeoutMs?=30000` | Request timeout. |
| `followRedirects?=true` | Set `false` to stop on the first redirect. |

Bind result: `{status, headers, body, json?}` — `json` is present when the body
parses as JSON.

```yaml
- run: http-request
  as: api
  params:
    url: "https://hn.algolia.com/api/v1/search?query={{q}}"
    method: GET
```

---

## Downloads & blob capture

### `download-each`

Native-click every match of `selector` in DOM order. Chrome saves each download
into the per-window download directory; the action polls for the new file after
each click and returns its path.

| Param | Notes |
|---|---|
| `selector` | Elements to click. |
| `timeoutPerFileMs?=30000` | Per-file poll timeout (empty path slot if it times out). |

Bind result: `[{path, basename}, …]`.

```yaml
- run: download-each
  as: downloads
  params: { selector: "a.download-link", timeoutPerFileMs: 30000 }
```

The CDP `Browser.setDownloadBehavior` call is already wired into the window
factory, so headless Chrome doesn't block downloads.

### `save-captures-to-dir`

For apps that decrypt blobs client-side and never trigger a real HTTP
download (Concio, ProtonMail-style). After an in-page `URL.createObjectURL`
hook has buffered captures into `window.__webtasks_captures`, this drains the
ready ones to disk. Pending entries stay buffered for a future drain.

| Param | Notes |
|---|---|
| `dir` | Server-side output directory. |
| `naming?={id}_{name}` | Filename template. Tokens: `{id} {name} {ext} {mime} {size} {ts}`. |

Bind result: `[{path, basename, size, mime, name, url}, …]`.

Full pattern (install hook → click files → wait → drain) is in
[cookbook.md §7b](cookbook.md#7b-client-side-decryption--install-a-js-hook--drain).

---

## Control flow

### `for-each`

Iterate a list, running the `do:` children once per item with a cloned bindings
map.

| Param | Notes |
|---|---|
| `over` | The list. Use `over: "{{ref}}"` — a single-token ref resolves to the **raw** list (see [templating.md](templating.md)). |
| `as?=item` | Name bound to the current item. `{{<as>_index}}` holds the 0-based index. |
| `continueOnError?=false` | When `true`, a failing iteration emits an `error` event and the loop continues. |
| `do:` | Steps run per item. |

```yaml
- run: for-each
  params:
    over: "{{chats}}"
    as: chat
    continueOnError: true
  do:
    - run: emit-event
      params: { text: "Processing {{chat.peerName}} ({{chat_index}})" }
```

### `loop`

A generic while-loop driven by a JS condition, re-evaluated each iteration.

| Param | Notes |
|---|---|
| `while?` | Inline JS expression — loop while truthy. |
| `until?` | Inline JS expression — loop until truthy. |
| `whileFn?` / `untilFn?` | A JS module name instead of an inline expression. |
| `pauseMs?=1000` | Pause between iterations. |
| `maxIterations?=1000` | Safety cap. |
| `do:` | Body. `loop_index` is bound inside. |

A bare expression is auto-wrapped (`until: "x === 1"` works without `return`).
`emit-event` in the body is how a long loop streams progress to an SSE caller.

```yaml
- run: loop
  params:
    untilFn: "concio/all-chats-done"
    pauseMs: 1500
    maxIterations: 200
  do:
    - run: call
      params: { task: "concio/watch" }
```

### `call`

Run another registered task's flow **in the current window with the current
bindings** — factor a reusable flow (e.g. a watch-loop calling a watch task).

| Param | Notes |
|---|---|
| `task` | Name of a registered task. |

```yaml
- run: call
  params: { task: "concio/watch" }
```

### `return`

Set the task's response payload to `value` alone and stop the remaining steps.
Without a `return`, the full output map is returned.

| Param | Notes |
|---|---|
| `value` | The sole response payload (any type). |

```yaml
- run: return
  params: { value: "{{results}}" }
```

---

## Values & data

### `set`

Assign a literal or templated value to a binding/output. `value` renders to
whatever type it is — string, list, map, number.

| Param | Notes |
|---|---|
| `value` | The value to assign to `as:`. |

```yaml
- run: set
  as: outDir
  params: { value: "{{out}}/{{owner}}/chats" }
```

### `export`

Render a list/map of records into CSV, NDJSON, or a markdown table.

| Param | Notes |
|---|---|
| `data` | The records (list of maps, or a single map). Usually `"{{ref}}"`. |
| `format?=csv` | `csv`, `ndjson`, or `md-table` (aliases `md`, `markdown`). |
| `columns?` | Explicit column order. Defaults to the sorted union of keys. |
| `path?`, `as?` | Output sink(s) — the rendered text. |

```yaml
- run: export
  as: csv
  params:
    data: "{{repos}}"
    format: csv
    columns: ["slug", "stars", "href"]
    path: "/tmp/repos.csv"
```

---

## Filesystem

### `read-file`

Read a file into the `as:` binding (as a string).

| Param | Notes |
|---|---|
| `path` | File to read. |
| `optional?=false` | When `true`, a missing file yields `""` instead of erroring — handy for resume/dedup. |

```yaml
- run: read-file
  as: seen
  params: { path: "{{out}}/state.json", optional: true }
```

### `write-files`

Store many files at once, creating parent directories as needed. The generic
"store content + make dirs" backend.

| Param | Notes |
|---|---|
| `root` | Base directory. Each file's `path` is joined under it (cleaned, no traversal). |
| `files` | List of `{path, content}` or `{path, bytesB64}`. Usually `"{{ref}}"`. |

Bind result: `{count, root, files: [{path, size}, …]}`.

```yaml
- run: write-files
  as: written
  params:
    root: "{{out}}"
    files: "{{built.files}}"
```

### `save-html`

Write an element's `outerHTML` to disk.

| Param | Notes |
|---|---|
| `selector?=.` | `.` = whole document. |
| `path` | Destination file. |

```yaml
- run: save-html
  params: { path: "/tmp/snapshot.html" }
```

---

## Events

### `emit-event`

Emit a custom progress event — visible to SSE/WebSocket callers, a no-op for
sync REST.

| Param | Notes |
|---|---|
| `kind?=status` | Event kind (`status`, `progress`, or anything). |
| `text` | Human-readable message (templated). |
| `data?` | Arbitrary structured payload. |

```yaml
- run: emit-event
  params:
    kind: "progress"
    text: "Harvested {{items.length}} items"
    data: { fraction: 0.5 }
```

See [http-api.md](http-api.md) for the SSE event protocol.

---

## How results are stored

The `as:` and `path:` params interact consistently across actions:

- **Data actions** (`extract`, `js`, `http-request`, `set`, `for-each` results,
  `write-files`, …) store their result under `as:` in both the response `data`
  and the live bindings. No `as:` → the result is discarded.
- **Artifact actions** (`screenshot`, `pdf`, `html-to-pdf`, `snapshot`,
  `record`, `export`, `capture-network`, `get-cookies`) accept:
  - `path:` only → written to disk, nothing bound.
  - `as:` only → returned inline (base64 for binary, text for `export`).
  - **both** → written to disk *and* bound as a
    `{path, size, bytesB64}` map (binary) or the text (export).
  - At least one of `path:` / `as:` is required, else the step errors.
- **`return value:`** sets the reserved `__result__` binding. When present, the
  HTTP response `data` is *that value alone*, not the full output map.

A param whose entire value is a single `{{ref}}` token resolves to the **raw**
bound value (list/map/number), not its stringified form — that's how
`for-each over: "{{chats}}"` and `write-files files: "{{built.files}}"` receive
real structured data. → [templating.md](templating.md)
