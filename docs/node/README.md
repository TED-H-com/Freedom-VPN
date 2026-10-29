# Node agent and watchdog

`fvpn-agent` (`node/agent/cmd/fvpn-agent`) is the program on every exit node. It enrolls once, proves its identity with mTLS, fetches signed desired state, applies isolation and QoS through `node/enforce`, renews leases, and reports observations. `fvpn-watchdog` (`node/watchdog/cmd/fvpn-watchdog`) is a separate process. The lab driver is `fvpn-labcontrol` (`node/internal/fakecontrol/cmd/fvpn-labcontrol`), which has to live under `node/` so it can import the fake control server. If the agent stops refreshing its heartbeat, the watchdog installs the fail-closed baseline and does not open a control connection (§16.3).

The usage meter and the updater are separate programs. This tree leaves them the interfaces below. It does not sample counters, upload usage, or apply a release.

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

`agent.json` holds paths and ids. It never holds a token, a private key, or a setup key. The agent rejects unknown fields. The watchdog ignores unknown fields so both programs can share one file.

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
| `/var/lib/vpn-node/state/desired-state.json` | 0644 | agent, only after the envelope verifies |
| `/var/lib/vpn-node/update-journal/forwarding.blocked` | — | updater latch; the agent only reads it |
| `/var/lib/vpn-node/update.lock` | — | flock shared by the agent, the installer and the updater (`node/internal/nodelock`) |
| `/run/vpn-node/heartbeat` | 0644 | agent; the watchdog only reads it |
| `/etc/vpn-node/agent.json` | — | installer; no secrets |

`usage.db` and the updater journal database are not created here. Every path above is configurable in `agent.json`. Tests use a temp directory.

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

On a verification or render failure the agent reports the reason and keeps member forwarding off. `ISOLATION_BASELINE_MISSING` is an enforce reason and is not an `ErrorCode`. The observation check uses `HEALTH_CHECK_FAILED` and the message is the literal `ISOLATION_BASELINE_MISSING`. `METER_PATH_INVALID` is a real error code. `netbird_backend` must be `kernel`; `userspace` and `unknown` keep forwarding off.

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
- `netbird_backend` is `kernel` and `Barrier.Health` succeeded
- the isolation baseline is present
- desired state is verified, in force, `published`, and names this `release_id`
- this process holds a fresh lease whose monotonic send time is set
- capacity is non-null and QoS renders
- if any device binding is present, at least one is admitted for at least one second

Data-plane health is reported as `unknown` and is not a gate. The tunnel probe check is `not_run`. IPv6 is always blocked. A null capacity blocks forwarding. An unknown local release blocks forwarding. `test_exempt_v4` is applied only when `test_mode` is true; a non-empty exempt list with `test_mode` false is a startup error. Host reserve `0` becomes the TEST ONLY stand-in of 1_000_000 bit/s until a measured reserve exists (待實測).

## Observations

Heartbeats default to 10 seconds. Observation reports default to 30 seconds (§16.5). Reports carry the applied generation, check results, control and data health, the running release, and failure reasons. They are queued in the local outbox and resent until the center accepts them. HTTP 202 means accepted, not verified. 409 and 422 are conflicts and are not retried. 429 and 5xx stay pending.

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

The watchdog reads the heartbeat, the monotonic clock, and whether table `inet fvpn` exists. It applies `enforce.ApplyBaseline` when the heartbeat is missing, older than `watchdog_stale_seconds` (default 30), from another boot, or when the baseline is gone or the policy deadline has passed. A healthy reading does not render serving rules and does not restart a kernel timeout. Its own restart therefore does not restart a deadline. If both processes die, nft element timeouts still expire members. The watchdog does not take the node lock; fail-closed outranks a busy flock.

## Interfaces for the other node tasks

Meter (`node/agent.Barrier`). The usage task implements this. A nil barrier is `NopBarrier`, whose `Health` returns true. `Health` false or an error sets `MeterFailed` and keeps forwarding off (`METER_PATH_INVALID`). The agent does not read the forward-hook counters for billing. Usage is defined as those counters (D-019); the meter task owns sampling. `FinalSample` and `OpenEpoch` stay on the meter side and must be idempotent while forwarding is blocked.

```go
type Barrier interface {
    Health(ctx context.Context) (ok bool, err error)
}
```

Updater:

- Lock: `node/internal/nodelock`, default `/var/lib/vpn-node/update.lock`. `Acquire` returns `ErrBusy` and does not wait forever.
- Latch: `forwarding.blocked` in the journal directory. `agent.ForwardingBlocked(journalDir)` is true while the file exists.
- Recover: the agent runs `updater_recover recover -config updater_config` once at boot (empty command skips it). Exit 0 clears the block. Exit 2, or any other failure, keeps members blocked. The agent does not import the updater package.
- Desired state: read the verified envelope at `desired_state_path` (default `/var/lib/vpn-node/state/desired-state.json`). The agent is the only writer, and only after verification.
- Health: stock updater checks do not know isolation or the tunnel. The agent reports those on the observation, not through the updater.

## Config example

```json
{
  "state_dir": "/var/lib/vpn-node/state",
  "identity_dir": "/var/lib/vpn-node/identity",
  "journal_dir": "/var/lib/vpn-node/update-journal",
  "lock_path": "/var/lib/vpn-node/update.lock",
  "control_url": "https://control.example.invalid",
  "server_ca_file": "/etc/vpn-node/ca.pem",
  "trust_bundle": "/etc/vpn-node/trust.json",
  "heartbeat_path": "/run/vpn-node/heartbeat",
  "datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
  "agent_release": "0.0.0-test",
  "release_id": "019a0000-0000-7000-8000-0000000000c1",
  "netbird_backend": "kernel",
  "updater_recover": "/opt/vpn-node/current/fvpn-updater",
  "updater_config": "/etc/vpn-node/updater.json"
}
```

`agent_release` must be a pinned version. `latest` is rejected. Prices, fair-use thresholds and the host reserve above are not in this file. The host reserve field, when set, is an operator override; leaving it at 0 selects the TEST ONLY stand-in.

## Decisions

- Trusted time is `CLOCK_MONOTONIC` plus the signed server time captured when the lease request is sent. `CLOCK_BOOTTIME` is not used.
- A process restart fail-closes until a lease response in that process. The same `lease_id` does not reset `mono_sent`. After reboot, only a strictly newer `issued_at` can enable forwarding.
- A 304 skips desired-state acceptance only. The step still renews a due lease and still enforces.
- Capacity null, an unknown release, and a non-kernel NetBird backend block forwarding.
- Data health `unknown` is not a forwarding gate.
- Lab exempt prefixes require `test_mode`.
- A bootstrap secret lost after HTTP 200 and before the 0600 write is unrecoverable.
- `ISOLATION_BASELINE_MISSING` is carried in `NodeError.Message` with code `HEALTH_CHECK_FAILED`.

## 待實測

- Suspend versus `CLOCK_MONOTONIC` versus nft timeout jiffies. The lab does not step the clock (`date -s` is not isolated in this container).
- Filesystem flush order beyond `synchronous=FULL` and a directory fsync.
- Real arm64 hardware. CI cross-compiles `GOARCH=arm64` only.
- The tunnel probe. The check is `not_run`.
- The production host rate reserve. The 1_000_000 bit/s stand-in is TEST ONLY.
