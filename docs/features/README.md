# Feature tickets

Feature tickets are the durable, detailed contracts for Patchworks' product behavior. `docs/spec.md` remains the high-level product summary; these files hold the current behavior, boundaries, and acceptance criteria for each ID family.

Read the category file for the ticket you are changing, then the linked area document for implementation policy. Feature files describe **what** the product does; area documents describe **how** the repository is built and operated.

## Categories

| Family | File | Current tickets |
|---|---|---:|
| Accounts & authentication | [authentication.md](authentication.md) | 7 (`AU-*`) |
| Bookmarks | [bookmarks.md](bookmarks.md) | 15 (`BK-*`) |
| Collections & sharing | [collections.md](collections.md) | 9 (`CL-*`) |
| Tags | [tags.md](tags.md) | 4 (`TG-*`) |
| Search & filtering | [search-filtering.md](search-filtering.md) | 5 (`SR-*`) |
| UI & interaction | [interaction.md](interaction.md) | 5 (`UX-*`) |
| Roadmap | [roadmap.md](roadmap.md) | 14 current (`FR-*`) |

Current status counts across the 59 category tickets:

| Family | Implemented | Partial | Planned |
|---|---:|---:|---:|
| `AU-*` | 3 | 4 | 0 |
| `BK-*` | 8 | 7 | 0 |
| `CL-*` | 4 | 5 | 0 |
| `TG-*` | 1 | 3 | 0 |
| `SR-*` | 2 | 3 | 0 |
| `UX-*` | 2 | 3 | 0 |
| `FR-*` | 0 | 0 | 14 |
| **Total** | **20** | **25** | **14** |

The feature-set summary is in [`docs/spec.md`](../spec.md#5-feature-set-current); the roadmap is in [`docs/spec.md`](../spec.md#7-future-developments). Access rules are consolidated in [`docs/spec.md`](../spec.md#6-access-control--permissions--impl).

## Status and priority

- `planned` — specified future work; no current implementation claim.
- `in-progress` — implementation is active but not yet a complete contract.
- `partial` — the main path exists, but a documented boundary or sub-behavior is missing.
- `implemented` — the stated acceptance criteria are supported by current code/tests or an explicit, verified non-code artifact.
- `deferred` — intentionally not being pursued at the current stage.
- `superseded` — historical ID whose scope landed under another ticket; provenance remains in the spec revision table and related-ticket links.

Priority applies to open work only: `P0` is a release-blocking or security/correctness issue, `P1` is important product or contract work, and `P2` is a smaller follow-up. Implemented tickets have no active priority. Do not create `RV-*` identifiers for review findings: put product findings under the existing ticket's **Open work** section, cross-cutting operational or policy findings in the relevant area document, and durable decisions in an ADR.

## Ticket-file template

Each active ticket should have a compact index entry and a section with:

- **Status**, **Priority**, and a link to its spec section.
- **Intent/behavior** and the rules that define the contract.
- **Authorization** and visibility boundaries where relevant.
- **UX/no-JS behavior** and failure semantics.
- **Acceptance criteria** that can be checked against code, tests, or a deliberate design decision.
- **Implementation notes**, linked to the relevant area doc or source/test when useful.
- **Related tickets** and an explicit **Open work** subsection (use `None recorded` when there is no known unresolved finding).

Feature files are current contracts, not commit histories. Git history and the spec revision table carry provenance and rationale.