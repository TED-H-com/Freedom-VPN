# Follow-up 19 gate ledger

Real command exits, including interrupted setup attempts and retained live
failures. Logs are private under `.agent/tmp/followup19/<gate>.log`.
Foreground gate order is preserved below. `go-with-disk-tmp=` disables only
the make helper shell cleanup trap; the caller supplies disk TMPDIR/GOTMPDIR.

| Gate | Exit | Seconds | Command |
|---|---:|---:|---|
| up-serving | 0 | 39.91 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-serving | 1 | 32.69 | `env FVPN_E2E_FOLLOWUP19=1 go test -tags e2e -count=1 -v -timeout 30m ./tests/e2e -run '^TestE2E_Followup19Installer$'` |
| decode-tests | 0 | 0.04 | `python3 -m unittest discover -s tests/e2e -p test_decode_capture.py` |
| endpoint-tests | 0 | 0.59 | `go test -count=1 ./tests/e2e/internal/streamstats ./tests/e2e/cmd/fvpn-e2e-origin` |
| down-admission | 0 | 42.54 | `make -f mk/e2e.mk e2e-down` |
| sealed-failure-cli | 0 | 0.89 | `go test -count=1 -v ./node/updater/cmd/fvpn-updater -run '^TestF903_Devstack18SealedServingFailureCLI$'` |
| d124-current-fixture | 0 | 0.45 | `go test -count=1 -v ./tests/e2e/cmd/fvpn-e2e-prepare` |
| up-serving-retry | 0 | 646.0 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-serving-retry | 1 | 145.56 | `env FVPN_E2E_FOLLOWUP19=1 go test -tags e2e -count=1 -v -timeout 30m ./tests/e2e -run '^TestE2E_Followup19Installer$'` |
| down-admission-second | 0 | 42.59 | `make -f mk/e2e.mk e2e-down` |
| up-serving-visible | 0 | 442.2 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-serving-visible | 0 | 231.33 | `env FVPN_E2E_FOLLOWUP19=1 go test -tags e2e -count=1 -v -timeout 30m ./tests/e2e -run '^TestE2E_Followup19Installer$'` |
| down-serving-visible | 0 | 43.85 | `make -f mk/e2e.mk e2e-down` |
| up-isolated | 0 | 48.59 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-isolated | 1 | 280.27 | `env FVPN_E2E_FOLLOWUP19=1 FVPN_E2E_CANDIDATE18=1 go test -tags e2e -count=1 -v -timeout 30m ./tests/e2e -run '^TestE2E_Followup19Installer$'` |
| down-isolated | 0 | 42.69 | `make -f mk/e2e.mk e2e-down` |
| up-native | 0 | 30.11 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-native | 0 | 81.23 | `env FVPN_E2E_FOLLOWUP19=1 go test -tags e2e -count=1 -v -timeout 25m ./tests/e2e -run '^TestE2E_Followup19Native$'` |
| down-native | 0 | 42.09 | `make -f mk/e2e.mk e2e-down` |
| up-dns | 0 | 31.8 | `env FVPN_E2E_TWO_RESOLVERS=true make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-dns | 1 | 266.52 | `env FVPN_E2E_FOLLOWUP19=1 FVPN_E2E_FIXTURES16=1 go test -tags e2e -count=1 -v -timeout 35m ./tests/e2e -run '^TestE2E_Followup16Fixtures$'` |
| down-dns | 0 | 42.76 | `make -f mk/e2e.mk e2e-down` |
| up-native-repeat | 0 | 29.86 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-native-repeat | 0 | 93.66 | `env FVPN_E2E_FOLLOWUP19=1 go test -tags e2e -count=1 -v -timeout 25m ./tests/e2e -run '^TestE2E_Followup19Native$'` |
| down-final | 0 | 42.01 | `make -f mk/e2e.mk e2e-down` |
| e2e-lint | 0 | 2.86 | `make -f mk/e2e.mk e2e-lint` |
| e2e-unit | 0 | 3.55 | `go test ./tests/e2e/...` |
| e2e-tagged | 0 | 4.31 | `go test -tags e2e ./tests/e2e/...` |
| committed-hygiene | 0 | 0.16 | `go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'` |
| handover | 0 | 0.73 | `go test -count=1 ./tests/handover/...` |
| go-lint | 0 | 3.52 | `make go-lint go-with-disk-tmp=` |
| build-amd64 | 0 | 0.57 | `env GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...` |
| build-arm64 | 0 | 1.02 | `env GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...` |
| diff-check | 0 | 0.0 | `git diff --check` |
| final-audit | 0 | 0.4 | `python3 .agent/tmp/followup19/audit.py` |

The first two serving attempts stopped before apply after other suites resumed.
A visible foreground `gates.sh` marker was then used for heavy commands.
The stopped-agent failure and the DNS campaign's MTU failure are retained as
exit 1. All final offline gates pass; this does not turn live failures green.
Decoder unit tests execute two tests; endpoint tests exercise real net.Pipe
partial-read/error counts; the sealed-failure CLI regression executes its named
test. D-124's existing command tests pass but contain no Decide fixture.
Final audit: 138 archived files byte-identical, 8,362 payload-free header
records, unchanged prior run roots, zero owned Docker inventory.
