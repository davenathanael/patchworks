# Logging

## Event model

Logging uses `log/slog`. The current request path has **two separate event
families**, not one entry containing every request detail:

- `middleware.Logger` uses `httplog.RequestLogger` at info level to record
  request completion, status, timing, and request metadata.
- `middleware.HandleError` emits a separate error event for an error that
  escapes a handler. It includes a fresh `error_id`, the chi `request_id`,
  method, path, and the error, then exposes that ID in `X-Error-Id` and the
  rendered full-page error or htmx toast.

The request event remains useful when the response is successful or expected;
the error event is for diagnosis. They must share correlation fields rather
than being described as one consolidated line.

## Levels and error policy

- Expected validation, authentication, authorization, not-found, and normal
  business outcomes should not be error-level events. A warning or
  informational operational signal may be added when it helps diagnosis,
  without exposing user data.
- Unexpected database, rendering, decoding, or dependency failures use
  `slog.Error`/`slog.ErrorContext`. The shared renderer exposes only a generic
  message and the correlation ID.
- `HandleError`'s sentinel classifier is the shared HTTP mapping, but its
  current implementation logs every error that reaches it at error level.
  Handlers should therefore render expected outcomes themselves; closing the
  remaining not-found/forbidden leakage into the error event is open work.
- Never log passwords, session cookies, encryption keys, database URLs,
  authorization headers, or raw request bodies. Treat user email, URLs, and
  form values as potentially sensitive.

## Correlation

Every request receives a chi request ID. Unexpected errors add an independent
non-guessable error ID; that ID must match the server event and the user-facing
reference. Lifecycle events that occur during a request should carry the
request ID when one exists. Background work, migrations, and startup/shutdown
events need their own stable event names and identifiers rather than inventing
request IDs.

## Startup, shutdown, and dependencies

The application currently logs a startup message and can panic when component
construction or `ListenAndServe` fails. It has no application shutdown event.
Migrations are run outside the HTTP process by dbmate/the deployment migration
service, so migration start, version, success, and failure belong to that
runner's operational log. Database ping failures, session-key failures, and
other dependency failures should be logged with actionable context before the
process exits, without secrets.

The target policy is:

- startup: environment, bind address, config-validation result, and dependency
  readiness, with failures at error level;
- shutdown: signal received, draining state, bounded completion, and close
  errors;
- migrations: migration identity, start/end, duration, and failure, never the
  database URL;
- dependencies: database/auth/HTTP-client readiness and failure, with retry or
  operator action where relevant.

Local development uses a human-readable debug logger; production uses JSON.
The request middleware and normal `slog` events must share the same field
vocabulary and correlation conventions.

## Open work

- **P1 — Standardize lifecycle and dependency logging:** implement the startup,
  shutdown, migration, and dependency events above, with actionable levels and
  no credentials. This is operational work under
  [FR-18](features/roadmap.md#fr-18--self-host-polish).
- **P1 — Align request/error schemas:** remove the stale one-entry claim,
  verify the `httplog` and `slog` field names, and test request/error
  correlation and redaction. See [UX-3](features/interaction.md#ux-3--render-unexpected-errors-safely).
- **P1 — Keep expected errors out of the error event:** render classified
  not-found/forbidden outcomes without error-level logging, while retaining
  generic responses and correlation for truly unexpected failures. See
  [UX-3](features/interaction.md#ux-3--render-unexpected-errors-safely).
- **P1 — Log configuration validation failures with context:** report the
  invalid setting and remedy without echoing secrets; the validation policy is
  in [docs/config.md](config.md).
