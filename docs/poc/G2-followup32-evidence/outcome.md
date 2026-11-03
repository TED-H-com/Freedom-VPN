# Follow-up 32: enforcement fixes and C1 revision

**Enforcement implementation and focused/live proofs delivered; complete consumer
verification is blocked on agent test-fixture compatibility. C1 remains docs
only and production retains the current unconditional guard.**

H-4 and H-2 tc were confirmed. M-2 was confirmed for post-shape clock failures;
its alleged reachable fatal `Verified:true` return was rejected because validation
prevents the only non-command conntrack error and Samples cannot fail today.
Read the [pre-edit judgments](judgment.md), [original failures](proofs-before.txt)
and [fixed/negative-control output](proofs-after.txt). Original tests were first
committed alone at `cdb384ed`; implementation is `240b0278`, design `f9aa1811`.

The narrow production changes make rollback independent of a fresh table read
and the canceled Apply context; reset the candidate's fixed chains/sets in one
self-contained batch; roll back both post-shape clock errors; and prepare tc
before nft while retiring classes only after guarded commit, before release and
verification. Counters, independent control path, foreign-filter quarantine,
100ms deadline budget and verified conntrack retry behavior remain. An extra or
missing final class fails verification. No old-policy restoration is added.

[The lab summary](lab-summary.md) proves the baseline's actual nft 1.0.9 form,
transaction abort atomicity, counter continuity, member closure and existing
live consumers. Kernel 7.0.0-38-generic is the tested lab platform. This is not
native arm64 qualification or a D-133 performance comparison.

| Required gate | Result / actual tail |
|---|---|
| 1 lint, unit, integration, build | **PASS**: enforce 1.404s, netlab 2.694s, 58 Python tests OK; tagged harness packages pass; amd64/arm64 builds complete. [Output](gate1.txt). |
| 2 single full lab | **PASS**, exit 0, suite 232.826s; total 241.010s. [Receipt](gate2.json), [output](gate2.txt.gz). |
| 3 node / installer | **FAIL**, exit 1, 90.141s; seven top-level agent fixture tests fail. Other packages pass. [Output](gate3-node.txt), [dated hand-off](consumer-handoff.md). No consumer edits or rerun. |
| 3 usage / counterread | **PASS**, exit 0; usage 2.483s, counterread 0.046s. [Output](gate3-usage.txt). |
| 4 evidence hygiene | **PASS** after a filename-only publication correction; 2.745s. Initial invalid-JSON failure is retained; the AT12 Go dump is now .txt with identical bytes. [Initial](gate4-initial.txt), [final](gate4-final.txt); the final gzip/pin publication [recheck](publication-hygiene.txt) also passes. |
| 5 handover | **PASS**, 3.426s after the correction. [Output](gate5-final.txt). |
| 6 root go-lint | **PASS**, exit 0, no output; pinned whole-module staticcheck included. [Receipt](gate6.json). |
| 7 go vet ./... | **PASS**, exit 0, no output. [Receipt](gate7.json). |
| 8 tracked Go formatting | **PASS**, empty gofmt -l. [Receipt](gate8.json). |
| 9 diff whitespace | **PASS**. [Receipt](gate9.json). |

[Gate receipts](gates.json) retain exact commands and measured durations where
captured. Final production/tests are unchanged since the live run. The only
subsequent changes publish evidence/docs. Every gated invocation was admitted
with `free`; an earlier wait for agent-core's lab was not a test result. All
scratch used disk `.agent/tmp`, no reservation/traps, no push/PR/tag/dependencies
or tool pins changed. [Owned Docker inventory](inventory.json) is empty for
containers, images, networks and volumes. No test process remains running.

The failing node consumer uses an nft simulator that requires braces on every
add chain and does not preserve existing sets/chains on add. Its tc model also
needs to represent the new two-phase mutation. The required “consumers without
edits” rule leaves those files unchanged; only that item stops, with an explicit
owner request. This is **not** an assertion that main's full test contract passes
or that the patch is ready to merge. The full failure and its source diagnosis
are retained without attributing untested secondary failures.

[C1 revision](../G2-followup32-design/revision.md) addresses eligibility, per-device
deadline/verified-serving provenance, atomic old-bank retirement, child-reaping
fencing, tooling prerequisites, empty eligibility and corrected frequency/exposure
figures. Its owner's paragraph asks the G5/G6 question; no decision is inferred.

Follow-ups **25–31 remain byte-identical**: [1,830-file retention check](retention.json).
Historical B1 stays FAIL and unexplained; F30 remains host-unqualified. Native
arm64, C1 implementation/proof/campaign and the orchestrator's quiet D-133 slot
remain later work. No historical failure is rewritten by this functional pass.
