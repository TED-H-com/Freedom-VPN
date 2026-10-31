# Enforcement timing hand-off — 2026-10-31

For agent-core follow-up 27 and isolation request 2 / ISO-F6-03 / D-104
(§16.3, AT13 §19). This is the reviewable hand-off intended for
`docs/node/followup19-netlab-agent-2026-10-31.md`. That destination is outside
this lane's explicit ownership; the orchestrator must place it there.

## API and retention

Call `result.Evidence.TimingEvidence()` immediately after `enforce.Apply`,
including on error. Call `evidence.TimingEvidence()` after each live
`enforce.VerifyServing`, also on error. Both return a **detached snapshot**;
conversion performs no I/O, clock sampling or mutation. Zero-value evidence
returns nil. Every Apply return carries its own snapshot, including failures
before render, rollback and quarantine. Live verification is a new operation,
never the Apply timestamp relabeled as a fresh read.

This is a **local diagnostic sibling of D-106**, not a new HTTP field.
`TCReadbackEvidence()`, its wire schema, digest projection and `read_at` are
unchanged. The netlab CLI exports the sibling as `timing` in apply/verify output
and its buffered Apply JSONL. Agent-core should serialize this same value in
its opt-in measurement surface after the call returns, retaining every Apply
and live read rather than only the latest heartbeat. No logging or callbacks
run inside the guard bracket. Normal production observations remain D-106.

`TimingEvidence` version 1 fields:

| JSON field | Meaning |
| --- | --- |
| `version`, `clock` | `1`, `CLOCK_MONOTONIC` |
| `operation` | `apply` or `verify` |
| `candidate_generation` | Decimal string; candidate, not applied-generation acknowledgement |
| `start_ns`, `end_ns` | Operation entry and final return-side sample, including failed cleanup time |
| `succeeded` | Returned error is nil |
| `verified` | Apply's serving result or the live verifier result; a conntrack-delete error may leave Apply verified |
| `deadline` | Apply timing below; zero values for standalone verify |
| `verifications` | Ordered actual verifier invocations; zero before verification, normally one, two during successful quarantine recovery |

All nanoseconds are **decimal JSON strings**, including durations and absolute
samples; never pass through a JavaScript Number. They are Linux
CLOCK_MONOTONIC samples comparable only in the same boot and time namespace.
The collector must retain boot/run identity and a wall/monotonic anchor for
cross-process packet correlation; these are not signed server timestamps.
Zero means the stage was not reached or its clock sample was unavailable.
Consumers must require positive, ordered endpoints and command/result flags;
zero or absent evidence cannot establish continuity. Suspend/resume and other
platform clock domains remain separate acceptance work.

## Apply and guard

`deadline` is also returned unchanged as `ApplyResult.Deadline`:

| JSON field | Exact sample / outcome |
| --- | --- |
| `granularity_ns` | D-104 timeout quantum, 1 ms |
| `install_bound_ns` | D-104 final-render-to-install allowance, 100 ms |
| `apply_start_ns` | Entry before validation, existing-state reads and shaping |
| `render_start_ns` | Final render/time-budget sample after tc shaping, before advancing the model anchor and calling Render; not the earlier planning render |
| `install_start_ns` | Immediately before Executor.NftBatch for the already-rendered guarded candidate |
| `installed_by_ns` | Immediately after that call returns, before context cleanup; an upper bound only when `install_ok=true` |
| `install_ok` | Candidate/guard command returned nil; does not prove budget compliance or serving |
| `release_start_ns`, `release_end_ns` | Immediately around releaseDeadlineGuard's Executor.NftBatch |
| `guard_released` | Release command returned nil; not a claim of later serving verification |

Both commands execute `nft -f -` through System. Install stdin contains the
candidate and `fv_deadline_guard` atomically; release stdin is exactly
`delete chain inet fvpn fv_deadline_guard\n`. No raw script, selector, packet
content, identity or executor error text is included in timing evidence.

For successful install/release, the outer bracket is
`[install_start_ns, release_end_ns]`; the definitely guarded inner interval is
`[installed_by_ns, release_start_ns]`. Kernel commits occur somewhere in their
command brackets. Failed install/release and rollback must not be treated as a
completed outer bracket. An overrun retains install timing, keeps
`guard_released=false`, leaves release samples zero, and preserves the failure
regardless of rollback success. Do not infer a guard end from Apply end.

D-104 uses `installed_by_ns - render_start_ns <= install_bound_ns`.
D-120 additionally bounds the **outer guard** for its particular AT10 stream;
that rule is not a loss tolerance for isolation's 80 ms renewal diagnostic.

## Verifier and precision executor reads

Each `verifications[]` item has:

- `start_ns`, `end_ns`: the complete verifier, including deferred policy reads
  and D-106 retention. The end precedes any caller rollback/quarantine.
- `nft_read.{start_ns,end_ns,ok}`: immediately around the one
  `Executor.Output(ctx, "nft", "-j", "list", "ruleset")` call. `ok` is command
  completion, not ruleset validation.
- `precision_read.{start_ns,end_ns,elements,read_ok,verified}`: immediately
  around `Executor.Output(ctx, "nft", "list", "set", "inet", "fvpn",
  "allowed_v4")`. This text read retains millisecond precision on nft 1.0.9;
  JSON's rounded seconds are not substituted. `read_ok` describes the command;
  `verified` describes parsing and lifetime checks. A failed ruleset check
  skips this stage (all zero/false). A malformed listing can have read_ok=true
  and verified=false. No address or raw listing is retained. `elements` counts
  the parsed/checked listing as in `ServingEvidence.DeadlineRead`.
- `tc_read.{start_ns,end_ns,ok}`: the aggregate readServingTC call covering its
  qdisc/class/filter executor reads and parsing, before authorization/QoS
  validation; it is not a single tc command or atomic kernel snapshot.
- `tc_sha256`: the matching D-106 `{reads,objects,truncated,tc}` digest.
- `verified`: complete verifier result, not only precision/tc success.

Apply's two recovery verifier items remain separate (link down, then link up).
The final D-106 evidence corresponds to the last returned verifier when one was
reached; failures before the second verifier still preserve the first timing.
Standalone live verification has one item and no invented Apply/guard samples.
Pre-Apply ownership/load failures retain their original D-106 phase and a
failed Apply timing record with no verifier item.

## Agent-core / isolation consumption

Retain `TimingEvidence` from every actual SystemPlane Apply and VerifyServing
with the operation result, immutable run/boot identity and capture-clock
anchor. Keep Apply, live verification and historical rejected evidence separate;
a clean deny-all verifier does not imply member forwarding. Feed the matching
precision sample/digest to isolation, together with the node-held absolute
lease deadline and actual renewal transaction records.

The 17 losses from run `7779d83c` cannot be retrospectively assigned new
producer timestamps. Its passive netlink receipts are delivery bounds. A new
assembled run must correlate packet ingress against these genuine brackets;
missing/ambiguous matches remain unproved. Do not apply D-120's allowance to
that independent diagnostic or erase the historical losses.
