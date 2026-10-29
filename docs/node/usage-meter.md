# Usage meter

`node/internal/meter` turns `forward_admission_v1` named counters into usage events (§10.1, §10.5, §10.7, §10.9, §13.6 steps 8 and 11). It does not install nftables and it does not change forwarding. The agent owns the data path and calls `Health` to decide whether members may forward. The meter calls `enforce.DetectResets` (values only). The agent, when it holds a `State`, calls `enforce.DetectMeterResets` for a table, forward chain, or generation marker cleared outside the process.

Usage is the inner IPv4 total length admitted at the forward hook, as exposed by the `node/enforce` counters (D-019). Bytes later dropped by HTB are not subtracted. DNS counters (`dns_service_v1`, names `nu_` / `nd_`) and diagnostic counters are not member usage.

## Sampling

`New` reads the counters once. A short restart keeps the open meter epoch only when the boot id is unchanged and `enforce.DetectResets` reports nothing. A missing counter, a counter that went backwards, or a wrap (which looks like a backwards counter) is `METER_EPOCH_CHANGE`: a new `meter_epoch_id`, and a gap on the old epoch. Negative deltas are never recorded and never offset a later sample.

`Run` samples every 10 seconds by default (`DefaultSampleInterval`, 建議預設). The wait uses `Config.Mono` (default `time.Now`, so `Sub` follows the monotonic clock) and wakes earlier at the next UTC month boundary. A wall-clock step does not shorten or stretch that interval. Once the wall clock is already at or past the boundary, `Run` samples immediately so the interval can be stamped there (§10.9). `Config.Now` is the UTC timestamp stored on the event.

`quality` is `complete`, `gap`, or `boundary_estimated`.

- `complete`: every changed device with a stable binding is in the event, and the interval does not cross a UTC month.
- `gap`: counters were rebuilt, the boot id changed (`power_loss`), the mapping changed (`unknown`), or the wall clock stepped back by at least one sample interval (`clock_anomaly`). Unknown bytes are not written as zero and claimed complete. Healthy devices in the same interval can still report their non-negative deltas inside the gap. A gap wins over `boundary_estimated`.
- `boundary_estimated`: the sample interval crosses a UTC month because the boundary sample did not succeed. The node does not split the bytes. The central projector is the authority for the split (§10.9). A sample that lands within one sample interval after the boundary is stamped on the boundary and stays `complete`.

A quiet interval updates the checkpoint time and does not invent an event. The first open with non-zero counters, or an open after a crash, emits a gap (`unknown`) and then baselines at the current readings. A clean `FinalSample` baselines the next process at the checkpointed readings and does not recount them (§13.6 step 11).

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

`Health` returns the store gate (§10.8) and, when the last counter read failed and the store has no reason of its own, `ForwardingAllowed=false` with reason `meter_path_invalid`. Member forwarding stops. The control channel is not stopped here. Sampling continues while a cap is open so the outbox can drain. If a sample cannot be persisted, `Health` reports `persist_failed` until a later commit succeeds (§10.7, last paragraph).

`Close` stops the sampler. It does not close the store.

## Fault injection

`Sample` asks the store hook to fail at `sample.read` (before the counter read) and `sample.plan` (before the commit). Store commit steps are in `docs/node/usage-store.md`. An injected crash is not reported as `persist_failed`. A crash after commit leaves the event durable; the next sample does not emit it again.
