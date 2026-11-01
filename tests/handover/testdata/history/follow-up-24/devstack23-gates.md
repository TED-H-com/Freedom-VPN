# Follow-up 23 gates — 2026-11-01

Commands ran in the foreground, sequentially; disk TMPDIR/GOTMPDIR, GOTOOLCHAIN=local, CGO_ENABLED=0, GOMAXPROCS=4 and GOFLAGS=-p=4. GOCACHE was read with `go env GOCACHE`. Root make’s shell cleanup wrapper was disabled with `go-with-disk-tmp=`; Python owns scratch cleanup. No new shell cleanup trap.

Before each stack start and after startup, `pgrep -af 'gates.sh|mk/at-'` was checked. Parent/wrapper and child make are one suite. Native start had capacity; DNS start waited for usage to finish while usage and members were active. No third integration suite was started. The final wrapper retains its visible gates.sh parent while its foreground command runs.

| Gate | Exit | Seconds | Command |
|---|---:|---:|---|
| up-native | 0 | 30.56 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| focused-native | 0 | 1.35 | `go test -tags e2e -count=1 -run ^TestAT01_ ./tests/e2e ./tests/e2e/internal/nativecapture` |
| native-live | 1 | 10.43 | `env FVPN_E2E_FOLLOWUP23=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run ^TestE2E_Followup23Native$` |
| focused-dns | 1 | 1.17 | `go test -tags e2e -count=1 -run TestAT20_|TestAT01_ ./tests/e2e ./tests/e2e/internal/dnsrelay` |
| focused-dns2 | 0 | 1.17 | `go test -tags e2e -count=1 -run TestAT20_|TestAT01_ ./tests/e2e ./tests/e2e/internal/dnsrelay` |
| native-live2 | 1 | 150.58 | `env FVPN_E2E_FOLLOWUP23=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run ^TestE2E_Followup23Native$` |
| down-native | 0 | 42.58 | `make -f mk/e2e.mk e2e-down` |
| hygiene-interim | 1 | 0.51 | `go test -count=1 ./tests/e2e -run ^TestCommittedFixtureEvidence_ValidAndHygieneClean$` |
| hygiene-interim2 | 0 | 0.52 | `go test -count=1 ./tests/e2e -run ^TestCommittedFixtureEvidence_ValidAndHygieneClean$` |
| focused-router | 0 | 1.12 | `go test -tags e2e -count=1 -run ^TestAT09_RouterSlots ./tests/e2e` |
| up-dns | 0 | 30.44 | `env FVPN_E2E_TWO_RESOLVERS=true make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| focused-dns-live-pre | 0 | 0.38 | `go test -tags e2e -count=1 -run TestAT20_|TestAT09_RouterSlots|TestAT01_ ./tests/e2e ./tests/e2e/internal/dnsrelay ./tests/e2e/internal/nativecapture` |
| dns-live | 1 | 75.71 | `env FVPN_E2E_FOLLOWUP23=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run ^TestE2E_Followup23DNS$` |
| focused-capture-final | 0 | 1.14 | `go test -tags e2e -count=1 -run ^TestAT01_ ./tests/e2e ./tests/e2e/internal/nativecapture` |
| down-dns | 0 | 42.68 | `make -f mk/e2e.mk e2e-down` |
| final-lint | 0 | 3.9 | `make -f mk/e2e.mk e2e-lint` |
| final-unit | 0 | 5.45 | `go test ./tests/e2e/...` |
| final-tagged | 0 | 5.76 | `go test -tags e2e ./tests/e2e/...` |
| final-hygiene | 0 | 0.54 | `go test -count=1 ./tests/e2e -run ^TestCommittedFixtureEvidence_ValidAndHygieneClean$` |
| final-handover | 0 | 1.28 | `go test -count=1 ./tests/handover/...` |
| final-go-lint | 0 | 3.74 | `make go-lint go-with-disk-tmp=` |
| final-build-amd64 | 0 | 1.14 | `env GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...` |
| final-build-arm64 | 0 | 1.43 | `env GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...` |
| final-diff | 0 | 0.0 | `git diff --check` |
| final-python | 0 | 0.04 | `python3 -B -m unittest discover -s tests/e2e -p test_*capture.py` |

The initial native-live attempt stopped before enrollment because the node daemon did not yet exist; no probe ran. native-live2 retains F11-01 and two black-hole failures. dns-live retains the singular-statistics diagnostic failure; its positive applications were not rerun. The same stopped capture’s readback supplies complete statistics without rewriting that verdict. All ordered final offline gates pass.

Actual tails below replace only the absolute worktree path with `<worktree>` for public hygiene; trailing terminal whitespace is trimmed.

## up-native

```text
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

## native-live

```text
=== RUN   TestE2E_Followup23Native
    followup23_live_test.go:25: exit status 1
--- FAIL: TestE2E_Followup23Native (10.07s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	10.082s
FAIL
```

## native-live2

```text
    scenarios_test.go:120: scenario router-ipv6: pass
    scenarios_test.go:120: scenario router-ipv4: pass
    scenarios_test.go:153: acceptance summary: pass=14 blocked=0 fail=3 run=a26d9b0f-bd1b-48df-9546-4105f9b3f328 source=c26303b57317c893ca169a87a662b0d0bd238daa
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup23Native (149.45s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	149.461s
FAIL
```

## down-native

```text
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e Removing
 Network fvpn-e2e Resource is still in use
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

## up-dns

```text
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

## dns-live

```text
    scenarios_test.go:120: scenario native-dns-config: pass
    scenarios_test.go:120: scenario native-dns-delivery: fail
    scenarios_test.go:153: acceptance summary: pass=11 blocked=0 fail=1 run=9bb2098e-1c6f-48a8-9fbd-fd99bd9fc49e source=1d396065c432485875add24a00761608afb293cc
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup23DNS (75.33s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	75.342s
FAIL
```

## focused-capture-final

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.007s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
```

## down-dns

```text
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Resource is still in use
 Network fvpn-e2e-origin Resource is still in use
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pg Removed
e2e stack is down
```

## final-lint

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

## final-unit

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

## final-tagged

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack	0.008s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

## final-hygiene

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.425s
```

## final-handover

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.707s
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
........
----------------------------------------------------------------------
Ran 8 tests in 0.001s

OK
```

Final artifact hygiene after analysis/provenance/gate documents: exit 0, 0.49 s.

```text
ok  github.com/TED-H-com/Freedom-VPN/tests/e2e 0.385s
```

The preliminary DNS unit failure caught reply-only digest fields being included
in the immutable request identity comparison; completion-field normalization
fixed it. The preliminary hygiene failure came from traversing pre-follow-up
2026-10-29 records under the widened dated-run root; the intended follow-up-only
scope was restored. Both corrected checks and the final gates pass.
