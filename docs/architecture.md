# Architecture

## Directory Tree

```
cmd/server/                 # Entry point (minimal, wires components)
internal/
  ├─ core/                  # Domain types, interfaces, business logic
  ├─ auth/                  # Password auth service, private interfaces
  ├─ http/
  │  ├─ handlers/           # Local interfaces + pure handlers
  │  ├─ middleware/         # Auth, logging chi middleware
  │  ├─ views/              # Gomponents components, layouts, partials
  │  └─ server.go           # Chi router setup
  ├─ db/                    # SQLC generated code + repo wrappers
  ├─ config/                # Env-based config
  ├─ logging/               # slog logger setup
  └─ components/            # DI container (wired in main())

## Layers

- **core/**: Pure Go types, service interfaces, no framework deps
- **http/handlers/**: Define local interfaces for deps, implement pure handler funcs
- **http/middleware/**: Chi-compatible middleware (session → context, request logging, unhandled-error rendering)
- **http/views/**: Gomponents HTML components. `views.go` has Page(), AppShell(); topic files for specific pages
- **db/**: SQLC-generated query code in `db/sqlc/`, plus repository adapter in `db/` that wraps sqlc queries
- **components/**: One `New(ctx)` constructor that builds and wires all deps
- **internal/auth/**: `Service` struct with private interfaces (`sessionStore`, `userStore`)

## Handler Pattern

Handlers return errors; one wrapper renders the unhandled ones.

```go
// 1. Define interface in handler file
type UserService interface {
    GetByID(ctx context.Context, id uuid.UUID) (core.User, error)
}

// 2. Error-returning handler factory (Handler = func(w, r) error)
func handleGetUser(svc UserService) Handler {
    return func(w http.ResponseWriter, r *http.Request) error {
        user, err := svc.GetByID(r.Context(), ...)
        if err != nil {
            return fmt.Errorf("get user: %w", err) // unexpected → wrapper: 500 + error id
        }
        return views.UserPage(user).Render(w)
    }
}

// 3. In handler.go (router): Adapt wraps error-returning handlers
r.Get("/user", Adapt(handleGetUser(comp.UserService)))
```

- Expected errors (validation, wrong credentials, email taken, not found) are classified in the handler: re-render the form/page with `views.FormErrors` (submitted values preserved, field-level `FieldError`), or redirect — they never reach the wrapper.
- Unexpected errors are returned; `middleware.HandleError` logs them once with a fresh `uuid` error id (`error_id` attr) and renders a styled error page (full-page) or a toast (htmx, `HX-Retarget: #toast-container`).
- Core defines sentinels only (`internal/core/errors.go`, no framework deps); db/auth wrap underlying errors with them so `pgconn`/`pgx` types never cross boundaries. `classify()` in `middleware/errors.go` is the only sentinel → HTTP status mapping.

## Key Decisions

- No global state or init() — everything wired via components.New()
- Panic at startup for unrecoverable errors; handle gracefully at runtime
- Handlers are the only orchestration layer (deps, service calls, error classification, responses); unexpected errors bubble to one middleware wrapper
- Views check context for partial/full page flag
