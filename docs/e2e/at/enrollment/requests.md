# Enrollment acceptance requests — follow-up 2

Current binding: `w4-enroll / 12000`. D-050 applies to every request. The
initial 66000 attempts are immutable history, not the current binding.

| ID | State | Owner | Exact remaining evidence / question |
|---|---|---|---|
| ENR-01 | closed | orchestrator / devstack #182 | #182 assigns legal base 12000 to w4-enroll. Active pins use 12000. |
| ENR-02 | closed | pki-enroll #195 | Contract revision 10 `Enrollment.token_sha256` (`^[0-9a-f]{64}$`) is SHA-256 of the exact plaintext token. Run `7615d23d-d572-4db1-a903-9610dd3bcef3` matched the harness digest on create, get, revoke and `enrollment.created` `changes.after.token_sha256`. Consumed and revoked events omit the field. Plaintext was absent from admin reads and the log streams. |
| ENR-03 | closed | w3-wire-nodes #200 | `GET /api/v1/admin/nodes/{node_id}/incarnations` (`listNodeIncarnations`). The same run: a fresh Node has one current incarnation equal to `getNode.current_incarnation_id` and `getEnrollment.node_incarnation_id`; the same-holder retry and the two-key race left the count at one; after `revokeNode` (202) and a second bound enroll, the list is newest-first with count 2, the previous row `retired` / `reinstall` (it was `revocation` before the replacement), usage `"0"`/`"0"` unchanged, cursor round-trip with no duplicates, and 404 for an unknown Node. |
| ENR-04 | closed | devstack follow-up 12 #204 | `JoinEnrolledIdentity` and `EnrollmentPeerInventory` ran on a second fresh stack. Discarded-response race: HTTP 201 and 409 `ENROLLMENT_KEY_MISMATCH`. Replay joined (enroll 200, bootstrap 200). Inventory `complete`: one setup key, `setup_key_uses` 1, one projected peer, `correlation_groups` 0. A second read stayed at one peer. |
| ENR-05 | blocked; follow-up 13 | devstack | Export/document `HoldUsageAcknowledgements`, `UsageLedgerSnapshot`, `CountedMemberTraffic`. Need counted traffic, nonempty unACKed ledger, immutable hashes/decimal sequence numbers and actual contiguous ACKs across installer rerun. Not merged on the source used for run `7615d23d`. |
| ENR-06 | blocked; follow-up 13 | devstack | Export/document `RerunInstaller`, `ControlledReinstall`. Need the real installer, preserved identity/ledger, new authorized token/incarnation, old-cert mTLS rejection and historical Node/usage readback. Not merged on the source used for run `7615d23d`. |
| ENR-07 | closed | devstack follow-up 12 #204 | `MachineIdentityFingerprints` returned distinct machine-id, agent SPKI and NetBird public-key hashes for `node` and `node2`. `StartClonedIdentityFixture` copied the enrolled source and both stages reached `enrolled` with equal hashes. The conflict-row remainder is ENR-08. |
| ENR-08 | open product gap | PKI/enrollment + node identity adapter | `StartClonedIdentityFixture.Observe` read `node_identity_conflicts` and `node_identity_endpoints.mapping_frozen` for the source incarnation. Run `7615d23d` observed `endpoint_rows` 0, `conflicts` 0, `mapping_frozen` false while source and clone shared one identity and both stages were `enrolled`. Devstack follow-up 12 records the same gap: `ObserveEndpoint` has no production caller. Need that caller to write the conflict row and freeze mapping. The harness already consumes the export and does not manufacture rows. |

[API surface review](api-surface-review.md) names the revision 9 reads, the
revision 10 fields that closed ENR-02 and ENR-03, and the follow-up 12 exports
measured here. No SQL read, row repair, fake observation or ad hoc identity
copy is proposed.

The merged lab, as of source `efba7fb`, exports `Enroll`, `StartSecondExit`,
`EnrollAt`, `PauseAgent`, durable restart, `JoinEnrolledIdentity`,
`EnrollmentPeerInventory`, `MachineIdentityFingerprints` and
`StartClonedIdentityFixture`. It does not export the ENR-05/ENR-06 names.
Shared hook documentation: [RUN.md](../../RUN.md).

Live run `7615d23d-d572-4db1-a903-9610dd3bcef3` is the closure measurement.
Run `9bb66c4e-e429-450b-8671-521d33926ede` measured the same token-hash and
incarnation checks on `746651e` before follow-up 12 was in this branch. Run
`bebfe230-26d4-4e3d-b052-81d258d04e07` retains the 429 failures. No current
subcheck failed on `7615d23d`. Complete ATs remain blocked.
