package handlers

import (
	"context"
	"fmt"
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

func handleGetHome(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getHome(w, r, comp.DB, comp.DB)
	}
}

func getHome(w http.ResponseWriter, r *http.Request, collections CollectionStore, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	vm, err := loadHomeVM(r, user, collections, bookmarks)
	if err != nil {
		return err
	}

	if views.IsHtmx(r) {
		err = vm.RenderFiltered(w)
	} else {
		err = vm.Render(w)
	}
	if err != nil {
		return fmt.Errorf("render home: %w", err)
	}
	return nil
}

// loadHomeVM builds the home view-model from the request's filters and the
// user's collections, tags and bookmarks.
func loadHomeVM(r *http.Request, user core.User, collections CollectionStore, bookmarks BookmarkStore) (*views.HomePageViewModel, error) {
	ctx := r.Context()

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
		return nil, fmt.Errorf("get collections: %w", err)
	}

	tags, err := bookmarks.GetTagsByUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}

	vm := &views.HomePageViewModel{
		User:         user,
		Collections:  collectionsList,
		Tags:         tags,
		CollectionID: collectionID,
		TagsFilter:   filterTags,
		Page:         filterPage,
		Search:       filterSearch,
		CurrentQuery: qs,
	}
	recent, all, err := loadBookmarks(ctx, bookmarks, user.ID, filterCollectionID, filterTags, filterSearch)
	if err != nil {
		return nil, fmt.Errorf("load bookmarks: %w", err)
	}
	vm.RecentBookmarks = recent
	vm.AllBookmarks = all
	return vm, nil
}

// loadBookmarks returns the bookmark lists for a user given the active filters.
// With a collection and/or tags filter, only the all list is populated; with no
// filters the recent list is populated too.
func loadBookmarks(ctx context.Context, bookmarks BookmarkStore, userID uuid.UUID, collectionID uuid.UUID, filterTags []string, search string) ([]core.Bookmark, []core.Bookmark, error) {
	if collectionID != uuid.Nil && len(filterTags) != 0 {
		all, err := bookmarks.GetBookmarksByCollectionAndTags(ctx, collectionID, filterTags, search)
		return nil, all, err
	}
	if collectionID != uuid.Nil {
		all, err := bookmarks.GetBookmarksByCollection(ctx, collectionID, search)
		return nil, all, err
	}
	if len(filterTags) != 0 {
		all, err := bookmarks.GetBookmarksByTags(ctx, userID, filterTags, search)
		return nil, all, err
	}
	recent, err := bookmarks.GetRecentBookmarksByUser(ctx, userID, search)
	if err != nil {
		return nil, nil, err
	}
	all, err := bookmarks.GetAllBookmarksByUser(ctx, userID, search)
	return recent, all, err
}

func handlePostBookmarks(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarks(w, r, comp.DB, comp.DB, comp.HTTPClient)
	}
}

type (
	titleFetcher interface {
		FetchPageTitle(ctx context.Context, u *url.URL) string
	}
)

// renderBookmarkFormErrors re-renders the add-bookmark form with field errors,
// preserving the submitted values. htmx requests retarget the swap to the form
// element itself, since the form's own hx-target is the bookmarks list
// (#bookmarks); plain requests re-render the full home page.
func renderBookmarkFormErrors(w http.ResponseWriter, r *http.Request, user core.User, collections CollectionStore, bookmarks BookmarkStore, f views.BookmarkForm) error {
	if views.IsHtmx(r) {
		collectionsList, err := collections.GetCollectionsByUser(r.Context(), user.ID)
		if err != nil {
			return fmt.Errorf("get collections: %w", err)
		}
		w.Header().Set("HX-Retarget", "#add-bookmark-form")
		w.Header().Set("HX-Reswap", "outerHTML")
		if err := views.NewBookmarkForm(f, collectionsList).Render(w); err != nil {
			return fmt.Errorf("render bookmark form: %w", err)
		}
		return nil
	}

	vm, err := loadHomeVM(r, user, collections, bookmarks)
	if err != nil {
		return err
	}
	vm.AddBookmark = f
	if err := vm.Render(w); err != nil {
		return fmt.Errorf("render home: %w", err)
	}
	return nil
}

func postBookmarks(w http.ResponseWriter, r *http.Request, collections CollectionStore, bookmarks BookmarkStore, fetcher titleFetcher) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	var formData views.BookmarkForm
	if err := form.NewDecoder(r.Body).Decode(&formData); err != nil {
		return fmt.Errorf("decode bookmark form: %w", err)
	}

	parsedURL, err := url.Parse(formData.URL)
	if err != nil {
		formData.Errors = views.FormErrors{"url": "that doesn't look like a valid URL"}
		return renderBookmarkFormErrors(w, r, user, collections, bookmarks, formData)
	}

	collectionID := uuid.Nil
	if formData.CollectionID != "" {
		collectionID, err = uuid.Parse(formData.CollectionID)
		if err != nil {
			return fmt.Errorf("invalid collection id %q in bookmark form: %w", formData.CollectionID, err)
		}
	}

	tags := strings.Split(formData.Tags, ",")
	filtered := tags[:0]
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			filtered = append(filtered, tag)
		}
	}
	tags = filtered

	title := fetcher.FetchPageTitle(ctx, parsedURL)

	if _, err := bookmarks.CreateBookmark(ctx, parsedURL, title, user.ID, collectionID, tags); err != nil {
		return fmt.Errorf("create bookmark: %w", err)
	}

	if views.IsHtmx(r) {
		collectionsList, err := collections.GetCollectionsByUser(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("get collections: %w", err)
		}
		tags, err := bookmarks.GetTagsByUser(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("get tags: %w", err)
		}
		recent, err := bookmarks.GetRecentBookmarksByUser(ctx, user.ID, "")
		if err != nil {
			return fmt.Errorf("get recent bookmarks: %w", err)
		}
		all, err := bookmarks.GetAllBookmarksByUser(ctx, user.ID, "")
		if err != nil {
			return fmt.Errorf("get all bookmarks: %w", err)
		}
		vm := views.HomePageViewModel{
			User:            user,
			Collections:     collectionsList,
			Tags:            tags,
			RecentBookmarks: recent,
			AllBookmarks:    all,
			CurrentQuery:    url.Values{},
		}
		if err := vm.RenderFiltered(w); err != nil {
			return fmt.Errorf("render filtered home: %w", err)
		}
		return nil
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
