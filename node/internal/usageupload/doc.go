// Package usageupload sends the usage outbox to the central ledger and
// applies the contiguous ACK (§10.5, §10.6, §10.8, §17.3).
//
// A batch goes out every 60 seconds (建議預設) or sooner when the outbox
// reaches the size threshold. Retries use bounded exponential backoff with
// jitter. The same request bytes keep the same Idempotency-Key.
//
// Per-event results move a contiguous prefix into the replay archive.
// blocked_seq and a hash conflict stop the prefix. A 503 does not move the
// ACK. A 409 LEDGER_EPOCH_MISMATCH fetches the signed usage checkpoint
// before the next attempt. The batch result is not a signed checkpoint and
// cannot change the ledger epoch or lower the ACK.
//
// Resends use the stored canonical payload byte for byte. This package does
// not rebuild an event.
package usageupload
