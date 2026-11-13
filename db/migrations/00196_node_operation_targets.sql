-- +goose Up
-- D-165: a completed capture is a durable, possibly empty set of exit pins.
CREATE TABLE node_operation_captures (
    id uuid PRIMARY KEY,
    operation_id uuid NOT NULL UNIQUE REFERENCES operations(id),
    event_id uuid NOT NULL REFERENCES outbox_events(id),
    subject_type text NOT NULL CHECK (subject_type IN ('member','device')),
    subject_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    captured_at timestamptz,
    completed_at timestamptz,
    outcomes jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(outcomes)='array')
);
COMMENT ON TABLE node_operation_captures IS 'Project Spec §4 (MEM03, API01), §9.4, §16.5; capture of the exits that a member or device operation affects, finished in its event''s receipt transaction. The operation''s nodes step completes only after its capture finishes.';
CREATE TABLE node_operation_targets (
    operation_id uuid NOT NULL REFERENCES node_operation_captures(operation_id),
    node_id uuid NOT NULL REFERENCES nodes(id),
    incarnation_id uuid NOT NULL REFERENCES node_incarnations(id),
    generation numeric(20,0) NOT NULL CHECK (generation >= 0 AND generation <= 18446744073709551615) CHECK (generation > 0),
    compiled_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    subject_present boolean NOT NULL,
    outcome text NOT NULL DEFAULT 'pending' CHECK (outcome IN ('pending','acknowledged','unconfirmed')),
    reason text,
    error_code text,
    error_message text,
    observed_at timestamptz,
    report_id uuid,
    acknowledged_incarnation_id uuid REFERENCES node_incarnations(id),
    observed_ref text,
    PRIMARY KEY (operation_id,node_id),
    CHECK (outcome<>'acknowledged' OR (observed_ref IS NOT NULL AND report_id IS NOT NULL AND acknowledged_incarnation_id IS NOT NULL)),
    CHECK (outcome<>'unconfirmed' OR reason IS NOT NULL)
);
CREATE INDEX node_operation_targets_node_idx ON node_operation_targets(node_id,operation_id);
COMMENT ON TABLE node_operation_targets IS 'Project Spec §9.4, §15.3, §16.5, §18.3; per-exit pin (node, incarnation, generation) that a member or device operation must see applied. Outcome pending, acknowledged, or unconfirmed with a reason; unconfirmed is never counted as applied.';

-- Pending removals are coalesced, not an unbounded envelope history.
CREATE TABLE node_subject_removals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id uuid NOT NULL REFERENCES nodes(id),
    subject_type text NOT NULL CHECK (subject_type IN ('member','device')),
    subject_id uuid NOT NULL,
    incarnation_id uuid NOT NULL REFERENCES node_incarnations(id),
    generation numeric(20,0) NOT NULL CHECK (generation >= 0 AND generation <= 18446744073709551615) CHECK (generation > 0),
    compiled_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (node_id,subject_type,subject_id)
);
CREATE INDEX node_subject_removals_subject_idx ON node_subject_removals(subject_type,subject_id,node_id);
COMMENT ON TABLE node_subject_removals IS 'Project Spec §9.4, §15.3, §16.5; latest outstanding removal of a member or device from an exit''s configuration, kept until that exit acknowledges it.';

-- +goose Down
DROP TABLE node_subject_removals;
DROP TABLE node_operation_targets;
DROP TABLE node_operation_captures;
