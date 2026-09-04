//go:build integration

package db

import (
	"context"
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
	be.Equal(t, 0, len(full.Bookmarks))

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
	_, err = testDB.CreateBookmark(ctx, u1, "Tagged Post", user.ID, colID, []string{"go", "web"})
	be.NilErr(t, err)

	u2, err := url.Parse("https://example.com/plain")
	be.NilErr(t, err)
	_, err = testDB.CreateBookmark(ctx, u2, "Plain Post", user.ID, colID, nil)
	be.NilErr(t, err)

	full, err := testDB.GetCollection(ctx, colID)
	be.NilErr(t, err)
	be.Equal(t, 2, len(full.Bookmarks))

	var taggedTags, plainTags []string
	found := make(map[string]bool)
	for i := range full.Bookmarks {
		found[full.Bookmarks[i].Title] = true
		if full.Bookmarks[i].Title == "Tagged Post" {
			taggedTags = full.Bookmarks[i].Tags
		}
		if full.Bookmarks[i].Title == "Plain Post" {
			plainTags = full.Bookmarks[i].Tags
		}
	}
	be.True(t, found["Tagged Post"])
	be.True(t, found["Plain Post"])
	be.Equal(t, 2, len(taggedTags))
	be.True(t, slices.Contains(taggedTags, "go"))
	be.True(t, slices.Contains(taggedTags, "web"))
	be.Equal(t, 0, len(plainTags))
}

func TestBookmarkRepository(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "bookmark@test.local", "hash")
	be.NilErr(t, err)

	u, err := url.Parse("https://example.com/post")
	be.NilErr(t, err)
	bk, err := testDB.CreateBookmark(ctx, u, "Example Post", user.ID, uuid.Nil, []string{"go", "web"})
	be.NilErr(t, err)
	be.Equal(t, "Example Post", bk.Title)

	recent, err := testDB.GetRecentBookmarksByUser(ctx, user.ID, "")
	be.NilErr(t, err)
	be.Equal(t, 1, len(recent))
	be.Equal(t, "Example Post", recent[0].Title)
	be.Equal(t, "https://example.com/post", recent[0].URL.String())

	tags, err := testDB.GetTagsByUser(ctx, user.ID)
	be.NilErr(t, err)
	be.Equal(t, 2, len(tags))
	be.True(t, slices.Contains([]string{tags[0].Name, tags[1].Name}, "go"))

	byTags, err := testDB.GetBookmarksByTags(ctx, user.ID, []string{"go"}, "")
	be.NilErr(t, err)
	be.Equal(t, 1, len(byTags))
	be.Equal(t, "Example Post", byTags[0].Title)
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
			viewerRole = m.Role
		}
	}
	be.Equal(t, "viewer", viewerRole)

	be.NilErr(t, testDB.RemoveMember(ctx, colID, member.ID))
	full, err = testDB.GetCollection(ctx, colID)
	be.NilErr(t, err)
	be.Equal(t, 1, len(full.Members)) // owner remains
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
