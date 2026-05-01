-- migrate:up
create unique index if not exists users_identity_id_idx on users (identity_id);

create table if not exists sessions (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    created_at timestamp not null default current_timestamp,
    expires_at timestamp not null
);

-- migrate:down
drop table if exists sessions;
drop index if exists users_identity_id_idx;
