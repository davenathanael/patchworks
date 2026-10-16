# Database & Persistence

## Stack and workflow

- PostgreSQL with `pgx/v5`; SQLC generates query code; dbmate owns migrations.
- Edit `resources/db/queries/*.sql`, run `mise run sql`, then update the
  corresponding adapter in `internal/db/`. Never hand-write SQLC query structs.
- Create a migration with `mise run migration`, review its up/down sections,
  then apply it with `mise run migrate`. `mise run migrate-reset` is a
  destructive local reset.
- `mise run compose` starts the development Postgres; `mise run test-integration`
  runs the build-tagged database tests.

## Adapter and interface ownership

Persistence interfaces are consumer-owned, not defined in `core`:

- `internal/http/handlers/bookmarks.go` defines `BookmarkStore` and
  `bookmarkCollectionStore`.
- `internal/http/handlers/collections.go` defines `CollectionStore`.
- `internal/auth/auth.go` defines private `sessionStore` and `userStore`
  interfaces; `internal/http/middleware/auth.go` defines `SessionReader`.

`*internal/db.DB` implements the methods required by those consumers. The core
package remains pure domain data, roles, cursors, and sentinels. SQLC rows and
Postgres error types stop at the DB/auth boundary and are translated to domain
errors where appropriate.

## Invariants and transaction boundaries

Current adapters rely on SQL predicates for author, archived-state, and
role-scoped access, and on foreign-key cascades for dependent rows. Important
boundaries are:

- Creating a collection and adding its first owner is one transaction.
- Creating a bookmark, its collection links, and its tags is one transaction.
- Replacing bookmark notes/tags is one transaction.
- Replacing the caller's collection links is transactional, but the access
  check and allowed-collection snapshot happen before that transaction; a
  concurrent membership or role change can race the write.
- The reading-list toggle reads the row and then calls a separate update, so
  concurrent requests can choose the same branch.
- Last-owner removal performs an owner count/check and delete as separate
  operations; concurrent removals can both pass the guard.
- The permanent bookmark delete currently lacks an `archived_at IS NOT NULL`
  predicate.
- The collection-plus-tag query joins matching tag rows before limiting, so a
  bookmark matching several selected tags can consume multiple slots or repeat.
- Bookmark URL mapping currently ignores parse errors from persisted text.

These are implementation invariants to preserve or explicitly change; product
behavior remains in the feature tickets. See [BK-10](features/bookmarks.md#bk-10--manage-archived-bookmarks),
[BK-14](features/bookmarks.md#bk-14--maintain-a-reading-list),
[CL-6](features/collections.md#cl-6--remove-members),
[CL-8](features/collections.md#cl-8--edit-bookmark-collection-membership),
and [TG-4](features/tags.md#tg-4--filter-by-tags).

## Schema and migration checks

There are seven migration files under `resources/db/migrations/`, but
`resources/db/schema.sql` is a stale dump: its `schema_migrations` ledger lists
only the first three versions, and its table definitions do not reflect later
changes such as bookmark notes, queued state, and the collection-role check.
The application does not use the dump at startup. The integration harness
applies each migration's up section directly, so a stale dump can currently
escape detection.

Migration checks must therefore verify both a clean apply and the resulting
schema, not merely that generated Go compiles. Keep the dump and migration
history synchronized, and add a drift check before release. The current
migration and schema conventions are also summarized in [docs/process.md](process.md).
The password-auth migration also needs an explicit upgrade contract for rows
created by the earlier identity-based schema; its current down path cannot
reconstruct a safe legacy credential.

## Query-plan and index work

There is no query-plan test suite or index inventory. Capture representative
`EXPLAIN (ANALYZE, BUFFERS)` plans for recent, searched, collection, tag,
session, and membership queries before adding indexes. Any index must be
introduced through a migration, justified by the observed access path, and
checked against write cost and schema drift. This is related work for
[SR-1](features/search-filtering.md#sr-1--search-title-and-url) and
[FR-7](features/roadmap.md#fr-7--full-text-search-and-tag-autocomplete).

## Open work

- **P0 — Enforce archived-only permanent deletion:** add the repository
  predicate and real-Postgres/route coverage described by
  [BK-10](features/bookmarks.md#bk-10--manage-archived-bookmarks).
- **P1 — Close transaction and authorization races:** make last-owner removal,
  bookmark collection replacement, and reading toggles atomic under the
  invariants in [CL-6](features/collections.md#cl-6--remove-members),
  [CL-8](features/collections.md#cl-8--edit-bookmark-collection-membership), and
  [BK-14](features/bookmarks.md#bk-14--maintain-a-reading-list).
- **P1 — Repair schema drift checks:** regenerate `schema.sql` from the full
  migration history and make a clean-apply/dump comparison part of the release
  checks, tracked with [FR-18](features/roadmap.md#fr-18--self-host-polish).
- **P2 — Measure query plans and add justified indexes:** start with the
  representative queries above and record the plan and write-cost rationale.
- **P2 — Remove dead generated/stale implementation artifacts:** remove or
  consolidate unused SQL queries and generated outputs, and refresh stale
  comments or TODOs when their durable guidance has moved to an area doc.
