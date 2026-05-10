package views

import (
	"fmt"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// CollectionFilter renders a filterable list of collections.
func CollectionFilter(items []CollectionItem, activeID string) Node {
	if len(items) == 0 {
		return Div()
	}

	listItems := make([]Node, 0, len(items))
	for _, item := range items {
		pageLink := A(
			Href(fmt.Sprintf("/?collection=%s", item.ID)),
			Span(Text(item.Name)),
			Span(Class("text-muted"), Text(fmt.Sprintf("(%d)", item.Count))),
		)
		if item.ID == activeID {
			pageLink = A(
				Href(fmt.Sprintf("/?collection=%s", item.ID)),
				Attr("aria-current", "page"),
				Span(Text(item.Name)),
				Span(Class("text-muted"), Text(fmt.Sprintf("(%d)", item.Count))),
			)
		}
		listItems = append(listItems, Li(pageLink))
	}

	ulItems := make([]Node, 0, len(listItems)+1)
	ulItems = append(ulItems, Class("filter-list"))
	ulItems = append(ulItems, listItems...)

	return Div(
		Class("mb-6"),
		H3(Class("section-heading"), Text("Collections")),
		Ul(ulItems...),
	)
}

// TagFilter renders a filterable list of tags.
func TagFilter(items []TagItem, activeTag string) Node {
	if len(items) == 0 {
		return Div()
	}

	listItems := make([]Node, 0, len(items))
	for _, item := range items {
		pageLink := A(
			Href(fmt.Sprintf("/?tag=%s", item.Name)),
			If(item.Name == activeTag, Attr("aria-current", "page")),
			Span(Text(item.Name)),
			Span(Class("text-muted"), Text(fmt.Sprintf("(%d)", item.Count))),
		)
		listItems = append(listItems, Li(pageLink))
	}

	ulItems := make([]Node, 0, len(listItems)+1)
	ulItems = append(ulItems, Class("filter-list"))
	ulItems = append(ulItems, listItems...)

	return Div(
		H3(Class("section-heading"), Text("Tags")),
		Ul(ulItems...),
	)
}
