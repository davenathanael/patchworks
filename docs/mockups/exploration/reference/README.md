# Design reference — layout & UI patterns

**These are reference material, not proposals and not the current design.**

They are kept because two patterns in here are worth carrying forward into whatever
comes next. They were built for Patchworks and still contain Patchworks content,
which is fine — they're being preserved as-is rather than genericised, so that the
proportions, spacing rhythm, and visual weight stay truthful.

## `split-sheet-reference.html`

Extracted from round 2, candidate 3. The part worth keeping:

- **The bottom sheet as the detail surface.** On narrow viewports the selected
  item's full record rises from the bottom edge as a sheet with a top edge
  shadow, so it reads as floating above the list rather than as a section of it.
  The same treatment suits every popover and popup panel in the app.
- **Sharp corners.** `--r: 2px` throughout. Buttons, inputs, chips, cards.
  Nothing is rounded. This is the visual correction applied to later candidates.
- **Actions permanently visible on rows.** No kebab menu, no hover-reveal.
  Every row shows its own action set, so nothing is hidden behind a gesture.
- **Top shadow on floating surfaces.** `0 -8px 32px` style elevation, used only
  for things that genuinely sit above content.

Not carried forward: the keyboard-first command model (j/k/s/r/⌘K) is a power-user
affordance that did not survive the "must be usable casually" requirement.

## `workbench-reference.html`

Extracted from round 1, candidate A. The part worth keeping:

- **The dense row.** Monospace metadata on a hairline rule, no card chrome. It is
  the densest readable list in any of the explorations and it aged the best.
- **The always-open inspector.** Notes and tags edited in place without leaving
  the list — no modal, no separate edit page, no navigation.
- **Real query syntax with a permanently visible operator legend.**

Not carried forward: the two-pane desktop layout, which read as a desktop app
that had been shrunk to fit a phone rather than designed for one.