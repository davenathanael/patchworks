# Frontend Stack

## Rendering

- **Full pages**: server-rendered HTML via gomponents (`Page()` + `AppShell()`).
- **Interactivity**: plain HTML form posts + redirects; no client-side framework.
  - htmx partial updates are planned but not yet implemented.

## Gomponents

- Reusable components in `internal/http/views/`.
- Composition over inheritance.
- Layout: `Page(title, children...)`, `AppShell(user, mainContent)`.
- Dot-import `maragu.dev/gomponents`, `maragu.dev/gomponents/html`.

## Styling

- **Design tokens**: [Open Props](https://open-props.style) loaded from CDN (`https://unpkg.com/open-props`); semantic app tokens in `app.css` `:root`.
- **Conventions**: classless base elements, semantic HTML structure, unprefixed modifiers — see `docs/html-css.md`.

## Assets

- `resources/static/css/app.css` — the only authored stylesheet; imports the Geist font.
- Served at `/static/*` via `http.FileServer`.
