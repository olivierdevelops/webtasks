# 22. AI authoring with Capy v0.20.0

How Capy makes webtasks easier for AI agents — based on
[Capy's AI agents guide](https://github.com/olivierdevelops/capy/blob/main/docs/ai-agents.md)
and the [v0.20 overview](00-v020-overview.md).

---

## The problem with agents writing YAML

Asking a model to emit task YAML directly costs ~25–45 lines per task, with:

- Invented `run:` keywords (`gotoo`, `extracts`)
- Wrong nesting under `params:` / `fields:`
- Drift between `call "task/a"` and the actual `name:` slug
- No validation until Chrome runs

---

## The Capy loop (deterministic steps 3–4)

```
1. lib.Introspect()              → agent learns allowed verbs + arg types
2. draft tasks/crawl/hn.capy     → ~12 lines of DSL (~40 tokens)
3. capy check webtasks.capy …    → parse error with line:col? retry
4. lib.RunMulti(source)          → tasks/…/*.yaml (NoOpHost, no FS access)
5. executor call crawl/hn        → existing HTTP API + Chrome
```

Only step 2 is the model. Steps 3–4 are Go code you ship.

---

## Token compression

| Artifact | Approx. tokens |
|---|---|
| Hand-written YAML task | 150–300 |
| Capy DSL source | 40–80 |
| Expanded YAML (deterministic) | 150–300 |

The agent writes the **short** form; your library expands it. The library is
loaded once per session — amortized across hundreds of task edits.

---

## Parser-as-sandbox

Combined with default **NoOpHost**:

- Tokens not in `webtasks.capy` → parse error, no file written
- No `env` / `read_file` during codegen unless you opt into `OSHost`
- Enum types (`PoolTag`, `ActionName`) block invented values

The model cannot emit arbitrary YAML keys — only shapes the grammar accepts.

---

## MCP integration (v0.20.0)

```bash
go install github.com/olivierdevelops/capy/cmd/capy-mcp@v0.20.0
```

Wire into Cursor / Claude Desktop per
[MCP docs](https://github.com/olivierdevelops/capy/blob/main/docs/mcp.md).
Agents can introspect, check, and transpile without shell access.

---

## Prompt assets

Provide agents:

1. [CAPY_FOR_LLMS.md](https://github.com/olivierdevelops/capy/blob/main/docs/CAPY_FOR_LLMS.md)
2. `capy docs webtasks.capy` output (auto-generated reference)
3. [actions.md](../actions.md) — semantic meaning of transpiled `run:` steps

Instruction:

> Author webtasks tasks in Capy DSL only. Run `capy check` before submitting.
> Do not output YAML directly.

---

## Metaprogramming (advanced)

Scripts can declare one-off grammar extensions with `define NAME … end` blocks
merged before evaluation — useful for generated task families without editing
the shared library. See
[metaprogramming.md](https://github.com/olivierdevelops/capy/blob/main/docs/metaprogramming.md).

---

Next: [Editor tooling →](23-editor-tooling.md)
