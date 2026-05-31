# 6. Action vocabulary in Capy

Every `run:` keyword from [actions.md](../actions.md) becomes one or more
`function` blocks in `webtasks.capy`. This chapter maps categories to DSL
shapes.

---

## Navigation and timing

| Action | Capy surface |
|---|---|
| `goto` | `goto "URL"` |
| `wait` | `wait 1s` / `wait 1000` |
| `wait-for` | `wait-for "selector" timeout 10s` |
| `wait-for-network-idle` | `wait-network-idle 800ms timeout 20s` |

Example transpilation:

```capy
wait-for "article.Box-row" timeout 15s
```

→

```yaml
- run: wait-for
  params:
    selector: "article.Box-row"
    timeoutMs: 15000
```

---

## Input and interaction

| Action | Capy surface |
|---|---|
| `sendkeys` | `sendkeys "selector" keys "{{text}}"` |
| `action(click)` | `click "selector"` |
| `action(double-click)` | `double-click "selector"` |
| `scroll-until-stable` | `scroll-until-stable "body" down stable 800ms max 20` |

Library implementation pattern:

```capy
function click
    arg literal "click"
    arg capture selector string
    append context.flow {
        run: "action",
        params: { action: "click", selector: selector }
    }
end
```

---

## JavaScript

| Action | Capy surface |
|---|---|
| `js` inline | `js result script: … end` (verbatim block) |
| `js` module | `js result fn "demo/page-stats.js" args ["{{url}}"]` |
| `await-js` | `await-js script: … end timeout 30s poll 500` |

Verbatim blocks use `block_verbatim`:

```capy
function js
    arg literal "js"
    arg capture as_name ident
    arg literal "script:"
    block_verbatim end
    append context.flow {
        run: "js",
        as: as_name,
        params: { script: body }
    }
end
```

---

## Extraction

```capy
extract VAR from "selector":
    field text "sub-selector" 
    field attr href on "a" 
end

extract VAR from "selector" repeat:
    ...
end
```

Field helper functions:

```capy
function field_text
    arg capture name ident
    arg literal "text"
    arg capture selector string
    append context._extractFields[name] {
        kind: "text",
        selector: selector
    }
end

function field_attr
    arg capture name ident
    arg literal "attr"
    arg capture attr ident
    arg literal "on"
    arg capture selector string
    append context._extractFields[name] {
        kind: "attr",
        selector: selector,
        name: attr
    }
end
```

Optional transform token:

```capy
title text ".titleline > a" trim
```

→ `transform: trim` in YAML.

---

## Rendering and capture

| Action | Capy surface |
|---|---|
| `screenshot` | `screenshot VAR selector "."` |
| `pdf` | `pdf VAR path "/tmp/x.pdf" format A4` |
| `html-to-pdf` | `html-to-pdf VAR html "{{tpl}}" path "…"` |
| `snapshot` | `snapshot VAR path "…"` |
| `emulate` | `emulate dark` / `emulate device "iPhone 12"` |
| `record` | `record VAR format gif fps 4: … end` |

---

## Network and session

| Action | Capy surface |
|---|---|
| `capture-network` | `capture-network VAR: … end` |
| `console` | `console VAR: … end` |
| `get-cookies` / `set-cookies` | `get-cookies VAR` / `set-cookies [{…}]` |
| `http-request` | `http-get VAR url "…"` / `http-post …` |

---

## Control flow

| Action | Capy surface |
|---|---|
| `call` | `call "other/task" ` |
| `return` | `return "{{expr}}"` |
| `loop` | `loop max N while js "…": … end` |
| `for-each` | `for-each x in "{{list}}": … end` |
| `set` | `set var "{{expr}}"` |

---

## Data and filesystem

| Action | Capy surface |
|---|---|
| `export` | `export csv path "/tmp/out.csv" data "{{items}}"` |
| `read-file` | `read-file VAR path "…"` |
| `write-files` | `write-files [{path: "…", content: "…"}]` |
| `save-html` | `save-html path "/tmp/page.html"` |
| `download-each` | `download-each VAR selector "a.download"` |

---

## Events

```capy
status "Loading GitHub trending"
emit progress "done scraping" data { fraction: 1.0 }
```

Maps to `status:` on next step and `run: emit-event`.

---

## Priority and disambiguation

Some tokens overlap (`wait` vs `wait-for`). Declare literals fully:

```capy
function wait-for
    priority 10
    arg literal "wait-for"
    ...
end

function wait
    priority 0
    arg literal "wait"
    ...
end
```

Higher priority wins when multiple functions could match a prefix.

---

Next: [Types and validation →](07-types-validation.md)
