# Follow-up 18 reader compatibility audit

Orchestrator note 1, 2026-11-03 15:58 EST, authorizes the reader correction and one second assembled run. Audited against `main` commit `27ff63928222c9642e4f5e180cf225108327a1e5`, which is also the unchanged worktree baseline. No production, timeout, case-order, D-146 or other acceptance predicate changed.

**One mismatch fixed:** `validateCommandLifecycle` omitted the production `retire` phase. `node/enforce/apply.go` emits it for the tc retirement batch after the nft install, introduced by netlab 32 (#423). The phase now goes through the same kind, timestamp order, Apply bounds, cancellation, child-start, CPU and context-switch checks as every other command. Unknown phases still reject the entire stream.

| Vocabulary | Production source and emitted values | Reader disposition |
|---|---|---|
| Command kinds | `node/enforce/command_lifecycle.go`: nft, tc, conntrack, ip, other | Exact existing five-value list matches; unchanged. |
| Command phases | `node/enforce/command_lifecycle.go` initializes prepare; `node/enforce/apply.go` emits quarantine, shape, install, retire, release, verify, revoke, rollback | Added retire only; all nine phases now match. |
| Timing operations | `node/enforce/timing.go`, `apply.go`, `verify_tc.go`: apply, verify | Command lifecycle exists on Apply; verify records without commands remain accepted. Apply-only attribution remains an explicit diagnostic selection. |
| Measurement sources | `node/agent/measurement.go`, `engine.go`, `serving.go`: apply, live, blocked_audit, baseline, renewal, heartbeat | Stream/counter readers retain raw JSON and do not reject sources using a closed list. Readiness selects the typed heartbeat/lease fields; recovery selects live verification separately. No mismatch. |
| Error stages | Agent emits apply, preflight, lease, qos, probe, identity, isolation, meter, usage and diagnostic tc/quarantine stages | Typed report decoding preserves stages. D-067's apply/preflight safety cause and lease-stage check, and the generation-hold lease-stage check, are acceptance predicates rather than exhaustive parser lists. Diagnostic tc/quarantine history does not substitute for an active safety cause. Unchanged. |
| Check names/statuses | `node/agent/decide.go`: dns_policy, internet_only, lease, meter_path, policy_signature, qos, release_supported, source_binding, storage, tunnel_probe; generated API check/status types | No local exhaustive decoder list. D-067 selects lease/qos/source_binding; hold checks select qos/source_binding. Other checks remain in original reports. Unchanged. |
| Baseline causes and resource/context states | `node/agent/measurement.go`, `node/enforce/command_lifecycle.go` | Raw strings retained; apply_rollback/policy_missing are diagnostic selections, not parser rejection lists. No mismatch. |

Reviewed `command_lifecycle_test.go`, `timing_stream_test.go`, `timing_export_test.go`, `counter_timing_test.go`, `apply_attribution_test.go`, `readiness_test.go`, `forwarding_test.go`, `provisional_test.go`, `generation_hold_test.go`, the probe readers, and current Python evidence consumers. The historical run-1 rejection analysis intentionally retains the old phase list and remains bound to its measured harness and immutable archive; it is not run 2's validator.

Tests cover a synthetic successful Apply with nft install followed by tc retire and guard release, unknown phase/kind rejection of the complete stream, and unchanged lifecycle rejection for invalid retire timestamps, invented exact child start, negative CPU and invalid cancellation. A separate regression test parses the exact original sequence-582 record from [run 1's byte-pinned evidence](history/follow-up-18-run1/F18-lifecycle-phase-compatibility.json).

[Run 1 archive](history/follow-up-18-run1/INDEX.md) preserves all observed failures and the 16 exact export-prefix joins. Correcting the reader does not rewrite any historical outcome or prove that every run-1 failure had this cause.

Run 2 live confirmation: [the original AT02 trace](history/follow-up-18-run2/AT02-enforcement-timing.json) retains 1,534 accepted producer records, including `sequence=584`, `timing.commands[28]` with `kind=tc` and `phase=retire`. The original counter-reader enforcement-attribution check passes. This confirms compatibility on this run; the unchanged D-135/D-136 availability assertions still fail.
