-- name: GetUserById :one
select * from users
where id = $1 limit 1;

-- name: UpsertUser :one
insert into users (id, email, identity_id, last_login_at)
values ($1, $2, $3, current_timestamp)
on conflict (identity_id) do update set
    email = excluded.email,
    last_login_at = current_timestamp
returning *;
