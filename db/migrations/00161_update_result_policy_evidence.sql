-- +goose Up
-- §13.10.3 / contract revision 3: the applied config generation is distinct
-- from the job generation. Keep it with the result's matching immutable policy
-- references, never the latest desired state's references. NULL refs mean that
-- this central database has no matching desired-state history (e.g. restore).
ALTER TABLE update_results ADD COLUMN policy_generation NUMERIC(20,0)
 GENERATED ALWAYS AS ((payload->>'policy_generation')::numeric) STORED
 CHECK(policy_generation BETWEEN 0 AND 18446744073709551615);
ALTER TABLE update_results ADD COLUMN policy_refs UUID[];
COMMENT ON COLUMN update_results.policy_generation IS 'Applied desired-state config_generation reported by the node; derived exactly from immutable result payload.';
COMMENT ON COLUMN update_results.policy_refs IS 'References from desired_states for this result node, incarnation and applied policy_generation; NULL when unavailable, never guessed from current desired state.';
