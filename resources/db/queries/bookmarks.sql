-- name: GetBookmarksByCollectionId :many
SELECT
    sqlc.embed(bookmarks),
    sqlc.embed(users),
    bookmark_tags.tag as tag,
    collection_bookmarks.added_at as added_at
FROM bookmarks
JOIN collection_bookmarks ON bookmarks.id = collection_bookmarks.bookmark_id
JOIN bookmark_tags ON bookmarks.id = bookmark_tags.bookmark_id
JOIN users ON bookmarks.author_id = users.id
WHERE collection_bookmarks.collection_id = $1;

-- name: GetRecentBookmarksByUserId :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
ORDER BY bookmarks.created_at DESC
LIMIT 10;


-- name: GetAllBookmarksByUserId :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
WHERE bookmarks.author_id = @author_id::uuid
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
ORDER BY bookmarks.created_at DESC
LIMIT 20;

-- name: GetBookmarksByCollectionAndTags :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
JOIN collection_bookmarks ON bookmarks.id = collection_bookmarks.bookmark_id
JOIN bookmark_tags ON bookmarks.id = bookmark_tags.bookmark_id
WHERE collection_bookmarks.collection_id = $1
AND bookmark_tags.tag = ANY(@tags::text[])
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
ORDER BY bookmarks.created_at DESC
LIMIT 20;

-- name: GetBookmarksByCollection :many
SELECT sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
JOIN collection_bookmarks ON bookmarks.id = collection_bookmarks.bookmark_id
WHERE collection_bookmarks.collection_id = $1
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
ORDER BY bookmarks.created_at DESC
LIMIT 20;

-- name: GetBookmarksByTags :many
SELECT DISTINCT ON (bookmarks.id)
    sqlc.embed(bookmarks), sqlc.embed(users)
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
JOIN bookmark_tags ON bookmarks.id = bookmark_tags.bookmark_id
WHERE bookmark_tags.tag = ANY(@tags::text[])
AND bookmarks.author_id = @author_id::uuid
AND (@search::text = '' OR bookmarks.title ILIKE '%' || @search::text || '%' OR bookmarks.url ILIKE '%' || @search::text || '%')
ORDER BY bookmarks.id, bookmarks.created_at DESC
LIMIT 20;


-- name: GetTagsByBookmarkIds :many
SELECT bookmark_id, tag
FROM bookmark_tags
WHERE bookmark_id = ANY(@bookmark_ids::uuid[]);


-- name: GetTagsByUserId :many
SELECT
    tag,
    COUNT(*) as bookmark_count
FROM bookmark_tags
WHERE author_id = $1
GROUP BY tag;

-- name: CreateBookmark :one
INSERT INTO bookmarks (id, url, title, author_id)
VALUES ($1, $2, $3, @author_id::uuid)
RETURNING *;

-- name: CreateBookmarkTags :copyfrom
INSERT INTO bookmark_tags (bookmark_id, tag, author_id)
VALUES ($1, $2, $3);

-- name: CreateCollectionBookmark :one
INSERT INTO collection_bookmarks (collection_id, bookmark_id)
VALUES ($1, $2)
RETURNING *;
