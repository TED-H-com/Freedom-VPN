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
Only the NetBird daemon owns/removes that directory. All consumers of the
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
real socket instead of contacting a NetBird management server. The control
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
The agent also uses the install-time `agent.json` release ID for challenge
matching (`node/agent/attest.go`); that must follow the current release before
candidate health can succeed. These files were not edited.

Final gate results and exact container evidence are recorded below once the
follow-up run completes.
