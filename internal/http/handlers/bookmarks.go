package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/ajg/form"
	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/davenathanael/patchwork/internal/http/views"
	"github.com/google/uuid"
)

func handleGetHome(components *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getHome(w, r, components.DB, components.DB, components.DB)
	}
}

type (
	collectionGetter interface {
		GetCollectionsByUser(ctx context.Context, userID uuid.UUID) ([]core.Collection, error)
	}

	tagGetter interface {
		GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error)
	}

	bookmarkGetter interface {
		GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error)
		GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error)
	}
)

func getHome(w http.ResponseWriter, r *http.Request, collectionGetter collectionGetter, tagGetter tagGetter, bookmarkGetter bookmarkGetter) {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	collections, err := collectionGetter.GetCollectionsByUser(ctx, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	tags, err := tagGetter.GetTagsByUser(ctx, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	recentBookmarks, err := bookmarkGetter.GetRecentBookmarksByUser(ctx, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	allBookmarks, err := bookmarkGetter.GetAllBookmarksByUser(ctx, user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	vm := views.HomePageViewModel{
		User:            user,
		Collections:     collections,
		Tags:            tags,
		RecentBookmarks: recentBookmarks,
		AllBookmarks:    allBookmarks,
	}
	err = vm.Render(w)
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
	bookmarkSaver interface {
		CreateBookmark(ctx context.Context, url *url.URL, title string, userID, collectionID uuid.UUID, tags []string) (core.Bookmark, error)
	}

	newBookmarkForm struct {
		URL          string `form:"url"`
		CollectionID string `form:"collection_id"`
		Tags         string `form:"tags"`
	}

	titleFetcher interface {
		FetchPageTitle(ctx context.Context, u *url.URL) string
	}
)

func postBookmarks(w http.ResponseWriter, r *http.Request, saver bookmarkSaver, fetcher titleFetcher) {
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

	if _, err := saver.CreateBookmark(ctx, parsedURL, title, user.ID, collectionID, tags); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
