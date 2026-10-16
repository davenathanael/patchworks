# Documentation & Planning Process

How Patchworks separates durable product contracts, implementation policy, decisions, and transient work. Applies to humans and agents alike.

## Document roles

- **`docs/spec.md`** is the high-level product summary and navigation point. It owns the feature-set summaries (§5), access control (§6), future intent (§7), non-goals (§3), and open questions (§8). Keep it concise and link to feature tickets rather than repeating their detailed acceptance criteria.
- **Feature docs** under `docs/features/` are the detailed **current contracts** for product behavior: scope, rules, boundaries, authorization, UX, failure semantics, acceptance criteria, and unresolved work. They are not commit-by-commit histories. Git history, the spec revision table, and ADRs carry provenance and rationale.
- **Area docs** (`docs/architecture.md`, `docs/auth.md`, `docs/db.md`, `docs/frontend.md`, `docs/testing.md`, `docs/config.md`, `docs/logging.md`, `docs/deploy.md`, and similar pages) explain how an area of the repository is built, operated, or verified. They hold implementation policy, architecture, patterns, and runbooks—not a catalog of product tickets or a second copy of a feature contract.
- **ADRs** under `docs/adr/` record the *why* behind substantial, load-bearing, or hard-to-reverse decisions, including alternatives and consequences. They are durable decision rationale, not task lists, status trackers, or implementation how-tos.

Reference the canonical document; do not duplicate its detail. When behavior and implementation policy meet, link from the feature doc to the relevant area doc (and ADR, if applicable).

## Feature tickets and category files

The existing `AU-*`, `BK-*`, `CL-*`, `TG-*`, `SR-*`, `UX-*`, and `FR-*` identifiers are **feature tickets**. Keep these IDs stable; do not renumber historical requirements. A ticket is durable even when its status changes, and a ticket file should be understandable without reading unrelated feature files.

Keep one category file per ticket family under `docs/features/`:

| Ticket family | Category file |
|---|---|
| `AU-*` | [`docs/features/authentication.md`](features/authentication.md) |
| `BK-*` | [`docs/features/bookmarks.md`](features/bookmarks.md) |
| `CL-*` | [`docs/features/collections.md`](features/collections.md) |
| `TG-*` | [`docs/features/tags.md`](features/tags.md) |
| `SR-*` | [`docs/features/search-filtering.md`](features/search-filtering.md) |
| `UX-*` | [`docs/features/interaction.md`](features/interaction.md) |
| `FR-*` | [`docs/features/roadmap.md`](features/roadmap.md) |

[`docs/features/README.md`](features/README.md) is the compact index and ticket-file template, not an additional category. Link tickets in the spec and relevant area docs to the category file; do not copy the full contract into the spec.

## Status and priority

Use exactly one of these status values for a feature ticket:

- `planned` — specified future work; no current implementation claim.
- `in-progress` — implementation is active but the contract is not complete.
- `partial` — the main path exists, but a boundary or sub-behavior is missing or uncertain.
- `implemented` — the stated acceptance criteria are supported by current code/tests or an explicit, verified non-code artifact.
- `deferred` — intentionally not being pursued at the current stage.
- `superseded` — a historical ticket whose scope landed under another ticket; preserve provenance in the spec revision table and related-ticket links.

A `Priority` field is for open work only and is mutable: `P0` means release-blocking or security/correctness work, `P1` means important product or contract work, and `P2` means a smaller follow-up. Implemented tickets have no active priority. Review findings are not a second permanent ID system: do not create `RV-*` IDs. Put a product finding under the existing ticket's **Open work** section; put a cross-cutting operational or policy finding in the relevant area doc, and put a durable decision in an ADR.

## Reading order for agents

1. Read `AGENTS.md`, then this process document.
2. Read [`docs/features/README.md`](features/README.md) and the category file for the ticket being changed.
3. Read the ticket's linked `docs/spec.md` section and the relevant area doc.
4. Read an ADR when the work depends on a load-bearing decision it records.
5. For the active task, consult the root `plan.md` when present; it is temporary scaffolding, not a competing source of truth.
6. Verify behavior claims against tests/code and, when provenance matters, targeted Git history. For historical tickets, start with the spec and revision history, then area docs, tests, Git history, and targeted code inspection; do not repeatedly reread the whole codebase. Mark uncertainty as `partial` instead of guessing.

## Ephemeral planning and migration

- **Small, clear tasks:** use no plan artifact; implement, review, and leave the change for the user.
- **Medium and larger tasks:** use the root `plan.md` for implementation-oriented scaffolding: goal, files, function/type names, order, risks, and verification. It is temporary and does not replace a feature ticket or area doc.
- `docs/plans/` is retired for ordinary implementation work. Do not create or maintain implementation plans there, and delete the root plan when its work is complete rather than archiving it by default.
- Before deleting a plan, migrate every durable item to its canonical home:
  - product behavior, boundaries, acceptance criteria, and unresolved product work → the existing feature ticket/category file;
  - repository implementation policy, architecture, operational runbooks, and cross-cutting practices → the relevant area doc;
  - a substantial, hard-to-reverse decision → an ADR;
  - only a concise product summary or a link needed for navigation → `docs/spec.md`.
  Temporary task breakdown, estimates, and implementation order have no durable home and may be discarded. After migration, check links and ensure no unique operational or product guidance was lost, then remove the plan.

## Documentation conventions

Keep documentation terse, scannable, and non-duplicating. Make normative rules clear, label current contracts as such, and use links to the single source of truth. Keep `AGENTS.md` a lean navigator; detailed topic knowledge lives in `docs/*.md`.
