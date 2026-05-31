# Recipes

Practical, copy-paste solutions for common automation problems — written in the
`.webtask` language. New to recipes? Start with [Writing tasks](writing-tasks.md);
look up any action in the [Actions reference](actions.md).

---

## Table of contents

- [1. Write your first task](#1-write-your-first-task)
- [2. Accept inputs from the caller](#2-accept-inputs-from-the-caller)
- [3. Extract a list of records](#3-extract-a-list-of-records)
- [4. Use a JS module instead of inline JS](#4-use-a-js-module)
- [5. Click an element by its visible text](#5-click-by-visible-text)
- [6. Load history by scrolling until stable](#6-scroll-until-stable)
- [7. Trigger downloads and capture the bytes](#7-downloads)
- [8. Stream progress to the caller (SSE)](#8-stream-progress)
- [9. Use secrets safely](#9-secrets)
- [10. Serve captured files over HTTP](#10-static-mounts)
- [11. Ship a deployment bundle](#11-deployment-bundle)

---

## 1. Write your first task {#1-write-your-first-task}

Create `tasks/recipes/title.webtask`:

```capy
task "recipes/title"
    pool default
    timeout 15000
    transport rest

    goto "https://example.com"
    extract page from "h1"
        title text "."
    end
end
```

The server hot-reloads on every request, so no restart is needed:

```bash
curl -s -X POST localhost:8765/tasks/recipes/title -d '{}'
# → { "ok": true, "data": { "page": { "title": "Example Domain" } } }
```

- `task "recipes/title"` is the URL slug (`POST /tasks/recipes/title`).
- `pool default` picks which browser pool to lease from.
- The steps run in order.
- `extract page …` puts the result into the response under `data.page`.

---

## 2. Accept inputs from the caller {#2-accept-inputs-from-the-caller}

Declare an input and reference it with `{{name}}` templating.

```capy
task "recipes/search"
    pool default
    timeout 30000
    transport rest
    input q string required doc "Query string"

    goto "https://duckduckgo.com/?q={{q}}"
    wait until "article[data-testid='result']" timeout 10000
    extract results from "article[data-testid='result']" repeat
        title text "h2"
        link  attr href on "a"
    end
end
```

```bash
curl -s -X POST localhost:8765/tasks/recipes/search -d '{"q":"chromedp"}'
```

Defaults and the `or:` fallback work too:

```capy
input q string default "go"
# …and inside a string you can fall back: "https://duckduckgo.com/?q={{q|or:go}}"
```

---

## 3. Extract a list of records {#3-extract-a-list-of-records}

`extract … repeat` runs the field spec against each match and returns an array.

```capy
extract items from "ul.product-list > li" repeat
    name  text ".name"
    price text ".price" trim
    sku   attr data-sku on "."
    url   attr href on "a"
    tags  html ".tags"
end
```

Field kinds: `text` (default), `attr ATTR on "SEL"`, `html`, `const "VALUE"`.
Add `trim` to a text field to strip whitespace.

---

## 4. Use a JS module instead of inline JS {#4-use-a-js-module}

Inline JS gets unreadable past a few lines. Drop the script into
`scripts/<path>.js` and reference it by name.

`scripts/recipes/click-by-text.js`:

```js
// Click the first descendant of arguments[1] whose text matches arguments[0].
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

Reference it from a recipe:

```capy
js clicked fn "recipes/click-by-text" args ["{{button}}", "form"]
```

The dispatched-event trick suits simple sites; for apps that require
`isTrusted` (most modern React/Vue), prefer the built-in `click` action.

---

## 5. Click an element by its visible text {#5-click-by-visible-text}

When you only know an element by its text, `click` can match on it and retarget
to a row ancestor:

```capy
click ".name" text "Nicholas Huang" match exact closest ".chat-row"
```

For sites that gate handlers on `isTrusted` *and* need a real MouseEvent
sequence on an inner node, a small `js` module (recipe 4) is sometimes clearer.

---

## 6. Load history by scrolling until stable {#6-scroll-until-stable}

For infinite-scroll panels (chat history, feeds):

```capy
scroll until stable ".chats.chat-content-scroll" direction up stable 2500 max 0
```

The action sets `scrollTop` and re-queries `scrollHeight` in a loop. Each new
prepend keeps it going; when the site stops loading, the height stabilises and
the action returns. (`max 0` = unbounded.)

---

## 7. Trigger downloads and capture the bytes {#7-downloads}

### 7a. Normal HTTP downloads → `download-each`

The browser writes each file to the per-window download directory; the action
clicks each match natively and polls for it.

```capy
download-each downloads selector "a.download-link" timeout 30000
```

Result: `data.downloads = [{ path, basename }, …]`.

### 7b. Client-side decryption → install a JS hook + drain

Apps that decrypt blobs in the browser bypass the normal download flow. Install
a hook on `URL.createObjectURL`, click each file, then drain the captures.

```capy
js _ fn "concio/install-download-hook"
js _ fn "concio/open-chat-by-name" args ["{{peerName}}"]
wait until ".chats.chat-content-scroll" timeout 15000
download-each _ selector ".chats.chat-content-scroll .chat-right-file-box" timeout 2500
wait 8000                                # let blob.arrayBuffer() finish
save-captures-to-dir saved dir "{{outDir}}" naming "{id}_{name}"
```

Result: `data.saved = [{ path, basename, size, mime, name, url }, …]`.

---

## 8. Stream progress to the caller (SSE) {#8-stream-progress}

The same `POST /tasks/<name>` endpoint switches to server-sent events when the
caller sends `Accept: text/event-stream`. Every `status` line becomes a `status`
event; the final response is a `done` event.

```bash
curl -N -X POST localhost:8765/tasks/crawl/trending-papers \
     -H 'Accept: text/event-stream' -d '{}'
```

Emit custom events with `emit`:

```capy
emit progress "Processed {{i}}/{{n}}" data { fraction: 0.5 }
```

→ Full protocol in the [HTTP API](http-api.md).

---

## 9. Use secrets safely {#9-secrets}

Declare a `CONCIO_PASSWORD` secret in the bundle (required + sensitive), resolved
at startup from the environment, a launcher flag, or an interactive prompt. Then
reference it in any recipe — no `input` entry needed:

```capy
sendkeys "#password" keys "{{CONCIO_PASSWORD}}"
```

Keep secrets out of shell history by storing them in a secret manager and
launching via `sm exec -- webtasks`. → [Secrets](deploy.md#secrets)

---

## 10. Serve captured files over HTTP {#10-static-mounts}

Declare a static mount in the bundle that exposes the downloads directory:

- `prefix: /downloads`
- `dir: ${WEBTASKS_DOWNLOADS_DIR:-build/downloads}`
- `list: true` (JSON listing at `GET /downloads`)
- `serve: true` (stream files at `GET /downloads/<file>`)

`${ENV}` / `${ENV:-default}` placeholders expand at startup.
→ [Static mounts](deploy.md#static-file-mounts)

---

## 11. Ship a deployment bundle {#11-deployment-bundle}

The binary contains no config. Ship it alongside a bundle:

```bash
executor package
# → dist/webtasks      (static binary)
# → dist/bundle.zip    (your tasks/, scripts/, config)

# On the target host (Chrome installed):
WEBTASKS_BUNDLE=$(pwd)/bundle.zip webtasks
```

The same binary serves any deployment — supply a different bundle to change
behaviour. The zip is read in-place (no extraction).
→ [Deployment](deploy.md)

---

## Troubleshooting

**"unknown task: foo/bar"** — the recipe wasn't picked up. List tasks with
`curl -s localhost:8765/tasks`. Hot-reload re-reads recipes on every call, so a
typo is the usual cause.

**"acquire timeout: <pool>"** — no free window for 30 s. Raise the pool size or
shorten upstream tasks. → [Pools](deploy.md#window-pools-sessions)

**"browser session was reset (tab crashed or detached)"** — Chrome lost the
target. The pool already replaced the window; re-run any required `setup`/login
task before retrying.

**Click does nothing** — the site may gate on `isTrusted`. Use the built-in
`click` action (native CDP click) rather than a JS-dispatched event. For inner
click handlers, use a `js` module that dispatches a proper MouseEvent sequence
(recipe 4).

**Secrets prompt blocks in CI** — `prompt` is skipped without a terminal, so
supply secrets via `--NAME=value` flags or env vars, or give optional secrets a
default. → [Secrets](deploy.md#secrets)
