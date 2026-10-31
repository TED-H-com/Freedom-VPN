# Fresh incarnation usage gap

## Resolution — 2026-10-31, follow-up 9 (D-130)

**Resolved on the assembled live stack.** Usage-agent PR #330 (`6b4c0c0c`)
uses whole-microsecond event endpoints; usage-central PR #331 (`800b4e84`)
applies migration `00193_usage_events_interval_precision.sql`. Run `891ee9b8-bf01-4583-a47a-9542c8b04c8a`
observed the fresh empty unknown-continuity gap at sequence 1 stored centrally as
**`2026-10-31T15:11:08.150045Z` → `2026-10-31T15:11:08.150046Z`**, strictly increasing by **1 µs** and identical to its canonical endpoints.
The CLI and central checkpoint separately observed **ACK 1**, then **ACK 2** after
the counted event. Saved production batch results independently retain both ACKs.
The accepted sequence-2 sample attributes **550 upload + 358 download = 908 bytes**;
the retired stream remains at ACK **3** with its own unchanged history.
All five AT07 composed rows, `installer_prerequisite`, and both probe processes pass.

The minting site is **`planBaseline`**, as corrected in usage-agent 5's
[dated note](../../../node/usage-meter.md#interval-precision-and-compatibility-d-130),
not the historical `planSwitch` attribution. At the exact archived ENR-07 clock,
#330's regression interval is `14:01:15.198143Z` → `14:01:15.198144Z`;
the interval above is this new run's actual central read-back.

Retired usage: **908 + 1816 = 2724 bytes**, original ACK **3**. The fresh stream independently advances **1 → 2**, attributing **550 upload + 358 download = 908 bytes** to its own incarnation, member, device and binding revision. Historical payloads, hashes, samples and totals remain unchanged.

AT20's `BeforePlusHeldByteEquality`, `OriginalPayloadAndHistoricalAttribution`,
`AgentSameKeyStoredResultReplay`, and `OriginACKAndIndependentCurrentACK` were
re-asserted in this run. [Full AT07 JSON](AT07.json) retains CLI, counter,
heartbeat, batch, canonical payload and central stored-interval observations.
[Raw run](history/891ee9b8-bf01-4583-a47a-9542c8b04c8a/AT07.json) and
[byte-identical follow-up 8 archive](history/follow-up-8/README.md) preserve provenance.

The raw diagnostic field `current_batches` also retains earlier batches without
an origin selector from before reenrollment. Select the actual current receipts
by request and response `node_incarnation_id`, `01a19283-b6f4-77bd-b1e8-b270090ae63d`: ACK 1
and ACK 2 both have `accepted` results. Earlier held requests with no response
remain visible; their events were delivered by the separate original-stream
hand-off batches.

The normal migration runner applied 00193. A read-only query on this run's own
PostgreSQL returned the following; no SQL evidence read altered a row:

```json
{
  "constraints": {
    "usage_samples_check": "CHECK ((isfinite(interval_start) AND isfinite(interval_end) AND (interval_start <= interval_end)))",
    "usage_ledger_gaps_check1": "CHECK ((isfinite(affected_from) AND isfinite(affected_until) AND (affected_from <= affected_until)))",
    "usage_events_interval_check": "CHECK ((((interval_start IS NULL) AND (interval_end IS NULL)) OR ((interval_start IS NOT NULL) AND (interval_end IS NOT NULL) AND isfinite(interval_start) AND isfinite(interval_end) AND (interval_start <= interval_end))))"
  },
  "server_version": "17.11 (Debian 17.11-1.pgdg13+2)",
  "migration_00193_applied": true
}
```

Stored columns permit equal projected endpoints for already sealed nanosecond
intervals; canonical bytes still require strict order. This fresh run proves
the new whole-microsecond path. Recovery of the historical sealed 1 ns payload
is covered by the [central owner's D-130 tests and note](../../../usage/2026-10-31-d130-interval-precision.md),
not by rewriting or injecting that old event into this run.

## Historical measurement — follow-up 8

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

Source correlation: `node/internal/meter/delta.go` `planBaseline` emits an unknown
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

[Historical AT07 JSON](history/follow-up-8/AT07.json) retains complete CLI/counter/heartbeat/batch observations.
The raw run remains unchanged in [history](history/f27a81a0-183e-46b4-85a8-304b2f64b635/AT07.json); the published
assessment changes only owner/evidence wording, not outcomes or prior checks.
