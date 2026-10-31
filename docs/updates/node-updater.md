# Node updater

`fvpn-updater` (`node/updater/cmd/fvpn-updater`) applies one node's authorized update. It downloads a signed release, checks it, and switches through the update journal. A crash resumes from the last durable stage. Member forwarding stays off while a transaction is applying or checking, and stays off after `recovery_required` (§13.5, §13.9).

The updater never executes a file from a manifest. It never changes `publication_intent`. It never deletes identity or the usage outbox to make space (§13.7). A local result stays in the outbox until the center accepts it. Local success is not acceptance (§13.6 step 13).

The central fleet orchestration, the agent process and the installer are separate. The updater talks to them through the interfaces in `node/updater/interfaces.go`.

## Commands

```text
fvpn-updater run     -config updater.json
fvpn-updater recover -config updater.json
fvpn-updater flush   -config updater.json
fvpn-updater status  -config updater.json
fvpn-updater status  -journal /var/lib/vpn-node/update-journal/journal.db
fvpn-updater apply   -config updater.json -pending /var/lib/vpn-node/update-staging/pending-release.json
fvpn-updater seed --config updater.json --manifest retained.manifest.json --tree /opt/vpn-node/releases/<release_id>
fvpn-updater version
```

`recover` runs before member services start.

| Exit | Meaning |
|---|---|
| 0 | The command completed without an operation error; `apply` never uses 0 for a sealed failed job. This is not permission to forward member traffic. `help` and `version` also exit 0. `status` exits 0 when it reads successfully, including when the snapshot says recovery is required. `seed` exits 0 only after verifying and recording (or re-verifying) the baseline; it makes no serving or health claim. |
| 2 | Only `ErrMembersBlocked`. Members must stay blocked. |
| 1 | Any other failure, including a config that does not load and `flush` without `result_url`. |
| 64 | Usage, an unknown command, a bad flag, or missing required flags. `seed -h` and `seed --help` print seed-specific flag help and exit 64. |

`run` holds the node lock, calls `recover`, applies an installer pending release when that file exists, then takes the current desired state. `apply` holds the same lock, calls `recover`, and applies the `-pending` file. It does not also read desired state. `flush` posts pending results and does not start an update. `flush` requires `result_url`. `Run`, `ApplyPending`, `Seed`, `Recover`, `Flush` and `Retain` take the node lock at entry, before they supersede an older job, claim, download or install.

The config file holds paths and ids. It does not hold the release private key. `allow_loopback` is TEST ONLY and must stay false on a node. `allowed_artifact_origins` is a list of `scheme://host[:port]` origins (D-046). An artifact, manifest or `result_url` whose origin matches an entry is exempt from the private and CGNAT address guard for that origin only, and the dial is pinned to the first address that host resolved to. Link-local, multicast, unspecified and metadata names stay refused. The default is empty, so the guard stands. `allow_loopback` does not allow private addresses. A compose dev stack sets the release and result origins explicitly. An entry has no user, path, query or fragment. An entry matches a URL when the scheme, host and effective port are equal. The effective port of a URL or entry without an explicit port is the scheme default: 443 for https and 80 for http. Stock health checks (`NodeChecks`) compare the release tree, the identity directory, and an immutable record of the pointers and the state-schema marker. They do not hash the mutable state directory. Isolation, DNS and the tunnel probe come from a challenge-bound attestation written by the candidate agent (D-053, §13.9). Without a valid attestation those three checks fail. A config field, an environment variable or a file that was already in the journal cannot mark them passed. The file contract is [`node-health-attestation.md`](node-health-attestation.md). An empty stop or start command fails closed. The updater does not guess systemd. `Config.Engine` installs a migrator that refuses a schema change, so a release that needs one fails before services stop when no real migrator was supplied.

The forwarding latch is `forwarding.blocked` in the journal directory. The agent must not enable member forwarding while that file exists or while `BlocksForwarding` is true. The nftables gate itself belongs to the agent.

## Installer hand-off (D-080, D-089, D-110)

Initial install and reenroll retain the exact verified signed envelope (`Verified.Raw`) as mode 0600 under update-staging. After releasing `/var/lib/vpn-node/update.lock`, the installer probes `<current>/fvpn-updater seed --help`, then invokes:

```sh
/opt/vpn-node/current/fvpn-updater seed \
  --config /etc/vpn-node/updater.json \
  --manifest /var/lib/vpn-node/update-staging/<release_id>.manifest.json \
  --tree /opt/vpn-node/releases/<release_id>
```

`--tree` is the resolved current release directory. `seed` takes the shared `node/nodelock`, re-verifies the envelope with `packaging/release`, checks platform compatibility and revocations, and compares the entire installed tree with the signed artifact. The manifest authenticates an archive hash, not a per-file inventory (§13.3). Seed obtains `/var/lib/vpn-node/update-staging/<artifact-sha>.tar.zst` when present, or downloads the signed artifact URL using the existing URL guards and bounded retry policy. It verifies a private archive copy, uses the apply path's safe extractor, and compares hashes covering every file's bytes, paths, modes and directories. Missing files, extra entries, symlinks and changed content fail closed. Verification scratch is removed on normal return.

**Offline installation:** the merged initial-install writer retains only the envelope, not the archive. Seed therefore needs the signed artifact URL to be reachable unless the matching archive is already staged. An unavailable archive cannot be replaced by trusting the live tree or a locally computed hash. Upgrade already stages this archive. There is no contract or installer-file change in this updater delivery.

The verified release, current pointer, state-schema marker and baseline evidence commit in one SQLite transaction, with no schema migration. `status` exposes `seeded_baseline` for the current release: `release_id`, `manifest_sha256`, `tree_hash`, `boot_id`, `seeded_at`. A baseline is installation evidence; it does not assert health, create an update job/result, promote `last_good`, or change publication/forwarding authorization. Seed performs no switch, service restart or rollback.

Seed exit 0 means verified and recorded. The same canonical envelope and tree are re-verified on replay, preserving the original seed timestamp and boot id, with no second release or baseline row. An established journal with a different current release, manifest, tree or state schema is refused; open work is refused too. Exit 1 reports the reason and leaves existing journal evidence and the current pointer intact. Usage/unknown commands exit 64; `seed -h`/`--help` print `-manifest` and `-tree` help with exit 64. The installer recognizes an older unsupported updater only from exit 64 with `unknown command "seed"`; other probe/seed failures become installer exit 1 and check drift. Probe and seed exits/stderr are retained in `install/baseline.json` and `check --json`.

Next, `fvpn-node-install upgrade` verifies a signed release, extracts it under releases, copies its archive into update-staging, retains `Verified.Raw` as mode 0600, and writes `pending-release.json` with a fresh per-invocation request UUID. It releases the shared lock and starts `fvpn-updater.service`. The updater's `run` consumes that record before desired state. The foreground equivalent is:

```sh
fvpn-updater apply --config /etc/vpn-node/updater.json \
  --pending /var/lib/vpn-node/update-staging/pending-release.json
```

`apply` reports a sealed `failed` job as nonzero with `outcome=failed` and its error code on stderr, including a failure sealed during recovery or replay. A new final-sample failure exits 1; recovery that still blocks forwarding retains exit 2. A healthy rollback remains the distinct `rolled_back` outcome. The meter checkpoint producer is an agent-core dependency; this exit correction does not manufacture a checkpoint.

Both apply paths take the lock, recover existing work, and use the journaled verify, switch, restart, health-check, rollback and result-outbox state machine (§13.5–§13.9). Contention reports `UPDATE_BUSY` (updater exit 1; installer busy exit 3). A missing default pending file is no work. An explicit missing `--pending` is an error, except when recovery just consumed it.

| Pending field | Treatment |
|---|---|
| `release_id` | Required nonzero UUID. |
| `manifest_sha256` | Required SHA-256 of the canonical signed manifest envelope. |
| `artifact_path` | Required regular archive file; symlinks are refused. |
| `tree_path` | Informational extracted tree; apply extracts again from a verified private archive copy. |
| `updater_command` | Informational; never executed. |
| `manifest_path` | Merged writer's retained signed envelope; a bounded regular file, mode 0600. |
| `request_id` | Merged writer's per-invocation UUID; included in the local integrity digest only. |
| `manifest` | Optional inline signed envelope supported by the consumer; takes precedence over `manifest_path`. |

The live current pointer must match journal evidence. An unseeded installer-created pointer still fails with `CHECKPOINT_FAILED`; it is never interpreted as an empty node. Missing verification material seals an identified request as `failed/verify/ARTIFACT_INVALID`; malformed or unidentifiable records remain in place.

The journal claim, status and wire result carry `source: "installer"` (contract revision 13). D-110 uses `internal/installerjob.ID(release_id, manifest_sha256, incarnation_id)`; attempt is 1. This UUIDv5 identity binds only the release, canonical manifest digest and authenticated incarnation. Paths and request IDs do not create additional jobs. The full canonical pending-record digest (including `manifest_path`, `request_id` and unknown fields) remains the local integrity and cleanup check. Identical replay does not reapply or reseal; changing a record for an existing identity is refused. Legacy local claims and acknowledged receipt history retain their original identities. Local jobs use the accepted update generation, or 1 when absent, without advancing the central fence. Policy generation is the already-applied generation, or 0 before policy. Publication intent is preserved (disabled when absent). Verified current/last-good releases may be rollback candidates, but installer outcomes keep member forwarding blocked until a valid desired state authorizes serving.

The `upgrade` staging command is owned by `packaging/installer/upgrade.go`, dispatched by `packaging/installer/cmd/fvpn-node-install/main.go`. F9-05 requires that lane to compare installed and requested release sequences before writing staging files, returning `DOWNGRADE` / exit 1 for a lower sequence. The updater CLI has no `upgrade` subcommand.

The terminal job and pending-file cleanup obligation commit together. Then the matching record is renamed to `pending-release.<state>.json` and its parent directory is fsynced. Recovery retries cleanup without another apply/result. The canonical record hash prevents an old completion from consuming a newly staged request.

Local success remains distinct from central acceptance. Central accepts eligible installer evidence under D-110/D-111. Against an older deployment or before release registration, HTTP 404 `RESOURCE_NOT_FOUND` for a result with `source: installer` stays pending, preserving the exact payload/hash and journal. Each flush sends at most one attempt for that result and durably defers the next attempt with the existing exponential/jitter schedule (200 ms initial, 30 s cap by default). Retryable failure holds later results behind it; no payload is manufactured or rewritten. The log says `central not yet accepting installer evidence`. A 409 `ACK_CONFLICT` is terminal, logged and retained. Source-less/central 404s remain permanent rejections. Central-issued results keep the default central source. D-114 handles legacy unaccepted identities as described below.

The lock holder, seed and job claim use the kernel boot id. Configured `boot_id` is only a logged fallback when the kernel value is unavailable; the installer now omits it. A missing desired-state file remains `ErrNoWork`; malformed or unreadable input still fails closed through `GuardSource`.

## Synthetic health probe configuration (D-063)

The optional `probe_peer` config block requests the dedicated synthetic peer's temporary exception while member forwarding stays blocked (§13.9). Omission (or null) keeps the previous behavior: the challenge has no `probe` member. Configuration does not mark any check passed. Both candidate checks and restored-release rollback checks carry the configured request, each with a fresh nonce.

Template fragment only; `PUBLIC_TEST_ENDPOINT_IPV4` must be replaced with the team's controlled public test endpoint before loading. The placeholder is deliberately not a deployable address:

```json
{
  "agent_config": "/etc/vpn-node/agent.json",
  "probe_peer": {
    "peer_id": "dedicated-probe-peer",
    "source_v4": "100.64.8.1/32",
    "destinations": [
      {"addr_v4": "PUBLIC_TEST_ENDPOINT_IPV4/32", "proto": "tcp", "port": 443}
    ],
    "resolver_dns": false
  }
}
```

Use the dedicated test peer's actual NetBird ID and `/32` overlay source; do not use a member peer. Specify 1–4 public `/32` destination sockets, each with lowercase `tcp` or `udp` and port 1–65535. `resolver_dns` optionally requests the agent's configured resolvers on TCP/UDP 53 in addition to those destinations. The issuer uses the enforcement package's specification deny list: RFC1918, CGNAT/overlay, loopback, link-local and metadata, documentation, benchmark, multicast, reserved and other specification-denied destinations fail configuration load. Artifact-origin/loopback test options do not relax this check. Peer IDs must be nonblank, at most 128 bytes and free of ASCII controls. Unknown JSON members, malformed values and trailing JSON are rejected with fixed, bounded errors that do not echo configuration values.

The updater also collects address exclusions at configuration load, engine construction, and every challenge issuance:

- Every global unicast IPv4 address returned by the host's interface inventory, including private and CGNAT interface addresses. An empty filtered list is refused as `node_addresses_unavailable`, including during interface bring-up and on loopback/link-local/IPv6-only hosts. A source or destination equal to any of these is refused. Other private/CGNAT `/32` sources remain valid overlay peer addresses; the destination deny list does not apply to sources.
- The literal address or all DNS answers for `control_url` in the agent configuration, and for updater `result_url` when set. Destinations equal to any of these addresses are refused.
- Each `dns.resolvers[].dns_addr` listener host and `dns.resolvers[].admin_url` host in the agent configuration. Destinations equal to a resolver endpoint are refused regardless of `resolver_dns` and regardless of the requested port.

`agent_config` selects the running agent's configuration file; omission or an empty value selects `/etc/vpn-node/agent.json`. Set it explicitly when the agent uses a different file. The updater reads only its address-bearing fields, accepts unrelated agent configuration fields, and never opens the resolver credentials file. The file must be regular, valid JSON and at most 1 MiB. `Lstat` refuses symlinks, FIFOs, devices and directories before open; `O_NOFOLLOW|O_NONBLOCK` and an `os.SameFile` comparison with the opened file also refuse replacements without blocking on a FIFO. These refusals use `agent_config_unavailable`. Interface enumeration, configuration reads and DNS resolution run afresh for each validation; all DNS lookups share a two-second deadline. A missing/malformed configuration, missing control URL, unavailable address source, failed/empty DNS lookup, or refused probe fails load with exactly `updater: invalid probe_peer`.

If an input becomes unavailable or an address becomes forbidden after load, issuance still writes a fresh challenge with `probe` omitted. The configured probe remains available for later revalidation. The CLI logs `health challenge probe omitted` once per refusal class per updater process, using only a fixed `reason`: `malformed`, `denied_destination`, `node_source`, `node_destination`, `control_destination`, `resolver_destination`, `agent_config_unavailable`, `node_addresses_unavailable`, `control_addresses_unavailable` or `resolver_addresses_unavailable`. No peer, address, hostname, URL, path or raw lookup error appears in this diagnostic. With `probe_peer` absent, none of these address inputs is required or read.

An omitted configured probe is also recorded outside logs:

- The protected challenge carries `"probe_omitted":{"peer_id":"dedicated-probe-peer","reason":"node_addresses_unavailable"}`. `probe` and `probe_omitted` cannot coexist. An unconfigured probe has neither field. The omission is copied and included in the challenge's `Same` comparison. A directly supplied malformed peer ID is represented as `invalid_peer_id`; loaded configurations already require a valid bounded ID.
- Before publishing the challenge, the updater commits its transaction ID, nonce, release, phase, issuance timestamp and omission to SQLite `meta` under `probe_omission/<txn_id>/<nonce>`. An apply transaction also receives the same JSON as a `txn_events.detail` row, atomically with the meta entry. `ProbeOmissions` reads the meta rows while that apply transaction is still open, including after the challenge sidecar is consumed. `cleanTxnNotes` deletes them when the transaction seals, and `Retain` deletes `probe_omission/<txn_id>/%` for every terminal row in `txns`. An open transaction therefore keeps at most six rows (three candidate attempts and three rollback attempts). The `txn_events` copy is not deleted. Standalone revocation recovery has no `txns` row, so its meta row is the only copy and `Retain` leaves it. Repeating an issuance does not duplicate it; conflicting evidence is refused. This records the issuance decision, not proof that the challenge was published or answered. A journal-write failure aborts publication; omission itself remains nonfatal.
- Local `health_attest` result diagnostics carry `probe_omitted` beside the observed checks. The HTTP `UpdateHealth` contract contains only status enums, with no diagnostic member, so no unsupported wire field is added. Agent-owner integration is required to set `ObservationReport.checks[name=tunnel_probe].detail` to `probe_omitted: <reason>` from the matching challenge, and retain that diagnostic with its check timestamp. Peer IDs and network addresses must stay out of that detail. The issuer does not reinterpret a later agent health status as part of this recording change.

Consumer compatibility: the agent calls the shared `healthattest.ReadChallenge`, which uses `DisallowUnknownFields`; unknown fields are **not** ignored. Agents rebuilt with this shared `ProbeOmitted` member can decode the additive field. Older binaries reject an omission-bearing challenge and fail closed; candidate and retained rollback releases need the shared reader update. Unconfigured and successfully validated probes retain their existing shape. Observation reporting belongs to the agent owner and is separate from recognizing the new member.

The production challenge uses `issued_at = now` and `expires_at = issued_at + 45s`; no config field changes that TTL. This is below D-063's 300s maximum. The consumer installs only after issuance, caps lifetime relative to issuance and expiry, and independently rejects member collisions, host addresses and additional denied ranges. Issuance is a request, not proof of successful installation or traffic. The [challenge contract](node-health-attestation.md#optional-synthetic-probe-d-063) describes exact limits, nonce binding, privacy and updater/agent ownership. Actual path verification still requires the authenticated synthetic probe and its challenge-bound attestation.

## What the updater accepts

Work comes from a desired-state envelope (§14.7). `FileSource` verifies it on every read with a `desired-state` purpose key. The window is half-open `[issued_at, expires_at)`. The node id and incarnation must match. An unverifiable or expired envelope blocks forwarding. A valid envelope with `update: null` is not a job (`ErrNoWork`) and does not by itself change the latch. A missing desired-state file is the same `ErrNoWork`: one info line, exit 0, and the forwarding latch stays as it was. An unreadable or malformed file still blocks forwarding.

The desired-state document has `serving_release_ids`. A verified envelope may also carry signed `revoked_release_ids` (revision 3, D-034). An absent member means no additional revocations and does not change the persisted signed set. An empty array adds nothing and does not clear that set. `revoked_release_ids` in the updater config stays the unsigned release-support list copied by the agent. The effective set is the union of that list and the persisted signed set. The result's `policy_generation` is the desired-state `config_generation` actually applied when the result was produced (D-043, §13.10.3). It is not the update job generation and not a generation that was only downloaded. A result is not posted before that generation is known. `boot_id` on a result is the boot that claimed the job. That boot id is `/proc/sys/kernel/random/boot_id` when the file is a UUID. `boot_id` in `updater.json` is only the fallback, and only when the kernel file is unreadable.

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

Rollback chooses the newest retained release that passes the checks below. Newest is `release_sequence`, then `installed_at`, then release id. Retained means a release the journal still records and whose tree retention has not removed. A revoked release is never selected and is never installed. The checks, in this order, are: a tree is present and its hash matches, the manifest signature verifies, the release is not revoked, the release is in the serving set, and the state schema is readable by that release. When the failed candidate's `rollback.allowed_release_ids` is non-empty, the target must also be in that list. A release missing from the serving set or from a non-empty allow list is `MINIMUM_VERSION_VIOLATION`. An empty allow list adds no extra restriction (D-034e). `StateReadableByPrevious` does not mark a schema readable when the journal schema is not that release's schema. A health check past the 8-attempt cap starts a rollback. It does not jump straight to recovery.

A currently installed release named by the persisted signed set is unhealthy. The updater stops that service, switches to the newest non-revoked release, issues a fresh recovery transaction UUID and rollback challenge, starts the target, and verifies its bound answer. A failed check stops the target and keeps forwarding blocked. `current` still names the selected tree, `last_good` stays unchanged, and `previous_release_id` names the revoked release. When no such release exists, the outcome is `recovery_required`, member forwarding stays blocked, and the revoked release does not pass the gate. The unsigned agent list still refuses installation. It does not, by itself, roll the current release back, so a legacy envelope keeps today's behaviour. The persisted signed set is a monotonic union: an id is never removed, an older `config_generation` cannot shrink it, and an absent member does not update it.

When no safe retained release can be restored, the `current` symlink stays where it is (the failed candidate, or the revoked release). Forwarding stays off. The symlink is not moved to an unverified tree.

A failure before services are stopped (for example the metering barrier) is not destructive. The journal ends `rolled_back`, the result state is `failed`, and forwarding follows the publication rule because the old process is still the one that is running. A stop failure is reported as `START_FAILED` with stage `stop_services` (there is no `STOP_FAILED` code) and the rollback is destructive, so `last_good` is started again.

`Block` failing does not finish the transaction. The stage stays `prepared`, the error is `UPDATE_BUSY`, and the journal still reports forwarding blocked.

Health checks are capped at 8 attempts and rollbacks at 3. Those caps are local safety limits (待實測), stored beside the journal as `checks-<txn>` and `rollback-<txn>`.

### Health attestation (D-053, §13.9)

Before `Services.Start` in the check stage, the updater writes `health-challenge.json` in the journal directory. The candidate agent answers with `health-attest.json` in that same directory. The updater accepts the answer only when the nonce and transaction match the live challenge, the release, node, incarnation and boot ids match, `generated_at` is after `issued_at` and before `expires_at`, the updater's clock is still before `expires_at`, the file was written after the challenge, `agent_version` equals the manifest's `node_agent`, `netbird_version` equals the manifest's `netbird`, and each status is a known `HealthCheckStatus`. Anything else fails closed. The typed reasons and the write rules are in [`node-health-attestation.md`](node-health-attestation.md).

`stepStart` may start the candidate before this file exists. The check stage calls `Start` again after the challenge is on disk. That second start is the signal that the challenge is live. A resume of `checking` writes a new nonce. An answer for the previous nonce does not match. The attempt past the 8-attempt cap starts rollback and does not issue a candidate challenge.

On a destructive rollback the updater stops the candidate, writes a new challenge for the restored release, then starts that release. The restored agent answers that challenge. A non-destructive rollback does not run this check again.

`not_run` from the agent is `failed` for the updater. The probe is mandatory for unpublished and disabled nodes (§13.9). A binding `not_run` ends that attempt. A missing or mismatched file keeps being read until the wait window, so a corrected rewrite inside the window can still pass. At the deadline the last reason is journaled and the three checks are `failed`. When the attestation itself does not validate, `processes` is `failed` as well, because the versions were not attested. `identity_and_state` stays the identity directory and the immutable pointers record (D-034c).

A passing check commits `last_good`, the state-schema marker, and `locally_succeeded` in one SQLite transaction. Verified rollback likewise commits `last_good` with `rolled_back`, so recovery cannot observe a promoted pointer with the earlier in-flight stage. Member forwarding is still the latest publication intent plus the serving policy (§13.9 服務狀態恢復). The attestation's `published` field does not open the gate. Health does not publish an unpublished or disabled exit, and a release outside the serving set stays blocked.

D-062 separates candidate evidence from the restored release's evidence. `txns.health_json` is the latest observed profile. `candidate_health_json` retains the latest candidate profile when rollback starts; subsequent rollback checks change only `health_json`. A resumed candidate challenge may replace the earlier candidate profile. Reasons remain in append-only `txn_events` rows (`kind: health_attest`, `phase: candidate` or `rollback`). A `rolled_back` result sends the restored release's checks in `health`, keeps the candidate failure in `error_code` (for example `HEALTH_CHECK_FAILED`), names the restored release in `current_release_id`, and names the candidate in `previous_release_id`. Contract 1.2.0 (revision 6) sends that candidate profile as `candidate_health` on `rolled_back` results. If rollback happened before candidate checks ran, the member is explicitly null, never an empty object. On every other state this updater omits the member; the contract also permits explicit null, as its null-candidate-health and preflight-failed vectors demonstrate. `health` remains the restored release's checks. When no retained release can be restored, the terminal code is `ROLLBACK_FAILED` and candidate evidence remains in the journal.

A terminal `failed` result before any health profile ran has `health: null`, an error code and `finished_at`; `current_release_id` names the release still current. For example, a tampered artifact reports `ARTIFACT_INVALID` with the unchanged current release. Once checks have run, a failed result carries the latest observed health. A passed health profile is never manufactured.

`seal` does not report `recovery_required` as stage `report`. `journalStageAPI` maps that journal stage to `report`, and `seal` replaces it with the `error_stage` `toRecovery` recorded (`rollback` or `migrate`, for example). D-062 requires health evidence for a terminal result whose stage is not before the health checks. `rollback` is after those stages. `beforeHealth` in central `SubmitResult` lists the stages through `start` and does not include `rollback` or `report`. When recovery is entered at `rollback` and no profile is stored, the result carries `blankHealth` for the release `current` still names: checks that did not run are `failed`, and a profile already stored for the attempt is kept. A pre-health recovery, such as `migrate` when the state is unsafe, keeps `health: null` together with that error stage. Central accepts the pre-health document only when health is null, and it rejects `stage` `report` with `health` null: `Terminal result at or after health checks requires health evidence` (HTTP 422 `VALIDATION_FAILED`).

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

Tables: `meta`, `jobs`, `releases`, `txns`, `txn_events`, `artifacts`, `result_outbox`, `result_history`. Schema version is `PRAGMA user_version = 2`. A newer version is refused. The transactional v1 upgrade adds `candidate_health_json`, delivery diagnostics and retry deadlines, and result-history stage. It copies the old candidate profile into its separate column and reconstructs restored health only from a recorded rollback event; missing evidence stays null. Existing outbox payloads, result IDs and hashes are immutable through migration. A previously sealed result may therefore be permanently rejected under the corrected central rules; its evidence is retained and later results can still be delivered.

`txns.stage` is the stage above. `destructive` sticks once set. `txn_events` is the append-only history. Releases are immutable. `current` and `last_good` are meta pointers.

`probe_omission/<txn_id>/<nonce>` meta rows are bounded by the open apply transaction. `cleanTxnNotes` removes that transaction's rows when it seals, together with the `epoch-<txn>`, `checks-<txn>` and `rollback-<txn>` sidecars. `Retain` removes the same keys for every terminal transaction still stored in `txns`, so a crash between the audit write and `cleanTxnNotes` does not leave them behind. At most six rows remain for the open transaction. Standalone revocation-recovery ids are not in `txns` and are not removed. The schema version stays 2.

The result outbox stores the canonical JSON and `payload_hash` (`canonjson.HashHex` of the result without `payload_hash`). `result_id` is UUIDv5 over `fvpn-update-result|<job>|<attempt>|<state>` so a replay sends the same id. UUIDv7 is not replay-stable. The same id and hash is a no-op. A different hash is `ACK_CONFLICT` and is kept, not resent. A late duplicate ACK is a no-op.

Contract 1.2.0 also covers rolled-back results already pending in a v2 journal. Immediately before sending a row missing `candidate_health`, the updater copies the matching job/attempt/generation/release transaction's `candidate_health_json` (or null before checks), adds only that member, and recomputes `payload_hash`. Both bytes and hash commit in one SQLite transaction before the HTTP call. The result ID, generation, attempt, timestamps, restored health and every other existing field are preserved; the result is not rebuilt from today's node state or rechecked. Retries reuse the updated canonical bytes, including across restart. This is an additive outbox compatibility update, not a schema migration: `user_version` remains 2. Existing candidate_health (including explicit null), rejected/conflicted rows and acknowledged history stay unchanged. A previously sent result whose old hash was accepted centrally but whose ACK was lost can still receive ACK_CONFLICT; the updater retains that conflict rather than inventing another result ID.

`POST /agent/update-results` is answered with 202 in the contract. A 409 marks `conflict`. Other 4xx responses except 429 and installer 404 `RESOURCE_NOT_FOUND` mark `rejected`; both states retain the exact payload for diagnostics, stop resending it, and allow later pending results to flush in insertion order. HTTP status, a recognized API error code, attempt time and local error code are stored separately (`last_http_status`, `last_api_code`, `last_attempt_at`, `last_error`). Arbitrary response messages and unknown codes are not stored. Only transport failures and 5xx map to `STORE_UNAVAILABLE`. A 429 stays pending with `RATE_LIMITED` and a durable `retry_not_before`: `Retry-After` accepts delay seconds or an HTTP date. Missing/malformed headers defer one second. Flush returns without sleeping until that deadline, including after restart; later results wait behind a retryable failure. Other unexpected responses remain pending as protocol errors. A successful flush means its pending batch was processed; inspect rejection diagnostics to distinguish central acceptance from permanent rejection.

Sidecar files next to the database (not inside it): `identity.snap` (identity tree hash) and `immutable.snap` / `immutable.record` (pointers and the state-schema marker). Both are taken after migration and before services start, and again for the pointers before rollback health runs against the selected tree. They are health-check snapshots, not completion markers; the terminal SQLite transaction records verified `last_good`. The mutable state directory is not in that record. `epoch-<txn>` is the new meter epoch for that transaction. It lives in the journal directory, not under state, and it is removed when the transaction completes. `FileMeter.OpenEpoch` does not write a sidecar. The check and rollback counters and `forwarding.blocked` are the other durable sidecars. `health-challenge.json` and `health-attest.json` are short-lived. The updater removes both after it records health. If the remove fails, it rewrites the challenge with `consumed: true`, which a later validation rejects. The next check overwrites the challenge with a new nonce.

The single-updater lock is `/var/lib/vpn-node/update.lock`, a kernel `flock(LOCK_EX|LOCK_NB)` held by `node/nodelock`. `node/internal/nodelock` is a thin alias so existing `node/` callers keep compiling. `packaging/installer` uses the same `node/nodelock`. The descriptor stays open until `Release`. Holder metadata (`pid`, `boot_id`, `start_time`) is written only after the lock is held. `boot_id` in that record is `/proc/sys/kernel/random/boot_id`. `start_time` is field 22 of `/proc/<pid>/stat` (clock ticks since boot), counted after the last `)` of the command name. A stale pid, a different boot id, or the same pid with a different start time does not steal a lock the kernel still holds, and the file is not removed on contention. A live flock is `UPDATE_BUSY`. A held lock whose record is stale is also `UPDATE_BUSY`, and the error names the reason (`boot id`, `pid`, `process` or `start time`). The agent, the installer and the updater take that same lock.

## Read-only status and agent observations (D-062e)

`fvpn-updater status` emits a JSON snapshot. For `rolled_back` it also shows `candidate_health` with the wire value (including null before candidate checks); `candidate_health_json` remains the local diagnostic field on all states. With no flags it reads the default journal database. `-journal` accepts an existing database **file**, not a directory; `-config` selects the journal path from an updater config. The flags are mutually exclusive. No engine, service command, migration, directory creation or updater lock is needed.

The agent can import `node/internal/updatejournal` and call:

```go
snapshot, err := updatejournal.Snapshot(ctx, journalPath)
```

`journalPath` is the full path to `journal.db`. `Snapshot` uses SQLite `mode=ro`, `query_only`, and one read transaction for a consistent WAL view while the updater writes or holds its flock. The caller's deadline or a one-second bound (`SnapshotTimeout`), whichever is sooner, limits the read; lock waits are capped at 50ms. Missing, unreadable or unsupported databases return an error. The agent must treat that as unavailable evidence, not successful completion. Versions 1 and 2 can be observed without migration; v1 rolled-back health is null because its old column held candidate health.

| JSON field / Go field | Meaning |
|---|---|
| `generation` / `Generation` | Latest job/result generation as a decimal string; null for an empty journal. |
| `job_id` / `JobID` | Job associated with that generation. |
| `stage`, `state` / `Stage`, `State` | API vocabulary for local progress or terminal result; empty before any job. |
| `journal_stage` / `JournalStage` | Exact local transaction stage, when one exists. |
| `current_release_id`, `last_good_release_id` / `CurrentRelease`, `LastGoodRelease` | Durable journal pointers, not an assertion that the process is healthy. |
| `seeded_baseline` / `SeededBaseline` | Verified installer baseline for current, when seeded; release id, manifest/tree hashes, original boot id and seed timestamp. No health or central acceptance claim. |
| `recovery_required` / `RecoveryRequired` | Persisted recovery gate or latest transaction requiring recovery. |
| `last_result_state` / `LastResultState` | Latest local result state from pending, rejected or acknowledged evidence; not an acceptance flag. |
| `last_rejection` / `LastRejection` | Null or the latest permanent rejection/conflict: result ID, HTTP status, API code, local code and attempt time. |
| `health` / `Health` | Latest profile for the selected transaction, null before checks. |
| `candidate_health_json` / `CandidateHealth` | Separate candidate profile for diagnosis, null before candidate checks. |

Generations are compared as unsigned decimal integers without floating point or signed SQLite casts. A newer open job takes precedence over an older terminal result; the older result state and rejection remain separately visible. Acknowledged history retains its result stage. The snapshot describes local durable evidence and never claims central completion.


## Legacy installer outbox receipts (D-114)

Before upload, an unaccepted `source: installer` receipt is compared with the
D-110 derivation of its own release, manifest digest and incarnation. For the
locally reproducible JSON/hash seal, the updater atomically queues a replacement
with the derived job ID, corresponding deterministic result ID and new payload
hash. Every other JSON member, including timestamps, health, checkpoint and
null/absent distinctions, is preserved. The new result ID also avoids reusing an
HTTP idempotency key previously sent with different bytes.

The original outbox row remains byte-for-byte as retired diagnostic evidence
(`status: rejected`, `last_error: installer_job_id_rederived`); an existing-journal
metadata link records its replacement result ID. This is local retirement, not a
central rejection or acceptance. Local job/transaction claims stay unchanged.
Replacement insertion, retirement and the audit link commit together. Recovery
still finds the original seal and cannot recreate it. The new pending seal is
replayed identically across restart until ACK, then moves to history. Existing
D-110 seals and acknowledged legacy history are never re-derived.

An unknown JSON member may bind an external seal the updater cannot reproduce.
That receipt keeps its original bytes for submission. Likewise, a legacy receipt
cannot overwrite an already occupied D-110 identity. HTTP 409 `STATE_CONFLICT`
with `details.reason: job_id_mismatch` is terminal: the updater retains the seal,
HTTP/API/reason and attempt count as rejected evidence, removes it from the
pending queue, and never retries it, including after restart. Ordinary 409
`ACK_CONFLICT` remains terminal too. No server message or arbitrary detail is
stored. Pending-file consumption remains the terminal job's durable cleanup
obligation, separate from central acceptance. Corrupt local hashes fail closed.

## Download and verify (§13.4)

The manifest hash is the SHA-256 of the canonical JSON of the whole envelope. The artifact hash is the raw file. `.partial.meta` is written when the `200` or `206` headers arrive, before the body, so an interrupted download still resumes with `If-Range`. A partial download resumes only when the content hash is unchanged and the server's range agrees with the bytes already stored. A `206` whose `ETag` differs from the ETag stored in `.partial.meta` is not appended: the partial is deleted and the attempt fetches the object again from byte zero. A `200` response to a `Range` request is the full representation (RFC 9110 §13.1.5, or a server that ignores `Range`): that body replaces the partial, the download does not fail with `ARTIFACT_INVALID`, and a body that still does not match costs at most one content retry. When `Content-Length` is known, the bytes written must equal it (`200`) or the prior offset plus it (`206`). A changed body deletes the partial and downloads the whole file again. One content retry is allowed. A second hash or signature failure is permanent and is not retried. Network errors use bounded backoff with full jitter. The default budget (待實測) is 5 attempts, 200 ms initial, 30 s cap, 60 s per attempt. Manifest downloads are capped at `canonjson.MaxBytes` (1 MiB).

Verification checks the release trust store, both Ubuntu 24.04 architectures (`amd64` and `arm64`, `tar.zst`), this node's artifact, size, SHA-256, control API major, policy schema, usage schema, state schema, minimum updater version and the disk reserve (`artifact + installed size + required_free_bytes` on the releases filesystem, and the artifact size on the staging filesystem). A duplicate matching platform entry is `VALIDATION_FAILED`. `product_version` is display only. `minimum_updater_version` is SemVer 2.0.0: three numeric components, no leading zeros, build metadata ignored, and a pre-release is older than the same core without one. `1.10.0` is newer than `1.9.0`. `1.0.0-alpha.1` is older than `1.0.0`.

URLs must be `https`. Loopback, link-local, multicast, unspecified, private (RFC 1918 and IPv6 ULA), CGNAT (`100.64.0.0/10`) and the metadata host names are refused. `allow_loopback` is the test exception and permits loopback only. The same check runs on every address `DialContext` resolves, so a public name that answers with a private address is refused (D-034f). An entry in `allowed_artifact_origins` exempts a URL from the private and CGNAT checks only when the scheme, host and effective port are equal (D-046). The effective port of a URL or entry without an explicit port is the scheme default: 443 for https and 80 for http. The connection is pinned to the first resolved address of that host. Every other origin keeps the guard.

F9-02 fixes extraction modes independently of process umask: the extraction root and nested directories are explicitly 0755; regular files explicitly receive the archive permission bits (zero retains the existing 0644 default). This includes implicitly created parent directories. Installer extraction and seed verification share this extractor, so umasks 022 and 077 produce the same tree and hash. Seed still rejects mode drift. Trees previously installed with umask-filtered permissions must be restored from the verified archive by the installer before seeding; seed does not silently rewrite or ignore their permissions.

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

- `boot_id` is the boot that claimed the job. It is read from `/proc/sys/kernel/random/boot_id`. `updater.json` supplies it only when that file is unreadable, and that fallback is logged. The installer's `refresh-boot-id` pre-start can be removed by the installer lane.
- D-080. An installer-staged release is applied by `run` when `pending-release.json` is present, and by `apply -pending`. The updater verifies the signed envelope and the artifact again, then uses the existing state machine. The journal intent and wire result record `source: installer` (D-089). The pending cleanup obligation is committed with the terminal job, then renamed and retried by recovery if necessary. `tree_path` is ignored and `updater_command` is not executed. Installer outcomes keep member forwarding blocked. The installer job does not advance `accepted_generation`. A missing desired-state file is `ErrNoWork`.
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
- D-053. Isolation, DNS and the tunnel probe come from the running candidate agent, bound to a per-transaction challenge in the journal directory. No config field, environment variable or pre-existing file can mark them passed. The attestation carries `node_id` so the node ids can be compared; the agent's `published` value is `published`, `unpublished` or `disabled`, and the updater never opens the gate from that field. Unknown JSON fields are rejected. The files are regular, mode 0600, not symlinks, and at most 64KiB. The journal directory is private (created as 0700), is not a symlink, and belongs to the effective user. Each check reason belongs to the fixed vocabulary documented in the file contract and is at most 160 bytes. `not_run` is a known status and is `failed` for the updater. `generated_at` must be strictly after `issued_at` and strictly before `expires_at`, and the updater's clock must still be before `expires_at`. A timestamp more than 2s ahead of the updater clock is rejected. Only an all-passed valid answer ends polling early; failed or not_run answers may converge during the wait window. A file present before the challenge is ignored. Equal modification times count as "not after"; a rewrite is a new inode, a later modification time, a different size, or a different SHA-256. `health_json` keeps the latest report and D-062 retains the candidate profile separately in `candidate_health_json`. Reasons go to `txn_events`. The rollback order is stop, then the restored release's challenge, then start. The wait, poll and TTL above are 待實測. The 8-attempt and 3-rollback caps stay.
