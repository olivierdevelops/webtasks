# Control flow demos

Five tasks demonstrating task composition, loops, JS conditions, and nested
recording.

---

## call

Run another task's flow inside the current window, then use its results.

```bash
executor call control/call
```

=== "Task YAML"

    ```yaml
    name: "control/call"
    flow:
      - run: call
        params: { task: "basics/title" }
      - run: return
        params: { value: "{{page.title}}" }
    ```

```mermaid
flowchart TD
    A["control/call starts"]
    B["call basics/title\n(reuses same window)"]
    C["page.title available\nin template context"]
    D["return title string"]

    A --> B --> C --> D
```

**Concepts:** `call` action, task reuse, template context from called task.

Build libraries of small tasks and compose them — the Concio bundle uses
this heavily (`setup` → `list-chats` → `get-messages`).

---

## loop

Repeat steps while a condition holds — iterate over pages, retry on failure.

```bash
executor call control/loop
```

=== "Pattern"

    ```yaml
    - run: loop
      params:
        maxIterations: 10
      while:
        run: js
        params:
          script: "return document.querySelector('.next-page') !== null"
      do:
        - run: extract
          as: page_items
          params: { … }
        - run: action
          params: { action: click, selector: ".next-page" }
    ```

**Concepts:** `loop` with `while:` condition, `do:` body, pagination.

See also [Crawl → quotes-paginated](crawl.md#quotes-paginated).

---

## loop-fn

Same as `loop` but the condition comes from a JS module.

```bash
executor call control/loop-fn
```

**Concepts:** `fn:` in loop conditions, cleaner complex logic.

---

## await-js

Block until a JavaScript expression returns true.

```bash
executor call control/await-js
```

=== "Use case"

    ```yaml
    - run: await-js
      params:
        script: "return document.querySelector('.loaded') !== null"
        timeoutMs: 30000
        pollMs: 500
    ```

**Concepts:** waiting for JS-driven UI state, SPAs, lazy widgets.

---

## record-step

Record a GIF of a single step (not the whole flow).

```bash
executor call control/record-step
```

**Concepts:** combining [Recording](recording.md) with granular step control.

---

## Control flow cheat sheet

| Action | Purpose |
|---|---|
| `call` | Run another task inline |
| `loop` | Repeat while condition true |
| `await-js` | Poll until JS returns true |
| `set` | Bind a variable |
| `return` | Early exit with value |
| `for-each` | Iterate over an array |

Full reference: [Actions](../actions.md)

---

## What's next?

- [Recording](recording.md) — visual capture of flows
- [Concio](concio.md) — production task chains
