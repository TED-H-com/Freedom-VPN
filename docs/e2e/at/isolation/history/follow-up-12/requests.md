# Current requests — Follow-up 4

Run `fa0074cc-3b1a-403d-b8c2-3a3439fe2b7e`, source
`491a307dd505e33b14dc0e993aa7e86b28e82f68`, based on main `205633f`
(#216, #219, #220, #231).

1. **Devstack follow-up 14: Docker embedded DNS NAT in the node netns blocks serving.** Owner `w3-devstack-e2e: node netns table ip nat POSTROUTING xt SNAT for 127.0.0.11`. After publication, `nft -j list ruleset` in `fvpn-w4-isolation-node` shows `table ip nat` chain `POSTROUTING` (hook `postrouting`, priority 100, policy accept) jumping to `DOCKER_POSTROUTING`, whose statement is `xt` target `SNAT` for `127.0.0.11`. Chain `OUTPUT` jumps to `DOCKER_OUTPUT` with `xt` target `DNAT` for the same address. `#220` `VerifyServing` fail-closes that `xt` statement as `FLOWTABLE_BYPASS`. The accepted observation at `2026-10-31T00:31:16.688055275Z` has lease, qos and source_binding `error_code=FLOWTABLE_BYPASS` and `reasons=["safety_check_failed"]`. Shared publish operation `22ecf90f-847a-481d-a58c-8fcd45cb32b7` stayed `running` with `NODE_NOT_SERVICEABLE member_forwarding_unconfirmed`. The isolation PUT returned `422 NODE_NOT_SERVICEABLE`. This is why post-revoke ingress, the forged source, the convergence window, and both expiry measurements were not taken. The harness must not delete or rewrite that table. The needed capability is a node network namespace whose ruleset `VerifyServing` can accept, with this Docker DNS NAT absent from the listing `nft -j list ruleset` returns inside the node.

2. **Post-revoke kernel ingress attribution stays unmeasured.** The export landed: `Stack.ObserveMemberIngress` / `InjectMemberIngress` (`docs/e2e/scenarios/harness-exports.md`). It was not called. Owner is the serving gap in request 1, not a missing inject primitive. Devstack run `4c97ebef` is not an AT13 pass.

3. **Unknown/spoofed member source stays unmeasured.** The same export would inject forged `100.64.7.7` and the live member, with `ctunku` +1 and zero forwarded bytes for the forged source and `u_<hex>` +1 for the admitted source. `ObserveMemberIngress` was not called. Owner is request 1.

4. **Designated DNS exception did not land.** Owner `w3-devstack-e2e: resolver on member underlay`. No export places the designated resolver behind the exit. `harness-exports.md` exposes only `InjectMemberIngress` and `ObserveMemberIngress`.

5. **IPv6 and member-platform rows did not land.** Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. Same export limit as request 4.

6. **Provisional lease code was not sampled.** Agent follow-up 18 is on this main (`205633f` includes #216). The harness asserts lease-check `AWAITING_MEMBER_GRANTS` and block reason `awaiting_member_grants` on a grant-less same-generation sample. This run produced no sample: publication never succeeded, and `convergence-timeline.json` has `samples: null`. The recorded code is null. Build `491a307dd505e33b14dc0e993aa7e86b28e82f68`.

7. **Whole-node and finite-member expiry were not re-measured.** The fleet-config PUT was not issued, so the ceiling was not changed and was not restored by this run. Owner is request 1. Follow-up 3's passing measurements remain under [history/follow-up-3](history/follow-up-3/).

8. **Acceptance/operator: the continuous fault matrix stays open.** Crash, NetBird restart, reboot, updater transition, and physical member-platform/arm64 runs are still owner `acceptance / operator`.

No `node/enforce` bypass request is filed. `#220` fail-closed on the ruleset it read. Deleting `table ip nat` or weakening `VerifyServing` is not an isolation-harness change.

# Current requests — Follow-up 5

Run `c0106e5d-d8e8-400b-abe0-181d6428fc55`, source
`7fa0443c532b1c9a395cd93620152fb6902584ae`, based on main `9e493e8` (#240).
`verify_foreign.go` in that tree is `2473acf` (#220).
`git log -1 origin/main -- node/enforce/verify_foreign.go` at record time is
`a4191fa` (#241, D-079). That commit is not an ancestor of the run.
Publication succeeded. The serving gate passed because follow-up 14 removed
`table ip nat` from the node before the agent started. `loopback_scoped` was
not emitted. The ruleset read-back has no Docker NAT rule.

1. **Provisional `AWAITING_MEMBER_GRANTS` was not sampled.** Owner `w2-agent-core`. Twelve convergence samples exist. The two grant-less samples (`2026-10-31T01:59:48Z` and `2026-10-31T01:59:50Z`) have `desired_bindings=0`, lease check `not_run`, and `error_code` null. At `2026-10-31T01:59:52Z` `lease_members` is already 1. No sample has `desired_bindings>0`, `lease_members=0`, and the same lease and desired generation. The check records `windows=0` and an empty code list. AT10 is subset passed, case blocked. The six destination cases passed on this run.

2. **Designated DNS exception did not land.** Owner `w3-devstack-e2e: resolver on member underlay`. The member route to `172.28.33.40` is `dev eth0`. No export places the designated resolver behind the exit. `harness-exports.md` exposes only `InjectMemberIngress` and `ObserveMemberIngress`. AT12 is subset passed, case blocked. The forged `100.64.7.7` source passed on this run (`ctunku` +1, zero forwarded bytes).

3. **IPv6 and member-platform rows did not land.** Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. Same export limit as request 2. AT11 is subset passed, case blocked. The metadata and deprecated-6to4 cases passed on this run.

4. **Acceptance/operator: the continuous fault matrix stays open.** Crash, NetBird restart, reboot, updater transition, and physical member-platform/arm64 runs are still owner `acceptance / operator`. AT13 is subset passed, case blocked. Post-revoke ingress, finite-member expiry, and the 60 s whole-node expiry (ceiling restored to 86400, both `fleet_config.update` audits) passed on this run.

No `node/enforce` change is requested. This run did not delete or rewrite nft. D-079 on later `origin/main` was not required for the pass: the lab node presented no Docker NAT rule for the classifier to scope.

# Current requests — Follow-up 6

Final run `bf671f3a-c6ac-4824-9107-425c4c5a5f02`, harness
`e711ebd822eda2ea1b5ba10b187818fda6777a74`, product baseline
`5d1e5ce8727ca824fa24197ce2568bd513bac7f3` (origin/main at task start,
through #256). Publication and counted member traffic succeeded.

The AT10 foreign-rule read-back is **`[]`**. D-079 is present (`a4191fa`,
#241), but this node has no Docker NAT and no `loopback_scoped` entries.
The serving pass is attributable to the lab fix **#240**, as in follow-up 5;
it is not evidence that this node needed D-079. No foreign nft/tc state was
planted to make that distinction.

1. **ISO-F6-01 — A serving apply during the interface-change check rejected tc state and remained fail-closed.**
   - **Affected cases:** AT10 interface-change continuity; AT12's four same-/cross-member peer checks subsequently lacked counted positive controls. Their final-run results remain blocked. Six AT10 destination checks passed before this event; later management, forged-source and withdrawal checks passed after the recorded recovery.
   - **Owner:** `w1-netlab`, with `w2-agent-core` for latched recovery and evidence transport.
   - **Observation/evidence:** `interface-change-evidence.json`, observation row `f2cfdff8-3e46-49e8-86f1-b38128ea6cd4`, report `01a19042-af78-79f0-8dd6-9d13ce85c2f2`, observed `2026-10-31T04:40:51.832647585Z`. The apply error is `HEALTH_CHECK_FAILED` / `TC_FOREIGN_FILTER`, timestamp `04:40:51.80844254Z`; safety failed and member forwarding was false. The interface probe made 3968 attempts in 60.026981429 s, delivered zero, and its after-control failed. The API capacity update was created later, at `04:40:52.423892Z`; it must not be reported as the trigger. No foreign tc state had been planted. The harness's temporary `tun-mirred` ingress injection occurs later in the run.
   - **Exact locations:** `tests/e2e/at/isolation/driver.go.txt:777–806 (`intranet`)` performs the disposable interface flap and capacity update; `node/enforce/verify_tc.go:220–255` classifies tc objects and returns `TC_FOREIGN_FILTER`; `node/enforce/apply.go:102` runs `VerifyServing`; `node/agent/engine.go:964–979` reports the reason but drops the attachment/handle detail. `driver.go.txt:peers` embeds the positive controls that timed out afterward.
   - **Required fix:** reproduce the interface/apply transition and retain the rejected tc snapshot with interface, attachment, handle and generation before rollback. Determine whether transient/owned state was rejected or an actual foreign object existed; correct the responsible path while keeping fail-closed behavior. The later read-only tc listing contains only default `noqueue` qdiscs and empty filter lists, but it is explicitly post-rollback and cannot identify the rejected object. No enforcement bypass is claimed.
   - **Recovery:** one additional real `docker restart fvpn-w4-isolation-node` at `04:51:02.744373Z`–`04:51:12.849221Z` cleared the latched condition; see `recovery-action.json`. No synthetic observation, grant, desired state or nft/tc edit was used. The same-run read-only recovery sampler proved the provisional window (19 samples), and subsequent positive controls passed. This assisted recovery does not promote the earlier interface or peer results.

2. **ISO-F6-02 — Product observations do not carry the new tc evidence.**
   - **Affected case:** AT10 serving read-back evidence. The mechanism verifier is measured separately and passed.
   - **Owner:** `w1-netlab`, coordinating observation transport with `w2-agent-core` and the contracts lane.
   - **Observation/evidence:** product observation row `4a1dc037-c2c0-4955-90a5-c19a2928e998`, report `01a19041-2044-7558-8ee1-2666e65e2711`, observed `2026-10-31T04:39:09.636348464Z`. AT10 `observation_report` and `tc_readback`, plus `serving-readback-metadata.json`, show neither `tc` nor `serving_evidence`. The same run's mechanism `lab-AT10-tc.json` records the returned tc section verbatim: owned HTB/fq_codel objects and empty filter attachments. No foreign tc state was installed for this read.
   - **Exact locations:** `node/enforce/verify_tc.go:104–141` returns `ServingEvidence.TC`; `node/enforce/apply.go:26` retains it in `ApplyResult`; `tests/netlab/cmd/fvpn-netlab/main.go:276–289` exposes it for the lab. `node/agent/engine.go:861` consumes the apply result but does not retain its evidence for reporting. `node/agent/observe.go:92–123`, `pkg/api/api.gen.go:4269` and `contracts/openapi.yaml:7392` define/build an observation without a tc evidence field.
   - **Required fix:** expose the redacted live serving evidence through an agreed product read-back surface, tied to generation and observation, so the assembled acceptance record can retain the actual verifier result. Do not weaken `VerifyServing`. This finding concerns evidence transport; #256's verifier is present and the clean mechanism read passed.

3. **Designated DNS exception did not land.** Owner `w3-devstack-e2e: resolver on member underlay`. AT12 is subset passed, case blocked. The member route to `172.28.33.40` is `dev eth0`. No export places the designated resolver behind the exit. `docs/e2e/scenarios/harness-exports.md` exposes only `InjectMemberIngress` and `ObserveMemberIngress` for these isolation paths. Required fix and affected check are unchanged from follow-up 5: export a resolver path behind the exit and prove TCP/UDP 53 separately from management-port denial. Location: `driver.go.txt:peers`, check `designated DNS exception`; the check retains the exact route. The forged `100.64.7.7` source passes on this run.

4. **IPv6 and member-platform rows did not land.** Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. AT11 is subset passed, case blocked. Same export limit as request 3. Metadata and deprecated-6to4 pass. Required fix is unchanged from follow-up 5: real dual-stack/platform paths and per-platform routes/captures. Location: `driver.go.txt:intranet`, check `IPv6 platform matrix`; the record contains the observed member IPv6 addresses. No later devstack work was imported into this pinned tree and the run did not wait for it.

5. **Acceptance/operator: the continuous fault matrix stays open.** Owner `acceptance / operator`. The continuous AT13 check remains blocked (the aggregate now fails on request 6). Crash, NetBird restart, reboot, updater transition, and physical member-platform/arm64 runs remain unproved. Location: `driver.go.txt:restart`, check `continuous crash/NetBird/reboot/updater transitions`, and `operator-run.md`. Required next move is the same continuous controlled-probe matrix as follow-up 5; ordinary restart recovery and cross-compilation are not substitutes.

6. **ISO-F6-03 — Whole-node expiry did not satisfy the near-deadline continuity assertion.**
   - **Affected case:** AT13 `short whole-node lease expiry` is **fail**; finite-member expiry and post-deadline ingress denial pass. AT13 aggregate is fail, with the separate continuous fault matrix still blocked.
   - **Owner:** `w2-agent-core` with `w4-at-isolation` for timing instrumentation and acceptance calibration.
   - **Observation/evidence:** same run, AT13 named observations `node_lease`, `forward_near_deadline`, `early_baseline_at`, `baseline_at` and `heartbeat_after_deadline`; derived values and limits are in `expiry-timing-analysis.json`. There is no central expiry-transition observation ID because control and worker were stopped at `05:00:29Z`. The node-held generation-29 lease expired at `2026-10-31T05:01:28.561634193Z`. Immediately after control loss, bytes increased 47750 → 50370. The request scheduled four seconds before expiry failed and bytes stayed 50370. The first baseline label is `05:01:27.784897861Z` (776.736 ms before the signed deadline); the post-deadline label is `05:01:28.675726567Z`. The heartbeat’s wall timestamp is `05:01:21.078247388Z` and is stale by the time it was read. Post-deadline ingress reached `fvmb0`, not `fvup0`, and bytes remained 50370.
   - **Exact locations:** `driver.go.txt:1188–1227` schedules the near-deadline request, timestamps before `forwardChain`, and requires no early baseline; `driver.go.txt:1346–1358` runs the counted request without start/end packet timestamps; `node/agent/decide.go:600–645` derives conservative deadlines and integer-second member allowances; `node/internal/timeanchor/anchor.go:99–110` anchors the deadline to the monotonic send time. Spec §16.3 explicitly requires conservative time handling.
   - **Required next move:** retain request start/end, kernel-read start/end, actual monotonic deadline and a continuous near-deadline ingress/egress trace; reconcile these with the signed deadline and investigate the lost pre-deadline positive. The existing strict result is preserved. The early label alone is not a validated product defect because the read completion is unrecorded and conservative deadlines are allowed. No forwarding after expiry was observed. The fleet ceiling was restored to 86400 through the API with a retained audit. The earlier diagnostic run passed this check; that pass is not substituted for this final-run failure.

7. **Handover claims must follow the refreshed acceptance records.** Owner `w4-handover` / orchestrator. `go test ./...` reaches `tests/handover/TestHandover_Followup13Claims`, which reads current AT10–AT13 records and requires their run ID and exact counts in `docs/ops/README.md` (`tests/handover/handover_test.go:1909–1911`). Those paths are outside this lane's ownership. Required fix: refresh the dated handover claims to this run and preserve the blocked and failed outcomes. Exact final failures are retained in `summary.md` and the hand-off report. Do not restore stale acceptance evidence to satisfy the old narrative.

Closed prerequisites / owned correction:

- **#250:** the formerly missing provisional proof is now measured during real node recovery: 19 qualifying 250 ms samples, exactly three `AWAITING_MEMBER_GRANTS` checks and one lease-stage error, then same-generation populated lease and counted admission. Ready-observed → populated is **4767 ms**; ready-received → populated is **4760 ms**, beside usage follow-up 3’s **5021 / 5025 ms**. Initial bootstrap still recorded zero qualifying samples; that distinct result is retained, not promoted. `provisional-recovery.json`, `provisional-ready-report.json` and the offline-reviewed AT10 record identify all timestamps and provenance.
- **#256:** clean mechanism tc read-back now retained verbatim; request 2 names the missing product observation surface.
- **#241:** D-079 classifier is present. **#240** explains the serving pass on this node because the Docker chains are absent.
- **#246:** the follow-up 5 teardown fix is verified by the final zero inventory in `cleanup-inventory.json`.
- **Owned disable wait:** the first full run stopped at `partially_succeeded` and restored publication too early. It is preserved under `history/follow-up-6-diagnostic/`. The final run waits for observation-driven completion before restoring intent and records the operation transitions. No product code changed.

## Devstack follow-up 15 response (2026-10-31)

Request 2: `ObserveDesignatedDNS` is exported through `labstack`. The prepared product configuration names the isolated origin-network resolver at port 53; its TCP/UDP forwarder reaches the node's filtering AdGuard. The member cannot use its Docker underlay for this destination. The scenario requires replies and both device DNS counters to move while Internet counters hold. This closes the lab's missing designated exit path, not the full AT12 case or native-client fallback.

Request 3: `member-ipv6` records the native member's IPv6 configuration and remains blocked. A dual-stack native fixture, IPv6 underlay route/capture matrix and per-platform leak measurements have not been provisioned. An interface address alone is never a pass. IPv4-only mode's `ctv6` mechanism tests in netlab do not substitute for AT11's client-platform evidence. Owner remains devstack/platform operators.

Request 1 remains agent-core's provisional-grant observation window. Request 4's continuous crash/restart/reboot/update/platform campaign remains acceptance/operator work; the new established transport helpers can measure active TCP and UDP during those faults. The control-loss scenario now uses those helpers. The node entrypoint retains deletion of Docker's `ip nat`; it does not claim D-079 classifier evidence, which belongs to netlab.

## Follow-up 7 requests — 2026-10-31

Current run `041e9e48-79a6-45dd-a19d-8b5a63290655`, harness `d1a90ea`, product
main `c6e3eca` through #272. AT10 blocked, AT11 blocked, AT12 pass, AT13 fail.
This assisted run did not recover serving after the injected foreign tc case.
[summary.md](summary.md) gives full timestamps; prior dated requests above are
history and must not be read as the current result.

1. **Netlab follow-up 14 + agent-core follow-up 23: ISO-F6-01 / D-096 recovery
   remains blocked.** Injection at 06:29:29.204706725Z produced accepted
   `TC_FOREIGN_FILTER` report `01a190a6-22f8-74d6-8178-96f3551b3f4c` at
   06:29:29.490009445Z. Member-down was confirmed by 06:29:29.832949690Z;
   uplink/control stayed up. After that read, 538 member probes produced zero
   uplink packets. One unsnatted packet escaped at 06:29:29.305982538Z, before
   quarantine; do not claim zero across the complete injection window.
   Filter removal completed at 06:29:32.889058275Z; generation 32 arrived at
   06:29:33.290828482Z. The harness restored the saved member paths at
   06:29:39.259867752Z, and an extra API generation 33/restoration was accepted
   at 06:31:25.419525174Z. Serving/counting did not recover before the clone
   scenario. No `member_path_unconfigured` report was captured: the attempted
   D-096 phase remained behind `TC_FOREIGN_FILTER` / `policy_missing`, so it
   does not prove clean Apply followed by the member-path hold. Investigate
   `node/agent/engine.go` apply/renewal decisions, `serving.go`, `quarantine.go`
   and `memberpath.go` using `foreign-redirect.json`, the receiver stream and
   `recovery-action.json`. Reproduce a clean newer-generation Apply, an actual
   D-096 hold, then recovery after the harness or NetBird restores the path.
   Preserve D-100; do not seed refreshed expected topology. The earlier ordinary
   interface/generation change passed on this run, so the unexpected follow-up
   6 rejection was not reproduced at that step.

2. **Netlab follow-up 14 / agent-core: ISO-F6-02 is partially addressed.** The
   same product observation now transports actual tc attachment objects and
   initial/current/apply quarantine evidence through `errors` stages. The
   read-back identifies flower/mirred and before/after administrative state.
   Clean-serving `tc`/`serving_evidence` remains absent at the initial counted
   control (`serving-readback-metadata.json`). Keep the clean and rejected
   evidence surfaces distinct; decide whether the agreed read-back must expose
   clean verifier evidence too. No contracts change is made in this lane.

3. **Agent-core follow-up 23: ISO-F6-03 decisive continuous timing.** Node-held
   generation-31 lease expires 06:29:02.925490361Z. The last successful member
   SYN-ACK is at 06:29:02.696232187Z; the first failed SYN enters fvmb0 at
   06:29:02.748690751Z with no fvup0 counterpart. Baseline read is bracketed at
   06:29:04.228093040–06:29:04.250071427Z (+1.303–1.325 s); first expiry heartbeat
   read is 06:29:10.466713660–06:29:10.496189725Z (+7.541–7.571 s). Near-deadline
   counted forwarding passed, post-deadline bytes stayed 126231, but the
   unchanged one-second-tick predicate failed. `whole-node-node-capture.txt`
   has 2389 packets/zero drops; `whole-node-probes.json` retains every one of
   251 timed attempts; both bracketed read files retain heartbeat wall/monotonic
   fields. No successful connect occurred after expiry. Reconcile integer
   kernel allowance/conservative anchor timing and delayed baseline/heartbeat
   under control loss without changing the AT definition (§16.3).

4. **Agent-core follow-up 23 + wire-nodes: D-087 recipe did not yield a qualifying
   report; rate limiting/backlog needs review.** With heartbeat/poll/observations
   all 1 s, the 100 ms receiver retained every POST, not a latest-state poll.
   Zero ready/passed single-lease-error provisional reports qualified. Thirty
   distinct bodies mention `awaiting_member_grants`, but the first two at
   06:31:47.886903310Z / 06:31:47.979152646Z also fail tunnel/safety checks; later
   bodies are frozen. These are not the provisional state. The stream includes
   3031 HTTP 429 responses; report `01a190a8-1b90-7a45-95af-4cf4146ee529`, observed
   at 06:31:38.640670555Z, was accepted only at 06:31:48.795212863Z. Investigate
   failed-serving observation flush/retry scheduling and control rate budgets;
   provide a reproducible safe transition if the recipe needs another trigger.
   No new duration can be compared with historical 4767 / 5021 / 5025 ms.

5. **Enrollment ENR-02/03 + agent-core: finish the live-grant freeze assertion.**
   The real clone caused `identity_frozen: true` on source acceptance at
   06:31:48.795212863Z. Renewal began 23.481 ms later, and a non-serving frozen
   heartbeat was written 183.868 ms after acceptance. Later reports/renewals
   omit generation ACKs. Source was already non-serving with empty grants,
   so this proves latch consumption but not withdrawal of live grants within
   one heartbeat or post-freeze restart persistence. Run that assertion from
   a serving enrolled source; do not promote this partial proof.

6. **Devstack / operator backlog.** The newly exported designated UDP/TCP DNS
   path passes with counter attribution on this run, closing the prior missing
   resolver-path blocker. AT11 still needs real dual-stack/member-platform
   paths, and AT13 still needs the continuous crash/NetBird/reboot/updater and
   physical-device matrix. The origin image has no tcpdump: the failed separate
   origin capture is retained, while the node `any` trace covers both fvmb0 and
   fvup0. The owned helper now makes its additional uplink capture explicit.

7. **Handover / orchestrator: D-098 archive available.** Pin final follow-up 6
   to `history/follow-up-6/`: 44 byte-identical files, final run
   `bf671f3a-c6ac-4824-9107-425c4c5a5f02`. Follow-up 5 is unchanged, and its dated
   claims test remains true without another handover edit. Current follow-up 7
   results must not replace those dated claims silently. The lane's standalone
   make target is not a `gates.sh` process; coordinating wrappers should count
   it too (one other gate at launch; two other wrappers appeared later).

## Follow-up 8 requests — 2026-10-31

Current run **`7779d83c-2f78-4699-ac88-8fcc59e8cf87`**, image/harness `dafb8f4d`,
product baseline `62f24e6d` through #312/#308. **AT10 blocked (8/1/0), AT11 blocked
(2/1/0), AT12 pass (10/0/0), AT13 blocked (6/1/0)**; counts are pass/blocked/fail.
The [current AT/ISO table](summary.md) and raw records supersede earlier dated
requests as current evidence. Earlier outcomes remain in their archives.

Closed by this run:

- **ISO-F6-01 / D-096:** all six [G2 assertions pass](recovery-verdict.json).
  Generation 30 serving → injected clsact/flower/mirred → dirty generation 31
  rejected without deleting foreign objects → removal → clean generation 32,
  matching populated lease and verified Apply/live read → native topology and
  counted recovery. No restart or harness route/address/link repair. Injection
  to contained observation ≤425.984 ms; removal to observed link-up ≤640.719 ms,
  serving ≤8.941 s, completed counted control ≤18.063 s. These are measured
  upper bounds, not production guarantees. Four redirected packets preceded
  quarantine; none reached the uplink from confirmed down until removal.
- **ISO-F6-02 / D-106:** 539 actual tc wire reads, all 539 canonical digests match.
  Revision-13 qos detail and historical tc_evidence errors remain retained.
  No unsolicited rejected object appeared; this harness created all observed
  foreign clsact/flower/mirred objects. The original unretained follow-up 6
  object's creator remains unknown, not attributed from a passing reproduction.
- **ISO-F6-03 packet predicate:** generation-30 deadline D=`30175198061283` ns;
  last positive **D−99.952819 ms**, first drop **D−61.767983 ms**, 161 positives /
  41 drops, no post-deadline or later positives, zero capture drops. No added
  cadence allowance. AT13 is no longer failed by that predicate, but remains
  blocked by the separate full fault/platform matrix.
- **D-120:** netlab 18 was present before launch; the printed AT10 verdict is
  **PASS under D-120**. 100 reloads, 101 verified Applies, 59 datagrams, **zero UDP
  losses and zero TCP timeouts**; therefore no loss fails conditions (i)–(v),
  and the per-loss table is empty. Maximum guard upper bound 47.066166 ms and
  render→installed 28.231196 ms, each ≤100 ms. The [complete analyzer output](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/at10-analysis.json)
  is the verdict input. This is an AT10 continuity rule, distinct from D-104.
- **D-098 / D-109:** final follow-up 6 and 7 archives verified 44/44 and 52/52
  SHA-256 matches, including corrected final verdicts. Handover already reads
  archives/immutable fixtures, so no living-file claim pin moved. Its current
  AT10 path-string rejection test does not read that file. Handover tests pass.
- **D-102 / D-108:** consumed RESPONSES.md's complete export helper; source and
  clone producers were stopped after measurements, then receiver drained and
  exported. Node `any` is the independent member/uplink record, with complete
  packet counts and zero kernel drops.

### Requests to orchestrator

1. **D-087 clean provisional recipe — agent-core + devstack.** Agent-core 24's
   recipe uses fake-control `SetMemberGrantsReady(false/true)` on a clean,
   unfrozen, fixed-generation published node with desired bindings. Current
   assembled public APIs expose no equivalent. Implement/export that controlled
   transition without changing publication/generation or writing projector rows;
   then this lane can measure it once. This item requires files outside this
   lane's ownership and was stopped independently. No synthetic transition was
   used. The receiver retained 544 observation POSTs and zero qualifying
   ready/passed provisional reports; no new duration replaces historical
   4,767 / 5,021 / 5,025 ms. [Recipe limitation and observations](provisional-receiver-analysis.json).

2. **Exact D-104 producer timing — agent-core / enforcement.** Export real
   Apply start, final-render sample, installed-by, guard install/release command
   brackets and executor precision-read samples in a suitable test-only
   measurement surface. Current `node/agent`/`node/enforce` are outside this
   lane's owned paths. The retained passive netlink timestamps are receipt
   bounds, not those missing samples. The original recorder's top-level-JSON
   gap is explicit; the corrected read-only recorder covers renewal/expiry/
   recovery without modifying product binaries. The [renewal trace](continuity-analysis.json)
   has two real renewals, 68 guard receipt intervals, 1,396 probes and **17
   losses**, all with ingress and no uplink egress. Its 80 ms dual-phase probes
   are not the 1.5 s D-120 stream; no tolerance was applied to them. Correlate
   these against genuine command brackets before claiming strict renewal
   continuity. The exact final expiry packet predicate passes independently.

3. **AT11 native IPv6/platform coverage — devstack + platform operators.**
   Supply real dual-stack/native member paths and the required platform route/
   capture matrix. The current metadata/6to4 passes and netlab IPv6-block proof
   do not provide those inputs. AT11 stays blocked.

4. **AT13 full continuous fault/platform matrix — acceptance + operators.**
   Execute continuous counted probes through crash, NetBird restart, OS reboot,
   updater transition and real device/platform paths, including physical arm64.
   Ordinary process restart, finite-member expiry and whole-node deadline
   measurements do not substitute for that matrix. AT13 stays blocked.

There are **four routed requests**. The assembled gate returned 2 for blocked
scope, not a fabricated all-green result. Owned lint/test/build, handover,
root go-lint and evidence hygiene pass; owned Docker inventory is zero.

## Follow-up 9 requests — 2026-10-31

Historical run **`2be2797b-7a28-4e9f-a1fe-5fffa4c1b047`**, live source `e496d826`,
product baseline `fb5e0025`, 953.07 s. **AT10 blocked (8/1/0), AT11 blocked
(2/1/0), AT12 pass (10/0/0), AT13 fail (5/1/1)**; counts are pass/blocked/fail.
The assembled gate returns 2 (lab=0, stack=1). No rerun replaces this outcome.
The [summary](history/follow-up-9/summary.md) preserves every measured result and its scope.

Delivered prerequisites: netlab 19 / agent-core 27 timing is consumed in 3,289
retained snapshots, including failures and both recovery verifiers. ISO-F6-01
passes all six assertions; all 557 D-106 wire digests match. D-120's separate
100-reload stream passes unchanged with one eligible loss among 50 datagrams.
D-093 reconciles 15 agreements, 13 open / two retired within simulator scope.

### Requests to orchestrator

1. **D-087 — wire-nodes 33, then devstack 20, then isolation follow-up 10.**
   Latest amendment confirmed: devstack 19 did not export the false/true grant
   transition. `internal/nodes/httpwire/state.go` / `runtime.go` need a TEST ONLY
   injectable grant-eligibility reader/factory using the existing
   `leases.WithEntitlementReader` option. Once wired, devstack must export the
   controlled future-grant transition without changing publication, bindings,
   generation or projector outputs. This lane stopped that item at the boundary.
   No synthetic transition was driven. Zero qualifying accepted reports among
   562 POSTs; no new duration. Historical **4,767 / 5,021 / 5,025 ms** remain.

2. **ISO-F6-03 new expiry failure — w2-agent-core, serving/heartbeat/watchdog path.**
   Exact locations: `node/agent/engine.go` `Step`, `finishServingStep`,
   `verifyCurrentServing`; `node/watchdog/watch.go` `Watch.needsBaseline`.
   Netlab supports verifier diagnostics. See the [focused failure index](history/follow-up-9/expiry-failure-correlation.json)
   and [every loss window](history/follow-up-9/producer-timing-analysis.json). Generation-30 lease
   deadline D=`42835508568707` ns stays valid. Apply 955 and live reads 956/957
   succeed; verifier 958 fails with `GENERATION_MISMATCH`, followed by blocked
   audit 959. The periodic kernel read shows generation-0 baseline, empty
   allowed_v4 and serving 0. Apply 961 subsequently restores generation 30 and
   forwarding under the same lease. The capture contains **96 early drops,
   then 65 positives, then 41 final expiry drops**. **95 early drops are outside
   every completed guard**, one overlaps the recovering install. First drop
   D−8,098.843780 ms; forwarding resumes D−3,298.889928 ms; last positive
   D−99.972452 ms. No post-deadline positive occurs, but later positives after
   the first drop fail the unchanged D-104 predicate. Investigate the
   control-loss I/O / heartbeat / concurrent baseline sequence. The writer of
   that baseline is **not identified** by the exported Engine timing; watchdog
   participation is an inference, and its retained log is empty. No netlab
   allowance or synthetic timestamp can close this gap. Product changes are
   outside this lane and were stopped independently.

   The **separate renewal finding** is fully correlated: 1,396 probes, two real
   signed renewals, 70 Applies/guards, **three losses** (IDs 54/294/1315). All
   three ingress reads lie inside install commands, with outer brackets
   **37.676074 / 36.090627 / 50.196105 ms**. Netlab/agent-core now have the bracket
   size evidence; every loss remains counted, with no D-120 tolerance. The old
   producer-export request is delivered. Historical **17** losses remain
   unclassified because their original run lacks exact producer timestamps.

3. **AT11 native IPv6/platform matrix — devstack and platform operators.**
   Supply the real dual-stack and member-platform routes/captures. Metadata and
   6to4 passes and the separate IPv6-block mechanism do not establish that
   matrix. This outside-scope item remains blocked.

4. **AT13 full continuous fault/platform matrix — acceptance and operators.**
   Execute continuous counted probes through crash, NetBird restart, OS reboot,
   updater transition and real platforms, including physical arm64. Ordinary
   restart recovery and cross-compilation do not substitute. This item remains
   blocked separately from the newly failed expiry check.

5. **Netlab 20 — resume pin verification on the immutable follow-up 8 paths.**
   All three are byte-identical and included in
   `docs/e2e/at/isolation/history/follow-up-8/archive.json`:

   | Immutable archive path | SHA-256 |
   |---|---|
   | `docs/e2e/at/isolation/history/follow-up-8/deadline-verdict.json` | `94d777e21b93689e899442f5e3b7d62463527099c8ce511de56d87b6b7253ae1` |
   | `docs/e2e/at/isolation/history/follow-up-8/recovery-verdict.json` | `b1a40773309c350664e502d87e8699fe9388c1211b94c2c241f5968bfaa452ce` |
   | `docs/e2e/at/isolation/history/follow-up-8/continuity-analysis.json` | `9e5d77f88c69c8b30fe3d78abe185c37bb9ccb610ea464f532b4a75d7ada5b5c` |

There are **five routed requests**, including the immutable-path hand-off.
D-098 archived 80/80 follow-up 8 files before rewriting root evidence; follow-up
6 and 7 are untouched (44/44 and 52/52). Handover already uses immutable claims,
so no file or pin moved. Owned lint/test/build, handover, root lint and hygiene
pass. Owned Docker inventory is zero. No push, PR, tag or product edit.

## Follow-up 10 requests — 2026-10-31

D-087 is delivered: the dedicated grant-eligibility runtime, natural signed
renewals and actual accepted ready/passed observations qualify the transition.
The independent original-wire audit passes. The current measured duration is
962 ms; the first attempt's 2,035 ms, historical isolation 4,767 / 5,021 /
5,025 ms and devstack 3,980 / 4,112 ms remain separate samples.
D-098 archived all 86 follow-up 9 files byte-identically before rewriting root
artifacts. Earlier archives and netlab's three follow-up 8 pins are unchanged.
The previously requested producer surface is delivered and consumed, including
baseline and heartbeat source kinds plus retained normal reason-log segments.

### Requests to orchestrator

1. **Agent-core / enforcement / netlab: investigate the retained install-bound
   failure and strict continuity losses.** The [first attempt](first-attempt.md)
   retains 49 failed Applies. Process `2e7247e0-7303-4080-bd68-ca9ce9abb565`,
   Apply 712, rendered at `54858887599306` ns and installed during
   `54858887720453`–`54859103490102` ns: render→installed **215.890796 ms**,
   beyond the unchanged 100 ms bound; no guard release completed. Baseline 713
   explicitly names the agent's `apply_rollback` write at
   `54859107743361`–`54859384608119` ns, followed by the two retained
   `verification_HEALTH_CHECK_FAILED` writes. Every failed Apply and all 279
   renewal losses remain in the archived
   [failure analysis](history/follow-up-10-attempt-1/attempt-failure-analysis.json).
   Later passing bounded-expiry measurements do not resolve this event,
   strict renewal losses, or historical ISO-F6-03. The historical follow-up 9
   96 early drops / 65 later positives / 41 final drops remain failed and its
   writer remains unattributed. No host-contention cause is claimed. Product
   paths are outside this lane; no product repair was attempted. The current
   sample retains 30 renewal losses among 1,396 probes, all inside completed
   guards, plus one pre-D ingress loss in its passing D-104 window. See the
   [exact current correlation](producer-timing-analysis.json).

2. **Node documentation owner: pin four historical follow-up 28 sources.**
   `docs/node/followup28-iso-f6-03-2026-10-31.md` lines 11–14 still link living
   isolation root files for its historical follow-up 9 event. Repoint
   `expiry-failure-correlation.json`, `producer-operations.jsonl`,
   `producer-timing-analysis.json` and `whole-node-continuous-reads.json` to
   `docs/e2e/at/isolation/history/follow-up-9/`. This file is outside owned paths,
   so that item alone is stopped. Handover has no such mutable file reads; its
   AT10/AT13 path-string rejection tests require no D-132 edit or pin move.

3. **AT11 operator item (carried from follow-up 9 request 3).** Devstack and
   platform operators supply the native dual-stack/member-platform route and
   capture matrix. Current mechanism and metadata/6to4 evidence does not close
   this item.

4. **AT13 operator item (carried from follow-up 9 request 4).** Acceptance and
   operators execute the full continuous fault/platform matrix, including
   physical arm64. Ordinary process restart, a bounded expiry pass and arm64
   cross-compilation do not close this item.

There are **four routed requests**. The [current summary](summary.md) and
[validation](validation-results.json) retain the exact assembled result and
completed-guard distributions. The two earlier follow-up 10 attempts are
independently archived, including their failures; no later row reclassifies them.

## Follow-up 11 requests — 2026-10-31

Agent-core 29 (#356) answered both follow-up 10 product/documentation requests:
the four historical Sources links now point at the byte-identical follow-up 9
archive; controlled CPU contention did not reproduce the install overrun.
The 49 first-attempt failures remain `cause_status=unproven`, with host context
**unavailable**, and all 279 renewal losses remain counted. New per-Apply host
samples and sanitized command status are consumed by run ID / sequence. D-134
and D-135 wording is adopted without changing packet predicates or historical
outcomes. No historical writer is identified by a fresh successful run.

### Requests to orchestrator

1. **2026-10-31 — AT11, devstack and platform operators (carried).** Supply the
   native dual-stack/member-platform routes and packet-capture matrix.
   Mechanism tests, metadata and 6to4 results do not establish real platform
   support. This work is outside the isolation lane and remains blocked.
2. **2026-10-31 — AT13, acceptance and operators (carried).** Execute the full
   continuous fault/platform matrix: agent crash, NetBird restart, OS reboot,
   updater transition and physical arm64. Ordinary process restart, bounded
   expiry and arm64 cross-compilation do not establish those windows. This
   outside-scope item remains blocked independently.

There are **two routed requests**. Agent-core/netlab's further command-lifecycle
instrumentation proposal is tracked in their own assessment; it is not
implemented by this consumer lane. The historical first-attempt install cause
and ISO-F6-03 baseline writer remain known gaps, without a new attribution or
a claim that those historical failures have passed.

## Follow-up 12 requests — 2026-11-01

Run `1c25ee44-1872-4cce-a9f5-d52491d87095`, runtime source `99e58bff`, baseline
`5bd751026073444087b32ac2a7115f8566104f8b` (#365 / #369). Command lifecycle
consumption is delivered: 956 Applies / 53,754 records, zero install overruns;
all seven pre-install failures and every loss remain counted. D-098 archived
all 103 edd986c4 root files first. No D-132 handover exception was needed.

### Requests to orchestrator

1. **2026-11-01 — AT11, devstack and platform operators (carried).** Supply
   native dual-stack/member-platform routes and packet captures. Metadata,
   6to4 and mechanism tests do not establish the platform matrix. Outside this
   lane; only this item remains stopped.
2. **2026-11-01 — AT13, acceptance/operators (carried).** Execute the full
   continuous crash/NetBird restart/OS reboot/updater/physical-arm64 matrix.
   Include qualified process-stop evidence: this run's clone fixture sends
   SIGTERM then SIGKILL after 20.007193 s (exit 137); its surviving stored lease
   and counted ordinary restart do not verify graceful agent shutdown.
   Shared clone lifecycle is `tests/e2e/second_exit.go`; no outside edit made.
3. **2026-11-01 — ISO-F6-01 observation convergence, agent-core / devstack,
   with isolation follow-up.** AT10 is now **8/1/0**, not the previous run's
   pass. Five assertions pass; the sixth requires a dirty-newer-generation
   rejection report which this run did not retain. All 12 kernel samples keep
   forwarding closed and foreign objects present. Generation-30 failed Applies
   2311/2320/2329/2338/2347/2356/2365 are retained, but the central samples all
   show generation 29 at 09:55:00.452272372 UTC. The old-generation rejection
   is accepted at 09:56:41.080290184, after removal at 09:56:25.090642995.
   No generation-30 rejection appears anywhere in the receiver. See
   [exact indexed evidence](iso-f6-01-observation-gap.json). Assess observation
   retry/first-deciding-body retention in `node/agent/observe.go` (`observe`,
   `flush`, `observationState`) after control restoration and provide a
   qualifying current observation/readiness boundary for the next independent
   sample. An owned harness readiness change can follow that assessment; no
   timeout inflation, substitute assertion or product change is made here.
   Those agent paths are outside this lane, so that item is stopped and routed.

There are **three requests**: two carried operator gaps and one new observation
evidence gap. The latter is not evidence of forbidden forwarding. Historical
install-overrun causes, the old ISO-F6-03 baseline writer, the original foreign-tc
creator and this run's later qdisc/read_failed cause remain unproven.
