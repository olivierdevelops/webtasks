# JS modules demos

Three tasks that load reusable JavaScript from `scripts/demo/*.js` instead of
inline `script:` blocks.

---

## Why JS modules?

Inline scripts get unreadable past a few lines. Drop scripts into
`bundle/scripts/` and reference them with `fn`:

```capy
js result fn "demo/page-stats.js" args ["{{url}}"]
```

The server loads `scripts/demo/page-stats.js` from the bundle and executes it
in the browser context.

```mermaid
flowchart LR
    Recipe["recipe<br/>fn 'demo/foo.js'"]
    Bundle["scripts/demo/foo.js"]
    Chrome["Chrome V8"]
    Result["JSON result"]

    Recipe --> Bundle --> Chrome --> Result
```

---

## meta-tags

Extract Open Graph and meta tags from any page.

```bash
curl -s -X POST localhost:8765/tasks/js-modules/meta-tags -d '{}'
curl -s -X POST localhost:8765/tasks/js-modules/meta-tags \
  -d '{"url":"https://github.com/chromedp/chromedp"}'
```

**Script:** `scripts/demo/get-meta-tags.js`

**Concepts:** one task ↔ one JS module, passing the current page DOM to JS.

---

## page-stats

DOM summary helper — counts elements, links, images, etc.

```bash
curl -s -X POST localhost:8765/tasks/js-modules/page-stats -d '{}'
```

**Script:** `scripts/demo/page-stats.js`

=== "Script pattern"

    ```js
    // scripts/demo/page-stats.js
    // arguments[0] = optional override selector root
    return {
      links: document.querySelectorAll("a").length,
      images: document.querySelectorAll("img").length,
      headings: document.querySelectorAll("h1,h2,h3").length,
    };
    ```

**Concepts:** returning structured objects from JS, reusable helpers.

---

## all-links

Collect every link on a page with text and href.

```bash
curl -s -X POST localhost:8765/tasks/js-modules/all-links -d '{}'
curl -s -X POST localhost:8765/tasks/js-modules/all-links -d '{"maxLinks":"50"}'
```

**Script:** `scripts/demo/all-links.js`

**Concepts:** passing `args` from the recipe into JS.

---

## Inline vs module

| Inline JS | `fn` module |
|---|---|
| Good for 1–5 lines | Good for anything longer |
| Lives in the recipe | Lives in `scripts/` |
| Hard to test/format | Normal `.js` file |
| No reuse | Shared across tasks |

Cookbook recipe: [Use a JS module](../cookbook.md)

---

## What's next?

- [Control → await-js](control.md#await-js) — wait until JS condition is true
- [Concio scripts](concio.md) — production JS modules for login, chat navigation
