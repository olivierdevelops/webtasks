# 24. Testing and CI

Keep Capy tasks correct without starting Chrome. Targets **Capy v0.20.0**.

---

## Layer 1 — Library validation

```bash
go install github.com/olivierdevelops/capy/cmd/capy@v0.20.0
capy check capy/webtasks.capy
```

Validates all `function`, `type`, and output block syntax.

---

## Layer 2 — Per-task check

```bash
for f in tasks/**/*.capy; do
  capy check capy/webtasks.capy "$f" || exit 1
done
```

Structured errors with line:column — feed directly to agents or CI logs.

---

## Layer 3 — Golden transpile tests

```bash
capy run capy/webtasks.capy "$f" > "${f%.capy}.golden.yaml"
```

During migration, golden equals the existing hand-written YAML.

CI:

```bash
diff -u expected.golden.yaml <(capy run capy/webtasks.capy task.capy)
```

For `RunMulti` libraries, diff every path in the output map.

---

## Layer 4 — Format gate

```bash
capy fmt tasks/**/*.capy --check
```

---

## Layer 5 — Go integration tests

```go
files, err := tr.Transpile(string(src))
yaml, err := tr.TaskYAML(files, tr.lib)
// yaml.Unmarshal → domain.TaskDef
```

---

## Layer 6 — Smoke tests (Chrome)

Unchanged — `executor call …` against live sites.

---

## Sample GitHub Actions job

```yaml
- run: go install github.com/olivierdevelops/capy/cmd/capy@v0.20.0
- run: capy check capy/webtasks.capy
- run: capy fmt tasks/**/*.capy --check
- run: ./scripts/verify-capy-golden.sh
- run: go test ./internal/infra/capyx/...
```

---

Next: [Troubleshooting →](25-troubleshooting.md)
