# Usage upload

`node/internal/usageupload` posts the outbox to `POST /api/v1/agent/usage-batches` and reads `GET /api/v1/agent/usage-checkpoint` (§10.5, §10.6, §10.8, §17.3).

`Run` sends every 60 seconds (`DefaultInterval`, 建議預設) or sooner when the outbox reaches 100 events or 512000 bytes of canonical payload. The spec does not set that size threshold; the contract maximum is 1000 events. Upload continues while `Health` says member forwarding is stopped.

## Batch bytes

Each event in the batch is the stored canonical payload, embedded unchanged. The client does not rebuild an event. `ledger_epoch` is JSON null until a verified checkpoint has been applied. A batch response never changes the ledger epoch.

`Idempotency-Key` is a new UUIDv7 for a new body. A retry of the same bytes reuses the key and the stored body. `HTTPTransport` sets `Content-Type: application/json` and that header. The agent supplies an `*http.Client` with the node mTLS certificate.

## Results

A 200 applies `ApplyACK`: contiguous `accepted` and `duplicate` results move into the archive, `blocked_seq` and `conflict` or `rejected` stop the prefix, and a later sequence is not archived while an earlier one is unconfirmed. A 503, including a frozen ledger, does not move the ACK. A 429 waits at least `Retry-After` (delta-seconds or HTTP-date) and does not move the ACK. A 422 or 403 keeps the events.

A 409 `LEDGER_EPOCH_MISMATCH`, or a 200 whose `ledger_epoch` is not the verified one, fetches the checkpoint and does not apply that batch's ACK. A 409 `USAGE_EVENT_CONFLICT` sets `blocked_seq` to the first sequence in the batch and keeps the payload. The contract path for a hash conflict is the per-event `conflict` status on 200 (`USAGE_EVENT_CONFLICT`).

Failures back off from 1 second, doubling, capped at 5 minutes, with ±20% jitter when `Config.Rand` is set. Those timings are engineering defaults; the spec does not set them.

## Restore

The checkpoint body is the raw recovery envelope. `ApplyCheckpoint` verifies it. A new signed ledger epoch may lower the watermark and puts the original events back on the outbox. The same epoch never accepts a lower ACK. Ranges this node cannot replay are sent as `replay_unavailable` on the next batch.

## Fault injection

`Config.Hook` may fail at `upload.build` (body is built, not yet remembered), `upload.post` (before the request) and `upload.apply` (before the ACK or the conflict block is written).
