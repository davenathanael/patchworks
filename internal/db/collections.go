package db

import (
	"context"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
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
