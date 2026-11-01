# Follow-up 25 gate results

All commands ran in the foreground. Integration attempts were sequential. Admission checked `pgrep -af 'gates.sh|mk/at-'` before startup and again after it; the lane waited while two other suite trees were present. Read-only checks did not require admission. No shared PostgreSQL or another lane’s Docker objects were modified.

Go **1.27.3**, `GOTOOLCHAIN=local`, `CGO_ENABLED=0`, `GOMAXPROCS=4`, `GOFLAGS=-p=4`. Disk `TMPDIR`/`GOTMPDIR` under `.agent/tmp`; `GOCACHE` came from `go env GOCACHE`. The root lint command used `go-with-disk-tmp=` with that disk environment, avoiding its shell cleanup trap. No new shell cleanup trap was written.

**Final verification gates pass.** The 32 original native requests retain **26 passes and 6 timeouts**, including the nonzero Go-test exits below. These are observed failures, not rewritten as successful gates. Live exit readback passes both real exit 0 and explicit SIGKILL; its repeated-stop reads remain unchanged.

| Gate | Exit | Wall seconds |
|---|---:|---:|
| kernel-ipv4-01-up | 0 | 31.15 |
| focused | 0 | 1.19 |
| kernel-ipv4-01-native | 1 | 59.55 |
| agent-exit-live | 0 | 3.3 |
| kernel-ipv4-01-down | 0 | 42.0 |
| kernel-ipv4-01-analysis | 0 | 0.07 |
| kernel-ipv4-02-up | 0 | 30.14 |
| kernel-ipv4-02-native | 0 | 54.07 |
| kernel-ipv4-02-down | 0 | 41.88 |
| kernel-ipv4-02-analysis | 0 | 0.05 |
| kernel-ipv4-03-up | 0 | 30.47 |
| kernel-ipv4-03-native | 0 | 38.3 |
| kernel-ipv4-03-down | 0 | 41.9 |
| kernel-ipv4-03-analysis | 0 | 0.05 |
| kernel-ipv4-04-up | 0 | 30.65 |
| kernel-ipv4-04-native | 0 | 53.79 |
| kernel-ipv4-04-down | 0 | 41.82 |
| kernel-ipv4-04-analysis | 0 | 0.05 |
| kernel-ipv4-05-up | 0 | 23.12 |
| kernel-ipv4-05-native | 0 | 53.76 |
| kernel-ipv4-05-down | 0 | 45.42 |
| kernel-ipv4-05-analysis | 0 | 0.05 |
| kernel-ipv4-06-up | 0 | 23.94 |
| kernel-ipv4-06-native | 0 | 53.85 |
| kernel-ipv4-06-down | 0 | 41.89 |
| kernel-ipv4-06-analysis | 0 | 0.05 |
| kernel-ipv4-07-up | 0 | 22.94 |
| kernel-ipv4-07-native | 0 | 54.11 |
| kernel-ipv4-07-down | 0 | 42.37 |
| kernel-ipv4-07-analysis | 0 | 0.05 |
| kernel-ipv4-08-up | 0 | 26.08 |
| kernel-ipv4-08-native | 0 | 53.92 |
| kernel-ipv4-08-down | 0 | 41.86 |
| kernel-ipv4-08-analysis | 0 | 0.05 |
| kernel-ipv6-01-up | 0 | 22.58 |
| kernel-ipv6-01-native | 0 | 53.53 |
| kernel-ipv6-01-down | 0 | 41.8 |
| kernel-ipv6-01-analysis | 0 | 0.05 |
| kernel-ipv6-02-up | 0 | 22.78 |
| kernel-ipv6-02-native | 1 | 44.47 |
| kernel-ipv6-02-down | 0 | 41.78 |
| kernel-ipv6-02-analysis | 0 | 0.07 |
| kernel-ipv6-03-up | 0 | 31.43 |
| kernel-ipv6-03-native | 0 | 54.77 |
| kernel-ipv6-03-down | 0 | 41.75 |
| kernel-ipv6-03-analysis | 0 | 0.05 |
| kernel-ipv6-04-up | 0 | 22.42 |
| kernel-ipv6-04-native | 0 | 53.59 |
| kernel-ipv6-04-down | 0 | 42.03 |
| kernel-ipv6-04-analysis | 0 | 0.05 |
| kernel-ipv6-05-up | 0 | 26.18 |
| kernel-ipv6-05-native | 0 | 54.09 |
| kernel-ipv6-05-down | 0 | 41.89 |
| kernel-ipv6-05-analysis | 0 | 0.05 |
| kernel-ipv6-06-up | 0 | 22.96 |
| kernel-ipv6-06-native | 0 | 54.06 |
| kernel-ipv6-06-down | 0 | 41.93 |
| kernel-ipv6-06-analysis | 0 | 0.05 |
| kernel-ipv6-07-up | 0 | 23.79 |
| kernel-ipv6-07-native | 0 | 54.72 |
| kernel-ipv6-07-down | 0 | 41.89 |
| kernel-ipv6-07-analysis | 0 | 0.05 |
| kernel-ipv6-08-up | 0 | 22.78 |
| kernel-ipv6-08-native | 0 | 54.16 |
| kernel-ipv6-08-down | 0 | 41.97 |
| kernel-ipv6-08-analysis | 0 | 0.05 |
| userspace-ipv4-01-up | 0 | 23.34 |
| userspace-ipv4-01-native | 0 | 53.78 |
| userspace-ipv4-01-down | 0 | 41.93 |
| userspace-ipv4-01-analysis | 0 | 0.05 |
| userspace-ipv4-02-up | 0 | 23.0 |
| userspace-ipv4-02-native | 0 | 38.78 |
| userspace-ipv4-02-down | 0 | 41.73 |
| userspace-ipv4-02-analysis | 0 | 0.05 |
| userspace-ipv4-03-up | 0 | 23.37 |
| userspace-ipv4-03-native | 0 | 53.82 |
| userspace-ipv4-03-down | 0 | 41.72 |
| userspace-ipv4-03-analysis | 0 | 0.05 |
| userspace-ipv4-04-up | 0 | 23.35 |
| userspace-ipv4-04-native | 1 | 45.63 |
| userspace-ipv4-04-down | 0 | 42.27 |
| userspace-ipv4-04-analysis | 0 | 0.06 |
| userspace-ipv4-05-up | 0 | 30.76 |
| userspace-ipv4-05-native | 0 | 54.12 |
| userspace-ipv4-05-down | 0 | 41.78 |
| userspace-ipv4-05-analysis | 0 | 0.05 |
| userspace-ipv4-06-up | 0 | 22.63 |
| userspace-ipv4-06-native | 0 | 53.78 |
| userspace-ipv4-06-down | 0 | 41.97 |
| userspace-ipv4-06-analysis | 0 | 0.05 |
| userspace-ipv4-07-up | 0 | 23.58 |
| userspace-ipv4-07-native | 0 | 54.5 |
| userspace-ipv4-07-down | 0 | 42.04 |
| userspace-ipv4-07-analysis | 0 | 0.05 |
| userspace-ipv4-08-up | 0 | 23.92 |
| userspace-ipv4-08-native | 0 | 38.96 |
| userspace-ipv4-08-down | 0 | 41.96 |
| userspace-ipv4-08-analysis | 0 | 0.05 |
| userspace-ipv6-01-up | 0 | 22.55 |
| userspace-ipv6-01-native | 1 | 44.54 |
| userspace-ipv6-01-down | 0 | 42.84 |
| userspace-ipv6-01-analysis | 0 | 0.08 |
| userspace-ipv6-02-up | 0 | 24.17 |
| userspace-ipv6-02-native | 0 | 54.03 |
| userspace-ipv6-02-down | 0 | 42.52 |
| userspace-ipv6-02-analysis | 0 | 0.05 |
| userspace-ipv6-03-up | 0 | 26.95 |
| userspace-ipv6-03-native | 1 | 59.57 |
| userspace-ipv6-03-down | 0 | 42.47 |
| userspace-ipv6-03-analysis | 0 | 0.08 |
| userspace-ipv6-04-up | 0 | 33.68 |
| userspace-ipv6-04-native | 0 | 53.95 |
| userspace-ipv6-04-down | 0 | 41.97 |
| userspace-ipv6-04-analysis | 0 | 0.05 |
| userspace-ipv6-05-up | 0 | 22.47 |
| userspace-ipv6-05-native | 1 | 58.79 |
| userspace-ipv6-05-down | 0 | 42.13 |
| userspace-ipv6-05-analysis | 0 | 0.07 |
| userspace-ipv6-06-up | 0 | 24.68 |
| userspace-ipv6-06-native | 0 | 53.93 |
| userspace-ipv6-06-down | 0 | 43.03 |
| userspace-ipv6-06-analysis | 0 | 0.2 |
| userspace-ipv6-07-up | 0 | 24.91 |
| userspace-ipv6-07-native | 0 | 54.02 |
| userspace-ipv6-07-down | 0 | 42.15 |
| userspace-ipv6-07-analysis | 0 | 0.05 |
| userspace-ipv6-08-up | 0 | 25.79 |
| userspace-ipv6-08-native | 0 | 54.21 |
| userspace-ipv6-08-down | 0 | 41.92 |
| userspace-ipv6-08-analysis | 0 | 0.05 |
| final-lint | 0 | 3.14 |
| final-unit | 0 | 4.38 |
| final-tagged | 0 | 6.16 |
| final-hygiene | 0 | 0.96 |
| final-handover | 0 | 1.27 |
| final-go-lint | 0 | 1.58 |
| final-build-amd64 | 0 | 1.14 |
| final-build-arm64 | 0 | 1.34 |
| final-diff | 0 | 0.01 |
| final-python | 0 | 0.05 |
| final-shell | 0 | 0.0 |
| additional handshake-focused unit check | 0 | not separately timed |

The additional focused check ran, while the owned stack was up, as `go test -tags e2e -count=1 ./tests/e2e/internal/nativecapture -run 'TestF1101_NativeHandshakeRemainsSeparate|TestAT01_SelectedPeer'`. It precedes teardown and never launches another integration suite.

## Real output tails

Absolute worktree paths in the lint output are replaced with `[worktree]` for public hygiene. Other lines below are the original output tails.

### focused

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.019s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.007s
```

### handshake-focused

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
```

### agent-exit-live

```text
=== RUN   TestE2E_AgentExitReadback
    scenarios_test.go:120: scenario agent-exit-graceful: pass
    scenarios_test.go:120: scenario agent-exit-sigkill: pass
    scenarios_test.go:153: acceptance summary: pass=2 blocked=0 fail=0 run=d513a413-4597-423c-9dd3-7c0669f03eca source=675600e13d6a0f9cbb4028f51f3fa1179b1cc513
--- PASS: TestE2E_AgentExitReadback (2.89s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	2.896s
```

### kernel-ipv4-01-native

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=7 blocked=0 fail=2 run=48dbf84e-00e5-4250-84d2-0a36b074ed1f source=ce90de48de0e3f6f8e42e7310d466033f70367d1
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup25Native (59.16s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	59.174s
FAIL
```

### kernel-ipv6-02-native

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=7 blocked=0 fail=2 run=953565d0-2138-4d16-b27a-1b38553ddb36 source=9c53e9ec50ae883f0c7788a3bae1284ad98df9ae
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup25Native (44.09s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	44.096s
FAIL
```

### userspace-ipv4-04-native

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=7 blocked=0 fail=2 run=98e0e91e-c32e-4f22-a422-16ba6fe014a2 source=6b075f9904315b890c5510f75889a0ef099028b7
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup25Native (44.44s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	44.451s
FAIL
```

### userspace-ipv6-01-native

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=7 blocked=0 fail=2 run=2cf62be6-7684-449d-8f8f-814da0f64cd6 source=d06bdb489b123093040061eb2a0e9c2e4f65f024
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup25Native (44.18s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	44.185s
FAIL
```

### userspace-ipv6-03-native

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=7 blocked=0 fail=2 run=1d253f8d-f771-4edc-af93-f4220df54877 source=95bb231bca7b35f86cb41dd9e8264d83bc3d4665
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup25Native (59.17s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	59.181s
FAIL
```

### userspace-ipv6-05-native

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: fail
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=7 blocked=0 fail=2 run=0c27cac0-ec38-4328-bb11-be2d3545ac04 source=95bb231bca7b35f86cb41dd9e8264d83bc3d4665
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup25Native (58.40s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	58.405s
FAIL
```

### final-lint

```text
cd [worktree] && go vet ./tests/e2e/...
cd [worktree] && go vet -tags e2e ./tests/e2e/...
cd [worktree] && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd [worktree] && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

### final-unit

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.031s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.068s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

### final-tagged

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.072s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

### final-hygiene

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.843s
```

### final-handover

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.754s
```

### final-go-lint

```text
(no output)
```

### final-build-amd64

```text
(no output)
```

### final-build-arm64

```text
(no output)
```

### final-diff

```text
(no output)
```

### final-python

```text
.............
----------------------------------------------------------------------
Ran 13 tests in 0.001s

OK
```

### final-shell

```text
(no output)
```

## Integrity and cleanup

All **288** private capture/trace/debug receipts match exact size, SHA-256 and mode 0600. The separately retained sampler binary matches its public receipt, for **289** total private receipt checks. All **122** follow-up 24 files match the initial complete-tree snapshot byte for byte. All new pins refer only to this immutable archive.

Final prefix-owned Docker inventory: **0 containers, 0 networks, 0 volumes, 0 image tags**. Shared neutral build cache and other lanes are outside this ownership check. No push, PR, merge, tag or main rewrite.
