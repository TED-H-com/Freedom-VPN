# Usage meter

`node/internal/meter` turns `forward_admission_v1` named counters into usage events (§10.1, §10.5, §10.7, §10.9, §13.6 steps 8 and 11). It does not install nftables and it does not change forwarding. The agent owns the data path and calls `Health` to decide whether members may forward. The meter calls `enforce.DetectResets` (values only). The agent, when it holds a `State`, calls `enforce.DetectMeterResets` for a table, forward chain, or generation marker cleared outside the process.

Usage is the inner IPv4 total length admitted at the forward hook, as exposed by the `node/enforce` counters (D-019). Bytes later dropped by HTB are not subtracted. DNS counters (`dns_service_v1`, names `nu_` / `nd_`) and diagnostic counters are not member usage.

## Sampling

`New` reads the counters once. A short restart keeps the open meter epoch only when the boot id is unchanged and `enforce.DetectResets` reports nothing. A missing counter, a counter that went backwards, or a wrap (which looks like a backwards counter) is `METER_EPOCH_CHANGE`: a new `meter_epoch_id`, and a gap on the old epoch. Negative deltas are never recorded and never offset a later sample.

`Run` samples every 10 seconds by default (`DefaultSampleInterval`, 建議預設). The wait uses `Config.Mono` (default `time.Now`, so `Sub` follows the monotonic clock) and wakes earlier at the next UTC month boundary. A wall-clock step does not shorten or stretch that interval. Once the wall clock is already at or past the boundary, `Run` samples immediately so the interval can be stamped there (§10.9). `Config.Now` supplies the wall-clock timestamp, normalized to UTC microseconds when an interval is minted as described below.

`quality` is `complete`, `gap`, or `boundary_estimated`.

- `complete`: every changed device with a stable binding is in the event, and the interval does not cross a UTC month.
- `gap`: counters were rebuilt, the boot id changed (`power_loss`), the mapping changed (`unknown`), or the wall clock stepped back by at least one sample interval (`clock_anomaly`). Unknown bytes are not written as zero and claimed complete. Healthy devices in the same interval can still report their non-negative deltas inside the gap. A gap wins over `boundary_estimated`.
- `boundary_estimated`: the sample interval crosses a UTC month because the boundary sample did not succeed. The node does not split the bytes. The central projector is the authority for the split (§10.9). A sample that lands within one sample interval after the boundary is stamped on the boundary and stays `complete`.

A quiet interval updates the checkpoint time and does not invent an event. The first open with non-zero counters, or an open after a crash, emits a gap (`unknown`) and then baselines at the current readings. A clean `FinalSample` baselines the next process at the checkpointed readings and does not recount them (§13.6 step 11).

## Interval precision and compatibility (D-130)

2026-10-31 — ENR-07 measured a fresh incarnation blocked at ACK 0: its
`planBaseline` unknown-continuity gap spanned
`2026-10-31T14:01:15.198144331Z` to `2026-10-31T14:01:15.198144332Z`, exactly
1 ns. Central validated the canonical nanosecond order, but PostgreSQL
`TIMESTAMPTZ` persisted both endpoints as `14:01:15.198144`, violating
`usage_events_interval_check`. The resulting 503 retried indefinitely, leaving
the following 908 counted bytes at sequence 2 unable to receive a contiguous
ACK. The evidence's `planSwitch` attribution is corrected here: the measured
`unknown counter continuity` event comes from `planBaseline`.

D-130 requires every newly minted event to have UTC whole-microsecond endpoints
and `interval_end - interval_start >= 1 µs`. One helper, `eventInterval`, applies
this rule to `planBaseline`, `planSample` (including a missing checkpoint),
`planClockGap` and `planSwitch`, for complete and estimated intervals as well as
gaps. It truncates both endpoints; without a prior checkpoint it uses the
microsecond immediately before the truncated end. With a checkpoint it keeps
the truncated start and advances an equal or earlier end to `start + 1 µs`.
Thus a switch during clock rollback cannot move the stream backward. The clock
sources, reads, monotonic schedule and month-boundary stamping stay unchanged.

`SampledAt` and durable `LastSample` equal the minted end; the following sample
starts there, including across meter-epoch switches. Quiet baselines and samples
also checkpoint whole microseconds without inventing an event; a quiet reopen
does not advance a checkpoint already at or ahead of its clock reading. Quiet
intervals can still separate emitted events. On upgrade, the first new start
uses the truncated legacy checkpoint: an old nanosecond endpoint cannot also be
a whole-microsecond endpoint. Only that new start is projected; the old event
and its hash remain untouched. Subsequent new intervals share exact endpoints.

Uploader behavior is unchanged (§10.5–§10.6). HTTP 503 retains the canonical
payload and ACK and retries with bounded backoff, which remains correct for real
outages. An HTTP 422, including a per-event validation rejection described in
its error body, logs the status/code, returns an error, retains the events and
ACK, and retries; it does not apply per-event receipts from a 422 body. Per-event
`rejected` results in an HTTP 200 response stop the current stream's contiguous
ACK before the rejected event and retain its payload. Retired-stream final
receipts keep their existing audit/retention behavior described in
[Usage upload](usage-upload.md#re-enrollment-hand-off-d-105).

Already sealed 1 ns events are never rewritten, re-minted or skipped. D-130's
central migration permits equal projected endpoints while strict order remains
validated on the canonical nanosecond values. Existing ENR-07 blocked streams
recover on their next retry after that central fix and migration, with no
node-side action. Canonical payload `schema_version: 1`, existing event hashes
and contracts revision 14 are unchanged. Central acceptance, ACK progress through
1 then 2, and attribution of the retained 550 upload + 358 download bytes remain
assembled acceptance assertions for usage AT follow-up 6 and enrollment
follow-up 9; local meter tests do not claim those central observations.

## Interfaces for the agent

```go
type Controller interface {
    Barrier(ctx context.Context) error
    FinalSample(ctx context.Context) error
    Health(ctx context.Context) (usagestore.Health, error)
}
```

`Barrier` pauses `Run` and waits until the in-flight sample finishes (§13.6 step 8). `ResumeSampling` continues a barrier that was not followed by `FinalSample`.

`FinalSample` calls `Barrier`, samples once more, and closes the meter epoch. If that sample has to open a new epoch because of a discontinuity, it closes the fresh epoch as well so the next process opens its own. `ErrNoEpoch` means there was nothing to finish. `ErrHold` means the wall clock jittered by less than one sample interval; the updater should retry rather than switch.

`Health` calls `usagestore.EvaluateGate`, which is the call that persists `gate_stopped` (§10.8). `OutboxStatus` does not. When the last counter read failed and the store has no reason of its own, `Health` returns `ForwardingAllowed=false` with reason `meter_path_invalid`. Member forwarding stops. The control channel is not stopped here. Sampling continues while a cap is open so the outbox can drain. If a sample cannot be persisted, `Health` reports `persist_failed` until a later commit succeeds (§10.7, last paragraph).

`Close` stops the sampler. It does not close the store.

## Reading the local ledger

`fvpn-agent usage snapshot` prints the meter epochs this sampler has opened and closed, the assigned sequences and the unacknowledged events. It does not sample and it does not open a new meter epoch. The field list and the lab contract are in `docs/node/usage-store.md`, section "Reading the local ledger". A restart of `fvpn-agent run` does not change those hashes or sequences, and the watermark in the snapshot is the persisted one.

## Fault injection

`Sample` asks the store hook to fail at `sample.read` (before the counter read) and `sample.plan` (before the commit). Store commit steps are in `docs/node/usage-store.md`. An injected crash is not reported as `persist_failed`. A crash after commit leaves the event durable; the next sample does not emit it again.


## Retirement window bound (D-112)

Production `Engine.ensureUsage` in `node/agent/usage.go` omits `Config.Interval`,
so `meter.New` resolves it to `DefaultSampleInterval = 10 * time.Second`.
`TestD112_ProductionSamplingInterval` pins the actual production construction
site and the resolved default. A production interval must be at most central's
`usage.SamplingIntervalBound`, whose default is 10 seconds. A deployment that
needs a longer interval must first raise central `sampling_interval_bound_seconds`,
then raise the meter interval, keeping it at or below the new bound. This ordering
is also recorded at the pin test. Revision 14's historical-interval
wording is superseded by D-112; central does not reconstruct sampling history.

Before a controlled retirement the enrollment owner stops forwarding and the
old sampler, preserving its final committed sample. Re-enrollment hands off
those stored events unchanged. No capability claim says a stalled sampler or
clock always fits the window; central rejects over-window events and the
uploader retains their payloads with a final local rejection audit.
