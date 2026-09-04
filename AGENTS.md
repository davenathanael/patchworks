# Patchwork

Personal bookmark manager. Go 1.26.2, chi v5, gomponents, pgx/v5 + SQLC, email/password auth.

## Quick Reference

|Command|Action|
|---|---|
|`mise run dev`|Hot-reload dev server|
|`mise run test`|Run all tests|
|`mise run lint`|golangci-lint|
|`mise run check`|Lint + test|
|`mise run sql`|SQLC generate|
|`mise run compose`|Docker Compose up (Postgres)|
|`mise run migrate`|Run migrations|
|`mise run migration`|Create a new migration file (`dbmate new`)|

## Architecture

DDD-inspired: pure `core/` domain + interface-driven handlers + repository pattern. `main()` wires everything via `internal/components`. See `docs/architecture.md`.

```
cmd/server/            # entry point only
internal/
  core/                # pure domain types + interfaces, no framework deps
  auth/                # password auth service, private interfaces
  http/handlers/       # local interfaces + pure handler funcs
  http/middleware/     # chi middleware (auth, request logging)
  http/views/          # gomponents components, layouts, partials
  http/server/         # chi router setup
  db/                  # SQLC-generated + repository wrappers
  config/              # env-based config
  logging/             # slog logger setup
  components/          # DI container
```

## Conventions (load-bearing for every task)

- **Handlers**: inject interfaces, return concretes; pure handlers with I/O abstracted. Interface in the handler file, factory func, wire in router.
- **Pure functions**: prefer pure logic; extract mutable/stateful I/O to boundaries. Sans-IO where possible.
- **No global state / no `init()`** — everything wired via `components.New()`.
- **Views**: gomponents server-side HTML; check context flag for full-page vs partial render. Dot-import `maragu.dev/gomponents`, `maragu.dev/gomponents/html`.
- **Styling**: Open Props tokens (CDN) + plain CSS in `app.css`. No build step. **No inline styles.** Classless base elements, semantic HTML, unprefixed modifiers. **Read `docs/css-guidelines.md` before writing any CSS**: styles must go in the correct `@layer`, use native nesting (never flat child selectors like `.note .note-text`), use app tokens, and declare standard properties before prefixed aliases (`line-clamp` before `-webkit-line-clamp`). See `docs/css-guidelines.md`.
- **Tests**: `github.com/carlmjohnson/be` (not testify); local fakes, no mock libs; table-driven only for pure functions. See `docs/testing.md`.
- **DB**: never hand-write query structs — SQLC-generated; repos wrap sqlc in `internal/db/`. See `docs/db.md`.
- **Config**: env-based via `caarlos0/env`; no secrets in code. See `docs/config.md`.
- **Logging**: slog, canonical one-line-per-request; error level only for unexpected errors. See `docs/logging.md`.
- **Auth**: email/password (argon2id), encrypted session cookie. See `docs/auth.md`.
- **Error handling**: `fmt.Errorf` wrapping; validate at boundaries; trust internal contracts.
- **Minimal comments**: explain why, not what.

## Agent Delivery (terse output)

- Unless the user explicitly asks for explanation, respond with **minimal prose**.
- Default delivery: what changed (files/symbols), commands run, verification result.
- No narrative, no step-by-step recap, no redundant summaries of obvious work.
- Add prose only when it carries new information.

## Docs discipline (read before adding context files)

Keep this file a **lean navigator**; keep detailed topic knowledge in `docs/*.md`. Rules for adding any new `.md` context:

1. **AGENTS.md stays short.** It is the only file loaded every session. Do not paste full topic content here.
2. **New topic → add a file in `docs/`** (e.g. `docs/feature-x.md`), written in neutral language that serves both humans and agents.
3. **Reference, don't duplicate.** Add a one-line lazy pointer in AGENTS.md (e.g. `See docs/feature-x.md`) and *pull detail on demand* — never inline a copy.
4. **Single source of truth.** One canonical copy per topic. If detail exists in a doc, point to it; don't re-describe it in AGENTS.md or create a second file elsewhere.
5. **No harness-specific files.** Don't add `CLAUDE.md`, `.cursorrules`, `opencode.json` instructions, etc. — AGENTS.md + docs/ works across harnesses.
6. **Terse delivery applies to docs too.** Write docs that are scannable: headings, bullets, commands. No filler.

## Reference Docs

- `docs/spec.md` — product spec: feature set (req IDs), roadmap, non-goals
- `docs/process.md` — how work is planned & documented (spec lifecycle, plan files → `docs/plans/`, ADRs)
- `docs/architecture.md` — layers, handler pattern, key decisions
- `docs/frontend.md` — gomponents, Open Props + plain CSS styling
- `docs/css-guidelines.md` — modern CSS/HTML guidelines (cascade layers, tokens, layout, support tiers)
- `docs/db.md` — SQLC workflow, repository pattern, migrations
- `docs/auth.md` — password auth, session/cookie details
- `docs/config.md` — env sections
- `docs/logging.md` — slog conventions
- `docs/testing.md` — approach, fakes, assertions
- `docs/adr/0001-styling-overhaul-mocha.md` — ADR: styling overhaul decisions (palette, variables, exceptions)

## Token-Saving Tips

- Keep AGENTS.md a navigator; lazy-load `docs/*.md` on a need-to-know basis.
- Use a read-only scout/subagent for codebase mapping instead of reading file after file.
- Grep SQLC queries instead of reading generated files.
- Read file sections (offset/limit) instead of whole files.
- Default to the cheapest adequate model; escalate only for hard reasoning tasks.
- Let long sessions compact/drop old context instead of re-sending it every turn.
