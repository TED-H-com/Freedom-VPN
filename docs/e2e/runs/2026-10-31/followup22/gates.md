# Follow-up 22 gate receipts

All commands ran in the foreground. Go uses disk TMPDIR/GOTMPDIR under
`.agent/tmp/followup22/scratch`, CGO_ENABLED=0, GOTOOLCHAIN=local,
GOMAXPROCS=4, GOFLAGS=-p=4 and GOCACHE from `go env GOCACHE`.
`go-with-disk-tmp=` bypasses the root Makefile's shell cleanup trap; the
existing disk environment supplies scratch, cleaned with Python after gates.
No dependencies changed. No pytest, tmpfs scratch or host networking used.

Final live source is `a1796b0d`; optional post-run WireGuard decoder is
`bf82cbe4`. The acceptance command deliberately exits **1** with **13/0/2**,
retaining the first-selection failure and asymmetric IPv6 failure. Offline
tagged tests skip live opt-in runners; their exit 0 is not live acceptance.

| Gate | Exit | Seconds | Command |
|---|---:|---:|---|
| up-ready | 0 | 38.13 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| focused-unit-final | 0 | 1.12 | `go test -tags e2e -count=1 -v -run ^TestAT01_ ./tests/e2e ./tests/e2e/internal/nativecapture` |
| live-final | 1 | 117.14 | `env FVPN_E2E_FOLLOWUP22=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run ^TestE2E_Followup22Native$` |
| down-final | 0 | 42.0 | `make -f mk/e2e.mk e2e-down` |
| lint-final | 0 | 1.66 | `make -f mk/e2e.mk e2e-lint` |
| unit-final | 0 | 3.12 | `go test ./tests/e2e/...` |
| tagged-final | 0 | 4.16 | `go test -tags e2e ./tests/e2e/...` |
| hygiene-final | 0 | 0.35 | `go test -count=1 ./tests/e2e -run ^TestCommittedFixtureEvidence_ValidAndHygieneClean$` |
| handover-final | 0 | 0.98 | `go test -count=1 ./tests/handover/...` |
| go-lint-final | 0 | 1.56 | `make go-lint go-with-disk-tmp=` |
| build-amd64-final | 0 | 1.13 | `env GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...` |
| build-arm64-final | 0 | 1.12 | `env GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...` |
| diff-final | 0 | 0.0 | `git diff --check` |
| python-final | 0 | 0.04 | `python3 -B -m unittest discover -s tests/e2e -p test_*capture.py` |

Additional first campaign: up 0 (31.06 s), focused tests 0 (1.13 s), live
**1** (102.54 s; 12/0/1), down 0 (42.10 s). An attempted second startup
was terminated with SIGTERM (recorded exit -15, 18.32 s), during image build,
when two other AT suites were observed. No live runner started in that attempt.
The corrected capacity guard waited, used a visible `gates.sh` marker, checked
`pgrep -af 'gates.sh|mk/at-'` immediately before up and again after up, counting
standalone AT make targets and excluding its own ancestry. Final startup/live
proceeded with only one other suite active. No three live suites were launched.

## Actual output tails

### up-ready — exit 0

```text
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### focused-unit-final — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.007s
=== RUN   TestAT01_SelectedPeerDoesNotLeakOtherKeys
--- PASS: TestAT01_SelectedPeerDoesNotLeakOtherKeys (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
```

### live-final — exit 1

```text
    scenarios_test.go:120: scenario ipv6-explicit-repeat1: pass
    scenarios_test.go:120: scenario ipv6-explicit-repeat2: pass
    scenarios_test.go:153: acceptance summary: pass=13 blocked=0 fail=2 run=9b3a89e7-fa86-426c-9737-ce55bf66aa77 source=a1796b0d2be7f0ef068f66347fccb11d55ed58a4
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup22Native (116.72s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	116.724s
FAIL
```

### down-final — exit 0

```text
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Network fvpn-e2e-origin Removed
 Network fvpn-e2e Removed
e2e stack is down
```

### lint-final — exit 0

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

### unit-final — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.103s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

### tagged-final — exit 0

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

### hygiene-final — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.238s
```

### handover-final — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.541s
```

### go-lint-final — exit 0

```text
(no output)
```

### build-amd64-final — exit 0

```text
(no output)
```

### build-arm64-final — exit 0

```text
(no output)
```

### diff-final — exit 0

```text
(no output)
```

### python-final — exit 0

```text
......
----------------------------------------------------------------------
Ran 6 tests in 0.000s

OK
```

## Provenance and cleanup

- 63 follow-up 21 SHA pins verify; each matches its byte-identical private copy.
- 48 raw pcaps across both campaigns match size/SHA receipts and mode 0600.
- All 27 final captures have exact decoded/captured counts, zero kernel drops,
  zero file/header truncations and zero decode errors.
- Public follow-up 22 pins target only immutable UUID run subtrees, never mutable
  README, response, report or gate-ledger files (D-132).
- Exact harness ownership filter reports **0 containers, 0 networks, 0 volumes,
  0 images** after down. Shared PostgreSQL and other lanes are untouched.
- First campaign artifacts, members 11, devstack 20 and all older evidence
  remain byte-identical. Only owned paths changed; no recipient AT file changed.
