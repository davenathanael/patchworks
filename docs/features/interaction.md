# UI & interaction (`UX-*`)

**Scope:** the cross-cutting interaction contract: server-rendered HTML, progressive enhancement, styling/build boundaries, error presentation, validation presentation, and the authenticated shell. Product intent is in [`docs/spec.md`](../spec.md#56-ui--interaction-model--impl); canonical frontend/style rules are in [`docs/frontend.md`](../frontend.md) and [`docs/css-guidelines.md`](../css-guidelines.md).

## Index

| ID | Status | Priority | Contract |
|---|---|---|---|
| [UX-1](#ux-1--server-rendering-with-progressive-enhancement) | partial | P0 | Complete HTML-first, CSRF-safe progressive enhancement |
| [UX-2](#ux-2--keep-the-frontend-server-rendered-and-buildless) | partial | P1 | Keep the frontend buildless with a defined asset trust boundary |
| [UX-3](#ux-3--render-unexpected-errors-safely) | implemented | — | Correlatable full-page/toast errors |
| [UX-4](#ux-4--show-inline-validation-errors) | partial | P1 | Bounded, accessible inline errors and value preservation |
| [UX-5](#ux-5--provide-the-authenticated-app-shell) | implemented | — | Navigation, relative time, and identity cues |

## UX-1 — Server rendering with progressive enhancement

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`UX-1`](../spec.md#56-ui--interaction-model--impl)
- **Intent/behavior:** Pages render as complete HTML with gomponents. Core forms and links must work as full-page fallbacks; htmx adds partial swaps for search/filtering, bookmark row actions, add-save feedback, and pager updates. Archive and archived restore/delete controls are currently htmx-only buttons, so those destructive flows have no working no-JS control, and no shared CSRF mechanism protects state-changing requests.
- **Authorization:** Enhancement never replaces server-side auth/role checks. Hiding or swapping a control is not permission enforcement.
- **UX/no-JS:** Core navigation, create/edit forms, collection delete/member-removal forms, reading-list actions, and ordinary filter links have server fallbacks. **Current gap:** archive and archived restore/delete views expose only htmx buttons. The target contract requires POST-form fallbacks and explicit server-rendered confirmation for irreversible deletion. The documented htmx/no-JS pagination difference is intentional; see [BK-15](bookmarks.md#bk-15--page-browse-lists).
- **Failure/semantics:** Partial and full-page failures use the same error model. A client enhancement failure should not turn an otherwise valid server response into a false success.
- **Acceptance criteria:** Every core mutation has a real HTML form/link fallback, including archive/restore/delete; irreversible actions require a confirmation step that does not depend on JavaScript; htmx returns fragments appropriate to their target; state-changing routes enforce CSRF without breaking plain forms; progressive controls do not create an alternate authorization path.
- **Implementation notes:** [`internal/http/views/views.go`](../../internal/http/views/views.go), [`internal/http/handlers/handler.go`](../../internal/http/handlers/handler.go), [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), and [`docs/frontend.md`](../frontend.md).
- **Related:** [AU-4](authentication.md#au-4--log-out), [AU-5](authentication.md#au-5--protect-application-routes), [BK-5](bookmarks.md#bk-5--complete-a-save), [BK-8](bookmarks.md#bk-8--edit-notes-and-tags), [BK-9](bookmarks.md#bk-9--archive-a-bookmark), [BK-10](bookmarks.md#bk-10--manage-archived-bookmarks), [BK-15](bookmarks.md#bk-15--page-browse-lists), [SR-5](search-filtering.md#sr-5--search-as-you-type).
- **Open work:**
  - **P0 — Add no-JS destructive fallbacks and confirmations:** render archive, restore, and permanent bookmark delete controls as POST paths; provide server-rendered confirmation for irreversible actions because `hx-confirm` is unavailable without htmx, and test both response modes. Collection delete and member-removal already have POST forms; they still need shared CSRF and confirmation coverage.
  - **P0 — Add shared CSRF protection:** issue/validate tokens for every state-changing form (including AU-4 logout), reject missing/invalid tokens before domain mutation, and cover plain and htmx requests.
  - **P0 — Bound every request body:** provide one shared `MaxBytesReader`/decoder policy for auth, bookmark, collection, membership, and future form routes before parsing; return a stable expected response and test each decoder with oversized input.
  - **P1 — Complete redirect-target allowlisting:** audit user-controlled return targets (including reading `next` and save/edit return paths); reject absolute, protocol-relative, encoded, backslash, and malformed variants, and test that only approved local paths are used.
  - **P1 — Add rendered/no-JS/role contract tests:** assert full-page and htmx fragments, keyboard/native form behavior, no-JS destructive fallbacks, and role-dependent controls against the composed router, not only handler fakes.

## UX-2 — Keep the frontend server-rendered and buildless

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`UX-2`](../spec.md#56-ui--interaction-model--impl)
- **Intent/behavior:** The application uses gomponents for HTML, htmx as a small progressive enhancement, Open Props tokens, and one authored stylesheet at `resources/static/css/app.css`. There is no client-side framework, bundler, or build step. The current page loads Open Props and htmx from a third-party CDN without a documented asset trust/integrity policy.
- **Authorization:** Not authorization-bearing; the constraint applies to all authenticated and public views.
- **UX/no-JS:** Core behavior is server-rendered; CDN assets are enhancements. Styling is not required to understand or submit forms.
- **Failure/semantics:** Missing optional assets may reduce enhancement/visual polish but must not change server authorization or data semantics.
- **Acceptance criteria:** No JS framework/build pipeline is introduced; views remain gomponents components; CSS changes follow the repository's token/layer/nesting rules; the app can be built and served from the documented static path; external assets are version-pinned and protected by an explicit integrity/CSP or self-hosting decision.
- **Implementation notes:** [`docs/frontend.md`](../frontend.md), [`docs/css-guidelines.md`](../css-guidelines.md), [`docs/adr/0001-styling-overhaul-mocha.md`](../adr/0001-styling-overhaul-mocha.md), and [`internal/http/views/views.go`](../../internal/http/views/views.go).
- **Related:** [UX-1](#ux-1--server-rendering-with-progressive-enhancement), [UX-5](#ux-5--provide-the-authenticated-app-shell), [BK-7](bookmarks.md#bk-7--show-bookmark-notes).
- **Open work:**
  - **P1 — Define third-party asset trust:** self-host or integrity-pin exact Open Props/htmx versions, align the policy with CSP, and test that missing optional assets cannot change core authorization or form submission.
  - **P2 — Add baseline security headers:** set an explicit CSP and appropriate clickjacking, MIME-sniffing, referrer, and transport headers on full-page and partial responses without breaking htmx or static assets.

## UX-3 — Render unexpected errors safely

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`UX-3`](../spec.md#56-ui--interaction-model--impl)
- **Intent/behavior:** Unexpected handler errors are logged once with a fresh non-guessable UUID error ID, returned in `X-Error-Id`, and rendered as a styled full-page error or an htmx toast. The displayed reference matches the server log reference.
- **Authorization:** Error rendering must not disclose protected resource details or bypass access checks. The middleware is shared across routes.
- **UX/no-JS:** Full-page errors are usable without JavaScript. htmx errors target the toast container; the toast has a dismiss control, while the full page provides a dashboard link.
- **Failure/semantics:** Expected validation, authentication, and authorization outcomes are handled by handlers where implemented; classified not-found/forbidden sentinels can still reach this shared path today, and the current error event logs them at error level. Unknown errors map to a generic user message; internal details remain in logs.
- **Acceptance criteria:** Every unexpected full-page error has a non-guessable reference and a generic message; htmx receives a toast target; the reference appears in the log and response; no raw database/internal error text is shown.
- **Implementation notes:** [`internal/http/middleware/errors.go`](../../internal/http/middleware/errors.go), [`internal/http/views/errors.go`](../../internal/http/views/errors.go), and middleware tests. Logging conventions are in [`docs/logging.md`](../logging.md).
- **Related:** [AU-5](authentication.md#au-5--protect-application-routes), [UX-1](#ux-1--server-rendering-with-progressive-enhancement), [UX-4](#ux-4--show-inline-validation-errors).
- **Open work:** None recorded.

## UX-4 — Show inline validation errors

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`UX-4`](../spec.md#56-ui--interaction-model--impl)
- **Intent/behavior:** Expected validation errors should appear next to the offending field, with bounded submitted values preserved. Shared text inputs use `aria-invalid` when an error exists and `aria-describedby`; top-level form errors use an alert/toast. Current validation covers mostly non-empty credentials/collection names and `url.Parse` errors, with no general length/count bounds.
- **Authorization:** Validation is a boundary concern and must not be used as the only authorization check.
- **UX/no-JS:** Forms re-render server-side for both plain and htmx requests. The current bookmark form retargets htmx errors to the form; other forms render a full page.
- **Failure/semantics:** Empty credentials, missing collection names, malformed URLs, and invalid form encodings are expected validation outcomes. Storage/remote errors are not rendered as field errors.
- **Acceptance criteria:** Invalid input identifies the field, preserves safe entered values, and does not mutate data; valid input has no stale error; form-level errors are announced; htmx error responses are swapped rather than dropped. **Current gaps:** most free-form fields have no server-side length/count policy, and textareas/selects are not uniformly wired to field-level error/ARIA helpers.
- **Implementation notes:** [`internal/http/handlers/errors.go`](../../internal/http/handlers/errors.go), [`internal/http/views/forms.go`](../../internal/http/views/forms.go), [`internal/http/views/errors.go`](../../internal/http/views/errors.go), and handler/view tests.
- **Related:** [AU-6](authentication.md#au-6--validate-empty-credentials), [BK-3](bookmarks.md#bk-3--reject-malformed-urls), [BK-11](bookmarks.md#bk-11--warn-about-duplicate-urls), [UX-3](#ux-3--render-unexpected-errors-safely).
- **Open work:**
  - **P1 — Define and enforce boundary validation:** set explicit maximums for URL, email, password, names/descriptions, notes, titles, and tags plus collection/tag counts; reject over-limit input before persistence/fetch and return field errors.
  - **P2 — Make field-error wiring uniform:** wire labels, controls, descriptions, `aria-invalid`, and live regions consistently for textareas/selects as well as shared text inputs, and test rendered error/no-error markup.

## UX-5 — Provide the authenticated app shell

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`UX-5`](../spec.md#56-ui--interaction-model--impl)
- **Intent/behavior:** Authenticated pages use a shared shell with top/side navigation to dashboard, collections, reading list, archived bookmarks, and logout. Bookmark rows show relative timestamps and collection/member views use email-derived initials for avatars.
- **Authorization:** The shell is rendered only after middleware authentication for application pages; it is not an access-control mechanism.
- **UX/no-JS:** Navigation links and logout work as ordinary links. Relative time is rendered server-side and does not require a client clock.
- **Failure/semantics:** If the user cannot be loaded, middleware handles authentication before shell rendering. Rendering errors use UX-3.
- **Acceptance criteria:** Authenticated pages expose the expected navigation; timestamps read as relative values; identity cues are present where designed; logout returns to login; public auth pages do not require the authenticated shell.
- **Implementation notes:** [`internal/http/views/views.go`](../../internal/http/views/views.go), [`internal/http/views/collections.go`](../../internal/http/views/collections.go), and [`docs/frontend.md`](../frontend.md).
- **Related:** [AU-5](authentication.md#au-5--protect-application-routes), [UX-1](#ux-1--server-rendering-with-progressive-enhancement), [BK-14](bookmarks.md#bk-14--maintain-a-reading-list), [BK-10](bookmarks.md#bk-10--manage-archived-bookmarks).
- **Open work:** None recorded. Mobile responsiveness beyond current support tiers is [FR-21](roadmap.md#fr-21--mobile-responsive-pass).
