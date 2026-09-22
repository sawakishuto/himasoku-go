-- name: RegisterUser :one
-- users と user_identities を 1 トランザクションで作る。
-- 直接 INSERT すると、identity の無い users 行が残りうる。
--
-- 返るのは落ち着き先の users.id。先に同じ sub で登録されていれば
-- 渡した user_id とは違う値が返るため、呼び出し側は必ず受け取る。
SELECT register_user(
    @user_id::uuid,
    @provider::text,
    @subject::text,
    @display_name::text,
    -- email は NULL 可。空文字のまま入れると、メールを持たない利用者が
    -- 2 人目から UNIQUE に当たる。NULL は重複とみなされない。
    sqlc.narg(email)::text
) AS user_id;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
-- email は NULL 可の列だが、引数として NULL を渡す意味はない
-- （SQL の = は NULL と一致しない）。::text で非 NULL に固定する。
SELECT * FROM users WHERE email = @email::text;
