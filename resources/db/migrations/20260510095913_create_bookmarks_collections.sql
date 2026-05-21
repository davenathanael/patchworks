-- migrate:up
create table if not exists bookmarks (
	id uuid primary key,
	url text not null,
	title text not null,
	created_at timestamp not null default current_timestamp,
	updated_at timestamp not null default current_timestamp,
	archived_at timestamp,
	author_id uuid references users(id) on delete cascade
);
create trigger update_updated_at before update on bookmarks
    for each row execute procedure moddatetime(updated_at);

create table if not exists tags (
	name text not null,
	author_id uuid references users(id) on delete cascade,
	primary key (name, author_id)
);

create table if not exists bookmark_tags (
    bookmark_id uuid references bookmarks(id) on delete cascade,
    tag text references tags(name) on delete cascade,
    primary key (bookmark_id, tag)
);

create table if not exists collections (
	id uuid primary key,
	name text not null,
	description text,
	created_at timestamp not null default current_timestamp,
	updated_at timestamp not null default current_timestamp
);
create trigger update_updated_at before update on collections
    for each row execute procedure moddatetime(updated_at);

create table if not exists collection_bookmarks (
    collection_id uuid references collections(id) on delete cascade,
    bookmark_id uuid references bookmarks(id) on delete cascade,
    added_at timestamp not null default current_timestamp,
    primary key (collection_id, bookmark_id)
);

create table if not exists collection_members (
    collection_id uuid references collections(id) on delete cascade,
    user_id uuid references users(id) on delete cascade,
    role text not null,
    added_at timestamp not null default current_timestamp,
    primary key (collection_id, user_id)
);

-- migrate:down
drop table if exists collection_members;
drop table if exists collection_bookmarks;
drop table if exists collections;
drop table if exists bookmark_tags;
drop table if exists tags;
drop table if exists bookmarks;
