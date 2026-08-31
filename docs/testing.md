# Testing Patterns

## Approach

- Table-driven tests **only** for pure functions with many test cases
- Avoid table-driven for I/O or stateful tests

## Mocking Strategy

- Define local interfaces in handler/service files
- Implement concrete fakes/stubs for integration tests
- No mocking libraries (no testify/mock, gomock, etc.)

## Assertions

- `github.com/carlmjohnson/be` package (not testify)
- Example: `be.NilErr(t, err)`, `be.Equal(t, expected, actual)`

## Test Structure

- Test files alongside implementation: `bookmarks.go` → `bookmarks_test.go`
- Integration tests connect to test Postgres via docker-compose

## Integration Tests (`internal/db/integration_test.go`)

- Compile-gated with `//go:build integration` — skipped by default.
- Run: `mise run test-integration` (needs `mise run compose` up first).
- Self-contained: drops/recreates `patchwork_test` on the Postgres in `TEST_DATABASE_URL` (falls back to `DATABASE_URL`), applies migrations from `resources/db/migrations/`.
- Unit tests must always be skippable without a database; never require Postgres in `go test ./...`.
