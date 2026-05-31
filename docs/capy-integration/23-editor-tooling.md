# 23. Editor tooling

Leverage Capy's introspection and playground for webtasks task authoring.

---

## capy docs — auto-generated reference

```bash
capy docs capy/webtasks.capy > docs/capy-integration/DSL_REFERENCE.generated.md
```

Regenerate when `webtasks.capy` changes — never hand-maintain parallel docs.

Output includes every `function`, `arg`, `description`, and `type` block.

---

## Introspect API

From Go ([embedding.md](https://github.com/olivierdevelops/capy/blob/main/docs/embedding.md)):

```go
for _, fn := range lib.Introspect() {
    // fn.Name, fn.Args, fn.Description, fn.Block
}
```

Build:

- VS Code completion for `goto`, `extract`, …
- Hover docs for `input` fields
- Diagnostics for unknown tokens before save

Optional webtasks endpoint: `GET /capy/introspect`.

---

## Playground

Capy's WASM playground:
[olivierdevelops.github.io/capy/playground/](https://olivierdevelops.github.io/capy/playground/)

Load `webtasks.capy` + a task script in the browser editor; see transpiled YAML
live. Useful for workshops and docs.

---

## CLI watch mode (author workflow)

```bash
while true; do
  capy run capy/webtasks.capy tasks/basics/title.capy | tee /tmp/out.yaml
  sleep 1
done
# edit title.capy in another pane
```

After Go integration, `executor server` hot-reloads transpiled tasks same as YAML.

---

## Diff-friendly reviews

Capy sources are shorter — PRs show intent:

```diff
     goto "https://news.ycombinator.com"
-    wait-for "tr.athing" timeout 5s
+    wait-for "tr.athing" timeout 15s
```

YAML diffs bury changes in `params:` nesting.

---

Next: [Testing and CI →](24-testing-ci.md)
