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

SELECT
    sqlc.embed(bookmarks),
    sqlc.embed(users),
    bookmark_tags.tag as tag
FROM bookmarks
JOIN users ON bookmarks.author_id = users.id
JOIN bookmark_tags ON bookmarks.id = bookmark_tags.bookmark_id
WHERE bookmarks.author_id = $1
ORDER BY bookmarks.created_at DESC
LIMIT 10;
