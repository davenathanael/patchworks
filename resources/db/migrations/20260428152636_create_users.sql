-- migrate:up
create extension if not exists moddatetime;

create table if not exists users (
    id uuid primary key,
    email text unique not null,
    identity_id text not null,
    created_at timestamp not null default current_timestamp,
    updated_at timestamp not null default current_timestamp,
    last_login_at timestamp
);
create trigger update_updated_at before update on users
    for each row execute procedure moddatetime();


-- migrate:down
drop table if exists users;
