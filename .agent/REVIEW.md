# Adversarial Review r19: intersection serving (D-139) and the follow-up 33 subset reference

Branch: `r19-intersection-design-review` (from `main` at `6eda25d0`)
Date: 2026-11-01
Reviewing model: `grok/grok-4.7`

Scope is D-139's provisional option (b), the design note `docs/node/followup33-intersection-serving-design-2026-11-01.md` (agent-core follow-up 33, #379), its test-only reference `node/agent/followup33_intersection_spec_test.go`, and the production node and central code that design would change. Wire-nodes follow-up 34 (#382, `6eda25d0`) is the stale-lease hold the design says partial service must stay distinct from. Option (c) stays in production: under D-135 / D-136 a restrictive generation change baselines the whole node until the next successful Apply. r18's "What held" is the premise of that window.

Proofs are `TestReviewR19_*`. A `TODO(review)` failure is a finding. A passing test is recorded under What held. Product code, the design note, and the reference were not changed.

Commands (2026-11-01, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, go1.27.3, `TMPDIR` and `GOTMPDIR` under `.agent/tmp`, `GOCACHE` from `go env GOCACHE`). No integration database was opened. The shared `fvpn-pg` was not touched.

Before the lab, `pgrep -af 'gates.sh|mk/at-'` showed two other suites (`mk/at-members.mk` in `w4-at-members`, and `bash .agent/tmp/followup26/gates.sh` in `w3-devstack-e2e`). The lab waited until only the gates script remained. The lab used `FVPN_LAB_PREFIX=fvpn-r19` on bridge `fvpn-r19-dns`. `netlab-down` removed the container, the network, and the scratch directory. The `fvpn-r19:lab` image was removed afterwards. No `fvpn-r19` container, network, or image remained.

```
go test -count=1 -timeout 240s -v -run 'TestReviewR19_' ./node/agent/ ./node/enforce/ ./node/watchdog/ ./internal/nodes/ ./internal/desiredstate/
go test -count=1 -timeout 120s -run '^TestF33_' ./node/agent
make -f mk/netlab.mk netlab-up FVPN_LAB_PREFIX=fvpn-r19
FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-r19 go test -tags netlab -count=1 -timeout 240s -v ./tests/netlab -run '^TestReviewR19_'
make -f mk/netlab.mk netlab-down FVPN_LAB_PREFIX=fvpn-r19
```

The review unit command exited 1. `node/agent` 2.075s, `node/enforce` 0.005s, `node/watchdog` 0.016s, `internal/nodes` 0.008s, `internal/desiredstate` 0.007s. `TestF33_` exited 0 (0.014s). The netlab test exited 1 in 3.63s on the `TODO(review)` line quoted under H-1.

## Summary

| ID | Severity | Owner | Where | Summary |
|---|---|---|---|---|
| H-1 | High | netlab, agent-core | `node/enforce/render_nft.go:189-203`, `node/enforce/apply.go:149-183` | A destination added to `denied_v4` drops new upload and leaves the established return forwarded. The conntrack entry stays. Today's `Render` also deletes the forward chain and flushes `allowed_v4`. |
| H-2 | High | agent-core | `node/agent/observe.go:286-289` | HTTP 422 is `MarkConflict`. The serving ruleset stays up. The design's unsupported-report response is supposed to abandon I and baseline. |
| M-1 | Medium | agent-core | `node/watchdog/heartbeat.go:57-83` | A v1 heartbeat with `serving=1`, `generation=4`, and `reason=generation_intersection` is a full generation-4 read. `Once` does not baseline. |
| M-2 | Medium | agent-core | `node/agent/observe.go:72-75`, `node/agent/observe.go:111-122` | After a restrictive document is accepted and before withdrawal, the report ACKs the old generation and copies policy refs and the binding revision from the live desired document, with forwarding still true. |
| M-3 | Medium | agent-core, wire-nodes, contracts | `node/agent/followup33_intersection_spec_test.go:107`, `:229`, `:274-276` | The reference admits a `MaxUint64` node ceil, keeps a resolver that is also denied, and treats `AtomicPolicy` as a boolean with no rate mode, selection interval, or witness snapshot. |
| M-4 | Medium | agent-core | `node/agent/followup33_intersection_spec_test.go:305` | Recomputing S with a later lease anchor and a rebuilt installed deadline moves D from 1m41s to 1m50s. |
| M-5 | Medium | installer | `node/updater/apply.go:308-318` | `stepRollback` stops services and starts the restored release. It does not install a baseline or rewrite the heartbeat before that process runs. |
| M-6 | Medium | wire-nodes, orchestrator | `internal/desiredstate/compiler.go:523-553` | `binding_revision` is the maximum included device revision. Desired state has no intersection witness. None of the 80 follow-up 31 transitions qualify for S. |
| M-7 | Medium | wire-nodes, contracts | `internal/nodes/observations.go:63-66`, `internal/nodes/publication_evidence.go:24-66` | `generation_intersection` is rejected before SQL. Stripped of that token, the proposed report is `check_failed`, not the follow-up 34 hold, and no partial-service cause exists. |
| M-8 | Medium | netlab | `node/enforce/render_nft.go:88-96`, `node/enforce/deadline.go:32` | A render against an existing `allowed_v4` flushes it and writes the model timeout. `State` has no live element lifetime. |
| L-1 | Low | agent-core | `node/agent/followup33_intersection_spec_test.go:342-356` | `requires_tc` sets `Baseline` and still returns the candidate bindings. |
| L-2 | Low | agent-core, netlab | `node/enforce/apply.go:418`, `node/agent/engine.go:1146-1159` | `conntrack -D` runs inside Apply. The 12-attempt counter is incremented and persisted only after Apply returns. |
| L-3 | Low | netlab, contracts | `node/enforce/evidence.go` `retainTC` | Generations 4 and 5 of the same tc objects share one sha256. `candidate_generation` is copied beside the digest. |
| L-4 | Low | agent-core | design note §2, the B row | "No return to I" is the state-table sentence. The reference has no state machine that enforces it. |

## Findings

### H-1 (High): an established download survives a new destination denial

- **Where:** The download path drops `ip daddr != @allowed_v4`, then drops `ct state != { established, related }`, then accepts `ip daddr <member>` with no remote-source match (`node/enforce/render_nft.go:181-203`). Upload drops `ip daddr @denied_v4` before the per-source accept (`node/enforce/render_nft.go:162`, `:177`). `Apply` runs `TCBatch` before `NftBatch`, and a failed nft install calls `installBaseline`, which removes the qdisc (`node/enforce/apply.go:149-183`). `replaceChain` deletes an existing forward chain inside the same nft script (`node/enforce/render_nft.go:481-485`).
- **Scenario:** Member `100.64.1.1` has a UDP flow to `203.0.113.10:9000`. Generation 4201 adds `203.0.113.10/32` to `ExtraDeniedV4` and leaves `Revoke` empty, so Apply does not call `conntrack -D`. The ruleset stays at `fvpn_generation=4201`. The conntrack entry remains. Uplink packets to that destination stop (`ctdeny` increases). Injected replies from `203.0.113.10` still arrive on the member path. A sibling TCP connection from `fv-a2` to `203.0.113.20:80` still completes. Source removal is a different rule: the download allowed-set drop is before the established check, so deleting the member address stops both directions without conntrack deletion. That source case is the existing `TestRevoke_RulesetStopsEstablished`, which this review did not re-run.
- **Proof:** `TestReviewR19_Q4_EstablishedDownloadSurvivesDestinationDeny` on `fvpn-r19` (kernel), and `TestReviewR19_Q4_CurrentRenderDoesNotRestrictEstablishedDownload` (render).

  ```
  TODO(review): generation 4201 kept the conntrack entry, dropped uplink packets (uplink=0 ctdeny=4) and forwarded the established return (return=4 injected=4). Sibling 203.0.113.20 still connects. Download accept does not match the new denial
  ```

  ```
  TODO(review): download accept follows the established drop and does not match @denied_v4 (add rule inet fvpn forward iifname "fv-ih" oifname "fv-fh" ip daddr 100.64.1.1 counter name "d_019a00000000700080000000000000a1" meta priority set 0x0001002f accept). Upload deny is an earlier drop. A render against an existing forward chain deletes that chain and flushes allowed_v4. Apply calls TCBatch before NftBatch
  ```
- **Fix:** netlab and agent-core. E needs its own nft transaction. On the download path, drop `ip saddr @denied_v4` before the established/related accept, in the same transaction that removes upload authorization. Do not call `Apply` or `Render` for that transaction: those delete the forward chain, flush set timeouts (M-8), shape tc first, and baseline the qdisc if nft fails. Conntrack deletion is not a substitute that can be deferred for this download gap.

### H-2 (High): a rejected observation leaves the serving ruleset in place

- **Where:** `flush` treats HTTP 409 and 422 as `MarkConflict` and continues (`node/agent/observe.go:286-289`). It does not call `baseline`. The design says an old strict schema that rejects the extra token must make the node abandon I and baseline (`docs/node/followup33-intersection-serving-design-2026-11-01.md:537-542`).
- **Scenario:** The node is serving. A real `report()` payload is queued and posted. The control plane answers 422 `VALIDATION_FAILED`. One POST happens. The outbox row leaves pending. Baseline count and `serving` stay where they were (`baselines=4`, `serving=true`). Reusing `flush` for an intersection report would keep S forwarding after central refused the only evidence of partial service.
- **Proof:** `TestReviewR19_Q3_UnsupportedObservationDoesNotBaseline`.

  ```
  TODO(review): HTTP 422 was MarkConflict and the serving ruleset stayed up (posts=1 baselines=4 serving=true). observe.go flush does not abandon an intersection or install the baseline
  ```
- **Fix:** agent-core. On an unsupported-report response while an intersection is installed, baseline and clear the partial-service state. Keep a transport failure distinct from 422 so a network error does not revoke an already accepted S. Do not enable the feature until that branch exists (Q5).

### M-1 (Medium): a v1 heartbeat reports an intersection as full service

- **Where:** `WriteHeartbeat` always starts with `v1` (`node/watchdog/heartbeat.go:57`). `ReadHeartbeat` requires that first line (`node/watchdog/heartbeat.go:81-83`). Extra keys are ignored. A reason with no spaces is stored as written.
- **Scenario:** The file says `serving=1`, `generation=4`, `reason=generation_intersection`, plus extra `transition` and `subset` lines. `ReadHeartbeat` succeeds. `Once` does not baseline. A file whose first line is `v2` is `heartbeat_unreadable`, and `Once` baselines once. An old watchdog beside a new agent that still writes v1 will treat S as full generation g whenever the agent marks serving.
- **Proof:** `TestReviewR19_Q3_V1HeartbeatTreatsIntersectionAsFullService`. `TestReviewR19_Q3_WriteHeartbeatStaysV1` passed: the writer still emits `v1`.

  ```
  TODO(review): ReadHeartbeat accepted a v1 file with reason generation_intersection, serving=true and generation=4, and Once left the ruleset alone. Extra transition lines were ignored. A v2 first line is heartbeat_unreadable and Once baselined
  ```
- **Fix:** agent-core. Ship the v2 heartbeat, and teach the watchdog to baseline on an intersection marker, before the feature can be turned on. A v1 serving file must not be how S is reported.

### M-2 (Medium): the g ACK copies the new desired document's policy refs

- **Where:** When the node is not frozen and `appliedGen > 0`, `report` sets the applied generation from `e.appliedGen` and copies `AppliedBindingRevision` and `AppliedPolicyRefs` from `e.desired` (`node/agent/observe.go:72-75`, `:111-122`). `MemberForwardingEnabled` is `e.serving`.
- **Scenario:** Generation 3 is serving. Generation 4 is accepted and not yet enforced, so the old prefix is still installed. The test then sets the live desired DNS generation to 999 and the binding revision to 4242. The report ACKs generation 3, stamps those two desired fields, and leaves `member_forwarding_enabled=true`.
- **Proof:** `TestReviewR19_Q3_ReportStampsLiveDesiredRefs`.

  ```
  TODO(review): after accepting generation 4 and before withdrawal, report ACK applied=3 stamps desired DNS generation 999 and binding revision 4242 with member_forwarding_enabled=true. Engine.report reads e.desired, not the retained applied snapshot
  ```
- **Fix:** agent-core. An intersection report has to pin the refs, the binding revision, and the verifier inputs to the subset that was actually installed. Copying the live desired document onto an ACK of g reports N's policy as applied.

### M-3 (Medium): the subset reference admits inputs §3.2 says must baseline

- **Where:** `f33CeilOK` is `Up > 0 && Down > 0` (`node/agent/followup33_intersection_spec_test.go:107`). Resolver admission is address intersection only (`:274-276`). `AtomicPolicy` is a caller boolean on `f33Input` (`:79`, gate at `:229`). `f33Policy` has no mode, rate guarantee, selection interval, or snapshot hash.
- **Scenario:** Three independent inputs each return a non-baseline result. A node ceil of `18446744073709551615` in the old, new, and installed shapes is admitted. Resolver `192.0.2.53` stays in `Candidate.Resolvers` after that address is added to `Denied`, and the result has no port, so an all-ports accept of the resolver address is not rejected. Unchanged ceils with `AtomicPolicy=true` keep devices `ab`. A shadow-to-enforce tightening, a second rate interval, or a witness from another snapshot that reuses entitlement revision 11 is invisible. The header of the reference already says signature checking and kernel realization are prerequisites. These three inputs are inside the function the header does claim to decide.
- **Proof:** `TestReviewR19_Q2_ReferenceAdmitsUnboundedCeil`, `TestReviewR19_Q2_ReferenceKeepsDeniedResolver`, `TestReviewR19_Q2_ReferenceIgnoresRateModeAndWitnessSnapshot`.

  ```
  TODO(review): f33Intersection admits node ceil 18446744073709551615. f33CeilOK only requires Up>0 and Down>0. A missing direction that nodeCapacityRatePair encodes as math.MaxInt64 is a real ceiling, not the node cap
  ```

  ```
  TODO(review): Baseline=false with resolver 192.0.2.53 also present in Denied. The result has no port. An all-ports accept of Candidate.Resolvers opens more than TCP/UDP 53
  ```

  ```
  TODO(review): unchanged ceils return devices ab with AtomicPolicy=true. f33Policy has no mode, rate guarantee, selection interval, or witness snapshot hash. A shadow-to-enforce tightening, a second rate interval, or a stale witness that reuses entitlement revision 11 is admitted
  ```
- **Fix:** agent-core for the reference, wire-nodes for the compiler, contracts for the signed snapshot. Reject a non-finite ceil. Encode the DNS exception as TCP/UDP 53 to a resolver that is not denied. Bind the witness to the desired snapshot hash and fail closed when member-rate mode, the guarantee, or the selection interval can change before the next full Apply. A compiler that stamps a stale entitlement revision with a true atomic bit must not be admitted.

### M-4 (Medium): a later anchor extends D

- **Where:** The admitted deadline is `min(lease.Anchor+window, installed deadline)` (`node/agent/followup33_intersection_spec_test.go:305`). Nothing pins the original lease issue time against a replay of that computation.
- **Scenario:** The fixture anchor is 1s and the installed deadlines are 101s, so D is 1m41s. The same identities with anchor 10s, now 11s, and installed deadlines of 110s produce D of 1m50s. The design says a later response, render, retry, or fresh g lease cannot move D forward.
- **Proof:** `TestReviewR19_Q1_ReferenceReplayExtendsDeadline`. This is a test-only reference result. Production has no intersection to prolong.

  ```
  TODO(review): recomputing S with Anchor 1s->10s and a rebuilt installed deadline moved D from 1m41s to 1m50s. The reference does not pin L
  ```
- **Fix:** agent-core. Pin D to the original same-boot lease anchor and the installed deadline captured at the first eligible O. A caller-supplied later anchor must baseline.

### M-5 (Medium): rollback starts the old binary without a baseline

- **Where:** On the destructive path, `stepRollback` stops services and starts `e.Layout.releaseDir(target)` (`node/updater/apply.go:308-318`). The function body contains neither a baseline call nor an nft invocation. The meter-wait failure path returns before that and preserves the baseline by not restarting.
- **Scenario:** The feature is on, S is installed, and the heartbeat is still a v1 serving file at generation g (M-1). An updater rollback starts the old binary while that ruleset and heartbeat are still in place. The old `Boot` can adopt a stored model it can verify. The design says rollback disables the feature and baselines before old binaries run. This function does not.
- **Proof:** `TestReviewR19_Q5_RollbackSourceDoesNotBaseline`. The updater was not executed. The proof is the function body.

  ```
  TODO(review): stepRollback stops services, switches the symlink, and starts the restored release. It does not install a baseline or rewrite the heartbeat before the old binary runs
  ```
- **Fix:** installer. Baseline and clear the serving heartbeat before the restored release is started, and refuse to roll back onto a binary that does not understand the feature while the feature flag is on. There is no feature flag in the current updater.

### M-6 (Medium): no merged restrictive change qualifies for S

- **Where:** The compiler stores `maxBindingRev`, the maximum `BindingRevision` among included devices, as `DesiredState.binding_revision` (`internal/desiredstate/compiler.go:523-524`, `:553`). `api.DesiredState` has no `IntersectionConstraints` field. `MemberLease.EntitlementRevision` exists on the lease and is not copied onto that desired document.
- **Scenario:** A two-device set whose revisions are 9 and 12 stays at 12 when the lower device is removed, and falls to 9 when the higher device is removed. That field is not a monotonic set generation. Follow-up 31 recorded 40 transitions in `generations-final/analysis.json` and 40 in `generations/analysis.json`. Every one of the 80 has a non-empty `baseline_sequences`. Nearest-rank baseline-to-apply on the final population is 173766526 ns and 195601257 ns, which is the published 173.766526 ms / 195.601257 ms table. Members run 258ba504 records 58 baseline calls and one transition of 215.189032 ms. The analysis files do not carry DNS, fair-use, or topology diffs, so this review does not invent a share of "pure revocations that would qualify if a witness existed." Against the documents that exist, the share is 0.
- **Proof:** `TestReviewR19_Q2_BindingRevisionIsTheMaxDeviceRevision`, `TestReviewR19_Q2_DesiredStateHasNoWitness` (pass: the field is absent and the compiler source still takes the max), and `TestReviewR19_Q4_QualifyingShareOnMergedEvidence`.

  ```
  TODO(review): DesiredState.binding_revision is maxBindingRev, not a monotonic set generation. A set whose revisions are 9 and 12 stays at 12 after the lower device is removed, and falls to 9 after the higher device is removed
  ```

  ```
  TODO(review): qualifying share for S on today's documents is 0 of 80 follow-up 31 transitions (40+40, every one baselined). generations-final nearest-rank baseline-to-apply p50=173766526 ns p99=195601257 ns. Members run 258ba504 records 58 AT02 baselines and one 215.189032 ms baseline-to-Apply. DesiredState has no intersection witness, and compiler BindingRevision is the max device revision
  ```
- **Fix:** wire-nodes and orchestrator. Do not map `binding_revision` onto the design's set revision. Emit a monotonic set revision and a signed per-member witness, or keep (c). The reference itself baselines a same-max removal and a decreased max (What held §2); the gap is that nothing in today's documents can take the qualifying path.

### M-7 (Medium): central has no partial-service cause, and the proposed token is rejected

- **Where:** `SubmitObservation` rejects a forwarding-block reason that fails `Valid()` before `commandtx.InTx` (`internal/nodes/observations.go:63-66`). `reportsGenerationHold` requires `policy_missing` plus a lease check or a lease-stage error of `STALE_GENERATION` (`internal/nodes/generation_hold.go:18-32`). `publicationForwardingFailure` returns `check_failed` on any failed check before it reads the forwarding bit (`internal/nodes/publication_evidence.go:96-105`). `ServingReadiness.IsServing` compares applied and desired generations and does not read the forwarding bit (`internal/nodes/transitions.go:125-133`). `publication_evidence.go` does not contain `generation_intersection`.
- **Scenario:** The proposed report carries `generation_intersection`, a failed lease check `STALE_GENERATION`, a `stage=lease` error of that code, applied generation 4, and forwarding true. `SubmitObservation` on a `Service` with a nil pool returns `VALIDATION_FAILED` and does not reach SQL. Stripping the token leaves `reportsGenerationHold` false. Adding `policy_missing` makes the hold true. `publicationForwardingFailure` on the stripped report returns `check_failed`. `IsServing` is false for applied 4 against desired 5, and true for the same readiness with applied 5. A report whose only positive signal is `member_forwarding_enabled=true` still has a non-empty forwarding-failure cause. The non-hold SQL path of `PublicationDataPathFailure` was not executed. That path returns `stale_generation` when applied is below desired (`publication_evidence.go:61-62`), the same cause string as ordinary lag and as the hold's success path.
- **Proof:** `TestReviewR19_Q3_ProposedReportIsRejectedAndIsNotTheHold`.

  ```
  TODO(review): SubmitObservation rejects forwarding_block_reason generation_intersection with VALIDATION_FAILED before SQL. Stripping the token leaves a failed STALE_GENERATION lease check that is check_failed, not reportsGenerationHold, unless policy_missing is added. IsServing is false while applied 4 < desired 5 and does not read the forwarding bit. PublicationDataPathFailure has no distinct partial-service cause
  ```
- **Fix:** wire-nodes and contracts. Add the token and a partial-service cause that is returned before the applied-below-desired early return, and keep it distinct from the follow-up 34 hold. Do not classify on `STALE_GENERATION` alone. The current reject is fail-closed for an old central that sees the token (What held §3). H-2 is the node side of that reject.

### M-8 (Medium): a re-render replaces the kernel element timeout with the model value

- **Where:** `flushOrFill` flushes a set that already exists and then adds elements (`node/enforce/render_nft.go:88-96`). `allowedElements` prints `timeout <milliseconds>` from the model (`node/enforce/render_nft.go:519`). `elementTimeout` takes a binding, a lease, and `now` (`node/enforce/deadline.go:32`). `State` records set presence and `ProbeRemain`. It has no allowed-element lifetime.
- **Scenario:** `smallModel`'s offline cap is 3600s. Rendering with `allowed_v4` already present flushes the set and writes `timeout 3600000ms`. A shorter timeout already installed in the kernel is not an input, so the flush replaces it with the model value. Named counters are a separate path and are kept (What held §4).
- **Proof:** `TestReviewR19_Q4_ShorterKernelTimeoutIsNotAnInput`.

  ```
  TODO(review): with allowed_v4 already present, Render flushes it and writes timeout 3600000ms from the model. State has no live element lifetime, and elementTimeout takes only the binding, the lease and now. A shorter kernel timeout is not an input
  ```
- **Fix:** netlab. The E transaction must not flush `allowed_v4`. Preserve the shorter live timeout, or baseline when the model cannot name it. `Render` cannot be that transaction.

### L-1 (Low): `requires_tc` still returns candidate bindings

- **Where:** `f33Intersection` builds `Candidate` before the tc check and returns that struct with `Baseline=true` and reason `requires_tc` (`node/agent/followup33_intersection_spec_test.go:342-356`).
- **Scenario:** Lowering the new node ceil to 80/150 marks baseline and keeps both candidate bindings. A caller that installs `Candidate` without reading `Baseline` forwards at the mathematical minimum the profile says it cannot enforce.
- **Proof:** `TestReviewR19_Q2_RequiresTCKeepsCandidate`.

  ```
  TODO(review): requires_tc returns Baseline=true and still retains 2 candidate bindings at node ceil 80/150. A caller that installs Candidate without reading Baseline forwards
  ```
- **Fix:** agent-core. A baseline result should carry an empty authorization set. The reference comment already warns; the return value does not enforce it.

### L-2 (Low): the conntrack attempt is counted after the delete

- **Where:** `deleteConntrack` runs `conntrack -D` inside Apply (`node/enforce/apply.go:413-418`). After a verified delete failure, `Engine.enforce` increments `conntrackAttempts` and then calls `persistConntrack` (`node/agent/engine.go:1145-1159`). Apply does not know the attempt counter. A successful apply with nothing pending calls `clearConntrackBlock` (`engine.go:1160-1164`).
- **Scenario:** The design persists a reservation before each batch so a crash cannot under-count the 12. The current order writes the count after the syscall. A crash between `conntrack -D` and the meta write leaves the stored count low. Restart does not zero a count that was stored (What held §5). Twelve attempts are an attempt count. The comment that this is about two minutes at a 10s step is not a wall-clock guarantee.
- **Proof:** `TestReviewR19_Q4_ConntrackAttemptIsCountedAfterTheDelete`. The crash itself was not injected.

  ```
  TODO(review): conntrack -D runs inside Apply and engine.go increments conntrackAttempts only after Apply returns, then persistConntrack. A crash between the delete and the meta write under-counts the 12-attempt bound. No reservation is persisted before the batch
  ```
- **Fix:** agent-core and netlab. Persist the reservation before the delete batch, and do not clear it because the generation changed while addresses remain.

### L-3 (Low): the tc digest does not cover candidate generation

- **Where:** `retainTC` hashes reads, objects, truncated, and tc. `CandidateGeneration` is stored on the evidence beside that digest. The same objects at generations 4 and 5 share sha256 `05dfb5a07722626aee5367df21db856f1b680224fbd8a9a28240ef5fe765db40`.
- **Scenario:** An S read and a later full read of the same tc objects are identical at the digest. `candidate_generation` can say `h` while the hash is the S read. The OpenAPI comment says that field is not an ACK. This review found no production classifier that treats it as a full Apply of h. Confusion requires a consumer that does.
- **Proof:** `TestReviewR19_Q3_TCDigestIgnoresCandidateGeneration`.

  ```
  TODO(review): retainTC generations 4 and 5 share sha256 05dfb5a07722626aee5367df21db856f1b680224fbd8a9a28240ef5fe765db40. evidence.go hashes reads, objects, truncated and tc, and copies Generation into candidate_generation separately
  ```
- **Fix:** netlab and contracts. Keep the digest stable, and require the intersection object, not `candidate_generation`, before anyone reads a digest as full generation h.

### L-4 (Low): "B never returns to I" is only stated

- **Where:** The B row of the state table says "No return to I." (`docs/node/followup33-intersection-serving-design-2026-11-01.md:75`). The reference has no state machine. No production transition returns from baseline to an intersection, because intersection is not implemented.
- **Scenario:** A later edit can add a B→I edge without failing the reference. The sentence is present. Nothing executable forbids the edge.
- **Proof:** `TestReviewR19_Q4_D133TextRequiresCeilings` locks the sentence and passed. The absence of a transition function is this finding, from reading the reference. There is no failing test that executes a forbidden edge.

- **Fix:** agent-core. When the state machine is written, make B, Q, and X unable to enter I, including across restart, and test that transition table.

## Recommendation

Note Q1 (exact-revision rule and signed witness): approve with changes. The reference's identity table excludes a changed device, peer, address, member, or revision, and it does not admit a new device. Approve that rule only together with a witness bound to the snapshot. `AtomicPolicy` as a boolean, a `MaxUint64` ceil, a denied resolver with no port, and today's max `binding_revision` are not that witness.

Note Q2 (observation, ACK, D-106): approve with changes. Keep the ACK at g, add a partial-service cause distinct from the follow-up 34 hold, baseline on 422, and ship a v2 heartbeat before enablement. Do not treat `candidate_generation` or `member_forwarding_enabled` as full service.

Note Q3 (nft-only E, no later than (c)): approve with changes. A new nft transaction can drop denied remote sources before the download established accept and can leave tc alone. `Render` and `Apply` cannot be that transaction. The kernel forwards the established return today (H-1).

Note Q4 (cost): reject shipping on this evidence. Qualifying share on the merged documents is 0 of 80. The promotion baseline is the same order as today's baseline-to-Apply (173.766526 ms / 195.601257 ms in the final population; 215.189032 ms on the recorded AT02 transition). The D-133 text would reject a higher ceiling and an equal candidate. The trial has not been run.

Note Q5 (rollout and rollback): approve with changes, disabled by default, and only after readers and the updater change. `stepRollback` does not baseline before the old binary. A v1 heartbeat and a 422 response both leave S looking like service.

Option (b) as a whole: reject implementation. Keep option (c).

## What held

### 1. Revocation safety of the state machine, against today's Step

`TestReviewR19_Q1_BoundaryIsBeforeSecondSyncLease`, `TestReviewR19_Q1_RetainedErrorRenewsBeforeWithdrawal`, `TestReviewR19_Q1_CorruptDesiredDoesNotInstall`, and `TestReviewR19_Q1_DeadlineReserveBoundary` passed.

On the ordinary path, `syncDesired` of a restrictive generation leaves the previous prefix installed, `needRenew` is false, and the following `syncLease` does not renew or baseline. `evaluate` plus `enforce` then baselines and the prefix is gone. That is option (c), and it is the boundary the design names: after desired acceptance and before `Engine.enforce`. The second `syncLease`, certificate, bootstrap, and meter reads sit in `Step` after `syncDesired` and before `enforce` (`node/agent/engine.go:586-630`). Probes do not. `answerHealth` runs at the end of `Step` (`engine.go:640`). The design sentence at note line 53 that lists probes in that window does not match this order. Inserting E immediately after acceptance does not require moving probe work.

On the retained-error path, `needRenew` is true and `syncLease` completes a renewal while the old prefix is still installed. (c) withdraws later, inside `enforce`. An E placed in front of that second `syncLease` is earlier than (c). An E placed after `TCBatch`, after the serving path's node-lock retries (`engine.go:987`, 3 attempts of 50ms), or only after a failed nft install, is where (c) would already have withdrawn: the non-serving branch baselines without that lock (`engine.go:974-980`).

A corrupted desired signature does not install and does not move the applied generation. Production cannot create an intersection from a replay of L, of N, or of an older desired document, because the feature is not in the binary. `verifyDesired` runs before `AcceptDesired`.

The reference's reserve boundary holds. Remaining equal to `DeadlineInstallBound` (100ms), and remaining equal to 100ms plus 1ns, baselines. Remaining of 101ms admits devices `ab`.

There is no implemented P/K/I/A/B/Q/X, so there is no production path through those states that forwards a binding N removed. The reference exclusions for those identities are §2. Graceful stop of a full serving generation still succeeds at `preserveServingOnStop` (§5). The design's ban on calling that helper from P/K/I/A has no code yet. Supersession by j>h, an empty grant, and an authenticated lease for the wrong generation are design-only. The reference has no transition that performs them, and production baselines instead.

`recoveryDecision` still clears bindings on a copy and is used only when `applyError` is set and the decision would not serve (`engine.go:963-971`). It does not authorize members. r18 already recorded the counterfactual. This review did not re-execute that copy.

### 2. The subset rule's identity table

`TestReviewR19_Q2_ReferenceIdentityExcludesChangedDevice` passed for peer, address, device, member, revision, added device, same set revision with a removal, decreased set revision, zero ceil, and an unverified input. `TestReviewR19_Q2_DesiredStateHasNoWitness` passed. `TestF33_` exited 0.

A device whose member, device, peer, address set, or per-binding revision changes is dropped. A newly added device is not admitted. Removing a device while the set revision stays the same baselines. Removing the device that held the maximum revision makes the compiler's published number fall, and the reference baselines that decrease. Zero ceil baselines. An unverified input baselines. Member-ID intersection would be unsafe; the reference does not do it. The cases the reference gets wrong are M-3, M-4, and L-1, not this identity table.

`TestReviewR19_Q4_D133TextRequiresCeilings` passed. The note still contains "Any higher or missing", "strictly lower restrictive p90", and "No return to I."

### 3. Classification of the proposed report, as far as the code goes without SQL

The assertions inside `TestReviewR19_Q3_ProposedReportIsRejectedAndIsNotTheHold` held before its `TODO(review)`: the token is `VALIDATION_FAILED` with a nil pool; `STALE_GENERATION` without `policy_missing` is not `reportsGenerationHold`; adding `policy_missing` is the hold; the stripped report's forwarding failure is `check_failed`; `IsServing` is false while applied is below desired and does not read the forwarding bit; `member_forwarding_enabled=true` alone is not an empty forwarding-failure cause. `TestReviewR19_Q3_WriteHeartbeatStaysV1` passed.

Old strict central fails closed by rejecting the token (M-7). The node does not fail closed on that reject (H-2). A permissive reader that ignores an unknown reason still has applied below desired, so `IsServing` is false, and a failed lease check is `check_failed` rather than success. That is not a deployment negotiation, and it is not a distinct partial-service classification. The watchdog fails closed on a v2 file and fails open on a v1 serving file (M-1).

`internal/fleetupdate/installer_results.go:132` and `internal/fleetupdate/results.go:208` AND `MemberForwardingEnabled` with publication and release checks. Those functions were read and not executed.

### 4. What the current transaction already preserves

`TestReviewR19_Q4_ExistingCounterIsNotRecreated` passed. A counter present in `State` is not emitted again as `add counter`. `writeClass` emits `class replace` with both `htb rate` and `ceil` (`node/enforce/render_tc.go:171-174`). The download-render test confirmed those strings are in the tc script.

Upload destination deny is before the per-source accept, so a new denial stops upload, including an established upload, without conntrack deletion. The kernel run measured uplink 0 and `ctdeny` +4 (H-1). Download source removal is the allowed-set drop before the established check. Destination restriction of an established download does not hold (H-1). Timeouts are not preserved (M-8). "Never later than (c)" holds only for an E placed at the boundary in §1, not for `Apply`.

The 12-attempt count survives restart and `clearConntrackBlock` resets it (`TestReviewR19_Q5_ConntrackCountSurvivesRestartAndClearResetsIt`). The reservation-before-batch order does not (L-2).

### 5. Crash, restart, and freeze against current Boot

`TestReviewR19_Q5_CurrentBootAdoptsVerifiedCheckpoint`, `TestReviewR19_Q5_PreserveServingOnStopSucceeds`, `TestReviewR19_Q5_ConntrackCountSurvivesRestartAndClearResetsIt`, and `TestReviewR19_Q5_InFlightObservationBlocksAdoption` passed.

Current `Boot` adopts a verified same-boot checkpoint and returns without baselining (`node/agent/engine.go:416` calls `adoptStoredServing`). An in-flight observation, recorded as an outbox attempt before HTTP (`observe.go:264`), sets `freezeUncertain` on the next boot (`freeze.go` load path) and `StorageOK` is false (`engine.go:775`), so adoption does not serve. `TestD088_LostFreezeResponseBlocksRestartUntilReplay` already covered a lost freeze response; this test re-proves the in-flight row. A completed freeze blocks through the latch. `freezeIdentity` sets the epoch, withdraws, then persists (`freeze.go:16-26`). Its production caller is the observation response (`observe.go:271`). `Step` is single-threaded, so that call cannot preempt an nft syscall already inside `Apply`. `Apply`'s agent path latches once under the node lock before shaping (`engine.go:1005`) and does not re-check the epoch before `NftBatch`. The design's "immediately before every kernel commit" is an implementation constraint on E. It is not a leak in option (c).

`preserveServingOnStop` succeeds for a full serving generation. The design forbids that helper from P/K/I/A. If a future implementation stores S in the g model and then preserves a serving heartbeat, the next `Boot` can adopt S as full g, because the verifier does not know an intersection mode. That adoption path is the passing test. The S model was not installed.

The design's no-adoption rule, including for a completed F, is a new restart cost against (c): every feature-enabled boot would pay a lease round trip and a full Apply. Current code does not do that. The note already lists it under its Q4. It is a cost, not a production defect.

The §4 crash table's v4 rows were not executed. The store schema is design-only. No row of current code adopts S, because S is not representable. No row of current code leaves a revoked binding forwarded across the restart this review ran: the in-flight observation boot stayed closed, and the verified checkpoint boot adopted the full model it had stored.

### 6. Cost, promotion, and the A/B text

The qualifying-share measurement is M-6: 0 of 80 on today's documents. The published final-population nearest-rank baseline-to-apply p50/p99 match 173766526 ns and 195601257 ns. A promotion baseline is another baseline-to-Apply of that order, and the recorded AT02 span is 215.189032 ms. The sibling outage during promotion is the same shape as option (c)'s outage, paid again at the end of a transition that (b) was meant to avoid.

Make-before-break through current `Apply` installs the tc class, rate and ceil included, before nft (`apply.go:149-152`). A looser h ceil would be in the kernel before the restriction. A failed nft install then removes the qdisc. The design's last paragraph of §6 already refuses that optimization. This review did not install such a sequence.

`TestReviewR19_Q4_D133TextRequiresCeilings` passed. The written rule demands strictly lower restrictive p90 and p99, and it fails a higher or missing ceiling even when loss percentiles improve. An equal candidate fails the strictly-lower bar. A percentile improvement that raises a ceiling fails the ceiling sentence. The 40+40 trial has not been run, so the text's ability to detect a regression is unread against a real A/B log.

### 7. Mixed fleets, as far as current readers go

New agent plus old central: the token is rejected (M-7) and the node keeps forwarding (H-2). New agent plus old watchdog: a v1 serving heartbeat is full g (M-1). New central plus old agent: the old agent never emits the token and stays on (c); this review did not need a second binary to see that the token is absent from `WriteHeartbeat` and from `ForwardingBlockReason`. Feature on plus updater rollback: M-5. No combination was executed as a mixed process pair.

## Not examined

- `PublicationDataPathFailure`'s non-hold SQL. No database was opened. The hold predicate, `IsServing`, `publicationForwardingFailure`, and the `Valid()` reject were executed in process.
- The shared `fvpn-pg` server. It was not contacted.
- A live updater rollback, installer unit stop, and the old binary's `Boot` against an S ruleset. `stepRollback` was read. `node/updater` is not an owned test path.
- Each design crash-table row on agentstore schema v4. That schema is not in the tree.
- The D-133 A/B trial, and any DNS, fair-use, or topology diff inside the 80 transitions. The analysis JSON does not record those diffs.
- `TestRevoke_RulesetStopsEstablished` on this prefix. Source-revocation of an established flow is that existing test. This review's kernel run is the destination case.
- Fleet-update result SQL. `installer_results.go:132` and `results.go:208` were read.
- Host networking, sudo, and containers outside `fvpn-r19`. None were used. Owned `fvpn-r19` inventory was removed at the end.
