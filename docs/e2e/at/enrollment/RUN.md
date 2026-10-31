# Enrollment follow-up 9 runbook

Binding `w4-enroll` / `12000`; only `fvpn-w4-enroll-*` Docker objects.
All fixtures are TEST ONLY. The runner checks `pgrep -af 'gates.sh|mk/at-'`
before each fresh phase and waits while two or more other suites are active.
TMPDIR/GOTMPDIR and coherent SQLite copies use `.agent/tmp`; Python/harness
cleanup removes the owned stack. No shell cleanup traps.

```sh
make -f mk/at-enrollment.mk at-enrollment-lint
make -f mk/at-enrollment.mk at-enrollment
make go-lint 'go-with-disk-tmp=python3 tests/e2e/at/enrollment/disktmp.py'
python3 tests/e2e/at/enrollment/disktmp.py go vet ./...
python3 tests/e2e/at/enrollment/disktmp.py go test ./...
git diff --check
```

## Measured result

AT06 **662 pass / 0 blocked / 0 failed**; AT07 **48 pass / 0 blocked / 0 failed**.
Run `891ee9b8-bf01-4583-a47a-9542c8b04c8a`, `2026-10-31T14:59:07Z`–`2026-10-31T15:16:38Z`.
Source `4d49994d7f3a653b44d006494ad922ded722b5b8`, clean at invocation: `True`.
Tree SHA-256 `b43d67cb8c7e850267ef82abc29df50c392e70c716ac7ed07a5a301a433aa60f`;
probe SHA-256 `59d39834d2278894900fc7f8bce7b758ea7c1bc7d134246cee1de3c859cadef7`.
All command exits are 0; final inventory `{'containers': 0, 'networks': 0, 'volumes': 0, 'images': 0}`.

Retired usage: **908 + 1816 = 2724 bytes**, original ACK **3**. The fresh stream independently advances **1 → 2**, attributing **550 upload + 358 download = 908 bytes** to its own incarnation, member, device and binding revision. Historical payloads, hashes, samples and totals remain unchanged.

## Sequence and evidence

Use PrepareInstallerNode/RegisterInstallerReleases, API enrollment, first-poll
hold, RerunInstaller, ConfigureInstallerServing and captured service restoration.
Publish by API, select the native exit, and observe strict four-chain enforcement,
hookless marker, TCP/UDP receipts and admission counters. Hold nonzero usage;
rerun with stopped file equality, restore and count again. Pace systemd restarts
without changing packaged limits. Stop consumers in documented order and require
two byte-equal CLI snapshots.

Revoke through ETag/Idempotency-Key API, observe retired history and external
operation steps, issue a same-node token, then use ControlledReinstall. Preserve
original payload/hash/ACK evidence across production adoption; restore and release
the hold for original-stream delivery. Replay its actual saved batch/key with
the current certificate and compare canonical results, totals and independent ACKs.

Re-publish after revoke, prove serving, then wait for the fresh empty gap's
own ACK 1 **before** generating fresh counted traffic. Require event 2, ACK 2,
550/358 central sample bytes and matching API totals. Retain successful CLI,
heartbeat and production batch evidence through a stopped coherent copy.
Read the gap's actual stored endpoints; this run reports `2026-10-31T15:11:08.150045Z` to `2026-10-31T15:11:08.150046Z`.
Only read-only evidence SQL is used. All four named AT20 assertions execute anew.

D-098 ran first: [51 archived files](history/follow-up-8/README.md), each matching
both the checksum index and original main bytes. Exact earlier deadline/cursor
and D-130 failures remain in `prior_checks`; explicit reviews resolve only those
known failures after the complete new proof passes. No failure was discarded.

## Limits

Physical arm64 and OS reboot remain unmeasured; arm64 compilation and agent process restart are separate evidence. Post-upgrade seed behaviour belongs to **devstack 19 / installer 7 / updater 18** and is not re-measured here. The [historical devstack 18 seed failure](../../runs/2026-10-31/followup18/README.md#candidate-health-and-post-upgrade-reenrollment) and [updater diagnosis](../../../updates/node-updater.md#post-upgrade-seed-diagnosis--2026-10-31) remain references, not new enrollment acceptance. This run performs no upgrade.
