# 3. YAML ↔ Capy mapping

Every field in [task-definition.md](../task-definition.md) maps to a Capy
construct. This table is the Rosetta stone for migration and library design.

---

## Task-level fields

| YAML field | Type | Capy DSL (proposed) | Context mutation |
|---|---|---|---|
| `name` | string | `task "crawl/hn"` … `end` | `set context.name slug` |
| `poolTag` | string | `pool default` | `set context.poolTag tag` |
| `transports` | list | `transport rest` (repeatable) | `append context.transports t` |
| `timeoutMs` | int | `timeout 20s` or `timeout 20000` | `set context.timeoutMs ms` |
| `setupTask` | string | `setup concio/setup` | `set context.setupTask name` |
| `input` | map | `input q string required` | `set context.input.q {...}` |
| `flow` | list | body statements inside `task` | `append context.flow cmd` |

---

## Input field schema

YAML:

```yaml
input:
  q:
    type: string
    required: true
    default: ""
    doc: "Search query"
```

Proposed Capy:

```capy
input q string required doc "Search query"
# or with default:
input language string default "" doc "GitHub language filter"
```

Inner representation in `context.input`:

```json
{
  "q": {
    "type": "string",
    "required": true,
    "doc": "Search query"
  }
}
```

The `file_template` serializes this map under `input:` in YAML.

---

## Flow command shape

YAML step:

```yaml
- status: "Loading page"
  run: goto
  as: page
  record: true
  params:
    url: "https://example.com"
  do:
    - run: wait-for
      params: { selector: "body" }
```

Proposed Capy:

```capy
status "Loading page"
goto "https://example.com"
# or block action:
record clip:
    goto "https://example.com"
    wait-for "body"
end
```

Context object appended to `context.flow`:

```json
{
  "run": "goto",
  "status": "Loading page",
  "params": { "url": "https://example.com" }
}
```

---

## Extract field specs

YAML:

```yaml
fields:
  title: { kind: text, selector: ".titleline > a", transform: trim }
  url:   { kind: attr, selector: ".titleline > a", name: "href" }
```

Proposed Capy (inside `extract` block):

```capy
extract stories from "tr.athing" repeat:
    rank  text ".rank" trim
    title text ".titleline > a"
    url   attr href on ".titleline > a"
end
```

Each field line appends to `context._currentExtract.fields`.

---

## Templating

No change. Transpiled YAML still uses `{{name}}` in string params. The Capy
DSL uses plain strings; templates are preserved verbatim in output:

```capy
goto "https://duckduckgo.com/?q={{q}}"
```

→ YAML `url: "https://duckduckgo.com/?q={{q}}"`

Optional sugar (library extension):

```capy
goto https://duckduckgo.com/?q={{q}}
```

(unquoted URL token captured as string)

---

## Side-by-side: full task

=== "YAML (today)"

    ```yaml
    name: "crawl/hackernews-top"
    poolTag: "default"
    transports: ["rest"]
    timeoutMs: 20000

    flow:
      - run: goto
        params: { url: "https://news.ycombinator.com" }
      - run: wait-for
        params: { selector: "tr.athing", timeoutMs: 10000 }
      - run: extract
        as: stories
        params:
          selector: "tr.athing"
          repeat: true
          fields:
            title: { kind: text, selector: ".titleline > a" }
    ```

=== "Capy (proposed)"

    ```capy
    task "crawl/hackernews-top"
        pool default
        timeout 20s
        transport rest

        goto "https://news.ycombinator.com"
        wait-for "tr.athing" timeout 10s
        extract stories from "tr.athing" repeat:
            title text ".titleline > a"
        end
    end
    ```

Output YAML is byte-for-byte equivalent modulo key ordering.

---

## Files that stay YAML

| File | Reason |
|---|---|
| `tasks/pool.yaml` | Simple numeric map; no flow |
| `secrets.yaml` | Declarative env contract |
| `static-mounts.yaml` | Path mounts |
| `scripts/**/*.js` | Unchanged — still referenced via `fn:` |

---

Next: [Proposed grammar overview →](04-proposed-grammar.md)
