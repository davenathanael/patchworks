-- migrate:up
drop index users_identity_id_idx;
alter table users drop column identity_id;
alter table users add column password_hash text;

-- migrate:down
alter table users add column identity_id text;
alter table users drop column password_hash;
create unique index users_identity_id_idx on users (identity_id);
