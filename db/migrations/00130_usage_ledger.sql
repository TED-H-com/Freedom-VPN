-- +goose Up
-- Central usage ledger extensions owned by internal/usage (Project Spec §10.6,
-- §10.8, §10.9, §15.4, §17.3). Additive and forward-only: existing columns,
-- constraints and triggers of 00001-00005 are not changed (§15.5 last
-- paragraph). See docs/usage/README.md for the algorithms that use them.

-- §17.3 step 2: ledger_epoch is the central data generation. The genesis
-- epoch has no recovery record; every later epoch is created by a signed
-- recovery record and names its predecessor.
CREATE TABLE usage_ledger_epochs (
 id UUID PRIMARY KEY,
 previous_epoch UUID UNIQUE REFERENCES usage_ledger_epochs,
 recovery_record_id UUID UNIQUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK((previous_epoch IS NULL) = (recovery_record_id IS NULL))
);
CREATE UNIQUE INDEX usage_ledger_epochs_genesis_key ON usage_ledger_epochs((true)) WHERE previous_epoch IS NULL;

CREATE TABLE usage_recovery_records (
 id UUID PRIMARY KEY,
 ledger_epoch UUID NOT NULL UNIQUE,
 previous_epoch UUID NOT NULL REFERENCES usage_ledger_epochs,
 restored_to TIMESTAMPTZ NOT NULL CHECK(isfinite(restored_to)),
 affected_since TIMESTAMPTZ NOT NULL CHECK(isfinite(affected_since)),
 reason TEXT NOT NULL CHECK(length(reason) > 0),
 actor TEXT NOT NULL CHECK(length(actor) > 0),
 -- Canonical JSON of the recovery-purpose envelope (pkg/envelope).
 signed_envelope BYTEA NOT NULL,
 signing_key_id TEXT NOT NULL CHECK(length(signing_key_id) > 0),
 record_sha256 BYTEA NOT NULL CHECK(octet_length(record_sha256) = 32),
 stream_count INTEGER NOT NULL CHECK(stream_count >= 0),
 status TEXT NOT NULL DEFAULT 'reconciling' CHECK(status IN ('reconciling','completed')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 completed_at TIMESTAMPTZ, completed_by TEXT,
 CHECK((status = 'completed') = (completed_at IS NOT NULL AND completed_by IS NOT NULL)),
 CHECK(affected_since <= restored_to)
);
ALTER TABLE usage_ledger_epochs ADD CONSTRAINT usage_ledger_epochs_recovery_fk
 FOREIGN KEY(recovery_record_id) REFERENCES usage_recovery_records;
ALTER TABLE usage_recovery_records ADD CONSTRAINT usage_recovery_records_epoch_fk
 FOREIGN KEY(ledger_epoch) REFERENCES usage_ledger_epochs DEFERRABLE INITIALLY DEFERRED;

-- Singleton: the current epoch and whether ingest is frozen (§17.3 step 1).
CREATE TABLE usage_ledger_state (
 id UUID PRIMARY KEY CHECK(id = '00000000-0000-0000-0000-000000000130'),
 current_epoch UUID NOT NULL UNIQUE REFERENCES usage_ledger_epochs,
 mode TEXT NOT NULL DEFAULT 'normal' CHECK(mode IN ('normal','frozen','reconciling')),
 mode_reason TEXT NOT NULL DEFAULT '',
 changed_by TEXT NOT NULL DEFAULT 'migration',
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementBegin
DO $$
DECLARE genesis UUID := gen_random_uuid();
BEGIN
 INSERT INTO usage_ledger_epochs(id) VALUES(genesis);
 INSERT INTO usage_ledger_state(id, current_epoch) VALUES('00000000-0000-0000-0000-000000000130', genesis);
END $$;
-- +goose StatementEnd
CREATE TRIGGER usage_ledger_state_keep BEFORE DELETE OR TRUNCATE ON usage_ledger_state FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER usage_ledger_state_updated BEFORE UPDATE ON usage_ledger_state FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER usage_ledger_epochs_immutable BEFORE UPDATE OR DELETE ON usage_ledger_epochs FOR EACH ROW EXECUTE FUNCTION immutable_row();
CREATE TRIGGER usage_ledger_epochs_no_truncate BEFORE TRUNCATE ON usage_ledger_epochs FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();
CREATE TRIGGER usage_recovery_records_identity BEFORE UPDATE ON usage_recovery_records FOR EACH ROW
 EXECUTE FUNCTION immutable_fields('id','ledger_epoch','previous_epoch','restored_to','affected_since','reason','actor','signed_envelope','signing_key_id','record_sha256','stream_count','created_at');
-- +goose StatementBegin
CREATE FUNCTION guard_usage_recovery_status() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.status = 'completed' AND (NEW.status, NEW.completed_at, NEW.completed_by) IS DISTINCT FROM (OLD.status, OLD.completed_at, OLD.completed_by) THEN
  RAISE EXCEPTION 'completed recovery record is immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER usage_recovery_records_status BEFORE UPDATE ON usage_recovery_records FOR EACH ROW EXECUTE FUNCTION guard_usage_recovery_status();
CREATE TRIGGER usage_recovery_records_keep BEFORE DELETE OR TRUNCATE ON usage_recovery_records FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();

-- §10.5, §10.9, §10.8 step 4: queryable event content next to the receipt.
-- kind = replay_unavailable marks a sequence the node can no longer replay;
-- it carries no records and therefore never invents bytes.
ALTER TABLE usage_events
 ADD COLUMN kind TEXT NOT NULL DEFAULT 'event' CHECK(kind IN ('event','replay_unavailable')),
 ADD COLUMN meter_epoch_id UUID,
 ADD COLUMN metering_version TEXT,
 ADD COLUMN quality TEXT CHECK(quality IN ('complete','gap','boundary_estimated')),
 ADD COLUMN interval_start TIMESTAMPTZ,
 ADD COLUMN interval_end TIMESTAMPTZ,
 ADD COLUMN ledger_epoch UUID REFERENCES usage_ledger_epochs,
 ADD CONSTRAINT usage_events_interval_check CHECK(
  (interval_start IS NULL AND interval_end IS NULL)
  OR (interval_start IS NOT NULL AND interval_end IS NOT NULL AND isfinite(interval_start) AND isfinite(interval_end) AND interval_start < interval_end)),
 ADD CONSTRAINT usage_events_tombstone_check CHECK(kind <> 'replay_unavailable' OR record_count = 0);

-- §10.6: same key, different hash. The stored event stays authoritative; the
-- other content is kept here, never written over the ledger.
CREATE TABLE usage_event_conflicts (
 id UUID PRIMARY KEY,
 node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 event_seq NUMERIC(20,0) NOT NULL CHECK(event_seq >= 0 AND event_seq <= 18446744073709551615) CHECK(event_seq > 0),
 existing_event_id UUID NOT NULL REFERENCES usage_events,
 existing_payload_hash BYTEA NOT NULL CHECK(octet_length(existing_payload_hash) = 32),
 conflicting_payload_hash BYTEA NOT NULL CHECK(octet_length(conflicting_payload_hash) = 32),
 conflicting_payload BYTEA NOT NULL,
 ledger_epoch UUID NOT NULL REFERENCES usage_ledger_epochs,
 first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(), last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 occurrences NUMERIC(20,0) NOT NULL DEFAULT 1 CHECK(occurrences >= 0 AND occurrences <= 18446744073709551615) CHECK(occurrences >= 1),
 FOREIGN KEY(node_id, incarnation_id) REFERENCES node_incarnations(node_id, id),
 CONSTRAINT usage_event_conflicts_content_key UNIQUE(node_id, incarnation_id, event_seq, conflicting_payload_hash),
 CHECK(existing_payload_hash <> conflicting_payload_hash)
);
CREATE TRIGGER usage_event_conflicts_identity BEFORE UPDATE ON usage_event_conflicts FOR EACH ROW
 EXECUTE FUNCTION immutable_fields('id','node_id','incarnation_id','event_seq','existing_event_id','existing_payload_hash','conflicting_payload_hash','conflicting_payload','ledger_epoch','first_seen_at');
CREATE TRIGGER usage_event_conflicts_keep BEFORE DELETE OR TRUNCATE ON usage_event_conflicts FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();

-- §10.6, §10.9: rejected events (format, integer, mapping or clock anomaly)
-- are kept for reconciliation and never counted.
CREATE TABLE usage_event_rejections (
 id UUID PRIMARY KEY,
 node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 event_seq NUMERIC(20,0) NOT NULL CHECK(event_seq >= 0 AND event_seq <= 18446744073709551615),
 content_sha256 BYTEA NOT NULL CHECK(octet_length(content_sha256) = 32),
 error_code TEXT NOT NULL,
 fields JSONB NOT NULL CHECK(jsonb_typeof(fields) = 'array'),
 payload BYTEA NOT NULL,
 ledger_epoch UUID NOT NULL REFERENCES usage_ledger_epochs,
 first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(), last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 occurrences NUMERIC(20,0) NOT NULL DEFAULT 1 CHECK(occurrences >= 0 AND occurrences <= 18446744073709551615) CHECK(occurrences >= 1),
 FOREIGN KEY(node_id, incarnation_id) REFERENCES node_incarnations(node_id, id),
 CONSTRAINT usage_event_rejections_content_key UNIQUE(node_id, incarnation_id, event_seq, content_sha256)
);
CREATE TRIGGER usage_event_rejections_identity BEFORE UPDATE ON usage_event_rejections FOR EACH ROW
 EXECUTE FUNCTION immutable_fields('id','node_id','incarnation_id','event_seq','content_sha256','payload','first_seen_at');
CREATE TRIGGER usage_event_rejections_keep BEFORE DELETE OR TRUNCATE ON usage_event_rejections FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();

-- §10.7, §10.8 step 4, §17.3 step 9: known data gaps of a stream with the
-- affected time window and months. Monthly totals never overwrite the ledger.
CREATE TABLE usage_ledger_gaps (
 id UUID PRIMARY KEY,
 node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('replay_unavailable','node_reported')),
 start_seq NUMERIC(20,0) NOT NULL CHECK(start_seq >= 0 AND start_seq <= 18446744073709551615) CHECK(start_seq > 0),
 end_seq NUMERIC(20,0) NOT NULL CHECK(end_seq >= 0 AND end_seq <= 18446744073709551615) CHECK(end_seq > 0),
 reason TEXT NOT NULL CHECK(length(reason) > 0),
 ledger_epoch UUID NOT NULL REFERENCES usage_ledger_epochs,
 affected_from TIMESTAMPTZ NOT NULL, affected_until TIMESTAMPTZ NOT NULL,
 affected_periods TEXT[] NOT NULL CHECK(cardinality(affected_periods) > 0),
 source_event_id UUID REFERENCES usage_events,
 recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(node_id, incarnation_id) REFERENCES node_incarnations(node_id, id),
 CONSTRAINT usage_ledger_gaps_range_key UNIQUE(node_id, incarnation_id, kind, start_seq),
 CHECK(start_seq <= end_seq),
 CHECK(isfinite(affected_from) AND isfinite(affected_until) AND affected_from < affected_until)
);
CREATE INDEX usage_ledger_gaps_window_idx ON usage_ledger_gaps USING gist(tstzrange(affected_from, affected_until, '[)'));
CREATE TRIGGER usage_ledger_gaps_append_only BEFORE UPDATE OR DELETE ON usage_ledger_gaps FOR EACH ROW EXECUTE FUNCTION guard_append_only();
CREATE TRIGGER usage_ledger_gaps_no_truncate BEFORE TRUNCATE ON usage_ledger_gaps FOR EACH STATEMENT EXECUTE FUNCTION guard_append_only();

-- §15.4, §15.4 last paragraphs: projection generations. Exactly one active
-- generation is the read pointer; at most one generation is being rebuilt.
CREATE TABLE usage_projection_generations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 projection_name TEXT NOT NULL CHECK(length(projection_name) > 0),
 generation NUMERIC(20,0) NOT NULL CHECK(generation >= 0 AND generation <= 18446744073709551615) CHECK(generation > 0),
 state TEXT NOT NULL CHECK(state IN ('building','active','superseded')),
 started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 activated_at TIMESTAMPTZ, superseded_at TIMESTAMPTZ,
 verification JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(verification) = 'object'),
 CONSTRAINT usage_projection_generations_key UNIQUE(projection_name, generation),
 CHECK(state = 'building' OR activated_at IS NOT NULL),
 CHECK(state <> 'superseded' OR superseded_at IS NOT NULL)
);
CREATE UNIQUE INDEX usage_projection_generations_active_key ON usage_projection_generations(projection_name) WHERE state = 'active';
CREATE UNIQUE INDEX usage_projection_generations_building_key ON usage_projection_generations(projection_name) WHERE state = 'building';
INSERT INTO usage_projection_generations(projection_name, generation, state, activated_at) VALUES('member_usage', 1, 'active', now());
CREATE TRIGGER usage_projection_generations_keep BEFORE DELETE OR TRUNCATE ON usage_projection_generations FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER usage_projection_generations_identity BEFORE UPDATE ON usage_projection_generations FOR EACH ROW
 EXECUTE FUNCTION immutable_fields('id','projection_name','generation','started_at');

-- §10.9: drill-down by node incarnation and device, incremented under the
-- same projection receipt as usage_rollups (never a second count source).
CREATE TABLE usage_rollup_details (
 id UUID PRIMARY KEY,
 projection_generation NUMERIC(20,0) NOT NULL CHECK(projection_generation >= 0 AND projection_generation <= 18446744073709551615) CHECK(projection_generation > 0),
 granularity TEXT NOT NULL CHECK(granularity IN ('month','day')),
 period TEXT NOT NULL,
 member_id UUID NOT NULL REFERENCES members,
 node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 device_id UUID NOT NULL,
 metering_version TEXT NOT NULL,
 upload_bytes NUMERIC(40,0) NOT NULL CHECK(upload_bytes >= 0),
 download_bytes NUMERIC(40,0) NOT NULL CHECK(download_bytes >= 0),
 total_bytes NUMERIC(41,0) GENERATED ALWAYS AS (upload_bytes + download_bytes) STORED,
 last_observed_at TIMESTAMPTZ NOT NULL,
 period_start TIMESTAMPTZ NOT NULL, period_end TIMESTAMPTZ NOT NULL,
 estimated BOOLEAN NOT NULL DEFAULT false, gap BOOLEAN NOT NULL DEFAULT false,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(device_id, member_id) REFERENCES member_devices(id, member_id),
 FOREIGN KEY(node_id, incarnation_id) REFERENCES node_incarnations(node_id, id),
 CONSTRAINT usage_rollup_details_key UNIQUE(projection_generation, granularity, period, member_id, node_id, incarnation_id, device_id),
 CHECK(isfinite(period_start) AND isfinite(period_end) AND period_start < period_end)
);
CREATE INDEX usage_rollup_details_member_idx ON usage_rollup_details(member_id, projection_generation, granularity, period_start);
CREATE INDEX usage_rollups_member_window_idx ON usage_rollups(member_id, projection_generation, granularity, period_start);

-- §15.4 usage_adjustments: append-only corrections, idempotent by caller ID.
CREATE INDEX usage_adjustments_member_period_idx ON usage_adjustments(member_id, period);

COMMENT ON TABLE usage_ledger_epochs IS 'Project Spec §17.3; see docs/usage/README.md.';
COMMENT ON TABLE usage_recovery_records IS 'Project Spec §17.3; see docs/usage/restore-runbook.md.';
COMMENT ON TABLE usage_ledger_state IS 'Project Spec §17.3 step 1; see docs/usage/restore-runbook.md.';
COMMENT ON TABLE usage_event_conflicts IS 'Project Spec §10.6; see docs/usage/README.md.';
COMMENT ON TABLE usage_event_rejections IS 'Project Spec §10.6, §10.9; see docs/usage/README.md.';
COMMENT ON TABLE usage_ledger_gaps IS 'Project Spec §10.7, §10.8, §17.3; see docs/usage/README.md.';
COMMENT ON TABLE usage_projection_generations IS 'Project Spec §15.4; see docs/usage/projection.md.';
COMMENT ON TABLE usage_rollup_details IS 'Project Spec §10.9; see docs/usage/projection.md.';
