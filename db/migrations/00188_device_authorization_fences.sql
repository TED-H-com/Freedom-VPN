-- +goose Up
-- D-076 / §17.4: a restored authorization is not a confirmed current entitlement.
-- device_authorization_observations records the withdrawn read-back. This
-- sibling row is what stops the next sync from granting that peer again.
-- Explicit re-authorization deletes the row. Observation writes stay off the
-- device row so they do not bump resource_version.
CREATE TABLE device_authorization_fences (
 device_id UUID PRIMARY KEY REFERENCES member_devices,
 fenced_at TIMESTAMPTZ NOT NULL,
 observed_group_ids UUID[] NOT NULL DEFAULT '{}'
);
COMMENT ON TABLE device_authorization_fences IS '§17.4: authorization-unconfirmed after an external withdrawal. Cleared only by explicit re-authorization.';

-- +goose Down
DROP TABLE device_authorization_fences;
