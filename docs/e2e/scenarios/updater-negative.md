# D-125 stopped-producer negative scenario

[Updater follow-up 18, F18-01](../../updates/node-followup18-2026-10-31.md)
classifies the old stopped-agent campaign as the intended fail-closed response
to a producer that cannot answer a fresh barrier identity. Its immutable
[follow-up 19 row](../runs/2026-10-31/followup19/04de29d3-e56b-4a32-b678-21462845ddb6/installer-quiescent-candidate.json)
stays **failed**, with campaign 13 / 0 / 1; it was expecting candidate success.
The [follow-up 19 pins](../followup19-archive.sha256) remain unchanged.

Future stopped-producer scenarios are **negative D-125 coverage**, never upgrade
success. Require all of the following; do not relax the positive upgrade gate:

1. Original apply exits **1**, sealed replay **1**, recover **2**.
2. Terminal wire outcome remains `failed`, `CHECKPOINT_FAILED` at final sample,
   with null health/usage acceptance; no candidate pointer or executable switch.
3. Forwarding remains blocked, `recovery_required=false`, and the existing
   baseline remains. Local journal `rolled_back` is nondestructive transaction
   completion, not a successful upgrade or a destructive recovery-required case.
4. Original failed receipt/seal bytes remain identical on replay/recover,
   with no duplicate accounting or manufactured checkpoint.

The legacy candidate-success runner is historical; its positive assertions
were not weakened. A future negative runner must state this separate contract
before execution. Genuine upgrade success requires a running producer to answer
the fresh latch before updater-owned stop/start, successful root candidate
switch and all health checks. Devstack 20's regular release layout is confirmed
closed by updates 11's 27/27 installer extension and 5/5 regular AT23–AT27;
that independent success does not relabel the old stopped-agent row.
