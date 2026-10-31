-- +goose Up
-- D-127: retain the first receipt and at most one immutable correction.
ALTER TABLE installer_update_results
 ADD COLUMN outcome_sequence SMALLINT NOT NULL DEFAULT 1 CHECK(outcome_sequence IN (1,2)),
 ADD COLUMN result_id UUID GENERATED ALWAYS AS ((payload->>'result_id')::uuid) STORED NOT NULL;
ALTER TABLE installer_update_results DROP CONSTRAINT installer_update_results_pkey;
ALTER TABLE installer_update_results ADD PRIMARY KEY(node_id,job_id,result_id);
ALTER TABLE installer_update_results ADD CONSTRAINT installer_update_results_outcome_key UNIQUE(node_id,job_id,outcome_sequence);

-- +goose Down
-- Restoring the old unique key fails atomically if corrections exist. Never
-- discard accepted evidence to downgrade; retain this schema once in use.
ALTER TABLE installer_update_results DROP CONSTRAINT installer_update_results_pkey;
ALTER TABLE installer_update_results ADD PRIMARY KEY(node_id,job_id);
ALTER TABLE installer_update_results DROP CONSTRAINT installer_update_results_outcome_key;
ALTER TABLE installer_update_results DROP COLUMN outcome_sequence, DROP COLUMN result_id;
