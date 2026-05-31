# Writing tasks

A task is a **`.webtask` recipe**: a readable, top-to-bottom script that says
what the browser should do. One file = one task = one HTTP endpoint. This page
is the complete language reference.

---

## Anatomy of a recipe

```capy
# A header describes the endpoint…
task "basics/title"
    pool default                 # which Chrome pool to use
    timeout 15000                # whole-run budget, milliseconds
    transport rest               # how callers invoke it

    # …then steps run in order, like a checklist.
    status "Opening the page"
    goto "https://example.com"

    status "Reading the title"
    extract page from "html"
        title   text "title"
        heading text "h1"
    end
end
```

Reading rules:

- Text in `"quotes"` is literal — URLs, selectors, messages.
- Unquoted words are **keywords** the language understands (`pool`, `goto`,
  `text`, `end`).
- `#` starts a comment.
- Blocks (`task … end`, `extract … end`) are closed explicitly with `end`.

Save this as `tasks/basics/title.webtask` and it becomes
`POST /tasks/basics/title`.

---

## Header keywords

These sit at the top of the `task … end` block and describe the endpoint.

| Keyword | Example | Meaning |
|---|---|---|
| `task "NAME"` | `task "crawl/hn"` | Unique slug → the endpoint path. |
| `pool TAG` | `pool default` | Which window pool runs it (`default`, or your own). |
| `timeout MS` | `timeout 20000` | Max milliseconds for the whole run. |
| `transport KIND` | `transport rest` | `rest`, `sse`, `websocket`, or `async`. Repeat for several. |
| `setup "TASK"` | `setup "concio/setup"` | Run another recipe first, in the same window. |
| `input …` | see below | Declare a value the caller can pass in. |

### Timeout cheat sheet

| You write | Means |
|---|---|
| `timeout 5000` | 5 seconds |
| `timeout 15000` | 15 seconds |
| `timeout 60000` | 1 minute |

### Declaring inputs

When a caller sends `{"q": "cats"}`, the value fills `{{q}}` placeholders in
URLs, selectors, and typed text.

```capy
input q string default "chromedp" doc "Search query"
input limit string default "10"
input verbose bool default "false"
```

`input` parts, in order:

| Part | Values | Meaning |
|---|---|---|
| **name** | `q`, `url`, … | Used as `{{name}}` later. |
| **type** | `string`, `int`, `bool`, `float` | Value kind (for the schema). |
| `required` | flag | Caller must supply it. |
| `default "…"` | any text | Used when the caller omits it. |
| `doc "…"` | text | Shown in `GET /tasks` and API docs. |

→ Placeholders resolve via [Templating](templating.md).

---

## Step keywords

Each step is one browser action. An optional `status "…"` **before** a step
sets the live progress message SSE callers see.

### Navigate — `goto`

```capy
goto "https://example.com"
goto "https://duckduckgo.com/?q={{q}}"
```

### Wait — `wait` / `wait until`

```capy
wait 2000                                 # pause 2 seconds
wait until "tr.athing" timeout 10000      # until an element exists, or time out
```

### Type — `sendkeys`

```capy
sendkeys "input[name='q']" keys "{{q}}"
```

### Click — `click`

```capy
click "button[type='submit']"
```

### Read data — `extract`

`extract` pulls named fields out of the page. Add `repeat` to collect many rows.

```capy
# One region (an object)
extract page from "html"
    title   text "title"
    heading text "h1"
end

# Many rows (an array)
extract stories from "tr.athing" repeat
    rank  text ".rank" trim
    title text ".titleline > a"
    url   attr href on ".titleline > a"
end
```

Field kinds inside an `extract … end` block:

| Field line | Captures |
|---|---|
| `name text "SEL"` | Visible text. |
| `name text "SEL" trim` | Text, trimmed of surrounding whitespace. |
| `name attr ATTR on "SEL"` | An HTML attribute (e.g. `href`, `src`). |
| `name html "SEL"` | Inner HTML. |

### Run JavaScript — `js`

```capy
js stats fn "demo/page-stats.js" args {}      # a module under scripts/
```

### Variables & return — `set` / `return`

```capy
set greeting "hello"
return "{{page.title}}"
```

### Compose — `call`

```capy
call "control/helper-task"
```

---

## Capture & render

| Keyword | Example | Produces |
|---|---|---|
| `screenshot` | `screenshot shot selector "."` | Base64 PNG of an element/page. |
| `pdf` | `pdf doc path "/tmp/out.pdf" format A4` | A PDF on disk. |
| `snapshot` | `snapshot snap path "/tmp/page.mhtml"` | An MHTML archive. |
| `emulate` | `emulate dark` | Device / color-scheme emulation. |
| `record` | `record clip format gif fps 4 … end` | An animated GIF / MP4 of a flow. |

→ Full parameters in the [Actions reference](actions.md).

---

## Network & backend

| Keyword | Example | Does |
|---|---|---|
| `wait-network-idle` | `wait-network-idle 800ms timeout 20000` | Wait for network to settle. |
| `get-cookies` / `set-cookies` | `get-cookies jar` | Read / write cookies. |
| `http-get` / `http-post` | `http-get resp url "https://api…"` | Outbound HTTP, no browser. |
| `export` | `export csv path "/tmp/out.csv" data "{{rows}}"` | Write CSV / NDJSON / Markdown. |

---

## Build one from scratch

Let's write a task that searches DuckDuckGo and returns result titles.

**1. Header** — name it, pick a pool, set a budget, declare the query input:

```capy
task "search/my-search"
    pool default
    timeout 20000
    transport rest
    input q string default "golang" doc "What to search for"
```

**2. Navigate** with the templated query:

```capy
    goto "https://duckduckgo.com/?q={{q}}"
```

**3. Wait** for results to render:

```capy
    wait until "article[data-testid='result']" timeout 12000
```

**4. Extract** the list, then close the block and the task:

```capy
    extract results from "article[data-testid='result']" repeat
        title text "h2"
        url   attr href on "a[data-testid='result-title-a']"
    end
end
```

Save it under `tasks/search/my-search.webtask`, then — with the server already
running — call it (hot-reload picks it up immediately):

```bash
curl -s -X POST localhost:8765/tasks/search/my-search \
  -H 'Content-Type: application/json' \
  -d '{"q":"chromedp"}' | python3 -m json.tool
```

---

## Quick reference

| Keyword | Template |
|---|---|
| `task` | `task "NAME" … end` |
| `pool` | `pool TAG` |
| `timeout` | `timeout MS` |
| `transport` | `transport KIND` |
| `setup` | `setup "TASK"` |
| `input` | `input NAME TYPE [required] [default "…"] [doc "…"]` |
| `status` | `status "MESSAGE"` |
| `goto` | `goto "URL"` |
| `wait` | `wait MS` |
| `wait until` | `wait until "SEL" timeout MS` |
| `sendkeys` | `sendkeys "SEL" keys "TEXT"` |
| `click` | `click "SEL"` |
| `extract` | `extract NAME from "SEL" [repeat] … end` |
| `js` | `js NAME fn "path.js" args {…}` |
| `set` / `return` | `set NAME "VALUE"` · `return "VALUE"` |
| `call` | `call "TASK"` |
| `screenshot` / `pdf` / `record` | see [Actions](actions.md) |

**Placeholders:** anything written `{{name}}` is filled at run time from caller
inputs or resolved secrets. → [Templating](templating.md)

---

## What's next

- **[Examples](demos/index.md)** — 38 working recipes to copy from
- **[Actions reference](actions.md)** — every step and its parameters
- **[Recipes](cookbook.md)** — solutions to common problems
- **[Deployment](deploy.md)** — pools, secrets, packaging
