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
