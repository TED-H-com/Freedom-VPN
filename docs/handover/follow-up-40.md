# Follow-up 40 — #430–#434; D-146/D-147 merged and rerun

Verified against `main` **3f7dbc254ba1ce001628af8151a4d81c6954412d**, the
final roadmap commit through **#434**, on **2026-11-03 EST**. The range starts
immediately after #429 / **1b87ab32** and contains exactly **#430, #431, #432,
#433 and #434**. No unbriefed PR merged inside the range. Main and the public
PR listing were at this cutoff when checked at task start.

**Running at briefing, status only:** agent-core **45** repeats D-150's
measurement; devstack **32** handles isolation F19-02's capacity PATCH 412;
members **19** handles finite-grant restoration on early returns. They were
launched with this session. No delivered or merged PR for these follow-ups
was present at task start. Their results are outside this cut; any later
delivery is **after the cutoff**.

**D-098 / D-109 / D-132:** `TestHandover_Followup40Claims` pins five final
roadmap rows and five public PR excerpts, ten frozen #430 handover inputs,
the unchanged r26 archive and an independent source audit, dated documents,
per-follow-up lab/capture archives, and Go test declarations on main.
Selected current-run records are frozen from the exact cutoff into the
owned `history/follow-up-40/` archive, with full source hashes. The extracted
case/check inventory pins the ten original JSON sources separately. No claims
test reads a living members, isolation or usage evidence root. Earlier claims
retain their assertions; only rewritten-document inputs move to frozen bytes.

## Handover 39 (#430)

[PR #430](https://github.com/TED-H-com/Freedom-VPN/pull/430) records the
previous cut, **#428 and #429 up to 1b87ab32**. Its actual claims function and
manifests confirm **11 subtests**, **two final roadmap rows and two PR
excerpts**, **ten frozen #428 inputs**, **15 dated documents and 75 Go test
declarations**, the exact r26 archive and a separate source audit. The register
has **53 decisions and 99 findings rows**. These are verified inventories,
not counts inferred from the PR wording.

The body records **codex `gpt-6-astra`**, launched **13:18 EST**, delivered
**13:50**, **six commits and 32 owned files**. Admission waited **872.4 s in
total**; **suite wait 0 s**. No continuation. r26 remains **19,545 bytes**,
SHA-256 **8cd7cc8be881389d5f50e70abe2e18eb13136b2281d66a261d9ad5820a3f0d28**.

The one requested correction is additive beside the
[r26 citation errata](../reviews/README.md#r26-dispositions-at-the-follow-up-39-cutoff-2026-11-03-est):
at **8dec4100**, `baselineDecision` spans **node/watchdog/watch.go:170–209**,
and `mono()` starts at **211**. The earlier **170–215** range includes that
next function; the function name and first line are right. Neither the archive
nor #430's erratum is edited. **No finding and no disposition changes.**

## Agent-core 44 (#431)

[PR #431](https://github.com/TED-H-com/Freedom-VPN/pull/431) and the
[dated outcome](../node/followup44-outcome-2026-11-03.md) record D-150's
measurement after #429 merged, with 43's frozen arms: **A 567080be**, **B
38420dad**. Foreign CPU above **0.5 cores** is marked rather than stopping
the leg; overlapping spans are excluded from both arms. Other D-138 rules
remain unchanged.

**No comparison:** no leg reached a scenario, and each arm has **zero pooled
legs**. [Results](../node/followup44-timing-results-2026-11-03.json) retain
zero samples and absent quantiles for all five pooled spans in each arm.

| Window | What happened | Boundary |
|---|---|---|
| 1 | A1 stopped at preflight on another lane's running lab container; admission had printed free because it checked processes, not containers. | No scenario or Step span; B1/A2/B2 not started. |
| 2 | A3's clock-reader wrapper received an admission message on Docker stdout; make exited 2 after 14.1 s, before any scenario. | B3/A4/B4 not started. Wrapper repaired by sending diagnostics to stderr, regression-tested; all 24 synthetic tests pass. |

Both reservations were released; no third was used. The measured and repaired
runner hashes are separate in the [repair record](../node/followup44-runner-repair-2026-11-03.md).
The [clock prerequisite failed as written](../node/followup44-clock-prerequisite-2026-11-03.md):
containers had a separate time namespace, but monotonic and boottime offsets
were zero and all three host-bracketed readings aligned. PR #431 records the
orchestrator's reading: a zero-offset namespace with aligned brackets reads
the same clock, which is the needed condition. **Agent-core 45 repeats the
measurement with that condition.** No 44 leg is requalified.

The admission change belongs to the orchestrator's private tooling. **As
recorded in #431's body**, a reservation is now refused while another lane's
`fvpn-*` container runs; plain admission is unchanged. This handover inspected
no private implementation and gives no operator procedure. The ordinary main
production lab passed **13 scenarios**, D087/D104 included, and ordinary
gates passed. That is functional evidence, not a comparison or timing leg.
No production code changed in #431.

## Netlab 34 (#432)

[PR #432](https://github.com/TED-H-com/Freedom-VPN/pull/432) and the
[dated design/evidence](../poc/G2-followup34-2026-11-03.md) put **D-146 and
D-147 on main**.

**D-146: TCP MSS 1160.** For standard WireGuard UDP transport, overhead is
outer IPv4 20 or IPv6 40, UDP 8, WireGuard header/tag 32 and 16-byte padding.
At receive MTU 1280, the largest inner packets are **1216 / 1200**; subtract
inner IPv4+TCP's fixed 40 bytes to obtain **1176 / 1160**. Choose the smaller
**1160 for both underlay families**. An inner 1200 gives outer **1260 / 1280**.
TCP options consume that budget (timestamps give 1148 data bytes); extra
encapsulation or IPv6 extension headers need their own budget.

One non-verdict clamp immediately precedes each admitted IPv4 address's
general upload accept, and one its download accept. SYN, including SYN-ACK,
is required, and only an existing MSS option **above 1160** is lowered.
Missing/smaller options are untouched. Clamps contain **no counter or
priority**. Other rule order/expressions, DNS-service accepts and probe paths
are unchanged. No tunnel MTU changes. **New handshakes only**; existing
connections need a new handshake to negotiate the ceiling.

**D-147: three accepts per admitted address**, for outer sources in
`resolver_v4`, `denied_v4` or `host_v4`, exact admitted destination, conntrack
**RELATED**, IPv4 ICMP **type 3/code 4**. They follow admission, conntrack
guards and DNS accepts, and precede download source-deny drops. Each uses the
existing device download counter and priority; the first match accepts, so
overlap counts once. They grant **no upload reachability, probe exception,
unsolicited download, other ICMP type/code or IPv6 forwarding**. Parser/live
verifier require complete shapes, values, cardinality and placement; mutations
and foreign mangle, including `nftrace`, are rejected.

| Fixed IPv4-underlay lab row | Before | With #432 |
|---|---|---|
| Receive-only 1280 TCP | Download 0/64,064 bytes, 15 oversized reverse datagrams lost; upload completed. | 64,064 bytes each direction; member SYN 1380→1160, origin SYN-ACK 1460→1160; smaller 1000→1000; no PTB. |
| Receive-only UDP | Oversized reply lost. | Still lost; QUIC not tested because the lab has no QUIC client. |
| Private router, upload MTU 1000 | Upload timeout/0 bytes, PMTU 1420; 15 fragmentation-needed errors dropped at ctdenyd. | Upload 64,064 bytes, PMTU 1000; ten errors/5,760 IP bytes forwarded once as download in class 1:2f. |
| Negatives | Source isolation retained. | Time-exceeded at ctdenyd, unsolicited error at ctinv, member ping to router at ctdeny with zero upload usage. |

Independent node-egress IP-byte sums match usage in each direction. Class
packet counts match; tc bytes equal IP bytes plus 14 per packet, with zero
qdisc drops. This lab proves **IPv4 underlay and IPv4 overlay only**. It keeps
tunnel 1420, so its before packets are 1480 outer bytes, not the native 1340.
Native/IPv6-underlay evidence comes separately from members 18 below. UDP and
QUIC over a receive-only link remain limitations; physical arm64 is unmeasured.

The full netlab gate and **all 13 ordinary real-agent scenarios** pass,
D-087/D-104 included. **Gate 3a failed first because of the orchestrator's
brief**, as #432 explicitly records: “without edits” left the owned agent test
translator unable to parse the new statements, failing 22 top-level fixture
tests. [Continuation 1](../poc/G2-followup34-evidence/continuation1.md),
14:00–14:12 EST, extended only `node/agent/followup12_test.go`: SYN flag,
maxseg comparison/mangle, ICMP type/code, singleton RELATED `in`, numeric
priorities. Unknown/malformed statements still reject the entire batch
atomically; existing assertions and production files stayed unchanged.
**Gate 3a then passed**, and gates 4–9 were rerun. The first failure and the
focused numeric-priority mismatch remain in evidence. This is no D-133
comparison or zero-loss claim.

## Members 18 (#433): two assembled runs

[PR #433](https://github.com/TED-H-com/Freedom-VPN/pull/433) records D-098's
77-file archive of run 81b638e8 first, then exactly two assembled runs.

**Run 1, 118271c1**, harness **b4317bd4** on #432 / **27ff6392**, remains
**0 PASS / 3 BLOCKED / 7 FAIL**, subchecks **442 / 0 / 35**, byte-identical
under [history/follow-up-18-run1](../e2e/at/members/history/follow-up-18-run1/INDEX.md).
The producer reader rejected production's tc **retire** phase, introduced by
netlab 32 (#423), from sequence **582**, an Apply at **19:50:26Z** during
AT02. Sixteen export-prefix receipts join that rejection. Later node readiness
checks lacked producer records, so positive forwarding did not launch. The
70-minute context expired before AT09's native matrix: **no D-146 row was
exercised**. Run 1's failures stay failed; this joined cause does not explain
every independent failure.

The orchestrator's **15:58 note 1** authorized exactly one more assembled run
after the reader fix. Only `retire` was added to accepted phases; every other
lifecycle rule stayed unchanged, unknown phases still reject, and the closed
value audit found no other mismatch. **Run 2, 50117c68**, harness **4eb5d7d3**
on #432, lasted **2,955 s**, with no external-suite overlap observed. It retained
**1,534 AT02 producer records**, including `retire` at sequence **584**.

| Case | Run 2 verdict | Pass / blocked / fail | Observed boundary |
|---|---|---|---|
| AT01 | BLOCKED | 19 / 0 / 0 | Initial bounded transfer lacks counted/origin proof; platform sign-out/network/sleep-wake still requires operators. |
| AT02 | FAIL | 191 / 0 / 2 | D-077 re-grant counted forwarding now passes; D-135/D-136 and F11-03 still fail. All 1,242 losses count. |
| AT03 | PASS | 98 / 0 / 0 | Normal finite-grant restoration passes; early-failure cleanup is not exercised. |
| AT04 | BLOCKED | 113 / 0 / 0 | Persistent withdrawal measured; rendered suspended/expired screens remain operator/portal work. |
| AT05 | BLOCKED | 31 / 0 / 0 | Preinstalled/cloud-init provenance and physical arm64 absent. |
| AT08 | PASS | 35 / 0 / 0 | Observed acceptance retained. |
| AT09 | FAIL | 70 / 5 / 2 | D-067 provisional and combined same-generation restart still fail; platform/QUIC rows blocked. |
| AT20 | BLOCKED | 89 / 0 / 0 | Production nameserver distribution, Private DNS/DoH and Unicode UI resources remain. |
| AT21 | BLOCKED | 13 / 0 / 0 | TEST ONLY peer-group configuration does not prove automatic nameserver distribution/platform integration. |
| AT22 | BLOCKED | 40 / 0 / 0 | Representative URL privacy, remaining retention/backup surfaces and commercial feed approval remain. |

Total **2 PASS / 6 BLOCKED / 2 FAIL**, subchecks **699 / 5 / 4**. AT02 records
6,250 sends, zero send errors, 983 absent node ingresses and 259 ingress-without-
egress losses: **1,242 counted**, `measurement_valid=false`, three failed
Applies, cause unproven. AT09 serving readiness is **6.02 s**, counted recovery
**12.55 s**, generation **66→66**. No attribution to agent-core 37–43 is made;
namespace recreation is not proof of warm-kernel adoption.

**Every AT09 subcheck keeps its observed status:** indices **0–24 PASS**,
**25–26 FAIL**, **27–58 PASS**, **59–63 BLOCKED**, **64–76 PASS**. The
[frozen 77-row inventory](../../tests/handover/testdata/history/follow-up-40/members-run2/F18-AT09-subchecks.md.txt)
names each check. It retains the three 1500 rows, known-PMTU control, relay,
same-socket relay-to-direct transition, routed IPv6/IPv4 1280 and hold join;
Linux QUIC and Android/iOS/Windows/macOS TCP remain blocked. No other predicate
changes.

**D-151, provisional and overturnable by the owner**, records the orchestrator's
reading of D-146 as the approved D-137 hardening. Receive-only MTU 1280 TCP
may pass the unchanged byte predicate only with same-run evidence: SYN MSS
above 1160 arrives as 1160; every reverse datagram fits 1280, none missing;
no PTB. Without that evidence the row stays **blocked under D-137**, preserving
its measurements. An originally failed transfer remains failed.

| Receive-only row | Selected underlay | Member SYN MSS | Origin SYN-ACK MSS | Reverse sent/arrived | Largest sent/arrived | PTB | Verdict |
|---|---|---|---|---|---|---|---|
| dual | IPv4 | 1240→1160 | 1460→1160 | 73/73 | 1132/1132 B | none | PASS |
| IPv6 | IPv6 | 1240→1160 | 1460→1160 | 74/74 | 1152/1152 B | none | PASS |
| IPv4, new explicit row | IPv4 | 1240→1160 | 1460→1160 | 72/72 | 1132/1132 B | none | PASS |

Each receives **64,064 bytes with 64 exact echoes**; upload and evidence checks
pass. Both PTB types are absent at member, node and origin. The original
[D-146 joins](../../tests/handover/testdata/history/follow-up-40/members-run2/F18-D146.json)
and immutable per-row header archives retain exact matching. Smaller-MSS
control is **not exercised** here; netlab 34's 1000→1000 control stands.
17's 1132-byte datagrams came from **1006-byte echo writes**, not a clamp.
The rewritten endpoint advertisements are direct evidence of clamp operation.
Historical **F8-02 stays failed**. UDP/QUIC receive-only without PTB,
other encapsulations, relay-at-1280, real platforms and physical arm64 remain
unverified/limited. IPv6 underlay does not mean IPv6 overlay: D-123 is unchanged.

The new request is public-API grant restoration after finite-grant early
failed-readiness/positive-probe returns, routed to **members 19**, open.
Run 2's successful normal cleanup does not close it. Both prior devstack
requests and seven carried groups remain as #433 records them.

## Isolation 19 (#434)

[PR #434](https://github.com/TED-H-com/Freedom-VPN/pull/434) and the
[dated D-147 record](../e2e/at/isolation/d147-related-frag-needed-2026-11-03.md)
retain the 140-file D-098 archive, separate D-147 test/record and unchanged
legacy member-accept predicate. Note 2 (**15:59**) added production's `retire`
phase before the assembled run; unknown phases still raise, the audit found
no other mismatch, and **23 synthetic reader tests pass**. The assembled run
produced no timed operations, so this fix was **not exercised at runtime**.

Fresh lab **1d746f9a** on committed **3e12f913** installs production rules via
normal Apply/live verification. All four source sets independently pass.
Each learns PMTU **1000**, completes **64,064 upload bytes** and forwards
**ten genuine fragmentation-needed errors / 5,760 IP bytes** as download.

| Exact source set | Upload packets / IP bytes | Download packets / IP bytes | tc upload / download bytes | Verdict |
|---|---|---|---|---|
| resolver_v4 only | 84 / 80,007 | 44 / 7,601 | 81,183 / 8,217 | PASS |
| denied_v4 only | 84 / 80,007 | 36 / 7,185 | 81,183 / 7,689 | PASS |
| host_v4 only | 84 / 80,007 | 38 / 7,289 | 81,183 / 7,821 | PASS |
| denied_v4 + host_v4 | 84 / 80,007 | 50 / 7,913 | 81,183 / 8,613 | PASS |

All use existing priority **0x0001002f / class 1:2f**; captures match usage
exactly, tc adds 14 Ethernet bytes per packet, zero drops. Overlap counts once.

| Source set | Time-exceeded | Unsolicited fragmentation-needed | Upload to source | Verdict under D-152 |
|---|---|---|---|---|
| resolver only | ctdenyd, 1 / 112 B | ctinv, 1 / 60 B | ctbadns, 1 / 84 B; zero usage | all PASS |
| denied only | ctdenyd, 1 / 112 B | ctinv, 1 / 60 B | ctdeny, 1 / 84 B; zero usage | all PASS |
| host only | ctdenyd, 1 / 112 B | ctinv, 1 / 60 B | cthost, 1 / 84 B; zero usage | all PASS |
| denied + host | ctdenyd, 1 / 112 B | ctinv, 1 / 60 B | ctdeny, 1 / 84 B; zero usage | all PASS |

**F19-01 is answered by provisional D-152**, which the owner may overturn.
The brief's literal `ctdeny` meant the applicable production destination-deny
rule: resolver-only **ctbadns**, host-only **cthost**, denied/overlap **ctdeny**.
No rule order changes. The requirement is unchanged: upload to the source
is dropped with zero upload usage. The original literal resolver/host ctdeny
subcases remain **not exercised** in their archived records; the observed
policy denials pass the clarified requirement.

The forward view is **27 member accepts, nine D-147 accepts, six D-146 clamps**
for three admitted addresses. Existing **AT10–AT13 lab rows all pass unchanged**
in the same session. Failed attempts stay failed: the inherited server read
only 2,048 bytes, counter names were wrong, then reused-lab streams contaminated
zero-usage checks. These were fixture failures; no production rule changed.

**One assembled run, 70beb28e**, is blocked. Its lab phase passes again; the
shared publisher's capacity PATCH returns **412 RESOURCE_VERSION_CONFLICT**,
and publication never reaches serving. AT10 **0/2/0**, AT11 **0/2/0**, AT12
**0/3/0**, AT13 **0/5/0**: **all 12 checks BLOCKED**, real-node D-147 not
exercised. Later hold/restart/revoke/expiry checks do not inherit older results.
The initial window is confounded under D-133: #434's body identifies the
orchestrator's #433 merge-chain gates in it. No retry. Lab success does not
promote these assembled rows or the full native fault/platform matrices.

**F19-02 remains OPEN, routed to devstack 32**, running at briefing. #434's
body records a one-shot ETag read/capacity PATCH. A concurrent agent observation
stored in the same second probably changed the version; **cause unproven**.
The routed work re-reads/retries 412 while retaining every attempt; no predicate
relaxation. Isolation's three prior requests remain carried.

## Decisions, register and hand-offs

No decision after D-150 existed on main before this cut. D-146/D-147 change
status/date only to **on `main` (#432)**; D-150 keeps its substance and records
44's attempt without comparison and 45's repeat. D-151/D-152 are the
orchestrator's provisional decisions, not new owner policy. **55 decisions**.

The [findings register](README.md#findings-register-at-the-follow-up-40-cutoff)
has **102 rows: four updated, three added, 95 unchanged**. New rows are F19-01
(answered), F19-02 (open/devstack 32), members 18 grant restoration
(open/members 19). Every unrelated row is byte-identical. Every historical
failure stays failed. No C1 implementation, F30-01 fix, upstream F11-01
resolution, complete platform acceptance or timing comparison is implied.
No scheduled-run result is supplied by this cut.

### Requests to orchestrator

**2026-11-03 EST — no new outside-owned pin or scope request; no item stopped.**
F19-01 is answered by D-152. Carry the already assigned devstack 32 capacity
PATCH work, members 19 finite-grant cleanup and agent-core 45 measurement.
All are status only here, with no promised date. Earlier owner-resource and
upstream requests remain with their recorded owners.
