# Testing Patterns

## Commands and test tiers

- **Unit/default:** `mise run test` runs `go test ./...` without Postgres. Unit
  and handler tests use local fakes and must remain runnable without a database.
- **Integration:** `mise run test-integration` runs
  `go test -tags integration ./internal/db/`. The file is guarded by
  `//go:build integration`, is excluded from the default command, and requires a
  reachable PostgreSQL instance (normally `mise run compose`). It uses
  `TEST_DATABASE_URL`, falling back to `DATABASE_URL`, and creates/drops the
  dedicated `patchwork_test` database.
- **Full local check:** `mise run check` runs lint and the default unit suite;
  it does not run the build-tagged integration suite. Release/CI coverage must
  invoke the integration command explicitly.

Use `github.com/carlmjohnson/be`, not testify. Keep table-driven tests for pure
functions; use ordinary subtests and local fakes for I/O and stateful behavior.
No mock framework is used. Tests live alongside implementation where practical.

## Current coverage and required boundaries

The database integration suite covers many repository invariants, including
session/user constraints, collection ownership, archive/restore behavior,
keyset pagination, tag attachment, and some role checks. It is not a substitute
for a composed HTTP security test, and it currently does not compare the
migration-applied schema with `resources/db/schema.sql`.

Handler tests commonly call handler functions with fakes. That is useful for
pure orchestration tests but does not prove middleware order, composed chi
routes, role behavior, or rendered no-JS/htmx responses. Security coverage must
exercise the composed router for anonymous, non-member, viewer, editor, and
owner requests, assert status and unchanged data, and cover both ordinary and
htmx requests.

The real-Postgres suite should additionally prove the invariants and races
called out in the feature contracts: archived-only deletion, concurrent
reading toggles, last-owner and collection-membership races, invalid persisted
URL handling, tag/filter count ordering, and distinct multi-tag pages. A
migration check should apply all seven migrations from a clean database and
compare the resulting schema with the canonical dump, rather than only testing
individual repository methods.

## Open work

- **P0 — Make the full check signal explicit:** ensure CI/release checks run
  lint, unit, and build-tagged integration tests and do not treat a skipped
  integration job as success. See [FR-18](features/roadmap.md#fr-18--self-host-polish).
- **P1 — Add route-level security matrix tests:** exercise the composed chi
  router across every collection and protected mutation route for anonymous,
  non-member, viewer, editor, and owner users. See
  [CL-9](features/collections.md#cl-9--enforce-collection-roles) and
  [SR-2](features/search-filtering.md#sr-2--combine-filters).
- **P1 — Add rendered, no-JS, and role contract tests:** assert full pages,
  htmx fragments, native form fallbacks, destructive confirmations, and
  role-dependent controls rather than relying only on handler fakes. See
  [UX-1](features/interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **P1 — Expand real-Postgres invariant and concurrency coverage:** include
  the race, archive/delete, invalid-URL, and tag/filter cases listed above. See
  [BK-15](features/bookmarks.md#bk-15--page-browse-lists) and
  [CL-6](features/collections.md#cl-6--remove-members).
- **P1 — Add migration/schema-drift checks:** run all migrations on a clean
  database and compare the resulting schema with the canonical dump, tracked
  with [FR-18](features/roadmap.md#fr-18--self-host-polish).
