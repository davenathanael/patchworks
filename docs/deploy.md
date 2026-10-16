# Deployment

CI validates pushes and pull requests, tags publish a multi-architecture image to GHCR, and `docker-compose.prod.yml` runs the self-hosted stack.

## Continuous integration (`.github/workflows/ci.yml`)

Every push and pull request starts two independent jobs:

- **Lint + unit tests** — `mise run lint`, then `mise run test`.
- **Integration tests (Postgres)** — starts a health-gated `postgres:18-alpine` service, sets `TEST_DATABASE_URL`, and runs `mise run test-integration` against it.

`jdx/mise-action` installs the tools from [`mise.toml`](../mise.toml); Go is pinned to 1.26.2 and golangci-lint to 2.11.4. The release workflow is a separate workflow with no dependency on these jobs, so a tag can publish even when CI is failing or has not completed.

## Image releases (`.github/workflows/release.yml`)

Pushing a tag matching `v*` builds `linux/amd64` and `linux/arm64` images and pushes them to `ghcr.io/davenathanael/patchworks`:

- the semantic version without the leading `v` (for example, `v1.2.3` produces `1.2.3`)
- mutable tag `latest`

The workflow pushes an image only; it does not create a GitHub Release. It uses `packages: write` and GitHub's Actions token. A newly published package is private by default: make the GHCR package public or authenticate each deployment host with `docker login ghcr.io`.

The [`Dockerfile`](../Dockerfile) is a multi-stage build: `golang:1.26.2-alpine` produces a static Linux binary, then `gcr.io/distroless/static-debian12:nonroot` receives the binary, static resources, and CA certificates. `CGO_ENABLED=0`, `GOOS=linux`, `-trimpath`, and `-ldflags="-s -w"` make the runtime image independent of the build host. The final image has no shell; debug it with logs and purpose-built diagnostics rather than `docker exec sh`. The [`.dockerignore`](../.dockerignore) keeps Git metadata, workflows, docs, local environment files, coverage output, and logs out of the build context.

For a local image:

```sh
docker build -t patchworks:local .
```

## Production stack (`docker-compose.prod.yml`)

The fixed Compose project name is `patchworks-prod`. It is separate from the default development project so the two stacks do not share services or the `db_data` volume.

| Service | Image and behavior |
|---|---|
| `db` | `postgres:18-alpine`; data persists in the `patchworks-prod_db_data` volume. `pg_isready` runs every 5 seconds with a 3-second timeout and 10 retries. |
| `migrate` | `amacneil/dbmate:2.32.0`; waits up to 60 seconds for the database, then runs `up`. It has no restart policy and exits after success or failure. |
| `app` | `ghcr.io/davenathanael/patchworks:${PATCHWORK_IMAGE_TAG:-latest}`; publishes `${HTTP_PORT:-8080}:8080` and restarts unless stopped. |

Startup order is `db` healthy → `migrate` exits successfully → `app` starts. A migration failure prevents a newly created app container from starting. `migrate` is started on each `docker compose up`; dbmate applies only pending migrations and exits without changes when none remain.

### Configuration

Compose reads values from the shell or a `.env` file at the project directory. `.env.example` is development-oriented; add the production values below rather than relying on its local `DATABASE_URL`.

| Variable | Required | Default | Notes |
|---|---:|---|---|
| `POSTGRES_PASSWORD` | yes | — | Used in the database URL. Use a strong value without URL-special characters unless they are correctly URL-encoded. |
| `SESSION_ENCRYPTION_KEY` | yes | — | Base64 for exactly 32 decoded bytes; generate once with `openssl rand -base64 32` and retain it across restarts. |
| `POSTGRES_USER` | no | `patchworks` | Database role used by both the database and migration/application URLs. |
| `POSTGRES_DB` | no | `patchworks` | Initial database name. Changing it does not rename an existing volume's database. |
| `HTTP_PORT` | no | `8080` | Host port mapped to the app's container port 8080. |
| `PATCHWORK_IMAGE_TAG` | no | `latest` | Pin a released version in production; `up -d` alone does not refresh a mutable tag. |

Compose hard-codes the app's `HOST=0.0.0.0`, `PORT=8080`, `ENVIRONMENT=production`, and `SESSION_COOKIE_SECURE=true`. It also builds `DATABASE_URL` for the internal `db` service; do not replace that URL with the localhost URL from `.env.example`.

The two required variables use Compose's fail-fast interpolation (`${VAR:?}`). The application additionally validates the session key after opening the database, so database failures can be reported before a malformed key.

### Host checkout and migrations

The migration image contains dbmate, **not this repository's migrations**. The service bind-mounts `./resources/db:/db:ro` from the host and reads `resources/db/migrations/` plus `resources/db/schema.sql`. The relative bind path is resolved from the Compose project directory, so run the command from the checkout or otherwise ensure that directory contains the intended `resources/db` tree.

There are currently **7 migration files** in the checkout. On a fresh database, `migrate` applies all 7; an existing database may already have some or all versions recorded in `schema_migrations`. Migration files and the `app` image are selected independently: pinning `PATCHWORK_IMAGE_TAG` does not pin or fetch the checkout's migrations.

`schema.sql` is a dbmate-generated dump, not an executable migration. The read-only production mount means dbmate cannot persist a regenerated schema into the checkout. Keep the checked-in dump synchronized through the development migration workflow; compare the live schema with the checkout before a release and investigate drift before migrating production.

## Deploying a release

From a host checkout containing the release's Compose file and matching migration files:

```sh
export POSTGRES_PASSWORD='...'
export SESSION_ENCRYPTION_KEY='...'
export PATCHWORK_IMAGE_TAG='1.2.3'  # recommended; omit only when intentionally tracking latest

# Refresh the pinned app before starting the stack. For a private package, log in first.
docker compose -f docker-compose.prod.yml pull app
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml ps -a
docker compose -f docker-compose.prod.yml logs migrate app
```

`docker compose up -d` does not automatically pull a newer image. Run `pull` explicitly before deployment. Setting `PATCHWORK_IMAGE_TAG` changes the requested tag, but a missing tag may still require `pull` before `up` can use it. To track `latest`, run `pull app` every time and accept that this is a mutable deployment.

To deploy a checkout revision with new migrations, update or pin the checkout first, then run `up -d`; the bind mount exposes the checkout's files to the new one-shot migration container. A Compose file copied without `resources/db/migrations/` is not a complete release artifact.

## Health and operational status

- `db` has a Compose healthcheck and gates both other services.
- `migrate` is a one-shot job, not a long-running healthchecked service. Check that it reached `Exited (0)`; any other exit code means migrations did not complete successfully.
- `app` has **no Docker healthcheck and no liveness/readiness endpoint**. A running container only means the process stayed up; it does not prove current database connectivity or application readiness. Use `docker compose ps`, startup/request logs, and an authenticated smoke flow as the present checks.
- The production app serves plain HTTP on the published port. Put a TLS-terminating reverse proxy in front of it. With `SESSION_COOKIE_SECURE=true`, browsers will not return the session cookie over ordinary remote HTTP, so login fails without HTTPS.

## Data safety, replacement, and rollback limits

The stack has no backup schedule, retention policy, restore automation, or restore verification. The named volume survives normal container replacement and `docker compose down`, but `docker compose down -v` deletes it. Before a risky migration or replacement, capture and test a PostgreSQL backup outside Docker and rehearse restoring it with the intended application version.

A release may add forward-only schema changes. Repointing `app` to an older image does not undo those database changes, and the startup sequence does not automatically restore the prior image or database. Treat schema compatibility as part of release selection:

1. Keep a pre-deployment backup.
2. Deploy the selected image only after its matching migration set has succeeded.
3. If the new app is unhealthy, stop traffic and assess whether the old image remains schema-compatible.
4. If data or schema must be reverted, restore the verified database backup together with the matching application and migration revision. Do not use `migrate down` as an automatic production rollback; individual down migrations are destructive and are not rehearsed by this stack.

Changing `POSTGRES_USER` or `POSTGRES_PASSWORD` does not update credentials in an existing Postgres volume. Restore from a backup or perform an explicit database credential rotation before relying on changed values. Changing `SESSION_ENCRYPTION_KEY` makes existing encrypted session cookies unreadable and logs users out even though session rows remain in PostgreSQL.

## Known limitations and backlog

- **Migrations are not bundled with the release image.** A deployment still needs a host checkout (or, at minimum, a Compose file plus a matching `resources/db` tree). The app image and checkout can drift.
- **Package the migration runner.** Build and publish a pinned migration image with the same release tag, copy `resources/db` into it, remove the bind mount, and run `docker compose pull` before `up -d`. This makes the Compose file plus the two versioned images the deployable unit; the Compose file itself still lives on the host.
- **Release gating is absent.** Tag publication does not wait for lint, unit, integration, image verification, or provenance. Immutable version/digest selection and a documented release gate are still needed.
- **Health and readiness are incomplete.** Add bounded application liveness/readiness checks and operational diagnostics; the current Postgres healthcheck only covers database startup.
- **Schema integrity is not enforced in production.** The checked-in dump can lag the migration set, and production cannot update it through the read-only mount.
- **Backup/restore is not implemented.** Before relying on self-hosting, define backups, retention, restore commands, integrity checks, and a rehearsed upgrade/rollback drill.
- **Runtime images and tool versions are only partly pinned.** Go and golangci-lint are pinned in `mise.toml`, and the Compose dbmate image is versioned. SQLC, watchexec, mise's local dbmate, GitHub Actions, base images, and app image tags such as `latest` remain mutable or unpinned.
- **Credential URLs are fragile.** `POSTGRES_PASSWORD` is interpolated without encoding into both service URLs; restrict it to safe characters or encode it correctly and test the resulting connection.
- **The app port is not safe by default.** Compose publishes the app on all host interfaces and the app serves plain HTTP; bind to localhost behind the documented TLS reverse proxy or provide an equivalent deployment guard before exposing it to a network.
- **Debugging is constrained.** The distroless app has no shell, and the PostgreSQL password is wrong against an existing volume, migration can fail while the app remains unstarted. Inspect service status and sanitized logs before retrying.
