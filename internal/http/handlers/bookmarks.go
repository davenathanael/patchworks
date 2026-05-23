package handlers

import (
	"context"
	"net/http"

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
