CREATE TABLE notification_deliveries (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id   uuid        NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
    device_id         uuid        NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    status            text        NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending', 'sent', 'failed')),
    apns_id           text,
    -- BadDeviceToken なら devices.revoked_at を打刻する。
    error_reason      text,
    attempts          int         NOT NULL DEFAULT 0,
    last_attempted_at timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notification_deliveries_once_per_device UNIQUE (notification_id, device_id)
);

-- 送信ワーカーの取り出し口（アウトボックス）。
CREATE INDEX notification_deliveries_pending_idx
    ON notification_deliveries (created_at)
    WHERE status = 'pending';
