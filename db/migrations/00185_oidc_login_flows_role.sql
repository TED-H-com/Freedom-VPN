-- +goose Up
-- §16.1, D-052 F-11: store requested login role in a dedicated column on
-- oidc_login_flows instead of prefixing verifier.
ALTER TABLE oidc_login_flows
 ADD COLUMN role TEXT CHECK(role IN ('admin','member'));

UPDATE oidc_login_flows
 SET role = 'admin',
     verifier = substr(verifier, length('admin:') + 1)
 WHERE role IS NULL AND verifier LIKE 'admin:%';

UPDATE oidc_login_flows
 SET role = 'member',
     verifier = substr(verifier, length('member:') + 1)
 WHERE role IS NULL AND verifier LIKE 'member:%';

-- +goose Down
UPDATE oidc_login_flows
 SET verifier = role || ':' || verifier
 WHERE role IN ('admin','member')
   AND verifier NOT LIKE 'admin:%'
   AND verifier NOT LIKE 'member:%';

ALTER TABLE oidc_login_flows
 DROP COLUMN role;
