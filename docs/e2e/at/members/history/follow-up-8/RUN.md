# Follow-up 8 run ledger

Run `69f92f49-73dc-4779-8101-dfb6a802a96b`, harness `d79accd9c693aed83713072c925b3b5005b5684c`, product baseline `6848add` (devstack follow-up 16, #284), `w4-members / 24000`. The live suite started from a clean committed tree (`build.source_dirty=false`). Source, executable, release and image hashes remain in every case JSON.

| Case | Full-case status | Subchecks pass / blocked / fail |
|---|---|---:|
| AT01 | blocked | 18 / 0 / 0 |
| AT02 | pass | 122 / 0 / 0 |
| AT03 | pass | 92 / 0 / 0 |
| AT04 | blocked | 99 / 0 / 0 |
| AT05 | blocked | 30 / 0 / 0 |
| AT08 | pass | 31 / 0 / 0 |
| AT09 | fail | 34 / 5 / 4 |
| AT20 | blocked | 81 / 0 / 0 |
| AT21 | blocked | 12 / 0 / 0 |
| AT22 | blocked | 39 / 0 / 0 |

**3 pass / 6 blocked / 1 fail; 558 passing, 5 blocked and 4 failed subchecks.** Live make exit 2 / Go exit 1, package duration 2312.839 seconds. The Go harness deliberately marks incomplete cases non-green; AT09 additionally has four failed assertions.

[summary.md](summary.md), [NATIVE.md](NATIVE.md), [AT02-TRACE.md](AT02-TRACE.md) and [D077.md](D077.md) index the deciding observations. F8-01 and F8-02 in [requests.md](requests.md) retain the restart and MTU failures. The consuming native matrix reconnects NetBird before each row; the selected family is measured, never assumed.

## Execution and gates

Live interval: `2026-10-31T08:46:33.300644Z` to `2026-10-31T09:25:06.802597Z`. Go 1.27.3, CGO_ENABLED=0, GOTOOLCHAIN=local. Area lint preceded stack startup/live; remaining gates ran sequentially after live and teardown. No suite started while two other suites were active. All scratch used disk; no shell cleanup trap.

| Gate (foreground order) | Exit |
|---|---:|
| make -f mk/at-members.mk at-members-lint | 0 |
| make -f mk/at-members.mk at-members | 2 |
| go test ./tests/e2e/at/members/... | 0 |
| make go-lint | 0 |
| go vet ./... | 0 |
| tracked Go formatting check | 0 |
| git diff --check | 0 |
| at-members-build: probe + shared fixture probe + member session, linux/amd64 and linux/arm64 | 0 |
| committed evidence hygiene and D-098 checksum tests | 0 |

```sh
make -f mk/at-members.mk at-members-lint
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 FVPN_E2E_TWO_RESOLVERS=true E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
python3 tests/e2e/at/members/disktmp.py go test ./tests/e2e/at/members/...
python3 tests/e2e/at/members/disktmp.py make go-lint
python3 tests/e2e/at/members/disktmp.py go vet ./...
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
make -f mk/at-members.mk at-members-build
python3 tests/e2e/at/members/disktmp.py go test -count=1 -v -run '^(TestCommittedEvidenceHygiene|TestD098_Followup7Archive)$' ./tests/e2e/at/members
```

## Preserved attempts and archive

- [Diagnostic attempt 1](history/follow-up-8/attempt-1/RUN.md), `942624c2`, complete but not final: 3 nominal pass / 3 blocked / 4 fail. Preserves the restore matcher false-positive, ETag race, relay bypass, DNS startup and empty-profile failures, plus the namespace restart finding.
- [Interrupted attempt 2](history/follow-up-8/attempt-2/RUN.md), `63a4fb5f`, stopped after the stale probe artifact references were found. The failed re-grant transfer remains in its partial record. All probe build/copy/hash reads now share one path; distinct source ports isolate successive transfers and their packet observers.
- [Follow-up 7 final run](history/follow-up-7/RUN.md), `297faa31`, is byte-identical under D-098. All 31 files were also compared directly against baseline commit 6848add. No historical result supplies a current pass.

## Cleanup and scope

Startup and teardown exited 0. Final assigned-prefix inventory: **0 containers, 0 networks, 0 volumes, 0 images**. Compose initially reported bridges still in use; the shared teardown then removed extra owned peers and bridges. Generated secret-bearing state was deleted using the documented pinned cleanup image with `--pull=never`, `--network none`, an owned state-reset container and only the assigned state mount. That container removed itself. Shared Postgres and other lanes were untouched.

Only the owned members tests/docs, mk/at-members.mk and .github/workflows/at-members.yml changed. No application, contract, migration, dependency or shared fixture source changed. No push or PR.

## Actual output tails

live-final:

```text
    live_test.go:175: AT09: fail; see docs/e2e/at/members/AT09.json
--- FAIL: TestATMembers_Live (2312.75s)
    --- FAIL: TestATMembers_Live/AT05 (15.24s)
    --- FAIL: TestATMembers_Live/AT01 (101.46s)
    --- PASS: TestATMembers_Live/AT02 (491.59s)
    --- PASS: TestATMembers_Live/AT03 (208.05s)
    --- FAIL: TestATMembers_Live/AT04 (629.58s)
    --- PASS: TestATMembers_Live/AT08 (135.34s)
    --- FAIL: TestATMembers_Live/AT20 (86.19s)
    --- FAIL: TestATMembers_Live/AT21 (33.56s)
    --- FAIL: TestATMembers_Live/AT22 (368.69s)
    --- FAIL: TestATMembers_Live/AT09 (237.01s)
=== RUN   TestAT09_SelectedExitFamilyExcludesOtherPeer
--- PASS: TestAT09_SelectedExitFamilyExcludesOtherPeer (0.00s)
=== RUN   TestAT02_OperationCompletionRequiresParentSuccess
--- PASS: TestAT02_OperationCompletionRequiresParentSuccess (0.00s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	2312.839s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.001s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

offline-final:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.080s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.001s
```

hygiene-final:

```text
=== RUN   TestCommittedEvidenceHygiene
    evidence_test.go:55: scanned 126 acceptance JSON records plus documentation
--- PASS: TestCommittedEvidenceHygiene (0.08s)
=== RUN   TestD098_Followup7Archive
--- PASS: TestD098_Followup7Archive (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.080s
```

down-final:

```text
 Volume fvpn-w4-members-pg Removing
 Network fvpn-w4-members Resource is still in use
 Network fvpn-w4-members-origin Resource is still in use
 Volume fvpn-w4-members-pgbackrest-repo Removed
 Volume fvpn-w4-members-nb Removed
 Volume fvpn-w4-members-pg Removed
e2e stack is down
```

Area lint, root lint, vet, formatting, diff and both-architecture builds exited 0 with empty stdout. The build target compiled all three helper binaries for both architectures.
