# Bookmarks (`BK-*`)

**Scope:** saving, displaying, editing, archiving, reading-list, duplicate handling, and paging bookmarks. Product intent is summarized in [`docs/spec.md`](../spec.md#52-bookmarks--impl); rendering policy is in [`docs/frontend.md`](../frontend.md) and [`docs/css-guidelines.md`](../css-guidelines.md); persistence is in [`docs/db.md`](../db.md).

## Index

| ID | Status | Priority | Contract |
|---|---|---|---|
| [BK-1](#bk-1--add-a-bookmark) | implemented | — | Add a bookmark from the dashboard |
| [BK-2](#bk-2--derive-a-title-when-needed) | partial | P0 | Safely derive an optional title with bounded fallback |
| [BK-3](#bk-3--reject-malformed-urls) | partial | P0 | Enforce an absolute HTTP(S) URL at save |
| [BK-4](#bk-4--show-recent-bookmarks) | implemented | — | Show the newest recent list |
| [BK-5](#bk-5--complete-a-save) | implemented | — | Redirect or return a partial after save |
| [BK-6](#bk-6--model-a-saved-bookmark) | partial | P1 | Persist identity, metadata, and recoverable URL state |
| [BK-7](#bk-7--show-bookmark-notes) | implemented | — | Display notes with native expansion |
| [BK-8](#bk-8--edit-notes-and-tags) | implemented | — | Edit notes/tags from the row |
| [BK-9](#bk-9--archive-a-bookmark) | partial | P0 | Soft-hide a bookmark with a no-JS fallback |
| [BK-10](#bk-10--manage-archived-bookmarks) | partial | P0 | Restore or permanently delete only archived rows |
| [BK-11](#bk-11--warn-about-duplicate-urls) | implemented | — | Soft duplicate warning with override |
| [BK-12](#bk-12--add-notes-at-save-time) | implemented | — | Store notes on the create form |
| [BK-13](#bk-13--select-multiple-collections-at-save) | implemented | — | Link a new bookmark to selected collections |
| [BK-14](#bk-14--maintain-a-reading-list) | partial | P1 | Atomic FIFO per-user reading queue |
| [BK-15](#bk-15--page-browse-lists) | partial | P1 | Correct keyset pagination across three list types |

## BK-1 — Add a bookmark

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-1`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** A signed-in user submits a URL from the dashboard form. The current form also accepts comma-separated tags, optional notes, a reading-list checkbox, and zero or more manageable collection IDs; duplicate selection is accepted and de-duplicated while preserving order.
- **Authorization:** Any signed-in user may create a personal bookmark. Each selected collection must grant that user owner/editor bookmark-management rights; the form only offers those collections, and the server rechecks every submitted ID.
- **UX/no-JS:** The form posts normally without JavaScript. htmx swaps the bookmark area and resets the form after success. The collection picker has a client-side search enhancement, but the complete checkbox list is rendered server-side.
- **Failure semantics:** A malformed URL is a field error. A selected collection that the user cannot manage fails the whole save with 403 and creates nothing. Invalid or unknown collection IDs are not silently accepted.
- **Acceptance criteria:** Valid input creates one author-owned bookmark with the validated URL text, tags/notes/queue state as submitted, and links to every distinct selected manageable collection; a no-collection save is valid; no collection write occurs when authorization fails.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), [`internal/http/views/links.go`](../../internal/http/views/links.go), and handler/integration tests.
- **Related:** [BK-3](#bk-3--reject-malformed-urls), [BK-11](#bk-11--warn-about-duplicate-urls), [BK-12](#bk-12--add-notes-at-save-time), [BK-13](#bk-13--select-multiple-collections-at-save), [BK-14](#bk-14--maintain-a-reading-list), [CL-5](collections.md#cl-5--invite-members-and-offer-manageable-collections), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:**
  - **P1 — Avoid post-commit false failures:** after a successful bookmark transaction, do not report a failed create solely because post-commit author/tag readback fails; return a successful persisted result or define safe enrichment behavior. Exact duplicate policy is owned by BK-11.

## BK-2 — Derive a title when needed

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`BK-2`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** The intended contract is that an omitted title is derived by fetching the page `<title>` at save time. The current create form has no title input, so every normal save invokes the title client. A successful 2xx response is read up to 1 MiB and a non-empty extracted title is stored; otherwise the URL string is used.
- **Authorization:** The fetch is performed while saving the authenticated user's bookmark; there is no separate user action or background worker.
- **UX/no-JS:** The fallback guarantees that a failed fetch does not prevent the save. The request is currently synchronous in the save handler, despite the spec wording that it “never blocks saving.”
- **Failure semantics:** Request errors, non-2xx responses, read errors, missing `<title>`, or empty title content fall back to the URL. The HTTP client has a 1.5-second timeout. The title extractor is deliberately simple and case-insensitive for locating tags; no HTML parser is involved.
- **Acceptance criteria:** A usable page title is stored when the bounded fetch succeeds; every fetch failure stores the URL fallback; saving remains successful when the remote site is unavailable. **Current gap:** the implementation does not yet distinguish an omitted title from a caller-supplied title because the form does not expose one, and the fetch is awaited before persistence.
- **Implementation notes:** [`internal/http/client/client.go`](../../internal/http/client/client.go), [`internal/http/client/title.go`](../../internal/http/client/title.go), [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), and [`internal/http/client/title_test.go`](../../internal/http/client/title_test.go). The historical plan explicitly flags title-fetch behavior for clarification.
- **Related:** [BK-1](#bk-1--add-a-bookmark), [BK-3](#bk-3--reject-malformed-urls), [UX-2](interaction.md#ux-2--keep-the-frontend-server-rendered-and-buildless).
- **Open work:**
  - **P0 — Make title fetching SSRF-safe:** permit only public HTTP(S) destinations; resolve and reject loopback, private, link-local, multicast, unspecified, and reserved addresses; re-check every redirect hop; disable unsafe transport negotiation as needed; and retain strict connect/response/byte limits.
  - **P1 — Add an optional title override:** when supplied, store a bounded user title without a remote fetch; when omitted, use the bounded derivation/fallback. Keep the no-JS form and specify whether derivation is synchronous or refreshed after save.
  - **P1 — Constrain redirect boundaries:** revalidate every remote title-fetch redirect under the SSRF policy, and allow only approved same-origin return paths for save/edit flows; reject absolute, protocol-relative, encoded, backslash, and malformed targets.
  - **P1 — Clarify non-blocking intent:** if fetch latency must not delay persistence, save the URL first and define an owned refresh mechanism; otherwise document the bounded synchronous behavior accurately.

## BK-3 — Reject malformed URLs

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`BK-3`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** The save boundary must accept only an absolute HTTP(S) bookmark URL with a valid host before duplicate lookup, remote title fetching, or persistence. The current code calls `url.Parse` but accepts parseable relative URLs and non-web schemes. A validation error renders the add form again with an inline URL error.
- **Authorization:** This is validation at the authenticated save boundary; no database or remote request occurs for a parse failure.
- **UX/no-JS:** Submitted URL, tags, notes, and collection selections are preserved in the re-rendered form. htmx retargets the response to `#add-bookmark-form`; plain requests render the full dashboard.
- **Failure semantics:** A malformed or disallowed URL is an expected 200 form response, not a 500. The error text is “that doesn't look like a valid URL.” Hostless/relative values, credentials where disallowed, and schemes other than HTTP(S) follow the same boundary.
- **Acceptance criteria:** Relative, hostless, malformed, and non-HTTP(S) URLs create no bookmark, produce an inline error, and preserve submitted form state; a valid absolute HTTP(S) URL proceeds to duplicate checking and save.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/http/views/forms.go`](../../internal/http/views/forms.go), and handler tests.
- **Related:** [BK-1](#bk-1--add-a-bookmark), [BK-2](#bk-2--derive-a-title-when-needed), [UX-4](interaction.md#ux-4--show-inline-validation-errors).
- **Open work:**
  - **P0 — Enforce strict URL syntax and scheme:** after parsing, require `http`/`https`, a non-empty host, and the chosen userinfo policy before any outbound or database work; add table-driven handler tests for the rejected classes.

## BK-4 — Show recent bookmarks

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-4`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** With no active search or filter, the dashboard shows the user's newest bookmarks, initially at most 10, ordered by `created_at DESC, id DESC`. A conditional Load more control extends the list through BK-15.
- **Authorization:** The query is scoped to the authenticated author's non-archived bookmarks.
- **UX/no-JS:** The initial page is server-rendered. htmx appends older rows without changing the URL; the no-JS fallback provides plain links and Back to latest.
- **Failure semantics:** Store failures are unexpected and use the shared error page/toast; an empty result renders the normal empty-state message.
- **Acceptance criteria:** The newest bookmark is first; archived rows never appear; the initial page is at most 10; older rows can be loaded without a separate “Your Bookmarks” list.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), [`internal/http/views/home.go`](../../internal/http/views/home.go), and tests.
- **Related:** [BK-15](#bk-15--page-browse-lists), [SR-1](search-filtering.md#sr-1--search-title-and-url), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:** None recorded.

## BK-5 — Complete a save

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-5`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** After a successful htmx create, the response re-renders the bookmark list (and filter-pill fragment) for the current user. A non-htmx create redirects to `/` with 303.
- **Authorization:** The response is built for the authenticated user and only the manageable collection subset is offered in the form.
- **UX/no-JS:** Plain form submission is a complete fallback; htmx avoids a full-page navigation and resets the form after a successful request.
- **Failure semantics:** Expected validation and authorization failures are handled before the success response; persistence failures remain unexpected errors.
- **Acceptance criteria:** A successful plain save lands on the dashboard; a successful htmx save returns a partial list and no redirect; a failed save does not present a success list or clear user input unexpectedly.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go) and [`internal/http/views/home.go`](../../internal/http/views/home.go).
- **Related:** [BK-1](#bk-1--add-a-bookmark), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement), [UX-3](interaction.md#ux-3--render-unexpected-errors-safely).
- **Open work:** None recorded.

## BK-6 — Model a saved bookmark

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`BK-6`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** A bookmark has a UUID, URL, title, author, creation/update timestamps, and nullable `archived_at`; current persistence also carries notes and `queued_at`. Archived state is a soft-delete marker used by BK-9/BK-10, not removal from storage.
- **Authorization:** The author is assigned at creation. Browse and owner-only operations scope by author; collection membership can make a bookmark visible to authorized collection members without changing its author.
- **UX/no-JS:** The model is rendered through the shared bookmark row; no detail page is introduced.
- **Failure semantics:** Database mapping errors are unexpected; missing or unauthorized owner fetches map to not found where the boundary contract requires concealment.
- **Acceptance criteria:** Saved data retains URL/title/author/timestamps; archive state is nullable and survives until restore or hard delete; notes and queue fields do not change bookmark identity. Repository/view mapping must surface or safely quarantine an invalid legacy URL rather than dereference a nil parse result.
- **Implementation notes:** [`internal/core/bookmarks.go`](../../internal/core/bookmarks.go), [`resources/db/migrations/20260510095913_create_bookmarks_collections.sql`](../../resources/db/migrations/20260510095913_create_bookmarks_collections.sql), [`resources/db/migrations/20260904070446_bookmark_notes.sql`](../../resources/db/migrations/20260904070446_bookmark_notes.sql), and [`resources/db/migrations/20260909072553_bookmarks_queued_at.sql`](../../resources/db/migrations/20260909072553_bookmarks_queued_at.sql).
- **Related:** [BK-7](#bk-7--show-bookmark-notes), [BK-9](#bk-9--archive-a-bookmark), [BK-10](#bk-10--manage-archived-bookmarks), [BK-14](#bk-14--maintain-a-reading-list), [CL-7](collections.md#cl-7--view-shared-bookmarks).
- **Open work:**
  - **P1 — Handle invalid persisted URLs:** stop ignoring URL parse errors in repository mappings; define a safe user-facing fallback or quarantined row, add a migration/repair check for existing invalid values, and test dashboard plus collection rendering.

## BK-7 — Show bookmark notes

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-7`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** Optional free text is shown under the domain and tags in each bookmark row. The text is clamped to one line; a native `<details>` More/Less affordance appears when the server-side length heuristic predicts overflow. CSS `:has()` expands the note without JavaScript or duplicating the text in the DOM.
- **Authorization:** Notes follow bookmark visibility; editing is author-only under BK-8.
- **UX/no-JS:** The native disclosure works without JavaScript. Empty notes render no note block; the toggle is not rendered for short notes.
- **Failure semantics:** A missing/invalid note is simply an empty optional value; rendering failures are unexpected.
- **Acceptance criteria:** A stored note appears in the row, is initially clamped, and can expand/collapse natively; empty notes add no visible block; the note is not duplicated for accessibility.
- **Implementation notes:** [`internal/http/views/links.go`](../../internal/http/views/links.go), [`resources/static/css/app.css`](../../resources/static/css/app.css), [`docs/css-guidelines.md`](../css-guidelines.md), and [`internal/http/views/links_test.go`](../../internal/http/views/links_test.go). Design evidence: [`docs/mockups/design-bookmark-actions.html`](../mockups/design-bookmark-actions.html).
- **Related:** [BK-8](#bk-8--edit-notes-and-tags), [BK-12](#bk-12--add-notes-at-save-time), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:** None recorded. The length threshold is a documented heuristic, not a claim of exact overflow measurement.

## BK-8 — Edit notes and tags

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-8`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** The row menu's “Edit notes & tags” action replaces the row with a panel containing a note textarea and comma-separated tag input. A save replaces the author's note and tag set. Title and URL/domain remain immutable identity fields.
- **Authorization:** Only the bookmark author may open or save the edit panel. A missing/non-author bookmark maps to 404.
- **UX/no-JS:** htmx swaps the row in place and returns the updated row; a plain GET renders a full edit page and a plain POST redirects home. Cancel returns to the dashboard in the no-JS page and to the row fragment with htmx.
- **Failure semantics:** Malformed IDs and non-author/missing rows are not found. Unexpected decode/update/render failures use the shared error response.
- **Acceptance criteria:** The author can add, change, or clear notes and tags; the updated row reflects the new values; another user cannot edit the bookmark; title/domain cannot be changed through this path.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), [`internal/http/views/links.go`](../../internal/http/views/links.go), and handler/repository tests.
- **Related:** [BK-7](#bk-7--show-bookmark-notes), [TG-1](tags.md#tg-1--attach-free-form-tags), [TG-4](tags.md#tg-4--filter-by-tags), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:** None recorded.

## BK-9 — Archive a bookmark

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`BK-9`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** The row menu offers Archive. Archiving sets `archived_at` and clears `queued_at`, then the row leaves the current list. All browse queries exclude archived rows. The current view has only an htmx button with `hx-confirm`, not a real no-JS form/confirmation path.
- **Authorization:** The author only may archive a bookmark. The action is available from dashboard and collection-detail rows for the author's own bookmark.
- **UX/no-JS:** htmx deletes the row after a successful empty response. Although the handler supports a plain POST redirect, the current view exposes no form without JavaScript; `hx-confirm` also provides no no-JS confirmation.
- **Failure semantics:** Missing or non-author rows map to 404. Repeating archive is idempotent at the repository boundary and refreshes the archive timestamp.
- **Acceptance criteria:** Archiving removes the row from recent, search, tag, collection, and other browse results; the bookmark remains in storage and appears on `/archived`; a queued bookmark is removed from the reading list; archive is reachable and confirmable without JavaScript.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), and integration tests.
- **Related:** [BK-10](#bk-10--manage-archived-bookmarks), [BK-14](#bk-14--maintain-a-reading-list), [BK-15](#bk-15--page-browse-lists), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:**
  - **P0 — Add a real no-JS archive form:** the current row exposes only an htmx button/`hx-confirm`; render a POST fallback and an explicit server-rendered confirmation path as required by UX-1.

## BK-10 — Manage archived bookmarks

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`BK-10`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** `/archived` lists the authenticated user's archived bookmarks newest archive first, with tags and collection links attached. Restore clears `archived_at`; permanent Delete removes the bookmark and cascades its tag/collection links.
- **Authorization:** Both management actions are author-only. The archived page is not a shared-collection view.
- **UX/no-JS:** htmx removes the row after Restore or Delete. The handlers redirect plain POSTs, but the current view renders only htmx-only buttons and `hx-confirm`; there is no working no-JS control or server-rendered destructive confirmation.
- **Failure semantics:** Missing/non-author rows map to 404. A failed delete leaves the row available for retry; a failed restore does not claim success.
- **Acceptance criteria:** Archived rows are listed with tags; restore and delete work through real no-JS forms; permanent delete has a server-rendered confirmation, is irreversible, removes dependent links, and succeeds only while `archived_at IS NOT NULL`; no archived row is exposed to another user.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/http/views/links.go`](../../internal/http/views/links.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), and integration tests.
- **Related:** [BK-9](#bk-9--archive-a-bookmark), [BK-14](#bk-14--maintain-a-reading-list), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:**
  - **P0 — Enforce archived-only deletion:** add `archived_at IS NOT NULL` to the repository delete predicate and add integration/route tests proving a non-archived bookmark cannot be permanently deleted through the endpoint.
  - **P0 — Add no-JS restore/delete forms and delete confirmation:** render POST fallbacks and a server-rendered irreversible-delete confirmation; retain htmx as the enhancement.

## BK-11 — Warn about duplicate URLs

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-11`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** Before saving, the server checks for the most recent bookmark owned by the same author with the exact URL string, including archived copies. A match produces a soft inline reminder with the existing title, relative save time, and archived status. The warned form preserves all submitted values and offers Save anyway; the override marker skips the check for that submission.
- **Authorization:** Duplicate lookup is author-scoped. A different user's identical URL is not a duplicate for this user.
- **UX/no-JS:** The reminder is server-rendered and works as a normal form re-submit; htmx retargets it to the add form.
- **Failure semantics:** A lookup/store error is unexpected. A near-match (including a different query string) is not considered a duplicate. No normalization is performed.
- **Acceptance criteria:** An exact same-author duplicate creates no bookmark until Save anyway; the reminder identifies the existing copy; Save anyway creates a second bookmark; submitted tags/notes/collections remain intact; non-duplicates save normally.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), [`internal/http/views/links.go`](../../internal/http/views/links.go), and handler/integration tests.
- **Related:** [BK-1](#bk-1--add-a-bookmark), [BK-3](#bk-3--reject-malformed-urls), [UX-4](interaction.md#ux-4--show-inline-validation-errors).
- **Open work:** None recorded.

## BK-12 — Add notes at save time

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-12`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** The create form has an optional notes textarea. Submitted text is stored with the bookmark and rendered by the same note block used for post-save edits. No new schema feature is required beyond the existing notes column.
- **Authorization:** Notes are part of the authenticated author's bookmark; later editing remains author-only.
- **UX/no-JS:** The field is part of the normal form and is preserved on validation/duplicate re-renders.
- **Failure semantics:** Empty notes are valid. Persistence failure follows the normal unexpected-error path.
- **Acceptance criteria:** Notes entered at save are available immediately in the row and later edit panel; clearing the field is supported; no notes field means no visible note block.
- **Implementation notes:** [`internal/http/views/links.go`](../../internal/http/views/links.go), [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), and create-form tests.
- **Related:** [BK-1](#bk-1--add-a-bookmark), [BK-7](#bk-7--show-bookmark-notes), [BK-8](#bk-8--edit-notes-and-tags).
- **Open work:** None recorded.

## BK-13 — Select multiple collections at save

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`BK-13`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** The create form reuses the collection picker pattern: a search field and checkbox list of collections the user can manage, with no collection selected for a fresh bookmark. One save links the new bookmark to all checked collections.
- **Authorization:** Every selected collection requires owner/editor permission. A tampered ID or a collection with only viewer permission fails the entire save with 403; no bookmark or partial link is created.
- **UX/no-JS:** The complete list is server-rendered and checkboxes submit normally. The search field is a client-side enhancement only.
- **Failure semantics:** Duplicate IDs are de-duplicated while order is retained. Invalid IDs are rejected; storage errors are unexpected.
- **Acceptance criteria:** Selecting multiple authorized collections creates one bookmark and all selected links; selecting none creates a personal bookmark; an unauthorized selection creates nothing; the picker never offers viewer-only collections.
- **Implementation notes:** [`internal/http/views/links.go`](../../internal/http/views/links.go), [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), and tests.
- **Related:** [BK-1](#bk-1--add-a-bookmark), [CL-5](collections.md#cl-5--invite-members-and-offer-manageable-collections), [CL-8](collections.md#cl-8--edit-bookmark-collection-membership), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:** None recorded.

## BK-14 — Maintain a reading list

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`BK-14`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** Each author has a FIFO queue at `/reading`, ordered by `queued_at ASC`. A user can enqueue from the create-form checkbox or the row-menu toggle, and can remove/dequeue from `/reading` with Mark as read. Queue state is independent of collection membership; archived bookmarks are excluded and archiving clears the queue timestamp. Restoring does not re-enqueue.
- **Authorization:** Only the author can view or mutate their queue. Shared collection membership does not grant reading-list access.
- **UX/no-JS:** The add checkbox and row form work as normal POSTs. `/reading` uses htmx to remove a marked row; without JavaScript, the form's validated same-site `next` redirect returns to the reading page (or `/` by default).
- **Failure semantics:** Archived, missing, or non-author rows cannot be enqueued/dequeued through the guarded repository; malformed IDs are not found. The handler currently reads queue state and then calls a separate update, so concurrent toggles can both choose the same branch.
- **Acceptance criteria:** Queue order is oldest-enqueued first; one logical toggle has one atomic effect under concurrent requests; Mark as read removes without creating history; archive auto-dequeues and restore does not requeue; the form checkbox is unchecked by default.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), [`internal/http/views/links.go`](../../internal/http/views/links.go), and tests.
- **Related:** [BK-1](#bk-1--add-a-bookmark), [BK-9](#bk-9--archive-a-bookmark), [BK-10](#bk-10--manage-archived-bookmarks), [FR-25](roadmap.md#fr-25--default-new-bookmarks-to-the-reading-list).
- **Open work:**
  - **P1 — Make the toggle atomic:** replace the read-then-enqueue/dequeue sequence with a single conditional statement or locked transaction, preserve archived guards, and add concurrent integration tests.
  - **P2 — Add a per-user default setting:** FR-25 is planned; until it exists, the add-form checkbox remains unchecked by default.

## BK-15 — Page browse lists

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`BK-15`](../spec.md#52-bookmarks--impl)
- **Intent/behavior:** Cursor/keyset pagination applies to the dashboard recent list, dashboard filtered list, and collection-detail bookmark list. Lists are newest-first with a stable `(created_at, id)` tie-break, page size 10 for recent and 20 for filtered/collection lists, and no offset/count or numbered pages. Collection + tag filtering currently joins tag rows before `LIMIT`, so a bookmark matching multiple selected tags can consume multiple page slots and appear more than once.
- **Authorization:** Dashboard pages are scoped to the authenticated author. Collection pages first require collection access; the collection query then returns that collection's non-archived bookmarks. Hidden UI is not the authorization boundary.
- **UX/no-JS:** A conditional Load more control appears only when older rows exist. htmx appends rows and out-of-band replaces the pager without changing the URL. Without JavaScript, `?older=` links render a single older window and Back to latest strips the cursor while preserving filters.
- **Failure semantics:** A malformed cursor degrades to the list head. Keyset predicates avoid gaps caused by concurrent archive/delete; a missing collection is handled by CL-9's 404 semantics.
- **Acceptance criteria:** No Load more appears when the page reaches the end; pages contain distinct bookmark IDs, are ordered and non-overlapping, and apply `LIMIT` after filtering/pagination rather than after tag-row expansion; equal timestamps use ID tie-break; htmx accumulates rows while no-JS steps windows; filter query values survive paging; Archived and Reading remain unbounded lists.
- **Implementation notes:** [`internal/core/cursor.go`](../../internal/core/cursor.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), [`internal/db/collections.go`](../../internal/db/collections.go), [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/http/views/links.go`](../../internal/http/views/links.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), and unit/integration tests. The intentional htmx/no-JS pagination difference is documented in the spec.
- **Related:** [BK-4](#bk-4--show-recent-bookmarks), [SR-2](search-filtering.md#sr-2--combine-filters), [SR-3](search-filtering.md#sr-3--keep-filters-in-the-url), [TG-4](tags.md#tg-4--filter-by-tags), [CL-3](collections.md#cl-3--browse-a-collection), [CL-9](collections.md#cl-9--enforce-collection-roles).
- **Open work:**
  - **P1 — Verify distinct-ID pagination after the TG-4 query fix:** cover multi-tag matches, exact page sizes, cursor continuity, and tag attachment so the shared query contract cannot regress.
  - **P1 — Expand database contract coverage:** add real-Postgres cases for archived-only delete, concurrent reading toggles, invalid persisted URL handling, last-owner/membership races, and all tag/filter count/order/dedup paths.
