-- register_user を 2 点変える。
--
-- 1. 集約の根が自分の ID を持つ形に寄せる。
--    DB 側で採番すると、アプリは関数が返るまで自分が作った行の ID を
--    知らない。失敗したときのログと実際に入った行を突き合わせられず、
--    entity.NewUser が採番した ID も捨てることになる。
--
-- 2. 戻り値を uuid ひとつにする。
--    RETURNS TABLE は sqlc が列を展開できず、呼び出し側が interface{} に
--    なる。プロフィールは呼び出し側が既に持っているので、返して嬉しいのは
--    「どの users 行に落ち着いたか」だけ。
--
-- 引数が増えるだけだと古い 4 引数版とのオーバーロードになり、
-- どちらが呼ばれるか曖昧になるので先に落とす。
DROP FUNCTION register_user(text, text, text, text);

CREATE FUNCTION register_user(
    p_user_id      uuid,
    p_provider     text,
    p_subject      text,
    p_display_name text,
    p_email        text
)
RETURNS uuid
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
DECLARE
    v_user_id uuid;
BEGIN
    SELECT i.user_id INTO v_user_id
    FROM user_identities i
    WHERE i.provider = p_provider AND i.subject = p_subject;

    -- 既に登録済みなら何もしない。プロフィールの上書きは PATCH /me の
    -- 役割で、再ログインの副作用にはしない。
    IF v_user_id IS NOT NULL THEN
        RETURN v_user_id;
    END IF;

    BEGIN
        INSERT INTO users (id, display_name, email)
        VALUES (p_user_id, p_display_name, p_email)
        RETURNING id INTO v_user_id;

        INSERT INTO user_identities (user_id, provider, subject)
        VALUES (v_user_id, p_provider, p_subject);
    EXCEPTION WHEN unique_violation THEN
        -- 同時に同じ sub で登録された場合だけを吸収する。
        -- users.email の一意制約や p_user_id の衝突はそのまま上げる。
        v_user_id := NULL;

        SELECT i.user_id INTO v_user_id
        FROM user_identities i
        WHERE i.provider = p_provider AND i.subject = p_subject;

        IF v_user_id IS NULL THEN
            RAISE;
        END IF;
    END;

    -- 返る ID は p_user_id と一致しないことがある。先に他のリクエストが
    -- 同じ sub で登録していれば既存の行の ID が返るため、呼び出し側は
    -- 手元の ID を捨ててこちらを正とする必要がある。
    RETURN v_user_id;
END;
$$;

REVOKE EXECUTE ON FUNCTION register_user(uuid, text, text, text, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION register_user(uuid, text, text, text, text) TO himasoku_app;
