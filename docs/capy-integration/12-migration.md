# 12. Migration strategy

How to adopt Capy in existing bundles without breaking production.

---

## Phase 0 — Documentation only (today)

- Read this guide
- Try CLI transpile on demo tasks
- No server changes required

```bash
go install github.com/olivierdevelops/capy/cmd/capy@latest
capy run docs/capy-integration/grammar/webtasks.capy \
       docs/capy-integration/grammar/samples/01-hello.capy
```

---

## Phase 1 — Dual format bundles

1. Add `capy/webtasks.capy` to bundle
2. Convert one task at a time: keep `.yaml`, add `.capy`, diff outputs
3. When identical, delete `.yaml`
4. Enable Go integration behind `WEBTASKS_CAPY_ENABLE=true`

Rollback: remove `.capy` files; YAML still works.

---

## Phase 2 — CI gate

```yaml
# .github/workflows/capy.yml
jobs:
  capy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go install github.com/olivierdevelops/capy/cmd/capy@latest
      - run: capy check capy/webtasks.capy
      - run: ./scripts/verify-capy-golden.sh
```

Golden script transpiles every `tasks/**/*.capy` and compares to
`*.golden.yaml` (or to existing YAML during migration).

---

## Phase 3 — Authoring defaults

- `executor init-task` scaffolds `.capy` not `.yaml`
- Demo bundle converts to Capy
- Docs site shows Capy first, YAML as "legacy view"

---

## Mechanical conversion checklist

For each YAML task:

- [ ] `name:` → `task "name" … end`
- [ ] `poolTag:` → `pool …`
- [ ] `timeoutMs:` → `timeout …`
- [ ] `transports:` → `transport …` lines
- [ ] `input:` → `input …` lines
- [ ] Each flow step → DSL statement
- [ ] `extract.fields` → `extract … : field lines end`
- [ ] Block `do:` → block action with nested steps
- [ ] Run `capy run` and diff YAML

---

## Team rollout

| Role | Action |
|---|---|
| Task authors | Learn DSL via [samples/](samples/01-hello.md) |
| Platform | Ship `capyx`, env vars, CI |
| AI agents | Point at [Capy for LLMs](https://github.com/olivierdevelops/capy/blob/main/docs/CAPY_FOR_LLMS.md) + `webtasks.capy` |
| Ops | No change — same HTTP API, same Chrome pools |

---

Next: [Sample 1 — hello →](samples/01-hello.md)
