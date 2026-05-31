# 5. The `webtasks.capy` library

The library is the **complete grammar** for webtasks task files. It lives at
[`grammar/webtasks.capy`](grammar/webtasks.capy) in this repo (proposed; not yet
wired into the Go binary).

Install Capy CLI (use `@main` until tagged releases match module path):

```bash
go install github.com/olivierdevelops/capy/cmd/capy@main
capy check docs/capy-integration/grammar/webtasks.capy
capy run docs/capy-integration/grammar/webtasks.capy \
       docs/capy-integration/grammar/samples/01-hello.capy
```

---

## Library structure

| Section | Purpose |
|---|---|
| `extension yaml` | Output is YAML task definition |
| `comments` | Allow `#` in user task files |
| `context` | Accumulate `name`, `poolTag`, `flow[]`, `input{}`, … |
| `type … end` | Validate pools, transports, field kinds, actions |
| `function task … end` | Top-level task block |
| `function goto … end` | One function per DSL statement shape |
| `function extract … end` | Block function for field specs |
| `file_template … end` | Serialize `context` → YAML |

---

## Context schema

```capy
context
    name ""
    poolTag "default"
    transports []
    timeoutMs 60000
    setupTask ""
    input {}
    flow []
    _status ""              # optional status for next step
    _record false           # record: true for next step
end
```

Scratch fields (`_status`, `_record`) are stripped by `file_template` — they
exist so standalone step functions can attach step-level YAML keys.

---

## Core functions (excerpt)

### Task wrapper

```capy
function task
    description "Define a webtasks automation task."
    arg literal "task"
    arg capture slug string "URL slug, e.g. crawl/hackernews-top"
    block_closer end
    set context.name slug
end
```

### Pool and timeout

```capy
type PoolTag
    options "default" "concio" "colab"
end

function pool
    arg literal "pool"
    arg capture tag PoolTag
    set context.poolTag tag
end

function timeout
    arg literal "timeout"
    arg capture ms int "Milliseconds"
    set context.timeoutMs ms
end
```

A sugar function `timeout_duration` can parse `20s` → `20000` using inner-DSL
`if (regex_match dur "[0-9]+s$")` — see full library file.

### Flow step: goto

```capy
function goto
    arg literal "goto"
    arg capture url string
    append context.flow {
        run: "goto",
        status: context._status,
        params: { url: url }
    }
    set context._status ""
end
```

After each step, clear `_status` so `status "…"` only applies to the **next**
step (matching author intent).

### Block extract

```capy
function extract
    arg literal "extract"
    arg capture as_name ident
    arg literal "from"
    arg capture selector string
    arg literal "repeat:"
    block_closer end
    append context.flow {
        run: "extract",
        as: as_name,
        params: {
            selector: selector,
            repeat: true,
            fields: context._extractFields
        }
    }
    set context._extractFields {}
end
```

Field lines inside the block use a separate `field_*` function family that
mutates `context._extractFields`.

---

## file_template — YAML emission

Phase 1 uses `toJSON` + a small indent pass, or hand-written template:

```capy
file_template
    write `name: ${toQuoted context.name}
poolTag: ${toQuoted context.poolTag}
timeoutMs: ${context.timeoutMs}
transports: ${toJSON context.transports}

`
    if context.input
        write `input:
`
        for name in (keys context.input)
            write `  ${name}:
`
            write `    type: ${toQuoted context.input[name].type}
`
        end
    end
    write `flow:
`
    for step in context.flow
        write `  - run: ${toQuoted step.run}
`
        if step.params
            write `    params: ${toJSONIndent step.params}
`
        end
    end
end
```

Phase 2 may switch to `(toYAML context)` once a helper lands in Capy or in
webtasks' transpiler wrapper.

---

## Extending the library

Adding a new engine action (e.g. a future `hover` primitive):

1. Add `"hover"` to `type ActionName options …`
2. Add `function hover … end` mirroring [actions.md](../actions.md)
3. Run `capy docs webtasks.capy >> DSL_REFERENCE.md`
4. No Go changes until the action exists in `runtask_impl.go`

This inverts today's workflow (Go first, docs second) for task authors.

---

Next: [Action vocabulary →](06-actions-grammar.md)
