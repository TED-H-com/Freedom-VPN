# Follow-up 35 checks

All required gate workloads ran after admission, sequentially. The canceled pre-admission waiter and the corrected evidence-description failure remain below. No acceptance scenario or run was retried.

## 01-e2e-lint

Exit 0; 4.896 s. UTC 2026-11-04T09:23:20.522114+00:00–2026-11-04T09:23:25.417887+00:00.

`make -f mk/e2e.mk e2e-lint`

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

## 02-isolation-lint

Exit 0; 4.558 s. UTC 2026-11-04T09:23:34.047557+00:00–2026-11-04T09:23:38.605108+00:00.

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

Exit 0; 180.492 s. UTC 2026-11-04T09:23:53.094964+00:00–2026-11-04T09:26:53.586763+00:00.

`go vet -tags e2e ./tests/e2e/...`

```text
(no output)
```

## 04-e2e-unit-not-admitted

Exit 241; 28.606 s. UTC 2026-11-04T09:24:32.484421+00:00–2026-11-04T09:25:01.090703+00:00.

`go test -count=1 ./tests/e2e/...`

```text
(no output)
```

## 04-e2e-unit

Exit 0; 29.163 s. UTC 2026-11-04T09:27:18.398619+00:00–2026-11-04T09:27:47.561762+00:00.

`go test -count=1 ./tests/e2e/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.063s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.108s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.003s
```

## 05-acceptance

Exit 2; 1434.417 s. UTC 2026-11-04T09:39:25.935949+00:00–2026-11-04T10:03:20.352841+00:00.

`FVPN_E2E_RUN_LABEL=followup35 make -f mk/e2e.mk e2e-test`

```text
--- PASS: TestD160_HeldBeforeSectionWaitsBeforeStart (0.02s)
=== RUN   TestD158_TwoUpdaterInvocationOutputsSurvive
    updater_invocations_test.go:56: two invocations retained byte-for-byte; exits 0/3 and start/end/PID receipts verified
--- PASS: TestD158_TwoUpdaterInvocationOutputsSurvive (0.02s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	1386.659s
FAIL
make[1]: *** [mk/e2e.mk:67: e2e-acceptance] Error 1
make[1]: Leaving directory '<worktree>'
make: *** [mk/e2e.mk:60: e2e-test] Error 2
```

## 06-e2e-down

Exit 0; 47.598 s. UTC 2026-11-04T10:04:01.130524+00:00–2026-11-04T10:04:48.728238+00:00.

`make -f mk/e2e.mk e2e-down`

```text
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Resource is still in use
 Network fvpn-e2e Resource is still in use
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
e2e stack is down
```

## 06-owned-inventory

Exit 0; 0.411 s. UTC 2026-11-04T10:05:45.844454+00:00–2026-11-04T10:05:46.255689+00:00.

`python3 -B .agent/tmp/followup35/inventory.py`

```text
{
  "at": "2026-11-04T10:05:45.877729+00:00",
  "prefix": "fvpn-e2e",
  "containers": [],
  "images": [],
  "networks": [],
  "volumes": []
}
```

## 07-handover

Exit 0; 4.752 s. UTC 2026-11-04T10:05:57.886377+00:00–2026-11-04T10:06:02.638535+00:00.

`go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.891s
```

## 08-go-lint

Exit 0; 6.321 s. UTC 2026-11-04T10:07:05.631315+00:00–2026-11-04T10:07:11.952406+00:00.

`make go-lint go-with-disk-tmp=`

```text
(no output)
```

## 09-gofmt

Exit 0; 0.226 s. UTC 2026-11-04T10:08:01.722224+00:00–2026-11-04T10:08:01.947917+00:00.

`bash -c 'test -z "$(gofmt -l $(git ls-files '"'"'*.go'"'"'))"'`

```text
(no output)
```

## 10-diff-check

Exit 0; 0.091 s. UTC 2026-11-04T10:08:18.579366+00:00–2026-11-04T10:08:18.670823+00:00.

`git diff --check`

```text
(no output)
```

## 11-evidence-hygiene

Exit 1; 9.331 s. UTC 2026-11-04T10:08:32.826541+00:00–2026-11-04T10:08:42.157359+00:00.

`go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

```text
--- FAIL: TestCommittedFixtureEvidence_ValidAndHygieneClean (9.01s)
    evidence_test.go:150: unsafe committed evidence: ../../docs/e2e/runs/2026-11-04/followup35/log-publication-receipts.txt
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	9.015s
FAIL
```

## 12-evidence-hygiene-fixed

Exit 0; 9.413 s. UTC 2026-11-04T10:08:59.609126+00:00–2026-11-04T10:09:09.022288+00:00.

`go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	9.088s
```

## 13-nightly-gate

Exit 0; 0.256 s. UTC 2026-11-04T10:10:16.577615+00:00–2026-11-04T10:10:16.833898+00:00.

`go run ./tests/e2e/cmd/fvpn-e2e-nightly-gate -summary docs/e2e/runs/2026-11-04/followup35/acceptance-summary.md -test-exit-code 2 -test-log docs/e2e/runs/2026-11-04/followup35/05-acceptance.txt`

```text
- Source commit: `9fbff94c84b0d29c2d78f420e76726f7c9a706ad`
- Run ID: `8a436a87-5efe-4c18-a7ec-72be170cb69b`
- Evaluated at: `2026-11-04T10:10:16Z`
- Counts: pass=32, expected_blocked=1, unblocked=0, unexpected_blocked=0, fail=0, error=0

### Expected-Blocked Rows (acceptance status remains `blocked`)

| Scenario | Acceptance Status | Owner | Reason | Detail |
|---|---|---|---|---|
| `member-ipv6` | **blocked** | w3-devstack-e2e / platform operators | Default native member container joins with --disable-ipv6 (DisableIPv6: true) and no IPv6 underlay or per-platform route/capture campaign is provisioned in the default devstack acceptance run (transport_scenario_test.go memberIPv6). |  |
```

## 14-staged-whitespace-before

Exit 2; 0.09 s. UTC 2026-11-04T10:10:57.582684+00:00–2026-11-04T10:10:57.672905+00:00.

`git diff --cached --check`

```text
docs/e2e/runs/2026-11-04/followup35/nightly-gate.md:14: new blank line at EOF.
```

## 15-staged-whitespace-after

Exit 0; 0.094 s. UTC 2026-11-04T10:10:57.783439+00:00–2026-11-04T10:10:57.877776+00:00.

`git diff --cached --check`

```text
(no output)
```

## 16-final-evidence-hygiene

Exit 0; 9.186 s. UTC 2026-11-04T10:13:45.802190+00:00–2026-11-04T10:13:54.988366+00:00.

`go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.878s
```

## 17-final-diff-check

Exit 0; 0.092 s. UTC 2026-11-04T10:14:59.963177+00:00–2026-11-04T10:15:00.055085+00:00.

`git diff --check`

```text
(no output)
```
