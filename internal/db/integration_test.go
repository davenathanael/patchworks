//go:build integration

package db

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration tests run against a real Postgres. They are compile-gated by the
// "integration" build tag (skipped by default) and self-contained: they drop
// and recreate a dedicated patchwork_test database and apply migrations.
//
// Run:	mise run test-integration   (requires `mise run compose` first)
// Env:	TEST_DATABASE_URL (falls back to DATABASE_URL)

const testDBName = "patchwork_test"

var testDB *DB

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		adminURL = os.Getenv("DATABASE_URL")
	}
	if adminURL == "" {
		fmt.Println("integration: set TEST_DATABASE_URL or DATABASE_URL and start Postgres to run")
		return 0
	}

	ctx := context.Background()
	testURL, err := createTestDatabase(ctx, adminURL)
	if err != nil {
		fmt.Println("integration setup:", err)
		return 1
	}
	defer dropTestDatabase(ctx, adminURL)

	testDB, err = New(ctx, testURL)
	if err != nil {
		fmt.Println("integration setup:", err)
		return 1
	}
	defer testDB.Close()

	return m.Run()
}

func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	email := "user@test.local"

	user, err := testDB.CreateUser(ctx, email, "hash-1")
	be.NilErr(t, err)
	be.Equal(t, email, user.Email)

	got, hash, found, err := testDB.GetUserByEmail(ctx, email)
	be.NilErr(t, err)
	be.True(t, found)
	be.Equal(t, user.ID, got.ID)
	be.Equal(t, "hash-1", hash)

	byID, err := testDB.GetUserByID(ctx, user.ID)
	be.NilErr(t, err)
	be.Equal(t, email, byID.Email)

	// A successful login stamps last_login_at (AU-7); core.User carries no
	// field for it — the write is the feature.
	be.NilErr(t, testDB.TouchLastLogin(ctx, user.ID))
	var stamped bool
	be.NilErr(t, testDB.Pool.QueryRow(ctx, `SELECT last_login_at IS NOT NULL FROM users WHERE id = $1`, user.ID).Scan(&stamped))
	be.True(t, stamped)

	// Unique email constraint rejects duplicates.
	_, err = testDB.CreateUser(ctx, email, "hash-2")
	be.Nonzero(t, err)
}

func TestSessionRepository(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "session@test.local", "hash")
	be.NilErr(t, err)

	id := uuid.New()
	expires := time.Now().Add(time.Hour)
	sess, err := testDB.CreateSession(ctx, id, user.ID, expires)
	be.NilErr(t, err)
	be.Equal(t, user.ID, sess.UserID)
	be.False(t, sess.ExpiresAt.Before(expires.Add(-time.Minute))) // timestamp rounding tolerance

	got, found, err := testDB.GetSessionByID(ctx, id)
	be.NilErr(t, err)
	be.True(t, found)
	be.Equal(t, id, got.ID)

	_, found, err = testDB.GetSessionByID(ctx, uuid.New())
	be.NilErr(t, err)
	be.False(t, found)

	be.NilErr(t, testDB.DeleteSession(ctx, id))
	_, found, err = testDB.GetSessionByID(ctx, id)
	be.NilErr(t, err)
	be.False(t, found)
}

func TestCollectionRepository(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "collection@test.local", "hash")
	be.NilErr(t, err)

	be.NilErr(t, testDB.CreateCollection(ctx, user.ID, "Work", "stuff"))

	lists, err := testDB.GetCollectionsByUser(ctx, user.ID)
	be.NilErr(t, err)
	be.Equal(t, 1, len(lists))
	be.Equal(t, "Work", lists[0].Name)
	colID := lists[0].ID

	upd, err := testDB.UpdateCollection(ctx, colID, "Work2", "new")
	be.NilErr(t, err)
	be.Equal(t, "Work2", upd.Name)

	full, err := testDB.GetCollection(ctx, colID)
	be.NilErr(t, err)
	be.Equal(t, "Work2", full.Name)
	bookmarks, err := testDB.GetCollectionBookmarks(ctx, colID, core.CursorPage{Limit: 20})
	be.NilErr(t, err)
	be.Equal(t, 0, len(bookmarks.Items))

	be.NilErr(t, testDB.DeleteCollection(ctx, colID))
	left, err := testDB.GetCollectionsByUser(ctx, user.ID)
	be.NilErr(t, err)
	be.Equal(t, 0, len(left))
}

// TestCollectionBookmarksIncludeUntagged guards the collection-detail data
// path: bookmarks without tags must appear in GetCollection (LEFT JOIN), with
// tags only on tagged bookmarks — mirroring the dashboard list behavior.
func TestCollectionBookmarksIncludeUntagged(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "colbookmark@test.local", "hash")
	be.NilErr(t, err)

	be.NilErr(t, testDB.CreateCollection(ctx, user.ID, "Reading", ""))
	lists, err := testDB.GetCollectionsByUser(ctx, user.ID)
	be.NilErr(t, err)
	colID := lists[0].ID

	u1, err := url.Parse("https://example.com/tagged")
	be.NilErr(t, err)
	_, err = testDB.CreateBookmark(ctx, u1, "Tagged Post", user.ID, "", []uuid.UUID{colID}, []string{"go", "web"}, false)
	be.NilErr(t, err)

	u2, err := url.Parse("https://example.com/plain")
	be.NilErr(t, err)
	_, err = testDB.CreateBookmark(ctx, u2, "Plain Post", user.ID, "", []uuid.UUID{colID}, nil, false)
	be.NilErr(t, err)

	full, err := testDB.GetCollectionBookmarks(ctx, colID, core.CursorPage{Limit: 20})
	be.NilErr(t, err)
	be.Equal(t, 2, len(full.Items))

	var taggedTags, plainTags []string
	found := make(map[string]bool)
	for i := range full.Items {
		found[full.Items[i].Title] = true
		if full.Items[i].Title == "Tagged Post" {
			taggedTags = full.Items[i].Tags
		}
		if full.Items[i].Title == "Plain Post" {
			plainTags = full.Items[i].Tags
		}
	}
	be.True(t, found["Tagged Post"])
	be.True(t, found["Plain Post"])
	be.Equal(t, 2, len(taggedTags))
	be.True(t, slices.Contains(taggedTags, "go"))
	be.True(t, slices.Contains(taggedTags, "web"))
	be.Equal(t, 0, len(plainTags))
}

func TestUpdateBookmarkNotesTags(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "edit@test.local", "hash")
	be.NilErr(t, err)
	other, err := testDB.CreateUser(ctx, "other@test.local", "hash")
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/post")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Example Post", user.ID, "", nil, []string{"go", "web"}, false)
	be.NilErr(t, err)

	updated, err := testDB.UpdateBookmarkNotesTags(ctx, bk.ID, user.ID, "a note", []string{"css"})
	be.NilErr(t, err)
	be.Equal(t, "a note", updated.Notes)
	be.AllEqual(t, []string{"css"}, updated.Tags)

	got, err := testDB.GetBookmarkByID(ctx, bk.ID, user.ID)
	be.NilErr(t, err)
	be.Equal(t, "a note", got.Notes)
	be.AllEqual(t, []string{"css"}, got.Tags)

	// author-only: another user can neither read nor update it
	_, err = testDB.GetBookmarkByID(ctx, bk.ID, other.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
	_, err = testDB.UpdateBookmarkNotesTags(ctx, bk.ID, other.ID, "hijack", nil)
	be.True(t, errors.Is(err, core.ErrNotFound))
}

func TestUpdateBookmarkCollectionIDs(t *testing.T) {
	ctx := context.Background()
	owner, err := testDB.CreateUser(ctx, "coledit@test.local", "hash")
	be.NilErr(t, err)
	other, err := testDB.CreateUser(ctx, "colshared@test.local", "hash")
	be.NilErr(t, err)

	be.NilErr(t, testDB.CreateCollection(ctx, owner.ID, "Mine A", ""))
	be.NilErr(t, testDB.CreateCollection(ctx, owner.ID, "Mine B", ""))
	be.NilErr(t, testDB.CreateCollection(ctx, other.ID, "Shared", ""))
	own, err := testDB.GetCollectionsByUser(ctx, owner.ID)
	be.NilErr(t, err)
	be.Equal(t, 2, len(own))
	shared, err := testDB.GetCollectionsByUser(ctx, other.ID)
	be.NilErr(t, err)
	be.Equal(t, 1, len(shared))

	u, err := url.Parse("https://example.com/post")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Example Post", owner.ID, "", []uuid.UUID{own[0].ID}, nil, false)
	be.NilErr(t, err)
	// other adds the bookmark to their (shared) collection directly
	_, err = testDB.Pool.Exec(ctx, `insert into collection_bookmarks (collection_id, bookmark_id) values ($1, $2)`, shared[0].ID, bk.ID)
	be.NilErr(t, err)

	updated, err := testDB.UpdateBookmarkCollectionIDs(ctx, bk.ID, owner.ID, []uuid.UUID{own[0].ID, own[1].ID})
	be.NilErr(t, err)
	// all memberships surface, including the shared one (only own are editable)
	be.Equal(t, 3, len(updated.CollectionIDs))
	be.True(t, slices.Contains(updated.CollectionIDs, own[0].ID))
	be.True(t, slices.Contains(updated.CollectionIDs, own[1].ID))
	be.True(t, slices.Contains(updated.CollectionIDs, shared[0].ID))

	// the shared collection's link is untouched (owner is not a member there)
	got, err := testDB.GetCollectionBookmarks(ctx, shared[0].ID, core.CursorPage{Limit: 20})
	be.NilErr(t, err)
	be.Equal(t, 1, len(got.Items))
	be.Equal(t, bk.ID, got.Items[0].ID)

	// unchecking own[0] drops only that link; the shared one stays
	updated, err = testDB.UpdateBookmarkCollectionIDs(ctx, bk.ID, owner.ID, []uuid.UUID{own[1].ID})
	be.NilErr(t, err)
	be.Equal(t, 2, len(updated.CollectionIDs))
	be.True(t, slices.Contains(updated.CollectionIDs, own[1].ID))
	be.True(t, slices.Contains(updated.CollectionIDs, shared[0].ID))

	// a member with manage rights over a containing collection may also edit:
	// other owns Shared, so they can manage the bookmark's link to it, while
	// the author's own links stay untouched
	updated, err = testDB.UpdateBookmarkCollectionIDs(ctx, bk.ID, other.ID, []uuid.UUID{shared[0].ID})
	be.NilErr(t, err)
	be.Equal(t, 2, len(updated.CollectionIDs))
	be.True(t, slices.Contains(updated.CollectionIDs, own[1].ID))
	be.True(t, slices.Contains(updated.CollectionIDs, shared[0].ID))
}

func TestBookmarkCollectionEditAccess(t *testing.T) {
	ctx := context.Background()
	author, err := testDB.CreateUser(ctx, "bmauthor@test.local", "hash")
	be.NilErr(t, err)
	editor, err := testDB.CreateUser(ctx, "bmeditor@test.local", "hash")
	be.NilErr(t, err)
	viewer, err := testDB.CreateUser(ctx, "bmviewer@test.local", "hash")
	be.NilErr(t, err)
	stranger, err := testDB.CreateUser(ctx, "bmstranger@test.local", "hash")
	be.NilErr(t, err)

	be.NilErr(t, testDB.CreateCollection(ctx, author.ID, "Authored", ""))
	colls, err := testDB.GetCollectionsByUser(ctx, author.ID)
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/shared")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Shared Post", author.ID, "", []uuid.UUID{colls[0].ID}, nil, false)
	be.NilErr(t, err)

	// author reads their own bookmark through the collection-edit fetch
	got, err := testDB.GetBookmarkForCollectionEdit(ctx, bk.ID, author.ID)
	be.NilErr(t, err)
	be.Equal(t, bk.ID, got.ID)

	// editor member reads + edits (drops the collection link)
	be.NilErr(t, testDB.AddMember(ctx, colls[0].ID, "bmeditor@test.local", "editor"))
	got, err = testDB.GetBookmarkForCollectionEdit(ctx, bk.ID, editor.ID)
	be.NilErr(t, err)
	be.Equal(t, bk.ID, got.ID)

	// viewer member and stranger cannot read or edit (before the editor's
	// destructive update, so membership is still the reason)
	be.NilErr(t, testDB.AddMember(ctx, colls[0].ID, "bmviewer@test.local", "viewer"))
	_, err = testDB.GetBookmarkForCollectionEdit(ctx, bk.ID, viewer.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
	_, err = testDB.UpdateBookmarkCollectionIDs(ctx, bk.ID, viewer.ID, nil)
	be.True(t, errors.Is(err, core.ErrNotFound))
	_, err = testDB.GetBookmarkForCollectionEdit(ctx, bk.ID, stranger.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
	_, err = testDB.UpdateBookmarkCollectionIDs(ctx, bk.ID, stranger.ID, nil)
	be.True(t, errors.Is(err, core.ErrNotFound))

	updated, err := testDB.UpdateBookmarkCollectionIDs(ctx, bk.ID, editor.ID, nil)
	be.NilErr(t, err)
	be.Equal(t, 0, len(updated.CollectionIDs))
}

func TestFindUserBookmarkByURL(t *testing.T) {
	ctx := context.Background()
	author, err := testDB.CreateUser(ctx, "dupauthor@test.local", "hash")
	be.NilErr(t, err)
	other, err := testDB.CreateUser(ctx, "dupother@test.local", "hash")
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/dup-check")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Dup Check", author.ID, "", nil, nil, false)
	be.NilErr(t, err)

	// exact URL, same author → found
	got, found, err := testDB.FindUserBookmarkByURL(ctx, author.ID, u.String())
	be.NilErr(t, err)
	be.True(t, found)
	be.Equal(t, bk.ID, got.ID)

	// different author → not found (per-author scope)
	_, found, err = testDB.FindUserBookmarkByURL(ctx, other.ID, u.String())
	be.NilErr(t, err)
	be.False(t, found)

	// an archived copy still reminds
	be.NilErr(t, testDB.ArchiveBookmark(ctx, bk.ID, author.ID))
	_, found, err = testDB.FindUserBookmarkByURL(ctx, author.ID, u.String())
	be.NilErr(t, err)
	be.True(t, found)

	// near-miss URL (added query string) → not found: exact match per FR-11 v1
	near, err := url.Parse("https://example.com/dup-check?x=1")
	be.NilErr(t, err)
	_, found, err = testDB.FindUserBookmarkByURL(ctx, author.ID, near.String())
	be.NilErr(t, err)
	be.False(t, found)
}

func TestArchiveBookmark(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "archive@test.local", "hash")
	be.NilErr(t, err)
	other, err := testDB.CreateUser(ctx, "archother@test.local", "hash")
	be.NilErr(t, err)

	be.NilErr(t, testDB.CreateCollection(ctx, user.ID, "Keep", ""))
	lists, err := testDB.GetCollectionsByUser(ctx, user.ID)
	be.NilErr(t, err)
	colID := lists[0].ID

	u, err := url.Parse("https://example.com/archived")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Archived Post", user.ID, "", []uuid.UUID{colID}, []string{"go"}, false)
	be.NilErr(t, err)

	recent, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, 1, len(recent.Items))

	be.NilErr(t, testDB.ArchiveBookmark(ctx, bk.ID, user.ID))
	be.NilErr(t, testDB.ArchiveBookmark(ctx, bk.ID, user.ID)) // idempotent

	// hidden from all browse lists and the edit fetch
	recent, err = testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, 0, len(recent.Items))
	_, err = testDB.GetBookmarkByID(ctx, bk.ID, user.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
	full, err := testDB.GetCollectionBookmarks(ctx, colID, core.CursorPage{Limit: 20})
	be.NilErr(t, err)
	be.Equal(t, 0, len(full.Items)) // collection browse filters archived too

	// author-only
	err = testDB.ArchiveBookmark(ctx, bk.ID, other.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
}

func TestArchivedLifecycle(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "archlife@test.local", "hash")
	be.NilErr(t, err)
	other, err := testDB.CreateUser(ctx, "archlife2@test.local", "hash")
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/archived")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Archived Post", user.ID, "", nil, []string{"go"}, false)
	be.NilErr(t, err)

	be.NilErr(t, testDB.ArchiveBookmark(ctx, bk.ID, user.ID))

	// the archived list returns it with tags attached
	list, err := testDB.GetArchivedBookmarksByUser(ctx, user.ID)
	be.NilErr(t, err)
	be.Equal(t, 1, len(list))
	be.Equal(t, "Archived Post", list[0].Title)
	be.AllEqual(t, []string{"go"}, list[0].Tags)

	// restore puts it back in browse
	be.NilErr(t, testDB.RestoreBookmark(ctx, bk.ID, user.ID))
	list, err = testDB.GetArchivedBookmarksByUser(ctx, user.ID)
	be.NilErr(t, err)
	be.Equal(t, 0, len(list))
	recent, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, 1, len(recent.Items))

	// archive again, then permanently delete
	be.NilErr(t, testDB.ArchiveBookmark(ctx, bk.ID, user.ID))
	be.NilErr(t, testDB.DeleteBookmark(ctx, bk.ID, user.ID))
	recent, err = testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, 0, len(recent.Items))
	_, err = testDB.GetBookmarkByID(ctx, bk.ID, user.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))

	// author-only restore/delete — on a fresh bookmark
	u2, err := url.Parse("https://example.com/again")
	be.NilErr(t, err)
	bk2, err := testDB.CreateBookmark(ctx, u2, "Another Post", user.ID, "", nil, nil, false)
	be.NilErr(t, err)
	be.NilErr(t, testDB.ArchiveBookmark(ctx, bk2.ID, user.ID))
	err = testDB.RestoreBookmark(ctx, bk2.ID, other.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
	err = testDB.DeleteBookmark(ctx, bk2.ID, other.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
}

func TestBookmarkRepository(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "bookmark@test.local", "hash")
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/post")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Example Post", user.ID, "", nil, []string{"go", "web"}, false)
	be.NilErr(t, err)
	be.Equal(t, "Example Post", bk.Title)

	recent, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, "Example Post", recent.Items[0].Title)
	be.Equal(t, "https://example.com/post", recent.Items[0].URL.String())
	be.Equal(t, "", recent.Items[0].Notes) // default: no note

	// notes round-trip — raw write for now (the repo update path lands with
	// the edit panel step); assert the mapper reads the column.
	_, err = testDB.Pool.Exec(ctx, `update bookmarks set notes = $1 where id = $2`, "Check the grid section", bk.ID)
	be.NilErr(t, err)
	recent, err = testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, "Check the grid section", recent.Items[0].Notes)

	tags, err := testDB.GetTagsByUser(ctx, user.ID)
	be.NilErr(t, err)
	be.Equal(t, 2, len(tags))
	be.True(t, slices.Contains([]string{tags[0].Name, tags[1].Name}, "go"))

	byTags, err := testDB.GetBookmarksByTags(ctx, user.ID, []string{"go"}, "", core.CursorPage{Limit: 20})
	be.NilErr(t, err)
	be.Equal(t, 1, len(byTags.Items))
	be.Equal(t, "Example Post", byTags.Items[0].Title)
}

func TestCollectionMembers(t *testing.T) {
	ctx := context.Background()
	owner, err := testDB.CreateUser(ctx, "owner@test.local", "hash")
	be.NilErr(t, err)
	member, err := testDB.CreateUser(ctx, "member@test.local", "hash")
	be.NilErr(t, err)

	be.NilErr(t, testDB.CreateCollection(ctx, owner.ID, "Shared", ""))
	collections, err := testDB.GetCollectionsByUser(ctx, owner.ID)
	be.NilErr(t, err)
	colID := collections[0].ID

	be.NilErr(t, testDB.AddMember(ctx, colID, "member@test.local", "viewer"))

	full, err := testDB.GetCollection(ctx, colID)
	be.NilErr(t, err)
	be.Equal(t, 2, len(full.Members)) // owner (auto-added) + viewer
	var viewerRole string
	for _, m := range full.Members {
		if m.User.ID == member.ID {
			viewerRole = string(m.Role)
		}
	}
	be.Equal(t, "viewer", viewerRole)

	be.NilErr(t, testDB.RemoveMember(ctx, colID, member.ID))
	full, err = testDB.GetCollection(ctx, colID)
	be.NilErr(t, err)
	be.Equal(t, 1, len(full.Members)) // owner remains
}

func TestGetCollectionAccess(t *testing.T) {
	ctx := context.Background()
	owner, err := testDB.CreateUser(ctx, "accessowner@test.local", "hash")
	be.NilErr(t, err)
	viewer, err := testDB.CreateUser(ctx, "accessviewer@test.local", "hash")
	be.NilErr(t, err)
	editor, err := testDB.CreateUser(ctx, "accesseditor@test.local", "hash")
	be.NilErr(t, err)
	outsider, err := testDB.CreateUser(ctx, "accessoutsider@test.local", "hash")
	be.NilErr(t, err)

	be.NilErr(t, testDB.CreateCollection(ctx, owner.ID, "Shared", ""))
	collections, err := testDB.GetCollectionsByUser(ctx, owner.ID)
	be.NilErr(t, err)
	colID := collections[0].ID

	// creator is the owner
	role, err := testDB.GetCollectionAccess(ctx, colID, owner.ID)
	be.NilErr(t, err)
	be.Equal(t, core.RoleOwner, role)

	// added members round-trip their role
	be.NilErr(t, testDB.AddMember(ctx, colID, viewer.Email, "viewer"))
	role, err = testDB.GetCollectionAccess(ctx, colID, viewer.ID)
	be.NilErr(t, err)
	be.Equal(t, core.RoleViewer, role)

	be.NilErr(t, testDB.AddMember(ctx, colID, editor.Email, "editor"))
	role, err = testDB.GetCollectionAccess(ctx, colID, editor.ID)
	be.NilErr(t, err)
	be.Equal(t, core.RoleEditor, role)

	// non-member has no row → not found
	role, err = testDB.GetCollectionAccess(ctx, colID, outsider.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))

	// nonexistent collection → not found
	role, err = testDB.GetCollectionAccess(ctx, uuid.New(), owner.ID)
	be.True(t, errors.Is(err, core.ErrNotFound))
}

// cursorLess reports whether a sorts strictly before b in the keyset order
// (created_at DESC, id DESC — Postgres compares uuids byte-wise).
func cursorLess(a, b core.BookmarkCursor) bool {
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	return bytes.Compare(a.ID[:], b.ID[:]) > 0
}

func keysetOrdered(t *testing.T, items []core.Bookmark) {
	t.Helper()
	for i := 1; i < len(items); i++ {
		be.True(t, cursorLess(core.CursorOf(items[i-1]), core.CursorOf(items[i])))
	}
}

func cursorPtr(c core.BookmarkCursor) *core.BookmarkCursor { return &c }

func TestBookmarkCursorPagination(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "pager@test.local", "hash")
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/paged")
	be.NilErr(t, err)
	for i := 0; i < 25; i++ {
		_, err = testDB.CreateBookmark(ctx, u, fmt.Sprintf("Post %02d", i), user.ID, "", nil, nil, false)
		be.NilErr(t, err)
	}
	// five bookmarks share one created_at so the id tiebreak decides their order
	_, err = testDB.Pool.Exec(ctx, `update bookmarks set created_at = '2026-01-01 12:00:00' where title like 'Post 0%'`)
	be.NilErr(t, err)

	first, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, 10, len(first.Items))
	be.True(t, first.HasOlder)
	be.False(t, first.HasNewer)
	keysetOrdered(t, first.Items)

	lastOfFirst := core.CursorOf(first.Items[len(first.Items)-1])
	second, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Older: cursorPtr(lastOfFirst), Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, 10, len(second.Items))
	be.True(t, second.HasOlder)
	be.True(t, second.HasNewer) // an older cursor marks a mid-feed window
	keysetOrdered(t, second.Items)
	for _, bm := range second.Items { // every item sits below the cursor
		be.True(t, cursorLess(lastOfFirst, core.CursorOf(bm)))
	}

	older := core.CursorOf(second.Items[len(second.Items)-1])
	third, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Older: cursorPtr(older), Limit: 10})
	be.NilErr(t, err)
	be.Equal(t, 5, len(third.Items))
	be.False(t, third.HasOlder)
	be.True(t, third.HasNewer)
	// the five created_at ties land together on the last page
	for _, bm := range third.Items {
		be.True(t, core.CursorOf(third.Items[0]).CreatedAt.Equal(core.CursorOf(bm).CreatedAt))
	}

	// the keyset order is stable across identical requests
	again, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "", core.CursorPage{Limit: 10})
	be.NilErr(t, err)
	for i, bm := range again.Items {
		be.Equal(t, first.Items[i].ID, bm.ID)
	}
}

func TestBookmarkTagsCursorPagination(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "tagpager@test.local", "hash")
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/tag-paged")
	be.NilErr(t, err)
	for i := 0; i < 12; i++ {
		_, err = testDB.CreateBookmark(ctx, u, fmt.Sprintf("Tagged %02d", i), user.ID, "", nil, []string{"go", "web"}, false)
		be.NilErr(t, err)
	}

	first, err := testDB.GetBookmarksByTags(ctx, user.ID, []string{"go", "web"}, "", core.CursorPage{Limit: 5})
	be.NilErr(t, err)
	be.Equal(t, 5, len(first.Items)) // match-any tags, each bookmark exactly once
	be.True(t, first.HasOlder)
	be.False(t, first.HasNewer)
	keysetOrdered(t, first.Items)

	seen := make(map[uuid.UUID]bool, len(first.Items))
	for _, bm := range first.Items {
		seen[bm.ID] = true
	}
	be.Equal(t, 5, len(seen))

	second, err := testDB.GetBookmarksByTags(ctx, user.ID, []string{"go", "web"}, "", core.CursorPage{Older: cursorPtr(core.CursorOf(first.Items[len(first.Items)-1])), Limit: 5})
	be.NilErr(t, err)
	be.Equal(t, 5, len(second.Items))
	be.True(t, second.HasOlder)
	be.True(t, second.HasNewer)
	for _, bm := range second.Items { // pages are disjoint
		be.False(t, seen[bm.ID])
	}
}

func TestCollectionBookmarksCursorPagination(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "colpager@test.local", "hash")
	be.NilErr(t, err)
	be.NilErr(t, testDB.CreateCollection(ctx, user.ID, "Paged", ""))
	lists, err := testDB.GetCollectionsByUser(ctx, user.ID)
	be.NilErr(t, err)
	colID := lists[0].ID

	u, err := url.Parse("https://example.com/col-paged")
	be.NilErr(t, err)
	for i := 0; i < 12; i++ {
		_, err = testDB.CreateBookmark(ctx, u, fmt.Sprintf("Col %02d", i), user.ID, "", []uuid.UUID{colID}, []string{"go", "web"}, false)
		be.NilErr(t, err)
	}

	first, err := testDB.GetCollectionBookmarks(ctx, colID, core.CursorPage{Limit: 5})
	be.NilErr(t, err)
	be.Equal(t, 5, len(first.Items)) // LIMIT counts bookmarks, not joined tag rows
	be.True(t, first.HasOlder)
	be.False(t, first.HasNewer)
	keysetOrdered(t, first.Items)
	for _, bm := range first.Items { // tag aggregation still intact
		be.Equal(t, 2, len(bm.Tags))
		be.True(t, slices.Contains(bm.Tags, "go"))
		be.True(t, slices.Contains(bm.Tags, "web"))
	}

	second, err := testDB.GetCollectionBookmarks(ctx, colID, core.CursorPage{Older: cursorPtr(core.CursorOf(first.Items[len(first.Items)-1])), Limit: 5})
	be.NilErr(t, err)
	be.Equal(t, 5, len(second.Items))
	be.True(t, second.HasOlder)
	be.True(t, second.HasNewer) // mid-feed window
	keysetOrdered(t, second.Items)
}

// --- test database setup ---

func createTestDatabase(ctx context.Context, adminURL string) (string, error) {
	testURL := replaceDBName(adminURL, testDBName)

	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		return "", err
	}
	defer admin.Close()

	if _, err := admin.Exec(ctx, "drop database if exists "+testDBName+" with (force)"); err != nil {
		return "", fmt.Errorf("drop test database: %w", err)
	}
	if _, err := admin.Exec(ctx, "create database "+testDBName); err != nil {
		return "", fmt.Errorf("create test database: %w", err)
	}

	if err := applyMigrations(ctx, testURL); err != nil {
		return "", err
	}
	return testURL, nil
}

func dropTestDatabase(ctx context.Context, adminURL string) {
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		return
	}
	defer admin.Close()
	_, _ = admin.Exec(ctx, "drop database if exists "+testDBName+" with (force)")
}

func replaceDBName(rawURL, name string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" || u.Path == "/" {
		return rawURL
	}
	segments := strings.Split(u.Path, "/")
	segments[len(segments)-1] = name
	u.Path = strings.Join(segments, "/")
	return u.String()
}

func applyMigrations(ctx context.Context, dbURL string) error {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return err
	}
	// Simple protocol allows multi-statement SQL (triggers contain semicolons).
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	files, err := filepath.Glob("../../resources/db/migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		contents, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}
		up := migrateUpSection(contents)
		if strings.TrimSpace(up) == "" {
			continue
		}
		if _, err := pool.Exec(ctx, up); err != nil {
			return fmt.Errorf("apply %s: %w", filepath.Base(file), err)
		}
	}
	return nil
}

// migrateUpSection returns the "up" half of a dbmate-style migration file.
func migrateUpSection(contents []byte) string {
	s := string(contents)
	const upMarker = "-- migrate:up\n"
	i := strings.Index(s, upMarker)
	if i < 0 {
		return s
	}
	s = s[i+len(upMarker):]
	if j := strings.Index(s, "-- migrate:down"); j >= 0 {
		s = s[:j]
	}
	return s
}
