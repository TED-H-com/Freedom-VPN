# Enrollment acceptance

Every required check must pass. Missing hooks keep the complete AT blocked.

| AT | Status | Decisive observation | Evidence |
|---|---|---|---|
| AT06 | blocked | HTTP 429 recurred (2) on limiter class browser, bucket BrowserRate 10/s burst 30. Owner: control. The bebfe230 review stays unresolved. | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | ENR-07 / devstack follow-up 14: the lab has no second /24 for the clone. StartClonedIdentityFixture(ctx, "node") attaches node-clone to the same lab bridge and the same origin bridge (last octet .23). The agent mTLS path is the local resolver proxy to agent.example.invalid:9443, whose TCP peer is that lab address, not trusted gateway 172.28.38.21/32, so X-Forwarded-For is not the compatibility domain. Interleave, conflict freeze, revokeNode plus re-enroll, and the quiet-window promotion were not run. | [AT07.json](AT07.json), [AT07.md](AT07.md) |

See [RUN.md](RUN.md) and [requests.md](requests.md). Immutable attempts are in `history/`.
