-- name: GetUserById :one
select * from users
where id = $1 limit 1;

-- name: GetUserByEmail :one
select * from users
where email = $1 limit 1;

-- name: CreateUser :one
insert into users (id, email, password_hash)
values ($1, $2, $3)
returning *;

-- name: SetUserLastLoginAt :exec
update users
set last_login_at = now()
where id = $1;
