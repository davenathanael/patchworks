# Frontend Stack

## Rendering and progressive enhancement

The application is server-rendered with gomponents. `Page` and `AppShell` in
`internal/http/views/views.go` provide the document shell; feature views build
semantic HTML and htmx fragments. htmx is an enhancement for partial updates,
search, filtering, row actions, and pagination. Forms and ordinary links are
the default no-JS path.

There is one current exception to that rule: bookmark **archive**, archived
**restore**, and permanent bookmark **delete** controls are htmx-only buttons
(`hx-post`/`hx-swap`/`hx-confirm`) in the rendered rows. Their handlers accept
plain POST requests and redirect when the response is not htmx, but the current
markup does not expose those fallbacks. A no-JS client therefore cannot reach
these destructive flows, and `hx-confirm` is not a server-rendered
confirmation. This is tracked in [BK-9](features/bookmarks.md#bk-9--archive-a-bookmark),
[BK-10](features/bookmarks.md#bk-10--manage-archived-bookmarks), and
[UX-1](features/interaction.md#ux-1--server-rendering-with-progressive-enhancement).

Do not add a client-side framework, bundler, or build step. Keep composition in
gomponents, use semantic forms/links, and dot-import
`maragu.dev/gomponents` and `maragu.dev/gomponents/html` as the existing view
convention. Styling follows [docs/css-guidelines.md](css-guidelines.md).

## Assets and trust boundary

The current `Page` loads Open Props from `https://unpkg.com/open-props` and
htmx 2.0.10 from `https://unpkg.com/htmx.org@2.0.10/dist/htmx.min.js`. The
Open Props URL is not an exact version pin, and neither asset currently has an
integrity attribute. The router also does not currently define a Content
Security Policy or the baseline clickjacking, MIME-sniffing, referrer, and
transport headers.

Until the policy is decided, treat CDN assets as optional presentation or
progressive enhancement: they must not change server authorization, form
semantics, or the ability to complete core actions without JavaScript. The
preferred target is either self-hosting exact, reviewed asset versions or
pinning exact CDN bytes with SRI. If a CDN remains, the CSP allowlist must be
explicit and minimal, and the policy must be tested for both full-page and
partial responses. Security headers must also cover static assets without
breaking htmx.

## Open work

- **P0 — Remove the htmx-only exception for destructive bookmark actions:** add
  real POST fallbacks and server-rendered confirmation for archive, restore, and
  permanent delete. See [UX-1](features/interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **P1 — Define third-party asset trust:** self-host or integrity-pin exact
  Open Props and htmx versions, align the decision with CSP, and test that
  missing optional assets cannot change core authorization or form submission.
  See [UX-2](features/interaction.md#ux-2--keep-the-frontend-server-rendered-and-buildless).
- **P2 — Add baseline security headers:** define and test CSP, clickjacking,
  MIME-sniffing, referrer, and transport policy on full pages, htmx fragments,
  and static assets. This is operational follow-up under
  [FR-18](features/roadmap.md#fr-18--self-host-polish).
