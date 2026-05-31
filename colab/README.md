# Colab bundle — drive Google Colab as a GPU provider

This bundle turns the webtasks engine into a remote-code runner: it opens a
Google Colab notebook in a real browser, runs Python in a cell, and streams
the output back live over SSE.

The Go binary stays generic — it knows nothing about Colab. Everything
Colab-specific is YAML tasks + JS modules in this bundle, composed from
generic engine actions (`js`, `loop`, `emit-event`, `return`).

## One-time setup: log in

Colab needs a Google session. Automated Google login trips bot defences, so
this bundle uses a **persistent Chrome profile** (`tasks/pool.yaml` →
`persistent: true`). Log in once, manually:

1. Start the server **headful** so you can see the browser:
   ```
   WEBTASKS_HEADLESS=false WEBTASKS_BUNDLE=$(pwd)/colab ./build/webtasks
   ```
2. Call `colab/open` — a Chrome window opens on Colab. Sign into Google in
   that window.
3. The session is saved under `~/.webtasks/profiles/profile-colab` (override
   with `WEBTASKS_PROFILE_DIR`) and survives restarts.

After that, the server can run headless.

## Run code

```
curl -N -X POST http://localhost:8080/tasks/colab/run-code \
  -H 'Accept: text/event-stream' \
  -H 'Content-Type: application/json' \
  -d '{"code": "import torch; print(torch.cuda.get_device_name(0))",
       "url": "https://colab.research.google.com/drive/<NOTEBOOK_ID>"}'
```

You receive an `output` SSE event per poll while the cell runs, then a
`done` event with the final output.

## Tasks

| Task | Purpose |
|---|---|
| `colab/open` | Open a notebook, report whether the session is authenticated. |
| `colab/run-code` | Place `code` in the active cell, run it, stream output until done. |

## Caveats

Colab's DOM is undocumented and changes frequently. The selectors in
`scripts/colab/*.js` are best-effort — if a cell never appears to finish, or
code isn't placed, adjust `cell-done.js` / `set-active-cell.js`. Keep the
`colab` pool at size 1.
