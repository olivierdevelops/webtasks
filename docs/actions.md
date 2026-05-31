# Action reference

The complete vocabulary a `.webtask` recipe understands. Every action is listed
here with its parameters, defaults, output shape, and an example. This is the
detailed companion to the condensed table in [Recipes](cookbook.md).

A step is one action. An optional `status "…"` line **before** a step sets the
live progress message SSE callers see; some actions open a block closed with
`end`:

```capy
status "Human text"           # optional: emitted as an SSE status event
goto "https://example.com"    # a simple step

record clip path "/tmp/run.gif"   # a block action…
    goto "https://example.com"    # …child steps run inside
    click "#more"
end
```

Conventions in the tables below:

- A param written `name?=default` is optional with that default; `name` with no
  `=` is required (or the step errors).
- **"Output"** describes what a step's result name receives, and is also what
  lands in the response `data`. Most capture actions accept a `path` (write to
  disk), a result name (return inline), or both.
- Param values are templated first — see [Templating](templating.md).

---

## Table of contents

- [Navigation & timing](#navigation-timing)
- [Input & interaction](#input-interaction)
- [Scrolling](#scrolling)
- [JavaScript](#javascript)
- [Extraction](#extraction)
- [Rendering & capture](#rendering-capture)
- [Recording](#recording)
- [Network & session](#network-session)
- [Downloads & blob capture](#downloads-blob-capture)
- [Control flow](#control-flow)
- [Values & data](#values-data)
- [Filesystem](#filesystem)
- [Events](#events)
- [How results are stored](#how-results-are-stored)

---

## Navigation & timing

### `goto`

Navigate the window to a URL.

| Param | Notes |
|---|---|
| `url` | Absolute URL. Templated, so `{{q}}` etc. work. |

```capy
goto "https://example.com/?q={{query}}"
```

### `wait`

Sleep for a fixed duration. Use sparingly — prefer `wait until` / `wait-network-idle`.

| Param | Notes |
|---|---|
| duration | Milliseconds. |

```capy
wait 8000
```

### `wait until`

Block until a selector is present in the DOM.

| Param | Notes |
|---|---|
| selector | CSS selector to wait for. |
| `timeout?=10000` | Fail with a timeout error if it never appears. |

```capy
wait until "article h3 a" timeout 15000
```

`wait until` matches DOM *presence*, not "data loaded". For SPAs that render a
skeleton first, wait for the loaded state (e.g. `.results .item:not(.loading)`).

---

## Input & interaction

### `sendkeys`

Focus an element and type text into it.

| Param | Notes |
|---|---|
| selector | The field to type into. |
| `keys` | The text to type (templated). |

```capy
sendkeys "#search" keys "{{query}}"
```

### `click`

A native, trusted click via CDP — works on sites that gate handlers on
`isTrusted`, unlike JS-dispatched events.

| Param | Notes |
|---|---|
| selector | Element to click (or to match by text when `text` is set). |
| `text?` | When set, click the first `selector` whose **visible text** matches. |
| `match?=exact` | `exact` or `contains` — only with `text`. |
| `closest?` | After matching by text, retarget the click to that element's nearest matching ancestor. Pattern: *match a label, click its row.* |

Click the first match of a selector:

```capy
click "button[type='submit']"
```

Click by visible text, retargeting to a row ancestor:

```capy
click ".name" text "Nicholas Huang" match exact closest ".chat-row"
```

---

## Scrolling

### `scroll until stable`

Repeatedly scroll a container and stop once its `scrollHeight` stops changing —
the standard way to exhaust infinite-scroll feeds and chat history.

| Param | Notes |
|---|---|
| selector | The scrollable container. |
| `direction?=up` | `up` (loads older content) or `down`. |
| `stable?=1500` | Exit once height has been unchanged for this many ms. |
| `max?=0` | Cap the number of scroll steps. `0` = unbounded. |

```capy
scroll until stable ".chats.chat-content-scroll" direction up stable 2500 max 0
```

---

## JavaScript

### `js`

The escape hatch: evaluate JavaScript in the page. `args` become its `arguments`
array; the return value becomes the result. Prefer dedicated actions; reach for
`js` only when no action fits.

| Param | Notes |
|---|---|
| `fn?` | Name of a JS module under the bundle's `scripts/` (`.js` optional). |
| `script?` | Inline JS source. |
| `args?` | List passed to the page as `arguments`. |
| `await?=false` | When `true`, the script runs async and a returned Promise is awaited. |

A module (`fn`) or inline `script` is required.

```capy
js stats fn "demo/page-stats" args ["{{section}}"]
```

Inline form:

```capy
js count script "return document.querySelectorAll('a').length;"
```

See [Writing tasks](writing-tasks.md) for the module-vs-inline trade-off.

---

## Extraction

### `extract`

Turn rendered HTML into typed JSON via a CSS-selector field spec — the
most-used action.

| Part | Notes |
|---|---|
| name | Where the result is stored. |
| `from "SEL"` | The row(s) to harvest. |
| `repeat` | Present → one object per match (a list); absent → one object. |
| field lines | One per field (below), closed with `end`. |

**Field lines:**

| Line | Captures |
|---|---|
| `name text "SEL"` | Trimmed text content. |
| `name text "SEL" trim` | Text, explicitly trimmed. |
| `name attr ATTR on "SEL"` | An HTML attribute (e.g. `href`). |
| `name html "SEL"` | Inner HTML. |
| `name const "VALUE"` | A literal constant tagged onto every record. |

```capy
extract repos from "article.Box-row" repeat
    slug        text "h2 a" trim
    href        attr href on "h2 a"
    description text "p"
    source      const "github-trending"
    summary     html ".desc"
end
```

With `repeat`, the result is a list of objects; without it, one object.

---

## Rendering & capture

All capture actions accept a `path` (write to disk) and/or a result name (return
base64 / a `{path,size,bytesB64}` map). At least one is required. See
[How results are stored](#how-results-are-stored).

### `screenshot`

| Param | Notes |
|---|---|
| `selector?=.` | `.` → viewport; a selector → that element only. |
| `fullPage?=false` | Capture the whole scrollable page. |
| `format?=png` | `png` or `jpeg`. |
| `quality?` | JPEG quality 0–100. |

```capy
screenshot shot_b64 fullPage true format png
```

### `pdf`

Render the current page to PDF.

| Param | Notes |
|---|---|
| `format?` | `A4`, `A3`, `Letter`, `Legal`. |
| `landscape?=false` | Orientation. |
| `printBackground?=true` | Print background graphics. |
| `scale?` | Render scale. |
| `pageRanges?` | e.g. `"1-3,5"`. |

```capy
pdf doc path "/tmp/page.pdf" format A4 printBackground true
```

### `html-to-pdf`

Render an HTML string (or file) to PDF — no live page needed. Accepts all the
`pdf` options above.

| Param | Notes |
|---|---|
| `html?` | Inline HTML source. |
| `file?` | Path to an HTML file (used when `html` is absent). |
| `css?` | CSS injected as a `<style>` prelude. |

```capy
html-to-pdf doc file "/tmp/report.html" css "body{font-family:sans-serif}" format A4 path "/tmp/report.pdf"
```

### `snapshot`

Capture an MHTML single-file archive (HTML + CSS + images inlined).

```capy
snapshot snap path "/tmp/page.mhtml"
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

```capy
emulate width 390 height 844 mobile true deviceScaleFactor 3
# … later …
emulate reset true
```

---

## Recording

### `record`

Screencast the page while the block's children run, then encode to an animated
GIF (pure Go) or MP4 (needs `ffmpeg`). A run that *fails* is still encoded and
saved — exactly the run worth inspecting.

| Param | Notes |
|---|---|
| `format?=gif` | `gif` or `mp4`. |
| `fps?=5` | Frames per second. |
| `quality?=80` | Encoder quality. |
| `maxFrames?=300` | Hard cap on frames. |
| `maxDurationMs?=30000` | Hard cap on wall-clock. |
| `path?` | Write the media to disk. |

At least one of `path` / a result name is required.

```capy
record clip format gif fps 4 path "/tmp/run.gif"
    goto "https://example.com"
    click "#more"
end
```

**Step-level recording.** Any step can carry `record` to screencast just that
one step (a GIF under `$TMPDIR/webtasks-recordings/`); if the step fails, the
error names the file. A debug aid, no block needed.

---

## Network & session

### `capture-network`

Record request/response entries (HAR-ish) while the block's children run.

| Param | Notes |
|---|---|
| `includeBodies?=false` | Capture response bodies. |
| `maxBodyBytes?=65536` | Truncate bodies past this size. |
| `urlFilter?` | Only record URLs containing this substring. |

```capy
capture-network har includeBodies true urlFilter "/api/"
    goto "https://example.com/app"
end
```

### `console`

Collect `console.*` messages while the block's children run.

```capy
console logs
    goto "https://example.com"
end
```

### `wait-network-idle`

Block until in-flight requests stay quiet.

| Param | Notes |
|---|---|
| `idle?=500` | Required quiet window (ms). |
| `timeout?=15000` | Give up after this long. |
| `maxInflight?=0` | Treat ≤ this many requests as "idle". |

```capy
wait-network-idle 800 timeout 20000
```

### `get-cookies`

Read cookies into the result (`name, value, domain, path, expires, httpOnly,
secure, sameSite`).

| Param | Notes |
|---|---|
| `urls?` | List of URLs to scope to (omit for all). |
| `path?` | Write the cookie list as JSON to disk. |

```capy
get-cookies jar
```

### `set-cookies`

Install cookies — useful for importing a session. Reads from inline cookies or a
JSON file.

```capy
set-cookies cookies [{ name: "session", value: "{{token}}", domain: ".example.com", path: "/" }]
```

### `http-get` / `http-post`

An outbound HTTP call with **no browser** involved. Result:
`{status, headers, body, json?}` — `json` is present when the body parses as
JSON.

| Param | Notes |
|---|---|
| url | Target URL. |
| `headers?` | Map of header → value. |
| `body?` | String, or a map/list (marshalled to JSON). |
| `timeout?=30000` | Request timeout. |

```capy
http-get api url "https://hn.algolia.com/api/v1/search?query={{q}}"
```

---

## Downloads & blob capture

### `download-each`

Native-click every match of `selector` in DOM order. Chrome saves each download
to the per-window directory; the action polls for each new file and returns its
path. Result: `[{path, basename}, …]`.

| Param | Notes |
|---|---|
| selector | Elements to click. |
| `timeout?=30000` | Per-file poll timeout. |

```capy
download-each downloads selector "a.download-link" timeout 30000
```

The CDP download-behaviour call is already wired into the window factory, so
headless Chrome doesn't block downloads.

### `save-captures-to-dir`

For apps that decrypt blobs client-side and never trigger a real HTTP download
(Concio, ProtonMail-style). After an in-page `URL.createObjectURL` hook has
buffered captures, this drains the ready ones to disk. Result:
`[{path, basename, size, mime, name, url}, …]`.

| Param | Notes |
|---|---|
| dir | Server-side output directory. |
| `naming?={id}_{name}` | Filename template. Tokens: `{id} {name} {ext} {mime} {size} {ts}`. |

The full pattern (install hook → click files → wait → drain) is in
[Recipes](cookbook.md).

---

## Control flow

### `for-each`

Iterate a list, running the block once per item with a cloned bindings map.

| Part | Notes |
|---|---|
| `item in "{{ref}}"` | A single-token ref resolves to the **raw** list. |
| `as?=item` | Name bound to the current item. `{{<as>_index}}` holds the 0-based index. |
| `continueOnError?` | When set, a failing iteration emits an `error` event and the loop continues. |

```capy
for-each chat in "{{chats}}" continueOnError true
    emit status "Processing {{chat.peerName}} ({{chat_index}})"
end
```

### `loop`

A generic loop driven by a JS condition, re-evaluated each iteration.

| Part | Notes |
|---|---|
| `while js "EXPR"` | Loop while truthy. |
| `until js "EXPR"` | Loop until truthy. |
| `while fn "MODULE"` / `until fn "MODULE"` | A JS module instead of an inline expression. |
| `pause?=1000` | Pause between iterations (ms). |
| `max?=1000` | Safety cap. |

A bare expression is auto-wrapped. `emit` in the body streams progress to an SSE
caller.

```capy
loop until fn "concio/all-chats-done" pause 1500 max 200
    call "concio/watch"
end
```

### `call`

Run another registered task's flow **in the current window with the current
bindings** — factor a reusable flow.

```capy
call "concio/watch"
```

### `return`

Set the task's response payload to `value` alone and stop the remaining steps.
Without a `return`, the full output map is returned.

```capy
return "{{results}}"
```

---

## Values & data

### `set`

Assign a literal or templated value to a binding/output. The value renders to
whatever type it is — string, list, map, number.

```capy
set outDir "{{out}}/{{owner}}/chats"
```

### `export`

Render a list/map of records into CSV, NDJSON, or a markdown table.

| Param | Notes |
|---|---|
| format | `csv`, `ndjson`, or `md` (markdown table). |
| data | The records (list of maps, or a single map). Usually `"{{ref}}"`. |
| `columns?` | Explicit column order. Defaults to the sorted union of keys. |
| `path?` | Output path. |

```capy
export csv path "/tmp/repos.csv" data "{{repos}}" columns ["slug", "stars", "href"]
```

---

## Filesystem

### `read-file`

Read a file into the result (as a string).

| Param | Notes |
|---|---|
| path | File to read. |
| `optional?=false` | When `true`, a missing file yields `""` instead of erroring. |

```capy
read-file seen path "{{out}}/state.json" optional true
```

### `write-files`

Store many files at once, creating parent directories as needed. Result:
`{count, root, files: [{path, size}, …]}`.

| Param | Notes |
|---|---|
| root | Base directory. Each file's `path` is joined under it (no traversal). |
| files | List of `{path, content}` or `{path, bytesB64}`. Usually `"{{ref}}"`. |

```capy
write-files written root "{{out}}" files "{{built.files}}"
```

### `save-html`

Write an element's `outerHTML` to disk.

| Param | Notes |
|---|---|
| `selector?=.` | `.` = whole document. |
| path | Destination file. |

```capy
save-html path "/tmp/snapshot.html"
```

---

## Events

### `emit`

Emit a custom progress event — visible to SSE/WebSocket callers, a no-op for
sync REST.

| Part | Notes |
|---|---|
| kind | `status`, `progress`, or anything. |
| text | Human-readable message (templated). |
| `data?` | Arbitrary structured payload. |

```capy
emit progress "Harvested {{items.length}} items" data { fraction: 0.5 }
```

See the [HTTP API](http-api.md) for the SSE event protocol.

---

## How results are stored

Result names and `path` interact consistently across actions:

- **Data actions** (`extract`, `js`, `http-get`, `set`, `for-each` results,
  `write-files`, …) store their result under the given name in both the response
  `data` and the live bindings. No name → the result is discarded.
- **Capture actions** (`screenshot`, `pdf`, `html-to-pdf`, `snapshot`,
  `record`, `export`, `capture-network`, `get-cookies`) accept:
    - `path` only → written to disk, nothing bound.
    - a result name only → returned inline (base64 for binary, text for `export`).
    - **both** → written to disk *and* bound as a `{path, size, bytesB64}` map
      (binary) or the text (export).
    - At least one is required, else the step errors.
- **`return`** sets the reserved result. When present, the HTTP response `data`
  is *that value alone*, not the full output map.

A param whose entire value is a single `{{ref}}` token resolves to the **raw**
bound value (list/map/number), not its stringified form — that's how
`for-each chat in "{{chats}}"` and `write-files files "{{built.files}}"` receive
real structured data. → [Templating](templating.md)
