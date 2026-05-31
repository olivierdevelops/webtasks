# 24. Testing and CI

Keep Capy tasks correct without starting Chrome.

---

## Layer 1 — Library validation

```bash
capy check capy/webtasks.capy
```

Catches:

- Unknown types
- Invalid `arg` ordering (optional before required)
- Broken inner-DSL syntax

Run on every PR touching `capy/webtasks.capy`.

---

## Layer 2 — Golden transpile tests

For each `tasks/**/*.capy`:

```bash
capy run capy/webtasks.capy "$f" > "${f%.capy}.golden.yaml"
```

Commit golden files. CI:

```bash
diff -u expected.golden.yaml <(capy run capy/webtasks.capy task.capy)
```

During YAML→Capy migration, golden can be the **existing YAML file** until
outputs stabilize.

---

## Layer 3 — Go integration tests

```go
func TestCapyTasksLoad(t *testing.T) {
    b := openBundle(t, "testdata/capy-bundle")
    reg := makeRegistry(t, b)
    d, ok := reg.Get("basics/title")
    require.True(t, ok)
    require.Len(t, d.Flow, 2)
}
```

---

## Layer 4 — Smoke tests (Chrome)

Unchanged — run transpiled tasks against live sites:

```bash
executor call crawl/hackernews-top
```

Separate **author-time** (Capy) from **runtime** (browser) tests.

---

## Sample CI workflow

```yaml
name: capy-tasks
on: [push, pull_request]
jobs:
  transpile:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go install github.com/olivierdevelops/capy/cmd/capy@latest
      - run: capy check capy/webtasks.capy
      - run: ./scripts/verify-capy-golden.sh
  integration:
    needs: transpile
    runs-on: ubuntu-latest
    steps:
      - run: go test ./internal/infra/capyx/...
```

---

Next: [Troubleshooting →](25-troubleshooting.md)
