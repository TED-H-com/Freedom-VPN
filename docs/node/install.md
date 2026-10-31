# Ubuntu node installer

`fvpn-node-install` is the host installer for one Freedom VPN exit node (§6.3, PKG01, §18.2). The supported baseline is Ubuntu 24.04 LTS on amd64 or arm64. Any other OS, version, or architecture is refused with exit 4 and `UNSUPPORTED_PLATFORM`.

The binary lives in the signed release archive next to `fvpn-agent`, `fvpn-updater`, and `fvpn-watchdog`. `deploy/node/install.sh` is only a download bootstrap. It does not enroll a node, install packages, or keep a secret.

## Verified download (§18.2, PKG01)

The release pipeline replaces `@@INSTALLER_SHA256@@` and `@@TRUST_ROOT@@` in `deploy/node/install.sh` before it publishes the script. A placeholder pin or a missing trust root is refused before any download.

```sh
install.sh \
  --manifest-url https://releases.example/r/<release_id>/manifest.json \
  --artifact-url https://releases.example/r/<release_id>/ubuntu-24.04-amd64.tar.zst \
  --out /var/tmp/fvpn-release \
  -- install --manifest /var/tmp/fvpn-release/manifest.json \
       --artifact /var/tmp/fvpn-release/artifact.tar.zst \
       --trust /var/tmp/fvpn-release/trust.json \
       ...
```

The script allows `https` URLs, and `http` only to `127.0.0.1` or `localhost` for a lab. It refuses a path segment of `latest`, a query string, a fragment, and a redirect (`curl` is not given `--location`). It downloads the named manifest envelope and the archive, extracts `fvpn-node-install`, compares that file to the pinned SHA-256, runs `fvpn-node-install verify-archive`, and execs the binary with the operator's arguments. Signature checks stay in `packaging/release`. The script does not invent install flags: repeat the `--out` paths on the `install` command.

`release_id` is a UUID. The installer refuses a `latest` tag in the path, the product version, or a component version (§13.4, §13.9).

## Token intake (§18.2, ENR03)

The enrollment token is never a command-line argument. `--token` and `--token=...` are refused with exit 64 before flag parsing, so the value does not need to be passed in a way that lands in shell history.

Pass `--token-file <path>` or omit it and type the token at the interactive prompt (the prompt reads `/dev/tty` and does not echo). The file must be a regular file, mode exactly `0600`, and not a symlink. After `fvpn-agent enroll` succeeds, the installer shreds the file (overwrite, sync, remove). A failed enrollment leaves the operator's file in place. `FVPN_ENROLLMENT_TOKEN` is stripped from child environments and is not read.

On a re-run of a node that already has identity, a missing token file is ignored. The first successful install has already shredded it.

## What install changes (§6.3)

`install` records each host change in `/var/lib/vpn-node/install/journal.json` before it makes the change. The record stores the action and the previous value. On failure it restores only that scope.

Order:

1. Identify Ubuntu 24.04 and amd64 or arm64.
2. Require `nft`, `tc`, `conntrack`, `ip`, and `systemctl`. Probe WireGuard with `ip link add fvwprobe type wireguard` and delete that link. The kernel minimum is 待實測; the probe does not invent one and does not load a module.
3. Require systemd as PID 1 (`/run/systemd/system` or `/proc/1/comm`).
4. Verify the signed manifest, the per-artifact SHA-256, and `release_id` through `packaging/release`.
5. If `netbird` is absent, install `netbird=0.82.0` from `https://pkgs.netbird.io/debian` (`stable`, signed by that repository's public key) and record the package. If a different version is already installed, refuse with `NETBIRD_VERSION`. The upstream `install.sh` is not run.
6. Take `/var/lib/vpn-node/update.lock` (LOCK_EX|LOCK_NB). A held lock exits 3 `busy` and changes nothing further (§13.5).
7. Create the directories below, extract the archive to `/opt/vpn-node/releases/<release_id>`, point `current` at `releases/<release_id>`, install the units, and write config.
8. Create `inet` table `fvpn` only when it is absent: an owner chain (comment `managed-by=fvpn-node-install`, no hook) and a `forward` chain with policy drop (AT13). Create an empty netns named `fvpn`. Create dummy links `fvmb0` and `fvup0` only when those names are absent. No addresses and no routes are added.
9. Run `fvpn-agent enroll` when `/var/lib/vpn-node/identity/node.json` is absent.
10. Release the lock, then start units in this order: `fvpn-updater-recover`, `fvpn-watchdog`, `fvpn-agent`, `fvpn-netbird`, `fvpn-node-netbird-up`, `fvpn-updater.timer`.

`host_reserve_bps` is required and has no default (待實測, §8). `--snat` is a non-loopback IPv4 address. Member and uplink names must match `^fv[a-z0-9_-]{1,13}$` and must differ. Defaults are `fvmb0` and `fvup0`.

`--desired-trust` is the node trust store for desired state. When it is omitted, the installer copies the release trust to `/etc/vpn-node/desired-trust.json`. That file loads as a node trust store. It verifies a desired state only when the desired state was signed by a key in that store.

The empty netns `fvpn` is a name reservation. `node/enforce` programs table `inet fvpn` in the host network namespace, so the data-plane interfaces stay there.

### Re-run (§6.3, AT07)

A second `install` of the same `release_id` and the same manifest digest, with matching units, directory modes, and `agent.json`, prints `already_current` and exits 0. It does not restart units and does not touch identity, `usage.db`, the update journal, or last-good state.

The same `release_id` with a different manifest digest is `IMMUTABLE` (exit 1). An equal or lower `release_sequence` with a different id is `DOWNGRADE` (exit 1). A higher sequence replaces binaries and units, moves `current`, and does not enroll again. A same-release run that finds a missing unit or a drifted directory mode prints `repaired`, restores those files and modes, and leaves identity bytes in place.

## check

`check` is read-only. It does not take the lock. `--json` prints one JSON object; without it, the same fields are text. Exit 0 is `healthy`. Exit 2 is `drifted`, with a single line and no stack trace.

It reports OS and arch, the installed `release_id` and manifest digest, unit states, directory modes, the lock holder when the recorded pid is alive, whether identity exists and which incarnation `node.json` carries, whether `usage.db` exists (never its contents), the NetBird client version and whether `netbird status` contains `Management: Connected`, and the drift list.

A management disconnect is reported and is not, by itself, drift. The updater oneshot is inactive between timer fires; `failed` is drift. `fvpn-node-firstboot` is not required to be active after enrollment.

## upgrade

`fvpn-updater` has no local-apply command (`run`, `recover`, `flush`, `status`, `version` only). `upgrade` does not fork that state machine.

It verifies the archive, extracts it under `/opt/vpn-node/releases/<release_id>`, copies the archive to `/var/lib/vpn-node/update-staging/<sha256>.tar.zst`, and writes `/var/lib/vpn-node/update-staging/pending-release.json` (mode 0600). It does not move `current`. It releases the lock, then `systemctl start fvpn-updater.service`.

The running updater does not read `pending-release.json`. `fvpn-updater run` applies a desired-state file. On a node with no desired state, that run fails, `GuardSource` may write the forwarding latch, and `upgrade` still exits 1 with status `staged` and `updater_failed`. `current` stays where it was. A held lock exits 3 `busy` before staging.

To move `current` without the updater, run `install` again with the newer archive. That path replaces binaries and units and does not re-enroll.

## remove

`remove` stops and disables the units, deletes the unit files, and deletes only the nft table, netns, and dummy links listed in `/var/lib/vpn-node/install/owned.json`. A table this installer did not create is left in place, including a pre-existing `inet fvpn`.

Without `--purge`, identity, `usage.db`, and `/var/lib/vpn-node/update-journal` stay. The message says they were kept. `--purge` deletes those trees and says `purge:`. Retired identity under `identity.retired/` is removed only with `--purge`.

## reenroll (§6.2, AT07)

```sh
fvpn-node-install reenroll --json --token-file /run/fvpn-token
```

`--token-file` is required. The command stops the units, moves the identity files and the agent store (`agent.db`, its WAL and SHM, and `desired-state.json`) to `/var/lib/vpn-node/identity.retired/<utc>/`, sets that directory to mode 0700, and does not delete it. `fvpn-agent enroll` is a no-op while `agent.db` still records an identity, so those files have to move with the certificate. `usage.db` is not moved.

It then enrolls with the new token, shreds the token file, refreshes `updater.json` with the new node id and incarnation, and starts the units again.

`usagestore.Open` returns `ErrIdentity` when the file's node or incarnation does not match the process (§10.6). `usageupload` resends the stored canonical bytes and has no re-home path. After reenroll the new agent logs `usage store` and does not upload the old incarnation's outbox. Those rows stay in `usage.db` until an operator or a later agent change deals with them. Central rejects the old certificate (ENR-02, §6.2). This installer does not test that rejection.

## image-prepare (§6.3, ENR03)

`image-prepare` refuses while `node.json` or `key.pem` exists unless `--i-am-building-an-image` is set (`identity_present`, exit 1). With the flag it deletes generated secrets: agent key, certificate, CSR, CA, setup key, and `node.json`; `agent.db` and `desired-state.json`; `usage.db` and its WAL; NetBird state under `/var/lib/netbird` and `/etc/netbird`; SSH host keys `ssh_host_*`. It truncates `/etc/machine-id` (and `/var/lib/dbus/machine-id` when that path is a regular file) so the id is regenerated on first boot. It does not delete binaries or unit files.

Member units are disabled so a clone does not start `fvpn-node-netbird-up` before a setup key exists. `fvpn-node-firstboot.service` stays enabled. It runs when `/var/lib/vpn-node/firstboot.token` exists and `node.json` does not. Place that token, mode 0600, before the first boot. The unit calls `fvpn-node-install firstboot`, which enrolls, enables the member units, and starts them. The token is not stored in the image.

## Directories

| Path | Mode | Role |
|---|---|---|
| `/etc/vpn-node` | 0755 | `agent.json`, `updater.json`, CA, trust bundles, `installed.json` |
| `/var/lib/vpn-node` | 0700 | durable node state |
| `/var/lib/vpn-node/identity` | 0700 | key, certificate, `node.json`, `setup.key` |
| `/var/lib/vpn-node/identity.retired/<utc>` | 0700 | previous identity after `reenroll` |
| `/var/lib/vpn-node/state` | 0700 | `agent.db`, `usage.db`, desired state |
| `/var/lib/vpn-node/update-journal` | 0700 | updater journal |
| `/var/lib/vpn-node/update-staging` | 0700 | staged archive and `pending-release.json` |
| `/var/lib/vpn-node/install` | 0700 | installer journal and `owned.json` |
| `/var/lib/vpn-node/firstboot.token` | 0600 | first-boot token; not part of the image |
| `/opt/vpn-node/releases/<release_id>` | 0755 | one extracted release |
| `/opt/vpn-node/current` | link | relative link `releases/<release_id>` |
| `/run/vpn-node` | 0755 | runtime directory (`RuntimeDirectory=vpn-node`) |
| `/var/lib/vpn-node/update.lock` | 0600 | exclusive flock (§13.5) |

## Units

Templates are `deploy/node/systemd/`. They are ordered after `network-online.target`. Long-running units use `Restart=on-failure`, `RestartSec=2s`, `RestartMaxDelaySec=30s`, `RestartSteps=5`, and start-limit `120s` / burst 5 in the `[Unit]` section. Hardening is `NoNewPrivileges=yes`, `ProtectSystem=strict`, `ProtectHome=yes`, `PrivateTmp=yes`, `UMask=0077`.

| Unit | Capabilities | Why |
|---|---|---|
| `fvpn-agent.service` | `CAP_NET_ADMIN`, `CAP_NET_RAW` (ambient too) | nftables, tc, ip, and raw sockets / conntrack. After NetBird and update recovery. Not `Requires=` on recovery. |
| `fvpn-watchdog.service` | `CAP_NET_ADMIN` only | nftables and tc baseline. No `PartOf`, `BindsTo`, or `Requires` on the agent (§17.1, "Agent 卡住或退出"). |
| `fvpn-updater.service` | none | Writes files and calls `systemctl`. `Type=oneshot`, `SuccessExitStatus=0 2`. `ExecStartPre` is `refresh-boot-id`. |
| `fvpn-updater.timer` | none | `OnBootSec=2min`, `OnUnitActiveSec=5min`. The interval is 待實測 (§13.6). |
| `fvpn-updater-recover.service` | none | `fvpn-updater recover` before member services. Exit 2 means members stay blocked and is success (§13.5). |
| `fvpn-netbird.service` | `CAP_NET_ADMIN`, `CAP_NET_RAW` | WireGuard device. `PrivateDevices=no` so `/dev/net/tun` stays visible. No setup key in the unit. |
| `fvpn-node-netbird-up.service` | `CAP_NET_ADMIN`, `CAP_NET_RAW` | `netbird up --setup-key-file` plus `--disable-firewall`. The key is a path, never an argument value. |
| `fvpn-node-firstboot.service` | none | One-shot enroll on a prepared image. |

`refresh-boot-id` rewrites `boot_id` in `updater.json` from `/proc/sys/kernel/random/boot_id`. The updater pins that UUID in config and does not read a boot-id file, so the units list `/etc/vpn-node` in `ReadWritePaths`. Otherwise `ProtectSystem=strict` would make the rewrite fail.

The watchdog is a separate service. Stopping or crashing the agent does not stop the watchdog.

## Exit codes

| Exit | Name | When |
|---|---|---|
| 0 | ok | `installed`, `already_current`, `repaired`, `healthy`, `staged` when the updater run itself succeeded, `removed`, `reenrolled`, `prepared`, `enrolled` |
| 1 | failed | rollback, enrollment failure, `DOWNGRADE`, `IMMUTABLE`, `identity_present`, `updater_failed` |
| 2 | drifted | `check` found drift |
| 3 | busy | `/var/lib/vpn-node/update.lock` is held |
| 4 | platform | `UNSUPPORTED_PLATFORM` |
| 5 | dependency | `DEPENDENCY_UNAVAILABLE` or `NETBIRD_VERSION` |
| 64 | usage | flags, a token on argv, a token file that is not mode 0600 |

`upgrade` can print status `staged` and still exit 1 when `fvpn-updater run` fails. The JSON status is the staging result. The exit status is the updater result.

## Host-safety checks (§6.3)

These sentences are what a test or an operator can check.

- The installer never runs `nft flush ruleset`.
- It adds `inet fvpn` only when `nft list table inet fvpn` fails, and it deletes that table only when `owned.json` says this installer created it.
- It never deletes an nft table it did not create. A foreign table survives install, rollback, and remove.
- It never adds or replaces the host default route.
- It never writes `/etc/resolv.conf` and never restarts systemd-resolved.
- Rollback removes directories and files this attempt created, and restores files it replaced, and leaves every other path alone. `/var/lib/vpn-node` and `/var/lib/vpn-node/install/` stay so the journal remains.
- A WireGuard probe link named `fvwprobe` does not remain after a successful probe or a failed install.
- Dummy links it creates have no addresses and no routes.
- The enrollment token is not accepted on the command line.
- `--token-file` must be mode 0600, and it is shredded only after enrollment succeeds.
- `remove` without `--purge` does not delete identity, `usage.db`, or the update journal.
- `reenroll` moves identity aside and does not delete it.
- `image-prepare` does not strip an enrolled identity unless `--i-am-building-an-image` is present.

`apt-get update` metadata is not rolled back if a later step fails after the NetBird repository was added. The package, the list file, and the keyring are. That gap is recorded in the journal as the package action only when install of the package was reached.

## Lock (§13.5)

The lock file is `/var/lib/vpn-node/update.lock`. The installer, the agent, and the updater use `LOCK_EX|LOCK_NB` on that file. The holder record is JSON `{pid, boot_id, start_time}` written only after the flock succeeds. The file is not unlinked when the lock is busy. `start_time` is field 22 of `/proc/<pid>/stat`.

`packaging/installer` cannot import `node/internal/nodelock`, so `lock.go` repeats that protocol. The agent takes the lock inside `node/agent/engine.go` while it applies a serving dataplane, and releases it at the end of that apply. The updater takes it for `run` and `recover`. The watchdog does not take it. The installer holds it across filesystem and nft changes and releases it before starting `fvpn-updater`.

## Lab wrappers

`RerunInstaller` is:

```sh
fvpn-node-install install --json \
  --manifest <manifest.json> --artifact <artifact.tar.zst> --trust <trust.json> \
  --control-url <https-url> --server-ca <ca.pem> --trust-bundle <node-trust.json> \
  --host-reserve-bps <positive-integer> --snat <ipv4> \
  --netbird-management-url <https-url> \
  --token-file <path-mode-0600>
```

Omit `--token-file` on a re-run when identity already exists. Hash, before and after, with the agent stopped if the hash must be byte-stable: `identity/key.pem`, `identity/cert.pem`, `identity/node.json`, `state/usage.db` (and `usage.db-wal` when present), `/etc/machine-id`, and `readlink /opt/vpn-node/current`. `already_current` keeps those hashes. A newer archive changes only the `current` link.

`ControlledReinstall` is:

```sh
fvpn-node-install reenroll --json --token-file <path-mode-0600>
```

Exit 0 and status `reenrolled` means the new `node.json` incarnation differs, `identity.retired/<utc>/` exists and is mode 0700, and `usage.db` was not deleted. The old certificate's rejection by central is still lab evidence (ENR-02).
