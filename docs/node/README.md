# Node agent and watchdog

`fvpn-agent` (`node/agent/cmd/fvpn-agent`) is the program on every exit node. It enrolls once, proves its identity with mTLS, fetches signed desired state, applies isolation and QoS through `node/enforce`, renews leases, and reports observations. `fvpn-watchdog` (`node/watchdog/cmd/fvpn-watchdog`) is a separate process. The lab driver is `fvpn-labcontrol` (`node/internal/fakecontrol/cmd/fvpn-labcontrol`), which has to live under `node/` so it can import the fake control server. If the agent stops refreshing its heartbeat, the watchdog installs the fail-closed baseline and does not open a control connection (§16.3).

The updater is a separate program. This process opens the usage ledger, samples forward-admission counters after forwarding is admitted, and uploads them. It does not apply a release.

## Commands

```text
fvpn-agent enroll -config /etc/vpn-node/agent.json -token-file path|-
fvpn-agent run    -config /etc/vpn-node/agent.json
fvpn-agent step   -config /etc/vpn-node/agent.json
fvpn-agent version

fvpn-watchdog run|once -config /etc/vpn-node/agent.json
fvpn-watchdog version
```

The enrollment token is read from the file, or from stdin when the path is `-`. It is never taken from argv or the environment, including `FVPN_ENROLLMENT_TOKEN`. After a successful enrollment the file is overwritten with zeros, synced, and removed. A failed enrollment leaves the file in place. A second `enroll` does not mint another key or incarnation (AT06, AT07).

`agent.json` holds paths and ids. It never holds a token, a private key, a setup key, or a resolver password. The password lives in `dns.resolver_credentials_file`. The agent rejects unknown fields. The watchdog ignores unknown fields so both programs can share one file.

## Boot order (§8.2, §13.5, §16.3)

1. `fvpn-updater recover` runs first. Exit 0 means the journal does not require forwarding to stay off. Exit 2 means members stay blocked. The unit templates in `node/agent/systemd/` set `SuccessExitStatus=0 2` and order this oneshot before the agent, the watchdog, and `fvpn-netbird.service`. The installer task installs the units.
2. The agent installs the fail-closed nftables baseline before it reads a stored lease.
3. Identity is loaded from the local store. A stored lease is not fresh until this process accepts a lease response.
4. Desired state is verified, then a lease and a time anchor are accepted.
5. Member forwarding is enabled only when every gate in `Decide` passes.

`run` repeats `step` on the shorter of the heartbeat and poll intervals. `step` is one pass.

## Paths

| Path | Mode | Who writes it |
|---|---|---|
| `/var/lib/vpn-node/identity/` | 0700 | agent: key, CSR, certificate, CA, `node.json`, `setup.key` |
| `/var/lib/vpn-node/state/agent.db` | 0600 | agent SQLite store (WAL, `synchronous=FULL`) |
| `/var/lib/vpn-node/state/usage.db` | 0600 | usage ledger (WAL, `synchronous=FULL`); path is `state_dir/usage.db` |
| `/var/lib/vpn-node/state/desired-state.json` | 0644 | agent, only after the envelope verifies |
| `/var/lib/vpn-node/update-journal/journal.db` | 0600 | updater journal; the agent only reads it (`updater_journal`) |
| `/var/lib/vpn-node/update-journal/forwarding.blocked` | — | updater latch; the agent only reads it |
| `/var/lib/vpn-node/update.lock` | — | flock shared by the agent, the installer and the updater (`node/internal/nodelock`) |
| `/run/vpn-node/heartbeat` | 0644 | agent; the watchdog only reads it |
| `/etc/vpn-node/agent.json` | — | installer; no secrets |

The updater journal database is not created here. `updater_journal` is that database file, default `/var/lib/vpn-node/update-journal/journal.db`, the same directory the updater uses. `journal_dir` is the directory, not the file: `forwarding.blocked`, `health-challenge.json`, and `health-attest.json` stay there. `usage.db` is created by the agent the first time an enrolled process boots. Every path above is configurable in `agent.json` except the usage file name, which is always `usage.db` inside `state_dir`. Tests use a temp directory.

The desired-state file is replaced by a temp file and `rename` in the same directory, after `AcceptDesired` has stored the verified envelope. An unverified document is never written there. The store is the anti-rollback authority: a generation older than the accepted one is rejected and the file is left as it was. On boot the agent rewrites the file from the stored envelope.

## Local store

`node/internal/agentstore` keeps identity references, the accepted desired-state generation, the last valid envelope, the lease and its monotonic send time, the time anchor, QoS class minors, the last counter snapshot, admitted addresses, observation outbox rows, and enrollment request state. Schema version is `PRAGMA user_version`. A newer schema refuses to open. Safety writes use `synchronous=FULL` and finish before the agent acknowledges the corresponding control result.

## Enrollment (§6)

The node generates an ECDSA P-256 key (mode 0600) and a CSR with an empty subject. `enrollment_request_id` and the incarnation id are stored before the request is sent, and the same CSR is reused, so a crash and a retry present one identity. The certificate and CA chain are stored before enrollment is marked complete. Bounded backoff is 5 attempts, 200 ms doubling, capped at 5 s.

## Control client (§14.6, D-015)

`node/internal/controlclient` is an mTLS client with a pinned CA, TLS 1.2 minimum, no redirects, and a 1 MiB body cap. Lease renewal, usage upload, and every other call have separate connection pools and concurrency budgets (lease 2, usage 1, other 2). A usage upload cannot take the last lease slot.

`POST /api/v1/agent/bootstrap` is called once over mTLS with no body. The setup key is written to `setup.key` (mode 0600) before any caller is told it exists, and it is never logged. A lost HTTP 200 cannot be fetched again (`BOOTSTRAP_CONSUMED`). `BOOTSTRAP_NOT_READY` (409) is retried later and does not by itself disable forwarding. `BOOTSTRAP_EXPIRED` (410) without a stored key ends bootstrap for this incarnation.

## Desired state and leases (§14.7, §16.3)

Envelopes are verified with `pkg/envelope` and the node trust bundle (`envelope.ParseNodeTrustStoreJSON`). The node id, incarnation, generation, and half-open validity window `[issued_at, expires_at)` must match. A bad signature, the wrong node, or an older generation does not replace the accepted state. A 304 does not move the lease, the ETag, or any deadline; the same step may still renew a lease that is actually due.

On a verification or render failure the agent reports the reason and keeps member forwarding off. `ISOLATION_BASELINE_MISSING` is a report-only `ErrorCode` (revision 3, HTTP status none). The observation uses that code, stage `isolation`, and the message `ISOLATION_BASELINE_MISSING`. It is never written as an HTTP response. `HEALTH_CHECK_FAILED` stays for a health check that is not one of the specific codes. `METER_PATH_INVALID` is a real error code. `netbird_backend` must be `kernel`; `userspace` and `unknown` keep forwarding off.

The agent translates the accepted document into an `enforce.Model`, then calls `enforce.Apply`. QoS class minors (`QoS.Assignments`) are stored so a restart does not renumber classes. The node lock is tried three times, 50 ms apart, around a serving apply. If it is busy the apply is skipped and `UPDATE_BUSY` is reported. A busy lock does not install the baseline when members are already being served. When the decision is already not to serve, the baseline is installed even if the lock is busy.

`node/internal/timeanchor` anchors trusted time at the moment the lease request is sent: signed `issued_at` plus `CLOCK_MONOTONIC`. Wall-clock jumps never extend a deadline. Replaying the same `lease_id` does not move `mono_sent`, so the kernel element timeout is the remaining time, not a new full TTL. After reboot (`boot_id` mismatch) the same lease does not enable forwarding; a strictly newer `issued_at` is required. A process restart on the same boot also keeps forwarding off until a lease response arrives in that process. A replay of the same lease may then enable forwarding without resetting the monotonic deadline.

Renewal starts one lead time before expiry. The default lead is 120 seconds, capped at half the node lease window, and jittered by ±10 percent.

## Forwarding gates

Member forwarding stays off unless all of these hold:

- enrolled, and not revoked (`IDENTITY_REVOKED` is persisted and is not followed by a new enrollment)
- bootstrap was not lost after a 200
- the time anchor checks out
- the store is usable and the observation outbox is under 32 pending rows
- `forwarding.blocked` is absent and `fvpn-updater recover` did not exit 2
- `netbird_backend` is `kernel` and the meter path is readable (`Barrier.Health`; only reason `meter_path_invalid` is `METER_PATH_INVALID`)
- the usage store can persist a sample, the usage outbox is inside its age and byte budgets, and the disk floor is clear
- the isolation baseline is present
- desired state is verified, in force, and `published`
- `serving_release_ids` is empty, or it contains this `release_id` (an empty list passes, the same rule as `CheckPublishable`; a non-empty list that omits this release blocks forwarding)
- this process holds a fresh lease whose monotonic send time is set
- capacity is non-null and QoS renders
- if any device binding is present, at least one is admitted for at least one second
- the data-path probe has finished and its check is `passed`

`safety_status` is the isolation and policy baseline (§5). It does not follow `d.Serve`. The baseline checks are `internet_only`, `dns_policy`, `policy_signature`, `meter_path`, `storage`, `release_supported`, and `tunnel_probe`. They are evaluated while publication is `unpublished` or `disabled`. `safety_status` is `passed` when every baseline check passes, `failed` when one fails, and `expired` only when a policy or lease deadline has passed while publication is `published`. A deadline on an unpublished document is `failed`, not `expired`.

`lease`, `qos`, and `source_binding` are serving checks. They are `not_run` while publication is `unpublished` or `disabled`, and while the desired state has no device bindings. They are `passed` or `failed` only when serving is desired (publication `published` and at least one binding). They do not turn baseline safety to `failed` in the unpublished state. A relay-only overlay (`data_health` `degraded` with `tunnel_probe` still `passed`) does not block forwarding by itself. A failed tunnel check does.

An unpublished desired generation is still applied. The agent installs the fail-closed baseline (member forwarding dropped, isolation baseline, kill switch), persists that generation, and reports `applied_config_generation` equal to the desired generation and `provisioning_state` `ready`. The serving model is not installed, so member traffic stays off. `d.Serve` stays fail-closed: false until publication is `published`, the lease is valid, and every gate above passes. An empty binding list can still serve, as it did before; a non-empty list with nobody admitted does not.

IPv6 is always blocked. A null capacity blocks forwarding. `test_exempt_v4` is applied only when `test_mode` is true; a non-empty exempt list with `test_mode` false is a startup error. `host_reserve_bps` is required. Missing or zero is a configuration error: the process prints the setting name and exits 64 before it opens a store, touches nftables, or dials the control plane (D-034). The production value is 待實測 and must be set by the operator. There is no built-in stand-in.

## Observations

Heartbeats default to 10 seconds. Observation reports default to 30 seconds (§16.5). Reports carry the applied generation, the baseline and serving checks, `safety_status`, `data_health`, the running release, and failure reasons. `data_health` is the probe result: `unknown` only before the first run, then `healthy`, `degraded`, or `failed`. `netbird.peer_id` stays null. NetBird `status --json` has no management peer id, and the agent does not invent one from the public key. Reports are queued in the local outbox and resent until the center accepts them. HTTP 202 means accepted, not verified. 409 and 422 are conflicts and are not retried. 429 and 5xx stay pending.

Each report calls `updatejournal.Snapshot` on `updater_journal` at report time. The call is read-only, does not create or migrate the database, and stops at the caller's deadline or one second. The `update` object is always present. Its members are `generation`, `job_id`, `stage`, and `state`. A null `generation` means no update state: the file is missing, the schema cannot be read, the snapshot missed its deadline, or the database has no job. That is not generation 0, and the next report does not reuse the previous snapshot. A recorded job fills those members from that snapshot. `locally_succeeded` is stage `report` and state `succeeded`. A destructive rollback is stage `report` and state `rolled_back`, and the restored release is the top-level `current_release_id`. Journal stage `recovery_required` is reported as stage `rollback` and state `failed`, which is the API vocabulary the snapshot already uses; `recovery_required` is not an observation state. `current_release_id` and `last_good_release_id` stay top-level fields. While `generation` is set they come from the journal pointers, and a pointer that is not a UUID is null. Otherwise they stay the configured release, with `last_good_release_id` set only after a generation has been applied. A snapshot error is logged once each time its reason class changes (`missing`, `deadline`, `canceled`, `unsupported_schema`, `not_a_file`, `unreadable`). The line does not include the path or the journal contents. The observation is still queued.

## Data-path probe (§5, §13.9)

Each step runs one probe before `Decide`. The probe does not change `publication_intent`, does not install a member allow, and does not open member traffic. A control-plane HTTP failure is not this probe. `control_url` is not a probe target.

Unpublished, the probe reads NetBird overlay reachability: `netbird status --json` (argv, no shell, 2 second limit). Management connected, and either at least one relay available or `relays.total` of 0, is a success (`tunnel_probe` `passed`, `data_health` `healthy`). Management connected with relays configured but none available is `degraded` and the check still `passed`. A status that cannot be read, or management disconnected, is `failed` for both the check and `data_health`. When `egress_probe_target` is set (`host:port`), the probe dials that TCP address only after management is connected. A miss is `degraded` and the check `failed`. A non-null signed `egress_probe` replaces that local target: the probe dials exactly those TCP endpoints, an omitted `timeout_ms` is 2000 ms, and `egress_probe_target` is ignored. Absent or null keeps the local target, or skips the leg when the local target is empty. The `tunnel_probe` check `detail` records which source was used. The health attestation reason stays the short overlay token (`overlay_up`, `overlay_partial`, or `overlay_down`).

Once publication is `published`, the same overlay and egress legs run, and both datapath interfaces (`member_ingress` and `uplink`) must exist. Missing interfaces are `degraded` and the check `failed`, so forwarding stays off. That is the member forwarding path this process can see without an nft exception. A dedicated authenticated test peer with a source, destination, and port allowlist (§13.9) is not installed here. `node/enforce` owns that exception. The lab has no NetBird daemon. It sets `netbird_status_file` to a fixture. `test_mode` does not skip the probe. The registered profile (retries and the stable window) is still 待實測. The agent runs the probe once per step.

## Health attestation (D-053)

The updater writes `health-challenge.json` in `journal_dir`, the same directory as `forwarding.blocked`. On start, and again on every step while that file is present, the agent answers with `health-attest.json` when the challenge names this node, this incarnation, and this boot id, and when this process's `release_id` is the challenge's candidate release. Any other challenge is ignored. The write is `healthattest.WriteAttestation`: a temporary file, mode 0600, fsync, rename over `health-attest.json`, then a directory fsync. The answer echoes the challenge transaction, nonce, and ids. `agent_version` is this process's `agent_release`. `netbird_version` is `daemonVersion` from the NetBird status document the probe just read. It is not copied from the challenge's `components.netbird`. A missing field, a URL, a space, a control character, or a token longer than 64 bytes is reported as an empty version. `published` is the latest publication intent (`published`, `unpublished`, or `disabled`). The file does not change that intent and does not open member forwarding.

The three statuses are this step's own `internet_only`, `dns_policy`, and `tunnel_probe` results, including when publication is `unpublished` or `disabled`. The probe is the data-path probe above, with `Published` false unless the intent is `published`, so member interfaces are not required and member forwarding stays closed. Reasons are short fixed strings (`baseline held`, `policy in force`, or the probe token `overlay_up`, `overlay_partial`, or `overlay_down`). They do not carry DNS names, URLs, or peer lists.

The agent answers only when the probe's wall time is strictly after `issued_at` and a verified desired document is in force. A document loaded from the store counts only when its signature, node, incarnation, and validity window all succeed; the stored ETag is kept only in that case. A sync counts when the control plane returns a response and the body verifies. `If-None-Match` is omitted while no verified document is held, so an envelope that expired while the process was stopped cannot come back as `304`. A transport error with no document in hand writes nothing. A placeholder `not_run` or a DNS failure written in that gap would end the updater's wait, because a file that validates is accepted immediately. The updater keeps polling until its deadline, and that deadline is the fail-closed bound. A verified document that later fails a check, including one that has left its validity window, is written as that result. `generated_at` is this process's wall clock and must fall strictly inside `(issued_at, expires_at)`. An expired challenge, a challenge that is not yet valid, or a consumed challenge is not answered. The same nonce and the same results are not rewritten. A new nonce, a changed result, or a missing file is a new rename, including after a process restart, so an attestation that was already on disk when the challenge appeared is replaced. A write error is logged and does not fail the step. A missing challenge does not fail the start.

Timing follows the updater constants. They are code defaults, the updater config cannot change them, and they are 待實測, not a verified SLO. `HealthAttestWait` is 30s per check attempt. `HealthAttestPoll` is 250ms. `HealthChallengeTTL` is 45s, so `expires_at` is `issued_at` plus 45s. The answer has to land inside both the 45s lifetime and the 30s wait. `Run` answers on boot when a stored desired document is already in force, then runs the first step, and both of those happen before the poll sleep. After that the step interval is the minimum of `heartbeat_seconds` and `poll_seconds`. Both default to 10s, which is inside the 30s wait. The lab sets both to 1s. The 8-attempt cap and the 3-rollback cap are unchanged. The attempt that crosses the cap does not issue a candidate challenge.

Failure semantics. A wrong release, boot, incarnation, or node leaves any existing answer untouched. An empty observed NetBird version is still written when the other binding fields are ready; `healthattest.Validate` then returns `netbird_version_mismatch`, which is not evidence, and the updater keeps reading until the deadline. A real `failed` status, and a `not_run` that means the check truly did not run, are written as themselves. Both end that attempt: `not_run` is `failed` for the updater. The agent does not send `not_run` while a result is still in flight. A passing attestation does not publish the node and does not install a member allow.

The heartbeat file is line-oriented, version `v1`:

```text
v1
boot_id <uuid>
mono_ns <CLOCK_MONOTONIC>
wall_unix_ns <unix nano>
policy_expires_mono_ns <monotonic deadline, or 0 when not serving>
baseline_ok 0|1
serving 0|1
generation <uint>
reason <token or ->
```

`mono_ns` is `CLOCK_MONOTONIC`, shared by the agent and the watchdog. `time.Since` is process-local and is not used for this comparison.

## Watchdog (§16.3)

The watchdog reads the heartbeat, the monotonic clock, and whether table `inet fvpn` exists. It applies `enforce.ApplyBaseline` when the heartbeat is missing, older than `watchdog_stale_seconds` (default 30), from another boot, or when the baseline is gone or the policy deadline has passed. A healthy reading does not render serving rules and does not restart a kernel timeout. Its own restart therefore does not restart a deadline. If both processes die, nft element timeouts still expire members. The watchdog does not take the node lock; fail-closed outranks a busy flock. It does not read `health-challenge.json` and it does not answer one. During `checking` the updater starts the candidate agent, and that process answers from boot and from its step loop. The watchdog is not required for the answer.

## Usage ledger

The agent opens `node/internal/usagestore`, `node/internal/meter` and `node/internal/usageupload` after it knows the node id. The database is `state_dir/usage.db`. The uploader uses the control client's usage class, so it has the same client certificate and the same TLS settings (TLS 1.2 minimum). Its concurrency budget stays separate from the lease budget (§14.6, §14.7). No second TLS stack is constructed.

Sampling starts only after a successful apply that admits forwarding. `meter.New` may reconcile an open epoch when the ledger is opened; the 10 second sample loop does not run before that apply. Counters are `u_<device>` and `d_<device>` on table `inet fvpn`. DNS counters are not usage.

`Barrier.Health` is the meter's health. Reason `meter_path_invalid` (and a `netbird_backend` other than `kernel`) is `METER_PATH_INVALID`. A full usage outbox (`outbox_age`, `outbox_bytes`, `archive_bytes`) blocks forwarding as `outbox_full`, the same reason as the observation outbox cap of 32, which remains its own gate. `persist_failed` stops forwarding as `meter_storage_failed` and is reported as `STORE_UNAVAILABLE` with message `persist_failed`. `disk_space` stops forwarding as `disk_low` and is reported as `DISK_SPACE_LOW` with message `disk_space`. The disk floor (`MinFreeBytes`) is still 待實測 and stays 0 until an operator sets it inside the store options; this configuration file does not carry it.

Stop and hand-off (§13.6 step 8) are `Barrier`, then the fail-closed baseline, then `FinalSample`. That sequence runs when forwarding is withdrawn (a failed gate, a failed apply, or a revoked identity) and when the process stops. A successful apply after that calls `ResumeSampling` (step 11). A clean close reason `final_sample` makes the next epoch `after_final_sample`, which checkpoints the counters as they are and does not recount them. `ErrNoEpoch` and a short wall-clock hold are not warnings.

After every serving apply, and on every sample, the agent holds the current nft `State` and calls `enforce.DetectMeterResets`. The meter package itself only compares counter values (`DetectResets`). A table, forward chain, or generation marker cleared outside the agent is reported as `METER_EPOCH_CHANGE` (report-only, not an HTTP status) when a usage counter was live. The agent then opens one epoch through the meter's existing sample switch, records a `counter_discontinuity` gap, and the serving apply installs the ruleset again. A counter that is zero in both snapshots is idle and does not open an epoch. The agent's own baseline, including the final sample, is not an external reset. A value drop the meter can already see is still `METER_EPOCH_CHANGE`, once per incident, and is not nudged a second time.

Ledger-epoch restore is `usageupload.Restore`: `GET /api/v1/agent/usage-checkpoint`, then `ApplyCheckpoint`, which verifies the `recovery` signature with the node trust store. The agent calls it at startup and whenever `NeedsRestore` is set. A lower watermark is accepted only for a verified new epoch. The agent does not check the signature itself. A failed restore does not by itself block forwarding. Upload continues while member forwarding is stopped.

## Interfaces for the updater

Updater:

- Lock: `node/internal/nodelock`, default `/var/lib/vpn-node/update.lock`. `Acquire` returns `ErrBusy` and does not wait forever.
- Latch: `forwarding.blocked` in the journal directory. `agent.ForwardingBlocked(journalDir)` is true while the file exists.
- Recover: the agent runs `updater_recover recover -config updater_config` once at boot (empty command skips it). Exit 0 clears the block. Exit 2, or any other failure, keeps members blocked. The agent does not import the updater package.
- Desired state: read the verified envelope at `desired_state_path` (default `/var/lib/vpn-node/state/desired-state.json`). The agent is the only writer, and only after verification.
- Health: the candidate agent writes `health-attest.json` for a live `health-challenge.json` in the journal directory (D-053). See Health attestation above. The file does not open the updater's gate. Stock updater checks still do not observe isolation or the tunnel on their own.

## Configuration

| Field | Required | Meaning |
|---|---|---|
| `control_url`, `server_ca_file`, `trust_bundle` | yes | Control origin, pinned CA, node trust bundle |
| `agent_release` | yes | Pinned version. `latest` is rejected |
| `datapath` | yes | `member_ingress` and `uplink` match `^fv[a-z0-9_-]{1,13}$` and differ. `snat` is IPv4 |
| `host_reserve_bps` | yes | Host rate reserve in bit/s. Missing or zero exits 64. Production value is 待實測 and is set by the operator |
| `netbird_backend` | yes for forwarding | `kernel`, `userspace` or `unknown`. Only `kernel` may forward |
| `netbird_status_file` | no | Lab seam. A NetBird `status --json` document read instead of the CLI. Empty runs `netbird status --json` |
| `egress_probe_target` | no | `host:port` for the data-path probe when desired state has no signed `egress_probe`. Empty skips that leg |
| `dns.resolvers[]` | no | Host-local AdGuard targets, at most 8. Each has `admin_url` (`http` or `https`, IP literal, explicit port, no user, query, or path) and `dns_addr` (`host:port` of the DNS listener). `dns.New` requires both. Loopback is allowed. A link-local or private address is allowed only when it is assigned on a host interface. Optional `id` is a short token |
| `dns.resolver_credentials_file` | with `dns.resolvers` | Absolute path of a root-owned mode `0600` JSON file `{"username","password"}`, one pair for every resolver. Loaded at process start and again on `SIGHUP` during `run`. A wrong mode or owner refuses the check with a fixed detail and does not log the secret or the URL |
| `state_dir`, `identity_dir`, `journal_dir`, `lock_path`, `heartbeat_path` | no | Defaults under `/var/lib/vpn-node` and `/run/vpn-node`. Usage ledger is `state_dir/usage.db`. `journal_dir` is the directory for the latch and the health files |
| `updater_journal` | no | Updater journal database file. Default `/var/lib/vpn-node/update-journal/journal.db`. A missing file is no update state. The agent does not create it |
| `release_id` | no | Local release UUID. Empty blocks forwarding as an unknown release |
| `test_mode`, `test_exempt_v4` | no | Exempt prefixes apply only when `test_mode` is true |
| `renew_lead_seconds`, `heartbeat_seconds`, `poll_seconds`, `observation_seconds`, `watchdog_stale_seconds` | no | Zero selects the defaults in §16.3 and §16.5 |

## Config example

```json
{
  "state_dir": "/var/lib/vpn-node/state",
  "identity_dir": "/var/lib/vpn-node/identity",
  "journal_dir": "/var/lib/vpn-node/update-journal",
  "updater_journal": "/var/lib/vpn-node/update-journal/journal.db",
  "lock_path": "/var/lib/vpn-node/update.lock",
  "control_url": "https://control.example.invalid",
  "server_ca_file": "/etc/vpn-node/ca.pem",
  "trust_bundle": "/etc/vpn-node/trust.json",
  "heartbeat_path": "/run/vpn-node/heartbeat",
  "datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
  "agent_release": "0.0.0-test",
  "release_id": "019a0000-0000-7000-8000-0000000000c1",
  "netbird_backend": "kernel",
  "host_reserve_bps": 1000000,
  "updater_recover": "/opt/vpn-node/current/fvpn-updater",
  "updater_config": "/etc/vpn-node/updater.json",
  "dns": {
    "resolvers": [
      {"id": "local", "admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}
    ],
    "resolver_credentials_file": "/etc/vpn-node/resolver-credentials.json"
  }
}
```

`agent_release` must be a pinned version. `latest` is rejected. Prices and fair-use thresholds are not in this file. `host_reserve_bps` is required and must be greater than zero. The number in the example is not a measured reserve. The production value is 待實測 and must be set by the operator. The only TEST ONLY value lives in the lab configuration.

`dns` is optional. Leave it out and a referenced resolver document still fails closed: outcome `not_run`, `DEPENDENCY_UNAVAILABLE`, detail "Resolver policy did not run.", and no AdGuard client. When `dns` is present, both `resolvers` and `resolver_credentials_file` are required. The credential file is not inline in `agent.json`. `fvpn-agent run` reloads that file on `SIGHUP` and does not stop. A file that is not a regular mode `0600` owned by uid 0 is not used. The next referenced document then fails `dns_policy` with `FORBIDDEN` and the detail "Resolver credentials are not usable." A desired state that does not name a document does not fail `dns_policy` for that file. `fvpn-agent step` and `enroll` load the file once at start.

## Decisions

- Trusted time is `CLOCK_MONOTONIC` plus the signed server time captured when the lease request is sent. `CLOCK_BOOTTIME` is not used.
- A process restart fail-closes until a lease response in that process. The same `lease_id` does not reset `mono_sent`. After reboot, only a strictly newer `issued_at` can enable forwarding.
- A 304 skips desired-state acceptance only. The step still renews a due lease and still enforces.
- Capacity null and a non-kernel NetBird backend block forwarding. A non-empty `serving_release_ids` that omits this release blocks forwarding. An empty list passes.
- Baseline safety does not require the node to be serving. An unpublished generation is acknowledged after the fail-closed baseline is installed.
- `data_health` `unknown` means the probe has not run yet. Forwarding stays off until `tunnel_probe` is `passed`. `degraded` with a passed check does not, by itself, block forwarding.
- Lab exempt prefixes require `test_mode`. The probe still runs when `test_mode` is true.
- A bootstrap secret lost after HTTP 200 and before the 0600 write is unrecoverable.
- `ISOLATION_BASELINE_MISSING` and `METER_EPOCH_CHANGE` are report-only observation codes. `HEALTH_CHECK_FAILED` is not used as their stand-in.
- `host_reserve_bps` missing or zero exits 64 before the kernel or the network. A zero that still reaches `Decide` fails closed and is not replaced.
- Usage caps do not reuse `METER_PATH_INVALID`. A clean hand-off does not recount admission bytes.
- A health attestation reports the checks this process just ran. It does not open member forwarding, does not change publication intent, and is ignored when the challenge names another release, boot, incarnation, or node. The watchdog does not answer it. Nothing is written until a verified desired document is in force. A transport error before that document is not an answer.
- Resolver admin URLs and the DNS listen address are host-local. The credential file is root-owned mode `0600` and is loaded at start and on `SIGHUP`. The URL and the password are not written to logs, observations, or check detail.

## 待實測

- Suspend versus `CLOCK_MONOTONIC` versus nft timeout jiffies. The lab does not step the clock (`date -s` is not isolated in this container).
- Filesystem flush order beyond `synchronous=FULL` and a directory fsync.
- Real arm64 hardware. CI cross-compiles `GOARCH=arm64` only.
- The §13.9 test-peer exception inside the datapath (source, destination, protocol, and port). This agent checks NetBird management and relay reachability, an optional egress target, and, once published, that the member interfaces exist. The registered timeout, retry count, and stable window are still 待實測. The probe currently runs once per step with a 2 second limit.
- The production host rate reserve (`host_reserve_bps`). It is 待實測 and must be set by the operator. The agent no longer substitutes a stand-in.
