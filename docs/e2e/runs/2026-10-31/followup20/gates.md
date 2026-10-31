# Follow-up 20 command ledger

All commands ran in the foreground with disk TMPDIR/GOTMPDIR. Live runners
use `-count=1`; up/live admission waits while two other suites are active.
Seconds include admission wait. Full original logs remain private under
`.agent/tmp/followup20`; actual exits below retain failures.

| Gate | Exit | Seconds | Command |
|---|---:|---:|---|
| up-grants | 2 | 18.56 | `env FVPN_E2E_GRANT_SEAM=1 make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| up-grants-retry | 0 | 31.81 | `env FVPN_E2E_GRANT_SEAM=1 make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-grants | 1 | 71.69 | `env FVPN_E2E_FOLLOWUP20=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup20Grants$'` |
| down-grants | 0 | 42.15 | `make -f mk/e2e.mk e2e-down` |
| up-grants-stable | 0 | 31.46 | `env FVPN_E2E_GRANT_SEAM=1 make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-grants-stable | 0 | 142.69 | `env FVPN_E2E_FOLLOWUP20=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup20Grants$'` |
| live-counter | 0 | 78.28 | `env FVPN_E2E_COUNTER20=1 go test -tags e2e -count=1 -v -timeout 10m ./tests/e2e -run '^TestE2E_Followup20CounterExpiry$'` |
| down-counter | 0 | 42.87 | `make -f mk/e2e.mk e2e-down` |
| up-grants-wire | 0 | 33.0 | `env FVPN_E2E_GRANT_SEAM=1 make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| down-idle-refresh | 0 | 42.1 | `make -f mk/e2e.mk e2e-down` |
| up-grants-final | 0 | 131.79 | `env FVPN_E2E_GRANT_SEAM=1 make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-grants-final | 0 | 142.09 | `env FVPN_E2E_FOLLOWUP20=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup20Grants$'` |
| live-counter-timing | 0 | 91.72 | `env FVPN_E2E_COUNTER20=1 go test -tags e2e -count=1 -v -timeout 10m ./tests/e2e -run '^TestE2E_Followup20CounterExpiry$'` |
| down-grants-final | 0 | 42.04 | `make -f mk/e2e.mk e2e-down` |
| up-regular | 0 | 43.04 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-regular | 0 | 162.16 | `env FVPN_E2E_FOLLOWUP20=1 go test -tags e2e -count=1 -v -timeout 25m ./tests/e2e -run '^TestE2E_Followup20Regular$'` |
| down-regular | 0 | 51.18 | `make -f mk/e2e.mk e2e-down` |
| up-broader1 | 0 | 39.38 | `env FVPN_E2E_TWO_RESOLVERS=true make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-broader1 | 1 | 340.52 | `env FVPN_E2E_FOLLOWUP20=1 FVPN_E2E_FIXTURES16=1 go test -tags e2e -count=1 -v -timeout 35m ./tests/e2e -run '^TestE2E_Followup16Fixtures$'` |
| down-broader1 | 0 | 42.94 | `make -f mk/e2e.mk e2e-down` |
| e2e-lint | 0 | 2.92 | `make -f mk/e2e.mk e2e-lint` |
| e2e-unit | 0 | 3.27 | `go test ./tests/e2e/...` |
| e2e-tagged | 0 | 4.22 | `go test -tags e2e ./tests/e2e/...` |
| committed-hygiene | 0 | 0.19 | `go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'` |
| handover | 0 | 0.6 | `go test -count=1 ./tests/handover/...` |
| go-lint | 0 | 2.25 | `make go-lint go-with-disk-tmp=` |
| build-amd64 | 0 | 1.03 | `env GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...` |
| build-arm64 | 0 | 2.47 | `env GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...` |
| diff-check | 0 | 0.0 | `git diff --check` |
| decode-tests | 0 | 0.04 | `python3 -m unittest discover -s tests/e2e -p test_decode_capture.py` |
| final-audit | 0 | 0.47 | `python3 .agent/tmp/followup20/audit.py` |

The first up failed on an owned Compose indentation error before the live run;
the correction and successful retry are retained. The initial D-087 run failed
the invariant check during natural initial FUP projection; the later runner
waits for stable outputs. The broader live runner exits 1 on the captured MTU
failure. None is relabeled as a successful gate.

One admission waiter was cancelled before it started any live command, so its
idle stack was removed and refreshed for the final timing-enabled run. It is
not counted as an executed live attempt.

The root lint invocation sets `go-with-disk-tmp=` only to avoid the existing
helper cleanup trap; the caller supplies disk temp paths and the same gofmt,
vet and pinned staticcheck v0.8.2 execute. No dependency files changed.

Final offline gates pass in the prescribed order. Both linux architectures
cross-build; live execution is amd64 only. The audit verifies 168 original
follow-up 19 archive files, unchanged follow-up 16–19 roots, 1,548 decoded
headers, 47 original wire hashes, all counter/timing joins and zero owned Docker
containers/networks/volumes/images. Shared PostgreSQL was not reconfigured.
