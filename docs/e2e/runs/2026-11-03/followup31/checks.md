# Follow-up 31 checks

Code/docs commit checked: `76df1913d8a0b7f77f8b2774743361dc2ecdf8a7`. Toolchain: `go version go1.27.3 linux/amd64`;
`CGO_ENABLED=0`, `GOTOOLCHAIN=local`, disk `TMPDIR` and `GOTMPDIR` set to
`<worktree>/.agent/tmp`; `GOCACHE` obtained with `go env GOCACHE`.

Before each heavy invocation, `bash "$FVPN_HOLD_SH"` printed `free`.
The bounded foreground waiter initially saw an updater reservation and checked
every 60 seconds until admission became free at 09:37:58Z. No reservation was
taken. The initial checks below ran sequentially in the foreground, once each.
The root lint's `go-with-disk-tmp=` override uses the already configured disk
scratch and disables its existing shell cleanup-trap wrapper. No new trap was
used.

| Check / exact command | Started UTC | Finished UTC | Exit |
|---|---|---|---:|
| `make -f mk/e2e.mk e2e-lint` | 2026-11-03T09:37:58Z | 2026-11-03T09:38:01Z | 0 |
| `make -f mk/at-isolation.mk at-isolation-lint` | 2026-11-03T09:38:01Z | 2026-11-03T09:38:04Z | 0 |
| `go test -count=1 ./tests/e2e/...` | 2026-11-03T09:38:04Z | 2026-11-03T09:38:18Z | 0 |
| `go test -count=1 ./tests/handover/...` | 2026-11-03T09:38:18Z | 2026-11-03T09:38:21Z | 0 |
| `make go-lint go-with-disk-tmp=` | 2026-11-03T09:38:21Z | 2026-11-03T09:38:23Z | 0 |
| `go test -v -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'` | 2026-11-03T09:38:23Z | 2026-11-03T09:38:32Z | 0 |
| `GOOS=linux GOARCH=amd64 go build -tags e2e ./tests/e2e/...` | 2026-11-03T09:38:32Z | 2026-11-03T09:38:33Z | 0 |
| `GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/e2e/...` | 2026-11-03T09:38:33Z | 2026-11-03T09:38:34Z | 0 |
| `test -z "$(gofmt -l $(git ls-files '*.go'))"` | 2026-11-03T09:38:34Z | 2026-11-03T09:38:34Z | 0 |
| `git diff --check` | 2026-11-03T09:38:34Z | 2026-11-03T09:38:34Z | 0 |
| `git diff --check main...HEAD` | 2026-11-03T09:38:34Z | 2026-11-03T09:38:34Z | 0 |

Item 1's fresh-stack proof was **not run**, following the explicit stop condition
when no independent source-binding mechanism exists. This is not a successful
stack gate: the requested source-address row remains **not exercised**. Zero
stack attempts and zero retries; no runtime report/audit IDs or hook receipts.

The [archive verification](archive-links.md) passed all 14 references. All 15
new relative documentation links in the initial change resolve. Section 8 diff
hygiene found zero matches. The symlink was accepted by the listed repository
checks, so no Markdown fallback was needed.

## Real output tails

`make -f mk/e2e.mk e2e-lint`:

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

`make -f mk/at-isolation.mk at-isolation-lint`:

```text
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

`go test -count=1 ./tests/e2e/...`:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/isolation	13.193s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	8.377s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/secops	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/secops/proxyprobe	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates	0.021s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates/stopobserve	0.101s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/usage	2.568s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-control	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-fixture-probe	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-idp	0.125s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-lease-read	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-member-session	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-native-observe	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-nbbootstrap	[no test files]
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-nightly-gate	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-nodeconfig	0.076s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-origin	0.405s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	0.018s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.116s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.023s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.064s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.003s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.005s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.003s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.003s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.003s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

`go test -count=1 ./tests/handover/...`:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	2.937s
```

`go test -v -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`:

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.62s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.633s
```

Root Go lint, both architecture builds, gofmt and diff checks emitted no output
and exited 0. No new dependencies were added. arm64 is compile-only.

## Final owned inventory

`FVPN_E2E_PREFIX=e2e` (this lane's existing allocation). No stack was started,
no Docker object was created or modified, and no database was touched. Each
read-only inventory command below ran after a separate `free` admission.

| Kind | UTC | Exit | Count |
|---|---|---:|---:|
| containers | 2026-11-03T09:39:06.728388Z | 0 | 0 |
| networks | 2026-11-03T09:39:06.777047Z | 0 | 0 |
| volumes | 2026-11-03T09:39:06.825471Z | 0 | 0 |
| images | 2026-11-03T09:39:06.874076Z | 0 | 0 |

The inventories used these exact Docker arguments (empty stdout in each case):

```sh
docker ps -a --filter 'name=^fvpn-e2e-' --format '{{.Names}}'
docker network ls --filter 'name=^fvpn-e2e-' --format '{{.Name}}'
docker volume ls --filter 'name=^fvpn-e2e-' --format '{{.Name}}'
docker image ls --format '{{.Repository}}:{{.Tag}}' 'fvpn-e2e-*'
```

No owned containers, networks, volumes or image tags remained. No test process
was left running.

## Final evidence recheck

After adding this gate ledger and the run index, the committed-evidence test
ran once more after `free` admission and exited 0. This covers the new evidence
files; it is not a stack retry. Its real tail:

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.80s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.807s
```
