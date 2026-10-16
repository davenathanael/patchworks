# Collections & sharing (`CL-*`)

**Scope:** collection CRUD, membership, role enforcement, and bookmark membership editing. Product intent is in [`docs/spec.md`](../spec.md#53-collections--sharing--impl--partial); the normative role/error matrix is [`docs/spec.md`](../spec.md#6-access-control--permissions--impl). Area policy: [`docs/architecture.md`](../architecture.md), [`docs/db.md`](../db.md), and [`docs/frontend.md`](../frontend.md).

## Index

| ID | Status | Priority | Contract |
|---|---|---|---|
| [CL-1](#cl-1--create-a-collection) | implemented | — | Create a named collection |
| [CL-2](#cl-2--list-your-collections) | implemented | — | List memberships and counts |
| [CL-3](#cl-3--browse-a-collection) | partial | P1 | View a collection's bookmark list; filtering is open work |
| [CL-4](#cl-4--edit-or-delete-a-collection) | implemented | — | Change details or remove a collection |
| [CL-5](#cl-5--invite-members-and-offer-manageable-collections) | partial | P1 | Add members and expose manageable choices |
| [CL-6](#cl-6--remove-members) | partial | P1 | Remove a member without racing the last owner |
| [CL-7](#cl-7--view-shared-bookmarks) | implemented | — | Share collection visibility, retain authors |
| [CL-8](#cl-8--edit-bookmark-collection-membership) | partial | P1 | Edit authorized membership links |
| [CL-9](#cl-9--enforce-collection-roles) | partial | P1 | Enforce and test owner/editor/viewer boundaries |

## CL-1 — Create a collection

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`CL-1`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** A signed-in user submits a name and optional description to `/collections`. The repository creates the collection and adds the creator as its owner in one transaction, then redirects to `/collections`.
- **Authorization:** Any authenticated user may create a collection; the creator becomes its first owner.
- **UX/no-JS:** The create page and form are server-rendered. A missing name re-renders inline with the submitted description; malformed form data shows a form-level error.
- **Failure/semantics:** Name is the only required field; description is optional. Storage failures are unexpected and do not claim creation.
- **Acceptance criteria:** Valid input creates one collection and owner membership; invalid input creates nothing and preserves values; creator ownership is atomic with collection creation.
- **Implementation notes:** [`internal/http/handlers/collections.go`](../../internal/http/handlers/collections.go), [`internal/db/collections.go`](../../internal/db/collections.go), and collection handler/repository tests.
- **Related:** [CL-2](#cl-2--list-your-collections), [CL-5](#cl-5--invite-members-and-offer-manageable-collections), [CL-9](#cl-9--enforce-collection-roles).
- **Open work:** None recorded.

## CL-2 — List your collections

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`CL-2`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** `/collections` lists collections in which the authenticated user is a member, with bookmark and member counts. The same membership data supplies the collection navigation/filter choices.
- **Authorization:** The query is scoped to the caller's memberships. A user does not receive unrelated collections in the list.
- **UX/no-JS:** Cards link to collection detail and show description/counts without JavaScript.
- **Failure/semantics:** Repository errors are unexpected; an empty membership set is a normal empty list.
- **Acceptance criteria:** Each visible collection includes its name, description when present, bookmark count, and member count; a non-member's collection is absent; creator membership appears immediately after creation.
- **Implementation notes:** [`internal/db/collections.go`](../../internal/db/collections.go), [`resources/db/queries/collections.sql`](../../resources/db/queries/collections.sql), and [`internal/http/views/collections.go`](../../internal/http/views/collections.go).
- **Related:** [CL-1](#cl-1--create-a-collection), [CL-3](#cl-3--browse-a-collection), [BK-13](bookmarks.md#bk-13--select-multiple-collections-at-save).
- **Open work:** None recorded.

## CL-3 — Browse a collection

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`CL-3`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** `/collections/{id}` shows collection metadata, its member list, and a keyset-paged list of its non-archived bookmarks with tags. Collection-page search/filtering is future scope, not part of the current ticket; the current page has no search/filter controls.
- **Authorization:** The handler resolves the caller's membership before loading the collection. Any member may view; non-members and nonexistent IDs receive 404. A member with an insufficient role for an action receives 403.
- **UX/no-JS:** Full page navigation and paging work without JavaScript; htmx pager requests return rows plus an out-of-band pager. No collection-page search/filter controls are rendered. Collection and member controls are conditionally rendered by role; bookmark row actions are not yet fully capability-aware and are tracked under CL-9 and UX-1.
- **Failure/semantics:** Archived bookmarks are excluded. Collection lookup, page loading, and rendering errors are unexpected; access errors use the shared 404/403 mapping.
- **Acceptance criteria:** Members can view the bookmark list; non-members cannot distinguish a private collection from a missing one; the list is newest-first, tag-aware, and paged by BK-15; member list is read-only for viewers. Collection-page search/filter is an explicit open-work extension, not current behavior.
- **Implementation notes:** [`internal/http/handlers/collections.go`](../../internal/http/handlers/collections.go), [`internal/db/collections.go`](../../internal/db/collections.go), [`internal/http/views/collections.go`](../../internal/http/views/collections.go), and handler/integration tests.
- **Related:** [CL-7](#cl-7--view-shared-bookmarks), [CL-8](#cl-8--edit-bookmark-collection-membership), [CL-9](#cl-9--enforce-collection-roles), [SR-1](search-filtering.md#sr-1--search-title-and-url), [BK-15](bookmarks.md#bk-15--page-browse-lists).
- **Open work:**
  - **P1 — Add collection-page search/filter:** render the shared search/filter controls, authorize and scope queries to the current collection, preserve filters across cursor links, and test plain-GET/no-JS plus htmx responses.

## CL-4 — Edit or delete a collection

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`CL-4`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** Owners and editors can edit a collection's name/description. Owners can delete it. Deletion removes the collection and cascades its bookmark links, but does not delete the underlying bookmarks.
- **Authorization:** The route checks the caller's role on every request: owner/editor may edit details; only owner may delete. Insufficient role is 403; non-member/nonexistent is 404.
- **UX/no-JS:** Edit is a normal form with inline validation; delete is a normal form/action with no client dependency.
- **Failure/semantics:** Missing name re-renders the edit form and preserves values. Repository failures are unexpected; no success redirect is emitted on failure.
- **Acceptance criteria:** Authorized edits persist and redirect to the collection; unauthorized edits/deletes do not mutate data; deleting a collection leaves its bookmarks intact and removes membership/links.
- **Implementation notes:** [`internal/http/handlers/collections.go`](../../internal/http/handlers/collections.go), [`internal/db/collections.go`](../../internal/db/collections.go), and collection tests.
- **Related:** [CL-1](#cl-1--create-a-collection), [CL-9](#cl-9--enforce-collection-roles), [CL-6](#cl-6--remove-members).
- **Open work:** None recorded.

## CL-5 — Invite members and offer manageable collections

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`CL-5`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** An owner submits an email and role. The current implementation adds an existing registered user to the collection; the default role is viewer, and the UI offers viewer/editor. The create-bookmark and collection-edit pickers expose only collections in which the current user can manage bookmarks (owner/editor), not every membership.
- **Authorization:** Only a collection owner can add/remove members. The server validates the requested role against the three fixed roles and resolves the invited email to a user; it does not create invitations for unregistered addresses.
- **UX/no-JS:** The member form is server-rendered and works without JavaScript. The collection picker is complete without JavaScript; its search box is an enhancement.
- **Failure/semantics:** An invalid role, unknown email, duplicate membership, or storage error is not converted into a successful membership. Viewer-only memberships do not qualify for bookmark-management pickers.
- **Acceptance criteria:** Owner can add an existing user with viewer/editor role; omitted role defaults to viewer; editor/viewer cannot add members; create/edit bookmark pickers contain only owner/editor collections. **Current gap:** the product wording says “invite,” but there is no pending-invitation or unregistered-email flow; the current behavior is direct membership addition for registered users.
- **Implementation notes:** [`internal/http/handlers/collections.go`](../../internal/http/handlers/collections.go), [`internal/db/collections.go`](../../internal/db/collections.go), [`internal/http/views/collections.go`](../../internal/http/views/collections.go), and tests. The future OAuth design is separate: [FR-2](roadmap.md#fr-2--oauth--oidc-login).
- **Related:** [CL-1](#cl-1--create-a-collection), [CL-6](#cl-6--remove-members), [CL-9](#cl-9--enforce-collection-roles), [BK-13](bookmarks.md#bk-13--select-multiple-collections-at-save).
- **Open work:**
  - **P1 — Clarify invitation semantics:** reconcile the ticket title/spec wording with the implemented direct-add behavior. If invitations are required, specify token, expiry, notification, acceptance, and revocation before adding schema or UI claims; otherwise rename the contract without changing IDs.
  - **P1 — Keep collection form limits centralized:** member/create/edit decoders use the shared request-body limit tracked in UX-1; retain member-specific email/role validation and expected oversized-input tests here.

## CL-6 — Remove members

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`CL-6`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** An owner removes another member from a collection. The repository checks the current owner count before deletion, but the read/check/delete sequence is not one transaction or lock, so concurrent removals can both pass the last-owner guard.
- **Authorization:** Only owners may remove members. Viewers/editors receive 403; non-members receive 404. The current member UI hides removal of the current user and only renders controls for owners.
- **UX/no-JS:** A normal form posts to the member deletion route and redirects back to the collection.
- **Failure/semantics:** Invalid member IDs are not found. Last-owner protection is enforced before deletion; a failed removal leaves membership intact.
- **Acceptance criteria:** Owner can remove a non-last-owner member; concurrent requests cannot remove every owner; unauthorized roles cannot remove anyone; removal does not delete the member's account or authored bookmarks; a last-owner conflict returns a stable expected outcome.
- **Implementation notes:** [`internal/http/handlers/collections.go`](../../internal/http/handlers/collections.go), [`internal/db/collections.go`](../../internal/db/collections.go), and collection/integration tests.
- **Related:** [CL-5](#cl-5--invite-members-and-offer-manageable-collections), [CL-9](#cl-9--enforce-collection-roles).
- **Open work:**
  - **P1 — Make last-owner removal atomic:** perform owner lookup/check/delete in one transaction with suitable locking or a conditional delete, and add concurrent database integration coverage.

## CL-7 — View shared bookmarks

- **Status:** `implemented`
- **Priority:** —
- **Spec:** [`CL-7`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** A collection member can view the collection and its bookmark links. A shared bookmark retains the original author rather than being reparented to the viewer or collection owner.
- **Authorization:** View access is collection membership-based and role-independent for owner/editor/viewer. The shared bookmark's author remains visible in the domain model and collection query.
- **UX/no-JS:** Collection detail is a full server-rendered page; bookmark links work normally.
- **Failure/semantics:** Non-members cannot access the collection page. Archived bookmarks remain hidden from collection browse.
- **Acceptance criteria:** Every member role can view shared links; a member cannot see a non-member collection through the detail route; the original author is not changed by viewing or collection membership.
- **Implementation notes:** [`internal/core/bookmarks.go`](../../internal/core/bookmarks.go), [`resources/db/queries/bookmarks.sql`](../../resources/db/queries/bookmarks.sql), and collection/integration tests.
- **Related:** [CL-3](#cl-3--browse-a-collection), [CL-8](#cl-8--edit-bookmark-collection-membership), [CL-9](#cl-9--enforce-collection-roles), [BK-6](bookmarks.md#bk-6--model-a-saved-bookmark).
- **Open work:** None recorded.

## CL-8 — Edit bookmark collection membership

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`CL-8`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** An author or an owner/editor member of a containing collection can open “Edit collections” from a bookmark row. The panel has a search field and checkbox list of manageable collections, pre-checks current memberships, and replaces the caller's manageable links in one save. Links to collections the caller cannot manage survive unchanged. From a collection page, unchecking that collection removes the row from the current list.
- **Authorization:** The initial fetch is author-or-manager scoped; viewers and strangers receive 404. On save, every added or removed collection requires owner/editor rights. Unchanged links pass even if the caller has been demoted, preserving existing membership.
- **UX/no-JS:** htmx swaps the row to the panel and back; plain GET renders a full edit page and plain POST redirects to the dashboard or originating collection. Search is client-side enhancement only.
- **Failure/semantics:** A non-manager cannot open or mutate the panel. A failed authorization or update leaves links unchanged. IDs not belonging to the caller are filtered from replacement rather than granting access.
- **Acceptance criteria:** Current memberships are pre-checked; one save adds/removes the caller's manageable links atomically; shared/foreign links are untouched; current-collection removal removes the row; viewers and non-members cannot access the editor. **Current gap:** the server-side replacement boundary depends on the caller's current collection list and changed-link checks; the exact persistence/authorization behavior should remain covered when this path is changed.
- **Implementation notes:** [`internal/http/handlers/bookmarks.go`](../../internal/http/handlers/bookmarks.go), [`internal/db/bookmarks.go`](../../internal/db/bookmarks.go), [`internal/http/views/links.go`](../../internal/http/views/links.go), and handler/integration tests.
- **Related:** [BK-13](bookmarks.md#bk-13--select-multiple-collections-at-save), [CL-5](#cl-5--invite-members-and-offer-manageable-collections), [CL-7](#cl-7--view-shared-bookmarks), [CL-9](#cl-9--enforce-collection-roles), [UX-1](interaction.md#ux-1--server-rendering-with-progressive-enhancement).
- **Open work:**
  - **P1 — Surface hidden memberships:** show non-manageable existing memberships as read-only/preserved rather than silently omitting them, so a demoted editor can understand what a save will retain.
  - **P1 — Make authorization and replacement atomic:** recheck the bookmark's author-or-manager eligibility and every changed collection inside the write transaction (with appropriate locking), rather than authorizing in the handler and updating later; add concurrent membership/role integration tests.
  - **P1 — Reject malformed replacement IDs:** invalid submitted collection UUIDs currently disappear during parsing; return an expected field/form error and leave all links unchanged.

## CL-9 — Enforce collection roles

- **Status:** `partial`
- **Priority:** `P1`
- **Spec:** [`CL-9`](../spec.md#53-collections--sharing--impl--partial)
- **Intent/behavior:** Every collection route resolves the caller's role before acting. Fixed roles are owner, editor, and viewer. Owners can view, manage bookmarks, edit details, delete the collection, and manage members; editors can view, manage bookmarks, and edit details; viewers can view only. Member lists are visible read-only to all members.
- **Authorization:** The server, not hidden buttons, is the boundary. Non-members and nonexistent targets receive 404; authenticated members lacking the required role receive 403. The canonical matrix is in [`docs/spec.md`](../spec.md#62-collection-actions-cl-9).
- **UX/no-JS:** The UI hides edit/delete/member-management controls according to role, but plain requests are still checked server-side. Full pages and forms work without JavaScript.
- **Failure/semantics:** Invalid route IDs map to 404. Role lookup, collection lookup, and action failures are surfaced through the shared expected/unexpected error classifier.
- **Acceptance criteria:** Owner/editor/viewer/non-member behavior matches the matrix for view, edit, delete, and member actions; a viewer can read but not mutate; non-members cannot distinguish private/missing collections; role values are constrained to the three fixed values. **Current gap:** the route matrix is implemented in handlers, but authorization is not yet covered by composed-router tests across every role and mutation route.
- **Implementation notes:** [`internal/core/collections.go`](../../internal/core/collections.go), [`internal/http/handlers/collections.go`](../../internal/http/handlers/collections.go), [`resources/db/migrations/20260904095802_collection_role_check.sql`](../../resources/db/migrations/20260904095802_collection_role_check.sql), and handler/core tests. The role matrix is intentionally not duplicated in this file; see the spec link above.
- **Related:** [CL-3](#cl-3--browse-a-collection), [CL-4](#cl-4--edit-or-delete-a-collection), [CL-5](#cl-5--invite-members-and-offer-manageable-collections), [CL-6](#cl-6--remove-members), [CL-8](#cl-8--edit-bookmark-collection-membership), [BK-13](bookmarks.md#bk-13--select-multiple-collections-at-save), [BK-15](bookmarks.md#bk-15--page-browse-lists).
- **Open work:**
  - **P1 — Test the route security matrix:** exercise the composed chi router (not only handler fakes) for anonymous, non-member, viewer, editor, and owner across every collection GET/POST/DELETE route, asserting 404/403/2xx and unchanged data. Public/guest sharing remains [FR-15](roadmap.md#fr-15--collection-sharing-links), not a gap in the current member-sharing contract.
  - **P1 — Make shared-row capabilities explicit:** pass author/manager capability context into bookmark-row rendering, hide author-only edit/archive/reading actions, and provide a cancel/refresh path for non-author editors without weakening server authorization.
