package views

import (
	"net/url"
	"time"

	"github.com/davenathanael/patchwork/internal/core"
	. "maragu.dev/gomponents"
	_ "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// HomePage is the dashboard home page after login.
func HomePage(user core.User) Node {
	// Stub data for initial render
	stubTime := time.Now().Add(-2 * time.Hour)
	url1, _ := url.Parse("https://example.com/go-generics")
	url2, _ := url.Parse("https://example.com/design-systems")
	stubLinks := []LinkItem{
		{
			ID:        "1",
			Title:     "Go generics deep dive",
			URL:       url1,
			Tags:      []string{"go", "tutorial"},
			CreatedAt: stubTime,
		},
		{
			ID:        "2",
			Title:     "Design systems at scale",
			URL:       url2,
			Tags:      []string{"design"},
			CreatedAt: stubTime,
		},
	}

	stubCollections := []CollectionItem{
		{ID: "all", Name: "All", Count: 142},
		{ID: "reading", Name: "Reading", Count: 23},
		{ID: "work", Name: "Work", Count: 18},
	}

	stubTags := []TagItem{
		{Name: "go", Count: 12},
		{Name: "design", Count: 7},
		{Name: "database", Count: 5},
	}

	stubPagination := PaginationProps{
		CurrentPage: 1,
		TotalPages:  3,
		BaseURL:     "/",
	}

	mainContent := Main(
		Class("container"),
		AddLinkBox(),
		RecentLinks(stubLinks),
		AllLinksView(stubLinks, stubPagination),
	)

	sidebar := SidebarNav(stubCollections, stubTags)

	return Page("Dashboard — Patchworks", AppShell(user, sidebar, mainContent))
}
