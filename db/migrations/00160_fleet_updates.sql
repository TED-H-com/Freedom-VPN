-- +goose Up
-- §13.3: product_version is display only, not a content identity.
ALTER TABLE releases DROP CONSTRAINT releases_version_key;
CREATE TABLE release_verifications (
 release_id UUID PRIMARY KEY REFERENCES releases,
 manifest_url TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('pending','verified','failed')),
 error_code TEXT,
 operation_id UUID REFERENCES operations,
 verified_at TIMESTAMPTZ
);
CREATE TABLE release_registration_rejections (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), release_id UUID,
 manifest_hash BYTEA NOT NULL CHECK(octet_length(manifest_hash)=32),
 error_code TEXT NOT NULL, eligible BOOLEAN NOT NULL DEFAULT false CHECK(NOT eligible),
 received_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
ALTER TABLE fleet_config ADD COLUMN approved_release_ids UUID[] NOT NULL DEFAULT '{}';
ALTER TABLE fleet_config ADD COLUMN revoked_release_ids UUID[] NOT NULL DEFAULT '{}';
ALTER TABLE fleet_config ADD COLUMN support_generation NUMERIC(20,0) NOT NULL DEFAULT 0 CHECK(support_generation BETWEEN 0 AND 18446744073709551615);
CREATE TRIGGER fleet_support_generation BEFORE UPDATE ON fleet_config FOR EACH ROW EXECUTE FUNCTION monotonic_fields('support_generation');
ALTER TABLE update_batches ADD COLUMN reason TEXT NOT NULL DEFAULT '';
ALTER TABLE update_batches ADD COLUMN online_at_snapshot INTEGER NOT NULL DEFAULT 0 CHECK(online_at_snapshot >= 0 AND online_at_snapshot <= target_count);
ALTER TABLE update_targets ADD COLUMN online_at_snapshot BOOLEAN NOT NULL DEFAULT false;
-- §13.1: enrolling jobs exist before execution identity; first-sync jobs have no target.
ALTER TABLE update_jobs ALTER COLUMN execution_incarnation_id DROP NOT NULL;
ALTER TABLE update_jobs ALTER COLUMN target_id DROP NOT NULL;
ALTER TABLE update_jobs ADD COLUMN node_id UUID REFERENCES nodes;
ALTER TABLE update_jobs ADD COLUMN release_id UUID REFERENCES releases;
ALTER TABLE update_jobs ADD COLUMN issued_at TIMESTAMPTZ;
ALTER TABLE update_jobs ADD COLUMN authorization_valid_until TIMESTAMPTZ;
ALTER TABLE update_jobs ADD COLUMN phase_since TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp();
ALTER TABLE update_jobs ADD COLUMN needs_attention BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE update_jobs ADD COLUMN recovery_required BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE update_jobs ADD COLUMN stage TEXT;
UPDATE update_jobs j SET node_id=t.node_id,release_id=b.target_release_id FROM update_targets t JOIN update_batches b ON b.id=t.batch_id WHERE j.target_id=t.id;
ALTER TABLE update_jobs ALTER COLUMN node_id SET NOT NULL;
ALTER TABLE update_jobs ALTER COLUMN release_id SET NOT NULL;
ALTER TABLE update_jobs ADD CONSTRAINT update_jobs_node_incarnation_fk FOREIGN KEY(node_id,execution_incarnation_id) REFERENCES node_incarnations(node_id,id);
ALTER TABLE update_jobs DROP CONSTRAINT update_jobs_state_check;
ALTER TABLE update_jobs ADD CHECK(state IN ('queued','waiting_online','downloading','verifying','applying','checking','succeeded','already_current','failed','rolled_back','superseded','superseded_incarnation','cancelled_revoked','cancelled_retired'));
CREATE INDEX update_jobs_node_generation ON update_jobs(node_id,generation DESC,attempt DESC);
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION guard_job_identity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (NEW.id,NEW.target_id,NEW.attempt,NEW.generation,NEW.node_id,NEW.release_id) IS DISTINCT FROM (OLD.id,OLD.target_id,OLD.attempt,OLD.generation,OLD.node_id,OLD.release_id)
 OR (OLD.execution_incarnation_id IS NOT NULL AND NEW.execution_incarnation_id IS DISTINCT FROM OLD.execution_incarnation_id)
 OR (OLD.result_id IS NOT NULL AND (NEW.result_id,NEW.result_hash) IS DISTINCT FROM (OLD.result_id,OLD.result_hash)) THEN
 RAISE EXCEPTION 'immutable job identity or result' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION check_job_success() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.state IN ('succeeded','already_current') AND
 (NEW.actual_release_id IS DISTINCT FROM NEW.release_id OR NEW.result_id IS NULL OR NEW.completed_at IS NULL OR NEW.observed_evidence='{}') THEN
 RAISE EXCEPTION 'target completion requires version and evidence' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TABLE fleet_node_state (
 node_id UUID PRIMARY KEY REFERENCES nodes, current_job_id UUID REFERENCES update_jobs,
 observation_incarnation_id UUID,
 observation_seq NUMERIC(20,0) NOT NULL DEFAULT 0 CHECK(observation_seq BETWEEN 0 AND 18446744073709551615),
 observed_update_generation NUMERIC(20,0) NOT NULL DEFAULT 0 CHECK(observed_update_generation BETWEEN 0 AND 18446744073709551615),
 last_observed_at TIMESTAMPTZ, received_at TIMESTAMPTZ,
 recovery_required BOOLEAN NOT NULL DEFAULT false,
 FOREIGN KEY(node_id,observation_incarnation_id) REFERENCES node_incarnations(node_id,id)
);
CREATE TABLE update_results (
 result_id UUID PRIMARY KEY, job_id UUID NOT NULL REFERENCES update_jobs,
 node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 payload_hash BYTEA NOT NULL CHECK(octet_length(payload_hash)=32), payload JSONB NOT NULL,
 ack JSONB NOT NULL, received_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id)
);
CREATE TABLE update_result_conflicts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), result_id UUID NOT NULL REFERENCES update_results,
 conflicting_hash BYTEA NOT NULL CHECK(octet_length(conflicting_hash)=32),
 received_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(), UNIQUE(result_id,conflicting_hash)
);
CREATE TABLE update_batch_progress (
 batch_id UUID PRIMARY KEY REFERENCES update_batches, state TEXT NOT NULL, counts JSONB NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE TRIGGER update_results_immutable BEFORE UPDATE OR DELETE ON update_results FOR EACH ROW EXECUTE FUNCTION immutable_row();
CREATE TRIGGER update_results_no_truncate BEFORE TRUNCATE ON update_results FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();
CREATE TRIGGER update_conflicts_immutable BEFORE UPDATE OR DELETE ON update_result_conflicts FOR EACH ROW EXECUTE FUNCTION immutable_row();
-- Cancellation is in the lifecycle owner's transaction, without changing snapshots or evidence (§13.1).
-- +goose StatementBegin
CREATE FUNCTION cancel_fleet_jobs() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.lifecycle_state IN ('revoked','retired') AND NEW.lifecycle_state IS DISTINCT FROM OLD.lifecycle_state THEN
 UPDATE update_jobs SET state='cancelled_'||NEW.lifecycle_state,phase_since=clock_timestamp()
 WHERE node_id=NEW.id AND state NOT IN ('succeeded','already_current','failed','rolled_back','superseded','superseded_incarnation','cancelled_revoked','cancelled_retired');
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER nodes_cancel_fleet_jobs AFTER UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION cancel_fleet_jobs();
CREATE TABLE update_retry_jobs (
 operation_id UUID NOT NULL REFERENCES operations,job_id UUID NOT NULL REFERENCES update_jobs,
 PRIMARY KEY(operation_id,job_id)
);
-- Existing DB primitive fixtures and older writers derive the new job columns.
-- +goose StatementBegin
CREATE FUNCTION derive_fleet_job() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.target_id IS NOT NULL THEN
 SELECT t.node_id,b.target_release_id INTO NEW.node_id,NEW.release_id FROM update_targets t JOIN update_batches b ON b.id=t.batch_id WHERE t.id=NEW.target_id;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER update_jobs_derive BEFORE INSERT ON update_jobs FOR EACH ROW EXECUTE FUNCTION derive_fleet_job();
COMMENT ON TABLE release_verifications IS 'Project Spec §13.3–4: verification state independent of immutable release content.';
COMMENT ON TABLE release_registration_rejections IS 'Project Spec §13.4: rejected registration hashes are permanently ineligible.';
COMMENT ON TABLE fleet_node_state IS 'Project Spec §13.8, §17.4: desired job pointer and ordered actual-version observation.';
COMMENT ON TABLE update_results IS 'Project Spec §13.10.3: append-only result inbox and original ACK.';
COMMENT ON TABLE update_result_conflicts IS 'Project Spec §13.11: conflicting result hashes retained without replacing accepted evidence.';
COMMENT ON TABLE update_batch_progress IS 'Project Spec §13.12: rebuildable mutually exclusive progress counts.';
COMMENT ON TABLE update_retry_jobs IS 'Project Spec §13.12: retry operation evidence references its exact attempts.';
