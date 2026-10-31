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
No dependency file changed. Physical arm64 remains unmeasured. Final audit
requires zero owned Docker containers, networks, volumes and images.
