# Enrollment acceptance — follow-up 2

Run `7615d23d-d572-4db1-a903-9610dd3bcef3`, 2026-10-30 20:49:33Z–20:55:13Z,
on `w4-enroll / 12000`. Source `efba7fba96e9e354900e9360b6ef4cfebf578ea7`
(dirty harness at invocation), tree SHA-256
`79cc1e81c8cc85f9044c425e3505608d51714bf91a90d0f444e0ed560899fc0b`.
Both fresh stacks and both live probes exit 0. The acceptance target exits
nonzero because AT06 still carries prior failures and AT07 still has blocked
subchecks.

| AT | Complete status | Decisive observation | Current subchecks | Record |
|---|---|---|---|---|
| AT06 | blocked | Prior failed check requires explicit review. The four retained failures are run `bebfe230-26d4-4e3d-b052-81d258d04e07` (`audit_read`, `issue_reinstall`, `incarnation_history`, `probe_process`), HTTP 429 `RATE_LIMITED`. The harness now honors `Retry-After`. That resolution is a harness change. Every current subcheck passed, including `token_sha256` (revision 10, PR #195) and `listNodeIncarnations` (PR #200) and `EnrollmentPeerInventory` (devstack follow-up 12, PR #204). | 205 passing, 0 blocked, 0 failed | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | `StartClonedIdentityFixture.Observe` showed the same machine, agent SPKI and NetBird key hashes on source and clone, both stages `enrolled`, `conflicts` 0, `mapping_frozen` false (ENR-08) | 15 passing, 7 blocked, 0 failed | [AT07.json](AT07.json), [AT07.md](AT07.md) |

ENR-02 and ENR-03 are closed on this run. ENR-04 is closed: the join cycle's
discarded-response race returned 201 and 409 `ENROLLMENT_KEY_MISMATCH`, the
replay joined with bootstrap HTTP 200, and both inventory reads were
`complete` with one setup key, one use and one projected peer. ENR-07
fingerprints are measured and differ across the two clean first boots. The
clone fixture ran; conflict rows did not appear. ENR-05 and ENR-06 stay blocked
on devstack follow-up 13.

Earlier immutable runs: initial `79b89e47-b23f-45f8-a683-28e0557dc1ba` and
`9db51ea3-d807-449e-a812-cfad739db675`; follow-up 1
`75dbb974-04d2-4e45-a6bd-cfe0d923bd2c`; the 429 attempt
`bebfe230-26d4-4e3d-b052-81d258d04e07`; the revision-10 measurement
`9bb66c4e-e429-450b-8671-521d33926ede` on `746651e` before follow-up 12 was in
this branch. [Requests](requests.md), [API review](api-surface-review.md) and
[RUN.md](RUN.md) carry the closure evidence. Teardown left owned containers,
networks, volumes and images at zero.
