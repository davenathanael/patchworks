# Roadmap (`FR-*`)

**Scope:** future product intent that has not landed as a current feature ticket. The authoritative roadmap is [`docs/spec.md`](../spec.md#7-future-developments); this file gives each current `FR-*` ID an explicit status, scope boundary, and acceptance direction without claiming implementation details that are not in the repository.

Historical FR IDs that landed under another family are not duplicated here. Their provenance is recorded in the spec revision table and in the relevant ticket's **Related** links (for example, FR-10 → BK-14, FR-4 → BK-15, and FR-2 remains current below).

## Index

| ID | Status | Priority | Planned intent |
|---|---|---|---|
| [FR-2](#fr-2--oauth--oidc-login) | planned | P1 | External identity login |
| [FR-7](#fr-7--full-text-search-and-tag-autocomplete) | planned | P2 | Better search and tag discovery |
| [FR-8](#fr-8--import-and-export) | planned | P2 | Browser import and collection export |
| [FR-9](#fr-9--link-health) | planned | P2 | Scheduled broken-link checks |
| [FR-12](#fr-12--email-verification-and-password-reset) | planned | P1 | Account recovery and verification |
| [FR-14](#fr-14--browser-extension-or-bookmarklet) | planned | P2 | One-click capture |
| [FR-15](#fr-15--collection-sharing-links) | planned | P2 | Secret, expiring, and public links |
| [FR-16](#fr-16--keyboard-shortcuts-or-command-palette) | planned | P2 | Faster navigation/actions |
| [FR-17](#fr-17--external-api) | planned | P2 | Automation/integration API |
| [FR-18](#fr-18--self-host-polish) | planned | P0 | Release, runtime, database, and recovery operations |
| [FR-19](#fr-19--readability-and-offline-snapshots) | planned | P2 | Save readable/offline page content |
| [FR-20](#fr-20--activity-log-for-shared-collections) | planned | P2 | Audit shared-collection changes |
| [FR-21](#fr-21--mobile-responsive-pass) | planned | P2 | Improve small-screen support |
| [FR-25](#fr-25--default-new-bookmarks-to-the-reading-list) | planned | P1 | Per-user reading-list default |

## FR-2 — OAuth / OIDC login

- **Status:** `planned`
- **Priority:** `P1`
- **Spec:** [`FR-2`](../spec.md#71-near-term--schemadesign-ready)
- **Intent:** Allow a user to sign in with an external OAuth/OIDC identity, initially framed around Google/GitHub, alongside or instead of local email/password authentication.
- **Planned scope:** Add an external-identity data model and provider redirect/callback/linking flow while preserving the existing nullable password-hash/user model. Provider identity uniqueness, account linking, callback validation, and failure/retry behavior must be specified before implementation.
- **Authorization/UX:** The identity flow must not allow an external provider to bypass account linking or collection authorization. Public auth pages need clear provider/error states and must retain a usable local-auth path unless product scope explicitly removes it.
- **Acceptance direction:** A user can complete a supported provider flow once, return to an authenticated session, and be identified consistently on later logins; duplicate/cross-account linking is rejected; invalid/tampered callbacks fail safely; existing password users are not silently merged.
- **Implementation notes:** [`docs/auth.md`](../auth.md) records the current nullable `password_hash` and proposed `user_identities` direction. No OAuth/OIDC table, provider, or route is currently implemented.
- **Related:** [AU-1](authentication.md#au-1--register-with-email-and-password), [AU-2](authentication.md#au-2--log-in-and-create-a-session), [FR-12](#fr-12--email-verification-and-password-reset).
- **Open work:** **P1 — Define identity-linking and provider contract before coding.**

## FR-7 — Full-text search and tag autocomplete

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-7`](../spec.md#72-medium-term)
- **Intent:** Improve bookmark discovery beyond current case-insensitive title/URL substring matching, with tag autocomplete in the filter UI.
- **Planned scope:** Evaluate Postgres full-text search (`tsvector` or an extension such as `pg_trgm`), required supporting indexes, and a tag suggestion/autocomplete interaction. Ranking, migration, and small-corpus behavior are intentionally unspecified.
- **Authorization/UX:** Search must preserve author/collection visibility and archived exclusion. Autocomplete must remain a normal server-rendered filter path, not a client-only data leak.
- **Acceptance direction:** Chosen search behavior improves relevant matching without changing the existing no-filter/URL-state model; tag suggestions are keyboard/accessibility friendly; no result crosses authorization boundaries.
- **Implementation notes:** Current [SR-1](search-filtering.md#sr-1--search-title-and-url) and [TG-3](tags.md#tg-3--show-tag-counts) are the baseline. No full-text index or autocomplete endpoint exists.
- **Related:** [SR-1](search-filtering.md#sr-1--search-title-and-url), [SR-5](search-filtering.md#sr-5--search-as-you-type), [TG-3](tags.md#tg-3--show-tag-counts).
- **Open work:** **P2 — Choose ranking/index strategy and define autocomplete data contract.**

## FR-8 — Import and export

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-8`](../spec.md#72-medium-term)
- **Intent:** Let users bring links in from browser HTML exports or OPML and export collections in JSON or HTML.
- **Planned scope:** Support one or more browser export formats and collection-oriented export formats. Import mapping, duplicate handling, folder/collection reconstruction, metadata preservation, and error reporting require a product decision; the format-priority question Q6 remains parked.
- **Authorization/UX:** Imports create data only for the authenticated user and must not grant access to another account's collections. Exports must include only data the requester is allowed to see.
- **Acceptance direction:** A selected import produces a preview/validation result and a clear partial-failure report; an export is deterministic and round-trips the documented subset; duplicate URLs use an explicit policy rather than silently changing BK-11.
- **Implementation notes:** No importer/exporter or additional format is present. [BK-11](bookmarks.md#bk-11--warn-about-duplicate-urls) and [CL-1](collections.md#cl-1--create-a-collection) are the relevant existing boundaries.
- **Related:** [BK-1](bookmarks.md#bk-1--add-a-bookmark), [BK-11](bookmarks.md#bk-11--warn-about-duplicate-urls), [CL-1](collections.md#cl-1--create-a-collection).
- **Open work:** **P2 — Resolve Q6 (first import format) and define export/import data contracts.**

## FR-9 — Link health

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-9`](../spec.md#72-medium-term)
- **Intent:** Periodically check saved URLs and show broken-link status/badges so stale bookmarks are easier to maintain.
- **Planned scope:** Define check frequency, HEAD/GET fallback, timeout/retry policy, concurrency, status history, and the background-job infrastructure. The current app has no scheduler/worker for this purpose.
- **Authorization/UX:** Only the bookmark owner should see or trigger private link-health metadata; a health badge must not become a server-side browsing bypass.
- **Acceptance direction:** Checks are bounded and observable; transient failures do not immediately mark a link permanently dead; archived/read-only state and existing URL visibility remain correct.
- **Implementation notes:** No background job, health column, or scheduled check exists. This ticket would introduce new infrastructure rather than extend an existing scheduler.
- **Related:** [BK-2](bookmarks.md#bk-2--derive-a-title-when-needed), [BK-9](bookmarks.md#bk-9--archive-a-bookmark), [UX-3](interaction.md#ux-3--render-unexpected-errors-safely).
- **Open work:** **P2 — Specify operational limits and job ownership before introducing background work.**

## FR-12 — Email verification and password reset

- **Status:** `planned`
- **Priority:** `P1`
- **Spec:** [`FR-12`](../spec.md#72-medium-term)
- **Intent:** Add account verification and password recovery so local accounts have a safe way to prove email ownership and recover access.
- **Planned scope:** Define verification state, token lifecycle/expiry/revocation, reset request privacy, notification delivery, token storage, and interactions with [FR-2](#fr-2--oauth--oidc-login). Current registration/login intentionally have neither flow.
- **Authorization/UX:** Reset responses must not reveal whether an email exists. Tokens must be single-use, time-bounded, and protected in logs/storage. Existing sessions should have an explicit invalidation policy after password reset.
- **Failure semantics (planned):** Delivery failures, invalid/expired/reused tokens, and account-state conflicts need explicit user-facing outcomes without revealing account existence. No current route exists, so this is a design boundary rather than implemented behavior.
- **Acceptance direction:** A legitimate user can verify and reset; expired/reused/tampered tokens fail safely; privacy-preserving request/response behavior is maintained; OAuth-only accounts are handled deliberately.
- **Implementation notes:** [`docs/auth.md`](../auth.md) identifies verification/reset as future work. No token table, mailer, or recovery route exists.
- **Related:** [AU-1](authentication.md#au-1--register-with-email-and-password), [AU-2](authentication.md#au-2--log-in-and-create-a-session), [AU-3](authentication.md#au-3--protect-the-session-cookie), [FR-2](#fr-2--oauth--oidc-login).
- **Open work:** **P1 — Choose delivery/provider and token/session policy.**

## FR-14 — Browser extension or bookmarklet

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-14`](../spec.md#73-longer-term--exploratory)
- **Intent:** Capture a current page into Patchworks with one click, without making an extension a prerequisite for the core app.
- **Planned scope:** Decide between a browser extension, bookmarklet, or both; define authenticated hand-off, selected metadata, collection/tag defaults, duplicate handling, and support/maintenance cost.
- **Authorization/UX:** The capture flow must use the existing authenticated bookmark contract and must not expose session credentials to a third-party page. Browser permission and privacy behavior need explicit review.
- **Acceptance direction:** A user can initiate a save from a supported browser context, lands in a normal authenticated save flow or confirms a save, and receives clear duplicate/error feedback; the server-rendered core remains usable without it.
- **Implementation notes:** No extension, bookmarklet, or capture endpoint exists. [BK-1](bookmarks.md#bk-1--add-a-bookmark) is the product boundary this would reuse.
- **Related:** [BK-1](bookmarks.md#bk-1--add-a-bookmark), [BK-11](bookmarks.md#bk-11--warn-about-duplicate-urls), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:** **P2 — Select capture surface and define secure hand-off.**

## FR-15 — Collection sharing links

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-15`](../spec.md#73-longer-term--exploratory)
- **Intent:** Share a collection without requiring a registered membership for three link classes: secret tokenized links, time-based secret links, and public pretty-slug links.
- **Planned scope:** Specify token generation/storage, expiry/revocation, public-indexing and privacy defaults, slug collision/uniqueness, authorization, abuse/rate controls, and whether links expose all collection bookmarks or a snapshot. Current product has registered member sharing only.
- **Authorization/UX:** Link capability must be explicit and revocable; an expired/revoked/wrong-kind link must not fall back to collection membership or leak existence. Sharing-link creation/removal should have clear owner-only controls.
- **Acceptance direction:** Each link kind has documented visibility and lifecycle; only the intended collection is exposed; expiry/revocation works; ordinary member role rules remain unchanged; public links have an explicit indexing/privacy decision.
- **Implementation notes:** No share-link schema, route, or view exists. Current member access is defined by [CL-9](collections.md#cl-9--enforce-collection-roles); the roadmap explicitly identifies public/guest sharing as a gap.
- **Related:** [CL-5](collections.md#cl-5--invite-members-and-offer-manageable-collections), [CL-7](collections.md#cl-7--view-shared-bookmarks), [CL-9](collections.md#cl-9--enforce-collection-roles), [FR-17](#fr-17--external-api).
- **Open work:** **P2 — Define capability and privacy model before schema work.**

## FR-16 — Keyboard shortcuts or command palette

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-16`](../spec.md#73-longer-term--exploratory)
- **Intent:** Speed up common navigation and actions for keyboard-oriented users.
- **Planned scope:** Choose shortcuts versus a command palette, define command inventory, discoverability, conflicts with browser/assistive technologies, and no-JS alternative expectations.
- **Authorization/UX:** Commands must invoke the same server-side actions and role checks as visible controls; keyboard shortcuts cannot become an alternate authorization path.
- **Acceptance direction:** Selected commands are discoverable, remappable or conflict-safe by decision, keyboard accessible, and equivalent to existing server flows; the core app remains usable without the enhancement.
- **Implementation notes:** No shortcut registry or client command layer exists. The current interaction baseline is [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Related:** [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement), [UX-4](interaction.md#ux-4--show-inline-validation-errors), [UX-5](interaction.md#ux-5--provide-the-authenticated-app-shell).
- **Open work:** **P2 — Choose interaction pattern and accessibility constraints.**

## FR-17 — External API

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-17`](../spec.md#73-longer-term--exploratory)
- **Intent:** Provide a supported automation/integration surface for managing bookmarks and collections outside the web UI.
- **Planned scope:** Define API versioning, authentication/token lifecycle, resource ownership, collection-role enforcement, rate limits, pagination, error format, and compatibility policy. No endpoint or public API contract exists.
- **Authorization/UX:** Every API operation must reuse the same author/collection permission rules as the web app; possession of a bookmark URL or UUID must not grant access. API credentials and scopes need a separate secure design.
- **Acceptance direction:** Documented clients can perform the approved CRUD/filter operations with stable errors and pagination; cross-user/resource access is denied without existence leaks; revocation/expiry is testable.
- **Implementation notes:** No API router, auth scheme, or public schema exists. [SR-1](search-filtering.md#sr-1--search-title-and-url)–[BK-15](bookmarks.md#bk-15--page-browse-lists) define the current web behavior an API would need to preserve or intentionally revise.
- **Related:** [AU-3](authentication.md#au-3--protect-the-session-cookie), [BK-15](bookmarks.md#bk-15--page-browse-lists), [CL-9](collections.md#cl-9--enforce-collection-roles), [FR-15](#fr-15--collection-sharing-links).
- **Open work:** **P2 — Approve API scope and authentication model.**

## FR-18 — Self-host polish

- **Status:** `planned`
- **Priority:** `P0`
- **Spec:** [`FR-18`](../spec.md#73-longer-term--exploratory)
- **Intent:** Make self-hosting operationally smoother beyond the current container/production deployment baseline. This is an operational backlog area; its findings remain scoped to FR-18 rather than creating a new ticket family.
- **Planned scope:** The repository has a multi-stage image, production Compose stack, healthchecked Postgres, a separate migration service, CI, and deployment documentation. Remaining work includes release gating/reproducibility, graceful lifecycle, application health, database startup/readiness, schema/migration integrity, security headers, observability, backup/restore, and operational cleanup. The existing CI/CD baseline is not completion of this ticket. Detailed cross-cutting findings live in the relevant area documents and the existing feature contracts.
- **Authorization/UX:** Operational endpoints and diagnostics must not expose secrets or bypass auth. Backup/restore must preserve the data and access-control model.
- **Acceptance direction:** A clean-host deploy, upgrade, rollback boundary, and restore drill are repeatable; startup and readiness are bounded; releases are gated and reproducible from pinned inputs; migrations and `schema.sql` agree; failures emit actionable, non-secret diagnostics; no credential leakage occurs. Product-facing security/UX requirements remain in their existing AU/BK/CL/SR/UX tickets.
- **Implementation notes:** Current deployment guidance is [`docs/deploy.md`](../deploy.md). The archived CI/CD plan's migration-image idea and private-GHCR follow-ups are operational backlog evidence, not new product IDs.
- **Related:** [UX-3](interaction.md#ux-3--render-unexpected-errors-safely), [AU-3](authentication.md#au-3--protect-the-session-cookie), [SR-1](search-filtering.md#sr-1--search-title-and-url), [BK-6](bookmarks.md#bk-6--model-a-saved-bookmark).
- **Open work:**
  - **P0 — Define a single required CI signal:** expose lint, unit, and integration results under a stable required status for protected branches and release tags, so missing/skipped jobs cannot look green.
  - **P0 — Gate releases on the full check set:** make tag publication depend on lint, unit, and integration success, then publish immutable version/digest metadata rather than relying on mutable `latest` alone.
  - **P0 — Make builds reproducible and attestable:** pin release actions/images by immutable digest where supported, record source revision/build metadata, and generate an SBOM/provenance or equivalent verification artifact.
  - **P1 — Package the migration runner:** ship a pinned migration image/bundle with the application release (or make the chosen checkout-based contract immutable and version-verified), so production migrations match the deployed application revision.
  - **P1 — Add app health and observability:** expose minimal liveness/readiness signals plus bounded request/runtime/database metrics or structured signals that support failure diagnosis without exposing secrets or personal data.
  - **P1 — Document and rehearse backup/restore:** provide an operator runbook and automated smoke check covering data, sessions, ownership/memberships, migrations, and integrity before/after restore.
  - **P1 — Validate configuration ranges and combinations:** reject invalid ports, timeouts, environment/session settings, and production-only combinations at startup with actionable operator errors.
  - **P1 — Standardize operational logging:** keep request logging canonical and log startup, shutdown, migration, and dependency failures at actionable levels with request/error correlation and no credentials.
  - **P1 — Decompose cross-cutting maintenance seams:** split oversized handler/repository/view responsibilities behind existing interfaces while preserving behavior, and keep authorization, no-JS, and integration tests at the boundaries.
  - **P2 — Remove stale operational artifacts:** after their durable guidance is linked, retire obsolete plans/templates and ensure production examples, image tags, and deployment instructions identify one canonical path.

## FR-19 — Readability and offline snapshots

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-19`](../spec.md#73-longer-term--exploratory)
- **Intent:** Optionally save a readable/offline snapshot of a linked page rather than only its URL and metadata.
- **Planned scope:** This is deliberately outside the initial product. It requires content fetching/storage, extraction/readability decisions, media/size limits, privacy/retention, refresh/removal behavior, and a clear distinction from the existing bounded title fetch in BK-2.
- **Authorization/UX:** Snapshots must follow bookmark visibility and deletion/archive rules; private content must not become public through sharing. Storage and resource limits are part of the product decision.
- **Acceptance direction:** If implemented, a user can opt in per bookmark, see snapshot availability, remove it, and understand storage/failure behavior without changing URL-only saves by default.
- **Implementation notes:** No snapshot schema, fetcher beyond [`internal/http/client/title.go`](../../internal/http/client/title.go), or reader UI exists. The spec lists this as a non-goal for the initial scope.
- **Related:** [BK-2](bookmarks.md#bk-2--derive-a-title-when-needed), [BK-9](bookmarks.md#bk-9--archive-a-bookmark), [BK-10](bookmarks.md#bk-10--manage-archived-bookmarks).
- **Open work:** **P2 — Keep exploratory until storage/privacy/product scope is approved.**

## FR-20 — Activity log for shared collections

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-20`](../spec.md#73-longer-term--exploratory)
- **Intent:** Record who added/removed or changed shared-collection resources so owners and members can understand activity.
- **Planned scope:** Define event types, actor/resource identity, ordering/retention, visibility by role, privacy of URLs/notes, and whether events are immutable. No event model exists.
- **Authorization/UX:** Activity visibility must be role-scoped and must not reveal private bookmarks or member data to unauthorized users. Logging must not become a substitute for security audit controls.
- **Acceptance direction:** Authorized users can inspect a coherent, tamper-resistant history for the collection; unauthorized users cannot query it; deletion/archival effects are represented according to the chosen retention policy.
- **Implementation notes:** No activity tables, events, or UI exist. Current access semantics are in [CL-9](collections.md#cl-9--enforce-collection-roles).
- **Related:** [CL-7](collections.md#cl-7--view-shared-bookmarks), [CL-9](collections.md#cl-9--enforce-collection-roles), [FR-15](#fr-15--collection-sharing-links).
- **Open work:** **P2 — Define event/visibility/retention contract.**

## FR-21 — Mobile responsive pass

- **Status:** `planned`
- **Priority:** `P2`
- **Spec:** [`FR-21`](../spec.md#73-longer-term--exploratory)
- **Intent:** Improve the interface beyond current support tiers for small screens and touch use.
- **Planned scope:** Audit layouts, navigation, row actions, forms, collection detail, reading/archived pages, and accessibility at target viewport sizes. Preserve server-only/no-JS operation and existing semantic HTML.
- **Authorization/UX:** Responsive changes must not hide or bypass role-gated actions; touch targets and focus order need explicit review. CSS work follows [`docs/css-guidelines.md`](../css-guidelines.md).
- **Acceptance direction:** Chosen viewport matrix is usable without horizontal loss of core controls; menus/forms remain keyboard accessible; no data or authorization behavior changes.
- **Implementation notes:** No dedicated mobile feature branch or issue is present. Current styling and support-tier policy are in [`docs/frontend.md`](../frontend.md) and [`docs/css-guidelines.md`](../css-guidelines.md).
- **Related:** [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement), [UX-2](interaction.md#ux-2--keep-the-frontend-server-rendered-and-buildless), [UX-5](interaction.md#ux-5--provide-the-authenticated-app-shell).
- **Open work:** **P2 — Define target devices/viewports and test matrix.**

## FR-25 — Default new bookmarks to the reading list

- **Status:** `planned`
- **Priority:** `P1`
- **Spec:** [`FR-25`](../spec.md#71-near-term--schemadesign-ready)
- **Intent:** Let each user choose whether newly saved bookmarks are queued to the reading list by default, building on BK-14's explicit add-form checkbox and row toggle.
- **Planned scope:** Add a per-user settings value and a settings-page control; define precedence between the default, the form checkbox, and an existing bookmark's queue state. No settings page or preference field currently exists.
- **Authorization/UX:** The preference is per user and cannot alter another user's queue. The UI must make the effective default and any per-save override understandable without JavaScript.
- **Acceptance direction:** A user can set/read the preference; new saves use it as the initial checkbox state; explicit form choices and later enqueue/dequeue actions still work; existing queue state is not unexpectedly rewritten.
- **Implementation notes:** [BK-14](bookmarks.md#bk-14--maintain-a-reading-list) and the `queued_at` migration are the existing foundation. No settings route/schema exists.
- **Related:** [BK-1](bookmarks.md#bk-1--add-a-bookmark), [BK-14](bookmarks.md#bk-14--maintain-a-reading-list), [UX-5](interaction.md#ux-5--provide-the-authenticated-app-shell).
- **Open work:** **P1 — Specify default/override precedence and settings UX.**
