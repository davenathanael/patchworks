# Authentication

## Current implementation

- Registration accepts email and password, hashes with argon2id, and stores an
  encoded PHC string. Login verifies the hash, stamps `last_login_at`, creates a
  database session, and redirects home.
- The cookie contains an encrypted session ID using AES-GCM. The database is
  the source of session validity; the cookie is `HttpOnly`, `SameSite=Lax`,
  path `/`, and has a 30-day lifetime. `Secure` follows
  `SESSION_COOKIE_SECURE`.
- Protected routes use `middleware.Auth` to load the session and user. Expired
  sessions are deleted only opportunistically when their cookie is presented.
- Logout currently revokes the current session through `GET /auth/logout`, and
  the application shell renders ordinary logout links. A mutating GET is the
  current security gap; the target is a normal `POST` form protected by the
  shared CSRF mechanism and usable without JavaScript.
- `internal/auth` owns private `sessionStore` and `userStore` interfaces;
  `*internal/db.DB` is wired as the implementation. The body-limit and error
  boundaries are described in [docs/architecture.md](architecture.md) and
  [docs/frontend.md](frontend.md).

Password hashing uses argon2id with 64 MiB memory, one iteration, four lanes,
and a 32-byte key. The stored format is
`$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>`.

## Boundary policy and open risks

The credential boundary currently rejects empty fields only. It does not define
email canonicalization/case policy or password minimum/maximum and
normalization rules. The database has an exact unique email constraint, not a
case/identity policy. Login also skips Argon2id entirely for an unknown email,
and there is no authentication rate limiter. These are tracked in
[AU-1](features/authentication.md#au-1--register-with-email-and-password) and
[AU-2](features/authentication.md#au-2--log-in-and-create-a-session).

There is no shared body limit for all form decoders. Bookmark paths use a
handler-local 1 MiB limit, while auth, collection, membership, and edit paths
still decode without a common policy. Apply one bounded decoder boundary to
every form, including future routes, as specified by
[UX-1](features/interaction.md#ux-1--server-rendering-with-progressive-enhancement).

Session lifecycle is also incomplete: expiry is fixed at 30 days, cleanup is
opportunistic, and rotation, idle expiry, revoke-all, and credential-change
invalidation are not defined. See [AU-3](features/authentication.md#au-3--protect-the-session-cookie)
and the recovery implications in [FR-12](features/roadmap.md#fr-12--email-verification-and-password-reset).

## Open work

- **P0 — Replace logout GET with CSRF-protected POST:** change the route and
  shell controls, reject missing/invalid tokens before revocation, and test
  plain and htmx requests. See [AU-4](features/authentication.md#au-4--log-out).
- **P0 — Equalize unknown-account work and rate-limit authentication:** use a
  fixed dummy Argon2id verification and a bounded per-IP/account strategy with
  generic responses. See [AU-2](features/authentication.md#au-2--log-in-and-create-a-session).
- **P1 — Define identity and password policy:** canonicalize email consistently
  and enforce explicit password bounds before hashing. See
  [AU-1](features/authentication.md#au-1--register-with-email-and-password).
- **P1 — Define session lifecycle:** add bounded expired-session cleanup,
  rotation/revocation rules, and credential-change invalidation. See
  [AU-3](features/authentication.md#au-3--protect-the-session-cookie).
- **P0 — Centralize request-body limits:** apply the shared bounded decoder
  policy to auth and all other form routes, as tracked by
  [UX-1](features/interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **P1 — Keep classified auth/resource errors out of unexpected-error logs:**
  preserve the shared status mapping while ensuring expected not-found and
  forbidden outcomes are not logged at error level. See
  [docs/logging.md](logging.md) and [UX-3](features/interaction.md#ux-3--render-unexpected-errors-safely).

Future identity and recovery behavior belongs to
[FR-2](features/roadmap.md#fr-2--oauth--oidc-login) and
[FR-12](features/roadmap.md#fr-12--email-verification-and-password-reset); this
page records only the current local-auth implementation and its operational
boundary.
