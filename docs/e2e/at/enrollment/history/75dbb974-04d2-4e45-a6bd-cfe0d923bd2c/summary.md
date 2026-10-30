# Enrollment acceptance

Every required check must pass. Missing hooks keep the complete AT blocked.

| AT | Status | Decisive observation | Evidence |
|---|---|---|---|
| AT06 | blocked | {'api_hash_matches': False, 'field': 'changes.after.token_hash', 'operation': 'listAuditEvents', 'owner': 'pki-enroll', 'request': 'ENR-02', 'resource_id': '01a18dff-1633-7ee4-a0c8-57652881a146'} | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | ENR-07 / devstack follow-up 12: MachineIdentityFingerprints is not exported; API identity comparison alone is partial | [AT07.json](AT07.json), [AT07.md](AT07.md) |

See [RUN.md](RUN.md) and [requests.md](requests.md). Immutable attempts are in `history/`.
