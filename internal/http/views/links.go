package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// PaginationProps holds pagination metadata.
type PaginationProps struct {
	CurrentPage int
	TotalPages  int
	BaseURL     string
}

// BookmarkForm is the shared add-bookmark form view-model: the ajg/form
// decode target and the render model. Zero value renders a fresh form.
type BookmarkForm struct {
	URL          string     `form:"url"`
	CollectionID string     `form:"collection_id"`
	Tags         string     `form:"tags"`
	Errors       FormErrors `form:"-"`
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
func NewBookmarkForm(form BookmarkForm, collections []core.Collection) Node {
	return Form(
		ID("add-bookmark-form"),
		Method("POST"),
		Action("/bookmarks"),
		Attr("hx-post", "/bookmarks"),
		Attr("hx-target", "#bookmarks"),
		Attr("hx-swap", "innerHTML"),
		Attr("hx-on::after-request", "if(event.detail.successful) this.reset()"),
		TextInput("URL", "url", "url", form.URL, form.Errors,
			ID("add-link"), Placeholder("https://example.com"), Required(),
			Attr("inputmode", "url"), Attr("enterkeyhint", "go"),
		),
		TextInput("Tags", "tags", "text", form.Tags, form.Errors, Placeholder("go, css, reading")),
		Button(Type("submit"), Text("Save")),
	)
}

func RecentLinks(links []core.Bookmark, collections []core.Collection) Node {
	return Section(
		H2(Text("Recent")),
		IfElse(
			len(links) > 0,
			Links(links, collections, ""),
			P(Class("muted"), Text("No links yet. Add one above to get started.")),
		),
	)
}

func FilteredLinksView(links []core.Bookmark, collections []core.Collection, p PaginationProps) Node {
	return Section(
		H2(Text("Filtered Links")),
		IfElse(
			len(links) > 0,
			Group{Links(links, collections, ""), Pagination(p)},
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

func Links(links []core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	return Ul(Class("link-list"), Map(links, func(link core.Bookmark) Node {
		return LinkRow(link, collections, currentCollectionID)
	}))
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

// EditPanelRow wraps an inline edit panel in the list item the row swap
// expects: swap targets are `closest li` + outerHTML, so fragments must be Li
// (a bare article would replace the li and break .link-list > li > article).
func EditPanelRow(panel Node) Node {
	return Li(panel)
}

// bookmarkMenu is the row's kebab menu. The trigger opens a native popover
// (popover="auto") with light-dismiss; both items swap the row for an inline
// edit panel (htmx) or navigate to a full edit page (no-JS).
func bookmarkMenu(link core.Bookmark, collections []core.Collection, currentCollectionID string) Node {
	editURL := fmt.Sprintf("/bookmarks/%s/edit", link.ID)
	collectionsURL := fmt.Sprintf("/bookmarks/%s/collections/edit", link.ID)
	if currentCollectionID != "" {
		collectionsURL += "?collection=" + currentCollectionID
	}
	menuID := "bookmark-menu-" + link.ID.String()
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
			A(
				Class("menu-item"),
				Href(collectionsURL),
				Attr("hx-get", collectionsURL),
				Attr("hx-target", "closest li"),
				Attr("hx-swap", "outerHTML"),
				Text("Edit collections"),
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

func Pagination(p PaginationProps) Node {
	if p.TotalPages <= 1 {
		return Div()
	}

	pages := make([]Node, 0, p.TotalPages)

	for i := 1; i <= p.TotalPages; i++ {
		pageNum := i
		pageURL := fmt.Sprintf("%s?page=%d", p.BaseURL, pageNum)

		pageLink := A(
			Href(pageURL),
			Class("button outline small"),
			If(pageNum == p.CurrentPage,
				Attr("aria-current", "page"),
			),
			Text(fmt.Sprintf("%d", pageNum)),
		)

		pages = append(pages, Li(pageLink))
	}

	return Nav(
		Class("mt-4"),
		Attr("aria-label", "Pagination"),
		Menu(Group(pages)),
	)
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
