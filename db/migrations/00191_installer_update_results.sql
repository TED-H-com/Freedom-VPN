-- +goose Up
-- D-089: node evidence is independent of central jobs and sealed batch snapshots.
CREATE TABLE installer_update_results (
 node_id UUID NOT NULL REFERENCES nodes,
 job_id UUID NOT NULL,
 incarnation_id UUID NOT NULL,
 release_id UUID NOT NULL REFERENCES releases,
 state TEXT NOT NULL CHECK(state IN ('downloading','verifying','applying','checking','succeeded','already_current','failed','rolled_back','superseded')),
 payload JSONB NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_hash BYTEA NOT NULL CHECK(octet_length(payload_hash)=32),
 ack JSONB NOT NULL CHECK(jsonb_typeof(ack)='object'),
 received_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(node_id,job_id),
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id),
 CHECK(payload->>'source' IS NOT DISTINCT FROM 'installer'),
 CHECK(ack->>'accepted' IS NOT DISTINCT FROM 'true')
);
CREATE INDEX installer_update_results_latest ON installer_update_results(node_id,incarnation_id,received_at DESC,job_id DESC);
CREATE TRIGGER installer_update_results_immutable BEFORE UPDATE OR DELETE ON installer_update_results FOR EACH ROW EXECUTE FUNCTION immutable_row();
CREATE TRIGGER installer_update_results_no_truncate BEFORE TRUNCATE ON installer_update_results FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();
COMMENT ON TABLE installer_update_results IS 'Project Spec §13.8, §13.10.3, §15.4; D-089 installer-applied node evidence, never central batch results.';

-- +goose Down
DROP TRIGGER installer_update_results_immutable ON installer_update_results;
DROP TRIGGER installer_update_results_no_truncate ON installer_update_results;
DROP INDEX installer_update_results_latest;
DROP TABLE installer_update_results;
