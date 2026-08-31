# Configuration

## Approach

- Env-based config loading via `caarlos0/env/v11`
- Config structs in `internal/config/config.go`
- `.env` file loaded by mise (not committed to git)
- No secrets in code — all externalized

## Config Sections

- **HTTPServer**: host, port, timeouts
- **DB**: connection URL
- **Session**: encryption key (base64-encoded). `SESSION_ENCRYPTION_KEY` must decode to exactly 32 bytes (AES-256); generate with `openssl rand -base64 32`. Boot fails fast on a missing or invalid key.
- **Environment**: local, production, etc.
