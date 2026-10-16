# Authentication (`AU-*`)

**Scope:** local email/password registration, login, server-backed sessions, logout, and protected-route behavior. Product intent and the feature summary are in [`docs/spec.md`](../spec.md#51-accounts--authentication--impl); implementation policy is in [`docs/auth.md`](../auth.md), with request/error wiring in [`docs/architecture.md`](../architecture.md).

## Index

| ID | Status | Priority | Contract |
|---|---|---|---|
| [AU-1](#au-1--register-with-email-and-password) | partial | P1 | Normalize identity and enforce password policy |
| [AU-2](#au-2--log-in-and-create-a-session) | partial | P0 | Verify credentials, bound work, and issue a session |
| [AU-3](#au-3--protect-the-session-cookie) | partial | P1 | Encrypt, constrain, and retire sessions safely |
| [AU-4](#au-4--log-out) | partial | P0 | Revoke the current session through a CSRF-protected POST |
| [AU-5](#au-5--protect-application-routes) | implemented | — | Redirect or challenge unauthenticated requests |
| [AU-6](#au-6--validate-empty-credentials) | implemented | — | Re-render incomplete forms safely |
| [AU-7](#au-7--record-successful-login) | implemented | — | Stamp `last_login_at` only after credential success |

## AU-1 — Register with email and password

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`AU-1`](../spec.md#51-accounts--authentication--impl)
- **Intent/behavior:** A signed-out visitor can submit an email and password at `/auth/register`. The server hashes the password with argon2id using the repository's configured parameters (64 MiB, one iteration, four lanes, 32-byte key), stores the encoded PHC string, and redirects to `/auth/login` on success. A duplicate email is re-rendered at status 200 with an inline/form-level error and does not create another user. The current boundary only rejects empty fields: it does not canonicalize email addresses or enforce a password length/complexity policy.
- **Authorization:** Public registration endpoint; no authenticated session is required.
- **UX/no-JS:** The server-rendered form works without JavaScript. Invalid form data, missing fields, and duplicate email preserve submitted values where the form model supports them. Unexpected failures use the shared error response.
- **Failure semantics:** Empty fields are expected validation errors. A unique-email database error is translated to `ErrEmailTaken`; other storage/hash failures are unexpected errors.
- **Acceptance criteria:** A valid registration creates one canonical user identity with an argon2id PHC hash and no plaintext password; duplicate registration creates no second user and shows the duplicate error; a successful registration does not silently log the user in. Invalid/overlong email or password input is rejected before hashing or persistence. **Current gap:** only empty values are rejected; the shared request-body limit is tracked in UX-1.
- **Implementation notes:** [`internal/auth/password.go`](../../internal/auth/password.go), [`internal/auth/auth.go`](../../internal/auth/auth.go), [`internal/http/handlers/auth.go`](../../internal/http/handlers/auth.go), and the auth unit tests are the evidence. The database has a unique email constraint.
- **Related:** [AU-2](#au-2--log-in-and-create-a-session), [AU-6](#au-6--validate-empty-credentials), [FR-12](roadmap.md#fr-12--email-verification-and-password-reset).
- **Open work:**
  - **P1 — Normalize email identity:** define the canonical form (including a case policy), use it for registration, lookup, and display, and add a migration/constraint strategy so equivalent addresses cannot become separate accounts.
  - **P1 — Enforce a password policy:** set explicit minimum/maximum and normalization rules at the registration boundary; reject violations before the expensive Argon2id operation and cover them with handler tests.
  - **P1 — Define the legacy-user migration path:** decide how users created before password authentication are upgraded, reset, or retired; the current migration makes `password_hash` nullable and does not transform existing identity rows into usable credentials.

## AU-2 — Log in and create a session

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`AU-2`](../spec.md#51-accounts--authentication--impl)
- **Intent/behavior:** A visitor submits email and password at `/auth/login`. The service looks up the user, verifies the stored argon2id hash, stamps the successful-login timestamp, creates a database session with a UUID and expiry, sets the encrypted cookie, and redirects to `/`.
- **Authorization:** Public login endpoint; the resulting session is scoped to the authenticated user.
- **UX/no-JS:** Plain form submission is supported. The login page links to registration.
- **Failure semantics:** Unknown email and wrong password use the generic `invalid email or password` message to avoid account enumeration through response text; malformed form encoding is a form-level `invalid form data` error. A failed `last_login_at` write aborts before session creation. Session creation or cookie encryption failure is unexpected and does not produce a successful login response. **Current gap:** an unknown email skips Argon2id entirely, creating a timing difference, and login has no body or attempt-rate bound.
- **Acceptance criteria:** Correct credentials create a valid server-side session and redirect home; invalid credentials do not create a session or disclose whether the email exists through status, body, or materially different password-verification work; a failed last-login stamp leaves no newly created session; oversized bodies and throttled attempts fail predictably.
- **Implementation notes:** See [`internal/auth/auth.go`](../../internal/auth/auth.go), [`internal/auth/auth_test.go`](../../internal/auth/auth_test.go), and [`internal/db/queries.go`](../../internal/db/queries.go).
- **Related:** [AU-1](#au-1--register-with-email-and-password), [AU-3](#au-3--protect-the-session-cookie), [AU-7](#au-7--record-successful-login).
- **Open work:**
  - **P0 — Equalize unknown-user work:** run a fixed dummy Argon2id verification when lookup finds no account, and test that unknown-user and wrong-password paths perform equivalent credential work.
  - **P0 — Rate-limit authentication attempts:** add a bounded per-IP/account strategy for login and registration, with a generic response and tests for throttling without enabling denial-of-service lockout.

## AU-3 — Protect the session cookie

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`AU-3`](../spec.md#51-accounts--authentication--impl)
- **Intent/behavior:** The cookie carries an encrypted session ID rather than a plaintext user/session token. Encryption uses AES-GCM with the configured key; the database remains the source of session validity. The cookie is named `session_id`, scoped to `/`, and has a 30-day lifetime. Sessions are deleted opportunistically when an expired cookie is presented, but there is no background expiry cleanup or explicit rotate/revoke-all lifecycle.
- **Authorization:** Only a valid, unexpired database session whose user can be loaded authenticates a request. Decryption, UUID parsing, missing sessions, missing users, and expiry all fail closed.
- **UX/no-JS:** Cookie behavior is transparent to the HTML flow. Configuration controls the `Secure` flag for local HTTP versus production HTTPS.
- **Failure semantics:** Invalid/tampered ciphertext, a short ciphertext, a missing user, or an expired session yields no authenticated user. Expired sessions are deleted on a best-effort basis.
- **Acceptance criteria:** Cookie values round-trip through encrypt/decrypt; tampering or a wrong key fails; the cookie is `HttpOnly`, `SameSite=Lax`, and `Secure` when configured; expiry is consistent between the database session and cookie age; expired rows are removed by a bounded lifecycle mechanism rather than only when reused. **Current gap:** rotation, revoke-all, and proactive cleanup are unspecified.
- **Implementation notes:** [`internal/auth/cookie.go`](../../internal/auth/cookie.go) and [`internal/auth/cookie_test.go`](../../internal/auth/cookie_test.go) document the cryptographic and cookie contract. The key must decode to exactly 32 bytes at startup; see [`docs/config.md`](../config.md).
- **Related:** [AU-2](#au-2--log-in-and-create-a-session), [AU-4](#au-4--log-out), [AU-5](#au-5--protect-application-routes).
- **Open work:**
  - **P1 — Define the session lifecycle:** choose absolute/idle expiry and rotation rules, add bounded cleanup of expired sessions, and specify revoke-all plus credential-change invalidation; cover expiry and revocation in repository/auth tests.
  - **P1 — Distinguish session lookup failures from unauthenticated requests:** preserve and log database/dependency errors instead of collapsing them into a normal login redirect.

## AU-4 — Log out

- **Status:** `partial`
- **Priority:** `P0`
- **Spec:** [`AU-4`](../spec.md#51-accounts--authentication--impl)
- **Intent/behavior:** The current `GET /auth/logout` deletes the server-side session when a valid cookie is present, clears the cookie, and redirects to `/auth/login`. The target contract is a CSRF-protected `POST` rendered as a normal form, so cross-site GET navigation cannot mutate session state.
- **Authorization:** The operation applies to the session identified by the request cookie; it does not accept an arbitrary session ID.
- **UX/no-JS:** The current logout control is an ordinary link and therefore works without JavaScript, but its GET method is the security gap. The replacement must be a normal POST form with a CSRF token and no JavaScript dependency.
- **Failure semantics:** Missing or malformed cookies are a no-op logout. A database deletion failure is unexpected and is surfaced through the shared error response; the cookie is not treated as a substitute for server-side revocation.
- **Acceptance criteria:** A CSRF-protected POST deletes the current valid session and expires the cookie; GET and missing/invalid CSRF do not revoke it; logout without a cookie remains safe; the browser is sent to the login page.
- **Implementation notes:** [`internal/auth/auth.go`](../../internal/auth/auth.go), [`internal/auth/cookie.go`](../../internal/auth/cookie.go), and auth tests.
- **Related:** [AU-3](#au-3--protect-the-session-cookie), [AU-5](#au-5--protect-application-routes).
- **Open work:**
  - **P0 — Move logout to POST with CSRF:** replace both shell links with a normal form, remove the mutating GET route, validate the shared CSRF mechanism, and add route tests for accepted and rejected requests.

## AU-5 — Protect application routes

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`AU-5`](../spec.md#51-accounts--authentication--impl)
- **Intent/behavior:** All application pages and actions are under the authentication middleware. A valid session is loaded into request context for handlers. An unauthenticated full-page request redirects to `/auth/login`; an htmx request receives `401` with `HX-Redirect: /auth/login`.
- **Authorization:** The middleware is the route boundary, not merely a UI convention. Handlers still enforce resource-specific ownership and collection roles.
- **UX/no-JS:** Redirect behavior works for ordinary browser navigation. The htmx response preserves the current partial-request context instead of returning a full login page.
- **Failure/semantics:** Missing, invalid, expired, or deleted sessions are unauthenticated. Unexpected handler errors are routed through the shared error renderer, not converted into successful redirects.
- **Acceptance criteria:** Public auth routes remain reachable; protected routes reject anonymous requests with the correct response mode; a valid session is available to protected handlers.
- **Implementation notes:** [`internal/http/middleware/auth.go`](../../internal/http/middleware/auth.go) and [`internal/http/server/server.go`](../../internal/http/server/server.go). See [`docs/architecture.md`](../architecture.md) for the handler boundary and [`docs/frontend.md`](../frontend.md) for htmx behavior.
- **Related:** [AU-2](#au-2--log-in-and-create-a-session), [AU-3](#au-3--protect-the-session-cookie), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement), [UX-3](interaction.md#ux-3--render-unexpected-errors-safely).
- **Open work:** None recorded.

## AU-6 — Validate empty credentials

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`AU-6`](../spec.md#51-accounts--authentication--impl)
- **Intent/behavior:** Empty email or password fields are rejected before the auth service is called. The form re-renders at status 200 with field-level messages and preserves the submitted email/password values in the response model.
- **Authorization:** This is an unauthenticated boundary check; it does not reveal whether an account exists.
- **UX/no-JS:** Validation is server-rendered and does not depend on browser JavaScript. Inputs are associated with field errors and `aria-invalid`/`aria-describedby` where rendered by the shared form component.
- **Failure semantics:** Malformed form encoding is an expected form error; service/database failures remain unexpected errors.
- **Acceptance criteria:** Empty email, empty password, and both-empty submissions do not call the auth service; errors identify the missing field; submitted values remain available for correction.
- **Implementation notes:** [`internal/http/handlers/errors.go`](../../internal/http/handlers/errors.go), [`internal/http/views/forms.go`](../../internal/http/views/forms.go), and handler tests.
- **Related:** [AU-1](#au-1--register-with-email-and-password), [AU-2](#au-2--log-in-and-create-a-session), [UX-4](interaction.md#ux-4--show-inline-validation-errors).
- **Open work:** None recorded.

## AU-7 — Record successful login

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`AU-7`](../spec.md#51-accounts--authentication--impl)
- **Intent/behavior:** After credentials verify, the service writes `users.last_login_at` using the database server time. The write occurs before session creation so a failed stamp aborts the login.
- **Authorization:** The write targets the authenticated user's ID; callers cannot choose another user's timestamp.
- **UX/no-JS:** No user-facing control is involved; the behavior is part of successful login.
- **Failure semantics:** Failed credentials do not touch the timestamp. A failed timestamp write returns an error and no session is created.
- **Acceptance criteria:** A successful login leaves a non-null `last_login_at`; failed authentication leaves it unchanged; a failed stamp prevents session creation.
- **Implementation notes:** [`internal/auth/auth.go`](../../internal/auth/auth.go), [`resources/db/queries/users.sql`](../../resources/db/queries/users.sql), and repository/integration tests.
- **Related:** [AU-2](#au-2--log-in-and-create-a-session), [AU-3](#au-3--protect-the-session-cookie).
- **Open work:** None recorded.
