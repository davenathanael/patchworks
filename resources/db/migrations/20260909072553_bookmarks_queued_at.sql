-- migrate:up
ALTER TABLE bookmarks ADD COLUMN IF NOT EXISTS queued_at timestamp;

-- migrate:down
ALTER TABLE bookmarks DROP COLUMN IF EXISTS queued_at;
