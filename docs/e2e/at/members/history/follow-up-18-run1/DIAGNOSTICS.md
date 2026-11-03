# Follow-up 18 diagnostics

[All case counts and failed checks](F18-results.json), [all AT09 subchecks](F18-AT09-subchecks.md), [D-146 packet/rule evidence](F18-D146.json), [restart observations](F18-AT09-restart.json), [agent transitions](F18-agent-transitions.json), [safe parser receipts](F18-F13-01-private-receipts.json), [full AT02 accounting](AT02-TRACE.md).

No timing change is attributed to a specific merged fix without an exact original-record join. [Previous narrative](history/follow-up-17/DIAGNOSTICS.md) and every historical failure remain frozen.


The measured lifecycle consumer rejects producer command 28, phase `retire`, at sequence 582. The [original byte-pinned join](F18-lifecycle-phase-compatibility.json) retains the command and **16 exact export-prefix matches**. Its whitelist lacks this production phase. The original rejected trace is unavailable, not a record of zero Applies. Supplemental raw-export joins preserve the observations but never repair the original D-135/D-136 verdict. This does not by itself explain every final central-readiness failure or packet loss.

The [same-member precondition join](F18-AT09-precondition.json) retains the last AT02 grant (`valid_until=2026-11-03T20:01:12Z`), subsequent failed readiness checks, absence of the normal restoration, and AT09's HTTP 200 / zero-exit result. The measured `finiteExpiry` source returns on a failed positive probe before restoring the grant. AT09 records the admitted-device prerequisite blocker and performs no native matrix row. No family, MSS, outer-size, no-PTB or bidirectional-byte result is available for dual/IPv6/IPv4 1280. This is a coverage gap, not a measured clamp failure.

The run reached its existing 70-minute context limit during the final AT09 prerequisite/readback work. Its final central hold join failed with unavailable readback; all ten case records are nevertheless marked complete by the original runner. No timeout, input or predicate was lengthened, no restoration was injected, and no run was retried. The four controlled populated-hold scenarios are **not exercised**. The separate cumulative AT02 incident does not substitute for them.
