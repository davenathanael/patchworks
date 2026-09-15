package views

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/davenathanael/patchworks/internal/core"
	"github.com/google/uuid"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// BookmarkForm is the shared add-bookmark form view-model: the ajg/form
// decode target and the render model. Zero value renders a fresh form.
type BookmarkForm struct {
	URL           string     `form:"url"`
	CollectionIDs []string   `form:"-"` // FR-24 picker: repeated keys, read via r.PostForm
	Tags          string     `form:"tags"`
	Notes         string     `form:"notes"`
	SaveAnyway    bool       `form:"save_anyway"`    // set on the warned re-render: confirmed duplicate
	Queued        bool       `form:"add_to_reading"` // "Add to reading list" checkbox (FR-10)
	Duplicate     *Duplicate `form:"-"`
	Errors        FormErrors `form:"-"`
}

// Duplicate is the FR-11 soft reminder: an existing bookmark of the same
// author with the exact same URL, offered with a Save-anyway override.
type Duplicate struct {
	Title    string
	SavedAt  time.Time
	Archived bool
}

// NewDuplicate builds the reminder model from the found bookmark.
func NewDuplicate(bm core.Bookmark) *Duplicate {
	return &Duplicate{Title: bm.Title, SavedAt: bm.CreatedAt, Archived: !bm.ArchivedAt.IsZero()}
}

// duplicateNotice renders the reminder line shown above the Save button.
func duplicateNotice(d Duplicate) string {
	msg := fmt.Sprintf("You already saved this link %s — %q.", relativeTime(d.SavedAt), d.Title)
	if d.Archived {
		msg += " That copy is archived."
	}
	return msg + " Save anyway to keep a second copy."
}

func NewBookmark(form BookmarkForm, collections []core.Collection) Node {
	return Details(
		Class("add"),
		Summary(Text("＋ Add bookmark")),
		Div(
			Class("panel"),
			NewBookmarkForm(form, collections),
		),
	)
}

// NewBookmarkForm renders the add-bookmark form fragment, optionally with
// field errors. htmx failure responses retarget the swap to
// #add-bookmark-form (the form's own hx-target is the bookmarks list).
// It doubles as a collections picker (FR-24): the edit-panel class powers the
// live client-side filter, and checked boxes link the new bookmark to those
// collections in the same save.
func NewBookmarkForm(form BookmarkForm, collections []core.Collection) Node {
	memberOf := make(map[string]bool, len(form.CollectionIDs))
	for _, cid := range form.CollectionIDs {
		memberOf[cid] = true
	}
	items := Map(collections, func(c core.Collection) Node {
		return Li(
			Label(
				Input(Type("checkbox"), Name("collection_id"), Value(c.ID.String()), If(memberOf[c.ID.String()], Checked())),
				Text(c.Name),
				Span(Class("count"), Text(fmt.Sprintf("%d", c.BookmarkCount))),
			),
		)
	})
	nodes := append([]Node{
		ID("add-bookmark-form"),
		Method("POST"),
		Action("/bookmarks"),
		Class("edit-panel"),
		Attr("hx-post", "/bookmarks"),
		Attr("hx-target", "#bookmarks"),
		Attr("hx-swap", "innerHTML"),
		Attr("hx-on::after-request", "if(event.detail.successful) this.reset()"),
		TextInput("URL", "url", "url", form.URL, form.Errors,
			ID("add-link"), Placeholder("https://example.com"), Required(),
			Attr("inputmode", "url"), Attr("enterkeyhint", "go"),
		),
		TextInput("Tags", "tags", "text", form.Tags, form.Errors, Placeholder("go, css, reading")),
		Label(Text("Notes"),
			Textarea(Name("notes"), Rows("3"), Placeholder("Optional notes"), Text(form.Notes)),
		),
		Div(Class("picker-title"), Text("Collections")),
		Input(Type("search"), Name("q"), Placeholder("Search collections"), Attr("enterkeyhint", "search")),
		Ul(Class("pick-list"), items),
		Label(
			Class("form-options"),
			Input(Type("checkbox"), Name("add_to_reading"), Value("true"), If(form.Queued, Checked())),
			Text("Add to reading list"),
		),
	}, duplicateNodes(form)...)
	return Form(nodes...)
}

// duplicateNodes renders the FR-11 reminder: notice line, save_anyway marker,
// and the Save-anyway submit; a fresh form gets the plain Save button.
func duplicateNodes(form BookmarkForm) []Node {
	if form.Duplicate == nil {
		return []Node{Button(Type("submit"), Text("Save"))}
	}
	return []Node{
		P(Class("muted"), Text(duplicateNotice(*form.Duplicate))),
		Input(Type("hidden"), Name("save_anyway"), Value("true")),
		Button(Type("submit"), Text("Save anyway")),
	}
}

func RecentLinks(page core.BookmarkPage, collections []core.Collection, pager ListPagerProps) Node {
	return Section(
		H2(Text("Recent")),
		IfElse(
			len(page.Items) > 0,
			Group{LinkList(pager.ListID, page.Items, collections, ""), ListPager(pager)},
			P(Class("muted"), Text("No links yet. Add one above to get started.")),
		),
	)
}

func FilteredLinksView(page core.BookmarkPage, collections []core.Collection, pager ListPagerProps) Node {
	return Section(
		H2(Text("Filtered Links")),
		IfElse(
			len(page.Items) > 0,
			Group{LinkList(pager.ListID, page.Items, collections, ""), ListPager(pager)},
			P(Class("muted"), Text("No links to display.")),
		),
	)
}

func IfElse(condition bool, trueNode, falseNode Node) Node {
	if condition {
		return trueNode
	}
	return falseNode
}

// LinkList renders the bookmark list UL, optionally with an element id so
// htmx pager buttons can target its rows.
func LinkList(id string, links []core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	return Ul(
		If(id != "", ID(id)),
		Class("link-list"),
		Map(links, func(link core.Bookmark) Node {
			return LinkRow(link, collections, currentCollectionID)
		}),
	)
}

// Links renders an untargeted bookmark list (no pager wiring).
func Links(links []core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	return LinkList("", links, collections, currentCollectionID)
}

func LinkRow(link core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	return Li(BookmarkArticle(link, collections, currentCollectionID))
}

// BookmarkArticle renders one bookmark row's article — the swap target for the
// row menu: the edit panel, the collections picker result, and cancellations
// all replace it. currentCollectionID is the page the row is rendered on
// ("" on the dashboard) — it lets the collections save drop the row when this
// collection is unchecked.
func BookmarkArticle(link core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	relTime := relativeTime(link.CreatedAt)

	tags := make([]Node, 0, len(link.Tags))
	for _, tag := range link.Tags {
		tags = append(tags, Li(Text(tag)))
	}

	return Article(
		Header(
			A(
				Href(link.URL.String()),
				Target("_blank"),
				Rel("noopener"),
				Text(link.Title),
			),
			Time(Attr("datetime", link.CreatedAt.Format(time.RFC3339)), Text(relTime)),
		),
		Footer(
			Small(Text(link.URL.Host)),
			Ul(tags...),
			Span(Class("row-actions"), bookmarkMenu(link, collections, currentCollectionID)),
		),
		noteBlock(link),
	)
}

// EditBookmarkPage is the no-JS fallback: a full page wrapping the edit panel.
func EditBookmarkPage(user core.User, panel Node) Node {
	return Page("Edit Bookmark — Patchworks", AppShell(user, Main(
		A(Class("back-link"), Href("/"), Text("← Dashboard")),
		panel,
	)))
}

// ArchivedPage lists the user's archived bookmarks — restore and permanent
// delete live here (FR-1 remainder).
func ArchivedPage(user core.User, bookmarks []core.Bookmark) Node {
	content := Main(
		Header(H1(Text("Archived"))),
		IfElse(len(bookmarks) > 0,
			Ul(Class("link-list"), Map(bookmarks, ArchivedRow)),
			P(Class("muted"), Text("Nothing archived yet.")),
		),
	)
	return Page("Archived — Patchworks", AppShell(user, content))
}

// ArchivedRow is the management row for the archived page: inline Restore and
// permanent Delete instead of the kebab menu.
func ArchivedRow(link core.Bookmark) Node {
	relTime := relativeTime(link.CreatedAt)
	tags := make([]Node, 0, len(link.Tags))
	for _, tag := range link.Tags {
		tags = append(tags, Li(Text(tag)))
	}
	restoreURL := fmt.Sprintf("/bookmarks/%s/restore", link.ID)
	deleteURL := fmt.Sprintf("/bookmarks/%s/delete", link.ID)

	return Li(
		Article(
			Header(
				A(
					Href(link.URL.String()),
					Target("_blank"),
					Rel("noopener"),
					Text(link.Title),
				),
				Time(Attr("datetime", link.CreatedAt.Format(time.RFC3339)), Text(relTime)),
			),
			Footer(
				Small(Text(link.URL.Host)),
				Ul(tags...),
				Span(Class("row-actions"),
					Group{
						Button(Class("button outline small"), Type("button"),
							Attr("hx-post", restoreURL),
							Attr("hx-target", "closest li"),
							Attr("hx-swap", "delete"),
							Text("Restore"),
						),
						Button(Class("button danger small"), Type("button"),
							Attr("hx-post", deleteURL),
							Attr("hx-confirm", "Delete permanently? This cannot be undone."),
							Attr("hx-target", "closest li"),
							Attr("hx-swap", "delete"),
							Text("Delete"),
						),
					},
				),
			),
			noteBlock(link),
		),
	)
}

// ReadingPage lists the user's FIFO reading queue; "Mark as read" dequeues
// (FR-10).
func ReadingPage(user core.User, bookmarks []core.Bookmark) Node {
	content := Main(
		Header(H1(Text("Reading list"))),
		IfElse(len(bookmarks) > 0,
			Ul(Class("link-list"), Map(bookmarks, ReadingRow)),
			P(Class("muted"), Text("Nothing queued yet.")),
		),
	)
	return Page("Reading list — Patchworks", AppShell(user, content))
}

// ReadingRow is a queued bookmark: plain external link plus "Mark as read",
// which deletes the row client-side via htmx (non-JS falls back to the hidden
// next redirect).
func ReadingRow(link core.Bookmark) Node {
	relTime := relativeTime(link.CreatedAt)
	tags := make([]Node, 0, len(link.Tags))
	for _, tag := range link.Tags {
		tags = append(tags, Li(Text(tag)))
	}
	readURL := fmt.Sprintf("/bookmarks/%s/reading", link.ID)

	return Li(
		Article(
			Header(
				A(
					Href(link.URL.String()),
					Target("_blank"),
					Rel("noopener"),
					Text(link.Title),
				),
				Time(Attr("datetime", link.CreatedAt.Format(time.RFC3339)), Text(relTime)),
			),
			Footer(
				Small(Text(link.URL.Host)),
				Ul(tags...),
				Span(Class("row-actions"),
					Form(
						Method("POST"),
						Action(readURL),
						Input(Type("hidden"), Name("next"), Value("/reading")),
						Button(Class("button outline small"), Type("submit"), Text("Mark as read")),
						Attr("hx-post", readURL),
						Attr("hx-target", "closest li"),
						Attr("hx-swap", "delete"),
					),
				),
			),
			noteBlock(link),
		),
	)
}

// EditPanelRow wraps an inline edit panel in the list item the row swap
// expects: swap targets are `closest li` + outerHTML, so fragments must be Li
// (a bare article would replace the li and break .link-list > li > article).
func EditPanelRow(panel Node) Node {
	return Li(panel)
}

// bookmarkMenu is the row's kebab menu. The trigger opens a native popover
// (popover="auto") with light-dismiss; both items swap the row for an inline
// edit panel (htmx) or navigate to a full edit page (no-JS). The
// "Edit collections" item renders only when the caller passed a non-empty
// collections list (callers pre-filter to manageable collections).
func bookmarkMenu(link core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	editURL := fmt.Sprintf("/bookmarks/%s/edit", link.ID)
	collectionsURL := fmt.Sprintf("/bookmarks/%s/collections/edit", link.ID)
	if currentCollectionID != "" {
		collectionsURL += "?collection=" + currentCollectionID
	}
	menuID := "bookmark-menu-" + link.ID.String()
	readingURL := fmt.Sprintf("/bookmarks/%s/reading", link.ID)
	readingLabel := "Add to reading list"
	if !link.QueuedAt.IsZero() {
		readingLabel = "Remove from reading list"
	}
	return Group{
		Button(
			Class("button ghost small"),
			Type("button"),
			Attr("popovertarget", menuID),
			Attr("aria-label", "Bookmark actions"),
			Text("⋯"),
		),
		Div(
			ID(menuID),
			Class("menu-card"),
			Attr("popover", "auto"),
			A(
				Class("menu-item"),
				Href(editURL),
				Attr("hx-get", editURL),
				Attr("hx-target", "closest li"),
				Attr("hx-swap", "outerHTML"),
				Text("Edit notes & tags"),
			),
			If(len(collections) > 0,
				A(
					Class("menu-item"),
					Href(collectionsURL),
					Attr("hx-get", collectionsURL),
					Attr("hx-target", "closest li"),
					Attr("hx-swap", "outerHTML"),
					Text("Edit collections"),
				),
			),
			Form(
				Method("POST"),
				Action(readingURL),
				Button(
					Class("menu-item"),
					Type("submit"),
					Text(readingLabel),
				),
			),
			Button(
				Class("menu-item danger"),
				Type("button"),
				Attr("hx-post", fmt.Sprintf("/bookmarks/%s/archive", link.ID)),
				Attr("hx-confirm", "Archive this bookmark? You can restore it later."),
				Attr("hx-target", "closest li"),
				Attr("hx-swap", "delete"),
				Text("Archive"),
			),
		),
	}
}

// CollectionEditPanel is the inline edit state for collection membership — the
// same row-replacement pattern as the notes panel: a search field plus a
// checkbox list of the user's collections, current membership pre-checked.
// One save replaces membership (add + remove in one go). currentCollectionID
// is hidden so the server can drop the row when this collection is unchecked.
func CollectionEditPanel(link core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	postURL := fmt.Sprintf("/bookmarks/%s/collections", link.ID)

	memberOf := make(map[uuid.UUID]bool, len(link.CollectionIDs))
	for _, cid := range link.CollectionIDs {
		memberOf[cid] = true
	}
	items := Map(collections, func(c core.Collection) Node {
		return Li(
			Label(
				Input(Type("checkbox"), Name("collections"), Value(c.ID.String()), If(memberOf[c.ID], Checked())),
				Text(c.Name),
				Span(Class("count"), Text(fmt.Sprintf("%d", c.BookmarkCount))),
			),
		)
	})
	tags := make([]Node, 0, len(link.Tags))
	for _, tag := range link.Tags {
		tags = append(tags, Li(Text(tag)))
	}

	return Article(
		Header(
			A(
				Href(link.URL.String()),
				Target("_blank"),
				Rel("noopener"),
				Text(link.Title),
			),
			Time(Attr("datetime", link.CreatedAt.Format(time.RFC3339)), Text(relativeTime(link.CreatedAt))),
		),
		Footer(
			Small(Text(link.URL.Host)),
			Ul(tags...), // collections editing changes membership, not tags — keep them visible
		),
		noteBlock(link),
		Form(
			Class("edit-panel"),
			Method("POST"),
			Action(postURL),
			Attr("hx-post", postURL),
			Attr("hx-target", "closest li"),
			Attr("hx-swap", "outerHTML"),
			If(currentCollectionID != "",
				Input(Type("hidden"), Name("current_collection"), Value(currentCollectionID)),
			),
			Div(Class("picker-title"), Text("Edit collections")),
			Input(Type("search"), Name("q"), Placeholder("Search collections"), Attr("enterkeyhint", "search")),
			Ul(Class("pick-list"), items),
			Div(Class("edit-actions"),
				Button(Type("submit"), Class("button small"), Text("Save")),
				A(
					Class("button ghost small"),
					Href("/"),
					Attr("hx-get", fmt.Sprintf("/bookmarks/%s", link.ID)),
					Attr("hx-target", "closest li"),
					Attr("hx-swap", "outerHTML"),
					Text("Cancel"),
				),
			),
		),
	)
}

// EditCollectionsPage is the no-JS fallback: a full page wrapping the panel.
func EditCollectionsPage(user core.User, panel Node) Node {
	return Page("Edit Collections — Patchworks", AppShell(user, Main(
		A(Class("back-link"), Href("/"), Text("← Dashboard")),
		panel,
	)))
}

// BookmarkEditPanel is the inline edit state replacing the row's article:
// notes textarea + comma-separated tags; title and domain stay read-only.
func BookmarkEditPanel(link core.Bookmark, errs FormErrors) Node {
	editURL := fmt.Sprintf("/bookmarks/%s/edit", link.ID)
	rowURL := fmt.Sprintf("/bookmarks/%s", link.ID)
	return Article(
		Header(
			A(
				Href(link.URL.String()),
				Target("_blank"),
				Rel("noopener"),
				Text(link.Title),
			),
			Time(Attr("datetime", link.CreatedAt.Format(time.RFC3339)), Text(relativeTime(link.CreatedAt))),
		),
		Footer(Small(Text(link.URL.Host))),
		Form(
			Class("edit-panel"),
			Method("POST"),
			Action(editURL),
			Attr("hx-post", editURL),
			Attr("hx-target", "closest li"),
			Attr("hx-swap", "outerHTML"),
			Label(Text("Note"),
				Textarea(Name("notes"), Rows("2"), Text(link.Notes)),
			),
			TextInput("Tags (comma-separated)", "tags", "text", strings.Join(link.Tags, ", "), errs),
			Div(Class("edit-actions"),
				Button(Type("submit"), Class("button small"), Text("Save")),
				A(
					Class("button ghost small"),
					Href("/"),
					Attr("hx-get", rowURL),
					Attr("hx-target", "closest li"),
					Attr("hx-swap", "outerHTML"),
					Text("Cancel"),
				),
			),
		),
	)
}

// noteBlock renders the bookmark's note under the domain & tags, clamped to one
// line with a native More/Less toggle when it likely overflows. The toggle is a
// <details>; CSS :has() unclamps the text on open — no JS, and the text lives
// in the DOM exactly once.
func noteBlock(link core.Bookmark) Node {
	if link.Notes == "" {
		return nil
	}
	return Div(
		Class("note"),
		P(Class("note-text"), Text(link.Notes)),
		If(shouldShowNoteToggle(link.Notes), Details(Class("note-toggle"), Summary())),
	)
}

// shouldShowNoteToggle is a conservative server-side heuristic for "the note
// overflows one line": CSS can't detect overflow, so we approximate by length.
// At note size (--font-size-0) a row column holds roughly 45-80 characters per
// line; a toggle that over-renders is merely redundant (clamped text hides
// nothing), and one that under-renders leaves the note truncated without a way
// to expand.
func shouldShowNoteToggle(notes string) bool {
	return len(notes) > 45
}

// ListPagerProps configures a ListPager: which list it pages, where the
// list lives, and the current query whose filters the pager links preserve.
type ListPagerProps struct {
	NavID  string     // pager nav element id, OOB-swapped on htmx requests
	ListID string     // id of the list UL the buttons swap into
	Base   string     // page path the pager links point at, e.g. "/"
	Query  url.Values // current query: filters are kept, cursors replaced
	Page   core.BookmarkPage
	OOB    bool // flag the nav for htmx out-of-band swap (pager fragments)
}

// ListPager renders the conditional pager for a paginated list (BK-15): a
// plain "Back to latest" link on no-JS mid-feed loads, plus a Load more button
// when older items exist. No page numbers, no offset. htmx requests append to
// the list in place and do NOT touch the URL — the position is ephemeral, so a
// refresh returns to the list head; only filters live in the URL. No-JS paging
// intentionally differs: each full load re-renders only the window (the batch
// older than the cursor), never the rows already scrolled past.
func ListPager(p ListPagerProps) Node {
	if len(p.Page.Items) == 0 || (!p.Page.HasOlder && !p.Page.HasNewer) {
		return nil
	}

	// The cursor pins the last visible row, the edge the batch extends past.
	older := p.Base + pagerURL(p.Query, "older", core.CursorOf(p.Page.Items[len(p.Page.Items)-1]))

	return Nav(
		ID(p.NavID),
		Class("pager mt-4"),
		Attr("aria-label", "List pagination"),
		If(p.OOB, Attr("hx-swap-oob", "true")),
		If(p.Page.HasNewer, backToLatest(p.Base+pagerURLWithoutCursors(p.Query))),
		If(p.Page.HasOlder, pagerButton("Load more", older, p.ListID, "beforeend")),
	)
}

// backToLatest is a plain link to the list head: a full load, no htmx.
// Styled like the Load more button — the pager reads as one control group.
func backToLatest(href string) Node {
	return A(Href(href), Class("button outline small"), Text("Back to latest"))
}

func pagerButton(label, href, listID, swap string) Node {
	return A(
		Href(href),
		Class("button outline small"),
		Attr("hx-get", href),
		Attr("hx-target", "#"+listID),
		Attr("hx-swap", swap),
		Text(label),
	)
}

// ListFragment renders the htmx response body for a pager click: bare list
// rows (the main swap target is the list UL itself) plus the pager nav
// flagged for out-of-band replacement with fresh cursors.
func ListFragment(page core.BookmarkPage, collections []core.Collection, currentCollectionID string, pager ListPagerProps) Node {
	nodes := make([]Node, 0, len(page.Items)+1)
	for _, link := range page.Items {
		nodes = append(nodes, LinkRow(link, collections, currentCollectionID))
	}
	pager.OOB = true
	nodes = append(nodes, ListPager(pager))
	return Group(nodes)
}

func relativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	return t.Format("Jan 2")
}
