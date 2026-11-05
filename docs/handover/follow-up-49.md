# Handover follow-up 49 — consolidated cut

This record covers **#482–#485 at `aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`**.
It retains the outcomes of [follow-up 48](follow-up-48.md) and earlier cuts.
For an introduction and operating links, start with the [delivery guide](../DELIVERY.md).

The four merged deliveries repair the clone configuration offline, retain the
previous handover, fix the recorded Node test failures and add a compatible
serving-read contract. They do not complete the outstanding live or real-device
acceptance work. Exact PR heads, merge commits and roadmap rows are in the
[source manifest](../../tests/handover/testdata/followup49-main.json).
PR #483's integrated head is `df8a4261e574ee21066255b3c7009f4115c4dfa0`.

## Clone configuration: #482

The accepted [devstack 37 analysis](../e2e/runs/2026-11-04/followup37/analysis.md)
identifies a static receiver mismatch: the copied runtime configuration could
point at port 9445, while the clone's container resolver proxy listens on 9444.
The offline fix in `tests/e2e/second_exit.go` keeps the enrollment URL unchanged,
sets the runtime URL to `https://127.0.0.1:9444`, and changes only that URL in
copied release configurations. Copied-release `datapath.snat` remains unchanged
and is a carried gap. Other JSON values and identity/durable-state bytes are
preserved; JSON formatting is not a preservation claim.

`TestD141_ClonePreparedConfigsDialContainerResolverProxy` checks these boundaries.
The historical inventory covers **30 files, not 30 runs**. F7 has four total rows
and three clone-observation rows; F8–F15 have three and two. Retained F16–F25
captures contain 90 total and 90 clone-observation rows with zero conflicts and
`frozen: false`; they do not establish quarantine or a consumer latch. F19 copied
F18's capture, and its own run ended before clone creation. Distinct post-change
stop reviews cover F16–F18; later copies are not independent experiments.

The historical selected clone release and runtime dial error were not recorded.
No historical AT10–AT13 outcome is re-scored. Live clone transport, consumer-latch
validation and before/after-stop clone logs remain outside this cut.

## Previous handover: #483

[Follow-up 48](follow-up-48.md) remains the dated record for #478–#481 at
`b921c75f180beb6c0051d585a86d6651b2c8d71a`, including r32 and the six historical
Node workflow failures. Its assertions and source snapshots remain intact.
The new manifest distinguishes the integrated PR head from the earlier native
delivery and verification trees; those intermediate trees are not PR heads.

## Node fixes and r33: #484

The [agent-core 53 note](../node/followup53-2026-11-04.md) explains the six recorded
Node CI failures through four supported mechanisms:

1. Short test `Boot` budgets exposed scheduling and I/O delays. Test setup uses
   normal boot budgets; production `local_budget.go` is net-unchanged. The R29
   helper waits for both health attestation and a file marker written after
   `Boot` returns.
2. Unchanged restart checkpoints at equality or rollback within one sample
   interval now preserve the open epoch and ledger. Reset and boot-change
   handling retain precedence. The [meter source](../../node/internal/meter/delta.go)
   and focused tests define the branch boundaries, including ordinary holds.
3. The D118 test waits for the initial 503 and restores availability after
   `Close`. The precise historical CI interleaving is inferred.
4. A real-time held-`FinalSample` fixture window is replaced with a manual meter
   clock. A deliberate 250 ms pre-boot delay reproduced the failure three times;
   it is a reproducer, not a captured CI schedule.

The [command inventory](../node/followup53-evidence/commands.json) retains the
failed development attempts and corrected focused count50/full-agent count3
runs with 24 joined load workers. The gate history retains the node-build
admission refusal (exit 125) and successful subsequent checks. The final agent
lab records 13 passes and one installer-marker skip; this is not new member-App
or isolation acceptance.

The [r33 archive](../reviews/r33-meter-restart-review.md) is byte-identical to its
review source. It records zero accepted findings. Final-tree replay passed
10 parent tests and 33 subtests; the review proof files remain branch-only.
[Review provenance](../../tests/handover/testdata/history/follow-up-49/review-provenance.json)
pins the source, archive and replay receipts.

## Diagnostic boundary and contracts: #484–#485

For **Isolation F17-01**, the demonstrated stale-lease fixture shows
`attachServingRead` overwriting a QoS detail needed by Central's
`pure_stale_generation` classification. This conditional result does not prove
that every node with a retained read can never produce that classification.
Independent failures retain precedence.

Provisional **D-164** reserves separate optional nullable `ServingReadEvidence`
on `ObservationReport.serving_read` and `NodeDetail.serving_read`. #485 ships
only the contract compatibility phase: API **1.11.0**, absent/null/value
preservation, generated types and vectors. Existing report hashes and old
historical assertions are preserved. The [contract notes](../contracts/README.md)
and [revision source](../../tests/handover/testdata/followup49-evidence.json)
record the D-112 sampling-bound and D-117 finite-grant wording corrections.
Central acceptance/projection, diagnostic consumers and producer rollout remain
pending. **F17-01 and D-163 remain open**; a field's presence does not activate a
pipeline or prove the previously unexercised case.

**Devstack F31-01** remains an unadopted structural verification proposal in the
agent-core note. Rule, allowed-element and kernel-route reads do not establish
per-device packet-origin authentication. No implementation or automatic dispatch
is authorized by this handover.

## Decisions, upstream sample and cutoff

The owner confirmed D-150–D-163; their technical substance is unchanged.
[Confirmation provenance](../../tests/handover/testdata/history/follow-up-49/owner-confirmation-provenance.json)
records the authority-only substitutions. D-164 remains provisional.

[NetBird discussion 8473 comment 19411545](https://github.com/netbirdio/netbird/discussions/8473#discussioncomment-19411545)
was posted at **2026-11-05T04:30:00Z**, using the frozen source commit
`52b93845b5cfeac1dd63e04c700857fcd098a7f6`. In the original run
`3eb3960d-c411-4648-8d77-bba40b436ae3`, node egress packet 158 and member ingress
packet 161 contain the same IPv6 `::1`-source WireGuard type-2 response. It reached
member underlay ingress without appearing in WireGuard receive accounting.
The surrounding header-error counter changed, but there is no packet-specific
kernel-drop trace and the source-selection mechanism remains unproven.
**Members F11-01 and Members F22-AT01 remain OPEN under D-148.**
[Comment provenance](../../tests/handover/testdata/history/follow-up-49/netbird-8473-comment-provenance.json)
retains the posted text, exact bytes and observation time; later clean initial
transfers do not close those findings.

At briefing, failed schedule **41836274140**, created at
**2026-11-04T13:14:20Z**, was the latest observed scheduled run. No newer schedule
is imported into this fixed cut. This is not a claim about the latest run now.
The [cutoff receipt](../../tests/handover/testdata/history/follow-up-49/after-cutoff-provenance.json)
separately records isolation follow-up 27 run
`d69f5652-aed6-4fdd-9e7c-edface751237` as
`completed_failed_awaiting_evidence_delivery`, unmerged and excluded from the cut.

## Evidence preservation

`TestHandover_Followup49Claims` checks the source manifests, immutable snapshots,
full source readings, exact evidence values and preserved historical assertions.
The [findings and decisions register](README.md) retains 67 decisions and 138
findings: six finding descriptions updated, none added, 132 unchanged.
The scope inventory changes only SHA-256 scalars for owned paths.

The approachable root README and delivery guide are current navigation documents.
The pre-polish follow-up49 README has its own source-pinned fixture for the new49
historical entrypoint assertion; generic link/path checks inspect the live files.
No earlier assertion body or historical snapshot is rewritten for this polish.

## Requests to orchestrator

Complete the existing integration checks and review this delivery. Carry the
known live clone, real-device, AT11/AT13, AT29/AT31, physical arm64, power-loss and
QUIC prerequisites through their existing acceptance procedures. D-164 later
phases, F31 implementation and architecture changes remain future decisions.
This record adds no product feature, live campaign, upstream post or release.
