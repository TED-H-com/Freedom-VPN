# Follow-up 17 — operator evidence at the #313 cutoff

Verified against main `7e044c71ecf1d07f63e89721bd213db6611a6a47`, 2026-10-31 EDT.
This cut reconciles #304–#313, including the intervening #310 and #311.
Handover follow-up 16 (#306) stopped at #303, recorded D-112–D-119, and left
#304 for this cut. Its dated failures remain historical failures.
`TestHandover_Followup17Claims` pins the roadmap rows, owner snapshots and
run-ID records under D-098 / D-109. Handover ran no new live acceptance.

**2026-10-31 08:17 EDT resume:** security-ops follow-up 13 (#316) landed
after this reconciliation through #313. Its SECOPS-21 closure and AT30
measurements are reserved for reconciliation in handover follow-up 18; see
the [dated owner handover](../e2e/at/secops/2026-10-31-follow-up-13-handover.md).
The claims below retain their original #313 cutoff.

## Installer scope and isolation recovery (§6.3, §8.2, ISO01, AT07, AT10, AT13)

Netlab follow-up 17 (#304) proves D-116: a hookless `inet fvpn_install` is
invisible to the four-chain `inet fvpn` verdict, for both successful and
broken-owned-ruleset controls. A hooked **drop** chain in the marker table is
still rejected; an extra chain inside `inet fvpn` still gives
`HOOK_PRIORITY_MISMATCH`. This is no installer-name exemption and no blanket
ban on harmless foreign hooks. The real agent reaches serving with the marker
present before enrollment. nft 1.0.9 needs text read-back to see table comments.
See the [dated scope proof](../poc/G2-installer-marker-2026-10-31.md).

Agent-core follow-up 24 (#305) breaks the ISO-F6-01 circular latch: retained tc
failure blocked ACK / renewal, while the stale lease blocked the verified
Apply needed to clear the failure. A deny-all candidate permits fresh
verification without reusing old grants; same- and newer-generation retries
both need that verification. Its real-kernel measurement is injection →
contained **897 ms**, removal → clean link-up **494 ms**, removal → counted
serving observation **3.161 s**, applied and lease generations **3**. These
are measured observations, not service guarantees. The original foreign
object's creator remains unproven.

That delivery also includes revision 13 observation transport with bounded
`qos` ReportDetail and D-106 `tc_evidence`, a durable FileMeter checkpoint
after committed usage and atomic fsynced publication, candidate release
identity tied to the running executable/current pointer/journal, the D-087
provisional transition with Retry-After and bounded jitter, and r16 M-1 durable
freeze recovery across failed persistence, restart and lost-response replay.
The [handover](../node/followup24-handover-2026-10-31.md),
[isolation](../node/followup24-isolation-2026-10-31.md) and
[updater](../node/followup24-updater-2026-10-31.md) notes retain those boundaries.

At #305, the unchanged privileged `TestD089_RealSeedThenJournaledUpgrade`
consumed the real checkpoint but failed candidate health / rollback because
the w5-control fixture returned 404 for desired state and lease. That failure
was routed to installer follow-up 5; D-104 continuity was deferred there.
#311 later passes the isolated installer suite **13/13**, with signed health
fixtures and a real TCP probe, including a stopped-probe failure control.
Its NetBird management status is TEST ONLY. This is harness-proved local
journaled upgrade; a serving-node updater handshake is **not yet proved**.

## AT10 continuity and IPv6 posture (§8.2–§8.3, AT09–AT13)

The [#304 reload measurements](../poc/G2-at10-reload-stream-2026-10-31.md)
record three UDP timeouts over two 100-reload runs. Every loss has node ingress
overlapping a verified Apply's guard-install bracket; zero are outside any
bracket. Guard upper bounds were approximately **18–55 ms**, with
render → installed **≤ 29.5 ms** (29.520 ms before rounding). None proves an
exact kernel drop instant. **AT10 failed under the pre-D-120 rule** in both
runs; the separate successful real-agent measurement does not erase that.

D-120 is an **AT10 test-acceptance definition, not a product guarantee**.
Netlab follow-up 18 (#308) implements pure loss classification. A UDP loss is
tolerated only when node ingress is inside / boundary of exactly one verified
Apply's outer guard bracket (install-command start → release-command end,
2 µs margin), the bracket upper bound and render → installed are each
≤ 100 ms, the next datagram succeeds, the entire run has zero TCP timeouts,
and there are ≤ 5 UDP losses per 100 reloads. Missing sidecar / trace / Apply
records, capture drops, count mismatches, ambiguous matches, outside or
unobserved losses and terminal/consecutive losses fail. The echo timeout and
AT13 revoke assertions stay unchanged. **D-104's 100 ms installation bound is
unchanged** and distinct from D-120's outer-bracket continuity condition.

**AT10 PASS under D-120** in both new runs: the full lane has **one loss among
48 datagrams**, at generation 1039's boundary inside one approximately
**25 ms bracket** (25.528070 ms upper bound), followed by a successful
datagram; the focused run has **zero losses**. Each has 100 reloads / 101
verified Applies and zero TCP timeouts. This does not close assembled D-104
continuity or isolation's outstanding AT13 rerun.

The [F8-02 lab](../poc/G2-f802-mtu1280-2026-10-31.md) completes IPv4 transfers
at MTU 1280, with fragmentation-needed errors traversing the ruleset and tc
in both directions. No missing IPv4 allow rule or required MSS clamp is
demonstrated. IPv6 forwarding hits `node/enforce/render_nft.go`'s two
`meta nfproto ipv6 … ctv6 drop` rules; `node/enforce/validate.go` accepts only
`IPv6Block`. **D-123: the v1.0 overlay is IPv6Block by design**: no IPv6
forwarding through the node and no end-to-end IPv6 MTU / PMTU acceptance row.
IPv6 overlay support would be separately scheduled work, never deletion of
the guard rules to make a test pass. Native IPv6 **underlay** evidence is a
different question and remains applicable. The members-8 **11,011-byte**
partial reconnect transfer remains unattributed; later native captures below
still do not establish its cause.

## Updater receipt and checkpoint boundaries (§13.5–§13.11, UPD05, AT23–AT27)

Updater follow-up 15 (#307) binds D-110 receipts to the central
`internal/installerjob` ID and verifies them against the catalog. D-114 gives
an unaccepted legacy receipt one real `job_id_mismatch` 409 and then its
replacement; the mismatch becomes a terminal diagnostic row, not an endless
resend. Accepted legacy receipts retain stored replay semantics; unknown seal
members stay byte-identical. F9-02 makes extraction deterministic under any
umask: directories 0755, archive file bits, zero mode → 0644, fsync after chmod
and final-root symlink refusal. F9-03 makes `apply` exit **1** on sealed
`CHECKPOINT_FAILED`, with exit **2** reserved for the forwarding-blocked
sentinel. Read the sealed outcome as well as the process exit.

The real updater ↔ central receipt interoperability test runs on PostgreSQL
through `updater-integration` and its workflow job. These are component and
isolated-database proofs, with no new live updates acceptance run or physical
arm64 result. F9-05 lower-sequence refusal was located in
`packaging/installer/upgrade.go` and routed to installer follow-up 5; #311
implements refusal under the shared lock before staging, including after a
journaled switch. Its local success does not replace live updates acceptance.

Updater follow-up 16 (#310) adds a bounded FileGate → FileMeter wait with
durable deadline/boot/latch identity and restart checks. Uncorrelated
checkpoints still fail closed with a null checkpoint and no service switch.
Fresh correlated success is **not yet proved**, pending updater 17 and
agent-core 26 `barrier_id`; the live failure below was measured on an earlier
base and is not a new measurement of #307's exit-code fix or #310's wait.

## Agent activation and acceptance scope (§10.5, §16.3, ENR02, MTR03, AT09)

Agent-core follow-up 25 (#312) consumes D-104 absolute deadlines through
`MemberLease.DeadlineMono` and paired `Model.NowMono`, removing the whole-second
gate/cap while retaining final expiry and heartbeat withdrawal. Its kernel
trace has last positive **D − 100.65 ms**, first dropped send **D − 0.72 ms**,
and no positive at/after D. Guard-inclusive availability is
**74,796/76,000 probes (98.4158%)**, across two renewals and 75 Applies;
all losses correlate with guarded brackets (median 31.5 ms / worst 55.2 ms).
This is a **TEST ONLY 1 s Apply cadence**, not production availability or
uninterrupted forwarding. D-104 assembled continuity remains open.

D-106 now sends authoritative `TCReadbackEvidence()` from Apply / live
verification / blocked audits, null before any retained read, without clearing
historical rejection rows. F8-01 same-generation restart recovers in
**1.419 s** from object removal at generation 2 through the deny-all candidate;
no generation bump or fabricated ACK. Retained Apply errors now compare lease
renewal with the valid desired generation. This is harness-proved recovery;
members' generation-30/full-history F8-01 acceptance is **not yet proved**.

**D-118 is active** in `node/agent/usage.go` via
`RetirePreviousAt: e.clock.Wall()`. Same-node re-enrollment preserves the genuine
**4,096-byte** final sample, uploads the retained stream first with
`origin_incarnation_id`, uses independent ACKs and restarts the current stream
at sequence 1. An unchanged incarnation retires nothing. This TLS/SQLite
agent proof does not replace central D-112 eligibility/window checks or
assembled retired-stream ENR-05 acceptance. See the
[agent's dated note](../node/2026-10-31-d104-d106-agent.md).

## Devstack 18: harness-proved versus not yet proved (#313)

The [published outcomes](../e2e/runs/2026-10-31/followup18/README.md) preserve
each attempt and its source commit. D-102 answers are in
[RESPONSES.md](../e2e/RESPONSES.md): retire through the real API, re-enroll the
same Node, save the origin snapshot, read retired ACKs from a coherent stopped
SQLite/WAL pair, and select central checkpoints by origin incarnation.
The current stream's CLI ACK is not the retired ACK. That answer's historical
“hook remaining” paragraph describes its base; #312 now activates D-118.
The fixture fetcher view recommends a compiled TEST ONLY transport to a
selected isolated origin, preserving real verification; it is not implemented
and does not authorize public hosting or a production SSRF relaxation. Portal D093-12
still needs fixture migration; the audit remains 13 open / 2 retired.

| Measured row | Harness-proved | Not yet proved / owning rerun |
|---|---|---|
| D-116 installer / ENR-05, run `09cb1198-435f-4609-9d0b-d53e6c1ed441` | Real installer node serves; installer rerun migrates the legacy marker. Held usage is 1,376 upload / 1,001 download bytes at sequence 2 / ACK 1; wrapper ledger/CLI bytes match; services and traffic restore; release catches up to sequence/ACK 3. **ENR-05 passed in this harness.** | This is same-incarnation preservation, not D-105 retired-stream delivery. Enrollment follow-up 8 must rerun the five AT07 rows. Full campaign **13 pass / 0 blocked / 1 fail**. |
| Serving updater, same run | **FAIL**: `apply` exits 0 while sealing `final_sample / failed / rolled_back`; checkpoint absent and current release unchanged. | Serving-node handshake **not yet proved** → updater 17 / agent-core 26 `barrier_id`. A process exit 0 is not success. |
| Quiescent candidate, run `8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179` | Explicit agent stop creates a genuine checkpoint; all five candidate checks and sealed local success pass. | Does not prove the serving handshake. Full campaign **13/0/1**. |
| Post-upgrade re-enrollment, same run | **FAIL**: `baseline_failed`, seed **HASH_MISMATCH**, ledger retained. | Successful post-upgrade re-enrollment **not yet proved** → installer 6 / updater 17. Earlier pre-upgrade reinstall success is a separate row. |
| SECOPS-21 first offer, run `5f74d7c3-6404-461d-9f47-296bd69958c2` | **2/0/0 passed**: first delivered generation **4 > retained 3**, HTTP 200, exact local digest and accepted/applied generation 4, central **202** receipt joined by report ID. | Full restore **not yet proved** by this hook → secops 13. **SECOPS-21 remains open**. Earlier first-304 failure is retained. |
| Native restart, run `daf790fc-81f7-49a2-8a65-f3ec8faec656` | `RestartNodeNamespace90` captures unchanged generation **3 → 3**, fresh serving after **5.96 s**, without tc repair or state clearing. | Generation-30/full-history F8-01 remains for members 9; kernel boot identity did not change. |
| Native MTU 1280, runs `2454c833` and `daf790fc` | **FAIL** in both selected underlay families across attempts, including a standalone failure; `ObserveSelectedNativeExit` and bounded capture exports identify the actual family. Neither family always fails. | Native F8-02 and root cause **not yet proved** → netlab 19 / wire-nodes 32. D-123 blocks IPv6 overlay forwarding; it does not erase native underlay failures. Native campaign **9/0/2**, including its retained first-304 failure. |

`ArmFirstDesiredOffer`, `ReleaseDesiredOffer` and `FirstDesiredOffer` capture
actual delivery, matching local acceptance and its applied receipt. A compiler
row, later 200, or unrelated ACK cannot replace the first response. The
`journal-copy` export includes SQLite and WAL coherently. These exports enable
owner reruns; they do not close their cases by themselves.

## DNS answers (§12.4–§12.5, DNS01–DNS04, AT20–AT22)

DNS-publish follow-up 8 (#309) corrects the previous uniform-NXDOMAIN wording.
With the pinned AdGuard Home and `blocking_mode: nxdomain`, blocked
**A / AAAA / HTTPS → NXDOMAIN**; blocked **CNAME / SVCB / TXT / MX / NS →
NODATA** (RCODE 0, zero answers, SOA in authority). The synthetic SOA TTL
honours `blocked_response_ttl`: **60 s** in the production-body measurement
(configurable starting default; production choice remains pending), **5 s**
in the fixtures. An unblocked earlier CNAME hop can still be returned; querying
the blocked hop prevents completion to an address. No uniform NXDOMAIN or
instant cache-eviction promise follows.

The invariant is **fail-closed for answers, not uniform NXDOMAIN**. The dated
per-type tables and answers are in [BACKEND.md §2.1.1–§2.1.4](../dns/BACKEND.md#211-expected-responses--measured-2026-10-31);
the CI conformance probe retains **432 exchanges**. This resolver measurement
does not promote native/platform AT20–AT22 acceptance.

## Remaining acceptance

Still open: SECOPS-21 full restore (secops 13); AT09 F8-01 / F8-02 (members 9,
netlab 19 / wire-nodes 32); the five AT07 rows (enrollment follow-up 8);
retired-stream usage acceptance (usage 5); D-104 assembled continuity / AT13
(isolation 8); AT31 candidate hardware / mixed relay and AT29 guest-clock
rows; real Windows, macOS, iOS and Android clients; physical arm64. Cross-builds,
component tests, smaller harness passes and the new AT10 definition do not
close those rows.
