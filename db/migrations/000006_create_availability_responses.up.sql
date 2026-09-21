CREATE TABLE availability_responses (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 送信者とグループはここから辿れるため、クライアントに申告させない。
    availability_id uuid        NOT NULL REFERENCES availabilities (id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    action          text        NOT NULL CHECK (action IN ('join', 'decline')),
    -- iOS の 30 秒無操作による自動辞退と、明示的な辞退を区別する。
    source          text        NOT NULL CHECK (source IN ('user', 'auto_timeout')),
    responded_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT availability_responses_once_per_user UNIQUE (availability_id, user_id)
);
