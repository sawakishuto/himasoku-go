CREATE TABLE availabilities (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id         uuid        NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    user_id          uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    duration_minutes int         NOT NULL CHECK (duration_minutes > 0),
    message          text,
    started_at       timestamptz NOT NULL DEFAULT now(),
    -- started_at + duration_minutes をトリガーで導出する。
    -- timestamptz + interval はタイムゾーン設定に依存し IMMUTABLE ではないため、
    -- 生成列（GENERATED ALWAYS AS ... STORED）は使えない。
    expires_at       timestamptz NOT NULL,
    cancelled_at     timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT availabilities_expires_after_start CHECK (expires_at > started_at)
);

CREATE FUNCTION set_availability_expires_at() RETURNS trigger AS $$
BEGIN
    NEW.expires_at = NEW.started_at + make_interval(mins => NEW.duration_minutes);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- INSERT 時に expires_at を無視して導出するため、アプリ側は値を渡さなくてよい。
CREATE TRIGGER availabilities_set_expires_at
    BEFORE INSERT OR UPDATE OF started_at, duration_minutes ON availabilities
    FOR EACH ROW
    EXECUTE FUNCTION set_availability_expires_at();

-- 「今ヒマな人」の引き当て用。
CREATE INDEX availabilities_active_by_group_idx
    ON availabilities (group_id, expires_at DESC)
    WHERE cancelled_at IS NULL;

CREATE INDEX availabilities_user_idx
    ON availabilities (user_id, started_at DESC);

CREATE TRIGGER availabilities_set_updated_at
    BEFORE UPDATE ON availabilities
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
