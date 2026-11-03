# Follow-up 28 — r19 fixes, native delivery delay and node diagnostics

Fixed `main` cutoff **aae6763f9dd662091835f9219205b882e4382bfe**, through
**#390 (updates 14)**, checked on 2026-11-01 EST. This cut is **#386–#390**.
Nothing was merged after the cutoff at task start. **Devstack 27 and isolation
15 were running**; their results do not enter this cut. Handover ran no new
live acceptance.

**D-098 / D-109 / D-132:** [five exact roadmap rows](../../tests/handover/testdata/followup28-main.json)
and [16 immutable snapshots](../../tests/handover/testdata/followup28-snapshots.json)
pin `TestHandover_Followup28Claims`: **19 subtests / 1,644 archive and run file
checks**. Snapshots retain source commit, full-source digest, byte offset,
length and excerpt digest. The inventory checks cover **987 devstack run
files, 609 netlab evidence files and 48 updates archive files**. No claims
read updates 14's living root evidence. Owned-document secops digests check
integrity, not acceptance.

## Previous cut — handover 27, #387

[Follow-up 27](follow-up-27.md) covered **#383–#385 at 37db5405**.
`TestHandover_Followup27Claims` retains **19 subtests / 3 exact roadmap rows /
14 immutable snapshots / 182 owner archive file checks**. Its register and
entrypoints are frozen under `tests/handover/testdata/history/follow-up-28/`.
Every earlier claims test stays true; only evolving owned inputs are redirected
or the exact new diagnostic note removed before checking the old guide body.

The live **#381 row is byte-identical** to
`tests/handover/testdata/history/follow-up-27/roadmap381.md`: SHA-256
**400d5cf91506f8b5fba942afa833924d839a16a9aabb8dddd9a5a80250eccbbc**.
Rechecked against this main cutoff; **no further action**.

## Devstack 26 — #386

Devstack 25's closed node was its own **TEST ONLY node tracer**. The unscoped
`meta nftrace set 1` in a foreign `mangle` table fails the unchanged verifier:
**ISOLATION_POLICY_INVALID**, **deny baseline**, **kernel generation 0 while
central applied generation stays 5**. Removal permits recovery at the same
generation. This is a **harness artifact, not a product defect**; it does not
retroactively determine every historical timeout's exact cause or instant.

The corrected **member-only observation is 4 / 0**. Passive **F14-style
re-grants are 17 / 1**. In **af718646**, generation 7 stayed serving while
handshake 0 persisted through the deadline; first member packets reached the
node **5.671 s** late, behind a **loopback-source handshake response**. These
small populations neither establish reliability rates nor close historical
member findings.

**F11-01:** the closed-node confounder is removed. The remaining issue is a
**native handshake / delivery delay of about five seconds** behind the
loopback-source response, consistent with devstack 24's **`::1` mechanism**.
The repair direction and any upstream report remain the owner's, now unblocked
by #386. Members 15's serving-node signature remains a separate, unproven cause.

Also delivered: counterread classifies duplicate owned chains as
**`duplicate_chain`** (`errors.As`, `ambiguous_listing`); the harness stop path
matches the shipped **15 s budget**, with **PID1 TERM forwarding** and 25 s
compose grace. A container's exit is not the agent's exit: the parent wait
receipt records the agent outcome. The historical stop gaps remain recorded.

[Dated analysis](../e2e/runs/2026-11-01/followup26/analysis.md), its
[run directory](../e2e/runs/2026-11-01/followup26/) and
[capture manifest](../e2e/followup26-captures.sha256) are D-098 evidence.
Proof names on main include `TestAT01_HistoricalNodeTraceRejectedByServingVerifier`,
`TestAT04_ParseFailureEvidence` and `TestAT04_WrappedDuplicateClassification`.

## Agent-core 34 — #388

**r19 M-2 fixed; production-reachable.** A due renewal signs lease **h** while
a busy Apply lock keeps **g** installed and serving; the ACK of g carried h's
live refs. The agent now keeps a **minimal applied snapshot committed atomically
with the ACK**, and reports refs **only when they match the applied generation**.
Restart retains that provenance; missing or mismatched legacy refs stay null.
`TestF34_ReportBeforeNewApplyOnBusyLock`,
`TestF34_AppliedSnapshotSurvivesNewDesiredAndRestart` and
`TestF34_MissingOrMismatchedSnapshotDoesNotBorrowNewDesired` cover those boundaries.

**r19 L-2 fixed.** A conntrack delete batch's count and identities are **reserved
in one transaction before Apply**. A crash or pre-delete failure consumes the
reservation; pending accounting survives generation changes, with the existing
12-attempt bound. This adds **about 1.1 ms (p99 1.59 ms) before the restricting
install on Steps with queued deletes**. Accepted under **D-138**, with the cost
explicit: A,B,A,B plus live repeat **12 / 12 pass, 6 qualified legs**, no B-only
failure and no attributable regression. This cost must remain in assembled
revocation/recovery timelines. The dated [hand-off](../node/followup34-applied-refs-conntrack-reservation-2026-11-01.md)
and [acceptance excerpt](../../tests/handover/testdata/history/follow-up-28/agent34-acceptance.md)
retain its limits. Pins use #388's row / PR and Go test names under D-109,
including `TestF34_ReservationFailurePreventsApply`,
`TestF34_GenerationChangeKeepsPendingReservationCount` and the converted
`TestReviewR19_Q4_ConntrackAttemptIsCountedAfterTheDelete`.

## Netlab 24 — #389

**r19 H-1 fixed.** A download packet whose source is a current **denied, host
or resolver address** drops **before any generic established / related accept**.
Exact designated-resolver **TCP / UDP source-port-53 replies stay ahead of the
guard**. The verifier requires the exact guards on **member, serving-probe and
baseline-probe paths**. The content-free **`diagnostic_v1` counter `ctdenyd`**
counts those return drops, **never usage**; upload diagnostic meanings stay intact.

The [per-follow-up evidence tree](../poc/G2-followup24-evidence/) is D-098 pinnable,
like follow-ups 20 and 21. The [dated hand-off](../poc/G2-followup24-2026-11-01.md)
and [acceptance excerpt](../../tests/handover/testdata/history/follow-up-28/netlab24-acceptance.md)
record **A,B,A,B accepted under D-138; no B-only failure**. **A2's D-120 failure
on unchanged main stays failed**: a lost final datagram has no recorded successor.
It and the earlier branch full-suite failure are a **measurement-end gap**,
routed to **netlab 25**; **both failed runs stay failed**, clause (iii) unchanged.
The single comparison window supplies descriptive timings, not a causal shift.

Main's proof names: `TestReviewR19_Q4_EstablishedDownloadSurvivesDestinationDeny`,
`TestReviewR19_Q4_CurrentRenderDoesNotRestrictEstablishedDownload`,
`TestAT13_ProbeDownloadDestinationDeny`, `TestAT13_DownloadSourceVerification`
and `TestMTR05_DownloadDenialIsDiagnostic`.

**Open owner question — RELATED ICMP, not a defect or a decision.**
`SpecDeniedV4` in `node/enforce/prefixes.go` includes **10/8, 172.16/12,
192.168/16 and 100.64/10**. A RELATED ICMP error, such as **fragmentation needed
or time exceeded**, with an outer source in that list is now dropped even when
its quoted connection is allowed. **No exception exists.** Existing F8-02
PMTU evidence uses an allowed origin source and does not settle this case.
The fail-closed source guard remains; an exception requires an owner decision
and evidence. A member's traceroute consequently shows no reply for on-path
hops whose addresses are in that deny list. **No fix is promised.** See the
[merged behaviour note](../../tests/handover/testdata/history/follow-up-28/netlab24-icmp.md).

## Updates 14 — #390, cutoff

The D-098 archive of **759d244a**, **48 root files**, is under
`docs/e2e/at/updates/history/follow-up-13/`. Devstack 25's **parent wait receipt**
and devstack 26's **15 s stop path** are adopted in
`tests/e2e/at/updates/stopobserve/`. Main's
`TestUPD05_StopObservationDistinguishesExitKillAndReap` and
`TestUPD05_ParentReceiptSurvivesReapRepeatAndRestart` cover outcome distinctions
and receipt reuse; `TestD098_Followup13ArchivePins` covers history.

**Run 2704600f on #387's tree**, cited by **roadmap row / PR only (D-132)**:

- **AT23–AT27 PASS 5 / 5; installer 27 / 27; 13 result POSTs**.
- **AT26 physical rollback** and the **uncached-fallback probe** are recorded.
- **All 11 stop commands returned 0**. **Six agent terminations** each have
  their own parent receipt, **exit 0**, including **AT23's first as a new sample**.
  The **five repeated stops reuse the prior receipt**.
- **No SIGKILL attempt**; the **stored lease was present at every stop and start**.

Run 2704600f's root files are **living until updates 15 archives them**; no
handover snapshot or claim pin reads those roots. **Updates F13-01 remains a
historical gap for run 759d244a; it was not filled**. It is **unrelated to members
F13-01**. A new complete sample cannot reconstruct the old first exit.

## Decisions and findings at #390

**No new decision since D-141. D-135, D-136, D-137, D-138, D-140 and D-141 stay
provisional.** D-141 keeps option (c) in production; r19's **design-only findings
stay recorded under D-141**, with no implementation or closure claimed.

The [findings register](README.md#findings-register-at-the-follow-up-28-cutoff)
marks **r19 H-1 fixed (#389); M-2 and L-2 fixed (#388)**, explains devstack 25's
closed-node timeouts as harness interference, and updates F11-01 to the native
delay above. **Members 15's serving-node signature → devstack 27**, cause
unproven. Netlab's D-120 missing-successor gap → netlab 25. Every other open
finding of follow-up 27 is unchanged. **Every historical failure stays failed.**

The [operator guide §3.4](operator-guide.md#34-node-egress-probe--sanitized-check-details-d-058a-d-058b-171-nodeagentprobego-nodeagentprobeexcgo)
adds the node-trace warning and known traceroute behaviour. Diagnose from the
member side, or capture packets without touching the node's ruleset.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation**, including AT02's
sibling outage given D-141; every loss counts, with no finite outage ceiling.

2026-11-01 — **Owner decision 2: D-137 hardening option**, retaining the failed
receive-only dual MTU-1280 limitation without upgrade or waiver.

2026-11-01 — **Owner decision 3: D-138 overturn option**.

2026-11-01 — **Owner decision 4: D-140 overturn option**, boundary schedule only;
unchanged predicate and historical failure.

2026-11-01 — **Owner decision 5: D-141 overturn option**, with its witness,
qualifying-share, promotion-cost and design-finding prerequisites.

2026-11-01 — **Owner decision 6: F11-01 repair direction and upstream report**,
now unblocked by #386; the remaining native delay does not establish the cause
of members 15's serving-node samples.

2026-11-01 — **Owner question 7: RELATED ICMP from spec-denied outer sources**
under §8.1 / §8.2 / §19 AT13. Keep the fail-closed source guard pending a
specified exception and evidence; no defect, decision or fix is asserted.

Already assigned, not new requests (2026-11-01): serving-node samples → devstack
27; isolation F14-01 → isolation 15 under D-140; D-120 measurement-end gap →
netlab 25. New `ctdenyd` consumer coverage → usage 8, devstack 28 and agent-core
35, per [#389 routing](../../tests/handover/testdata/history/follow-up-28/netlab24-routing.md).
Agent-core 34's pre-install cost → isolation / members timelines. Populated
stale-hold, alarm, escalation and active-stop retention remain with their
existing owners. These outside-owned items stop with their owners; no
outside-owned pin needed changing. **Seven owner requests; no production
value, evidence result or acceptance predicate changed.**
