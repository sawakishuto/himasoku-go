DROP INDEX IF EXISTS notifications_unexpanded_idx;

ALTER TABLE notifications
    DROP COLUMN IF EXISTS expanded_at;
