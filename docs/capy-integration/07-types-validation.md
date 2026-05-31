# 7. Types and validation

Capy types turn webtasks' implicit schema into **compile-time checks**. This
chapter lists recommended `type` blocks for `webtasks.capy`.

---

## Why types matter for browser automation

Without validation, these reach production:

```
got "https://example.com"          # typo
pool prod                          # undefined pool → runtime lease error
extract x from "div" repeat true   # repeat: string "true" → subtle bug
transport websoket                 # typo in transports metadata
```

With `type` + `options` / `pattern`:

```
function "pool" arg "tag": value "prod" not in options for type "PoolTag"
```

Authors fix tasks **before** starting Chrome.

---

## Recommended type definitions

### PoolTag

```capy
type PoolTag
    options "default" "concio" "colab"
end
```

Extend when `pool.yaml` declares new pools. Optional: generate `options` list
from `pool.yaml` at bundle pack time via a small Go tool.

### Transport

```capy
type Transport
    options "rest" "sse" "websocket" "async"
end
```

### FieldKind

```capy
type FieldKind
    options "text" "attr" "html" "const"
end
```

### Transform

```capy
type Transform
    options "trim" "lower" "upper" "int" "long"
end
```

### PdfFormat

```capy
type PdfFormat
    options "letter" "legal" "a4" "a3"
end
```

### ExportFormat

```capy
type ExportFormat
    options "csv" "ndjson" "md" "markdown" "md-table"
end
```

### InputType (documentation hint)

```capy
type InputType
    options "string" "int" "bool" "float"
end
```

---

## Regex-constrained types

### TaskSlug

```capy
type TaskSlug
    pattern "^[a-z][a-z0-9_/\\-]*$"
end
```

Prevents spaces and uppercase in slugs that become URL paths.

### CSSSelector (light check)

```capy
type CSSSelector
    base string
    pattern ".+"    # non-empty; full CSS validation is impractical
end
```

### Duration

```capy
type Duration
    pattern "^[0-9]+(ms|s|m)$|^[0-9]+$"
end
```

Matches `10s`, `800ms`, `60000`.

### ModulePath

```capy
type ModulePath
    pattern "^[a-z][a-z0-9_/\\-]*\\.js$"
end
```

For `fn "demo/page-stats.js"`.

---

## Group types for inline params

Markdown-style optional metadata:

```capy
type Bracketed
    group_open "["
    group_close "]"
end

function input
    arg literal "input"
    arg capture name ident
    arg capture typ InputType
    arg capture flags Bracketed   # [required doc "…"]
    ...
end
```

Source: `input q string [required doc "Query"]`

---

## Sandboxing for AI-generated tasks

Combine types with closed `options` lists:

| Type | Effect |
|---|---|
| `PoolTag` | Cannot invent pools |
| `Transport` | Cannot register fake transports |
| `ActionName` (enum of all `run:` values) | Cannot call undefined actions |
| `ModulePath` | JS modules must match `scripts/**/*.js` layout |

For Concio-style bundles, add:

```capy
type ConcioTask
    options "concio/setup" "concio/list-chats" "concio/get-messages"
end

function call
    arg literal "call"
    arg capture target ConcioTask
    ...
end
```

Agents cannot `call "rm -rf"` — not in the enum.

---

## Validation workflow

```bash
# Per task
capy run capy/webtasks.capy tasks/crawl/hn.capy > /dev/null && echo OK

# Library + all tasks
capy check capy/webtasks.capy
for f in tasks/**/*.capy; do
  capy run capy/webtasks.capy "$f" > "tasks/${f%.capy}.gen.yaml"
done
diff -ru tasks/ tasks-check/   # golden test
```

In CI:

```yaml
- run: go install github.com/olivierdevelops/capy/cmd/capy@latest
- run: capy check capy/webtasks.capy
- run: ./scripts/verify-capy-tasks.sh
```

---

Next: [Integration architecture →](08-architecture.md)
