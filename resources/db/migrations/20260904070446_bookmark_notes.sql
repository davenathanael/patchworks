-- migrate:up
alter table bookmarks add column notes text not null default '';

-- migrate:down
alter table bookmarks drop column notes;