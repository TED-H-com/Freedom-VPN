# Follow-up 18 run 1 archive

Run `118271c1-616e-46a2-a744-d67b00fb94fb`, harness `b4317bd4e4ad178609b71d3fd4fbb986cb9947db`, baseline `27ff63928222c9642e4f5e180cf225108327a1e5`. All **72** root files are frozen byte-identically, including gzip and sidecars. [SHA-256 pins](SHA256SUMS), [original results](summary.md), [original records](F18-results.json), [navigation references](LINK-ALIASES.json).

The observed result remains **0 PASS / 3 BLOCKED / 7 FAIL; 442 passed / 35 failed subchecks**. No D-146 matrix row was exercised. The [retained producer evidence](F18-lifecycle-phase-compatibility.json) proves the measured reader rejects the production `tc` phase `retire` at sequence 582, command index 28; 16 earlier export receipts match exact prefixes of that original stream. The rejected trace explains those reader failures, not every independent failed predicate. [Grant-restoration path and AT09 prerequisite](F18-AT09-precondition.json), [every AT09 check](F18-AT09-subchecks.md).

Orchestrator note 1 (2026-11-03, 15:58 EST) authorizes a reader correction and exactly one second assembled run. These files preserve the state when the note arrived; any earlier statement that no second run was authorized is historical. Run 1's verdicts stay unchanged. The original offline gate failed on a missing gzip-sidecar navigation alias; that gate receipt is retained privately. Earlier archives and their pinned files remain unchanged.

[Original payload-free capture archive](../follow-up-18-captures/118271c1-616e-46a2-a744-d67b00fb94fb/analysis.json). Private retained files, including both raw producer streams, are moved to `.agent/tmp/f18-run1/` before run 2 can write any of their former paths. No handover claims test reads a moved root record; no D-132 test-pointer edit is required.
