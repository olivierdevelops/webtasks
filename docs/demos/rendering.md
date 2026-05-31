# Rendering demos

Five tasks for capturing pages as PDF, PNG, MHTML, and emulated device views.

---

## pdf

Render a page to PDF via Chrome's print engine.

```bash
executor call rendering/pdf
executor call rendering/pdf '{"url":"https://news.ycombinator.com","path":"/tmp/hn.pdf"}'
```

=== "Task YAML"

    ```yaml
    input:
      url:  { type: string, default: "https://example.com" }
      path: { type: string, default: "/tmp/webtasks-demo/example.pdf" }

    flow:
      - run: goto
        params: { url: "{{url}}" }
      - run: wait-for
        params: { selector: "body", timeoutMs: 10000 }
      - run: pdf
        as: pdf
        params:
          path: "{{path}}"
          format: "A4"
          printBackground: true
    ```

Returns both a server-side file at `path` **and** base64 in `data.pdf`.

**Concepts:** `pdf` action, `printBackground`, dual output (`path` + `as`).

---

## snapshot

MHTML snapshot — single-file archive of the page with resources inlined.

```bash
executor call rendering/snapshot
```

**Concepts:** `snapshot` action, archival capture, offline viewing.

---

## fullpage-shot

Full-page screenshot (entire scrollable document, not just viewport).

```bash
executor call rendering/fullpage-shot
```

Compare with [Basics → screenshot](basics.md#screenshot) (viewport only).

**Concepts:** `screenshot` with full-page mode, large PNG output.

---

## html-to-pdf

Render arbitrary HTML string to PDF without navigating to a URL.

```bash
executor call rendering/html-to-pdf
```

**Concepts:** `html-to-pdf`, generating reports from templates server-side.

---

## emulate-dark

Emulate dark-mode / device preferences before capture.

```bash
executor call rendering/emulate-dark
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
