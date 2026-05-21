-- name: GetCollectionsByUser :many
SELECT sqlc.embed(collections), sqlc.embed(collection_members)
FROM collections
JOIN collection_members ON collections.id = collection_members.collection_id
WHERE collection_members.user_id = $1;

-- name: GetCollectionById :one
SELECT sqlc.embed(collections), sqlc.embed(collection_members)
FROM collections
JOIN collection_members ON collections.id = collection_members.collection_id
WHERE collections.id = $1;

-- name: CreateCollection :one
INSERT INTO collections (id, name, description) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateCollection :one
UPDATE collections SET name = $2, description = $3 WHERE id = $1 RETURNING *;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = $1;

-- name: AddCollectionMember :exec
INSERT INTO collection_members (collection_id, user_id, role) VALUES ($1, $2, $3);

-- name: RemoveCollectionMember :exec
DELETE FROM collection_members WHERE collection_id = $1 AND user_id = $2;
