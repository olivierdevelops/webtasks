# 25. Troubleshooting

---

## `capy check` fails on webtasks.capy

| Error | Fix |
|---|---|
| `unknown type "Foo"` | Add `type Foo … end` or fix typo in `arg capture` |
| `optional arg must be trailing` | Move `default` args to end of arg list |
| `mixed group_open and pattern` | Type uses both — pick one constraint style |

---

## `capy run` fails on task file

| Error | Fix |
|---|---|
| `no function matched` | Typo in verb; check `capy docs` for valid shapes |
| `not in options for type "PoolTag"` | Use declared pool or extend type |
| `unexpected character '#'` | Add `comments line "#" end` to library |
| Indentation errors | Use 4 spaces per level (Capy lexer rule) |

---

## Transpiled YAML won't load in webtasks

| Symptom | Fix |
|---|---|
| `missing required input` | Check `input` block emitted correctly |
| `unknown action` | DSL verb maps to wrong `run:` — fix library function |
| `yaml: unmarshal errors` | Inspect emit — numbers must not be quoted strings |

Debug:

```bash
capy run capy/webtasks.capy task.capy | python3 -c 'import sys,yaml; yaml.safe_load(sys.stdin)'
```

---

## Capy vs capylang (PyPI)

The PyPI package `capylang` (Anistick math utilities) is **unrelated**.
webtasks integration uses **[github.com/olivierdevelops/capy](https://github.com/olivierdevelops/capy)** only.

---

## Local path `capylang-claude`

If your checkout lives at `capylang-claude/`, it is the same project as
`olivierdevelops/capy` — use the GitHub repo for installs and docs:

```bash
go install github.com/olivierdevelops/capy/cmd/capy@latest
```

---

Next: [Roadmap →](26-roadmap.md)
