# Authentication

## Flow

1. User registers at `/auth/register` (email + password) → password hashed with argon2id, user created.
2. User logs in at `/auth/login` (email + password) → password verified, session created, session cookie set.
3. Session cookie is encrypted (AES-GCM), valid for 30 days.
4. Protected routes read the session cookie, load the user, inject into request context.
5. `/auth/logout` deletes the session and clears the cookie.

## Structure

- `internal/auth/` — `Service` (register, login, logout, get user from cookie), password hashing (`password.go`), cookie encryption (`cookie.go`).
  - Private interfaces: `sessionStore`, `userStore`.
  - Implementations wired in `internal/components/` via `*db.DB` directly.
- `internal/http/middleware/auth.go` — Extracts session cookie, loads user, injects into context.

## Password Hashing

- argon2id via `golang.org/x/crypto/argon2`.
- Parameters: memory 64 MiB, iterations 1, parallelism 4, key length 32 bytes.
- Stored as an encoded PHC string: `$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>`.

## Key Details

- Session cookie encrypted with base64-decoded key from config (AES-GCM).
- Session expiry: 30 days.
- Cookie config: HttpOnly, Secure, SameSite=Lax.

## Future: OAuth / OIDC

Designed for easy migration to Google/GitHub/OIDC login:

- `users.password_hash` is nullable — OAuth-only users have no password.
- Add a `user_identities` table (`provider`, `provider_user_id`, `user_id`, unique on provider + id) to link external accounts.
- Add an OAuth provider that does the redirect/callback dance and upserts into `users` + `user_identities`.
