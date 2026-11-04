# Follow-up 33 — shared helpers, offline validation

This follow-up exports the publisher's bounded conflict handling for AT callers,
restores revoked grants after a failed finite transition, and rolls back partial
native-router setup. No stack, Docker, database container or live run is part of
this follow-up. Consumer owners provide live proof in their next authorized
assembled runs. No timing, physical arm64 or network acceptance is claimed.

Implementation commits: `6a9c89b0` (export), `15d9e8c6` (grant recovery),
`25f738c7` (router rollback). The final records commit also refines offline
coverage with an actual dropped create response and the labstack import check.

The [gate ledger](checks.md) preserves each executed gate, UTC bounds, exit,
real output tails and links to full sanitized output. Admission denials are
waits, not gate results. All scratch is on disk under `.agent/tmp`; no reservation
was taken. The hygiene test scans this directory too.

The [dated adoption entries](../../../RESPONSES.md) document the exported API,
predicates and receipt shape, every routed AT owner, members' duplicate recovery
semantics, and router rollback access. Earlier records and all AT files remain
unchanged.

Coverage: API01 / §14.5 conflict contract; ENT01, ENT02, BILL02 / §9.2 finite
plan recovery; AT02 fixture continuity and D-137 router setup. These tests prove
helper control flow, not the consumer's eventual acceptance cases.

Validation completed at 2026-11-04T00:33:54Z (2026-11-03 local): all focused
stub tests, e2e lint, isolation overlay lint, e2e-tagged vet of every AT package,
non-stack e2e tests, handover tests, root Go lint and committed-evidence hygiene
passed. Linux arm64 cross-compilation of the tagged shared harness also passed;
this is compile-only evidence. The unchanged existing node-write tests passed,
including their actual ten-second timeout and no-implicit-replay cases.
There were no executed failing gates. Admission denied entry while isolation
was running; those waits were not test results.

Commit `e1903a99` added adoption records and test refinements. Commit `925207c2`
retained the original sanitized revoke error text while preserving the cause
for `errors.Is`; offline tests also verify that raw domain errors are excluded
from public error records. The final evidence commit contains the output files
and final checks. No AT consumer edits or new live evidence were produced.
