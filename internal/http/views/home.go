package views

import (
	"io"
	"net/url"

	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type HomePageViewModel struct {
	User            core.User
	Collections     []core.Collection
	Tags            []core.Tag
	RecentBookmarks []core.Bookmark
	AllBookmarks    []core.Bookmark
	CollectionID    string
	TagsFilter      []string
	Page            int
	Search          string
	CurrentQuery    url.Values
}

func (vm *HomePageViewModel) bookmarks() Node {
	stubPagination := PaginationProps{
		CurrentPage: 1,
		TotalPages:  3,
		BaseURL:     "/",
	}
	hasFilters := vm.CollectionID != "" || len(vm.TagsFilter) > 0 || vm.Search != ""
	if hasFilters {
		return FilteredLinksView(vm.AllBookmarks, stubPagination)
	}
	return Group{
		If(len(vm.RecentBookmarks) > 0, RecentLinks(vm.RecentBookmarks)),
		BookmarksList(vm.AllBookmarks, stubPagination),
	}
}

func (vm *HomePageViewModel) Render(w io.Writer) error {
	mainContent := Main(
		NewBookmark(vm.Collections),
		FilterBar(vm.Collections, vm.CollectionID, vm.Tags, vm.TagsFilter, vm.Search, vm.CurrentQuery),
		Div(ID("bookmarks"), vm.bookmarks()),
	)

	return Page("Dashboard — Patchworks", AppShell(vm.User, mainContent)).Render(w)
}

// RenderBookmarks renders only the bookmarks fragment, for htmx partial updates.
func (vm *HomePageViewModel) RenderBookmarks(w io.Writer) error {
	return vm.bookmarks().Render(w)
}
