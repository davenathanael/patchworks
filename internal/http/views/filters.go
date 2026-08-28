package views

import (
	"fmt"
	"net/url"
	"slices"

	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

const TopTagCount = 15

func FilterBar(
	collections []core.Collection,
	activeCollectionID string,
	tags []core.Tag,
	activeTags []string,
	search string,
	currentQuery url.Values,
) Node {
	hasFilters := activeCollectionID != "" || len(activeTags) > 0 || search != ""

	return Form(
		Method("GET"),
		Action("/"),
		Div(Class("filter-bar"),
			Div(Class("search-field"),
				Input(
					Type("search"),
					Name("search"),
					Placeholder("Search bookmarks..."),
					Value(search),
					Attr("enterkeyhint", "search"),
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
		),
	)
}

func CollectionPills(items []core.Collection, activeID string, currentQuery url.Values) Node {
	if len(items) == 0 {
		return Div()
	}

	pills := make([]Node, 0, len(items))
	for _, item := range items {
		href := fmt.Sprintf("/%s", BuildQueryString(currentQuery, "collection_id", item.ID.String()))
		pills = append(pills,
			A(
				Class("filter-pill"),
				Href(href),
				If(item.ID.String() == activeID, Attr("aria-current", "page")),
				Text(fmt.Sprintf("%s (%d)", item.Name, item.BookmarkCount)),
			),
		)
	}

	return Div(pills...)
}

func TagPills(items []core.Tag, activeTags []string, currentQuery url.Values) Node {
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

func TagPillList(items []core.Tag, activeTags []string, currentQuery url.Values) Node {
	pills := make([]Node, 0, len(items))
	for _, item := range items {
		href := fmt.Sprintf("/%s", BuildQueryString(currentQuery, "tags", item.Name))
		pills = append(pills,
			A(
				Class("tag-pill"),
				Href(href),
				If(slices.Contains(activeTags, item.Name), Attr("aria-current", "page")),
				Text(fmt.Sprintf("#%s (%d)", item.Name, item.BookmarkCount)),
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
