# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 28](../handover/follow-up-28.md) (2026-11-01 EST):** Through #390: r19 H-1 / M-2 / L-2 fixed; devstack 25 closure explained as tracer interference; native delivery delay remains. Updates 14 adopts parent receipts, AT23–AT27 5 / 5. D-135–D-138, D-140 and D-141 remain provisional. Node diagnostics and traceroute guidance added; RELATED ICMP owner question open. Historical failures stay failed.
**Previous cut — [follow-up 27](../handover/follow-up-27.md) (2026-11-01 EST):** Through #385: isolation F14-01 fails the D-104 boundary; members 15 adds serving-node ingress gaps; r19 archived with routed findings. D-140 permits boundary scheduling only. D-141 closes D-139: option (b) is not implemented; option (c) stays. Historical failures stay failed. No new operator guidance.

## Follow-up 28 acceptance snapshot (2026-11-01 EST)

Fixed main **aae6763f**, **#386–#390**. Devstack 27 and isolation 15 were running
at task start; no later results enter this cut. [Evidence and dated requests](../handover/follow-up-28.md).

| Delivery | Accepted reading at #390 |
|---|---|
| Devstack 26 (#386) | Devstack 25's closed node was its TEST ONLY foreign nft tracer: ISOLATION_POLICY_INVALID, deny baseline, kernel 0 / central applied 5. Harness artifact, not a product defect. Corrected member-only 4 / 0; F14-style re-grants 17 / 1, af718646 delivery 5.671 s late behind a loopback-source response. Native delay remains; owner repair/upstream direction unblocked. duplicate_chain classification and 15 s stop path with PID1 TERM forwarding delivered. Dated run evidence pinned. |
| Handover 27 (#387) | #383–#385 at 37db5405; 19 subtests / 3 exact rows / 14 snapshots / 182 archive file checks retained. #381 live row equals its frozen snapshot, sha256 400d5cf9…; no action. |
| Agent-core 34 (#388) | r19 M-2 production-reachable, fixed by applied snapshot committed with ACK and generation-matched refs. L-2 fixed by durable reservation before Apply; about 1.1 ms, p99 1.59 ms before restricting install when deletes are queued. Accepted under D-138, cost explicit. |
| Netlab 24 (#389) | r19 H-1 fixed; download denied/host/resolver source guards precede generic established / related accepts, exact DNS replies stay ahead. ctdenyd is diagnostic_v1, never usage. A,B,A,B accepted under D-138; no B-only failure. A2 D-120 on unchanged main stays failed; both missing-successor runs route to netlab 25. RELATED ICMP question is open, not a defect or decision. |
| Updates 14 (#390) | 759d244a archived: 48 roots. 2704600f on #387's tree: AT23–AT27 PASS 5 / 5; installer 27 / 27; 13 POSTs, AT26 physical rollback, uncached fallback. All 11 stops returned 0; six separate agent exit-0 parent receipts (AT23 first is a new sample), five repeat reuse. No SIGKILL attempt; stored lease present at every stop/start. Parent receipt and 15 s path adopted. Current run cited by roadmap row / PR only until updates 15 archives it. Updates F13-01 remains the unfilled historical gap, unrelated to members F13-01. |

No new decision since D-141; D-135 / D-136 / D-137 / D-138 / D-140 / D-141
remain provisional. Members 15's serving-node signature remains with devstack
27, cause unproven. New operator guidance: diagnose from the member side or
capture without changing the node ruleset; the verifier closes an invalidated
node. Traceroute gets no reply from spec-denied on-path sources under the new
source guard; owner question open, no fix promised. Historical failures stay
failed. Older snapshots below retain their own cutoff.

## Follow-up 27 acceptance snapshot (2026-11-01 EST)

Fixed cutoff **37db5405**, **#383–#385**, with no merged-after-cutoff PR at task
start. Devstack 26, netlab 24 and agent-core 34 were running; no result from them
enters this cut. **Merged after the cutoff: #386.**
[Pinned evidence and decisions](../handover/follow-up-27.md).

| Delivery | Accepted reading at #385 |
|---|---|
| Isolation 14 (#383) | 38074149 archived: 112 roots. 56d20530 on #382: AT10 PASS 9 / 0 / 0, ISO-F6-01 6 / 6; AT12 PASS 10 / 0 / 0; AT11 BLOCKED 2 / 1 / 0. AT13 FAIL 5 / 1 / 1, isolation F14-01, distinct from members F14-01: closure (D−162.2, D−97.6] ms straddles D−101 ms. One zero-member bootstrap hold joined and ended generation_matched; no populated hold, alarm, escalation or active-stop retention exercised. Living roots cited by roadmap row / PR only. |
| Handover 26 (#384) | Previous #374–#382 cut at 9dde5eeb plus 5f4b455c: 21 subtests / 9 exact rows / 17 snapshots / 1,075 archive and run file checks retained. |
| Members 15 (#385) | 258ba504 archived: 70 roots. e80bb592: 2 pass / 6 blocked / 2 fail, 644 / 6 / 2 subchecks. AT02 FAIL: 93 successful Applies / 0 failed, 65 paired baselines, 277 / 4,131 attempts without egress (249 absent at ingress, 28 ingress without egress). AT01 first selection fails after generation-5 readiness: five member SYNs, zero node or origin packets; three re-grants pass. Members F14-01 stays failed historically. 682d4434 relay order live-qualified; receive-only dual MTU-1280 FAIL, 12,012 bytes in 12,011 ms. Gzip and plain AT02 index remain living, claims pinned by roadmap row / PR only. |
| r19 | Exact 41,050-byte archive. H-1 → netlab 24; M-2 / L-2 → agent-core 34, routed; in progress. M-2 production reachability unknown. Design-only findings are D-141 reconsideration conditions. |

D-140 permits only the boundary schedule change; the predicate stays byte-identical
and run 56d20530 stays failed. D-141 closes D-139: option (b) is not implemented;
option (c) stays. D-135 / D-136 / D-137 / D-138 remain provisional. Owner decisions:
D-135 + D-136 confirmation (AT02 sibling outage given D-141), D-137 hardening,
D-138 / D-140 / D-141 overturn options, F11-01 repair direction awaiting devstack
26. Serving-node missing-ingress samples route to devstack separately from the
closed-node timeouts; cause unproven. Historical failures stay failed.
**No new operator guidance.** Older snapshots below retain their own cutoff.

## Follow-up 26 acceptance snapshot (2026-11-01 EST)

Fixed main **9dde5eeb**, #374–#382, including **5f4b455c** correcting #381.
Nothing merged after the cutoff at task start. r19, isolation 14, members 15
and devstack 26 were running; no results imported. Handover ran no new live
acceptance. [Evidence and dated hand-offs](../handover/follow-up-26.md).

| Lane | Delivery and remaining boundary |
|---|---|
| Handover 25 (#374) | Previous #370–#373 cut at 9cd7a053: 18 subtests / 4 rows / 15 snapshots / 270 file checks; r18 archived. |
| Members 14 (#375) | 4182dd3d archived. 258ba504: AT03 / AT08 pass; AT02 fails, 58 brackets / 21 losses; AT04 duplicate chains fails; AT09 relay confounded by harness order, reorder not live-qualified. F14-01 new, cause unproven. Living roots cited only by row / PR. |
| Agent-core 32 (#376) | r18 M-1 / M-2 / M-3 / L-2 and ISO-F6-01 fixed; cooperative shutdown 13 s, distinct stale hold and alarm. |
| Netlab 23 (#377) | r18 L-1 fixed; members F13-01 origin unproven after 0 / 3,300 duplicates, typed rejection remains. |
| Installer 9 (#378) | Agent TimeoutStopSec=15s, watchdog stop ordering; 20 s per-unit replacement stop, stop-failed journal / rollback. One dated operator-guide §3.1 line; no other new operator guidance. |
| Agent-core 33 (#379) | D-139 option (b) design, 7 tests / 558 table subtests; signed constraints or baseline; final Apply and feature-enabled restart baseline. Option (c) stays in production; no acceptance result. |
| Isolation 13 (#380) | 1c25ee44 archived. 38074149: AT10 PASS 9 / 0 / 0, ISO-F6-01 6 / 6 by accepted reports; AT12 PASS 10 / 0 / 0; AT11 / AT13 operator blocks. Alarm unexercised, historical block retained. Living roots cited only by row / PR. |
| Devstack 25 (#381) | 32 attempts, 26 pass / 6 timeout, all end closed. Four already sampled 0 / false before request; 0c27cac0 5 / true → 0 / false. Cause unproven. TEST ONLY receipt delivered; updates adopts next, historical AT23 unchanged. |
| Wire-nodes 34 (#382) | r18 M-3 central recognition independent of ACK, own signed lease only; specific classification, migration 00194 boot retention, one audit escalation per incident. Assembled assertions pending; no AT pass. |

D-135 / D-136 / D-137 / D-138 stay provisional; **D-139 is a provisional design
track pending r19 and a recorded representation decision before implementation**.
F11-01 repair direction waits on devstack 26; F14-01 and the TEST ONLY harness
stop-path gap also route there. Alarm / central escalation need an assembled run.
[Current findings and owner decisions](../handover/README.md#findings-register-at-the-follow-up-26-cutoff).

## Follow-up 25 acceptance snapshot (2026-11-01 EST)

Fixed main **9cd7a053**, through **#373**, all #370–#373. Nothing merged after
the cutoff at task start. Agent-core 32, members 14 and devstack 25 are running;
no results imported. [Full evidence and dated hand-offs](../handover/follow-up-25.md).
No new operator guidance or live acceptance run by handover.

| Lane | Delivery and remaining boundary |
|---|---|
| Devstack 24 (#370) | F11-01 located before IPv6 prerouting, Ip6InHdrErrors 0→1; loopback-source validation consistent, no drop tracepoint. Same request succeeds with one SYN; old IPv6 timeout not reproduced in four attempts; separate IPv4 d556673b timeout failed/unattributed. Members F13-01 raw bytes/parser error retained privately with public digest, cause unproven. NetBird 0.82.0 pinned; owner repair choice. |
| Handover 24 (#371) | #363–#369 at 5bd75102; 21 subtests / 7 rows / 21 snapshots / 357 file checks. Historical cutoff stays frozen. |
| Updates 13 (#372) | 44-file 2babf9b6 archive; 759d244a regular AT23–AT27 5 / 5, installer 27 / 27. Eleven successful stop commands, lease kept; AT23's first process exit unobserved (updates F13-01 → devstack 25, distinct from members F13-01). Startup port collision retained; configurable slot. |
| Isolation 12 (#373) | 103-file edd986c4 archive; 1c25ee44 AT12 10 / 0 / 0 pass, AT10 8 / 1 / 0 blocked on ISO-F6-01 assertion 6: kernel denial held, no generation-30 report at central. Earlier #360 pass is history. AT11 / AT13 operator-blocked; per-Apply commands joined, producer stream lossless gzip. |
| r18 review | Exact archive, five OPEN findings: M-1 / M-2 / M-3 / L-2 → agent-core 32; L-1 → netlab. TimeoutStopSec → installer and central stale-hold reading → wire-nodes, both after agent-core 32. What held is the review's statement, not acceptance. |

No new decision since D-138; D-135 / D-136 (04:3x correction) / D-137 / D-138
stay provisional. Owner decisions: D-135 + D-136 confirmation, D-137 hardening,
D-138 overturn option and F11-01 repair direction. Every historical failure
stays failed; every other open finding retains follow-up 24's status.

## Follow-up 24 acceptance snapshot (2026-11-01 EST)

Fixed main **5bd75102**, through **#369**, all #363–#369. Nothing merged after
the cutoff at task start. r18 and devstack 24 are running, with no results in
this cut. [Full evidence and dated hand-offs](../handover/follow-up-24.md).

| Lane | Verified delivery and remaining boundary |
|---|---|
| Security-ops 14 (#363) | Interval-start periods in PITR/rate-class fixtures; offline month/year-boundary proof; replay per retained period. Product ingestion unchanged, no live suite, recorded rows unchanged. |
| Handover 23 (#364) | #357–#362 at bc011b6f; 16 subtests / 6 rows / 18 snapshots / 421 file checks; month-boundary incident; D-136 / D-137 provisional. |
| Netlab 22 (#365) | 132-file D-098 archive; opt-in TEST ONLY lifecycle/render capture; two 90-minute windows, no qualified second B agent leg. D-138 accepts observation; D-135 assessment, every loss retained. Requests 1–2 delivered by #369. |
| Usage AT 7 (#366) | 19-file 71b1a160 archive; interval/summary/held-snapshot period fixes and offline UTC-boundary proof; November b8a6e9e1 AT14–AT20 pass; 0 requests. |
| Devstack 23 (#367) | Routed 1280: IPv6 PTB / learned PMTU, IPv4 DF=0 transit fragmentation. F11-01 traced to NetBird 0.82.0 loopback proxy; positive IDNA; F12-04 not reproduced. 3 requests. |
| Members 13 (#368) | 52-file b98b78ee archive; 4182dd3d 2 pass / 6 blocked / 2 fail. AT02 61 baseline brackets / 21 losses, no failed Apply; AT04 F13-01 parser failures. F12-02 live-qualified, F12-03 quiet-window pass, historical rows stay failed. Known-PMTU passes; receive-only limitations. 10 requests. |
| Agent-core 31 (#369) | D-136 correction; 80 transitions / 1,013 losses retained in two populations, no ceiling. TestLab04 cancellation fix and recorded graceful exit; local completion at most 10 s, control I/O cancellable. TEST ONLY caller requires both gates. Intersection serving is for r18 review only; no failed live timing threshold, no D-133 run. |

D-135 / D-136 / D-137 / D-138 stay provisional. Owner decisions: D-135 +
D-136 confirmation, D-137 hardening option, D-138 overturn option. F13-01 raw
bytes/parser error and F11-01 drop investigation are assigned to devstack 24;
F12-04 original-query instrumentation and routed-hop adoption to members.
No historical failure is promoted. Install-overrun cause, F9-INFLIGHT,
F18-01 intended negative, ISO-F6-03 writer, Dependabot #22 and
blocked_response_ttl owner value retain their boundaries.

## Follow-up 23 acceptance snapshot (2026-11-01 EDT)

Fixed main cutoff `bc011b6f`, through **#362**, after #361. Handover ran no new
live acceptance. See [full evidence](../handover/follow-up-23.md) and the
[decision / findings register](../handover/README.md#follow-up-23-decisions-register-2026-11-01-edt).
Security-ops 14 and netlab 22 / members 13 / devstack 23 / agent-core 31 /
usage AT 7 were absent from main at start; their results are excluded.

| Delivery | Measured scope at the cutoff |
|---|---|
| Agent-core 30 (#357) | Month-boundary TestUsageCheckpointRoundTrip fixed with one 2026-11-01T00:00:00Z clock and period from interval start; 87-file sweep, one fix, six other-lane findings routed. Product validation unchanged. |
| Handover 22 (#358) | Frozen #356 cut: 19 subtests / 5 rows / 22 snapshots; 114-file devstack-20 and 82-file netlab-20 inventories; D-134, provisional D-135. |
| Updates 12 (#359) | 43 files archived; 2babf9b6 regular AT23–AT27 5 / 5 through shared TEST ONLY wrapper, installer 27 / 27. F11-01 closed by regular run; F11-02 closed #351; F18-01 intended negative. First path-bug attempt retained; 0 requests. |
| Isolation 11 (#360) | 100 files archived; 49 historical failed installs replay with host / verification unavailable, cause unproven. edd986c4 AT10 9 / 0 / 0 and AT12 10 / 0 / 0 pass, AT11 / AT13 operator-blocked. Fresh D-104 passes, all losses counted; 952 Applies / zero install overruns, seven intended pre-install rejections. 2 requests. |
| Members 12 (#361) | 43 files archived; b98b78ee F11-02 closed by populated-grant rows, F11-03 blocked (89 Applies succeeded). New F12-01 stale-lease baseline bracket / nine losses, F12-02 timing-export cap, F12-03 negative-cache and F12-04 UDP SERVFAIL. Streamed reader not live-qualified; 12 requests. |
| Devstack 22 (#362) | 63 files archived; native IPv6 F8-02 16,016 / 64,064 bytes, 15 oversized reverse losses, no PTB. Sender constraint passes by fragmentation. Five-SYN pattern reproduced with loopback-source handshake response, no type-4 data. Capture exports; 2 requests. |

[Operator additions](../handover/operator-guide.md#operator-changes-at-362)
cover provisional D-136 stale-lease baseline semantics and D-137 AT09 scope.
D-135 + D-136 require confirmation together; D-137's lower-tunnel-MTU / MSS
hardening remains an owner option. Historical failures stay failed and
unattributed, including F8-02's receive-only silent-drop limitation. The
incident lesson is to derive a half-open interval's month from its own start,
never pair a calendar literal with the wall clock. Earlier snapshots below
retain their cutoffs.

## Follow-up 22 acceptance snapshot (2026-10-31 EDT)

Fixed main cutoff `45209609`, through **#356**. Handover ran no new live
acceptance. See [full evidence](../handover/follow-up-22.md) and the
[frozen decision / findings register](../../tests/handover/testdata/history/follow-up-23/register22.md).
Devstack 22, members 12, isolation 11, updates 12 and netlab 22 were not merged
at start; their results are excluded. D-135 remains provisional, pending the
owner's confirmation.

| Delivery | Measured scope at the cutoff |
|---|---|
| Devstack 21 (#352) | 114 follow-up 20 files archived; updates F11-01 fixed, shared-wrapper 33b9621a 3 / 0 / 0. DNS cba88bbc 13 / 0 / 0, 36 / 36 complete traces, 85 requests / 70 retained identities; export validated, F9-INFLIGHT not reproduced. Members F11-01 interval bounded; F11-03 install stage established, cause unproven. F18-01 intended D-125 negative; first origin-slot .24 collision retained. |
| Installer 8 (#353) | No behaviour change; field-by-field producer/consumer match and literal-JSON regression. Per-distinct-archive cost corrected; F10-01 CLOSED, privileged 13 / 13. Distinct-endpoint CA / private mirrors, physical arm64, power loss, production remain unmeasured. |
| Handover 21 (#354) | Frozen #351 cut: 21 subtests / 7 rows / 13 snapshots / 350 owner-archive files. |
| Netlab 21 (#355) | 82 prior files archived. F8-02 IPv6 fixture: 19 reverse 1,360-byte packets absent, zero capture drops / no PTB; one mechanism explains both underlays, native capture pending. D-134 reserve [D−101 ms, D], full guard distributions; no predicate change. F11-03 baseline→recovery measured with every loss counted. |
| Agent-core 29 (#356) | Controlled CPU contention does NOT reproduce install overrun: max render→installed 53.133 ms; historical cause unproven. Opt-in TEST ONLY host context, 1,874 Apply brackets audited, D-106 unchanged. Node-wide rollback until next successful Apply, no outage ceiling; S2 later member_path_unconfigured after successful verifier. AT02 stays failed. No D-133 A/B triggered. |

The [operator changes](../handover/operator-guide.md#operator-changes-at-356)
cover D-134 expiry-window semantics and provisional D-135 rollback only.
Updates F11-01 and installer-side F10-01 are closed. F8-02 native capture,
members F11-01 / F11-02 / F11-03, install-overrun cause, F9-INFLIGHT,
ISO-F6-03 writer, Dependabot #22 and production blocked_response_ttl remain
open; F18-01 remains the intended negative. Historical failures stay failed
and unattributed. The snapshots below retain their original cutoffs.

## Follow-up 21 acceptance snapshot (2026-10-31 EDT)

Fixed main cutoff `6d1e320d`, through **#351**. Handover ran no new live
acceptance. See [full evidence](../handover/follow-up-21.md) and the
[frozen decision / findings register](../../tests/handover/testdata/history/follow-up-22/register21.md).
No new decision; D-133 remains the latest. Devstack 21, netlab 21 and agent-core
29 were not on main at start; their results are excluded.

| Delivery | Measured scope at the cutoff |
|---|---|
| Dns-publish 9 (#345) | F9-HIT / F9-OTHER / F9-ROUND and inherent F9-INFLIGHT on pinned defaults; TTL is a maximum lifetime. Mandatory dns-positive-cache gate. Historical F10-03 failed / unattributed; blocked_response_ttl still owner's. |
| Updates 11 (#346) | daddad24: installer 27 / 27; regular AT23–AT27 5 / 5, physical rollback witnessed by AT27; generated reporting without adapter, 13 POSTs. Artifact-origin question closed by measurement, no rejection or exception needed. F18-01 remains negative; updates F11-01 to devstack 21. |
| Handover 20 (#347) | Frozen #344 cut: 18 subtests / 7 pinned rows / 38 immutable inputs, D-133 registration retained. |
| Members 11 (#348) | 7edb263f: F10-02 new row 186 / 186, F10-03 new row passes, historical failures unchanged; F10-04 closed. Full AT02 121 / 0 / 2 (F11-02 / F11-03); AT09 37 / 5 / 1 (F8-02 IPv6 recurrence); new F11-01 route gap. 2 pass / 6 blocked / 2 fail. |
| Portal 19 (#349) | Dependabot #22 cannot close from lockfile: full override breaks jsx-a11y; files byte-identical. Development only, no production-bundle path. portal-verify 74 / 74; portal-e2e 9 / 9; owner disposition pending. |
| Isolation 10 (#350) | bfcc3880: real D-087 AT10 PASS 9 / 0 / 0, 962.448 ms; AT12 PASS 10 / 0 / 0; AT13 BLOCKED 6 / 1 / 0, fresh D-104 passes but historical ISO-F6-03 writer unattributed; AT11 BLOCKED 2 / 1 / 0. Independent 968-guard distributions and both failed attempts retained. |
| Updater 19 (#351) | Installer-shaped config builds HTTPSink with exact policy and distinct server trust; no reporting-field divergence. Updates F11-02 offline paragraph fixed: verified archive retained, only absence downloads. D-132 audit zero docs/e2e reads; physical arm64/power loss/fleet/production/worst-case bounds still unmeasured. |

Open: ISO-F6-03 writer; F8-02 on both underlays; members F11-01 / F11-02 /
F11-03; updates F11-01; F18-01; Dependabot #22; two D-133 measured limits;
blocked_response_ttl; AT11 real platform routes; AT13 full fault matrix including
physical arm64; AT29 / AT31 resources; real clients and QUIC / PMTU tooling;
commercial feed approval. The [operator changes](../handover/operator-guide.md#operator-changes-at-351)
cover positive-cache semantics, retained verified archives and the dependency
alert only. The previous snapshot below remains historical at #344.

## Follow-up 20 acceptance snapshot (2026-10-31 EDT)

Fixed main cutoff `96ec0630`, through **#344**. See the
[full handover](../handover/follow-up-20.md) and
[frozen decision / findings register](../../tests/handover/testdata/history/follow-up-21/register20.md).
This table consumes merged evidence; handover ran no new live acceptance.

| Delivery | Measured scope at the cutoff |
|---|---|
| Usage 6 (#338) | AT14–AT20 pass, run 71b1a160. Production planBaseline gap sequence 1 / zero records / 1 µs reaches ACK 1 before traffic; event 2 is 1,066 + 1,513 = 2,579 bytes. D-130 newly minted-event measurement closed; follow-up 5 archived (59 files). |
| Handover 19 (#339) | Frozen #337 cut: 16 subtests / 13 roadmap rows / 37 immutable inputs; D-129–D-132 and migration 00193 rollout / Down guidance preserved. |
| Fleet-update 13 (#340) | D-132 owned D062 fixture copies, empty-result failure and mutable-path guard; measured D-110 / D-127 central acceptance recorded. No product change. |
| Installer 7 (#341) | Updates F10-02 FIXED; generated HTTPS origin / server CA / mTLS reporting, exact control-origin allowlist, TEST ONLY loopback opt-in; privileged 13/13. Distinct artifact host remains an owner decision for updates 11. |
| Updater 18 (#342) | Missing sink fails explicit flush; result_url_missing keeps a sealed receipt pending, repaired config sends it once. Root executable / whole-root integrity fixtures; race/load 40/40. F18-01 devstack confirmation pending. |
| Devstack 20 (#343) | D-087 seam 7/0/0 twice, 3,980 / 4,112 ms; F10-01 shared side FIXED, switching 11/0/0. Counter diagnostics 247 samples; DNS 36 complete traces. Broader 15/0/1 retains instrumented MTU-1280 failure 10,010 / 64,064 bytes; F8-02 still unattributed. |
| Agent-core 28 (#344) | Verified same-boot adoption and same-lease repair; new baseline/heartbeat/watchdog evidence contract. ISO-F6-03 writer unattributed. D-133 A/B/A/B passes four live and four full agent suites; initial 152.125 ms failure attributed to host contention. Two measured limits stay open for netlab 21, not defects of record. README restart request closed. |

At #344, updates 11 (regular AT23–AT27 / artifact origin), isolation 10
(ISO-F6-03 / assembled AT10), members 11 (F10-02 / F10-03 / complete AT09 under
D-129), and dns-publish 9 (F10-03) are pending, with no results on main.
F18-01 belongs to devstack 21; F8-02 and D-133 limits to netlab 21. Native MTU
1280 under D-123, AT11 routes, AT13 continuous faults including physical arm64,
AT29 / AT31 resources, OS reboot, power loss and real platform clients stay open.
Dependabot #22 is an open medium minimatch dependency-hygiene item for portal,
pending reachability assessment. No acceptance tolerance or product value changed.

Historical follow-up 17 reconciliation: [follow-up 17](#follow-up-17-acceptance-snapshot-2026-10-31-edt),
main `7e044c71` through #313. Earlier tables retain the verdicts at their own cutoffs.

Operational runbooks, deployment guides, parameter calibration references, and disaster recovery checklists are maintained under [`docs/handover/`](../handover/README.md) alongside domain-specific operational references:

## Hand-Over & Production Runbooks (`docs/handover/`)

- [**Production Deployment Runbook (`../handover/deploy.md`)**](../handover/deploy.md) — PostgreSQL 17 + WAL archiving, TLS & mTLS listeners, OIDC + MFA (`issuer.Canonical` literal-slash rule), NetBird CE `0.82.0`, AdGuard Home `v0.107.59`, release origin, 5 domain config files, PKI & signing key ceremonies, node host prerequisites (`nft` with `D-079` `#241` `loopback_scoped` rule, `tc`, `ip`, `conntrack`), `fvpn-node-install` host installer & lifecycle commands (`#239`, `docs/node/install.md`), first node enrollment, fleet offline lease ceiling configuration (`GET`/`PUT /api/v1/admin/fleet-config`, `D-065`, `#176`, `#179`), `D-073` (`#228`) two-replica durable roaming candidate (`node_identity_endpoints` `candidate_hash` / `candidate_first_seen_at` / `candidate_last_seen_at` under `SELECT ... FOR UPDATE`), and post-`complete-recovery` review and re-authorization of devices fenced in `device_authorization_fences` (`D-076`, `D-077`, `D-081`, `#227`, `#238`, `#242`, `#244`, `GET /api/v1/admin/devices?authorization_unconfirmed=true` and `POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize`).
- [**Central PostgreSQL Backups, PITR & Post-Restore Reconciliation (`backups.md`)**](backups.md) — Pinned `pgBackRest 2.60.0` configuration, continuous WAL archiving (`archive_timeout = 300s`), systemd schedules, repository encryption & vault separation (`§16.2`, `§17.2`), monitoring queries/log lines, `deploy/backup/restore-pitr.sh` / `deploy/backup/verify.sh` / `deploy/backup/post-restore-checklist.sql`, ordered post-restore reconciliation protocol (`§17.3`–`§17.4`), and automated CI drill evidence ([`backups/`](backups/)).
- [**Backup and Restore Drill Record (`../handover/restore-drill.md`)**](../handover/restore-drill.md) — `AT30` point-in-time recovery checklist, Wave 4 measured RPO/RTO evidence (including `#221` live `AT30` second half and `#242` live **`AT30` `passed`** run `42396951-2d9a-49f4-95c3-45f33d9519d5`: `7.17 s` isolated `T0` restore, `112.35 s` protocol RTO, `121.88 s` total outage RTO, `90.301 s` pre-replay DB RPO, `0 s` / `0 B` post-replay service usage RPO, `SECOPS-18`, `SECOPS-19`, and `SECOPS-05` closed, `SECOPS-21` remains open after `#245`), `fvpn-usage-admin` ledger freeze/recovery/replay, Stripe + NetBird state reconciliation, and `D-076` / `D-077` / `D-081` (`#227`, `#238`, `#242`, `#244`) post-`complete-recovery` step to list fenced devices via `GET /api/v1/admin/devices?authorization_unconfirmed=true` (`00188_device_authorization_fences.sql`) and re-authorize explicitly via `POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize` (`409 DEVICE_LIMIT_REACHED` per `D-081`) where the member is entitled.
- [**Parameter Calibration Guide (`../handover/calibration.md`)**](../handover/calibration.md) — Business and engineering parameter matrix (`none`, `test only`, `proposed`), spec constraints, Fair-Use serving-exit recompilation and `fair_use.applying` convergence (`#222`), serving lease ceiling (`max_offline_lease_seconds` `60..86400`, `D-065`), `D-074` (`#224`) `node.publish` retry budget (`worker_attempts = 10`, `~4.3–8.5 minutes`), updater `probe_peer` bounds, whole-ruleset nftables foreign-chain verification (`#220`, `#241` `D-079` `loopback_scoped` rule in `127.0.0.0/8` and `::1`), per-class HTTP rate limiters (`D-060`, `browser_rate` `10/s` burst `30` confirmed in `#233`, `agent_rate` `5/s` burst `10`, and `D-078` `#229` on `429` with `Retry-After`), and decision ownership (`Ted` vs. `core team`).
- [**Operator Guide (`../handover/operator-guide.md`)**](../handover/operator-guide.md) — Daily/weekly checklists, reading operations (worker contention is not failure `#205`/`#208`), audit trails (transition-only discovery denial audits on `fleet_config`), and current OpenAPI `1.10.0` revision 14 contract status (`50` operations; all 50 are wired on `main` after `revokeMemberGrant` registration in #297, including `listAdminDevices` and `authorizeMemberDevice` in `#238`/`#244` (`D-077`, `D-081`), `MemberDevice.authorization_unconfirmed` always emitted on `GET /api/v1/member/devices` (`D-077`, `D-076`), `NodeDetail.usage_outbox` (`#243`, `00189_node_incarnation_usage_outbox.sql`, `SECOPS-19` closed, with `ETag` participation), `listNodeIncarnations` `#200`, `Enrollment.token_sha256` `#195`, `awaiting_member_grants` / `ErrorCode.AWAITING_MEMBER_GRANTS` `D-071` `#213`/`#216`/`#223`/`#238`/`#243`, `NodeDetail.identity_conflict` `D-072` `#211`/`#213`/`#217`/`#223`/`#235` with `ETag` participation, and the original 12 node enforcement verification reasons plus `TC_FOREIGN_FILTER` in `ErrorCode` (#261)), node lifecycle (`#239` `fvpn-node-install` `docs/node/install.md`, `D-055`, `D-057`, `D-058`, `D-061`, `D-067` provisional grant-less lease renewal and central `60 s` expiry backstop `#193`/`#198`/`#206`/`#216`, `D-070` revoke `nodes` step outcomes `conntrack_delete_pending`/`conntrack_delete_succeeded`/`node_failed_closed` `#206` with `#217` exit-group + `90 s` freshness fence, `D-075` `#228` one-shot stale-generation grace column `node_incarnations.stale_generation_grace_at` (`00187_identity_roaming_and_readiness_grace.sql`), `D-072`/`D-073` cloned-identity conflict runbook `#211`/`#213`/`#217`/`#223`/`#228` with durable roaming candidate in `node_identity_endpoints`, `D-074` `#224` gated `node.publish` waiting state, `#225`/`#232` pure-read `fvpn-agent usage snapshot -config /etc/vpn-node/agent-run.json [-json]` and `ledger_epoch` `null` until `ApplyCheckpoint`, `SECOPS-19` `#226`/`#243` `ObservationReport.usage` and `NodeDetail.usage_outbox`, `#220`/`#241` `D-079` whole-ruleset nftables read-back and `loopback_scoped` rule, 12-attempt `conntrack -D` fail-closed bound `#193`/`#202`, `empty_exception` vs. `populated_verified` publication attachment basis, `data_path_probe` reasons, and `ISOLATION_POLICY_INVALID` on pending `conntrack -D`), fleet offline lease ceiling (`GET`/`PUT /api/v1/admin/fleet-config`, `D-065`, `#176`, `#179`, `60..86400` s, `fleet_config.update` audit, and lease renewal timing), fleet update batches (`D-068` `#189`: `needs_attention` / `unconfirmed` / null-only pre-result seeding / current-job-only attribution / incarnation-wide accepted-result authority / `update_observation.unexpected_release`) and `D-053` / amended `D-064` health attestation with `probe_omitted` evidence, rollback recovery `blankHealth` / `probe_omission` pruning (`#192`), and separate candidate/restored health (`#177`, `#186`, `#212`), threat-feed and resolver-document troubleshooting (`D-054`, `D-056`), ledger freeze/adjustments (`#214`, `#221`, `#242`), member suspension, `D-076`/`D-077`/`D-081` (`#227`, `#238`, `#242`, `#244`) post-restore device authorization fences (`device_authorization_fences`, `00188_device_authorization_fences.sql`, `GET /api/v1/admin/devices?authorization_unconfirmed=true`, `POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize`), Fair-Use enforcement reach and `fair_use.applying` convergence (`#222`), identity continuity and duplicate pre-checks, per-class rate limiters (`#190`, `#215` `urn:fvpn:node:<uuid>` URI SAN mTLS key with `sha256(leaf.Raw)` fallback, full-table LRU tail walk, `/48` anonymous resident cap `min(max_rate_principals/2, 64)` (#252), zero-width `::` parse rule, single-walk body identity, and `D-078` `#229`/`#233` confirming `browser_rate` `10/s` burst `30`), `trusted_proxies`, 60-peer admission and counted-traffic figures (qualified as shared development host, not candidate hardware, `#178`, `#203`), request-body limits (`D-060`, `#169`, `#174`, `#178`, `#182`, `#190`, `#203`, `#215`, `SECOPS-15` closed), and key rotation.
- [**Member Guide (`../handover/member-guide.md`)**](../handover/member-guide.md) — Bilingual (`zh-TW` / `en`) guide covering candidate client versions (`needs operator run`), explicit exit selection (`netbird networks select <fvpn-exit-…>`), and automatic central device authorization (`D-052`) for Windows, macOS, iOS, and Android, manual verification scripts (`AT01`, `AT09`, `AT21`, `AT29`, including `#231` members follow-up 6 `AT02` `42/0` strict `fvmb0` ingress injection and `AT09` `36.976 s` / `41.695 s` provisional window, `#240` devstack `29/0/0`, and `#242` `AT30` passed live), device limits, self-service revocation (`AT02`, `AT04`), and `D-076`/`D-077` (`#227`, `#238`, `#244`) post-restore `pending` state with `authorization_unconfirmed: true` until operator re-authorization, usage (`IPv6Block` mode) & FUP display (`AT17`–`AT19`), DNS behaviour and DoT/DoQ vs. DoH limits, and support flow.
- [**Adversarial Review Archive (`../reviews/README.md`)**](../reviews/README.md) — Self-contained archive of Wave 3 and Wave 4 adversarial review reports (`r3-d055-review` through `r15-d072-d075-review`), invariants that held, findings with status and closing PRs against `main`, and proof tests.

## Domain Operational References

- **Control & Worker**: [`docs/control/README.md`](../control/README.md), [`docs/control/COMPOSITION.md`](../control/COMPOSITION.md), [`docs/control/OPERATIONS.md`](../control/OPERATIONS.md), [`docs/control/WORKER.md`](../control/WORKER.md)
- **Database & Invariants**: [`docs/db/README.md`](../db/README.md) (`F-01`–`F-15`)
- **PKI & Key Rotation**: [`docs/pki/README.md`](../pki/README.md), [`docs/pki/enrollment.md`](../pki/enrollment.md), [`docs/pki/rotation.md`](../pki/rotation.md)
- **Usage Ledger & Recovery**: [`docs/usage/README.md`](../usage/README.md), [`docs/usage/restore-runbook.md`](../usage/restore-runbook.md)
- **Fleet Updates & Node Updater**: [`docs/updates/release-process.md`](../updates/release-process.md), [`docs/updates/central-lifecycle.md`](../updates/central-lifecycle.md), [`docs/updates/node-updater.md`](../updates/node-updater.md), [`docs/updates/node-health-attestation.md`](../updates/node-health-attestation.md)
- **DNS Policy, Threat Feeds & Privacy**: [`docs/dns/PUBLISH.md`](../dns/PUBLISH.md), [`docs/dns/FEEDS.md`](../dns/FEEDS.md), [`docs/dns/BACKEND.md`](../dns/BACKEND.md), [`docs/dns/PRIVACY.md`](../dns/PRIVACY.md)

## Dated Acceptance Evidence

[Security/operations evidence](../e2e/at/secops/README.md) ([`summary.md`](../e2e/at/secops/summary.md), [`requests.md`](../e2e/at/secops/requests.md), [`AT30.md`](../e2e/at/secops/AT30.md), [`restore-reconcile.md`](../e2e/at/secops/restore-reconcile.md)) retains the follow-up 2 isolated physical PITR across ledger, grant and audit boundaries (`history/follow-up-6/pitr.json`). Follow-up 4 ([`limiter.md`](../e2e/at/secops/limiter.md), [`publication.md`](../e2e/at/secops/publication.md)), follow-up 5 (`#165`, [`history/follow-up-5/counted-flow.json`](../e2e/at/secops/history/follow-up-5/counted-flow.json), [`history/follow-up-5/post-revoke.json`](../e2e/at/secops/history/follow-up-5/post-revoke.json), [`history/follow-up-5/deadlines.json`](../e2e/at/secops/history/follow-up-5/deadlines.json)), follow-up 6 (`#178`), follow-up 7 (`#203`, run `ab96acac-8713-485a-9c29-88f20110e2e5`, [`capacity.json`](../e2e/at/secops/capacity.json), [`deadlines.json`](../e2e/at/secops/deadlines.json), [`limiter.json`](../e2e/at/secops/limiter.json), [`node-telemetry.json`](../e2e/at/secops/node-telemetry.json), [`gateway-inventory.json`](../e2e/at/secops/gateway-inventory.json), [`secret-scan.json`](../e2e/at/secops/secret-scan.json)), follow-up 8 (`#221`, run `0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8`), and follow-up 9 (`#242`, run `42396951-2d9a-49f4-95c3-45f33d9519d5`, [`AT30.md`](../e2e/at/secops/AT30.md), [`restore-reconcile.md`](../e2e/at/secops/restore-reconcile.md), [`restore-reconcile.json`](../e2e/at/secops/restore-reconcile.json)) prove per-class limiter boundaries, trusted-proxy principal isolation (`gateway-principals`), and post-`#169`/`#190` cross-class limiter isolation (`120` anonymous webhook requests -> `110 × 429` while admin and mTLS lease requests return `200`), the `8 MiB` `SubmitUsageBatch` body boundary (`8,388,607` / `8,388,608` / `8,388,609` B -> `200` / `200` / `413`, `SECOPS-15` closed), live zero-binding publication (`SECOPS-11`, with `#160` adding the `empty_exception` vs. `populated_verified` attachment basis gate), live counted serving flow (`SECOPS-09` closed), post-revoke coherence (`F3-02` passing), `AT29` deadline/watchdog fault cutover through the `D-065` fleet API (`GET`/`PUT /api/v1/admin/fleet-config` `428`/`412`/replay/audit and finite grace `FVPN_E2E_GRACE_SECONDS=90`), **`SECOPS-10` 60-peer native admission and counted traffic (`20×3` = `60` peers, `904.360 s` sequential and `300.386 s` / `296.017 s` concurrent `ScaleNativeMembers` wall, `60/60` counted peers, `5,960/5,960` HTTP `200` device reads with `0` wrong-member responses, and `82` node telemetry samples at `5 s` cadence — measured on the shared development host, not candidate hardware)**, `SECOPS-16` secret-scan verification (`0` leaks across ordinary streams, `60` root-owned `0600` pre-delete login files classified under `D-066`), and **live `AT30` passed (`#242` run `42396951-2d9a-49f4-95c3-45f33d9519d5`: `7.17 s` isolated `T0` restore, `6/6` checklist sections, `120.12 s` fail-safe negative, 5-step `fvpn-usage-admin` protocol replaying `3/3` retained-node events `78833` B + `1` post-`T1` event `10160` B with `0` lost / `0` double-counted / `0` gaps, `90.301 s` pre-replay DB RPO, `0 s` / `0 B` post-replay service usage RPO, `112.35 s` protocol RTO, `121.88 s` total outage RTO, `SECOPS-19` observation payload and local snapshot `3` unacked / `"2352"` B -> `0`, `SECOPS-18` `D-076` fence on `peerM2D2` `01a18f9b-a2a0-728f-8482-b08cc9b1a3ee` + `AuthorizeDevice` re-grant + 2 negatives, `SECOPS-05` closed, and `SECOPS-21` remains open after `#245`)**. **`AT30` is `passed`** (`1 pass / 4 blocked / 0 fail` across `AT28`–`AT32`: `AT29` recorded as blocked on `SECOPS-17` `past_due` payment truth (lab fix merged in `#240`; acceptance record remains dated), established TCP/UDP transports, and `needs operator run` guest clock rollback; `AT31` blocked on forced-relay control and candidate hardware; `AT32` blocked on bulk final per-stream attribution and physical origins/platforms). [Updates follow-up 4](../e2e/at/updates/follow-up-4.md) passes `AT23`, `AT24`, `AT25`, `AT26`, and `AT27` on `main`, [updates follow-up 5 (`#158`)](../e2e/at/updates/history/follow-up-5/summary.md) ([`pre-result.md`](../e2e/at/updates/pre-result.md), [`probe-issuance.md`](../e2e/at/updates/probe-issuance.md)) verifies `AT23`, `AT25`, and `AT26` plus pre-result observation seeding and probe-issuance revalidation, [updates follow-up 6 (`#177`) and follow-up 7 (`#186`)](../e2e/at/updates/summary.md) ([`requests.md`](../e2e/at/updates/requests.md)) pass `AT24`, `AT26` (`rolled_back` with revision 7 evidence), and `AT27` and resolve `F5-01` under `F6-02` and `D-068` (`#189`), and [updates follow-up 8 (`#212`, run `d50da75fd14fc72c`)](../e2e/at/updates/summary.md) on a fresh `#204` stack (`w4-updates` / `42000`, after devstack follow-up 13 `#204` exported `Stack.ProbePeer`, `Stack.ProbeWatchContainer`, and `Stack.WaitForProbeDatagram`, resolving `F5-02`) passes all five `AT23`, `AT24`, `AT25`, `AT26`, and `AT27` in one live run; physical `arm64` and published member traffic during updates remain unmeasured. Read these records with their source revision; later code merges alone do not change acceptance verdicts.

[Enrollment acceptance (`#173`, `#187`, `#210`, `#229`)](../e2e/at/enrollment/RUN.md) ([`summary.md`](../e2e/at/enrollment/summary.md), [`requests.md`](../e2e/at/enrollment/requests.md), [`AT06.md`](../e2e/at/enrollment/AT06.md), [`AT07.md`](../e2e/at/enrollment/AT07.md)) runs on stack `w4-enroll` / port base `12000` (`6000 + 6000*n`, `n = 0..9`), passed `95/95` subchecks in follow-up 1 (`#187`) and `105/105` (`46/46` in `AT06`, `59/59` in `AT07`, run `20261030T203711Z`) in follow-up 2 (`#210`, verifying `ENR-02` `Enrollment.token_sha256` and `ENR-03` `listNodeIncarnations`), and in follow-up 3 (`#229`, run `60a7102e-b824-4927-a576-62033f458157`) closed `ENR-08` on the production `nodes.Service` caller (`ObserveEndpointTx` + `NodeDetail.identity_conflict` ETag participation) and recorded `AT06` as `254 pass / 1 blocked / 0 fail` (the one blocked item is `rate_limit_review`: browser-class `429`s on `GET /api/v1/admin/audit-events` and `GET /api/v1/admin/nodes/{id}/incarnations` under `BrowserRate` `10/s` burst `30`, retried after `Retry-After` per `D-078`; in control follow-up 21 `#233`, [`docs/control/AUTH.md`](../control/AUTH.md#browser-class-calibration) and `TestFollowup21_BrowserClassCalibration` confirmed `browser_rate` stays `10/s`, burst `30`) and `AT07` as `15 pass / 7 blocked / 0 fail` (`w5-node-installer` `#239` subsequently delivered `fvpn-node-install`, `deploy/node/install.sh`, and `deploy/node/systemd/` in [`docs/node/install.md`](../node/install.md)). [Isolation acceptance follow-up 3 (`#207`, run `c359fa37-61e0-4469-9f73-5a1e60949c3b`, archived under `docs/e2e/at/isolation/history/follow-up-3/`) and follow-up 4 (`#237`, run `fa0074cc-3b1a-403d-b8c2-3a3439fe2b7e`)](../e2e/at/isolation/summary.md) ([`README.md`](../e2e/at/isolation/README.md), [`requests.md`](../e2e/at/isolation/requests.md), [`AT10.md`](../e2e/at/isolation/AT10.md), [`AT11.md`](../e2e/at/isolation/AT11.md), [`AT12.md`](../e2e/at/isolation/AT12.md), [`AT13.md`](../e2e/at/isolation/AT13.md)) recorded `AT10`–`AT13` on commit `205633f` (before `#240` and `#241` `D-079`) as `0 pass / 4 blocked / 0 fail` (`14/4/0` subchecks) because Docker's embedded DNS `ip nat` (`127.0.0.11`) rules triggered `FLOWTABLE_BYPASS` before `#241`'s `D-079` `loopback_scoped` rule landed, while all four kernel-enforcement mechanism controls (`lab-AT10`–`lab-AT13`) passed; follow-up 5 (`#246`) later measured the #240 lab fix; see the acceptance table below for its remaining case blockers. [Usage acceptance follow-up 2 (`#214`, run `2393df75-6261-42bc-99b4-df3c57c4b741`)](../e2e/at/usage/README.md) ([`requests.md`](../e2e/at/usage/requests.md)) passes `AT14`–`AT18` live on counted serving flow (`node_up = 420`, `node_down = 668`) while `AT19` remains **subset passed, case blocked** (the #214 record names Fair-Use recompilation and ACK-reader defects; #222 fixes the implementation, this historical record is superseded by follow-up 4 (#257) below). [Members acceptance follow-up 5 (`#201`) and follow-up 6 (`#231`, run `80db103b-622c-42b1-946d-4628daaa213d`)](../e2e/at/members/summary.md) ([`README.md`](../e2e/at/members/README.md), [`requests.md`](../e2e/at/members/requests.md)), [portal follow-up 10 (`#175`), follow-up 13 (`#208`), follow-up 14 (`#223`), and follow-up 15 (`#235`)](../../apps/portal/e2e/portal.spec.ts), [agent follow-up 14–19 (`#181`, `#193`, `#198`, `#202`, `#216`, `#226`) and usage-agent follow-up 1–2 (`#225`, `#232`)](../../node/agent/review_d063_test.go), [devstack follow-up 9 (`#153`)](../e2e/followup9-verification.md), [devstack follow-up 10 (`#170`)](../e2e/RUN.md), [devstack follow-up 11 (`#182`)](../e2e/followup11-verification.md), [devstack follow-up 12 (`#196`, `#199`)](../e2e/scenarios/device-revoke.md), [devstack follow-up 13 (`#204`, `#219`, run `4c97ebef`)](../e2e/RUN.md), and [devstack follow-up 14 (`#240`, acceptance run `4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d` `29 pass / 0 blocked / 0 fail`, exports run `0a687f3a-2351-4234-b43d-882109267967` `16 pass / 0 blocked / 0 fail`)](../e2e/RUN.md) ([`gateway-principals.md`](../e2e/scenarios/gateway-principals.md), [`device-revoke.md`](../e2e/scenarios/device-revoke.md), [`requests.md`](../e2e/requests.md)) verify `AT02` (`42/0` subchecks in `#231` including strict `fvmb0` ingress injection via `Stack.ObserveMemberIngress` where the revoked device raises `ctunku` `0 -> 1` with nothing forwarded while the admitted sibling moves `u_` `31 -> 32`), `AT03`, and `AT08` (`3 pass / 7 blocked / 0 fail`, `384` passing subchecks in `#231`), `D-067` same-generation cold-restart lease recovery (`44 s` in `#201` resolving `F4-01`, and in `#231` `AT09` `36.976 s` to serving-ready and `41.695 s` to first counted byte on unchanged generation `23`), counted Linux lab member forwarding, real device revocation (`PauseNativeControl` stale-client revocation with 3-point `tcpdump` capture confirming `peer_tx_packets > 0`, `mor0_rx_packets == 0`, `uplink_packets == 0`, plus `#219` strict post-revoke member-ingress proof on `fvmb0` via `Stack.InjectMemberIngress` / `Stack.ObserveMemberIngress` where a revoked device frame raises `ctunku` by `1` while an admitted sibling moves `u_<hex>` by `1`), 4th-device replacement, dedicated updater probe peer traversal (`update-good` with `probe_omission_count=0` and exported `Stack.ProbePeer`), a second real exit hook (`StartSecondExit` / `EnrollAt`), lab `trusted_proxies` (`/32` gateway trust) and nginx `X-Forwarded-For` (`gateway-principals`, `SECOPS-13`), the **TEST ONLY** finite-grace prepare option (`fvpn-e2e-prepare -grace-seconds N`, `SECOPS-04`), read-only node telemetry (`StartNodeTelemetry`, workload measurement only), and 60-peer native admission (`SECOPS-10`, `SECOPS-16`); the earlier `AT02`, `AT03`, `AT04`, and `AT08` records in [`docs/e2e/at/members/requests.md`](../e2e/at/members/requests.md) are historical pre-`#153` records (`DOC15`), and native four-platform checks (`AT01`, `AT09`, `AT21`) remain `needs operator run`.

### Follow-up 13 acceptance snapshot (2026-10-30 EDT)

Verified against `origin/main` at `d9e28f8`. UTC evidence dated 2026-10-31 below was captured on 2026-10-30 EDT. These are the owners' runs; no new live acceptance run is claimed here.

| Lane / evidence | Measured result and remaining limits |
|---|---|
| [Isolation follow-up 4 (`#237`)](../e2e/at/isolation/history/follow-up-4/summary.md) | AT10–AT13 blocked before admission by Docker embedded-DNS NAT / `FLOWTABLE_BYPASS`; mechanism controls passed. |
| [Isolation follow-up 5 (`#246`)](../e2e/at/isolation/summary.md), run `c0106e5d-d8e8-400b-abe0-181d6428fc55` | AT10 `7 pass / 1 blocked / 0 fail`; AT11 `2 pass / 1 blocked / 0 fail`; AT12 `9 pass / 1 blocked / 0 fail`; AT13 `6 pass / 1 blocked / 0 fail`: **subset passed, case blocked** for every full AT. Twelve convergence samples, six destination positives, forged-source and post-revoke ingress, finite-member expiry, and whole-node `60 s` expiry passed (ceiling restored to `86400`). Blockers: unobserved provisional-grant window, IPv6/platform matrix, designated DNS path, continuous transition coverage. This binary used #240's lab NAT deletion; it did not include D-079 (#241). |
| [Usage follow-up 2 (`#214`)](../e2e/at/usage/summary.md) | AT14–AT18 passed; AT19 remains the recorded blocked case. Historical follow-up 13 status; superseded by usage follow-up 4 (#257) below. |
| [Devstack follow-up 14 (`#240`)](../e2e/runs/2026-10-31/followup14/run.md) | Acceptance `29 pass / 0 blocked / 0 fail`, run `4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d`; exports `16 pass / 0 blocked / 0 fail`, run `0a687f3a-2351-4234-b43d-882109267967`. Lab hooks do not promote the blocked full AT cases. |
| [Security-ops follow-up 9 (`#242`)](../e2e/at/secops/AT30.md) | AT30 passed, run `42396951-2d9a-49f4-95c3-45f33d9519d5`: D-076 fence/two negatives/re-grant, SECOPS-18 and SECOPS-19 closed, replay `3/3`, RPO `0 B`, protocol RTO `112.35 s`, total RTO `121.88 s`. AT28/AT29/AT31/AT32 remain **subset passed, case blocked** on their recorded remaining work. |


### Follow-up 14 acceptance snapshot (2026-10-31 EDT)

Verified against main `c6e3eca`. These are committed owner runs on the shared development host, not new handover runs or candidate-hardware certification. D-098 uses final owner archives for isolation follow-up 6 and secops follow-up 10 (repointed in follow-up 15), with the remaining run-ID-named files under `tests/handover/testdata/`; the older isolation follow-up 13 claims still read `history/follow-up-5/`. No dated test reads a lane's current evidence file.

| Lane | Measured result and remaining limits |
|---|---|
| Usage follow-up 4 (#257), run `351d59aa`, product `e8a5e04` | AT14–AT19 all pass live on the fixed serving lab; synthetic ACK/traffic workarounds removed. D-067 bring-up/restart issuance `5021/5022 ms`, serving read-back `5848/5819 ms`, counted flow `6837/6135 ms`, no `LEASE_EXPIRED`. AT19 enforce transition `7.0 s` (`7032 ms`) and threshold transition `29.9 s` (`29864 ms`) to `applying=false` on two carrying exits; ENR-03 retained per-incarnation usage passed. The image did not contain agent-core follow-up 20, so these times do not measure that new schedule. This supersedes the earlier pre-#240-image limitation. |
| Members follow-up 7 (#268), run `297faa31-2f90-4aab-8cf0-6818629f46c8`, product `d9e28f8` | `3 pass / 7 externally blocked / 0 fail`, `471` passing subchecks. D-077/D-081 proved through production HTTP: fence `1→0` only on explicit `202` grant; `409 STATE_CONFLICT` and `409 DEVICE_LIMIT_REACHED` preserve it. AT09 `36.997 s` serving-ready, `41.729 s` first counted byte. Portal rendering of D-077 was not exercised by members. Native platforms, physical arm64 and external fixture gaps remain. |
| Devstack follow-up 15 (#266), acceptance run `2347f2a4-02aa-4f56-94f2-c3bde88b5a6d`, exports run `12d10041-9d33-4fbb-a4b0-9d2a21bce785` | Acceptance `32 pass / 1 blocked / 0 fail` (IPv6, IPv4-only native fixture). Exports `21 pass / 1 fail`: ENR-06 real installer-enrolled node fails first product start because `/var/run/netbird.sock` is outside the packaged units' write allowlist; [installer finding](../e2e/runs/2026-10-31/followup15/installer-finding.md), owner installer follow-up 3. |
| Security-ops follow-up 10 (#265), run `0e8ff9a4-4fa1-4154-ae35-847a5be4d985`, product `d9e28f8` | AT30 passed again: database RPO `95.507504 s`, service usage RPO `0 s`, protocol RTO `93.532554563 s`, total outage RTO `106.868537734 s`. SECOPS-16/17/20 closed, SECOPS-18/19 revalidated, SECOPS-05 retained; D-077 proved through HTTP. SECOPS-21 open (offer `11`, retained `14`); #270 D-095 is merged but D-099 HTTP-cache fix remains in flight. AT29/AT31 were deferred to devstack follow-up 15 exports. |
| Isolation follow-up 6 (#271), run `bf671f3a-c6ac-4824-9107-425c4c5a5f02` | AT10 provisional window: `19` samples at `250 ms`, ready → populated lease `4,767 ms` (beside usage follow-up 3’s `5,021 / 5,025 ms`). Captured during **assisted recovery from a latched `TC_FOREIGN_FILTER`**, not an unattended clean pass. Mechanism cases pass; product AT10/AT11/AT12 blocked, AT13 **fail** (near-deadline continuity; post-deadline drop passed). AT10 `7 pass / 1 blocked / 0 fail`; AT11 `2 pass / 1 blocked / 0 fail`; AT12 `5 pass / 5 blocked / 0 fail`; AT13 `5 pass / 1 blocked / 1 fail`. D-098 pins use the follow-up-6 history snapshots. |

Portal follow-up 16 (#255) independently implements the admin devices list, confirmed Re-authorize with neutral receipt and read-back, and node usage-outbox block. The restore runbooks now point to that portal flow as well as the API. Contract revision 13 (#261) supplies the optional freeze annotation, installer source enum and real `TC_FOREIGN_FILTER`; see [operator guide](../handover/operator-guide.md) for the current producer/consumer boundaries and [coming decisions](../handover/README.md#follow-up-14-historical-cutoff-2026-10-31-edt).

### Follow-up 15 acceptance snapshot (2026-10-31 EDT)

Verified on main `b79ee23`, through #290 (resumed from `5ce76e9` / #287).
These are owner measurements, not new
handover runs or hardware certification. Follow-up 14's cutoff was `c6e3eca`
(through #272); none of #273–#277 was a delivered claim there. D-098 hashes and
source commits are in `tests/handover/testdata/followup15-snapshots.json`; the
merged rows are in `tests/handover/testdata/followup15-main.json`.

| Owner delivery | Dated evidence and limits |
|---|---|
| Isolation follow-up 7 (#281), run `041e9e48-79a6-45dd-a19d-8b5a63290655`, product `c6e3eca` | **Assisted, not an unattended clean pass.** AT12 `10 pass / 0 blocked / 0 fail`; AT10 `7 pass / 2 blocked / 0 fail`; AT11 `2 pass / 1 blocked / 0 fail`; AT13 `5 pass / 1 blocked / 1 fail`. AT10 clean Apply / D-096 recovery was not reached; an explicit pre-quarantine packet is retained. AT13 missed the one-second baseline tick; zero successes after expiry does not erase the failure. Pin: run-ID-named AT10–AT13 and recovery-action snapshots under `tests/handover/testdata/`. This new claim set does not replace follow-up 5 or 6. |
| Isolation follow-up 6 archive (#281), run `bf671f3a-c6ac-4824-9107-425c4c5a5f02` | All **44 files byte-identical** to the final follow-up 6 tree; Followup14 claims now read [history/follow-up-6](../e2e/at/isolation/history/follow-up-6/summary.md), including the 19 samples / 4,767 ms window. Follow-up 5 claims stay on their existing archive. |
| Netlab follow-up 14 (#285), measured 2026-10-31T06:51Z, source `f004b78` | Real NetBird 0.82.0: 10 recoveries, median **54.4 ms**, worst **78.6 ms**, no route/address repair; 10/10 deliberate reconnects raise the link in **1.8–2.0 s**, zero redirected frames. Bounds include reads/readiness and a one-second harness hold; not production deadlines or uninterrupted containment. D-108 closes D-096 for the pinned native IPv4 NetBird-owned path only. The separate recovery latch after `TC_FOREIGN_FILTER` + newer generation goes to agent-core follow-up 24. [Measurement and limits](../poc/G2-followup14.md); pins: roadmap #285 and `TestD096_RealNetBird`, `TestISO_F6_RetainedFailureBlocksNewGeneration`. |
| Wire-nodes follow-up 29 (#277), policies follow-up 8 (#286) | D-099 HTTP fix under the ordinary one-step proposal interpretation; permanent `TestAT30_D095_HTTPRestoreCacheProbe` verifies the exact 503 hold then offer **15** above retained **14**. Pre-#277 negative control fails **304 / generation 11**. ACKs are simulated: **live SECOPS-21 remains open**. Keyed installer replay and enrollment `next` are fixed in #277. [HTTP note](../nodes/2026-10-31-secops-21-handover.md), [live closure checklist](../policies/2026-10-31-secops-21-handover.md). |
| Portal follow-up 17 (#275), wire-nodes follow-up 30 (#287) | D-101 closes Portal CI red since #259 with real mTLS `POST /api/v1/agent/lease-renewals`; fixtures never fabricate lease evidence. D-107 projects `member_forwarding_unconfirmed (lease_unconfirmed)` through revision 13 `Operation.steps[].message`. D-074 hold remains `running`; a polling timeout is not failure. [Diagnostics](../nodes/2026-10-31-publication-gate-cause.md). |
| Installer follow-up 3 (#279) | ENR-06 daemon socket ownership and shared runtime lifecycle fixed. Offline seed and firstboot verified with real daemons; NetBird `up` is a TEST ONLY socket adapter. Full journaled product upgrade stays a required **red** test: absent meter checkpoint, `CHECKPOINT_FAILED` / `final_sample`, null health, original release retained despite updater exit 0. **9 pass / 1 required fail**; agent-core follow-up 24 owns the checkpoint. [Dated owner note](../node/install-followup3-2026-10-31.md); pin `TestD089_RealSeedThenJournaledUpgrade`. |
| Updates follow-up 9 (#283), run `89fc6769-0a09-403d-ae88-84f736ee1dce`, product `5575470` | Regular **AT23–AT27 all pass** on a systemd lab node; D-080 fresh-node no-work and kernel boot-identity checks pass. Installer extension is **blocked**, live target exit 2: F9-01 socket (later fixed by #279), F9-02 umask-sensitive seed, F9-03 missing checkpoint / misleading exit 0, F9-04 null central baseline rejects installer failure, F9-05 downgrade accepted for staging. F9-04 routes to D-111; the others retain owner follow-ups. [Requests](../e2e/at/updates/requests.md). Run-ID-named AT23–AT27 / installer-handoff / requests snapshots pin the original findings. Physical arm64 and the full §19 population remain unmeasured. |
| Enrollment follow-up 5 (#282), run `4011dd32-73ef-4f49-b78b-6cf1ab8496c9` | Historical **AT06 344 pass / 0 blocked / 0 fail**; **AT07 19 pass / 6 blocked / 0 fail**. ENR-05 same-slot exports and ENR-06 installer startup block the required checks in this run. #279 fixes startup after this measured baseline; enrollment follow-up 6 verifies the fix below. Pin: [follow-up 5 archive](../e2e/at/enrollment/history/follow-up-5/summary.md), eight original files byte-identical; the older immutable run pin is retained. D-105 was pending at this baseline. |
| Enrollment follow-up 6 (#288), run `1af27741-cbd9-4fdb-aa1c-7d02790e7a46`, product `945eff8` | **AT06 376 pass / 0 blocked / 0 fail** (344 + 32 archive/attempt scan rows); **AT07 26 pass / 5 blocked / 0 fail**. ENR-06 installer startup verified fixed on #279: install, idempotent rerun and controlled reinstall exit 0; same Node, fresh incarnation, old certificate denied on both routes. Five AT07 rows remain blocked on dev-stack installer-node exports and ENR-05: usage hold/release, ledger snapshot, member transport and post-wrapper service restoration. No full journaled-upgrade pass is claimed. Pin: [immutable run](../e2e/at/enrollment/history/1af27741-cbd9-4fdb-aa1c-7d02790e7a46/summary.md). |
| Contracts revision 14 (#289) | API **1.10.0 / 50 operations**. `revokeMemberGrant` specifies admin session + CSRF + idempotency key and `202 Operation`, kind/audit `member.grant_revoke`; generated stub remains `501 NOT_IMPLEMENTED` until wire-members follow-up 11 registers it. D-105 optional non-null `UsageBatchRequest.origin_incarnation_id` preserves immutable retired events and ACK per original stream. D-106 optional nullable `ObservationReport.tc_evidence` carries bounded read-back evidence. Vectors: `member-grant-revoke.json`, `usage-retired-handoff.json`, `tc-readback-evidence.json`. Consumers and live acceptance remain pending; D-110/D-111 fleet-update behavior remains coming. |
| Netlab follow-up 15 (#290) | D-104 monotonic member deadline mechanism merged: **1 ms quantum**, **100 ms guarded-install bound**, nft text read-back. Enforce-level trace: last positive **D−100.7 ms**, first drop **D−0.8 ms**, no later positive. **Assembled AT13 stays open** for agent-core follow-ups 24/25 and isolation follow-up 8. Other kernels, physical arm64 and suspend remain unmeasured. [Dated mechanism and trace](../poc/G2-deadline-contract-2026-10-31.md); pins: roadmap #290, `TestD104_AT13_MillisecondDeadline`, `TestD104_AllowedPrecisionRead`, `TestD104_AT13_InstallOverrunNeverOpensGuard`, `TestD104_AT13_DeadlineContinuity`. |
| Db follow-up 2 (#280), fleet-update follow-up 10 (#273) | Invariant 16 is now a `db-lint` rule: migrations above 00188 need a working Down, with Up → DownTo(188) → Up proof. D-089 central acceptance / D-097 projection ship with migration 00191 and Down; a current central job wins over the installer summary. D-110 / D-111 remain coming. Pins: `Test15_5_MigrationDownLint`, `Test15_5_MigrationsRoundTrip`, installer acceptance/projection tests. |
| Security-ops follow-up 11 (#274) | **104 files byte-identical** in [follow-up 10 archive](../e2e/at/secops/history/follow-up-10/restore-reconcile.json); the older AT30 claim now reads that archive. Live **AT29 / AT31 / AT30 not run**. D-103 uses the harness's existing `fvpn-w4-secops-*` namespace; follow-up 12 owns the live runs and SECOPS-21 closure. No new RPO/RTO claim. |
| Agent-core follow-up 23 (#276) | D-100 and the real-emission `identity_frozen` consumer pass; **ISO-F6-03 / AT13 remains failed**. Whole-second truncation reproduced in agent and renderer: diagnostic lab exit 0 is not acceptance. D-104 now has netlab 15 merged (#290); agent-core 24/25 → isolation follow-up 8 re-assertion remain. [Dated note](../node/followup23-2026-10-31.md); pins `TestD100_TopologyRefreshRequiresCleanNewerPublishedApply`, `TestD088_RealHTTPWireFreezeConsumer`. |
| Devstack follow-up 16 (#284), run `a81785fa-77eb-428a-b846-fb33196a2dcf`, source `ec84d71` | **16/16 acceptance fixtures pass**: finite-grant transition, native IPv4/IPv6/dual and MTU 1500/1280 selectors, two-resolver native DNS with RR/cache/rebinding probes, retained pre-enrollment third node. QUIC and non-Linux selectors explicitly report unavailable; no platform pass is inferred. ENR-06 was not rerun. Pin: [immutable run directory](../e2e/runs/2026-10-31/followup16/a81785fa-77eb-428a-b846-fb33196a2dcf/finite-grant-downgrade.json), all 16 JSON records hashed. D-102 answers are in `docs/e2e/RESPONSES.md`. |

The [decision register and coming table](../handover/README.md#follow-up-17-decisions-register-2026-10-31-edt)
now distinguish the #303 merged consumers from pending activation and live evidence;
the table above retains the historical #290 status. [r16 dispositions](../reviews/README.md#r16-dispositions-at-the-follow-up-17-cutoff)
retain the four routed findings and distinguish the two known #277 fixes.

### Follow-up 16 acceptance snapshot (2026-10-31 EDT)

Verified on main `982c209`, through #303. #292 is handover follow-up 15's own
#290 reconciliation, not another live run. D-098 archives preserve prior
measurements; D-109 claims pin owned wording, exact roadmap rows and owner Go
test names. These are shared-development-host results, not candidate hardware.

| Delivery | Verified outcome and remaining boundary |
|---|---|
| Wire-nodes 31 (#291); fleet-update 11 (#300) | r16 H-1 fixed: fenced devices excluded from signed bindings and leases, worker recompiles on fence; M-2/L-1 fixed: rollback restores versions, deterministic D-110 job ID, D-111 eligible null-baseline seed, migration 00191 round trip. M-1 stays pending agent-core 24. Main's updater still uses the legacy whole-record ID pending updater 15. |
| Portal 18 (#293) | Publication-gate cause shown; D-093 has 15 entries, D093-06 / D093-10 retired, 13 open. D093-12 SQL release verification remains open pending a fixture-transport decision. |
| Devstack 17 (#294) | Run `a7c41436-82b3-4665-b75c-e2887f2e1009`: **8 pass / 1 blocked / 3 fail**. Installer owner-chain proof identifies the fifth chain as `HOOK_PRIORITY_MISMATCH`; full upgrade lacks a genuine checkpoint. These measured failures are not rewritten after #302. |
| Netlab 16 (#295) | Clean newer Apply recovers after foreign-object removal in the verifier; the ISO-F6-01 latch is in `node/agent`. D-106 helper is merged; agent-core 24/25 recovery/wiring and assembled isolation 8 remain pending. AT13 stays failed. |
| Usage-central 4 (#296); usage-agent 3 (#299) | D-105 central acceptance uses current mTLS plus same-node retired origin, one stream per batch and original-stream ACK. D-112 central bound defaults to 10 s; #299 pins meter interval ≤ bound. D-115 delegates identity only for schema-valid explicit-origin `submitUsageBatch`. Node retention/upload is merged inactive until agent-core 25's D-118 hook; usage AT 5 / live ENR-05 remain pending. |
| Wire-members 11 (#297) | `revokeMemberGrant` registered; durable 202 remains asynchronous. D-117: every grant is finite, including far-future grants. `member.grant_revoked` joins #291's fence batch. |
| Enrollment 7 (#298) | Run `59ef4bf8-7d4a-490f-a74d-6b54394375e0`: **AT06 424 pass / 0 blocked / 0 fail**, **AT07 36 pass / 5 blocked / 0 fail**. Five rows blocked on D-116 / D-105 at the measured base: three installer-serving rows and two historical-usage rows. Serving-source freeze withdrew one lease member and six rules within **227 ms**, then survived a process restart with `identity_frozen` and zero rules/lease members. Hold/release had zero un-ACKed bytes and proves no counted tail. Follow-up 6 is archived; later merges require a rerun. |
| Security-ops 12 (#301) | Restore run `dd845a7f-2660-431c-a8e6-c513108cc07d`: **AT30 passed**, pre-replay DB RPO **106.45 s**, post-replay service usage RPO **0 s**, protocol RTO **66.90 s**, total outage RTO **74.53 s**, replay **5 / 5 / 0 / 0 / 0** (events/hashes/lost/duplicate/gaps). D-095 exact, desired 10 / stored count 9 unchanged while stopped. **SECOPS-21 stays open**: probe/stored 16 > retained 15, but first-offer delivery / local acceptance / ACK unproven; capture → devstack 18 / wire-nodes 32. Transport run `cf81f6f8-4ccf-43d2-aae2-10121f6b169d`: AT29 finite-grant, lease/SIGKILL and revoke sub-cases pass; AT31 bounded relay pass (6.41 s to relay, 32.22 s to restored direct). Full AT29 / AT31 blocked on guest clock, manual withdrawal export, candidate hardware and 60-peer mixed relay. Follow-up 11 archived. |
| Installer 4 (#302) | D-116 implemented: hookless `inet fvpn_install`, legacy marker migrated on rerun/upgrade, exactly four rendered chains in `inet fvpn`; F9-01 confirmed. Privileged **10 pass / 1 fail**: full journaled upgrade remains red on missing checkpoint. Baseline proof is not serving; devstack 18 rerun and netlab 17 scope tests pending. |
| Members 8 (#303) | Run `69f92f49-73dc-4779-8101-dfb6a802a96b` on #284 fixtures: **3 pass / 6 blocked / 1 fail**, subchecks **558 pass / 5 blocked / 4 fail**. AT02 / AT03 / AT08 pass with `ReplaceFiniteGrant`, real expiry **3/0 → 0/3 → 3/0** and counted recovery. AT01 / AT04 / AT05 / AT20–AT22 blocked only on platform/operator/selector rows; every executed check there passes. **AT09 fails**: same-generation namespace restart never recovers serving (F8-01 → agent-core 25); dual-stack MTU-1280 transfer incomplete, 11,011 of 64,064 bytes in about 12 s (F8-02 → devstack 18 / netlab 18). D-119 resolves the finite-expiry question: deadline means expiry, a boundary plan change needs a billing-issued replacement grant. Follow-up 7 uses its final 31-file archive; local run-ID pins retired. |

Pending means no merge or live pass is claimed: agent-core follow-up 24, updater
follow-up 15 and netlab follow-up 17. The [decision register](../handover/README.md#follow-up-17-decisions-register-2026-10-31-edt)
records D-112–D-119 and the [review index](../reviews/README.md#r16-dispositions-at-the-follow-up-17-cutoff)
records the current r16 dispositions.

## Follow-up 17 acceptance snapshot (2026-10-31 EDT)

Main `7e044c71`, #304–#313; [operator evidence and source links](../handover/follow-up-17.md).
No new live acceptance was run by handover. Pins are in `TestHandover_Followup17Claims`.

| Merged delivery | Verified scope and remaining limit |
|---|---|
| Netlab 17 (#304) | D-116 hookless marker scope and real-agent serving proved; hooked drop still rejected, fifth owned chain still HOOK_PRIORITY_MISMATCH. Three UDP losses across two reload runs; **AT10 failed under the pre-D-120 rule**. |
| Agent-core 24 (#305) | Deny-all recovery candidate breaks ISO-F6-01; 897 ms containment / 494 ms clean link / 3.161 s counted serving, generations 3. Revision 13 details, durable checkpoint, candidate identity, D-087 and r16 M-1 delivered. Original installer health/rollback 404 failure retained; D-104 continuity deferred at that cut. |
| Handover 16 (#306) | Historical #303 cutoff with D-112–D-119; #304 reconciled here. |
| Updater 15 (#307) | D-110 catalog-bound receipts; D-114 terminal mismatch/replacement; F9-02 modes and F9-03 exit codes; real PostgreSQL interoperability. Component / isolated-database proof, no new live updates acceptance. |
| Netlab 18 (#308) | **AT10 PASS under D-120** in both runs: one loss among 48 datagrams in the lane, none focused. IPv4 MTU-1280 and bidirectional PMTU-error traversal proved. D-123 preserves IPv6Block; native F8-02 remains unattributed. |
| DNS-publish 8 (#309) | A/AAAA/HTTPS NXDOMAIN; CNAME/SVCB/TXT/MX/NS NODATA; SOA TTL honours 60/5 settings. 432-exchange conformance probe. Fail-closed for answers, not uniform NXDOMAIN. |
| Updater 16 (#310) | Bounded FileMeter wait and restart safeguards; current uncorrelated evidence refused. Fresh correlated success not yet proved → updater 17 / agent-core 26. |
| Installer 5 (#311) | Local privileged suite **13/13** with real checkpoint, signed candidate health and TCP probe; lower sequence refused before staging. Harness-proved local upgrade, not serving-node handshake or physical arm64. |
| Agent-core 25 (#312) | D-104 absolute deadlines and D-106 authoritative evidence consumed; D-118 active; same-generation kernel recovery 1.419 s. TEST ONLY guard-inclusive 74,796/76,000 (98.4158%) is not production availability. AT13 / AT09 / retired-stream acceptance not promoted. |
| Devstack 18 (#313) | ENR-05 real installer rerun **harness-proved**; final first-offer supplement **2/0/0 passed**, delivery 4 > retained 3, exact local digest and 202 receipt. **SECOPS-21 full restore remains open** → secops 13. |
| Devstack 18 retained failures (#313) | Serving `apply` exits 0 but seals **final_sample / failed / rolled_back**, no checkpoint or switch → updater 17 / agent-core 26 `barrier_id`. Post-upgrade seed **HASH_MISMATCH** → installer 6 / updater 17. Native MTU 1280 fails in both underlay families across attempts → netlab 19 / wire-nodes 32. All three remain **FAIL**, not yet proved fixed. Quiescent candidate health does not close the serving failure. |

AT09 F8-01 / F8-02, the five AT07 rows (enrollment 8), D-104 assembled
continuity / AT13, retired-stream usage 5, AT31 hardware and AT29 guest-clock
rows, real platform clients and physical arm64 remain open. The generation-3
native restart passes in 5.96 s; generation-30/full-history recovery remains
not yet proved. D-120 is a test-acceptance definition, not a product guarantee;
D-104's 100 ms installation bound is unchanged. D-123 excludes IPv6 overlay
forwarding and end-to-end IPv6 MTU / PMTU acceptance, without excusing native
underlay failures. Process-only ownership decisions are kept in the
[decision register](../handover/README.md#follow-up-17-decisions-register-2026-10-31-edt).

## Follow-up 18 acceptance snapshot (2026-10-31 EDT)

Verified on main `6d0022d8` through #324; [claims and sources](../handover/follow-up-18.md).
Handover ran no new live acceptance. Earlier tables retain their original cutoff.

| Merge | Current disposition |
|---|---|
| Security-ops 13 (#316) | SECOPS-21 closed on the assembled harness: delivered 19 > 15, matching digest and applied 202. AT30 passed, RPO 92.163 s (−14.287) / RTO 81.016 s (+6.486); AT29 / AT31 resources remain blocked. |
| Members 9 (#317) | F8-01 closed in generation 30; AT09 fails on F9-01 pending agent-core 27 / members 10. F8-02 remains unattributed. |
| Netlab 19 (#318) | MSS unchanged at observed hops; kernel-only pass with zero tc drops neither reproduces nor clears native F8-02. TimingEvidence() enables isolation 9 correlation; 17 historical losses stay open. |
| Handover 17 (#319) | Through #313 plus #316 manifest reconciliation; eleven subtests, 242 digests / 141 paths. |
| Agent-core 26 (#320) | D-125 exact latch UUID; D-118 node-lock adoption and lifetime usage.db.lock. Chain resume changed the cancelled-ACK test only. |
| Wire-nodes 32 (#321) | D-126 mixed report stored/projected value-unchanged, publication blocked, stored ACK replay with 202 Accepted. D-128 is JCS/hash equality, not literal JSONB whitespace retention. D093-12 option (ii) is advice only. |
| Fleet-update 12 (#322) | D-127 first receipt plus one eligible correction; stored replay, unchanged generation fence. Migration 00192 before control. PostgreSQL harness-proved; live captures not yet run. |
| Updater 17 (#323) | D-125 consumed; component proof, live serving-node rerun not yet proved. Pre-#307 extractor reproduces seed HASH_MISMATCH; installer 6 / devstack 19 still consume the fix. |
| Usage AT 5 (#324) | AT14–AT20 pass in live run 170ea3e3; controlled-installer retired-stream hand-off, immutable tail and same-key replay. Follow-up 4 archived; full enrollment case remains separate. |

Still open: AT09 F9-01; F8-02 attribution; the three devstack 18 historical
live failures (pre-#307 serving handshake, post-upgrade seed HASH_MISMATCH,
native MTU 1280 in both families); five AT07 rows; isolation's 17 renewal losses;
AT29 / AT31 resources; real clients; physical arm64 and power-loss qualification.

## Follow-up 19 acceptance snapshot (2026-10-31 EDT)

Verified against main `57a8cc9f` through #337. [Evidence and immutable pins](../handover/follow-up-19.md).
Counts are pass / blocked / fail. No new live run was performed by handover.

| Delivery | Verified disposition |
|---|---|
| Agent-core 27 (#325) | D-126 deliberate single composition, no reporter race; D-087 producer already emits qualifying evidence; TimingEvidence after Apply/live/audit. |
| Installer 6 (#326) | ENR-05 consumed; 13/13 privileged tests, correlated FileMeter and post-switch seed without tree repair. |
| Handover 18 (#327) | Through #324, r17 exact archive, D-124–D-128; 12 subtests / 26 snapshots retained. |
| Members 10 (#328) | F9-01 CLOSED; AT09 38/5/0 at 29 → 29, D-129 removes exact-30 requirement. F10-02 AT02 and F10-03 AT20 remain failed; native 4/4 does not attribute F8-02. |
| Enrollment 8 (#329) | Historical AT06 501/0/1, AT07 43/1/2; ENR-07 precision defect. Minting attribution corrected to planBaseline in #333. |
| Usage-agent 5 / usage-central 5 (#330 / #331) | D-130 whole-microsecond minting and stored projection; migration 00193, canonical strict order, automatic sealed-event retry. |
| Devstack 19 (#332) | Serving 14/0/0 closes prior serving/seed failures; stopped-agent 13/0/1 retained. Native 7/0/0 twice, DNS 15/0/1 with 12,012-byte MTU-1280 failure. |
| Enrollment 9 (#333) | ENR-07 CLOSED; AT06 662/0/0, AT07 48/0/0, fresh ACK 1 → 2 and 908 bytes; retired 2,724 / ACK 3 preserved. |
| Isolation 9 (#334) | AT12 10/0/0, AT10 8/1/0, AT11 2/1/0, AT13 5/1/1 FAIL. ISO-F6-03 OPEN: 96 early drops, 95 outside guards. Agent-core 28 not on main at cutoff. |
| Wire-nodes 33 (#335) | D-131 lease-only TEST ONLY factory; no production control surface. Devstack 20 → isolation 10 consumption still pending. |
| Netlab 20 (#336) | Members/isolation archive pin correction; 200/200 pins, 20 regression cases; no runtime acceptance promotion. |
| Updates 10 (#337) | Installer extension 23/23; regular AT23 blocked on binary layout, AT24–AT27 blocked. Empty result_url remains F10-02. D-132 pins two fleet-update tests to immutable history. |

Current open findings and owners are in the [findings register](../handover/README.md#findings-register-at-the-follow-up-19-cutoff).
ENR-07, F9-01 and the devstack 18 serving/seed failures are no longer open.
ISO-F6-03 is a real product defect; physical arm64, OS reboot, power-loss,
real platform routes/clients, full AT13 faults and AT29/AT31 resources remain open.
