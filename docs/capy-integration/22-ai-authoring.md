# 22. AI authoring with Capy

How [Capy](https://github.com/olivierdevelops/capy) makes webtasks easier for
AI agents (Claude, Cursor, etc.) — based on
[docs/ai-agents.md](https://github.com/olivierdevelops/capy/blob/main/docs/ai-agents.md).

---

## Token compression

Agents emit **short DSL**; Capy expands to verbose YAML deterministically.

| Task | YAML lines | Capy lines | Expansion ratio |
|---|---|---|---|
| basics/title | ~22 | ~9 | ~2.4× |
| crawl/hackernews-top | ~25 | ~12 | ~2.1× |
| interaction/form-fill | ~45 | ~18 | ~2.5× |
| concio/get-messages (sketch) | ~60 | ~15 | ~4× |

In an agent loop, the library (`webtasks.capy`) is loaded **once** in context;
each invocation only sends the short task source.

---

## Sandboxing

The grammar is the security boundary:

```capy
type ActionName
    options "goto" "wait-for" "extract" "call" ...
end

type PoolTag
    options "default" "concio"
end
```

An agent **cannot** transpile:

- Undefined actions (`run: eval`)
- Wrong pools (`pool admin`)
- Arbitrary `call` targets (when restricted by enum)

No post-hoc regex filtering of YAML — invalid source fails at `capy run`.

---

## Context documents for agents

Provide agents three files:

1. [`CAPY_FOR_LLMS.md`](https://github.com/olivierdevelops/capy/blob/main/docs/CAPY_FOR_LLMS.md) — Capy mechanics
2. `capy/webtasks.capy` — the grammar (or `capy docs` output)
3. [actions.md](../actions.md) — semantic reference for each transpiled action

Prompt pattern:

> Author a webtasks task in Capy DSL matching `webtasks.capy`. Transpile with
> `capy run` before submitting. Do not output YAML directly.

---

## MCP integration

Capy ships [`capy-mcp`](https://github.com/olivierdevelops/capy/blob/main/docs/mcp.md):

```bash
go install github.com/olivierdevelops/capy/cmd/capy-mcp@latest
```

Agents can call transpile/check without shell access — wire into Cursor MCP
config alongside webtasks server tools.

---

## Error feedback loop

Capy errors are caret-pointed:

```
tasks/new-task.capy:8: function "goto" arg "url": value "" does not match type "string"
```

Agents read the error, fix line 8, re-run — faster than debugging a silent
YAML logic bug in Chrome.

---

Next: [Editor tooling →](23-editor-tooling.md)
