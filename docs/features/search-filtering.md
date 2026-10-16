# Search & filtering (`SR-*`)

**Scope:** free-text search, collection/tag filter composition, URL state, clearing filters, and progressive search-as-you-type. Product intent is in [`docs/spec.md`](../spec.md#55-search--filtering--impl); UI conventions are in [`docs/frontend.md`](../frontend.md), with query/persistence details in [`docs/db.md`](../db.md).

## Index

| ID | Status | Priority | Contract |
|---|---|---|---|
| [SR-1](#sr-1--search-title-and-url) | partial | P2 | Bounded, literal case-insensitive title/URL search |
| [SR-2](#sr-2--combine-filters) | partial | P0 | Authorize and correctly combine collection, tags, and search |
| [SR-3](#sr-3--keep-filters-in-the-url) | partial | P2 | Keep ordinary and htmx filter state linkable |
| [SR-4](#sr-4--clear-filters) | implemented | — | Return to the unfiltered dashboard |
| [SR-5](#sr-5--search-as-you-type) | implemented | — | Debounce htmx list updates |

## SR-1 — Search title and URL

- **Status:** `partial`
- **Priority:** `P2`
- **Spec:** [`SR-1`](../spec.md#55-search--filtering--impl)
- **Intent/behavior:** Free-text search is a case-insensitive literal substring match over bookmark title or URL, bounded to a safe input length. Search is scoped to the authenticated user's non-archived bookmarks when used on the dashboard. The current unescaped `ILIKE '%' || @search || '%'` treats user `%`/`_` as wildcards and has no explicit length bound.
- **Authorization:** Search results do not cross the caller's bookmark visibility boundary. The query uses `ILIKE` over title and URL, not notes or tags.
- **UX/no-JS:** The search input is part of a GET form and works as a normal full-page request without JavaScript. htmx progressively replaces the list.
- **Failure/semantics:** An empty search is the unfiltered recent view. A query failure is unexpected and follows the shared error path.
- **Acceptance criteria:** A term matches title or URL regardless of case as literal text; SQL wildcard characters in user input are escaped; overlong input receives a bounded expected outcome; archived bookmarks do not appear; search does not mutate stored data.
- **Implementation notes:** [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), and handler/integration tests.
- **Related:** [SR-2](#sr-2--combine-filters), [SR-3](#sr-3--keep-filters-in-the-url), [SR-5](#sr-5--search-as-you-type), [BK-4](bookmarks.md#bk-4--show-recent-bookmarks).
- **Open work:**
  - **P2 — Escape and bound search input:** escape LIKE metacharacters (or use a parameter-safe equivalent), reject/cap excessive query length at the boundary, and add tests proving `%`/`_` are literal. Full-text ranking and tag autocomplete are [FR-7](roadmap.md#fr-7--full-text-search-and-tag-autocomplete).
  - **P2 — Measure query indexes:** capture representative `EXPLAIN (ANALYZE, BUFFERS)` plans for recent/filtered/collection/tag/session queries, then add only migration-backed indexes justified by access paths and write cost; verify generated schema remains drift-free.

## SR-2 — Combine filters

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`SR-2`](../spec.md#55-search--filtering--impl)
- **Intent/behavior:** The dashboard supports a single collection filter, a multi-tag filter, collection + tags, and free-text search composed with either filter shape. When any filter is active, the filtered list replaces the default recent list. Collection-page search/filtering remains an explicit extension tracked by CL-3/SR-2; it is not current behavior.
- **Authorization:** Tag and dashboard search queries are author-scoped. Collection filter choices are limited to manageable collections in the view, but a hand-edited `collection_id` currently reaches the bookmark query without a membership/role check. The collection-page target must authorize membership before filtering.
- **UX/no-JS:** Filter pills and the search form submit ordinary GET requests. The active state is reflected in the query string and list heading.
- **Failure/semantics:** An invalid collection UUID is treated as no collection filter. Empty tag values do not create a meaningful tag filter. Tag-only results de-duplicate multi-tag matches; the current collection + tag query does not, so BK-15/TG-4 track that correction. Archived rows are excluded by browse queries.
- **Acceptance criteria:** Search, collection, tags, and combined requests return the correct intersection/union semantics; no-filter mode shows recent bookmarks; filtered mode is distinct from recent mode; unknown, non-member, and insufficient-role collection IDs cannot disclose or mutate collection bookmarks. Collection-page filtering, when added, retains its route context and passes BK-15.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/http/views/filters.go`](../../internal/http/views/filters.go), and [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql).
- **Related:** [SR-1](#sr-1--search-title-and-url), [TG-4](tags.md#tg-4--filter-by-tags), [BK-15](bookmarks.md#bk-15--page-browse-lists), [CL-3](collections.md#cl-3--browse-a-collection), [CL-9](collections.md#cl-9--enforce-collection-roles).
- **Open work:**
  - **P0 — Authorize dashboard collection filters:** before any filtered bookmark query, require an allowed collection role and use non-disclosing 404 semantics for missing/non-member IDs; add route/handler tests with forged UUIDs for every role.

## SR-3 — Keep filters in the URL

- **Status:** `partial`
- **Priority:** `P2`
- **Spec:** [`SR-3`](../spec.md#55-search--filtering--impl)
- **Intent/behavior:** Active search, collection, and tag values are represented in query parameters so a filtered view can be bookmarked, copied, or reached without client state. Filter pill links preserve other values and toggle tags, but selecting a different collection currently appends another `collection_id`; htmx search-as-you-type does not update browser history.
- **Authorization:** URL state is input, not authority. The server still applies user and collection visibility checks.
- **UX/no-JS:** Plain links/forms carry the query string. htmx filter links use `hx-push-url`, but the search input does not; pagination preserves filters while replacing/removing cursor parameters.
- **Failure/semantics:** Stale/malformed cursor values do not prevent a list from rendering. Unknown query keys are ignored by the current load path.
- **Acceptance criteria:** Reloading/copying a filtered URL produces the same active filter state; selecting a collection replaces the prior single collection value; toggling one tag preserves the others; debounced htmx search leaves the address bar shareable; clearing removes filter state; paging does not lose filters.
- **Implementation notes:** [`internal/http/views/filters.go`](../../internal/http/views/filters.go), [`internal/core/cursor.go`](../../internal/core/cursor.go), and view/handler tests.
- **Related:** [SR-2](#sr-2--combine-filters), [SR-4](#sr-4--clear-filters), [BK-15](bookmarks.md#bk-15--page-browse-lists).
- **Open work:**
  - **P1 — Replace, do not append, collection state:** give the single-value collection key set/replace semantics (including a deliberate clear action) and test generated links plus hand-edited multi-value input.
  - **P2 — Push htmx search into URL state:** debounce and replace/push the current search into browser history without losing collection/tag filters; cover back/forward and reload behavior.

## SR-4 — Clear filters

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`SR-4`](../spec.md#55-search--filtering--impl)
- **Intent/behavior:** When any search, collection, or tag filter is active, the filter bar exposes Clear filters. It returns to `/` with no active filter, restoring the default recent-list mode.
- **Authorization:** Clearing filters does not change data or bypass collection authorization; it simply removes client/query state.
- **UX/no-JS:** Clear filters is a normal link to `/` and also has an htmx GET enhancement that updates the list and URL.
- **Failure/semantics:** With no filters, the control is not shown. A failed clear request is unexpected; the current URL remains a valid fallback.
- **Acceptance criteria:** Clear filters removes `search`, `collection_id`, and `tags` state; the dashboard returns to recent mode; no bookmark is modified.
- **Implementation notes:** [`internal/http/views/filters.go`](../../internal/http/views/filters.go) and handler/view tests.
- **Related:** [SR-2](#sr-2--combine-filters), [SR-3](#sr-3--keep-filters-in-the-url), [BK-4](bookmarks.md#bk-4--show-recent-bookmarks).
- **Open work:** None recorded.

## SR-5 — Search-as-you-type

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`SR-5`](../spec.md#55-search--filtering--impl)
- **Intent/behavior:** With htmx available, changes to the search input trigger a debounced request after 500 ms and replace the bookmark list/filter fragments. The server remains the source of query semantics and results.
- **Authorization:** The same dashboard/visibility rules apply to the partial request as to a full GET.
- **UX/no-JS:** Without JavaScript, submitting the form performs a full GET and the same search behavior remains available. The debounce is only an interaction enhancement.
- **Failure/semantics:** A failed partial request goes through the shared error response; the last valid page is not silently presented as a new successful result.
- **Acceptance criteria:** Typing does not issue a request on every immediate keystroke; after the delay the list reflects the entered term; disabling JavaScript still permits search through form submission.
- **Implementation notes:** [`internal/http/views/filters.go`](../../internal/http/views/filters.go), [`internal/http/views/home.go`](../../internal/http/views/home.go), and handler tests. htmx loading is documented in [`docs/frontend.md`](../frontend.md).
- **Related:** [SR-1](#sr-1--search-title-and-url), [SR-3](#sr-3--keep-filters-in-the-url), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:** None recorded.
