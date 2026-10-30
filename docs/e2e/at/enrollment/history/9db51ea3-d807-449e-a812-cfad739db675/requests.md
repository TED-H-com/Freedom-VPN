# Enrollment acceptance requests

Binding: `w4-enroll / 66000`. D-050 applies to every request. These are missing
acceptance prerequisites or read surfaces, not claims of a live product defect.
No projector row may be seeded or repaired to close them.

| ID | Owner | Affected evidence | Requested interface or action |
|---|---|---|---|
| ENR-01 | orchestrator + w3-devstack-e2e | All live AT06/AT07 checks | Assign an unused legal port slot and authorize the binding change. `parseLayout` permits eight bases `18000 + 6000*n`, n=0..7; 66000 selects slot 8 and exceeds 65535 even before offsets. Do not widen the validator to accept impossible ports. |
| ENR-02 | PKI/enrollment + control audit owner | AT06 token hash in admin audit | Expose an allowlisted `token_hash` (SHA-256 hex) in the enrollment resource's `/api/v1/admin/audit-events` evidence, or an approved equivalent audit digest. `appendDomainAudit` currently emits only `reason`. A locally calculated hash is not server storage evidence. |
| ENR-03 | nodes/control contract owner + PKI | AT06 one incarnation; AT07 old incarnation/history | Provide an admin read surface for complete incarnation count/history per Node, including immutable ID, public-key digest and revoked state, and retained usage attribution. `NodeDetail.current_incarnation_id` proves only the current pointer. No database query or uniqueness-constraint inference replaces the required API evidence. |
| ENR-04 | w3-devstack-e2e + NetBird adapter owner | AT06 actual peer uniqueness after lost response and race | Document/export `JoinEnrolledIdentity` for harness-generated key/CSR enrollment, with secrets remaining private, normal one-time mTLS bootstrap, and `EnrollmentPeerInventory` returning complete read-back counts correlated to the enrollment operation. Existing `EnrollAt` generates its own request and cannot exercise our identical-byte response-loss request. Zero peers before joining cannot prove peer idempotence. |
| ENR-05 | w3-devstack-e2e + usage agent owner | AT07 counted, not yet ACKed usage and continuity | Document/export `HoldUsageAcknowledgements(ctx, slot, hold)` at the transport boundary and `UsageLedgerSnapshot(ctx, slot)` with immutable event hashes/decimal sequences and actual ACK watermark. Pair with a shared `CountedMemberTraffic` hook that polls native exit availability, selects the API-published exit and confirms device counters plus controlled-origin receipt. Release the hold through the same hook and verify contiguous ACKs, no duplicate events and unchanged history. `PauseAgent` stops the producer too and is not this hook. |
| ENR-06 | w3-devstack-e2e + packaging/installer owner | AT07 installer rerun and controlled reinstall | Document/export `RerunInstaller(ctx, slot)` executing the real installer on supported disposable node storage, preserving identity and outbox; `ControlledReinstall(ctx, slot, enrollment)` must use a fresh API-authorized token after revocation, preserve stable Node/history and retain a local old-cert mTLS probe. Export only its HTTP/TLS rejection verdict. Container restart is durable resume, not installer rerun. Do not copy keys, erase identity or edit ledgers from this harness. |
| ENR-07 | w3-devstack-e2e + packaging/agent identity owner | AT07 machine identity and cloned enrolled image boundary | Document/export `MachineIdentityFingerprints(ctx, slot)` returning only hashes of OS machine-id, Agent SPKI and NetBird public key; `StartClonedIdentityFixture` must create the disposable cloned-image condition before boot and expose conflict/freeze observations. Existing `StartSecondExit` is a clean base-image second boot, explicitly not an enrolled-identity clone. No ad hoc copy into a running container is allowed. |

ENR-01 is the initial decisive blocker. ENR-02/03 follow the current public API
and audit writer; ENR-04..07 follow the documented lab export inventory in
[`docs/e2e/RUN.md`](../../../../RUN.md) and `tests/e2e/internal/labstack`. They remain
requests even after a valid slot is assigned. The separate AT07 first-boot
probe uses `Enroll`, `StartSecondExit`, `EnrollAt(..., "node2")`, public
`cert.pem` readback, exact image IDs and admin current-incarnation readback.

The API already supports a 60-second enrollment lifetime and revocation. No
clock manipulation, TTL override below the contract minimum, direct PKI service
call or enrollment database fixture is needed for those AT06 negatives.
