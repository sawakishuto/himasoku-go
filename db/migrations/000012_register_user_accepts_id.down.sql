-- 000011 時点の、DB 側で採番する 4 引数版に戻す。
DROP FUNCTION register_user(uuid, text, text, text, text);

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
            v_user_id := NULL;

            SELECT i.user_id INTO v_user_id
            FROM user_identities i
            WHERE i.provider = p_provider AND i.subject = p_subject;

            IF v_user_id IS NULL THEN
                RAISE;
            END IF;
        END;
    END IF;

    RETURN QUERY
        SELECT u.id, u.display_name, u.email, v_created
        FROM users u
        WHERE u.id = v_user_id;
END;
$$;

REVOKE EXECUTE ON FUNCTION register_user(text, text, text, text) FROM PUBLIC;
GRANT  EXECUTE ON FUNCTION register_user(text, text, text, text) TO himasoku_app;
