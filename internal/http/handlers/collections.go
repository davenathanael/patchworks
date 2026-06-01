package handlers

import (
	"context"
	"net/http"

	"github.com/ajg/form"
	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/davenathanael/patchwork/internal/http/views"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func handleGetCollections(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getCollections(w, r, comp.DB)
	}
}

type ()

func getCollections(w http.ResponseWriter, r *http.Request, collectionGetter collectionGetter) {
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

	err = views.ListCollectionsPage(collections, user).Render(w)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func handleGetCollectionCreation(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getCollectionCreation(w, r, comp.DB)
	}
}

type ()

func getCollectionCreation(w http.ResponseWriter, r *http.Request, collectionGetter collectionGetter) {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	err := views.CreateCollectionsPage(user).Render(w)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func handlePostCollection(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		postCollection(w, r, comp.DB)
	}
}

type (
	collectionSaver interface {
		CreateCollection(ctx context.Context, userID uuid.UUID, name, description string) error
	}

	newCollectionForm struct {
		Name        string `form:"name"`
		Description string `form:"description"`
	}
)

func postCollection(w http.ResponseWriter, r *http.Request, collectionSaver collectionSaver) {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	var formData newCollectionForm
	if err := form.NewDecoder(r.Body).Decode(&formData); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	err := collectionSaver.CreateCollection(ctx, user.ID, formData.Name, formData.Description)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/collections", http.StatusSeeOther)
}

func handleGetCollectionById(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getCollectionById(w, r, comp.DB)
	}
}

type collectionWithBookmarksGetter interface {
	GetCollection(ctx context.Context, id uuid.UUID) (core.CollectionWithBookmarks, error)
}

func getCollectionById(w http.ResponseWriter, r *http.Request, getter collectionWithBookmarksGetter) {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid collection id", 400)
		return
	}

	collection, err := getter.GetCollection(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = views.CollectionPage(collection.Collection, collection.Bookmarks, user).Render(w)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func handleGetCollectionEdit(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getCollectionEdit(w, r, comp.DB)
	}
}

func getCollectionEdit(w http.ResponseWriter, r *http.Request, getter collectionWithBookmarksGetter) {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid collection id", 400)
		return
	}

	collection, err := getter.GetCollection(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = views.EditCollectionPage(collection.Collection, user).Render(w)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

type collectionUpdater interface {
	UpdateCollection(ctx context.Context, id uuid.UUID, name, description string) (core.Collection, error)
}

func handlePutCollectionById(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		putCollectionById(w, r, comp.DB)
	}
}

type updateCollectionForm struct {
	Name        string `form:"name"`
	Description string `form:"description"`
}

func putCollectionById(w http.ResponseWriter, r *http.Request, updater collectionUpdater) {
	ctx := r.Context()
	if _, ok := middleware.UserFromContext(ctx); !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid collection id", 400)
		return
	}

	var formData updateCollectionForm
	if err := form.NewDecoder(r.Body).Decode(&formData); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	_, err = updater.UpdateCollection(ctx, id, formData.Name, formData.Description)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/collections/"+id.String(), http.StatusSeeOther)
}

type collectionMemberAdder interface {
	AddMember(ctx context.Context, collectionID uuid.UUID, email string, role string) error
}

func handlePostCollectionMember(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		postCollectionMember(w, r, comp.DB)
	}
}

type addMemberForm struct {
	Email string `form:"email"`
	Role  string `form:"role"`
}

func postCollectionMember(w http.ResponseWriter, r *http.Request, adder collectionMemberAdder) {
	ctx := r.Context()
	if _, ok := middleware.UserFromContext(ctx); !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	collectionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid collection id", 400)
		return
	}

	var formData addMemberForm
	if err := form.NewDecoder(r.Body).Decode(&formData); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if formData.Role == "" {
		formData.Role = "viewer"
	}

	err = adder.AddMember(ctx, collectionID, formData.Email, formData.Role)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/collections/"+collectionID.String(), http.StatusSeeOther)
}

type collectionMemberRemover interface {
	RemoveMember(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID) error
}

func handleDeleteCollectionMember(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deleteCollectionMember(w, r, comp.DB)
	}
}

func deleteCollectionMember(w http.ResponseWriter, r *http.Request, remover collectionMemberRemover) {
	ctx := r.Context()
	if _, ok := middleware.UserFromContext(ctx); !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	collectionID, err := uuid.Parse(chi.URLParam(r, "collectionId"))
	if err != nil {
		http.Error(w, "invalid collection id", 400)
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, "invalid user id", 400)
		return
	}

	err = remover.RemoveMember(ctx, collectionID, memberID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/collections/"+collectionID.String(), http.StatusSeeOther)
}

type collectionDeleter interface {
	DeleteCollection(ctx context.Context, id uuid.UUID) error
}

func handleDeleteCollectionById(comp *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deleteCollectionById(w, r, comp.DB)
	}
}

func deleteCollectionById(w http.ResponseWriter, r *http.Request, deleter collectionDeleter) {
	ctx := r.Context()
	if _, ok := middleware.UserFromContext(ctx); !ok {
		http.Error(w, "user not found in context", 500)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid collection id", 400)
		return
	}

	err = deleter.DeleteCollection(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/collections", http.StatusSeeOther)
}
