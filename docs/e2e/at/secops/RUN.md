# Follow-up 6 run instructions

From this worktree, with Go 1.27.3 and Docker available:

```sh
make -f mk/at-secops.mk at-secops-followup6
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
make go-lint
```

The runner fixes `FVPN_E2E_PREFIX=w4-secops` and `FVPN_E2E_PORT_BASE=30000`.
It refuses an occupied namespace. It uses the standard `mk/e2e.mk` for two
fresh starts and unconditional teardown; each phase is bounded to 50 minutes.
The first runs the unchanged sequential 20-user × 3-peer profile. The second
uses `ScaleNativeMembers` on a separate stack, with a real admin-login refresh
callback. All eligibility comes from finite API grants. All device and binding
rows come from product discovery. SQL is read-only.

The helper phase also measures anonymous webhook saturation while admin reads
and certified lease renewals proceed, then reuses the rate-class probe for
usage sizes 8 MiB−1, 8 MiB and 8 MiB+1. No serving policy, counted traffic,
node resource capacity or relay throughput is inferred from admission.
Control follow-up 18 is absent from the measured main; its untrusted-forwarded-
header row remains a later owner-dependent run.

Public evidence stores response statuses, counts, hashes and timings.
Transient native device-flow scans return only presence, byte count, mode and
recognized-key counts. `JoinNativeMember` scans before deleting its file; the
exported bulk helper returns aggregate scan counts, supplemented here by an
independent exported-scanner watcher. A protected positive key count is not
reported as zero. Private raw logs remain under the ignored task state.

After both phases, render records with:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tests/e2e/at/secops/followup6_records.py
```

A read-only gateway observer records socket states and file-descriptor counts; it never exports addresses. In the recorded run it was invoked alongside the target partway through the first phase. The reusable target now starts it automatically and stops it in cleanup. The recorded initial worker-count command used unavailable `ps`; those zeros are invalid and are explicitly qualified. The helper phase uses `/proc` and confirms two workers.

The full target returns 2 while full acceptance gates remain. Inspect the live
probe exit separately from the full-case verdict. Previous run records are
preserved under `history/follow-up-5/`; they retain their original source and
time and are never relabeled as this run. Physical operator procedures remain
in `operator-runs.md`. The workflow has no dependency changes.
