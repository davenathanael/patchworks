# CSS & HTML Guidelines

Distilled reference for writing modern, tiny, maintainable styling. Combines three
sources into one working guideline: [SmolCSS](https://smolcss.dev/) (minimal layout
recipes), [ModernCSS](https://moderncss.dev/) (architecture, theming, container
queries), and [modern.css](https://modern-css.com/) (old-hack-to-native-replacement
reference). Neutral language — usable for humans and agents, in this project or any
other.

## Principles

- **Native CSS first.** Every old hack has a native replacement; prefer it over JS,
  preprocessor logic, or extra markup. No floats, no `transform` centering, no margin
  hacks, no `!important` specificity wars.
- **Semantic HTML does the heavy lifting.** Style base elements (`button`, `input`,
  `article`, `details`…), not a class on every node. One class per component, target
  children with nesting.
- **Progressive enhancement.** Write a base that works without the newest features,
  then layer modern CSS on top. If a feature fails, the task still works.
- **No build step.** Everything must run in current evergreen browsers natively
  (native nesting, `@layer`, `@property`, `light-dark()` all qualify).
- **Tiny + maintainable.** Cascade layers order the cascade; design tokens
  centralize values; components expose custom-property APIs.

## Architecture

### Cascade layers

Organize every stylesheet into ordered layers. Order = priority (later wins).
Un-layered styles always beat layered ones, so layer everything (except the
third-party token import, which stays top-level).

```css
@layer reset, theme, base, layout, components, utilities, states;
```

| Layer | Contents |
|---|---|
| `reset` | box-sizing, margin reset, scroll-padding |
| `theme` | `:root` design tokens, `color-scheme`, `accent-color` |
| `base` | element defaults: `body`, headings, `a`, form controls, `:focus-visible` |
| `layout` | page/section structure: containers, nav, grid/flex utilities |
| `components` | `.card`, `.button`, `.link-list`, etc. |
| `utilities` | single-purpose helpers (`.muted`, spacing) |
| `states` | `:hover`/`:focus` overrides, `prefers-reduced-motion` |

Use `@layer` instead of `!important` to control override order.

### Design tokens (custom properties)

Primitives come from a token library (Open Props: `--size-*`, `--gray-*`,
`--font-size-*`, `--radius-*`). Map them to semantic app tokens in `:root`, using
`light-dark()` so dark mode is one declaration per token.

```css
:root {
  color-scheme: light dark;
  accent-color: var(--primary);
  --primary: light-dark(var(--indigo-7), var(--indigo-5));
  --background: light-dark(var(--gray-0), var(--gray-12));
  --text-1: light-dark(var(--gray-12), var(--gray-1));
  --border: light-dark(var(--gray-3), var(--gray-8));
}
```

Prefer `oklch()` for brand colors (perceptually uniform); derive variants with
relative color syntax instead of hardcoding.

```css
--brand: oklch(0.55 0.2 264);
--brand-hover: oklch(from var(--brand) calc(l - 0.08) c h);
```

### Reset additions

```css
*, *::before, *::after { box-sizing: border-box; }
* { margin: 0; }

/* Relative, legible underlines on all links */
a {
  text-decoration-thickness: max(0.08em, 1px);
  text-underline-offset: 0.15em;
}

/* Focus ring as a component-overridable custom-property API */
:focus-visible {
  --outline-size: max(2px, 0.15em);
  outline: var(--outline-width, var(--outline-size))
           var(--outline-style, solid)
           var(--outline-color, var(--primary));
  outline-offset: var(--outline-offset, var(--outline-size));
}

/* Anchor/focus scroll offset */
:target { scroll-margin-top: 2rem; }
:focus   { scroll-padding-block-end: 8vh; }
```

### Native nesting

Use CSS nesting for all descendant/compound selectors. `&` appends to the parent;
without `&` the nested selector becomes a descendant.

```css
.card {
  & > img { border-radius: var(--radius) var(--radius) 0 0; }
  :is(h2, h3) { font-size: clamp(1.25rem, 5cqi, 1.5rem); }
  &:hover { box-shadow: var(--shadow-3); }
}
```

## Layout

### Intrinsic container

Cap content width and add a responsive gutter in one line — no media queries.

```css
.container {
  width: min(100% - 3rem, var(--container-max, 60ch));
  margin-inline: auto;
}
```

### Responsive grid / flexbox grid

Grid: equal columns that wrap; `min()` prevents overflow in narrow spaces.

```css
.grid {
  --min: 30ch; --gap: 1rem;
  display: grid;
  gap: var(--gap);
  grid-template-columns: repeat(auto-fit, minmax(min(100%, var(--min)), 1fr));
}
```

Flexbox: same idea, but the odd/last item grows to fill the row.

```css
.flex-grid {
  --min: 20rem; --gap: 1rem;
  display: flex; flex-wrap: wrap; gap: var(--gap);
}
.flex-grid > * { flex: 1 1 var(--min); }
```

### Centering

```css
.parent { display: grid; place-items: center; }   /* or place-content */
.parent { display: flex; justify-content: center; align-items: center; }
.child  { margin: auto; }                          /* inside a grid parent */
```

### Spacing, ratios, positioning

- `gap` on flex/grid — never margin hacks on children.
- `aspect-ratio: 16 / 9` — never the padding-top hack.
- `inset: 0` — never four `top/right/bottom/left` lines.
- `position: sticky; top: 0` — never JS scroll listeners.
- Logical properties: `margin-inline`, `padding-block` — never `left`/`right`.
- `min-height: 100svh` (not `100vh`) for mobile viewports.
- `object-fit: cover` — never `background-image` hacks.

### Container queries

Components respond to their own space, not the viewport. Style containers respond to
a `container-type`; query with the range syntax.

```css
.card { container-type: inline-size; }

@container (inline-size > 35ch) {
  .card { grid-auto-flow: column; gap: 5cqi; }
}
```

Container units (`cqi`, `cqw`) enable fluid type/sizing inside a component:
`font-size: clamp(1.25rem, 5cqi, 1.5rem)`.

## Components

### Classless base elements

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

### One class per component

One class on the root; semantic children styled with nesting — never
`.card-title`, `.card-body`, `.card-footer`, etc.

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
.link-list > li { /* item */ }
.link-list article header { /* title row */ }
.link-list article footer ul li { /* tag badge */ }
```

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

### Markup minimalism

- **No redundant wrappers ("divitis").** A wrapper with exactly one child, existing
  only as a styling hook, is redundant. Style the child directly, or target it
  through its parent: `<details class="add-bookmark"><summary>Add</summary><form>…</form></details>`,
  styled via `.add-bookmark form`.
- **No classes that restate the element ("classitis").** `.form`, `.button` (on a
  `<button>`), `.link` (on an `<a>`), `.list` (on a `<ul>`) are redundant.
- **Prefer structural selectors** when structure is stable: `> *`, `:first-child` /
  `:last-child` / `:only-child`, `:nth-child()`, and sibling combinators (`+`, `~`).
  Element tags are free selectors — use them before reaching for a class.

### Unprefixed modifiers

Modifiers are single, non-prefixed classes scoped to their element via nesting.
The same class can mean something different on a different element.

```css
button, .button {
  &.outline { /* secondary variant */ }
  &.small   { /* size variant */ }
}
p      { &.muted { color: var(--text-3); } }
button { &.muted { opacity: .6; } }
```

Anti-patterns: `.text-muted` → `.muted`; `.button-outline` → `.outline`;
`.card-title` → element nesting; `.dropdown-button` / `.dropdown-item` →
`<details class="dropdown">` + `<summary>` + nested element selectors.

### Component custom-property API

Expose variants via custom properties with fallbacks; states override the property,
not the declaration.

```css
.button {
  color: var(--button-color, var(--primary));
  background: var(--button-bg, var(--accent));
}
.button:hover { --button-bg: var(--accent-hover); }
.button:focus-visible { --outline-offset: -0.35em; }  /* inset ring */
```

### `:is()`, `:where()`, `:has()`

- `:is(h1, h2, h3)` — group selectors without repetition.
- `:where(ul, ol)` — apply a reset at zero specificity (easy to override).
- `:has()` — style a parent based on its contents (no JS):
  - `.form-group:has(input:invalid) { border-color: red; }`
  - `.button:where(:has(.icon)) { display: flex; gap: 0.5em; }`
  - wrap the `:has()` clause in `:where()` to keep base specificity.

### Focus & form states

- `:focus-visible` — ring only for keyboard users, not mouse clicks.
- `:focus-within` — outline a card when any child link is focused:
  `.card:focus-within { outline: 2px solid var(--primary); outline-offset: -4px; }`
- `accent-color` — theme checkboxes/radios/range without rebuilding them.
- `:user-valid` / `:user-invalid` — validate after interaction, no JS `.touched`.

## Typography

```css
h1 { font-size: clamp(1.5rem, 2.5vw, 2.5rem); }  /* fluid, no breakpoints */
h1, h2 { text-wrap: balance; }                    /* balanced headlines */
article p { text-wrap: pretty; }                  /* avoid orphan words */
.title { line-clamp: 3; -webkit-line-clamp: 3; overflow: hidden; }  /* truncate */
```

- Load fonts with `font-display: swap` (never invisible-text FOIT).
- Use variable fonts (`font-weight: 100 900`) over multiple files.

## Color & theming

- `color-scheme: light dark` — native dark-mode form controls/scrollbars.
- `light-dark(light, dark)` — one declaration per token for dark mode.
- `color-mix(in oklch, var(--primary) 88%, black)` — hover/derived shades.
- `oklch()` / `color(display-p3 …)` — perceptually uniform, wide-gamut colors.
- `contrast-color(var(--bg))` — automatic readable text (track: not shipped yet).

## Support tiers

No build step, so target evergreen browsers. Treat features by availability:

**Use now** (Baseline widely available): grid/flex, `gap`, `aspect-ratio`, `inset`,
`position: sticky`, logical properties, custom properties + `var()`, `min()`/`max()`/
`clamp()`, `:is()`/`:where()`/`:has()`/`:focus-visible`/`:nth-child(n of …)`,
`@layer`, `@supports`, `@property`, native nesting, `line-clamp`, `font-display`,
`object-fit`, `backdrop-filter`, `scroll-snap`, `scrollbar-gutter`,
`content-visibility`, `<dialog>`, `popover`, `color-mix()`, `accent-color`.

**Progressive enhancement** (Baseline newly available — use with a fallback):
`light-dark()`, `oklch()`, `text-wrap: balance/pretty`, container queries (`@container`
+ `cqi` units), `subgrid`, `@scope`, `@starting-style`, `transition-behavior:
allow-discrete`, View Transitions API, scroll-driven animations (`animation-timeline`),
anchor positioning, `:user-valid`/`:user-invalid`, `field-sizing: content` (Chromium
only), `zoom`, `scrollbar-color`/`scrollbar-width`.

**Track / experimental** (limited — do not build critical paths): `display:
grid-lanes` (masonry), `shape()`/`border-shape`/`corner-shape`, `contrast-color()`,
`@function`, `if()`, `sibling-index()`/`sibling-count()`, typed `attr()`,
`text-box-trim`, `base-select` (`appearance: base-select`).

Verify current status on [caniuse](https://caniuse.com) / MDN before relying on the
bottom two tiers — support ships quickly.

## Anti-patterns (old → modern)

| Instead of | Use |
|---|---|
| `position: absolute` + `transform: translate(-50%,-50%)` | `display: grid; place-items: center` |
| `padding-bottom: 56.25%` aspect ratio | `aspect-ratio: 16 / 9` |
| margin hacks on `:not(:last-child)` | `gap` on the flex/grid parent |
| `@media (min-width)` per component | `@container` queries |
| Sass variables / `$color` | CSS custom properties + `var()` |
| `!important` specificity wars | `@layer` ordering + `:where()` resets |
| Sass `lighten()`/`darken()`/`mix()` | `color-mix()` + relative color syntax |
| duplicate `@media prefers-color-scheme` blocks | `light-dark()` + `color-scheme` |
| `:focus` outline on all inputs | `:focus-visible` |
| JS `closest()` / class toggle on parent | `:has()` |
| JS scroll listener + sticky class | `position: sticky` |
| JS `oninput` textarea resize | `field-sizing: content` |
| JS carousel / Swiper | `scroll-snap-type` (+ `::scroll-button()` when ready) |
| JS click-outside / focus trap | `<dialog>` + `popover` + `closedby` |
| `100vh` on mobile | `100svh` / `100dvh` |
| `background-image` cover | `object-fit: cover` |

## App conventions (this repo)

- **Stack**: gomponents server-side HTML, Open Props tokens via CDN, plain CSS in
  `app.css`, htmx for partial updates. No build step, no inline styles.
- **Classless base elements** + one class per component + unprefixed modifiers
  (`.muted`, `.outline`, `.small`), scoped to elements via nesting.
- **Semantic structure** mirrors the "component" pattern above: `<ul>` of
  `<li><article><header>/<footer>`. Use `> li` to separate a list's items from
  nested lists.
- **No utility framework, no Tailwind.** Structural selectors (`> *`, `:first-child`,
  sibling combinators) over extra classes.
- **Forms**: `main > form` is a grid; `<label>` wraps its control and stacks label
  text above via `label { display: grid; gap }`. Placeholder-only fields are an
  anti-pattern — pair a `<label>`.
- **Tokens**: Open Props primitives; semantic tokens in `:root` (see `theme` layer).
- See `docs/frontend.md` for the gomponents/htmx rendering stack.
