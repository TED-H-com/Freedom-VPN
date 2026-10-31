# Retired-stream hand-off — 2026-10-31

D-105 (ENR-05) lets the current incarnation explicitly deliver a retired
incarnation's surviving un-ACKed tail. D-112 defines the central retirement-window
bound. This implements the revision 14 API shapes; the orchestrator's D-112
supersedes revision 14's wording about historical sampling configuration, pending
contracts revision 15. No migration or automatic central drain is introduced.

## Acceptance and ACK ownership

The batch `node_id` and `node_incarnation_id` must match the verified current
mTLS identity. An absent `origin_incarnation_id` retains live-stream behavior.
A supplied origin must belong to that same node, have a non-null `revoked_at`,
and precede the presenting incarnation's `created_at` **strictly**. The read-only
join uses `nodes.current_incarnation_id` and the two `node_incarnations` records;
retirement and enrollment timestamps come from central, never event payloads.

Every event must name the selected original stream. Mixed streams reject the
whole batch before any ledger or checkpoint change. Events retain their original
`node_id`, `node_incarnation_id`, `event_seq`, meter epoch and `payload_sha256`.
The response `node_incarnation_id`, `highest_contiguous_seq`, `blocked_seq`, and
all per-event results name that original stream. Batch `replay_unavailable`
ranges also belong to it. Sending a retired tail cannot advance the current
incarnation's checkpoint. Send separate batches for separate streams.

D-112 checks `event.interval_end <= origin.revoked_at + bound`, inclusive.
`usage.SamplingIntervalBound` is the exported default, `10 * time.Second`.
`usage.Config.SamplingIntervalBound` overrides it at `usage.NewService`; zero
selects the default, and negative values fail construction. Live ingestion has
no clock-skew allowance, so this check adds none. It does not limit how late a
valid historical event may arrive.

The bound is a **central parameter**, not reconstructed sampling history. The
node meter must run with an interval **at most the central bound**. Deployments
using a longer interval must raise the central bound before delivering those
tails. The revision 14 vector's 30 seconds is a TEST ONLY service override.
The production configuration-file knob is a request to the wiring lane; the
current handler uses the 10-second service default.

## Results and audit evidence

| Condition | HTTP / result | Ledger effect |
|---|---|---|
| Eligible new event within the window | 200, `accepted` | Commit and project once under original identity; advance only contiguous ACK |
| Same original event key and digest, new HTTP idempotency key or none | 200, `duplicate` | No additional samples or bytes |
| Same HTTP `Idempotency-Key` and request | Stored original 200 result, including its original timestamp/status | Durable response replay, no re-ingestion |
| Same event key, changed digest, within window | 200, `conflict` / `USAGE_EVENT_CONFLICT` | Keep original and conflict evidence; no extra bytes |
| Unknown, foreign, active origin, or presenter enrollment at/before retirement | 409 `STATE_CONFLICT` | No ledger/checkpoint mutation |
| Batch/event identity mismatch, including a second stream | 403 `INCARNATION_MISMATCH` | No ledger/checkpoint mutation |
| Event ends after retirement plus bound | 200, `rejected` / `VALIDATION_FAILED` | Rejection evidence; never advance ACK for that event |
| Retired certificate presented | 403 `IDENTITY_REVOKED` from the existing PKI authenticator | Handler never ingests the body |
| Frozen ledger / old ledger epoch | 503 `DEPENDENCY_UNAVAILABLE` / 409 `LEDGER_EPOCH_MISMATCH` | Existing recovery fences apply |

Window validation applies to each handed-off event, including resends, before
ordinary event deduplication. A rejected event does not roll back other valid
events in the batch; later committed events remain stored behind the contiguous
ACK barrier. Structural/hash and historical binding validation remain in force.

The new audit action is `usage.ingest_rejected`, result `failed`, with actor
`node:<presenting node>` and resource `node:<presenting node>`. Metadata is
allowlisted: presenting/origin incarnation IDs, error code, bounded reason, and
optional decimal-string event sequence; the HTTP request ID is retained when
available. Reasons are `batch_identity`, `event_identity`, `presenter_inactive`,
`origin_ineligible`, `retired_window`, `event_validation`, and `event_conflict`.
No event body, certificate, secret or browsing information enters this audit.

Whole-batch identity/origin failures commit their audit separately from the
rejected HTTP command transaction. Handed-off per-event rejection/conflict audit
shares the event evidence transaction. If audit persistence fails, the request
returns 503; the event transaction rolls back. Accepted and duplicate events do
not emit rejection audit rows. Retired certificate rejection occurs upstream
in the existing PKI authentication path, before this ingestion audit action.

## Assertions available to adjacent lanes

Usage-agent follow-up 3 can assert that a surviving tail is uploaded under the
new current certificate with `origin_incarnation_id` set and original event
bytes/digests unchanged. It must route each returned ACK to that original spool,
retry until ACK, keep batches to one stream, and assert its sampling interval is
at most the central bound. It must not re-mint events under the new incarnation
or use the retired certificate.

Usage AT follow-up 5 can assert that final attributed bytes equal the committed
pre-handoff total plus accepted held bytes, exactly once despite retries. Match
historical member, device, binding revision, period, direction and original
stream; the presenting incarnation does not replace any of them. The contract
fixture retains a byte count above 2^53. Also assert an unchanged current-stream
ACK, audited foreign/mixed/window failures, and separate progress of valid
same-stream events around a rejected sequence. Live network and agent spool
acceptance remain those lanes' work; these central integration tests are not
real-device evidence.

## Restore-runbook impact

No operator procedure change is required in `restore-runbook.md` (§17.3, AT30).
The freeze-before-service-start ordering and D-095 fresh-observation hold remain
in force. `FreezeIngest` still fences in-flight per-event writes through the
ledger-state locks. `BeginRecovery` discovers streams from checkpoints/events,
including retired ones, and recomputes each original stream under the new epoch.
Hand-off retains the ordinary epoch mismatch and reconciling-only
`replay_unavailable` checks, with gaps/tombstones attributed to the origin.
`CompleteRecovery` does not change stream identity or release observation holds.

Same-key HTTP replay returns its already-stored result even while frozen; it
neither performs ingestion nor creates a newer ACK. Fresh delivery must use the
current ledger epoch and pass the same freeze gates as live-stream delivery.
The new tests cover frozen hand-off, a freeze during checkpoint initialization,
epoch rotation, retired-stream gap attribution, and recovery completion.
