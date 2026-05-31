# Basics demos

Five minimal tasks that teach the core building blocks: navigate, wait, extract,
screenshot, and inline JavaScript.

---

## title

The smallest possible task — open example.com and return the page title.

```bash
curl -s -X POST localhost:8765/tasks/basics/title -d '{}'
```

=== "Recipe (.webtask)"

    ```capy
    task "basics/title"
        pool default
        timeout 15000
        transport rest

        status "Visiting example.com"
        goto "https://example.com"

        status "Reading title and heading"
        extract page from "html"
            title   text "title"
            heading text "h1"
            body    text "p"
        end
    end
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

**Concepts:** `goto`, `extract`, named results, `status` lines.

---

## screenshot

Capture a viewport PNG of any URL. Returns base64 under `data.png_b64`.

```bash
curl -s -X POST localhost:8765/tasks/basics/screenshot -d '{}'
curl -s -X POST localhost:8765/tasks/basics/screenshot -d '{"url":"https://news.ycombinator.com"}'
```

=== "Recipe (.webtask)"

    ```capy
    task "basics/screenshot"
        pool default
        timeout 15000
        transport rest
        input url string default "https://example.com"

        goto "{{url}}"
        wait until "body" timeout 10000
        screenshot png_b64 selector "."
    end
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
curl -s -X POST localhost:8765/tasks/basics/inline-js -d '{}'
```

The task uses an inline `js` step to read `document.title` and return structured
data without a separate `.js` file.

**Concepts:** `js`, inline scripts, returning values from JS.

See also: [JS modules](js-modules.md) for reusable scripts in `scripts/`.

---

## save-html

Dump the fully rendered HTML to a server-side path.

```bash
curl -s -X POST localhost:8765/tasks/basics/save-html -d '{}'
```

**Concepts:** writing files, server-side output, rendered DOM vs source.

---

## wait-then-click

Navigate, wait for an element, click it, then extract the result.

```bash
curl -s -X POST localhost:8765/tasks/basics/wait-then-click -d '{}'
```

=== "Flow pattern"

    ```capy
    goto "https://example.com"
    wait until "button.submit" timeout 10000
    click "button.submit"
    extract result from "html"
        message text ".result"
    end
    ```

**Concepts:** `wait until`, `click`, chaining steps.

---

## What's next?

- [Crawl demos](crawl.md) — extract lists from real websites
- [Writing tasks](../writing-tasks.md) — author from scratch
