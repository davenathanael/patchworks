# Tags (`TG-*`)

**Scope:** free-form tag attachment, author-scoped tag vocabulary, filter-bar counts, and multi-tag matching. Product intent is in [`docs/spec.md`](../spec.md#54-tags--impl); tag and bookmark queries are covered by [`docs/db.md`](../db.md) and [`docs/frontend.md`](../frontend.md).

## Index

| ID | Status | Priority | Contract |
|---|---|---|---|
| [TG-1](#tg-1--attach-free-form-tags) | partial | P1 | Attach normalized, de-duplicated free-form tags |
| [TG-2](#tg-2--scope-tags-to-the-author) | implemented | — | Keep tag namespaces author-scoped |
| [TG-3](#tg-3--show-tag-counts) | partial | P1 | Show deterministic, non-archived tag choices and counts |
| [TG-4](#tg-4--filter-by-tags) | partial | P1 | Match any selected tag once across all filter shapes |

## TG-1 — Attach free-form tags

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`TG-1`](../spec.md#54-tags--impl)
- **Intent/behavior:** A bookmark may carry multiple tags entered as comma-separated text on the create form or edit panel. Input is split, trimmed, and empty entries are discarded; the remaining strings are stored as bookmark tags.
- **Authorization:** Tags are attached by the bookmark author at creation and may be replaced by that author through BK-8. There is no tag-level management screen.
- **UX/no-JS:** The field is a normal text input and works without JavaScript. The current create form and edit panel preserve the submitted tag text on validation re-render.
- **Failure/semantics:** Empty tags are valid and are not persisted. Input is trimmed, but repeated values remain and collide with the composite primary key during insert; storage failures are unexpected.
- **Acceptance criteria:** Multiple non-empty tags are attached to one bookmark; whitespace-only entries do not create tags; repeated inputs create one tag row without failing the whole create/edit; a bookmark can have zero tags; editing replaces the author's tag set.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), and [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql).
- **Related:** [BK-1](bookmarks.md#bk-1--add-a-bookmark), [BK-8](bookmarks.md#bk-8--edit-notes-and-tags), [TG-2](#tg-2--scope-tags-to-the-author), [TG-4](#tg-4--filter-by-tags).
- **Open work:**
  - **P1 — De-duplicate tag input:** define the exact/case normalization policy, apply it before insert on create and edit, and add tests proving repeated values do not trigger a constraint error.

## TG-2 — Scope tags to the author

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`TG-2`](../spec.md#54-tags--impl)
- **Intent/behavior:** Tag rows carry the bookmark author's identity, and the tag cloud/count query is scoped to the authenticated user. A user's tag namespace is their own; there is no global tag catalog or cross-author tag management.
- **Authorization:** Tag filters on the dashboard operate within the caller's bookmarks. Shared collection bookmarks retain their original author and tags; that does not merge the owner's tag namespace into another user's namespace.
- **UX/no-JS:** Tag pills are normal links generated from the current user's tag query.
- **Failure/semantics:** A missing tag row simply contributes no choice/count. Query errors are unexpected.
- **Acceptance criteria:** The tag cloud does not expose another user's private tag namespace; tags on a shared bookmark remain attached to that bookmark's author; counts are computed for the requested author.
- **Implementation notes:** [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), and [`docs/db.md`](../db.md).
- **Related:** [TG-1](#tg-1--attach-free-form-tags), [TG-3](#tg-3--show-tag-counts), [CL-7](collections.md#cl-7--view-shared-bookmarks).
- **Open work:** None recorded.

## TG-3 — Show tag counts

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`TG-3`](../spec.md#54-tags--impl)
- **Intent/behavior:** The dashboard filter bar shows tag pills with each tag's non-archived bookmark count, ordered deterministically (for example count descending, then tag ascending) before selecting the first 15; additional tags are placed under a native “See more” disclosure. The current query has no ordering and counts archived tag rows.
- **Authorization:** Counts are loaded for the authenticated user's tag namespace. The count query aggregates tag rows by author and tag but currently does not exclude archived bookmarks.
- **UX/no-JS:** Pills and the See more disclosure are server-rendered and work without JavaScript. htmx can refresh the filter-pill fragment after a save/filter request.
- **Failure/semantics:** An empty tag set renders no tag navigation. A count query failure is unexpected.
- **Acceptance criteria:** Each visible tag has a stable position and a count of non-archived bookmarks; the first 15 are directly available; remaining tags remain reachable through the disclosure; counts include neither archived rows nor another user's namespace.
- **Implementation notes:** [`internal/http/views/filters.go`](../../internal/http/views/filters.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), and [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go).
- **Related:** [TG-2](#tg-2--scope-tags-to-the-author), [TG-4](#tg-4--filter-by-tags), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:**
  - **P1 — Make top-tag ordering deterministic:** sort the aggregated list by the chosen count/name rule before slicing at 15, with a stable-order integration test.
  - **P1 — Exclude archived bookmarks from counts:** join tags to non-archived bookmarks or otherwise encode the same filter used by browse queries; test archive/restore count transitions.

## TG-4 — Filter by tags

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`TG-4`](../spec.md#54-tags--impl)
- **Intent/behavior:** The user can select one or more tags. A bookmark matches when it has any selected tag and must appear once. Tag-only filtering uses `EXISTS`, but the current collection + tag query joins matching tag rows before limiting, so a bookmark matching multiple selected tags can consume multiple page slots and appear repeatedly.
- **Authorization:** The tag-only query is scoped to the authenticated author. The target contract requires collection access to be checked before a collection-plus-tag query. The current dashboard path forwards a forged `collection_id` to the query without that check; the P0 correction is tracked by [SR-2](search-filtering.md#sr-2--combine-filters).
- **UX/no-JS:** Tag pills are links that toggle values in the query string, so filtering works by ordinary navigation. htmx updates the list and out-of-band filter pills.
- **Failure/semantics:** An empty selected-tag list means no tag filter. A bookmark with multiple matching tags appears once; archived rows are excluded by browse queries.
- **Acceptance criteria:** One selected tag returns matching bookmarks; multiple selected tags use OR semantics; each bookmark appears once in every shape; tag + search + collection combinations return complete, non-duplicated pages; the active filter is visible in the URL.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), and unit/integration tests. The tag-only path uses `EXISTS`; the combined collection + tag path still uses a joining query.
- **Related:** [TG-1](#tg-1--attach-free-form-tags), [TG-3](#tg-3--show-tag-counts), [SR-2](search-filtering.md#sr-2--combine-filters), [SR-3](search-filtering.md#sr-3--keep-filters-in-the-url), [BK-15](bookmarks.md#bk-15--page-browse-lists).
- **Open work:**
  - **P1 — De-duplicate combined collection + tag results:** use `EXISTS` or a distinct-ID CTE before the keyset limit, and add integration tests for a bookmark matching several selected tags. Tag autocomplete is future scope in [FR-7](roadmap.md#fr-7--full-text-search-and-tag-autocomplete).
