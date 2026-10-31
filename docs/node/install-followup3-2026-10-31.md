# Installer follow-up 3 — 2026-10-31

For the devstack and handover lanes. All identities, signing keys, certificates
and release IDs in this suite are generated TEST ONLY fixtures. No production
credentials or account data are retained here.

## ENR-06 root cause and fix

`ProtectSystem=strict` made NetBird's default `/var/run/netbird.sock` unwritable.
Several services also declared `RuntimeDirectory=vpn-node` with its default
remove-on-stop lifecycle. Stopping a consumer could remove the agent heartbeat
directory while other consumers still used it.

The packaged daemon now owns `/run/fvpn-netbird` (0700) and binds
`unix:///run/fvpn-netbird/netbird.sock`. Installer up/status calls explicitly use
that address; the agent unit supplies `NB_DAEMON_ADDR` for its status probe.
Only the NetBird daemon owns/removes that directory. Its working directory is
`/var/lib/netbird`, so relative update-result cleanup stays in writable state. All consumers of the
shared `/run/vpn-node` retain it through stop/restart with
`RuntimeDirectoryPreserve=yes`; reboot clears it. The narrower socket directory
avoids granting write access to all of `/run`.

The firstboot unit no longer orders itself before the agent: it synchronously
starts the agent after enrollment, so that ordering would deadlock the job.
The generated updater start command consumes the adapter's appended release
path as a shell positional argument instead of sending it to `systemctl` as a
unit name. Neither updater unit gains write access to `/etc/vpn-node`.

## Reconciliation and offline seed

`check` reads the updater's read-only journal snapshot and compares its current
release with the live symlink and unit templates. `installed.json` remains an
install-time record, exposed as `install_record`. A stale pointer, unreadable
journal or recovery-required latch is still drift. The old manifest digest is
not presented as the new release's digest when status cannot supply one.

Initial install retains the authenticated archive under
`update-staging/<artifact-sha>.tar.zst`, mode 0600. The streaming atomic copy is
reverified before publication. This costs one compressed archive per retained
release and allows seed/reenroll seed without accessing the artifact URL.

## Validation and outstanding integration

The focused non-container reconciliation test runs the real seed/apply engine,
SQLite journal and status CLI. It injects TEST ONLY meter, service and network
health adapters; it proves the installer check logic, not product dataplane
health. It covers a successful journal switch, unchanged install history,
stale-current detection and unreadable-journal refusal.

The privileged Ubuntu 24.04 suite now uses the pinned NetBird 0.82.0 binary for
service, status and version. Only `up` is a TEST ONLY adapter that checks the
real socket instead of contacting a NetBird management server. Containers run
with `--network=none`; control is loopback-only and seed is entirely offline. The control
endpoint issues actual certificates for the agent CSR but is a test server.
Devstack must still rerun its ENR-06 wrappers with real NetBird registration.

The required fully journaled product upgrade currently fails at `final_sample`:
`state/meter.checkpoint` is absent. The generated updater configuration selects
`updater.FileMeter`, but the agent does not produce that file. No meter or
health evidence was fabricated in the container suite. Its upgrade test stays
a failing required test, separate from install/reenroll/remove/image cases.
No full verify/switch/restart/health or post-switch product pass is claimed.

The missing integration is outside this lane: `node/updater/host.go` consumes
the file; `node/agent/usage.go` closes meter epochs but does not publish it.
Code review also finds that the agent uses the install-time `agent.json`
release ID for challenge matching (`node/agent/attest.go`); that must follow the
current release before candidate health can succeed. This later health issue
was not reached in the container run. These files were not edited. The harness
also needs real signed desired state and a lab probe topology for successful
network-health evidence; its current fake control supplies enrollment only.

## Final gate evidence

At the beginning of this follow-up, `fs.inotify.max_user_instances` was 128 and
26 containers were running. A later full-suite start observed 128 / 41. No
inotify failure recurred; no host sysctl or networking was changed.

| Gate | Result |
|---|---|
| `installer-lint installer-unit installer-build` | PASS; pinned staticcheck/vet, unit tests, amd64 and arm64 builds |
| `make go-lint` | PASS; exit 0, no output |
| `gofmt -l .`, `git diff --check`, committed-evidence hygiene | PASS |
| `installer-test` | FAIL; 9 cases passed, 1 required product-upgrade case failed |

Final privileged suite output (TEST ONLY fixtures):

```text
--- PASS: TestMissingTool (1.05s)
--- PASS: TestTokenArgvRefused (1.27s)
--- PASS: TestFaultRollback (5.16s)
ENR-06: real NetBird socket bound; enrollment -> services active -> check healthy; shared runtime survives all consumer stop/start cycles
--- PASS: TestInstallLifecycle (19.84s)
--- PASS: TestRemovePurge (7.61s)
seeded image -> two firstboot services: distinct incarnations/keys/machine IDs; original seed evidence preserved; check healthy
--- PASS: TestImagePrepareTwoBoots (33.25s)
--- PASS: TestBootstrap (0.14s)
--- PASS: TestD089_MergedUpdaterSeedProbe (3.44s)
verified baseline must apply: exit=0 state=failed stage=final_sample current=019a0000-0000-7000-8000-0000000000c1 meter_checkpoint_present=false
--- FAIL: TestD089_RealSeedThenJournaledUpgrade (20.16s)
real seed refused different journal; installed tree kept; original baseline and identity unchanged; check reports baseline/current drift
--- PASS: TestD089_SeedRefusalKeepsInstalledTreeAndJournal (9.04s)
FAIL github.com/TED-H-com/Freedom-VPN/tests/installer 105.529s
```

The updater exit alone is insufficient: in this failed attempt it returned 0
while its durable result said `state: failed`, `stage: final_sample`,
`journal_stage: rolled_back`, with `health: null` and the original current
release. The suite checks those fields and deliberately fails.

Install/re-run/reenroll checks freeze the agent before hashing `usage.db` so
normal process shutdown/WAL activity cannot be mistaken for installer drift.
Seed refusal preserves the original seed evidence and key hash while retaining
the completed new installation for diagnosis. Image tests invoke the actual
firstboot systemd service, consume each token, and require a healthy `check`.

All containers and images named `fvpn-w5-node-installer-*` were removed. No
matching networks or volumes remain. There is no real arm64 execution evidence;
arm64 is cross-compiled.

Devstack: rerun the ENR-06 wrappers against this packaged unit set with the real
management server. Handover: use the current/install-record distinction and
retain the failed full-upgrade gate until agent/updater metering and health
integration is supplied. Do not count the unit test's injected health adapters
as product acceptance.
