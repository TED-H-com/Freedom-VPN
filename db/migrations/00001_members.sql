-- +goose Up

-- Goose's default metadata timestamp is timezone-less; normalize it too (§15.1).
ALTER TABLE goose_db_version ALTER COLUMN tstamp TYPE TIMESTAMPTZ USING tstamp AT TIME ZONE 'UTC';
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- §15.2: stable identity, independent of email.
CREATE TABLE members (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 canonical_identity_ref UUID,
 email TEXT, display_name TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'pending_payment' CHECK (status IN ('pending_payment','active','expired','suspended','closed')),
 resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), closed_at TIMESTAMPTZ
);
CREATE INDEX members_status_idx ON members(status);

-- §2, §15.2: pairwise subjects remain scoped; email is not a mapping key.
CREATE TABLE identity_bindings (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members,
 issuer TEXT NOT NULL, client_scope TEXT NOT NULL, subject TEXT NOT NULL,
 stable_claim_ref TEXT NOT NULL, binding_method TEXT NOT NULL, verified_at TIMESTAMPTZ NOT NULL,
 CONSTRAINT identity_bindings_subject_key UNIQUE(issuer,client_scope,subject),
 CONSTRAINT identity_bindings_member_key UNIQUE(id,member_id)
);
ALTER TABLE members ADD CONSTRAINT members_canonical_identity_fk FOREIGN KEY(canonical_identity_ref,id) REFERENCES identity_bindings(id,member_id);

-- §15.2: verified external user mapping, not server peers.
CREATE TABLE netbird_user_bindings (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members,
 deployment_id TEXT NOT NULL, external_user_id TEXT NOT NULL,
 identity_binding_id UUID NOT NULL, mapping_proof TEXT NOT NULL, revision NUMERIC(20,0) CHECK (revision >= 0 AND revision <= 18446744073709551615) NOT NULL,
 verified_at TIMESTAMPTZ NOT NULL,
 FOREIGN KEY(identity_binding_id,member_id) REFERENCES identity_bindings(id,member_id),
 CONSTRAINT netbird_user_bindings_external_key UNIQUE(deployment_id,external_user_id)
);

-- §15.2: immutable plan revisions are separate from the active pointer.
CREATE TABLE plans (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), code TEXT NOT NULL UNIQUE,
 active_revision_id UUID, resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE plan_revisions (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), plan_id UUID NOT NULL REFERENCES plans,
 revision NUMERIC(20,0) CHECK (revision >= 0 AND revision <= 18446744073709551615) NOT NULL, device_limit INTEGER NOT NULL CHECK(device_limit >= 0),
 exit_group_ids UUID[] NOT NULL DEFAULT '{}',
 fup_policy_revision_id UUID, provider TEXT, provider_price_ref TEXT,
 schema_version INTEGER NOT NULL CHECK(schema_version > 0),
 payload JSONB NOT NULL CHECK(jsonb_typeof(payload) = 'object'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CONSTRAINT plan_revisions_revision_key UNIQUE(plan_id,revision),
 CONSTRAINT plan_revisions_plan_key UNIQUE(id,plan_id),
 CHECK((provider IS NULL) = (provider_price_ref IS NULL))
);
ALTER TABLE plans ADD CONSTRAINT plans_active_revision_fk FOREIGN KEY(active_revision_id,id) REFERENCES plan_revisions(id,plan_id);

-- §9.2, §15.2: billing facts and primary active subscription are independent.
CREATE TABLE subscriptions (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members,
 provider TEXT NOT NULL, external_subscription_id TEXT NOT NULL, external_customer_id TEXT NOT NULL,
 plan_revision_id UUID NOT NULL REFERENCES plan_revisions,
 provider_status TEXT NOT NULL, is_primary BOOLEAN NOT NULL DEFAULT true, is_active BOOLEAN NOT NULL DEFAULT false,
 paid_from TIMESTAMPTZ, paid_through TIMESTAMPTZ,
 cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
 grace_from TIMESTAMPTZ, grace_until TIMESTAMPTZ, grace_reason TEXT,
 last_reconciled_at TIMESTAMPTZ, schema_version INTEGER NOT NULL DEFAULT 1 CHECK(schema_version > 0),
 provider_snapshot JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(provider_snapshot) = 'object'),
 resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CONSTRAINT subscriptions_external_key UNIQUE(provider,external_subscription_id),
 CONSTRAINT subscriptions_member_key UNIQUE(id,member_id),
 CHECK((paid_from IS NULL AND paid_through IS NULL) OR (paid_from IS NOT NULL AND paid_through IS NOT NULL AND isfinite(paid_from) AND isfinite(paid_through) AND paid_from < paid_through)),
 CHECK((grace_from IS NULL AND grace_until IS NULL) OR (grace_from IS NOT NULL AND grace_until IS NOT NULL AND isfinite(grace_from) AND isfinite(grace_until) AND grace_from < grace_until AND grace_reason IS NOT NULL AND grace_reason ~ '[^[:space:]]'))
);
CREATE UNIQUE INDEX subscriptions_primary_active_key ON subscriptions(member_id) WHERE is_primary AND is_active;
CREATE INDEX subscriptions_plan_idx ON subscriptions(plan_revision_id);

-- §9.3, §15.4: caller encrypts authenticated webhook body before insertion.
CREATE TABLE inbox_events (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), source TEXT NOT NULL, external_event_id TEXT NOT NULL,
 payload_hash BYTEA NOT NULL CHECK(octet_length(payload_hash)=32), encrypted_payload BYTEA NOT NULL,
 received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 processing_status TEXT NOT NULL DEFAULT 'pending' CHECK(processing_status IN ('pending','processing','succeeded','failed')),
 attempt INTEGER NOT NULL DEFAULT 0 CHECK(attempt >= 0),
 CONSTRAINT inbox_events_external_key UNIQUE(source,external_event_id)
);

-- §9.3, §15.2: unsigned amounts; record_type distinguishes refunds/corrections.
CREATE TABLE billing_records (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members,
 subscription_id UUID, provider TEXT NOT NULL,
 record_type TEXT NOT NULL, external_reference TEXT NOT NULL,
 amount_minor BIGINT NOT NULL CHECK(amount_minor >= 0 AND amount_minor <= 9223372036854775807),
 currency CHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 service_start TIMESTAMPTZ NOT NULL, service_end TIMESTAMPTZ NOT NULL,
 source_event_id UUID NOT NULL REFERENCES inbox_events,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CONSTRAINT billing_records_external_key UNIQUE(provider,record_type,external_reference),
 CONSTRAINT billing_records_subscription_member_fk FOREIGN KEY(subscription_id,member_id) REFERENCES subscriptions(id,member_id),
 CHECK(isfinite(service_start) AND isfinite(service_end) AND service_start < service_end)
);

-- §9.2, §15.2: finite and attributable manual/trial/grace grants.
CREATE TABLE entitlement_grants (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members,
 source TEXT NOT NULL, valid_from TIMESTAMPTZ NOT NULL, valid_until TIMESTAMPTZ NOT NULL,
 reason TEXT NOT NULL CHECK(reason ~ '[^[:space:]]'), created_by TEXT NOT NULL CHECK(created_by ~ '[^[:space:]]'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), revoked_at TIMESTAMPTZ, revocation_reason TEXT,
 CHECK(isfinite(valid_from) AND isfinite(valid_until) AND valid_from < valid_until),
 CHECK(revoked_at IS NULL OR (revocation_reason IS NOT NULL AND revocation_reason ~ '[^[:space:]]'))
);

-- §9.2, §15.2: recomputable projection; never holds a node lease expiry.
CREATE TABLE entitlements (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL UNIQUE REFERENCES members,
 revision NUMERIC(20,0) CHECK (revision >= 0 AND revision <= 18446744073709551615) NOT NULL, plan_revision_id UUID REFERENCES plan_revisions,
 device_limit INTEGER NOT NULL CHECK(device_limit >= 0),
 valid_from TIMESTAMPTZ NOT NULL, effective_service_end TIMESTAMPTZ NOT NULL,
 admin_suspended BOOLEAN NOT NULL DEFAULT false,
 source_refs UUID[] NOT NULL DEFAULT '{}', computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK(isfinite(valid_from) AND isfinite(effective_service_end) AND valid_from < effective_service_end)
);
CREATE INDEX entitlements_plan_idx ON entitlements(plan_revision_id);

-- §9.4, §15.2: authorize while holding the member row lock.
CREATE TABLE member_devices (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members,
 deployment_id TEXT NOT NULL, peer_id TEXT NOT NULL, name TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','authorized','revoked','closed')),
 authorized_at TIMESTAMPTZ, revoked_at TIMESTAMPTZ, closed_at TIMESTAMPTZ,
 resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CONSTRAINT member_devices_peer_key UNIQUE(deployment_id,peer_id),
 CONSTRAINT member_devices_member_key UNIQUE(id,member_id)
);
CREATE INDEX member_devices_member_status_idx ON member_devices(member_id,status);

-- §2, §15.2: historical IP ownership uses half-open ranges in an overlay scope.
CREATE TABLE device_bindings (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), device_id UUID NOT NULL,
 member_id UUID NOT NULL REFERENCES members, overlay_scope TEXT NOT NULL, overlay_ip INET NOT NULL,
 binding_revision NUMERIC(20,0) CHECK (binding_revision >= 0 AND binding_revision <= 18446744073709551615) NOT NULL, valid_from TIMESTAMPTZ NOT NULL, valid_until TIMESTAMPTZ NOT NULL,
 peer_ref TEXT NOT NULL,
 FOREIGN KEY(device_id,member_id) REFERENCES member_devices(id,member_id),
 CONSTRAINT device_bindings_revision_key UNIQUE(device_id,binding_revision),
 CHECK(isfinite(valid_from) AND isfinite(valid_until) AND valid_from < valid_until),
 CHECK(masklen(overlay_ip) = CASE family(overlay_ip) WHEN 4 THEN 32 ELSE 128 END),
 CONSTRAINT device_bindings_no_overlap EXCLUDE USING gist(overlay_scope WITH =, overlay_ip WITH =, tstzrange(valid_from,valid_until,'[)') WITH &&)
);

-- §15.2: groups express membership, never a source trust tier.
CREATE TABLE member_groups (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), code TEXT NOT NULL UNIQUE, name TEXT NOT NULL UNIQUE
);
CREATE TABLE assignments (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID REFERENCES members,
 plan_revision_id UUID REFERENCES plan_revisions, group_id UUID NOT NULL REFERENCES member_groups,
 valid_from TIMESTAMPTZ NOT NULL, valid_until TIMESTAMPTZ NOT NULL,
 CHECK(num_nonnulls(member_id,plan_revision_id)=1), CHECK(isfinite(valid_from) AND isfinite(valid_until) AND valid_from < valid_until),
 CONSTRAINT assignments_member_no_overlap EXCLUDE USING gist(member_id WITH =,group_id WITH =,tstzrange(valid_from,valid_until,'[)') WITH &&) WHERE(member_id IS NOT NULL),
 CONSTRAINT assignments_plan_no_overlap EXCLUDE USING gist(plan_revision_id WITH =,group_id WITH =,tstzrange(valid_from,valid_until,'[)') WITH &&) WHERE(plan_revision_id IS NOT NULL)
);
