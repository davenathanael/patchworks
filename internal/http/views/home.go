package views

import (
	"io"
	"net/url"

	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	_ "maragu.dev/gomponents/components"
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
	showRecentLinks := len(vm.RecentBookmarks) > 0
	mainContent := Main(
		Class("container"),
		NewBookmark(collectionItems),
		If(showRecentLinks, RecentLinks(ToLinkItems(vm.RecentBookmarks))),
		BookmarksList(ToLinkItems(vm.AllBookmarks), stubPagination),
	)

	sidebar := SidebarNav{
		Collections:        collectionItems,
		ActiveCollectionID: vm.CollectionID,
		Tags:               ToTagItems(vm.Tags),
		ActiveTags:         vm.TagsFilter,
		CurrentQuery:       vm.CurrentQuery,
	}

	return Page("Dashboard — Patchworks", AppShell(vm.User, sidebar, mainContent)).Render(w)
}
