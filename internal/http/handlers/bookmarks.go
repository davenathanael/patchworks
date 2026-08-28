package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ajg/form"
	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/davenathanael/patchwork/internal/http/views"
	"github.com/google/uuid"
)

// BookmarkStore is the interface for bookmark and tag persistence.
type BookmarkStore interface {
	GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error)
	GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID, search string) ([]core.Bookmark, error)
	GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID, search string) ([]core.Bookmark, error)
	GetBookmarksByCollectionAndTags(ctx context.Context, collectionID uuid.UUID, tags []string, search string) ([]core.Bookmark, error)
	GetBookmarksByCollection(ctx context.Context, collectionID uuid.UUID, search string) ([]core.Bookmark, error)
	GetBookmarksByTags(ctx context.Context, userID uuid.UUID, tags []string, search string) ([]core.Bookmark, error)
	CreateBookmark(ctx context.Context, url *url.URL, title string, userID, collectionID uuid.UUID, tags []string) (core.Bookmark, error)
}

func handleGetHome(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getHome(w, r, comp.DB, comp.DB)
	}
}

func getHome(w http.ResponseWriter, r *http.Request, collections CollectionStore, bookmarks BookmarkStore) {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	qs := r.URL.Query()
	filterTags := qs["tags"]
	filterCollectionID, err := uuid.Parse(qs.Get("collection_id"))
	if err != nil {
		filterCollectionID = uuid.Nil
	}
	filterPage, err := strconv.Atoi(qs.Get("page"))
	if err != nil {
		filterPage = 0
	}
	filterSearch := qs.Get("search")
	collectionID := ""
	if filterCollectionID != uuid.Nil {
		collectionID = filterCollectionID.String()
	}

	collectionsList, err := collections.GetCollectionsByUser(ctx, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	tags, err := bookmarks.GetTagsByUser(ctx, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	vm := views.HomePageViewModel{
		User:         user,
		Collections:  collectionsList,
		Tags:         tags,
		CollectionID: collectionID,
		TagsFilter:   filterTags,
		Page:         filterPage,
		Search:       filterSearch,
		CurrentQuery: qs,
	}
	if filterCollectionID != uuid.Nil && len(filterTags) != 0 {
		result, err := bookmarks.GetBookmarksByCollectionAndTags(ctx, filterCollectionID, filterTags, filterSearch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		vm.AllBookmarks = result
	} else if filterCollectionID != uuid.Nil {
		result, err := bookmarks.GetBookmarksByCollection(ctx, filterCollectionID, filterSearch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		vm.AllBookmarks = result
	} else if len(filterTags) != 0 {
		result, err := bookmarks.GetBookmarksByTags(ctx, user.ID, filterTags, filterSearch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		vm.AllBookmarks = result
	} else {
		recentBookmarks, err := bookmarks.GetRecentBookmarksByUser(ctx, user.ID, filterSearch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		vm.RecentBookmarks = recentBookmarks

		allBookmarks, err := bookmarks.GetAllBookmarksByUser(ctx, user.ID, filterSearch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		vm.AllBookmarks = allBookmarks
	}

	if views.IsHtmx(r) {
		err = vm.RenderFiltered(w)
	} else {
		err = vm.Render(w)
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func handlePostBookmarks(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		postBookmarks(w, r, comp.DB, comp.HTTPClient)
	}
}

type (
	newBookmarkForm struct {
		URL          string `form:"url"`
		CollectionID string `form:"collection_id"`
		Tags         string `form:"tags"`
	}

	titleFetcher interface {
		FetchPageTitle(ctx context.Context, u *url.URL) string
	}
)

func postBookmarks(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore, fetcher titleFetcher) {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	var formData newBookmarkForm
	if err := form.NewDecoder(r.Body).Decode(&formData); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	parsedURL, err := url.Parse(formData.URL)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	collectionID := uuid.Nil
	if formData.CollectionID != "" {
		collectionID, err = uuid.Parse(formData.CollectionID)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	}

	tags := strings.Split(formData.Tags, ",")
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}

	title := fetcher.FetchPageTitle(ctx, parsedURL)

	if _, err := bookmarks.CreateBookmark(ctx, parsedURL, title, user.ID, collectionID, tags); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if views.IsHtmx(r) {
		recent, err := bookmarks.GetRecentBookmarksByUser(ctx, user.ID, "")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		all, err := bookmarks.GetAllBookmarksByUser(ctx, user.ID, "")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		vm := views.HomePageViewModel{
			User:            user,
			RecentBookmarks: recent,
			AllBookmarks:    all,
		}
		if err := vm.RenderBookmarks(w); err != nil {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
