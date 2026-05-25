package views

import (
	"fmt"
	"time"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// AddLinkBox renders the quick add-link form.
func AddLinkBox() Node {
	urlInput := Input(ID("add-link"), Type("url"), Name("url"), Placeholder("https://example.com"), Required())
	submitBtn := Button(Type("submit"), Text("Add"))

	return Section(
		Class("card mb-4"),
		H2(Text("Add a link")),
		Form(
			Method("POST"),
			Action("/bookmarks"),
			FieldSet(Class("group"), urlInput, submitBtn),
			Details(
				Class("mt-2"),
				Summary(Text("Tags & Collection")),
				Div(
					Class("grid gap-2 mt-2"),
					Label(
						Span(Text("Collection")),
						Select(
							Name("collection_id"),
							Option(Value(""), Text("— Select —")),
							Option(Value("reading"), Text("Reading")),
							Option(Value("work"), Text("Work")),
						),
					),
					Label(
						Span(Text("Tags")),
						Input(
							Type("text"),
							Name("tags"),
							Placeholder("comma, separated"),
						),
					),
				),
			),
		),
	)
}

// RecentLinks renders a section of the most recently added links.
func RecentLinks(links []LinkItem) Node {
	return Section(
		Class("dashboard-section"),
		H2(Text("Recent")),
		IfElse(
			len(links) > 0,
			Links(links),
			P(Text("No links yet. Add one above to get started.")),
		),
	)
}

// BookmarksList renders the paginated all-links section.
func BookmarksList(links []LinkItem, p PaginationProps) Node {
	return Section(
		Class("dashboard-section"),
		H2(Text("Your Bookmarks")),
		IfElse(
			len(links) > 0,
			Group{Links(links), Pagination(p)},
			P(Text("No bookmarks to display.")),
		),
	)
}

// FilteredLinksView renders the paginated filtered-links section.
func FilteredLinksView(links []LinkItem, p PaginationProps) Node {
	return Section(
		Class("dashboard-section"),
		H2(Text("Filtered Links")),
		IfElse(
			len(links) > 0,
			Group{Links(links), Pagination(p)},
			P(Text("No links to display.")),
		),
	)
}

func IfElse(condition bool, trueNode, falseNode Node) Node {
	if condition {
		return trueNode
	}
	return falseNode
}

func Links(links []LinkItem) Node {
	return Ul(Class("unstyled"), Map(links, LinkRow))
}

// LinkRow renders a single link row (used in RecentLinks and AllLinksView).
func LinkRow(link LinkItem) Node {
	relTime := relativeTime(link.CreatedAt)

	tags := make([]Node, 0, len(link.Tags)+1)
	tags = append(tags, Class("flex gap-1 flex-wrap"))
	for _, tag := range link.Tags {
		tags = append(tags, Span(Class("tag-badge"), Text(tag)))
	}

	tagGroup := Div(tags...)

	return Li(Class("unstyled"),
		Div(
			Class("link-row"),
			Div(
				A(
					Href(link.URL.String()),
					Class("link-title"),
					Target("_blank"),
					Rel("noopener"),
					Text(link.Title),
				),
				Div(
					Class("link-meta"),
					Span(Class("text-muted"), Text(link.Domain())),
					tagGroup,
				),
			),
			Span(Class("text-muted"), Text(relTime)),
		),
	)
}

// Pagination renders numbered pagination controls.
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
		Menu(Class("buttons"), Group(pages)),
	)
}

// relativeTime formats a time as a relative string (e.g., "2h ago").
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
