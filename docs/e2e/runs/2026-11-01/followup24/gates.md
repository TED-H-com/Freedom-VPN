# Follow-up 24 gates — 2026-11-01

All live attempts are retained. The first stopped at trace installation; the IPv4 five-SYN timeout remains failed. Final offline gates pass.

| Gate | Exit | Seconds |
|---|---:|---:|
| focused-1 | 0 | 1.16 |
| up-1 | 0 | 31.24 |
| focused-post-up | 0 | 0.41 |
| native-1 | 1 | 43.14 |
| focused-2 | 0 | 1.19 |
| down-1 | 0 | 41.77 |
| up-2 | 0 | 30.05 |
| focused-post-up-2 | 0 | 1.19 |
| reader-build | 0 | 0.45 |
| native-2 | 0 | 45.82 |
| down-2 | 0 | 41.85 |
| python-trace | 0 | 0.04 |
| interim-hygiene | 1 | 0.57 |
| interim-hygiene-2 | 0 | 0.54 |
| up-3 | 0 | 32.54 |
| focused-post-up-3 | 0 | 1.22 |
| native-3 | 1 | 59.15 |
| down-3 | 0 | 41.73 |
| up-4 | 0 | 31.01 |
| focused-post-up-4 | 0 | 1.19 |
| native-4 | 0 | 54.16 |
| down-4 | 0 | 41.76 |
| up-5 | 0 | 31.07 |
| focused-post-up-5 | 0 | 0.4 |
| native-5 | 0 | 47.78 |
| down-5 | 0 | 41.73 |
| final-lint | 0 | 3.01 |
| final-unit | 0 | 3.33 |
| final-tagged | 0 | 4.25 |
| final-hygiene | 0 | 0.49 |
| final-handover | 0 | 0.94 |
| final-go-lint | 0 | 1.14 |
| final-build-amd64 | 0 | 1.16 |
| final-build-arm64 | 0 | 1.18 |
| final-diff | 0 | 0.01 |
| final-python | 0 | 0.04 |

The first focused offline check overlapped startup; focused-post-up repeats it after startup completed. Initial hygiene rejected the certified empty origin capture from the pre-selection failure; zero records now require its original stopped/zero-drop/24-byte-pcap receipt. No live result was replaced.

Foreground up/test/down runs were sequential. Capacity was checked before and after startup: at most one other suite was observed. No third integration suite started. All raw data and Go scratch are on disk under `.agent/tmp`; `GOCACHE` came from `go env GOCACHE`. Root lint used `go-with-disk-tmp=` with that disk environment, avoiding shell cleanup traps.

Final owned inventory: **0 containers, 0 networks, 0 volumes, 0 images**. All **154** follow-up 23 pins still verify. 41 private capture/trace/debug receipts verify exactly and have mode 0600. No physical arm64 run; both tagged cross-builds pass.

Real output tails below; the worktree path in make output is replaced by `[worktree]` and trailing terminal whitespace is trimmed.

## native-1

```text
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=6 blocked=0 fail=2 run=95b1285f-cf5b-456a-843b-be4fe1849d2c source=54c50d033ddd05474a0f762b152cc97c81b42511
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup24Native (42.76s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	42.769s
FAIL
```

## native-2

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=8 blocked=0 fail=0 run=e201f325-d0ef-4bbe-8d91-16cdab9ab8bc source=711755de88ea936ab968d2ebf90681a8145e2e1d
--- PASS: TestE2E_Followup24Native (45.46s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	45.465s
```

## native-3

```text
    scenarios_test.go:120: scenario fixture-member-admission: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=6 blocked=0 fail=2 run=d556673b-8164-4b5a-92c4-9333049097e3 source=ac5fec110763fbc6a0f5e3f8701a0bc7d04dd3da
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup24Native (58.73s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	58.740s
FAIL
```

## native-4

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=9 blocked=0 fail=0 run=cacad79c-d64b-4692-8b08-290df23e7a6a source=2eba2bc81f59382c6e1fae2e50befebf847bd9bc
--- PASS: TestE2E_Followup24Native (53.80s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	53.813s
```

## native-5

```text
    scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:153: acceptance summary: pass=9 blocked=0 fail=0 run=951bd882-8106-4514-a36a-a1d82ca71c76 source=2eba2bc81f59382c6e1fae2e50befebf847bd9bc
--- PASS: TestE2E_Followup24Native (47.41s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	47.419s
```

## down-5

```text
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

## final-lint

```text
cd [worktree] && go vet ./tests/e2e/...
cd [worktree] && go vet -tags e2e ./tests/e2e/...
cd [worktree] && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd [worktree] && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

## final-unit

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

## final-tagged

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack	0.007s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

## final-hygiene

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.392s
```

## final-handover

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.686s
```

## final-go-lint

```text
(no output)
```

## final-build-amd64

```text
(no output)
```

## final-build-arm64

```text
(no output)
```

## final-diff

```text
(no output)
```

## final-python

```text
..........
----------------------------------------------------------------------
Ran 10 tests in 0.002s

OK
```
