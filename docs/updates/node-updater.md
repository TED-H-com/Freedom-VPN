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

`recover` runs before member services start.

| Exit | Meaning |
|---|---|
| 0 | The journal does not require forwarding to stay off. `help` and `version` also exit 0. |
| 2 | Only `ErrMembersBlocked`. Members must stay blocked. |
| 1 | Any other failure, including a config that does not load and `flush` without `result_url`. |
| 64 | Usage, an unknown command, a bad flag, or a missing `-config`. |

`run` calls `recover` first, then takes the current desired state. `flush` posts pending results and does not start an update. `flush` requires `result_url`. `Run`, `Recover`, `Flush` and `Retain` take the node lock at entry, before they supersede an older job, claim, download or install.

The config file holds paths and ids. It does not hold the release private key. `allow_loopback` is TEST ONLY and must stay false on a node. `allowed_artifact_origins` is a list of `scheme://host[:port]` origins (D-046). An artifact, manifest or `result_url` whose origin matches an entry is exempt from the private and CGNAT address guard for that origin only, and the dial is pinned to the first address that host resolved to. Link-local, multicast, unspecified and metadata names stay refused. The default is empty, so the guard stands. `allow_loopback` does not allow private addresses. A compose dev stack sets the release and result origins explicitly. An entry has no user, path, query or fragment. An entry matches a URL when the scheme, host and effective port are equal. The effective port of a URL or entry without an explicit port is the scheme default: 443 for https and 80 for http. Stock health checks (`NodeChecks`) compare the release tree, the identity directory, and an immutable record of the pointers and the state-schema marker. They do not hash the mutable state directory. Isolation, DNS and the tunnel probe come from a challenge-bound attestation written by the candidate agent (D-053, §13.9). Without a valid attestation those three checks fail. A config field, an environment variable or a file that was already in the journal cannot mark them passed. The file contract is [`node-health-attestation.md`](node-health-attestation.md). An empty stop or start command fails closed. The updater does not guess systemd. `Config.Engine` installs a migrator that refuses a schema change, so a release that needs one fails before services stop when no real migrator was supplied.

The forwarding latch is `forwarding.blocked` in the journal directory. The agent must not enable member forwarding while that file exists or while `BlocksForwarding` is true. The nftables gate itself belongs to the agent.

## What the updater accepts

Work comes from a desired-state envelope (§14.7). `FileSource` verifies it on every read with a `desired-state` purpose key. The window is half-open `[issued_at, expires_at)`. The node id and incarnation must match. An unverifiable or expired envelope blocks forwarding. A valid envelope with `update: null` is not a job (`ErrNoWork`) and does not by itself change the latch.

The desired-state document has `serving_release_ids`. A verified envelope may also carry signed `revoked_release_ids` (revision 3, D-034). An absent member means no additional revocations and does not change the persisted signed set. An empty array adds nothing and does not clear that set. `revoked_release_ids` in the updater config stays the unsigned release-support list copied by the agent. The effective set is the union of that list and the persisted signed set. The result's `policy_generation` is the desired-state `config_generation` actually applied when the result was produced (D-043, §13.10.3). It is not the update job generation and not a generation that was only downloaded. A result is not posted before that generation is known. `boot_id` on a result is the boot that claimed the job.

A job is refused when the attempt is below 1, the node is revoked, the incarnation does not match, the release is revoked, the generation is older than the accepted one, or the authorization window has ended. Authorization is half-open: `now < authorization_valid_until`. Those refusals are recorded. A stale or revoked job is not claimed.

`already_current` is reported only when the current release matches the job and `last_good`, the journal is not holding forwarding, the last terminal outcome is not `failed`, `rolled_back` or `recovery_required`, the manifest still verifies, compatibility holds, and the release is in the serving set. Services are not restarted (§13.6). A manifest, compatibility or serving-list failure on that path blocks the gate before the function returns. A current release that is outside the serving set is `MINIMUM_VERSION_VIOLATION`, the gate is blocked, and services are not restarted. `recovery_required` is never reported as `already_current`.

The latest publication intent and the forwarding-blocked bit live in one journal record (`publication_intent`, `forwarding_blocked`, `terminal_outcome`), written in the same step that changes them. `BlocksForwarding` and `reapplyPublication` read that record. They do not replay a stale `LastTxn`. There is no `failed` stage. A failed job stores the terminal outcome `failed`, and that outcome blocks forwarding. A verified `rolled_back` does not, by itself, keep members off.

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

Rollback chooses the newest retained release that passes the checks below. Newest is `release_sequence`, then `installed_at`, then release id. Retained means a release the journal still records and whose tree retention has not removed. A revoked release is never selected and is never installed. The checks, in this order, are: a tree is present and its hash matches, the manifest signature verifies, the release is not revoked, the release is in the serving set, and the state schema is readable by that release. When the failed candidate's `rollback.allowed_release_ids` is non-empty, the target must also be in that list. The serving set and the allow list fail closed: missing either one is `MINIMUM_VERSION_VIOLATION`. An empty allow list adds no extra restriction. `StateReadableByPrevious` does not mark a schema readable when the journal schema is not that release's schema. A health check past the 8-attempt cap starts a rollback. It does not jump straight to recovery.

A currently installed release named by the persisted signed set is unhealthy. The updater rolls it back to that newest non-revoked release. When no such release exists, the outcome is `recovery_required`, member forwarding stays blocked, and the revoked release does not pass the gate. The unsigned agent list still refuses installation. It does not, by itself, roll the current release back, so a legacy envelope keeps today's behaviour. The persisted signed set is a monotonic union: an id is never removed, an older `config_generation` cannot shrink it, and an absent member does not update it.

When no safe retained release can be restored, the `current` symlink stays where it is (the failed candidate, or the revoked release). Forwarding stays off. The symlink is not moved to an unverified tree.

A failure before services are stopped (for example the metering barrier) is not destructive. The journal ends `rolled_back`, the result state is `failed`, and forwarding follows the publication rule because the old process is still the one that is running. A stop failure is reported as `START_FAILED` with stage `stop_services` (there is no `STOP_FAILED` code) and the rollback is destructive, so `last_good` is started again.

`Block` failing does not finish the transaction. The stage stays `prepared`, the error is `UPDATE_BUSY`, and the journal still reports forwarding blocked.

Health checks are capped at 8 attempts and rollbacks at 3. Those caps are local safety limits (待實測), stored beside the journal as `checks-<txn>` and `rollback-<txn>`.

### Health attestation (D-053, §13.9)

Before `Services.Start` in the check stage, the updater writes `health-challenge.json` in the journal directory. The candidate agent answers with `health-attest.json` in that same directory. The updater accepts the answer only when the nonce and transaction match the live challenge, the release, node, incarnation and boot ids match, `generated_at` is after `issued_at` and before `expires_at`, the updater's clock is still before `expires_at`, the file was written after the challenge, `agent_version` equals the manifest's `node_agent`, `netbird_version` equals the manifest's `netbird`, and each status is a known `HealthCheckStatus`. Anything else fails closed. The typed reasons and the write rules are in [`node-health-attestation.md`](node-health-attestation.md).

`stepStart` may start the candidate before this file exists. The check stage calls `Start` again after the challenge is on disk. That second start is the signal that the challenge is live. A resume of `checking` writes a new nonce. An answer for the previous nonce does not match. The attempt past the 8-attempt cap starts rollback and does not issue a candidate challenge.

On a destructive rollback the updater stops the candidate, writes a new challenge for the restored release, then starts that release. The restored agent answers that challenge. A non-destructive rollback does not run this check again.

`not_run` from the agent is `failed` for the updater. The probe is mandatory for unpublished and disabled nodes (§13.9). A binding `not_run` ends that attempt. A missing or mismatched file keeps being read until the wait window, so a corrected rewrite inside the window can still pass. At the deadline the last reason is journaled and the three checks are `failed`. When the attestation itself does not validate, `processes` is `failed` as well, because the versions were not attested. `identity_and_state` stays the identity directory and the immutable pointers record (D-034c).

A passing check sets `last_good` and reaches `locally_succeeded`. Member forwarding is still the latest publication intent plus the serving policy (§13.9 服務狀態恢復). The attestation's `published` field does not open the gate. Health does not publish an unpublished or disabled exit, and a release outside the serving set stays blocked.

`txns.health_json` is the first health report, the candidate's. Later stages do not replace it. Schema `user_version` stays 1. Reasons that do not fit `UpdateHealth` are an append-only `txn_events` row, `kind` `health_attest`. Rollback health is a second row with `phase` `rollback`. The `UpdateResult.health` object (§13.10.3) is the candidate report: statuses only. When no retained release can be restored, the terminal code is `ROLLBACK_FAILED` and the candidate statuses and reasons remain. When `last_good` is restored and its own check passes, the stage is `rolled_back` and the transaction error stays `HEALTH_CHECK_FAILED`.

Which component versions are verifiable at which stage:

| Component | During `checking` and during rollback health |
|---|---|
| `node_agent` | The attested `agent_version` must equal the manifest of the release under test, and the release tree hash must match. |
| `netbird` | The attested `netbird_version` must equal that manifest, and the tree hash must match. |
| `node_updater` | Tree hash only. The process running the check is the previous release's updater, or the supervisor binary. The challenge still carries the manifest version. The detail field is `updater_version_check=tree_hash_only`. |
| `enforcement` | Tree hash only. The attestation has no running enforcement version. The challenge still carries the manifest version. The detail field is `enforcement_version_check=tree_hash_only`. |

### Parameters

These windows are code constants. The updater config cannot set them. They are suggested defaults pending measurement (待實測), not a verified SLO. When a test or a later measurement sets `Engine.Attest` and the TTL is not greater than the wait, the updater uses wait plus one poll interval.

| Name | Default | Role |
|---|---|---|
| `HealthAttestWait` | 30s | How long one check attempt waits for a valid attestation. |
| `HealthAttestPoll` | 250ms | How often that attempt reads `health-attest.json`. |
| `HealthChallengeTTL` | 45s | `expires_at` is `issued_at` plus this. It is longer than the wait so an answer written at the end of the window is still inside the challenge. |
| check attempts | 8 | `checks-<txn>`. The next attempt starts rollback and does not issue a candidate challenge. |
| rollback attempts | 3 | `rollback-<txn>`. The next attempt is `recovery_required`. |

## Journal format

The journal is SQLite in WAL mode with `synchronous=FULL`, one connection, at `/var/lib/vpn-node/update-journal/journal.db` (mode 0600, directory 0700). `synchronous=FULL` is the journal fsync. The order of filesystem flushes around it is 待實測.

Tables: `meta`, `jobs`, `releases`, `txns`, `txn_events`, `artifacts`, `result_outbox`, `result_history`. Schema version is `PRAGMA user_version`. A newer version is refused.

`txns.stage` is the stage above. `destructive` sticks once set. `txn_events` is the append-only history. Releases are immutable. `current` and `last_good` are meta pointers.

The result outbox stores the canonical JSON and `payload_hash` (`canonjson.HashHex` of the result without `payload_hash`). `result_id` is UUIDv5 over `fvpn-update-result|<job>|<attempt>|<state>` so a replay sends the same id. UUIDv7 is not replay-stable. The same id and hash is a no-op. A different hash is `ACK_CONFLICT` and is kept, not resent. A late duplicate ACK is a no-op.

`POST /agent/update-results` is answered with 202 in the contract. 409 is a conflict. Anything else leaves the row pending.

Sidecar files next to the database (not inside it): `identity.snap` (identity tree hash) and `immutable.snap` / `immutable.record` (pointers and the state-schema marker). Both are taken after migration and before services start, and again for the pointers only after a rollback restores `last_good`. The mutable state directory is not in that record. `epoch-<txn>` is the new meter epoch for that transaction. It lives in the journal directory, not under state, and it is removed when the transaction completes. `FileMeter.OpenEpoch` does not write a sidecar. The check and rollback counters and `forwarding.blocked` are the other durable sidecars. `health-challenge.json` and `health-attest.json` are short-lived. The updater removes both after it records health. If the remove fails, it rewrites the challenge with `consumed: true`, which a later validation rejects. The next check overwrites the challenge with a new nonce.

The single-updater lock is `/var/lib/vpn-node/update.lock`, a kernel `flock(LOCK_EX|LOCK_NB)` held by `node/internal/nodelock`. The descriptor stays open until `Release`. Holder metadata (`pid`, `boot_id`, `start_time`) is written only after the lock is held. `start_time` is field 22 of `/proc/<pid>/stat` (clock ticks since boot), counted after the last `)` of the command name. A stale pid, a different boot id, or the same pid with a different start time does not steal a lock the kernel still holds, and the file is not removed on contention. A live flock is `UPDATE_BUSY`. A held lock whose record is stale is also `UPDATE_BUSY`, and the error names the reason (`boot id`, `pid`, `process` or `start time`). The agent, the installer and the updater take that same lock.

## Download and verify (§13.4)

The manifest hash is the SHA-256 of the canonical JSON of the whole envelope. The artifact hash is the raw file. `.partial.meta` is written when the `200` or `206` headers arrive, before the body, so an interrupted download still resumes with `If-Range`. A partial download resumes only when the content hash is unchanged and the server's range agrees with the bytes already stored. A `206` whose `ETag` differs from the ETag stored in `.partial.meta` is not appended: the partial is deleted and the attempt fetches the object again from byte zero. A `200` response to a `Range` request is the full representation (RFC 9110 §13.1.5, or a server that ignores `Range`): that body replaces the partial, the download does not fail with `ARTIFACT_INVALID`, and a body that still does not match costs at most one content retry. When `Content-Length` is known, the bytes written must equal it (`200`) or the prior offset plus it (`206`). A changed body deletes the partial and downloads the whole file again. One content retry is allowed. A second hash or signature failure is permanent and is not retried. Network errors use bounded backoff with full jitter. The default budget (待實測) is 5 attempts, 200 ms initial, 30 s cap, 60 s per attempt. Manifest downloads are capped at `canonjson.MaxBytes` (1 MiB).

Verification checks the release trust store, both Ubuntu 24.04 architectures (`amd64` and `arm64`, `tar.zst`), this node's artifact, size, SHA-256, control API major, policy schema, usage schema, state schema, minimum updater version and the disk reserve (`artifact + installed size + required_free_bytes` on the releases filesystem, and the artifact size on the staging filesystem). A duplicate matching platform entry is `VALIDATION_FAILED`. `product_version` is display only. `minimum_updater_version` is SemVer 2.0.0: three numeric components, no leading zeros, build metadata ignored, and a pre-release is older than the same core without one. `1.10.0` is newer than `1.9.0`. `1.0.0-alpha.1` is older than `1.0.0`.

URLs must be `https`. Loopback, link-local, multicast, unspecified, private (RFC 1918 and IPv6 ULA), CGNAT (`100.64.0.0/10`) and the metadata host names are refused. `allow_loopback` is the test exception and permits loopback only. The same check runs on every address `DialContext` resolves, so a public name that answers with a private address is refused (D-034f). An entry in `allowed_artifact_origins` exempts a URL from the private and CGNAT checks only when the scheme, host and effective port are equal (D-046). The effective port of a URL or entry without an explicit port is the scheme default: 443 for https and 80 for http. The connection is pinned to the first resolved address of that host. Every other origin keeps the guard.

Extraction accepts regular files and directories only (D-047). It rejects absolute paths, `..`, NUL, backslash, device files, fifos, sockets, hard links, symlinks (including a relative symlink inside the tree), every other member type, setuid/setgid/sticky modes, more than 4096 members, and any path that overlaps identity, state or the journal. A directory entry that cleans to `.` (`./` from `tar -C dir -cf … .`) is the archive root. Any `.` or `..` segment other than that archive-root `./` is rejected, including a `..` that `path.Clean` would fold back inside the tree. `MaxBytes` caps the sum of extracted regular-file bytes (`installed_size_bytes`). The decompressed tar stream is capped separately at `MaxBytes + MaxEntries*8KiB + 2KiB + 1`: 8KiB per member covers a PAX header, the 512-byte ustar header and block padding, and 2KiB covers the two 512-byte end blocks, each with margin. `fvpn-release` rejects symlinks, hard links and devices at pack time, so that failure happens on the build machine. Overlap is symmetric and is checked before the release directory is removed. Nothing in the archive is executed. The updater's own `current` symlink stays outside the release tree.

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
| `rolling_back` | `rolled_back` when a retained release is safe | blocked until the restored release checks pass |
| `rolled_back` (before the result is enqueued) | result `rolled_back` | not blocked when the restored release may serve |
| `recovery_required` (before the result is enqueued) | result with `recovery_required` | stay blocked, exit 2 |

Identity and the usage outbox are not rewritten by the update or the rollback (§13.7). The meter `FinalSample` is read from the checkpoint the meter wrote. `OpenEpoch` returns a new id and writes nothing into state. The updater stores that id in the journal note `epoch-<txn>` so a resume does not open a second epoch, and deletes the note when the transaction completes.

## Decisions

- `boot_id` is the boot that claimed the job. Keep.
- Hostile archive paths, devices and symlink escapes stay `VALIDATION_FAILED` on the local operation. The UpdateResult reports those failures, and artifact hash, signature and size failures, as `ARTIFACT_INVALID`. That code is report-only: its catalog HTTP status is zero and it is not an HTTP status. Hash and size stay permanent after the bounded re-download. Signature and archive validation stay permanent. The local `OpError` and its retry bit are unchanged.
- Refusing loopback, link-local and metadata names was the original bar, and pinning one resolved address was not required. D-034f supersedes the address check: private and CGNAT ranges are refused, and `DialContext` applies that check to every resolved address. D-046 exempts an origin listed in `allowed_artifact_origins` from the private and CGNAT checks and pins the dial to that host's first resolved address. Artifact fetches move behind the agent's mTLS client in Wave 3.
- Update results are accepted on HTTP 202 only. The contract wins over the handover text.
- Signed `revoked_release_ids` on a verified desired-state envelope (revision 3) are persisted as a monotonic union in the journal. An id is never removed (§17.2). The member is accepted only when `config_generation` is not older than the last applied generation; an older generation is `STALE_GENERATION` and does not shrink the set or return a revoked release to service (§13.4, §15.3). An absent member does not update the signed set. An empty member adds nothing and does not clear it. The updater never installs a revoked release, never selects one as a rollback target, and does not let one pass the forwarding gate. A signed-revoked current release stops member forwarding immediately (`recovery_required`). An update job whose intent release is not revoked still downloads, verifies, installs and health-checks that release; serving resumes only after those checks. When there is no such job, or the intent release is itself revoked, the node rolls back to the newest retained release that is not revoked, or stays in `recovery_required` with forwarding blocked. A later job for a non-revoked release can still rescue it.
- `policy_generation` is the desired-state `config_generation` actually applied when the result was produced (D-043, §13.10.3). It is not the update job generation and not a desired state that was only received. A result is not posted before that generation is known. A superseded job keeps the generation it applied, not a newer generation accepted for a later job.
- When no safe retained release can be restored, `current` stays on the failed candidate or the revoked release, the symlink is never moved to an unverified tree, and members stay blocked. Confirmed.
- A checkpoint failure before `stop_services` ends as `rolled_back`/`failed` without blocking members, because nothing destructive happened. Confirmed: §13.9 step 4 applies only after the destructive stage.
- Toolchain 1.27.3 arrives in a separate orchestrator task (decision D-014). Keep `GOTOOLCHAIN=local` and `go.mod` at 1.27.1.
- D-034a. Release archives contain regular files and directories only. `ExtractArchive` rejects symlinks, hard links and every other member type. `fsutil.TreeHash` returns an error on a symlink. The `current` pointer stays a symlink outside the tree.
- D-034b. Publication intent and the forwarding-blocked state are one journal record, written in the same step that changes them. `reapplyPublication` and `BlocksForwarding` read that record, not a stale `LastTxn`. `maybeAlreadyCurrent` does not short-circuit when `current` is not `last_good`, when forwarding is blocked, or when the last terminal outcome is `failed`, `rolled_back` or `recovery_required`. A manifest, compatibility or serving-list failure blocks the gate before it returns. Outcome `failed` is blocking. There is no `failed` stage.
- D-034c. Persistent data is the identity directory, the pointers record and the state-schema marker, snapshotted after migration and before start. The meter epoch hand-off is the journal note `epoch-<txn>`, removed when the transaction completes. `Config.Engine` supplies a migrator that fails a schema change before services stop.
- D-034d. `nodelock.Acquire` takes `flock(LOCK_EX|LOCK_NB)` before it writes pid, boot id and process start time, and keeps the descriptor until `Release`. Stale detection compares boot id, pid and `/proc/<pid>/stat` field 22, and does not steal a held lock. A held lock with a stale record returns `UPDATE_BUSY` and names the reason. `Run`, `Recover`, `Flush` and `Retain` acquire the lock at entry.
- D-034e. A non-empty `allowed_release_ids` is a fail-closed intersection with `serving_release_ids`. Rollback does not override `SchemaReadable=false`. The health-check cap goes through rollback. `versionAtLeast` is SemVer 2.0.0, including pre-release ordering.
- D-034f. `.partial.meta` is written when the success headers arrive. Manifest downloads are capped at `canonjson.MaxBytes`. A known `Content-Length` must match the bytes written. A `206` with a different ETag restarts from byte zero. A `200` to a `Range` request is kept as the full representation. Archives have an entry cap. `MaxBytes` caps extracted file bytes, and the decompressed stream cap is `MaxBytes + MaxEntries*8KiB + 2KiB + 1`. `pathsOverlap` is symmetric and is checked before `RemoveAll`. `checkURL` rejects private and CGNAT ranges, and `DialContext` checks resolved addresses. `ensureSpace` also checks staging.
- D-046. `allowed_artifact_origins` is empty by default. An entry matches a URL when the scheme, host and effective port are equal. The effective port of a URL or entry without an explicit port is the scheme default: 443 for https and 80 for http. A match exempts that origin from the private and CGNAT guard only, and the dial is pinned to the first resolved address of that host. `allow_loopback` does not imply private networks.
- D-047. Release trees contain only regular files and directories. `ExtractArchive` rejects symlinks, hard links and devices. `fvpn-release` rejects them at pack time. A `./` directory entry is the archive root. Any other `.` or `..` segment is rejected.
- D-034g. Usage and flag errors exit 64. Exit 2 means only `ErrMembersBlocked`.
- F-15. `SigningKey.TrustStore` uses `envelope.NewNodeTrustStore`. `fvpn-release verify` uses `envelope.ParseNodeTrustStoreJSON`. `CheckArtifact` rejects duplicate matching platform entries.
- D-053. Isolation, DNS and the tunnel probe come from the running candidate agent, bound to a per-transaction challenge in the journal directory. No config field, environment variable or pre-existing file can mark them passed. The attestation carries `node_id` so the node ids can be compared; the agent's `published` value is `published`, `unpublished` or `disabled`, and the updater never opens the gate from that field. Unknown JSON fields are rejected. The files are regular, mode 0600, not symlinks, and at most 64KiB. A check reason is at most 160 bytes and has no control characters. `not_run` is a known status and is `failed` for the updater. `generated_at` must be strictly after `issued_at` and strictly before `expires_at`, and the updater's clock must still be before `expires_at`. There is no second future-skew bound inside that window. A file present before the challenge is ignored. Equal modification times count as "not after"; a rewrite is a new inode, a later modification time, a different size, or a different SHA-256. `health_json` keeps the first report. Reasons go to `txn_events`. The rollback order is stop, then the restored release's challenge, then start. The wait, poll and TTL above are 待實測. The 8-attempt and 3-rollback caps stay.
