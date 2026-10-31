-- +goose Up
-- SECOPS-19 / D-075 / D-077 §2: persist the latest accepted ObservationReport.usage
-- status and its central receipt timestamp on node_incarnations so GET /admin/nodes/{id}
-- projects NodeDetail.usage_outbox directly from the incarnation row (resetting to null
-- on re-enrolment) without reading node_observations.report_payload or audit_events.
ALTER TABLE node_incarnations
    ADD COLUMN usage_outbox JSONB,
    ADD COLUMN usage_outbox_observed_at TIMESTAMPTZ,
    ADD CONSTRAINT node_incarnations_usage_outbox_check CHECK (
        (usage_outbox IS NULL) = (usage_outbox_observed_at IS NULL)
        AND (usage_outbox IS NULL OR jsonb_typeof(usage_outbox) = 'object')
    );

-- +goose Down
ALTER TABLE node_incarnations
    DROP CONSTRAINT IF EXISTS node_incarnations_usage_outbox_check,
    DROP COLUMN IF EXISTS usage_outbox_observed_at,
    DROP COLUMN IF EXISTS usage_outbox;
