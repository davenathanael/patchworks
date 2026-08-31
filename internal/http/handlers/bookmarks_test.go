package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/google/uuid"
)

var testUser = core.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Email: "me@example.com"}

func TestGetHomeNoFilters(t *testing.T) {
	bm := &fakeBookmarkStore{
		tags:   []core.Tag{{Name: "go", BookmarkCount: 1}},
		recent: []core.Bookmark{mustBookmark(t, "https://a.com", "A")},
		all:    []core.Bookmark{mustBookmark(t, "https://b.com", "B")},
	}
	col := &fakeCollectionStore{collections: []core.Collection{{ID: uuid.New(), Name: "Work"}}}

	rec := httptest.NewRecorder()
	getHome(rec, mustAuthedRequest(t, http.MethodGet, "/", nil), col, bm)

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "all", bm.last) // else-branch ran (only it calls GetAllBookmarksByUser)
}

func TestGetHomeCollectionFilter(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?collection_id="+id.String(), nil), col, bm)

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "collection", bm.last)
	be.Equal(t, id, bm.gotCollectionID)
}

func TestGetHomeTagsFilter(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?tags=go", nil), col, bm)

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "tags", bm.last)
}

func TestGetHomeCollectionAndTagsFilter(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?collection_id="+id.String()+"&tags=go", nil), col, bm)

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "collection+tags", bm.last)
	be.Equal(t, id, bm.gotCollectionID)
}

func TestGetHomeWithoutUser(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	getHome(rec, r, &fakeCollectionStore{}, &fakeBookmarkStore{})

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetHomeStoreError(t *testing.T) {
	rec := httptest.NewRecorder()
	col := &fakeCollectionStore{err: errFake}
	bm := &fakeBookmarkStore{}

	getHome(rec, mustAuthedRequest(t, http.MethodGet, "/", nil), col, bm)

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostBookmarksCreatesAndRedirects(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}
	fetcher := fakeTitleFetcher{title: "Example"}

	rec := httptest.NewRecorder()
	postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&tags=go%2C+web"), col, bm, fetcher)

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/", rec.Header().Get("Location"))
	be.Equal(t, 1, len(bm.created))
	created := bm.created[0]
	be.Equal(t, "http://example.com", created.bk.URL.String())
	be.Equal(t, "Example", created.bk.Title)
	be.Equal(t, testUser.ID, created.bk.Author.ID)
	be.Equal(t, uuid.Nil, created.colID)
	be.AllEqual(t, []string{"go", "web"}, created.bk.Tags)
}

func TestPostBookmarksEmptyTags(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&tags=%20%2C%20"), col, bm, fakeTitleFetcher{title: "Example"})

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, 1, len(bm.created))
	be.Equal(t, 0, len(bm.created[0].bk.Tags)) // no empty-string tag reaches the store
}

func TestPostBookmarksInvalidCollectionID(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&collection_id=not-a-uuid"), col, bm, fakeTitleFetcher{})

	be.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPostBookmarksMalformedURL(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2F%25"), col, bm, fakeTitleFetcher{})

	be.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPostBookmarksWithoutUser(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/bookmarks", nil)

	postBookmarks(rec, r, &fakeCollectionStore{}, &fakeBookmarkStore{}, fakeTitleFetcher{})

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostBookmarksHtmxRendersFiltered(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}
	rec := httptest.NewRecorder()

	r := mustFormRequest(t, "url=http%3A%2F%2Fexample.com")
	r.Header.Set("HX-Request", "true")
	postBookmarks(rec, r, col, bm, fakeTitleFetcher{title: "Example"})

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "", rec.Header().Get("Location")) // no redirect on htmx
}

// --- fakes & helpers ---

var errFake = errors.New("boom")

type createdBookmark struct {
	bk    core.Bookmark
	colID uuid.UUID
}

type fakeBookmarkStore struct {
	err             error
	tags            []core.Tag
	recent          []core.Bookmark
	all             []core.Bookmark
	last            string // which query method ran last
	gotCollectionID uuid.UUID
	created         []createdBookmark
}

func (f *fakeBookmarkStore) GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error) {
	return f.tags, f.err
}

func (f *fakeBookmarkStore) GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID, search string) ([]core.Bookmark, error) {
	f.last = "recent"
	return f.recent, f.err
}

func (f *fakeBookmarkStore) GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID, search string) ([]core.Bookmark, error) {
	f.last = "all"
	return f.all, f.err
}

func (f *fakeBookmarkStore) GetBookmarksByCollectionAndTags(ctx context.Context, collectionID uuid.UUID, tags []string, search string) ([]core.Bookmark, error) {
	f.last = "collection+tags"
	f.gotCollectionID = collectionID
	return f.all, f.err
}

func (f *fakeBookmarkStore) GetBookmarksByCollection(ctx context.Context, collectionID uuid.UUID, search string) ([]core.Bookmark, error) {
	f.last = "collection"
	f.gotCollectionID = collectionID
	return f.all, f.err
}

func (f *fakeBookmarkStore) GetBookmarksByTags(ctx context.Context, userID uuid.UUID, tags []string, search string) ([]core.Bookmark, error) {
	f.last = "tags"
	return f.all, f.err
}

func (f *fakeBookmarkStore) CreateBookmark(ctx context.Context, u *url.URL, title string, userID, collectionID uuid.UUID, tags []string) (core.Bookmark, error) {
	b := core.Bookmark{ID: uuid.New(), URL: u, Title: title, Author: core.User{ID: userID}, Tags: tags}
	f.created = append(f.created, createdBookmark{bk: b, colID: collectionID})
	return b, f.err
}

// fakeTitleFetcher returns a fixed title for any URL.
type fakeTitleFetcher struct{ title string }

func (f fakeTitleFetcher) FetchPageTitle(ctx context.Context, u *url.URL) string { return f.title }

func mustBookmark(t *testing.T, rawURL, title string) core.Bookmark {
	t.Helper()
	u, err := url.Parse(rawURL)
	be.NilErr(t, err)
	return core.Bookmark{ID: uuid.New(), URL: u, Title: title}
}

func mustAuthedRequest(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()
	r := httptest.NewRequestWithContext(context.Background(), method, target, body)
	return r.WithContext(middleware.WithUser(r.Context(), testUser))
}

func mustFormRequest(t *testing.T, encoded string) *http.Request {
	t.Helper()
	r := mustAuthedRequest(t, http.MethodPost, "/bookmarks", strings.NewReader(encoded))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}
