# Node agent and watchdog

`fvpn-agent` (`node/agent/cmd/fvpn-agent`) is the program on every exit node. It enrolls once, proves its identity with mTLS, fetches signed desired state, applies isolation and QoS through `node/enforce`, renews leases, and reports observations. `fvpn-watchdog` (`node/watchdog/cmd/fvpn-watchdog`) is a separate process. The lab driver is `fvpn-labcontrol` (`node/internal/fakecontrol/cmd/fvpn-labcontrol`), which has to live under `node/` so it can import the fake control server. If the agent stops refreshing its heartbeat, the watchdog installs the fail-closed baseline and does not open a control connection (§16.3).

The updater is a separate program. This process opens the usage ledger, samples forward-admission counters after forwarding is admitted, and uploads them. It does not apply a release.

The Ubuntu installer is [install.md](install.md).

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
2. The agent checks `nft`, `tc`, `ip` and `conntrack` through its executor, then installs the fail-closed nftables baseline before it reads a stored lease. The host requires nftables, iproute2 and conntrack. A missing executable keeps forwarding off and reports `DEPENDENCY_UNAVAILABLE` with `safety_check_failed` in the first observation; its log identifies only the executable name.
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
| `/var/lib/vpn-node/update.lock` | — | flock shared by the agent's serving apply, the installer and the updater (`node/internal/nodelock`); the probe exception only checks that another process holds it and never keeps it (amended D-064) |
| `/run/vpn-node/heartbeat` | 0644 | agent; the watchdog only reads it |
| `/etc/vpn-node/agent.json` | — | installer; no secrets |

The updater journal database is not created here. `updater_journal` is that database file, default `/var/lib/vpn-node/update-journal/journal.db`, the same directory the updater uses. `journal_dir` is the directory, not the file: `forwarding.blocked`, `health-challenge.json`, and `health-attest.json` stay there. `usage.db` is created by the agent the first time an enrolled process boots. Every path above is configurable in `agent.json` except the usage file name, which is always `usage.db` inside `state_dir`. Tests use a temp directory.

The desired-state file is replaced by a temp file and `rename` in the same directory, after `AcceptDesired` has stored the verified envelope. An unverified document is never written there. The store is the anti-rollback authority: a generation older than the accepted one is rejected and the file is left as it was. On boot the agent rewrites the file from the stored envelope.

## Local store

`node/internal/agentstore` keeps identity references, the accepted desired-state generation, the last valid envelope, the lease and its monotonic send time, the time anchor, QoS class minors, the last counter snapshot, admitted addresses keyed by `(addr, device_id, member_id)`, observation outbox rows, and enrollment request state. Schema version is `PRAGMA user_version` (3). A newer schema refuses to open. An older `applied_address` that stored only the address is rebuilt in place; those rows keep the nil UUID for device and member. Safety writes use `synchronous=FULL` and finish before the agent acknowledges the corresponding control result.

## Enrollment (§6)

The node generates an ECDSA P-256 key (mode 0600) and a CSR with an empty subject. `enrollment_request_id` and the incarnation id are stored before the request is sent, and the same CSR is reused, so a crash and a retry present one identity. The certificate and CA chain are stored before enrollment is marked complete. Bounded backoff is 5 attempts, 200 ms doubling, capped at 5 s.

## Control client (§14.6, D-015)

`node/internal/controlclient` is an mTLS client with a pinned CA, TLS 1.2 minimum, no redirects, and a 1 MiB body cap. Lease renewal, usage upload, and every other call have separate connection pools and concurrency budgets (lease 2, usage 1, other 2). A usage upload cannot take the last lease slot.

`POST /api/v1/agent/bootstrap` is called once over mTLS with no body. The setup key is written to `setup.key` (mode 0600) before any caller is told it exists, and it is never logged. A lost HTTP 200 cannot be fetched again (`BOOTSTRAP_CONSUMED`). `BOOTSTRAP_NOT_READY` (409) is retried later and does not by itself disable forwarding. `BOOTSTRAP_EXPIRED` (410) without a stored key ends bootstrap for this incarnation.

## Desired state and leases (§14.7, §16.3)

Envelopes are verified with `pkg/envelope` and the node trust bundle (`envelope.ParseNodeTrustStoreJSON`). The node id, incarnation, generation, and half-open validity window `[issued_at, expires_at)` must match. A bad signature, the wrong node, or an older generation does not replace the accepted state. A 304 does not move the lease, the ETag, or any deadline; the same step may still renew a lease that is actually due.

On a verification or render failure the agent reports the reason and keeps member forwarding off. `ISOLATION_BASELINE_MISSING` is a report-only `ErrorCode` (revision 3, HTTP status none). The observation uses that code, stage `isolation`, and the message `ISOLATION_BASELINE_MISSING`. It is never written as an HTTP response. `HEALTH_CHECK_FAILED` stays for a health check that is not one of the specific codes. `METER_PATH_INVALID` is a real error code. `netbird_backend` must be `kernel`; `userspace` and `unknown` keep forwarding off.

The agent translates the accepted document into an `enforce.Model`, then calls `enforce.Apply`. QoS class minors (`QoS.Assignments`) are stored so a restart does not renumber classes. The node lock is tried three times, 50 ms apart, around a serving apply. If it is busy the apply is skipped and `UPDATE_BUSY` is reported. A busy lock does not install the baseline when members are already being served. When the decision is already not to serve, the baseline is installed even if the lock is busy.

An apply failure restores the baseline and becomes an input to the forwarding decision, except for the verified conntrack deletion failure described below. It reports `safety_check_failed`, `member_forwarding_enabled=false`, and failed `source_binding`, `qos` and `lease` checks carrying the apply error code. Heartbeat does not say `serving`, and the failed apply cannot acknowledge a generation or trigger a generation-driven lease renewal. The failure remains evidence until a successful serving apply verifies the ruleset.

D-085 live serving verification also runs on every serving step: before reconciliation against the retained applied model and class assignments, after Apply returns before acknowledging its generation, and immediately before the deferred heartbeat. `Plane.VerifyServing` / `SystemPlane.VerifyServing` call `enforce.VerifyServing` to read live nft and tc; a 304, unchanged generation, or busy apply lock cannot reuse cached verification. Each read has a timeout of `min(1 second, min(heartbeat_seconds, poll_seconds) / 2)`. The run loop subtracts step work from the shorter heartbeat/poll interval instead of sleeping another full interval; grant-less renewal slots can wake it earlier. Other blocking control work or CPU stalls can still overrun the scheduled cadence, and the watchdog remains the independent stale-heartbeat backstop.

A rejected or timed-out live read uses the usage hand-off and baseline path, clears serving and policy expiry, and reports the failure while retaining control access. A failure found before reconciliation ends that step without Apply; recovery needs a later clean full Apply. `TC_FOREIGN_FILTER` remains the message under `HEALTH_CHECK_FAILED` until the contract consumer update; a plain timeout reports `serving verification timed out`. `rejectServing(ctx, model, evidence, err)` is the follow-up 22 integration point for D-090, with the model and `enforce.ServingEvidence` available before withdrawal. The current baseline does not quarantine a retained foreign ingress redirect. These checks close cached heartbeat claims; sequential nft/tc/netlink reads and the interval from the last read to the heartbeat are still non-atomic.

Run `make -f mk/node.mk node-test-live-netlab` for the foreground container regression and per-step read-cost measurement. It uses real nft/tc with an in-process engine and TLS fake control on isolated veth links, creates only `fvpn-w2-agent-core-*` Docker objects, uses worktree disk scratch, and cleans up on exit. It measures read latency without traffic; it does not establish §11.7 throughput or real-device/arm64 performance.

Revocation is enforced by the ruleset first: removed bindings are dropped before established-flow acceptance. Each tracked address is `(addr, device_id, member_id)`, in memory and in `applied_address`. Moving the same IPv4 to another device or member queues that IPv4 for `conntrack -D` before the new device is served, including when the previous generation was unpublished and the tracked set was left in place. The same triple admitted again is not deleted. A migrated row with a nil device and member is not that triple. When the address is bound to a device, the agent queues `conntrack -D` for it and keeps that identity until the delete succeeds, on the same twelve-attempt bound as any other revoke.

If serving and QoS verification passed (`ApplyResult.Verified`) and only the conntrack flush failed (`IsConntrackDeleteFailure`), the agent retains and acknowledges the applied generation and keeps forwarding for up to twelve consecutive failed attempts. The count is attempts, not wall time: one verified serving apply is one attempt, so a long quiet gap does not spend the window. Twelve attempts at the default 10 second step are about two minutes. That is long enough that one transient `conntrack` error does not drop every member, and short enough that a broken delete cannot leave a revoked device's flows in place. The pending set is the identities whose `conntrack -D` failed. A re-admitted `(addr, device, member)` is dropped after that attempt, so the next tick does not run `conntrack -D` against the active device. Only the addresses whose delete failed stay pending and are retried; every other address in that revoke set is retired. While the count is under twelve the observation is one `ISOLATION_POLICY_INVALID` error at stage `apply`, message `conntrack delete pending; serving ruleset verified`, with `member_forwarding_enabled=true`, `safety_status=passed`, and every check passed. The `serving` heartbeat stays.

On the twelfth consecutive failure, or immediately when `LookPath("conntrack")` fails after preflight, the agent installs the baseline. `member_forwarding_enabled` is false, `safety_status` is `failed`, `forwarding_block_reasons` contains `safety_check_failed`, and `internet_only` is `failed` with `ISOLATION_POLICY_INVALID`. The detail is `Conntrack delete exceeded the retry bound.` or `Conntrack is not installed.` The attempt count is stored, so a restart does not open a new window. A later flush that deletes the pending set, or the binary returning before the twelfth failure, clears the block. A verification failure still restores the baseline at once. Startup preflight uses `enforce.RequireConntrack` so a missing binary fails closed before serving begins.

`node/internal/timeanchor` anchors trusted time at the moment the lease request is sent: signed `issued_at` plus `CLOCK_MONOTONIC`. Wall-clock jumps never extend a deadline. Replaying the same `lease_id` does not move `mono_sent`, so the kernel element timeout is the remaining time, not a new full TTL. After reboot (`boot_id` mismatch) the same lease does not enable forwarding; a strictly newer `issued_at` is required. A process restart on the same boot also keeps forwarding off until a lease response arrives in that process. A replay of the same lease may then enable forwarding without resetting the monotonic deadline.

Renewal starts one lead time before expiry. The default lead is 120 seconds, capped at half the node lease window, and jittered by ±10 percent. A lease that carries member grants still waits for that lead.

A lease for this desired generation whose member grant set is empty, while device bindings are still desired, is provisional (D-067). The node lease is still inside its monotonic deadline, and the lease generation matches the desired generation. The agent does not forge a grant, force the node ready, or shorten a production timeout.

Every accepted grant-less lease gets a monotonic renewal schedule, including an empty lease received before desired bindings arrive. The first wait is one heartbeat capped at 60 seconds; later waits double up to 60 seconds. Each slot is measured from the request send time, and the lease's renewal lead can bring it forward. Thus the central 60-second lease issued while a published node is still `applying`, with `renew_lead_seconds=120` capped to half its window, is scheduled at or before the 30-second lead slot instead of backing off to expiry. Grant-less slots have no jitter. A lease with grants clears this schedule and retains the ordinary renewal lead and jitter.

The run loop shortens its sleep to the next slot and handles a due grant-less renewal before observation retries and desired-state HTTP work. Time already spent in a control pass adds no extra full poll interval after the slot. HTTP `Retry-After` remains an independent lower bound; failures or replayed leases past their lead retry without a busy loop. A restart starts a fresh backoff streak, but replay never extends the original lease deadline (§16.3). The 60-second schedule bounds local waiting, not central availability or member eligibility; ongoing work, request latency, and server backoff can delay completion.

The [devstack follow-up 14 record](../e2e/runs/2026-10-31/followup14/member-traffic.md) measured 80.622 seconds from the first `ready` observation to the first populated lease. That query spans setup: the [publication record](../e2e/runs/2026-10-31/followup14/publish.md) still shows the node unpublished, generation 1, at 70.304 seconds after that first readiness. A ready node can have no eligible members. For renewal latency, compare successive lease requests, or readiness after a published node has eligible desired bindings, rather than the first-ever `ready` to first-ever non-empty lease.

While that deadline holds, member forwarding stays off and `safety_status` stays `passed`. `forwarding_block_reasons` contains `awaiting_member_grants` and does not contain `lease_expired`. Contract revision 11 is adopted. The `lease` check is `failed`, with `error_code` `AWAITING_MEMBER_GRANTS` and detail `Serving lease has no member grants yet.` The `qos` and `source_binding` checks in that state are `failed` with `error_code` `AWAITING_MEMBER_GRANTS` and the details `Rate policy could not be applied.` and `No member binding was admitted.` `errors` contains one entry: code `AWAITING_MEMBER_GRANTS`, stage `lease`, message `Serving lease has no member grants yet.` A deadline that has passed is a different report. `forwarding_block_reasons` then contains `lease_expired`, and the `lease` check is `failed` with `LEASE_EXPIRED` and the detail `Serving lease is not in force.` An empty desired binding list is not provisional. It reports neither `awaiting_member_grants` nor `AWAITING_MEMBER_GRANTS`. A lease for another generation is not provisional either. It reports `policy_missing`, and the `lease` check is `STALE_GENERATION`. A lease that carries grants has no `lease` failure.

For isolation sampling, the `AWAITING_MEMBER_GRANTS` window runs from acceptance of the issued empty lease for a published generation with desired bindings until the first renewal returning grants, while the matching lease stays in force: D-087 schedules one heartbeat, then doubling, never more than 60 seconds of local waiting between requests, plus central publication/eligibility time and the I/O/backoff qualifications above. Start capturing before empty-lease issuance, configure test `heartbeat_seconds=1`, `poll_seconds=1`, `observation_seconds=1`, and archive every `POST /api/v1/agent/observations` body at the harness control receiver (drain that complete capture every 100 ms, not just the latest snapshot every 2 s): retain `report_id`, `observed_at`, `lease.lease_id`, `member_forwarding_enabled=false`, `safety_status=passed`, `forwarding_block_reasons` containing `awaiting_member_grants`, `checks` for `lease`/`qos`/`source_binding` with `error_code=AWAITING_MEMBER_GRANTS`, and the single `errors` entry with that code and `stage=lease`.

The same step that applies a desired generation also requests a lease when the stored lease's config generation is not that generation. The request does not wait for the lead. The lease the agent serves against carries the applied generation. A lease from an older generation is never a basis for `serving=1`, and it does not by itself produce `lease_expired`. `lease_expired` means the anchored deadline of the lease for the applied generation has passed: monotonic time has reached `expires_at` measured from the send. Until a lease for the applied generation is accepted, heartbeat and observation use `policy_missing`. The forwarding-block vocabulary has no `lease_pending` or `lease_renewing`, and `policy_missing` is the existing token for a serving document that is not in hand. A failed renewal leaves forwarding off with that reason. The lead, its jitter, the rest of the bounded reason set, and the §13.9 forwarding latch are unchanged.

A step makes at most one lease renewal request. A `Retry-After` response retains a monotonic retry deadline across steps without invalidating a fresh lease for the applied generation. While that lease remains in force, forwarding and the `serving` heartbeat continue; the `lease` check passes with a detail that renewal is waiting for `Retry-After`. Its original monotonic deadline still withdraws forwarding with `lease_expired`, and a generation mismatch withdraws it with `policy_missing`. The agent uses the client's single-attempt method so heartbeats and observations continue during backoff; the ordinary `RenewLease` method waits for the deadline or context cancellation without making an automatic second request.

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
- this process holds a fresh lease for the applied generation, and its monotonic send time is set
- capacity is non-null and QoS renders
- if any device binding is present, at least one is admitted for at least one second
- the data-path probe has finished and its check is `passed`

`safety_status` is the isolation and policy baseline (§5). It does not follow `d.Serve`. The baseline checks are `internet_only`, `dns_policy`, `policy_signature`, `meter_path`, `storage`, `release_supported`, and `tunnel_probe`. They are evaluated while publication is `unpublished` or `disabled`. `safety_status` is `passed` when every baseline check passes, `failed` when one fails, and `expired` only when a policy or lease deadline has passed while publication is `published`. A deadline on an unpublished document is `failed`, not `expired`.

`lease`, `qos`, and `source_binding` are serving checks. Except when an enforcement apply or executable preflight has failed, they are `not_run` while publication is `unpublished` or `disabled`, and while the desired state has no device bindings. They are `passed` or `failed` only when serving is desired (publication `published` and at least one binding). They do not turn baseline safety to `failed` in the unpublished state. A relay-only overlay (`data_health` `degraded` with `tunnel_probe` still `passed`) does not block forwarding by itself. A failed tunnel check does.

An unpublished desired generation is still applied. The agent installs the fail-closed baseline (member forwarding dropped, isolation baseline, kill switch), persists that generation, and reports `applied_config_generation` equal to the desired generation and `provisioning_state` `ready`. The serving model is not installed, so member traffic stays off. `d.Serve` stays fail-closed: false until publication is `published`, the lease is valid, and every gate above passes. An empty binding list can still serve, as it did before; a non-empty list with nobody admitted does not.

IPv6 is always blocked. A null capacity blocks forwarding. `test_exempt_v4` is applied only when `test_mode` is true; a non-empty exempt list with `test_mode` false is a startup error. `host_reserve_bps` is required. Missing or zero is a configuration error: the process prints the setting name and exits 64 before it opens a store, touches nftables, or dials the control plane (D-034). The production value is 待實測 and must be set by the operator. There is no built-in stand-in.

## Observations

Heartbeats default to 10 seconds. Observation reports default to 30 seconds (§16.5). Reports carry the applied generation, the baseline and serving checks, `safety_status`, `data_health`, the running release, and failure reasons. `data_health` is the probe result: `unknown` only before the first run, then `healthy`, `degraded`, or `failed`. `netbird.peer_id` stays null. NetBird `status --json` has no management peer id, and the agent does not invent one from the public key. Reports are queued in the local outbox and resent until the center accepts them. HTTP 202 means accepted, not verified. 409 and 422 are conflicts and are not retried. 429 and 5xx stay pending.

Each report calls `updatejournal.Snapshot` on `updater_journal` at report time. The call is read-only, does not create or migrate the database, and stops at the caller's deadline or one second. The `update` object is always present. Its members are `generation`, `job_id`, `stage`, and `state`. A null `generation` means no update state: the file is missing, the schema cannot be read, the snapshot missed its deadline, or the database has no job. That is not generation 0, and the next report does not reuse the previous snapshot. A recorded job fills those members from that snapshot. `locally_succeeded` is stage `report` and state `succeeded`. A destructive rollback is stage `report` and state `rolled_back`, and the restored release is the top-level `current_release_id`. Journal stage `recovery_required` is reported as stage `rollback` and state `failed`, which is the API vocabulary the snapshot already uses; `recovery_required` is not an observation state. `current_release_id` and `last_good_release_id` stay top-level fields. `current_release_id` defaults to the running binary's configured release, including when the journal pointer is empty or invalid. A journal pointer overrides it only when the job ID is a valid nonzero UUID and the transaction reached `locally_succeeded` or `rolled_back`. Claimed, in-progress, and recovery-required jobs cannot override it. With a recorded generation, `last_good_release_id` still comes from the journal pointer (null if invalid); otherwise it is the configured release only after a generation has been applied. A snapshot error is logged once each time its reason class changes (`missing`, `deadline`, `canceled`, `unsupported_schema`, `not_a_file`, `unreadable`). The line does not include the path or the journal contents. The observation is still queued.

The `usage` object is live outbox status, read from the open usage store when the report is built (§10.5, §10.6, §10.8). The read is bounded to about 2 seconds. Before the store is open, and when the read fails, the report keeps the zero shape: null `ledger_epoch`, `meter_epoch_id`, `blocked_seq` and `oldest_unacked_at`, and `"0"` or 0 for the other fields. A read that reaches the time limit sends the last successful status from this process. Neither outcome adds an `errors[]` entry, and neither changes `safety_status`, `provisioning_state`, or member forwarding. The agent logs `usage status unavailable: <reason>` or `usage status available` once when that state changes.

- `ledger_epoch` is the central ledger epoch adopted from a verified checkpoint, or null when none has been accepted.
- `meter_epoch_id` is the open meter epoch, or null when none is open.
- `last_durable_event_seq` is the highest durable event sequence, `"0"` when no event exists.
- `central_acked_through` is the contiguous ACK watermark, `"0"` when none has been acknowledged.
- `blocked_seq` is the lowest sequence that cannot be acknowledged, or null when nothing is blocked.
- `unacked_event_count` is the number of events still in the outbox.
- `unacked_payload_bytes` is the sum of the canonical payload bytes of those unacknowledged events.
- `oldest_unacked_at` is the time the oldest outbox row was stored, or null when the outbox is empty.
- `acked_archive_bytes` is the sum of archived payload bytes still retained for replay.

Central stores the whole report, including `usage`, in `node_observations.report_payload`. The same acceptance updates the `nodes` row's control status, last-seen and last-observed times, applied generation, provisioning state, data health, safety status, observed member forwarding, current and last-good release ids, latest checks, and recent errors. The usage fields remain inside that observation payload.

## Data-path probe (§5, §13.9)

Each step computes a preliminary `Decide` before probing, then recomputes it with the probe result. Losing a serving gate first withdraws members to the baseline. The probe does not change `publication_intent`, does not install a member allow, and does not open member traffic. A control-plane HTTP failure is not this probe. `control_url` is not a probe target.

Unpublished, the probe reads NetBird overlay reachability: `netbird status --json` (argv, no shell, 2 second limit). Management connected, and either at least one relay available or `relays.total` of 0, is a success (`tunnel_probe` `passed`, `data_health` `healthy`). Management connected with relays configured but none available is `degraded` and the check still `passed`. A status that cannot be read, or management disconnected, is `failed` for both the check and `data_health`. When `egress_probe_target` is set (`host:port`), the probe dials that TCP address only after management is connected. A miss is `degraded` and the check `failed`. A non-null signed `egress_probe` replaces that local target: the probe dials exactly those TCP endpoints, an omitted `timeout_ms` is 2000 ms, and `egress_probe_target` is ignored. Absent or null keeps the local target, or skips the leg when the local target is empty. The `tunnel_probe` check `detail` normally records the operator `ProbeSource` sentence: `Egress probe used the signed targets.`, `Egress probe used the local target.`, `Egress probe used no extra target.`, or `Signed egress probe was rejected.` The health attestation reason stays the short overlay token (`overlay_up`, `overlay_partial`, or `overlay_down`).

A live matching challenge may instead carry `probe_omitted`: a dedicated synthetic peer was configured, but the issuer refused it under D-063. The observation then reports exactly `probe_omitted: <reason>` as `tunnel_probe.detail`, taking precedence over the operator source sentence. The reason comes only from the shared reader's fixed omission vocabulary; the detail is bounded to 500 characters and never includes the peer ID or an address. The agent retains the reason and the check's `ObservedAt` after sidecar removal or consumption, until a later challenge with a probe or a later omission replaces that evidence. Ordinary overlay checks do not replace it. An omission does not count as a run synthetic probe: `ProbeRan=false` keeps the existing `not_run` status and unknown data health, without adding an error code; a missing baseline still fails the check. Overlay reachability and an observed NetBird version cannot turn the omission into a passing synthetic probe.

Compatibility boundary: `healthattest.ReadChallenge` still uses `DisallowUnknownFields`; agents rebuilt with updater follow-up 11 (#168) decode `probe_omitted`, while older candidate or retained rollback binaries reject omission-bearing challenges and fail closed. Unrelated unknown members, invalid omissions, and challenges containing both `probe` and `probe_omitted` remain rejected.

Once publication is `published`, the same overlay and egress legs run, and both datapath interfaces (`member_ingress` and `uplink`) must exist. Missing interfaces are `degraded` and the check `failed`, so forwarding stays off. When the health challenge names a synthetic probe (§13.9), this process installs that exception for the dedicated test peer in the challenge and removes it when the probe returns. `node/enforce` still owns the rules. The lab has no NetBird daemon. It sets `netbird_status_file` to a fixture. `test_mode` does not skip the probe. The registered profile (retries and the stable window) is still 待實測. The agent runs the probe once per step.

## Health attestation (D-053)

The updater writes `health-challenge.json` in `journal_dir`, the same directory as `forwarding.blocked`. On start, and again on every step while that file is present, the agent answers with `health-attest.json` when the challenge names this node, this incarnation, and this boot id, and when this process's `release_id` is the challenge's candidate release. Any other challenge is ignored. The write is `healthattest.WriteAttestation`: a temporary file, mode 0600, fsync, rename over `health-attest.json`, then a directory fsync. The answer echoes the challenge transaction, nonce, and ids. `agent_version` is this process's `agent_release`. `netbird_version` is `daemonVersion` from the NetBird status document the probe just read. It is not copied from the challenge's `components.netbird`. A missing field, a URL, a space, a control character, or a token longer than 64 bytes is reported as an empty version. `published` is the latest publication intent (`published`, `unpublished`, or `disabled`). The file does not change that intent and does not open member forwarding.

The three statuses are this step's own `internet_only`, `dns_policy`, and `tunnel_probe` results, including when publication is `unpublished` or `disabled`. The probe is the data-path probe above, with `Published` false unless the intent is `published`, so member interfaces are not required and member forwarding stays closed. Reasons are short fixed strings (`baseline held`, `policy in force`, or the probe token `overlay_up`, `overlay_partial`, or `overlay_down`). An omitted synthetic probe uses the existing accepted token `probe refused`, including for `not_run`; the observation's `probe_omitted: <reason>` detail is never copied into the attestation reason. No new attestation token is needed. Reasons do not carry DNS names, URLs, or peer lists.

The agent answers only when the probe's trusted time is strictly after `issued_at` and a verified desired document is in force. A document loaded from the store counts only when its signature, node, incarnation, and validity window all succeed; the stored ETag is kept only in that case. A sync counts when the control plane returns a response and the body verifies. `If-None-Match` is omitted while no verified document is held, so an envelope that expired while the process was stopped cannot come back as `304`. A transport error with no document in hand writes nothing. A placeholder `not_run` or a DNS failure written in that gap would end the updater's wait, because a file that validates is accepted immediately. The updater keeps polling until its deadline, and that deadline is the fail-closed bound. A verified document that later fails a check, including one that has left its validity window, is written as that result. `generated_at` uses the same signed-time anchor as probe installation; without trusted time neither installation nor an answer is allowed and the decision reports `clock_untrusted`. It and must fall strictly inside `(issued_at, expires_at)`. An expired challenge, a challenge that is not yet valid, or a consumed challenge is not answered. The same nonce and the same results are not rewritten. A new nonce, a changed result, or a missing file is a new rename, including after a process restart, so an attestation that was already on disk when the challenge appeared is replaced. A write error is logged and does not fail the step. A missing challenge does not fail the start.

Timing follows the updater constants. They are code defaults, the updater config cannot change them, and they are 待實測, not a verified SLO. `HealthAttestWait` is 30s per check attempt. `HealthAttestPoll` is 250ms. `HealthChallengeTTL` is 45s, so `expires_at` is `issued_at` plus 45s. The answer has to land inside both the 45s lifetime and the 30s wait. `Run` answers on boot when a stored desired document is already in force, then runs the first step, and both of those happen before the poll sleep. After that the step interval is the minimum of `heartbeat_seconds` and `poll_seconds`. Both default to 10s, which is inside the 30s wait. The lab sets both to 1s. The 8-attempt cap and the 3-rollback cap are unchanged. The attempt that crosses the cap does not issue a candidate challenge.

Failure semantics. A wrong release, boot, incarnation, or node leaves any existing answer untouched. An empty observed NetBird version is still written when the other binding fields are ready; `healthattest.Validate` then returns `netbird_version_mismatch`, which is not evidence, and the updater keeps reading until the deadline. A real `failed` status, and a `not_run` that means the check truly did not run, are written as themselves. Both end that attempt: `not_run` is `failed` for the updater. The agent does not send `not_run` while a result is still in flight. A passing attestation does not publish the node and does not install a member allow.

A challenge `probe` object names the test peer, its /32 source, the destination sockets, and whether resolver DNS is included. The updater now issues it when its validated `probe_peer` configuration names a dedicated peer and team-controlled public sockets; otherwise it omits the object. D-064 is amended: the §13.9 exception is installed only while the node update lock is held by another process (`nodelock.TryAcquire` on `lock_path` returns busy) and a live challenge names this process and is `Same` as the challenge being answered. With a free lock, no update is in progress: no exception is installed, and the challenge is answered from the node's normal state. Lock errors other than contention do not authorize installation. Installation and the answer both use `trustedNow()`, require `issued_at < now < expires_at`, and refuse a missing anchor or clock anomaly. Installation additionally requires at least one second remaining and an issuance window no longer than `ProbeMaxSeconds` (300). `updatejournal.Snapshot` currently has no active transaction ID; when that API exposes one, the challenge `TxnID` must match it as well.

The agent durably records the issuance nonce, canonical JSON challenge digest (no destination list), original boot-relative monotonic deadline and completion before an exception can be reopened by a restart. A surviving incomplete issuance can only be re-capped to the smaller of its original remaining deadline and `expires_at − trustedNow`; an issuance already cleared cannot be reinstalled. Later steps in the same process never reinstall it, even if writing the answer fails. `TimeoutCeilSeconds` is the remaining whole seconds at installation. Removal runs immediately when the probe returns, before the attestation; kernel elements expire by the original `ExpiresAt` if the process stalls or crashes.

A probe has at most `ProbeMaxDestinations` (4) destinations, none of which may overlap a member `AllowedV4`. `ProbeException.TimeoutCeilSeconds`, capped by the remaining kernel timeout already in `State`, stops a re-render from extending `@probe_v4` or `@probe_dst_v4` past the original `ExpiresAt` and past `ProbeMaxSeconds` (300).

Under a held update lock, both installation and removal use `ApplyBaselineModel`: the exception model has no member bindings, and removal re-applies the bare baseline. A previously serving node withdraws member forwarding before installation. The serving apply in `enforce()` still takes the update lock and reports `UPDATE_BUSY` as before. Cleanup failure sets `baselineOK` and serving false and marks the probe `probe_clear_failed`. Checks are recomputed after enforcement changes baseline, meter or apply health and before attestation; a missing baseline fails both `internet_only` and `tunnel_probe`.

A member peer, a source that is not a /32, and any source or destination colliding with a current member binding or node address are refused, including node admin ports. Overlay source ranges (private or CGNAT) are expected and are not destination-filtered. Destination sockets are checked against `enforce.SpecDeniedV4()` and must be exact /32 TCP or UDP sockets. If `resolver_dns` is set, the model includes the resolvers from the desired document; DNS rule ordering belongs to `node/enforce`. The watchdog still calls `ApplyBaseline`, which carries no probe. `probe_up` and `probe_down` stay outside member usage; member counter resets detected by the normal serving apply still open the meter epoch and record the discontinuity.

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

`dns` is optional. Leave it out and a referenced resolver document still fails closed: outcome `not_run`, `DEPENDENCY_UNAVAILABLE`, detail "Resolver policy did not run.", and no AdGuard client. When `dns` is present, both `resolvers` and `resolver_credentials_file` are required. The credential file is not inline in `agent.json`. `fvpn-agent run` reloads that file on `SIGHUP` and does not stop. A credential file must be regular mode `0600`, owned by uid/gid `0:0`, with exactly one hard link. The device and inode from `Lstat` and the opened file's `Stat` must match; a replacement or symlink is refused. The next referenced document then fails `dns_policy` with `FORBIDDEN` and the detail "Resolver credentials are not usable." A desired state that does not name a document does not fail `dns_policy` for that file. `fvpn-agent step` and `enroll` load the file once at start.

Resolver apply evidence is bound to the signed reference's digest and decimal generation. The hash is compared without case and with surrounding whitespace removed; generation is compared as a decimal string, never through a floating-point number. The applied record stores the verified applier digest.

| Resolver evidence | Outcome | `dns_policy` / error |
|---|---|---|
| Verified read-back, matching digest and generation | `applied` | `passed` |
| Applier digest differs from the signed reference | `failed` | `failed` / `VALIDATION_FAILED` |
| Applier generation differs from the signed reference | `failed` | `failed` / `VALIDATION_FAILED` |
| Applier unavailable or no verified read-back | `not_run` | `failed` / `DEPENDENCY_UNAVAILABLE` |
| Rollback did not restore every resolver | `partial` | `failed` / `ROLLBACK_FAILED` |
| Other apply error | `failed` | `failed` / `VALIDATION_FAILED` |

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
- A health attestation reports the checks this process just ran. It does not open member forwarding, does not change publication intent, and is ignored when the challenge names another release, boot, incarnation, or node. The watchdog does not answer it. Nothing is written until a verified desired document is in force. A transport error before that document is not an answer. A challenge that names a §13.9 test peer installs that exception only for the probe, then removes it.
- `dns_policy` passes only after this process has applied the named resolver document (same id and digest). A revoked skip does not call Apply again and does not report that check passed.
- Resolver admin URLs and the DNS listen address are host-local. The credential file is root-owned mode `0600` and is loaded at start and on `SIGHUP`. The URL and the password are not written to logs, observations, or check detail.
- Contract revision 11 adopted. A provisional lease (D-067, D-071) reports `ErrorCode` `AWAITING_MEMBER_GRANTS`.
- SECOPS-19: observation usage block populated. The report reads the open usage store at build time; a missing store or a failed read keeps the zero shape and adds no error, and a timed-out read reuses the last good status.

## 待實測

- Suspend versus `CLOCK_MONOTONIC` versus nft timeout jiffies. The lab does not step the clock (`date -s` is not isolated in this container).
- Filesystem flush order beyond `synchronous=FULL` and a directory fsync.
- Real arm64 hardware. CI cross-compiles `GOARCH=arm64` only.
- The registered probe timeout, retry count, and stable window are still 待實測. The agent runs the probe once per step with a 2 second limit. A health challenge that names a §13.9 test peer installs that exception only for the check.
- The production host rate reserve (`host_reserve_bps`). It is 待實測 and must be set by the operator. The agent no longer substitutes a stand-in.
