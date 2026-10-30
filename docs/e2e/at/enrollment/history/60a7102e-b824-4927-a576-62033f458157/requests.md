# Enrollment acceptance requests — follow-up 3

Current binding: `w4-enroll / 12000`. D-050 applies to every request. The
initial 66000 attempts are immutable history, not the current binding.

| ID | State | Owner | Exact remaining evidence / question |
|---|---|---|---|
| ENR-01 | closed | orchestrator / devstack #182 | #182 assigns legal base 12000 to w4-enroll. Active pins use 12000. |
| ENR-02 | closed | pki-enroll #195 | Contract revision 10 `Enrollment.token_sha256` (`^[0-9a-f]{64}$`) is SHA-256 of the exact plaintext token. Run `7615d23d-d572-4db1-a903-9610dd3bcef3` matched the harness digest on create, get, revoke and `enrollment.created` `changes.after.token_sha256`. Consumed and revoked events omit the field. Plaintext was absent from admin reads and the log streams. |
| ENR-03 | closed | w3-wire-nodes #200 | `GET /api/v1/admin/nodes/{node_id}/incarnations` (`listNodeIncarnations`). The same run: a fresh Node has one current incarnation equal to `getNode.current_incarnation_id` and `getEnrollment.node_incarnation_id`; the same-holder retry and the two-key race left the count at one; after `revokeNode` (202) and a second bound enroll, the list is newest-first with count 2, the previous row `retired` / `reinstall` (it was `revocation` before the replacement), usage `"0"`/`"0"` unchanged, cursor round-trip with no duplicates, and 404 for an unknown Node. |
| ENR-04 | closed | devstack follow-up 12 #204 | `JoinEnrolledIdentity` and `EnrollmentPeerInventory` ran on a second fresh stack. Discarded-response race: HTTP 201 and 409 `ENROLLMENT_KEY_MISMATCH`. Replay joined (enroll 200, bootstrap 200). Inventory `complete`: one setup key, `setup_key_uses` 1, one projected peer, `correlation_groups` 0. A second read stayed at one peer. |
| ENR-05 | blocked | w4-usage, then devstack | Follow-up 13 recheck still holds on this tree (`99ab668`, #219). `node/internal/usageupload` acknowledges every batch. `HoldUsageAcknowledgements`, `UsageLedgerSnapshot` and `CountedMemberTraffic` are not in `docs/e2e/RUN.md` or `tests/e2e`. Not emulated. |
| ENR-06 | blocked | packaging/installer, then devstack | Follow-up 13 recheck still holds. `packaging/` is the release archive. `RerunInstaller` and `ControlledReinstall` are not exported. A container restart, key erasure, or copied identity rows are not that proof. Not emulated. |
| ENR-07 | blocked; devstack follow-up 14 | devstack | Fingerprints stay measured: `MachineIdentityFingerprints` distinguishes `node` and `node2`, and `StartClonedIdentityFixture` copies one enrolled identity. The conflict path needs the clone's agent mTLS client address on a second lab network (a different IPv4 `/24`, not a second port on `172.28.38.0/24` or `192.0.11.0/24`). `StartClonedIdentityFixture(ctx, "node")` has no network argument. `startSecondExit` puts `node-clone` on `fvpn-w4-enroll` and `fvpn-w4-enroll-origin` at origin last octet `.23`, beside the source SNAT `.20`. Trusted proxies are only `172.28.38.21/32`. The agent reaches control through the local resolver proxy to `agent.example.invalid:9443`, so the TCP peer is the netns lab address and `X-Forwarded-For` is not the compatibility domain. Need an export that places that certificate on a second lab network. |
| ENR-08 | closed as a product gap; conflict unmeasured | wire-nodes #211 and #217 | `ObserveEndpointTx` runs inside lease renewal and observation submit (#211, `e9f1a70`). `GET /api/v1/admin/nodes/{node_id}` projects `identity_conflict` `{mapping_frozen, conflict_count, last_conflict_at}` (#217, `6b98422`). The "no production caller" statement from run `7615d23d` is stale. This harness does not close the conflict proof: the lab cannot present two compatibility domains (ENR-07). |

[API surface review](api-surface-review.md) names the revision 9 reads, the
revision 10 fields that closed ENR-02 and ENR-03, and the follow-up 12 exports
measured here. No SQL read, row repair, fake observation or ad hoc identity
copy is proposed.

The merged lab exports `Enroll`, `StartSecondExit`, `EnrollAt`, `PauseAgent`,
durable restart, `JoinEnrolledIdentity`, `EnrollmentPeerInventory`,
`MachineIdentityFingerprints` and `StartClonedIdentityFixture`. It does not
export the ENR-05/ENR-06 names, and the clone export has no second-network
argument. Shared hook documentation: [RUN.md](../../../../RUN.md).

Run `7615d23d-d572-4db1-a903-9610dd3bcef3` closed ENR-02, ENR-03 and ENR-04 and
measured fingerprints. Run `9bb66c4e-e429-450b-8671-521d33926ede` measured the
same token-hash and incarnation checks on `746651e` before follow-up 12 was in
this branch. Run `bebfe230-26d4-4e3d-b052-81d258d04e07` retains the 429 failures
until the review below applies. Follow-up 3's run id is the AT JSON `run_id`.

## AT06 retained 429 review

Run `bebfe230-26d4-4e3d-b052-81d258d04e07` failed `audit_read`, `issue_reinstall`, `incarnation_history` and `probe_process` with HTTP 429 `RATE_LIMITED` on the admin reads and the enrollment create. Those routes use limiter class `browser`, bucket `BrowserRate` (`internal/config.Defaults`: 10/s, burst 30). Owner of that bucket is control. The harness change that honors `Retry-After` (admin `callHeaders` up to 6, agent enroll `post` up to 4, same `Idempotency-Key`) is `286eb57b44c2e783257767918680a0e1f097f106`, merged as #210. The review resolves those four prior failures only when this run's `rate_limit_review` records `http_429` 0 and the named checks pass. The failed rows stay in `prior_checks`. A later 429 keeps AT06 blocked and names class `browser` / owner control. AT07's `probe_process` fail from the same run is the process exit of that burst. It stays an unreviewed prior failure. Current AT07 blockers decide that AT.

## Devstack follow-up 13 recheck (2026-10-30)

Rechecked on devstack parent `52dc770`. ENR-05 stays blocked. `node/internal/usageupload` still acknowledges every batch. There is no withhold-only-acks switch and no approved read-only ledger snapshot with immutable event hashes, decimal sequence and epoch identity, and the contiguous ACK watermark across restart. `PauseAgent` is not that hook. Do not edit or delete outbox rows. Owner: w4-usage, then devstack. Symbols: `HoldUsageAcknowledgements(ctx, slot, hold)`, `UsageLedgerSnapshot(ctx, slot)`, `CountedMemberTraffic`.

ENR-06 stays blocked. `packaging/` is only the release archive. There is no installer invocation or disposable Ubuntu service entry point to wrap. A container restart, key erasure, or copied identity rows are not that proof. Owner: packaging/installer, then devstack. Symbols: `RerunInstaller(ctx, slot)`, `ControlledReinstall(ctx, slot, enrollment)`.
