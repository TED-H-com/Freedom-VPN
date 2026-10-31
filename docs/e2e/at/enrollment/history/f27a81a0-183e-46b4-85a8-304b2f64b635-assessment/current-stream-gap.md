# Fresh incarnation usage gap — follow-up 8

**Blocked; owner: agent/meter, with usage-central timestamp/validation handling.**
Run `f27a81a0-183e-46b4-85a8-304b2f64b635`, source `569ef8d5cedc48d24bff4b1f67165aa702d269a6`. TEST ONLY own-prefix installer
and PostgreSQL; no product code, event, retirement timestamp or database row edited.

The public API and ControlledReinstall establish the same Node with a new
incarnation. The original stream uploads and ACKs through 3; its canonical
payloads and historical attribution pass AT20's assertions. After API
republication, the replacement serves with the hookless marker and four rendered
chains and fresh TCP/UDP traffic counts. The current CLI retains empty gap event
1 followed by counted event 2 (550 upload + 358 download bytes); current ACK stays
0 throughout the 90-second window. Central event/sample reads both succeed and
find no accepted current event. The upload hold marker was independently checked
absent, and the current signed checkpoint returned HTTP 200 with ACK/max-seen 0.

The stopped coherent SQLite copy contains the real production batch:

```json
{
  "central_response_present": false,
  "idempotency_key": "01a19245-fe34-7075-aec4-05943bfce181",
  "request": {
    "events": [
      {
        "event_seq": "1",
        "gap": {
          "last_checkpoint_at": null,
          "reason": "unknown"
        },
        "interval_end": "2026-10-31T14:01:15.198144332Z",
        "interval_start": "2026-10-31T14:01:15.198144331Z",
        "meter_epoch_id": "01a19243-bcbe-72a0-9293-5db7d26853c8",
        "metering_version": "forward_admission_v1",
        "node_id": "01a19240-8ead-777a-abee-237d9c69d756",
        "node_incarnation_id": "01a19243-bc51-7ac5-b20a-7ecdb7d36a93",
        "payload_sha256": "149be4f8c4bcb2b33e1edb36980e8ce70bec40ca653d7a83a3f6c980850cd8c4",
        "quality": "gap",
        "records": [],
        "schema_version": 1
      },
      {
        "event_seq": "2",
        "interval_end": "2026-10-31T14:03:38.180117348Z",
        "interval_start": "2026-10-31T14:03:33.15863239Z",
        "meter_epoch_id": "01a19245-d73f-77f4-aadf-af5525d78b9e",
        "metering_version": "forward_admission_v1",
        "node_id": "01a19240-8ead-777a-abee-237d9c69d756",
        "node_incarnation_id": "01a19243-bc51-7ac5-b20a-7ecdb7d36a93",
        "payload_sha256": "6fad9a72e8fdb783d145ff4cec74293a65a8e83cc55a76936a56593e4140351f",
        "quality": "complete",
        "records": [
          {
            "binding_revision": "1",
            "device_id": "01a1923f-3cae-7e15-b547-a08502760a66",
            "download_bytes": "358",
            "member_id": "01a1923f-1faf-73ff-b026-9fb77ed6ab40",
            "period_id": "2026-10",
            "upload_bytes": "550"
          }
        ],
        "schema_version": 1
      }
    ],
    "ledger_epoch": "d3c46868-7277-427e-92dc-6c06e3bb8fe0",
    "node_id": "01a19240-8ead-777a-abee-237d9c69d756",
    "node_incarnation_id": "01a19243-bc51-7ac5-b20a-7ecdb7d36a93"
  }
}
```

The gap's interval is `2026-10-31T14:01:15.198144331Z` to `2026-10-31T14:01:15.198144332Z`:
exactly **1 ns**. A read-only cast of those exact values in the same owned
PostgreSQL 17 image during the join phase gives:

```json
{
  "input_end": "2026-10-31T14:01:15.198144332Z",
  "stored_end": "2026-10-31T14:01:15.198144+00:00",
  "input_start": "2026-10-31T14:01:15.198144331Z",
  "stored_start": "2026-10-31T14:01:15.198144+00:00",
  "server_version": "17.11 (Debian 17.11-1.pgdg13+2)",
  "equal_after_cast": true,
  "strictly_increasing": false,
  "scope": "Read-only exact gap-interval cast on the same owned PostgreSQL 17 image during join startup; no rows changed."
}
```

The ingestion constraint requires strictly increasing persisted timestamps.
Actual database errors (only the constraint line is published; private query
logs are not committed):

```text
2026-10-31 14:01:23.343 UTC [131] ERROR:  new row for relation "usage_events" violates check constraint "usage_events_interval_check"
2026-10-31 14:03:28.010 UTC [107] ERROR:  new row for relation "usage_events" violates check constraint "usage_events_interval_check"
2026-10-31 14:03:29.015 UTC [107] ERROR:  new row for relation "usage_events" violates check constraint "usage_events_interval_check"
2026-10-31 14:03:31.021 UTC [107] ERROR:  new row for relation "usage_events" violates check constraint "usage_events_interval_check"
2026-10-31 14:03:35.026 UTC [107] ERROR:  new row for relation "usage_events" violates check constraint "usage_events_interval_check"
2026-10-31 14:03:43.032 UTC [107] ERROR:  new row for relation "usage_events" violates check constraint "usage_events_interval_check"
2026-10-31 14:03:59.037 UTC [131] ERROR:  new row for relation "usage_events" violates check constraint "usage_events_interval_check"
```

The agent journal records retryable `usageupload: ledger unavailable` for these
attempts, and there is no committed batch replay result for the current batches.
The current checkpoint is independently readable. This is a product timestamp
precision defect, not a missing hold release, marker/verifier failure, or a
claim that a 202 completed installation.

Source correlation: `node/internal/meter/delta.go` `planSwitch` emits an unknown
continuity gap with `start := now.Add(-time.Nanosecond)`; other gap paths also use
1 ns. `internal/usage/ingest.go` accepts the increasing nanosecond interval and
inserts it into TIMESTAMPTZ columns. `db/migrations/00130_usage_ledger.sql`
defines `usage_events_interval_check`; the equal PostgreSQL endpoints violate it
and the transaction returns a retryable dependency error. The exact cast and
repeated constraint errors corroborate that causal path.

Request the owning lanes to make generated gap intervals compatible with central
persistence precision and define recovery for already immutable blocked gap
payloads. Preserve canonical original bytes and attribution; do not skip events,
rewrite them in this harness, weaken the constraint, or perform SQL retirement.
Rerun the fresh current stream through ACK after the owner fix lands.

[AT07 JSON](AT07.json) retains complete CLI/counter/heartbeat/batch observations.
The raw run remains unchanged in [history](../f27a81a0-183e-46b4-85a8-304b2f64b635/AT07.json); the published
assessment changes only owner/evidence wording, not outcomes or prior checks.
