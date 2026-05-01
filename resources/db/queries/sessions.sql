-- name: CreateSession :one
insert into sessions (id, user_id, expires_at)
values ($1, $2, $3)
returning *;

-- name: GetSessionById :one
select * from sessions
where id = $1 limit 1;

-- name: DeleteSession :exec
delete from sessions where id = $1;
