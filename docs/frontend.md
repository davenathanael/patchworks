# Frontend Stack

## Rendering

- **Full pages**: server-rendered HTML via gomponents (`Page()` + `AppShell()`).
- **Interactivity**: htmx 2.x for partial updates (search, filter, add bookmark); forms and links fall back to full-page requests without JS.

## Gomponents

- Reusable components in `internal/http/views/`.
- Composition over inheritance.
- Layout: `Page(title, children...)`, `AppShell(user, mainContent)`.
- Dot-import `maragu.dev/gomponents`, `maragu.dev/gomponents/html`.

## Styling

- **Design tokens**: [Open Props](https://open-props.style) loaded from CDN (`https://unpkg.com/open-props`); semantic app tokens in `app.css` `:root`.
- **Conventions**: see `docs/css-guidelines.md` (canonical styling/HTML guideline).

## Assets

- `resources/static/css/app.css` — the only authored stylesheet; imports the Geist font.
- Served at `/static/*` via `http.FileServer`.
