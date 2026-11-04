# Follow-up 34 checks

All commands ran in the foreground with disk TMPDIR/GOTMPDIR. Admission was separate from consumer output; see [admission summary](admission-summary.txt) and the disclosed recursive-make gap in [analysis](analysis.md).

| Check | Exit | Seconds |
|---|---:|---:|
| [01-e2e-lint](01-e2e-lint.txt) | 0 | 4.876 |
| [02-isolation-lint](02-isolation-lint.txt) | 0 | 4.429 |
| [03-e2e-vet](03-e2e-vet.txt) | 0 | 0.218 |
| [04-e2e-unit](04-e2e-unit.txt) | 0 | 24.909 |
| [05-acceptance](05-acceptance.txt) | 2 | 1414.126 |
| [06-e2e-down](06-e2e-down.txt) | 0 | 1608.737 |
| [07-handover](07-handover.txt) | 0 | 4.617 |
| [08-e2e-lint-after-list](08-e2e-lint-after-list.txt) | 0 | 3.135 |
| [09-e2e-unit-after-list](09-e2e-unit-after-list.txt) | 0 | 24.899 |
| [10-go-lint](10-go-lint.txt) | 0 | 4.514 |
| [11-gofmt](11-gofmt.txt) | 0 | 0.149 |
| [12-diff-check](12-diff-check.txt) | 0 | 0.006 |
| [13-diff-check-final](13-diff-check-final.txt) | 0 | 0.006 |
| [14-evidence-hygiene](14-evidence-hygiene.txt) | 0 | 8.71 |
| [15-staged-whitespace-before](15-staged-whitespace-before.txt) | 2 | 0.005 |
| [16-staged-whitespace-after](16-staged-whitespace-after.txt) | 0 | 0.005 |
| [nightly-after](nightly-after.txt) | 1 | 0.127 |
| [nightly-before](nightly-before.txt) | 1 | 0.223 |

## 01-e2e-lint

`make -f mk/e2e.mk e2e-lint`

UTC 2026-11-04T05:35:30.351094+00:00–2026-11-04T05:35:35.226740+00:00; exit 0.

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

## 02-isolation-lint

`make -f mk/at-isolation.mk at-isolation-lint`

UTC 2026-11-04T05:35:49.569053+00:00–2026-11-04T05:35:53.997890+00:00; exit 0.

```text
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

## 03-e2e-vet

`go vet -tags e2e ./tests/e2e/...`

UTC 2026-11-04T05:36:14.333256+00:00–2026-11-04T05:36:14.551156+00:00; exit 0.

```text
(no output)
```

## 04-e2e-unit

`go test -count=1 ./tests/e2e/...`

UTC 2026-11-04T05:36:19.741106+00:00–2026-11-04T05:36:44.649642+00:00; exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.066s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.006s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.003s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.003s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.001s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

## 05-acceptance

`FVPN_E2E_RUN_LABEL=followup34 make -f mk/e2e.mk e2e-test`

UTC 2026-11-04T05:37:34.607929+00:00–2026-11-04T06:01:08.734172+00:00; exit 2.

```text
    scenarios_test.go:133: scenario lease-expiry: pass
    scenarios_test.go:133: scenario revoke: pass
    scenarios_test.go:133: scenario second-exit: pass
    scenarios_test.go:166: acceptance summary: pass=31 blocked=2 fail=0 run=af4c130b-5029-401b-bd0f-e8afaad3d37f source=ab0899e6799a4d992c87215c5a6705e7e2fa506f
    scenarios_test.go:173: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Acceptance (1360.50s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	1379.184s
FAIL
make: *** [mk/e2e.mk:63: e2e-test] Error 1
```

## 06-e2e-down

`make -f mk/e2e.mk e2e-down`

UTC 2026-11-04T06:01:16.576926+00:00–2026-11-04T06:28:05.313850+00:00; exit 0.

```text
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Resource is still in use
 Network fvpn-e2e Resource is still in use
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pg Removed
e2e stack is down
```

## 07-handover

`go test -count=1 ./tests/handover/...`

UTC 2026-11-04T06:28:37.974372+00:00–2026-11-04T06:28:42.591188+00:00; exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.764s
```

## 08-e2e-lint-after-list

`make -f mk/e2e.mk e2e-lint`

UTC 2026-11-04T06:28:42.591473+00:00–2026-11-04T06:28:45.726829+00:00; exit 0.

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

## 09-e2e-unit-after-list

`go test -count=1 ./tests/e2e/...`

UTC 2026-11-04T06:28:45.727132+00:00–2026-11-04T06:29:10.626275+00:00; exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.059s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.001s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

## 10-go-lint

`make go-lint go-with-disk-tmp=`

UTC 2026-11-04T06:29:10.626566+00:00–2026-11-04T06:29:15.140287+00:00; exit 0.

```text
(no output)
```

## 11-gofmt

`bash -c 'test -z "$(gofmt -l $(git ls-files '"'"'*.go'"'"'))"'`

UTC 2026-11-04T06:29:15.140512+00:00–2026-11-04T06:29:15.289925+00:00; exit 0.

```text
(no output)
```

## 12-diff-check

`git diff --check`

UTC 2026-11-04T06:29:15.290210+00:00–2026-11-04T06:29:15.296691+00:00; exit 0.

```text
(no output)
```

## 13-diff-check-final

`git diff --check`

UTC 2026-11-04T06:31:30.360273+00:00–2026-11-04T06:31:30.366715+00:00; exit 0.

```text
(no output)
```

## 14-evidence-hygiene

`go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

UTC 2026-11-04T06:31:35.507506+00:00–2026-11-04T06:31:44.217193+00:00; exit 0.

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.445s
```

## 15-staged-whitespace-before

`git diff --cached --check`

UTC 2026-11-04T06:33:12.902044+00:00–2026-11-04T06:33:12.906761+00:00; exit 2.

```text
docs/e2e/runs/2026-11-04/followup34/nightly-gate-after.md:20: new blank line at EOF.
docs/e2e/runs/2026-11-04/followup34/nightly-gate-before.md:26: new blank line at EOF.
```

## 16-staged-whitespace-after

`git diff --cached --check`

UTC 2026-11-04T06:33:13.127101+00:00–2026-11-04T06:33:13.131738+00:00; exit 0.

```text
(no output)
```

## nightly-after

`go run ./tests/e2e/cmd/fvpn-e2e-nightly-gate -summary docs/e2e/runs/2026-11-04/followup34/acceptance-summary.md -test-exit-code 2 -test-log docs/e2e/runs/2026-11-04/followup34/05-acceptance.txt`

UTC 2026-11-04T06:28:37.846967+00:00–2026-11-04T06:28:37.974003+00:00; exit 1.

```text
| `member-ipv6` | **blocked** | w3-devstack-e2e / platform operators | Default native member container joins with --disable-ipv6 (DisableIPv6: true) and no IPv6 underlay or per-platform route/capture campaign is provisioned in the default devstack acceptance run (transport_scenario_test.go memberIPv6). |  |

### Unexpected Blocked Rows (not on expected-blocked list; fails nightly gate)

| Scenario | Acceptance Status | Detail |
|---|---|---|
| `update-good` | **blocked** |  |

fvpn-e2e-nightly-gate: nightly gate failed (pass=31 expected_blocked=1 unexpected_blocked=1 fail=0 error=0)
exit status 1
```

## nightly-before

`go run ./tests/e2e/cmd/fvpn-e2e-nightly-gate -summary docs/e2e/runs/2026-11-04/followup34/acceptance-summary.md -test-exit-code 2 -test-log docs/e2e/runs/2026-11-04/followup34/05-acceptance.txt`

UTC 2026-11-04T06:28:05.925227+00:00–2026-11-04T06:28:06.148563+00:00; exit 1.

```text
| `lease-expiry` | **pass** | w2-agent-core | Candidate fvpn-agent started by fvpn-updater during update-good stepStart runs boot-time recoverOnce() while parent fvpn-updater still holds /var/lib/vpn-node/update.lock, hitting UPDATE_BUSY and latching recoverBlocked=true (reason recovery_required, serving 0) in memory until node-restart restarts the agent, so countedFlow finds no device admission counters during the 90s readiness poll. |  |

### Unexpected Blocked Rows (not on expected-blocked list; fails nightly gate)

| Scenario | Acceptance Status | Detail |
|---|---|---|
| `update-good` | **blocked** |  |

fvpn-e2e-nightly-gate: nightly gate failed (pass=31 expected_blocked=1 unexpected_blocked=1 fail=0 error=0)
exit status 1
```
