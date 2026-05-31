# Secrets

Tasks reference sensitive values (passwords, API keys) via templating
(`{{CONCIO_PASSWORD}}`) — but the values must never live in the task YAML. The
bundle declares *what* secrets the server needs in `secrets.yaml`; the server
resolves them at startup from the environment, CLI args, or an interactive
prompt, and publishes them as process env vars so templating can find them.

Implemented in [secrets.go](../internal/orchestrator/secrets.go); resolved
values feed the templating env fallback. → [templating.md](templating.md)

---

## Declaring secrets

`<bundle>/secrets.yaml`:

```yaml
secrets:
  - name: CONCIO_PASSWORD
    description: "Concio account password"
    required: true
    sensitive: true               # silent input when prompting
    sources: ["env", "arg", "prompt"]
  - name: API_KEY
    required: false
    default: ""
    sources: ["env", "arg"]
```

| Field | Default | Notes |
|---|---|---|
| `name` | — | The env-var name the value is published under, and the `{{name}}` token tasks use. |
| `description` | — | Shown in the interactive prompt. |
| `required` | `false` | If `true` and unresolved, the server **refuses to start** with `required secret "…" not found`. |
| `sensitive` | `false` | When prompting, read silently (no echo). Also hides the value in the startup audit log. |
| `default` | `""` | Fallback value if no source yields one (recorded as source `default`). |
| `sources` | `["env","arg","prompt"]` | Resolution order (below). |

The file is optional — no `secrets.yaml` means no declared secrets.

---

## Resolution chain

For each declared secret, the loader walks `sources` **in order** and takes the
first that yields a non-empty value:

| Source | How |
|---|---|
| `env` | `os.Getenv(name)` — e.g. `CONCIO_PASSWORD=… ./webtasks`. |
| `arg` | A `--name=value` launcher flag — e.g. `./webtasks --CONCIO_PASSWORD=…`. Parsed from the args passed to `Run`. |
| `prompt` | Interactive TTY prompt. Silent when `sensitive: true`. **Skipped entirely when there's no controlling terminal** (e.g. CI), so a missing required secret produces the clean "not found" error rather than a stdin EOF. |

If none yield a value, the `default` is used; if there's still nothing and the
secret is `required`, startup fails.

Resolved values are exported with `os.Setenv(name, value)`, which is precisely
what the templating layer's env fallback reads. At startup the server prints an
audit line per resolved secret (`- NAME (from env)`, value hidden if sensitive).

---

## Using a secret in a task

Once declared and resolved, reference it like any binding — the env fallback
does the work:

```yaml
- run: sendkeys
  params: { selector: "#password", keys: "{{CONCIO_PASSWORD}}" }
```

No `input:` entry is needed; the value comes from the process env, not the
request body. → [templating.md](templating.md)

---

## Recommended workflow: `sm exec --`

Don't keep secrets in shell history or files. Keep them in a secret manager
(`sm`) and inject them at launch so `secrets.yaml`'s `env` source resolves them:

```bash
sm exec -- ./build/webtasks
```

The `executor server` / `executor concio-server` commands already wrap the
binary this way. The `concio-test` command instead expects credentials already
in the environment, so wrap *it* (`sm exec -- executor concio-test`) if your
vault holds them. → [cli.md](cli.md)

---

## CI / non-interactive notes

- `prompt` is silently skipped without a TTY, so supply secrets via `env` or
  `arg` (both precede `prompt` in the default order).
- For optional secrets, give a `default:` so a missing value doesn't block.
- `WEBTASKS_HEADLESS=true` affects Chrome, **not** prompting — the two are
  independent.
