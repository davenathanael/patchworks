package views

import (
	"fmt"
	"time"

	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// PaginationProps holds pagination metadata.
type PaginationProps struct {
	CurrentPage int
	TotalPages  int
	BaseURL     string
}

func NewBookmark(collections []core.Collection) Node {
	urlInput := Input(ID("add-link"), Type("url"), Name("url"), Placeholder("https://example.com"), Required(), Attr("inputmode", "url"), Attr("enterkeyhint", "go"))
	submitBtn := Button(Type("submit"), Text("Add"))

	return Details(
		Class("add-bookmark"),
		Summary(Text("+ Add Bookmark")),
		Div(
			Class("add-form"),
			Form(
				Method("POST"),
				Action("/bookmarks"),
				Attr("hx-post", "/bookmarks"),
				Attr("hx-target", "#bookmarks"),
				Attr("hx-swap", "innerHTML"),
				Attr("hx-on::after-request", "if(event.detail.successful) this.reset()"),
				FieldSet(
					Class("input-group"),
					urlInput,
					submitBtn,
				),
				Div(
					Class("add-form-row"),
					Select(
						Name("collection_id"),
						Option(Value(""), Text("Collection"), Disabled(), Selected()),
						Map(collections, func(i core.Collection) Node {
							return Option(Value(i.ID.String()), Text(i.Name))
						}),
					),
					Input(
						Type("text"),
						Name("tags"),
						Placeholder("Tags"),
					),
				),
			),
		),
	)
}

func RecentLinks(links []core.Bookmark) Node {
	return Section(
		H5(Class("section-heading"), Text("Recent")),
		IfElse(
			len(links) > 0,
			Links(links),
			P(Class("muted"), Text("No links yet. Add one above to get started.")),
		),
	)
}

func BookmarksList(links []core.Bookmark, p PaginationProps) Node {
	return Section(
		H5(Class("section-heading"), Text("Your Bookmarks")),
		IfElse(
			len(links) > 0,
			Group{Links(links), Pagination(p)},
			P(Class("muted"), Text("No bookmarks to display.")),
		),
	)
}

func FilteredLinksView(links []core.Bookmark, p PaginationProps) Node {
	return Section(
		H5(Class("section-heading"), Text("Filtered Links")),
		IfElse(
			len(links) > 0,
			Group{Links(links), Pagination(p)},
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

func Links(links []core.Bookmark) Node {
	return Ul(Class("link-list"), Map(links, LinkRow))
}

func LinkRow(link core.Bookmark) Node {
	relTime := relativeTime(link.CreatedAt)

	tags := make([]Node, 0, len(link.Tags))
	for _, tag := range link.Tags {
		tags = append(tags, Li(Text(tag)))
	}

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
			),
		),
	)
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
