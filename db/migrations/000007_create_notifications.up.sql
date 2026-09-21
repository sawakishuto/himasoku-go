CREATE TABLE notifications (
    -- この id をペイロードの notification_id としてそのまま送る。
    -- iOS は「応答済み」判定のキーに使うため、宛先ごとに 1 行必要。
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    kind              text        NOT NULL CHECK (kind IN ('availability_invite', 'response_result')),
    availability_id   uuid        REFERENCES availabilities (id) ON DELETE CASCADE,
    recipient_user_id uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title             text        NOT NULL,
    body              text        NOT NULL,
    payload           jsonb       NOT NULL DEFAULT '{}',
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notifications_recipient_idx
    ON notifications (recipient_user_id, created_at DESC);

CREATE INDEX notifications_availability_idx
    ON notifications (availability_id);
