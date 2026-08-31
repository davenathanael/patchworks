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
	return Form(
		Method("GET"),
		Action("/"),
		Div(Class("filter-bar"),
			Label(Text("Search"),
				Input(
					Type("search"),
					Name("search"),
					Placeholder("Search bookmarks…"),
					Value(search),
					Attr("enterkeyhint", "search"),
					Attr("hx-get", searchURL(currentQuery)),
					Attr("hx-trigger", "input changed delay:500ms"),
					Attr("hx-target", "#bookmarks"),
				),
			),
			Div(ID("filters"), filterPills(collections, activeCollectionID, tags, activeTags, search, currentQuery)),
		),
	)
}

// filterPills renders the collection/tag pills and the clear-filters link.
// It is also rendered out-of-band so the pills reflect the current filter state.
func filterPills(
	collections []core.Collection,
	activeCollectionID string,
	tags []core.Tag,
	activeTags []string,
	search string,
	currentQuery url.Values,
) Node {
	hasFilters := activeCollectionID != "" || len(activeTags) > 0 || search != ""

	return Group{
		Nav(CollectionPills(collections, activeCollectionID, currentQuery)),
		Nav(TagPills(tags, activeTags, currentQuery)),
		If(hasFilters,
			A(
				Href("/"),
				Class("clear-link"),
				Attr("hx-get", "/"),
				Attr("hx-target", "#bookmarks"),
				Attr("hx-push-url", "true"),
				Text("Clear filters"),
			),
		),
	}
}

func CollectionPills(items []core.Collection, activeID string, currentQuery url.Values) Node {
	if len(items) == 0 {
		return Group{}
	}

	pills := make([]Node, 0, len(items))
	for _, item := range items {
		href := fmt.Sprintf("/%s", BuildQueryString(currentQuery, "collection_id", item.ID.String()))
		pills = append(pills,
			A(
				Class("filter-pill"),
				Href(href),
				Attr("hx-get", href),
				Attr("hx-target", "#bookmarks"),
				Attr("hx-push-url", "true"),
				If(item.ID.String() == activeID, Attr("aria-current", "page")),
				Text(fmt.Sprintf("%s (%d)", item.Name, item.BookmarkCount)),
			),
		)
	}

	return Group(pills)
}

func TagPills(items []core.Tag, activeTags []string, currentQuery url.Values) Node {
	if len(items) == 0 {
		return Group{}
	}

	topN := TopTagCount
	if len(items) <= topN {
		return TagPillList(items, activeTags, currentQuery)
	}

	return Group{
		TagPillList(items[:topN], activeTags, currentQuery),
		Details(
			Class("filter-details"),
			Summary(Text("See more")),
			TagPillList(items[topN:], activeTags, currentQuery),
		),
	}
}

func TagPillList(items []core.Tag, activeTags []string, currentQuery url.Values) Node {
	pills := make([]Node, 0, len(items))
	for _, item := range items {
		href := fmt.Sprintf("/%s", BuildQueryString(currentQuery, "tags", item.Name))
		pills = append(pills,
			A(
				Class("tag-pill"),
				Href(href),
				Attr("hx-get", href),
				Attr("hx-target", "#bookmarks"),
				Attr("hx-push-url", "true"),
				If(slices.Contains(activeTags, item.Name), Attr("aria-current", "page")),
				Text(fmt.Sprintf("#%s (%d)", item.Name, item.BookmarkCount)),
			),
		)
	}

	return Group(pills)
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

// searchURL returns the current filter query without the search param, so the
// search input's own value is appended fresh by htmx.
func searchURL(currentQuery url.Values) string {
	q := make(url.Values, len(currentQuery))
	for k, v := range currentQuery {
		q[k] = slices.Clone(v)
	}
	q.Del("search")
	if len(q) == 0 {
		return "/"
	}
	return "/?" + q.Encode()
}
