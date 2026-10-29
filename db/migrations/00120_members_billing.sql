-- +goose Up
-- §9.2, §15.2: manual decisions are independent of provider snapshots.
ALTER TABLE members ADD COLUMN admin_suspended BOOLEAN NOT NULL DEFAULT false,
 ADD COLUMN suspension_reason TEXT, ADD COLUMN suspended_at TIMESTAMPTZ,
 ADD COLUMN suspended_by UUID, ADD COLUMN notes TEXT NOT NULL DEFAULT '',
 ADD COLUMN member_group_ids UUID[] NOT NULL DEFAULT '{}';
UPDATE members SET admin_suspended=true, suspension_reason='Legacy suspension retained during migration', suspended_at=updated_at, resource_version=resource_version+1 WHERE status='suspended';
ALTER TABLE entitlement_grants ADD COLUMN plan_revision_id UUID REFERENCES plan_revisions,
 ADD COLUMN operation_id UUID REFERENCES operations;

-- A pending member has no service interval; never fabricate a deadline (§9.2).
ALTER TABLE entitlements ALTER COLUMN valid_from DROP NOT NULL,
 ALTER COLUMN effective_service_end DROP NOT NULL,
 ADD COLUMN state TEXT NOT NULL DEFAULT 'pending_payment' CHECK(state IN ('pending_payment','active','cancel_at_period_end','past_due','grace','expired','suspended','closed')),
 ADD COLUMN projection JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(projection)='object'),
 ADD COLUMN next_transition_at TIMESTAMPTZ;
ALTER TABLE entitlements ADD CONSTRAINT entitlements_nullable_interval CHECK(
 (valid_from IS NULL AND effective_service_end IS NULL) OR
 (valid_from IS NOT NULL AND effective_service_end IS NOT NULL));
CREATE INDEX entitlements_transition_idx ON entitlements(next_transition_at) WHERE next_transition_at IS NOT NULL;
CREATE TABLE entitlement_history (
 id UUID PRIMARY KEY, member_id UUID NOT NULL REFERENCES members,
 revision NUMERIC(20,0) NOT NULL CHECK(revision >= 0 AND revision <= 18446744073709551615) CHECK(revision > 0),
 projection JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(member_id,revision)
);
CREATE TRIGGER entitlement_history_immutable BEFORE UPDATE OR DELETE ON entitlement_history FOR EACH ROW EXECUTE FUNCTION immutable_row();
CREATE TRIGGER entitlement_history_no_truncate BEFORE TRUNCATE ON entitlement_history FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();
COMMENT ON TABLE entitlement_history IS '§9.2, §15.5: immutable entitlement revisions, distinct from node leases.';
CREATE UNIQUE INDEX billing_policy_event_once ON outbox_events(aggregate_id,event_type,aggregate_version)
 WHERE event_type IN ('member.entitlement_changed','member.suspended','device.authorized','device.revoked','netbird.device_sync');

ALTER TABLE subscriptions ADD COLUMN sync_token UUID, ADD COLUMN sync_until TIMESTAMPTZ,
 ADD COLUMN reconciliation_anomalies TEXT[] NOT NULL DEFAULT '{}',
 ADD CONSTRAINT subscriptions_sync_lease CHECK((sync_token IS NULL)=(sync_until IS NULL));
CREATE TABLE billing_customers (
 id UUID PRIMARY KEY, member_id UUID NOT NULL REFERENCES members, provider TEXT NOT NULL,
 external_customer_id TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(member_id,provider), UNIQUE(provider,external_customer_id)
);
COMMENT ON TABLE billing_customers IS '§14.3, §15.2: verified provider customer mapping; never selected by request body.';
ALTER TABLE inbox_events ADD COLUMN subscription_ref TEXT, ADD COLUMN event_type TEXT,
 ADD COLUMN lease_token UUID, ADD COLUMN lease_until TIMESTAMPTZ,
 ADD COLUMN last_error_code TEXT, ADD COLUMN processed_at TIMESTAMPTZ;
CREATE INDEX billing_inbox_pending_idx ON inbox_events(received_at,id) WHERE processing_status<>'succeeded';
ALTER TABLE billing_records ADD COLUMN original_invoice_ref TEXT;
CREATE TABLE billing_sessions (
 id UUID PRIMARY KEY, member_id UUID NOT NULL REFERENCES members, provider TEXT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('checkout','portal')), plan_revision_id UUID REFERENCES plan_revisions,
 operation_id UUID REFERENCES operations, return_url TEXT NOT NULL,
 external_session_id TEXT, encrypted_response BYTEA, expires_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), lease_token UUID, lease_until TIMESTAMPTZ,
 UNIQUE(provider,external_session_id)
);
COMMENT ON TABLE billing_sessions IS '§14.3, §14.5: durable session intent; hosted URLs encrypted and never grant entitlement.';
ALTER TABLE member_devices ADD COLUMN platform TEXT NOT NULL DEFAULT 'unknown',
 ADD COLUMN last_seen_at TIMESTAMPTZ;
-- Observation writes must not mutate desired device state (F-05, F-14).
CREATE TABLE device_authorization_observations (
 device_id UUID PRIMARY KEY REFERENCES member_devices,
 observed_authorized BOOLEAN NOT NULL,
 observed_at TIMESTAMPTZ NOT NULL,
 observed_version NUMERIC(20,0) NOT NULL CHECK(observed_version >= 0 AND observed_version <= 18446744073709551615),
 observed_entitlement_revision NUMERIC(20,0) NOT NULL CHECK(observed_entitlement_revision >= 0 AND observed_entitlement_revision <= 18446744073709551615)
);

COMMENT ON TABLE device_authorization_observations IS '§9.4, §15.5: read-back evidence fenced by device and entitlement revisions.';

-- Pending projections require nullable intervals (§9.2). Preserve ALL base
-- lease/device guards, adding a fail-closed guard for the nullable extension.
-- +goose StatementBegin
CREATE FUNCTION billing_check_service_interval() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE e entitlements; m members;
BEGIN
 IF public.owner_setting('fvpn.restore') THEN RETURN NEW; END IF;
 IF TG_TABLE_NAME='member_devices' THEN
  IF NEW.status<>'authorized' THEN RETURN NEW; END IF;
 END IF;
 SELECT * INTO STRICT m FROM members WHERE id=NEW.member_id FOR UPDATE;
 SELECT * INTO e FROM entitlements WHERE member_id=NEW.member_id FOR SHARE;
 IF NOT FOUND OR e.valid_from IS NULL OR e.effective_service_end IS NULL OR m.admin_suspended THEN
  RAISE EXCEPTION 'service interval absent or manually suspended' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER zz_billing_device_interval BEFORE INSERT OR UPDATE ON member_devices FOR EACH ROW EXECUTE FUNCTION billing_check_service_interval();
CREATE TRIGGER zz_billing_lease_interval BEFORE INSERT ON serving_lease_members FOR EACH ROW EXECUTE FUNCTION billing_check_service_interval();
-- Bootstrap previously existing projections through the ordinary sweep path.
UPDATE entitlements SET next_transition_at=clock_timestamp() WHERE projection='{}';
