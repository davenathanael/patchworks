# Authentication

## Flow

1. User hits `/auth/login` → redirect to PocketID OIDC provider
2. PocketID redirects to `/auth/callback` with auth code
3. Server exchanges code for tokens (PKCE)
4. Upserts user in DB
5. Creates session record in DB
6. Sets encrypted session cookie

## Structure

- `pkg/auth/oidc/` — OIDC provider wrapper (token exchange, claims parsing)
- `pkg/auth/` — Service orchestrator (initiate login, handle callback, logout, get user from cookie)
  - Private interfaces: `oidcClient`, `sessionStore`, `userStore`
  - Implementations wired in `internal/components/` via `db.NewAuthAdapter`
- `internal/http/middleware/auth.go` — Extracts session cookie, loads user, injects into context

## Key Details

- PKCE with S256 code challenge method
- OAuth state stored in short-lived cookie (5 min)
- Session cookie encrypted with base64-decoded key from config
- Session expiry: 30 days
- Cookie config: HttpOnly, Secure, SameSite=Lax
