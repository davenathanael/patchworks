package db

import (
	"net/url"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
	"github.com/google/uuid"
)

// ToUser converts a sqlc.User row to a core.User struct.
func toUser(row sqlc.User) core.User {
	return core.User{
		ID:    row.ID,
		Email: row.Email,
	}
}

// ToSession converts a sqlc.Session row to a core.Session struct.
func toSession(row sqlc.Session) core.Session {
	return core.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt.Time,
	}
}

func toTags(rows []sqlc.GetTagsByUserIdRow) []core.Tag {
	var tags []core.Tag
	for _, row := range rows {
		tags = append(tags, core.Tag{
			Name:          row.Tag,
			BookmarkCount: int(row.BookmarkCount),
		})
	}
	return tags
}

func toBookmarks(rows []sqlc.GetRecentBookmarksByUserIdRow, tagRows []sqlc.GetTagsByBookmarkIdsRow) []core.Bookmark {
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
			Notes:      row.Bookmark.Notes,
			CreatedAt:  row.Bookmark.CreatedAt.Time,
			UpdatedAt:  row.Bookmark.UpdatedAt.Time,
			ArchivedAt: row.Bookmark.ArchivedAt.Time,
			Author:     toUser(row.User),
			Tags:       tagsByBookmark[row.Bookmark.ID],
		}
	}

	return bookmarks
}

func toBookmark(createdBookmark sqlc.Bookmark, tags []string, user core.User) core.Bookmark {
	parsedUrl, _ := url.Parse(createdBookmark.Url) // TODO: handle invalid URL? but DB should not contain any invalid URLs to begin with
	return core.Bookmark{
		ID:         createdBookmark.ID,
		URL:        parsedUrl,
		Title:      createdBookmark.Title,
		Notes:      createdBookmark.Notes,
		CreatedAt:  createdBookmark.CreatedAt.Time,
		UpdatedAt:  createdBookmark.UpdatedAt.Time,
		ArchivedAt: createdBookmark.ArchivedAt.Time,
		Author:     user,
		Tags:       tags,
	}
}

// smaller utilities

func groupMembersByCollectionID(memberRows []sqlc.GetMembersByCollectionIdsRow) map[uuid.UUID][]core.CollectionMember {
	membersByCollection := make(map[uuid.UUID][]core.CollectionMember)
	for _, mr := range memberRows {
		membersByCollection[mr.CollectionMember.CollectionID] = append(
			membersByCollection[mr.CollectionMember.CollectionID],
			core.CollectionMember{
				User:    toUser(mr.User),
				Role:    mr.CollectionMember.Role,
				AddedAt: mr.CollectionMember.AddedAt.Time,
			},
		)
	}
	return membersByCollection
}

func Map[T, U any](items []T, f func(T) U) []U {
	result := make([]U, 0, len(items))
	for _, item := range items {
		result = append(result, f(item))
	}
	return result
}
