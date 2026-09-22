DROP FUNCTION IF EXISTS join_group_by_invite_code(text);
DROP FUNCTION IF EXISTS register_device(text, text, text);

DROP POLICY IF EXISTS notification_deliveries_own_read ON notification_deliveries;
ALTER TABLE notification_deliveries DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS notifications_create        ON notifications;
DROP POLICY IF EXISTS notifications_recipient_read ON notifications;
ALTER TABLE notifications DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS availability_responses_write_self ON availability_responses;
DROP POLICY IF EXISTS availability_responses_read       ON availability_responses;
ALTER TABLE availability_responses DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS availabilities_cancel_own  ON availabilities;
DROP POLICY IF EXISTS availabilities_share_self  ON availabilities;
DROP POLICY IF EXISTS availabilities_group_read  ON availabilities;
ALTER TABLE availabilities DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS group_members_leave_self ON group_members;
DROP POLICY IF EXISTS group_members_join_self  ON group_members;
DROP POLICY IF EXISTS group_members_read       ON group_members;
ALTER TABLE group_members DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS groups_create      ON groups;
DROP POLICY IF EXISTS groups_member_read ON groups;
ALTER TABLE groups DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS devices_own ON devices;
ALTER TABLE devices DISABLE ROW LEVEL SECURITY;

DROP FUNCTION IF EXISTS app_is_group_member(uuid);
DROP FUNCTION IF EXISTS app_current_user_id();

REVOKE ALL ON notification_deliveries FROM himasoku_app;
REVOKE ALL ON notifications           FROM himasoku_app;
REVOKE ALL ON availability_responses  FROM himasoku_app;
REVOKE ALL ON availabilities          FROM himasoku_app;
REVOKE ALL ON group_members           FROM himasoku_app;
REVOKE ALL ON groups                  FROM himasoku_app;
REVOKE ALL ON devices                 FROM himasoku_app;
REVOKE ALL ON users                   FROM himasoku_app;
REVOKE USAGE ON SCHEMA public FROM himasoku_app;

DROP ROLE IF EXISTS himasoku_app;
