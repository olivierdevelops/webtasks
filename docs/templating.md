# Templating

Every string parameter in a task's `flow:` (and every `status:` line) is run
through the templating engine before the action sees it. The syntax is
`{{name}}`.

Implemented in `MakeTemplating`
([makes.go](../internal/orchestrator/features/makes.go)) and applied recursively
by `renderParams`/`renderValue`
([runtask_impl.go](../internal/orchestrator/usecases/runtask_impl.go)).

---

## What gets templated

`renderValue` walks each param recursively:

- **strings** are substituted,
- **list items** are each substituted,
- **map values** are each substituted,
- other types (numbers, bools) pass through unchanged.

So `headers: { Authorization: "Bearer {{TOKEN}}" }` and
`args: ["{{a}}", "{{b}}"]` both work.

---

## Lookup order

For a token `{{key}}`:

1. **The task's bindings** — declared inputs (after `bindInputs`), plus every
   prior step's `as:` result, plus any iteration variables (`for-each`'s
   `{{item}}`/`{{item_index}}`, `loop`'s `{{loop_index}}`).
2. **The process environment** — `os.Getenv(key)`. This is how resolved
   secrets reach templating: the secrets loader exports each one as an env var,
   so `{{CONCIO_PASSWORD}}` resolves even though it was never an input.
   → [secrets.md](secrets.md)

If neither resolves (value is `nil` or empty string), the `or:` fallback is
used, else the token renders to an empty string.

---

## Supported forms

```text
{{user}}                  # plain lookup
{{user|or:guest}}         # fallback when the value is missing/empty
{{item.address.city}}     # dotted path into a nested map
```

### `or:` fallback

```yaml
- run: goto
  params: { url: "https://duckduckgo.com/?q={{q|or:go}}" }
```

If `q` is missing or empty, `go` is used. The fallback also applies *after* the
env-var check — so it's a true last resort.

### Dotted paths

A token may walk into a nested map value:

```yaml
# given a prior step bound `chat = { peerName: "Ann", meta: { id: 7 } }`
text: "{{chat.peerName}} (#{{chat.meta.id}})"
```

Each segment must exist and resolve to a map for the next segment; otherwise the
token renders empty.

---

## Single-token raw resolution

There is one important special case, handled in `renderValue`:

> When a param's **entire** value is exactly one `{{ref}}` token, it resolves to
> the **raw bound value** — a list, map, number, or bool — *not* its
> stringified form.

This is what lets structured data flow between steps:

```yaml
- run: for-each
  params:
    over: "{{chats}}"          # receives the actual list, not "[map[...] ...]"
    as: chat
  do: …

- run: write-files
  params:
    root: "{{out}}"
    files: "{{built.files}}"   # receives the real slice of {path,content} maps

- run: export
  params:
    data: "{{repos}}"          # receives the list of record maps
```

The rule is precise: the value must match `^{{ ref }}$` (optional surrounding
whitespace) **and** the bound value must be non-string and non-nil. A
string-valued binding, or a token embedded in surrounding text
(`"id-{{n}}"`), still goes through normal string substitution.

---

## Tips

- Declare inputs with sensible `default:`s so demo calls work with no body, but
  reference them in the flow as `{{name}}` regardless.
- For secrets, never inline the value — declare it in `secrets.yaml` and use
  `{{SECRET_NAME}}`; the env fallback does the rest.
- To pass a whole list/map to a block action, use the bare single-token form
  (`over: "{{chats}}"`) so it arrives as real data, not a string.
