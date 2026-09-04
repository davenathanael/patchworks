# Patchwork — Product Specification

**Status:** Draft v1.4 · **Last updated:** 2026-09-04 · **Owner:** Dave Nathanael

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

> Scope: this document describes **what** Patchwork is and does (features, roadmap). Technical detail lives in the `docs/` pages (see `docs/process.md`). Requirement IDs (`AU-1`, `BK-1`, …) are referenceable from tickets and tests.

---

## 1. Overview

Patchwork is a **personal bookmark manager** — a self-hostable web app for saving, organizing, and rediscovering links. It is designed for one person to own their bookmarks, with opt-in sharing through *collections* rather than a social platform.

Core ideas:

- **Personal-first.** The product is the owner's tool; multi-user behavior exists only where a user chooses to share a collection.
- **Fast, low-ceremony saving.** Add a bookmark with URL + optional title, pick a collection, tag it. Done.
- **Rediscovery over archiving.** Recent links up top, search-as-you-type, one-click collection/tag filters. Bookmarks are for finding things again, not for hoarding.
- **Works without JavaScript.** Server-rendered HTML; htmx progressively enhances. No client-side build step, no JS framework.

## 2. Goals

- Save a bookmark (URL + title) in a few seconds, optionally into a collection and with tags.
- Find a saved bookmark again: free-text search, collection filter, tag filter, any combination.
- Organize bookmarks into named collections, optionally shared with other users.
- Keep the UI fast, accessible, and dependency-light; run anywhere Postgres runs.
- Stay a personal tool: predictable, boring, self-hostable.

**How we know it works (signals):** adding a bookmark takes < 5 seconds end-to-end; a saved bookmark is findable again via search *or* filter; collections render correctly for invited members; the app runs server-side-only (no JS required for any core flow).

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
| **Invited member** | A user invited into one or more of the owner's collections. Can view/manage shared collections per their role. | Secondary — collections area only |
| **Anonymous visitor** | Not yet registered. Sees only the auth pages. | Tertiary |

## 5. Feature set (current)

Status legend: **impl** = implemented, **partial** = partially implemented (gaps noted), **planned** = not yet built (see §7).

### 5.1 Accounts & authentication — *impl*

- **AU-1** Register with email + password; the password is hashed with argon2id (64 MiB, t=1, p=4) and stored as a PHC string. Duplicate email is rejected with an inline error.
- **AU-2** Login verifies credentials and issues a session stored server-side.
- **AU-3** The session cookie is AES-GCM encrypted, `HttpOnly`, `Secure`, `SameSite=Lax`, 30-day expiry. Invalid credentials produce a generic "invalid email or password" message (no account enumeration).
- **AU-4** Logout deletes the session and clears the cookie.
- **AU-5** Protected pages require a valid session: full-page requests redirect to `/auth/login`; htmx requests get `401` + `HX-Redirect`.
- **AU-6** Empty email/password are rejected with inline field errors; submitted values are preserved on re-render.
- **AU-7** A successful login stamps `users.last_login_at` (server time). Failed logins don't touch it; a failed stamp aborts the login before a session is created.
- **Gaps:** no email verification, no password reset (→ FR-12).

### 5.2 Bookmarks — *impl / partial*

- **BK-1** Add a bookmark from the dashboard: URL + optional title, optional collection, optional tags.
- **BK-2** When no title is given, the server fetches the page's `<title>` at save time. The fetch is bounded (1 MiB body read) and never blocks saving — on any failure the URL string is used as the title. (No scheduler: linking in the fetch, not a background job.)
- **BK-3** Malformed URLs are rejected with an inline field error and the form re-renders with submitted values preserved.
- **BK-4** The dashboard shows *recent bookmarks* (latest 10), newest first. Everything else is reached through search and filters (SR-1–SR-5).
- **BK-5** After adding, htmx requests re-render just the bookmark list; non-htmx requests redirect to the dashboard.
- **BK-6** Each saved bookmark has a title, URL, author, timestamps, and `archived_at` (nullable, *unused*).
- **BK-7** Bookmark notes: an optional free-text note per bookmark, shown under the domain & tags in the bookmark row, clamped to one line with a native More/Less expand (CSS `:has()`, no JS, no text duplication; the toggle appears only when the note likely overflows). Editing/clearing via BK-8. Design: `docs/mockups/design-bookmark-actions.html`. *(formerly FR-22 — see Rev 0.5)*
- **BK-8** *(formerly FR-3, edit half)* Edit a bookmark's notes and tags (author-only) from an inline row panel — one menu item ("Edit notes & tags") opens a textarea + comma-separated tags; htmx swaps the row to the panel and back, plain requests fall back to an edit page. Title and domain are immutable (the bookmark's identity). Design: `docs/mockups/design-bookmark-actions.html`.
- **BK-9** *(formerly FR-1, archive half)* Archive a bookmark from the row menu (dashboard and collection detail): native confirm (`hx-confirm`), one htmx post sets `archived_at`, the row leaves the current list. Archived bookmarks are hidden from recent/search/filtered/collection browse (`archived_at IS NULL` on all browse queries); the management side lives in BK-10.
- **BK-10** *(formerly FR-1, remainder; also closes FR-3's hard-delete half)* Archived page: `/archived` lists the user's archived bookmarks (newest archive first, tags attached) with inline **Restore** (clears `archived_at`, back into browse) and permanent **Delete** (author-only, own confirm, tags + collection links cascade).
- **BK-11** *(formerly FR-11)* Duplicate URL detection on save: an exact-URL bookmark by the same author (any archived state) triggers a soft inline reminder — existing title + when it was saved — with a **Save anyway** override (`save_anyway=true` marker on the warned re-render); submitted values are preserved. Exact string match, no URL normalization (a normalized column is a schema-ready follow-up if needed). Non-duplicate saves are unchanged.
- **Gaps:** The `page` query param is parsed but pagination rendering is a stub (→ FR-4). The add form takes a single collection and has no notes field; multi-collection membership is post-save via CL-8 (→ FR-23, FR-24).

### 5.3 Collections & sharing — *impl / partial*

- **CL-1** Create a collection with a name and description; validation errors re-render inline.
- **CL-2** List the user's collections with bookmark counts (dashboard side nav + `/collections` page).
- **CL-3** Open a collection to browse its bookmarks (with the same search/filter behavior as the dashboard).
- **CL-4** Edit a collection's name/description; delete a collection (cascades its bookmark links).
- **CL-5** Invite members by email with a role (default **viewer**); a new bookmark form offers only the user's own collections.
- **CL-6** Remove members from a collection.
- **CL-7** Members of a collection can view it and its bookmarks; shared bookmarks retain their original author.
- **CL-8** *(formerly FR-13)* "Edit collections" — add an existing bookmark to / remove it from the user's collections via an inline row panel: a search field plus a checkbox list of the user's collections with current membership pre-checked; one save replaces membership. Works from the dashboard and collection detail — from a collection page, unchecking that collection removes the row from the list. Only the user's own membership links are replaced; links into shared collections are never touched (no-JS fallback: a full edit page). Design: `docs/mockups/design-bookmark-actions.html`.
- **CL-9** *(formerly FR-6)* Enforce collection roles — every collection route checks the caller's role. Matrix: **owner** can view, add/remove bookmarks, edit details/title, remove the collection, and edit members (add/remove); **editor** can view, add/remove bookmarks, and edit details/title; **viewer** can view only. Non-members get **404** (indistinguishable from a missing collection); an insufficient role gets **403**. The member list (names + roles) stays visible read-only to all members. Full matrix: §6.
- **Gaps:** No public/guest sharing (→ FR-15).

### 5.4 Tags — *impl*

- **TG-1** Tags are free-form strings attached at bookmark creation; a bookmark can carry multiple tags.
- **TG-2** Tags are scoped per author (a user's tag namespace is their own).
- **TG-3** The filter bar shows a tag cloud with per-tag bookmark counts (top 15).
- **TG-4** Filter by one or more tags; a bookmark matches if it has any of the selected tags (results de-duplicated).
- **Gaps:** no tag-level management — tag changes happen per-bookmark (→ BK-8).

### 5.5 Search & filtering — *impl*

- **SR-1** Free-text search matches bookmark title **and** URL, case-insensitive substring (`ILIKE`).
- **SR-2** Filters: collection (single), tags (multi), and collection + tags combined; search composes with all of them.
- **SR-3** Active filters are expressed in the URL query string, so filtered views are shareable/bookmarkable.
- **SR-4** A "Clear filters" action returns to the unfiltered dashboard.
- **SR-5** Search-as-you-type: input changes trigger a debounced (500 ms) htmx re-render of the bookmark list.

### 5.6 UI & interaction model — *impl*

- **UX-1** All pages are server-rendered HTML (gomponents) and work without JavaScript; htmx adds partial swaps, out-of-band filter-pill refreshes, and error toasts.
- **UX-2** No client-side build step; styling is Open Props design tokens + plain CSS in `resources/static/css/app.css`.
- **UX-3** Unexpected errors render a styled error page with a non-guessable error ID (full page) or a dismissible toast (htmx); the same ID is logged server-side for correlation.
- **UX-4** Expected validation errors render inline under the offending field with `aria-invalid`/`aria-describedby`; submitted values are preserved across re-renders.
- **UX-5** Authenticated app shell: top nav + side nav listing collections; relative timestamps ("3h ago") in bookmark lists; avatar from email initials.

## 6. Access control & permissions — *impl*

Cross-cutting rules for who can do what. Requirement wording lives in the
`AU-*` / `BK-*` / `CL-*` items; this section is the consolidated, testable
matrix.

### 6.1 Roles

- Every collection has members with exactly one role; the collection's creator
  becomes **owner** (CL-4). Invites default to **viewer** (CL-5).
- Three fixed roles (Q4): **owner**, **editor**, **viewer**. A user's role is
  scoped per collection — the same user can be owner in one collection and
  viewer in another.
- The last owner of a collection cannot be removed.
- Roles are constrained at the database level (`role ∈ {owner, editor, viewer}`).

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
| Create a personal bookmark (no collection) | any signed-in user (becomes its author) |
| Create a bookmark into collections | member with **owner/editor** in each selected collection (single select today; multi-select → FR-24) |
| Change a bookmark's collection memberships (CL-8) | the bookmark's **author**, or any **owner/editor** member of a collection containing it; every collection **added or removed** requires owner/editor rights for the actor (else 403) — unchanged memberships pass, so a demoted member keeps existing links |
| View any bookmark inside a collection | any member of that collection (shared bookmarks keep their original author, CL-7) |
| Edit notes & tags (BK-8) | author only |
| Archive / restore (BK-9, BK-10) | author only |
| Hard-delete (BK-10) | author only |

### 6.4 Error semantics

- A **non-member** (or nonexistent target) gets **404** — the request is
  indistinguishable from a missing collection; no existence leak.
- An authenticated **member with an insufficient role** gets **403**.
- All checks run server-side on every request; hidden UI affordances (menu
  items, buttons, forms) match the rules but are never the boundary.

## 7. Future developments

Phases are approximate; items marked **(schema-ready)** have database or design work already in the repo and are the cheapest wins.

### 7.1 Near-term — schema/design-ready

- **FR-2** OAuth / OIDC login (Google/GitHub). Design documented in `docs/auth.md`: `users.password_hash` is already nullable; add a `user_identities` table + provider dance. *(schema-ready)*
- **FR-23** Notes on the add-bookmark form: optional free-text notes field at save time, stored and rendered exactly like post-save notes (BK-7/BK-8). No schema change.
- **FR-24** Multi-collection selection at save: replace the create form's single collection select with the CL-8 picker pattern — search field + checkbox list of manageable collections (§6.3: owner/editor per selected collection), nothing pre-checked (fresh bookmark); one save links all selected collections.
- **FR-4** Real pagination: `page` param is already plumbed; compute totals/pages server-side and replace the stub renderer.

### 7.2 Medium-term

- **FR-7** Full-text search (Postgres `tsvector` or `pg_trgm`), tag autocomplete in filters.
- **FR-8** Import from browser HTML export / OPML; export collections to JSON/HTML.
- **FR-9** Link health: periodic HEAD/GET checks, broken-link badges. *(needs a background job — new infra)*
- **FR-10** Reading list — a per-user FIFO queue of bookmarks to read, independent of collections:
  - Page **`/reading`** (side-nav entry) lists queued bookmarks **oldest-enqueued first**; each row keeps the usual link/menu plus a **Mark as read** action that dequeues it. Removal only — no read history.
  - Enqueue paths: an **"Add to reading list"** checkbox on the add-bookmark form (lands together with FR-23's form revision), and an **Add to / remove from reading list** row-menu toggle (like Archive) for existing bookmarks.
  - Detached from collections: a queued bookmark may belong to any number of collections or none; the queue is keyed to the author (bookmarks are author-scoped), so enqueue state can be a `queued_at` timestamp on the bookmark rather than a join table.
  - Archiving a queued bookmark auto-dequeues it; restoring does **not** re-enqueue.
  - Queue visibility follows bookmark visibility (own bookmarks only; no sharing).
- **FR-12** Email verification and password reset flows (prerequisite for serious OAuth/password UX).

### 7.3 Longer-term / exploratory

- **FR-14** Browser extension / bookmarklet for one-click save.
- **FR-15** Collection sharing links, three kinds: **secret** (tokenized path), **time-based** (secret link with expiry), **public** (pretty slug instead of a token path).
- **FR-16** Keyboard shortcuts or a command palette.
- **FR-17** External API for automation/integrations.
- **FR-18** Self-host polish: containerized app, backup/restore story, healthchecks.
- **FR-19** Readability/offline snapshot of saved pages ("save the article, not just the link").
- **FR-20** Activity/audit log for shared collections (who added/removed what).
- **FR-21** Mobile responsive pass beyond the current support tiers.

## 8. Open questions

| # | Question | Notes |
|---|----------|-------|
| Q6 | Import priority: which format first (Chrome/Firefox HTML, Safari, OPML)? | Parked — not exploring soon |

Resolved (2026-09-03, answers folded into the relevant requirement):

- Q1/Q3 → BK-8 — per-bookmark editing of tags + collection membership, from the bookmark row; no tag-level management.
- Q2 → FR-1 — archive hides from default lists, stays in collections; soft-delete; hard-delete only for archived bookmarks.
- Q4 → CL-9 — three fixed roles.
- Q5 → FR-15 — secret, time-based, and public (pretty slug) share links.
- Q7 → §3 non-goals + BK-8 — the list is the interface; no detail page.
