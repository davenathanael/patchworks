# Configuration

## Loading and current ordering

Configuration is environment-based and parsed by `caarlos0/env/v11` from the
structs in `internal/config/config.go`. `mise` loads the local `.env`; secrets
are not stored in code. The current sections are:

- `DB`: `DATABASE_URL`.
- `HTTPServer`: `HOST`, `PORT`, `READ_TIMEOUT_SECONDS`, and
  `WRITE_TIMEOUT_SECONDS`.
- `Session`: `SESSION_ENCRYPTION_KEY` and `SESSION_COOKIE_SECURE` (default
  `true`).
- `Environment`: `ENVIRONMENT` (default `local`).

`components.New` currently loads and parses configuration, opens and pings the
database, and only then calls `createAuthService`, where the base64 session key
is decoded. Therefore a bad session key is discovered **after** the database
connection is opened. The boot still fails, but the current ordering is not a
pure-config fail-fast check.

## Validation policy

`config.Load` currently supplies parser errors and defaults but does not perform
range or combination validation. The intended startup policy is:

- require a non-empty, parseable database URL without logging credentials;
- require an explicitly supported environment value rather than treating every
  unknown value as local; `IsLocal` currently treats every value other than
  `production` as local;
- require a non-empty host and a numeric port in the supported range;
- require positive, intentionally bounded read/write timeouts;
- require `SESSION_ENCRYPTION_KEY` to be valid base64 decoding to exactly 32
  bytes for AES-256;
- require `SESSION_COOKIE_SECURE=true` in production; an explicit `false` may be
  used only for a documented plain-HTTP local/LAN setup, never as a production
  combination;
- return actionable setting names and remedies, while never echoing secrets.

All cheap configuration checks, especially session-key decoding, should run
before opening external dependencies. Dependency readiness and lifecycle
logging are covered in [docs/logging.md](logging.md) and
[docs/architecture.md](architecture.md).

## Open work

- **P1 — Add semantic configuration validation:** validate environment,
  database URL, host/port, timeout ranges, session key length, and
  secure-cookie/production combinations before the DB is opened. Coordinate
  with [AU-3](features/authentication.md#au-3--protect-the-session-cookie) and
  [FR-18](features/roadmap.md#fr-18--self-host-polish).
- **P1 — Standardize operator errors:** make missing/invalid settings fail at
  startup with actionable, non-secret diagnostics; do not rely on a later
  request or panic-only failure. See [docs/logging.md](logging.md).
- **P1 — Keep the production/local cookie policy explicit:** document and test
  the allowed `ENVIRONMENT`/`SESSION_COOKIE_SECURE` combinations rather than
  inferring them from the current `IsLocal` implementation.
