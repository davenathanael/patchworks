# ADR 0002: Converged UI design system

**Status:** Accepted · **Date:** 2026-10-17 · **Supersedes:** [ADR 0001](./0001-styling-overhaul-mocha.md)

## Context

ADR 0001 set the styling direction in August 2026: mocha palette, 4px radius,
Fraunces + Geist, sidebar shell at a `46rem` breakpoint, cascade layers, token
discipline. Four rounds of design exploration followed (`docs/mockups/exploration/`),
comparing four divergent directions in round 1, three information architectures in
round 2, and a converged design with multiple skins in rounds 3 and 4.

The exploration invalidated parts of ADR 0001's visual decisions while confirming its
engineering discipline. A live ADR that is silently contradicted is worse than no ADR,
so this record supersedes it explicitly.

What the rounds established:

- The sidebar is a desktop affordance being punished on mobile. Navigation is a
  bottom tab bar on mobile and a top strip on desktop.
- Notes are long-form prose — the dominant content — and the design that serves them
  best is note-first, not row-first.
- Fraunces is a display face. Set at 15px across four lines it is uncomfortable to
  read; a prose face is required for notes.
- Two warm palettes were preferred over the alternatives, both muted rather than
  saturated.
- Dark mode needs to be designed, not derived by inverting the light values.

## Decision

### Carried forward from ADR 0001, unchanged

1. **Cascade layers** `reset, theme, base, layout, components, utilities, states`,
   every rule in a layer, native nesting throughout.
2. **No build step.** Server-rendered gomponents; Open Props from CDN for spacing,
   typography and shadows only — **never** for palette.
3. **The palette is a hand-authored `oklch` `light-dark()` token set**, not derived
   from any ramp. This was the most durable call in 0001 and still holds.
4. **Tokenise rather than hardcode**, with documented exceptions for values the scale
   genuinely cannot express.

### Superseded

| ADR 0001 | Now |
|---|---|
| One mocha palette | **Sepia** (default, muted warm grey) and **Mocha** (warmer), each in light and dark |
| `4px` radius | **`2px`**, sharp |
| Fraunces + Geist only | **Fraunces** (display only) · **Source Serif 4** (prose) · **Geist** (chrome) · **Geist Mono** or **JetBrains Mono** (metadata) |
| Sidebar shell past `46rem` | **Top strip + bottom tabs**, no sidebar; item breakpoint at **`54rem`** |
| Fractional literal spacing throughout | Enumerated scale, with ADR 0001 §5's exception list re-derived rather than carried |

### New

5. **Fraunces never sets prose.** It is a display face. Every note, description and
   long-form string is set in Source Serif 4. This is not a preference — measured
   evidence is in `docs/mockups/exploration/round3/type.html`.
6. **Dark mode is designed, not inverted.** Backgrounds sit at oklch L≈0.27, not the
   conventional 0.18–0.20. Text is measured against the real background: 13:1 body,
   8:1 secondary, 5.5:1 muted metadata.
7. **Focus is always visible and never signalled by colour alone.** The focus ring is
   a **fixed blue across all four theme combinations**, not the accent — it must read
   as "you are here" regardless of palette and stay legible on an accent fill.
8. **No state is conveyed by colour alone.** Every status carries a glyph *and* a
   word: `✓` checked, `○` unselected, `✕` error, `!` warning, dashed border for stale
   and for "creates something new".
9. **Depth is borders plus a button shadow.** Only genuinely floating surfaces — the
   inspection sheet, dialogs, toasts — carry a real shadow.

## Consequences

- `resources/static/css/app.css` is replaced, not patched: the palette, type system,
  radius and breakpoints all change, so editing in place would leave contradictory
  tokens behind.
- `internal/http/views/` is rewritten. The existing package is built around the
  sidebar shell and popover menus that this decision removes.
- `collections` gains a `handle` column distinct from `name`, because the `@` syntax
  and the display title are allowed to differ.
- The scheme is no longer a single palette. Any future skin is a token override, not
  a fork — the exploration proved the layout is stable across all of them.
- Feeds, share pages and bulk selection are explicitly out of scope for this rebuild.
  See `docs/implementation-brief.md` §10.

## Links

- Design: `docs/mockups/exploration/round4/`
- Implementation brief: `docs/implementation-brief.md`
- Type evidence: `docs/mockups/exploration/round3/type.html`
- Patterns kept for other projects: `docs/mockups/exploration/reference/`