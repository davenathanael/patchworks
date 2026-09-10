package core

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/carlmjohnson/be"
	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	uid := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tests := []struct {
		name   string
		cursor BookmarkCursor
	}{
		{"zero time", BookmarkCursor{ID: uid}},
		{"whole second", BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), ID: uid}},
		{"nanoseconds", BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 123456789, time.UTC), ID: uid}},
		{"non-utc zone", BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.FixedZone("UTC+5:30", 5*3600+1800)), ID: uid}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DecodeCursor(EncodeCursor(tc.cursor))
			be.True(t, got != nil)
			be.Equal(t, tc.cursor.ID, got.ID)
			be.Equal(t, tc.cursor.CreatedAt.UTC(), got.CreatedAt)
		})
	}
}

func TestDecodeCursorMalformed(t *testing.T) {
	valid := BookmarkCursor{CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), ID: uuid.New()}
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"not base64", "!!!not base64!!!"},
		{"missing id", encodeRaw(valid.CreatedAt.Format(time.RFC3339Nano))},
		{"bad time", encodeRaw("nope|" + valid.ID.String())},
		{"bad uuid", encodeRaw(valid.CreatedAt.Format(time.RFC3339Nano) + "|nope")},
		{"arbitrary payload", "YWJj"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			be.True(t, DecodeCursor(tc.token) == nil)
		})
	}
}

func TestCursorOf(t *testing.T) {
	id := uuid.New()
	b := Bookmark{ID: id, CreatedAt: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)}
	be.Equal(t, BookmarkCursor{CreatedAt: b.CreatedAt, ID: id}, CursorOf(b))
}

// encodeRaw builds a cursor payload with full control over its (possibly
// malformed) contents.
func encodeRaw(raw string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}
