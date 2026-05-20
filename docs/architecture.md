# Architecture

## Directory Tree

```
cmd/server/                 # Entry point (minimal, wires components)
internal/
  ├─ core/                  # Domain types, interfaces, business logic
  ├─ http/
  │  ├─ handlers/           # Local interfaces + pure handlers
  │  ├─ middleware/         # Auth, logging chi middleware
  │  ├─ views/              # Gomponents components, layouts, partials
  │  └─ server.go           # Chi router setup
  ├─ db/                    # SQLC generated code + repo wrappers
  ├─ config/                # Env-based config
  └─ components/            # DI container (wired in main())
pkg/
  └─ auth/oidc/             # OIDC provider integration
```

## Layers

- **core/**: Pure Go types, service interfaces, no framework deps
- **http/handlers/**: Define local interfaces for deps, implement pure handler funcs
- **http/middleware/**: Chi-compatible middleware (session → context, request logging)
- **http/views/**: Gomponents HTML components. `views.go` has Page(), AppShell(); topic files for specific pages
- **db/**: SQLC-generated query code in `db/sqlc/`, plus repository adapter in `db/` that wraps sqlc queries
- **components/**: One `New(ctx)` constructor that builds and wires all deps
- **pkg/auth/**: `Service` struct with private interfaces (`sessionStore`, `userStore`, `oidcClient`)

## Handler Pattern

```go
// 1. Define interface in handler file
type UserService interface {
    GetByID(ctx context.Context, id uuid.UUID) (core.User, error)
}

// 2. Pure handler factory
func handleGetUser(svc UserService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user, err := svc.GetByID(r.Context(), ...)
        if err != nil { http.Error(...); return }
        views.UserPage(user).Render(w)
    }
}

// 3. In handler.go (router), pass concrete dep from components
r.Get("/user", handleGetUser(comp.UserService))
```

## Key Decisions

- No global state or init() — everything wired via components.New()
- Panic at startup for unrecoverable errors; handle gracefully at runtime
- Views check context for partial/full page flag
