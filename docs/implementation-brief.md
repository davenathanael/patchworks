# Implementation brief — Patchworks UI rebuild

**Status:** Ready for delegation · **Date:** 2026-10-17
**Supersedes:** design decisions in [ADR 0001](./adr/0001-styling-overhaul-mocha.md) — see [ADR 0002](./adr/0002-converged-ui-design-system.md)
**Design source:** [`docs/mockups/exploration/round4/`](./mockups/exploration/round4/index.html)

---

## 0. How to use this document

This brief is written to be read **cold**, with no access to the design conversation.
Everything you need to start is either here or linked from here.

- `docs/mockups/exploration/round4/app.html` — every screen, rendered
- `docs/mockups/exploration/round4/components.html` — every component in every state, with the live skin/theme switcher
- `docs/mockups/exploration/round4/index.html` — the decision summary
- `docs/mockups/exploration/round3/type.html` — why the type system is what it is
- `docs/mockups/exploration/reference/` — patterns kept from earlier rounds for reuse in *other* projects, **not** part of this build

Repo conventions in `AGENTS.md`, `docs/architecture.md`, `docs/css-guidelines.md`,
`docs/db.md`, and `docs/testing.md` are unchanged and still binding. This document
only covers what the design exploration decided.

---

## 1. Scope

This is a **full rewrite of `internal/http/views/`** against the round-4 design,
plus the schema work needed to support it. It is not an incremental restyle.

Three things are **out of scope** and must not be built:

- RSS / feeds (see §10)
- Public share pages (see §10)
- Bulk selection (see §10)

Everything else in the design is in scope.

---

## 2. Final design decisions

### 2.1 Information architecture

- **Mobile-first.** Desktop shows more of the same thing; it is never a different
  design. The primary list item is a **card** below `54rem` and a **row** at or
  above it. Same data, same component, two arrangements.
- **One list is the interface.** There is no bookmark detail page. Selecting an item
  opens an **inspection surface**: a bottom sheet on mobile (with a top-edge shadow,
  so it reads as floating above the list), a docked side pane at `54rem` and up.
- **The note is the point.** Notes are long-form prose — summaries, elaboration,
  explanations written for someone else. The list shows a **four-line excerpt**
  clamped, never a truncation, and never an expander (see §3.4). The full note is
  edited in the inspection sheet as a real `<textarea>`.
- **Reading queue** is a list, not a reader. The current item is **pinned to the top**
  and marked; the rest follow in insertion order. Titles open the original in a new tab.
  There is no in-app reader.
- **Navigation.** Five destinations: Inbox, Queue, Feeds, Collections, You.
  Mobile: a bottom tab bar with the quick-add composer pinned above it.
  Desktop: a top strip, plus list tabs inside each surface. **No sidebar.**
- **Account** lives at the fifth tab on mobile and behind the avatar in the top strip
  on desktop. Nothing is hidden behind an avatar-only menu on mobile.

### 2.2 Type system

Four roles, four faces. Loading from `fonts.bunny.net`, as today.

| Role | Face | Used for |
|---|---|---|
| Display | **Fraunces** | Page titles, section headings, brand, collection titles. **Never prose.** |
| Prose | **Source Serif 4** | Notes, descriptions, the note editor, share-page bodies |
| Chrome | **Geist** | Labels, buttons, nav, form controls |
| Metadata | **Geist Mono** (Sepia) / **JetBrains Mono** (Mocha) | Hosts, handles, dates, counts, operators, keys |

> Fraunces is a display face. Its loose fit and high optical contrast make it
> uncomfortable at 15px across four lines, which is exactly how notes are read.
> Do not set prose in it. The evidence is in
> [`round3/type.html`](./mockups/exploration/round3/type.html).

### 2.3 Colour, radius, depth

Two themes, each in light and dark:

- **Sepia** (default) — muted warm grey, low chroma
- **Mocha** — warmer brown, closer to the pre-rebuild palette

Defined as `oklch` + `light-dark()` token sets, per ADR-0001's discipline: the
palette is **not** mapped onto Open Props ramps.

- **Radius `2px`** globally (`0` is permitted per skin). Sharp.
- **Depth is borders plus a button shadow.** Only genuinely floating surfaces — the
  sheet, the dialog, toasts — get a real shadow. Nothing else is elevated.
- **Dark mode backgrounds sit at oklch L≈0.27**, not the usual 0.18–0.20.
  The old darker value forced text lighter to compensate and flattened everything.
  Measured contrast: body 13:1, secondary 8:1, muted metadata 5.5:1 — all WCAG AA.

### 2.4 Tokens required

Beyond what exists in `resources/static/css/app.css` today:

```
--text          prose face (Source Serif 4) — NEW
--on-accent     text colour on an accent fill, differs per theme
--danger        error
--warn          warning (feeds stale, partial failure) — NEW
--ok            success / queued state
--focus-ring    keyboard focus, FIXED BLUE in both themes — NEW
--focus-halo    field focus halo — NEW
```

`--focus-ring` is deliberately **not** the accent. It must read as "you are here"
across two palettes and both themes, and must stay legible on top of an accent fill.
A brown ring fails that. Do not "fix" it.

---

## 3. Resolved open questions

These eight were open at the end of round 4. All are now decided.

### 3.1 Note commits by debounced autosave

Save ~800ms after typing stops, plus on blur and on sheet close. Show a small
"Saving…" / "Saved" indicator in the sheet header. **No explicit Save button.**
Nothing to forget, nothing lost.

### 3.2 No filter sheet; sorting gets a control

Query syntax is the only way to filter — it does more and it is already built.
Sorting has no syntax equivalent, so it gets a **small control in the list header
bar**: newest, oldest, queue order, most tagged. No filter sheet.

### 3.3 Titles clamp to two lines

A long title clamps to 2 lines with an ellipsis on **all** widths, so every row keeps
a predictable height and the list rhythm survives. The full title is available in the
inspection sheet.

### 3.4 The note expander is gone

Remove "Show note ↓". The list shows the four-line excerpt; the full note is the
sheet's job. One route to the content, not two.

### 3.5 Bulk selection is a later phase

Not in this delivery. Recorded so it isn't lost. The design exists in
`docs/mockups/exploration/round2/split-sheet.html` as a reference for whoever picks it up.

### 3.6 The archived view gets full treatment

Same list component, different action set: **Restore** and **Delete**. Same states
(unread tint, note excerpt, tags) so the component is genuinely covered for every
state it can be in. Restore and Delete are mutually exclusive actions on a row.

### 3.7 Quick-add uses a native `<dialog>`, degrading in place

One `<dialog>` element: `showModal()` on mobile-presented full-screen, a centred
popover at desktop widths. It works without JavaScript — it just renders as a normal
block — so style it to read acceptably either way. **No separate no-JS route and no
polyfill.** The title/header lives *inside* the dialog; nothing floats above it.

### 3.8 Safe-area insets are required, not optional

The bottom tab bar and the pinned composer must use
`padding-block-end: calc(<padding> + env(safe-area-inset-bottom))`. This is a
requirement, not a follow-up.

---

## 4. Data model changes

Existing schema is in `resources/db/schema.sql`; queries in `resources/db/queries/`;
migrations in `resources/db/migrations/` using dbmate, and SQLC regenerates from
`resources/db/queries/` via `mise run sql`.

### 4.1 Collection title and handle

`collections.name` becomes the **display title**; add a separate **handle** for the
`@` syntax. They are independent and need not match — "Project A" / `@project-a`.

```sql
-- migrate:up
ALTER TABLE collections ADD COLUMN handle text;
UPDATE collections SET handle = regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g')
  WHERE handle IS NULL;
UPDATE collections SET handle = trim(both '-' FROM handle) WHERE handle = '';
ALTER TABLE collections ALTER COLUMN handle SET NOT NULL;
CREATE UNIQUE INDEX collections_handle_key ON collections (handle);

-- migrate:down
DROP INDEX collections_handle_key;
ALTER TABLE collections DROP COLUMN handle;
```

Handle rules: lowercase, `[a-z0-9-]`, must not start or end with `-`. Collision
resolution appends `-2`, `-3`, … and the UI must show the result before save.
`collections.name` keeps its existing unique constraint.

### 4.2 Feed read state is out of scope

No schema for feeds in this delivery. Do not add feed tables speculatively.
See §10.

### 4.3 No new bookmark columns

Everything in the design is expressible with the existing `bookmarks` table plus the
`queued_at` column already added by
`20260909072553_bookmarks_queued_at.sql`. **No migration required for bookmarks.**

---

## 5. `internal/http/views/` — full rewrite

The package is replaced. Existing files `links.go` (632 LOC, kebab popovers),
`views.go` (sidebar shell), `home.go`, and `filters.go` are **removed**, not evolved.
`collections.go` and `auth.go` are rewritten to the new visual language but keep
their existing handler contracts.

### 5.1 Target file layout

| File | Contents |
|---|---|
| `shell.go` | `AppShell`, `TopStrip`, `BottomTabs`, `SkipLink` |
| `bookmark_list.go` | `BookmarkList`, `BookmarkItem`, states: unread/read/queued/error |
| `inspection.go` | `InspectionSheet` — the record view + note editor |
| `quick_add.go` | `QuickAddDialog` — the `<dialog>` |
| `queue.go` | `ReadingQueue`, current-item-pinned variant |
| `collections.go` | `ListCollections`, `CollectionCard`, `CollectionDetail`, share & members |
| `archived.go` | `ArchivedList` — same item component, restore/delete actions |
| `search.go` | `SearchBand`, parsed query pills, operator legend |
| `account.go` | `AccountPage` — the You surface |
| `filters.go` | `SortControl` (replaces the old filter pills) |
| `components.go` | primitives: `Button`, `Pill`, `Tag`, `Badge`, `Field`, `Alert`, `Toast`, `Empty`, `Skeleton` |
| `states.go` | `Empty`, `Loading`, `ErrorState` |
| `forms.go` | existing `TextInput` helpers, extended with error state |
| `errors.go` | existing `FormErrors`, `Toast`, `ErrorPage`, extended |

Handler signatures in `internal/http/handlers/` are unchanged except where §6
adds routes. Handlers keep returning concretes; view funcs stay pure.

### 5.2 Component contract

Every component in `docs/mockups/exploration/round4/components.html` maps to a constructor in
`components.go` taking a value type — never a `map[string]any`. Status attributes
(`data-unread`, `data-sel`, `data-err`) drive state; there are no per-state
constructor variants.

---

## 6. Routes

Existing routes stay. Additions:

| Method | Path | Purpose | JS-free fallback |
|---|---|---|---|
| `POST` | `/bookmarks/{id}/note` | Autosave note body | form post, same page |
| `POST` | `/collections/{id}/handle` | Update handle | form post |
| `GET` | `/collections/{id}` | Collection detail (exists; reworked) | — |
| `GET` | `/archived` | Archived list (exists) | — |
| `GET` | `/archived/{id}/restore` | Restore | form post |

Sorting is a query parameter on the existing list routes: `?sort=newest|oldest|queued|tagged`.

---

## 7. Interaction specification

### 7.1 htmx vs plain form

- **Plain `<form>` POST** for everything that must work without JavaScript and has a
  sensible page to land on: quick-add save, archive, restore, delete, mark read,
  collection CRUD, share, members, note autosave.
- **htmx** for in-place refinements only: search-as-you-type, list row swaps, and
  the inspection sheet's open/close on desktop.
- Every destructive action has both a `DELETE`/`POST` pair, per UX-1.
- `POST` twins carry the shared CSRF token. **Shared CSRF is not yet implemented** —
  it is a prerequisite for this delivery, not an optional extra.

### 7.2 Note autosave

Debounced 800ms. Also fires on blur and on sheet close. The response is a partial
containing only the status indicator, swapped by htmx, and it must **not** re-render
the textarea — doing so would lose the caret position. On failure, show an inline
`.ferr` under the field plus a `.toast`; **never** a silent failure, and never a toast
alone.

### 7.3 Keyboard

No command palette, no `j`/`k` navigation — both were dropped as too steep for casual
use (FR-16 remains planned). What is required:

- Every interactive element is reachable by `Tab` and shows a `:focus-visible` ring.
- Dialogs trap focus, close on `Escape`, and restore focus to the invoker.
- Standard keys only: `Enter` submits, `Escape` cancels, `Tab` moves.

### 7.4 Long-press vs text selection on mobile

The item is tappable to open the sheet. Because notes are long prose, **long-press
must select text, not open the sheet** — use a click handler with a movement
threshold rather than a `touchstart` timer. Getting this wrong makes notes
unreadable on a phone, which is the single most likely mobile regression in this
build.

---

## 8. CSS migration plan

`resources/static/css/app.css` (1156 LOC) is **replaced**, not patched. The design
changes the palette, the type system, the radius, and the breakpoints; editing in
place would leave contradictory tokens behind.

What carries forward from the current file and ADR-0001:

- Cascade layers `reset, theme, base, layout, components, utilities, states`
- Every rule in a layer; native nesting
- No build step; Open Props from CDN for spacing/typography/shadows only, **never**
  for palette
- Tokenise rather than hardcode, with documented exceptions

What changes:

| | Before | After |
|---|---|---|
| Breakpoint | `46rem` | `54rem` |
| Radius | `4px` | `2px` |
| Type | Fraunces + Geist | + Source Serif 4 for prose, + a mono metadata face |
| Palette | single mocha set | Sepia + Mocha, each light and dark |
| Nav | sidebar shell | top strip + bottom tabs, no sidebar |
| New tokens | — | `--text --warn --ok --on-accent --focus-ring --focus-halo` |

Before writing any CSS, read `docs/css-guidelines.md`.

---

## 9. Accessibility requirements

These are acceptance criteria, not aspirations.

1. **Focus.** Every focusable element has a visible `:focus-visible` ring. A skip
   link is the first focusable element on every authenticated page.
2. **Never colour alone.** Every status carries a glyph **and** a word. Checked pills
   show `✓`, unselected `○`, failed `✕`, warning `!`, stale a dashed border. This must
   hold in greyscale.
3. **Contrast.** Text meets WCAG AA against its actual background in all four theme
   combinations. Measured values are in §2.3.
4. **Names.** Every control has an accessible name; icon-only buttons carry
   `aria-label`.
5. **Errors are wired.** Invalid fields set `aria-invalid` and `aria-describedby`
   pointing at the error node. The error node is announced.
6. **Dialogs** use `<dialog>` with `showModal()` for the focus trap; they are labelled
   by their heading.
7. **Reduced motion.** `@media (prefers-reduced-motion: reduce)` kills the skeleton
   shimmer and the toast slide.
8. **Live regions.** Toasts use `role="status"`; errors that block progress use
   `role="alert"`.

---

## 10. Deferred — explicitly not in this delivery

Do not build these. They are listed so an implementer doesn't "helpfully" add them.

- **RSS / feeds.** Needs `feeds`, `subscriptions`, `feed_items` tables, a background
  fetch job, per-item read state separate from bookmark state, auto-tagging, and an
  SSRF-safe fetcher (BK-2 is already a P0 gap for bookmarks). Large, and it must not
  be smuggled in as a view change.
- **Public share pages.** Design is complete in `docs/mockups/exploration/round4/app.html` §8 but needs an
  unauthenticated route, a token entity, and an expiry policy (FR-15). Out.
- **Bulk selection.** Later phase; reference in `docs/mockups/exploration/round2/split-sheet.html`.
- **Command palette / `j`·`k` navigation.** Dropped; FR-16 remains planned.
- **In-app reader.** Deferred; the queue is a list.
- **OAuth/OIDC, import/export, link health, activity log.** Unrelated roadmap items
  (FR-2, FR-8, FR-9, FR-20); untouched by this work.

---

## 11. Build order

Not obvious, and getting it wrong means rework. Schema first, styles last.

**Slice 1 — Foundations**
New token block in `app.css` (Sepia + Mocha × light/dark, new roles), cascade layers
preserved. `components.go` primitives: `Button`, `Pill`, `Tag`, `Badge`, `Field`.
Unit tests for the pure view funcs via `be`. *Nothing else touches the app yet.*

**Slice 2 — Shell**
`shell.go`: top strip, bottom tabs, skip link, safe-area insets. Router points at an
empty shell for every authenticated route. Verify focus order end to end.

**Slice 3 — Collection handle migration**
Migration, SQLC regenerate, slugify + collision logic as a pure function with
table-driven tests. Handle editing in the existing collection form.

**Slice 4 — Bookmark list**
`bookmark_list.go` at both widths, with unread/queued/error states. Four-line note
excerpt, two-line title clamp, `#hashtags`, sort control, cursor pagination. Replaces
`links.go` and `home.go`.

**Slice 5 — Inspection sheet**
`inspection.go` at both widths. Note editor with debounced autosave (§7.2). This is
where the caret-loss bug lives if autosave re-renders the field.

**Slice 6 — Quick add**
`quick_add.go` as a `<dialog>`, verified working with JS disabled (§3.7).
Title fetched server-side before save.

**Slice 7 — Queue, archived, collections, account**
`queue.go` with pinned current item; `archived.go` full treatment; `collections.go`
rewritten; `account.go`.

**Slice 8 — Search**
`search.go`: realtime parsing, parsed pills, operator legend on **both** widths, and
the sort control wired to real queries.

**Slice 9 — Error and empty states**
`states.go` across every surface, including the feed-error treatments from
`docs/mockups/exploration/round4/app.html` §11 as *patterns* — the components ship even though feeds don't.

Slices 1–4 are the risky ones. 5 and 6 have the most easily-missed details.

---

## 12. Testing expectations

Per `docs/testing.md`: `github.com/carlmjohnson/be`, local fakes, no mock libraries,
table-driven tests for pure functions only.

Must have tests:

- Slugify and collision resolution — table-driven, including unicode, leading/trailing
  hyphens, empty input, collisions.
- Query parsing (`tag:`, `is:`, `after:`, `-`, `OR`, quoted strings) — table-driven.
- Note autosave debounce logic.
- Every view component's state rendering — unread, queued, error, empty, long title,
  long note.

Not required: browser tests. There is no existing browser-test infrastructure and
introducing one is out of scope.

---

## 13. Definition of done

- Every screen in `docs/mockups/exploration/round4/app.html` is implemented, at both widths.
- Every component and state in `docs/mockups/exploration/round4/components.html` exists in
  `internal/http/views/components.go` or a sibling file.
- All four theme combinations render, verified by screenshot.
- All eight §3 accessibility requirements pass.
- Quick-add works with JavaScript disabled.
- Keyboard-only pass through the whole app with no trap and no invisible control.
- Greyscale pass: no state is distinguishable by colour alone.
- `mise run check` is green.