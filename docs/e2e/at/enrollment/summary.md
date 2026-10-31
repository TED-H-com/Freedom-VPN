# Enrollment acceptance

Every required check must pass. Missing hooks keep the complete AT blocked.

| AT | Status | Decisive observation | Evidence |
|---|---|---|---|
| AT06 | blocked | HTTP 429 recurred (2) on limiter class browser, bucket BrowserRate 10/s burst 30. Owner: control. The bebfe230 review stays unresolved. | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | ENR-07 / devstack follow-up 14: the lab has no second /24 for the clone. StartClonedIdentityFixture(ctx, "node") attaches node-clone to the same lab bridge and the same origin bridge (last octet .23). The agent mTLS path is the local resolver proxy to agent.example.invalid:9443, whose TCP peer is that lab address, not trusted gateway 172.28.38.21/32, so X-Forwarded-For is not the compatibility domain. Interleave, conflict freeze, revokeNode plus re-enroll, and the quiet-window promotion were not run. | [AT07.json](AT07.json), [AT07.md](AT07.md) |

See [RUN.md](RUN.md) and [requests.md](requests.md). Immutable attempts are in `history/`.

## Follow-up 3 — run `60a7102e-b824-4927-a576-62033f458157`

UTC `2026-10-30T23:05:38Z`–`2026-10-30T23:10:54Z`. Source `99ab668` dirty, tree `62e86bf6b54985af168edc6d49193a272c4442007b03074df8686e07f37be406`. Both probes exited 0. Cleanup `{containers:0,networks:0,volumes:0,images:0}`.

AT06: 254 passing, 1 blocked, 0 failed. The blocked check is `rate_limit_review`: two HTTP 429 responses, both limiter class `browser` (`GET /api/v1/admin/audit-events` and `GET /api/v1/admin/nodes/{id}/incarnations`). `audit_read`, `issue_reinstall` and `incarnation_history` passed after `Retry-After`. The bebfe230 review stays unresolved. Owner of the bucket: control.

AT07: 15 passing, 7 blocked, 0 failed. The clone and the source shared `172.28.38.0/24` (`172.28.38.40` and `172.28.38.129`) and `192.0.11.0/24` (`192.0.11.20` and `192.0.11.23`). After both stages were `enrolled`, `Observe` reported one endpoint row, zero conflicts, `mapping_frozen` false. `getNode` projected `identity_conflict` `{false, 0, null}`. The interleave, freeze, revoke plus re-enroll, and quiet-window steps were not run. ENR-05 and ENR-06 stay blocked on the follow-up 13 recheck and were not emulated.

Run `35e542f8-730e-4be0-8214-e4a9dd818c40` is the earlier attempt in this follow-up. Its process exit is the same two 429s failing the Go test. That history directory is unchanged.
