# Usage upload

`node/internal/usageupload` posts the outbox to `POST /api/v1/agent/usage-batches` and reads `GET /api/v1/agent/usage-checkpoint` (§10.5, §10.6, §10.8, §17.3).

`Run` sends every 60 seconds (`DefaultInterval`, 建議預設) or sooner when the outbox reaches 100 events or 512000 bytes of canonical payload. The spec does not set that size threshold; the contract maximum is 1000 events. Upload continues while `Health` says member forwarding is stopped.

## Batch bytes

Each event in the batch is the stored canonical payload, embedded unchanged. The client does not rebuild an event. `ledger_epoch` is JSON null until a verified checkpoint has been applied. A batch response never changes the ledger epoch. `OutboxStatus.ledger_epoch` follows that same rule: an ACK can advance `central_acked_through` while the epoch stays null. Null means no verified generation has been accepted (§17.3).

`overThreshold` calls `OutboxStatus` and does not evaluate the forwarding gate. `meter.Health` → `EvaluateGate` is what persists `gate_stopped`, from the agent run loop. A status read during upload does not stop or resume member forwarding.

`Idempotency-Key` is a new UUIDv7 for a new body. A retry of the same bytes reuses the key and the stored body. `HTTPTransport` sets `Content-Type: application/json` and that header. `fvpn-agent` does not build a second TLS stack: it posts through the control client's usage class (`Exchange`), which already uses the node certificate and TLS 1.2 minimum. That class has its own concurrency budget, separate from the lease budget. `HTTPTransport` remains for a caller that already holds an `*http.Client`.

## Results

A 200 applies `ApplyACK`: contiguous `accepted` and `duplicate` results move into the archive, `blocked_seq` and `conflict` or `rejected` stop the prefix, and a later sequence is not archived while an earlier one is unconfirmed. A 503, including a frozen ledger, does not move the ACK. A 429 waits at least `Retry-After` (delta-seconds or HTTP-date) and does not move the ACK. A 422 or 403 keeps the events.

For the current stream, a 409 `LEDGER_EPOCH_MISMATCH`, or a 200 whose `ledger_epoch` is not the verified one, fetches the checkpoint and does not apply that batch's ACK. Retired streams follow the hand-off rules below. A 409 `USAGE_EVENT_CONFLICT` sets `blocked_seq` to the first sequence in the batch and keeps the payload. The contract path for a hash conflict is the per-event `conflict` status on 200 (`USAGE_EVENT_CONFLICT`). Fake control's default is that 200 form. `SetUsageConflict` keeps the whole-request 409 path for the regression test. The uploader accepts both.

Failures back off from 1 second, doubling, capped at 5 minutes, with ±20% jitter when `Config.Rand` is set. Those timings are engineering defaults; the spec does not set them.

## Restore

Restore applies to the current incarnation. The checkpoint body is the raw recovery envelope. `ApplyCheckpoint` verifies it with the node trust store (`recovery` purpose). A new signed ledger epoch may lower the watermark and puts the original events back on the outbox. The same epoch never accepts a lower ACK. Ranges this node cannot replay are sent as `replay_unavailable` on the next batch.

`Restore` is that fetch plus `ApplyCheckpoint`. The agent calls it at startup and again whenever `NeedsRestore` is true. `NeedsRestore` is set when the fetch fails or the body is not applied, and it is cleared after a successful apply or HTTP 404. A 404 means the center has not published a checkpoint; the ledger epoch stays null and forwarding is not failed for that reason. `Once` still fetches the checkpoint itself when a batch returns `LEDGER_EPOCH_MISMATCH` or the store reports that the batch epoch is not the verified one. The agent does not reimplement the signature check.

## Fault injection

`Config.Hook` may fail at `upload.build` (body is built, not yet remembered), `upload.post` (before the request) and `upload.apply` (before the ACK or the conflict block is written).

## Reading the local ledger

`fvpn-agent usage snapshot` is the read-only view of this outbox and its watermark. The field list, exit codes and the lab contract `UsageLedgerSnapshot` (this command's `-json`) are in `docs/node/usage-store.md`. The command does not upload and does not apply an ACK. A 503 from the center leaves the snapshot byte-identical, including `payload_sha256` of the bytes this package resends.


## Re-enrollment hand-off (D-105)

The production hand-off is active as of 2026-10-31; see agent-core follow-up 25's
[D-118 activation evidence](#d-118-activation) below. After that store transition,
`Once` chooses the oldest locally retired
stream with pending events, in original sequence order and subject to the same
batch limits. It then sends the current stream. Each request selects one stream:
`node_id` and `node_incarnation_id` always name the current mTLS presenter;
`origin_incarnation_id` is present only for a retired stream. The nested events
are the retained canonical bytes. No event, sequence, meter epoch, attribution
or hash is rebuilt. The same current transport sends both streams, and the
retired certificate is never loaded.

`UsageBatchResult.node_incarnation_id` must equal the selected original stream
(and `node_id` must match); a mismatched ACK changes neither stream. Original
ACKs and receipts are stored separately. An ACK drains the retired tail and
stops sending it. An ambiguous/lost response retries the saved body with the
same `Idempotency-Key`, including after reopening `usage.db`. Local fixtures
use `usage-retired-handoff.json` and both `UsageBatchRequest.handoff-*` examples,
including the original sequence 1234 and bytes above 2^53.

Whole-request `409 STATE_CONFLICT` (foreign, active or pre-retirement origin)
and `403 INCARNATION_MISMATCH` are final for that origin. Per-event `rejected`
or `conflict` receipts are final for that original event. Each is audited once
in `handoff_rejections` and never retried indefinitely, even after restart.
The local transaction retains the rejected payload and blocked sequence. Later
valid events are delivered once without pretending the blocker was ACKed; the
current stream continues independently. A local audit/receipt commit failure
keeps the event pending and retries it.

Network errors, 429, 503, a revoked presenter certificate and ledger-epoch
mismatches retain the payload and use capped exponential backoff. Each failed
retired attempt allows a current-stream attempt before retrying the origin,
so a recovery problem cannot starve live uploads. There is no arbitrary attempt
count that silently discards usage. The uploader bounds delay (default 1 second
to 5 minutes), respects `Retry-After`, and stops retrying final receipts.

### Retained recovery gaps

2026-10-31 — V1 semantics (§10.8, §17.3) are **reported, never re-fetched**.
The new incarnation reports the
retired origin's retained `replay_unavailable` ranges once with its hand-off,
under that origin's verified `ledger_epoch`. "Once" means one hand-off lifecycle:
lost responses retry the same saved body and `Idempotency-Key` until ACK. Only
the origin's contiguous ACK can clear covered ranges; subsequent batches and
restarts do not report those acknowledged ranges again. Retirement also keeps a
copy in `retired_history` as local audit evidence. No gap or event is re-minted
in the current stream. Central records these unavailable ranges as gaps during
reconciliation; this does not recover the missing bytes.

The agent never fetches a restore checkpoint for retired ranges. The checkpoint
GET has no origin selector and binds the current incarnation, so a current
checkpoint cannot lower the origin's ACK, replace its epoch or recreate its
cleared pending ranges. Neither a retired-stream 409 `LEDGER_EPOCH_MISMATCH` nor
a 200 with a different epoch starts a GET or sets `NeedsRestore`. The tail and
unconfirmed ranges remain local, with bounded upload backoff; current-stream
restore/upload continues independently. This also applies after reopening.

Revision 14 requires at least one event in a batch. A retired range with no
pending event therefore stays local and does not schedule an upload or restore;
it is not claimed as reported. A final origin/event rejection likewise retains
unconfirmed evidence without inventing an ACK. Cross-epoch retired recovery and
range-only reporting require a future transport contract. No contract, schema
or migration is changed by this v1 decision.

`TestD105_RetiredGapsReportedNeverRefetched` proves original bytes/hashes,
retired-first ordering, a new cursor at 1, independent ACKs, lost-ACK retry and
no re-report or GET after restart. `TestD105_RetiredEpochMismatchNeverFetchesCheckpoint`
and `TestD105_RetiredRangeWithoutEventsStaysLocal` cover the failure/unsupported
paths. These package tests complement the D-118 agent test below; assembled
ENR-05 reinstall and real arm64 evidence belong to usage AT follow-up 5 and
enrollment follow-up 8.

### Retirement window bound (D-112)

D-112's window is a central bound, default 10 seconds, checked against each
original event's interval end. The uploader sends whatever the ledger retained
up to its last sample; it does not trim intervals, discard over-window bytes or
substitute the locally observed retirement time. Central's over-window
`rejected` / `VALIDATION_FAILED` receipt is recorded as final for that event.
The vector's 30-second bound is a TEST ONLY override, not the production default.
`TestD112_ProductionSamplingInterval` pins the production 10-second meter default.
To increase it, raise central `sampling_interval_bound_seconds` first, then the
meter interval, keeping the interval at or below the bound.


### D-118 activation

2026-10-31 — D-118 activation: `Engine.ensureUsage` now supplies
`RetirePreviousAt: e.clock.Wall()` when opening the ledger for the verified
current identity. The normal closing meter barrier commits the final sample
before the stopped agent's ledger is adopted; restart after identity installation
also performs the atomic retirement. The agent regression obtains a new mTLS
certificate for the same node, retains the final unacknowledged sample byte for
byte, uploads the retired stream first with `origin_incarnation_id`, verifies
independent ACKs and a current stream beginning at sequence 1, and proves that
reopening the unchanged incarnation creates no additional retirement. This
activates the package hand-off described above for ENR-05; real assembled
reinstall evidence remains with enrollment follow-up 8 and usage AT follow-up 5.
