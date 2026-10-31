-- +goose Up
-- D-130 / ENR-07: canonical event intervals are strictly ordered at nanosecond
-- precision. TIMESTAMPTZ columns are truncated microsecond projections, so a
-- valid interval can have equal stored endpoints. Keep finite/presence guards.
ALTER TABLE usage_events
 DROP CONSTRAINT usage_events_interval_check,
 ADD CONSTRAINT usage_events_interval_check CHECK(
  (interval_start IS NULL AND interval_end IS NULL)
  OR (interval_start IS NOT NULL AND interval_end IS NOT NULL AND isfinite(interval_start) AND isfinite(interval_end) AND interval_start <= interval_end));

-- Samples and node-reported gaps share the event's projected interval. Their
-- NOT NULL columns retain the presence requirement. No ledger rows are changed.
ALTER TABLE usage_samples
 DROP CONSTRAINT usage_samples_check,
 ADD CONSTRAINT usage_samples_check CHECK(
  isfinite(interval_start) AND isfinite(interval_end) AND interval_start <= interval_end);
ALTER TABLE usage_ledger_gaps
 DROP CONSTRAINT usage_ledger_gaps_check1,
 ADD CONSTRAINT usage_ledger_gaps_check1 CHECK(
  isfinite(affected_from) AND isfinite(affected_until) AND affected_from <= affected_until);

-- +goose Down
-- Fails atomically once equal projected endpoints exist. Never discard accepted
-- immutable events to downgrade; retain this schema once those rows are in use.
ALTER TABLE usage_events
 DROP CONSTRAINT usage_events_interval_check,
 ADD CONSTRAINT usage_events_interval_check CHECK(
  (interval_start IS NULL AND interval_end IS NULL)
  OR (interval_start IS NOT NULL AND interval_end IS NOT NULL AND isfinite(interval_start) AND isfinite(interval_end) AND interval_start < interval_end));
ALTER TABLE usage_samples
 DROP CONSTRAINT usage_samples_check,
 ADD CONSTRAINT usage_samples_check CHECK(
  isfinite(interval_start) AND isfinite(interval_end) AND interval_start < interval_end);
ALTER TABLE usage_ledger_gaps
 DROP CONSTRAINT usage_ledger_gaps_check1,
 ADD CONSTRAINT usage_ledger_gaps_check1 CHECK(
  isfinite(affected_from) AND isfinite(affected_until) AND affected_from < affected_until);
