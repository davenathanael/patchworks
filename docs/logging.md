# Logging

## Strategy

- Library: slog (standard library)
- Pattern: Canonical log lines — one entry per HTTP request

## Structure

```
method=GET path=/users status=200 duration=15ms user_id=abc123
```

## Conventions

- Error level only for unexpected/unhandled errors
- Normal business logic failures use Info/Warn level
- Request-scoped logger in context via httplog middleware
- All request details consolidated into one final log entry
- Unhandled request errors are logged once by `middleware.HandleError` with `error_id` (fresh uuid) + `request_id` attrs; that `error_id` is also surfaced to the user (error page / toast) for support correlation

## Configuration

- Logger configured in middleware, controlled by `config.Environment.IsLocal()`
- Local dev: human-readable output
- Production: structured JSON
