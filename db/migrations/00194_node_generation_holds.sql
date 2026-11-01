-- +goose Up
-- r18 M-3 / §5.2, §15.3: diagnostic incidents outlive agent process streaks.
-- No serving authorization and no reuse of the D-075 readiness grace mark.
CREATE TABLE node_generation_holds (
 id UUID PRIMARY KEY,
 node_id UUID NOT NULL,
 incarnation_id UUID NOT NULL,
 first_seen_at TIMESTAMPTZ NOT NULL,
 last_seen_at TIMESTAMPTZ NOT NULL,
 first_observed_at TIMESTAMPTZ NOT NULL,
 latest_report_id UUID NOT NULL,
 escalated_at TIMESTAMPTZ,
 ended_at TIMESTAMPTZ,
 end_reason TEXT CHECK (end_reason IN ('generation_matched','identity_inactive','identity_frozen','members_revoked')),
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id),
 FOREIGN KEY(node_id,incarnation_id,latest_report_id) REFERENCES node_observations(node_id,incarnation_id,report_id),
 UNIQUE(id,node_id,incarnation_id),
 CHECK (isfinite(first_seen_at) AND isfinite(last_seen_at) AND last_seen_at >= first_seen_at),
 CHECK ((ended_at IS NULL) = (end_reason IS NULL))
);
CREATE UNIQUE INDEX node_generation_holds_active ON node_generation_holds(node_id,incarnation_id) WHERE ended_at IS NULL;
COMMENT ON TABLE node_generation_holds IS 'Project Spec §5.1, §5.2, §15.3, §16.3; retained serving-lease generation hold incidents and escalation; see docs/nodes/followup34-generation-hold-2026-11-01.md.';

-- Each sample retains the local counters, including their reset on process
-- restart. Raw summaries stay in the immutable observation, never in audit text.
CREATE TABLE node_generation_hold_reports (
 hold_id UUID NOT NULL,
 node_id UUID NOT NULL,
 incarnation_id UUID NOT NULL,
 report_id UUID NOT NULL,
 boot_id UUID NOT NULL,
 lease_id UUID NOT NULL REFERENCES serving_leases(id),
 applied_generation NUMERIC(20,0) CHECK (applied_generation BETWEEN 0 AND 18446744073709551615),
 lease_generation NUMERIC(20,0) NOT NULL CHECK (lease_generation BETWEEN 0 AND 18446744073709551615),
 reported_steps NUMERIC(20,0) CHECK (reported_steps BETWEEN 0 AND 18446744073709551615),
 reported_elapsed_ms NUMERIC(20,0) CHECK (reported_elapsed_ms BETWEEN 0 AND 18446744073709551615),
 reported_alarm BOOLEAN,
 summary_valid BOOLEAN NOT NULL,
 PRIMARY KEY(node_id,incarnation_id,report_id),
 FOREIGN KEY(hold_id,node_id,incarnation_id) REFERENCES node_generation_holds(id,node_id,incarnation_id),
 FOREIGN KEY(node_id,incarnation_id,report_id) REFERENCES node_observations(node_id,incarnation_id,report_id),
 CHECK ((reported_steps IS NULL) = (reported_elapsed_ms IS NULL) AND (reported_steps IS NULL) = (reported_alarm IS NULL)),
 CHECK (NOT summary_valid OR reported_steps IS NOT NULL)
);
CREATE INDEX node_generation_hold_reports_history ON node_generation_hold_reports(hold_id);
COMMENT ON TABLE node_generation_hold_reports IS 'Project Spec §5.1, §5.2, §15.3, §16.3; per-report generation hold and boot evidence; see docs/nodes/followup34-generation-hold-2026-11-01.md.';

-- +goose Down
DROP TABLE node_generation_hold_reports;
DROP TABLE node_generation_holds;
