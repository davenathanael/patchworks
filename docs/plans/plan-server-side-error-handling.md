# Plan: Server-side error reporting & handling (handlers orchestrate, one wrapper renders the rest)

Status: **implemented — all steps landed, `mise run check` green**

## Goal

- Core errors live in `internal/core/errors.go` (framework-free; no net/http).
- Dependency layers (db, auth) **wrap** underlying errors with core sentinels; pgconn types never cross `internal/db`.
- Handlers become `type Handler func(w, r) error`; they classify expected errors themselves (re-render form with inline field errors / toast / redirect), and `return err` for everything else.
- One shared wrapper (`middleware.HandleError`) logs unexpected errors with a **non-guessable error ID** (uuid) and renders a styled error page ("Back to dashboard" → `/`) for full-page requests, and a **toast into a `#toast-container`** for htmx requests.
- Existing htmx uses stay untouched; auth middleware redirect becomes htmx-aware (latent bug fix).

## Recommended design (from research)

- **Sentinels in core + status map in middleware**: `errors.Is(err, core.Err...)` in one `classify()` in `middleware/errors.go`. Keeps core pure; handlers classify by identity, `classify` maps sentinel → status (safety net for leaks).
- **Handler adapter**: `Adapt(h Handler) http.HandlerFunc` — chi middleware chain untouched; registration sites wrap with `Adapt(...)`.
- **Expected errors render at 200** (both full-page and htmx — avoids htmx's no-swap-on-4xx/5xx default). Unexpected errors keep real statuses + `X-Error-Id` header; a 5-line `htmx:beforeSwap` snippet allows swapping only responses carrying `HX-Retarget` (our toasts).
- **Error ID**: `uuid.New()` generated inside `HandleError` (chi's `chimw.RequestID` is client-spoofable + enumerable counter — rejected). Same ID goes into the slog line, `X-Error-Id` header, and rendered page/toast.
- **Form errors**: `views.FormErrors map[string]string` (field → message; `"form"` = top-level alert), re-render with submitted values preserved, `aria-invalid`/`aria-describedby`, `FieldError` component under inputs. No flash store (direct render fits htmx partials; PRG-success flashes deferred).
- **Unexpected errors only log at error level** (matches docs/logging.md); expected-error paths log nothing.

## Options considered (research findings)

| Option | Verdict |
|---|---|
| AppHandler `func(w,r) error` + one wrapper (Patio/Grafana pattern) | **Adopted** |
| Panic-as-error-flow middleware (chi 06_error_handling example) | Rejected — panics for expected errors; Recoverer already the safety net |
| Typed errors carrying status/Kind in core (joeshaw hybrid, httpwrap/chiwrap) | Rejected — couples core to net/http; no caller needs fields today; sentinel+table is simpler |
| chi `CaptureErr` (chi#983) | Rejected — unmerged, per-handler, no ID/page rendering |
| Flatten pg err: `fmt.Errorf("create user: %w: %v", ErrEmailTaken, pgErr)` | **Recommend** — sentinel identity for `errors.Is`, pg detail in message, pgconn type stays in `db/` |
| Full chain: custom error embedding pg err + custom `Is` | Alternative — keeps `errors.As`→pgconn deep-stack, more machinery, pgconn escapes boundary |
| Error ID = `chimw.RequestID` | Rejected — spoofable/enumerable (fails requirement) |
| Flash messages / session store for errors | Rejected — no store exists; direct render fits htmx |

## Files & symbols

**New files**
- `internal/core/errors.go` (extend): sentinels `ErrEmailTaken`, `ErrInvalidCredentials`, `ErrNotFound` (collections/bookmarks 404 live here) +
- `internal/http/middleware/errors.go`: `HandleError(w, r, err)`, `classify(err) (status, userMsg)`, `errorIDKey` ctx key, `ErrorIDFromContext(ctx)`
- `internal/http/handlers/errors.go`: `type Handler func(w, r) error`, `Adapt(h Handler) http.HandlerFunc` (+ shared pure `validateCredentials`)
- `internal/http/views/errors.go`: `FormErrors` map type, `FieldError(field, errs)`, `Toast(kind, msg, errorID)`, `ErrorPage(msg, errorID)` (full standalone `Page`, not AppShell)
- `resources/static/css/app.css`: append `.error-page`, `.toast …`, `.form-error`, `.toast-dismiss` (Open Props tokens, cascade layers, no inline styles)
- Tests: `middleware/errors_test.go` (classify table; htmx vs full-page; error ID present), pure validator tests

**Modified files**
- `internal/db/queries.go:45-51` — wrap 23505 with `ErrEmailTaken` chain (flatten pg detail); keep `errors.As` check in `db/`
- `internal/auth/auth.go:15,58-60` — delete local `ErrInvalidCredentials` (use core's); `Register` passes wrapped db error through instead of re-returning bare sentinel
- `internal/http/handlers/auth.go` (11 sites), `bookmarks.go` (15), `collections.go` (31) — convert to `Handler`; expected → classify+render, unexpected → `return fmt.Errorf("…: %w", err)`
- `internal/http/handlers/handler.go` — wrap all registrations in `Adapt(...)`
- `internal/http/middleware/auth.go:27` — htmx-aware: `401` + `HX-Redirect: /auth/login` on htmx, else current 302
- `internal/http/views/views.go:24,30` — htmx `beforeSwap` script in `Page` head; `Div(ID("toast-container"))` in `AppShell`
- `internal/http/views/auth.go` — `LoginPage/RegisterPage(email, password string, errs FormErrors)` render values + `FieldError`s + top alert
- `internal/http/views/links.go` — `ID("add-bookmark-form")` on NewBookmark form; field errors target it (hx-target conflict)
- Tests: `auth_handlers_test.go`, `bookmarks_test.go`, `collections_test.go`
- Docs: docs/architecture.md (handler pattern), docs/logging.md (`error_id` attr)

## Usage pattern

```go
// handler (auth.go)
func handlePostLogin(svc AuthLoginHandler) Handler {
    return func(w http.ResponseWriter, r *http.Request) error {
        var f credentialsForm
        if err := form.NewDecoder(r.Body).Decode(&f); err != nil {
            return renderLogin(w, f, views.FormErrors{"form": "invalid form data"})
        }
        if errs := validateCredentials(f); len(errs) > 0 {
            return renderLogin(w, f, errs)
        }
        if err := svc.Login(w, r, f.Email, f.Password); err != nil {
            if errors.Is(err, core.ErrInvalidCredentials) {
                return renderLogin(w, f, views.FormErrors{"form": "invalid email or password"})
            }
            return fmt.Errorf("login: %w", err) // -> wrapper -> 500 + error ID page/toast
        }
        http.Redirect(w, r, "/", http.StatusFound)
        return nil
    }
}
```

```go
// middleware/errors.go (unexpected errors only)
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
    status, userMsg := classify(err) // ErrInvalidCredentials->401, ErrEmailTaken->409, ErrNotFound->404, else 500
    errorID := uuid.New().String()
    ctx := context.WithValue(r.Context(), errorIDKey, errorID)
    slog.ErrorContext(ctx, "unexpected request error", "error_id", errorID,
        "request_id", chimw.GetReqID(r.Context()), "method", r.Method, "path", r.URL.Path, "error", err)
    w.Header().Set("X-Error-Id", errorID)
    if views.IsHtmx(r) {
        w.Header().Set("HX-Retarget", "#toast-container")
        w.Header().Set("HX-Reswap", "innerHTML")
        w.WriteHeader(status)
        return views.Toast("error", userMsg, errorID).Render(w)
    }
    w.WriteHeader(status)
    return views.ErrorPage(userMsg, errorID).Render(w)
}
```

```go
// registration (handler.go) — chi chain untouched
r.Post("/auth/login", Adapt(handlePostLogin(comp.AuthService)))
```

```go
// views/errors.go
type FormErrors map[string]string // field -> message; "form" = top-level alert
func FieldError(field string, errs FormErrors) Node { /* P(class form-error, aria-describedby) */ }
func Toast(kind, message, errorID string) Node      { /* .toast.toast-error + dismiss button */ }
func ErrorPage(message, errorID string) Node        { /* standalone Page + back-to-dashboard A */ }
```

## Order of changes (each step compiles + tests green)

1. `core/errors.go` sentinels (+ `ErrNotFound`)
2. `db/queries.go` wrap + `auth/auth.go` pass-through (remove local sentinel; update deps/tests together)
3. `views/errors.go` components + `views.go` container/beforeSwap + auth form signatures
4. `middleware/errors.go` + `middleware/auth.go` htmx redirect
5. Handlers: `Handler`/`Adapt`, convert auth.go → bookmarks.go → collections.go (biggest; needs per-endpoint expected-error decisions, e.g. collection-not-found → 404)
6. Register via `Adapt`; `#add-bookmark-form` ID + retarget
7. `app.css` styles
8. Tests + docs (`docs/architecture.md`, `docs/logging.md`)
9. Run `mise run check` (lint + test)

## Risks

- **Mechanical churn**: every handler factory return type + registration site changes; go build after each file.
- **Write-before-return**: a handler that writes then returns an error leaves the wrapper unable to render (headers committed). Audit `bookmarks.go` success paths (partial multi-render) before converting.
- **Test breakage**: handler tests assert status + plain-text body; bodies become HTML.
- **htmx target conflict (existing)**: NewBookmark form targets `#bookmarks` on success; error re-render must retarget to the form itself (new ID).
- **ErrInvalidCredentials move** must land atomically with its usages.
- **Auth middleware 302-on-htmx** (pre-existing bug): stale-session htmx request would swap the login page into the dashboard; fixed in step 4 regardless — could be a separate small PR.

## Open questions for approval

1. Core error style: **sentinels + classify table** (recommended) vs typed status-carrying errors in core?
2. db wrapping: **flatten pg detail into message** (recommended) vs full `%w` chain keeping pgconn in the chain?
3. htmx unexpected errors: **toast into #toast-container** (recommended) vs full error page swap?
4. Scope: implement all steps in this session, or land in stages (harness: core→middleware→views first, handlers next)?