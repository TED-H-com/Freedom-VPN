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

## Follow-up 7 measured result — 2026-10-31

Run `59ef4bf8-7d4a-490f-a74d-6b54394375e0`, UTC `2026-10-31T08:58:59Z`–`2026-10-31T09:05:27Z`.
AT06 424/0/0; AT07 36/5/0. Full and join rate-limit reviews each record 0 HTTP 429.
Source `5412e257ee0735877239ae18543f6b81a98a2ae2` is clean at invocation.
Tree SHA-256 `077fd665095300e9e51a99e32cae684a56638202262fe5de8e97c75244f95247`;
probe SHA-256 `1795cb53d40d96c463b8abb93ffc666124458e1f56f26ef2ae090459ec35fab8`.

Admission was checked before both starts: 08:59:00 UTC and 09:04:01 UTC,
each with one other suite (members). The runner tears down the prior phase
before any admission wait. All three teardown commands exit 0; the final
inventory has zero containers, networks, volumes and images.

```text
fresh_teardown: exit 0
fresh: admitted with 1 other suite(s)
fresh_startup: exit 0
compile_probe: exit 0
live_probe: exit 0
join_teardown: exit 0
join: admitted with 1 other suite(s)
join_startup: exit 0
live_probe_join: exit 0
final_teardown: exit 0
```

The live make exit is 2 for AT07's five documented blockers. All current
assertions pass. Prior failures are retained and explicitly reviewed: D-078
and the diagnostic member-name lookup are resolved by the fresh observations.
The initial follow-up-7 attempt stopped in join admission after a completed
full phase and successful early teardown; [review](history/d83ac47d-0074-42d7-bb3b-65a2db536be5/REVIEW.md).

The serving freeze bound is 227 ms from central conflict to completed live-rule
read-back, before/including any receipt delivery latency; no exact receipt
arrival time is invented. The agent's configured heartbeat is 5 seconds.
After restart, SQLite identity_frozen=1, fresh heartbeat/report, zero rules and
zero lease members establish persistence. No packet-throughput or OS reboot
claim follows from these checks.
