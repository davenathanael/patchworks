# Plan: Bookmark row actions (notes, edit, collections, archive)

Goal: implement the settled row design (`docs/mockups/design-bookmark-actions.html`) one
piece at a time. Design decisions already approved: 3-item row menu everywhere
(Edit notes & tags · Edit collections · Archive), notes = 1-line clamp + native
More/Less expand, edit panel inline (notes + tags, title/domain immutable),
Edit-collections = preloaded search+checkbox popover, Archive = `hx-confirm` and
removes the row from the current list. Spec consolidation is done (spec.md v0.4,
FR-1/3/13/22 reworded). **The user commits each step themselves — leave every
step uncommitted and hand back for review.**

Order of implementation (each step is a committable unit):

1. **✅ Spec consolidation** (done): `docs/spec.md` v0.4 — FR-1/3/13/22 reworded
   with settled design, FR-13 + FR-22 moved to §6.1, Rev row 0.4.
   Mockup: `docs/mockups/design-bookmark-actions.html`. Plan: this file.

2. **✅ Notes column (FR-22, schema)** (done) — migration
   `20260904070446_bookmark_notes.sql` (`notes TEXT NOT NULL DEFAULT ''`, up/down);
   `core.Bookmark.Notes`; mapped in `internal/db/{mappings,collections}.go`;
   sqlc regen; integration test asserts notes round-trip (`mise run check` +
   `test-integration` green). Dev DB still needs `mise run migrate`.

3. **✅ Notes display + expand (LinkRow)** (done) — `views/links.go`
   `noteBlock` renders the note under domain & tags, `-webkit-line-clamp: 1` +
   native `details.note-toggle` with CSS `:has()` expand (no JS, no text
   duplication); `shouldShowNoteToggle` server-side overflow heuristic (>60
   chars, table-tested in `links_test.go`). CSS in `app.css` components layer.
   Spec: FR-22 landed → §5.2, Rev 0.5. `mise run check` green.

4. **✅ Edit notes & tags panel (FR-3 → BK-8)** (done) — "Edit notes & tags"
   menu item (first of the 3 — Edit collections + Archive join in steps 5–6);
   GET /bookmarks/{id}/edit (htmx inline panel / no-JS edit page),
   POST /bookmarks/{id}/edit (author-only; `UpdateBookmarkNotesTags` +
   `GetBookmarkByID`; tags replaced wholesale); GET /bookmarks/{id} = row
   fragment for Cancel. Shared `splitTags`. Spec: FR-3 edit half → §5.2 as
   BK-8, remainder (hard-delete) stays FR-3; Rev 0.6. `check` + integration green.

5. **✅ Edit collections popover (FR-13 → CL-8)** (done) — second menu item,
   preloaded search+checkbox picker (`popover="auto"`, anchored like the menu);
   POST /bookmarks/{id}/collections replaces the user's own membership links
   (shared links untouched); from a collection page, unchecking that collection
   drops the row (HX-Reswap: delete). Membership rides on `core.Bookmark.
   CollectionIDs` (batched `attachCollectionIDs` in the repo). Spec: FR-13 →
   §5.3 as CL-8; Rev 0.7. `check` + integration green.

6. **Archive (FR-1, action only)** — menu item Archive everywhere (dashboard +
   collection detail), `hx-confirm="Archive this bookmark? You can restore it
   later."`, htmx post `POST /bookmarks/{id}/archive` sets `archived_at`, row
   leaves the current list. Archived page (list + restore + hard-delete) is a
   separate later step.

7. **Archived page (FR-1, remainder)** — separate step, later: list archived
   bookmarks, restore, hard-delete; archive/filter interplay per spec.

## Risks / notes
- Notes migration touches all sqlc bookmark rows → mapping updates in
  `internal/db/{bookmarks,collections}.go` and `mappings.go`.
- `:has()` requires evergreen browsers (already allowed — css-guidelines.md).
- hx-confirm is client-side; server must still enforce author-only + idempotency.
- Row menu needs a title for the kebab (aria-label) — keep existing `details.menu`
  pattern from mockup.
- Each step: `mise run check` + `mise run test-integration` (needs compose).