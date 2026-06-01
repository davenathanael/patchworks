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
	CurrentQuery    url.Values
}

func (vm *HomePageViewModel) Render(w io.Writer) error {
	stubPagination := PaginationProps{
		CurrentPage: 1,
		TotalPages:  3,
		BaseURL:     "/",
	}
	collectionItems := ToCollectionItems(vm.Collections)
	tagItems := ToTagItems(vm.Tags)
	hasFilters := vm.CollectionID != "" || len(vm.TagsFilter) > 0

	mainContent := Main(
		Class("container"),
		NewBookmark(collectionItems),
		FilterBar(collectionItems, vm.CollectionID, tagItems, vm.TagsFilter, vm.CurrentQuery),
		If(hasFilters,
			FilteredLinksView(ToLinkItems(vm.AllBookmarks), stubPagination),
		),
		If(!hasFilters,
			Group{
				If(len(vm.RecentBookmarks) > 0, RecentLinks(ToLinkItems(vm.RecentBookmarks))),
				BookmarksList(ToLinkItems(vm.AllBookmarks), stubPagination),
			},
		),
	)

	return Page("Dashboard — Patchworks", AppShell(vm.User, mainContent)).Render(w)
}
