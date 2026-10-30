# Node health attestation

This is the file contract between `fvpn-updater` and the candidate `fvpn-agent` for the §13.9 isolation, DNS and tunnel checks (D-053). The Go types, the atomic writers and `Validate` live in `node/internal/healthattest`. That package is standard library plus `pkg/api`. The agent imports it. The updater owns issuance and acceptance. This document is the producer contract for the agent.

A passing attestation sets `last_good` and can reach `locally_succeeded`. It does not publish the node and it does not open member forwarding. Serving stays the latest publication intent and the serving policy (§13.9 服務狀態恢復). The updater half is in [`node-updater.md`](node-updater.md).

## Files

Both files live in the updater journal directory, the same directory the agent already reads for `forwarding.blocked` (`/var/lib/vpn-node/update-journal` by default).

| File | Writer | Constant |
|---|---|---|
| `health-challenge.json` | updater | `healthattest.ChallengeFile` |
| `health-attest.json` | candidate agent | `healthattest.AttestationFile` |

Write them with `healthattest.WriteChallenge` and `healthattest.WriteAttestation`. Those helpers create a temporary file in the same directory, set mode `0600`, `fsync`, rename over the destination, and `fsync` the directory. One challenge is live at a time. A new check overwrites it. After health is recorded the updater removes both files. If the remove fails, it rewrites the challenge with `"consumed": true`. `Validate` rejects a consumed challenge.

The journal directory must already exist, must not be a symlink, must have no group/other permission bits, and must belong to the effective user where the platform exposes ownership. Updater setup creates it as `0700` and rejects unsafe existing directories without changing their permissions.

A reader refuses a symlink, any file that is not a regular file, any mode other than `0600`, a file larger than 64KiB, unknown JSON fields, and trailing data after the JSON value. Readers verify the opened file is the same inode as the checked path and bound every read to 64KiB plus one byte. The consumer hashes and decodes the same bytes when checking freshness, so replacement between separate observation and decoding cannot bypass the check.

## Challenge

The updater fills this before the check-stage `Services.Start`. Times are UTC RFC 3339. Ids are lowercase canonical UUIDs (`8-4-4-4-12`). `nonce` is 64 lowercase hex characters, 256 random bits, new for every issuance.

```json
{
  "txn_id": "018f5b2a-6c3e-7a10-8d11-111111111111",
  "nonce": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "release_id": "018f5b2a-6c3e-7a10-8d11-222222222222",
  "node_id": "018f5b2a-6c3e-7a10-8d11-333333333333",
  "incarnation_id": "018f5b2a-6c3e-7a10-8d11-444444444444",
  "boot_id": "018f5b2a-6c3e-7a10-8d11-555555555555",
  "issued_at": "2026-10-28T15:00:00Z",
  "expires_at": "2026-10-28T15:00:45Z",
  "components": {
    "node_agent": "1.2.0",
    "netbird": "0.76.2",
    "node_updater": "1.2.0",
    "enforcement": "1.2.0"
  }
}
```

`components` are the versions the manifest of `release_id` declares. `node_agent` and `netbird` are what the attestation must echo as the running versions. `node_updater` and `enforcement` are there so the agent can see the declared versions. During `checking`, the process running the updater is still the previous release's binary, or the supervisor binary. The updater does not compare that process with `components.node_updater`. It checks the candidate tree hash instead. `enforcement` has no attested running version; the tree hash covers it too.

### Optional synthetic probe (D-063)

When updater `probe_peer` is configured, `Issue` copies it into the challenge's optional `probe` member. Without configuration, `probe` is omitted. The request is a dedicated NetBird test peer, not a member identity:

| Member | Issuer limit |
|---|---|
| `peer_id` | Required nonblank opaque NetBird peer ID, at most 128 bytes, no ASCII control characters. |
| `source_v4` | One usable IPv4 `/32`, the dedicated peer's overlay address. No bare address, IPv6, wider prefix, unspecified, loopback, link-local, multicast or reserved source. |
| `destinations` | Required array of 1–4 exact destination sockets, even when resolver DNS is requested. |
| `destinations[].addr_v4` | One public IPv4 `/32`, outside `enforce.SpecDeniedV4()`; no RFC1918, CGNAT/overlay (`100.64.0.0/10`), loopback, link-local/metadata (`169.254.169.254` included), documentation, benchmark, multicast, reserved or other specification-denied ranges. |
| `destinations[].proto`, `.port` | Exactly `tcp` or `udp`, with port 1–65535. |
| `resolver_dns` | Optional boolean, false by default. Requests TCP/UDP port 53 to the agent's configured resolvers; it supplies no arbitrary resolver address. |

If a configured probe fails issuance validation, the challenge instead carries optional `probe_omitted` with its bounded `peer_id` and fixed refusal `reason`. It cannot coexist with `probe`; absence of both means no probe was configured. `Issue` copies the omission and `Same` compares both fields. The updater commits it to the journal before publishing the challenge and includes it in local health diagnostics; see [recording and consumer compatibility](node-updater.md#synthetic-health-probe-configuration-d-063). The shared reader rejects unknown fields, so consumers must be rebuilt with this additive member. The agent owner must carry its fixed reason into the tunnel check's observation `detail`; `UpdateHealth` has no detail field.

The updater sets `issued_at = now` and `expires_at = issued_at + HealthChallengeTTL` (45s). It never backdates issuance to bypass the consumer's strict `now.After(issued_at)` condition. The agent may install only after issuance and before expiry; it must reject a challenge span over 300s and cap exception expiry at both `expires_at` and `issued_at + 300s`, never installation time plus 300s. The shared issuer also refuses a probe-bearing input with TTL over 300s. Times, nonce, probe fields and destination order survive the file round trip. `Same` compares the entire probe as well as the challenge binding, and issuance copies the probe so later configuration mutation cannot change that binding. A `tunnel_probe: passed` answer still requires the exact nonce. Consumption rejects the challenge even when its probe is otherwise valid.

Ownership: the updater validates the static configuration and the current node, control and resolver address exclusions at load and each candidate/rollback issuance. If revalidation refuses the probe, the fresh challenge omits `probe` and logs a fixed refusal class once per updater process; see [issuer inputs and diagnostics](node-updater.md#synthetic-health-probe-configuration-d-063). The agent authenticates the dedicated peer against its runtime state, rejects member peer IDs and source/destination addresses matching member bindings, rejects node/host or additionally denied ranges, bounds the grant, and installs/removes the nftables exception (D-063, §13.9). The updater has no live member inventory and does not install networking rules or infer that issuance means installation. Member forwarding stays blocked during the check; publication intent is unchanged. The grant and probe counters are separate from member traffic and usage. A missing or refused grant cannot itself establish forwarding health.

Privacy: probe configuration values belong in protected configuration and the mode-0600 challenge. The omission audit additionally permits the dedicated test peer ID and a fixed reason in protected journal metadata and health diagnostics. Source/destination addresses and lists never enter that audit. Logs and wire check details contain only the fixed reason, never the peer ID. Issuer validation reports the fixed `updater: invalid probe_peer`; malformed configuration JSON reports `updater: invalid config JSON`. Neither includes a peer ID, source, destination list, protocol value or rejected JSON value. Existing attestation reason vocabulary still applies.

## Attestation

The agent writes one answer for the challenge it just read. Echo the challenge ids and the nonce. `node_id` is required so the updater can compare node ids. `published` is the agent's latest publication intent, one of `published`, `unpublished`, `disabled`. It is a report. The updater does not open the gate from it.

```json
{
  "txn_id": "018f5b2a-6c3e-7a10-8d11-111111111111",
  "nonce": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "release_id": "018f5b2a-6c3e-7a10-8d11-222222222222",
  "node_id": "018f5b2a-6c3e-7a10-8d11-333333333333",
  "agent_version": "1.2.0",
  "netbird_version": "0.76.2",
  "boot_id": "018f5b2a-6c3e-7a10-8d11-555555555555",
  "incarnation_id": "018f5b2a-6c3e-7a10-8d11-444444444444",
  "generated_at": "2026-10-28T15:00:01Z",
  "published": "unpublished",
  "checks": {
    "internet_only": {"status": "passed", "reason": "baseline held"},
    "dns_policy": {"status": "passed", "reason": "policy loaded"},
    "tunnel_probe": {"status": "passed", "reason": "probe reached the lab endpoint"}
  }
}
```

Each status is `passed`, `failed` or `not_run`. Reasons use a fixed vocabulary and remain bounded to 160 bytes. Unknown strings are rejected as `bad_reason` before any attestation detail reaches the updater journal (SEC02). The accepted values are:

- Empty string, `baseline held`, `baseline missing`.
- `policy loaded`, `policy in force`, `policy not in force`.
- `not run`, `probe not run`, `probe skipped`, `probe refused`, `probe failed`.
- `overlay up`, `overlay_up`, `overlay_down`, `overlay_partial`.
- `probe reached the lab endpoint` (the original contract example).

The real producer's fixed reasons are included. A new producer reason requires an explicit consumer vocabulary update; an arbitrary lowercase token is not sufficient.

`generated_at` is strictly after `issued_at` and strictly before `expires_at`. The updater's clock at validation must also still be before `expires_at`. A timestamp more than `MaxClockSkew` (2s) ahead of the updater's clock is rejected as `generated_in_future`; this allowance never relaxes the issue/expiry bounds.

## Ordering

1. The switch stage may call `Services.Start` before any challenge exists. If `health-challenge.json` is absent, the agent keeps running and polls. It does not write an attestation for a challenge it has not read, and it does not treat a missing challenge as a failed process start.
2. The check stage, after the 8-attempt cap and before its `Services.Start`, writes `health-challenge.json` (overwrite). It snapshots `health-attest.json` first.
3. The check stage calls `Start` again. That call is the signal that the challenge is live. `Start` is idempotent.
4. The agent reads the challenge, runs the checks below, and writes `health-attest.json` with an atomic rename.
5. The updater polls until the wait window. It records health, appends a `txn_events` detail (`kind` `health_attest`, `phase` `candidate`), and removes both files.
6. On a destructive rollback the updater stops the candidate first, writes a new challenge whose `release_id` is the restored release, then starts that release. The restored agent answers that challenge, with that release's component versions. The updater appends a second detail with `phase` `rollback`. D-062 retains the candidate profile in `candidate_health_json`; the latest rollback profile becomes `health_json` and the rolled-back result's `health`. Contract 1.2.0 exposes the stored candidate profile as `candidate_health` on rolled-back results, or null when no candidate check ran. Other result states omit that member. `fvpn-updater status` shows both the stored diagnostic field and the rollback wire member. The result names the restored release as current and the candidate as previous, while preserving the candidate failure code. Revocation recovery also stops the revoked service before switching, then issues a fresh recovery transaction UUID and rollback challenge before starting the retained release. On health failure it stops that target, retains the selected tree as `current`, leaves `last_good` unchanged, and reports the revoked release as `previous_release_id`.

A resume of `checking` issues a new nonce. An answer that still carries the previous nonce fails `nonce_mismatch`.

## Timing

The defaults are code constants, pending measurement (待實測), not a verified SLO. The updater config cannot change them.

| Name | Default |
|---|---|
| `HealthAttestWait` | 30s per check attempt |
| `HealthAttestPoll` | 250ms |
| `HealthChallengeTTL` | 45s (`expires_at = issued_at + TTL`) |

The 8-attempt cap and the 3-rollback cap are unchanged. The attempt that crosses the cap does not issue a candidate challenge. Each attempt that does run uses a new nonce and its own wait window.

A missing file, a wrong nonce, a wrong release, a stale boot id, a version mismatch, or an expired `generated_at` does not end the wait by itself. The updater keeps the last reason and continues until the deadline, so one corrected atomic rewrite inside the window can still pass. Only a valid answer with all three checks `passed` ends the wait early. Valid `failed` and `not_run` answers can be replaced during warm-up. At the deadline the consumer returns the latest valid answer, even if a later read was missing or malformed; without a valid answer it returns the last rejection reason. The deadline uses the injected updater clock, with a context timeout to bound a stalled clock.

## What the agent checks before it answers

Run all three for an unpublished node and for a disabled node. Member forwarding stays closed while they run. The probe uses the dedicated probe identity. The probe does not change `publication_intent` and does not put the exit into service (§13.9).

| Check | Before answering `passed` |
|---|---|
| `internet_only` | The outer Internet-only enforcement, source identification, member expiry and cross-member blocking hold. A failure keeps the block. |
| `dns_policy` | The configured DNS and the security policy are usable. |
| `tunnel_probe` | An authenticated synthetic probe, using the dedicated probe peer that is not in a member group, crosses the real tunnel and the candidate forwarding path and reaches the team-controlled Internet test endpoint. The outer rules allow that identity only, for the exact source, destination, protocol and port, plus the required DNS. Comparing a claimed source IP is not this check. A curl from the host is not this check. Probe bytes stay out of member usage. Remove the short-lived probe grant when the check finishes or the attempt times out. |

`agent_version` is the agent binary this process is running. `netbird_version` is the NetBird client version this process observes. Both must equal the challenge `components` for the release under test. On a rollback challenge those components belong to the restored release.

`not_run` is a real status and the updater maps it to `failed`. The probe is mandatory, including when the node is unpublished or disabled. Send `not_run` only when the check truly did not run, and expect the attempt to fail. A later passing answer within the wait window can still complete the attempt; a final `not_run` fails closed.

## What the updater accepts

`healthattest.Validate` returns an empty reason when the binding holds. It returns `not_run` when the binding holds and any of the three statuses is `not_run`. Every other reason means the file is not evidence: the three checks fail, and `processes` fails because the versions were not attested. `identity_and_state` is decided from the identity directory and the immutable pointers record, independent of this file.

Acceptance requires all of the following:

- The on-disk challenge is the one the updater just wrote (`Same`), and it is not `consumed`.
- `txn_id`, `nonce`, `release_id`, `node_id`, `incarnation_id` and `boot_id` match that challenge. Ids compare case-insensitively and must be UUID-shaped. The nonce matches exactly, lowercase hex.
- `generated_at` is after `issued_at` and before `expires_at`, and the updater's clock is before `expires_at`; it is no more than 2s ahead of that clock.
- The attestation path was rewritten after the pre-challenge snapshot. A new inode, a later modification time, a different size, or a different SHA-256 counts. An equal modification time does not. On the development host two immediate writes can share one timestamp, so the agent must rename a new file into place (`WriteAttestation` does this). An in-place update that keeps the inode, the timestamp and the bytes is invisible.
- `agent_version` equals `components.node_agent` and `netbird_version` equals `components.netbird`. An empty manifest version fails closed.
- `published` is a known publication intent.
- Each status is a known `HealthCheckStatus`, and each reason is within 160 bytes and belongs to the fixed vocabulary above.

A file that was already at `health-attest.json` when the challenge was written is `attestation_precedes_challenge`. The updater journals that and ignores the file until a rewrite. The refusal reasons are the `healthattest.Reason*` constants: `txn_mismatch`, `nonce_mismatch`, `release_mismatch`, `node_mismatch`, `incarnation_mismatch`, `boot_mismatch`, `generated_before_issue`, `generated_in_future`, `expired`, `agent_version_mismatch`, `netbird_version_mismatch`, `unknown_status`, `not_run`, `bad_publication`, `bad_reason`, `malformed`, `consumed`, `bad_mode`, `bad_owner`, `symlink`, `too_large`, `attestation_precedes_challenge`, `attestation_missing`, `challenge_replaced`, `manifest_unavailable`, `unreadable`.

## Negative cases

The agent must not bypass a failed or missing check.

- Do not add a config field, an environment variable, or a pre-written file that marks `internet_only`, `dns_policy` or `tunnel_probe` passed. The updater does not read `FVPN_HEALTH_CHECKS_PASSED`, and `LoadConfig` rejects unknown fields such as `health_checks`.
- Do not answer with `not_run` and expect a pass. Do not skip the probe because the node is unpublished or disabled.
- Do not open member forwarding because health passed, because `published` in the attestation is `published`, or because the job intent is `published` while the release is outside the serving set. The updater's gate stays in charge. While the probe runs, member forwarding stays closed.
- Do not publish an unpublished or disabled exit so the probe has a path. Use the probe identity only. Do not widen that exception to a whole source range, an arbitrary destination, or an admin port.
- Do not reuse a nonce. Do not report a previous boot id or a previous release id while answering the challenge that is on disk. A rollback challenge names the restored release; echoing the failed candidate's release id is `release_mismatch`.
- Do not leave the previous transaction's `health-attest.json` in place. A file present before the new challenge is ignored.
- Do not write the attestation in place when that preserves the inode, the modification time and the bytes. Rename a new `0600` file over `health-attest.json`.
- Do not report an agent or NetBird version other than the one this process is running. A mismatch fails the attempt, including `processes`.
