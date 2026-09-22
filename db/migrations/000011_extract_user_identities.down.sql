DROP FUNCTION IF EXISTS register_user(text, text, text, text);
DROP FUNCTION IF EXISTS resolve_user_by_identity(text, text);

ALTER TABLE users ADD COLUMN firebase_uid text;

UPDATE users u
SET firebase_uid = i.subject
FROM user_identities i
WHERE i.user_id = u.id
  AND i.provider = 'firebase';

-- firebase 以外のプロバイダだけで登録された利用者がいると、ここで失敗する。
-- 列に戻せない状態であることを黙って握り潰さず、明示的に止める。
ALTER TABLE users ALTER COLUMN firebase_uid SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_firebase_uid_key UNIQUE (firebase_uid);

DROP POLICY IF EXISTS user_identities_own ON user_identities;
REVOKE ALL ON user_identities FROM himasoku_app;
DROP TABLE IF EXISTS user_identities;
