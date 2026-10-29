-- +goose Up
-- §9.2, §15.2: manual decisions are independent of provider snapshots.
ALTER TABLE members ADD COLUMN admin_suspended BOOLEAN NOT NULL DEFAULT false,
 ADD COLUMN suspension_reason TEXT, ADD COLUMN suspended_at TIMESTAMPTZ,
 ADD COLUMN suspended_by UUID, ADD COLUMN notes TEXT NOT NULL DEFAULT '',
 ADD COLUMN member_group_ids UUID[] NOT NULL DEFAULT '{}';
UPDATE members SET admin_suspended=true, suspension_reason='Legacy suspension retained during migration', suspended_at=updated_at WHERE status='suspended';
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
 revision NUMERIC(20,0) NOT NULL CHECK(revision BETWEEN 1 AND 18446744073709551615),
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
 ADD COLUMN last_seen_at TIMESTAMPTZ, ADD COLUMN observed_authorized BOOLEAN NOT NULL DEFAULT false,
 ADD COLUMN observed_at TIMESTAMPTZ, ADD COLUMN observed_version NUMERIC(20,0) NOT NULL DEFAULT 0,
 ADD COLUMN observed_entitlement_revision NUMERIC(20,0) NOT NULL DEFAULT 0;
-- IP ownership is a historical fact, not an entitlement lease. Infinity here
-- means an open binding until explicit revocation, never infinite VPN access.
-- Zero-duration intervals are permitted for authorize/revoke at the same instant.
ALTER TABLE device_bindings DROP CONSTRAINT device_bindings_check;
ALTER TABLE device_bindings ADD CONSTRAINT device_bindings_interval CHECK(valid_from<=valid_until);

-- §16.3: nullable pending projections must not weaken the existing lease guard.
-- Keep legacy finite projections compatible while denying absent rights and
-- terminal member state; the engine supplies intervals only from paid/grant facts.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_member_lease() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE l serving_leases; e entitlements; m members;
BEGIN
 SELECT * INTO STRICT l FROM serving_leases WHERE id=NEW.lease_id FOR SHARE;
 SELECT * INTO STRICT m FROM members WHERE id=NEW.member_id FOR SHARE;
 SELECT * INTO STRICT e FROM entitlements WHERE member_id=NEW.member_id FOR SHARE;
 IF e.valid_from IS NULL OR e.effective_service_end IS NULL
    OR e.admin_suspended OR m.admin_suspended OR m.status IN ('closed','suspended')
    OR e.revision <> NEW.entitlement_revision
    OR NEW.valid_from < greatest(l.issued_at,e.valid_from)
    OR NEW.expires_at > least(l.expires_at,e.effective_service_end) THEN
  RAISE EXCEPTION 'lease exceeds entitlement or node deadline' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- Bootstrap previously existing projections through the ordinary sweep path.
UPDATE entitlements SET next_transition_at=clock_timestamp() WHERE projection='{}';
