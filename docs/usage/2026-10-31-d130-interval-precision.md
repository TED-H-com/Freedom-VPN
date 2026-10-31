# D-130: persist canonical usage intervals at ledger precision

## ENR-07 defect

Enrollment follow-up 8 measured a fresh incarnation whose first sealed event was
an unknown-continuity gap with no records. Its interval was exactly
`2026-10-31T14:01:15.198144331Z` to `2026-10-31T14:01:15.198144332Z` (1 ns).
Central accepted the strict nanosecond order, but both PostgreSQL `TIMESTAMPTZ`
endpoints became `2026-10-31T14:01:15.198144Z`. The old
`usage_events_interval_check` rejected equality, causing retryable
`503 DEPENDENCY_UNAVAILABLE`. ACK stayed at 0 and sequence 2's 550 upload plus
358 download bytes remained unacknowledged. The sealed event could not be
rewritten, replaced or skipped (§10.5–§10.7, MTR03/MTR04).

The measured evidence is in [current-stream-gap.md](../e2e/at/enrollment/current-stream-gap.md)
and the `controlled_reinstall` row of [AT07.json](../e2e/at/enrollment/AT07.json).
Those files retain the pre-fix observation; this change does not revise their results.

## Rule and implementation

`validateEventFields` continues to require `interval_end > interval_start` on
the original nanosecond values. Equal or reversed canonical endpoints return
per-event `rejected` / `VALIDATION_FAILED`, field `/events/0/interval_end`
(`0` is the event's batch index), reason `interval_order`.

After validation and historical binding checks, `ingestSingleEvent` explicitly
truncates each endpoint with `UTC().Truncate(time.Microsecond)` for the event,
sample and gap rows. It does not round, widen the stored interval or alter the
event. Strict canonical order implies nondecreasing stored endpoints; valid
submicrosecond intervals can therefore have equal stored endpoints.

Canonical payload bytes, `payload_sha256`, original identity, member/device/
binding revision, direction counts, per-event results and ACK/replay rules are
unchanged (§10.2, §10.5–§10.6, §14.5). Initial period splitting and elapsed
nanoseconds still use the original interval. Contract revision 14 and retirement
records are untouched.

Projection rebuild reads the durable microsecond columns. If endpoints coincide,
both original endpoints were in the same microsecond and hence the same UTC day
and month. The projector assigns all bytes to that bucket without estimation;
gap overlap lookup uses that bucket's upper bound, including at midnight. This
does not rewrite stored timestamps or payloads and works after raw-payload
retention expires (§10.9, §15.4–§16.4).

Unexpected insert errors, including constraint violations, retain the existing
retryable `503 DEPENDENCY_UNAVAILABLE` classification with the public message
`usage ledger database unavailable`. The failed event transaction rolls back and
returns no ACK. SQL details stay out of the response. A remaining constraint
failure warrants central diagnosis; it is not a definitive rejection of a sealed
node event. D-130 removes the precision mismatch by changing the schema invariant.

## Migration 00193

[00193_usage_events_interval_precision.sql](../../db/migrations/00193_usage_events_interval_precision.sql)
is automatically embedded by the existing `*.sql` migration source. Apply it
through the normal migration runner before resuming ingestion:

- `usage_events_interval_check`: both endpoints null, or both non-null and finite
  with `interval_start <= interval_end`.
- `usage_samples_check`: finite endpoints with `interval_start <= interval_end`;
  the existing `NOT NULL` columns remain. A byte-carrying 1 ns event needs this
  corresponding change as well as the header change.
- `usage_ledger_gaps_check1`: finite endpoints with
  `affected_from <= affected_until`; the existing `NOT NULL` columns remain.
  The ENR-07 gap otherwise fails here after passing the header insert.

Goose applies these replacements transactionally and validates existing rows.
Existing strict intervals already satisfy the new checks. No events, payloads,
samples, counts, checkpoints, hashes or retirement records are updated. No
constraint is disabled or bypassed at runtime. Reversed endpoints, partial-null
headers and infinities remain forbidden.

The Down migration restores the strict checks only if existing data permits it.
Once equal projected endpoints exist, Down fails atomically and preserves the
schema and evidence. Never delete or modify immutable evidence to downgrade.

## Automatic retry and restore impact

Before deployment an operator sees repeated ledger-unavailable responses and
`usage_events_interval_check` database errors, current ACK 0, and the retained
events 1 and 2. After the migration, the next normal retry accepts the same sealed
gap and advances ACK to 1; accepting sequence 2 advances it to 2 and attributes
the 908 bytes to the original member, device and binding revision. A batch
containing both can return ACK 2 directly. A same-key replay returns the stored
original result; a new-key duplicate does not add bytes (§10.6, §14.5).

This requires no node-spool edit, SQL data mutation, operator re-enrollment,
retirement adjustment, epoch rotation or event skipping. Normal deployment of
the migration and central code is sufficient for this defect's recovery.

The [restore runbook](restore-runbook.md) still applies for an actual database
restore: freeze before service startup, rotate the epoch, synchronize checkpoints,
replay and complete recovery (§17.3, AT30). A restored pre-00193 schema must receive
the migration before ingestion resumes. Replay-unavailable headers can still have
both interval columns null. Rebuild now handles coincident stored endpoints even
without retained raw payloads. Neither the runbook nor [README](README.md) states
a strict stored timestamp invariant, so their procedures are unchanged; the
README's strict validation rule remains correct at canonical precision.

## Evidence and adjacent-lane assertions

`internal/usage/interval_precision_integration_test.go` covers:

- Populated schema 00192, its real HTTP 503, migration without changing existing
  rows, retry with the same event/key, ACK 1 then ACK 2 and 908-byte attribution.
- Original canonical bytes/hash, same-key stored results (including old ACK 1
  after the stream reaches 2), new-key duplicate without another sample.
- Byte-carrying 1 ns intervals at the exact ENR-07 time, above the half-microsecond
  rounding point, and at a UTC day boundary; rebuild after actual payload purge.
- Canonical equal/reversed rejection with `interval_order`, validated stored
  finite/null/order guards, reversible pre-use migration and refused post-use Down.

Usage AT follow-up 6 and enrollment follow-up 9 can now test automatic delivery
of already sealed ENR-07 events through the public API: current-stream ACK reaches
2, 550/358 bytes remain attributable, canonical hashes stay fixed, retries do not
duplicate bytes, and the retired stream's history remains separate. They should
record fresh live evidence for those assertions. Central PostgreSQL/HTTP tests
and amd64/arm64 cross-builds do not establish a live node or real-arm64 run.
Usage-agent follow-up 5 separately owns minting new whole-microsecond intervals
with minimum width 1 µs; accepting existing sealed nanosecond events does not
depend on that node change.
