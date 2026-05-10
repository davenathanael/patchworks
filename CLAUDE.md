# Patchwork — Claude Code Project Guide

## 1. Project Overview
- **Language/Version**: Go 1.26.2
- **Web**: chi v5 + gomponents, fixi.js for interactivity
- **Database**: pgx/v5 + SQLC + repository pattern
- **Architecture**: DDD-inspired; pure domain core, interface-driven handlers, chi middleware
- **Rendering**: Server-side HTML with partial fragment support (fixi/HTMX patterns)

## 2. Code Organization

```
cmd/server/                 # Entry point only
internal/
  ├─ core/                  # Pure domain types, business logic, service interfaces
  ├─ http/
  │  ├─ handlers/           # Handler contracts (local interfaces for deps), pure handlers
  │  ├─ middleware/         # Chi middleware (auth, logging, etc.)
  │  ├─ views/              # Gomponents components, layouts, partials
  │  └─ server.go           # Chi setup
  ├─ db/                    # SQLC generated code + repository pattern layer
  ├─ config/                # Configuration loading (env-based)
  └─ components/            # Dependency injection container
pkg/
  └─ auth/oidc/             # OIDC/PocketID auth logic (interfaces for migration)
```

**Convention**: Handlers inject interfaces (not concrete types); all stateful calls abstracted. Dependencies resolved in `main()` via components package.

## 3. Code Style & Preferences

### Pure Functions & State Extraction
- **Prefer pure functions**: Calculations, transformations, data mapping should be pure
- **Extract state**: Move mutable/stateful logic outside pure logic (pass state as parameters)
- **Sans-IO**: Use where possible (usually rare); keep I/O concerns separate from business logic

### Function Design
- **Accept interfaces, return concretes**: Flexibility on input, clarity on output
- **Simple names**: Prefer clarity over cleverness; names should indicate intent
- **Minimal comments**: Only explain non-obvious decisions or constraints
- **Orthogonal pure functions**: When a function applies to multiple data types/classes (e.g., map, filter, reduce style):
  - Only if you foresee different types—don't abstract prematurely
  - If only one type, don't create generalization
  - Inspired by Clojure core functions (functional composition over inheritance)
- **No unnecessary abstractions**: One-off operations don't need helpers; trust internal code guarantees

### Handlers & HTTP Layer
- **Pure handlers**: All stateful/I/O calls abstracted as interfaces
- **Dependencies as interfaces**: Injected at handler construction; enables easy testing/mocking
- **Error handling at boundaries**: Validate user input; trust internal code contracts

## 4. Authentication & Sessions

### Flow
1. User redirected to PocketID
2. Callback handler receives auth code + user info
3. Session created in DB
4. Session ID stored in encrypted cookie

### Structure
- **pkg/auth/oidc/** — All OIDC logic (token exchange, user creation, session init)
  - Interfaces for clean future migration (e.g., to different OIDC provider or auth method)
  - Token validation, user extraction, session lifecycle
- **Middleware** — Session verification, user context injection
- **Note**: Error handling & session security (CSRF, SameSite, etc.) TBD as auth stabilizes

## 5. Views & Rendering

### Full Pages vs Partials
- **Full pages**: Initial HTML responses (nav, layout, all assets)
- **Partials**: HTML fragments for fixi.js/HTMX updates (reusable components, no layout)
- **Pattern**: Views check context flag; return full page or partial accordingly
- **Gomponents**: Reusable components in `internal/http/views/`; composition over inheritance

## 6. Frontend Stack

### Interactivity
- **fixi.js ecosystem**: No build step, progressive enhancement
- **Pattern**: Server renders HTML; fixi.js adds smooth interactivity via partial updates

### Styling Strategy
- **Design system**: OAT CSS (variables + layout primitives) + OpenProps (fallback values)
- **Constraint**: No Tailwind, no build steps, no arbitrary values (px/rem)
- **Priority order**:
  1. Use OAT utilities first (`.flex`, `.hstack`, `.gap-1`, `.mt-2`, etc.)
  2. Redefine OAT CSS variables in `app.css` if customization needed
  3. Use OpenProps variable values when OAT lacks coverage
  4. Last resort: custom CSS classes in `app.css`

#### OAT Utilities Available
- **Layout**: `.flex`, `.flex-col`, `.hstack` (horizontal stack), `.vstack` (vertical stack)
- **Alignment**: `.items-center`, `.justify-center`, `.justify-between`, `.justify-end`, `.align-left`, `.align-center`, `.align-right`
- **Spacing**: `.gap-1`, `.gap-2`, `.gap-4`, `.mt-2`, `.mt-4`, `.mt-6`, `.mb-2`, `.mb-4`, `.mb-6`, `.p-4`
- **Other**: `.unstyled` (removes list/link styles), `.text-light`, `.text-lighter`, `.w-100`

#### OAT CSS Variables (all available via `:root`)
- **Spacing**: `--space-1` through `--space-18` (0.25rem to 4.5rem)
- **Colors**: `--color-bg-*`, `--color-text-*`, `--color-border`, `--primary`, `--danger`, `--success`
- **Typography**: `--font-sans`, `--font-mono`, `--text-1` through `--text-8`
- **Layout**: `--border-radius`, `--radius-small` through `--radius-full`, `--shadow-*`
- **Animation**: `--transition-fast`, `--transition`

#### Custom Classes in app.css
Only add when no OAT utility covers the pattern. Current custom classes:
- `.link-title` — link truncation (overflow ellipsis)
- `.link-list` — link row container styling
- `.text-muted` — secondary text color/size (extends OAT color system)
- `.tag-badge` — compact tag styling
- `.filter-list` — sidebar filter list (flexbox + hover states)
- `.dashboard-section` — section spacing + heading typography

## 7. Testing Patterns

### Approach
- **Table-driven**: Use only for pure functions with many test cases; avoid for stateful/I/O tests
- **Mocking strategy**:
  - Define local interfaces in handlers/services
  - Implement concrete fakes/stubs for integration tests
  - Avoid mocking libraries
- **Assertions**: `earthboundkid/be` (not testify)

## 8. Database & Persistence

### SQLC
- Generate SQL query code (`*.sql` → `*.sql.go`); never hand-write query structs
- Regenerate with `mise run sql` after schema changes

### Repository Pattern
- Wrap SQLC queries in domain-driven repositories
- Interfaces defined in `internal/core/`; implementations in `internal/db/`
- Example: `UserRepository` interface (with `GetByID`, `Create`, etc.) implemented in `db/users.go`

### Migrations
- dbmate for versioned SQL: `db/migrations/`
- Run via `mise run migrate` (or `mr` for reset)

## 9. Logging

### Strategy
- **Library**: slog (structured logging)
- **Pattern**: Canonical log lines (one entry per HTTP request context)
- **Structure**: Main message + structured fields:
  ```
  method=GET path=/users status=200 duration=15ms user_id=abc123
  ```
- **Error level**: Only for unexpected/unhandled errors (not normal business logic failures)
- **Middleware**: Request-scoped logger in context; consolidate all request details into one final log entry

### Future Exploration
- Alternative canonical logging approaches (e.g., JSON-based, distributed tracing)

## 10. Error Handling

### Current Approach
- **Simple**: `fmt.Errorf`, `errors.New`, error wrapping
- **Boundary validation**: Validate user input at HTTP layer; trust internal contracts

### Future Exploration
- Custom error types (domain errors with codes/messages)
- HTTP status code mapping
- Global error handling middleware
- Standardized response format (JSON/HTML templates)
- Session/CSRF security best practices

**Note**: Structure handlers to accommodate hierarchy & middleware integration later.

## 11. Configuration

- Env-based config loading (follow existing patterns in `internal/config/`)
- Environment file: `.env` (loaded via mise)
- No secrets in code; externalize all config

## 12. Development Workflow

### Mise Tasks
```bash
mise run dev          # Hot-reload via watchexec
mise run test         # Run all tests
mise run lint         # golangci-lint
mise run check        # Lint + test
mise run sql          # SQLC generate
mise run compose      # Docker Compose up (Postgres)
mise run compose-stop # Docker Compose down
mise run migrate      # Run migrations
mise run migration    # Create new migration
```

### Docker
- Minimal image (Go binary only)
- Dependency: Postgres (via docker-compose)

## 13. Token-Saving Tips (RTK)

- Grep SQLC queries instead of reading generated files
- Reference common patterns via summaries instead of full file reads
- Use rtk for all CLI operations (auto-hooked)
