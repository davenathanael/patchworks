package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (db *DB) GetCollectionsByUser(ctx context.Context, userID uuid.UUID) ([]core.Collection, error) {
	rows, err := db.querier.ListUserCollections(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return []core.Collection{}, nil
	}

	collectionIDs := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		collectionIDs[i] = row.Collection.ID
	}

	memberRows, err := db.querier.GetMembersByCollectionIds(ctx, collectionIDs)
	if err != nil {
		return nil, err
	}
	membersByCollection := groupMembersByCollectionID(memberRows)

	collections := make([]core.Collection, len(rows))
	for i, row := range rows {
		collections[i] = core.Collection{
			ID:            row.Collection.ID,
			Name:          row.Collection.Name,
			Description:   row.Collection.Description.String,
			CreatedAt:     row.Collection.CreatedAt.Time,
			UpdatedAt:     row.Collection.UpdatedAt.Time,
			BookmarkCount: int(row.BookmarkCount),
			Members:       membersByCollection[row.Collection.ID],
		}
	}

	return collections, nil
}

func (db *DB) CreateCollection(ctx context.Context, userID uuid.UUID, name, description string) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	querier := db.querier.WithTx(tx)

	collectionID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	_, err = querier.CreateCollection(ctx, sqlc.CreateCollectionParams{
		ID:          collectionID,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: len(description) > 0},
	})
	if err != nil {
		return err
	}

	err = querier.AddCollectionMember(ctx, sqlc.AddCollectionMemberParams{
		CollectionID: collectionID,
		UserID:       userID,
		Role:         "owner",
	})
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) GetCollection(ctx context.Context, id uuid.UUID) (core.CollectionWithBookmarks, error) {
	collectionRow, err := db.querier.GetCollectionById(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.CollectionWithBookmarks{}, fmt.Errorf("get collection: %w", core.ErrNotFound)
		}
		return core.CollectionWithBookmarks{}, err
	}

	memberRows, err := db.querier.GetMembersByCollectionIds(ctx, []uuid.UUID{id})
	if err != nil {
		return core.CollectionWithBookmarks{}, err
	}

	bookmarkRows, err := db.querier.GetBookmarksByCollectionId(ctx, id)
	if err != nil {
		return core.CollectionWithBookmarks{}, err
	}

	members := groupMembersByCollectionID(memberRows)[id]

	bookmarksByID := make(map[uuid.UUID]*core.Bookmark)
	var ordered []*core.Bookmark
	for _, row := range bookmarkRows {
		bm, exists := bookmarksByID[row.Bookmark.ID]
		if !exists {
			parsedURL, _ := url.Parse(row.Bookmark.Url)
			bm = &core.Bookmark{
				ID:         row.Bookmark.ID,
				URL:        parsedURL,
				Title:      row.Bookmark.Title,
				Notes:      row.Bookmark.Notes,
				CreatedAt:  row.Bookmark.CreatedAt.Time,
				UpdatedAt:  row.Bookmark.UpdatedAt.Time,
				ArchivedAt: row.Bookmark.ArchivedAt.Time,
				Author:     toUser(row.User),
			}
			bookmarksByID[row.Bookmark.ID] = bm
			ordered = append(ordered, bm)
		}
		if row.Tag.Valid {
			bm.Tags = append(bm.Tags, row.Tag.String)
		}
	}
	bookmarks := make([]core.Bookmark, len(ordered))
	for i, bm := range ordered {
		bookmarks[i] = *bm // dereference after tags are fully collected
	}
	if err := db.attachCollectionIDs(ctx, bookmarks); err != nil {
		return core.CollectionWithBookmarks{}, err
	}

	return core.CollectionWithBookmarks{
		Collection: core.Collection{
			ID:          collectionRow.Collection.ID,
			Name:        collectionRow.Collection.Name,
			Description: collectionRow.Collection.Description.String,
			CreatedAt:   collectionRow.Collection.CreatedAt.Time,
			UpdatedAt:   collectionRow.Collection.UpdatedAt.Time,
			Members:     members,
		},
		Bookmarks: bookmarks,
	}, nil
}

func (db *DB) UpdateCollection(ctx context.Context, id uuid.UUID, name, description string) (core.Collection, error) {
	row, err := db.querier.UpdateCollection(ctx, sqlc.UpdateCollectionParams{
		ID:          id,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: len(description) > 0},
	})
	if err != nil {
		return core.Collection{}, err
	}

	return core.Collection{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (db *DB) AddMember(ctx context.Context, collectionID uuid.UUID, email string, role string) error {
	userRow, err := db.querier.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	err = db.querier.AddCollectionMember(ctx, sqlc.AddCollectionMemberParams{
		CollectionID: collectionID,
		UserID:       userRow.ID,
		Role:         role,
	})
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) RemoveMember(ctx context.Context, collectionID uuid.UUID, userID uuid.UUID) error {
	memberRows, err := db.querier.GetMembersByCollectionIds(ctx, []uuid.UUID{collectionID})
	if err != nil {
		return err
	}

	owners := 0
	for _, row := range memberRows {
		if row.CollectionMember.Role == "owner" {
			owners++
		}
	}

	isOwner := false
	for _, row := range memberRows {
		if row.CollectionMember.UserID == userID && row.CollectionMember.Role == "owner" {
			isOwner = true
			break
		}
	}

	if isOwner && owners <= 1 {
		return fmt.Errorf("cannot remove the last owner")
	}

	err = db.querier.RemoveCollectionMember(ctx, sqlc.RemoveCollectionMemberParams{
		CollectionID: collectionID,
		UserID:       userID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	return db.querier.DeleteCollection(ctx, id)
}
