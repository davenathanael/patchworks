# ADR 0001: Mocha/chocolate styling overhaul

**Status:** Accepted · **Date:** 2026-08-31

## Context

- The app shipped with default Open Props styling. User wanted a redesign: sharper corners (dislikes pill/rounded), a chocolate-y mood, compact density, and a real wide-screen layout.
- Direction follows the approved redesign mockup, kept under `docs/mockups/` (`overhaul-chocolate-wide.html`): frame parity between mockup and CSS is a hard requirement.
- No build step: gomponents server-rendered HTML, htmx partials, Open Props from CDN (`https://unpkg.com/open-props`, resolves to 1.7.23), app styles in one file (`resources/static/css/app.css`, ~800 LOC).
- Concern: hardcoded values were scattered (`gap: 0.25rem`, `font-size: 0.8rem`, …). Goal: Tailwind-style enumeration through CSS variables — pick from a scale, not invent numbers.

## Decision

1. **Palette is verbatim, not approximate.** Port the mockup's `oklch` `light-dark()` tokens into `:root` exactly. Do **not** map to Open Props brown/choco ramp steps — they are far more saturated/warmer and were visually rejected. Open Props covers spacing, typography, shadows, and error red only.
2. **Sharp, compact geometry.** 4px global radius, tight spacing at 16px base, Fraunces serif display face + Geist UI font (both loaded from fonts.bunny.net).
3. **Mobile-first with container queries.** Sidebar hidden on narrow, topnav shown; past ~46rem shell width the app becomes a framed shell (centered, bordered, hairline shadow). Everything nests inside a `@container viewport` so the breakpoint tracks the shell, not the viewport.
4. **Cascade layers** `reset, theme, base, layout, components, utilities, states;` with every rule in a layer (un-layered beats layered, so Open Props stays top-level via the CDN link tag). Native nesting throughout.
5. **Maximize variable usage.** Prefer the nearest Open Props token over a literal (`0.3rem → var(--size-1)` is 0.25rem — accepted per user's "want 0.25, use 0.2" rule). Exceptions only where the scale genuinely doesn't fit.

## Consequences

- Spacing, font sizes, weights, line-heights, letter-spacing, radii all become enumerated — new components inherit the same scale without new numbers, and values are tweakable in one place.
- Palette survives Open Props version churn (it does not reference Open Props color ramps). The size/font/lineheight/letterspacing tokens in use are long-stable in Open Props.
- Deliberate literals that remain (each documented in a comment or self-evident):
  - `--radius: 4px` / `--radius-sm: 2px` (global one-off; user-blessed)
  - shell: gutter `6rem`, radius `12px` (no Open Props token — scale is 2/5px/1/2/4/8rem), padding `1.4rem 2.25rem`, hairline shadow
  - control min-heights `2.15rem` / `1.8rem` (button hit targets), auth card `23rem`, card min-width `17rem`, sidebar `15.5rem`, nav-link `0.9rem`
  - `em`-based paddings/font sizes (already font-relative — tokenizing them would be a lie)
  - container breakpoint `46rem`, transition `120ms` (Open Props 1.7.23 has no `--speed-*` tokens)
- UI work stays uncommitted; the user commits.

## Pointers

- Design spec: `docs/mockups/overhaul-chocolate-wide.html`
- Palettes in CSS: `resources/static/css/app.css` `:root` / `@layer theme`
- Conventions: `docs/css-guidelines.md`, `docs/frontend.md`