# Frontend Stack

## Rendering

- **Full pages**: initial HTML with nav, layout, assets
- **Partials**: HTML fragments for fixi.js updates (no layout wrapper)
- Views check context flag for full vs partial rendering

## Gomponents

- Reusable components in `internal/http/views/`
- Composition over inheritance
- Layout: `Page(title, children...)`, `AppShell(user, sidebar, main)`
- Dot-import `maragu.dev/gomponents`, `maragu.dev/gomponents/html`

## Styling (OAT CSS + OpenProps)

- No Tailwind, no build step, no arbitrary values (px/rem)
- **Priority**: OAT utilities → OAT variables → custom CSS in `app.css`
- **No inline styles** unless explicitly requested
- **Nested CSS** with a top-level class naming the component (e.g., `.filter-bar`, `.collection-row`), then nested sub-classes for parts (e.g., `.filter-bar .search-field`, `.collection-row .collection-info`)
- **Limit utility classes** to 1–3 per element. Beyond that, define a component class in `app.css` or use element/data selectors (like OAT does)

### Modern CSS Principles

Follow [modern-css.com](https://modern-css.com/) guidelines. Key patterns already in use or to prefer:

- **`gap` on flex** — not just grid
- **`margin-inline` / `padding-inline`** — logical properties, direction-aware
- **`inset` shorthand** — instead of `top/right/bottom/left`
- **`text-wrap: balance`** — on headings for better line breaks
- **`text-wrap: pretty`** — on prose/descriptions to avoid orphan words
- **`font-display: swap`** — prevent invisible text during font load
- **`color-scheme: light dark`** — OAT handles this
- **`@layer`** — OAT uses `theme, base, components, animations, utilities`
- **`prefers-reduced-motion`** — OAT handles this
- **`:where()`** — zero-specificity resets (OAT uses it)
- **`content-visibility: auto`** — for long lists in the future
- **`field-sizing: content`** — auto-growing textareas when available
- **`inputmode` / `enterkeyhint`** — better mobile keyboard UX
- **`loading="lazy"`** — for images below the fold

When OAT CSS contradicts a modern-css recommendation, defer to OAT for consistency.

### OAT Layout

`.flex`, `.flex-col`, `.hstack`, `.vstack`, `.items-center`, `.justify-center`, `.justify-between`, `.justify-end`, `.align-left`, `.align-center`, `.align-right`

### OAT Spacing

`.gap-1`, `.gap-2`, `.gap-4`, `.mt-2`, `.mt-4`, `.mt-6`, `.mb-2`, `.mb-4`, `.mb-6`, `.p-4`

### OAT Other

`.unstyled`, `.text-light`, `.text-lighter`, `.w-100`

### OAT CSS Variables (`:root`)

- **Spacing**: `--space-1` through `--space-18` (0.25rem–4.5rem)
- **Colors**: `--color-bg-*`, `--color-text-*`, `--color-border`, `--primary`, `--danger`, `--success`
- **Typography**: `--font-sans`, `--font-mono`, `--text-1` through `--text-8`
- **Layout**: `--border-radius`, `--radius-small` through `--radius-full`, `--shadow-*`
- **Animation**: `--transition-fast`, `--transition`

### Custom Classes (in app.css)

Only when no OAT utility exists:

- `.link-title` — truncation ellipsis
- `.link-list` — link row container
- `.text-muted` — secondary text
- `.tag-badge` — compact tag
- `.filter-list` — sidebar filter
- `.dashboard-section` — section spacing

## Interactivity (fixi.js)

- No build step, progressive enhancement
- Server renders HTML; fixi.js adds partial updates
- Use `data-*` attributes for fixi.js behaviors
