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
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, "/", nil), col, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "recent", bm.last) // else-branch ran (only it calls GetRecentBookmarksByUser)
}

func TestGetHomeCollectionFilter(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?collection_id="+id.String(), nil), col, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "collection", bm.last)
	be.Equal(t, id, bm.gotCollectionID)
}

func TestGetHomeTagsFilter(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?tags=go", nil), col, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "tags", bm.last)
}

func TestGetHomeCollectionAndTagsFilter(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?collection_id="+id.String()+"&tags=go", nil), col, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "collection+tags", bm.last)
	be.Equal(t, id, bm.gotCollectionID)
}

func TestGetHomeWithoutUser(t *testing.T) {
	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return getHome(w, r, &fakeCollectionStore{}, &fakeBookmarkStore{})
	}, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetHomeStoreError(t *testing.T) {
	col := &fakeCollectionStore{err: errFake}
	bm := &fakeBookmarkStore{}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return getHome(w, r, col, bm)
	}, mustAuthedRequest(t, http.MethodGet, "/", nil))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostBookmarksCreatesAndRedirects(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}
	fetcher := fakeTitleFetcher{title: "Example"}

	rec := httptest.NewRecorder()
	be.NilErr(t, postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&tags=go%2C+web"), col, bm, fetcher))

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
	be.NilErr(t, postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&tags=%20%2C%20"), col, bm, fakeTitleFetcher{title: "Example"}))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, 1, len(bm.created))
	be.Equal(t, 0, len(bm.created[0].bk.Tags)) // no empty-string tag reaches the store
}

func TestPostBookmarksInvalidCollectionID(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarks(w, r, col, bm, fakeTitleFetcher{})
	}, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&collection_id=not-a-uuid"))

	be.Equal(t, http.StatusInternalServerError, rec.Code) // tampered select value, not reachable via the UI
}

func TestPostBookmarksMalformedURL(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	err := postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2F%25"), col, bm, fakeTitleFetcher{})
	be.NilErr(t, err)

	be.Equal(t, http.StatusOK, rec.Code) // re-rendered home page with inline field error
	be.True(t, containsBody(rec, "valid URL"))
}

func TestPostBookmarksMalformedURLPreservesValues(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{collections: []core.Collection{{ID: uuid.New(), Name: "Work"}}}

	rec := httptest.NewRecorder()
	err := postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2F%25&tags=go%2C+web"), col, bm, fakeTitleFetcher{})
	be.NilErr(t, err)

	be.Equal(t, http.StatusOK, rec.Code)
	be.True(t, containsBody(rec, "valid URL"))
	be.True(t, containsBody(rec, `value="http://%"`)) // submitted URL preserved
	be.True(t, containsBody(rec, `value="go, web"`))  // submitted tags preserved
	be.True(t, containsBody(rec, "Dashboard"))        // full page, not a bare fragment
}

func TestPostBookmarksWithoutUser(t *testing.T) {
	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarks(w, r, &fakeCollectionStore{}, &fakeBookmarkStore{}, fakeTitleFetcher{})
	}, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/bookmarks", nil))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostBookmarksHtmxRendersFiltered(t *testing.T) {
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
	col := &fakeCollectionStore{}
	rec := httptest.NewRecorder()

	r := mustFormRequest(t, "url=http%3A%2F%2Fexample.com")
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarks(rec, r, col, bm, fakeTitleFetcher{title: "Example"}))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "", rec.Header().Get("Location")) // no redirect on htmx
}

func TestPostBookmarksHtmxMalformedURL(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}
	rec := httptest.NewRecorder()

	r := mustFormRequest(t, "url=http%3A%2F%2F%25&tags=go%2C+web")
	r.Header.Set("HX-Request", "true")
	err := postBookmarks(rec, r, col, bm, fakeTitleFetcher{})
	be.NilErr(t, err)

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "#add-bookmark-form", rec.Header().Get("HX-Retarget")) // error retargets the form, not #bookmarks
	be.True(t, containsBody(rec, "valid URL"))
	be.True(t, containsBody(rec, `value="go, web"`)) // submitted tags preserved in the fragment
}

func TestGetBookmarkByIdHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}

	rec := httptest.NewRecorder()
	r := routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String(), bm.one.ID.String())
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, getBookmarkById(rec, r, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "byid", bm.last)
	be.True(t, containsBody(rec, "Post"))
	be.True(t, containsBody(rec, "Bookmark actions")) // kebab menu present
}

func TestGetBookmarkEditHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}
	bm.one.Notes = "existing note"
	bm.one.Tags = []string{"go"}

	rec := httptest.NewRecorder()
	r := routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String()+"/edit", bm.one.ID.String())
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, getBookmarkEdit(rec, r, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.True(t, containsBody(rec, `name="notes"`))
	be.True(t, containsBody(rec, "existing note")) // pre-filled
	be.True(t, containsBody(rec, `name="tags"`))
	be.True(t, containsBody(rec, `value="go"`)) // tags joined into the input
	be.True(t, containsBody(rec, "Cancel"))
}

func TestGetBookmarkEditFullPage(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}

	rec := httptest.NewRecorder()
	be.NilErr(t, getBookmarkEdit(rec, routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String()+"/edit", bm.one.ID.String()), bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.True(t, containsBody(rec, "Edit Bookmark — Patchworks")) // full page, not a fragment
}

func TestPostBookmarkEditHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}

	rec := httptest.NewRecorder()
	r := routeFormRequest(t, bm.one.ID, "notes=new+note&tags=css%2C+web")
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarkEdit(rec, r, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "update", bm.last)
	be.Equal(t, "new note", bm.one.Notes)
	be.AllEqual(t, []string{"css", "web"}, bm.one.Tags)
	be.True(t, containsBody(rec, "new note")) // updated row fragment
}

func TestPostBookmarkEditRedirect(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}

	rec := httptest.NewRecorder()
	be.NilErr(t, postBookmarkEdit(rec, routeFormRequest(t, bm.one.ID, "notes=x&tags="), bm))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/", rec.Header().Get("Location"))
	be.Equal(t, "update", bm.last)
}

func TestPostBookmarkEditNotAuthor(t *testing.T) {
	bm := &fakeBookmarkStore{err: core.ErrNotFound}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkEdit(w, r, bm)
	}, routeFormRequest(t, uuid.New(), "notes=x"))

	be.Equal(t, http.StatusNotFound, rec.Code)
}

// --- fakes & helpers ---

// serve runs a Handler through Adapt so status codes are written by the
// HandleError wrapper (mirrors real registration).
func serve(h Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

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
	one             core.Bookmark // single-bookmark target (edit routes)
	last            string        // which query method ran last
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

func (f *fakeBookmarkStore) GetBookmarkByID(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	f.last = "byid"
	return f.one, f.err
}

func (f *fakeBookmarkStore) UpdateBookmarkNotesTags(ctx context.Context, id, userID uuid.UUID, notes string, tags []string) (core.Bookmark, error) {
	f.last = "update"
	if f.err != nil {
		return core.Bookmark{}, f.err
	}
	f.one.Notes = notes
	f.one.Tags = tags
	return f.one, nil
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
