# Enrollment follow-up 8 — 2026-10-31

**Four formerly blocked rows pass; controlled_reinstall remains blocked by a new
agent/meter precision defect.** D-116's ownership marker is no longer a blocker.
Run `f27a81a0-183e-46b4-85a8-304b2f64b635`: AT06 501 pass / 0 blocked / 1 fail; AT07 43 pass / 1 blocked / 2 fail.

AT06's fresh subtest passed in 61.16 s and the independent join probe passed.
Its 501 assertion/read/scan rows pass. Its only failed row is the shared full
process exit caused by AT07, so the aggregate gate is honestly left blocked.
AT07's two failures are the precise installer prerequisite assertion and the
same process exit; neither is hidden or relabeled as success.

Before **908** + held **1816** = after **2724 bytes**. Two original events remain attributed once only to the retired incarnation. Origin ACK **3**, independent current ACK **0** through same-key replay; the new stream starts at sequence **1**.

Freeze withdrawal was **334 ms**, **+107 ms** against the 227 ms baseline, below the 5000 ms bound. Real receipt replay and durable latch/restart passed.

| Row | Status | Owner | Evidence |
|---|---|---|---|
| installer_rerun | pass | installer | Same identity and stopped file equality, restored serving and counted traffic; AT07 JSON and text rulesets. Devstack 18 installer-marker-migration / installer-wrappers corroborate. |
| controlled_reinstall | blocked | agent/meter; usage-central | Wrapper, new identity, retained files, strict verifier, serving and fresh traffic pass. New current stream cannot ACK its 1 ns gap; [deciding evidence](current-stream-gap.md). This is not D-116. |
| counted_unacked_usage | pass | agent / meter | Installer-node nonzero held upload/download, TCP/UDP receipts and counter increases; devstack 18 installer-usage corroborates. |
| identity_ledger_preserved | pass | usage / agent | Original canonical payloads/hashes/ACK survive D-118 adoption; original stream delivered first using the current presenter. |
| historical_usage_retained | pass | usage / control | AT20 named equality, historical samples, canonical same-key replay and independent ACKs all pass. |

Per-row status/owner/evidence: [AT06](AT06.md), [AT07](AT07.md),
[raw AT07 run](../f27a81a0-183e-46b4-85a8-304b2f64b635/AT07.json), [published assessment](../f27a81a0-183e-46b4-85a8-304b2f64b635-assessment/README.md).
Full text read-back: [installer rulesets](installer-rulesets.md).
Assertion definitions: [usage AT20](../../../usage/AT20.md). Corroborating installer rows:
[devstack 18](../../../../runs/2026-10-31/followup18/README.md).

D-098: [follow-up 7](../follow-up-7/README.md) is byte-identical and checksum
indexed. The two corrected enrollment assertion mistakes remain in their
[deadline](../57b90cf6-d233-4209-a114-765ef15376f7/REVIEW.md) and
[cursor](../805ba242-f343-44ab-9221-d250d81b95b9/REVIEW.md) archives.
The new-stream failure also occurred in [a5c706dd](../a5c706dd-7b2a-437e-88f2-3d1cd7b403cd/AT07.json);
the final diagnostic run retains its deciding payload and database evidence.
[Runbook](RUN.md); [one request to orchestrator](requests.md).
