-- name: RegisterDevice :one
-- push_token の upsert と所有者付け替えは RLS では書けないため関数に閉じる。
-- 呼び出し前に SET LOCAL app.current_user_id が必要。
SELECT * FROM register_device(
    @platform::text,
    @push_token::text,
    sqlc.narg(apns_environment)::text
);

-- name: GetDeviceByPushToken :one
-- push_token は UNIQUE。RLS 下では自分の行だけ返る。
SELECT * FROM devices WHERE push_token = @push_token::text;

-- name: ListDevicesByUserID :many
SELECT * FROM devices
WHERE user_id = @user_id::uuid
ORDER BY last_registered_at DESC;

-- name: UpdateDevice :one
UPDATE devices
SET
    platform = @platform,
    apns_environment = sqlc.narg(apns_environment),
    revoked_at = sqlc.narg(revoked_at),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteDevice :exec
DELETE FROM devices WHERE push_token = @push_token::text;
