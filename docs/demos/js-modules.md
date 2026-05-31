# JS modules demos

Three tasks that load reusable JavaScript from `scripts/demo/*.js` instead of
inline `script:` blocks.

---

## Why JS modules?

Inline `script:` blocks get unreadable past a few lines. Drop scripts into
`bundle/scripts/` and reference them with `fn:`:

```yaml
- run: js
  as: result
  params:
    fn: "demo/page-stats.js"
    args: ["{{url}}"]
```

The server loads `scripts/demo/page-stats.js` from the bundle and executes it
in the browser context.

```mermaid
flowchart LR
    YAML["Task YAML\nfn: demo/foo.js"]
    Bundle["scripts/demo/foo.js"]
    Chrome["Chrome V8"]
    Result["JSON result"]

    YAML --> Bundle --> Chrome --> Result
```

---

## meta-tags

Extract Open Graph and meta tags from any page.

```bash
executor call js-modules/meta-tags
executor call js-modules/meta-tags '{"url":"https://github.com/chromedp/chromedp"}'
```

**Script:** `demo/scripts/demo/get-meta-tags.js`

**Concepts:** one task ↔ one JS module, passing the current page DOM to JS.

---

## page-stats

DOM summary helper — counts elements, links, images, etc.

```bash
executor call js-modules/page-stats
```

**Script:** `demo/scripts/demo/page-stats.js`

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
executor call js-modules/all-links
executor call js-modules/all-links '{"maxLinks":"50"}'
```

**Script:** `demo/scripts/demo/all-links.js`

**Concepts:** passing args from YAML into JS via `args:` array.

---

## Inline vs module

| Inline `script:` | `fn:` module |
|---|---|
| Good for 1–5 lines | Good for anything longer |
| Lives in the YAML | Lives in `scripts/` |
| Hard to test/format | Normal `.js` file |
| No reuse | Shared across tasks |

Cookbook recipe: [§4 Use a JS module](../cookbook.md#4-use-a-js-module-instead-of-inline-script)

---

## What's next?

- [Control → await-js](control.md#await-js) — wait until JS condition is true
- [Concio scripts](concio.md) — production JS modules for login, chat navigation
