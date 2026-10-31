# Follow-up 21 gate ledger

Foreground processes; disk scratch; `GOCACHE` read from `go env GOCACHE`.
Admission was checked before up/live and after startup. Two other active suites
delayed the fresh installer measurement; no third live suite was started.

| Gate | Exit | Seconds | Command |
|---|---:|---:|---|
| up | 0 | 33.83 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-dns | 0 | 215.59 | `go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup21DNS$'` |
| live-installer | 1 | 15.39 | `go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup21Installer$'` |
| down-first | 0 | 42.89 | `make -f mk/e2e.mk e2e-down` |
| up-installer | 0 | 35.94 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| focused-unit | 0 | 1.16 | `go test -tags e2e -count=1 -v ./tests/e2e ./tests/e2e/internal/dnsrelay -run '^(TestAT20_|TestENR02_InstallerFixture)'` |
| live-installer-fresh | 0 | 30.34 | `go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup21Installer$'` |
| down-installer | 0 | 42.63 | `make -f mk/e2e.mk e2e-down` |
| up-dns-final | 0 | 27.16 | `make -f mk/e2e.mk e2e-up E2E_RESET=1` |
| live-dns-final | 0 | 209.92 | `go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup21DNS$'` |
| down-final | 0 | 42.7 | `make -f mk/e2e.mk e2e-down` |
| e2e-lint | 0 | 5.33 | `make -f mk/e2e.mk e2e-lint` |
| e2e-unit | 0 | 4.74 | `go test ./tests/e2e/...` |
| e2e-tagged | 0 | 4.59 | `go test -tags e2e ./tests/e2e/...` |
| committed-hygiene | 0 | 0.19 | `go test -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'` |
| handover | 0 | 0.84 | `go test -count=1 ./tests/handover/...` |
| go-lint | 0 | 6.22 | `make go-lint go-with-disk-tmp=` |
| build-amd64 | 0 | 1.09 | `env GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...` |
| build-arm64 | 0 | 3.2 | `env GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...` |
| diff-check | 0 | 0.0 | `git diff --check` |
| final-audit | 1 | 0.15 | `python3 .agent/tmp/followup21/audit.py` |
| final-audit-corrected | 0 | 0.39 | `python3 .agent/tmp/followup21/audit.py` |

The first installer attempt exits 1 during origin attachment because the DNS
campaign retained node3 in the installer fixture’s existing origin slot. Its
one recorded login is only a partial record, never a passed installer campaign.
The fresh installer run passes on an independent reset stack. The first DNS
functional run passes while reporting incomplete diagnostic streams; the final
reader preserves intentional restart boundaries and the final integrity row
requires complete whole-sequence records. No original evidence is overwritten.

Final offline checks use the prescribed order. Root `go-lint` uses
`go-with-disk-tmp=` only to avoid the existing helper cleanup trap; the caller
provides disk temp directories and the same gofmt/vet/staticcheck v0.8.2 checks.
No dependency file changed. Physical arm64 remains unmeasured. The first custom
scope audit compared directly to advancing main and included another lane’s newly
merged files; it exited 1. The corrected audit uses main...HEAD plus working-tree
changes, as required by AGENTS. Both actual exits are retained. Final audit
requires zero owned Docker containers, networks, volumes and images.
