# Patchworks — Product Specification

**Status:** Draft v1.11 · **Last updated:** 2026-09-24 · **Owner:** Dave Nathanael

| Rev | Date | Change |
|-----|------|--------|
| 0.1 | 2026-09-02 | Initial draft from codebase + docs audit |
| 0.2 | 2026-09-03 | Per review: answered §7 Q1–Q5, Q7 (kept Q6 parked); archive ≈ soft-delete + archived filter; 3 fixed roles; 3 share-link types; bookmark notes added (FR-22); dropped §8 (tech → `docs/`); doc workflow → `docs/process.md` |
| 0.3 | 2026-09-04 | BK-4: dashboard shows recent bookmarks only; all bookmarks reached via search/filter (removed redundant "Your Bookmarks" show-all list) |
| 0.4 | 2026-09-04 | Row-actions design settled (see `docs/mockups/design-bookmark-actions.html`): one 3-item menu per bookmark (Edit notes & tags · Edit collections · Archive); notes = 1-line clamp + native expand; title/domain immutable; FR-1/FR-3/FR-13/FR-22 reworded; FR-13, FR-22 moved to §6.1 |
| 0.5 | 2026-09-04 | FR-22 landed in §5.2 as **BK-7** — notes column + row display (1-line clamp, native More/Less expand via CSS `:has()`) |
| 0.6 | 2026-09-04 | FR-3 edit half landed in §5.2 as **BK-8** — inline Edit notes & tags panel (author-only, title/domain immutable); FR-3 remainder (hard-delete) stays in §6.1 |
| 0.7 | 2026-09-04 | FR-13 landed in §5.3 as **CL-8** — Edit collections popover (search + checkboxes, preloaded membership, one save; own links only) |
| 0.8 | 2026-09-04 | FR-1 archive half landed in §5.2 as **BK-9** — Archive row action (hx-confirm, `archived_at` set, row leaves list) + `archived_at IS NULL` filters on all browse queries; FR-1 remainder (restore/hard-delete/Archived page) stays in §6.1 |
| 0.9 | 2026-09-04 | FR-1 fully landed in §5.2 as **BK-10** — Archived page (`/archived`) with restore + permanent delete; FR-1 removed from §6.1 |
| 1.0 | 2026-09-04 | FR-6 landed in §5.3 as **CL-9** — collection roles enforced (404 non-member / 403 insufficient role); new §6 “Access control & permissions” — consolidated action matrix for tickets/tests; roadmap renumbered (Future developments → §7, Open questions → §8); FR-6 removed from roadmap |
| 1.1 | 2026-09-04 | FR-3 retired from §7.1 — its hard-delete remainder was already landed as BK-10 (Archived-page permanent delete); stale FR-3 pointers fixed (§3 non-goals, tag gaps, Q1/Q3, §6.3 matrix, §5.2 save-form gap → CL-8) |
| 1.2 | 2026-09-04 | FR-5 landed in §5.1 as **AU-7** — successful logins stamp `users.last_login_at`; FR-5 removed from §7.1 |
| 1.3 | 2026-09-04 | FR-11 landed in §5.2 as **BK-11** — soft duplicate-URL reminder with Save-anyway override; FR-11 removed from §7.2 |
| 1.4 | 2026-09-04 | Reading list fully designed in §7.2 (**FR-10**, was one-liner): per-user FIFO queue at `/reading`, row-menu toggle, mark-as-read, archive interaction; new **FR-23** (create-form notes) and **FR-24** (multi-collection at save, reusing the CL-8 picker) in §7.1; §5.2 gaps + §6.3 matrix updated to point at them |
| 1.5 | 2026-09-08 | FR-23 landed in §5.2 as **BK-12** — notes field on the add-bookmark form; FR-24 landed in §5.2 as **BK-13** — multi-collection picker at save (reuses the CL-8 picker); both removed from §7.1 |
| 1.6 | 2026-09-09 | FR-10 landed in §5.2 as **BK-14** — reading list (FIFO queue at `/reading`, `queued_at` on bookmarks, add-form checkbox + row-menu toggle, mark-as-read dequeue, archive auto-dequeues); **FR-25** added to §7.1; FR-10 removed from §7.2 |
| 1.7 | 2026-09-09 | **FR-4** landed in §5.2 as **BK-15** — cursor pagination (Load previous / Load more) for dashboard recent + filtered lists and collection page; BK-4 amended; FR-4 removed from §7.1 |
| 1.8 | 2026-09-09 | **BK-15** amended — dropped Load previous (append flow already holds newer rows); added Back-to-latest link for deep-linked windows; single-direction Load more only |
| 1.9 | 2026-09-09 | **BK-15** amended — htmx Load more no longer pushes the cursor into the URL (position is ephemeral; refresh returns to the list head); `?older` windows remain for the no-JS fallback with the Back-to-latest link |
| 1.10 | 2026-09-09 | **BK-15** clarified — documented the intentional htmx/no-JS discrepancy: cumulative append vs window-stepping (each full load renders only the batch older than the cursor) |
| 1.11 | 2026-09-24 | Documentation migration: feature-ticket terminology, category links, and status/priority reconciliation against the current contracts; no product behavior changes |

> Scope: this document describes **what** Patchworks is and does (features, roadmap). Area policy lives in the `docs/` pages; see [`docs/process.md`](process.md). The `AU-*`, `BK-*`, `CL-*`, `TG-*`, `SR-*`, `UX-*`, and `FR-*` identifiers are feature tickets, with detailed contracts in the linked category files.

---

## 1. Overview

Patchworks is a **personal bookmark manager** — a self-hostable web app for saving, organizing, and rediscovering links. It is designed for one person to own their bookmarks, with opt-in sharing through *collections* rather than a social platform.

Core ideas:

- **Personal-first.** The product is the owner's tool; multi-user behavior exists only where a user chooses to share a collection.
- **Fast, low-ceremony saving.** Add a bookmark from its URL, pick collections, tag it, and let the server derive the title. Done.
- **Rediscovery over archiving.** Recent links up top, search-as-you-type, one-click collection/tag filters. Bookmarks are for finding things again, not for hoarding.
- **Server-rendered by design.** Forms and links are intended to work without JavaScript, while htmx progressively enhances them. Current no-JS gaps are tracked by UX-1.

## 2. Goals

- Save a bookmark from its URL in a few seconds, optionally into collections and with tags.
- Find a saved bookmark again: free-text search, collection filter, tag filter, any combination.
- Organize bookmarks into named collections, optionally shared with other users.
- Keep the UI fast, accessible, and dependency-light; run anywhere Postgres runs.
- Stay a personal tool: predictable, boring, self-hostable.

**How we know it works (signals):** adding a bookmark takes < 5 seconds end-to-end; a saved bookmark is findable again via search *or* filter; collections render correctly for shared members; the app remains server-rendered, with remaining no-JS gaps tracked under UX-1.

## 3. Non-goals

- **Not a social network or public discovery platform.** No follower graphs, feeds, likes, or public profiles.
- **No content publishing.** We don't render, host, or redistribute saved pages; we store links and metadata.
- **Not a read-later/reader app (yet).** Saving and organizing are core; offline snapshots and readability are future work (FR-19), deliberately out of the initial scope.
- **No browser extension required** to use the core app (roadmap item FR-14, not a prerequisite).
- **No client-side rendering stack.** The app is server-rendered by design; a JS framework would be a rewrite, not an enhancement.
- **No bookmark detail page.** The bookmark list *is* the interface; editing happens on the row itself (→ BK-8).
- **No multi-tenancy or orgs** beyond simple shared collections.
- **No invasive analytics.** The only signals come from server logs.

## 4. Users and personas

| Persona | Description | Priority |
|---|---|---|
| **Owner** | The single primary user. Registers, saves bookmarks, organizes collections, curates tags. | Primary — everything below serves this persona |
| **Shared member** | An existing user added to one or more of the owner's collections. Can view/manage shared collections per their role. | Secondary — collections area only |
| **Anonymous visitor** | Not yet registered. Sees only the auth pages. | Tertiary |

<a id="5-feature-set-current"></a>
## 5. Feature set (current)

Each item below is a **feature ticket** with a compact status summary; the linked category file is the current contract and owns acceptance criteria and open work. See [`docs/process.md`](process.md) for status and priority definitions. Priorities apply only to open work: **P0** = critical, **P1** = important, **P2** = follow-up.

<a id="51-accounts--authentication--impl"></a>
### 5.1 Accounts & authentication — *partial*

[AU-* feature tickets](features/authentication.md) · **Family status:** partial

- **AU-1** *(partial · P1)* — Register with email/password and an Argon2id PHC hash; duplicate email is rejected. Email canonicalization and a password policy remain open.
- **AU-2** *(partial · P0)* — Verify credentials, record the successful login, and create a server-backed session. Unknown-user work equivalence and authentication throttling remain open.
- **AU-3** *(partial · P1)* — Use an AES-GCM encrypted, HTTP-only session cookie backed by database validity. Session rotation, revocation, and bounded expiry cleanup remain open.
- **AU-4** *(partial · P0)* — Delete the current session and clear its cookie. The current GET-based flow is not yet a CSRF-protected POST.
- **AU-5** *(implemented)* — Require authentication on application routes, redirecting full requests and challenging htmx requests.
- **AU-6** *(implemented)* — Reject empty credentials with accessible inline errors and preserve submitted values.
- **AU-7** *(implemented)* — Stamp `last_login_at` with database time only after successful credential verification and before session creation.

<a id="52-bookmarks--impl"></a>
### 5.2 Bookmarks — *partial*

[BK-* feature tickets](features/bookmarks.md) · **Family status:** partial

- **BK-1** *(implemented)* — Save a URL with optional tags, notes, reading-list state, and zero or more manageable collections.
- **BK-2** *(partial · P0)* — Derive a title with a bounded synchronous fetch and use the URL as fallback. SSRF-safe fetch/redirect policy and an optional title override remain open.
- **BK-3** *(partial · P0)* — Re-render URL parse failures inline without persistence; strict absolute HTTP(S) and host validation remain open.
- **BK-4** *(implemented)* — Show the newest non-archived bookmarks on the dashboard, initially ten, with pagination through BK-15.
- **BK-5** *(implemented)* — Return the updated list fragment for htmx saves and redirect plain saves to the dashboard.
- **BK-6** *(partial · P1)* — Persist bookmark identity, URL/title, author, timestamps, optional notes/queue state, and archive state. Safe handling of invalid persisted URLs remains open.
- **BK-7** *(implemented)* — Show optional notes under the row metadata with a native one-line expansion control.
- **BK-8** *(implemented)* — Let the author replace notes and tags from the row or a no-JS edit page; URL and title remain identity fields.
- **BK-9** *(partial · P0)* — Soft-archive an author-owned bookmark, hide it from browse results, and dequeue it. A no-JS form and confirmation remain open.
- **BK-10** *(partial · P0)* — List archived bookmarks and restore or permanently delete them. The archived-only delete guard and no-JS forms/confirmation remain open.
- **BK-11** *(implemented)* — Warn on an exact same-author URL duplicate and preserve all input through an explicit Save anyway override.
- **BK-12** *(implemented)* — Accept optional notes when saving and render them through the existing bookmark-note contract.
- **BK-13** *(implemented)* — Link one saved bookmark to every selected collection for which the author has owner/editor rights.
- **BK-14** *(partial · P1)* — Maintain an author-scoped oldest-first reading queue with enqueue, dequeue, and archive behavior. Atomic concurrent toggling remains open.
- **BK-15** *(partial · P1)* — Use newest-first cursor pagination on dashboard recent/filtered and collection lists. Distinct-ID paging across combined collection/tag filters remains open.

<a id="53-collections--sharing--impl--partial"></a>
### 5.3 Collections & sharing — *partial*

[CL-* feature tickets](features/collections.md) · **Family status:** partial

- **CL-1** *(implemented)* — Let a signed-in user create a named collection with an optional description; the creator becomes its owner.
- **CL-2** *(implemented)* — List the caller's collection memberships with bookmark and member counts.
- **CL-3** *(partial · P1)* — Let members browse a collection's newest non-archived bookmarks with pagination. Collection-page search/filtering remains open.
- **CL-4** *(implemented)* — Let owners/editors edit details and owners remove a collection and its links without deleting underlying bookmarks.
- **CL-5** *(partial · P1)* — Let owners directly add an existing registered user by email and role. Invitation semantics remain open.
- **CL-6** *(partial · P1)* — Let an owner remove a member while protecting the last owner; atomic concurrent removal remains open.
- **CL-7** *(implemented)* — Let every member view shared collection links while shared bookmarks retain their original author.
- **CL-8** *(partial · P1)* — Let an author or collection manager edit manageable membership links while preserving other links. Visibility and atomic authorization/replacement gaps remain open.
- **CL-9** *(partial · P1)* — Enforce owner/editor/viewer collection roles server-side, using 404 for non-members and 403 for insufficient roles. Full route-matrix coverage remains open.
- **Future:** Public or guest collection sharing remains [FR-15](features/roadmap.md#fr-15--collection-sharing-links).

<a id="54-tags--impl"></a>
### 5.4 Tags — *partial*

[TG-* feature tickets](features/tags.md) · **Family status:** partial

- **TG-1** *(partial · P1)* — Attach multiple comma-separated, non-empty tags; normalization and de-duplication remain open.
- **TG-2** *(implemented)* — Keep each author's tag namespace separate from other users' private tags.
- **TG-3** *(partial · P1)* — Show tag counts with the first 15 available and the rest disclosed. Deterministic ordering and archived-row exclusion remain open.
- **TG-4** *(partial · P1)* — Filter with multi-tag OR semantics alongside search and collection filters. Combined collection/tag results can still duplicate bookmark IDs.
- Tag changes happen per bookmark through [BK-8](features/bookmarks.md#bk-8--edit-notes-and-tags); tag-level management is not current scope.

<a id="55-search--filtering--impl"></a>
### 5.5 Search & filtering — *partial*

[SR-* feature tickets](features/search-filtering.md) · **Family status:** partial

- **SR-1** *(partial · P2)* — Search title and URL by case-insensitive substring. Literal wildcard escaping and query-length bounds remain open.
- **SR-2** *(partial · P0)* — Compose dashboard search, one collection, and multiple tags; forged collection IDs still need server-side authorization.
- **SR-3** *(partial · P2)* — Keep active filters in ordinary URLs; single-collection replacement and htmx search URL updates remain open.
- **SR-4** *(implemented)* — Clear all dashboard filters and return to recent-list mode.
- **SR-5** *(implemented)* — Debounce htmx search-as-you-type while retaining a full-GET fallback.

<a id="56-ui--interaction-model--impl"></a>
### 5.6 UI & interaction model — *partial*

[UX-* feature tickets](features/interaction.md) · **Family status:** partial

- **UX-1** *(partial · P0)* — Render gomponents HTML with htmx as progressive enhancement. No-JS destructive controls/confirmations, shared CSRF, and request-body bounds remain open.
- **UX-2** *(partial · P1)* — Keep the frontend buildless with plain CSS and no client framework. Version-pinned or self-hosted third-party assets and an explicit trust policy remain open.
- **UX-3** *(implemented)* — Render unexpected full-page and htmx errors with a shared, non-guessable correlation ID.
- **UX-4** *(partial · P1)* — Render expected validation errors inline with accessible associations and preserved safe values. Broader input limits and uniform field wiring remain open.
- **UX-5** *(implemented)* — Provide the authenticated navigation shell, relative timestamps, and email-derived identity cues.

<a id="6-access-control--permissions--impl"></a>
## 6. Access control & permissions — *normative matrix*

These cross-cutting rules are normative. Ticket status and open verification remain in the linked AU/BK/CL feature contracts.

### 6.1 Roles

- Every collection has members with exactly one role; its creator becomes **owner** (CL-1). Direct member additions default to **viewer** (CL-5); no pending-invitation flow is defined.
- Three fixed roles (Q4): **owner**, **editor**, **viewer**. A user's role is scoped per collection.
- The last owner of a collection cannot be removed.
- Roles are constrained at the database level to `owner`, `editor`, or `viewer`.

### 6.2 Collection actions (CL-9)

| Action | owner | editor | viewer | non-member |
|---|---|---|---|---|
| View the collection page & browse its bookmarks | ✅ | ✅ | ✅ | 404 |
| View the member list (read-only) | ✅ | ✅ | ✅ | 404 |
| Edit details — name/description (CL-4) | ✅ | ✅ | 403 | 404 |
| Remove the collection (CL-4) | ✅ | 403 | 403 | 404 |
| Add/remove members (CL-5, CL-6) | ✅ | 403 | 403 | 404 |

### 6.3 Bookmark actions

| Action | Who is allowed |
|---|---|
| Create a personal bookmark (no collection) | Any signed-in user, who becomes its author |
| Create a bookmark into collections | Owner/editor member of every selected collection (BK-13) |
| Change collection memberships (CL-8) | The author or an owner/editor member of a containing collection; each added or removed collection requires owner/editor rights, while unchanged memberships are preserved |
| View a bookmark inside a collection | Any member of that collection (CL-7) |
| Edit notes & tags (BK-8) | Author only |
| Archive / restore (BK-9, BK-10) | Author only |
| Permanently delete (BK-10) | Author only, and only while the bookmark is archived |
| Add to / remove from the reading list, mark as read (BK-14) | Author only |
| Load more (BK-15) | Author on dashboard lists; collection access on a collection page (CL-3) |

### 6.4 Error semantics

- A **non-member** or nonexistent target gets **404**, indistinguishable from a missing collection.
- An authenticated member with an insufficient role gets **403**.
- All checks run server-side on every request; hidden controls are never the authorization boundary.

## 7. Future developments

[FR-* roadmap feature tickets](features/roadmap.md) · **Family status:** planned

Phases are approximate. Detailed scope, dependencies, and open work live in [`docs/features/roadmap.md`](features/roadmap.md).

<a id="71-near-term--schemadesign-ready"></a>
### 7.1 Near-term

- **FR-2** *(planned · P1)* — OAuth/OIDC login with explicit external-identity linking and account-conflict policy.
- **FR-25** *(planned · P1)* — A per-user default for queueing newly saved bookmarks to the reading list.

### 7.2 Medium-term

- **FR-7** *(planned · P2)* — Full-text search and tag autocomplete.
- **FR-8** *(planned · P2)* — Browser import and collection export; the first import format remains Q6.
- **FR-9** *(planned · P2)* — Scheduled link-health checks and broken-link status.
- **FR-12** *(planned · P1)* — Email verification and password reset.

### 7.3 Longer-term / exploratory

- **FR-14** *(planned · P2)* — Browser extension or bookmarklet for capture.
- **FR-15** *(planned · P2)* — Secret, expiring, and public collection-sharing links.
- **FR-16** *(planned · P2)* — Keyboard shortcuts or a command palette.
- **FR-17** *(planned · P2)* — A versioned external API with explicit authentication and authorization.
- **FR-18** *(planned · P0)* — Complete release, runtime, migration-integrity, observability, and backup/restore operations beyond the current container/Compose baseline.
- **FR-19** *(planned · P2)* — Optional readable/offline page snapshots.
- **FR-20** *(planned · P2)* — A role-scoped activity history for shared collections.
- **FR-21** *(planned · P2)* — A responsive and touch-focused pass beyond current support tiers.

## 8. Open questions

| # | Question | Notes |
|---|----------|-------|
| Q6 | Import priority: which format first (Chrome/Firefox HTML, Safari, OPML)? | Parked — not exploring soon |

Resolved (2026-09-03, answers folded into the relevant feature ticket):

- Q1/Q3 → BK-8 / CL-8 — per-bookmark tag and collection-membership editing; no tag-level management.
- Q2 → BK-9 / BK-10 — archive is soft-delete: the bookmark leaves browse results but retains collection links; permanent deletion is archived-only.
- Q4 → CL-9 — three fixed roles.
- Q5 → FR-15 — secret, time-based, and public (pretty slug) share links.
- Q7 → §3 non-goals + BK-8 — the list is the interface; no detail page.
