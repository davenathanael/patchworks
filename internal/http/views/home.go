package views

import (
	"io"
	"net/url"

	"github.com/davenathanael/patchworks/internal/core"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type HomePageViewModel struct {
	User         core.User
	Collections  []core.Collection
	Tags         []core.Tag
	Recent       core.BookmarkPage // no filters: latest 10 with Load more
	Filtered     core.BookmarkPage // any filter active
	AddBookmark  BookmarkForm
	CollectionID string
	TagsFilter   []string
	Search       string
	CurrentQuery url.Values
}

func (vm *HomePageViewModel) bookmarks() Node {
	if vm.hasFilters() {
		return FilteredLinksView(vm.Filtered, vm.Collections, vm.filteredPager())
	}
	return Group{
		If(len(vm.Recent.Items) > 0, RecentLinks(vm.Recent, vm.Collections, vm.recentPager())),
	}
}

// hasFilters reports whether a filtered list (not the recent list) is active.
func (vm *HomePageViewModel) hasFilters() bool {
	return vm.CollectionID != "" || len(vm.TagsFilter) > 0 || vm.Search != ""
}

// recentPager pages the dashboard's recent list: the nav rides inside the
// bookmarks area, buttons swap into the recent UL.
func (vm *HomePageViewModel) recentPager() ListPagerProps {
	return ListPagerProps{
		NavID:  "recent-pager",
		ListID: "recent-list",
		Base:   "/",
		Query:  vm.CurrentQuery,
		Page:   vm.Recent,
	}
}

// filteredPager pages the dashboard's filtered list.
func (vm *HomePageViewModel) filteredPager() ListPagerProps {
	return ListPagerProps{
		NavID:  "bookmarks-pager",
		ListID: "filtered-list",
		Base:   "/",
		Query:  vm.CurrentQuery,
		Page:   vm.Filtered,
	}
}

func (vm *HomePageViewModel) Render(w io.Writer) error {
	mainContent := Main(
		NewBookmark(vm.AddBookmark, vm.Collections),
		FilterBar(vm.Collections, vm.CollectionID, vm.Tags, vm.TagsFilter, vm.Search, vm.CurrentQuery),
		Div(ID("bookmarks"), vm.bookmarks()),
	)

	return Page("Dashboard — Patchworks", AppShell(vm.User, mainContent)).Render(w)
}

// RenderBookmarks renders only the bookmarks fragment, for htmx partial updates.
func (vm *HomePageViewModel) RenderBookmarks(w io.Writer) error {
	return vm.bookmarks().Render(w)
}

// RenderFiltered renders the filters (out-of-band) and bookmarks fragments,
// so the pills reflect the current filter state after an htmx request.
func (vm *HomePageViewModel) RenderFiltered(w io.Writer) error {
	filters := Div(
		ID("filters"),
		Attr("hx-swap-oob", "true"),
		filterPills(vm.Collections, vm.CollectionID, vm.Tags, vm.TagsFilter, vm.Search, vm.CurrentQuery),
	)
	if err := filters.Render(w); err != nil {
		return err
	}
	return vm.bookmarks().Render(w)
}

// RenderListItems renders the htmx response for a pager click: bare list
// rows for the active list (the swap target is the list UL) plus the
// out-of-band pager nav with fresh cursors.
func (vm *HomePageViewModel) RenderListItems(w io.Writer) error {
	if vm.hasFilters() {
		return ListFragment(vm.Filtered, vm.Collections, "", vm.filteredPager()).Render(w)
	}
	return ListFragment(vm.Recent, vm.Collections, "", vm.recentPager()).Render(w)
}
