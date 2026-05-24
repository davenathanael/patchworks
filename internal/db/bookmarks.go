package db

import (
	"context"
	"net/url"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
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

func (db *DB) CreateBookmark(ctx context.Context, url *url.URL, title string, userID, collectionID uuid.UUID, tags []string) (core.Bookmark, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return core.Bookmark{}, err
	}

	querier := db.querier.WithTx(tx)

	defer func() error {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return nil
	}()

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

	return ToBookmark(createdBookmark, tags, user), nil
}
