-- migrate:up
alter table collection_members add constraint collection_members_role_check check (role in ('owner','editor','viewer'));

-- migrate:down
alter table collection_members drop constraint collection_members_role_check;
