# Node updater

`fvpn-updater` (`node/updater/cmd/fvpn-updater`) applies one node's authorized update. It downloads a signed release, checks it, and switches through the update journal. A crash resumes from the last durable stage. Member forwarding stays off while a transaction is applying or checking, and stays off after `recovery_required` (§13.5, §13.9).

The updater never executes a file from a manifest. It never changes `publication_intent`. It never deletes identity or the usage outbox to make space (§13.7). A local result stays in the outbox until the center accepts it. Local success is not acceptance (§13.6 step 13).

The central fleet orchestration, the agent process and the installer are separate. The updater talks to them through the interfaces in `node/updater/interfaces.go`.

## Commands

```text
fvpn-updater run     -config updater.json
fvpn-updater recover -config updater.json
fvpn-updater flush   -config updater.json
fvpn-updater version
```

`recover` runs before member services start. Exit 0 means the journal does not require forwarding to stay off. Exit 2 means members must stay blocked (`ErrMembersBlocked`). Exit 1 is any other failure.

`run` calls `recover` first, then takes the current desired state. `flush` posts pending results and does not start an update. `flush` requires `result_url`.

The config file holds paths and ids. It does not hold the release private key. `allow_loopback` is TEST ONLY and must stay false on a node. Stock health checks (`NodeChecks`) compare the release tree and the identity and state trees. Isolation, DNS and the tunnel probe fail closed until the agent supplies its own checks (§13.9). An empty stop or start command fails closed. The updater does not guess systemd.

The forwarding latch is `forwarding.blocked` in the journal directory. The agent must not enable member forwarding while that file exists or while `BlocksForwarding` is true. The nftables gate itself belongs to the agent.

## What the updater accepts

Work comes from a desired-state envelope (§14.7). `FileSource` verifies it on every read with a `desired-state` purpose key. The window is half-open `[issued_at, expires_at)`. The node id and incarnation must match. An unverifiable or expired envelope blocks forwarding. A valid envelope with `update: null` is not a job (`ErrNoWork`) and does not by itself change the latch.

The desired-state document has `serving_release_ids` and no revoked-release list. `revoked_release_ids` in the updater config is the release-support revoked set, copied by the agent. The result's `policy_generation` is the desired-state `config_generation`. `boot_id` on a result is the boot that claimed the job.

A job is refused when the attempt is below 1, the node is revoked, the incarnation does not match, the release is revoked, the generation is older than the accepted one, or the authorization window has ended. Authorization is half-open: `now < authorization_valid_until`. Those refusals are recorded. A stale or revoked job is not claimed.

`already_current` is reported only when the current release matches the job, the manifest still verifies, compatibility holds, and the release is in the serving set. Services are not restarted (§13.6). A current release that is outside the serving set is `MINIMUM_VERSION_VIOLATION` and is not restarted.

## State machine

Journal stages (§13.6 steps 6 to 12, plus the rollback stage):

| Stage | Meaning | Member forwarding |
|---|---|---|
| `prepared` | Journal row written. `last_good` and the state schema are recorded. | blocked by the journal |
| `blocked` | Outer forwarding block is active. | blocked |
| `final_sampled` | Final usage sample is durable. | blocked |
| `stopped` | Replaceable services are stopped. The stop is destructive. | blocked |
| `migrated` | State migration finished, or none was required. | blocked |
| `switched` | `current` points at the new release. | blocked |
| `checking` | New services are up. Health checks run. | blocked |
| `rolling_back` | §13.9 rollback is in progress. This stage is not in the §13.6 list. It is durable so a crash can finish the rollback. | blocked |
| `locally_succeeded` | Every check passed and `last_good` moved. | allowed only when publication is `published` and the release is in the serving set |
| `rolled_back` | A verified earlier release is current again, or the old process was still intact. | same publication rule, when `last_good` is safe |
| `recovery_required` | No safe release can serve. | stays blocked |

`checking` and `rolling_back` may repeat. Every non-terminal stage may move to `rolling_back` or `recovery_required`. The three terminal stages do not reopen.

A newer generation while the job is queued, downloading or verifying supersedes it and keeps a verified artifact (§13.8). A newer generation during applying or checking waits until the open transaction is terminal.

`publication_intent` is copied onto the result and never edited. `disabled` and `unpublished` do not unblock. An empty serving set does not unblock. A candidate that is not in the serving set does not unblock, even after local success.

### Rollback decision (§13.9)

`last_good` is restored only when all of these hold, in this order: a tree is present and its hash matches, the manifest signature verifies, the release is not revoked, the release is in the serving set, and the state schema is readable by that release. Otherwise the outcome is `recovery_required` and members stay blocked.

When `last_good` is unsafe, the `current` symlink is left on the failed candidate. Forwarding stays off. The symlink is not moved to an unverified tree.

A failure before services are stopped (for example the metering barrier) is not destructive. The journal ends `rolled_back`, the result state is `failed`, and forwarding follows the publication rule because the old process is still the one that is running. A stop failure is reported as `START_FAILED` with stage `stop_services` (there is no `STOP_FAILED` code) and the rollback is destructive, so `last_good` is started again.

`Block` failing does not finish the transaction. The stage stays `prepared`, the error is `UPDATE_BUSY`, and the journal still reports forwarding blocked.

Health checks are capped at 8 attempts and rollbacks at 3. Those caps are local safety limits (待實測), stored beside the journal as `checks-<txn>` and `rollback-<txn>`.

## Journal format

The journal is SQLite in WAL mode with `synchronous=FULL`, one connection, at `/var/lib/vpn-node/update-journal/journal.db` (mode 0600, directory 0700). `synchronous=FULL` is the journal fsync. The order of filesystem flushes around it is 待實測.

Tables: `meta`, `jobs`, `releases`, `txns`, `txn_events`, `artifacts`, `result_outbox`, `result_history`. Schema version is `PRAGMA user_version`. A newer version is refused.

`txns.stage` is the stage above. `destructive` sticks once set. `txn_events` is the append-only history. Releases are immutable. `current` and `last_good` are meta pointers.

The result outbox stores the canonical JSON and `payload_hash` (`canonjson.HashHex` of the result without `payload_hash`). `result_id` is UUIDv5 over `fvpn-update-result|<job>|<attempt>|<state>` so a replay sends the same id. UUIDv7 is not replay-stable. The same id and hash is a no-op. A different hash is `ACK_CONFLICT` and is kept, not resent. A late duplicate ACK is a no-op.

`POST /agent/update-results` is answered with 202 in the contract. 409 is a conflict. Anything else leaves the row pending.

Sidecar files next to the database (not inside it): `identity.snap` and `state.snap` (tree hashes taken before the transaction), `epoch-<txn>` (the new meter epoch, so `OpenEpoch` is idempotent), the check and rollback counters, and `forwarding.blocked`.

The single-updater lock is `/var/lib/vpn-node/update.lock`, a `flock` held by `node/internal/nodelock`. The agent, the installer and the updater take that same lock.

## Download and verify (§13.4)

The manifest hash is the SHA-256 of the canonical JSON of the whole envelope. The artifact hash is the raw file. A partial download resumes only when the content hash is unchanged and the server's range agrees with the bytes already stored. A changed body deletes the partial and downloads the whole file again. One content retry is allowed. A second hash or signature failure is permanent and is not retried. Network errors use bounded backoff with full jitter. The default budget (待實測) is 5 attempts, 200 ms initial, 30 s cap, 60 s per attempt.

Verification checks the release trust store, both Ubuntu 24.04 architectures (`amd64` and `arm64`, `tar.zst`), this node's artifact, size, SHA-256, control API major, policy schema, usage schema, state schema, minimum updater version and the disk reserve (`artifact + installed size + required_free_bytes`). `product_version` is display only. `minimum_updater_version` is compared as three dotted numeric components, so `1.10.0` is newer than `1.9.0`.

URLs must be `https`. Loopback, link-local and the metadata host names are refused. `allow_loopback` is the test exception. The resolved address is not pinned.

Extraction rejects absolute paths, `..`, NUL, backslash, device files, fifos, sockets, hard links, setuid/setgid/sticky modes, symlinks that point outside the release, and any path that overlaps identity, state or the journal. A relative symlink inside the tree is allowed. Nothing in the archive is executed.

`current` is a relative symlink, switched by rename in the same directory, then the parent directory is fsynced. A different filesystem fails closed.

Retention keeps `current`, `last_good`, the open transaction, open jobs and any release named by an unacknowledged result. It does not delete identity or the usage outbox.

## Crash recovery (AT25)

The hook used by tests fires after the stage is durable and before that stage's side effect. On the next start the journal is read before member services. A non-terminal stage keeps forwarding blocked and the transaction is finished or rolled back.

| Crash after this stage is durable | Restart ends in | Members |
|---|---|---|
| `prepared` | `locally_succeeded` | blocked until checks pass |
| `blocked` | `locally_succeeded` | blocked until checks pass |
| `final_sampled` | `locally_succeeded` | blocked until checks pass |
| `stopped` | `locally_succeeded` | blocked until checks pass |
| `migrated` | `locally_succeeded` | blocked until checks pass |
| `switched` | `locally_succeeded` | blocked until checks pass |
| `checking` | `locally_succeeded` | blocked until checks pass |
| `locally_succeeded` (before the result is enqueued) | result `succeeded` in the outbox | not blocked by the journal |
| `rolling_back` | `rolled_back` when `last_good` is safe | blocked until the restored release checks pass |
| `rolled_back` (before the result is enqueued) | result `rolled_back` | not blocked when the restored release may serve |
| `recovery_required` (before the result is enqueued) | result with `recovery_required` | stay blocked, exit 2 |

Identity and the usage outbox are not rewritten by the update or the rollback (§13.7). The meter `FinalSample` and `OpenEpoch` calls are idempotent. The updater also stores the epoch id in `epoch-<txn>` so a second process does not open a second epoch.

## Decisions

- `boot_id` is the boot that claimed the job. Keep.
- Hostile archive paths, devices and symlink escapes stay `VALIDATION_FAILED`. A dedicated code is deferred to the next contract revision.
- Refusing loopback, link-local and metadata names is sufficient for this phase. Pinning the resolved address is not required. Artifact fetches move behind the agent's mTLS client in Wave 3.
- Update results are accepted on HTTP 202 only. The contract wins over the handover text.
- `revoked_release_ids` stay an unsigned agent-supplied list for now. The signed serving set is the only thing that gates forwarding. Adding the list to the signed desired state is a candidate for contract revision 3.
- `policy_generation` is the desired-state `config_generation` for now (revision 3 candidate).
- When `last_good` cannot be restored, `current` stays on the failed candidate, the symlink is never moved to an unverified tree, and members stay blocked. Confirmed.
- A checkpoint failure before `stop_services` ends as `rolled_back`/`failed` without blocking members, because nothing destructive happened. Confirmed: §13.9 step 4 applies only after the destructive stage.
- Toolchain 1.27.3 arrives in a separate orchestrator task (decision D-014). Keep `GOTOOLCHAIN=local` and `go.mod` at 1.27.1.
