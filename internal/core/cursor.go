package core

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BookmarkCursor pins a position in a keyset-ordered bookmark list. Ids are
// uuid v4 with no time information, so created_at alone is not a stable
// boundary — the pair is.
type BookmarkCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// CursorPage requests one batch of a keyset-paginated list: Older selects the
// batch strictly before its cursor (nil for the first page), Limit is the
// page size.
type CursorPage struct {
	Older *BookmarkCursor
	Limit int
}

// BookmarkPage is one visible batch plus the pager flags: HasOlder says
// whether items exist past the last visible item; HasNewer marks a mid-feed
// window (an older cursor was supplied, so newer items exist but are already
// held client-side).
type BookmarkPage struct {
	Items    []Bookmark
	HasOlder bool
	HasNewer bool
}

// CursorOf extracts the cursor marking b's position in a created_at DESC list.
func CursorOf(b Bookmark) BookmarkCursor {
	return BookmarkCursor{CreatedAt: b.CreatedAt, ID: b.ID}
}

// EncodeCursor packs a cursor into one opaque base64url token.
func EncodeCursor(c BookmarkCursor) string {
	raw := c.CreatedAt.Format(time.RFC3339Nano) + "|" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor reverses EncodeCursor. Any malformed input decodes to nil so a
// stale or hand-edited link degrades to a fresh view instead of an error.
func DecodeCursor(token string) *BookmarkCursor {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil
	}
	createdAt, id, ok := strings.Cut(string(raw), "|")
	if !ok {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil
	}
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return nil
	}
	// created_at is a timestamp column (no zone): normalize so the value sent
	// back to Postgres is the same wall clock the row was read with.
	return &BookmarkCursor{CreatedAt: parsed.UTC(), ID: idUUID}
}
