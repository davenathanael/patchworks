# Architecture

## Runtime and wiring

```
cmd/server/main.go              # calls server.Run()
internal/components/components.go # config, DB, auth, and HTTP client wiring
internal/http/server/server.go # HTTP lifecycle and server construction
internal/http/handlers/         # route handlers and local dependency interfaces
internal/http/middleware/       # auth, request logging, and error rendering
internal/http/views/            # gomponents pages and fragments
internal/auth/                  # password/session service with private stores
internal/db/                    # SQLC-generated queries and DB adapters
internal/core/                  # pure domain types, roles, cursors, and sentinels
```

`cmd/server/main.go` is intentionally only an entry point. It calls
[`internal/http/server/server.go`](../internal/http/server/server.go)'s
`server.Run()`. `Run` calls `components.New`, constructs the chi handler with
`handlers.New`, and builds the `http.Server`; there is no `main`-level
`Adapt` step.

## Layers and ownership

- `core/` contains framework-free domain values and error sentinels. It does
  not own persistence interfaces.
- `auth/` orchestrates password, session, and cookie behavior behind its
  private `sessionStore` and `userStore` interfaces.
- `http/handlers/` owns consumer-side interfaces such as `BookmarkStore` and
  `CollectionStore`. It orchestrates authentication context, calls services or
  repositories, classifies expected failures, and chooses full-page or htmx
  rendering. It also owns the `Handler` function type, which implements
  `http.Handler` directly.
- `http/views/` renders gomponents nodes. Views do not perform authorization or
  persistence.
- `http/middleware/` provides session authentication, request logging, and the
  shared unexpected-error response.
- `components/` is the composition root. It loads configuration, opens and
  pings the database, creates the auth service, and exposes concrete adapters.
  There is no global state or `init()` wiring.
- `db/` wraps SQLC-generated query code. Query structs are generated; adapters
  translate SQLC rows to `core` values and wrap database errors.

## Handler boundary

Handlers are orchestration functions, not a second domain layer. They parse
and validate request input, resolve the authenticated user and resource access,
call the injected dependency, and render or redirect. Expected outcomes such as
field validation, invalid credentials, and duplicate email are handled in the
handler. Other expected sentinels (notably not-found and forbidden) can still
be returned to the shared classifier, so they may currently reach the error
logger; that boundary is tracked as open work. Unexpected dependency, decode,
or rendering errors are returned to the shared `middleware.HandleError` path.

The request logger and error logger are separate events. The request middleware
records request completion; `HandleError` records a separate error event with
an `error_id`, `request_id`, method, path, and error. The same error ID is
returned in `X-Error-Id` and shown in the full-page error or htmx toast. Do
not describe all request details as one consolidated log entry.

## Operational lifecycle

The current server sets read and write timeouts, but `Run` uses a background
context and has no signal handling or bounded `http.Server.Shutdown` path. It
panics on startup/listen failures, and the configured timeouts do not yet define
header, idle, database, remote-fetch, or shutdown budgets. Startup must also
close a newly opened database pool when readiness fails, and application
readiness must be observable rather than inferred from process liveness.

## Open work

- **P1 — Add graceful shutdown:** handle termination signals, stop accepting
  requests, bound `Server.Shutdown`, close dependencies in order, and make
  shutdown failures observable. See [FR-18](features/roadmap.md#fr-18--self-host-polish).
- **P1 — Define the end-to-end timeout policy:** cover read-header, read,
  write, idle, database operations, remote title fetches, and shutdown with
  intentional bounds and tests. See [FR-18](features/roadmap.md#fr-18--self-host-polish)
  and [BK-2](features/bookmarks.md#bk-2--derive-a-title-when-needed).
- **P1 — Keep expected errors out of the error event:** ensure not-found,
  forbidden, and other classified sentinels are rendered with their intended
  status without being treated as unexpected failures. See
  [docs/logging.md](logging.md) and [UX-3](features/interaction.md#ux-3--render-unexpected-errors-safely).
- **P1 — Make startup failures actionable:** replace panic-only lifecycle
  failures with structured, correlated diagnostics while preserving fail-fast
  behavior. See [docs/logging.md](logging.md) and [FR-18](features/roadmap.md#fr-18--self-host-polish).
- **P2 — Keep the execution root explicit:** stop serving static files from a
  hard-coded relative working directory; configure or embed the asset root so
  the binary behaves consistently outside the repository checkout.
- **P2 — Keep error context useful:** propagate the generated error ID into the
  request context used by callers, or remove the ineffective context accessor;
  do not expose an API that cannot return the documented value.
- **P2 — Decompose oversized handlers:** split the large bookmark/collection
  orchestration units behind their existing local interfaces without changing
  the handler and authorization boundaries.
