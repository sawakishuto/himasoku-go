-- 端末ごとの notification_deliveries への展開は配信ワーカーが行う
-- （API ロールは他人の devices を RLS で引けないため）。
-- 展開済みかどうかを打刻しておかないと、送信先の端末が 1 台も無い通知を
-- ワーカーが毎回走査し続けることになる。
ALTER TABLE notifications
    ADD COLUMN expanded_at timestamptz;

-- ワーカーの取り出し口。展開済みの行は索引から外れる。
CREATE INDEX notifications_unexpanded_idx
    ON notifications (created_at)
    WHERE expanded_at IS NULL;
