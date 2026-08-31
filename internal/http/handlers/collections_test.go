package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestGetCollections(t *testing.T) {
	col := &fakeCollectionStore{collections: []core.Collection{{ID: uuid.New(), Name: "Work"}}}

	rec := httptest.NewRecorder()
	getCollections(rec, mustAuthedRequest(t, http.MethodGet, "/collections", nil), col)

	be.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCollectionsStoreError(t *testing.T) {
	col := &fakeCollectionStore{err: errFake}

	rec := httptest.NewRecorder()
	getCollections(rec, mustAuthedRequest(t, http.MethodGet, "/collections", nil), col)

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostCollectionCreatesAndRedirects(t *testing.T) {
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	postCollection(rec, mustFormRequest(t, "name=Work&description=stuff"), col)

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections", rec.Header().Get("Location"))
	be.Equal(t, 1, len(col.created))
	be.Equal(t, testUser.ID, col.created[0].userID)
	be.Equal(t, "Work", col.created[0].name)
	be.Equal(t, "stuff", col.created[0].description)
}

func TestPostCollectionWithoutUser(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/collections", nil)

	postCollection(rec, r, &fakeCollectionStore{})

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetCollectionById(t *testing.T) {
	col := &fakeCollectionStore{got: core.CollectionWithBookmarks{
		Collection: core.Collection{ID: uuid.New(), Name: "Work"},
		Bookmarks:  []core.Bookmark{mustBookmark(t, "https://a.com", "A")},
	}}
	id := uuid.New()

	rec := httptest.NewRecorder()
	getCollectionById(rec, routeRequest(t, http.MethodGet, "/collections/"+id.String(), id.String()), col)

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, 1, col.getCalls)
}

func TestGetCollectionByIdInvalidID(t *testing.T) {
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	getCollectionById(rec, routeRequest(t, http.MethodGet, "/collections/nope", "nope"), col)

	be.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPutCollectionById(t *testing.T) {
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	putCollectionById(rec, routeFormRequest(t, id, "name=Renamed&description=new"), col)

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections/"+id.String(), rec.Header().Get("Location"))
	be.Equal(t, id, col.updatedID)
}

func TestPostCollectionMemberDefaultsRole(t *testing.T) {
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	postCollectionMember(rec, routeFormRequest(t, id, "email=bob%40x.com"), col)

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, 1, len(col.members))
	be.Equal(t, "bob@x.com", col.members[0].email)
	be.Equal(t, "viewer", col.members[0].role) // empty role defaults to viewer
}

func TestDeleteCollectionById(t *testing.T) {
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	deleteCollectionById(rec, routeRequest(t, http.MethodDelete, "/collections/"+id.String(), id.String()), col)

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections", rec.Header().Get("Location"))
	be.AllEqual(t, []uuid.UUID{id}, col.deletedIDs)
}

func TestDeleteCollectionMember(t *testing.T) {
	col := &fakeCollectionStore{}
	colID, userID := uuid.New(), uuid.New()
	target := "/collections/" + colID.String() + "/members/" + userID.String() + "/delete"

	r := mustAuthedRequest(t, http.MethodPost, target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Keys = []string{"collectionId", "userId"}
	rctx.URLParams.Values = []string{colID.String(), userID.String()}
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	deleteCollectionMember(rec, r, col)

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections/"+colID.String(), rec.Header().Get("Location"))
	be.Equal(t, 1, len(col.removed))
	be.Equal(t, colID, col.removed[0].collectionID)
	be.Equal(t, userID, col.removed[0].userID)
}

// --- fakes & helpers ---

type fakeCollectionStore struct {
	err         error
	collections []core.Collection
	got         core.CollectionWithBookmarks
	created     []struct {
		userID      uuid.UUID
		name        string
		description string
	}
	updatedID  uuid.UUID
	deletedIDs []uuid.UUID
	members    []struct {
		collectionID uuid.UUID
		email        string
		role         string
	}
	removed []struct {
		collectionID uuid.UUID
		userID       uuid.UUID
	}
	getCalls int
}

func (f *fakeCollectionStore) GetCollectionsByUser(ctx context.Context, userID uuid.UUID) ([]core.Collection, error) {
	return f.collections, f.err
}

func (f *fakeCollectionStore) CreateCollection(ctx context.Context, userID uuid.UUID, name, description string) error {
	f.created = append(f.created, struct {
		userID      uuid.UUID
		name        string
		description string
	}{userID, name, description})
	return f.err
}

func (f *fakeCollectionStore) GetCollection(ctx context.Context, id uuid.UUID) (core.CollectionWithBookmarks, error) {
	f.getCalls++
	f.got.ID = id
	return f.got, f.err
}

func (f *fakeCollectionStore) UpdateCollection(ctx context.Context, id uuid.UUID, name, description string) (core.Collection, error) {
	f.updatedID = id
	return core.Collection{ID: id, Name: name, Description: description}, f.err
}

func (f *fakeCollectionStore) AddMember(ctx context.Context, collectionID uuid.UUID, email, role string) error {
	f.members = append(f.members, struct {
		collectionID uuid.UUID
		email        string
		role         string
	}{collectionID, email, role})
	return f.err
}

func (f *fakeCollectionStore) RemoveMember(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID) error {
	f.removed = append(f.removed, struct {
		collectionID uuid.UUID
		userID       uuid.UUID
	}{collectionID, userID})
	return f.err
}

func (f *fakeCollectionStore) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	f.deletedIDs = append(f.deletedIDs, id)
	return f.err
}

// routeRequest returns an authed request carrying a chi route param under key "id".
func routeRequest(t *testing.T, method, target, id string) *http.Request {
	t.Helper()
	r := mustAuthedRequest(t, method, target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Keys = []string{"id"}
	rctx.URLParams.Values = []string{id}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// routeFormRequest is a form request carrying the chi "id" route param.
func routeFormRequest(t *testing.T, id uuid.UUID, encoded string) *http.Request {
	t.Helper()
	r := mustFormRequest(t, encoded)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Keys = []string{"id"}
	rctx.URLParams.Values = []string{id.String()}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
