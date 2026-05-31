# Rendering demos

Five tasks for capturing pages as PDF, PNG, MHTML, and emulated device views.

---

## pdf

Render a page to PDF via Chrome's print engine.

```bash
curl -s -X POST localhost:8765/tasks/rendering/pdf -d '{}'
curl -s -X POST localhost:8765/tasks/rendering/pdf \
  -d '{"url":"https://news.ycombinator.com","path":"/tmp/hn.pdf"}'
```

=== "Recipe (.webtask)"

    ```capy
    task "rendering/pdf"
        pool default
        timeout 20000
        transport rest
        input url  string default "https://example.com"
        input path string default "/tmp/webtasks-demo/example.pdf"

        goto "{{url}}"
        wait until "body" timeout 10000
        pdf doc path "{{path}}" format A4 printBackground true
    end
    ```

Returns both a server-side file at `path` **and** base64 in `data.doc`.

**Concepts:** `pdf` action, `printBackground`, dual output (`path` + `as`).

---

## snapshot

MHTML snapshot — single-file archive of the page with resources inlined.

```bash
curl -s -X POST localhost:8765/tasks/rendering/snapshot -d '{}'
```

**Concepts:** `snapshot` action, archival capture, offline viewing.

---

## fullpage-shot

Full-page screenshot (entire scrollable document, not just viewport).

```bash
curl -s -X POST localhost:8765/tasks/rendering/fullpage-shot -d '{}'
```

Compare with [Basics → screenshot](basics.md#screenshot) (viewport only).

**Concepts:** `screenshot` with full-page mode, large PNG output.

---

## html-to-pdf

Render arbitrary HTML string to PDF without navigating to a URL.

```bash
curl -s -X POST localhost:8765/tasks/rendering/html-to-pdf -d '{}'
```

**Concepts:** `html-to-pdf`, generating reports from templates server-side.

---

## emulate-dark

Emulate dark-mode / device preferences before capture.

```bash
curl -s -X POST localhost:8765/tasks/rendering/emulate-dark -d '{}'
```

**Concepts:** device emulation, `prefers-color-scheme`, pre-capture setup.

---

## Capture comparison

| Action | Output | Best for |
|---|---|---|
| `screenshot` | PNG (base64) | Quick visual check |
| `pdf` | PDF file | Reports, printing |
| `snapshot` | MHTML | Archival, legal hold |
| `html-to-pdf` | PDF from string | Generated documents |

Full params: [Actions → rendering](../actions.md)

---

## What's next?

- [Recording](recording.md) — animated GIF instead of static capture
- [Network → capture](network.md) — HAR alongside rendering
