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
)

func (db *DB) GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error) {
	tags, err := db.querier.GetTagsByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toTags(tags), nil
}

func (db *DB) GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID, search string) ([]core.Bookmark, error) {
	rows, err := db.querier.GetRecentBookmarksByUserId(ctx, sqlc.GetRecentBookmarksByUserIdParams{
		AuthorID: userID,
		Search:   search,
	})
	if err != nil {
		return nil, err
	}

	return db.toBookmarksWithTags(ctx, rows)
}

func (db *DB) GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID, search string) ([]core.Bookmark, error) {
	rows, err := db.querier.GetAllBookmarksByUserId(ctx, sqlc.GetAllBookmarksByUserIdParams{
		AuthorID: userID,
		Search:   search,
	})
	if err != nil {
		return nil, err
	}

	recent := Map(rows, func(r sqlc.GetAllBookmarksByUserIdRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	})
	return db.toBookmarksWithTags(ctx, recent)
}

func (db *DB) GetBookmarksByCollectionAndTags(ctx context.Context, collectionID uuid.UUID, tags []string, search string) ([]core.Bookmark, error) {
	rows, err := db.querier.GetBookmarksByCollectionAndTags(ctx, sqlc.GetBookmarksByCollectionAndTagsParams{
		CollectionID: collectionID,
		Tags:         tags,
		Search:       search,
	})
	if err != nil {
		return nil, err
	}

	recent := Map(rows, func(r sqlc.GetBookmarksByCollectionAndTagsRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	})
	return db.toBookmarksWithTags(ctx, recent)
}

func (db *DB) GetBookmarksByCollection(ctx context.Context, collectionID uuid.UUID, search string) ([]core.Bookmark, error) {
	rows, err := db.querier.GetBookmarksByCollection(ctx, sqlc.GetBookmarksByCollectionParams{
		CollectionID: collectionID,
		Search:       search,
	})
	if err != nil {
		return nil, err
	}

	recent := Map(rows, func(r sqlc.GetBookmarksByCollectionRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	})
	return db.toBookmarksWithTags(ctx, recent)
}

func (db *DB) GetBookmarksByTags(ctx context.Context, userID uuid.UUID, tags []string, search string) ([]core.Bookmark, error) {
	rows, err := db.querier.GetBookmarksByTags(ctx, sqlc.GetBookmarksByTagsParams{
		Tags:     tags,
		AuthorID: userID,
		Search:   search,
	})
	if err != nil {
		return nil, err
	}

	recent := Map(rows, func(r sqlc.GetBookmarksByTagsRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	})
	return db.toBookmarksWithTags(ctx, recent)
}

func (db *DB) CreateBookmark(ctx context.Context, url *url.URL, title string, userID, collectionID uuid.UUID, tags []string) (core.Bookmark, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return core.Bookmark{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	querier := db.querier.WithTx(tx)
	bookmarkID := uuid.New()

	createdBookmark, err := querier.CreateBookmark(ctx, sqlc.CreateBookmarkParams{
		ID:       bookmarkID,
		Url:      url.String(),
		Title:    title,
		AuthorID: userID,
	})
	if err != nil {
		return core.Bookmark{}, err
	}

	if collectionID != uuid.Nil {
		_, err = querier.CreateCollectionBookmark(ctx, sqlc.CreateCollectionBookmarkParams{
			BookmarkID:   bookmarkID,
			CollectionID: collectionID,
		})
		if err != nil {
			return core.Bookmark{}, err
		}
	}

	if len(tags) > 0 {
		tagParams := make([]sqlc.CreateBookmarkTagsParams, 0, len(tags))
		for _, tag := range tags {
			tagParams = append(tagParams, sqlc.CreateBookmarkTagsParams{
				BookmarkID: bookmarkID,
				Tag:        tag,
				AuthorID:   userID,
			})
		}
		_, err = querier.CreateBookmarkTags(ctx, tagParams)
		if err != nil {
			return core.Bookmark{}, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return core.Bookmark{}, err
	}

	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		return core.Bookmark{}, err
	}

	return toBookmark(createdBookmark, tags, user), nil
}

func (db *DB) toBookmarksWithTags(ctx context.Context, rows []sqlc.GetRecentBookmarksByUserIdRow) ([]core.Bookmark, error) {
	if len(rows) == 0 {
		return []core.Bookmark{}, nil
	}

	bookmarkIDs := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		bookmarkIDs[i] = row.Bookmark.ID
	}

	tagRows, err := db.querier.GetTagsByBookmarkIds(ctx, bookmarkIDs)
	if err != nil {
		return nil, err
	}

	return toBookmarks(rows, tagRows), nil
}

// GetBookmarkByID returns one of the user's own bookmarks with its tags.
// Author-only: the query matches author_id, so other users' bookmarks are not
// visible (404).
func (db *DB) GetBookmarkByID(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	row, err := db.querier.GetBookmarkById(ctx, sqlc.GetBookmarkByIdParams{ID: id, AuthorID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, fmt.Errorf("get bookmark: %w", core.ErrNotFound)
		}
		return core.Bookmark{}, err
	}
	tags, err := db.tagsForBookmarks(ctx, []uuid.UUID{id})
	if err != nil {
		return core.Bookmark{}, err
	}
	return toBookmark(row.Bookmark, tags[id], toUser(row.User)), nil
}

// UpdateBookmarkNotesTags replaces a bookmark's notes and tags in one
// transaction. Author-only: the UPDATE matches author_id; a mismatch returns
// ErrNotFound. Tags are replaced wholesale (delete + insert), like a fresh save.
func (db *DB) UpdateBookmarkNotesTags(ctx context.Context, id, userID uuid.UUID, notes string, tags []string) (core.Bookmark, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return core.Bookmark{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	querier := db.querier.WithTx(tx)

	if _, err := querier.UpdateBookmarkNotesTags(ctx, sqlc.UpdateBookmarkNotesTagsParams{
		ID:       id,
		AuthorID: userID,
		Notes:    notes,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, fmt.Errorf("update bookmark: %w", core.ErrNotFound)
		}
		return core.Bookmark{}, err
	}

	if err := querier.DeleteBookmarkTags(ctx, sqlc.DeleteBookmarkTagsParams{BookmarkID: id, AuthorID: userID}); err != nil {
		return core.Bookmark{}, err
	}
	if len(tags) > 0 {
		tagParams := make([]sqlc.CreateBookmarkTagsParams, 0, len(tags))
		for _, tag := range tags {
			tagParams = append(tagParams, sqlc.CreateBookmarkTagsParams{BookmarkID: id, Tag: tag, AuthorID: userID})
		}
		if _, err := querier.CreateBookmarkTags(ctx, tagParams); err != nil {
			return core.Bookmark{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Bookmark{}, err
	}

	return db.GetBookmarkByID(ctx, id, userID)
}

// tagsForBookmarks groups tag rows by bookmark id for quick lookup.
func (db *DB) tagsForBookmarks(ctx context.Context, bookmarkIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	tagRows, err := db.querier.GetTagsByBookmarkIds(ctx, bookmarkIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID][]string, len(bookmarkIDs))
	for _, tr := range tagRows {
		out[tr.BookmarkID] = append(out[tr.BookmarkID], tr.Tag)
	}
	return out, nil
}
