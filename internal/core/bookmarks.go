package core

import (
	"net/url"
	"time"

	"github.com/google/uuid"
)

type (
	Collection struct {
		ID            uuid.UUID
		Name          string
		Description   string
		CreatedAt     time.Time
		UpdatedAt     time.Time
		Members       []CollectionMember
		BookmarkCount int
		Role          CollectionRole
	}

	CollectionMember struct {
		User    User
		Role    CollectionRole
		AddedAt time.Time
	}

	CollectionWithBookmarks struct {
		Collection
		Bookmarks []Bookmark
	}

	Bookmark struct {
		ID            uuid.UUID
		URL           *url.URL
		Title         string
		Notes         string
		CreatedAt     time.Time
		UpdatedAt     time.Time
		ArchivedAt    time.Time
		Author        User
		Tags          []string
		CollectionIDs []uuid.UUID
	}

	Tag struct {
		Name          string
		BookmarkCount int
	}
)
