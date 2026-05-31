# Basics demos

Five minimal tasks that teach the core building blocks: navigate, wait, extract,
screenshot, and inline JavaScript.

---

## title

The smallest possible task — open example.com and return the page title.

```bash
executor call basics/title
```

=== "Task YAML"

    ```yaml
    name: "basics/title"
    poolTag: "default"
    transports: ["rest"]
    timeoutMs: 15000

    flow:
      - status: "Visiting example.com"
        run: goto
        params: { url: "https://example.com" }

      - status: "Reading title and heading"
        run: extract
        as: page
        params:
          selector: "html"
          repeat: false
          fields:
            title:   { kind: text, selector: "title" }
            heading: { kind: text, selector: "h1" }
            body:    { kind: text, selector: "p" }
    ```

=== "Response"

    ```json
    {
      "ok": true,
      "data": {
        "page": {
          "title": "Example Domain",
          "heading": "Example Domain",
          "body": "This domain is for use in documentation examples …"
        }
      }
    }
    ```

**Concepts:** `goto`, `extract`, `as:` naming, `status:` lines.

---

## screenshot

Capture a viewport PNG of any URL. Returns base64 under `data.png_b64`.

```bash
executor call basics/screenshot
executor call basics/screenshot '{"url":"https://news.ycombinator.com"}'
```

=== "Task YAML"

    ```yaml
    input:
      url: { type: string, required: false, default: "https://example.com" }

    flow:
      - run: goto
        params: { url: "{{url}}" }
      - run: wait-for
        params: { selector: "body", timeoutMs: 10000 }
      - run: screenshot
        as: png_b64
        params: { selector: "." }
    ```

=== "Save to disk"

    ```bash
    curl -s -X POST localhost:8765/tasks/basics/screenshot \
      -H 'Content-Type: application/json' \
      -d '{"url":"https://example.com"}' \
      | python3 -c '
    import json, sys, base64
    b = json.load(sys.stdin)["data"]["png_b64"]
    open("/tmp/shot.png", "wb").write(base64.b64decode(b))
    print("saved /tmp/shot.png")
    '
    ```

**Concepts:** `input` schema, `{{url}}` templating, `screenshot` action.

---

## inline-js

Run arbitrary JavaScript inline with templated arguments.

```bash
executor call basics/inline-js
```

The task uses a `script:` block to read `document.title` and return structured
data without a separate `.js` file.

**Concepts:** `run: js`, inline `script:` blocks, returning values from JS.

See also: [JS modules](js-modules.md) for reusable scripts in `scripts/`.

---

## save-html

Dump the fully rendered HTML to a server-side path.

```bash
executor call basics/save-html
```

**Concepts:** `write-files`, server-side file output, rendered DOM vs source.

---

## wait-then-click

Navigate, wait for an element, click it, then extract the result.

```bash
executor call basics/wait-then-click
```

=== "Flow pattern"

    ```yaml
    flow:
      - run: goto
        params: { url: "…" }
      - run: wait-for
        params: { selector: "button.submit", timeoutMs: 10000 }
      - run: action
        params: { action: click, selector: "button.submit" }
      - run: extract
        as: result
        params: { … }
    ```

**Concepts:** `wait-for`, `action(click)`, chaining steps.

---

## What's next?

- [Crawl demos](crawl.md) — extract lists from real websites
- [Build your own task](../build-your-own-task.md) — author from scratch
