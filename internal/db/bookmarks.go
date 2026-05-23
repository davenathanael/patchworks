package db

import (
	"context"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
)

func (db *DB) GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error) {
	tags, err := db.querier.GetTagsByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}

	return ToTags(tags), nil
}

func (db *DB) GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error) {
	rows, err := db.querier.GetRecentBookmarksByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}

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

	return ToBookmarks(rows, tagRows), nil
}

func (db *DB) GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error) {
	rows, err := db.querier.GetAllBookmarksByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}

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

	return ToBookmarksFromAllBookmarks(rows, tagRows), nil
}
