# Follow-up 32 checks — 2026-11-03

All commands ran in the foreground after `bash "$FVPN_HOLD_SH"` printed
`free`. Private wrappers checked admission again before nested make/npm/docker
commands. No reservation was taken. Admission denials (exit 125) before commands
started are waits, not test results. Scratch: `<worktree>/.agent/tmp` for
TMPDIR/GOTMPDIR; GOCACHE from `go env GOCACHE`; CGO disabled; local Go toolchain.
Python used `-B` / `PYTHONDONTWRITEBYTECODE=1`. No shell cleanup trap.

The first two e2e-lint invocations found an unused import and non-stack-only
unused live metadata. Both are retained below. The metadata stayed in the
tagged session; the shared HTTP body moved to a minimal client. No lint rule
was suppressed. Later transport-specific checks were rerun after adding the
explicit protection against Go implicit replays and redirect following.

| Invocation | Started UTC | Finished UTC | Exit |
|---|---|---|---:|
| `e2e-lint` | 2026-11-03T22:55:48.617962Z | 2026-11-03T22:55:49.994951Z | 2 |
| `e2e-lint-fixed` | 2026-11-03T22:56:03.689186Z | 2026-11-03T22:56:05.779206Z | 2 |
| `e2e-lint-final` | 2026-11-03T22:56:42.104719Z | 2026-11-03T22:56:45.675492Z | 0 |
| `synthetic` | 2026-11-03T22:56:52.788974Z | 2026-11-03T22:57:03.168555Z | 0 |
| `isolation-lint` | 2026-11-03T22:57:14.883472Z | 2026-11-03T22:57:19.483703Z | 0 |
| `e2e-unit` | 2026-11-03T22:57:19.503437Z | 2026-11-03T22:57:43.342612Z | 0 |
| `handover` | 2026-11-03T22:57:43.361950Z | 2026-11-03T22:57:47.799113Z | 0 |
| `go-lint` | 2026-11-03T22:57:47.818736Z | 2026-11-03T22:57:51.536103Z | 0 |
| `hygiene` | 2026-11-03T22:57:51.555938Z | 2026-11-03T22:58:00.063017Z | 0 |
| `e2e-lint-transport` | 2026-11-03T23:00:01.485379Z | 2026-11-03T23:00:05.545026Z | 0 |
| `synthetic-transport` | 2026-11-03T23:00:05.564747Z | 2026-11-03T23:00:15.945864Z | 0 |
| `isolation-lint-transport` | 2026-11-03T23:00:15.966058Z | 2026-11-03T23:00:20.466994Z | 0 |
| `e2e-unit-transport` | 2026-11-03T23:00:20.487732Z | 2026-11-03T23:00:45.299350Z | 0 |
| `handover-transport` | 2026-11-03T23:05:15.156113Z | 2026-11-03T23:05:19.955928Z | 0 |
| `go-lint-transport` | 2026-11-03T23:05:19.975627Z | 2026-11-03T23:05:21.887679Z | 0 |
| `build-amd64` | 2026-11-03T23:05:21.908427Z | 2026-11-03T23:05:23.037539Z | 0 |
| `build-arm64` | 2026-11-03T23:05:23.057495Z | 2026-11-03T23:05:24.289731Z | 0 |
| `initial-inventory` | 2026-11-03T23:05:24.311433Z | 2026-11-03T23:05:24.404425Z | 0 |
| `forced-stack-up` | 2026-11-03T23:05:24.425508Z | 2026-11-03T23:05:59.494537Z | 0 |
| `forced-publish` | 2026-11-03T23:06:10.847403Z | 2026-11-03T23:06:41.422688Z | 1 |
| `forced-pre-enrollment-status` | 2026-11-03T23:07:56.139972Z | 2026-11-03T23:07:56.240904Z | 0 |
| `forced-stack-down` | 2026-11-03T23:07:56.260378Z | 2026-11-03T23:09:18.593779Z | 0 |
| `ordinary-stack-up` | 2026-11-03T23:09:18.613202Z | 2026-11-03T23:09:43.460679Z | 0 |
| `ordinary-publish` | 2026-11-03T23:10:07.717977Z | 2026-11-03T23:10:59.511374Z | 0 |
| `ordinary-stack-down` | 2026-11-03T23:11:13.498082Z | 2026-11-03T23:11:35.940625Z | 0 |
| `final-containers` | 2026-11-03T23:12:48.506286Z | 2026-11-03T23:12:48.594458Z | 0 |
| `final-networks` | 2026-11-03T23:12:48.614512Z | 2026-11-03T23:12:48.702451Z | 0 |
| `final-volumes` | 2026-11-03T23:12:48.722443Z | 2026-11-03T23:12:48.811379Z | 0 |
| `final-images` | 2026-11-03T23:12:48.831451Z | 2026-11-03T23:12:48.919353Z | 0 |
| `final-hygiene` | 2026-11-03T23:12:48.939246Z | 2026-11-03T23:12:58.006513Z | 0 |

The first live invocation failed before enrollment because the initial private
admission wrapper prefixed Docker stdout with `free`. `lab.stage` parses that
output as JSON; the wrapper was corrected to capture admission output before
executing Docker. The failed invocation is retained and is not retried. The
second invocation is the separately requested ordinary path on a new stack.
The forced invocation set both `FVPN_E2E_FOLLOWUP32=1` and
`FVPN_E2E_PUBLISH_STALE_ETAG=1`; the ordinary invocation set only the former and
explicitly unset the latter. Each used a separate fresh reset stack.

Both cross-builds used `GOOS=linux`; `build-amd64` used `GOARCH=amd64`, and
`build-arm64` used `GOARCH=arm64`. arm64 is compile-only.

Final inventories at 23:12:48Z: **0 containers, 0 networks, 0 volumes, 0 images**.
Each inventory command below exited 0 with empty stdout. Both e2e-down calls
completed; no test process was left running.

## Actual output tails

`e2e-lint` — `make -f mk/e2e.mk e2e-lint`

```text
free
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
# github.com/TED-H-com/Freedom-VPN/tests/e2e
# [github.com/TED-H-com/Freedom-VPN/tests/e2e]
vet: tests/e2e/followup27_live_test.go:12:2: "github.com/google/uuid" imported and not used
make: *** [mk/e2e.mk:34: e2e-lint] Error 1
```

`e2e-lint-fixed` — `make -f mk/e2e.mk e2e-lint`

```text
free
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
tests/e2e/http_session.go:15:2: field role is unused (U1000)
tests/e2e/http_session.go:16:2: field memberID is unused (U1000)
exit status 1
make: *** [mk/e2e.mk:35: e2e-lint] Error 1
```

`e2e-lint-final` — `make -f mk/e2e.mk e2e-lint`

```text
free
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

`synthetic` — `go test -v -count=1 ./tests/e2e -run '^TestAPI01_Node'`

```text
--- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure (0.10s)
    --- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure/write_transport (0.00s)
    --- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure/read_transport (0.00s)
    --- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure/read_503 (0.00s)
    --- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure/missing_etag (0.00s)
    --- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure/deadline (0.10s)
=== RUN   TestAPI01_NodeIntendedRequiresEveryDesiredValue
--- PASS: TestAPI01_NodeIntendedRequiresEveryDesiredValue (0.00s)
=== RUN   TestAPI01_NodeWriteTotalDeadline
--- PASS: TestAPI01_NodeWriteTotalDeadline (10.01s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	10.126s
```

`isolation-lint` — `make -f mk/at-isolation.mk at-isolation-lint`

```text
free
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

`e2e-unit` — `go test -count=1 ./tests/e2e/...`

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.019s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.058s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.107s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.003s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

`handover` — `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.740s
```

`go-lint` — `make go-lint go-with-disk-tmp=`

```text
free
```

`hygiene` — `go test -v -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.26s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.271s
```

`e2e-lint-transport` — `make -f mk/e2e.mk e2e-lint`

```text
free
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

`synthetic-transport` — `go test -v -count=1 ./tests/e2e -run '^TestAPI01_Node'`

```text
=== RUN   TestAPI01_NodeIntendedRequiresEveryDesiredValue
--- PASS: TestAPI01_NodeIntendedRequiresEveryDesiredValue (0.00s)
=== RUN   TestAPI01_NodeWriteTotalDeadline
--- PASS: TestAPI01_NodeWriteTotalDeadline (10.01s)
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect/reused_connection_eof
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect/redirect
--- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect (0.00s)
    --- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect/reused_connection_eof (0.00s)
    --- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect/redirect (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	10.127s
```

`isolation-lint-transport` — `make -f mk/at-isolation.mk at-isolation-lint`

```text
free
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

`e2e-unit-transport` — `go test -count=1 ./tests/e2e/...`

```text
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.019s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.062s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.004s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.004s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

`handover-transport` — `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	4.401s
```

`go-lint-transport` — `make go-lint go-with-disk-tmp=`

```text
free
```

`build-amd64` — `go build -tags e2e ./tests/e2e/...`

```text
(no output)
```

`build-arm64` — `go build -tags e2e ./tests/e2e/...`

```text
(no output)
```

`initial-inventory` — `docker ps -a --filter 'name=^fvpn-e2e-' --format '{{.Names}}'`

```text
free
```

`forced-stack-up` — `make -f mk/e2e.mk e2e-up E2E_RESET=1`

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

`forced-publish` — `go test -v -tags e2e -count=1 -timeout 18m ./tests/e2e -run '^TestE2E_Followup32Publisher$'`

```text
=== RUN   TestE2E_Followup32Publisher
    scenarios_test.go:131: scenario admin-login: pass
    followup32_live_test.go:52: underlay requires a running fresh node waiting for enrollment
    scenarios_test.go:164: acceptance summary: pass=1 blocked=0 fail=0 run=e22872b7-1112-4082-8e2b-235275f4a659 source=fff680c6c2cd539b77cc534ce9d090f8ad9a8ec3
--- FAIL: TestE2E_Followup32Publisher (30.08s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	30.090s
FAIL
```

`forced-pre-enrollment-status` — `docker exec fvpn-e2e-node cat /var/lib/vpn-node/e2e/status.json`

```text
{"stage":"waiting_token"}
```

`forced-stack-down` — `make -f mk/e2e.mk e2e-down`

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-nb Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

`ordinary-stack-up` — `make -f mk/e2e.mk e2e-up E2E_RESET=1`

```text
Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

`ordinary-publish` — `go test -v -tags e2e -count=1 -timeout 18m ./tests/e2e -run '^TestE2E_Followup32Publisher$'`

```text
scenarios_test.go:131: scenario admin-login: pass
    scenarios_test.go:131: scenario serving-catalogue: pass
    scenarios_test.go:131: scenario enroll-bootstrap: pass
    scenarios_test.go:131: scenario desired-state: pass
    scenarios_test.go:131: scenario publish: pass
    scenarios_test.go:131: scenario publisher-retry-proof: pass
    scenarios_test.go:131: scenario fixture-member-admission: pass
    scenarios_test.go:131: scenario publisher-serving: pass
    scenarios_test.go:164: acceptance summary: pass=8 blocked=0 fail=0 run=c52b760c-c289-4220-a722-6c5a059819b9 source=fff680c6c2cd539b77cc534ce9d090f8ad9a8ec3
--- PASS: TestE2E_Followup32Publisher (51.35s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	51.358s
```

`ordinary-stack-down` — `make -f mk/e2e.mk e2e-down`

```text
Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-nb Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-pg Removed
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Network fvpn-e2e Removed
 Network fvpn-e2e-origin Removed
e2e stack is down
```

`final-containers` — `docker ps -a --filter 'name=^fvpn-e2e-' --format '{{.Names}}'`

```text
(no output)
```

`final-networks` — `docker network ls --filter 'name=^fvpn-e2e-' --format '{{.Name}}'`

```text
(no output)
```

`final-volumes` — `docker volume ls --filter 'name=^fvpn-e2e-' --format '{{.Name}}'`

```text
(no output)
```

`final-images` — `docker image ls --format '{{.Repository}}:{{.Tag}}' 'fvpn-e2e-*'`

```text
(no output)
```

`final-hygiene` — `go test -v -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.80s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.809s
```

## Final repository checks

The first staged diff check found trailing spaces copied from Docker's progress
output into this Markdown ledger. Line-end whitespace was normalized; no
scenario record or outcome was changed. The subsequent formatting and diff
checks passed. Output tails here normalize trailing whitespace only.

`test -z "$(gofmt -l $(git ls-files '*.go'))"`, `git diff --check`, and
`git diff --cached --check` exited 0. Full staged-diff §8 hygiene found no
forbidden matches. All changed paths are owned; no AT-lane path is modified.

Final artifact recheck after adding both complete run logs and the ledger:

`go test -v -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'` ran 2026-11-03T23:13:40.176890Z–2026-11-03T23:13:49.076674Z, exit 0. Its real tail:

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.65s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.655s
```

## Resume gates — 2026-11-03

Admission output was captured in its own private file for all top-level and
nested commands. The wrapper dry command is recorded in
[wrapper-dry-resume.json](wrapper-dry-resume.json): 23:20:51.010299Z–
23:20:51.098110Z, exit 0, complete stdout JSON, empty stderr. Second setup
waited through two admission denials and resumed only on `free`; see
[resume.md](resume.md#reservation-during-second-stack-setup). No reservation
was taken. No denial became a test result.

Initial checks validate the body-digest patch in `419c065c`. The `proof-fix-*`
checks validate the final code in `59cae103`. Each forced test set both
`FVPN_E2E_FOLLOWUP32=1` and `FVPN_E2E_PUBLISH_STALE_ETAG=1`. The ordinary
test set the former and unset the latter. Every invocation used a fresh reset
stack. `go-with-disk-tmp=` disables the root make recipe's cleanup trap while
retaining the explicitly set disk TMPDIR/GOTMPDIR; no shell cleanup trap ran.

| Invocation | Started UTC | Finished UTC | Exit |
|---|---|---|---:|
| `e2e-lint` | 2026-11-03T23:22:27.472994Z | 2026-11-03T23:22:31.191393Z | 0 |
| `synthetic` | 2026-11-03T23:22:31.211254Z | 2026-11-03T23:22:41.583857Z | 0 |
| `e2e-unit` | 2026-11-03T23:22:41.604463Z | 2026-11-03T23:23:04.962022Z | 0 |
| `handover` | 2026-11-03T23:23:04.981386Z | 2026-11-03T23:23:08.978874Z | 0 |
| `go-lint` | 2026-11-03T23:23:08.998174Z | 2026-11-03T23:23:11.150666Z | 0 |
| `isolation-lint` | 2026-11-03T23:23:11.170681Z | 2026-11-03T23:23:15.040034Z | 0 |
| `build-amd64` | 2026-11-03T23:23:34.893267Z | 2026-11-03T23:23:35.855707Z | 0 |
| `build-arm64` | 2026-11-03T23:23:35.875118Z | 2026-11-03T23:23:36.857379Z | 0 |
| `initial-containers` | 2026-11-03T23:23:36.877523Z | 2026-11-03T23:23:36.962036Z | 0 |
| `forced-stack-up` | 2026-11-03T23:23:36.981052Z | 2026-11-03T23:24:10.058350Z | 0 |
| `forced-publish-1` | 2026-11-03T23:24:17.453976Z | 2026-11-03T23:24:39.910663Z | 1 |
| `conflict-response-diagnostic` | 2026-11-03T23:25:54.707224Z | 2026-11-03T23:25:55.290974Z | 0 |
| `proof-fix-synthetic` | 2026-11-03T23:28:53.547477Z | 2026-11-03T23:28:53.926921Z | 0 |
| `forced-stack-down-1` | 2026-11-03T23:28:41.645040Z | 2026-11-03T23:29:04.089703Z | 0 |
| `proof-fix-e2e-lint` | 2026-11-03T23:29:06.207554Z | 2026-11-03T23:29:09.872818Z | 0 |
| `proof-fix-e2e-unit` | 2026-11-03T23:29:09.894149Z | 2026-11-03T23:29:34.363551Z | 0 |
| `proof-fix-handover` | 2026-11-03T23:29:34.384940Z | 2026-11-03T23:29:38.403675Z | 0 |
| `proof-fix-go-lint` | 2026-11-03T23:29:38.424060Z | 2026-11-03T23:29:39.933026Z | 0 |
| `proof-fix-isolation-lint` | 2026-11-03T23:29:39.953418Z | 2026-11-03T23:29:44.265568Z | 0 |
| `forced-stack-up-2` | 2026-11-03T23:29:54.963150Z | 2026-11-03T23:32:26.988290Z | 0 |
| `forced-publish-2` | 2026-11-03T23:33:05.102975Z | 2026-11-03T23:34:11.474516Z | 0 |
| `forced-stack-down-2` | 2026-11-03T23:34:20.698495Z | 2026-11-03T23:34:43.241421Z | 0 |
| `ordinary-stack-up-resume` | 2026-11-03T23:34:43.260988Z | 2026-11-03T23:35:07.700663Z | 0 |
| `ordinary-publish-resume` | 2026-11-03T23:35:13.618785Z | 2026-11-03T23:36:20.163561Z | 0 |
| `ordinary-stack-down-resume` | 2026-11-03T23:36:33.299854Z | 2026-11-03T23:36:55.669351Z | 0 |
| `final-containers-resume` | 2026-11-03T23:36:55.689373Z | 2026-11-03T23:36:55.774643Z | 0 |
| `final-networks-resume` | 2026-11-03T23:36:55.794168Z | 2026-11-03T23:36:55.879773Z | 0 |
| `final-volumes-resume` | 2026-11-03T23:36:55.899541Z | 2026-11-03T23:36:55.984026Z | 0 |
| `final-images-resume` | 2026-11-03T23:36:56.003509Z | 2026-11-03T23:36:56.088463Z | 0 |

Real output tails follow. Line-end whitespace from Docker progress is normalized only in this ledger. Run JSON and complete test logs are unchanged.

`e2e-lint` — `make -f mk/e2e.mk e2e-lint`

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

`synthetic` — `go test -v -count=1 ./tests/e2e -run '^TestAPI01_Node'`

```text
    --- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure/missing_etag (0.00s)
    --- PASS: TestAPI01_NodeWriteStopsOnReadOrTransportFailure/deadline (0.10s)
=== RUN   TestAPI01_NodeIntendedRequiresEveryDesiredValue
--- PASS: TestAPI01_NodeIntendedRequiresEveryDesiredValue (0.00s)
=== RUN   TestAPI01_NodeWriteTotalDeadline
--- PASS: TestAPI01_NodeWriteTotalDeadline (10.00s)
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect/reused_connection_eof
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect/redirect
--- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect (0.00s)
    --- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect/reused_connection_eof (0.00s)
    --- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect/redirect (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	10.120s
```

`e2e-unit` — `go test -count=1 ./tests/e2e/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	0.015s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.118s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.018s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.066s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.001s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

`handover` — `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.663s
```

`go-lint` — `make go-lint go-with-disk-tmp=`

```text
(no output)
```

`isolation-lint` — `make -f mk/at-isolation.mk at-isolation-lint`

```text
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

`build-amd64` — `go build -tags e2e ./tests/e2e/...`

```text
(no output)
```

`build-arm64` — `go build -tags e2e ./tests/e2e/...`

```text
(no output)
```

`initial-containers` — `docker ps -a --filter 'name=^fvpn-e2e-' --format '{{.Names}}'`

```text
(no output)
```

`forced-stack-up` — `make -f mk/e2e.mk e2e-up E2E_RESET=1`

```text
 Container fvpn-e2e-postgres Waiting
 Container fvpn-e2e-node-netns Started
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

`forced-publish-1` — `go test -v -tags e2e -count=1 -timeout 18m ./tests/e2e -run '^TestE2E_Followup32Publisher$'`

```text
=== RUN   TestE2E_Followup32Publisher
    scenarios_test.go:131: scenario admin-login: pass
    scenarios_test.go:131: scenario serving-catalogue: pass
    scenarios_test.go:131: scenario enroll-bootstrap: pass
    scenarios_test.go:131: scenario desired-state: pass
    scenarios_test.go:131: scenario publish: pass
    scenarios_test.go:131: scenario publisher-retry-proof: blocked
    scenarios_test.go:164: acceptance summary: pass=5 blocked=1 fail=0 run=af726b13-9f9c-4675-b81f-edd78d3a5c24 source=419c065cbf93b83b0114f84071fee62981d90d99
    scenarios_test.go:171: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Followup32Publisher (21.96s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	21.968s
FAIL
```

`conflict-response-diagnostic` — `go run -tags e2e .agent/tmp/followup32/resume/diagnose.go`

```text
{"body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","error_code":"RESOURCE_VERSION_CONFLICT","finished_at":"2026-11-03T23:25:55.2830024Z","idempotency_key":"4573558b-666d-41c9-a39c-6e65f500145b","if_match":"\"0\"","method":"PATCH","path":"/api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9","response_bytes":197,"response_sha256":"7c447c7403b114b944a4eee3dcf52e2e3289851014cefec779094b70e1e0d711","started_at":"2026-11-03T23:25:55.279449214Z","status":412}
```

`proof-fix-synthetic` — `go test -v -count=1 ./tests/e2e -run '^TestAPI01_PublisherProofOptionalConflictVersion$'`

```text
=== RUN   TestAPI01_PublisherProofOptionalConflictVersion
=== RUN   TestAPI01_PublisherProofOptionalConflictVersion/{}
=== RUN   TestAPI01_PublisherProofOptionalConflictVersion/{"current_resource_version":"6"}
--- PASS: TestAPI01_PublisherProofOptionalConflictVersion (0.00s)
    --- PASS: TestAPI01_PublisherProofOptionalConflictVersion/{} (0.00s)
    --- PASS: TestAPI01_PublisherProofOptionalConflictVersion/{"current_resource_version":"6"} (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	0.004s
```

`forced-stack-down-1` — `make -f mk/e2e.mk e2e-down`

```text
 Container fvpn-e2e-node-netns Stopped
 Container fvpn-e2e-node-netns Removing
 Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e Removing
 Network fvpn-e2e-origin Removing
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e-origin Removed
 Network fvpn-e2e Removed
e2e stack is down
```

`proof-fix-e2e-lint` — `make -f mk/e2e.mk e2e-lint`

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

`proof-fix-e2e-unit` — `go test -count=1 ./tests/e2e/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	0.014s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.126s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.020s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.072s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.104s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.005s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

`proof-fix-handover` — `go test -count=1 ./tests/handover/...`

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.674s
```

`proof-fix-go-lint` — `make go-lint go-with-disk-tmp=`

```text
(no output)
```

`proof-fix-isolation-lint` — `make -f mk/at-isolation.mk at-isolation-lint`

```text
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

`forced-stack-up-2` — `make -f mk/e2e.mk e2e-up E2E_RESET=1`

```text
 Container fvpn-e2e-postgres Waiting
 Container fvpn-e2e-node-netns Started
 Container fvpn-e2e-adguard Starting
 Container fvpn-e2e-adguard Started
 Container fvpn-e2e-idp Healthy
 Container fvpn-e2e-netbird Healthy
 Container fvpn-e2e-postgres Healthy
 Container fvpn-e2e-member Starting
 Container fvpn-e2e-control Waiting
 Container fvpn-e2e-member Started
 Container fvpn-e2e-control Healthy
 Container fvpn-e2e-node Starting
 Container fvpn-e2e-node Started
e2e stack is up
```

`forced-publish-2` — `go test -v -tags e2e -count=1 -timeout 18m ./tests/e2e -run '^TestE2E_Followup32Publisher$'`

```text
=== RUN   TestE2E_Followup32Publisher
    scenarios_test.go:131: scenario admin-login: pass
    scenarios_test.go:131: scenario serving-catalogue: pass
    scenarios_test.go:131: scenario enroll-bootstrap: pass
    scenarios_test.go:131: scenario desired-state: pass
    scenarios_test.go:131: scenario publish: pass
    scenarios_test.go:131: scenario publisher-retry-proof: pass
    scenarios_test.go:131: scenario fixture-member-admission: pass
    scenarios_test.go:131: scenario publisher-serving: pass
    scenarios_test.go:164: acceptance summary: pass=8 blocked=0 fail=0 run=95fb7eab-6795-48df-b3f2-ed014710d292 source=59cae103757567bb32feb6db71cc8bb89dbb5d6c
--- PASS: TestE2E_Followup32Publisher (65.89s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	65.898s
```

`forced-stack-down-2` — `make -f mk/e2e.mk e2e-down`

```text
 Container fvpn-e2e-node-netns Stopped
 Container fvpn-e2e-node-netns Removing
 Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-pgbackrest-repo Removing
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pg Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e-origin Removed
 Network fvpn-e2e Removed
e2e stack is down
```

`ordinary-stack-up-resume` — `make -f mk/e2e.mk e2e-up E2E_RESET=1`

```text
 Container fvpn-e2e-postgres Waiting
 Container fvpn-e2e-node-netns Started
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

`ordinary-publish-resume` — `go test -v -tags e2e -count=1 -timeout 18m ./tests/e2e -run '^TestE2E_Followup32Publisher$'`

```text
=== RUN   TestE2E_Followup32Publisher
    scenarios_test.go:131: scenario admin-login: pass
    scenarios_test.go:131: scenario serving-catalogue: pass
    scenarios_test.go:131: scenario enroll-bootstrap: pass
    scenarios_test.go:131: scenario desired-state: pass
    scenarios_test.go:131: scenario publish: pass
    scenarios_test.go:131: scenario publisher-retry-proof: pass
    scenarios_test.go:131: scenario fixture-member-admission: pass
    scenarios_test.go:131: scenario publisher-serving: pass
    scenarios_test.go:164: acceptance summary: pass=8 blocked=0 fail=0 run=3ad796d4-b669-42d2-aadf-46e60146822a source=59cae103757567bb32feb6db71cc8bb89dbb5d6c
--- PASS: TestE2E_Followup32Publisher (66.10s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	66.110s
```

`ordinary-stack-down-resume` — `make -f mk/e2e.mk e2e-down`

```text
 Container fvpn-e2e-node-netns Stopped
 Container fvpn-e2e-node-netns Removing
 Container fvpn-e2e-node-netns Removed
 Volume fvpn-e2e-nb Removing
 Volume fvpn-e2e-pg Removing
 Volume fvpn-e2e-pgbackrest-repo Removing
 Network fvpn-e2e-origin Removing
 Network fvpn-e2e Removing
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pgbackrest-repo Removed
 Volume fvpn-e2e-pg Removed
 Network fvpn-e2e-origin Removed
 Network fvpn-e2e Removed
e2e stack is down
```

`final-containers-resume` — `docker ps -a --filter 'name=^fvpn-e2e-' --format '{{.Names}}'`

```text
(no output)
```

`final-networks-resume` — `docker network ls --filter 'name=^fvpn-e2e-' --format '{{.Name}}'`

```text
(no output)
```

`final-volumes-resume` — `docker volume ls --filter 'name=^fvpn-e2e-' --format '{{.Name}}'`

```text
(no output)
```

`final-images-resume` — `docker image ls --format '{{.Repository}}:{{.Tag}}' 'fvpn-e2e-*'`

```text
(no output)
```

All four final inventory commands above exited 0 with exactly zero output
bytes. A separate assertion checked the four output files were empty. The
[receipt comparison](receipt-verification.txt) checked both passing runs' exact
attempt counts, ordered UTC times, fresh ETags, same keys/body hashes, durable
operation counts and all eight scenarios. The first resumed blocked proof is
retained; it was not reclassified.

The evidence hygiene command was not started at 23:38:22Z because admission
reported another lane's netlab work. A bounded foreground loop rechecked every
60 seconds and started only on `free`; this was an admission wait, not a test
result.

`go test -v -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'` — 2026-11-03T23:39:43.984319Z–2026-11-03T23:39:52.568157Z, exit 0.

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.33s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.336s
```

The first staged diff check reported one extra blank line at EOF in the new
`resume.md`; that documentation whitespace was corrected. No run record was
changed. Subsequent tracked-file gofmt, working/staged `git diff --check`, owned
path checks and the full private-string scan passed. Both original invocation
directories remain byte-identical to `193f34c1`; prior RESPONSES/README/ledger
bytes are exact prefixes, and the first resumed blocked run remains unchanged.

Final artifact recheck after adding the preceding ledger result: `go test -v -count=1 ./tests/e2e -run '^TestCommittedFixtureEvidence_ValidAndHygieneClean$'`, 2026-11-03T23:41:10.763746Z–2026-11-03T23:41:19.378738Z, exit 0.

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.34s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.355s
```
