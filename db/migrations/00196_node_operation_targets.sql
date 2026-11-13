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
CREATE TABLE node_operation_targets (
    operation_id uuid NOT NULL REFERENCES node_operation_captures(operation_id),
    node_id uuid NOT NULL REFERENCES nodes(id),
    incarnation_id uuid NOT NULL REFERENCES node_incarnations(id),
    generation numeric(20,0) NOT NULL CHECK (generation>0 AND generation<=18446744073709551615),
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

-- Pending removals are coalesced, not an unbounded envelope history.
CREATE TABLE node_subject_removals (
    node_id uuid NOT NULL REFERENCES nodes(id),
    subject_type text NOT NULL CHECK (subject_type IN ('member','device')),
    subject_id uuid NOT NULL,
    incarnation_id uuid NOT NULL REFERENCES node_incarnations(id),
    generation numeric(20,0) NOT NULL CHECK (generation>0 AND generation<=18446744073709551615),
    compiled_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (node_id,subject_type,subject_id)
);
CREATE INDEX node_subject_removals_subject_idx ON node_subject_removals(subject_type,subject_id,node_id);

-- +goose Down
DROP TABLE node_subject_removals;
DROP TABLE node_operation_targets;
DROP TABLE node_operation_captures;
