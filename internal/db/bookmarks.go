package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/davenathanael/patchworks/internal/core"
	"github.com/davenathanael/patchworks/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (db *DB) GetTagsByUser(ctx context.Context, userID uuid.UUID) ([]core.Tag, error) {
	tags, err := db.querier.GetTagsByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toTags(tags), nil
}

// cursorArgs translates a keyset page request into the nullable cursor
// parameters the paginated queries take; the +1 fetches the proof row past
// the page edge.
func cursorArgsOf(page core.CursorPage) (args struct {
	olderAt pgtype.Timestamp
	olderID uuid.UUID
	limit   int32
}) {
	// handlers pass small page-size constants; the clamp only exists so the
	// int32 LIMIT param can't overflow on a hostile value.
	limit := page.Limit
	if limit < 0 || limit > math.MaxInt32-1 {
		limit = 0
	}
	args.limit = int32(limit) + 1
	if page.Older != nil {
		args.olderAt = pgtype.Timestamp{Time: page.Older.CreatedAt, Valid: true}
		args.olderID = page.Older.ID
	}
	return args
}

// pageSlice computes the visible window of an n-row Limit+1 keyset fetch and
// the pager flags: a surplus row proves the list continues past the fetched
// edge, and an older cursor in the request marks a mid-feed window (newer
// items exist client-side by construction).
func pageSlice(n int, page core.CursorPage) (lo, hi int, hasOlder, hasNewer bool) {
	lo, hi = 0, n
	hasNewer = page.Older != nil
	if n <= page.Limit {
		return lo, hi, false, hasNewer
	}
	return lo, page.Limit, true, hasNewer
}

func (db *DB) GetRecentBookmarksByUser(ctx context.Context, userID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error) {
	args := cursorArgsOf(page)
	rows, err := db.querier.GetRecentBookmarksByUserId(ctx, sqlc.GetRecentBookmarksByUserIdParams{
		AuthorID:  userID,
		Search:    search,
		OlderAt:   args.olderAt,
		OlderID:   args.olderID,
		PageLimit: args.limit,
	})
	if err != nil {
		return core.BookmarkPage{}, err
	}
	return db.paged(ctx, rows, page)
}

func (db *DB) GetAllBookmarksByUser(ctx context.Context, userID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error) {
	args := cursorArgsOf(page)
	rows, err := db.querier.GetAllBookmarksByUserId(ctx, sqlc.GetAllBookmarksByUserIdParams{
		AuthorID:  userID,
		Search:    search,
		OlderAt:   args.olderAt,
		OlderID:   args.olderID,
		PageLimit: args.limit,
	})
	if err != nil {
		return core.BookmarkPage{}, err
	}
	return db.paged(ctx, Map(rows, func(r sqlc.GetAllBookmarksByUserIdRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	}), page)
}

func (db *DB) GetBookmarksByCollectionAndTags(ctx context.Context, collectionID uuid.UUID, tags []string, search string, page core.CursorPage) (core.BookmarkPage, error) {
	args := cursorArgsOf(page)
	rows, err := db.querier.GetBookmarksByCollectionAndTags(ctx, sqlc.GetBookmarksByCollectionAndTagsParams{
		CollectionID: collectionID,
		Tags:         tags,
		Search:       search,
		OlderAt:      args.olderAt,
		OlderID:      args.olderID,
		PageLimit:    args.limit,
	})
	if err != nil {
		return core.BookmarkPage{}, err
	}
	return db.paged(ctx, Map(rows, func(r sqlc.GetBookmarksByCollectionAndTagsRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	}), page)
}

func (db *DB) GetBookmarksByCollection(ctx context.Context, collectionID uuid.UUID, search string, page core.CursorPage) (core.BookmarkPage, error) {
	args := cursorArgsOf(page)
	rows, err := db.querier.GetBookmarksByCollection(ctx, sqlc.GetBookmarksByCollectionParams{
		CollectionID: collectionID,
		Search:       search,
		OlderAt:      args.olderAt,
		OlderID:      args.olderID,
		PageLimit:    args.limit,
	})
	if err != nil {
		return core.BookmarkPage{}, err
	}
	return db.paged(ctx, Map(rows, func(r sqlc.GetBookmarksByCollectionRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	}), page)
}

func (db *DB) GetBookmarksByTags(ctx context.Context, userID uuid.UUID, tags []string, search string, page core.CursorPage) (core.BookmarkPage, error) {
	args := cursorArgsOf(page)
	rows, err := db.querier.GetBookmarksByTags(ctx, sqlc.GetBookmarksByTagsParams{
		Tags:      tags,
		AuthorID:  userID,
		Search:    search,
		OlderAt:   args.olderAt,
		OlderID:   args.olderID,
		PageLimit: args.limit,
	})
	if err != nil {
		return core.BookmarkPage{}, err
	}
	return db.paged(ctx, Map(rows, func(r sqlc.GetBookmarksByTagsRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	}), page)
}

// paged trims the Limit+1 fetch to the visible rows and attaches their tags
// and collections, so only the page's ids hit those queries.
func (db *DB) paged(ctx context.Context, rows []sqlc.GetRecentBookmarksByUserIdRow, page core.CursorPage) (core.BookmarkPage, error) {
	lo, hi, hasOlder, hasNewer := pageSlice(len(rows), page)
	items, err := db.toBookmarksWithTags(ctx, rows[lo:hi])
	if err != nil {
		return core.BookmarkPage{}, err
	}
	return core.BookmarkPage{Items: items, HasOlder: hasOlder, HasNewer: hasNewer}, nil
}

func (db *DB) CreateBookmark(ctx context.Context, url *url.URL, title string, userID uuid.UUID, notes string, collectionIDs []uuid.UUID, tags []string, queued bool) (core.Bookmark, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return core.Bookmark{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	querier := db.querier.WithTx(tx)
	bookmarkID := uuid.New()

	createdBookmark, err := querier.CreateBookmark(ctx, sqlc.CreateBookmarkParams{
		ID:       bookmarkID,
		Url:      url.String(),
		Title:    title,
		Notes:    notes,
		QueuedAt: pgtype.Timestamp{Time: time.Now(), Valid: queued},
		AuthorID: userID,
	})
	if err != nil {
		return core.Bookmark{}, err
	}

	// dedupe while keeping order — the form may submit the same collection twice
	seen := make(map[uuid.UUID]bool, len(collectionIDs))
	links := make([]uuid.UUID, 0, len(collectionIDs))
	for _, cid := range collectionIDs {
		if !seen[cid] {
			seen[cid] = true
			links = append(links, cid)
		}
	}

	if len(links) > 0 {
		linkParams := make([]sqlc.CreateBookmarkCollectionLinksParams, 0, len(links))
		for _, cid := range links {
			linkParams = append(linkParams, sqlc.CreateBookmarkCollectionLinksParams{CollectionID: cid, BookmarkID: bookmarkID})
		}
		if _, err = querier.CreateBookmarkCollectionLinks(ctx, linkParams); err != nil {
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

	b := toBookmark(createdBookmark, tags, user)
	b.CollectionIDs = links
	return b, nil
}

func (db *DB) toBookmarksWithTags(ctx context.Context, rows []sqlc.GetRecentBookmarksByUserIdRow) ([]core.Bookmark, error) {
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

	bookmarks := toBookmarks(rows, tagRows)
	if err := db.attachCollectionIDs(ctx, bookmarks); err != nil {
		return nil, err
	}
	return bookmarks, nil
}

// attachCollectionIDs fills each bookmark's CollectionIDs with the collections
// it belongs to (batched, one query for the whole list).
func (db *DB) attachCollectionIDs(ctx context.Context, bookmarks []core.Bookmark) error {
	if len(bookmarks) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(bookmarks))
	for i, bm := range bookmarks {
		ids[i] = bm.ID
	}
	rows, err := db.querier.GetCollectionIdsByBookmarkIds(ctx, ids)
	if err != nil {
		return err
	}
	byBookmark := make(map[uuid.UUID][]uuid.UUID)
	for _, row := range rows {
		byBookmark[row.BookmarkID] = append(byBookmark[row.BookmarkID], row.CollectionID)
	}
	for i := range bookmarks {
		bookmarks[i].CollectionIDs = byBookmark[bookmarks[i].ID]
	}
	return nil
}

// GetBookmarkByID returns one of the user's own bookmarks with its tags.
// Author-only: the query matches author_id, so other users' bookmarks are not
// visible (404).
func (db *DB) GetBookmarkByID(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	row, err := db.querier.GetBookmarkById(ctx, sqlc.GetBookmarkByIdParams{ID: id, AuthorID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, fmt.Errorf("get bookmark: %w", core.ErrNotFound)
		}
		return core.Bookmark{}, err
	}
	return db.bookmarkWithTags(ctx, id, row.Bookmark, row.User)
}

// FindUserBookmarkByURL returns the author's most recent bookmark with the
// exact URL, regardless of archived state — the FR-11 soft-reminder lookup.
// found=false when the author has no bookmark with that URL.
func (db *DB) FindUserBookmarkByURL(ctx context.Context, userID uuid.UUID, rawURL string) (core.Bookmark, bool, error) {
	row, err := db.querier.FindUserBookmarkByUrl(ctx, sqlc.FindUserBookmarkByUrlParams{AuthorID: userID, Url: rawURL})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, false, nil
		}
		return core.Bookmark{}, false, fmt.Errorf("find bookmark by url: %w", err)
	}
	// No tags/collection links needed for the reminder.
	return toBookmark(row.Bookmark, nil, toUser(row.User)), true, nil
}

// GetBookmarkForCollectionEdit returns a bookmark for the collections picker:
// the author themself, or a member with manage rights (owner/editor) in a
// collection containing the bookmark. Viewers and strangers get ErrNotFound,
// matching the hidden panel. Fine-grained per-collection role checks stay
// with the caller.
func (db *DB) GetBookmarkForCollectionEdit(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	row, err := db.querier.GetBookmarkForCollectionEdit(ctx, sqlc.GetBookmarkForCollectionEditParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, fmt.Errorf("get bookmark for collection edit: %w", core.ErrNotFound)
		}
		return core.Bookmark{}, err
	}
	return db.bookmarkWithTags(ctx, id, row.Bookmark, row.User)
}

// bookmarkWithTags resolves a fetched bookmark row into a core.Bookmark with
// its tags and collection links attached.
func (db *DB) bookmarkWithTags(ctx context.Context, id uuid.UUID, b sqlc.Bookmark, u sqlc.User) (core.Bookmark, error) {
	tags, err := db.tagsForBookmarks(ctx, []uuid.UUID{id})
	if err != nil {
		return core.Bookmark{}, err
	}
	bookmarks := []core.Bookmark{toBookmark(b, tags[id], toUser(u))}
	if err := db.attachCollectionIDs(ctx, bookmarks); err != nil {
		return core.Bookmark{}, err
	}
	return bookmarks[0], nil
}

// UpdateBookmarkNotesTags replaces a bookmark's notes and tags in one
// transaction. Author-only: the UPDATE matches author_id; a mismatch returns
// ErrNotFound. Tags are replaced wholesale (delete + insert), like a fresh save.
func (db *DB) UpdateBookmarkNotesTags(ctx context.Context, id, userID uuid.UUID, notes string, tags []string) (core.Bookmark, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return core.Bookmark{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	querier := db.querier.WithTx(tx)

	if _, err := querier.UpdateBookmarkNotesTags(ctx, sqlc.UpdateBookmarkNotesTagsParams{
		ID:       id,
		AuthorID: userID,
		Notes:    notes,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, fmt.Errorf("update bookmark: %w", core.ErrNotFound)
		}
		return core.Bookmark{}, err
	}

	if err := querier.DeleteBookmarkTags(ctx, sqlc.DeleteBookmarkTagsParams{BookmarkID: id, AuthorID: userID}); err != nil {
		return core.Bookmark{}, err
	}
	if len(tags) > 0 {
		tagParams := make([]sqlc.CreateBookmarkTagsParams, 0, len(tags))
		for _, tag := range tags {
			tagParams = append(tagParams, sqlc.CreateBookmarkTagsParams{BookmarkID: id, Tag: tag, AuthorID: userID})
		}
		if _, err := querier.CreateBookmarkTags(ctx, tagParams); err != nil {
			return core.Bookmark{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Bookmark{}, err
	}

	return db.GetBookmarkByID(ctx, id, userID)
}

// tagsForBookmarks groups tag rows by bookmark id for quick lookup.
func (db *DB) tagsForBookmarks(ctx context.Context, bookmarkIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	tagRows, err := db.querier.GetTagsByBookmarkIds(ctx, bookmarkIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID][]string, len(bookmarkIDs))
	for _, tr := range tagRows {
		out[tr.BookmarkID] = append(out[tr.BookmarkID], tr.Tag)
	}
	return out, nil
}

// ArchiveBookmark sets archived_at (soft-delete). Author-only; ErrNotFound if
// not the author. Idempotent — archiving twice just refreshes the timestamp.
func (db *DB) ArchiveBookmark(ctx context.Context, id, userID uuid.UUID) error {
	if _, err := db.querier.ArchiveBookmark(ctx, sqlc.ArchiveBookmarkParams{ID: id, AuthorID: userID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("archive bookmark: %w", core.ErrNotFound)
		}
		return err
	}
	return nil
}

// GetArchivedBookmarksByUser lists the user's archived bookmarks, newest
// archive first, with tags and collection membership attached.
func (db *DB) GetArchivedBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error) {
	rows, err := db.querier.GetArchivedBookmarksByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}
	recent := Map(rows, func(r sqlc.GetArchivedBookmarksByUserIdRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	})
	return db.toBookmarksWithTags(ctx, recent)
}

// GetQueuedBookmarksByUser lists the user's reading-list bookmarks, oldest
// queue entry first (FIFO), with tags and collection membership attached.
func (db *DB) GetQueuedBookmarksByUser(ctx context.Context, userID uuid.UUID) ([]core.Bookmark, error) {
	rows, err := db.querier.GetQueuedBookmarksByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}
	recent := Map(rows, func(r sqlc.GetQueuedBookmarksByUserIdRow) sqlc.GetRecentBookmarksByUserIdRow {
		return sqlc.GetRecentBookmarksByUserIdRow(r)
	})
	return db.toBookmarksWithTags(ctx, recent)
}

// EnqueueBookmark stamps queued_at (reading-list add). Author-only; ErrNotFound
// if not the author. The guard rejects already-queued or archived rows — the
// toggle decides between enqueue and dequeue before calling.
func (db *DB) EnqueueBookmark(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	row, err := db.querier.EnqueueBookmark(ctx, sqlc.EnqueueBookmarkParams{ID: id, AuthorID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, fmt.Errorf("enqueue bookmark: %w", core.ErrNotFound)
		}
		return core.Bookmark{}, err
	}
	return db.updatedBookmark(ctx, id, userID, row)
}

// DequeueBookmark clears queued_at ("mark as read"). Author-only; ErrNotFound
// if not the author.
func (db *DB) DequeueBookmark(ctx context.Context, id, userID uuid.UUID) (core.Bookmark, error) {
	row, err := db.querier.DequeueBookmark(ctx, sqlc.DequeueBookmarkParams{ID: id, AuthorID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Bookmark{}, fmt.Errorf("dequeue bookmark: %w", core.ErrNotFound)
		}
		return core.Bookmark{}, err
	}
	return db.updatedBookmark(ctx, id, userID, row)
}

// updatedBookmark resolves a just-returned bookmark row into a full
// core.Bookmark, like GetBookmarkByID does.
func (db *DB) updatedBookmark(ctx context.Context, id, userID uuid.UUID, b sqlc.Bookmark) (core.Bookmark, error) {
	user, err := db.querier.GetUserById(ctx, userID)
	if err != nil {
		return core.Bookmark{}, err
	}
	return db.bookmarkWithTags(ctx, id, b, user)
}

// RestoreBookmark clears archived_at, bringing the bookmark back into the
// browse lists. Author-only; ErrNotFound if not the author.
func (db *DB) RestoreBookmark(ctx context.Context, id, userID uuid.UUID) error {
	if _, err := db.querier.RestoreBookmark(ctx, sqlc.RestoreBookmarkParams{ID: id, AuthorID: userID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("restore bookmark: %w", core.ErrNotFound)
		}
		return err
	}
	return nil
}

// DeleteBookmark permanently removes a bookmark (its tags and collection links
// cascade). Author-only; ErrNotFound if not the author.
func (db *DB) DeleteBookmark(ctx context.Context, id, userID uuid.UUID) error {
	deleted, err := db.querier.DeleteBookmark(ctx, sqlc.DeleteBookmarkParams{ID: id, AuthorID: userID})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("delete bookmark: %w", core.ErrNotFound)
	}
	return nil
}

// UpdateBookmarkCollectionIDs replaces the bookmark's links to the user's own
// (member) collections, in one transaction. Links to collections the user is
// not a member of are left untouched — a shared bookmark keeps its place in
// other users' collections. Access: the author, or a member with manage rights
// (owner/editor) in a collection containing the bookmark (ErrNotFound
// otherwise); per-collection role enforcement stays with the caller.
func (db *DB) UpdateBookmarkCollectionIDs(ctx context.Context, bookmarkID, userID uuid.UUID, collectionIDs []uuid.UUID) (core.Bookmark, error) {
	current, err := db.GetBookmarkForCollectionEdit(ctx, bookmarkID, userID)
	if err != nil {
		return core.Bookmark{}, err // author-or-manager check + 404
	}

	allowed, err := db.GetCollectionsByUser(ctx, userID)
	if err != nil {
		return core.Bookmark{}, err
	}
	allowedIDs := make(map[uuid.UUID]bool, len(allowed))
	for _, c := range allowed {
		allowedIDs[c.ID] = true
	}

	// dedupe + restrict to the user's own collections
	seen := make(map[uuid.UUID]bool, len(collectionIDs))
	filtered := make([]uuid.UUID, 0, len(collectionIDs))
	for _, cid := range collectionIDs {
		if !seen[cid] && allowedIDs[cid] {
			seen[cid] = true
			filtered = append(filtered, cid)
		}
	}

	allowedSlice := make([]uuid.UUID, 0, len(allowed))
	for _, c := range allowed {
		allowedSlice = append(allowedSlice, c.ID)
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return core.Bookmark{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	querier := db.querier.WithTx(tx)
	if err := querier.DeleteBookmarkCollectionLinks(ctx, sqlc.DeleteBookmarkCollectionLinksParams{BookmarkID: bookmarkID, Column2: allowedSlice}); err != nil {
		return core.Bookmark{}, err
	}
	if len(filtered) > 0 {
		linkParams := make([]sqlc.CreateBookmarkCollectionLinksParams, 0, len(filtered))
		for _, cid := range filtered {
			linkParams = append(linkParams, sqlc.CreateBookmarkCollectionLinksParams{CollectionID: cid, BookmarkID: bookmarkID})
		}
		if _, err := querier.CreateBookmarkCollectionLinks(ctx, linkParams); err != nil {
			return core.Bookmark{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Bookmark{}, err
	}

	updated := current
	updated.CollectionIDs = survivingCollectionIDs(current.CollectionIDs, filtered, allowedIDs)
	return updated, nil
}

// survivingCollectionIDs returns the bookmark's links outside the caller's
// collections (they survive the update untouched) followed by the filtered
// links that remain after it.
func survivingCollectionIDs(current, filtered []uuid.UUID, allowed map[uuid.UUID]bool) []uuid.UUID {
	kept := make([]uuid.UUID, 0, len(current)+len(filtered))
	for _, cid := range current {
		if !allowed[cid] {
			kept = append(kept, cid)
		}
	}
	return append(kept, filtered...)
}
