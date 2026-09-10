-- name: GetBookmarksByCollectionId :many
-- The CTE applies the keyset predicate + LIMIT to bookmark ids before the
-- tag join, so the page counts bookmarks, not joined tag rows.
WITH page AS (
    SELECT bookmarks.id
    FROM bookmarks
    JOIN collection_bookmarks ON bookmarks.id = collection_bookmarks.bookmark_id
    WHERE collection_bookmarks.collection_id = @collection_id::uuid
    AND bookmarks.archived_at IS NULL
    AND (
        (@older_at::timestamp IS NULL OR (bookmarks.created_at, bookmarks.id) < (@older_at::timestamp, @older_id::uuid))
    )
    ORDER BY bookmarks.created_at DESC, bookmarks.id DESC
    LIMIT @page_limit::int
)
SELECT
    sqlc.embed(bookmarks),
    sqlc.embed(users),
    bookmark_tags.tag as tag,
    collection_bookmarks.added_at as added_at
FROM page
JOIN bookmarks ON bookmarks.id = page.id
JOIN collection_bookmarks ON bookmarks.id = collection_bookmarks.bookmark_id
LEFT JOIN bookmark_tags ON bookmarks.id = bookmark_tags.bookmark_id
JOIN users ON bookmarks.author_id = users.id
ORDER BY bookmarks.created_at DESC, bookmarks.id DESC;

-- name: GetRecentBookmarksByUserId :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid
AND bookmarks.archived_at IS NULL
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
AND (
    (@older_at::timestamp IS NULL OR (bookmarks.created_at, bookmarks.id) < (@older_at::timestamp, @older_id::uuid))
)
ORDER BY bookmarks.created_at DESC, bookmarks.id DESC
LIMIT @page_limit::int;


-- name: GetAllBookmarksByUserId :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid
AND bookmarks.archived_at IS NULL
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
AND (
    (@older_at::timestamp IS NULL OR (bookmarks.created_at, bookmarks.id) < (@older_at::timestamp, @older_id::uuid))
)
ORDER BY bookmarks.created_at DESC, bookmarks.id DESC
LIMIT @page_limit::int;

-- name: GetBookmarksByCollectionAndTags :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
JOIN collection_bookmarks ON bookmarks.id = collection_bookmarks.bookmark_id
JOIN bookmark_tags ON bookmarks.id = bookmark_tags.bookmark_id
WHERE collection_bookmarks.collection_id = @collection_id::uuid
AND bookmark_tags.tag = ANY(@tags::text[])
AND bookmarks.archived_at IS NULL
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
AND (
    (@older_at::timestamp IS NULL OR (bookmarks.created_at, bookmarks.id) < (@older_at::timestamp, @older_id::uuid))
)
ORDER BY bookmarks.created_at DESC, bookmarks.id DESC
LIMIT @page_limit::int;

-- name: GetBookmarksByCollection :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
JOIN collection_bookmarks ON bookmarks.id = collection_bookmarks.bookmark_id
WHERE collection_bookmarks.collection_id = @collection_id::uuid
AND bookmarks.archived_at IS NULL
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
AND (
    (@older_at::timestamp IS NULL OR (bookmarks.created_at, bookmarks.id) < (@older_at::timestamp, @older_id::uuid))
)
ORDER BY bookmarks.created_at DESC, bookmarks.id DESC
LIMIT @page_limit::int;

-- name: GetBookmarksByTags :many
-- EXISTS keeps the original join's match-any-tag semantics while dropping the
-- DISTINCT ON workaround: each bookmark row appears exactly once, so the
-- keyset ordering is well-defined.
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid
AND bookmarks.archived_at IS NULL
AND EXISTS (
    SELECT 1 FROM bookmark_tags bt
    WHERE bt.bookmark_id = bookmarks.id AND bt.tag = ANY(@tags::text[])
)
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
AND (
    (@older_at::timestamp IS NULL OR (bookmarks.created_at, bookmarks.id) < (@older_at::timestamp, @older_id::uuid))
)
ORDER BY bookmarks.created_at DESC, bookmarks.id DESC
LIMIT @page_limit::int;


-- name: GetTagsByBookmarkIds :many
SELECT bookmark_id, tag
FROM bookmark_tags
WHERE bookmark_id = ANY(@bookmark_ids::uuid[]);


-- name: GetBookmarkById :one
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.id = @id::uuid AND bookmarks.author_id = @author_id::uuid AND bookmarks.archived_at IS NULL;

-- name: GetBookmarkForCollectionEdit :one
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.id = @id::uuid AND bookmarks.archived_at IS NULL
  AND (
    bookmarks.author_id = @user_id::uuid
    OR EXISTS (
      SELECT 1
      FROM collection_bookmarks cb
      JOIN collection_members cm ON cm.collection_id = cb.collection_id
      WHERE cb.bookmark_id = bookmarks.id
        AND cm.user_id = @user_id::uuid
        AND cm.role IN ('owner', 'editor')
    )
  );

-- name: FindUserBookmarkByUrl :one
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid AND bookmarks.url = @url
ORDER BY bookmarks.created_at DESC
LIMIT 1;

-- name: ArchiveBookmark :one
UPDATE bookmarks
SET archived_at = now(), queued_at = NULL
WHERE id = @id::uuid AND author_id = @author_id::uuid
RETURNING *;

-- name: GetArchivedBookmarksByUserId :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid
AND bookmarks.archived_at IS NOT NULL
ORDER BY bookmarks.archived_at DESC;

-- name: RestoreBookmark :one
UPDATE bookmarks
SET archived_at = NULL
WHERE id = @id::uuid AND author_id = @author_id::uuid
RETURNING *;

-- name: GetQueuedBookmarksByUserId :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid
AND bookmarks.queued_at IS NOT NULL
AND bookmarks.archived_at IS NULL
ORDER BY bookmarks.queued_at ASC;

-- name: EnqueueBookmark :one
UPDATE bookmarks
SET queued_at = now()
WHERE id = @id::uuid AND author_id = @author_id::uuid
  AND queued_at IS NULL AND archived_at IS NULL
RETURNING *;

-- name: DequeueBookmark :one
UPDATE bookmarks
SET queued_at = NULL
WHERE id = @id::uuid AND author_id = @author_id::uuid
RETURNING *;

-- name: DeleteBookmark :execrows
DELETE FROM bookmarks
WHERE id = @id::uuid AND author_id = @author_id::uuid;

-- name: UpdateBookmarkNotesTags :one
UPDATE bookmarks
SET notes = @notes::text
WHERE id = @id::uuid AND author_id = @author_id::uuid
RETURNING *;

-- name: DeleteBookmarkTags :exec
DELETE FROM bookmark_tags
WHERE bookmark_id = $1 AND author_id = $2;


-- name: GetTagsByUserId :many
SELECT
    tag,
    COUNT(*) as bookmark_count
FROM bookmark_tags
WHERE author_id = $1
GROUP BY tag;

-- name: CreateBookmark :one
INSERT INTO bookmarks (id, url, title, notes, queued_at, author_id)
VALUES ($1, $2, $3, $4, @queued_at::timestamp, @author_id::uuid)
RETURNING *;

-- name: CreateBookmarkTags :copyfrom
INSERT INTO bookmark_tags (bookmark_id, tag, author_id)
VALUES ($1, $2, $3);

-- name: CreateCollectionBookmark :one
INSERT INTO collection_bookmarks (collection_id, bookmark_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetCollectionIdsByBookmarkIds :many
SELECT bookmark_id, collection_id
FROM collection_bookmarks
WHERE bookmark_id = ANY(@bookmark_ids::uuid[]);

-- name: DeleteBookmarkCollectionLinks :exec
DELETE FROM collection_bookmarks
WHERE bookmark_id = $1 AND collection_id = ANY($2::uuid[]);

-- name: CreateBookmarkCollectionLinks :copyfrom
INSERT INTO collection_bookmarks (collection_id, bookmark_id)
VALUES ($1, $2);
