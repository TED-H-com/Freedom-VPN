# Enrollment follow-up 8 runbook

Binding `w4-enroll` / `12000`; Docker objects `fvpn-w4-enroll-*`. All fixtures
are TEST ONLY. Check `pgrep -af 'gates.sh|mk/at-'` before each start and wait
while two or more other suites are active. The runner does this for both fresh
phases and always tears down its own stack with Python/harness cleanup.
TMPDIR/GOTMPDIR and SQLite copies use `.agent/tmp`; no shell cleanup traps.

```sh
make -f mk/at-enrollment.mk at-enrollment-lint
make -f mk/at-enrollment.mk at-enrollment
make go-lint 'go-with-disk-tmp=python3 tests/e2e/at/enrollment/disktmp.py'
python3 tests/e2e/at/enrollment/disktmp.py go vet ./...
python3 tests/e2e/at/enrollment/disktmp.py go test ./...
git diff --check
```

## Measured result

Run `f27a81a0-183e-46b4-85a8-304b2f64b635`, UTC `2026-10-31T13:54:03Z`–`2026-10-31T14:07:16Z`.
Source `569ef8d5cedc48d24bff4b1f67165aa702d269a6`, clean at invocation: `True`.
Assignment baseline `6d0022d8` includes #302/#312/#324.
Tree SHA-256 `858a0e271f91f331697616ecfc82a64686d1c80f79d3d52b62056302dce68a12`;
probe SHA-256 `2b5ad5522bb634a1b16b41129c4a83a22e6c6782945fdd158bcec5776a69c386`.
AT06 501 pass / 0 blocked / 1 fail; AT07 43 pass / 1 blocked / 2 fail. Live make exits 2: full probe 1,
join probe 0, all three teardown commands 0. AT06's actual subtest passes;
its aggregate failure is the shared process exit from AT07, not an inferred
AT06 product failure. Final inventory: `{"containers": 0, "images": 0, "networks": 0, "volumes": 0}`.

Before **908** + held **1816** = after **2724 bytes**. Two original events remain attributed once only to the retired incarnation. Origin ACK **3**, independent current ACK **0** through same-key replay; the new stream starts at sequence **1**.

Freeze withdrawal was **334 ms**, **+107 ms** against the 227 ms baseline, below the 5000 ms bound. Real receipt replay and durable latch/restart passed.

## Sequence and evidence boundaries

PrepareInstallerNode/RegisterInstallerReleases, API token, first-poll hold and
RerunInstaller create a real installed node. Configure the resolver backend,
restore captured services, publish by API, select the native exit and require
fresh serving, text marker read-back, TCP/UDP receipts and admission counters.
Wait 90s for baseline sampling/upload (AT20's 10s + 60s allowance), hold usage,
generate bytes and rerun through the shared wrapper. Preserve systemd limits
by pacing restarts; compare each wrapper's stopped before/after files.

Save services, stop updater timer/service, watchdog and agent in documented
order. Require two byte-equal CLI snapshots. Retire by ETag/Idempotency-Key API,
observe retired history and external route/identity cleanup, obtain a fresh
same-node token and call ControlledReinstall. A stopped old agent cannot ACK
withdrawal; successful external steps are not mislabeled full-operation success.

Follow [AT20](../../../usage/AT20.md): preserve original canonical payload/hash/ACK,
allow production D-118 adoption, restore services, release the hold and observe
the origin stream first. Read coherent stopped DB/WAL copies, API totals/history,
and central event/sample/checkpoint evidence. Replay the saved production batch
under its same key/current certificate and compare canonical result, unchanged
current ACK and unchanged totals. No SQL transition or synthetic event is used.

Revoke disables publication. After accounting, re-publish through the API and
prove fresh serving and counted traffic. The additional current-stream ACK
wait exposes the [1 ns gap defect](current-stream-gap.md). Do not promote that
row or rerun endlessly before its owners fix it. The final run's current CLI,
batch bodies, central reads and constraint evidence are retained.

[D-098 archive](../follow-up-7/README.md) predates replacement writes.
[Raw run](../f27a81a0-183e-46b4-85a8-304b2f64b635/AT07.json) is unchanged. [Published assessment](../f27a81a0-183e-46b4-85a8-304b2f64b635-assessment/README.md)
only corrects owner/evidence wording for the reached-but-blocked reinstall row;
all statuses, check names and prior results are identical. See [requests](requests.md).
