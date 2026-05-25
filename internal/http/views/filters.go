package views

import (
	"fmt"
	"net/url"
	"slices"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// CollectionFilter renders a filterable list of collections.
func CollectionFilter(items []CollectionItem, activeID string, currentQuery url.Values) Node {
	if len(items) == 0 {
		return Div()
	}

	listItems := make([]Node, 0, len(items))
	for _, item := range items {
		pageLink := A(
			Href(fmt.Sprintf("/%s", BuildQueryString(currentQuery, "collection_id", item.Name))),
			If(item.ID == activeID, Attr("aria-current", "page")),
			// Span(Text(item.Name)),
			Span(Text(fmt.Sprintf("%s (%d)", item.Name, item.Count))),
			// Span(Class("text-muted"), Text(fmt.Sprintf("(%d)", item.Count))),
		)
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
func TagFilter(items []TagItem, activeTags []string, currentQuery url.Values) Node {
	if len(items) == 0 {
		return Div()
	}

	listItems := make([]Node, 0, len(items))
	for _, item := range items {
		pageLink := A(
			Href(fmt.Sprintf("/%s", BuildQueryString(currentQuery, "tags", item.Name))),
			If(slices.Contains(activeTags, item.Name), Attr("aria-current", "page")),
			Span(Text(fmt.Sprintf("%s (%d)", item.Name, item.Count))),
			// Span(Text(item.Name)),
			// Span(Class("text-muted"), Text(fmt.Sprintf("(%d)", item.Count))),
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

func BuildQueryString(qs url.Values, key, value string) string {
	queries := make(url.Values, len(qs))
	for k, v := range qs {
		queries[k] = slices.Clone(v)
	}

	if queries.Has(key) && slices.Contains(queries[key], value) {
		queries[key] = slices.DeleteFunc(queries[key], func(s string) bool {
			return s == value
		})
	} else if queries.Has(key) && !slices.Contains(queries[key], value) {
		queries.Add(key, value)
	} else {
		queries.Set(key, value)
	}

	if len(queries) == 0 {
		return ""
	}
	return "?" + queries.Encode()
}
