# HTML & CSS Conventions

Portable markup and styling rules for small-to-medium web apps. Framework-agnostic:
applies whether you render HTML via templates, a server-side component library
(gomponents, React, JSX), or plain files.

## Philosophy

1. **Write semantic HTML; get styling for free.** Prefer element selectors over classes.
2. **One class per component** (or none); target its expected child elements with nested selectors.
3. **Modifiers are unprefixed, single-purpose classes** (`.muted`, `.outline`), scoped to elements via nesting.
4. **No utility framework, no inline styles, no build step.** Plain CSS + design tokens.

## Classless base elements

Style these directly, no class required:

- `button` — the canonical button.
- `input`, `select`, `textarea` — form fields.
- `a` — base link color/underline.
- `summary` — `cursor: pointer; user-select: none`.
- `ul`, `menu` — list reset (`list-style: none; padding: 0; margin: 0`).
- `section` — default bottom margin for page sections.
- `address` — `font-style: normal` (contact text, not italic).

The `.button` class exists only for **non-`button` elements styled as buttons**
(e.g. `<a class="button" href="/logout">`). A real `<button>` needs no class.

## Semantic component structure

A reusable component gets **one class** and uses semantic child elements, styled
with nested selectors — never `.card-title`, `.card-body`, `.card-footer`, etc.

```html
<ul class="link-list">
  <li>
    <article>
      <header>
        <a href="/…">Title</a>
        <time datetime="2024-01-01">3d ago</time>
      </header>
      <footer>
        <small>example.com</small>
        <ul><li>tag</li></ul>
      </footer>
    </article>
  </li>
</ul>
```

```css
.link-list {
  > li {
    &:hover { background: var(--surface-2); }

    article {
      header {
        display: flex;
        justify-content: space-between;
        gap: var(--size-2);

        a { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; }
        time { color: var(--text-3); }
      }

      footer {
        ul { /* tag list */ li { /* tag badge */ } }
      }
    }
  }
}
```

Element mapping:

| Semantic element | Use for |
|---|---|
| `<article>` | a self-contained item (bookmark, collection, member) |
| `<header>` / `<footer>` | primary / secondary parts of an item |
| `<h1>`–`<h6>` | headings, hierarchical |
| `<a>` | the item's title/link |
| `<small>` | secondary metadata (counts, roles) |
| `<time datetime="…">` | timestamps |
| `<address>` | email / contact |
| `<ul>` / `<li>` | lists of anything (tags, items) |
| `<section>` | page-level groupings |

Use `> li` (direct child) to distinguish a list's own items from nested lists
(e.g. tags inside a link item).

## Markup minimalism

Two classic anti-patterns inflate markup with wrappers and classes that restate
what the element already is.

### No redundant wrapper elements ("divitis")

A wrapper containing exactly one child, existing only as a styling hook, is
redundant. Style the child directly, or target it through its parent.

```html
<!-- bad: the div adds nothing -->
<div class="add-form"><form>…</form></div>

<!-- good: target the form through its parent -->
<details class="add-bookmark">
  <summary>Add</summary>
  <form>…</form>
</details>
```

```css
.add-bookmark form { display: grid; gap: var(--size-2); }
```

### No classes that restate the element ("classitis")

If an element is already uniquely identifiable by its tag and context, don't add
a class that just repeats it. `.form`, `.button` (on a `<button>`), `.link` (on
an `<a>`), `.list` (on a `<ul>`) are redundant — the tag already says it.

```html
<!-- bad -->
<form class="form">…</form>

<!-- good -->
<form>…</form>
```

```css
main form { display: grid; gap: var(--size-2); }
```

### Prefer structural selectors

When structure is stable, structural selectors beat classes: `> *` (direct
children), `:first-child` / `:last-child` / `:only-child` (position),
`:nth-child()` (index), and sibling combinators (`+`, `~`).

Element tags (`form`, `fieldset`, `label`, `summary`, `nav`, `main`, `header`,
`footer`, `article`, `time`, `address`, `small`) are free selectors — use them
before reaching for a class.

## Unprefixed modifiers

Modifiers are single, non-prefixed classes scoped to their element via nesting.
The same class can mean something different on a different element.

```css
button, .button {
  /* base */
  &.outline { /* secondary variant */ }
  &.small   { /* size variant */ }
}

p      { &.muted { color: var(--text-3); } }
button { &.muted { opacity: .6; } }
```

Anti-patterns:

- `.text-muted`, `.button-outline`, `.card-title` → `.muted`, `.outline`, and element nesting.
- `.dropdown-button`, `.dropdown-header`, `.dropdown-item` → `<details class="dropdown">` with `<summary>`, `<h2>`, `<ul>`.

## Design tokens

- Load a token library (e.g. [Open Props](https://open-props.style)) for primitives:
  `--size-*`, `--font-size-*`, `--radius-*`, `--gray-*`, `--font-*`, `--shadow-*`.
- Define semantic app tokens in `:root`, mapped to primitives via `light-dark()`.

```css
:root {
  color-scheme: light dark;
  --primary: light-dark(var(--indigo-7), var(--indigo-5));
  --primary-foreground: light-dark(#fff, var(--gray-12));
  --background: light-dark(var(--gray-0), var(--gray-12));
  --surface-2: light-dark(var(--gray-1), var(--gray-11));
  --text-1: light-dark(var(--gray-12), var(--gray-1));
  --text-2: light-dark(var(--gray-9), var(--gray-5));
  --text-3: light-dark(var(--gray-6), var(--gray-5));
  --border: light-dark(var(--gray-3), var(--gray-8));
}
```

## Modern CSS — prefer these

- **CSS nesting** for all descendant, sibling, and compound selectors.
- **Logical properties** (`margin-inline`, `padding-inline`, `inset`) over physical.
- **`gap`** on flex/grid, never margin hacks.
- **`text-wrap: balance`** (headings) / **`pretty`** (prose).
- **`light-dark()`** + **`color-scheme`** for automatic dark mode.
- **`:focus-visible`** for keyboard-only focus rings.
- **`prefers-reduced-motion`** to disable transitions/animations.
- **`field-sizing: content`** for auto-growing textareas.
- **`color-mix()`** for derived hover/active colors (instead of `opacity`).
- **`accent-color`** so native form controls match the theme.
- **`100svh`** viewport units (with `100vh` fallback).
- **`:is()` / `:where()`** for selector grouping; `:where()` for zero-specificity.

## Rules

- No inline styles.
- No utility framework (Tailwind, etc.); no build step.
- One class per component; nest everything else.
- Modifiers are unprefixed and element-scoped.
- Keep specificity low.

## Anti-patterns

| Instead of | Do |
|---|---|
| `.dropdown-button`, `.dropdown-item`, … | `<details>` + `<summary>` + nested element selectors |
| `.text-muted`, `.button-outline` | `.muted`, `.outline` |
| `<div>`-soup, a class on every node | semantic elements + nested selectors |
| `margin-left` / `margin-right` | `margin-inline` |
| `min-height: 100vh` | `min-height: 100svh` |
| `opacity: .9` on hover | `color-mix()` |
