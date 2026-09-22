-- ============================================================================
-- Row Level Security
--
-- アプリのバグ（WHERE 句の付け忘れ、パスパラメータの検証漏れ）が
-- そのまま他人のデータの露出にならないよう、DB 側に最後の防波堤を置く。
--
-- 重要: RLS はテーブル所有者には既定で適用されない。現在の himasoku は
-- Superuser かつ BYPASSRLS のため、このロールで接続している限りポリシーは
-- 一切評価されない。API は必ず himasoku_app で接続すること。
-- ============================================================================

-- ------------------------------------------------------------------ ロール

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'himasoku_app') THEN
        -- パスワードは意図的に設定しない。ローカルは make db-app-password、
        -- 本番はシークレット管理側で与える。マイグレーションに資格情報を残さない。
        CREATE ROLE himasoku_app LOGIN PASSWORD NULL;
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO himasoku_app;

-- 付与する操作は各ユースケースで実際に必要なものだけに絞る。
-- RLS は「どの行か」を制御するが、「そもそも触れるか」は GRANT の仕事。
GRANT SELECT, INSERT, UPDATE ON users                  TO himasoku_app;
GRANT SELECT, INSERT, UPDATE ON devices                TO himasoku_app;
GRANT SELECT, INSERT         ON groups                 TO himasoku_app;
GRANT SELECT, INSERT, UPDATE ON group_members          TO himasoku_app;
GRANT SELECT, INSERT, UPDATE ON availabilities         TO himasoku_app;
GRANT SELECT, INSERT         ON availability_responses TO himasoku_app;
GRANT SELECT, INSERT         ON notifications          TO himasoku_app;
-- 配信行の作成はワーカー（所有者接続）の担当。API 側は参照のみ。
GRANT SELECT                 ON notification_deliveries TO himasoku_app;

-- ------------------------------------------------------------------ ヘルパー

-- リクエストの実行主体。ハンドラがトランザクション冒頭で
--   SET LOCAL app.current_user_id = '<users.id>'
-- を発行する。未設定なら NULL を返し、全ポリシーが偽になって何も見えない。
CREATE FUNCTION app_current_user_id() RETURNS uuid
    LANGUAGE sql
    STABLE
    SET search_path = public, pg_temp
AS $$
    SELECT nullif(current_setting('app.current_user_id', true), '')::uuid;
$$;

-- group_members のポリシーから group_members を引くと無限再帰になる。
-- SECURITY DEFINER で所有者として実行し、RLS を経由せずに判定する。
CREATE FUNCTION app_is_group_member(target_group_id uuid) RETURNS boolean
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
    SELECT EXISTS (
        SELECT 1
        FROM group_members
        WHERE group_id = target_group_id
          AND user_id  = app_current_user_id()
          AND left_at IS NULL
    );
$$;

REVOKE EXECUTE ON FUNCTION app_is_group_member(uuid) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION app_is_group_member(uuid) TO himasoku_app;

-- ------------------------------------------------------------------ users
--
-- users だけは RLS を掛けない。理由は 2 つある。
--   1. 認証ミドルウェアは firebase_uid から users.id を引く。この時点では
--      app.current_user_id がまだ確定しておらず、自分の行すら見えない。
--   2. UC-07（グループメンバー一覧）で同じグループの他人の display_name を
--      読む必要がある。「自分の行だけ」では成立しない。
-- 代償として、アプリロールは全ユーザーの email / firebase_uid を読めてしまう。
-- 外部に出す列はハンドラ側で絞ること。

-- ------------------------------------------------------------------ devices

ALTER TABLE devices ENABLE ROW LEVEL SECURITY;

-- 端末トークンは他人に見せない。通知送信時の引き当てはワーカーが所有者接続で行う。
CREATE POLICY devices_own ON devices
    FOR ALL
    USING      (user_id = app_current_user_id())
    WITH CHECK (user_id = app_current_user_id());

-- ------------------------------------------------------------------ groups

ALTER TABLE groups ENABLE ROW LEVEL SECURITY;

-- 招待コードによる参加は join_group_by_invite_code() に閉じ込める。
-- 非メンバーに groups の SELECT を許すと、コードの総当たりが可能になる。
CREATE POLICY groups_member_read ON groups
    FOR SELECT
    USING (app_is_group_member(id));

CREATE POLICY groups_create ON groups
    FOR INSERT
    WITH CHECK (created_by = app_current_user_id());

-- UPDATE / DELETE のポリシーは置かない。該当するユースケースが無く、
-- ポリシー不在は「一行も対象にならない」を意味するので安全側に倒れる。

-- ------------------------------------------------------------------ group_members

ALTER TABLE group_members ENABLE ROW LEVEL SECURITY;

-- 同じグループに在籍していれば他メンバーの行も見える（UC-07）。
CREATE POLICY group_members_read ON group_members
    FOR SELECT
    USING (app_is_group_member(group_id));

CREATE POLICY group_members_join_self ON group_members
    FOR INSERT
    WITH CHECK (user_id = app_current_user_id());

-- 退出（left_at の打刻）は自分の行のみ。他人を勝手に脱退させられない。
CREATE POLICY group_members_leave_self ON group_members
    FOR UPDATE
    USING      (user_id = app_current_user_id())
    WITH CHECK (user_id = app_current_user_id());

-- ------------------------------------------------------------------ availabilities

ALTER TABLE availabilities ENABLE ROW LEVEL SECURITY;

CREATE POLICY availabilities_group_read ON availabilities
    FOR SELECT
    USING (app_is_group_member(group_id));

-- 在籍していないグループに暇を流せない。UC-09 の在籍チェックの二重化。
CREATE POLICY availabilities_share_self ON availabilities
    FOR INSERT
    WITH CHECK (
        user_id = app_current_user_id()
        AND app_is_group_member(group_id)
    );

-- 取り消せるのは共有した本人だけ（UC-13）。
CREATE POLICY availabilities_cancel_own ON availabilities
    FOR UPDATE
    USING      (user_id = app_current_user_id())
    WITH CHECK (user_id = app_current_user_id());

-- ------------------------------------------------------------------ availability_responses

ALTER TABLE availability_responses ENABLE ROW LEVEL SECURITY;

-- 自分の応答と、自分が流した暇に対する応答が見える。
CREATE POLICY availability_responses_read ON availability_responses
    FOR SELECT
    USING (
        user_id = app_current_user_id()
        OR EXISTS (
            SELECT 1 FROM availabilities a
            WHERE a.id = availability_id
              AND a.user_id = app_current_user_id()
        )
    );

-- availabilities は自身の RLS で在籍中のグループのものしか見えない。
-- したがって EXISTS が真になる時点で「在籍者による応答」が保証される。
CREATE POLICY availability_responses_write_self ON availability_responses
    FOR INSERT
    WITH CHECK (
        user_id = app_current_user_id()
        AND EXISTS (SELECT 1 FROM availabilities a WHERE a.id = availability_id)
    );

-- ------------------------------------------------------------------ notifications

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY notifications_recipient_read ON notifications
    FOR SELECT
    USING (recipient_user_id = app_current_user_id());

-- 注意: 通知は常に「他人宛」に作られるため、INSERT ... RETURNING は使えない。
-- RETURNING は挿入した行を読み返す操作であり、上の SELECT ポリシーに掛かって
-- 「new row violates row-level security policy」で失敗する。
-- id が必要なときは Go 側で uuid を採番して渡すこと（sqlc なら :exec を使う）。

-- Rails 版では sender_firebase_uid が自己申告だったため、任意の相手に
-- 「◯◯が共感しています」を送れた。その穴をポリシーとして塞ぐ。
CREATE POLICY notifications_create ON notifications
    FOR INSERT
    WITH CHECK (
        -- 招待通知: 自分が流した暇について、同じグループの在籍者宛にのみ作れる
        (
            kind = 'availability_invite'
            AND EXISTS (
                SELECT 1 FROM availabilities a
                WHERE a.id = availability_id
                  AND a.user_id = app_current_user_id()
                  AND EXISTS (
                      SELECT 1 FROM group_members m
                      WHERE m.group_id = a.group_id
                        AND m.user_id  = notifications.recipient_user_id
                        AND m.left_at IS NULL
                  )
            )
        )
        OR
        -- 結果通知: 自分が応答した暇について、その共有者宛にのみ作れる
        (
            kind = 'response_result'
            AND EXISTS (
                SELECT 1 FROM availabilities a
                WHERE a.id = availability_id
                  AND a.user_id = notifications.recipient_user_id
                  AND EXISTS (
                      SELECT 1 FROM availability_responses r
                      WHERE r.availability_id = a.id
                        AND r.user_id = app_current_user_id()
                  )
            )
        )
    );

-- ------------------------------------------------------------------ notification_deliveries

ALTER TABLE notification_deliveries ENABLE ROW LEVEL SECURITY;

-- 配信行の作成には宛先の devices.id が要るが、他人の端末は RLS で見えない。
-- そのため notifications を展開して配信行を作るのはワーカー（所有者接続）の
-- 役割とし、API ロールには自分宛の配信状況の参照だけを許す。
CREATE POLICY notification_deliveries_own_read ON notification_deliveries
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM notifications n
            WHERE n.id = notification_id
              AND n.recipient_user_id = app_current_user_id()
        )
    );

-- ------------------------------------------------------------------ 境界をまたぐ操作
--
-- 以下は設計上どうしても他人の行に触れる必要があり、ポリシーでは表現できない。
-- SECURITY DEFINER 関数に閉じ込め、関数そのものを検査点にする。

-- 同じ端末で別アカウントにサインインし直すと、同一の push_token が
-- 別ユーザーの行として既に存在する（UC-02 の所有者付け替え）。
-- push_token を知っていること自体が端末の占有証明になる。
CREATE FUNCTION register_device(
    p_platform         text,
    p_push_token       text,
    p_apns_environment text
) RETURNS devices
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
DECLARE
    v_user_id uuid := app_current_user_id();
    v_device  devices;
BEGIN
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'app.current_user_id is not set';
    END IF;

    INSERT INTO devices (user_id, platform, push_token, apns_environment)
    VALUES (v_user_id, p_platform, p_push_token, p_apns_environment)
    ON CONFLICT (push_token) DO UPDATE
        SET user_id            = v_user_id,
            platform           = EXCLUDED.platform,
            apns_environment   = EXCLUDED.apns_environment,
            last_registered_at = now(),
            -- 再インストールで同じトークンが再発行された場合に失効を解除する
            revoked_at         = NULL
    RETURNING * INTO v_device;

    RETURN v_device;
END;
$$;

REVOKE EXECUTE ON FUNCTION register_device(text, text, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION register_device(text, text, text) TO himasoku_app;

-- 参加前は非メンバーなので groups も group_members も一行も見えない。
-- 招待コードの照合をこの関数の中だけで行い、コードの総当たりを防ぐ。
CREATE FUNCTION join_group_by_invite_code(p_invite_code text) RETURNS groups
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
DECLARE
    v_user_id uuid := app_current_user_id();
    v_group   groups;
BEGIN
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'app.current_user_id is not set';
    END IF;

    SELECT * INTO v_group FROM groups WHERE invite_code = p_invite_code;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    -- group_members_active_uniq は在籍中の重複だけを禁じる部分索引のため、
    -- ON CONFLICT の推論に同じ述語を書いて対象を一致させる。
    INSERT INTO group_members (group_id, user_id)
    VALUES (v_group.id, v_user_id)
    ON CONFLICT (group_id, user_id) WHERE left_at IS NULL DO NOTHING;

    RETURN v_group;
END;
$$;

REVOKE EXECUTE ON FUNCTION join_group_by_invite_code(text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION join_group_by_invite_code(text) TO himasoku_app;
