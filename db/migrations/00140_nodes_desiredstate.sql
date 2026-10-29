-- +goose Up
-- +goose StatementBegin

ALTER TABLE nodes
    ADD COLUMN capacity JSONB CHECK (capacity IS NULL OR jsonb_typeof(capacity) = 'object'),
    ADD COLUMN control_status TEXT NOT NULL DEFAULT 'offline' CHECK (control_status IN ('online', 'stale', 'offline')),
    ADD COLUMN data_health TEXT NOT NULL DEFAULT 'unknown' CHECK (data_health IN ('unknown', 'healthy', 'degraded', 'failed')),
    ADD COLUMN safety_status TEXT NOT NULL DEFAULT 'unknown' CHECK (safety_status IN ('unknown', 'passed', 'failed', 'expired')),
    ADD COLUMN member_forwarding_observed BOOLEAN,
    ADD COLUMN last_seen_at TIMESTAMPTZ,
    ADD COLUMN last_observed_at TIMESTAMPTZ,
    ADD COLUMN observed_exit_ipv4 TEXT,
    ADD COLUMN observed_exit_ipv6 TEXT,
    ADD COLUMN latest_checks JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(latest_checks) = 'array'),
    ADD COLUMN recent_errors JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(recent_errors) = 'array'),
    ADD COLUMN drift JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(drift) = 'array');

CREATE INDEX nodes_control_status_idx ON nodes (control_status, last_seen_at);

ALTER TABLE node_incarnations
    ADD COLUMN os_version TEXT NOT NULL DEFAULT '24.04' CHECK (btrim(os_version) <> '');

ALTER TABLE node_observations
    ADD COLUMN member_forwarding_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN provisioning_state TEXT CHECK (provisioning_state IS NULL OR provisioning_state IN ('pending', 'applying', 'ready', 'failed')),
    ADD COLUMN report_payload JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(report_payload) = 'object');

CREATE TABLE desired_state_envelopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    desired_state_id UUID NOT NULL REFERENCES desired_states(id),
    node_id UUID NOT NULL REFERENCES nodes(id),
    incarnation_id UUID NOT NULL REFERENCES node_incarnations(id),
    generation NUMERIC(20, 0) NOT NULL CHECK (generation >= 0 AND generation <= 18446744073709551615),
    etag TEXT NOT NULL CHECK (btrim(etag) <> ''),
    envelope JSONB NOT NULL CHECK (jsonb_typeof(envelope) = 'object'),
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT desired_state_envelopes_node_incarnation_key UNIQUE (node_id, incarnation_id),
    CONSTRAINT desired_state_envelopes_window_check CHECK (
        isfinite(issued_at)
        AND isfinite(expires_at)
        AND expires_at > issued_at
    )
);
COMMENT ON TABLE desired_state_envelopes IS 'Latest signed desired-state envelope and ETag per node incarnation (§5, §14.4, §14.7).';

CREATE TRIGGER desired_state_envelopes_touch_updated_at
    BEFORE UPDATE ON desired_state_envelopes
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE serving_leases
    ADD COLUMN renewal_request_id UUID,
    ADD COLUMN authorization_generation NUMERIC(20, 0) NOT NULL DEFAULT 0 CHECK (authorization_generation >= 0 AND authorization_generation <= 18446744073709551615),
    ADD COLUMN envelope JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(envelope) = 'object');

CREATE UNIQUE INDEX serving_leases_renewal_request_key
    ON serving_leases (node_id, incarnation_id, renewal_request_id)
    WHERE renewal_request_id IS NOT NULL;

ALTER TABLE operations
    ADD COLUMN error_code TEXT,
    ADD COLUMN error_message TEXT;

CREATE TRIGGER operations_touch_updated_at
    BEFORE UPDATE ON operations
    FOR EACH ROW
    WHEN (OLD.status NOT IN ('succeeded', 'superseded'))
    EXECUTE FUNCTION touch_updated_at();

ALTER TABLE external_bindings
    ADD COLUMN observed_hash BYTEA CHECK (observed_hash IS NULL OR octet_length(observed_hash) = 32),
    ADD COLUMN sync_state TEXT NOT NULL DEFAULT 'pending' CHECK (sync_state IN ('pending', 'synced', 'in_sync', 'drifted', 'error')),
    ADD COLUMN last_verified_at TIMESTAMPTZ,
    ADD COLUMN last_error TEXT;

-- +goose StatementEnd
