# Documentation & Planning Process

How Patchwork records decisions, plans work, and keeps the spec alive. Applies to human and agent work alike.

## Source of truth: `docs/spec.md`

- spec.md is the single source of truth for the **feature set** (§5), **access control** (§6), **future intent** (§7), **non-goals** (§3), and **open questions** (§8).
- Out of ideas for what to build? Read spec.md §7 — that is the backlog.
- Feature lifecycle inside spec.md:
  - An implemented FR-* item moves from §7 to the relevant §5 requirement and is renumbered into that section's ID family (e.g. FR-22 → BK-7); provenance is recorded in the Rev table (e.g. "FR-22 landed in §5.2 as BK-7").
  - Rejected items move to §3 Non-goals with a one-line *why*.
  - Answered questions leave §8; the answer becomes normative text in §5/§7.
  - Bump the Rev table on every normative change.

## Planning work

- **Small/clear tasks:** no plan artifact — implement immediately, review, commit.
- **Medium+ tasks:** the planning stage may produce a plan (goal, files, function/type names, order of changes, risks). A plan is implementation-oriented — never a product spec.
- **Plan files are transient; nothing plan-related persists at the project root:**
  - In-flight plans live at the repo root as `plan.md` (or `plan-{slug}.md`) so the current session and reviewer share them.
  - When the work lands: delete the plan, or archive it to `docs/plans/plan-{name}.md` if it holds lasting value (e.g. a large expedition worth re-reading).
  - Plans written long before implementation (no date in sight) go straight to `docs/plans/plan-{name}.md`.

## Decisions: ADR

- Load-bearing or hard-to-reverse decisions get a `docs/adr/NNNN-slug.md` (see `docs/adr/0001-styling-overhaul-mocha.md`). Prefer an ADR over expanding a plan or relying on chat history.
- Document decisions, not implementation detail — code and `docs/` pages are the reference for *how*.

## Doc discipline

- Everything stays terse, scannable, and non-duplicating: reference, don't repeat.
- `AGENTS.md` stays a lean navigator; detailed topic knowledge lives in `docs/*.md` (see AGENTS.md "Docs discipline").