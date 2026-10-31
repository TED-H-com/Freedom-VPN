-- +goose Up
-- D-073: durable roaming-candidate state on node_identity_endpoints so
-- multi-replica fvpn-control and process restarts evaluate interleaving
-- vs roaming deterministically inside the per-request transaction.
ALTER TABLE node_identity_endpoints
    ADD COLUMN candidate_hash BYTEA CHECK (octet_length(candidate_hash) = 32),
    ADD COLUMN candidate_first_seen_at TIMESTAMPTZ,
    ADD COLUMN candidate_last_seen_at TIMESTAMPTZ,
    ADD CONSTRAINT node_identity_endpoints_candidate_check CHECK (
        (candidate_hash IS NULL) = (candidate_first_seen_at IS NULL)
        AND (candidate_hash IS NULL) = (candidate_last_seen_at IS NULL)
    );

-- D-075: move the r14 M-2 one-shot concurrent-advance readiness grace mark
-- out of audit_events onto node_incarnations.
ALTER TABLE node_incarnations
    ADD COLUMN stale_generation_grace_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE node_incarnations
    DROP COLUMN IF EXISTS stale_generation_grace_at;

ALTER TABLE node_identity_endpoints
    DROP CONSTRAINT IF EXISTS node_identity_endpoints_candidate_check,
    DROP COLUMN IF EXISTS candidate_last_seen_at,
    DROP COLUMN IF EXISTS candidate_first_seen_at,
    DROP COLUMN IF EXISTS candidate_hash;
