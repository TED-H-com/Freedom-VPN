# Follow-up 27 gate ledger

2026-11-01. Commands ran in the foreground. Fresh stack starts, focused runners and teardown precede the ordered final lint/test/hygiene/handover/build gates. Original failed attempts are retained. Absolute worktree paths in command output are replaced by `<worktree>`; output tails otherwise remain literal.

The final post-teardown sequence passed all nine entries (the architecture build gate has two commands). All eleven fresh starts and eleven teardowns passed. The final populated hold was 13 / 0 / 0 and every native run was 16 / 0 / 0 under the fixture conditions. Earlier hold failures were harness readback errors, and the initial Python unit failure exposed the TEST ONLY marker-length assumption; each remains recorded below. The corrected Python suite passes six tests.

| Label | Started UTC | Seconds | Exit | Command |
|---|---|---:|---:|---|
| hold-01-up | 2026-11-01T14:49:16Z | 30.36 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| hold-unit | 2026-11-01T14:50:19Z | 0.543 | 0 | `go test -tags e2e -count=1 ./tests/e2e/cmd/fvpn-e2e-nodeconfig -run TestAT13_` |
| hold-proxy-build | 2026-11-01T14:50:26Z | 0.469 | 0 | `go build -trimpath -o .agent/e2e-build/fvpn-e2e-nodeconfig ./tests/e2e/cmd/fvpn-e2e-nodeconfig` |
| hold-01-live | 2026-11-01T14:50:27Z | 178.025 | 1 | `env FVPN_E2E_FOLLOWUP27=hold FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| hold-readback-build | 2026-11-01T14:54:26Z | 0.239 | 0 | `go build -o .agent/tmp/followup27/lease-read ./tests/e2e/cmd/fvpn-e2e-lease-read` |
| hold-01-down | 2026-11-01T14:54:55Z | 42.453 | 0 | `make -f mk/e2e.mk e2e-down` |
| hold-02-up | 2026-11-01T14:56:29Z | 30.005 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| analysis-unit | 2026-11-01T14:58:36Z | 0.047 | 1 | `python3 -m unittest discover -s tests/e2e -p test_followup27_analysis.py` |
| analysis-unit-corrected | 2026-11-01T14:58:52Z | 0.048 | 0 | `python3 -m unittest discover -s tests/e2e -p test_followup27_analysis.py` |
| hold-02-live | 2026-11-01T14:58:02Z | 177.595 | 1 | `env FVPN_E2E_FOLLOWUP27=hold FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| hold-02-down | 2026-11-01T15:02:00Z | 42.414 | 0 | `make -f mk/e2e.mk e2e-down` |
| hold-03-up | 2026-11-01T15:03:16Z | 30.628 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| hold-03-live | 2026-11-01T15:03:47Z | 187.894 | 0 | `env FVPN_E2E_FOLLOWUP27=hold FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| hold-03-down | 2026-11-01T15:06:55Z | 42.39 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv4-01-up | 2026-11-01T15:07:37Z | 30.083 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| ipv4-01-live | 2026-11-01T15:08:07Z | 235.111 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv4-01-down | 2026-11-01T15:12:02Z | 22.599 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv4-01-analysis | 2026-11-01T15:12:25Z | 0.833 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/e4c58b1d-505d-40b1-b899-e929f3d20846 .agent/tmp/followup27/e4c58b1d-505d-40b1-b899-e929f3d20846` |
| ipv4-02-up | 2026-11-01T15:12:26Z | 32.659 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| analysis-archive-unit | 2026-11-01T15:15:32Z | 0.057 | 0 | `python3 -m unittest discover -s tests/e2e -p test_followup27_analysis.py` |
| archive-ipv4-01 | 2026-11-01T15:16:00Z | 2.619 | 0 | `python3 tests/e2e/archive_followup27.py docs/e2e/runs/2026-11-01/followup27/e4c58b1d-505d-40b1-b899-e929f3d20846` |
| ipv4-02-live | 2026-11-01T15:12:58Z | 236.089 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv4-02-down | 2026-11-01T15:16:55Z | 22.527 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv4-02-analysis | 2026-11-01T15:17:17Z | 0.787 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/0b6e58f1-a47f-45d4-ae63-60adc117a790 .agent/tmp/followup27/0b6e58f1-a47f-45d4-ae63-60adc117a790` |
| ipv4-03-up | 2026-11-01T15:17:18Z | 31.629 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| archive-ipv4-02 | 2026-11-01T15:18:19Z | 2.512 | 0 | `python3 tests/e2e/archive_followup27.py docs/e2e/runs/2026-11-01/followup27/0b6e58f1-a47f-45d4-ae63-60adc117a790` |
| ipv4-03-live | 2026-11-01T15:17:50Z | 235.68 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv4-03-down | 2026-11-01T15:21:45Z | 22.544 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv4-03-analysis | 2026-11-01T15:22:08Z | 0.774 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/0a8a05f7-7c54-4863-a98e-a0e92e73e7c4 .agent/tmp/followup27/0a8a05f7-7c54-4863-a98e-a0e92e73e7c4` |
| ipv4-04-up | 2026-11-01T15:22:09Z | 30.503 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| ipv4-04-live | 2026-11-01T15:22:39Z | 235.084 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv4-04-down | 2026-11-01T15:26:34Z | 22.645 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv4-04-analysis | 2026-11-01T15:26:57Z | 0.968 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/d992b04f-f83f-4393-b375-891cfb90cf7d .agent/tmp/followup27/d992b04f-f83f-4393-b375-891cfb90cf7d` |
| ipv6-01-up | 2026-11-01T15:26:58Z | 23.043 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| ipv6-01-live | 2026-11-01T15:27:21Z | 235.195 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv6 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv6-01-down | 2026-11-01T15:31:16Z | 23.787 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv6-01-analysis | 2026-11-01T15:31:40Z | 0.855 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/e530b86a-ab8a-411a-bfbf-6130355c6721 .agent/tmp/followup27/e530b86a-ab8a-411a-bfbf-6130355c6721` |
| ipv6-02-up | 2026-11-01T15:31:41Z | 32.788 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| ipv6-02-live | 2026-11-01T15:32:14Z | 225.703 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv6 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv6-02-down | 2026-11-01T15:36:00Z | 23.664 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv6-02-analysis | 2026-11-01T15:36:23Z | 0.773 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/0876ef42-5948-4a1f-b15f-ee83785e3771 .agent/tmp/followup27/0876ef42-5948-4a1f-b15f-ee83785e3771` |
| ipv6-03-up | 2026-11-01T15:36:24Z | 23.033 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| ipv6-03-live | 2026-11-01T15:36:47Z | 235.602 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv6 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv6-03-down | 2026-11-01T15:40:43Z | 22.8 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv6-03-analysis | 2026-11-01T15:41:06Z | 0.899 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/ae1a2829-7834-485d-a4df-557cf6d090d3 .agent/tmp/followup27/ae1a2829-7834-485d-a4df-557cf6d090d3` |
| ipv6-04-up | 2026-11-01T15:41:07Z | 32.861 | 0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| initiation-origin-unit | 2026-11-01T15:43:57Z | 0.063 | 0 | `python3 -m unittest discover -s tests/e2e -p test_followup27_analysis.py` |
| ipv6-04-live | 2026-11-01T15:41:40Z | 234.953 | 0 | `env FVPN_E2E_FOLLOWUP27=native FVPN_E2E_FOLLOWUP26_FAMILY=ipv6 go test -v -tags e2e -count=1 -timeout 20m ./tests/e2e -run '^TestE2E_Followup27$'` |
| ipv6-04-down | 2026-11-01T15:45:35Z | 22.66 | 0 | `make -f mk/e2e.mk e2e-down` |
| ipv6-04-analysis | 2026-11-01T15:45:57Z | 0.844 | 0 | `python3 tests/e2e/analyze_followup27.py docs/e2e/runs/2026-11-01/followup27/1ecee8fe-69a4-4a38-8f72-aa392ee88eb1 .agent/tmp/followup27/1ecee8fe-69a4-4a38-8f72-aa392ee88eb1` |
| e2e-lint | 2026-11-01T15:48:13Z | 2.899 | 0 | `make -f mk/e2e.mk e2e-lint` |
| e2e-test | 2026-11-01T15:48:16Z | 9.163 | 0 | `go test ./tests/e2e/...` |
| e2e-tagged-test | 2026-11-01T15:48:25Z | 10.408 | 0 | `go test -tags e2e ./tests/e2e/...` |
| committed-evidence | 2026-11-01T15:48:35Z | 9.338 | 0 | `go test -count=1 ./tests/e2e -run TestCommittedFixtureEvidence_ValidAndHygieneClean` |
| handover | 2026-11-01T15:48:45Z | 2.1 | 0 | `go test -count=1 ./tests/handover/...` |
| go-lint | 2026-11-01T15:48:47Z | 1.473 | 0 | `make go-lint` |
| build-amd64 | 2026-11-01T15:48:48Z | 1.395 | 0 | `env GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...` |
| build-arm64 | 2026-11-01T15:48:50Z | 1.555 | 0 | `env GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...` |
| diff-check | 2026-11-01T15:48:51Z | 0.008 | 0 | `git diff --check` |

## Concurrency and teardown

One suite at a time in this lane. Admission checked before and after all eleven starts; waits while two other suites ran were 20.042 s and 40.067 s. Total admission-check wall time: 60.531 s. Read-only analysis is not an integration suite.

| Admission check | Wait seconds | Other suites after wait |
|---|---:|---|
| hold-01-up | 20.042 | mk/at-updates.mk |
| hold-01-up-after-start | 0.023 | mk/at-updates.mk |
| hold-02-up | 40.067 | mk/at-isolation.mk |
| hold-02-up-after-start | 0.026 | mk/at-isolation.mk |
| hold-03-up | 0.02 | mk/at-isolation.mk |
| hold-03-up-after-start | 0.023 | mk/at-isolation.mk |
| ipv4-01-up | 0.021 | mk/at-isolation.mk |
| ipv4-01-up-after-start | 0.023 | mk/at-isolation.mk |
| ipv4-02-up | 0.026 | w1-netlab |
| ipv4-02-up-after-start | 0.022 | w1-netlab |
| ipv4-03-up | 0.018 | none |
| ipv4-03-up-after-start | 0.02 | none |
| ipv4-04-up | 0.018 | none |
| ipv4-04-up-after-start | 0.021 | none |
| ipv6-01-up | 0.021 | w4-at-updates |
| ipv6-01-up-after-start | 0.02 | w4-at-updates |
| ipv6-02-up | 0.018 | none |
| ipv6-02-up-after-start | 0.021 | none |
| ipv6-03-up | 0.02 | none |
| ipv6-03-up-after-start | 0.021 | none |
| ipv6-04-up | 0.019 | w4-at-usage |
| ipv6-04-up-after-start | 0.021 | w4-at-usage |

After the final teardown, the harness ownership filter found **zero containers, zero networks, zero volumes and zero mutable image tags**. Shared neutral build cache/upstream image tags and other prefixes were untouched. No shared PostgreSQL drop or reconfiguration, host network mutation or sudo was used. Scratch is disk-backed under `.agent/tmp`; GOCACHE came from `go env GOCACHE`. Live hardware was amd64; arm64 evidence is compilation only.

## Real output tails

### hold-01-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### hold-unit — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-nodeconfig	0.018s
```

### hold-proxy-build — exit 0

```text
(no stdout)
```

### hold-01-live — exit 1

```text
scenarios_test.go:120: scenario admission-serving: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario two-member-serving: pass
    scenarios_test.go:120: scenario populated-hold: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=11 blocked=0 fail=1 run=132c9279-6cbf-4d6b-b626-aaa60bf1f9e3 source=97e38f396583363850fdc826f053abf8c1077df0
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup27 (176.87s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	176.877s
FAIL
```

### hold-readback-build — exit 0

```text
(no stdout)
```

### hold-01-down — exit 0

```text
Container fvpn-e2e-netbird Removed
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Resource is still in use
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### hold-02-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### analysis-unit — exit 1

```text
ERROR: test_stream_sequence_is_explicit_and_closed (test_followup27_analysis.Followup27Analysis.test_stream_sequence_is_explicit_and_closed)
----------------------------------------------------------------------
Traceback (most recent call last):
  File "<worktree>/tests/e2e/test_followup27_analysis.py", line 28, in test_stream_sequence_is_explicit_and_closed
    self.assertEqual(out['fixture_sequence'], str(2**53+9))
                     ~~~^^^^^^^^^^^^^^^^^^^^
KeyError: 'fixture_sequence'

----------------------------------------------------------------------
Ran 4 tests in 0.000s

FAILED (errors=1)
```

### analysis-unit-corrected — exit 0

```text
....
----------------------------------------------------------------------
Ran 4 tests in 0.000s

OK
```

### hold-02-live — exit 1

```text
scenarios_test.go:120: scenario admission-serving: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario two-member-serving: pass
    scenarios_test.go:120: scenario populated-hold: fail
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=11 blocked=0 fail=1 run=ff177856-14b8-4c58-9880-2e5d8f9a0d16 source=1327d8bdbfd4fc9e29b86dddf212725ad1cfa2a2
    scenarios_test.go:155: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup27 (176.21s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	176.223s
FAIL
```

### hold-02-down — exit 0

```text
Container fvpn-e2e-netbird Removed
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Network fvpn-e2e Resource is still in use
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### hold-03-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### hold-03-live — exit 0

```text
scenarios_test.go:120: scenario publish: pass
    scenarios_test.go:120: scenario admission-serving: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario two-member-serving: pass
    scenarios_test.go:120: scenario matching-lease-serving: pass
    scenarios_test.go:120: scenario populated-hold: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=13 blocked=0 fail=0 run=cf799c62-df90-4e7e-ab53-610ab9058a7b source=cc12bcd7f39314a37d0a9d58ae64d5c16f8640af
--- PASS: TestE2E_Followup27 (186.70s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	186.709s
```

### hold-03-down — exit 0

```text
Container fvpn-e2e-netbird Removed
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e Resource is still in use
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### ipv4-01-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### ipv4-01-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=e4c58b1d-505d-40b1-b899-e929f3d20846 source=9d7df53501ee798b6cbe954771dccfda3640ca43
--- PASS: TestE2E_Followup27 (234.72s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	234.743s
```

### ipv4-01-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e-origin Removed
 Network fvpn-e2e Removed
e2e stack is down
```

### ipv4-01-analysis — exit 0

```text
{"run": "e4c58b1d-505d-40b1-b899-e929f3d20846", "first": "pass", "syns": 1, "transit": {"member_packet": 378, "node_packet": 265, "lower_ns": "105912113", "upper_ns": "105972855"}, "stream_attempts": 7304, "missing_ingress": 0, "gaps": 0, "loopback_responses": 1, "verified": 8}
```

### ipv4-02-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### analysis-archive-unit — exit 0

```text
.....
----------------------------------------------------------------------
Ran 5 tests in 0.009s

OK
```

### archive-ipv4-01 — exit 0

```text
{"files": 7}
```

### ipv4-02-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=0b6e58f1-a47f-45d4-ae63-60adc117a790 source=fcf04fd5dcd5fbba91a7c0c635822c26636fcd12
--- PASS: TestE2E_Followup27 (235.63s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	235.653s
```

### ipv4-02-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e-origin Removed
 Network fvpn-e2e Removed
e2e stack is down
```

### ipv4-02-analysis — exit 0

```text
{"run": "0b6e58f1-a47f-45d4-ae63-60adc117a790", "first": "pass", "syns": 1, "transit": {"member_packet": 395, "node_packet": 282, "lower_ns": "106703229", "upper_ns": "106727972"}, "stream_attempts": 7351, "missing_ingress": 0, "gaps": 0, "loopback_responses": 1, "verified": 8}
```

### ipv4-03-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### archive-ipv4-02 — exit 0

```text
{"files": 7}
```

### ipv4-03-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=0a8a05f7-7c54-4863-a98e-a0e92e73e7c4 source=b43fb15af52ed54aa4e13a600a3b2d92469e189b
--- PASS: TestE2E_Followup27 (234.46s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	234.488s
```

### ipv4-03-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### ipv4-03-analysis — exit 0

```text
{"run": "0a8a05f7-7c54-4863-a98e-a0e92e73e7c4", "first": "pass", "syns": 1, "transit": {"member_packet": 378, "node_packet": 206, "lower_ns": "3350965", "upper_ns": "3369257"}, "stream_attempts": 7337, "missing_ingress": 0, "gaps": 0, "loopback_responses": 2, "verified": 8}
```

### ipv4-04-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### ipv4-04-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=d992b04f-f83f-4393-b375-891cfb90cf7d source=757052242f7a5beeb2526a44194af8e257dc99da
--- PASS: TestE2E_Followup27 (234.67s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	234.693s
```

### ipv4-04-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### ipv4-04-analysis — exit 0

```text
{"run": "d992b04f-f83f-4393-b375-891cfb90cf7d", "first": "pass", "syns": 1, "transit": {"member_packet": 385, "node_packet": 282, "lower_ns": "104996068", "upper_ns": "105044803"}, "stream_attempts": 7308, "missing_ingress": 0, "gaps": 0, "loopback_responses": 1, "verified": 8}
```

### ipv6-01-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### ipv6-01-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=e530b86a-ab8a-411a-bfbf-6130355c6721 source=cd190549aec32f8902c8b85d03f96ebd49a7dfa9
--- PASS: TestE2E_Followup27 (234.84s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	234.860s
```

### ipv6-01-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Network fvpn-e2e Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### ipv6-01-analysis — exit 0

```text
{"run": "e530b86a-ab8a-411a-bfbf-6130355c6721", "first": "pass", "syns": 1, "transit": {"member_packet": 467, "node_packet": 219, "lower_ns": "104935676", "upper_ns": "104965715"}, "stream_attempts": 7342, "missing_ingress": 0, "gaps": 0, "loopback_responses": 1, "verified": 8}
```

### ipv6-02-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### ipv6-02-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=0876ef42-5948-4a1f-b15f-ee83785e3771 source=cd190549aec32f8902c8b85d03f96ebd49a7dfa9
--- PASS: TestE2E_Followup27 (225.32s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	225.341s
```

### ipv6-02-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### ipv6-02-analysis — exit 0

```text
{"run": "0876ef42-5948-4a1f-b15f-ee83785e3771", "first": "pass", "syns": 1, "transit": {"member_packet": 418, "node_packet": 170, "lower_ns": "3534703", "upper_ns": "3559238"}, "stream_attempts": 7297, "missing_ingress": 0, "gaps": 0, "loopback_responses": 2, "verified": 8}
```

### ipv6-03-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### ipv6-03-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=ae1a2829-7834-485d-a4df-557cf6d090d3 source=cd190549aec32f8902c8b85d03f96ebd49a7dfa9
--- PASS: TestE2E_Followup27 (235.16s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	235.181s
```

### ipv6-03-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

### ipv6-03-analysis — exit 0

```text
{"run": "ae1a2829-7834-485d-a4df-557cf6d090d3", "first": "pass", "syns": 1, "transit": {"member_packet": 455, "node_packet": 169, "lower_ns": "3585356", "upper_ns": "3630897"}, "stream_attempts": 7322, "missing_ingress": 0, "gaps": 0, "loopback_responses": 2, "verified": 8}
```

### ipv6-04-up — exit 0

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

### initiation-origin-unit — exit 0

```text
......
----------------------------------------------------------------------
Ran 6 tests in 0.009s

OK
```

### ipv6-04-live — exit 0

```text
scenarios_test.go:120: scenario initial-selection: pass
    scenarios_test.go:120: scenario stream-serving: pass
    scenarios_test.go:120: scenario post-stream-serving: pass
    scenarios_test.go:120: scenario member-stream: pass
    scenarios_test.go:120: scenario native-session: pass
    scenarios_test.go:120: scenario fixture-member-admission: pass
    scenarios_test.go:120: scenario native-debug-export: pass
    scenarios_test.go:120: scenario central-record: pass
    scenarios_test.go:153: acceptance summary: pass=16 blocked=0 fail=0 run=1ecee8fe-69a4-4a38-8f72-aa392ee88eb1 source=d616bb1410d7aea35e9d5bad98f751bed7324839
--- PASS: TestE2E_Followup27 (234.50s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	234.525s
```

### ipv6-04-down — exit 0

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e-origin Removed
 Network fvpn-e2e Removed
e2e stack is down
```

### ipv6-04-analysis — exit 0

```text
{"run": "1ecee8fe-69a4-4a38-8f72-aa392ee88eb1", "first": "pass", "syns": 1, "transit": {"member_packet": 470, "node_packet": 195, "lower_ns": "4766312", "upper_ns": "4812435"}, "stream_attempts": 7358, "missing_ingress": 0, "gaps": 0, "loopback_responses": 2, "verified": 8}
```

### e2e-lint — exit 0

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

### e2e-test — exit 0

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.022s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.056s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

### e2e-tagged-test — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.018s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.062s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack	0.009s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	(cached)
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.003s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	(cached)
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	(cached)
```

### committed-evidence — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	9.168s
```

### handover — exit 0

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	1.487s
```

### go-lint — exit 0

```text
(no stdout)
```

### build-amd64 — exit 0

```text
(no stdout)
```

### build-arm64 — exit 0

```text
(no stdout)
```

### diff-check — exit 0

```text
(no stdout)
```
