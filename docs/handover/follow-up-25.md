# Follow-up 25 — located drop, exit-status gap and r18 review

Fixed `main` cutoff **9cd7a0531819d17163d81c524bc5539bb3edc014**, through
**#373**, checked for the 2026-11-01 EST handover. This cut includes every PR
**#370–#373** after the previous #369 cut. Nothing was merged after the cutoff
at task start. **Agent-core 32, members 14 and devstack 25 are running**;
no results from those lanes enter this cut. Handover ran no new live acceptance
and adds **no new operator guidance**.

**D-098 / D-109 / D-132:** [four exact roadmap rows](../../tests/handover/testdata/followup25-main.json)
and [15 immutable snapshots](../../tests/handover/testdata/followup25-snapshots.json)
pin the cut in `TestHandover_Followup25Claims`. Snapshots retain source commit,
full-source digest, byte offset, length and excerpt digest. Updates 13 and
isolation 12 root records are living roots until their next owner archives;
this cut cites their roadmap rows / PR hand-offs, never reads or copies those
roots. Owner archive and devstack capture inventories are checked separately.
Current secops scope-review digests are live owned-document integrity, not
frozen acceptance evidence. Earlier claims keep their assertions through
frozen-input redirects.

## Devstack 24 — #370

[F11-01 analysis and owner options](../e2e/runs/2026-11-01/followup24/analysis.md),
with evidence pinned by `docs/e2e/followup24-captures.sha256`.
In **cacad79c**, the node's **`::1`-source handshake response reaches the member's
netdev ingress and is dropped before IPv6 prerouting**: ingress 1, prerouting 0,
input 0; **Ip6InHdrErrors 0→1**. This is **consistent with kernel receive
validation of a loopback source on a non-loopback interface**. There is **no
drop tracepoint**, so the kernel cause remains a consistency statement, not
proven instruction-level attribution. No nft policy drop or WireGuard roam
to `::1` is demonstrated.

That same request **succeeded with one SYN in 9 ms**; type-4 data came first.
The old IPv6 **five-SYN/no-type-4 timeout did not reproduce in four completed
attempts**. A separate IPv4 first request, **d556673b**, timed out with **five
SYNs**, despite type-4 traffic. It stays **failed and unattributed**; the located
IPv6 boundary does not explain it. Earlier instrumentation failure is retained.
No historical failure becomes a pass and no natural reproduction rate is inferred.

**Members F13-01:** an **exit-0 parse rejection** now retains the **raw bytes and
exact parser error privately**, in mode-0600 files, exported before teardown.
Public evidence carries a **digest**, size, line count, structural class and
error kind. Regression proofs on main:
`TestAT04_ParseFailureEvidence` and `TestAT04_ParseRetentionFailureAndLines`
in `tests/e2e/internal/counterread/read_test.go`, plus the fixture
`TestE2E_Followup24Native` in `tests/e2e/followup24_live_test.go`.
The old failed bytes cannot be recovered; **cause unproven**, no parser defect
or parser repair inferred. Members' failed AT04 rows stay failed.

The repair direction for F11-01 remains an **owner decision**: an **upstream
NetBird fix and a qualified pin**, **userspace WireGuard against the agreed
kernel baseline**, or an **IPv4 preference** with defined fallback. Changing
the proxy factory alone is not an established fix; the analysis retains costs
and risks. **NetBird 0.82.0 stays pinned**, no repair option applied. Devstack 25
is assigned first-selection and userspace-WireGuard evidence; that is pending work.

## Handover 24 — #371

The [previous cut](follow-up-24.md) covers **#363–#369 at 5bd75102**.
`TestHandover_Followup24Claims`: **21 subtests / 7 rows / 21 snapshots /
357 file checks**. Its exclusion of #370 remains true for its historical cutoff;
#370 belongs in this cut. The previous register and entrypoints are frozen;
a whole-function hash verifies that only their input redirects changed.

## Updates 13 — #372

D-098 archive of **2babf9b6**, **44 root files**, under
`docs/e2e/at/updates/history/follow-up-12/`, checked by
`TestD098_Followup12ArchivePins` in `tests/e2e/at/updates/archive_test.go`.
Run **759d244a on #369's tree**: regular **AT23–AT27 5 / 5**, installer
**27 / 27**. All **11 updater agent stop commands returned success**, with
**the stored lease kept** at every stop and start. Five observed process exits
have status 0; five calls find the agent already stopped. **AT23's first agent
exit is unobserved**, reaped before the observer saw its wait status.

This is **updates F13-01**, distinct from **members F13-01**. A successful stop
command is not a complete process-exit observation; no agent/updater defect or
universal graceful-stop proof is inferred. It routes to **devstack 25's TEST ONLY
exit-status / SIGKILL readback**. Main's
`TestUPD05_StopObservationDistinguishesExitKillAndReap` in
`tests/e2e/at/updates/stopobserve/main_test.go` checks exit, kill and already-reaped
states. The old 2babf9b6 direct stop durations and exits were not captured;
stage brackets cannot replace them.

The **pre-test port collision** remains under
`docs/e2e/at/updates/history/follow-up-13-startup/`: startup exit 2, no regular
or installer test started. The **configurable port slot** keeps default 42000;
the single complete AT invocation used 54000. No failed AT predicate was retried.
See [#372 hand-off](../../tests/handover/testdata/history/follow-up-25/updates13-handoff.md)
and [owner routing](../../tests/handover/testdata/history/follow-up-25/updates13-routing.md).
These measured AT results do not resolve r18's worst-case shutdown finding.

## Isolation 12 — #373

D-098 archive of **edd986c4**, **103 root files**, under
`docs/e2e/at/isolation/history/follow-up-11/`; main's
`TestD098_Followup11ArchivePins` in `tests/e2e/at/isolation/producer_timing_test.go`.
Run **1c25ee44** on #369's tree:

| Case | Result at this cutoff |
|---|---|
| AT12 | **PASS 10 / 0 / 0** |
| AT10 | **BLOCKED 8 / 1 / 0** on **ISO-F6-01 assertion 6** |
| AT11 | **BLOCKED 2 / 1 / 0**, native platform/operator rows |
| AT13 | **BLOCKED 6 / 1 / 0**, continuous fault/platform/operator rows |

**Kernel denial held** in all twelve dirty samples, but **no generation-30
TC_FOREIGN_FILTER report reached central**. The generation-29 rejection was
accepted after removal. Local failed Applies and later recovery cannot replace
the missing sixth assertion. **AT10's earlier pass (#360) is history; this run's
block stands**. The observation gap routes to **agent-core 32**, with a later
independent isolation sample after its fix/readiness boundary. No forbidden
forwarding or causal regression is inferred from the missing report.

The consumer joins the **per-Apply `commands` records**: **956 Applies / 53,754
commands**, seven failed pre-install Applies with cause unproven; **949 completed
guards**, outer p99 **49.8 ms**. The **producer stream is a lossless gzip**.
Fifteen stale-lease windows follow corrected D-136; their losses were not
captured and must not be called zero. These results are pinned by #373's row
and [hand-off](../../tests/handover/testdata/history/follow-up-25/isolation12-handoff.md),
not the living producer stream. [Routing](../../tests/handover/testdata/history/follow-up-25/isolation12-routing.md)
keeps AT11 / AT13 operator resources separate from the observation gap.

## Adversarial review r18 — archived, findings open

The [raw report](../reviews/r18-generation-change-review.md) is **byte-for-byte**
`origin/r18-generation-change-review:.agent/REVIEW.md`, commit
**1a65598fe6ebc5ba394284857ece59d0a572458b** (review base #369 / 5bd75102).
SHA-256: **f266a0705292975f5416bda5a082a3f7e779f45e82cb9b9ce54843ce0cb8e33d**.
No review proof files are adopted here; its proof names belong to the review
branch, not main acceptance. [Review dispositions](../reviews/README.md#r18-dispositions-at-the-follow-up-25-cutoff).

| Finding | Status and owner at #373 |
|---|---|
| r18 M-1 | **OPEN → agent-core 32 (running)**: control observation POST after SIGTERM; the deferred context lacks the cancellable control budget. |
| r18 M-2 | **OPEN → agent-core 32 (running)**: shutdown stacks past the **10 s** bound (review measured **14.021 s**); the unit has no `TimeoutStopSec`. **Installer** sets the value **after agent-core 32** establishes the budget. |
| r18 M-3 | **OPEN → agent-core 32 (running)**: replayed lease holds the node on the baseline, **no cap or distinct alarm**, despite the new-generation ACK. **Wire-nodes** handles central's stale-hold reading **after agent-core 32**. |
| r18 L-1 | **OPEN → netlab**: TEST ONLY command sample ignores the install deadline. |
| r18 L-2 | **OPEN → agent-core 32 (running)**: the written D-133 candidate rule does not compare stricter member/node ceilings. |

**What held is the review's statement, not an acceptance result**: no removed
grant forwarded on any reviewed stale path; observation keys out of production;
torn checkpoints fail closed. The review ran no live lab or PostgreSQL suite,
did not reproduce the 80-transition population or physical arm64, and did not
implement intersection serving. M-1 / M-2 qualify the earlier #369 shutdown
claims; the historical TestLab04 cancellation fix is not erased and is not
proof of a universal shutdown bound.

## Decisions and findings at #373

**No new decision since D-138. D-135, D-136, D-137 and D-138 stay provisional.**
D-136 retains the **04:3x EST, 2026-11-01 correction**: verified deny-all
candidate only with a retained Apply error; ordinary stale-lease path
baseline→ACK→renewal→Apply→verify→heartbeat; **no finite outage ceiling**.
Every baseline bracket and loss stays counted. The decision rows are unchanged
from follow-up 24, including D-138's owner overturn option.

The [findings register](README.md#findings-register-at-the-follow-up-25-cutoff)
adds **updates F13-01**, **ISO-F6-01 assertion 6**, and **r18 M-1 / M-2 / M-3 /
L-1 / L-2**. It updates **members F11-01** to located, with devstack 25 gathering
first-selection/userspace-WireGuard evidence, and **members F13-01** to raw
bytes kept by #370, cause unproven. **Every other follow-up 24 finding is
unchanged**. Every historical failure stays failed. In particular, install-overrun
cause remains unproven; F9-INFLIGHT, intended-negative F18-01, ISO-F6-03's
unattributed writer, Dependabot #22 owner disposition and the production
blocked_response_ttl owner value retain their boundaries.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation**, together,
including the 04:3x correction, all losses and no finite outage ceiling.

2026-11-01 — **Owner decision 2: D-137 hardening option**, supported scope and
whether lower tunnel MTU or an MSS clamp is wanted.

2026-11-01 — **Owner decision 3: D-138 overturn option**, confirm or overturn
the provisional observation acceptance and pooled qualified-leg assessment.

2026-11-01 — **Owner decision 4: F11-01 repair direction**, upstream NetBird fix
and qualified pin, userspace WireGuard against the agreed kernel baseline, or
IPv4 preference. Costs and risks remain in #370's analysis; no option is applied.

Already assigned, not new requests (2026-11-01): **agent-core 32** owns r18
M-1 / M-2 / M-3 / L-2 and ISO-F6-01 assertion 6 (`node/agent/observe.go`,
`node/agent/serving.go`, `node/agent/engine.go`, generation/availability note).
**Netlab** owns r18 L-1 (`node/enforce/command_lifecycle.go` and resource sampling).
**Installer**, after agent-core 32, owns `TimeoutStopSec` in
`deploy/node/systemd/fvpn-agent.service`; **wire-nodes**, after agent-core 32,
owns central's stale-hold reading (`internal/nodes/observations.go`,
`internal/nodes/publication_evidence.go`). **Devstack 25** owns updates F13-01
TEST ONLY readback (`deploy/devstack/agent-service.sh`,
`deploy/devstack/entrypoint-node.sh`) and members F11-01 first-selection /
userspace-WireGuard evidence. **Members 14** consumes #370's raw parse evidence
and #367's routed hop and instruments the original F12-04 query. **Operators**
retain AT11 / AT13 platform/fault resources and AT29 / AT31 resources.
These outside-owned items stop with their owners; the rest is delivered.
**No outside-owned pin needed changing.** No new production value is selected.
