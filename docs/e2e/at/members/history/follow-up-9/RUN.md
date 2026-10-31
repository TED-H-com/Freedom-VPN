# Follow-up 9 run ledger

Run `11e305fe-6360-4dbd-95b1-67274aa4988c`, harness `49fbb48e013b824c4ce107d3cb551db79b915b42`, product baseline `7e044c71`.
The suite started from a clean committed tree (`build.source_dirty=false`);
per-binary/source/image/release digests remain in each case JSON. Assigned stack:
`w4-members / 24000`, two resolvers, disk scratch under `.agent/tmp`.

**3 pass / 6 blocked / 1 fail; 560 pass / 5 blocked / 2 fail subchecks.**
AT02, AT03 and AT08 pass. AT09 fails on F9-01's provisional-code assertion and
its composite recovery/lease assertion. All other executed checks pass.
F8-01 recovers at unchanged generation 30; the native dual/1280 row completes
64,064 bytes. The original F8-02 cause remains unattributed. [Case table](summary.md),
[measurements](NATIVE.md), [full restart capture](AT09-restart.json),
[dated owner requests](requests.md).

The live gate ran from `2026-10-31T11:34:34.108947+00:00` to
`2026-10-31T12:09:25.097085+00:00`. It exits 2 honestly while acceptance is non-green.
Admission checked `pgrep -af 'gates.sh|mk/at-'` and waited until fewer than two
other suites were active; one other suite was active at this fresh startup.
Gates were foreground and sequential. No shared PostgreSQL, host networking,
or another lane's Docker objects were changed.

| Gate, in order | Exit |
|---|---:|
| make -f mk/at-members.mk at-members-lint | 0 |
| make -f mk/at-members.mk at-members | 2 |
| go test ./tests/e2e/at/members/... | 0 |
| make go-lint | 0 |
| go vet ./... | 0 |
| tracked Go formatting check | 0 |
| git diff --check | 0 |
| at-members-build: three helpers, linux/amd64 + linux/arm64 | 0 |
| committed-evidence hygiene + D-098 archive checks | 0 |
| handover immutable members8 claim | 0 |

Area lint, root lint, vet, formatting, diff and cross-build have empty stdout and exit 0. The build target compiles the area probe, shared fixture probe and member-session helper for both architectures. This is not physical arm64 qualification.

Actual live output tail:

```text
=== RUN   TestATMembers_Live/AT21
    live_test.go:175: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:175: AT22: blocked; see docs/e2e/at/members/AT22.json
=== RUN   TestATMembers_Live/AT09
    live_test.go:175: AT09: fail; see docs/e2e/at/members/AT09.json
--- FAIL: TestATMembers_Live (2090.31s)
    --- FAIL: TestATMembers_Live/AT05 (15.30s)
    --- FAIL: TestATMembers_Live/AT01 (71.83s)
    --- PASS: TestATMembers_Live/AT02 (415.53s)
    --- PASS: TestATMembers_Live/AT03 (183.98s)
    --- FAIL: TestATMembers_Live/AT04 (596.69s)
    --- PASS: TestATMembers_Live/AT08 (109.75s)
    --- FAIL: TestATMembers_Live/AT20 (66.62s)
    --- FAIL: TestATMembers_Live/AT21 (33.59s)
    --- FAIL: TestATMembers_Live/AT22 (368.82s)
    --- FAIL: TestATMembers_Live/AT09 (222.13s)
=== RUN   TestAT09_SelectedExitFamilyExcludesOtherPeer
--- PASS: TestAT09_SelectedExitFamilyExcludesOtherPeer (0.00s)
=== RUN   TestAT02_OperationCompletionRequiresParentSuccess
--- PASS: TestAT02_OperationCompletionRequiresParentSuccess (0.00s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	2090.431s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Actual offline output tail:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.125s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	(cached)
```

Actual hygiene output tail:

```text
=== RUN   TestCommittedEvidenceHygiene
    evidence_test.go:55: scanned 139 acceptance JSON records plus documentation
--- PASS: TestCommittedEvidenceHygiene (0.10s)
=== RUN   TestD098_Followup7Archive
--- PASS: TestD098_Followup7Archive (0.00s)
=== RUN   TestD098_Followup8Archive
--- PASS: TestD098_Followup8Archive (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.105s
```

Actual handover output tail:

```text
=== RUN   TestHandover_Followup16Claims
=== RUN   TestHandover_Followup16Claims/members8_passes_and_AT09_failure
--- PASS: TestHandover_Followup16Claims (0.01s)
    --- PASS: TestHandover_Followup16Claims/members8_passes_and_AT09_failure (0.01s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.013s
```

Stack startup exit **0**; teardown exit **0**.
Independent final inventory: `{"containers":0,"networks":0,"volumes":0,"images":0}`.
No task-owned databases or containers remain.

D-098 archived all **32** follow-up 8 root files byte-identically, verified
against baseline `7e044c71`, with [SHA256SUMS](history/follow-up-8/SHA256SUMS).
All 13 existing handover snapshots match those bytes; its members8 claim test
passes without editing an outside-owned pin.

[Interrupted follow-up 9 attempt 1](history/follow-up-9/attempt-1/RUN.md)
preserves three stale audit-selector failures and a read-only diagnostic.
Only the owned selector changed from `grant.revoke` to the current production
`member.grant_revoke`; the fresh full run passes all 122 AT02 checks. No failed
attempt is rewritten or counted as a current pass.

Reproduce after obtaining a shared-host slot:

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
python3 tests/e2e/at/members/disktmp.py go test -count=1 -v -run '^(TestCommittedEvidenceHygiene|TestD098_Followup[78]Archive)$' ./tests/e2e/at/members
```
