-- +goose Up
-- +goose StatementBegin

ALTER TABLE policy_definitions
    ADD COLUMN generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL DEFAULT 0,
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TRIGGER policy_definitions_monotonic_generation
    BEFORE UPDATE ON policy_definitions
    FOR EACH ROW EXECUTE FUNCTION monotonic_fields('generation');

CREATE TRIGGER policy_definitions_touch_updated_at
    BEFORE UPDATE ON policy_definitions
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE policy_revisions
    ADD COLUMN reason TEXT CHECK (length(reason) <= 500) NOT NULL DEFAULT '';

CREATE TABLE dns_compiled_bundles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_revision_id UUID NOT NULL REFERENCES policy_revisions(id),
    profile_code TEXT CHECK (profile_code ~ '[^[:space:]]') NOT NULL,
    bundle_revision NUMERIC(20,0) CHECK (bundle_revision >= 0 AND bundle_revision <= 18446744073709551615) NOT NULL,
    bundle_sha256 TEXT CHECK (bundle_sha256 ~ '^[0-9a-f]{64}$') NOT NULL,
    valid_from TIMESTAMPTZ CHECK (isfinite(valid_from)) NOT NULL,
    valid_until TIMESTAMPTZ CHECK (isfinite(valid_until)) NOT NULL,
    bundle JSONB CHECK (jsonb_typeof(bundle) = 'object') NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dns_compiled_bundles_profile_revision_key UNIQUE (profile_revision_id),
    CONSTRAINT dns_compiled_bundles_valid_window_check CHECK (valid_until > valid_from)
);
COMMENT ON TABLE dns_compiled_bundles IS 'Immutable compiled DNS policy bundles produced on dns_profile activation (§12.4, §12.5, §14.2).';

CREATE TRIGGER dns_compiled_bundles_immutable
    BEFORE UPDATE OR DELETE ON dns_compiled_bundles
    FOR EACH ROW EXECUTE FUNCTION immutable_row();

CREATE TRIGGER dns_compiled_bundles_no_truncate
    BEFORE TRUNCATE ON dns_compiled_bundles
    FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();

CREATE TABLE fup_member_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members(id),
    down_bps BIGINT CHECK (down_bps >= 0 AND down_bps <= 9223372036854775807) NOT NULL,
    up_bps BIGINT CHECK (up_bps >= 0 AND up_bps <= 9223372036854775807) NOT NULL,
    priority INTEGER CHECK (priority BETWEEN 0 AND 7) NOT NULL,
    valid_from TIMESTAMPTZ CHECK (isfinite(valid_from)) NOT NULL DEFAULT now(),
    valid_until TIMESTAMPTZ CHECK (isfinite(valid_until)) NOT NULL,
    reason TEXT CHECK (reason ~ '[^[:space:]]' AND length(reason) <= 500) NOT NULL,
    actor_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ CHECK (revoked_at IS NULL OR isfinite(revoked_at)),
    revoked_by UUID,
    revocation_reason TEXT,
    CONSTRAINT fup_member_overrides_valid_window_check CHECK (valid_until > valid_from),
    CONSTRAINT fup_member_overrides_revocation_check CHECK (
        (revoked_at IS NULL AND revoked_by IS NULL AND revocation_reason IS NULL)
        OR (revoked_at IS NOT NULL AND revoked_by IS NOT NULL AND revocation_reason IS NOT NULL AND revocation_reason ~ '[^[:space:]]')
    )
);
COMMENT ON TABLE fup_member_overrides IS 'Time-limited audited fair-use member rate overrides (§11.1, §11.6, FUP04).';

CREATE INDEX fup_member_overrides_member_active_idx
    ON fup_member_overrides (member_id, valid_until DESC)
    WHERE revoked_at IS NULL;

CREATE TRIGGER fup_member_overrides_keep
    BEFORE DELETE OR TRUNCATE ON fup_member_overrides
    FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();

CREATE TRIGGER fup_member_overrides_identity
    BEFORE UPDATE ON fup_member_overrides
    FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','down_bps','up_bps','priority','valid_from','valid_until','reason','actor_id','created_at');

CREATE TRIGGER fup_member_overrides_revoked
    BEFORE UPDATE ON fup_member_overrides
    FOR EACH ROW EXECUTE FUNCTION irreversible_timestamps('revoked_at','revoked_by','revocation_reason');

CREATE TABLE fup_member_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members(id),
    period_id TEXT CHECK (period_id ~ '^[0-9]{4}-(0[1-9]|1[0-2])$') NOT NULL,
    period_start TIMESTAMPTZ CHECK (isfinite(period_start)) NOT NULL,
    period_end TIMESTAMPTZ CHECK (isfinite(period_end)) NOT NULL,
    policy_revision_id UUID NOT NULL REFERENCES policy_revisions(id),
    mode TEXT CHECK (mode IN ('shadow', 'enforce')) NOT NULL,
    would_apply_tier_index INTEGER CHECK (would_apply_tier_index BETWEEN 0 AND 15) NOT NULL,
    would_apply_tier_name JSONB CHECK (jsonb_typeof(would_apply_tier_name) = 'object') NOT NULL DEFAULT '{}'::jsonb,
    would_apply_down_bps BIGINT CHECK (would_apply_down_bps >= 0 AND would_apply_down_bps <= 9223372036854775807) NOT NULL,
    would_apply_up_bps BIGINT CHECK (would_apply_up_bps >= 0 AND would_apply_up_bps <= 9223372036854775807) NOT NULL,
    would_apply_priority INTEGER CHECK (would_apply_priority BETWEEN 0 AND 7) NOT NULL,
    would_apply_reason TEXT CHECK (would_apply_reason IN ('normal', 'fair_use_tier', 'admin_override')) NOT NULL,
    enforced_state TEXT CHECK (enforced_state IN ('normal', 'adjusted', 'override')) NOT NULL DEFAULT 'normal',
    enforced_tier_index INTEGER CHECK (enforced_tier_index BETWEEN 0 AND 15) NOT NULL DEFAULT 0,
    enforced_tier_name JSONB CHECK (jsonb_typeof(enforced_tier_name) = 'object') NOT NULL DEFAULT '{}'::jsonb,
    enforced_down_bps BIGINT CHECK (enforced_down_bps IS NULL OR (enforced_down_bps >= 0 AND enforced_down_bps <= 9223372036854775807)),
    enforced_up_bps BIGINT CHECK (enforced_up_bps IS NULL OR (enforced_up_bps >= 0 AND enforced_up_bps <= 9223372036854775807)),
    enforced_priority INTEGER CHECK (enforced_priority IS NULL OR enforced_priority BETWEEN 0 AND 7),
    enforced_reason TEXT CHECK (enforced_reason IS NULL OR enforced_reason IN ('normal', 'fair_use_tier', 'admin_override')),
    generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL DEFAULT 0,
    total_bytes NUMERIC(20,0) CHECK (total_bytes >= 0 AND total_bytes <= 18446744073709551615) NOT NULL DEFAULT 0,
    aggregate_as_of TIMESTAMPTZ CHECK (isfinite(aggregate_as_of)) NOT NULL,
    override_id UUID REFERENCES fup_member_overrides(id),
    override_valid_until TIMESTAMPTZ CHECK (override_valid_until IS NULL OR isfinite(override_valid_until)),
    impact_summary JSONB CHECK (jsonb_typeof(impact_summary) = 'object') NOT NULL DEFAULT '{}'::jsonb,
    last_computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fup_member_states_member_period_key UNIQUE (member_id, period_id),
    CONSTRAINT fup_member_states_period_window_check CHECK (period_end > period_start)
);
COMMENT ON TABLE fup_member_states IS 'Per-member, per-period fair-use evaluation state, shadow impact and enforced generation (§11.1, §11.2, §11.5).';

CREATE TRIGGER fup_member_states_keep
    BEFORE DELETE OR TRUNCATE ON fup_member_states
    FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();

CREATE TRIGGER fup_member_states_identity
    BEFORE UPDATE ON fup_member_states
    FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','period_id','period_start','period_end','created_at');

CREATE TRIGGER fup_member_states_monotonic_generation
    BEFORE UPDATE ON fup_member_states
    FOR EACH ROW EXECUTE FUNCTION monotonic_fields('generation');

CREATE TRIGGER fup_member_states_touch_updated_at
    BEFORE UPDATE ON fup_member_states
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE fup_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members(id),
    period_id TEXT CHECK (period_id ~ '^[0-9]{4}-(0[1-9]|1[0-2])$') NOT NULL,
    policy_revision_id UUID NOT NULL REFERENCES policy_revisions(id),
    mode TEXT CHECK (mode IN ('shadow', 'enforce')) NOT NULL,
    would_apply_tier_index INTEGER CHECK (would_apply_tier_index BETWEEN 0 AND 15) NOT NULL,
    would_apply_down_bps BIGINT CHECK (would_apply_down_bps >= 0 AND would_apply_down_bps <= 9223372036854775807) NOT NULL,
    would_apply_up_bps BIGINT CHECK (would_apply_up_bps >= 0 AND would_apply_up_bps <= 9223372036854775807) NOT NULL,
    would_apply_priority INTEGER CHECK (would_apply_priority BETWEEN 0 AND 7) NOT NULL,
    would_apply_reason TEXT CHECK (would_apply_reason IN ('normal', 'fair_use_tier', 'admin_override')) NOT NULL,
    enforced_state TEXT CHECK (enforced_state IN ('normal', 'adjusted', 'override')) NOT NULL,
    enforced_tier_index INTEGER CHECK (enforced_tier_index BETWEEN 0 AND 15) NOT NULL,
    enforced_down_bps BIGINT CHECK (enforced_down_bps IS NULL OR (enforced_down_bps >= 0 AND enforced_down_bps <= 9223372036854775807)),
    enforced_up_bps BIGINT CHECK (enforced_up_bps IS NULL OR (enforced_up_bps >= 0 AND enforced_up_bps <= 9223372036854775807)),
    enforced_priority INTEGER CHECK (enforced_priority IS NULL OR enforced_priority BETWEEN 0 AND 7),
    enforced_reason TEXT CHECK (enforced_reason IS NULL OR enforced_reason IN ('normal', 'fair_use_tier', 'admin_override')),
    generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL,
    generation_incremented BOOLEAN NOT NULL DEFAULT false,
    relaxed BOOLEAN NOT NULL DEFAULT false,
    relaxation_reason TEXT,
    total_bytes NUMERIC(20,0) CHECK (total_bytes >= 0 AND total_bytes <= 18446744073709551615) NOT NULL,
    aggregate_as_of TIMESTAMPTZ CHECK (isfinite(aggregate_as_of)) NOT NULL,
    impact_summary JSONB CHECK (jsonb_typeof(impact_summary) = 'object') NOT NULL DEFAULT '{}'::jsonb,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON TABLE fup_evaluations IS 'Append-only fair-use evaluation log for shadow mode and enforced policy generations (§11.1, §11.2, §11.5).';

CREATE INDEX fup_evaluations_member_period_idx
    ON fup_evaluations (member_id, period_id, computed_at DESC);

CREATE TRIGGER fup_evaluations_no_row_change
    BEFORE UPDATE OR DELETE ON fup_evaluations
    FOR EACH ROW EXECUTE FUNCTION guard_append_only();

CREATE TRIGGER fup_evaluations_no_truncate
    BEFORE TRUNCATE ON fup_evaluations
    FOR EACH STATEMENT EXECUTE FUNCTION guard_append_only();

-- +goose StatementEnd
