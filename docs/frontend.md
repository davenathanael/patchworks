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
- **Priority**: OAT utilities → override OAT vars in `app.css` → OpenProps vars → custom CSS in `app.css`

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
