# Database & Persistence

## Stack

- pgx/v5 driver, SQLC for code generation
- dbmate for migrations

## Workflow

1. Write SQL query in `resources/db/queries/*.sql`
2. Run `mise run sql` → generates `internal/db/sqlc/*.sql.go`
3. Create/update repository wrapper in `internal/db/` (if needed)
4. Run `mise run migrate` to apply schema changes

## Repository Pattern

- Interfaces defined in `internal/core/` (domain layer)
- Implementations in `internal/db/` wrapping SQLC queries
- Example: core.UserRepository → db.userRepo using sqlc.Queries

## Conventions

- Never hand-write query structs — always use SQLC generation
- Domain ↔ DB model mapping in `internal/db/mappings.go`
- Transactions: SQLC's `Queries` accepts pgx.Tx for transactional operations

## Migrations

- `mise run migration` — create new migration file
- `mise run migrate` — apply pending migrations
- `mise run migrate-reset` — drop all and re-migrate
