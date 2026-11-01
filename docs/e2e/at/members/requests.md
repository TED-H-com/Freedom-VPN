# Members requests and external blockers

Current full-case status and evidence: [summary.md](summary.md). Follow-up 10 run
`2be2d32f` is immutable in [follow-up 10](history/follow-up-10/RUN.md), with its
[35-file checksum index](history/follow-up-10/SHA256SUMS) (D-098). Follow-up 9 run
`11e305fe` is immutable in [follow-up 9](history/follow-up-9/RUN.md), with its
[33-file checksum index](history/follow-up-9/SHA256SUMS) (D-098). Historical
run `69f92f49` and its requests are immutable in [follow-up 8](history/follow-up-8/requests.md),
with [32-file checksum index](history/follow-up-8/SHA256SUMS) (D-098).

## Follow-up 13 — 2026-11-01 (provisional D-135, D-136, D-137)

D-098 first: all 52 root files for run b98b78ee are byte-identical in
[follow-up 12](history/follow-up-12/RUN.md), with [SHA-256 pins](history/follow-up-12/SHA256SUMS).
AT01/AT09 within that archive still belong to run 7edb263f, as recorded then.

**F12-01 interpretation under provisional D-136:** the generation-11 lease
while policy advances to generation 13 causes the designed node-wide
`policy_missing` baseline until renewal and the next successful Apply.
Original producer sequences **362–364 baseline, 365 renewal, 366 Apply**,
**236.970251 ms**, **nine losses** (one inside a completed guard, eight outside)
are unchanged. The complete sibling stream retains **3,976 attempts / 21 losses**
(13 inside completed guards, eight outside). No failed Apply occurred.
The bracket rule covers every node-wide baseline call; overlapping calls are
reported separately and never summed as unique losses. Recovery is required;
no bracket or guard excludes a loss or turns a failed availability row into PASS.
Stronger availability is agent-core design under D-133, outside this lane.
[F12-01 immutable records](history/follow-up-12/F12-01-policy-missing.json).

F11-03 remains blocked on **no failed Apply observed**: all 95 AT02 Applies
in this assembled run succeeded. Netlab 22's executor lifecycle is merged, but its hand-off
and roadmap leave the engine opt-in to agent-core 31; unavailable command
records must not be invented. Baseline call success is retained separately
from a failed Apply: failed cleanup is not proof of a committed baseline.

**AT09 under provisional D-137:** the separate selected-native-peer route-MTU
1280 control measures known PMTU. Receive-only 1280 without PTB remains an
explicit PMTU black-hole limitation, never a supported-path PASS, even if a
particular bounded transfer completes. Historical F8-02 rows remain failed.
Reactive PTB is pending devstack 23; no router-hop export exists on this baseline.

F12-02's streamed reader is now live-qualified across all 22 readiness and
12 monitor stream checks. F12-03 uses a six-second quiet window, then a fresh negative and
its observed remaining SOA TTL minus one second of precision uncertainty.
The existing combined before-TTL assertion is unchanged. F12-04 remains with
devstack; the old native SERVFAIL and both positive resolver receipts stay recorded.

### Fresh results

- **F11-01:** first selection succeeds with one SYN, node/origin receipt and counted traffic. All three captures are complete (269 / 292 / 77 records, zero drops). Type-4 transport data appears before a loopback-source IPv6 response arrives after the successful request. This differs from devstack 22's five-SYN/no-type-4 reproduction; neither historical cause nor a fix is inferred. The original prehistory and sleep/wake branch remain. [Own capture and comparison](F13-F11-01-forwarding.json).
- **AT02 under provisional D-135 + D-136:** 95 successful Applies, zero failed Applies; F11-03 stays **blocked on no failed Apply observed**. All 61 baseline calls are paired to the next successful Apply, with original rows/run identities. Three overlap the sibling window; the other 58 have command recovery evidence and explicitly unavailable sibling coverage. **4,029 attempts / 4,029 ingresses / 4,008 egresses / 21 losses**; 13 inside completed guards, eight outside. No losses are excluded. Sequences **402–404** are successful `policy_missing` baselines on the generation-11 lease, **405** renews to 13, **406** applies successfully. The first bracket is **219.395230 ms**, contains nine losses (one inside a guard, eight outside), and has first subsequent egress **14.828914 ms** after Apply. Repeated baseline brackets overlap and are never summed as unique losses. The no-loss row stays FAIL, even though recovery is measured. [Full accounting](F13-AT02-recovery.json), [original transition](F13-AT02-policy-missing.json).
- **F12-02 live-qualified:** all **22 readiness checks** across AT03/AT04/AT08 and **12 monitor stream checks** pass. AT03 streams 1,160,894–1,277,148 bytes; AT08 reaches **2,455,378 bytes / 2,262 records**. AT04's primary refund monitor has 196 samples with **392 exact producer identities beyond 1 MiB**, cumulative offsets 1,407,676–1,527,951. Secondary streams are reported at their actual smaller sizes when applicable, never padded. [Qualification](F13-F12-02-stream-qualification.json), [immutable stream and failed monitor](history/follow-up-13-stream/4182dd3d-7f1f-453e-8c1f-265d097c3abc/SHA256SUMS). Run b98b78ee's eleven readiness failures and early-finished monitor remain failed in history.
- **F13-01:** AT04 has two separate failed persistent-withdrawal checks: primary refund and secondary suspension each contain one `nft_parse` / `parse` sample with command exit 0. The remaining status reads succeed; observed reply cutoffs are 3.558769767 s and 2.596756019 s, with active senders and stable counters. Incomplete reads keep both rows failed. Raw nft output/parser errors were not retained, so their cause is unproven. [Failed rows, neighbors and producer windows](F13-01-counter-parse.json); shared reader/parser changes are handed off.
- **F12-03 quiet window passes:** six seconds without fixture queries (measured 6.004276509 s), then fresh negative SOA TTL **5 → 3**, same uniquely matched primary, approximately **1.221 s** between query completions, before the receipt-plus-four-second conservative boundary. No new authority fetch or outstanding completion is recorded. The combined before-TTL assertion is unchanged and passes. No shared fixture, cache setting or flush changed. [Exact window](F13-F12-03-negative-cache.json). Historical TTL 2 / 1.228672687 s F12-03 stays failed. F12-04's original SERVFAIL and positive resolver receipts remain unchanged and owned by devstack; this run does not diagnose them.
- **AT09 verdict: BLOCKED under provisional D-137.** Known-PMTU 1280 completes **64,064 bytes in 1 ms**, with route metric 1280 before/after traffic, exact removal and **11 delivered reverse IPv6 fragment pairs**. Reactive PTB remains pending devstack 23 on this tested baseline. Both receive-only 1280 rows also transfer **64,064 bytes in 1 ms**, but remain explicit **PMTU black-hole limitation rows, never PASS**. Reverse data packets are only 1,132 bytes on IPv4 / 1,152 bytes on IPv6, so oversized loss was not reproduced; no PTB appears. Historical F8-02 failures stay failed. [Separate rows and captures](F13-D137-native-paths.json).

### Pending request groups — 2026-11-01

**Current count: ten groups.** Work outside the owned lane stops for these items;
no outside-owned file was modified.

1. **F8-02 / D-137 — devstack 23:** reactive PTB router-hop export/evidence remains pending (`tests/e2e/native_underlay.go` and shared native harness). Known-PMTU passes; receive-only rows are limitations, never PASS. No global MTU/MSS clamp is requested.
2. **F4-03 — platform/operator:** real platform/QUIC, physical second NAT/roaming and PMTU resources; `tests/e2e` native fixture/platform integrations are outside this lane.
3. **F4-04 — portal/devstack/operator:** rendered expired/suspended screens (`apps/portal`); API/traffic evidence does not certify UI.
4. **F4-10 — packaging/devstack/operator:** boot-flavour provenance and physical arm64; `packaging` and shared boot fixtures. Cross-build is separate.
5. **F4-05/F4-07 — devstack/native integration/operator:** positive IDNA wire fixture and production nameserver distribution / Private DNS / browser DoH (`tests/e2e/native_dns_fixture.go`, platform integrations).
6. **F4-06 — usage/database/backup/business:** representative URL/privacy and remaining raw/rollup/file/backup retention surfaces and commercial feed approval; `internal/usage`, retention/backup components and operator resources.
7. **F11-01 — native NetBird/WireGuard integration:** fresh first selection succeeds with one SYN/type-4 data; no historical cause or repair is proven. Retain devstack 22's handshake/session-acceptance investigation (`deploy/devstack/Dockerfile.node`, shared native capture/transport fixtures). [Own comparison](F13-F11-01-forwarding.json).
8. **F11-03 — orchestrator / agent-core 31 / netlab:** **no failed Apply observed** in AT02; retain the blocked failed-Apply measurement. The TEST ONLY engine caller for netlab 22's lifecycle instrumentation is outside scope (`node/agent/engine.go`); no failed Apply is manufactured. Stronger admitted-sibling availability under D-133 is separate design work.
9. **F12-04 — devstack/native integration:** original native SERVFAIL with positive primary/secondary receipts remains unattributed (`tests/e2e/native_dns_fixture.go`, `tests/e2e/dns_trace.go`). New successful RR rows do not diagnose the original delivery/client-processing segment.
10. **F13-01 — devstack / enforcement owner:** diagnose two `nft_parse` / `parse` samples with exit 0, primary refund and secondary suspension. [Original failures and producer windows](F13-01-counter-parse.json). `tests/e2e/internal/counterread/read.go` calls `node/enforce/parse_nft.go` but retains neither raw failed bytes nor the parser error. Any safe diagnostic/classification expansion belongs to those owners. No retry or predicate relaxation replaces either failed row; underlying cause remains unproven.

F12-01's interpretation question is answered by provisional D-136; old numbers
stay unchanged. F12-02 is live-qualified. F12-03 has a fresh quiet-window pass
without shared fixture changes. F11-02 stays closed by populated-grant evidence.
D-135 / D-136 / D-137 remain provisional pending owner confirmation; no stronger
availability or hardening choice is inferred.

## Historical follow-up 12 — 2026-10-31 (D-098, D-134, D-135)

Run 7edb263f is preserved byte-identically in [follow-up 11](history/follow-up-11/RUN.md),
with [43-file SHA-256 index](history/follow-up-11/SHA256SUMS); attempt-1 stays.
The original F11-02 and F11-03 failed rows retain their original expectations.

**F11-02 readiness predicate:** before any positive forwarding probe, require
an accepted report received within 15 seconds for the current incarnation,
`member_forwarding_enabled=true`, no forwarding block reasons, and
`desired = applied = observed = report's lease generation`. That exact lease
must be unexpired and have populated signed member grants. The intended device
must be authorized with a current binding and a live member grant in that lease;
its binding must appear in the node's local desired state at the same generation.
Join the report's boot ID and lease ID to an original producer serving heartbeat
with the same generation and populated members; retain its run ID and sequence.
A missing prerequisite stops the positive probe. Counted forwarding remains an
independent assertion. **F11-02 fresh result:** both D-077 re-grants pass at
generations 10 and 21 with one populated member grant, joined to producer run
`3065df99` sequences 219 and 602. [Exact rows](F12-F11-02-readiness.json). No generation, grant or observation is manufactured.

**F11-03 / D-135:** failed Apply installs node-wide generation-0 baseline,
preserving revocation; no prior-generation restoration. Require revoked-source
drop and bracket the admitted sibling outage from baseline command start/end
to the next successful Apply end. Retain every loss and failure; continuous
20 ms native UDP/40002 metadata and the separate UDP/9 ingress controls remain
independent. Report first egress after recovery (one-second observation bound,
not an outage ceiling). Missing verification, failed verification and later
member-path withdrawal are separate. Missing failed Apply makes this row blocked.
The producer stream retains `host.before_apply`, `host.after_apply`, command kind,
optional exit/signal and original run/sequence. Only a failed install exceeding
its recorded bound with valid ordered host samples, >1 runnable at both ends
and zero aggregate idle delta receives the conservative “fail-closed install
overrun with recorded host contention” label. All other observations remain
explicit; **cause_status=unproven in every case**. The label never changes a
verdict. [Agent-core 29](../../../node/followup29-install-contention-2026-10-31.md)
did not reproduce the overrun under CPU contention; no historical cause is claimed.
D-134's expiry window is [D−101 ms,D]; every pre-D and renewal loss remains counted,
and no fixed outer-guard ceiling is inferred.

**AT20 / dns-publish 9 request 2:** [BACKEND §2.1.5](../../../dns/BACKEND.md#215-f10-03--positive-cache-and-ttl-measured-2026-10-31)
F9-HIT / F9-OTHER / F9-ROUND / F9-INFLIGHT govern attribution. Keep the actual
responder, TTL, before/after settings, whole authority request history, outstanding
sets and completion times. Different unique responders establish independent
caches; fallback needs failure/timeout plus secondary receipts. Same-resolver
fetch is not fallback; sub-second expiry and older outstanding completions must
be considered. Missing lifecycle/invalidation evidence leaves cause unproven.
The before-TTL assertion is unchanged. Run 2be2d32f and F10-03 stay failed/unattributed.

**Owner answers consumed:** devstack 21 bounds F11-01 to after member inner egress
and before observable node inner ingress; it identifies F11-03's failed install
stage, not cause. Netlab 21 reproduced the receive-link MTU mechanism on IPv6
(19 reverse 1,360-byte packets missing before member underlay ingress, no PTB)
and measured real rollback sibling closure. **F11-01 and F8-02 wait for devstack
22's route-selection exports/native captures; no AT01/AT09 re-observation here.**
Their prior case records and failed measurements remain recorded. AT01 setup
mutations required for AT02 run inside AT02; the remaining regular order is unchanged.

Pre/post-startup admission is a hard stop at two other active suites (nested
make/wrapper processes count as one suite). Run `assembled.py` through `disktmp.py`;
it tears down the owned stack on all exits. The live make target also hard-checks.

**Current request count: twelve groups** — six carried (F8-02; F4-03; F4-04;
F4-10; F4-05/F4-07; F4-06), F11-01, F11-03 and F12-01–F12-04.
F11-02 is closed by the fresh measurement; the old failure stays recorded.
This run has eight fresh cases, 468 passed and 14 failed checks, four full cases
blocked and four failed. AT01/AT09 remain historical and are excluded from that total.

### F12-01 — admitted sibling loss during policy_missing transition, 2026-10-31

Owners: **agent-core / netlab**, with devstack if transition-input exports are
needed. Run `b98b78ee` has zero failed Applies, so F11-03 stays blocked. The
continuous admitted sibling stream has 21 losses, all retained; 13 ingress
timestamps are in completed guards and eight outside. All eight outside-guard
losses are within the separate baseline-362 → successful-Apply-366 command
bracket: **236.970251 ms**, with nine total losses (one also in the guard).
Original same-boot producer run `3065df99` records three successful
`policy_missing` baseline calls 362–364 while retaining the generation-11 lease,
then a populated generation-13 renewal 365 and successful verified Apply 366.
[Exact records and packets](F12-01-policy-missing.json), [all 21 losses](F12-AT02-recovery.json).
Please interpret the missing-policy transition and its sibling availability
contract from these records. This is neither an install overrun nor a failed
verification or failed-Apply rollback. Underlying cause remains unproven; no
configuration change, traffic retry, waiver or new outside-owned edit is made.

### F12-02 — producer export capacity, 2026-10-31

Owner: **members consumer**, corrected within this lane; **orchestrator** for a
future assembled qualification. The single run `b98b78ee` is retained unchanged.
AT03's positive probe stops at readiness. The cumulative original timing stream
first crosses the shared `lab.docker` 1 MiB diagnostic limit at `22:03:37.574134738Z`,
producer run `6a7edc0d`, sequence 187. The failed gate is at `22:05:34Z`.
A read-only call to the same `EnforcementTiming` export returns its explicit
1 MiB error; the existing `ExportContainerFile` returns 1,591,639 bytes successfully.
[Failed row, original producer window and export diagnosis](F12-02-stream-limit.json).
The serving report and producer heartbeat are present; the failure is not proof
of a product forwarding failure. The old gate did not retain every poll's stage,
so the export limit is a sufficient blocker, not proof that every other predicate
succeeded at every instant.

AT04 retains six readiness failures and a seventh failed withdrawal control:
the secondary 85-second monitor ended at `22:07:43.568861416Z`, before refund
input `22:09:19.760245577Z` after the primary readiness waits. Its missing first
failure is not a product withdrawal latency; the generated negative sentinel
is not interpreted as a duration. [Original failed rows](F12-AT04-failures.json).

The owned consumer now streams to disk using that existing exported hook,
rejects partial JSONL records and keeps original decimal strings/fields. A
>1 MiB regression and partial-tail negative pass. Future failures include their
last prerequisite stage and sanitized reader error. No shared harness edit or
larger diagnostic-buffer limit is requested. **No live rerun is made:** this
single run's affected failures remain failures; the corrected consumer has
read-only export and offline coverage, awaiting the next authorized assembled run.

### F12-03 — negative cache window, 2026-10-31

Owners: **members / dns-publish**, with **devstack** for any isolated fixture
needed. The unchanged combined before-TTL row fails on its negative half;
positive primary TTL 10→8 is an inferred hit. Initial negative SOA TTL is 2;
same-primary positive refetch follows 1.228672687 seconds later, within the
whole-second expiry uncertainty. The older NXDOMAIN authority completion was
already about 3.49 seconds before that initial negative receipt. No outstanding
completion or resolver switch is present. [Exact row and history](F12-03-negative-cache.json).
Please settle a repeatable quiet negative-cache setup/window using actual
remaining TTL and the measured precision; keep this failure. No current assertion,
cache setting or historical verdict is changed. Any shared fixture expansion
belongs to devstack (`tests/e2e/native_dns_fixture.go` and the matching
`tests/e2e/internal/dnsrelay/trace.go` case mapping), outside this lane.

### F12-04 — native SERVFAIL with positive resolver receipts, 2026-10-31

Owner: **devstack / native integration** (`tests/e2e/native_dns_fixture.go`,
`tests/e2e/dns_trace.go`), with netlab for packet interpretation if required.
UDP RR case 01 returns client RCODE 2 after 5,001 ms despite RCODE 0 / TTL 5
receipts from primary and secondary, each completed in less than 2 ms.
The complete trace has **no unique matching responder** for the observed error.
[Original failed row and exchanges](F12-04-native-servfail.json). Please locate
the unobserved reply-delivery/client-processing segment using native exports;
this lane makes no cache, AdGuard SERVFAIL or transport-cause claim and no
repeat query intended to replace the failed row. Both resolver contacts are
recorded; successful fallback is not established. `cause_status=unproven`.

## Follow-up 11 — 2026-10-31 (D-098, D-102, D-129, D-132)

Run `7edb263f-1f6e-4f94-9e8e-26099f96a699`, harness `33abb2fd`, baseline
`96ec0630` (devstack 20 #343, agent-core 28 #344, netlab 20 #336).
**2 complete cases pass / 6 blocked / 2 fail; 558 pass / 5 blocked / 3 fail checks.**
AT02's finite-expiry reader row and AT20's unchanged before-TTL row pass as new
measurements; F10-02/F10-03 stay historically failed and unattributed.
F10-01 is closed by netlab 20; F10-04 is closed under D-129.

The unchanged full AT09 sequence recovers at generation **30 → 30**, serving
in **6.032389860 s**, first counted byte in **11.399224550 s**, with three
D-126-valid provisional reports (including the original mixed safety shape).
The initial tc rejection remains in the full capture. This timing differs from
follow-up 10 on the baseline that includes [agent-core 28](../../../node/followup28-iso-f6-03-2026-10-31.md);
no causal latency comparison is claimed. The full case still fails its
MTU-1280 row, and all five unavailable selectors remain blocked with their
resource reasons. No forced generation, state repair or topology repair.

**Nine pending request groups:** the six carried groups (F8-02; F4-03;
F4-04; F4-10; F4-05/F4-07; F4-06), plus F11-01/F11-02/F11-03 below.
The two historical diagnostic-export requests are answered, with original
failure attribution still unavailable. No new request is filed outside this
owned register.

## Historical follow-up 10 — 2026-10-31 (D-102, D-126, D-128)

Baseline `d3e9a0f1` includes agent-core 27 (#325), wire-nodes 32 (#321) and
netlab 19 (#318). **F9-01 is closed as a predicate mismatch under D-126**;
see the producer/central answers in its section below. The replay accepts the
three original provisional observations, including the mixed safety report,
and rejects unexplained safety codes. It does not rewrite the archived failed
run or prove live recovery. That full mutation sequence is recorded in
[archived RUN.md](history/follow-up-10/RUN.md); full-case status continues to include resource blockers.

**Historical run `2be2d32f`: 2 pass / 6 blocked / 2 fail; 560 / 5 / 2 subchecks.**
AT09 has 38 pass / 5 blocked / 0 fail checks, with live mixed-report acceptance
and recovery, but the full sequence produced generation 29, not 30 (F10-04).
F10-02 (AT02 unreadable sample) and F10-03 (AT20 before-TTL cache result) remain
failed. F10-01 is the external pin correction. The new dated sections below
retain captures and owners; none of these findings erases older evidence.

F8-02 stays unattributed. Netlab 19's answer and devstack 19's exact-sequence
decode ownership are recorded below. The dated F4-03 / F4-04 / F4-05 / F4-06 /
F4-07 / F4-10 resource requests and AT20–AT22 DNS expectations are unchanged.

## Follow-up 9 — 2026-10-31 (D-102)

Baseline `7e044c71` includes agent-core 25 (#312), devstack 18 (#313),
dns-publish 8 (#309), and netlab 18. The original full mutation sequence
adopts `RestartNodeNamespace90`; no standalone scenario pass changes AT09.
Run `11e305fe-6360-4dbd-95b1-67274aa4988c`, harness `49fbb48e`, finishes **3 pass / 6 blocked / 1 fail**. AT09 remains failed on F9-01 below. F8-01 recovers in the original generation-30 sequence; the native MTU-1280 row passes this time, while the historical F8-02 cause remains unattributed.

### D-119 — finite expiry and billing replacement

Resolved by orchestrator, 2026-10-31: retire the inherited “3/0 → 2/1 deadline transition” wording.

Plan v1.0 has no scheduled plan-change contract. A finite
grant's deadline means expiry (0/3 when no other grant covers the member).
A period-boundary lower plan is a billing event that calls `ReplaceFiniteGrant`
then, immediately revoking and creating the replacement. AT02 retains natural
expiry → 0/3 and immediate replacement → 3/0 → 2/1 → 3/0. No new production
schedule, fallback grant or overlapping different-plan grant is fabricated.

### F8-01 — closed by full mutation sequence recovery, 2026-10-31

Owners: **w2-agent-core / w3-devstack-e2e**, with **w3-wire-nodes** for lease
readback. [Agent-core 25](../../../node/2026-10-31-d104-d106-agent.md) keeps
recovery fail-closed and renews against valid desired generation while an Apply
error is retained. The [devstack 18 evidence index](../../runs/2026-10-31/followup18/README.md)
reports unchanged generation 3 recovery in 5.96 s with the initial
`TC_FOREIGN_FILTER` retained. It explicitly does not close the original
generation-30 failure. This lane records the complete 90-second window,
boot/final nft/tc, heartbeats, observations, agent log, generation equality,
first fresh serving receipt and first counted byte without repair.

**Follow-up 9 result (2026-10-31):** desired/applied generation **30 → 30**; namespace restart
`2026-10-31T12:06:31.816656225Z`; first fresh serving receipt
`12:06:47.545258Z` (**15.728602 s**); first counted byte
`12:06:53.190469871Z` (**21.373814 s**). A new generation-30 lease at
`12:06:47.408641Z` has seven member grants. The capture retains **88 heartbeat
samples over 90.337054 s**, boot/final nft/tc, every central observation and the
agent log, including the initial `TC_FOREIGN_FILTER`. No generation, topology
or state repair occurred. [Complete historical capture](history/follow-up-9/AT09-restart.json).
This closes the original non-recovery failure, not the full AT09 case.

### F8-02 — MTU 1280 bulk transfer incomplete

Owner: **w3-devstack-e2e / native fixture investigation**; wire-nodes and netlab
supply path evidence. The archived failure remains **11,011 / 64,064 bytes in
12,011 ms**, direct IPv4 selected before/after, counters +66,296/+45,963.
[Devstack 18](../../runs/2026-10-31/followup18/README.md) reproduced partial
transfers with and without reconnect and with either underlay family; measured
SYN MSS 1240/1460 and no fragmentation-needed/PTB in its bounded captures.
It cannot assign the exact loss/MSS/fragmentation or fixture read/echo cause.
[Netlab 18](../../../poc/G2-f802-mtu1280-2026-10-31.md) completes IPv4 at 1280
and proves IPv4 fragmentation-needed errors traverse nft/tc in both directions.
Neither independent result attributes the original native failure.

**Follow-up 9 result (2026-10-31):** dual underlay / MTU 1280 completes **64,064 / 64,064 bytes in 1 ms**, with direct IPv4 selected before and after; device counters increase **67,652 upload / 68,139 download bytes**. All three preceding 1500 rows also complete. [Archived NATIVE.md](history/follow-up-9/NATIVE.md) and [AT09.json](history/follow-up-9/AT09.json) retain every family/path and counter. This is a passing row in the full mutation sequence, not proof of a root-cause fix or an overall AT09 pass.

**D-123 (2026-10-31): IPv6Block is the v1.0 overlay mode.** The rendered
`ctv6` drops in both directions prove the netlab IPv6 no-leak control; no
end-to-end IPv6 MTU/PMTU row exists or is claimed. Native family selectors
control underlay availability; all bulk payloads here use the IPv4 origin.
Keep F8-02 unattributed between NetBird reconnect behavior and fixture echo/read
behavior until the same row has SYN/MSS, ICMP errors, interface MTUs and
application read counts sufficient to decide. Do not call it an IPv6 problem.

**Netlab 19 answer (2026-10-31, #318):**
[G2 F8-02 attribution](../../../poc/G2-f802-native-attribution-2026-10-31.md)
finds no single transport cause: the failed and successful IPv4 captures have
unchanged 1240/1460 SYN/SYN-ACK MSS, and published SYN-only summaries cannot
resolve TCP retransmission, outer fragmentation or endpoint read/write stalls.
No central/nft/tc cause or production workaround is established. **Devstack 19
owns the exact-sequence decode/capture**: full payload-free TCP and outer
headers, capture completeness and endpoint read/write/partial-line counts,
with the original 12-second deadline and 64,064-byte requirement. The original
11,011-byte failure remains failed; the follow-up 9 MTU-1280 pass is no fix proof.

**Follow-up 11 answer, 2026-10-31:**
[Devstack 20's instrumented failure](../../runs/2026-10-31/followup20/capture-cce1a688/README.md),
run `cce1a688`, transfers **10,010 / 64,064 bytes in 12,011 ms at MTU 1280**
on **attempt 1 of 3**; the campaign stops after that failure. Nineteen reverse
outer IPv4 datagrams of length **1,340** are absent at the member; all seventeen
of length **1,132** arrive. No PTB is captured. Endpoint counts, full payload-free
headers, original pcap pins, TCP_INFO and capture limits are retained there.
**Netlab 21 owns interpretation.** Receive-link MTU rejection remains an
inference, not an established exact drop reason. This new failure does not
rewrite the historical members boundary or any later native bulk measurement.

**New members follow-up 11 measurement, 2026-10-31:** full AT09 is
**fail, 37 pass / 5 blocked / 1 fail**. The dual/1280 row returns
**13,013 / 64,064 bytes in 12,010 ms**, `bulk_echo_incomplete` / endpoint
`timeout`, with **direct P2P IPv6 underlay selected both before and after**.
The payload is IPv4 (D-123). Endpoint write/read counts are **64,108 / 13,081**,
with **13 complete lines**, zero partial-line bytes, `bytes_acked=64109`,
`notsent_bytes=0`, and `total_retrans=0` in the retained client TCP_INFO.
Named device counters increase **66,216 upload / 38,241 download** bytes.
All three MTU-1500 rows pass (64,064 bytes in 2 / 1 / 1 ms).
[Exact failed row](history/follow-up-11/F11-F8-02-mtu1280.json), [all rows](NATIVE.md).

This new failed transfer remains assigned to **netlab 21** with F8-02. Its
IPv6 underlay and 13,013-byte boundary are distinct from devstack 20's IPv4
outer-datagram capture and the historical 11,011-byte members failure. No
per-datagram capture is claimed for this run, and the client TCP_INFO alone
does not identify the lost response hop. No MTU/MSS workaround, retry or
assertion change is made.

### F9-01 — mixed provisional lease and tc-rejection observation, 2026-10-31

**Closed 2026-10-31 under D-126 / D-128.**
[Agent-core 27's hand-off](../../../node/followup27-members-2026-10-31.md)
and [PR #325](https://github.com/TED-H-com/Freedom-VPN/pull/325) confirm the
existing deliberate synchronous single-decision composition, **no reporter
race and no producer fix**. `12:06:32.527615Z` is receiver time; the captured
body's `observed_at` is `12:06:32.521093013Z`. Reasons accumulate; a retained
enforcement failure overrides the three dependent checks, while the sole
lease-stage error retains `AWAITING_MEMBER_GRANTS` and its provisional message.

[Wire-nodes 32, PR #321](https://github.com/TED-H-com/Freedom-VPN/pull/321)
proves acceptance, blocked publication, stored-ACK replay and value-unchanged
storage/projection of this exact report (JSON-value equality, identical JCS
and SHA-256). Under D-128, JSONB need not preserve whitespace/key order;
`SubmitObservation` remains 202 Accepted. No schema or HTTP change is needed.

The retained revision-11 predicate now permits each failed lease/qos/source_binding
check to carry AWAITING, or the active enforcement safety code only with
`safety_check_failed`, failed safety and matching apply/preflight stage evidence.
Historical tc/quarantine rows alone cannot justify that override. Clean
provisional checks still require AWAITING, including the lease detail; exactly
one stage=lease error must retain the provisional cause; `lease_expired` stays
rejected. The offline test replays the entire six-report captured sequence and
its JCS values, accepts all three provisional rows, and rejects the mixed row
without the safety reason as well as missing active-stage evidence, overwritten
or duplicate lease causes, expiry and a passed dependent check. The captured tc
rejection remains in the immutable archive and is never converted to safety
success. Full AT09 acceptance remains separate from recovery or replay success.

**Historical follow-up 9 failure/request (superseded by the answer above):**

Owners: **w2-agent-core / w3-wire-nodes**, with orchestrator contract review.
The [archived restart capture](history/follow-up-9/AT09-restart.json) contains three non-serving observations. At
`2026-10-31T12:06:32.527615Z`, reasons include both `safety_check_failed` and
`awaiting_member_grants`; safety is failed, and lease/qos/source_binding checks
carry `TC_FOREIGN_FILTER`. Its sole stage=lease error still carries
`AWAITING_MEMBER_GRANTS`. The neighboring reports at `12:06:32.473389Z` and
`12:06:37.574574Z` carry the expected AWAITING codes on all three checks.
There is no `lease_expired` report, and serving subsequently recovers.

The retained revision-11 assertion requires AWAITING codes for every observed
provisional report, so **`D-067 revision 11 provisional lease observation` fails**.
The composite **`same-generation D-067 restart recovery and lease sequence`**
check also fails through that predicate, despite matching generation, a new
populated lease, serving and counted traffic. These are the run's only two
failed checks; AT09 remains **fail**, with 36 pass / 5 blocked / 2 fail subchecks.

Please settle the intended check-code precedence when a real boot tc rejection
coexists with a provisional lease, against [node D-067 wording](../../../node/README.md)
and [central read-model semantics](../../../nodes/README.md). If the TC override
is intended, publish the mixed-state contract before narrowing this assertion;
otherwise correct the observation. The evidence alone does not establish
which owner should change. No expectation is weakened or old rejection erased
here, and this reporting gap is not presented as a renewed F8-01 outage.

### F4-05 / F4-07 — DNS wording retired; other gaps retained

DNS-wording part retired **2026-10-31**, per [dns-publish follow-up 8,
BACKEND.md §2.1.1–§2.1.4](../../../dns/BACKEND.md). Native assertions cite:

| Queries | Exact filtered response | Backend row / five-second profile |
|---|---|---|
| A direct / blocked alias / subdomain | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-A scopes D / C / W; F8-TTL5 |
| AAAA | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-AAAA; F8-TTL5 |
| Direct blocked CNAME | RCODE 0 NODATA, zero answers, SOA TTL 5 | F8-CNAME; F8-TTL5 |
| SVCB | RCODE 0 NODATA, zero answers, SOA TTL 5 | F8-SVCB; F8-TTL5 |
| HTTPS | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-HTTPS; F8-TTL5 |
| AT21 fallback A, AT22 blocked A sentinel / last-good | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-A (sentinel W); F8-TTL5 |

Every query retains observed RCODE, answer types, SOA presence/TTL; cache state
is separate. Positive and deliberately absent upstream controls are not
synthetic filtering rows. TXT/MX/NS (not queried here) are NODATA under F8-TXT /
F8-MX / F8-NS; no type used by this matrix lacks a table row. The AT22 unique
sentinel now sits under the already configured `lab.example.invalid` wildcard
block, so it is a genuine filtering control rather than an upstream NXDOMAIN.

Remaining owners: **devstack/native integration** for positive IDNA wire data
and production automatic nameserver distribution; **operator/platform** for
Private DNS/browser DoH. ConfigureNativeDNS distribution remains TEST ONLY.
No complete AT20–AT22 status is promoted by resolver-only measurements.

### F4-03 / F4-04 / F4-06 / F4-10 — resources still required

- **Platform/operator resources:** AT01 native Android/iOS/Windows/macOS,
  AT09 QUIC, physical second NAT/roaming/PMTU; exact existing procedures in
  [OPERATORS.md](OPERATORS.md) and [PROCEDURES.md](PROCEDURES.md).
- **Portal/devstack/operator:** AT04 rendered expired/suspended screens.
- **Packaging/devstack/operator:** AT05 preinstalled/cloud-init provenance and
  physical arm64. Cross-compilation supplies no physical-device claim.
- **Devstack/usage/database/backup:** AT22 raw-usage/rollup/file/backup retention
  and representative URL privacy. Commercial feed approval remains business
  work. Guarded audit purge and TEST ONLY feed last-good do not close these.

### Retired identifiers — 2026-10-31

F4-08 / F4-09: never defined; retired 2026-10-31. The follow-up 4 ledger is
F4-01 … F4-07 and F4-10. No replacement definition or owner is invented.
The separate worker RestartPolicy issue remains closed by #240.

### D-098 handover pins

The members8 claim test already reads immutable `tests/handover/testdata/`
snapshots, so its claims remain true without editing outside this lane.
Handover may adopt the now-published follow-up-8 archive and SHA256SUMS in its
own lane; no outside pin change is required for this rerun.

### Harness correction retained — 2026-10-31

[Interrupted attempt 1](history/follow-up-9/attempt-1/RUN.md) keeps all three
stale audit-selector failures and the read-only diagnostic. Current production
uses `member.grant_revoke`; the owned query now matches it. The final fresh run
passes all 122 AT02 checks without reducing the single-audit replay assertion.
No request to another lane is needed for this correction.

## Follow-up 10 pin request — 2026-10-31 (D-098, D-102)

### F10-01 — netlab attribution source pin, closed 2026-10-31

[Netlab 20, PR #336](https://github.com/TED-H-com/Freedom-VPN/pull/336)
corrects `docs/poc/G2-followup19-evidence/source-pins.json` to
[history/follow-up-8/AT09.json](history/follow-up-8/AT09.json), retaining SHA-256
`c89ba7b98285a0883889c83455ed24368e1b73859978fe11a1f008012d09ea3c`.
The corresponding attribution link is corrected on this baseline. Both files
were verified read-only. D-132 limits product-test pins to immutable `history/`
paths; living members root evidence may be replaced. The unchanged provenance
text in `internal/nodes/testdata/d126-mixed-observation.txt` names the original
follow-up 9 run; its original source is preserved in the follow-up 9 archive.
No outside-owned file or handover snapshot is edited.

## Follow-up 10 new failure — 2026-10-31 (D-102)

### F10-02 — AT02 finite-expiry withdrawal has an unreadable counter sample

Owners: **members harness / devstack counter reader**, with **agent-core / netlab**
if the missing read is traced to enforcement. Run `2be2d32f-bfd9-46f5-bf29-afaa6617ae94`,
harness `731be5a7`, retains **AT02 fail: 121 pass / 0 blocked / 1 fail**.
[`persistent withdrawal member-a3`](history/follow-up-10/F10-02-withdrawal.json) has 234 samples;
the one at `2026-10-31T13:35:45.333735552Z` has `read_ok=false` and counters
`Present=false`, while the transport status is running without an error.
The adjacent reads succeed. The current capture does not retain the counter
reader's error category, so its root cause is not attributed.

The finite deadline is `13:36:39Z`. Last TCP and UDP reply increments are
observed at `13:36:38.987828874Z` (12.171126 ms before expiry); senders stay
active, and final counters remain stable at 32,447 upload / 26,893 download
bytes. Genuine expiry to 0/3, its three audit events, restoration to 3/0 and
counted traffic pass. These observations do not establish a post-expiry leak,
but do not repair the missing read: `finishPersistent` requires every sampled
read to succeed, so **the failure is retained without a retry or relaxation**.

Please expose the bounded counter-read error category and correlate the failed
read with the original enforcement/read timestamps before attributing it.
[Archived AT02.json](history/follow-up-10/AT02.json) keeps the complete mutation evidence; the linked capture
copies the failed check and all 234 samples by JSON value. No product or
persistent-withdrawal assertion was changed in this follow-up.

**Follow-up 11 diagnostic answer, 2026-10-31:**
[Devstack 20](../../RESPONSES.md#members--2026-10-31-follow-up-20-f10-02--f10-04-d-102)
adds reader categories, independent status diagnosis and same-boot monotonic
enforcement brackets. Runs `218e18c9` (214 samples) and `977efa83` (247 samples,
494 producer joins to 242 trace records) have no failed read. The historical
`13:35:45.333735552Z` error cannot be recovered: **F10-02 remains historically
failed and unattributed**. Members 11 enables the observer before enrollment,
retains both reader diagnoses, rejects incomplete joins and keeps the existing
every-read-success predicate with no retry. Its new result is a separate row.

**New follow-up 11 measurement, 2026-10-31:** run `7edb263f` passes the
unchanged finite-expiry withdrawal check: **186/186 reads succeed**, counter
and status categories all `none`, both last reply increments at
`18:34:36.591993718Z`, **408.006282 ms before** the `18:34:37Z` deadline.
Senders remain active, counters stabilize, and sampling extends through the
original target-plus-four-second window. Across the two withdrawal checks,
**214 samples / 428 before-after producer records** join the **798-record**
gap-free stream, including renewals and explicit baseline/heartbeat sources.
[Full producer stream](AT02-enforcement-timing.json), [AT02](AT02.json).
The diagnostic-export request is answered; no failed reader sample recurs.
Full AT02 still **fails 121/0/2** on the separate F11-02/F11-03 checks below.
This does not repair or attribute historical F10-02.

### F10-03 — AT20 positive cache changes before the retained TTL, 2026-10-31

Owners: **devstack/native DNS fixture / dns-publish**, with the members harness
for query-path evidence. Run `2be2d32f`, harness `731be5a7`, retains **AT20 fail:
80 pass / 0 blocked / 1 fail**. The unchanged `positive/negative cache before
TTL` assertion fails; the per-type DNS filtering/RCODE/SOA/TTL rows still pass.
[Cache capture](history/follow-up-10/F10-03-cache.json), [complete AT20 evidence](history/follow-up-10/AT20.json).

The initial positive query at `13:52:11Z` returns the old `other_ipv4` answer
with TTL 10 (2,502 ms query duration). After the authority-only change, at
`13:52:12.96280019Z` the positive query already returns `private_ipv4`, TTL 10,
`expected_answer=false` (1 ms). The negative control remains NXDOMAIN with
SOA TTL 1. At `13:52:25.057177644Z`, both after-TTL expectations pass.
No cache flush or DNS expectation change was made.

The native two-resolver path does not identify the responding resolver per
query in this capture; the 2,502 ms initial latency does not by itself prove
fallback, cache bypass or a product TTL defect. Please correlate resolver
selection/cache receipts and the authority phase timestamps before assigning
cause. Preserve this failed before-TTL row. Positive IDNA, production resolver
distribution and platform requests remain open independently.

**Follow-up 11 diagnostic answer, 2026-10-31:**
[Devstack 20](../../RESPONSES.md#members--dns-publish--2026-10-31-follow-up-20-f10-03-d-102)
retains 36/36 complete query traces in `cce1a688`. Its initial positive and
before-TTL replies identify primary, old class, TTL 10 then 9; the latter has
no authority exchange (inferred hit only). After TTL, primary returns the
private class with TTL 10 and an observed authority exchange. Authority serial
1002 after the before-TTL query is a separate sample, not the cached A record's
serial. Negative policy cache state stays unknown unless separately attributable.
The historical **2,502 ms query and premature changed answer remain failed and
unattributed**; the missing responder cannot be recovered. Members 11 preserves
the unchanged assertion, the complete shared `dns_trace`, and the distinction
between native milliseconds and broader wrapper/query bounds. Multiple matches
stay ambiguous; no private AdGuard cache flag is claimed.

**New follow-up 11 measurement, 2026-10-31:** run `7edb263f` keeps AT20
**blocked, 81 pass / 0 blocked / 0 fail** on its resource boundaries, with the
unchanged cache assertion passing. All **39/39 native query traces** are
complete and uniquely identify **primary**. Initial positive: `other_ipv4`,
TTL **10**, observed authority exchange, separately sampled authority serial
**1001**. Before TTL at `18:50:32.944519920Z`: primary still returns
`other_ipv4`, TTL **9**, **no authority exchange** (inferred positive hit only),
while the authority sample is **1002**. After TTL: `private_ipv4`, TTL **10**,
observed exchange, authority sample **1002**. [Full AT20](AT20.json),
[cache trace](F11-AT20-cache.json).

Each positive native probe reports **1 ms**. Their broader retained query-call
bounds are approximately **42.558 / 33.577 / 40.232 ms**; the later
`authority_read_at` also shows separate trace-collection overhead. Negative
before-TTL remains NXDOMAIN / SOA TTL **1**, with SOA serial **1001** in the
resolver receipt while the separate authority sample is **1002**; its cache
state remains **unknown**. No private AdGuard flag or cached-A serial is claimed.
The trace-export request is answered, with no new premature answer in this run.
The original 2,502 ms query and premature answer remain failed and unattributed;
a passing rerun proves no historical TTL fix. Dns-publish 9 is not part of this
run's baseline and supplies no claim here.

### F10-04 — same-generation qualification, closed 2026-10-31 under D-129

The complete unchanged mutation order on baseline `d3e9a0f1`, harness
`731be5a7`, run `2be2d32f`, recovered at desired/applied **29 → 29**:
serving in **15.751594344 s**, first counted byte in **21.357052110 s**,
with **three provisional reports under D-126** and no `lease_expired`.
[Immutable capture](history/follow-up-10/AT09-restart.json) and
[complete AT09](history/follow-up-10/AT09.json) retain 38 pass / 5 blocked / 0 fail.

**D-129 (orchestrator decision, 2026-10-31): same-generation recovery qualifies
at whatever generation the unchanged mutation sequence yields; 30 is not a
contract value.** This makes the generation-29 recovery qualification complete.
[Devstack 20's F10-04 answer](../../RESPONSES.md#members--2026-10-31-follow-up-20-f10-02--f10-04-d-102)
confirms the hooks preserve the product-issued generation and add no mutation
to force 30. Its earlier exact-30 reconciliation request is superseded by D-129.
There is no forced generation, state repair or topology repair.
The original generation-30 F8-01 closure remains the immutable
[follow-up 9 run](history/follow-up-9/RUN.md). Recovery qualification does not
pass the complete AT09 case while any selector row remains blocked.

## Follow-up 11 new observation — 2026-10-31 (D-102)

### F11-01 — AT01 initial native path does not reach the node

Owner: **devstack / native integration**, with **netlab** for the first missing
transport hop if needed. Run `7edb263f-1f6e-4f94-9e8e-26099f96a699`, harness
`33abb2fd`, leaves AT01 **blocked**, 16 pass / 0 blocked / 0 fail subchecks,
plus its platform blockers. The existing forwarding gate retains F4-01's
missing-path blocker; the sleep/wake analogue and second transfer are not
executed after this prerequisite fails. [Complete capture](history/follow-up-11/F11-01-forwarding.json).

At `2026-10-31T18:27:29Z`, route selection changes from `eth0` to `fvnb0` in
the native `netbird` table. The member daemon is connected. The bounded
controlled-origin TCP/443 probe observes **five outbound packets at member
fvnb0, zero at node and origin**, zero origin requests and no device-byte
increments. Both node reads are available at **generation 5**, with six member
accept rules; all retained diagnostic counters are zero. The matching current
generation lease is retained in [AT01.json](AT01.json).

Please attribute the selected-route-to-node transport gap from this evidence;
a generation-0 baseline or node enforcement drop is not established. No retry,
route repair, topology repair or passing later transfer replaces this row.
The evidence-collection check passes because its capture completed; it does
not claim positive forwarding. The complete AT01 case remains blocked.

### F11-02 — D-077 transfer starts before populated-grant serving, 2026-10-31

Owner: **members harness readiness**, with **wire-nodes** only if the observed
empty-to-populated renewal schedule needs contract review. [Capture](history/follow-up-11/F11-02-capture.json),
[accepted observations](history/follow-up-11/F11-AT02-observations.json), [producer stream](history/follow-up-11/AT02-enforcement-timing.json).
Run `7edb263f` fails `D-077 re-grant counted forwarding` at `18:29:53Z`.
The prerequisite at `18:29:39Z` passes desired/applied/latest-lease generation
**10**, but retains **zero member grants**, forwarding false and
`AWAITING_MEMBER_GRANTS`. The current `forwardingGate` checks generation and
expiry, not populated-grant serving. The controlled transfer at `18:29:40Z`
therefore observes no receipt or byte increments, generation 10 → 0, and
`ctbase +300` bytes.

Producer sequence **269** accepts the empty generation-10 lease; **270** applies
it, and **275** records a successful agent baseline call with cause
`awaiting_member_grants`, bracket **54921971917713–54922010047959 ns**.
The accepted report at `18:30:09.584931Z` is subsequently serving. This supports
a premature harness prerequisite; it does not establish a producer defect or
an unauthorized forwarding leak. Please require actual populated-grant serving
before the future D-077 positive probe, preserving the existing transfer assertion
and this failed run. No readiness change, retry or rerun is made here.

### F11-03 — admitted sibling control meets generation-13 rollback, 2026-10-31

Owners: **agent-core / netlab**, with **devstack** for the shared
`ObserveMemberIngress` fixture. [Complete failed check and producer window](history/follow-up-11/F11-03-capture.json),
[accepted observations](history/follow-up-11/F11-AT02-observations.json). The strict ingress check at
`18:31:56Z` fails only its positive sibling control: the revoked source increments
`ctunku` by **1 packet / 29 bytes** with unchanged device counters and empty
uplink/origin captures. The admitted sibling instead increments `ctbase` by
**1 packet / 29 bytes**, with zero device upload delta and no uplink/origin packet.
Both injections report success using the existing tun-mirred mechanism.

The accepted generation-13 report is serving at `18:31:04.782079Z`, then reports
`HEALTH_CHECK_FAILED` / `stage=apply`, `safety_check_failed`, forwarding false
from `18:31:20.766966Z` through `18:31:56.065559Z`. The producer records failed
applies, `apply_rollback` and `verification_HEALTH_CHECK_FAILED` baseline calls.
For example, sequence **453** is a failed apply; **454** records successful
rollback at **55052609280751–55052943484892 ns**; **455–456** retain verification
baseline calls. Sequence **459** successfully applies the same generation;
the accepted `18:31:59.629518Z` report serves again.

Please attribute the failed Apply/fixture interaction from the original producer
records. The rollback writer is recorded, but the underlying health-check
failure is not identified by the sanitized `apply failed` report. These are
wall-window correlations with original same-boot producer brackets; the older
injection capture has no independent monotonic bracket, so no exact kernel
commit timestamp or injection-caused defect is asserted. Later recovery does
not repair the failed sibling control. No state/topology repair or retry is made.
