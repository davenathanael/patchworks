package db

import (
	"net/url"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
	"github.com/google/uuid"
)

// ToUser converts a sqlc.User row to a core.User struct.
func ToUser(row sqlc.User) core.User {
	return core.User{
		ID:    row.ID,
		Email: row.Email,
	}
}

// ToSession converts a sqlc.Session row to a core.Session struct.
func ToSession(row sqlc.Session) core.Session {
	return core.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt.Time,
	}
}

func ToCollections(rows []sqlc.ListUserCollectionsRow) []core.Collection {
	var collections []core.Collection
	for _, row := range rows {
		collections = append(collections, core.Collection{
			ID:            row.Collection.ID,
			Name:          row.Collection.Name,
			Description:   row.Collection.Description.String,
			CreatedAt:     row.Collection.CreatedAt.Time,
			UpdatedAt:     row.Collection.UpdatedAt.Time,
			BookmarkCount: int(row.BookmarkCount),
		})
	}
	return collections
}

func ToTags(rows []sqlc.GetTagsByUserIdRow) []core.Tag {
	var tags []core.Tag
	for _, row := range rows {
		tags = append(tags, core.Tag{
			Name:          row.Tag,
			BookmarkCount: int(row.BookmarkCount),
		})
	}
	return tags
}

func ToBookmarks(rows []sqlc.GetRecentBookmarksByUserIdRow, tagRows []sqlc.GetTagsByBookmarkIdsRow) []core.Bookmark {
	tagsByBookmark := make(map[uuid.UUID][]string)
	for _, tr := range tagRows {
		tagsByBookmark[tr.BookmarkID] = append(tagsByBookmark[tr.BookmarkID], tr.Tag)
	}

	bookmarks := make([]core.Bookmark, len(rows))
	for i, row := range rows {
		parsedUrl, _ := url.Parse(row.Bookmark.Url) // TODO: handle invalid URL? but DB should not contain any invalid URLs to begin with
		bookmarks[i] = core.Bookmark{
			ID:         row.Bookmark.ID,
			URL:        parsedUrl,
			Title:      row.Bookmark.Title,
			CreatedAt:  row.Bookmark.CreatedAt.Time,
			UpdatedAt:  row.Bookmark.UpdatedAt.Time,
			ArchivedAt: row.Bookmark.ArchivedAt.Time,
			Author:     ToUser(row.User),
			Tags:       tagsByBookmark[row.Bookmark.ID],
		}
	}

	return bookmarks
}

func ToBookmarksFromAllBookmarks(rows []sqlc.GetAllBookmarksByUserIdRow, tagRows []sqlc.GetTagsByBookmarkIdsRow) []core.Bookmark {
	rowsAsRecentBookmarks := make([]sqlc.GetRecentBookmarksByUserIdRow, 0, len(rows))
	for _, row := range rows {
		rowsAsRecentBookmarks = append(rowsAsRecentBookmarks, sqlc.GetRecentBookmarksByUserIdRow(row))
	}

	return ToBookmarks(rowsAsRecentBookmarks, tagRows)
}

// smaller utilities

func groupMembersByCollectionID(memberRows []sqlc.GetMembersByCollectionIdsRow) map[uuid.UUID][]core.CollectionMember {
	membersByCollection := make(map[uuid.UUID][]core.CollectionMember)
	for _, mr := range memberRows {
		membersByCollection[mr.CollectionMember.CollectionID] = append(
			membersByCollection[mr.CollectionMember.CollectionID],
			core.CollectionMember{
				User:    ToUser(mr.User),
				Role:    mr.CollectionMember.Role,
				AddedAt: mr.CollectionMember.AddedAt.Time,
			},
		)
	}
	return membersByCollection
}
