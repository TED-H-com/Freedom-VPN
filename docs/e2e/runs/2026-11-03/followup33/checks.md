# Follow-up 33 offline gates

Admission is checked separately before each command. No stack or Docker command
runs in this follow-up. Scratch uses `<worktree>/.agent/tmp`; GOCACHE comes from
`go env GOCACHE`. `go-with-disk-tmp=` avoids the root target's shell cleanup trap
while retaining the supplied disk scratch.

Initial status: execution gates await host admission; the denied focused
invocation did not start go test and is not a test failure. Executed gates and
actual outputs will be appended below; earlier entries are retained.

## focused

Command: `go test -v -count=1 -tags e2e ./tests/e2e ./tests/e2e/internal/labstack -run ^(TestAPI01_(NodeWrite|NodeIntended|ExportedNode)|TestENT01_(FiniteGrantRecovery|ExpiredFiniteGrantHasNoEffects)|TestAT02_NativeRouterRollbackEveryStep)`

UTC: 2026-11-04T00:32:33.208379Z–2026-11-04T00:32:44.918830Z; exit 0. Full output: [focused.txt](focused.txt).

```text
=== RUN   TestAPI01_NodeWriteTotalDeadline
--- PASS: TestAPI01_NodeWriteTotalDeadline (10.01s)
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect/reused_connection_eof
=== RUN   TestAPI01_NodeWriteNoImplicitReplayOrRedirect/redirect
--- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect (0.00s)
    --- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect/reused_connection_eof (0.00s)
    --- PASS: TestAPI01_NodeWriteNoImplicitReplayOrRedirect/redirect (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	10.138s
=== RUN   TestAPI01_NodeWriteImportSurface
--- PASS: TestAPI01_NodeWriteImportSurface (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack	0.007s
```

## e2e-lint

Command: `make -f mk/e2e.mk e2e-lint`

UTC: 2026-11-04T00:33:06.251055Z–2026-11-04T00:33:09.277615Z; exit 0. Full output: [e2e-lint.txt](e2e-lint.txt).

```text
cd <worktree> && go vet ./tests/e2e/...
cd <worktree> && go vet -tags e2e ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
cd <worktree> && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...
```

## isolation-lint

Command: `make -f mk/at-isolation.mk at-isolation-lint`

UTC: 2026-11-04T00:33:09.335135Z–2026-11-04T00:33:13.468322Z; exit 0. Full output: [isolation-lint.txt](isolation-lint.txt).

```text
+ go vet ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go vet -tags netlab ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/isolation/... ./tests/netlab/at/...
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab/at/...
+ go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
+ go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
```

## e2e-tagged-vet

Command: `go vet -tags e2e ./tests/e2e/...`

UTC: 2026-11-04T00:33:13.525955Z–2026-11-04T00:33:13.689863Z; exit 0. Full output: [e2e-tagged-vet.txt](e2e-tagged-vet.txt).

```text
(no output)
```

## e2e-unit

Command: `go test -count=1 ./tests/e2e/...`

UTC: 2026-11-04T00:33:13.747913Z–2026-11-04T00:33:38.035404Z; exit 0. Full output: [e2e-unit.txt](e2e-unit.txt).

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-payment	0.014s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-prepare	0.115s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/cmd/fvpn-e2e-stream	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/agentwait	0.018s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread	0.062s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay	0.103s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin	0.004s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture	0.002s
?   	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/offertrace	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth	0.002s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats	0.002s
```

## handover

Command: `go test -count=1 ./tests/handover/...`

UTC: 2026-11-04T00:33:38.096561Z–2026-11-04T00:33:42.112769Z; exit 0. Full output: [handover.txt](handover.txt).

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.702s
```

## go-lint

Command: `make go-lint go-with-disk-tmp=`

UTC: 2026-11-04T00:33:42.173715Z–2026-11-04T00:33:44.153562Z; exit 0. Full output: [go-lint.txt](go-lint.txt).

```text
(no output)
```

## arm64-compile

Command: `env GOOS=linux GOARCH=arm64 go test -c -tags e2e -o .agent/tmp/followup33-arm64.test ./tests/e2e`

UTC: 2026-11-04T00:33:44.213308Z–2026-11-04T00:33:45.647358Z; exit 0. Full output: [arm64-compile.txt](arm64-compile.txt).

```text
(no output)
```

## evidence-hygiene

Command: `go test -v -count=1 ./tests/e2e -run ^TestCommittedFixtureEvidence_ValidAndHygieneClean$`

UTC: 2026-11-04T00:33:45.707913Z–2026-11-04T00:33:54.811928Z; exit 0. Full output: [evidence-hygiene.txt](evidence-hygiene.txt).

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.87s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.882s
```

## final-evidence-hygiene

Command: `go test -v -count=1 ./tests/e2e -run ^TestCommittedFixtureEvidence_ValidAndHygieneClean$`

UTC: 2026-11-04T00:34:53.797750Z–2026-11-04T00:35:02.740130Z; exit 0. Full output: [final-evidence-hygiene.txt](final-evidence-hygiene.txt).

```text
=== RUN   TestCommittedFixtureEvidence_ValidAndHygieneClean
--- PASS: TestCommittedFixtureEvidence_ValidAndHygieneClean (8.71s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	8.721s
```

## Final source checks

`gofmt`, both unstaged/staged whitespace checks, full diff hygiene and ownership checks passed. Exact output:

```text
gofmt: empty
diff --check: pass
cached diff --check: pass
full diff private-string hygiene: pass
owned paths only; no AT edits: pass
RESPONSES existing bytes preserved: pass
existing node_write_test.go unchanged: pass
arm64 compile artifact removed: pass
```

For shell reproduction, quote the focused selection as one argument (the gate entries above display argument values):

```sh
go test -v -count=1 -tags e2e ./tests/e2e ./tests/e2e/internal/labstack -run '^(TestAPI01_(NodeWrite|NodeIntended|ExportedNode)|TestENT01_(FiniteGrantRecovery|ExpiredFiniteGrantHasNoEffects)|TestAT02_NativeRouterRollbackEveryStep)'
```
