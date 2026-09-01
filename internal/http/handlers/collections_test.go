package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/views"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestGetCollections(t *testing.T) {
	col := &fakeCollectionStore{collections: []core.Collection{{ID: uuid.New(), Name: "Work"}}}

	rec := httptest.NewRecorder()
	be.NilErr(t, getCollections(rec, mustAuthedRequest(t, http.MethodGet, "/collections", nil), col))

	be.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCollectionsStoreError(t *testing.T) {
	col := &fakeCollectionStore{err: errFake}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return getCollections(w, r, col)
	}, mustAuthedRequest(t, http.MethodGet, "/collections", nil))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostCollectionCreatesAndRedirects(t *testing.T) {
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, postCollection(rec, mustFormRequest(t, "name=Work&description=stuff"), col))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections", rec.Header().Get("Location"))
	be.Equal(t, 1, len(col.created))
	be.Equal(t, testUser.ID, col.created[0].userID)
	be.Equal(t, "Work", col.created[0].name)
	be.Equal(t, "stuff", col.created[0].description)
}

func TestPostCollectionWithoutUser(t *testing.T) {
	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return postCollection(w, r, &fakeCollectionStore{})
	}, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/collections", nil))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetCollectionById(t *testing.T) {
	col := &fakeCollectionStore{got: core.CollectionWithBookmarks{
		Collection: core.Collection{ID: uuid.New(), Name: "Work"},
		Bookmarks:  []core.Bookmark{mustBookmark(t, "https://a.com", "A")},
	}}
	id := uuid.New()

	rec := httptest.NewRecorder()
	be.NilErr(t, getCollectionById(rec, routeRequest(t, http.MethodGet, "/collections/"+id.String(), id.String()), col))

	be.Equal(t, http.StatusOK, rec.Code)
	be.Equal(t, 1, col.getCalls)
}

func TestGetCollectionByIdInvalidID(t *testing.T) {
	col := &fakeCollectionStore{}

	rec := serve(func(w http.ResponseWriter, r *http.Request) error {
		return getCollectionById(w, r, col)
	}, routeRequest(t, http.MethodGet, "/collections/nope", "nope"))

	be.Equal(t, http.StatusNotFound, rec.Code) // invalid id classifies as ErrNotFound -> 404 page
}

func TestPutCollectionById(t *testing.T) {
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	be.NilErr(t, putCollectionById(rec, routeFormRequest(t, id, "name=Renamed&description=new"), col))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections/"+id.String(), rec.Header().Get("Location"))
	be.Equal(t, id, col.updatedID)
}

func TestPostCollectionMemberDefaultsRole(t *testing.T) {
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	be.NilErr(t, postCollectionMember(rec, routeFormRequest(t, id, "email=bob%40x.com"), col))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, 1, len(col.members))
	be.Equal(t, "bob@x.com", col.members[0].email)
	be.Equal(t, "viewer", col.members[0].role) // empty role defaults to viewer
}

func TestDeleteCollectionById(t *testing.T) {
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	be.NilErr(t, deleteCollectionById(rec, routeRequest(t, http.MethodDelete, "/collections/"+id.String(), id.String()), col))

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
	be.NilErr(t, deleteCollectionMember(rec, r, col))

	be.Equal(t, http.StatusSeeOther, rec.Code)
	be.Equal(t, "/collections/"+colID.String(), rec.Header().Get("Location"))
	be.Equal(t, 1, len(col.removed))
	be.Equal(t, colID, col.removed[0].collectionID)
	be.Equal(t, userID, col.removed[0].userID)
}

func TestPostCollectionMissingNameRerenders(t *testing.T) {
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, postCollection(rec, mustFormRequest(t, "description=stuff"), col))

	be.Equal(t, http.StatusOK, rec.Code) // re-render create page with inline error
	be.True(t, containsBody(rec, "name is required"))
	be.True(t, containsBody(rec, "stuff")) // submitted description preserved
	be.Equal(t, 0, len(col.created))       // nothing created
}

func TestPostCollectionInvalidFormData(t *testing.T) {
	col := &fakeCollectionStore{}

	rec := httptest.NewRecorder()
	be.NilErr(t, postCollection(rec, mustFormRequest(t, "name=%zz"), col))

	be.Equal(t, http.StatusOK, rec.Code) // re-render create page with top-level alert
	be.True(t, containsBody(rec, "invalid form data"))
	be.Equal(t, 0, len(col.created))
}

func TestPutCollectionByIdMissingNameRerenders(t *testing.T) {
	col := &fakeCollectionStore{}
	id := uuid.New()

	rec := httptest.NewRecorder()
	be.NilErr(t, putCollectionById(rec, routeFormRequest(t, id, "name=&description=new"), col))

	be.Equal(t, http.StatusOK, rec.Code) // re-render edit page with inline error
	be.True(t, containsBody(rec, "name is required"))
	be.Equal(t, uuid.Nil, col.updatedID) // nothing updated
}

func TestValidateCollection(t *testing.T) {
	tests := []struct {
		name  string
		form  views.CollectionForm
		field string // field expected to carry an error; "" = must validate clean
	}{
		{"valid form has no errors", views.CollectionForm{Name: "Work", Description: "stuff"}, ""},
		{"description is optional", views.CollectionForm{Name: "Work"}, ""},
		{"missing name", views.CollectionForm{Description: "stuff"}, "name"},
		{"missing both", views.CollectionForm{}, "name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateCollection(tt.form)
			if tt.field == "" {
				be.Equal(t, 0, len(errs))
				return
			}
			be.True(t, errs[tt.field] != "")
		})
	}
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
