-- ============================================================================
-- 認証の識別子を users から切り出す
--
-- 動機は 2 つ。
--   1. 1 人の利用者が複数のプロバイダ（Google / Apple / LINE）で
--      サインインできるようにする。列のままでは 1 対 1 に固定される。
--   2. users は UC-07 のメンバー一覧で広く参照され、RLS の対象外にしている。
--      認証の主体識別子を同じテーブルに置くと、全利用者ぶんが誰にでも読める。
--
-- OIDC の sub は発行者（iss）の中でしか一意性が保証されない。
-- したがってキーは (provider, subject) の組になる。
-- ============================================================================

CREATE TABLE user_identities (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- iss クレームと 1 対 1 に対応する短い名前。対応付けは検証層が持つ。
    -- iss をそのまま持たないのは、URL の表記揺れを避けて索引を素直にするため。
    provider   text        NOT NULL CHECK (provider IN ('firebase', 'google', 'apple', 'line')),
    -- ID トークンの sub クレーム。
    subject    text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    -- 同じ (provider, subject) を 2 人に紐づけない。
    -- 他人の identity を自分に付け替える経路をこの制約で塞ぐ。
    CONSTRAINT user_identities_provider_subject_uniq UNIQUE (provider, subject)
);

-- 同じプロバイダを 1 人に二重に紐づけない。
-- user_id での引き当て（連携済み一覧）もこの索引で賄える。
CREATE UNIQUE INDEX user_identities_user_provider_uniq
    ON user_identities (user_id, provider);

CREATE TRIGGER user_identities_set_updated_at
    BEFORE UPDATE ON user_identities
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ------------------------------------------------------------------ 移行

INSERT INTO user_identities (user_id, provider, subject)
SELECT id, 'firebase', firebase_uid
FROM users;

ALTER TABLE users DROP COLUMN firebase_uid;

-- ------------------------------------------------------------------ RLS

ALTER TABLE user_identities ENABLE ROW LEVEL SECURITY;

-- 自分の連携だけが見える。users と違い広く参照される必要がないので、
-- ここは自分の行に限定できる。
--
-- 新しいプロバイダの連携は、この INSERT ポリシーの下で素の INSERT として行える。
-- 他人の identity を奪おうとしても user_identities_provider_subject_uniq が弾く。
CREATE POLICY user_identities_own ON user_identities
    FOR ALL
    USING      (user_id = app_current_user_id())
    WITH CHECK (user_id = app_current_user_id());

GRANT SELECT, INSERT, DELETE ON user_identities TO himasoku_app;

-- ------------------------------------------------------------------ 引き当て

-- 認証ミドルウェアは app.current_user_id が確定する前にこれを呼ぶ。
-- その時点では自分の行すら RLS で見えないため、SECURITY DEFINER にする。
-- 「本人確認そのもの」は、業務のリクエストより一段内側の操作になる。
CREATE FUNCTION resolve_user_by_identity(p_provider text, p_subject text)
RETURNS uuid
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
    SELECT user_id
    FROM user_identities
    WHERE provider = p_provider
      AND subject  = p_subject;
$$;

REVOKE EXECUTE ON FUNCTION resolve_user_by_identity(text, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION resolve_user_by_identity(text, text) TO himasoku_app;

-- ------------------------------------------------------------------ 登録

-- 2 テーブルにまたがるため、ON CONFLICT 1 文では冪等にできない。
-- CTE で書くと users への INSERT だけが競合に関係なく実行され、
-- 再送のたびに誰にも紐づかない users 行が残る。
--
-- EXCEPTION ブロックは暗黙のセーブポイントを作るので、競合したときだけ
-- users の INSERT を巻き戻して引き直せる。
CREATE FUNCTION register_user(
    p_provider     text,
    p_subject      text,
    p_display_name text,
    p_email        text
)
RETURNS TABLE (user_id uuid, display_name text, email text, created boolean)
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
DECLARE
    v_user_id uuid;
    v_created boolean := false;
BEGIN
    SELECT i.user_id INTO v_user_id
    FROM user_identities i
    WHERE i.provider = p_provider AND i.subject = p_subject;

    IF v_user_id IS NULL THEN
        BEGIN
            INSERT INTO users (display_name, email)
            VALUES (p_display_name, p_email)
            RETURNING id INTO v_user_id;

            INSERT INTO user_identities (user_id, provider, subject)
            VALUES (v_user_id, p_provider, p_subject);

            v_created := true;
        EXCEPTION WHEN unique_violation THEN
            -- ここに来る理由は 2 つある。同時に同じ sub で登録された場合と、
            -- users.email の一意制約に当たった場合。前者だけを吸収する。
            v_user_id := NULL;

            SELECT i.user_id INTO v_user_id
            FROM user_identities i
            WHERE i.provider = p_provider AND i.subject = p_subject;

            IF v_user_id IS NULL THEN
                RAISE;
            END IF;
        END;
    END IF;

    -- 既存ユーザーのプロフィールは上書きしない。
    -- 変更は PATCH /me の役割で、再ログインの副作用にしない。
    RETURN QUERY
        SELECT u.id, u.display_name, u.email, v_created
        FROM users u
        WHERE u.id = v_user_id;
END;
$$;

REVOKE EXECUTE ON FUNCTION register_user(text, text, text, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION register_user(text, text, text, text) TO himasoku_app;
