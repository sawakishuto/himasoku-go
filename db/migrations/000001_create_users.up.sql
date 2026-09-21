-- updated_at を持つ全テーブルで共有する。個々のクエリでの更新漏れを防ぐ。
CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    firebase_uid text        NOT NULL UNIQUE,
    display_name text        NOT NULL,
    -- 匿名ログインを許容するため NULL 可。NULL は一意制約の対象外。
    email        text        UNIQUE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
