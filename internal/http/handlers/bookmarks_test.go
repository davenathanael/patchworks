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
	"time"

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

func TestGetHomeRecentCursorPage(t *testing.T) {
	cursor := core.BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), ID: uuid.New()}
	bm := &fakeBookmarkStore{recent: []core.Bookmark{mustBookmark(t, "https://a.com", "A")}, hasOlder: true}

	rec := httptest.NewRecorder()
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?older="+core.EncodeCursor(cursor), nil), &fakeCollectionStore{}, bm))

	be.Equal(t, "recent", bm.last)
	be.Equal(t, 10, bm.page.Limit) // recent page size
	be.Equal(t, cursor, *bm.page.Older)
}

func TestGetHomeFilteredPageSizes(t *testing.T) {
	colID := uuid.New()
	tests := []struct {
		name      string
		target    string
		wantLast  string
		wantLimit int
	}{
		{"search", "/?search=go", "all", 20},
		{"collection", "/?collection_id=" + colID.String(), "collection", 20},
		{"tags", "/?tags=go", "tags", 20},
		{"collection+tags", "/?collection_id=" + colID.String() + "&tags=go", "collection+tags", 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}}
			rec := httptest.NewRecorder()
			be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, tc.target, nil), &fakeCollectionStore{}, bm))
			be.Equal(t, tc.wantLast, bm.last)
			be.Equal(t, tc.wantLimit, bm.page.Limit)
		})
	}
}

func TestGetHomeCursorEdgeCases(t *testing.T) {
	cursor := core.BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), ID: uuid.New()}
	tests := []struct {
		name      string
		target    string
		wantOlder *core.BookmarkCursor
	}{
		{"invalid cursor ignored", "/?older=!!!garbage!!!", nil},
		{"empty cursor ignored", "/?older=", nil},
		{"newer param ignored", "/?newer=" + core.EncodeCursor(cursor), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bm := &fakeBookmarkStore{recent: []core.Bookmark{mustBookmark(t, "https://a.com", "A")}}
			rec := httptest.NewRecorder()
			be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, tc.target, nil), &fakeCollectionStore{}, bm))
			b := bm.page
			be.Equal(t, tc.wantOlder != nil, b.Older != nil)
			if b.Older != nil {
				be.Equal(t, *tc.wantOlder, *b.Older)
			}
		})
	}
}

func TestGetHomeRecentShowsLoadMoreOnly(t *testing.T) {
	bm := &fakeBookmarkStore{recent: []core.Bookmark{mustBookmark(t, "https://a.com", "A")}, hasOlder: true}

	rec := httptest.NewRecorder()
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, "/", nil), &fakeCollectionStore{}, bm))

	be.True(t, containsBody(rec, "Load more"))
	be.True(t, !containsBody(rec, "Load previous"))
	be.True(t, containsBody(rec, `id="recent-pager"`))
	be.True(t, containsBody(rec, `id="recent-list"`))
}

func TestGetHomeFilteredPagerPreservesFilters(t *testing.T) {
	colID := uuid.New()
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}, hasOlder: true}

	rec := httptest.NewRecorder()
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, "/?tags=go&collection_id="+colID.String(), nil), &fakeCollectionStore{}, bm))

	be.True(t, containsBody(rec, `id="bookmarks-pager"`))
	be.True(t, containsBody(rec, "Load more"))
	be.True(t, containsBody(rec, "tags=go")) // plain hrefs keep filters
	be.True(t, containsBody(rec, "collection_id="+colID.String()))
	// fresh load: no back link, no load previous
	be.False(t, containsBody(rec, "Back to latest"))
	be.False(t, containsBody(rec, "Load previous"))
}

func TestGetHomeOlderBatchShowsBackToLatest(t *testing.T) {
	cursor := core.BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), ID: uuid.New()}
	colID := uuid.New()
	bm := &fakeBookmarkStore{all: []core.Bookmark{mustBookmark(t, "https://b.com", "B")}, hasOlder: true, hasNewer: true}

	rec := httptest.NewRecorder()
	target := "/?tags=go&collection_id=" + colID.String() + "&older=" + core.EncodeCursor(cursor)
	be.NilErr(t, getHome(rec, mustAuthedRequest(t, http.MethodGet, target, nil), &fakeCollectionStore{}, bm))

	be.True(t, containsBody(rec, "Back to latest"))
	be.True(t, containsBody(rec, `href="/?collection_id=`+colID.String()+`&amp;tags=go"`)) // cursor params stripped, filters kept
	be.True(t, !containsBody(rec, "Load previous"))
}

func TestGetHomeHtmxPagerFragment(t *testing.T) {
	cursor := core.BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), ID: uuid.New()}
	bm := &fakeBookmarkStore{recent: []core.Bookmark{mustBookmark(t, "https://a.com", "A")}, hasOlder: true, hasNewer: true}

	r := mustAuthedRequest(t, http.MethodGet, "/?older="+core.EncodeCursor(cursor), nil)
	r.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	be.NilErr(t, getHome(rec, r, &fakeCollectionStore{}, bm))

	be.True(t, containsBody(rec, "A"))                               // the fetched rows
	be.True(t, strings.Count(rec.Body.String(), "hx-swap-oob") == 1) // pager nav only, no filters OOB
	be.True(t, containsBody(rec, `id="recent-pager"`))
	be.True(t, containsBody(rec, "Back to latest")) // OOB pager offers the way up
	be.True(t, !containsBody(rec, "Dashboard"))     // items fragment, not the full page
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
	be.Equal(t, 0, len(created.colIDs))
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

func TestPostBookmarksDuplicateURLWarns(t *testing.T) {
	existing := mustBookmark(t, "https://example.com/dup", "Already Saved")
	bm := &fakeBookmarkStore{dup: &existing}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	r := mustFormRequest(t, "url=https%3A%2F%2Fexample.com%2Fdup&tags=go")
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarks(rec, r, col, bm, fakeTitleFetcher{title: "Fresh"}))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, 0, len(bm.created)) // nothing saved until the reminder is confirmed
	be.Equal(t, "#add-bookmark-form", rec.Header().Get("HX-Retarget"))
	be.True(t, containsBody(rec, "Save anyway"))
	be.True(t, containsBody(rec, "Already Saved"))
	be.True(t, containsBody(rec, `value="go"`)) // submitted tags preserved
}

func TestPostBookmarksForceSavesDuplicate(t *testing.T) {
	existing := mustBookmark(t, "https://example.com/dup", "Already Saved")
	bm := &fakeBookmarkStore{dup: &existing}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	r := mustFormRequest(t, "url=https%3A%2F%2Fexample.com%2Fdup&save_anyway=true")
	be.NilErr(t, postBookmarks(rec, r, col, bm, fakeTitleFetcher{title: "Fresh"}))

	be.Equal(t, 1, len(bm.created)) // confirmed → saves
}

func TestPostBookmarksWithNotes(t *testing.T) {
	saved := mustBookmark(t, "https://example.com", "Example")
	saved.Notes = "Worth rereading"
	bm := &fakeBookmarkStore{recent: []core.Bookmark{saved}}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	r := mustFormRequest(t, "url=http%3A%2F%2Fexample.com&notes=Worth+rereading")
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarks(rec, r, col, bm, fakeTitleFetcher{title: "Example"}))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, 1, len(bm.created))
	be.Equal(t, "Worth rereading", bm.created[0].bk.Notes) // notes stored
	be.True(t, containsBody(rec, "Worth rereading"))       // note rendered in the row fragment
}

func TestPostBookmarksMultipleCollections(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{accessRole: core.RoleOwner}
	first, second := uuid.New(), uuid.New()

	rec := httptest.NewRecorder()
	body := "url=http%3A%2F%2Fexample.com&collection_id=" + first.String() +
		"&collection_id=" + second.String() + "&collection_id=" + first.String()
	be.NilErr(t, postBookmarks(rec, mustFormRequest(t, body), col, bm, fakeTitleFetcher{title: "Example"}))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, 1, len(bm.created))
	be.AllEqual(t, []uuid.UUID{first, second}, bm.created[0].colIDs) // deduped, order kept
	be.Equal(t, 2, col.accessCalls)                                  // manage rights checked per distinct collection
}

func TestPostBookmarksCollectionAccessDenied(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{accessRole: core.RoleViewer}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarks(w, r, col, bm, fakeTitleFetcher{})
	}, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&collection_id="+uuid.New().String()))

	be.Equal(t, http.StatusForbidden, rec.Code)
	be.Equal(t, 0, len(bm.created)) // whole save fails, nothing created
}

func TestPostBookmarkFormErrorsPreserveNotesAndPicks(t *testing.T) {
	cid := uuid.New()
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{collections: []core.Collection{{ID: cid, Name: "Work", BookmarkCount: 2, Role: core.RoleOwner}}}

	rec := httptest.NewRecorder()
	body := "url=http%3A%2F%2F%25&tags=go&notes=keep+me&collection_id=" + cid.String()
	be.NilErr(t, postBookmarks(rec, mustFormRequest(t, body), col, bm, fakeTitleFetcher{}))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, 0, len(bm.created))
	be.True(t, containsBody(rec, "valid URL"))
	be.True(t, containsBody(rec, "keep me"))    // notes preserved on re-render
	be.True(t, containsBody(rec, "checked"))    // selected collection re-checked
	be.True(t, containsBody(rec, `value="go"`)) // tags preserved
}

func TestGetBookmarkByIdHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}

	rec := httptest.NewRecorder()
	r := routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String(), bm.one.ID.String())
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, getBookmarkById(rec, r, bm, &fakeCollectionStore{}))

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
	be.True(t, strings.HasPrefix(rec.Body.String(), "<li>")) // row-swap expects Li
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
	be.NilErr(t, postBookmarkEdit(rec, r, bm, &fakeCollectionStore{}))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "update", bm.last)
	be.Equal(t, "new note", bm.one.Notes)
	be.AllEqual(t, []string{"css", "web"}, bm.one.Tags)
	be.True(t, containsBody(rec, "new note")) // updated row fragment
}

func TestPostBookmarkEditRedirect(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}

	rec := httptest.NewRecorder()
	be.NilErr(t, postBookmarkEdit(rec, routeFormRequest(t, bm.one.ID, "notes=x&tags="), bm, &fakeCollectionStore{}))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/", rec.Header().Get("Location"))
	be.Equal(t, "update", bm.last)
}

func TestPostBookmarkEditNotAuthor(t *testing.T) {
	bm := &fakeBookmarkStore{err: core.ErrNotFound}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkEdit(w, r, bm, &fakeCollectionStore{})
	}, routeFormRequest(t, uuid.New(), "notes=x"))

	be.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetBookmarkCollectionsEditHtmx(t *testing.T) {
	cid := uuid.New()
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}
	bm.one.CollectionIDs = []uuid.UUID{cid}
	col := &fakeCollectionStore{collections: []core.Collection{{ID: cid, Name: "Work", BookmarkCount: 3, Role: core.RoleOwner}}}

	rec := httptest.NewRecorder()
	r := routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String()+"/collections/edit", bm.one.ID.String())
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, getBookmarkCollectionsEdit(rec, r, bm, col))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "byid-collection", bm.last)
	be.True(t, strings.HasPrefix(rec.Body.String(), "<li>")) // row-swap expects Li
	be.True(t, containsBody(rec, "Edit collections"))
	be.True(t, containsBody(rec, "Work"))
	be.True(t, containsBody(rec, `name="collections"`))
	be.True(t, containsBody(rec, "checked")) // current membership pre-checked
}

func TestGetBookmarkCollectionsEditFullPage(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, getBookmarkCollectionsEdit(rec, routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String()+"/collections/edit", bm.one.ID.String()), bm, col))

	be.Equal(t, http.StatusOK, rec.Code)
	be.True(t, containsBody(rec, "Edit Collections — Patchworks")) // full page, not a fragment
}

func TestEditorManagesOtherBookmarkCollections(t *testing.T) {
	cid := uuid.New()
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}
	bm.one.CollectionIDs = []uuid.UUID{cid}
	col := &fakeCollectionStore{accessRole: core.RoleEditor, collections: []core.Collection{{ID: cid, Name: "Work", Role: core.RoleEditor}}}

	// a non-author editor can open the picker for a bookmark they didn't add
	rec := httptest.NewRecorder()
	r := routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String()+"/collections/edit", bm.one.ID.String())
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, getBookmarkCollectionsEdit(rec, r, bm, col))
	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "byid-collection", bm.last)
	be.True(t, containsBody(rec, "Work"))

	// and can remove it from the collection they manage
	rec = httptest.NewRecorder()
	r = routeFormRequest(t, bm.one.ID, "") // empty checked set → removed from Work
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarkCollections(rec, r, bm, col))
	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "update-collections", bm.last)
	be.Equal(t, 0, len(bm.one.CollectionIDs))
	be.True(t, containsBody(rec, "Post"))
}

func TestNonManagerCannotEditOthersBookmarkCollections(t *testing.T) {
	bm := &fakeBookmarkStore{
		err: errors.Join(core.ErrNotFound), // non-author without manage rights reads as 404
		one: mustBookmark(t, "https://example.com", "Post"),
	}
	col := &fakeCollectionStore{accessRole: core.RoleViewer}

	r := routeRequest(t, http.MethodGet, "/bookmarks/"+bm.one.ID.String()+"/collections/edit", bm.one.ID.String())
	r.Header.Set("HX-Request", "true")
	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return getBookmarkCollectionsEdit(w, r, bm, col)
	}, r)
	be.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostBookmarkCollectionsHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}
	col := &fakeCollectionStore{accessRole: core.RoleOwner, collections: []core.Collection{{ID: uuid.New(), Name: "Work", Role: core.RoleOwner}}}

	rec := httptest.NewRecorder()
	body := "collections=" + col.collections[0].ID.String()
	r := routeFormRequest(t, bm.one.ID, body)
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarkCollections(rec, r, bm, col))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "update-collections", bm.last)
	be.AllEqual(t, []uuid.UUID{col.collections[0].ID}, bm.one.CollectionIDs)
	be.True(t, containsBody(rec, "Post")) // updated row fragment
}

func TestPostBookmarkCollectionsRemovesFromCurrentCollection(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}
	col := &fakeCollectionStore{}
	current := uuid.New()

	rec := httptest.NewRecorder()
	// unchecked the current collection → empty checked set
	r := routeFormRequest(t, bm.one.ID, "current_collection="+current.String())
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarkCollections(rec, r, bm, col))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "delete", rec.Header().Get("HX-Reswap")) // htmx deletes the row
}

func TestPostBookmarkCollectionsRedirect(t *testing.T) {
	bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Post")}
	current := uuid.New()

	rec := httptest.NewRecorder()
	r := routeFormRequest(t, bm.one.ID, "current_collection="+current.String())
	be.NilErr(t, postBookmarkCollections(rec, r, bm, &fakeCollectionStore{}))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections/"+current.String(), rec.Header().Get("Location"))
}

func TestPostBookmarkCollectionsNotAuthor(t *testing.T) {
	bm := &fakeBookmarkStore{err: core.ErrNotFound}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkCollections(w, r, bm, &fakeCollectionStore{})
	}, routeFormRequest(t, uuid.New(), "collections="+uuid.New().String()))

	be.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostBookmarkArchiveHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{}

	rec := httptest.NewRecorder()
	r := routeFormRequest(t, uuid.New(), "")
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarkArchive(rec, r, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "archive", bm.last)
	be.Equal(t, "", rec.Body.String()) // htmx deletes the row client-side, no body needed
}

func TestPostBookmarkArchiveRedirect(t *testing.T) {
	bm := &fakeBookmarkStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, postBookmarkArchive(rec, routeFormRequest(t, uuid.New(), ""), bm))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/", rec.Header().Get("Location"))
}

func TestPostBookmarkArchiveNotAuthor(t *testing.T) {
	bm := &fakeBookmarkStore{err: core.ErrNotFound}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkArchive(w, r, bm)
	}, routeFormRequest(t, uuid.New(), ""))

	be.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetArchivedPage(t *testing.T) {
	bm := &fakeBookmarkStore{archived: []core.Bookmark{mustBookmark(t, "https://example.com", "Old Post")}}

	rec := httptest.NewRecorder()
	be.NilErr(t, getArchived(rec, mustAuthedRequest(t, http.MethodGet, "/archived", nil), bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "archived", bm.last)
	be.True(t, containsBody(rec, "Archived"))
	be.True(t, containsBody(rec, "Old Post"))
	be.True(t, containsBody(rec, "Restore"))
	be.True(t, containsBody(rec, "Delete"))
}

func TestPostBookmarkRestoreHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{}

	rec := httptest.NewRecorder()
	r := routeFormRequest(t, uuid.New(), "")
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarkRestore(rec, r, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "restore", bm.last)
	be.Equal(t, "", rec.Body.String())
}

func TestPostBookmarkRestoreRedirect(t *testing.T) {
	bm := &fakeBookmarkStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, postBookmarkRestore(rec, routeFormRequest(t, uuid.New(), ""), bm))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/archived", rec.Header().Get("Location"))
}

func TestPostBookmarkRestoreNotAuthor(t *testing.T) {
	bm := &fakeBookmarkStore{err: core.ErrNotFound}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkRestore(w, r, bm)
	}, routeFormRequest(t, uuid.New(), ""))

	be.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostBookmarkDeleteHtmx(t *testing.T) {
	bm := &fakeBookmarkStore{}

	rec := httptest.NewRecorder()
	r := routeFormRequest(t, uuid.New(), "")
	r.Header.Set("HX-Request", "true")
	be.NilErr(t, postBookmarkDelete(rec, r, bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "delete", bm.last)
}

func TestPostBookmarkDeleteNotAuthor(t *testing.T) {
	bm := &fakeBookmarkStore{err: core.ErrNotFound}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postBookmarkDelete(w, r, bm)
	}, routeFormRequest(t, uuid.New(), ""))

	be.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetReadingPage(t *testing.T) {
	bm := &fakeBookmarkStore{queued: []core.Bookmark{
		mustBookmark(t, "https://first.com", "First"),
		mustBookmark(t, "https://second.com", "Second"),
	}}

	rec := httptest.NewRecorder()
	be.NilErr(t, getReading(rec, mustAuthedRequest(t, http.MethodGet, "/reading", nil), bm))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, "queued", bm.last)
	be.True(t, containsBody(rec, "Reading list"))
	be.True(t, containsBody(rec, "Mark as read"))
	first := strings.Index(rec.Body.String(), "First")
	second := strings.Index(rec.Body.String(), "Second")
	be.True(t, first != -1 && first < second) // FIFO: queue order preserved
}

func TestPostBookmarkReading(t *testing.T) {
	t.Run("enqueues when not queued", func(t *testing.T) {
		bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Fresh")}

		rec := httptest.NewRecorder()
		be.NilErr(t, postBookmarkReading(rec, routeFormRequest(t, bm.one.ID, ""), bm))

		be.Equal(t, http.StatusSeeOther, rec.Code)
		be.Equal(t, "enqueue", bm.last)
		be.Equal(t, "/", rec.Header().Get("Location")) // no next → default
	})

	t.Run("dequeues when queued", func(t *testing.T) {
		bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Queued")}
		bm.one.QueuedAt = time.Now()

		rec := httptest.NewRecorder()
		be.NilErr(t, postBookmarkReading(rec, routeFormRequest(t, bm.one.ID, "next=/reading"), bm))

		be.Equal(t, http.StatusSeeOther, rec.Code)
		be.Equal(t, "dequeue", bm.last)
		be.Equal(t, "/reading", rec.Header().Get("Location"))
	})

	t.Run("htmx deletes the row", func(t *testing.T) {
		bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Queued")}
		bm.one.QueuedAt = time.Now()

		rec := httptest.NewRecorder()
		r := routeFormRequest(t, bm.one.ID, "next=/reading")
		r.Header.Set("HX-Request", "true")
		be.NilErr(t, postBookmarkReading(rec, r, bm))

		be.Equal(t, http.StatusOK, rec.Code)
		be.Equal(t, "", rec.Body.String())
	})

	t.Run("plain submit follows next", func(t *testing.T) {
		bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Fresh")}

		rec := httptest.NewRecorder()
		be.NilErr(t, postBookmarkReading(rec, routeFormRequest(t, bm.one.ID, "next=/collections"), bm))

		be.Equal(t, http.StatusSeeOther, rec.Code)
		be.Equal(t, "/collections", rec.Header().Get("Location"))
	})

	t.Run("plain submit rejects off-site next", func(t *testing.T) {
		bm := &fakeBookmarkStore{one: mustBookmark(t, "https://example.com", "Fresh")}

		rec := httptest.NewRecorder()
		be.NilErr(t, postBookmarkReading(rec, routeFormRequest(t, bm.one.ID, "next=//evil.example.com"), bm))

		be.Equal(t, http.StatusSeeOther, rec.Code)
		be.Equal(t, "/", rec.Header().Get("Location"))
	})

	t.Run("not found", func(t *testing.T) {
		bm := &fakeBookmarkStore{err: core.ErrNotFound}

		rec := serve(func(w http.ResponseWriter, r *http.Request) error {
			return postBookmarkReading(w, r, bm)
		}, routeFormRequest(t, uuid.New(), ""))

		be.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestPostBookmarksQueued(t *testing.T) {
	bm := &fakeBookmarkStore{}
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, postBookmarks(rec, mustFormRequest(t, "url=http%3A%2F%2Fexample.com&add_to_reading=true"), col, bm, fakeTitleFetcher{title: "Example"}))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.True(t, bm.created[0].queued)
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
	bk     core.Bookmark
	colIDs []uuid.UUID
	queued bool
}

type fakeBookmarkStore struct {
	err             error
	tags            []core.Tag
	recent          []core.Bookmark
	all             []core.Bookmark
	archived        []core.Bookmark
	queued          []core.Bookmark
	one             core.Bookmark  // single-bookmark target (edit routes)
	dup             *core.Bookmark // FR-11 reminder: author's existing bookmark for the URL
	last            string         // which query method ran last
	gotCollectionID uuid.UUID
	page            core.CursorPage // captured from the last list call
	hasOlder        bool            // pager flags returned by the list calls
	hasNewer        bool
	created         []createdBookmark
}

func (f *fakeBookmarkStore) GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error) {
	return f.tags, f.err
}

func (f *fakeBookmarkStore) GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error) {
	f.last = "recent"
	f.page = page
	return core.BookmarkPage{Items: f.recent, HasOlder: f.hasOlder, HasNewer: f.hasNewer}, f.err
}

func (f *fakeBookmarkStore) GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error) {
	f.last = "all"
	f.page = page
	return core.BookmarkPage{Items: f.all, HasOlder: f.hasOlder, HasNewer: f.hasNewer}, f.err
}

func (f *fakeBookmarkStore) GetBookmarksByCollectionAndTags(ctx context.Context, collectionID uuid.UUID, tags []string, search string, page core.CursorPage) (core.BookmarkPage, error) {
	f.last = "collection+tags"
	f.gotCollectionID = collectionID
	f.page = page
	return core.BookmarkPage{Items: f.all, HasOlder: f.hasOlder, HasNewer: f.hasNewer}, f.err
}

func (f *fakeBookmarkStore) GetBookmarksByCollection(ctx context.Context, collectionID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error) {
	f.last = "collection"
	f.gotCollectionID = collectionID
	f.page = page
	return core.BookmarkPage{Items: f.all, HasOlder: f.hasOlder, HasNewer: f.hasNewer}, f.err
}

func (f *fakeBookmarkStore) GetBookmarksByTags(ctx context.Context, userID uuid.UUID, tags []string, search string, page core.CursorPage) (core.BookmarkPage, error) {
	f.last = "tags"
	f.page = page
	return core.BookmarkPage{Items: f.all, HasOlder: f.hasOlder, HasNewer: f.hasNewer}, f.err
}

func (f *fakeBookmarkStore) CreateBookmark(ctx context.Context, u *url.URL, title string, userID uuid.UUID, notes string, collectionIDs []uuid.UUID, tags []string, queued bool) (core.Bookmark, error) {
	b := core.Bookmark{ID: uuid.New(), URL: u, Title: title, Notes: notes, Author: core.User{ID: userID}, Tags: tags}
	f.created = append(f.created, createdBookmark{bk: b, colIDs: collectionIDs, queued: queued})
	return b, f.err
}

func (f *fakeBookmarkStore) GetBookmarkByID(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	f.last = "byid"
	return f.one, f.err
}

func (f *fakeBookmarkStore) GetBookmarkForCollectionEdit(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	f.last = "byid-collection"
	return f.one, f.err
}

func (f *fakeBookmarkStore) FindUserBookmarkByURL(ctx context.Context, userID uuid.UUID, rawURL string) (core.Bookmark, bool, error) {
	f.last = "find-dup"
	if f.err != nil {
		return core.Bookmark{}, false, f.err
	}
	if f.dup == nil {
		return core.Bookmark{}, false, nil
	}
	return *f.dup, true, nil
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

func (f *fakeBookmarkStore) UpdateBookmarkCollectionIDs(ctx context.Context, bookmarkID, userID uuid.UUID, collectionIDs []uuid.UUID) (core.Bookmark, error) {
	f.last = "update-collections"
	if f.err != nil {
		return core.Bookmark{}, f.err
	}
	f.one.CollectionIDs = collectionIDs
	return f.one, nil
}

func (f *fakeBookmarkStore) ArchiveBookmark(ctx context.Context, id, userID uuid.UUID) error {
	f.last = "archive"
	return f.err
}

func (f *fakeBookmarkStore) GetArchivedBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error) {
	f.last = "archived"
	return f.archived, f.err
}

func (f *fakeBookmarkStore) RestoreBookmark(ctx context.Context, id, userID uuid.UUID) error {
	f.last = "restore"
	return f.err
}

func (f *fakeBookmarkStore) DeleteBookmark(ctx context.Context, id, userID uuid.UUID) error {
	f.last = "delete"
	return f.err
}

func (f *fakeBookmarkStore) GetQueuedBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error) {
	f.last = "queued"
	return f.queued, f.err
}

func (f *fakeBookmarkStore) EnqueueBookmark(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	f.last = "enqueue"
	if f.err != nil {
		return core.Bookmark{}, f.err
	}
	return f.one, nil
}

func (f *fakeBookmarkStore) DequeueBookmark(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	f.last = "dequeue"
	if f.err != nil {
		return core.Bookmark{}, f.err
	}
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
