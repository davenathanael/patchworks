package views

import (
	"fmt"
	"net/url"
	"slices"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func FilterBar(
	collections []CollectionFilterItem,
	activeCollectionID string,
	tags []TagItem,
	activeTags []string,
	currentQuery url.Values,
) Node {
	hasFilters := activeCollectionID != "" || len(activeTags) > 0

	return Div(
		Class("filter-bar"),
		Div(Class("search-field"),
			Label(
				Attr("data-field"),
				Input(
					Type("search"),
					Name("q"),
					Placeholder("Search bookmarks..."),
					Value(currentQuery.Get("q")),
					Attr("enterkeyhint", "search"),
				),
			),
		),
		Div(Class("pill-group"),
			CollectionPills(collections, activeCollectionID, currentQuery),
		),
		Div(Class("pill-group"),
			TagPills(tags, activeTags, currentQuery),
		),
		If(hasFilters,
			Div(
				A(
					Href("/"),
					Class("clear-link"),
					Text("Clear filters"),
				),
			),
		),
	)
}

func CollectionPills(items []CollectionFilterItem, activeID string, currentQuery url.Values) Node {
	if len(items) == 0 {
		return Div()
	}

	pills := make([]Node, 0, len(items))
	for _, item := range items {
		href := fmt.Sprintf("/%s", BuildQueryString(currentQuery, "collection_id", item.ID))
		pills = append(pills,
			A(
				Class("filter-pill"),
				Href(href),
				If(item.ID == activeID, Attr("aria-current", "page")),
				Text(fmt.Sprintf("%s (%d)", item.Name, item.Count)),
			),
		)
	}

	return Div(pills...)
}

func TagPills(items []TagItem, activeTags []string, currentQuery url.Values) Node {
	if len(items) == 0 {
		return Div()
	}

	topN := TopTagCount
	if len(items) <= topN {
		return TagPillList(items, activeTags, currentQuery)
	}

	return Div(
		TagPillList(items[:topN], activeTags, currentQuery),
		Details(
			Class("filter-details"),
			Summary(Text("See more")),
			TagPillList(items[topN:], activeTags, currentQuery),
		),
	)
}

func TagPillList(items []TagItem, activeTags []string, currentQuery url.Values) Node {
	pills := make([]Node, 0, len(items))
	for _, item := range items {
		href := fmt.Sprintf("/%s", BuildQueryString(currentQuery, "tags", item.Name))
		pills = append(pills,
			A(
				Class("tag-pill"),
				Href(href),
				If(slices.Contains(activeTags, item.Name), Attr("aria-current", "page")),
				Text(fmt.Sprintf("#%s (%d)", item.Name, item.Count)),
			),
		)
	}

	return Div(pills...)
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
