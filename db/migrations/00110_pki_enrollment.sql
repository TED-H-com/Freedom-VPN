-- +goose Up
-- §6.1, §15.3. Metadata is an immutable snapshot of the enrollment command.
ALTER TABLE node_enrollments
 ADD COLUMN delivery TEXT NOT NULL DEFAULT 'installer' CHECK(delivery IN ('installer','preinstalled')),
 ADD COLUMN name TEXT, ADD COLUMN region TEXT, ADD COLUMN notes TEXT,
 ADD COLUMN member_group_ids UUID[] NOT NULL DEFAULT '{}',
 ADD COLUMN incarnation_id UUID REFERENCES node_incarnations,
 ADD COLUMN request_hash BYTEA CHECK(octet_length(request_hash)=32),
 ADD COLUMN certificate_id UUID REFERENCES node_certificates,
 ADD COLUMN operation_id UUID REFERENCES operations;
ALTER TABLE node_certificates ADD COLUMN usable_until TIMESTAMPTZ;
CREATE TABLE node_certificate_renewals (
 incarnation_id UUID NOT NULL REFERENCES node_incarnations,
 request_id UUID NOT NULL, request_hash BYTEA NOT NULL CHECK(octet_length(request_hash)=32),
 certificate_id UUID NOT NULL REFERENCES node_certificates,
 PRIMARY KEY(incarnation_id,request_id)
);
-- Only authenticated delivery may claim ciphertext. Never persist a secret in
-- operations, audit, idempotency or outbox payloads (§6.4, §14.7).
CREATE TABLE node_bootstrap_secrets (
 incarnation_id UUID PRIMARY KEY REFERENCES node_incarnations,
 operation_id UUID NOT NULL UNIQUE REFERENCES operations,
 external_key_id TEXT NOT NULL UNIQUE,
 encryption_key_id TEXT, ciphertext BYTEA,
 expires_at TIMESTAMPTZ NOT NULL CHECK(isfinite(expires_at)),
 delivered_at TIMESTAMPTZ, erased_at TIMESTAMPTZ,
 CHECK((encryption_key_id IS NULL)=(ciphertext IS NULL)),
 CHECK((erased_at IS NULL)=(ciphertext IS NOT NULL))
);
CREATE TABLE node_identity_endpoints (
 incarnation_id UUID PRIMARY KEY REFERENCES node_incarnations,
 endpoint_hash BYTEA NOT NULL CHECK(octet_length(endpoint_hash)=32),
 last_seen_at TIMESTAMPTZ NOT NULL, mapping_frozen BOOLEAN NOT NULL DEFAULT false
);
CREATE TABLE node_identity_conflicts (
 id UUID PRIMARY KEY, incarnation_id UUID NOT NULL REFERENCES node_incarnations,
 previous_hash BYTEA NOT NULL CHECK(octet_length(previous_hash)=32),
 conflicting_hash BYTEA NOT NULL CHECK(octet_length(conflicting_hash)=32),
 observed_at TIMESTAMPTZ NOT NULL,
 UNIQUE(incarnation_id,previous_hash,conflicting_hash)
);

-- §6.2: forward-only lifecycle and revocation are enforced below the services.
-- +goose StatementBegin
CREATE FUNCTION pki_guard_terminal() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_TABLE_NAME='nodes' THEN
  IF OLD.lifecycle_state IN ('revoked','retired') AND NEW.lifecycle_state NOT IN ('revoked','retired') THEN
   RAISE EXCEPTION 'terminal node identity' USING ERRCODE='23514';
  END IF;
 ELSIF OLD.revoked_at IS NOT NULL AND NEW.revoked_at IS DISTINCT FROM OLD.revoked_at THEN
  RAISE EXCEPTION 'terminal revocation' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION pki_guard_enrollment() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.state<>'pending' AND NEW.state<>OLD.state
    OR OLD.request_id IS NOT NULL AND NEW.request_id IS DISTINCT FROM OLD.request_id
    OR OLD.incarnation_id IS NOT NULL AND NEW.incarnation_id IS DISTINCT FROM OLD.incarnation_id
    OR OLD.request_hash IS NOT NULL AND NEW.request_hash IS DISTINCT FROM OLD.request_hash
    OR OLD.certificate_id IS NOT NULL AND NEW.certificate_id IS DISTINCT FROM OLD.certificate_id
    OR OLD.operation_id IS NOT NULL AND NEW.operation_id IS DISTINCT FROM OLD.operation_id THEN
  RAISE EXCEPTION 'terminal enrollment binding' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION pki_guard_overlap() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.usable_until IS NOT NULL AND (NEW.usable_until IS NULL OR NEW.usable_until>OLD.usable_until) THEN
  RAISE EXCEPTION 'certificate overlap cannot increase' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER pki_nodes_terminal BEFORE UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION pki_guard_terminal();
CREATE TRIGGER pki_incarnations_terminal BEFORE UPDATE ON node_incarnations FOR EACH ROW EXECUTE FUNCTION pki_guard_terminal();
CREATE TRIGGER pki_certificates_terminal BEFORE UPDATE ON node_certificates FOR EACH ROW EXECUTE FUNCTION pki_guard_terminal();
CREATE TRIGGER pki_enrollments_terminal BEFORE UPDATE ON node_enrollments FOR EACH ROW EXECUTE FUNCTION pki_guard_enrollment();
CREATE TRIGGER pki_certificates_overlap BEFORE UPDATE ON node_certificates FOR EACH ROW EXECUTE FUNCTION pki_guard_overlap();
CREATE TRIGGER pki_certificates_identity BEFORE UPDATE ON node_certificates FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','incarnation_id','issuer','serial','fingerprint','public_certificate','not_before','not_after');
CREATE TRIGGER pki_certificates_keep BEFORE DELETE OR TRUNCATE ON node_certificates FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER pki_incarnations_keep BEFORE DELETE OR TRUNCATE ON node_incarnations FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER pki_renewals_immutable BEFORE UPDATE OR DELETE ON node_certificate_renewals FOR EACH ROW EXECUTE FUNCTION immutable_row();
