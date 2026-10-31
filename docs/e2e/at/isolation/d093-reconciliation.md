# D-093 reconciliation — isolation follow-up 10

Consumed the [devstack 19 reconciliation](../../portal-d093-reconciliation-followup19-2026-10-31.md)
and checked it against the [portal follow-up 18 audit](../../../portal/DEV.md#d-093-harness-audit--2026-10-31-follow-up-18)
on product baseline `96ec0630`. All fifteen classifications agree: **13 open,
2 retired within the portal simulator's scope**. This isolation run does not
retire any portal row.

| IDs | Status | Isolation evidence boundary |
|---|---|---|
| D093-01, 02, 03, 04 | Open | Forwarding/check/error values are actual agent observation bodies and retained local verification; counted controls and kernel captures independently test them. No successful observation is written by this harness. |
| D093-05 | Open | Central freeze receipt remains a receipt annotation. The separate clone fixture's real response and agent state are retained, without claiming complete freeze acceptance here. |
| D093-06 | Retired in simulator scope | This run separately retains durable applied model/member-path state and compares applied, lease and kernel generations; simulated swap is not that proof. |
| D093-07, 08 | Open | Source agent identity, native NetBird and boot identity are real; configured TEST ONLY release/provider inputs remain fixtures. Container restart is not kernel reboot. |
| D093-09 | Open | Counted origin controls and admission-counter growth are measured; this lane does not assert portal fixed usage values as traffic or a complete accounting proof. |
| D093-10 | Retired in simulator scope | Designated TCP/UDP DNS probes use the real resolver path. Portal metadata retirement does not imply signature/time checks or kernel DNS enforcement. |
| D093-11, 12 | Open | This run does not claim packaged updater acceptance or portal release verification. Devstack 19's D-125 serving campaign is separate evidence. |
| D093-13, 14 | Open | No projector rollup or authorization output is fabricated for these isolation measurements. Existing shared fixture exceptions remain explicitly separate. |
| D093-15 | Open setup | Synthetic IdP/payment/provider inputs drive real central workers; they do not assert worker completion. |

The new producer timing is a local TEST ONLY diagnostic sibling of D-106,
written by the real agent after operations. It changes neither the observation
schema nor any observation's safety/forwarding result. D-087 now has the documented wire-nodes 33 / devstack 20 input seam: one
constructor-injected eligibility reader in the dedicated TEST ONLY control.
It changes future lease grants only; the desired compiler, entitlements and
device projections remain real and are compared byte-for-byte. The assembled
transition and its original signed wire evidence do not retire any portal row.
Devstack 20 also aligned its release root and genuine meter checkpoint; this
supersedes the shared harness limitation described in follow-up 19 for D093-11,
while the portal simulator's own D093-11 classification remains open.
