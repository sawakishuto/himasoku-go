CREATE TABLE devices (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    platform           text        NOT NULL CHECK (platform IN ('ios', 'android')),
    -- APNs トークンは再発行されるため主キーにはしない。
    push_token         text        NOT NULL UNIQUE,
    apns_environment   text        CHECK (apns_environment IN ('sandbox', 'production')),
    -- BadDeviceToken を受けたらここに打刻して送信対象から外す。
    revoked_at         timestamptz,
    last_registered_at timestamptz NOT NULL DEFAULT now(),
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- 送信対象の引き当ては常に「生きている端末」に限定される。
CREATE INDEX devices_active_by_user_idx
    ON devices (user_id)
    WHERE revoked_at IS NULL;

CREATE TRIGGER devices_set_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
