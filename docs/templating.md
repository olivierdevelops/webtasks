# Templating

Every string parameter in a recipe (and every `status` line) is run through the
templating engine before the action sees it. The syntax is `{{name}}`.

---

## What gets templated

Templating walks each parameter recursively:

- **strings** are substituted,
- **list items** are each substituted,
- **map values** are each substituted,
- other types (numbers, bools) pass through unchanged.

So `headers { Authorization: "Bearer {{TOKEN}}" }` and `args ["{{a}}", "{{b}}"]`
both work.

---

## Lookup order

For a token `{{key}}`:

1. **The task's bindings** — declared inputs, plus every prior step's result,
   plus any iteration variables (`for-each`'s `{{item}}`/`{{item_index}}`,
   `loop`'s `{{loop_index}}`).
2. **The process environment** — this is how resolved secrets reach templating:
   the secrets loader exports each one as an env var, so `{{CONCIO_PASSWORD}}`
   resolves even though it was never an input. → [Secrets](deploy.md#secrets)

If neither resolves, the `or:` fallback is used, else the token renders to an
empty string.

---

## Supported forms

```text
{{user}}                  # plain lookup
{{user|or:guest}}         # fallback when the value is missing/empty
{{item.address.city}}     # dotted path into a nested map
```

### `or:` fallback

```capy
goto "https://duckduckgo.com/?q={{q|or:go}}"
```

If `q` is missing or empty, `go` is used. The fallback applies *after* the
env-var check — a true last resort.

### Dotted paths

A token may walk into a nested map value:

```capy
# given a prior step bound chat = { peerName: "Ann", meta: { id: 7 } }
emit status "{{chat.peerName}} (#{{chat.meta.id}})"
```

Each segment must exist and resolve to a map for the next; otherwise the token
renders empty.

---

## Single-token raw resolution

There is one important special case:

> When a param's **entire** value is exactly one `{{ref}}` token, it resolves to
> the **raw bound value** — a list, map, number, or bool — *not* its stringified
> form.

This is what lets structured data flow between steps:

```capy
for-each chat in "{{chats}}"          # receives the actual list
    # …
end

write-files written root "{{out}}" files "{{built.files}}"   # the real slice

export csv path "/tmp/out.csv" data "{{repos}}"              # the list of records
```

The rule is precise: the value must match `^{{ ref }}$` (optional surrounding
whitespace) **and** the bound value must be non-string and non-nil. A
string-valued binding, or a token embedded in surrounding text (`"id-{{n}}"`),
still goes through normal string substitution.

---

## Tips

- Declare inputs with sensible defaults so demo calls work with no body, but
  reference them in the recipe as `{{name}}` regardless.
- For secrets, never inline the value — declare it in the bundle's secrets
  config and use `{{SECRET_NAME}}`; the env fallback does the rest.
- To pass a whole list/map to a block action, use the bare single-token form
  (`"{{chats}}"`) so it arrives as real data, not a string.
