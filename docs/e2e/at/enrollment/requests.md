# Enrollment acceptance requests — follow-up 4

Current binding: `w4-enroll / 12000`. D-050 applies to every request. The
initial 66000 attempts are immutable history, not the current binding.

| ID | State | Owner | Exact remaining evidence / question |
|---|---|---|---|
| ENR-01 | closed | orchestrator / devstack #182 | #182 assigns legal base 12000 to w4-enroll. Active pins use 12000. |
| ENR-02 | closed | pki-enroll #195 | Contract revision 10 `Enrollment.token_sha256` (`^[0-9a-f]{64}$`) is SHA-256 of the exact plaintext token. Run `7615d23d-d572-4db1-a903-9610dd3bcef3` matched the harness digest on create, get, revoke and `enrollment.created` `changes.after.token_sha256`. Consumed and revoked events omit the field. Plaintext was absent from admin reads and the log streams. |
| ENR-03 | closed | w3-wire-nodes #200 | `GET /api/v1/admin/nodes/{node_id}/incarnations` (`listNodeIncarnations`). The same run: a fresh Node has one current incarnation equal to `getNode.current_incarnation_id` and `getEnrollment.node_incarnation_id`; the same-holder retry and the two-key race left the count at one; after `revokeNode` (202) and a second bound enroll, the list is newest-first with count 2, the previous row `retired` / `reinstall` (it was `revocation` before the replacement), usage `"0"`/`"0"` unchanged, cursor round-trip with no duplicates, and 404 for an unknown Node. |
| ENR-04 | closed | devstack follow-up 12 #204 | `JoinEnrolledIdentity` and `EnrollmentPeerInventory` ran on a second fresh stack. Discarded-response race: HTTP 201 and 409 `ENROLLMENT_KEY_MISMATCH`. Replay joined (enroll 200, bootstrap 200). Inventory `complete`: one setup key, `setup_key_uses` 1, one projected peer, `correlation_groups` 0. A second read stayed at one peer. |
| ENR-05 | blocked, waiting on devstack f15 | w4-usage, then devstack | Expected exports: `HoldUsageAcknowledgements(ctx, slot, hold)`, `UsageLedgerSnapshot(ctx, slot)`, `CountedMemberTraffic`. `fvpn-agent usage snapshot [-json]` is merged and is not these hooks. Not emulated. |
| ENR-06 | blocked, waiting on devstack f15 | packaging/installer, then devstack | Expected exports: `RerunInstaller(ctx, slot)`, `ControlledReinstall(ctx, slot, enrollment)`. `fvpn-node-install install --json` and `reenroll --json --token-file` are on main (`#239`) and are not these wrappers. The clone-conflict entrypoint re-enroll is not ENR-06. Not emulated. |
| ENR-07 | measured | devstack follow-up 14 #240 | Fingerprints stay measured. On run `ec2e6279-0649-4d47-b31a-cf13aeba7303`, `StartClonedIdentityFixture(ctx, "node")` presented the node certificate from `fvpn-w4-enroll-node-clone-net` (`172.29.38.0/24`, control `172.29.38.30`, clone `172.29.38.23`). `lab-hosts` maps `agent.example.invalid` to that control address; `add_host` is false. Central recorded one conflict, `mapping_frozen`, one `node.identity.conflict` audit, and a stable count. Forwarding was withdrawn. Operator revoke (202) plus the node entrypoint after the identity files were removed produced a new incarnation and `identity_conflict` `{mapping_frozen: false, conflict_count: 0, last_conflict_at: null}`. The clone's later desired-state request returned 403 `IDENTITY_REVOKED`. |
| ENR-08 | closed; conflict measured | wire-nodes #211 and #217 | `ObserveEndpointTx` runs inside lease renewal and observation submit (#211, `e9f1a70`). `GET /api/v1/admin/nodes/{node_id}` projects `identity_conflict` (#217, `6b98422`). Run `ec2e6279-0649-4d47-b31a-cf13aeba7303` measured the frozen object and the cleared object after revoke plus re-enroll. |

[API surface review](api-surface-review.md) names the revision 9 reads, the
revision 10 fields that closed ENR-02 and ENR-03, and the follow-up 12 exports
measured here. No SQL read, row repair, fake observation or ad hoc identity
copy is proposed.

The merged lab exports `Enroll`, `StartSecondExit`, `EnrollAt`, `PauseAgent`,
durable restart, `JoinEnrolledIdentity`, `EnrollmentPeerInventory`,
`MachineIdentityFingerprints`, `StartClonedIdentityFixture` and
`AttachClonePresentationNetwork` (`tests/e2e/internal/labstack`). It does not
export the ENR-05/ENR-06 names. Shared hook documentation: [RUN.md](../../RUN.md).

Run `7615d23d-d572-4db1-a903-9610dd3bcef3` closed ENR-02, ENR-03 and ENR-04 and
measured fingerprints. Run `9bb66c4e-e429-450b-8671-521d33926ede` measured the
same token-hash and incarnation checks on `746651e` before follow-up 12 was in
this branch. Run `bebfe230-26d4-4e3d-b052-81d258d04e07` retains the 429 failures;
D-078 re-scores them below. Follow-up 3 measurement is run `60a7102e-b824-4927-a576-62033f458157`. Attempt `35e542f8-730e-4be0-8214-e4a9dd818c40` is retained and was not rewritten. Follow-up 4 measurement is run `ec2e6279-0649-4d47-b31a-cf13aeba7303`. Attempt `1d5c02a4-b254-4a9b-b687-b1535428fb3f` is retained and was not rewritten.

## AT06 retained 429 review

Run `bebfe230-26d4-4e3d-b052-81d258d04e07` failed `audit_read`, `issue_reinstall`, `incarnation_history` and `probe_process` with HTTP 429 `RATE_LIMITED` on the admin reads and the enrollment create. Those routes use limiter class `browser`, bucket `BrowserRate` (`internal/config.Defaults`: 10/s, burst 30). Owner of that bucket is control. The harness change that honors `Retry-After` (admin `callHeaders` up to 6, agent enroll `post` up to 4, same `Idempotency-Key`) is `286eb57b44c2e783257767918680a0e1f097f106`, merged as #210.

Run `60a7102e-b824-4927-a576-62033f458157`, scored at the time, did not resolve that review. `rate_limit_review` recorded `http_429` 2, both class `browser`: `GET /api/v1/admin/audit-events` and `GET /api/v1/admin/nodes/{id}/incarnations`. The named checks passed after the retry. Disposition at that time: `recurs`. The raw rows stay in `prior_checks`. D-078, recorded under the 2026-10-31 heading below, re-scores that run and withdraws the zero-count criterion.

Attempt `35e542f8-730e-4be0-8214-e4a9dd818c40` saw the same two 429s and failed the Go process because that check used status `fail`. Those two prior rows stay visible. Under D-078 they stop blocking with the bebfe230 rows when `rate_limit_review` passes the behaviour rule. AT07's `probe_process` fail from bebfe230 stays an unreviewed prior failure. Current AT07 blockers decide that AT.

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
argument. Shared hook documentation: [RUN.md](../../RUN.md).

Run `7615d23d-d572-4db1-a903-9610dd3bcef3` closed ENR-02, ENR-03 and ENR-04 and
measured fingerprints. Run `9bb66c4e-e429-450b-8671-521d33926ede` measured the
same token-hash and incarnation checks on `746651e` before follow-up 12 was in
this branch. Run `bebfe230-26d4-4e3d-b052-81d258d04e07` retains the 429 failures
until the review below applies. Follow-up 3 measurement is run `60a7102e-b824-4927-a576-62033f458157`. Attempt `35e542f8-730e-4be0-8214-e4a9dd818c40` is retained and was not rewritten.

## AT06 retained 429 review

Run `bebfe230-26d4-4e3d-b052-81d258d04e07` failed `audit_read`, `issue_reinstall`, `incarnation_history` and `probe_process` with HTTP 429 `RATE_LIMITED` on the admin reads and the enrollment create. Those routes use limiter class `browser`, bucket `BrowserRate` (`internal/config.Defaults`: 10/s, burst 30). Owner of that bucket is control. The harness change that honors `Retry-After` (admin `callHeaders` up to 6, agent enroll `post` up to 4, same `Idempotency-Key`) is `286eb57b44c2e783257767918680a0e1f097f106`, merged as #210.

Run `60a7102e-b824-4927-a576-62033f458157` did not resolve that review. `rate_limit_review` recorded `http_429` 2, both class `browser`: `GET /api/v1/admin/audit-events` and `GET /api/v1/admin/nodes/{id}/incarnations`. The named checks passed after the retry. Disposition: `recurs`. AT06 stays blocked. Owner: control.

Attempt `35e542f8-730e-4be0-8214-e4a9dd818c40` saw the same two 429s and failed the Go process because that check used status `fail`. Those two prior rows stay visible. They stop blocking only together with the bebfe230 rows, and only when a later run records `http_429` 0. AT07's `probe_process` fail from bebfe230 stays an unreviewed prior failure. Current AT07 blockers decide that AT.

## Devstack follow-up 13 recheck (2026-10-30)

Rechecked on devstack parent `52dc770`. ENR-05 stays blocked. `node/internal/usageupload` still acknowledges every batch. There is no withhold-only-acks switch and no approved read-only ledger snapshot with immutable event hashes, decimal sequence and epoch identity, and the contiguous ACK watermark across restart. `PauseAgent` is not that hook. Do not edit or delete outbox rows. Owner: w4-usage, then devstack. Symbols: `HoldUsageAcknowledgements(ctx, slot, hold)`, `UsageLedgerSnapshot(ctx, slot)`, `CountedMemberTraffic`.

ENR-06 stays blocked. `packaging/` is only the release archive. There is no installer invocation or disposable Ubuntu service entry point to wrap. A container restart, key erasure, or copied identity rows are not that proof. Owner: packaging/installer, then devstack. Symbols: `RerunInstaller(ctx, slot)`, `ControlledReinstall(ctx, slot, enrollment)`.

## Devstack follow-up 14 (2026-10-30)

`StartClonedIdentityFixture(ctx, slot)` is unchanged. The clone now presents that certificate from `fvpn-<prefix>-node-clone-net`, a second IPv4 /24 (lab second octet plus one). `AttachClonePresentationNetwork` attaches that bridge. The process container does not share the netns `/etc/hosts`, and Docker rejects `--add-host` together with `--network container:`. The fixture rewrites the clone's `lab-hosts` so `agent.example.invalid` is the presentation control address; the node entrypoint appends that file. A caller that starts its own process container must do the same rewrite and must not pass `--add-host`. ENR-05 and ENR-06 were not started. The ENR-08 sentence above ("ObserveEndpoint has no production caller") describes run `7615d23d`, before D-072. A later enrollment run should read the admin `identity_conflict` object instead of treating zero SQL conflict rows as the expected gap.

## D-078 behaviour score (2026-10-31)

The criterion that `rate_limit_review` resolves only when a later run records `http_429` 0 is withdrawn. A 429 that carries `Retry-After` and whose retry of the same request succeeds is the limiter working. The review passes when every observed 429 carried `Retry-After`, the next attempt of that request succeeded, every sample class is `browser`, and the join phase stayed at 0. The count and the per-endpoint list stay as evidence. A 429 without `Retry-After`, a retry that is still 429, a non-browser class on this browser-table review, or any join-phase 429 is a finding. Control f21 may still re-calibrate the browser class (`runs/w2-control-core/f21-items.md`). Owner if a finding recurs: control, class `browser`, bucket `BrowserRate` 10/s burst 30.

Run `60a7102e-b824-4927-a576-62033f458157` is re-scored in `prior_failure_review.rescored_follow_up_3` on run `ec2e6279-0649-4d47-b31a-cf13aeba7303`. Raw status stayed blocked, raw `http_429` 2, both class `browser`: `GET /api/v1/admin/audit-events` and `GET /api/v1/admin/nodes/01a18f10-3f31-7ae7-890a-55354e77cec0/incarnations`. Those samples and the count stay in `prior_checks`. The saved samples do not include a `Retry-After` header value (`header_value_recorded` false); none is invented. `audit_read`, `issue_reinstall` and `incarnation_history` passed, and that run's join phase recorded `http_429` 0. The reasoning string is in the JSON. Disposition on `ec2e6279` for AT06 is `resolved`.

On `ec2e6279` itself the full phase recorded `http_429` 2, both class `browser`, `retry_after` true, `retry_succeeded` true: `GET /api/v1/admin/audit-events` and `GET /api/v1/admin/nodes/01a18fc0-5976-7ec9-a193-fbff2be60784/incarnations`. The join phase recorded `http_429` 0 and `samples` `[]`. `rate_limit_review` passed. AT06 passed: 305 checks, 0 blocked, 0 failed.

## ENR-07 second half (2026-10-31)

Pass claims are from run `ec2e6279-0649-4d47-b31a-cf13aeba7303` only. `cloned_identity_conflict` passed.

`AttachClonePresentationNetwork` on `fvpn-w4-enroll-node-clone-netns` returned network `fvpn-w4-enroll-node-clone-net`, subnet `172.29.38.0/24`, control `172.29.38.30`, clone `172.29.38.23`. Lab subnet `172.28.38.0/24`, origin `192.0.11.0/24`. `presentation_differs` true. `lab-hosts` `agent.example.invalid` is `172.29.38.30`. `add_host` false. `same_lab_slash24` and `same_origin_slash24` stayed true because the clone netns remains attached to those bridges (`172.28.38.129` and `192.0.11.23`). The compatibility domain used by the agent is the presentation `/24`.

`Observe` before the clone finished booting returned `read_ok` false. After both stages were `enrolled`: `conflicts` 0, `endpoint_rows` 1, `mapping_frozen` false, `same_identity` true. Inside the 120s window central then recorded `conflicts` 1, `mapping_frozen` true, one `node.identity.conflict` audit. Eight seconds later the audit count was still 1 and `Observe` was still `conflicts` 1, `mapping_frozen` true. `getNode` HTTP 200 projected `identity_conflict` `{mapping_frozen: true, conflict_count: 1, last_conflict_at present}`, `provisioning_state` `failed`, `member_forwarding_observed` false, `safety_status` `failed`.

Recovery used operator revoke (HTTP 202, reason `TEST ONLY ENR-07 clone conflict`) and the node entrypoint after the identity files were removed. That path is not ENR-06. Old incarnation `01a18fc1-4a36-76d7-b0de-b274b6d4486c`, replacement `01a18fc3-386b-716e-960f-fa63faeb507a`, stage `enrolled`. The replacement projection is the object `{mapping_frozen: false, conflict_count: 0, last_conflict_at: null}`, `provisioning_state` `applying`. `Observe` on the old incarnation stayed `conflicts` 1, `mapping_frozen` true, `same_identity` false (source agent SPKI changed). After the clone was started again, its desired-state request with the old certificate returned HTTP 403 `IDENTITY_REVOKED` (`clone_denied` true). The replacement projection stayed cleared.

Attempt `1d5c02a4-b254-4a9b-b687-b1535428fb3f` observed the same conflict shape and HTTP 403, and wget omitted `--content-on-error`, so the error code was empty and the probe exited nonzero. That directory is unchanged. The `IDENTITY_REVOKED` pass is not claimed from it. AT07 `prior_failure_review` disposition is `not_applied`: the review names that attempt's `cloned_identity_conflict` and `probe_process`, and `bebfe230` plus `35e542f8` AT07 `probe_process` rows stay unreviewed. Current AT07 blockers are the six ENR-05/ENR-06 checks.

## ENR-05 / ENR-06 (2026-10-31)

Both stay blocked, waiting on devstack f15. Expected exports: `HoldUsageAcknowledgements(ctx, slot, hold)`, `UsageLedgerSnapshot(ctx, slot)`, `CountedMemberTraffic`, `RerunInstaller(ctx, slot)`, `ControlledReinstall(ctx, slot, enrollment)`. `fvpn-agent usage snapshot [-json]` is merged and is not the usage-hold hooks. `fvpn-node-install` (`install --json`, `reenroll --json --token-file`) is on main (`#239`) and is not the installer wrappers. The clone-conflict entrypoint re-enroll is not ENR-06. Not emulated. AT07 stays blocked, so `make -f mk/at-enrollment.mk at-enrollment` exits 2 by design.

## Devstack follow-up 15 response (2026-10-31)

ENR-05: `HoldUsageAcknowledgements(ctx, slot, hold)` is a network-side availability fault at the node's mTLS proxy: only POST `/api/v1/agent/usage-batches` receives retryable 503 without reaching control. Releasing the marker preserves the central ledger epoch. Control-side freeze/recovery is a different protocol and mints a new epoch on completion. `UsageLedgerSnapshot` decodes the merged read-only CLI into `UsageSnapshot`; `UsageLedgerSnapshotBytes` exposes exact stdout for repeat comparison. Decimal sequences and byte values stay strings. `CountedMemberTraffic` reads the selected device counters; `StartEstablishedTransports` supplies real traffic. The exports suite checks endpoint scope, stable repeated reads, real accumulation, restart hash preservation and ACK catch-up. No outbox rows are edited.

ENR-06: `PrepareInstallerNode`, `StageInstallerToken`, `RerunInstaller` and `ControlledReinstall` target only slot `installer-node`, a dedicated privileged Ubuntu 24.04 systemd container with the pinned real NetBird client and lab control enrollment. `InstallerOutcome` returns the actual exit, status/code, identity IDs and before/after hashes (including WAL when present and current symlink text). Agent/watchdog/updater services are stopped before hash sampling. `SelectInstallerRelease` selects the staged older/newer signed fixture; `ProbeInstallerExit` retains nonzero outcomes including `staged` plus failure. The follow-up 15 installer record is authoritative for live product failures; wrapper availability is not an AT07 pass. Old-certificate rejection remains ENR-02. `e2e-down` owns the installer container and image.

ENR-01/04 remain the existing slot and enrolled-identity exports. ENR-07 was delivered in #240: `StartClonedIdentityFixture` presents the copied certificate on the second /24 and the exports scenario reads the D-072 identity conflict. The retained browser-class 429 review belongs to control; this follow-up does not override or skip rate limits. ENR-02/03/08 conclusions remain the enrollment harness's.

Live ENR-06 diagnosis (2026-10-31): real enrollment succeeded in the dedicated systemd node, then install returned exit 1 because `fvpn-netbird.service` cannot bind `/var/run/netbird.sock` under its `ProtectSystem=strict` write allowlist. The agent also logged `mkdir /run/vpn-node: read-only file system` during service restart. Owner: packaging/installer and `deploy/node/systemd`. `tests/installer` uses a NetBird stub, so its proof does not exercise this socket. No service override or stub is used here. Successful idempotent re-run/newer-archive/controlled-reenroll retention assertions remain unreached until the product service startup succeeds; the wrapper surfaces the actual failure.
