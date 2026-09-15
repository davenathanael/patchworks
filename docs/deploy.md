# Deployment

CI/CD + production self-host. Workflow files lived in `.github/workflows/`.

## CI (`.github/workflows/ci.yml`)

Runs on every push (all branches) and pull request:

- **check** — `mise run lint` + `mise run test` (golangci-lint + unit tests)
- **integration** — `mise run test-integration` against a `postgres:18-alpine` service (`TEST_DATABASE_URL`)

Tooling comes from `mise.toml` (pinned: go 1.26.2, golangci-lint 2.11.4).

## Image releases (`.github/workflows/release.yml`)

Pushing a tag `v*` builds amd64+arm64 and pushes to GHCR:

- `ghcr.io/davenathanael/patchworks:<tag>` and `:latest`
- Image-only — no GitHub Release entry is created
- GHCR package is **private by default**: toggle to public once in package settings
  (Settings → Packages → patchworks → Make public) or machines must `docker login ghcr.io`

## Local image build

```sh
docker build -t patchworks:local .
```

Multi-stage Dockerfile: golang builder → distroless static binary + `resources/static/` (served from disk).

## Production stack (`docker-compose.prod.yml`)

Services: `db` (postgres:18-alpine, named volume, healthcheck) → `migrate` (dbmate one-shot, bind-mounts `./resources/db`) → `app` (GHCR image, waits for `migrate` to complete successfully).

```sh
cp .env.example .env
# set:
#   POSTGRES_PASSWORD        (any strong password; avoid URL-special chars — it's embedded in DATABASE_URL)
#   SESSION_ENCRYPTION_KEY   (openssl rand -base64 32)
docker compose -f docker-compose.prod.yml up -d
```

Compose fails fast with a clear message if `POSTGRES_PASSWORD` or `SESSION_ENCRYPTION_KEY` is missing (`${VAR:?}`).

Optional vars:

| Var | Default | Notes |
|---|---|---|
| `POSTGRES_USER` | `patchworks` | |
| `POSTGRES_DB` | `patchworks` | |
| `HTTP_PORT` | `8080` | host port for the app |
| `PATCHWORK_IMAGE_TAG` | `latest` | pin a specific image version |

Redeploy (host has the repo checkout):

```sh
git pull
POSTGRES_PASSWORD=... SESSION_ENCRYPTION_KEY=... docker compose -f docker-compose.prod.yml up -d
```

`git pull` brings new migration files for the `migrate` service; `up -d` pulls the current app image,
re-runs migrate (no-op if nothing is pending), and restarts the app.

Notes:

- The `migrate` service binds `./resources/db` (migrations + schema) read-only — run from a repo checkout. Migrations come from the checkout, not from the app image.
- `app` uses `depends_on: migrate: service_completed_successfully`: if migrate fails, the app does not start (fail fast).
- `migrate` re-runs on every `up -d`; dbmate is idempotent (only pending migrations apply).
- App env: `HOST=0.0.0.0`, `ENVIRONMENT=production`, `SESSION_COOKIE_SECURE=true` (served over HTTPS or localhost only).
- Postgres data persists in the `db_data` volume.
