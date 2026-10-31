# Shared systemd installer fixture

TEST ONLY. Run on the assigned devstack prefix; this lane uses `e2e / 18000`.
All commands below target owned containers. `e2e-down` removes the installer,
its AdGuard sidecar and image, along with the rest of this prefix.

`Stack.PrepareInstallerNode(ctx)` stages `older` (sequence 10) and `newer`
(sequence 11). Both archives contain real Linux amd64/arm64 cross-builds at
the installer root and the unchanged `deploy/node/systemd` units. The shared
lab release signer signs them. No placeholder other-architecture archive or
separate installer signing key is used. Physical arm64 execution is unmeasured.

`RegisterInstallerReleases(ctx, admin)` uploads those archives/manifests and
waits for public release-list `verification_state=verified`. Its return values
include release IDs and manifest SHA-256 digests. Activate `release_support`
through `ActivatePolicy`, including every release your campaign needs; this
helper does not silently replace that policy. For custom versions/sequences,
use `BuildInstallerRelease` followed by `RegisterRelease`. The caller owns the
build directory under `.agent/tmp` and removes it after upload/staging.

Issue an installer enrollment through the admin API, then call:

```go
err = stack.StageInstallerToken(ctx, "installer-node", token)
out, err := stack.RerunInstaller(ctx, "installer-node")
// Preserve out and err independently: exit 0 alone is not a health result.
restoreErr := stack.RestoreInstallerServices(ctx, "installer-node", out.Services)
```

`RerunInstaller` and `ControlledReinstall` stop agent, watchdog, updater and
timer before sampling. They leave them stopped while the caller compares the
quiescent files. `out.Services` preserves the services active before sampling
and those started by the installer. Pass it to `RestoreInstallerServices`,
including on a wrapper error. Restoration starts only captured active services;
it does not activate an inactive updater oneshot, enable units or remove capture
barriers. A failed unit stays failed unless the installer actually started it.
`InstallerServices` exports a fresh state read for comparison. The NetBird
daemon is not stopped by sampling. Its socket is
`/run/fvpn-netbird/netbird.sock`; `/run/vpn-node` must survive consumer stops.

Hash fields serialize as `key_sha256`, `cert_sha256`, `node_json_sha256`,
`usage_db_sha256`, `usage_wal_sha256`, `machine_id_sha256`; `current_target`
is a symlink target. Missing files remain `absent`. Compare `out.Before` with
`out.Hashes` for an idempotent call. Comparing two calls separated by running
the agent can mistake legitimate WAL writes for installer changes.

## First-poll and apply capture

Before installation, `HoldInstallerFirstPoll(ctx, true)` withholds only the
authenticated desired-state GET at the installer proxy. Bootstrap still runs
and writes the real setup key required by `netbird-up`. This permits
missing-desired-state and fresh updater no-work capture before the first
successful desired-state poll. Release with `false`, then restore/start the
agent/watchdog. Holding the entire service would prevent bootstrap and cause
installation to time out; the first diagnostic attempt retains that failure.

Before `upgrade`, `HoldInstallerApply(ctx, true)` holds the automatic updater
unit. The packaged ExecStart remains unchanged. Capture the pending document
and current symlink before invoking the real updater `apply` in the foreground.
Release with `false`; this only removes this helper's drop-in and marker and
reloads systemd. It never manufactures a desired state, pending release,
checkpoint, attestation or journal result. Other runtime drop-ins are retained.

The installer-generated updater uses the kernel boot ID. The ordinary lab
entrypoint now copies `/proc/sys/kernel/random/boot_id` to its agent boot input.
Container restart therefore preserves boot ID. It proves process restart,
not OS reboot; acceptance restart checks require fresh heartbeat/readback.

## Serving member and ENR-05

`PrepareInstallerNode` starts the real AdGuard sidecar before NetBird joins,
as in the ordinary lab node. This keeps NetBird's unused DNS stub from occupying
overlay port 53 before AdGuard. After installation, call
`ConfigureInstallerServing(ctx)`. It supplies the private resolver credentials
through a pipe between the owned containers and copies the generated config to
`/etc/vpn-node/agent-lab.json` with the real resolver endpoint and TEST ONLY
5-second reporting/polling intervals. Runtime agent/watchdog drop-ins select
that file. Packaged units and generated `agent.json` are preserved. Call it again
after reenrollment or release change, then restore/start the services.

Create the DNS/support/plan policies and capacity through product APIs, publish
the installer exit, create a finite member grant, join the native member using
`JoinNativeMember`, and select that exit with the native client's `networks
select`. Wait for actual serving checks, accepted generation, device counters
and traffic. `TestE2E_Followup17Installer` supplies this composed example.
Configuration, enrollment and a successful publication request do not establish
forwarding by themselves.

```go
transport, err := stack.MemberTransportStack(memberContainer, "installer-node")
before, err := transport.CountedMemberTraffic(ctx, deviceID)
err = stack.HoldUsageAcknowledgements(ctx, "installer-node", true)
_, err = transport.StartEstablishedTransports(ctx, deviceID)
held, err := stack.UsageLedgerSnapshot(ctx, "installer-node")
err = stack.HoldUsageAcknowledgements(ctx, "installer-node", false)
err = transport.StopEstablishedTransports(ctx)
```

The hold marker is `/var/lib/vpn-node/e2e/hold-usage`. The installer proxy at
loopback 9444 checks the current verified client identity first, then withholds
only POST `usage-batches` with 503. Other routes continue upstream. Snapshots
invoke `/opt/vpn-node/current/fvpn-agent usage snapshot -config
/etc/vpn-node/agent.json -json`; that config points at the same durable store
as the serving copy. `UsageLedgerSnapshotBytes` retains exact CLI bytes.

This is same-incarnation hold/release evidence. Reenrollment retains the old
usage store but does not authorize delivery of retired-incarnation events.
The usage/installer lanes still own that protocol and the once-only central
total across revocation/reinstall. Never replay old events under the new identity.

## Complete capture exports

The origin image includes `tcpdump`; other service images do not acquire it.
Use `ExportContainerFile(ctx, slot, absoluteContainerPath, diskPath)` for large
files. It streams stdout into a mode-0600 temporary file next to the destination,
syncs and atomically renames on success, and returns byte count and SHA-256.
Failed commands preserve the previous destination. Quiesce the producer first
when a complete closed capture is required. These are private scratch files;
redact before publishing. The diagnostic `lab.docker` now errors on overflow
instead of returning a silently truncated success. The isolation receiver's
export uses the shared streaming method.

Run the focused campaign after `e2e-up E2E_RESET=1`:

```sh
FVPN_E2E_FOLLOWUP17=1 FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000 \
  go test -tags e2e -count=1 -v -timeout 30m ./tests/e2e \
  -run '^TestE2E_Followup17Installer$'
```

The journaled upgrade row remains required and fails until genuine metering
and subsequent product health succeed. A failed durable result with process
exit 0 is still a failure. No checkpoint is supplied by this fixture.
