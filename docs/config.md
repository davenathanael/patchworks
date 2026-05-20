# Configuration

## Approach

- Env-based config loading via `caarlos0/env/v11`
- Config structs in `internal/config/config.go`
- `.env` file loaded by mise (not committed to git)
- No secrets in code — all externalized

## Config Sections

- **HTTPServer**: host, port, timeouts
- **DB**: connection URL
- **OIDC**: issuer, client ID/secret, redirect URL
- **Session**: encryption key (base64-encoded)
- **Environment**: local, production, etc.
