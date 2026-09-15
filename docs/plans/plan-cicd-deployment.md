# Plan: CI/CD + production deployment

**Status:** landed (2026-09-15) · archived from root `plan.md`

## Goal

CI on every push/PR (lint, unit, integration tests), tag-triggered GHCR image push, production docker-compose stack.

## Decisions

- GHCR image only — no GitHub Release entry created on tags.
- Prod migrations run via a **dbmate one-shot compose service** (not a background job, not CI).
- Pin `go = 1.26.2`, `golangci-lint = 2.11.4` in `mise.toml` for deterministic CI.
- Compose project name is `patchwork-prod` (see gotcha below).

## Delivered

| File | What |
|---|---|
| `Dockerfile` | Multi-stage: `golang:1.26.2-alpine` builder → `distroless/static-debian12:nonroot`; binary + `resources/static/`, CA certs copied. ~15 MB, non-root. |
| `.dockerignore` | Excludes `.git`, `.github`, `docs`, `.env*`, `coverage.out`. |
| `.github/workflows/ci.yml` | Push (all branches) + PR. `check` = `mise run lint` + `mise run test`; `integration` = postgres:18 service + `TEST_DATABASE_URL` + `mise run test-integration`. Tooling via `jdx/mise-action`. |
| `.github/workflows/release.yml` | Tag `v*` → QEMU + buildx, amd64+arm64, push `ghcr.io/davenathanael/patchwork:{semver,latest}`. `packages: write`. |
| `docker-compose.prod.yml` | `db` (healthcheck, named volume) → `migrate` (dbmate, bind-mounts `./resources/db:ro`, `--wait up`) → `app` (GHCR image, `depends_on: migrate: service_completed_successfully`). |
| `mise.toml` | Pinned go + golangci-lint. |
| `docs/deploy.md` | Ops reference: CI jobs, image tags, prod stack, vars table, redeploy flow. |
| `AGENTS.md` | Quick-ref row + `docs/deploy.md` pointer. |

## Docker build flags

- `CGO_ENABLED=0` — static binary, no libc (required by distroless-static); pgx/argon2/chi are pure Go.
- `GOOS=linux` — cross-compile from macOS.
- `-trimpath` — no host paths in the binary (reproducible).
- `-ldflags="-s -w"` — strip symbol table + DWARF; stack traces stay readable (Go keeps the pclntab line table).

## Verification (empirical)

- `docker build` OK (~15 MB); container boots, config loads, pgx connects.
- Full prod stack with local image tagged as GHCR: db healthy → migrate applies all 8 migrations → app serves `/auth/login` 200 → register (302) persists `ci@test.local`.
- `app` starts only after migrate succeeds. Failure path (wrong password vs existing volume): migrate `Exited (2)`, compose aborts, `app` stays `Created`, nothing served.
- `migrate` re-runs on every `up -d` (verified via `StartedAt` deltas), idempotent — exit 0, no pending migrations.
- Workflow YAML parses; `docker compose config` validates; `:?` guards fail fast with instructive messages.

## Gotchas learned

- **Compose project collision.** `name:` must be `patchwork-prod`. The dev stack (`docker-compose.yml`, dir-name project `patchwork`) already owns project `patchwork` — db + dex/caddy/zitadel containers and `patchwork_db_data`. Without the rename, prod `up` adopts the dev db (wrong password, migrations never run, app crash-loops).
- **`SESSION_ENCRYPTION_KEY` must decode to exactly 32 bytes** (AES-256). A 33-byte key panic surfaces only *after* DB connect succeeds — DB errors mask it.
- **Bind mount is read-only**; dbmate logs `Writing: /db/schema.sql` after migrating but this is best-effort — exit stays 0, repo `schema.sql` unchanged.
- **Distroless = no shell** — no `docker exec` debugging; certs are copied in because distroless ships none, and the app fetches page titles over HTTPS (BK-2).
- **`SESSION_COOKIE_SECURE=true` in prod** means real HTTPS is required in front (reverse proxy); plain HTTP with a real hostname breaks login cookies.
- **Migrations are not in the app image.** The `migrate` service reads them from the host checkout (`./resources/db`) — no git inside the container. Deploy = `git pull && docker compose -f docker-compose.prod.yml up -d`.

## Backlog (not implemented)

### Bake migrations into an image (removes the host-checkout requirement)

The bind mount ties deploys to a repo checkout on the host (`git pull` before `up -d`). To make deploys purely image-driven:

1. Add `Dockerfile.migrate`:
   ```dockerfile
   FROM amacneil/dbmate:2.32.0
   COPY resources/db /db
   ```
2. `release.yml`: build + push a second image `ghcr.io/davenathanael/patchwork-migrate` on the same tags (reuse metadata-action output; add a second build-push step).
3. `docker-compose.prod.yml`: drop the `./resources/db:/db:ro` bind mount; set `migrate.image: ghcr.io/davenathanael/patchwork-migrate:${PATCHWORK_IMAGE_TAG:-latest}`; keep the dbmate env vars (`DBMATE_MIGRATIONS_DIR=/db/migrations`, `DBMATE_SCHEMA_FILE=/db/schema.sql`) and the `--wait up` command.

Result: host needs no checkout (only the compose file) — `docker compose pull && up -d`. Migrations are pinned to the same tag as the app. Tradeoff: a second published image; compose file still lives on the host.

### Pre-existing lint failure (blocks CI on first push)

`mise run check` fails with 3 gosec issues: `r.ParseForm()` without `http.MaxBytesReader` (`internal/http/handlers/*.go`, e.g. `bookmarks.go:758`). Not introduced by this work. Fix: wrap form-parsing handlers with a bounded body reader.

### Other follow-ups

- GHCR packages are private by default — toggle to public in package settings (or `docker login ghcr.io` on hosts).
- `mise.toml` still uses `latest` for sqlc/dbmate/watchexec (only go + golangci-lint pinned).
- `POSTGRES_PASSWORD` is interpolated into `DATABASE_URL` — avoid URL-special characters.
