package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ajg/form"
	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/davenathanael/patchwork/internal/http/views"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// CollectionStore is the interface for collection persistence and membership.
type CollectionStore interface {
	GetCollectionsByUser(ctx context.Context, userID uuid.UUID) ([]core.Collection, error)
	CreateCollection(ctx context.Context, userID uuid.UUID, name, description string) error
	GetCollection(ctx context.Context, id uuid.UUID) (core.CollectionWithBookmarks, error)
	UpdateCollection(ctx context.Context, id uuid.UUID, name, description string) (core.Collection, error)
	AddMember(ctx context.Context, collectionID uuid.UUID, email string, role string) error
	RemoveMember(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID) error
	DeleteCollection(ctx context.Context, id uuid.UUID) error
}

func handleGetCollections(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getCollections(w, r, comp.DB)
	}
}

func getCollections(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	collectionsList, err := collections.GetCollectionsByUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get collections: %w", err)
	}

	if err := views.ListCollectionsPage(collectionsList, user).Render(w); err != nil {
		return fmt.Errorf("render collections page: %w", err)
	}
	return nil
}

func handleGetCollectionCreation(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getCollectionCreation(w, r)
	}
}

func getCollectionCreation(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	return renderCreateCollectionPage(w, views.CollectionForm{}, user)
}

func handlePostCollection(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postCollection(w, r, comp.DB)
	}
}

// renderCreateCollectionPage re-renders the create-collection page with the
// form's values and errors at 200.
func renderCreateCollectionPage(w http.ResponseWriter, f views.CollectionForm, user core.User) error {
	if err := views.CreateCollectionsPage(user, f).Render(w); err != nil {
		return fmt.Errorf("render create collection page: %w", err)
	}
	return nil
}

func postCollection(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	var f views.CollectionForm
	if err := form.NewDecoder(r.Body).Decode(&f); err != nil {
		f.Errors = views.FormErrors{"form": "invalid form data"}
		return renderCreateCollectionPage(w, f, user)
	}

	if errs := validateCollection(f); len(errs) > 0 {
		f.Errors = errs
		return renderCreateCollectionPage(w, f, user)
	}

	if err := collections.CreateCollection(ctx, user.ID, f.Name, f.Description); err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	http.Redirect(w, r, "/collections", http.StatusSeeOther)
	return nil
}

func handleGetCollectionById(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getCollectionById(w, r, comp.DB)
	}
}

func getCollectionById(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse collection id %q: %w", rawID, core.ErrNotFound)
	}

	collection, err := collections.GetCollection(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("get collection: %w", err)
	}

	allCollections, err := collections.GetCollectionsByUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get collections: %w", err)
	}

	if err := views.CollectionPage(collection.Collection, collection.Bookmarks, user, allCollections).Render(w); err != nil {
		return fmt.Errorf("render collection page: %w", err)
	}
	return nil
}

func handleGetCollectionEdit(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return getCollectionEdit(w, r, comp.DB)
	}
}

func getCollectionEdit(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse collection id %q: %w", rawID, core.ErrNotFound)
	}

	collection, err := collections.GetCollection(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return err
		}
		return fmt.Errorf("get collection: %w", err)
	}

	f := views.CollectionForm{
		Name:        collection.Name,
		Description: collection.Description,
	}
	return renderEditCollectionPage(w, f, user, collection.ID)
}

func handlePutCollectionById(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return putCollectionById(w, r, comp.DB)
	}
}

// renderEditCollectionPage re-renders the edit-collection page with the
// form's values and errors at 200.
func renderEditCollectionPage(w http.ResponseWriter, f views.CollectionForm, user core.User, id uuid.UUID) error {
	if err := views.EditCollectionPage(user, f, id).Render(w); err != nil {
		return fmt.Errorf("render edit collection page: %w", err)
	}
	return nil
}

func putCollectionById(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse collection id %q: %w", rawID, core.ErrNotFound)
	}

	var f views.CollectionForm
	if err := form.NewDecoder(r.Body).Decode(&f); err != nil {
		f.Errors = views.FormErrors{"form": "invalid form data"}
		return renderEditCollectionPage(w, f, user, id)
	}

	if errs := validateCollection(f); len(errs) > 0 {
		f.Errors = errs
		return renderEditCollectionPage(w, f, user, id)
	}

	if _, err := collections.UpdateCollection(ctx, id, f.Name, f.Description); err != nil {
		return fmt.Errorf("update collection: %w", err)
	}

	http.Redirect(w, r, "/collections/"+id.String(), http.StatusSeeOther) // #nosec G710 -- id is a validated UUID, no open redirect
	return nil
}

func handlePostCollectionMember(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return postCollectionMember(w, r, comp.DB)
	}
}

type addMemberForm struct {
	Email string `form:"email"`
	Role  string `form:"role"`
}

func postCollectionMember(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	if _, ok := middleware.UserFromContext(ctx); !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	collectionID, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse collection id %q: %w", rawID, core.ErrNotFound)
	}

	var formData addMemberForm
	if err := form.NewDecoder(r.Body).Decode(&formData); err != nil {
		return fmt.Errorf("decode member form: %w", err)
	}

	if formData.Role == "" {
		formData.Role = "viewer"
	}

	if err := collections.AddMember(ctx, collectionID, formData.Email, formData.Role); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	http.Redirect(w, r, "/collections/"+collectionID.String(), http.StatusSeeOther) // #nosec G710 -- a validated UUID, no open redirect
	return nil
}

func handleDeleteCollectionMember(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return deleteCollectionMember(w, r, comp.DB)
	}
}

func deleteCollectionMember(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	if _, ok := middleware.UserFromContext(ctx); !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "collectionId")
	collectionID, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse collection id %q: %w", rawID, core.ErrNotFound)
	}

	rawMemberID := chi.URLParam(r, "userId")
	memberID, err := uuid.Parse(rawMemberID)
	if err != nil {
		return fmt.Errorf("parse user id %q: %w", rawMemberID, core.ErrNotFound)
	}

	if err := collections.RemoveMember(ctx, collectionID, memberID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	http.Redirect(w, r, "/collections/"+collectionID.String(), http.StatusSeeOther) // #nosec G710 -- a validated UUID, no open redirect
	return nil
}

func handleDeleteCollectionById(comp *components.Components) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return deleteCollectionById(w, r, comp.DB)
	}
}

func deleteCollectionById(w http.ResponseWriter, r *http.Request, collections CollectionStore) error {
	ctx := r.Context()
	if _, ok := middleware.UserFromContext(ctx); !ok {
		return fmt.Errorf("user not found in context")
	}

	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("parse collection id %q: %w", rawID, core.ErrNotFound)
	}

	if err := collections.DeleteCollection(ctx, id); err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}

	http.Redirect(w, r, "/collections", http.StatusSeeOther)
	return nil
}
