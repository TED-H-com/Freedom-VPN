-- +goose Up
-- D-013 / §15.3: keep the base immutable, unique release version.
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
-- §13.1: first-sync is a separate, sealed one-node system snapshot. The public
-- all_nodes path retains the base population validator without modification.
ALTER TABLE update_batches ADD COLUMN first_sync_node_id UUID REFERENCES nodes;
ALTER TABLE update_batches DROP CONSTRAINT update_batches_scope_check;
ALTER TABLE update_batches ADD CONSTRAINT update_batches_scope_check CHECK(scope IN ('all_nodes','first_sync'));
ALTER TABLE update_batches ADD CONSTRAINT update_batches_first_sync_check CHECK(
 (scope='all_nodes' AND first_sync_node_id IS NULL) OR
 (scope='first_sync' AND first_sync_node_id IS NOT NULL AND target_count=1
  AND reason='first_sync' AND created_by='00000000-0000-0000-0000-000000000003'));
DROP TRIGGER update_batches_count ON update_batches;
CREATE CONSTRAINT TRIGGER update_batches_count AFTER INSERT ON update_batches DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.scope='all_nodes') EXECUTE FUNCTION check_snapshot_count();
-- +goose StatementBegin
CREATE FUNCTION check_first_sync_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (SELECT count(*) FROM update_targets WHERE batch_id=NEW.id) <> 1
 OR NOT EXISTS(SELECT 1 FROM update_targets t JOIN nodes n ON n.id=t.node_id
  WHERE t.batch_id=NEW.id AND t.node_id=NEW.first_sync_node_id
  AND n.lifecycle_state NOT IN ('revoked','retired')
  AND t.incarnation_at_snapshot IS NOT DISTINCT FROM n.current_incarnation_id
  AND t.execution_incarnation_id=n.current_incarnation_id) THEN
  RAISE EXCEPTION 'first_sync requires exactly its authenticated node' USING ERRCODE='23514';
 END IF;
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER update_batches_first_sync_count AFTER INSERT ON update_batches DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.scope='first_sync') EXECUTE FUNCTION check_first_sync_snapshot();
-- Unbound enrolling targets have no job yet; preserve the base NOT NULLs/FKs.
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
CREATE INDEX update_jobs_node_generation ON update_jobs(node_id,generation DESC,attempt DESC);
CREATE TRIGGER update_jobs_fleet_identity BEFORE UPDATE ON update_jobs FOR EACH ROW EXECUTE FUNCTION immutable_fields('node_id','release_id');
CREATE TABLE fleet_node_state (
 node_id UUID PRIMARY KEY REFERENCES nodes, current_job_id UUID,
 current_target_id UUID REFERENCES update_targets,
 observation_incarnation_id UUID,
 observation_seq NUMERIC(20,0) NOT NULL DEFAULT 0 CHECK(observation_seq BETWEEN 0 AND 18446744073709551615),
 observed_update_generation NUMERIC(20,0) NOT NULL DEFAULT 0 CHECK(observed_update_generation BETWEEN 0 AND 18446744073709551615),
 last_observed_at TIMESTAMPTZ, received_at TIMESTAMPTZ,
 recovery_required BOOLEAN NOT NULL DEFAULT false,
 FOREIGN KEY(node_id,observation_incarnation_id) REFERENCES node_incarnations(node_id,id)
);
CREATE TABLE update_results (
 result_id UUID PRIMARY KEY, job_id UUID NOT NULL,
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
-- The base guards permit cancellation but do not initiate it. Cancellation is in the lifecycle owner's transaction, without changing snapshots or evidence (§13.1).
-- +goose StatementBegin
CREATE FUNCTION cancel_fleet_jobs() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.lifecycle_state IN ('revoked','retired') AND NEW.lifecycle_state IS DISTINCT FROM OLD.lifecycle_state THEN
 UPDATE update_targets t SET target_state='cancelled_'||NEW.lifecycle_state
 WHERE t.node_id=NEW.id AND t.target_state='active'
 AND (EXISTS(SELECT 1 FROM update_jobs j WHERE j.target_id=t.id AND j.state IN ('queued','waiting_online','downloading','verifying','applying','checking'))
 OR (t.target_generation=NEW.update_generation AND NOT EXISTS(SELECT 1 FROM update_jobs j WHERE j.target_id=t.id)));
 UPDATE update_jobs SET state='cancelled_'||NEW.lifecycle_state,phase_since=clock_timestamp()
 WHERE node_id=NEW.id AND state NOT IN ('succeeded','already_current','failed','rolled_back','superseded','superseded_incarnation','cancelled_revoked','cancelled_retired');
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER nodes_cancel_fleet_jobs AFTER UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION cancel_fleet_jobs();
CREATE TABLE update_retry_jobs (
 operation_id UUID NOT NULL REFERENCES operations,job_id UUID NOT NULL,
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
-- §13.8: a node's generation/attempt identifies one work item, including first-sync.
CREATE UNIQUE INDEX update_jobs_node_attempt_key ON update_jobs(node_id,generation,attempt);

-- Preserve the base job ledger's DELETE/TRUNCATE guard (including its 23514
-- contract). An inbound FK would mask that trigger with PostgreSQL 0A000 on
-- TRUNCATE. Validate these additive references on write; the base forbids removal
-- or ID changes, so validated references cannot later become orphaned.
-- +goose StatementBegin
CREATE FUNCTION require_fleet_job() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE job UUID := (to_jsonb(NEW)->>TG_ARGV[0])::uuid;
BEGIN
 IF job IS NOT NULL THEN
  PERFORM 1 FROM update_jobs WHERE id=job FOR KEY SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'unknown fleet job reference' USING ERRCODE='23503'; END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER fleet_node_job_reference AFTER INSERT OR UPDATE ON fleet_node_state FOR EACH ROW EXECUTE FUNCTION require_fleet_job('current_job_id');
CREATE CONSTRAINT TRIGGER update_result_job_reference AFTER INSERT OR UPDATE ON update_results FOR EACH ROW EXECUTE FUNCTION require_fleet_job('job_id');
CREATE CONSTRAINT TRIGGER update_retry_job_reference AFTER INSERT OR UPDATE ON update_retry_jobs FOR EACH ROW EXECUTE FUNCTION require_fleet_job('job_id');
