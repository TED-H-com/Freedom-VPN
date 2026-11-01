## Recommendation and D-133 acceptance plan

Recommend **(b) for adversarial design review**, while retaining **(c) in
production** until its provenance/ACK and revocation-first boundary are resolved.
It targets the actual unchanged-sibling outage during a revocation. The safe
restriction of (a) helps only non-restrictive changes; its unrestricted version
violates the no-later revocation requirement. This is a proposal, not approval
to alter a signed-generation contract or to waive AT02.

Freeze A (current main) and B (reviewed candidate) with source/binary hashes;
use the same host, kernel, inputs, fixture image, observation mode, probe cadence,
Go runtime settings and exact signed/model/rendered bytes where reusable. Run
**A1,B1,A2,B2**, with both full agent and live suites in each leg. Record
`vmstat 5 7` before startup, before traffic and after; exclude the since-boot row
from qualification, require no competing suite and six intervals r≤3/id≥80.
Keep rejected preflights, failed legs and unqualified runs. Never replace a
failure with a later pass or compare an instrumented B to an uninstrumented A.

Each leg has at least 40 restrictive and 40 non-restrictive changes. Record
publication/receipt, E command start/end, lease request/acceptance, each Apply,
completed/uncompleted guards, live and path verification, serving heartbeat,
first subsequent sibling egress and every loss inside/outside guards. Publish
all vectors and nearest-rank p50/p90/p99/max, including failure/censoring counts.
Use full run/boot/sequence joins and availability flags. Compare total outage,
not a guard-only percentile. No cause is inferred from CPU/cgroup correlation.

Acceptance requires no later revoked-source cutoff than paired A's permitted
cutoff/old D, no new/additional/stale binding admitted, no larger allowed
expiry bound, and improved sibling outage/loss distributions in both B legs. As a proposed
trial acceptance rule, all four reported percentiles must be non-worse, with
strictly lower p90 and p99 for restrictive transitions in both B legs; retain
normal and injected-failure populations separately and reject unmatched safety
failures. This is a review criterion, not a new production SLA.
A failed live timing gate triggers the D-133 comparison and remains unresolved
until attributed; no threshold inflation. Also inject delayed/replayed/wrong-
generation/empty leases, exhausted I/O budgets/Retry-After, desired supersession,
expiry during each stage, tc/nft/path failure, rollback failure, address/member
reuse, freeze during each stage, same-boot restart and reboot, and crash between
every kernel commit and durable write. Verify meter continuity, no duplicated
accounting, exact revocation cleanup, control access and honest observations.
Cross-compile amd64/arm64 and record real arm64 qualification separately.
