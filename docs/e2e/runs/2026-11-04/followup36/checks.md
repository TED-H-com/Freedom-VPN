# Follow-up 36 checks

Required gate workloads ran in foreground order after admission. Elapsed command time includes any recorded admission wait. Acceptance scenarios were not retried.

## 01-e2e-lint

Exit 0; 1266.155 s. UTC 2026-11-04T14:09:05.605353+00:00–2026-11-04T14:30:11.760440+00:00.

`make -f mk/e2e.mk e2e-lint`

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

## 02-isolation-lint

Exit 0; 4.531 s. UTC 2026-11-04T14:30:58.781921+00:00–2026-11-04T14:31:03.312721+00:00.

`make -f mk/at-isolation.mk at-isolation-lint`

```text
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

## 03-e2e-vet

Exit 0; 0.258 s. UTC 2026-11-04T14:31:14.080715+00:00–2026-11-04T14:31:14.339064+00:00.

`go vet -tags e2e ./tests/e2e/...`

```text
(no output)
```

## 04-e2e-unit

Exit 0; 39.121 s. UTC 2026-11-04T14:31:19.655895+00:00–2026-11-04T14:31:58.777145+00:00.

`go test -count=1 ./tests/e2e/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.063s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.003s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

## 05-acceptance

Exit 2; 1439.271 s. UTC 2026-11-04T14:32:49.364796+00:00–2026-11-04T14:56:48.636097+00:00.

`FVPN_E2E_RUN_LABEL=followup36 make -f mk/e2e.mk e2e-test`

```text
--- PASS: TestD160_HeldBeforeSectionWaitsBeforeStart (0.02s)
=== RUN   TestD158_TwoUpdaterInvocationOutputsSurvive
    updater_invocations_test.go:56: two invocations retained byte-for-byte; exits 0/3 and start/end/PID receipts verified
--- PASS: TestD158_TwoUpdaterInvocationOutputsSurvive (0.02s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	1398.737s
FAIL
make[1]: *** [mk/e2e.mk:67: e2e-acceptance] Error 1
make[1]: Leaving directory '<worktree>'
make: *** [mk/e2e.mk:60: e2e-test] Error 2
```

## 05-observation-export

Exit 0; 600.802 s. UTC 2026-11-04T14:56:59.157377+00:00–2026-11-04T15:06:59.959138+00:00.

`python3 -B .agent/tmp/followup36/export_observations.py`

```text
  "window_start": "2026-11-04T14:32:49.364796+00:00",
  "window_end": "2026-11-04T14:56:48.636097+00:00",
  "count": 123,
  "interval_count": 122,
  "minimum_seconds": "0.081075",
  "median_seconds": "5.162159",
  "maximum_seconds": "10.254596",
  "scope": "Every accepted regular-node row in the acceptance command window, including state changes, outages and replay. Receipt intervals are not submission timing or guaranteed periodicity."
}
Prepared cadence: {"agent-enroll.json": {"sha256": "b13f4b9847fac5ab41ee708254e18639af170cba189560ea7d62808f7c40a45f", "fields": {"heartbeat_seconds": 5, "poll_seconds": 5, "observation_seconds": 5}}, "agent-run.json": {"sha256": "e67665ed30ca57baa68010d6cc831d5b042e2de355ed50d9c0ecca55e117ce06", "fields": {"heartbeat_seconds": 5, "poll_seconds": 5, "observation_seconds": 5}}}
```

## 06-e2e-down

Exit 0; 47.165 s. UTC 2026-11-04T15:08:23.137847+00:00–2026-11-04T15:09:10.302627+00:00.

`make -f mk/e2e.mk e2e-down`

```text
 Volume fvpn-e2e-pgbackrest-repo Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e-origin Resource is still in use
 Network fvpn-e2e Resource is still in use
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
e2e stack is down
```

## 06-owned-inventory

Exit 0; 0.499 s. UTC 2026-11-04T15:09:38.355608+00:00–2026-11-04T15:09:38.854514+00:00.

`python3 -B .agent/tmp/followup36/inventory.py`

```text
{
  "at": "2026-11-04T15:09:38.457418+00:00",
  "prefix": "fvpn-e2e",
  "containers": [],
  "images": [],
  "networks": [],
  "volumes": []
}
```

## 07-handover

Exit 0; 4.924 s. UTC 2026-11-04T15:09:45.940619+00:00–2026-11-04T15:09:50.864594+00:00.

`go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	4.035s
```

## 08-go-lint

Exit 0; 4.132 s. UTC 2026-11-04T15:09:56.357059+00:00–2026-11-04T15:10:00.488571+00:00.

`make go-lint go-with-disk-tmp=`

```text
(no output)
```

## 09-gofmt

Exit 0; 0.208 s. UTC 2026-11-04T15:10:31.367483+00:00–2026-11-04T15:10:31.575485+00:00.

`bash -c 'test -z "$(gofmt -l $(git ls-files '"'"'*.go'"'"'))"'`

```text
(no output)
```

## 10-diff-check

Exit 0; 0.092 s. UTC 2026-11-04T15:10:34.647670+00:00–2026-11-04T15:10:34.739736+00:00.

`git diff --check`

```text
(no output)
```

## 11-evidence-hygiene

Exit 0; 8.693 s. UTC 2026-11-04T15:10:38.837151+00:00–2026-11-04T15:10:47.530516+00:00.

`go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.390s
```

## 12-nightly-gate

Exit 0; 0.26 s. UTC 2026-11-04T15:10:52.972509+00:00–2026-11-04T15:10:53.232793+00:00.

`go run ./tests/e2e/cmd/fvpn-e2e-nightly-gate -summary docs/e2e/runs/2026-11-04/followup36/acceptance-summary.md -test-exit-code 2 -test-log docs/e2e/runs/2026-11-04/followup36/05-acceptance.txt`

```text
- Source commit: `f6edbda9909268106f7a8e08c26aab8254ae1481`
- Run ID: `c17589fa-7e48-4dee-b574-7e29aca1effa`
- Evaluated at: `2026-11-04T15:10:53Z`
- Counts: pass=32, expected_blocked=1, unblocked=0, unexpected_blocked=0, fail=0, error=0

### Expected-Blocked Rows (acceptance status remains `blocked`)

| Scenario | Acceptance Status | Owner | Reason | Detail |
|---|---|---|---|---|
| `member-ipv6` | **blocked** | w3-devstack-e2e / platform operators | Default native member container joins with --disable-ipv6 (DisableIPv6: true) and no IPv6 underlay or per-platform route/capture campaign is provisioned in the default devstack acceptance run (transport_scenario_test.go memberIPv6). |  |
```

## 13-final-diff-check

Exit 0; 60.131 s. UTC 2026-11-04T15:14:05.385745+00:00–2026-11-04T15:15:05.516395+00:00.

`git diff --cached --check`

```text
(no output)
```

## 14-final-evidence-hygiene

Exit 0; 8.757 s. UTC 2026-11-04T15:17:07.229024+00:00–2026-11-04T15:17:15.985756+00:00.

`go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.454s
```
