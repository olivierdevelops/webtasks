# 25. Troubleshooting

---

## Install / module path

Use the tagged release when it resolves:

```bash
go install github.com/olivierdevelops/capy/cmd/capy@v0.20.0
go get github.com/olivierdevelops/capy@v0.20.0
```

If `@v0.20.0` fails with a module-path mismatch (`luowensheng/capy` vs
`olivierdevelops/capy`), use `@main` until the tag is republished:

```bash
go install github.com/olivierdevelops/capy/cmd/capy@main
```

---

## `capy check` fails on webtasks.capy

| Error | Fix |
|---|---|
| `unknown type "Foo"` | Add `type Foo … end` or fix typo |
| `optional arg must be trailing` | Move `default` args to end of arg list |
| `expected newline after if cond` | Put `if` body on following lines; use `else`/`end` |
| `mixed group_open and pattern` | Pick one constraint style per type |

See [migration-guide.md](https://github.com/olivierdevelops/capy/blob/main/docs/migration-guide.md).

---

## `capy run` fails on task file

| Error | Fix |
|---|---|
| `no function matched` | Typo in verb; run `capy docs webtasks.capy` |
| `not in options for type "PoolTag"` | Extend type or fix pool name |
| `unexpected character '#'` | Add `comments line "#" end` to library |

---

## Transpiled YAML won't load

```bash
capy run capy/webtasks.capy task.capy | python3 -c 'import sys,yaml; yaml.safe_load(sys.stdin)'
```

Check `flow` structure — fields must be maps, not arrays (known v0.1 grammar bug).

---

## capylang (PyPI) ≠ Capy

The PyPI package `capylang` (Anistick math utilities) is unrelated. Use
**[github.com/olivierdevelops/capy](https://github.com/olivierdevelops/capy)** only.

---

## Local checkout names

A local folder named `capylang-claude` may be a Capy checkout — the canonical
remote is `olivierdevelops/capy` on GitHub.

---

Next: [Roadmap →](26-roadmap.md)
