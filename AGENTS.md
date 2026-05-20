# Patchworks

Personal bookmark manager. Go 1.26.2, chi v5, gomponents, pgx/v5 + SQLC, OIDC auth (PocketID), fixi.js.

## Quick Reference

| Command | Action |
|---|---|
| `mise run dev` | Hot-reload dev server |
| `mise run test` | Run all tests |
| `mise run lint` | golangci-lint |
| `mise run check` | Lint + test |
| `mise run sql` | SQLC generate |
| `mise run compose` | Docker Compose up (Postgres) |
| `mise run migrate` | Run migrations |

## Architecture

DDD-inspired layers. See @docs/architecture.md.

## Conventions

- **Handlers**: inject interfaces, return concretes, pure handlers with abstracted I/O
- **Views**: gomponents server-side HTML, OAT CSS + OpenProps. See @docs/frontend.md
- **DB**: SQLC generated, wrapped in repos. See @docs/db.md
- **Auth**: OIDC via PocketID. See @docs/auth.md
- **Tests**: earthboundkid/be, local fakes — no mock libs. See @docs/testing.md
- **Logging**: slog, canonical log lines. See @docs/logging.md
- **Config**: env-based via caarlos0/env. See @docs/config.md
- **Error handling**: fmt.Errorf wrapping, validate at boundaries

## Go Conventions

- Pure functions where possible; extract I/O to boundaries
- Accept interfaces, return concretes
- Minimal comments — explain why, not what
- No unnecessary abstractions
- Prefer concrete fakes over mock libraries

<!--
To externalize these Go rules to a shared GitHub source, replace the section above with:
See https://raw.githubusercontent.com/YOUR_ORG/rules/main/go.md
Then add to opencode.json:
{ "instructions": ["AGENTS.md", "https://raw.githubusercontent.com/YOUR_ORG/rules/main/go.md"] }
-->

## Token-Saving Tips

- Use @explore for codebase research (read-only, token-efficient)
- Use @general for multi-step parallel tasks
- Reference @docs/* files for details — they load on demand
- Grep SQLC queries instead of reading generated files
