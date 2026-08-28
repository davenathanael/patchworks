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

func (vm *HomePageViewModel) Render(w io.Writer) error {
	stubPagination := PaginationProps{
		CurrentPage: 1,
		TotalPages:  3,
		BaseURL:     "/",
	}
	hasFilters := vm.CollectionID != "" || len(vm.TagsFilter) > 0 || vm.Search != ""

	mainContent := Main(
		NewBookmark(vm.Collections),
		FilterBar(vm.Collections, vm.CollectionID, vm.Tags, vm.TagsFilter, vm.Search, vm.CurrentQuery),
		If(hasFilters,
			FilteredLinksView(vm.AllBookmarks, stubPagination),
		),
		If(!hasFilters,
			Group{
				If(len(vm.RecentBookmarks) > 0, RecentLinks(vm.RecentBookmarks)),
				BookmarksList(vm.AllBookmarks, stubPagination),
			},
		),
	)

	return Page("Dashboard — Patchworks", AppShell(vm.User, mainContent)).Render(w)
}
