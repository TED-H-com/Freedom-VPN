# ENR-06: real NetBird cannot start under the packaged systemd unit

Owner: packaging/installer and `deploy/node/systemd`.

The dedicated Ubuntu 24.04 systemd container uses the pinned NetBird 0.82.0
binary, generated signed releases and real enrollment against lab control.
The installer test image's NetBird stub is deliberately not used.

Focused run `299afccc-d98b-4bde-821e-6930bdbbfa69` enrolled incarnation
`01a19012-d25f-75c8-b452-fcefa52dfe43`, then `install --json` returned 1 while
starting `fvpn-node-netbird-up.service`. The unit journal showed:

```text
Error: listen daemon interface: listen unix /var/run/netbird.sock: bind: read-only file system
Error: failed to connect to daemon error: context deadline exceeded
heartbeat err="mkdir /run/vpn-node: read-only file system"
```

`fvpn-netbird.service` invokes the default daemon socket with
`ProtectSystem=strict`, but its write allowlist includes `/run/vpn-node`,
not `/run/netbird.sock`. The socket failure repeats on systemd restart.
The heartbeat symptom was also observed; shared `RuntimeDirectory=vpn-node`
lifecycle across the units needs review by the owner rather than an inferred
lab workaround. The unrelated missing HOME/XDG warning is not the primary
failure evidence.

The harness preserves the real exit, hashes, safe diagnostic booleans and
owner in `installer-wrappers`. Initial installation must succeed before the
successful rerun/newer-release/reenroll hash assertions can run. This is a
failed scenario, not a skipped one, and no AT07 pass is claimed. No packaged
unit or product code was edited.

The first attempt in `history/exports-first` instead used the wrong lab
listener for anonymous enrollment. The harness now adapts the installer's
single URL through a local TLS proxy: only enrollment goes to the public
listener; runtime calls require the current verified node identity and use
central's direct mTLS listener. The focused run above includes that adapter.
