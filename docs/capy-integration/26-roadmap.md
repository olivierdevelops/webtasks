# 26. Roadmap

Phased integration of Capy into webtasks. **Nothing here is implemented yet**
— this guide is the specification.

---

## v0 — Documentation (current)

- [x] Integration guide (this document set)
- [x] Proposed `webtasks.capy` skeleton
- [x] Sample tasks in Capy syntax
- [ ] Upstream link from main docs index

---

## v0.1 — CLI-only adoption

- [ ] Complete `grammar/webtasks.capy` covering all demo tasks
- [ ] `scripts/transpile-capy-tasks.sh` for manual YAML generation
- [ ] Golden files for demo bundle
- [ ] Document in demo README

Authors transpile locally; server still loads YAML.

---

## v0.2 — Go integration

- [ ] `go get github.com/olivierdevelops/capy`
- [ ] `internal/infra/capyx` transpiler
- [ ] `bundle.WalkCapy`
- [ ] Extended `MakeTaskRegistry`
- [ ] Env vars: `WEBTASKS_CAPY_LIB`, `WEBTASKS_CAPY_ENABLE`
- [ ] Integration tests

---

## v0.3 — Demo bundle migration

- [ ] Convert `demo/tasks/**/*.yaml` → `.capy`
- [ ] Keep golden YAML in CI
- [ ] Update MkDocs demo pages with Capy-first examples

---

## v1.0 — Production ready

- [ ] `GET /capy/introspect` (optional)
- [ ] `executor init-task` scaffolds `.capy`
- [ ] Concio bundle port (selected tasks)
- [ ] VS Code grammar / snippets (from introspect JSON)

---

## v1.x — Direct JSON (optional)

- [ ] `file_template` emits JSON matching `TaskDef`
- [ ] Skip YAML hop in registry loader
- [ ] Faster load, stricter schema validation with Go struct tags

---

## Non-goals

- Replacing `pool.yaml`, `secrets.yaml`, or JS modules with Capy
- Executing user Capy as browser logic (transpile only)
- Forking Capy — stay on upstream `olivierdevelops/capy`

---

[← Back to index](index.md)
