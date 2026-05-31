# Build your own task

A practical, end-to-end walkthrough for going from "I want to scrape /
automate X" to a working `POST /tasks/my-thing` endpoint. Assumes you've
read the [README](../README.md) and have the server running against your
own bundle.

The big picture:

```
1. Pick a site, open it in Chrome, find selectors
2. Sketch the flow as numbered steps
3. Translate each step into one YAML command
4. Define inputs the caller will supply
5. Extract data into shaped JSON
6. Drop the file in your bundle, hot-reload, call it
```

---

## 0. Set yourself up

```bash
mkdir -p my-bundle/tasks my-bundle/scripts
# minimal pool config (one default window):
cat > my-bundle/tasks/pool.yaml <<EOF
pools:
  default: { size: 1 }
EOF

WEBTASKS_BUNDLE=$(pwd)/my-bundle WEBTASKS_HEADLESS=false executor server &
```

Setting `WEBTASKS_HEADLESS=false` opens a visible Chrome window — invaluable
while you're learning what selectors actually match. Switch back to headless
once the task works.

The server **hot-reloads** YAML on every request. Edit a file, immediately
re-run `executor call …` — no restart needed.

---

## 1. Find your selectors in DevTools

Open the target page in regular Chrome. DevTools → Elements → right-click
the node you want → **Copy → Copy selector**. That's your starting selector.
Refine it by hand:

| What you want | Good selector pattern |
|---|---|
| All rows of a table | `tr.athing`, `article.row`, etc. |
| One field inside a row | `.title`, `[data-field='price']` |
| A button labelled "Submit" | `button[type='submit']`, `form button` |
| An anchor by its `href` start | `a[href^='/papers/']` |

Sanity-check by pasting into the console: `document.querySelectorAll('your-selector').length`.
Aim for selectors that are stable across page versions — prefer semantic
attributes (`[name=…]`, `[data-…]`) over auto-generated class hashes
(`._3kF9p`).

---

## 2. Sketch the flow in plain English

A useful template:

```
1. Navigate to <URL>
2. Wait until <thing> is visible
3. (Type things into inputs)
4. (Click a button)
5. Wait for results
6. Extract data into <shape>
7. (Save / download anything)
```

Concrete example — "list trending repos on GitHub for a language":

```
1. Goto https://github.com/trending/<language>?since=<since>
2. Wait for article.Box-row to be present
3. Extract every article.Box-row → { slug, href, description, stars, forks }
```

---

## 3. Translate each step into one YAML command

The full action vocabulary is in [cookbook.md → Action reference](cookbook.md#action-reference).
Common pattern:

```yaml
name: "my/github-trending"
poolTag: "default"
transports: ["rest"]
timeoutMs: 30000

input:
  language: { type: string, required: false, default: "" }
  since:    { type: string, required: false, default: "daily" }

flow:
  - run: goto
    params: { url: "https://github.com/trending/{{language}}?since={{since}}" }

  - run: wait-for
    params: { selector: "article.Box-row", timeoutMs: 15000 }

  - run: extract
    as: repos
    params:
      selector: "article.Box-row"
      repeat: true
      fields:
        slug:        { kind: text, selector: "h2 a", transform: trim }
        href:        { kind: attr, selector: "h2 a", name: "href" }
        description: { kind: text, selector: "p" }
        stars:       { kind: text, selector: "a[href$='/stargazers']", transform: trim }
        forks:       { kind: text, selector: "a[href$='/forks']", transform: trim }
```

Save this to `my-bundle/tasks/my/github-trending.yaml`. Try it:

```bash
executor call my/github-trending
executor call my/github-trending '{"language":"go","since":"weekly"}'
```

---

## 4. Inputs done right

The `input:` block declares (a) what the caller can pass and (b) what shows
up in `GET /tasks` so other consumers know how to use you.

```yaml
input:
  query:       { type: string, required: true,  doc: "What to search for" }
  pageSize:    { type: int,    required: false, default: 25 }
  includeAds:  { type: bool,   required: false, default: false, doc: "…" }
```

Reference values in any string param as `{{name}}`:

```yaml
- run: goto
  params: { url: "https://example.com/?q={{query}}&n={{pageSize}}" }
```

Tips:

- `required: true` with no caller value → the server errors before any
  browser action runs. Use this for things the task literally cannot work
  without (a URL, a user id).
- `required: false` + `default:` → the task is callable with no args. Use
  this for "sensible defaults" that make the task runnable as a demo.
- Templating falls back to **environment variables** when a binding is
  missing, so `{{API_TOKEN}}` works if a secret named `API_TOKEN` is
  declared in `secrets.yaml`.

---

## 5. Shape the output with `extract`

Most tasks return at least one object. The `extract` action takes a
selector + a field spec, and either returns one object (`repeat: false`) or
one per match (`repeat: true`).

```yaml
- run: extract
  as: data
  params:
    selector: "article.product"   # rows to harvest
    repeat: true
    fields:
      name:   { kind: text, selector: ".name" }
      price:  { kind: text, selector: ".price", transform: int }
      inStock:{ kind: const, value: true }
      image:  { kind: attr, selector: "img", name: "src" }
      desc:   { kind: html, selector: ".desc" }
```

Field kinds:

| Kind | What it returns |
|---|---|
| `text` (default) | Trimmed `.textContent` of `selector` |
| `attr` | `getAttribute(name)` value |
| `html` | Inner HTML of `selector` |
| `const` | The literal `value:` (lets you tag every record) |

Transforms (applied to the text/attr result):

- `int` / `long` — parse a number (returns `null` if it can't)
- `trim` — strip whitespace
- `lower` / `upper` — case

`as:` names the key in the response. With `as: data`, the body becomes
`{ "ok": true, "data": { "data": [...] } }`. Pick keys that read well from
the caller's side.

---

## 6. Multi-step flows

Several `extract` blocks can run in a row, each writing to its own `as:`.

```yaml
flow:
  - run: goto
    params: { url: "https://example.com/u/{{user}}" }
  - run: wait-for
    params: { selector: ".profile" }

  - run: extract
    as: profile
    params:
      selector: ".profile"
      repeat: false
      fields:
        name:  { kind: text, selector: "h1" }
        bio:   { kind: text, selector: ".bio" }

  - run: extract
    as: repos
    params:
      selector: ".repo-card"
      repeat: true
      fields:
        name: { kind: text, selector: ".name" }
        url:  { kind: attr, selector: "a", name: "href" }
```

Response: `{ "data": { "profile": {…}, "repos": […] } }`.

---

## 7. When CSS isn't enough — use a JS module

For non-selector logic (find by text, walk a tree, post-process), drop a JS
file in `bundle/scripts/<path>.js` and reference it with `fn:`. The script
runs as the body of a function whose `arguments` come from the YAML.

`my-bundle/scripts/my/total-price.js`:

```js
// arguments[0] = optional currency symbol to strip
const symbol = arguments[0] || '$';
let total = 0;
for (const el of document.querySelectorAll('.cart-row .price')) {
    const n = parseFloat(el.textContent.replace(symbol, '').replace(',', ''));
    if (!isNaN(n)) total += n;
}
return { total, currency: symbol };
```

```yaml
- run: js
  as: cart
  params:
    fn: "my/total-price"
    args: ["£"]
```

Three reasons to prefer `fn:` over inline `script:`:

1. Modules can be tested in DevTools console as-is
2. Reused across tasks
3. Don't clutter the YAML with 50-line JS blobs

---

## 8. Stream progress for slow tasks

Any `status:` field on a command becomes an event when the caller asks for
SSE. Add `emit-event` for custom progress:

```yaml
flow:
  - status: "Loading product list"
    run: goto
    params: { url: "https://example.com/catalog" }

  - run: emit-event
    params:
      kind: "progress"
      text: "Catalog loaded, harvesting"
      data: { fraction: 0.3 }

  - run: extract
    as: items
    params: { selector: ".product", repeat: true, fields: {…} }
```

Caller side:

```bash
executor call my/slow-task '{}' true     # SSE stream
```

Sync REST callers see nothing extra — events are dropped on the floor.

---

## 9. Iterate quickly

Workflow that keeps the feedback loop tight:

1. **Run with `WEBTASKS_HEADLESS=false`** so you can watch the browser. Add
   a `wait: { duration: "5_000" }` at the end of a flow to pause before
   shutdown if you want to inspect state.
2. **Use `save-html`** when the page is dynamic and you want to study the
   DOM offline:
   ```yaml
   - run: save-html
     params: { path: "/tmp/snapshot.html" }
   ```
   Then `cat /tmp/snapshot.html | grep …` or open it in your editor.
3. **Use `screenshot`** to verify the page actually rendered before
   extraction. Capture to `as: png_b64`, decode with:
   ```bash
   executor call my/task | jq -r .data.png_b64 | base64 -d > /tmp/shot.png
   ```
4. **`executor call <name> '{…}' true`** — SSE mode shows you which step
   each event came from, so you see exactly where the task hangs.

---

## 10. Common patterns

### Open by visible text (no stable selector)

```yaml
- run: js
  params:
    fn: "my/click-by-text"
    args: ["Submit", "form"]
```

Pair with a small JS module:

```js
const target = arguments[0], scopeSel = arguments[1];
const root = scopeSel ? document.querySelector(scopeSel) : document;
const el = Array.from(root.querySelectorAll('*'))
    .find(e => e.children.length === 0 && e.textContent.trim() === target);
if (!el) return false;
const opts = { bubbles: true, cancelable: true, view: window, button: 0 };
el.dispatchEvent(new MouseEvent('mousedown', opts));
el.dispatchEvent(new MouseEvent('mouseup', opts));
el.dispatchEvent(new MouseEvent('click', opts));
return true;
```

### Walk many pages

`extract` doesn't loop, so build the flow once per page (or have the
**caller** loop — call `my/page` with `{"page":1}`, `{"page":2}`, etc.).

### Login then do something

```yaml
- run: goto
  params: { url: "https://example.com/login" }
- run: wait-for
  params: { selector: "#username" }
- run: sendkeys
  params: { selector: "#username", keys: "{{user}}" }
- run: sendkeys
  params: { selector: "#password", keys: "{{PASSWORD}}" }   # from secrets.yaml
- run: action
  params: { action: click, selector: "button[type='submit']" }
- run: wait-for
  params: { selector: ".dashboard", timeoutMs: 10000 }
```

Declare the password in `secrets.yaml`:

```yaml
secrets:
  - name: PASSWORD
    sensitive: true
    required: true
    sources: ["env", "arg", "prompt"]
```

### Wait for "real" data, not just DOM presence

`wait-for` matches "selector present in DOM". For SPAs that show a
skeleton first, prefer waiting for the loaded state:

```yaml
- run: wait-for
  params: { selector: ".results .item:not(.loading)", timeoutMs: 20000 }
```

Or use `wait` + `js` to poll a JS predicate.

### Download a file the site links to

```yaml
- run: download-each
  as: downloaded
  params:
    selector: "a.download-link"
    timeoutPerFileMs: 30000
```

Files land in the per-window download dir (exposed at `/downloads` via the
default mount). Result is `[{ path, basename }, …]`.

---

## 11. Test → ship checklist

Before adding the task to a shared bundle:

- [ ] Runs cleanly with no args (sensible defaults)
- [ ] Runs cleanly with realistic args
- [ ] Returns sensible JSON shape (object with named keys, not raw lists at
      the top level)
- [ ] No hardcoded secrets in YAML — use `{{ENV_NAME}}` + `secrets.yaml`
- [ ] `status:` strings present on slow steps so SSE callers know what's
      happening
- [ ] Doesn't depend on a window being "already opened to X" — every task
      starts from a clean window

---

## 12. Cheatsheet

```yaml
name: "category/short-name"
poolTag: "default"                       # which window pool to use
transports: ["rest"]                     # sync REST; add "sse" if you stream
timeoutMs: 30000                         # whole-task budget

input:                                   # caller-supplied values
  field: { type: string, required: true, default: "x", doc: "…" }

flow:
  - status: "Human-readable progress"    # appears in SSE
    run: goto
    params: { url: "https://…?q={{field}}" }

  - run: wait-for
    params: { selector: "h1", timeoutMs: 10000 }

  - run: sendkeys
    params: { selector: "#q", keys: "{{field}}" }

  - run: action
    params: { action: click, selector: "button" }

  - run: scroll-until-stable
    params: { selector: ".feed", direction: down, stableMs: 1500, maxIterations: 10 }

  - run: js
    as: extras
    params: { fn: "my/helper", args: ["{{field}}"] }

  - run: extract
    as: items
    params:
      selector: ".item"
      repeat: true
      fields:
        title: { kind: text, selector: ".t" }
        href:  { kind: attr, selector: "a", name: "href", transform: trim }

  - run: emit-event
    params: { kind: "progress", text: "harvested {{items.length}} items" }

  - run: download-each
    as: files
    params: { selector: "a.attachment", timeoutPerFileMs: 30000 }
```

That's the whole shape. Everything else is selectors and inputs.
