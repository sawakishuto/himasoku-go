CREATE TABLE group_members (
    id        uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id  uuid        NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    user_id   uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role      text        NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
    joined_at timestamptz NOT NULL DEFAULT now(),
    -- 退出は論理削除。再参加の履歴を残す。
    left_at   timestamptz
);

-- 在籍中の重複のみを禁じる。退出後の再参加は許容される。
CREATE UNIQUE INDEX group_members_active_uniq
    ON group_members (group_id, user_id)
    WHERE left_at IS NULL;

CREATE INDEX group_members_active_by_user_idx
    ON group_members (user_id)
    WHERE left_at IS NULL;
