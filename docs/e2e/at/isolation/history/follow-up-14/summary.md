# Isolation follow-up 14 — 2026-11-01

One assembled run **`56d20530-8042-4d0c-8156-de5a47ce8b61`**, runtime
**`716a16335cb5335473416740c76a75ce95536796`**, source dirty **false**.
Product baseline **`9dde5eeb60a6cb6b9b10974c3100ef1ea1352698`**, including
#380, #382 and #377. Driver **1106.62 s**; full command **1242.433 s**, exit **2**.

| Case | Result: pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **pass: 9 / 0 / 0**, ISO-F6-01 **6 / 6** |
| [AT11](AT11.json) | **blocked: 2 / 1 / 0**, native platform matrix |
| [AT12](AT12.json) | **pass: 10 / 0 / 0** |
| [AT13](AT13.json) | **fail: 5 / 1 / 1**, D-104 fails; operator matrix remains blocked |

Spec §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04, ENT02,
SEC01, DNS03. Central hold evidence supports §5.1–5.2, §15.3, §16.3 / AT29.
This is an independent sample after `38074149`. No retry run, loss waiver,
changed assertion, timeout increase or historical promotion.

## D-098 and execution identity

First commit **`5e66101c`** archived all **112** run-38074149 root files
byte-identically in [history/follow-up-13](history/follow-up-13/archive.json).
Every file was compared with its baseline Git blob. The new manifest pins both
compressed and decompressed producer bytes, using follow-up 12's field shape.
Earlier archives are untouched. Handover's mutable-root strings are rejection
inputs; no D-132 one-file exception edit was needed.

[RUN.md](RUN.md) records the baseline, actual `go env GOCACHE`, commands and
instrumentation boundaries. [Preflight](host-preflight.json) retains `vmstat 5 7`:
six interval rows show runnable **1–3**, idle **74–85%**, I/O wait **8–9%**.
No other suite tree was present at initial admission; one members suite was
present after stack startup. Admission was checked before both stacks and
after startup. No third suite was started. These are observations, not causes.
[Cleanup](cleanup-inventory.json): containers, images, networks and volumes
under the owned prefix are all **zero**.

## Central hold: exact join passes, requested extended exercise is partial

[Original central export](generation-holds-central.json.gz),
[read-only review](generation-holds-review.json), and
[operation history](publication-operation-history.json) retain one accepted
hold. It is an **empty-member bootstrap lease**, not a populated serving-path
hold. The exact IDs are:

| Evidence | ID / value |
|---|---|
| Accepted report | `01a196fd-f5e3-7211-89dd-2bb4b6b373d2` |
| Signed lease | `a0959dd8-d2bf-4592-9850-5a0a660ac264`, generation **2** |
| Applied generation | **3** |
| Node's last delivered signed desired generation | **3**, original receiver transaction **9** |
| Central desired generation at later poll | **5**; explicitly not the receipt-time value |
| Incident | `01a196fd-f5e9-7e03-933e-de3166035e0d` |
| Per-report sample | same node/incarnation/report ID, joined to that incident and lease |
| First / last seen | both **12:03:08.392762 UTC** |
| Summary | steps **1**, elapsed_ms **0**, alarm **false**, valid **true** |
| End | **12:03:09.449702 UTC**, `generation_matched` |
| Matching receipt | `01a196fd-f9ff-7327-8feb-bf74771969b9` |
| Escalations | **0**, no alarm/escalation threshold exercised |

The report carries `policy_missing` and lease-stage `STALE_GENERATION` and
forwarding=false. Applied **3 == node desired 3** is observed against the
original signed response completed before the report. The later central poll's
5 is retained separately. `PublicationDataPathFailure` returns
**`stale_generation`** in original and reversed check/error order. The reversal
is a labelled read-only evaluation of the same report, never another ingestion.
The dependent failed qos/source_binding shape was not present in this bootstrap
report, so that specific precedence shape is not a fresh live proof.

Producer process `8ba0e32d-25b3-4152-b58a-88ba49f1d5e3`, sequence **15**,
records a successful `policy_missing` baseline with that lease. Sequence **16**
records `baseline_ok=true`, generation 3, `serving=false`. No independent member
packet was captured during this empty-member hold, so counted forwarding
closure is **not independently exercised** for it. A reported flag or a failed
counter read is not substituted for that missing packet evidence.

All seven retained publication completions were checked against incident
intervals and stored references. No successful publication completes in the
hold interval or credits its report. The generation-3 publication completes
at **12:03:09.545896 UTC**, after the exact matching-lease receipt ends the
incident. The operation observer began later during this same assembled run;
these initial completions are retained database state, not earlier continuously
observed transitions. Baseline `agent_policy` ACK and data-path confirmation
remain separate.

[All producer brackets](policy-missing-analysis.json) contain **21**
`policy_missing` windows, **17** populated old/new-generation windows.
Every one is enumerated in the central review with its old lease ID and
producer sequence. **None of those 17 has an accepted hold report**, hence none
has a per-report sample or incident ID to invent. Their baseline→next successful
Apply spans are **201.545605–257.694360 ms**, with later verification and serving
heartbeat retained. One bracket (sequence **1534**, lease generation 30→31)
has five captured renewal ingresses and four losses. The other 16 have no
captured ingress. Missing coverage is unavailable, never zero loss.

The [single control-cut attempt](generation-hold-control-cut.json) sent a real
capacity PATCH at **12:19:12.562216187 UTC**, received **202**, and found no new
accepted open hold within its one 15-second polling window. It therefore did
not pause control or invoke the extra stop. No seam, transport interception or
second attempt was added. **Alarm/escalation, active-incident stop retention
with local-counter reset, later distinct incident, and live unrelated-failure
precedence are not exercised.** The accepted ISO-F6-01 rejections do not carry
the hold marker, so they cannot supply a same-incident precedence proof.
Kernel reboot is **not exercised**, under the carried AT13 operator matrix.
The initial ended incident remains in final history after ordinary restart and
terminal stops; that does not qualify active-hold retention or reset counters.

## F14-01: whole-node D-104 remains failed

[Verdict](deadline-verdict.json) and [compact finding](follow-up-14-findings.json):
**160 positives / 42 drops**, last positive **D−162.220103 ms**, first drop
**D−97.638039 ms**. The unchanged failure is **last positive earlier than
D−101 ms**. Near-deadline counted forwarding and post-deadline ingress denial
pass. No positive at/after D was observed. Two pre-D losses remain counted.

Boundary packet **161** was scheduled at D−101 ms, sent at **D−100.700360 ms**,
and captured at ingress at **D−97.638039 ms**, with no egress. That ingress is
inside the allowed conservative transition. The evidence does not establish
a drop before D−101 ms or the exact kernel-drop cause; it also does not meet
the last-positive predicate. There is no scheduling allowance or retrospective
promotion. Agent-core/netlab/orchestrator review is routed in
[request 3](requests.md#follow-up-14-requests--2026-11-01).

Renewal has **1396 probes / 1381 positives / 15 losses**, two actual accepted
renewals and **69** completed guards. **11** lost ingresses overlap completed
outer guards; **4** (packets 15–18) are outside. They lie in the sequence-1534
baseline→Apply bracket; outer baseline-call overlap is not an exact kernel-drop
attribution. All remain losses and all original command joins are retained.
D-135 / corrected D-136 require later recovery and provide no outage ceiling.

## AT10, D-087, D-120 and command evidence

D-087 passes at fixed generation **15**, with byte-equal invariants, natural
signed leases, actual accepted reports, **286** original wire hashes and **35**
desired responses. True→accepted serving is **2017 ms**; independently counted
bytes **2669→5299**. Earlier samples remain independent.

ISO-F6-01 passes all six unchanged assertions. Readiness took **4.549314022 s**
within the original 90 seconds. Three dirty samples remain within the original
twelve-sample limit. Accepted transactions **1566 / 1567** at
**12:19:38.809585899 / 12:19:38.866412402 UTC** carry dirty candidate **34**,
ACK **33**, forwarding=false, `TC_FOREIGN_FILTER`, `stage=apply_candidate`,
`verified=false`. Both join desired transaction **1564**, producer process
`665f4c6c-2768-45d5-98c3-ca4ddecb185a`, Apply **2533**, original commands
**0–36**. This is pre-install evidence, with no installed verifier substituted.
Clean recovery generation **35** retains its separate Apply/live verifier,
digest, topology and packet control.

[D-120](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt)
passes: **101** verified Applies, **64** datagrams, **0 UDP losses / 0 TCP
timeouts**. Its allowance is unchanged and does not cover renewal or expiry
losses. The separate mechanism D-104 passes; it does not replace the assembled
whole-node failure.

**8132** contiguous producer records, **980** Applies, **one** pre-install
failure, no observed install overrun. All **980** per-Apply command/lifecycle
contexts are available. **55164** original command records, **55157** ok;
the other seven are conntrack exit-1 results within successful Applies.
D-106 digests **663 / 663** recompute and match. Exact child starts and hidden
host ancestors remain unavailable; cancellation callbacks observed=0.

Netlab 23's resource omission is explicit on all **979** install commands,
before and after: `skip_reason=install_bound`. No resource sample is inferred
there. Command timestamps, exits and process CPU/switch fields stay available.
[Command summary](command-lifecycle-summary.json) keeps every failed record and
original indices. The complete stream is lossless gzip with
[immutable two-form pins](history/follow-up-14-stream/archive.json).

Nearest-rank milliseconds over **979** completed guards:

| Measure | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer | 31.128482 | 40.689136 | 49.218688 | 52.588252 |
| Install | 15.849010 | 21.898948 | 27.440421 | 31.241963 |
| Render→installed | 15.953637 | 21.986037 | 27.534820 | 31.329338 |

The prior 951-guard sample remains in [follow-up 13](history/follow-up-13/summary.md):
outer p50/p90/p99/max **31.476965 / 40.656124 / 48.912754 / 55.460784**;
install **16.201976 / 22.493861 / 27.189380 / 29.107159**; render→installed
**16.302182 / 22.578520 / 27.277997 / 29.208894**. No A/B cause or historical
ceiling is inferred.

## Process stops and F13-01 counter-read decision

[Direct terminal samples](agent-core32-regression.json): source/clone
SIGTERM→exit upper bounds **7.725929 / 6.522051 ms**, both below 13 seconds,
with stored lease IDs and original anchors preserved. Exit status is
unavailable. M-1 remains **not observable here**: Step cancellation and
agent-side POST completion are not exported.

The clone fixture's Docker PID1 stop is separately **20.053570446 s**, followed
by SIGKILL/exit137. Later teardown events remain separate. Ordinary restart's
signal/exit events are **unavailable**: the observer began afterward and daemon
backfill did not retain them. Its real stop/start brackets, lease survival and
counted recovery remain in [restart-lifecycle.json](restart-lifecycle.json).
The final runner starts lifecycle capture before the driver on future runs;
that wiring was added after this run and does not repair its missing events.

Each current AT record now carries the counter-read interpretation from
[netlab 23](../../../poc/G2-followup23-evidence/decision.json):
`*enforce.DuplicateChainError` / **`duplicate_chain`**, **0 / 3,300** reproduced,
origin **unproven**. The [experiment](../../../poc/G2-followup23-evidence/reproduction-summary.json)
never licenses a retry or acceptance change. A read failure cannot prove
counters frozen or removed, or forwarding closed. Untyped `parser_rejected`
rows are not relabelled as typed duplicates. Every historical failure remains
failed in its archive.

Six text captures had terminal blank lines normalized; exact originals are
retained privately. JSON/JSONL, packet records and printed verdicts are
unchanged by that normalization. Large original producer and central exports
are lossless gzip with SHA-256 for both forms at immutable history paths.
[Validation](validation-results.json) records required gates; the one assembled
failure is retained. Four hand-offs: two carried operator rows, F14-01, and the
missing controlled populated-hold exercise. No production/shared-harness change.
