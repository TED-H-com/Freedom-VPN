# Follow-up 26 gates

All commands ran in the foreground. Integration concurrency was checked before and after each fresh stack startup. The private coordinator waited while two other suites were active; public campaign rows retain counts rather than other lanes' process metadata. Go 1.27.3, `GOTOOLCHAIN=local`, `CGO_ENABLED=0`, disk scratch under `.agent/tmp/followup26`, Go cache obtained with `go env GOCACHE`.

Final required post-teardown gates all exit 0. Initial compilation errors are retained below; they sent no request. Four native-runner failures are the retained ce8ccbe1 first-selection timeout, af718646 re-grant timeout and two deliberate tracer-hold probes. They remain failures. The stop and counterread focused runners pass.

| Command label | Start (UTC) | Exit | Elapsed seconds |
|---|---|---:|---:|
| ipv4-01-up | 2026-11-01T11:55:19Z | 0 | 32.642 |
| ipv4-01-native | 2026-11-01T11:55:56Z | 1 | 0.146 |
| focused-counterread | 2026-11-01T11:56:53Z | 0 | 0.229 |
| ipv4-01-native-compiled | 2026-11-01T11:56:04Z | 0 | 59.534 |
| container-stop-live | 2026-11-01T11:57:24Z | 0 | 18.412 |
| ipv4-01-down | 2026-11-01T11:59:54Z | 0 | 22.155 |
| ipv4-02-up | 2026-11-01T12:00:58Z | 0 | 31.719 |
| ipv4-02-native | 2026-11-01T12:01:30Z | 1 | 0.117 |
| ipv4-02-down | 2026-11-01T12:01:30Z | 0 | 22.032 |
| ipv4-03-up | 2026-11-01T12:01:52Z | 0 | 30.682 |
| focused-trace-regression | 2026-11-01T12:02:53Z | 0 | 0.998 |
| focused-trace-final | 2026-11-01T12:06:32Z | 0 | 1.023 |
| focused-session-compile | 2026-11-01T12:11:23Z | 0 | 0.974 |
| ipv4-03-native | 2026-11-01T12:22:19Z | 0 | 92.546 |
| ipv4-03-down | 2026-11-01T12:23:52Z | 0 | 22.555 |
| ipv4-04-up | 2026-11-01T12:24:14Z | 0 | 32.488 |
| ipv4-04-native | 2026-11-01T12:24:47Z | 0 | 86.427 |
| ipv4-04-down | 2026-11-01T12:26:13Z | 0 | 22.347 |
| ipv4-05-up | 2026-11-01T12:26:36Z | 0 | 23.988 |
| ipv4-05-native | 2026-11-01T12:27:00Z | 0 | 77.25 |
| ipv4-05-down | 2026-11-01T12:28:17Z | 0 | 22.615 |
| ipv4-06-up | 2026-11-01T12:28:40Z | 0 | 26.438 |
| ipv4-06-native | 2026-11-01T12:29:06Z | 0 | 88.056 |
| ipv4-06-down | 2026-11-01T12:30:34Z | 0 | 22.605 |
| ipv4-07-up | 2026-11-01T12:30:57Z | 0 | 31.754 |
| ipv4-07-native | 2026-11-01T12:31:29Z | 0 | 87.901 |
| ipv4-07-down | 2026-11-01T12:32:57Z | 0 | 22.508 |
| ipv4-08-up | 2026-11-01T12:33:19Z | 0 | 31.708 |
| ipv4-08-native | 2026-11-01T12:33:51Z | 0 | 86.209 |
| ipv4-08-down | 2026-11-01T12:35:17Z | 0 | 22.461 |
| ipv6-01-up | 2026-11-01T12:35:40Z | 0 | 24.751 |
| ipv6-01-native | 2026-11-01T12:36:05Z | 0 | 86.209 |
| ipv6-01-down | 2026-11-01T12:37:31Z | 0 | 22.622 |
| ipv6-02-up | 2026-11-01T12:37:53Z | 0 | 23.578 |
| ipv6-02-native | 2026-11-01T12:38:17Z | 0 | 92.264 |
| ipv6-02-down | 2026-11-01T12:39:49Z | 0 | 22.517 |
| ipv6-03-up | 2026-11-01T12:40:12Z | 0 | 23.859 |
| ipv6-03-native | 2026-11-01T12:40:36Z | 0 | 71.725 |
| ipv6-03-down | 2026-11-01T12:41:48Z | 0 | 22.487 |
| ipv6-04-up | 2026-11-01T12:42:10Z | 0 | 29.093 |
| ipv6-04-native | 2026-11-01T12:42:39Z | 0 | 88.251 |
| ipv6-04-down | 2026-11-01T12:44:07Z | 0 | 22.61 |
| ipv6-05-up | 2026-11-01T12:44:30Z | 0 | 37.028 |
| ipv6-05-native | 2026-11-01T12:45:07Z | 0 | 86.316 |
| ipv6-05-down | 2026-11-01T12:46:34Z | 0 | 22.365 |
| ipv6-06-up | 2026-11-01T12:46:56Z | 0 | 31.393 |
| ipv6-06-native | 2026-11-01T12:47:27Z | 1 | 71.263 |
| ipv6-06-down | 2026-11-01T12:48:39Z | 0 | 22.782 |
| ipv6-07-up | 2026-11-01T12:49:02Z | 0 | 24.441 |
| ipv6-07-native | 2026-11-01T12:49:26Z | 1 | 92.436 |
| ipv6-07-down | 2026-11-01T12:50:58Z | 0 | 22.408 |
| ipv6-08-up | 2026-11-01T12:51:21Z | 0 | 39.206 |
| ipv6-08-native | 2026-11-01T12:52:00Z | 0 | 87.222 |
| ipv6-08-down | 2026-11-01T12:53:27Z | 0 | 23.015 |
| corrected-ipv4-01-up | 2026-11-01T12:53:50Z | 0 | 24.021 |
| corrected-ipv4-01-native | 2026-11-01T12:54:15Z | 0 | 92.084 |
| corrected-ipv4-01-down | 2026-11-01T12:55:47Z | 0 | 22.47 |
| corrected-ipv4-02-up | 2026-11-01T12:56:09Z | 0 | 29.714 |
| corrected-ipv4-02-native | 2026-11-01T12:56:39Z | 0 | 87.14 |
| corrected-ipv4-02-down | 2026-11-01T12:58:06Z | 0 | 22.247 |
| corrected-ipv6-01-up | 2026-11-01T12:58:28Z | 0 | 22.637 |
| corrected-ipv6-01-native | 2026-11-01T12:58:51Z | 0 | 92.899 |
| corrected-ipv6-01-down | 2026-11-01T13:00:24Z | 0 | 22.751 |
| corrected-ipv6-02-up | 2026-11-01T13:00:47Z | 0 | 34.405 |
| corrected-ipv6-02-native | 2026-11-01T13:01:21Z | 0 | 87.226 |
| corrected-ipv6-02-down | 2026-11-01T13:02:48Z | 0 | 22.467 |
| ipv4-09-up | 2026-11-01T13:03:11Z | 0 | 22.674 |
| ipv4-09-native | 2026-11-01T13:03:34Z | 0 | 56.664 |
| container-stop-final | 2026-11-01T13:04:30Z | 0 | 17.68 |
| focused-final | 2026-11-01T13:04:48Z | 0 | 0.464 |
| ipv4-09-down | 2026-11-01T13:04:49Z | 0 | 22.097 |
| trace-hold-ipv4-up | 2026-11-01T13:06:19Z | 0 | 27.07 |
| trace-hold-ipv4-native | 2026-11-01T13:06:47Z | 1 | 76.858 |
| trace-hold-ipv4-down | 2026-11-01T13:08:03Z | 0 | 22.271 |
| trace-hold-ipv6-up | 2026-11-01T13:08:26Z | 0 | 22.902 |
| trace-hold-ipv6-native | 2026-11-01T13:08:49Z | 1 | 61.187 |
| container-stop-fault-final | 2026-11-01T13:09:50Z | 0 | 17.808 |
| focused-fault-final | 2026-11-01T13:10:08Z | 0 | 0.398 |
| trace-hold-ipv6-down | 2026-11-01T13:10:08Z | 0 | 22.134 |
| e2e-lint | 2026-11-01T13:11:56Z | 0 | 2.64 |
| e2e-test | 2026-11-01T13:11:58Z | 0 | 6.877 |
| e2e-tagged-test | 2026-11-01T13:12:05Z | 0 | 5.77 |
| committed-evidence | 2026-11-01T13:12:11Z | 0 | 1.822 |
| handover | 2026-11-01T13:12:13Z | 0 | 1.063 |
| go-lint | 2026-11-01T13:12:14Z | 0 | 1.012 |
| build-amd64 | 2026-11-01T13:12:15Z | 0 | 0.802 |
| build-arm64 | 2026-11-01T13:12:16Z | 0 | 0.787 |
| diff-check | 2026-11-01T13:12:17Z | 0 | 0.005 |

## Real output tails

### ipv4-01-native

Command: `env FVPN_E2E_FOLLOWUP26=1 FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup26Native$'`

Exit 1.

```text
# github.com/TED-H-com/Freedom-VPN/tests/e2e [github.com/TED-H-com/Freedom-VPN/tests/e2e.test]
tests/e2e/followup18_native_test.go:640:6: undefined: e
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e [build failed]
FAIL
```

### ipv4-02-native

Command: `env FVPN_E2E_FOLLOWUP26=1 FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 FVPN_E2E_REPRODUCE_TRACE26=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup26Native$'`

Exit 1.

```text
# github.com/TED-H-com/Freedom-VPN/tests/e2e [github.com/TED-H-com/Freedom-VPN/tests/e2e.test]
tests/e2e/kernel_observer_test.go:33:27: undefined: enforce.ReasonCode
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e [build failed]
FAIL
```

### ipv6-06-native

Command: `env FVPN_E2E_FOLLOWUP26=1 FVPN_E2E_FOLLOWUP26_FAMILY=ipv6 FVPN_E2E_REPRODUCE_TRACE26=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup26Native$'`

Exit 1.

```text
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario post-capture-recovery: pass
    scenarios_test.go:120: scenario regrant-session: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=10 blocked=0 fail=3 run=ce8ccbe1-ed7d-4fc0-b1ed-786ef0e7578c source=487f4fe40de7fea0a47842fba3f168bed188df97
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup26Native (70.87s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	70.885s
FAIL
```

### ipv6-07-native

Command: `env FVPN_E2E_FOLLOWUP26=1 FVPN_E2E_FOLLOWUP26_FAMILY=ipv6 FVPN_E2E_REPRODUCE_TRACE26=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup26Native$'`

Exit 1.

```text
    scenarios_test.go:120: scenario post-capture-recovery: pass
    scenarios_test.go:120: scenario regrant-serving: pass
    scenarios_test.go:120: scenario regrant-session-ready: pass
    scenarios_test.go:120: scenario regrant-session: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=14 blocked=0 fail=1 run=af718646-166a-4a3f-af55-3f7a3ee8e17b source=487f4fe40de7fea0a47842fba3f168bed188df97
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup26Native (91.99s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	92.003s
FAIL
```

### trace-hold-ipv4-native

Command: `env FVPN_E2E_FOLLOWUP26=1 FVPN_E2E_TRACE_HOLD26=1 FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 FVPN_E2E_REPRODUCE_TRACE26=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup26Native$'`

Exit 1.

```text
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario post-capture-recovery: pass
    scenarios_test.go:120: scenario regrant-session: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=10 blocked=0 fail=3 run=86153543-1251-48f4-b368-cf11e5b0939d source=4d7163379f80ab0630a78353c7ab10de24387c32
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup26Native (76.40s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	76.413s
FAIL
```

### trace-hold-ipv6-native

Command: `env FVPN_E2E_FOLLOWUP26=1 FVPN_E2E_TRACE_HOLD26=1 FVPN_E2E_FOLLOWUP26_FAMILY=ipv6 FVPN_E2E_REPRODUCE_TRACE26=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup26Native$'`

Exit 1.

```text
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario post-capture-recovery: pass
    scenarios_test.go:120: scenario regrant-session: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=10 blocked=0 fail=3 run=a3154028-4296-4a61-afba-fee20fc0aa5f source=4d7163379f80ab0630a78353c7ab10de24387c32
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup26Native (60.82s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	60.825s
FAIL
```

### container-stop-fault-final

Command: `env FVPN_E2E_STOP26=1 go test -tags e2e -count=1 -v -timeout 5m ./tests/e2e -run '^TestE2E_ContainerStop26$'`

Exit 0.

```text
=== RUN   TestE2E_ContainerStop26
    scenarios_test.go:120: scenario container-stop-graceful: pass
    scenarios_test.go:120: scenario container-stop-forced: pass
    scenarios_test.go:153: acceptance summary: pass=2 blocked=0 fail=0 run=2cd90524-b768-4abc-8314-b9cded71a9ad source=4d7163379f80ab0630a78353c7ab10de24387c32
--- PASS: TestE2E_ContainerStop26 (17.37s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	17.381s
```

### focused-fault-final

Command: `go test -tags e2e -count=1 -v ./tests/e2e ./tests/e2e/internal/counterread -run 'TestAT01_HistoricalNodeTrace|TestAT04_'`

Exit 0.

```text
    --- PASS: TestAT04_ParseFailureEvidence/interleaved (0.00s)
    --- PASS: TestAT04_ParseFailureEvidence/unexpected (0.00s)
    --- PASS: TestAT04_ParseFailureEvidence/semantic (0.00s)
    --- PASS: TestAT04_ParseFailureEvidence/syntax (0.01s)
    --- PASS: TestAT04_ParseFailureEvidence/duplicate_exact (0.00s)
    --- PASS: TestAT04_ParseFailureEvidence/duplicate_different (0.00s)
=== RUN   TestAT04_ParseRetentionFailureAndLines
--- PASS: TestAT04_ParseRetentionFailureAndLines (0.00s)
=== RUN   TestAT04_WrappedDuplicateClassification
--- PASS: TestAT04_WrappedDuplicateClassification (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.045s
```

### trace-hold-ipv6-down

Command: `make -f mk/e2e.mk e2e-down`

Exit 0.

```text
 Container fvpn-e2e-netbird Removed
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### e2e-lint

Command: `make -f mk/e2e.mk e2e-lint`

Exit 0.

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

### e2e-test

Command: `go test ./tests/e2e/...`

Exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	0.015s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.117s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.027s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.068s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.103s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

### e2e-tagged-test

Command: `go test -tags e2e ./tests/e2e/...`

Exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.121s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.022s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.065s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.103s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack	0.008s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

### committed-evidence

Command: `go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

Exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	1.671s
```

### handover

Command: `go test -count=1 ./tests/handover/...`

Exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.814s
```

### go-lint

Command: `make go-lint`

Exit 0.

```text
(no stdout)
```

### build-amd64

Command: `env GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...`

Exit 0.

```text
(no stdout)
```

### build-arm64

Command: `env GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...`

Exit 0.

```text
(no stdout)
```

### diff-check

Command: `git diff --check`

Exit 0.

```text
(no stdout)
```

## Supplemental checks

`env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/e2e -p test_followup26_analysis.py`: exit 0.

```text
...
----------------------------------------------------------------------
Ran 3 tests in 0.000s

OK
```

`sh -n deploy/devstack/agent-service.sh deploy/devstack/entrypoint-node.sh`: exit 0.

```text
(no stdout)
```

