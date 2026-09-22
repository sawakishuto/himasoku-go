-- name: ResolveUserIDByIdentity :one
-- 認証ミドルウェアが app.current_user_id の確定前に呼ぶ。
-- その時点では RLS で自分の行すら見えないため、SECURITY DEFINER 関数に委ねる。
-- 未登録なら NULL が返る。
SELECT resolve_user_by_identity(@provider::text, @subject::text) AS user_id;
