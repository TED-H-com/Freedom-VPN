# Isolation follow-up 8 — 2026-10-31

One assembled run: **`7779d83c-2f78-4699-ac88-8fcc59e8cf87`**, harness/image
source `dafb8f4d5b143dc1b425c471678d39a841033006`, product baseline
`62f24e6d` including agent-core #312 and netlab #308. The product tree is
unchanged. Assembled measurement: **11:23:35–11:39:26 UTC**, 950.65 s.
The live gate honestly returned **exit 2: lab=0, stack=1** because three
acceptance cases retain blocked scope. There are **no failed assembled checks**.
The recovery sequence was unattended: no agent restart, link raise, route/address
repair, observation write or topology-expectation override was used to recover.

| Case | Status / checks | Owner of remaining work | Deciding evidence |
|---|---|---|---|
| AT10 / ISO01, ISO02 | **blocked**, 8 pass / 1 blocked / 0 fail | Agent-core + devstack: D-087 clean provisional recipe | [AT10](AT10.json), [six recovery assertions](recovery-verdict.json); separate D-120 reload gate passes |
| AT11 / ISO01, ISO03 | **blocked**, 2 pass / 1 blocked / 0 fail | Devstack / platform operators | [AT11](AT11.json): native IPv6/platform matrix remains unavailable |
| AT12 / ISO01, ISO04, DNS03 | **pass**, 10 pass / 0 blocked / 0 fail | None for measured scope | [AT12](AT12.json): peers, management, designated DNS, spoofed source and group withdrawal |
| AT13 / ISO02, ENT02, SEC01 | **blocked**, 6 pass / 1 blocked / 0 fail | Acceptance / operators: full continuous fault/platform matrix | [AT13](AT13.json), [exact whole-node expiry verdict](deadline-verdict.json) |
| ISO-F6-01 | **pass**, all six G2 assertions | None for this assembled recovery reproduction | [verdict](recovery-verdict.json), [raw sequence](foreign-redirect.json), [timing and packets](foreign-redirect-analysis.json), [persisted Apply state](recovery-applied-state.json) |
| ISO-F6-02 / D-106 | **pass**, real clean/rejected wire evidence | None for this transport/digest reproduction | [539 retained reads](tc-evidence-index.json), [complete receiver](receiver-capture.json); 539 canonical digest matches |
| ISO-F6-03 / D-104 | **packet predicate pass; producer timing gap open** | Agent-core: exact Apply/render/install and executor read samples | [expiry analysis](expiry-timing-analysis.json), [packet trace](whole-node-packets.json), [precision/heartbeat reads](whole-node-continuous-reads.json) |

Spec coverage: §19 AT10–AT13, §8.2–8.3 and §16.3. Mechanism and assembled
results remain separate; no physical arm64 or complete platform qualification
is claimed. Arm64 helper cross-compilation passed.

## Whole-node expiry at true precision

The node-held generation-30 lease was issued **11:36:17.685283389Z**, expiring
**11:37:17.685283389Z**. Its persisted monotonic send anchor is
`30115198061283` ns, giving **D=`30175198061283` ns**. The retained heartbeat
agrees exactly with D in 149 reads; it later clears the deadline to zero.
The wall anchor, signed envelopes and node state are retained separately.
The renderer qualification on this same kernel measured **1 ms quantum** and
**100 ms installation bound**; the acceptance allowance is exactly **101 ms**.

| Assembled packet measurement | Result |
|---|---|
| Last positive egress | `30175098108464` ns = **D−99.952819 ms** |
| Last-positive margin against D−101 ms | **1.047181 ms** |
| First dropped ingress | `30175136293300` ns = **D−61.767983 ms** |
| First-drop margin against D+100 ms | **161.767983 ms** |
| Positives / drops | **161 / 41**, all 202 probes have ingress/read evidence |
| Positive at/after D or later than first drop | **0 / 0** |
| Independent node `any` capture | **685 packets, zero kernel drops** |

The phased 100 ms stream and arbitrary 37 ms offset stream share the explicit
D−101 ms boundary sample. Socket and packet readers retain IDs, monotonic send,
ingress, egress and read-completion times. These are userspace samples, not
hardware timestamps. No cadence allowance was added.

Before control loss, the [repeated-renewal measurement](continuity-analysis.json)
retained **two actual signed renewals**, **68 observed guard receipt intervals**,
and **1,396 probes: 1,379 echoes / 17 losses (98.7822% echo availability)**.
Every probe had node ingress; all 17 losses lacked uplink egress; AF_PACKET
reported zero drops. Every loss and its nearest guard receipt pair remains
retained. This 80 ms dual-phase diagnostic is **not the D-120 echo stream** and
has no D-120 tolerance applied to it. A passing final expiry boundary does not
prove loss-free renewal continuity.

Exact assembled Apply start/final-render/installed-by and executor precision-read
samples are not exported by current main. Periodic precision reads, actual
renewal transactions, heartbeat/watchdog events and passive netlink guard
receipts are retained; receipt intervals are **not command brackets or installation
bounds**. The initial recorder missed top-level nft JSON events. A read-only
recorder was attached before restart/renewal/expiry/recovery; the initial gap is
explicit, and no running product binary or test action changed. This remaining
producer instrumentation request does not turn measured packet failure into a
pass or erase the 17 renewal losses.

## ISO-F6-01 recovery and its failure control

Initial serving generation was **30**. The harness explicitly created the
foreign clsact/flower/mirred objects at
**11:37:43.872702246–11:37:43.918606726Z**. The real rejected `tc_evidence` read
completed at **11:37:44.137163225Z**, report
`01a191c0-581c-7ede-a9b7-a9050b491f6f`, with `TC_FOREIGN_FILTER` and verified=false.
QoS detail independently retains its live-read completion annotation.
Member-down/non-serving was confirmed by **11:37:44.298686009Z**:
**425.983763 ms** from injection start, an observation upper bound.

The newer **dirty generation 31** stayed non-serving and retained the foreign
objects through twelve snapshots. The agent did not delete them. The harness
removed the whole injected clsact at
**11:37:50.982759852–11:37:51.006897696Z**; fresh qdisc/filter reads prove removal.
The retained heartbeat still contains historical rejected rows, which are
explicitly excluded from the current-object absence check.

Clean generation **32** was requested at **11:37:51.059693316Z**. The first
verified read at **11:37:51.456907563Z** was a deny-all recovery model and was
still non-forwarding; it is not the serving proof. Link-up was observed by
**11:37:51.647616293Z**, **640.718597 ms** after removal. The first matching
populated-lease serving observation is `01a191c0-93c6-7b48-b608-fecdb644a311`,
observed **11:37:59.494731430Z**, fresh live read **11:37:59.484479000Z**.
Serving was sampled by **11:37:59.948010343Z**, **8.941112647 s** after removal.
The final counted control completed at **11:38:09.070190323Z**, **18.063292627 s**
after removal, admission bytes **181526 → 185076**. These include polling and
probe time; they are not production latency guarantees.

NetBird owns the native IPv4 configuration. The address remained while down,
and routes returned when the link came up; the reads do not distinguish kernel
reactivation from daemon reconciliation. No harness repair occurred. Persisted applied model, applied/lease generation
**32**, generation-32 member-path hashes, kernel generation, marker clearance,
serving heartbeat/observation and independent member/uplink packets all agree.
The full serving tc digest is
`aed914bd48372be8b76e9b092e830918704747b6b04b778caebfd99079e13fb5`, recomputed using
the exact netlab RFC 8785 projection. Historical rejected rows remain historical.
`objects_total` is not labelled a foreign-object count.

The node `any` trace has **2,405 parsed/reported packets, zero kernel drops**.
There were **four unsnatted redirected UDP/9 packets before quarantine**, plus
one ordinary SNAT packet during the injection command bracket. From confirmed
member-down to removal there were **zero uplink UDP/9 packets**. No blanket
zero-packet claim covers the injection window. All twelve rejected wire reads
belong to this labelled injection; **no unsolicited rejected object appeared**.
Its creator is this TEST ONLY harness. The creator of the unretained historical
follow-up 6 object remains unknown; this run does not invent an attribution.

## AT10 D-120 verdict

Netlab 18 was merged before launch. The actual
[printed verdict](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt)
is **AT10 PASS under D-120**: **100 reloads, 101 verified Applies, 59 datagrams,
zero UDP losses, zero TCP timeouts**, complete sidecars/captures, and zero
mismatches/unobserved ingress. Thus the per-loss table is empty; there is no
loss ID/generation/bracket to excuse. Maximum outer guard upper bound was
**47.066166 ms**; maximum render→installed was **28.231196 ms**, each below
100 ms. [All analyzer inputs and durations](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/at10-analysis.json)
are retained. This does not prove guard-related loss impossible or change D-104.

## D-087 and remaining scope

Agent-core 24's provisional recipe requires a fake-control
`SetMemberGrantsReady(false/true)` switch while keeping generation/publication
and desired bindings fixed. No assembled public API exposes that transition;
that item could not be run within this lane's ownership. No synthetic transition
or projector mutation was substituted. The receiver captured **544 observation
POSTs, zero qualifying ready/passed provisional reports**. Four HTTP 429
responses are retained. The historical 4,767 / 5,021 / 5,025 ms comparators stay
historical; this run supplies no new provisional duration.

The complete clone/freeze fixture and acceptance bodies remain retained without
expanding this report's claims. The continuous fault/platform matrix and exact
Apply timing exports remain open. See [dated requests](requests.md#follow-up-8-requests--2026-10-31).

## Archives, gates and cleanup

D-098 archived the corrected final records first: follow-up 6 run
`bf671f3a-c6ac-4824-9107-425c4c5a5f02` (**44/44 hashes match**) and follow-up 7 run
`041e9e48-79a6-45dd-a19d-8b5a63290655` (**52/52 match**). Handover already pins
these through archives/immutable fixtures; **no living-file claim pin moved**.
The current-file path in the D-109 rejection regression is not a file read.

Owned lint/test/build, handover tests, root `make go-lint`, formatting,
diff and committed-evidence hygiene checks pass. The sole live make gate
returns 2 for the documented blocked AT10/AT11/AT13 scope; the D-120/D-104 lab
suite passed in **22.307 s**, owned mechanisms in **19.962 s**.
[Validation output tails](validation-results.json) retain the actual results.
After teardown: **containers=0, images=0, networks=0, volumes=0** for
`fvpn-w4-isolation-*`; [inventory](cleanup-inventory.json).
