package views

import (
	"net/url"
	"time"
)

// LinkItem represents a bookmark link with metadata.
type LinkItem struct {
	ID        string
	Title     string
	URL       *url.URL
	Tags      []string
	CreatedAt time.Time
}

func (l LinkItem) Domain() string {
	return l.URL.Host
}

// CollectionItem represents a collection of links.
type CollectionItem struct {
	ID    string
	Name  string
	Count int
}

// TagItem represents a tag with a count.
type TagItem struct {
	Name  string
	Count int
}

// PaginationProps holds pagination metadata.
type PaginationProps struct {
	CurrentPage int
	TotalPages  int
	BaseURL     string
}
