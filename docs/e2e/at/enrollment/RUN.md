# Enrollment follow-up 7 runbook

Binding: w4-enroll / 12000. All fixtures and identifiers are TEST ONLY.
First inspect `pgrep -af 'gates.sh|mk/at-'`; wait while two or more other
suites are active. Use the foreground runner and its unconditional Python
teardown. Disk scratch lives under `.agent/tmp`; no shell cleanup traps.

```sh
make -f mk/at-enrollment.mk at-enrollment-lint
make -f mk/at-enrollment.mk at-enrollment
make go-lint 'go-with-disk-tmp=python3 tests/e2e/at/enrollment/disktmp.py'
python3 tests/e2e/at/enrollment/disktmp.py go vet ./...
python3 tests/e2e/at/enrollment/disktmp.py go test ./...
git diff --check
```

The live target intentionally remains nonzero while any required AT07 row is
blocked. Root AT06/AT07 JSON files retain per-row observations, exact source,
probe digest, UTC intervals, command exits and cleanup inventory. Immutable
attempts remain in history. The follow-up-6 archive predates all new writes.

Independent installer checks compare each wrapper's quiescent before/after
files, restore its captured services with the shared helper, and exercise the
installer-node authenticated proxy hold/release plus exact ledger snapshots.
The five composed serving/retired-stream rows retain D-116/D-105/D-112 blockers.
The ordinary enrolled node supplies the separate populated-lease freeze source;
it is never substituted for installer-node usage retention.
