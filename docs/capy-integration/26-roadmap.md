# 26. Roadmap

Phased integration of **Capy v0.20.0** into webtasks.

---

## v0 — Documentation (current)

- [x] Integration guide (27 chapters)
- [x] [v0.20 overview](00-v020-overview.md) aligned with upstream
- [x] Proposed `webtasks.capy` skeleton (passes `capy check`)
- [ ] Pin `github.com/olivierdevelops/capy v0.20.0` in `go.mod` (when implementing)

---

## v0.1 — CLI-only adoption

- [ ] Complete `grammar/webtasks.capy` for all demo tasks
- [ ] `capy fmt` + golden YAML CI
- [ ] `capy watch` documented in demo README

---

## v0.2 — Go integration

- [ ] `internal/infra/capyx` with `RunMulti` + `NoOpHost`
- [ ] `bundle.WalkCapy` + extended `MakeTaskRegistry`
- [ ] `WEBTASKS_CAPY_LIB` env var

---

## v0.3 — Demo bundle migration

- [ ] Convert `demo/tasks/**/*.yaml` → `.capy`
- [ ] Update MkDocs demo pages (Capy-first)

---

## v1.0 — Production

- [ ] `GET /capy/introspect` + `/capy/docs`
- [ ] MCP workflow for agents
- [ ] `capy build capy/webtasks.capy -o webtasks-capy` for CI
- [ ] Concio bundle port (selected tasks)

---

## Non-goals

- Replacing `pool.yaml`, `secrets.yaml`, JS modules with Capy
- Executing user Capy at browser runtime
- Forking Capy — track upstream `olivierdevelops/capy`

---

[← Back to index](index.md)
