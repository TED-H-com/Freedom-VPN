# Installer follow-up 4 — 2026-10-31

D-116 removes the installer ownership marker from the enforcement table. The installer now creates `inet fvpn_install` with one empty regular `owner` chain, comment `managed-by=fvpn-node-install`, and no hook. Enforcement stays unchanged. Base source: `107b9791069284e7cd881d3643e24dd098feb780`; implementation commits: `51ccbd8`, `09a2dd3`, and `2dc5344` (final code used by the container run).

## Ownership and migration

Install, same-release rerun, and upgrade reconcile the marker under the shared update lock. A follow-up 3 `inet fvpn owner` chain is removed only when it is regular, empty, and has the exact installer comment. Unexpected ownership shapes fail with `NFT_MARKER`. Creation and migration have read-back checks. Repeated runs keep the same marker and leave the agent's four chains intact. The migration is forward-only; rollback never recreates the incompatible chain.

Remove, purge and rollback delete only the validated `inet fvpn_install` marker. **They preserve `inet fvpn`, including when an old `owned.json` or journal says the installer created it.** No global ruleset flush, host default-route change, or DNS rewrite is introduced. The existing owned netns/link cleanup remains.

`check --json` reports `installer_marker.table`, `present`, and `legacy_owner`; text check reports the same. A legacy or malformed marker is drift. An absent marker after reboot is informational because it has no forwarding role; rerun/upgrade recreates it. Agent/watchdog baseline restoration remains independent.

## Container observations

TEST ONLY, privileged Ubuntu 24.04 amd64 containers, nft 1.0.9, real packaged binaries/units and real updater seed. `TestAT07_D116InstallerMarkerAndAgentRules` passed in 5.42 s in the final full suite. After install/enrollment, the production baseline verifier returned success. The enrollment-only control supplies no signed desired state, lease or native traffic topology, so **serving is not claimed**. Read-back was:

```text
VerifyBaseline=pass
VerifyServing=serving generation must be non-zero; 0 is the boot baseline
baseline_ok 1
serving 0
reason clock_untrusted
```

The serving verifier was called with the actual generation-0 baseline model; its error is retained rather than presented as serving success. The test reads the complete ruleset, asserts exactly `forward`, `input`, `output`, `postrouting` inside `inet fvpn`, and asserts exactly one empty regular marker outside it. Actual `nft list ruleset`:

```nft
table inet foreignw5 {
	chain keep {
		type filter hook input priority filter; policy accept;
	}
}
table inet fvpn_install {
	chain owner {
		comment "managed-by=fvpn-node-install"
	}
}
table inet fvpn {
	counter ctbase {
		packets 0 bytes 0
	}

	counter ctv6 {
		packets 0 bytes 0
	}

	counter ctin {
		packets 0 bytes 0
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
	}

	chain forward {
		type filter hook forward priority mangle; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1"
		iifname "fvmb0" counter name "ctbase" drop
		oifname "fvmb0" counter name "ctbase" drop
		iifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
	}

	chain input {
		type filter hook input priority mangle; policy accept;
		iifname "fvmb0" counter name "ctin" drop
		ip saddr 100.64.0.0/10 counter name "ctin" drop
	}

	chain output {
		type filter hook output priority mangle; policy accept;
	}

	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;
	}
}
```

The migration fixture then stopped the agent, removed the new marker table and inserted the exact legacy empty owner chain (TEST ONLY reproduction). Same-release `install` removed that chain and recreated the separate marker. Two reruns retained byte-identical agent-table JSON and the same marker handle; `check` reported the new marker and no legacy owner. The foreign table, default route and DNS digest stayed unchanged. Remove and purge cases separately passed with `fvpn_install` absent and `fvpn` retained. Unit coverage also proves migration through `upgrade` and refusal of modified/hooked/nonempty markers.

The initial container attempt exposed nft block syntax; the next read-back exposed nft 1.0.9 creating the table without its nested chain. The final code uses explicit `create table` and `add chain` commands in one nft transaction and validates the result. Those initial failures are not counted as passes.

## F9-01 confirmation

The final `TestInstallLifecycle` passed in 15.00 s and reproduced the relevant packaged-unit conditions from [F9-01](../e2e/at/updates/requests.md): `ProtectSystem=strict` remained enabled, the real pinned NetBird 0.82.0 daemon bound `/run/fvpn-netbird/netbird.sock`, `/var/run/netbird.sock` was absent, enrollment completed, services became active and `check` was healthy. Stopping/starting every shared-runtime consumer preserved `/run/vpn-node`; NetBird restart recreated its socket. Journals contained no `read-only file system` diagnostic, covering the original `listen unix /var/run/netbird.sock: bind: read-only file system` failure. The #279 fix needs no further socket change here. Daemon/status/version are real; only management `up` registration is substituted with a real socket status check, so this does not claim a native tunnel or serving acceptance.

## Temporary updater contract

[Install documentation](install.md#temporary-updater-contract-retire-after-follow-up-15--agent-core-follow-up-24) records the current limitations, with owner and retirement markers:

- F9-02: packaged installer/seed umask is 077; standalone seed must match. Pending updater follow-up 15.
- F9-03: both updater oneshots accept exits 0 and 2; the timer invokes the same service. `apply` can exit 0 while sealing failure. `check` exposes terminal `updater.state`; full `status` exposes the sealed result. Layout health does not imply update success. Pending updater follow-up 15.
- F9-03 checkpoint: the generated `state/meter.checkpoint` remains absent; the real journaled upgrade must stay failed until the genuine barrier is connected. Pending agent-core follow-up 24.
- F9-05: a lower sequence can currently be reported `staged` by upgrade; staging is not application. Pending updater follow-up 15.

No updater, agent, or updater-test implementation changed in this lane. Follow-up 5 reruns the full journaled upgrade once both owner fixes are on main.

## Gates

| Gate | Result |
|---|---|
| `make -f mk/installer.mk installer-lint installer-unit installer-build` | PASS; vet/staticcheck, unit tests, amd64/arm64 builds |
| `make -f mk/installer.mk installer-test` | **FAIL overall**: 10 pass / 1 fail; 62.076 s. D-116 and F9-01 pass. |
| `go test -count=1 ./tests/handover/...` | PASS |
| `make go-lint` | PASS; Python disk-scratch wrapper replaces the make helper's shell trap |
| gofmt, `git diff --check`, committed-evidence hygiene | PASS |

The remaining required container failure is unchanged product work:

```text
--- PASS: TestInstallLifecycle (15.00s)
--- PASS: TestRemovePurge (6.12s)
--- PASS: TestImagePrepareTwoBoots (14.72s)
verified baseline must apply: exit=0 state=failed stage=final_sample current=019a0000-0000-7000-8000-0000000000c1 meter_checkpoint_present=false
--- FAIL: TestD089_RealSeedThenJournaledUpgrade (6.37s)
--- PASS: TestD089_SeedRefusalKeepsInstalledTreeAndJournal (6.98s)
--- PASS: TestAT07_D116InstallerMarkerAndAgentRules (5.42s)
FAIL github.com/TED-H-com/Freedom-VPN/tests/installer 62.076s
```

Admission waited while two other suites were active; only one remained before starting. Initial admission read-back: `max_user_instances=128`, 36 running containers. No inotify failure occurred. All installer-prefixed Docker objects were cleaned. Arm64 was cross-built, not executed on hardware.

## Requests to devstack and handover (D-102)

1. After merge, rebuild the packaged installer release and rerun ENR-05 / the real installer-node serving fixture. Include a same-release rerun of an old follow-up 3 installation: retain the original marker verdict, invoke the installer migration, then observe the strict agent verifier, native member traffic and held usage. Capture `nft -j list ruleset`, `installer_marker`, heartbeat/central serving state, and counters. Four-chain/marker evidence here does not substitute for real serving or nonzero usage acceptance. Keep the ENR-06 service-restoration wrapper contract from devstack follow-up 17.
2. Keep the full journaled upgrade open for updater follow-up 15 and agent-core follow-up 24. Route this note to installer follow-up 5 after both merge, and retire each temporary contract paragraph only after its corresponding live check passes. No fabricated checkpoint or health response is authorized.
