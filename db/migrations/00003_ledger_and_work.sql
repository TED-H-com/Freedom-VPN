-- +goose Up

-- §10.6, §15.4: durable receipt; raw body lives separately for retention.
CREATE TABLE usage_events (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 event_seq NUMERIC(20,0) CHECK (event_seq >= 0 AND event_seq <= 18446744073709551615) NOT NULL CHECK(event_seq > 0), schema_version INTEGER NOT NULL CHECK(schema_version > 0),
 payload_hash BYTEA NOT NULL CHECK(octet_length(payload_hash)=32), record_count INTEGER NOT NULL CHECK(record_count >= 0),
 received_at TIMESTAMPTZ NOT NULL DEFAULT now(), quality_flags TEXT[] NOT NULL DEFAULT '{}',
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id),
 CONSTRAINT usage_events_stream_key UNIQUE(node_id,incarnation_id,event_seq)
);
-- §15.4, §16.4: optional raw bytes can expire without losing the receipt/hash.
CREATE TABLE usage_event_payloads (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), event_id UUID NOT NULL UNIQUE REFERENCES usage_events,
 payload BYTEA NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE usage_samples (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), event_id UUID NOT NULL REFERENCES usage_events,
 record_index INTEGER NOT NULL CHECK(record_index >= 0), member_id UUID NOT NULL REFERENCES members, device_id UUID NOT NULL,
 binding_revision NUMERIC(20,0) CHECK (binding_revision >= 0 AND binding_revision <= 18446744073709551615) NOT NULL, meter_epoch_id UUID NOT NULL,
 interval_start TIMESTAMPTZ NOT NULL, interval_end TIMESTAMPTZ NOT NULL,
 upload_bytes NUMERIC(20,0) CHECK (upload_bytes >= 0 AND upload_bytes <= 18446744073709551615) NOT NULL, download_bytes NUMERIC(20,0) CHECK (download_bytes >= 0 AND download_bytes <= 18446744073709551615) NOT NULL, metering_version TEXT NOT NULL,
 monotonic_elapsed_ns NUMERIC(20,0) CHECK (monotonic_elapsed_ns >= 0 AND monotonic_elapsed_ns <= 18446744073709551615) NOT NULL, clock_quality TEXT NOT NULL, last_clock_sync_at TIMESTAMPTZ,
 quality_flags TEXT[] NOT NULL DEFAULT '{}',
 FOREIGN KEY(device_id,member_id) REFERENCES member_devices(id,member_id),
 FOREIGN KEY(device_id,binding_revision) REFERENCES device_bindings(device_id,binding_revision),
 CONSTRAINT usage_samples_record_key UNIQUE(event_id,record_index),
 CHECK(isfinite(interval_start) AND isfinite(interval_end) AND interval_start < interval_end)
);
CREATE TABLE projection_receipts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), sample_id UUID NOT NULL REFERENCES usage_samples,
 projection_name TEXT NOT NULL, generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL, period TEXT NOT NULL,
 applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CONSTRAINT projection_receipts_projection_key UNIQUE(sample_id,projection_name,generation,period)
);
CREATE TABLE usage_checkpoints (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 highest_contiguous_seq NUMERIC(20,0) CHECK (highest_contiguous_seq >= 0 AND highest_contiguous_seq <= 18446744073709551615) NOT NULL DEFAULT 0, max_seen_seq NUMERIC(20,0) CHECK (max_seen_seq >= 0 AND max_seen_seq <= 18446744073709551615) NOT NULL DEFAULT 0,
 ledger_epoch UUID NOT NULL, prefix_proof BYTEA, updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id),
 CONSTRAINT usage_checkpoints_stream_key UNIQUE(node_id,incarnation_id),
 CHECK(highest_contiguous_seq <= max_seen_seq)
);
CREATE TABLE usage_rollups (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members,
 period TEXT NOT NULL, granularity TEXT NOT NULL, projection_generation NUMERIC(20,0) CHECK (projection_generation >= 0 AND projection_generation <= 18446744073709551615) NOT NULL,
 metering_version TEXT NOT NULL,
 upload_bytes NUMERIC(40,0) NOT NULL CHECK(upload_bytes >= 0),
 download_bytes NUMERIC(40,0) NOT NULL CHECK(download_bytes >= 0),
 total_bytes NUMERIC(41,0) GENERATED ALWAYS AS (upload_bytes + download_bytes) STORED,
 applied_event_checkpoint UUID REFERENCES usage_events,
 last_observed_at TIMESTAMPTZ NOT NULL, aggregate_as_of TIMESTAMPTZ NOT NULL,
 period_start TIMESTAMPTZ NOT NULL, period_end TIMESTAMPTZ NOT NULL,
 estimated BOOLEAN NOT NULL DEFAULT false, gap BOOLEAN NOT NULL DEFAULT false,
 reconciliation_in_progress BOOLEAN NOT NULL DEFAULT false,
 CONSTRAINT usage_rollups_period_key UNIQUE(member_id,period,granularity,projection_generation),
 CHECK(isfinite(period_start) AND isfinite(period_end) AND period_start < period_end)
);
CREATE TABLE usage_adjustments (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), member_id UUID NOT NULL REFERENCES members, period TEXT NOT NULL,
 original_event_id UUID REFERENCES usage_events, interval_start TIMESTAMPTZ, interval_end TIMESTAMPTZ,
 upload_bytes NUMERIC(40,0) NOT NULL, download_bytes NUMERIC(40,0) NOT NULL,
 reason TEXT NOT NULL CHECK(length(reason)>0), actor TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK(original_event_id IS NOT NULL OR interval_start IS NOT NULL),
 CHECK((interval_start IS NULL AND interval_end IS NULL) OR (interval_start IS NOT NULL AND interval_end IS NOT NULL AND isfinite(interval_start) AND isfinite(interval_end) AND interval_start < interval_end))
);

-- §14.5, §15.4: read-back evidence belongs to each required operation step.
CREATE TABLE operations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), kind TEXT NOT NULL, principal TEXT NOT NULL,
 resource_type TEXT NOT NULL, resource_id UUID, desired_generation NUMERIC(20,0) CHECK (desired_generation >= 0 AND desired_generation <= 18446744073709551615),
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','running','succeeded','partially_succeeded','failed','superseded')),
 schema_version INTEGER NOT NULL DEFAULT 1 CHECK(schema_version > 0),
 steps JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(steps) = 'object'),
 observed_refs JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(observed_refs) = 'object'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), completed_at TIMESTAMPTZ,
 CHECK(completed_at IS NULL OR created_at <= completed_at),
 CHECK(status <> 'succeeded' OR (completed_at IS NOT NULL AND observed_refs <> '{}'))
);

-- §13.2, §15.4: target release never implicitly changes serving policy.
CREATE TABLE fleet_config (
 id UUID PRIMARY KEY CHECK(id = '00000000-0000-0000-0000-000000000001'),
 desired_release_id UUID REFERENCES releases,
 supported_release_policy TEXT NOT NULL DEFAULT 'deny_all' CHECK(supported_release_policy IN ('deny_all','allowlist','minimum_component')),
 supported_release_ids UUID[] NOT NULL DEFAULT '{}', minimum_serving_version TEXT, minimum_serving_component TEXT,
 generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL DEFAULT 0, resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK(supported_release_policy <> 'minimum_component' OR (minimum_serving_version IS NOT NULL AND minimum_serving_component IS NOT NULL))
);
INSERT INTO fleet_config(id) VALUES ('00000000-0000-0000-0000-000000000001');

-- §13.8, §15.4: batch denominator and membership are sealed on creation.
CREATE TABLE update_batches (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), operation_id UUID NOT NULL UNIQUE REFERENCES operations,
 target_release_id UUID NOT NULL REFERENCES releases, scope TEXT NOT NULL CHECK(scope='all_nodes'),
 created_by TEXT NOT NULL, target_count INTEGER NOT NULL CHECK(target_count >= 0),
 generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE update_targets (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), batch_id UUID NOT NULL REFERENCES update_batches,
 node_id UUID NOT NULL REFERENCES nodes, incarnation_at_snapshot UUID, execution_incarnation_id UUID,
 target_generation NUMERIC(20,0) CHECK (target_generation >= 0 AND target_generation <= 18446744073709551615) NOT NULL,
 FOREIGN KEY(node_id,incarnation_at_snapshot) REFERENCES node_incarnations(node_id,id),
 FOREIGN KEY(node_id,execution_incarnation_id) REFERENCES node_incarnations(node_id,id),
 CONSTRAINT update_targets_node_key UNIQUE(batch_id,node_id),
 CONSTRAINT update_targets_execution_key UNIQUE(id,execution_incarnation_id,target_generation)
);
CREATE TABLE update_jobs (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), target_id UUID NOT NULL REFERENCES update_targets,
 attempt INTEGER NOT NULL CHECK(attempt > 0), execution_incarnation_id UUID NOT NULL, generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL,
 state TEXT NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','waiting_online','downloading','verifying','applying','checking','succeeded','failed','rolled_back','superseded','superseded_incarnation')),
 actual_release_id UUID REFERENCES releases, last_good_release_id UUID REFERENCES releases,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), started_at TIMESTAMPTZ, completed_at TIMESTAMPTZ,
 last_result_at TIMESTAMPTZ, last_observed_at TIMESTAMPTZ,
 result_id UUID UNIQUE, result_hash BYTEA CHECK(octet_length(result_hash)=32), error_code TEXT,
 schema_version INTEGER NOT NULL DEFAULT 1 CHECK(schema_version > 0),
 observed_evidence JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(observed_evidence) = 'object'),
 FOREIGN KEY(target_id,execution_incarnation_id,generation) REFERENCES update_targets(id,execution_incarnation_id,target_generation),
 CONSTRAINT update_jobs_attempt_key UNIQUE(target_id,attempt),
 CHECK((result_id IS NULL) = (result_hash IS NULL)),
 CHECK(completed_at IS NULL OR created_at <= completed_at),
 CHECK(started_at IS NULL OR completed_at IS NULL OR started_at <= completed_at),
 CHECK(state <> 'succeeded' OR (actual_release_id IS NOT NULL AND result_id IS NOT NULL AND completed_at IS NOT NULL AND observed_evidence <> '{}'))
);

-- §14.5, §15.4: fenced lease ownership for at-least-once workers.
CREATE TABLE outbox_events (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), aggregate_id UUID NOT NULL, aggregate_type TEXT NOT NULL,
 aggregate_version NUMERIC(20,0) CHECK (aggregate_version >= 0 AND aggregate_version <= 18446744073709551615) NOT NULL, event_type TEXT NOT NULL,
 schema_version INTEGER NOT NULL CHECK(schema_version > 0), payload JSONB NOT NULL CHECK(jsonb_typeof(payload) = 'object'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), attempt INTEGER NOT NULL DEFAULT 0 CHECK(attempt >= 0),
 available_at TIMESTAMPTZ NOT NULL DEFAULT now(), delivered_at TIMESTAMPTZ,
 lease_owner TEXT, lease_token UUID, lease_until TIMESTAMPTZ,
 CHECK(num_nonnulls(lease_owner,lease_token,lease_until) IN (0,3)),
 CHECK(lease_until IS NULL OR isfinite(lease_until))
);
CREATE INDEX outbox_events_available_idx ON outbox_events(available_at,id) WHERE delivered_at IS NULL;
CREATE TABLE idempotency_requests (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), principal TEXT NOT NULL, method TEXT NOT NULL, route TEXT NOT NULL,
 key_hash BYTEA NOT NULL CHECK(octet_length(key_hash)=32), request_hash BYTEA NOT NULL CHECK(octet_length(request_hash)=32),
 state TEXT NOT NULL DEFAULT 'in_progress' CHECK(state IN ('in_progress','completed')),
 resource_type TEXT, resource_id UUID, operation_id UUID REFERENCES operations,
 response_status INTEGER CHECK(response_status BETWEEN 100 AND 599),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), expiry TIMESTAMPTZ NOT NULL,
 CONSTRAINT idempotency_requests_scope_key UNIQUE(principal,method,route,key_hash),
 CHECK(isfinite(expiry) AND created_at < expiry),
 CHECK(state <> 'completed' OR (response_status IS NOT NULL AND (resource_id IS NOT NULL OR operation_id IS NOT NULL)))
);
CREATE INDEX idempotency_requests_expiry_idx ON idempotency_requests(expiry);
CREATE TABLE audit_events (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), actor TEXT NOT NULL, actor_type TEXT NOT NULL,
 action TEXT NOT NULL, resource TEXT NOT NULL, request_id UUID, operation_id UUID REFERENCES operations,
 schema_version INTEGER NOT NULL DEFAULT 1 CHECK(schema_version > 0),
 before_diff JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(before_diff) = 'object'),
 after_diff JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(after_diff) = 'object'),
 result TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_resource_time_idx ON audit_events(resource,occurred_at);
CREATE INDEX audit_events_actor_time_idx ON audit_events(actor,occurred_at);

-- §16.4: configurable recommended defaults; billing retention remains undecided.
CREATE TABLE retention_config (
 id UUID PRIMARY KEY CHECK(id = '00000000-0000-0000-0000-000000000002'),
 raw_usage_days INTEGER NOT NULL DEFAULT 35 CHECK(raw_usage_days > 0),
 rollup_months INTEGER NOT NULL DEFAULT 13 CHECK(rollup_months > 0),
 service_log_days INTEGER NOT NULL DEFAULT 7 CHECK(service_log_days > 0),
 audit_days INTEGER NOT NULL DEFAULT 180 CHECK(audit_days > 0)
);
INSERT INTO retention_config(id) VALUES('00000000-0000-0000-0000-000000000002');
