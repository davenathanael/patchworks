package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/ajg/form"
	"github.com/davenathanael/patchworks/internal/components"
	"github.com/davenathanael/patchworks/internal/core"
	"github.com/davenathanael/patchworks/internal/http/middleware"
	"github.com/davenathanael/patchworks/internal/http/views"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Page sizes per list (FR-4): dashboard recent 10, filtered and collection
// detail 20.
const (
	recentPageSize   = 10
	filteredPageSize = 20
)

// maxFormBody bounds request bodies before r.ParseForm (gosec G120).
// Bookmark forms are tiny; 1 MiB is generous headroom.
const maxFormBody int64 = 1 << 20

// BookmarkStore is the interface for bookmark and tag persistence.
type BookmarkStore interface {
	GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error)
	GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error)
	GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error)
	GetBookmarksByCollectionAndTags(ctx context.Context, collectionID uuid.UUID, tags []string, search string, page core.CursorPage) (core.BookmarkPage, error)
	GetBookmarksByCollection(ctx context.Context, collectionID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error)
	GetBookmarksByTags(ctx context.Context, userID uuid.UUID, tags []string, search string, page core.CursorPage) (core.BookmarkPage, error)
	CreateBookmark(ctx context.Context, url *url.URL, title string, userID uuid.UUID, notes string, collectionIDs []uuid.UUID, tags []string, queued bool) (core.Bookmark, error)
	GetBookmarkByID(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error)
	FindUserBookmarkByURL(ctx context.Context, userID uuid.UUID, rawURL string) (core.Bookmark, bool, error)
	GetBookmarkForCollectionEdit(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error)
	UpdateBookmarkNotesTags(ctx context.Context, id, userID uuid.UUID, notes string, tags []string) (core.Bookmark, error)
	UpdateBookmarkCollectionIDs(ctx context.Context, bookmarkID, userID uuid.UUID, collectionIDs []uuid.UUID) (core.Bookmark, error)
	ArchiveBookmark(ctx context.Context, id, userID uuid.UUID) error
	GetArchivedBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error)
	RestoreBookmark(ctx context.Context, id, userID uuid.UUID) error
	DeleteBookmark(ctx context.Context, id, userID uuid.UUID) error
	GetQueuedBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error)
	EnqueueBookmark(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error)
	DequeueBookmark(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error)
}

// bookmarkCollectionStore is CollectionStore plus per-collection role lookups,
// used to guard bookmark↔collection membership writes.
type bookmarkCollectionStore interface {
	CollectionStore
	GetCollectionAccess(ctx context.Context, collectionID, userID uuid.UUID) (core.CollectionRole, error)
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
		if isPaging(r) {
			err = vm.RenderListItems(w)
		} else {
			err = vm.RenderFiltered(w)
		}
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
		Collections:  manageableCollections(collectionsList),
		Tags:         tags,
		CollectionID: collectionID,
		TagsFilter:   filterTags,
		Search:       filterSearch,
		CurrentQuery: qs,
	}
	older := parseCursor(qs)
	recent, filtered, err := loadBookmarks(ctx, bookmarks, user.ID, filterCollectionID, filterTags, filterSearch, core.CursorPage{Older: older})
	if err != nil {
		return nil, fmt.Errorf("load bookmarks: %w", err)
	}
	vm.Recent = recent
	vm.Filtered = filtered
	return vm, nil
}

// parseCursor reads the ?older paging cursor; a malformed value yields nil so
// a stale or hand-edited link degrades to a fresh view.
func parseCursor(qs url.Values) *core.BookmarkCursor {
	return core.DecodeCursor(qs.Get("older"))
}

// isPaging reports whether the request is a pager click rather than a
// filter navigation.
func isPaging(r *http.Request) bool {
	return r.URL.Query().Has("older")
}

// cursorPageOf builds the keyset page request for a list endpoint from the
// request's cursor and the list's page size.
func cursorPageOf(r *http.Request, limit int) core.CursorPage {
	return core.CursorPage{Older: parseCursor(r.URL.Query()), Limit: limit}
}

// listPagerProps builds the pager view-model for one paginated list: buttons
// target the list UL, the nav is OOB-swapped on pager clicks.
func listPagerProps(navID, listID, base string, qs url.Values, page core.BookmarkPage) views.ListPagerProps {
	return views.ListPagerProps{
		NavID:  navID,
		ListID: listID,
		Base:   base,
		Query:  qs,
		Page:   page,
	}
}

// loadBookmarks returns the bookmark pages for a user given the active
// filters. With a collection and/or tags filter, only the filtered page is
// populated; with no filters the recent page is populated too. The recent
// list is capped at 10 items, the filtered lists at 20 (BK-4, FR-4).
func loadBookmarks(ctx context.Context, bookmarks BookmarkStore, userID uuid.UUID, collectionID uuid.UUID, filterTags []string, search string, cursors core.CursorPage) (core.BookmarkPage, core.BookmarkPage, error) {
	if collectionID != uuid.Nil && len(filterTags) != 0 {
		cursors.Limit = filteredPageSize
		filtered, err := bookmarks.GetBookmarksByCollectionAndTags(ctx, collectionID, filterTags, search, cursors)
		return core.BookmarkPage{}, filtered, err
	}
	if collectionID != uuid.Nil {
		cursors.Limit = filteredPageSize
		filtered, err := bookmarks.GetBookmarksByCollection(ctx, collectionID, search, cursors)
		return core.BookmarkPage{}, filtered, err
	}
	if len(filterTags) != 0 {
		cursors.Limit = filteredPageSize
		filtered, err := bookmarks.GetBookmarksByTags(ctx, userID, filterTags, search, cursors)
		return core.BookmarkPage{}, filtered, err
	}
	if search != "" {
		cursors.Limit = filteredPageSize
		filtered, err := bookmarks.GetAllBookmarksByUser(ctx, userID, search, cursors)
		return core.BookmarkPage{}, filtered, err
	}
	cursors.Limit = recentPageSize
	recent, err := bookmarks.GetRecentBookmarksByUser(ctx, userID, "", cursors)
	if err != nil {
		return core.BookmarkPage{}, core.BookmarkPage{}, err
	}
	return recent, core.BookmarkPage{}, nil
}

// manageableCollections keeps only the collections whose role for the
// requesting user allows managing bookmarks, so forms never offer more.
func manageableCollections(collections []core.Collection) []core.Collection {
	return slices.DeleteFunc(slices.Clone(collections), func(c core.Collection) bool {
		return !c.Role.Allows(core.PermManageBookmarks)
	})
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
// checkDuplicateURL returns the FR-11 reminder for an exact-URL bookmark of
// the same author; saveAnyway (the warned form's hidden field) skips the check.
func checkDuplicateURL(ctx context.Context, bookmarks BookmarkStore, userID uuid.UUID, u *url.URL, saveAnyway bool) (*views.Duplicate, error) {
	if saveAnyway {
		return nil, nil
	}
	dup, found, err := bookmarks.FindUserBookmarkByURL(ctx, userID, u.String())
	if err != nil {
		return nil, fmt.Errorf("find duplicate bookmark: %w", err)
	}
	if !found {
		return nil, nil
	}
	return views.NewDuplicate(dup), nil
}

func renderBookmarkFormErrors(w http.ResponseWriter, r *http.Request, user core.User, collections CollectionStore, bookmarks BookmarkStore, f views.BookmarkForm) error {
	if views.IsHtmx(r) {
		collectionsList, err := collections.GetCollectionsByUser(r.Context(), user.ID)
		if err != nil {
			return fmt.Errorf("get collections: %w", err)
		}
		w.Header().Set("HX-Retarget", "#add-bookmark-form")
		w.Header().Set("HX-Reswap", "outerHTML")
		if err := views.NewBookmarkForm(f, manageableCollections(collectionsList)).Render(w); err != nil {
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

func postBookmarks(w http.ResponseWriter, r *http.Request, collections bookmarkCollectionStore, bookmarks BookmarkStore, fetcher titleFetcher) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	var formData views.BookmarkForm
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBody)
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parse bookmark form: %w", err)
	}
	// the picker's q and collection_id keys are handled outside ajg/form
	decoder := form.NewDecoder(nil)
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.DecodeValues(&formData, r.PostForm); err != nil {
		return fmt.Errorf("decode bookmark form: %w", err)
	}
	// checkbox lists arrive as repeated keys, which ajg/form only supports
	// with explicit indexes — read them natively instead
	formData.CollectionIDs = r.PostForm["collection_id"]

	parsedURL, err := url.Parse(formData.URL)
	if err != nil {
		formData.Errors = views.FormErrors{"url": "that doesn't look like a valid URL"}
		return renderBookmarkFormErrors(w, r, user, collections, bookmarks, formData)
	}

	collectionIDs, err := parseBookmarkCollectionIDs(ctx, collections, user.ID, formData.CollectionIDs)
	if err != nil {
		return err
	}

	var dup *views.Duplicate
	dup, err = checkDuplicateURL(ctx, bookmarks, user.ID, parsedURL, formData.SaveAnyway)
	if err != nil {
		return err
	}
	if dup != nil {
		formData.Duplicate = dup
		return renderBookmarkFormErrors(w, r, user, collections, bookmarks, formData)
	}

	tags := splitTags(formData.Tags)

	title := fetcher.FetchPageTitle(ctx, parsedURL)

	if _, err := bookmarks.CreateBookmark(ctx, parsedURL, title, user.ID, formData.Notes, collectionIDs, tags, formData.Queued); err != nil {
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
		recent, err := bookmarks.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: recentPageSize})
		if err != nil {
			return fmt.Errorf("get recent bookmarks: %w", err)
		}
		vm := views.HomePageViewModel{
			User:         user,
			Collections:  manageableCollections(collectionsList),
			Tags:         tags,
			Recent:       recent,
			CurrentQuery: url.Values{},
		}
		if err := vm.RenderFiltered(w); err != nil {
			return fmt.Errorf("render filtered home: %w", err)
		}
		return nil
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

// parseBookmarkCollectionIDs validates and dedupes the picker's collection
// ids, enforcing manage rights per collection: one unmanageable id fails the
// whole save (nothing is created).
func parseBookmarkCollectionIDs(ctx context.Context, collections bookmarkCollectionStore, userID uuid.UUID, raws []string) ([]uuid.UUID, error) {
	collectionIDs := make([]uuid.UUID, 0, len(raws))
	seen := make(map[uuid.UUID]bool, len(raws))
	for _, raw := range raws {
		if raw == "" {
			continue
		}
		cid, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid collection id %q in bookmark form: %w", raw, err)
		}
		if seen[cid] {
			continue
		}
		seen[cid] = true
		if err = requireManageBookmarks(ctx, collections, userID, cid); err != nil {
			return nil, err
		}
		collectionIDs = append(collectionIDs, cid)
	}
	return collectionIDs, nil
}

func handleGetBookmarkById(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getBookmarkById(w, r, comp.DB, comp.DB)
	}
}

// getBookmarkById returns the bookmark row fragment (htmx cancel/refresh) or
// redirects to the dashboard for plain requests.
func getBookmarkById(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore, collections CollectionStore) error {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	bm, err := loadEditableBookmark(r, user, bookmarks)
	if err != nil {
		return err
	}

	if views.IsHtmx(r) {
		allCollections, err := collections.GetCollectionsByUser(r.Context(), user.ID)
		if err != nil {
			return fmt.Errorf("get collections: %w", err)
		}
		if err := views.LinkRow(bm, manageableCollections(allCollections), "").Render(w); err != nil {
			return fmt.Errorf("render bookmark row: %w", err)
		}
		return nil
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func handleGetBookmarkCollectionsEdit(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getBookmarkCollectionsEdit(w, r, comp.DB, comp.DB)
	}
}

// getBookmarkCollectionsEdit renders the inline membership panel (htmx
// fragment) or a full edit page (no-JS fallback). The ?collection= query param
// carries the page's collection so the save can drop the row when it's
// unchecked.
func getBookmarkCollectionsEdit(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore, collections CollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	bm, err := loadBookmarkForCollectionEdit(r, user, bookmarks)
	if err != nil {
		return err
	}
	allCollections, err := collections.GetCollectionsByUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get collections: %w", err)
	}

	currentCollection := r.URL.Query().Get("collection")
	panel := views.CollectionEditPanel(bm, manageableCollections(allCollections), currentCollection)
	if views.IsHtmx(r) {
		if err := views.EditPanelRow(panel).Render(w); err != nil {
			return fmt.Errorf("render collections edit panel: %w", err)
		}
		return nil
	}
	if err := views.EditCollectionsPage(user, panel).Render(w); err != nil {
		return fmt.Errorf("render edit collections page: %w", err)
	}
	return nil
}

func handleGetBookmarkEdit(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getBookmarkEdit(w, r, comp.DB)
	}
}

// getBookmarkEdit renders the inline edit panel (htmx fragment) or a full edit
// page (no-JS fallback).
func getBookmarkEdit(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	bm, err := loadEditableBookmark(r, user, bookmarks)
	if err != nil {
		return err
	}

	panel := views.BookmarkEditPanel(bm, views.FormErrors{})
	if views.IsHtmx(r) {
		if err := views.EditPanelRow(panel).Render(w); err != nil {
			return fmt.Errorf("render bookmark edit panel: %w", err)
		}
		return nil
	}
	if err := views.EditBookmarkPage(user, panel).Render(w); err != nil {
		return fmt.Errorf("render edit bookmark page: %w", err)
	}
	return nil
}

func handlePostBookmarkEdit(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkEdit(w, r, comp.DB, comp.DB)
	}
}

// postBookmarkEdit saves notes + tags (author-only) and returns the updated row
// fragment (htmx) or redirects to the dashboard (plain form submit).
func postBookmarkEdit(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore, collections CollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse bookmark id %q: %w", rawID, core.ErrNotFound)
	}

	var f editBookmarkForm
	if err = form.NewDecoder(r.Body).Decode(&f); err != nil {
		return fmt.Errorf("decode edit bookmark form: %w", err)
	}
	tags := splitTags(f.Tags)

	updated, err := bookmarks.UpdateBookmarkNotesTags(ctx, id, user.ID, f.Notes, tags)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("update bookmark: %w", err)
	}

	if views.IsHtmx(r) {
		allCollections, err := collections.GetCollectionsByUser(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("get collections: %w", err)
		}
		if err := views.LinkRow(updated, manageableCollections(allCollections), "").Render(w); err != nil {
			return fmt.Errorf("render bookmark row: %w", err)
		}
		return nil
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func handlePostBookmarkArchive(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkArchive(w, r, comp.DB)
	}
}

// postBookmarkArchive soft-deletes the bookmark. htmx deletes the row client-
// side (hx-swap="delete"); plain submits redirect to the dashboard.
func postBookmarkArchive(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse bookmark id %q: %w", rawID, core.ErrNotFound)
	}

	if err := bookmarks.ArchiveBookmark(ctx, id, user.ID); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("archive bookmark: %w", err)
	}

	if views.IsHtmx(r) {
		return nil
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func handlePostBookmarkReading(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkReading(w, r, comp.DB)
	}
}

// postBookmarkReading toggles the reading-list queue: queued bookmarks are
// dequeued ("mark as read"), everything else is enqueued. htmx requests get
// an empty 200 so the client deletes the row; plain submits redirect to next
// (validated same-site relative, else /).
func postBookmarkReading(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse bookmark id %q: %w", rawID, core.ErrNotFound)
	}

	b, err := bookmarks.GetBookmarkByID(ctx, id, user.ID)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("get bookmark: %w", err)
	}

	if b.QueuedAt.IsZero() {
		if _, err := bookmarks.EnqueueBookmark(ctx, id, user.ID); err != nil {
			if errors.Is(err, core.ErrNotFound) {
				return err
			}
			return fmt.Errorf("enqueue bookmark: %w", err)
		}
	} else {
		if _, err := bookmarks.DequeueBookmark(ctx, id, user.ID); err != nil {
			if errors.Is(err, core.ErrNotFound) {
				return err
			}
			return fmt.Errorf("dequeue bookmark: %w", err)
		}
	}

	if views.IsHtmx(r) {
		return nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBody)
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parse reading form: %w", err)
	}
	// next is restricted to same-site relative paths (or "/") by safeNext
	http.Redirect(w, r, safeNext(r.PostForm.Get("next")), http.StatusSeeOther)
	return nil
}

// safeNext allows only same-site relative redirects: a path starting with a
// single slash, never protocol-relative //host.
func safeNext(next string) string {
	if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") {
		return next
	}
	return "/"
}

func handlePostBookmarkCollections(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkCollections(w, r, comp.DB, comp.DB)
	}
}

func handleGetArchived(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getArchived(w, r, comp.DB)
	}
}

// getArchived renders the archived-bookmarks page (list + restore/delete).
func getArchived(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	archived, err := bookmarks.GetArchivedBookmarksByUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get archived bookmarks: %w", err)
	}
	if err := views.ArchivedPage(user, archived).Render(w); err != nil {
		return fmt.Errorf("render archived page: %w", err)
	}
	return nil
}

func handleGetReading(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getReading(w, r, comp.DB)
	}
}

// getReading renders the FIFO reading list (FR-10); rows dequeue themselves
// via "Mark as read".
func getReading(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	queued, err := bookmarks.GetQueuedBookmarksByUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get queued bookmarks: %w", err)
	}
	if err := views.ReadingPage(user, queued).Render(w); err != nil {
		return fmt.Errorf("render reading page: %w", err)
	}
	return nil
}

func handlePostBookmarkRestore(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkRestore(w, r, comp.DB)
	}
}

// postBookmarkRestore clears archived_at; the htmx client deletes the row
// (hx-swap="delete"), plain submits return to the archived page.
func postBookmarkRestore(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse bookmark id %q: %w", rawID, core.ErrNotFound)
	}
	if err := bookmarks.RestoreBookmark(ctx, id, user.ID); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("restore bookmark: %w", err)
	}
	if views.IsHtmx(r) {
		return nil
	}
	http.Redirect(w, r, "/archived", http.StatusSeeOther)
	return nil
}

func handlePostBookmarkDelete(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkDelete(w, r, comp.DB)
	}
}

// postBookmarkDelete permanently removes the bookmark; the htmx client deletes
// the row, plain submits return to the archived page.
func postBookmarkDelete(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse bookmark id %q: %w", rawID, core.ErrNotFound)
	}
	if err := bookmarks.DeleteBookmark(ctx, id, user.ID); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("delete bookmark: %w", err)
	}
	if views.IsHtmx(r) {
		return nil
	}
	http.Redirect(w, r, "/archived", http.StatusSeeOther)
	return nil
}

// postBookmarkCollections replaces the bookmark's collection membership from
// the picker's checked boxes. From a collection page, unchecking that
// collection drops the row (HX-Reswap: delete); otherwise the updated row is
// swapped in. Plain submits redirect back to where they came from.
func postBookmarkCollections(w http.ResponseWriter, r *http.Request, bookmarks BookmarkStore, collections bookmarkCollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	bm, err := loadBookmarkForCollectionEdit(r, user, bookmarks)
	if err != nil {
		return err
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBody)
	if err = r.ParseForm(); err != nil {
		return fmt.Errorf("parse edit collections form: %w", err)
	}
	collectionIDs := make([]uuid.UUID, 0, len(r.PostForm["collections"]))
	for _, raw := range r.PostForm["collections"] {
		if cid, perr := uuid.Parse(raw); perr == nil {
			collectionIDs = append(collectionIDs, cid)
		}
	}
	currentCollection := r.PostFormValue("current_collection")

	// Only collections being added or removed need manage rights; unchanged
	// memberships pass, so a demoted member keeps what they already have.
	for _, cid := range changedCollectionIDs(bm.CollectionIDs, collectionIDs) {
		if err = requireManageBookmarks(ctx, collections, user.ID, cid); err != nil {
			return err
		}
	}

	updated, err := bookmarks.UpdateBookmarkCollectionIDs(ctx, bm.ID, user.ID, collectionIDs)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("update bookmark collections: %w", err)
	}

	if views.IsHtmx(r) {
		return renderBookmarkCollectionsRow(ctx, w, collections, user.ID, updated, collectionIDs, currentCollection)
	}

	target := "/"
	if cid, perr := uuid.Parse(currentCollection); perr == nil {
		target = "/collections/" + cid.String()
	}
	http.Redirect(w, r, target, http.StatusSeeOther) // #nosec G710 -- target is "/" or a parsed UUID's canonical String(), no open redirect
	return nil
}

// renderBookmarkCollectionsRow writes the htmx fragment after a collections
// edit: from a collection page it removes the row when that collection was
// unchecked, otherwise it re-renders the updated bookmark row.
func renderBookmarkCollectionsRow(ctx context.Context, w http.ResponseWriter, collections bookmarkCollectionStore, userID uuid.UUID, updated core.Bookmark, checked []uuid.UUID, currentCollection string) error {
	if currentID, perr := uuid.Parse(currentCollection); perr == nil && !slices.Contains(checked, currentID) {
		w.Header().Set("HX-Reswap", "delete")
		return nil
	}
	allCollections, err := collections.GetCollectionsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get collections: %w", err)
	}
	if err = views.LinkRow(updated, manageableCollections(allCollections), currentCollection).Render(w); err != nil {
		return fmt.Errorf("render bookmark row: %w", err)
	}
	return nil
}

// requireManageBookmarks returns core.ErrForbidden unless the user's role in
// the collection allows managing bookmarks; core.ErrNotFound (non-member or
// missing collection) propagates as a 404.
func requireManageBookmarks(ctx context.Context, collections bookmarkCollectionStore, userID, collectionID uuid.UUID) error {
	role, err := collections.GetCollectionAccess(ctx, collectionID, userID)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("get collection access: %w", err)
	}
	if !role.Allows(core.PermManageBookmarks) {
		return core.ErrForbidden
	}
	return nil
}

// changedCollectionIDs returns the ids in exactly one of the two sets — the
// memberships a request would add or remove relative to the current state.
func changedCollectionIDs(current, next []uuid.UUID) []uuid.UUID {
	changed := make([]uuid.UUID, 0, len(current)+len(next))
	for _, id := range next {
		if !slices.Contains(current, id) {
			changed = append(changed, id)
		}
	}
	for _, id := range current {
		if !slices.Contains(next, id) {
			changed = append(changed, id)
		}
	}
	return changed
}

// loadBookmarkID parses the :id route param, mapping a malformed id to
// core.ErrNotFound (404).
func loadBookmarkID(r *http.Request) (uuid.UUID, error) {
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse bookmark id %q: %w", rawID, core.ErrNotFound)
	}
	return id, nil
}

// loadEditableBookmark parses the :id route param and fetches the user's own
// bookmark (ErrNotFound propagates as a 404).
func loadEditableBookmark(r *http.Request, user core.User, bookmarks BookmarkStore) (core.Bookmark, error) {
	id, err := loadBookmarkID(r)
	if err != nil {
		return core.Bookmark{}, err
	}
	bm, err := bookmarks.GetBookmarkByID(r.Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return core.Bookmark{}, err
		}
		return core.Bookmark{}, fmt.Errorf("get bookmark: %w", err)
	}
	return bm, nil
}

// loadBookmarkForCollectionEdit fetches the bookmark for the collections
// picker: the author, or a member with manage rights (owner/editor) in a
// collection containing it. Viewers and strangers get ErrNotFound (404),
// matching the hidden panel.
func loadBookmarkForCollectionEdit(r *http.Request, user core.User, bookmarks BookmarkStore) (core.Bookmark, error) {
	id, err := loadBookmarkID(r)
	if err != nil {
		return core.Bookmark{}, err
	}
	bm, err := bookmarks.GetBookmarkForCollectionEdit(r.Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return core.Bookmark{}, err
		}
		return core.Bookmark{}, fmt.Errorf("get bookmark: %w", err)
	}
	return bm, nil
}

// splitTags trims and filters empty tags from a comma-separated input.
func splitTags(raw string) []string {
	tags := strings.Split(raw, ",")
	filtered := tags[:0]
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			filtered = append(filtered, tag)
		}
	}
	return filtered
}

type editBookmarkForm struct {
	Notes string `form:"notes"`
	Tags  string `form:"tags"`
}
