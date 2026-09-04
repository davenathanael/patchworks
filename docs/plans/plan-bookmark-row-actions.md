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

2. **Notes column (FR-22, schema)** — `dbmate new` migration:
   `ALTER TABLE bookmarks ADD COLUMN notes TEXT NOT NULL DEFAULT ''`.
   - `mise run sql` (sqlc regen), `core.Bookmark.Notes string` field,
     `internal/db/bookmarks.go` mapping (`toBookmarks` + GetCollection rows).
   - Verify: `mise run check`, integration test asserts notes round-trip.

3. **Notes display + expand (LinkRow)** — Variant A: clamp to 1 line, native
   `<details class="note-toggle">` + CSS `:has()` expand (no JS, no text
   duplication). Render the toggle only when the note overflows (server decides:
   render toggle iff note length implies > 1 line, or always render — collapse is
   harmless for short notes). Views: `views/links.go LinkRow`; CSS in `app.css`.
   - Spec: move FR-22 → §5.2 (impl), Rev bump.

4. **Edit notes & tags panel (FR-3 + FR-22)** — menu item opens inline panel
   (note textarea + comma-separated tags), title/domain read-only display.
   - New handler `PUT /bookmarks/{id}` (author-only), repo method
     `UpdateBookmarkNotesTags`, sqlc query.
   - htmx: swap row → panel → row; non-htmx fallback post + redirect.
   - Verify: handler tests + integration test (notes+tags persist).

5. **Edit collections popover (FR-13)** — search + checkbox list, **preloaded**
   server-side at page render with all user's own collections + current
   membership; Save = one htmx post (add/remove via existing
   `collection_bookmarks` table); `hx-confirm` not needed (Save/Cancel).
   - Views: popover fragment in LinkRow (needs current membership per bookmark —
     repo method `GetCollectionIDsByBookmark` or batch); handlers `POST
     /bookmarks/{id}/collections`.
   - From a collection page the current collection arrives pre-checked (uncheck =
     remove). Shared `LinkRow` needs membership data → view-model extension.

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