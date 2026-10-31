# Follow-up 6 — provisional window measured; tc rejection remains open

Final run `bf671f3a-c6ac-4824-9107-425c4c5a5f02`, harness `e711ebd822eda2ea1b5ba10b187818fda6777a74` (`source_dirty=false`). Images were built on origin/main baseline `5d1e5ce8727ca824fa24197ce2568bd513bac7f3` (through #256); the commits above that baseline change only this lane’s harness and evidence. Product UTC `2026-10-31T04:37:28Z`–`2026-10-31T05:01:32Z`. Node incarnation `01a1903f-a5db-7d5f-a305-f41a17c0e519`. Immutable image IDs are in each case JSON. Mechanism lab 18.549 s; assembled test 1444.38 s. The full make exited **2**, `lab=0 stack=1`, because blocked and failed acceptance cases remain. The final run required one additional real node restart after a tc rejection; that intervention is retained in `recovery-action.json`.

| Product case | Result | Passed / blocked / failed subchecks | Remaining blocked or failed checks |
|---|---|---|---|
| [AT10 JSON](AT10.json) / [Markdown](AT10.md) | blocked | 7 / 1 / 0 | node interface and serving generation change |
| [AT11 JSON](AT11.json) / [Markdown](AT11.md) | blocked | 2 / 1 / 0 | IPv6 platform matrix |
| [AT12 JSON](AT12.json) / [Markdown](AT12.md) | blocked | 5 / 5 / 0 | same-member A1-to-A2; same-member A2-to-A1; cross-member A1-to-B1; cross-member B1-to-A1; designated DNS exception |
| [AT13 JSON](AT13.json) / [Markdown](AT13.md) | fail | 5 / 1 / 1 | continuous crash/NetBird/reboot/updater transitions; short whole-node lease expiry |

Requirement mapping (spec §4 and §19): AT10 → ISO01/ISO02; AT11 → ISO01/ISO03; AT12 → ISO01/ISO04/DNS03; AT13 → ISO02/ENT02/SEC01. The offline-lease interpretation is §16.3.

## Provisional lease sampling — measured on genuine recovery

[Agent-core #250](https://github.com/TED-H-com/Freedom-VPN/pull/250) states an approximately **5 s** lab window, beside usage follow-up 3’s **5021 / 5025 ms**. The read-only observer captured **19 qualifying samples** at a **250 ms** target cadence (in-window median **250.001 ms**, maximum **251.026 ms**). Ready-observed → populated lease was **4767 ms**. Ready-received → populated lease was **4760.713 ms** (**4760 ms** truncated), the comparable timestamp definition used by the usage helper.

| Event | UTC timestamp | Evidence |
|---|---|---|
| Node ready observation | `2026-10-31T04:51:43.419446069Z` | report `01a1904c-a0bb-76dc-8cdd-f804c12b371a` |
| Central acceptance of ready | `2026-10-31T04:51:43.426204Z` | observation `b09b4bb2-b018-478e-86ec-5c21d42b7c46` |
| First qualifying poll | `2026-10-31T04:51:43.671715545Z` | generation `15`, bindings > 0, zero grants |
| Last qualifying poll | `2026-10-31T04:51:48.172657421Z` | same provisional lease |
| Populated lease issued | `2026-10-31T04:51:48.186917Z` | same generation |
| Accepted and kernel forwarding admitted | `2026-10-31T04:51:48.422301658Z` | report forwarding true, live member accept present |
| Counted member transfer confirmed | `2026-10-31T04:51:53.384928680Z` | AT12 node-management-22 heartbeat, generation 15, bytes 18545 → 21177 |

Every qualifying sample has grant-less lease `9fd0e002-19b7-4365-acfc-cf1dfca754d4`, desired bindings, matching generation/observed lease, ready/passed status, exactly the lease/qos/source_binding checks with `AWAITING_MEMBER_GRANTS`, exactly one `errors[]` entry at stage `lease`, report forwarding false and no live member accept. [Full real report](provisional-ready-report.json), [samples and all 19 timestamps](provisional-recovery.json). The first and last live samples span **4500.942 ms**. These are repeated state polls inside one window, not 19 distinct agent reports. Database and kernel reads are sequential and separately timestamped.

This window followed the extra recovery restart recorded in [recovery-action.json](recovery-action.json), after the tc rejection described below. The observer made no mutations and submitted no synthetic agent observation. Its source was `604f4596e29572c654e2e285acad301a199ca672`; stack images and the main driver remained `e711ebd822eda2ea1b5ba10b187818fda6777a74`. The explicit offline review checks same-run/build provenance and the counted control, then changes only the provisional check. The original main-driver AT10 record is preserved under [history/follow-up-6-bootstrap-only](history/follow-up-6-bootstrap-only/). The positive control’s own `at` is the start of its wait; correlation uses the post-transfer heartbeat above.

The initial bootstrap remains separately recorded: **452** polls, median **249.992 ms**, maximum **281.919 ms**, **0** same-generation grant-less/bindings-positive samples, **0** accepted awaiting-grants reports. It changed from an empty generation-5 lease to a populated generation-6 lease; it did not traverse the qualifying state. [Timeline](convergence-timeline.json), [analysis](sampling-analysis.json), [issuance history](bootstrap-renewal-evidence.json). Those setup latencies are not substituted for the recovery measurement. The 55 s ready / 60 s request fake-clock proof in #250 uses a 90 s heartbeat and is distinct from this lab.

## Interface-change tc rejection and assisted recovery

AT10’s six destination controls passed. During the subsequent interface-change test, the agent emitted `HEALTH_CHECK_FAILED` / `TC_FOREIGN_FILTER` at `2026-10-31T04:40:51.80844254Z`. Observation `f2cfdff8-3e46-49e8-86f1-b38128ea6cd4` / report `01a19042-af78-79f0-8dd6-9d13ce85c2f2` shows safety failed and forwarding off. The interface loop made 3968 attempts over 60.026981429 s with zero delivered, but its after-control failed; AT10 continuity stays blocked. The four AT12 peer checks then lacked counted positive controls and also stay blocked. No foreign tc state was planted before this event.

The concurrent capacity update was created at `04:40:52.423892Z`, after the error, so it is not identified as the trigger. [Chronological evidence](interface-change-evidence.json) includes the later read-only, post-rollback tc state, which cannot identify the rejected object. A real node restart at `04:51:02.744373Z`–`04:51:12.849221Z` recovered forwarding through the normal entrypoint. Later management, forged source, withdrawal and the disable/revoke/restart/finite-member AT13 controls passed; whole-node expiry failed its continuity assertion. These do not promote the earlier blocked checks. Netlab and agent-core own the required reproduction and rejected-state evidence; [requests](requests.md) gives exact locations.

## Serving gate and D-079 attribution

`foreign_rules`, verbatim: `[]`.

`foreign_rule_classification`, verbatim: `no foreign rule; Docker embedded DNS NAT absent; serving pass attributable to lab fix #240; D-079 #241 present but not exercised`.

D-079 **#241** is built (`verify_foreign.go` commit `a4191fa`). There are no foreign rules or Docker NAT chains for it to classify and no `loopback_scoped` entries. The pass is attributable to the lab fix **#240**. Accepted serving report `01a19041-2044-7558-8ee1-2666e65e2711`, observation row `4a1dc037-c2c0-4955-90a5-c19a2928e998`, at `2026-10-31T04:39:09.636348464Z` is retained verbatim in AT10. The harness did not change foreign nft/tc state.

## tc section, verbatim

The **mechanism lab** returned this `VerifyServing` evidence on #256 (`lab-AT10-tc.json`); HTB/fq_codel objects are owned and every filter attachment is empty:

```json
[{"interface":"fv-fh","attachment":"qdisc","objects":[{"kind":"htb","handle":"1:","root":true},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"fq_codel","handle":"2f:","parent":"1:2f"}]},{"interface":"fv-fh","attachment":"root","objects":[]},{"interface":"fv-fh","attachment":"ingress","objects":[]},{"interface":"fv-fh","attachment":"egress","objects":[]},{"interface":"fv-fh","attachment":"1:1","objects":[]},{"interface":"fv-fh","attachment":"1:2f","objects":[]},{"interface":"fv-fh","attachment":"1:30","objects":[]},{"interface":"fv-fh","attachment":"1:fffe","objects":[]},{"interface":"fv-fh","attachment":"2f:","objects":[]},{"interface":"fv-fh","attachment":"30:","objects":[]},{"interface":"fv-ih","attachment":"qdisc","objects":[{"kind":"htb","handle":"1:","root":true},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"fq_codel","handle":"2f:","parent":"1:2f"}]},{"interface":"fv-ih","attachment":"root","objects":[]},{"interface":"fv-ih","attachment":"ingress","objects":[]},{"interface":"fv-ih","attachment":"egress","objects":[]},{"interface":"fv-ih","attachment":"1:1","objects":[]},{"interface":"fv-ih","attachment":"1:2f","objects":[]},{"interface":"fv-ih","attachment":"1:30","objects":[]},{"interface":"fv-ih","attachment":"1:fffe","objects":[]},{"interface":"fv-ih","attachment":"2f:","objects":[]},{"interface":"fv-ih","attachment":"30:","objects":[]}]
```

The **assembled product observation** has no `tc` or `serving_evidence` section (`has_tc=false`, `has_serving_evidence=false`). [Metadata](serving-readback-metadata.json) identifies the observation above. This is routed to netlab with agent-core/contracts; the lab section is not relabeled as an assembled observation. No foreign tc fixture was planted.

## AT13 and the owned wait correction

- **disable published exit: pass.** result=timeout; route=192.0.6.10 via 172.28.33.1 dev eth0 src 172.28.33.128 uid 0      cache
- **revoke established TCP and UDP: pass.** binding removed and new serving generation=true; sibling counted=true; native route withdrawal and kernel revoke are not independently attributed
- **post-revoke kernel ingress attribution: pass.** revoked 100.89.119.208 ctunku+1 and zero forwarded=true; sibling u_01a19040ecc07e89a1b8e577f056f35c +1=true
- **node restart with durable identity: pass.** container restart=fvpn-w4-isolation-node; recovered counted=true; continuous forbidden-flow coverage not established
- **lease member expiry under control loss: pass.** five new TCP requests after finite signed member deadline; control and worker stopped; admission bytes frozen=true; route retained throughout=true; node lease has a later deadline
- **short whole-node lease expiry: fail.** 60s lease held after control loss; near-deadline forward=false; baseline within one tick=false; ingress without uplink=true

Finite-member deadline `2026-10-31T05:00:00Z`, control stopped `2026-10-31T04:59:35Z`; post-deadline bytes `2632` → `2632`. Whole-node lease, verbatim: `{"issued_at":"2026-10-31T05:00:28.561634193Z","expires_at":"2026-10-31T05:01:28.561634193Z","generation":"29","max_offline_seconds":60,"member_count":2,"source":"node-store"}`. Signed deadline `2026-10-31T05:01:28.561634193Z`, baseline at `2026-10-31T05:01:28.675726567Z`, tick `5s`.

Fleet PUT evidence: `200 etag="3" body={"default_max_offline_lease_seconds":86400,"max_offline_lease_seconds":60,"resource_version":"3","updated_at":"2026-10-31T05:00:23.781625Z"}`; restore: `200 etag="4" body={"default_max_offline_lease_seconds":86400,"max_offline_lease_seconds":86400,"resource_version":"4","updated_at":"2026-10-31T05:01:32.451479Z"}`. Both `fleet_config.update` audit payloads are retained in AT13.

Whole-node expiry remains **fail**: the near-deadline request did not count, and `early_baseline_at=2026-10-31T05:01:27.784897861Z` labels a read begun 776.736 ms before the signed deadline. The read completion was not recorded; the heartbeat was stale. [Timing analysis and limitations](expiry-timing-analysis.json) distinguish those observations from an exact transition or validated product defect. The negative after expiry passed with bytes frozen at 50370. Agent-core and acceptance must reconcile monotonic timing and retain finer probe/read timestamps; the strict case result is unchanged.

Disable operation transitions on the corrected run:

- `2026-10-31T04:54:37.600526439Z pending`
- `2026-10-31T04:54:39.607503706Z partially_succeeded`
- `2026-10-31T04:54:43.61813201Z succeeded`

The first full run (`82eac082`, harness `5ddd112`) stopped its disable wait at `partially_succeeded`, before agent acknowledgment, and restored intent. The owned helper now waits for observation-driven completion. That diagnostic run is preserved under [history/follow-up-6-diagnostic](history/follow-up-6-diagnostic/); it is not the final acceptance run.

The AT13 evaluator SHA256 remains `d4cd4dfd0957cace3ef9cd63168a4653c75ac1d0ba07d5c5ca1af00603f38d53`; `capture.go` and `capture_review_test.go` were not changed. IPv6/platform, designated-DNS and the continuous fault matrix retain follow-up 5’s blockers and owners. See [requests](requests.md).

## Gates and teardown

Area lint, root `make go-lint`, `go vet ./...`, owned tests and amd64/arm64 builds passed. `go test ./...` fails only in `tests/handover/TestHandover_Followup13Claims`: the dated `docs/ops/README.md` does not yet contain this run ID and its refreshed AT12/AT13 counts; it also requires a handover refresh when AT13 changes to fail. Those paths are outside this lane; the orchestrator must refresh the handover. The pre-refresh whole-module run passed, but that is not the final evidence gate. The live gate remains exit 2 for blocked and failed product cases. `git diff --check` and the committed-evidence hygiene test pass. arm64 cross-compilation is not a hardware run. [Command exits and actual output tails](validation-results.json) retain the exact final handover failures.

Final inventory at `2026-10-31T05:02:34.039822Z`: **`{"containers":0,"networks":0,"volumes":0,"images":0}`**. [Full inventory](cleanup-inventory.json). The teardown fix from #246 leaves no task-prefixed object, including the target fixture and cache tags.

Follow-up 5’s 33 records remain byte-identical under [history/follow-up-5](history/follow-up-5/). [RUN.md](RUN.md) documents reproduction and evidence boundaries.
