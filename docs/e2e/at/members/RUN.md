# Follow-up 10 run ledger

Run `2be2d32f-bfd9-46f5-bf29-afaa6617ae94`, harness `731be5a7af7657660d96c240695747a23a3f5c96`, product baseline `d3e9a0f13e1847f5218049660f8f043a4250374f` (agent-core 27 #325, wire-nodes 32 #321, netlab 19 #318). Assigned `w4-members / 24000`, two resolvers. The suite started from a clean committed tree; source, binary, release and image hashes are retained in every case JSON.

**2 complete cases pass / 6 blocked / 2 fail; 560 pass / 5 blocked / 2 fail subchecks.**

F9-01 is closed under D-126 / D-128. The exact archived generation-30
sequence passes the offline predicate replay, including the mixed report; bare
unexplained safety codes and five other invalid variants remain rejected.
Agent-core 27 confirms deliberate synchronous composition, no reporter race or
producer fix. Wire-nodes 32 proves JSON-value/JCS/digest preservation; JSONB
wire whitespace/key order is not the contract, and receipt remains 202.

**Fresh AT09 is blocked, with 38 pass / 5 blocked / 0 fail checks.** The same
complete mutation order and `RestartNodeNamespace90` recovered, and all three
fresh provisional reports passed the D-126 predicate, including the mixed
`TC_FOREIGN_FILTER` report. Desired/applied were **29 → 29**; serving in
**15.751594344 s**, first counted byte in **21.357052110 s**; 88 heartbeat
samples over **90.353953401 s**, boot/final nft/tc, all six observations and
agent log retained. Initial tc rejection is never erased. **F10-04: exact
requested generation-30 qualification was not obtained.** No extra mutation,
state/topology repair or generation bump forced the number. The prior F8-01
closure remains the original generation-30 follow-up 9 evidence.

All four native bulk rows pass. Dual/1280 transfers **64,064/64,064 bytes in
1 ms**, with direct IPv4 selected before/after and counters +67,236/+68,243.
The original F8-02 failure remains unattributed per netlab 19; devstack 19 owns
the exact-sequence decode/capture and application read/write evidence. This
pass is no root-cause fix proof. D-123 IPv6Block scope remains unchanged.

**F10-02: AT02 fails 121/0/1.** One of 234 persistent finite-expiry samples has
`read_ok=false` at `13:35:45.333735552Z`; the underlying error is not retained.
Last TCP/UDP increments are `13:36:38.987828874Z`, 12.171126 ms before the
`13:36:39Z` deadline. Senders remain active and final counters stable; expiry
0/3, three audit rows, restoration 3/0 and counted traffic pass. The missing
read still fails the existing all-reads-valid predicate; no leak or root cause
is inferred and no retry replaces it. Complete failed-check capture:
`F10-02-withdrawal.json` plus the full AT02 record.

**F10-03: AT20 fails 80/0/1.** The positive query initially returns the old
answer with TTL 10 (2,502 ms); at `13:52:12.96280019Z` the before-TTL query
already returns the changed private address class with TTL 10 (1 ms). The
negative control retains NXDOMAIN/SOA TTL 1; both after-TTL controls pass.
Responder selection is not captured, so neither fallback nor a product TTL
defect is attributed. `F10-03-cache.json` preserves the authority change and
queries. All per-type RCODE/SOA/TTL rows pass with unchanged expectations;
AT21's 12 and AT22's 39 executed checks pass. DNS/resource blockers are retained.

| Case / requirement (§19) | Full status | Pass / blocked / fail |
|---|---|---:|
| AT01 / APP01 ID01 MEM01 | blocked | 18 / 0 / 0 |
| AT02 / MEM01 MEM02 MEM03 SEC01 | fail | 121 / 0 / 1 |
| AT03 / BILL01 BILL03 ENT01 | pass | 92 / 0 / 0 |
| AT04 / BILL02 ENT02 MEM03 | blocked | 99 / 0 / 0 |
| AT05 / NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 |
| AT08 / NODE02 NODE03 EXIT01 EXIT03 | pass | 31 / 0 / 0 |
| AT09 / EXIT02 APP01 NFR01 | blocked | 38 / 5 / 0 |
| AT20 / DNS01 DNS02 ISO04 | fail | 80 / 0 / 1 |
| AT21 / DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 |
| AT22 / SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 |

Live interval: `2026-10-31T13:28:37.389828+00:00` to `2026-10-31T14:03:09.707520+00:00`;
Go package duration **2071.604 s**, make exit **2** (Go exit 1). This is the
honest non-green full result, including the two failures and six blocked cases.
Area lint preceded startup/live. At admission `pgrep -af 'gates.sh|mk/at-'`
showed one other live suite (enrollment); no third suite was started by this lane.
All gates were foreground and sequential, with disk scratch under `.agent/tmp`.
Startup and teardown both exited 0. Root lint, vet, formatting, diff and
cross-build produced empty stdout and exited 0. No dependencies changed.

| Foreground gate, in order | Exit |
|---|---:|
| make -f mk/at-members.mk at-members-lint | 0 |
| make -f mk/at-members.mk at-members | 2 |
| go test ./tests/e2e/at/members/... | 0 |
| make go-lint | 0 |
| go vet ./... | 0 |
| tracked Go formatting check | 0 |
| git diff --check | 0 |
| at-members-build: three helpers, linux/amd64 + linux/arm64 | 0 |
| committed-evidence hygiene, D-098 archive checks and D-126 replay | 0 |
| handover immutable members8 claim | 0 |
| archive comparison to baseline and all 13 snapshots | 0 |

D-098 was first: commit `598d546c` archives **all 33** follow-up 9 root files
byte-identically from baseline `d3e9a0f1`, with SHA256SUMS, including the failed
F9-01 capture and existing run ledger. The follow-up 9 attempt-1 subdirectory
is untouched. Follow-up 8's **32** checksums and all **13** handover snapshots
still match; its immutable claim test passes. No outside-owned pin was edited.

Final independent owned Docker inventory is **0 containers / 0 networks /
0 volumes / 0 images**. Private stack state was removed using the pinned cleanup
image with `--pull=never`, `--network none`, an owned `fvpn-w4-members-state-reset`
container and only the assigned state mount; that container removed itself.
Generated stack build artifacts were removed. Shared `fvpn-pg`, other lanes'
Docker objects and host networking were untouched. No sudo or cleanup traps.

[Summary](summary.md), [native and DNS measurements](NATIVE.md), [restart capture](AT09-restart.json), [dated requests and new failures](requests.md).

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
python3 tests/e2e/at/members/disktmp.py go test -count=1 -v -run '^(TestCommittedEvidenceHygiene|TestD098_Followup[789]Archive|TestAT09_D126CapturedProvisionalSequence)$' ./tests/e2e/at/members
```

Actual live output tail:

```text
=== RUN   TestATMembers_Live/AT01
    live_test.go:175: AT01: blocked; see docs/e2e/at/members/AT01.json
=== RUN   TestATMembers_Live/AT02
    live_test.go:175: AT02: fail; see docs/e2e/at/members/AT02.json
=== RUN   TestATMembers_Live/AT03
=== RUN   TestATMembers_Live/AT04
    live_test.go:175: AT04: blocked; see docs/e2e/at/members/AT04.json
=== RUN   TestATMembers_Live/AT08
=== RUN   TestATMembers_Live/AT20
    live_test.go:175: AT20: fail; see docs/e2e/at/members/AT20.json
=== RUN   TestATMembers_Live/AT21
    live_test.go:175: AT21: blocked; see docs/e2e/at/members/AT21.json
=== RUN   TestATMembers_Live/AT22
    live_test.go:175: AT22: blocked; see docs/e2e/at/members/AT22.json
=== RUN   TestATMembers_Live/AT09
    live_test.go:175: AT09: blocked; see docs/e2e/at/members/AT09.json
--- FAIL: TestATMembers_Live (2071.45s)
    --- FAIL: TestATMembers_Live/AT05 (15.59s)
    --- FAIL: TestATMembers_Live/AT01 (73.67s)
    --- FAIL: TestATMembers_Live/AT02 (428.45s)
    --- PASS: TestATMembers_Live/AT03 (183.22s)
    --- FAIL: TestATMembers_Live/AT04 (567.86s)
    --- PASS: TestATMembers_Live/AT08 (109.53s)
    --- FAIL: TestATMembers_Live/AT20 (69.12s)
    --- FAIL: TestATMembers_Live/AT21 (30.32s)
    --- FAIL: TestATMembers_Live/AT22 (368.41s)
    --- FAIL: TestATMembers_Live/AT09 (219.24s)
=== RUN   TestAT09_SelectedExitFamilyExcludesOtherPeer
--- PASS: TestAT09_SelectedExitFamilyExcludesOtherPeer (0.00s)
=== RUN   TestAT02_OperationCompletionRequiresParentSuccess
--- PASS: TestAT02_OperationCompletionRequiresParentSuccess (0.00s)
=== RUN   TestATMembers_AssignedStackGate
    preflight_test.go:27: corrected layout accepted: w4-members / 24000
--- PASS: TestATMembers_AssignedStackGate (0.00s)
=== RUN   TestAT09_D126CapturedProvisionalSequence
=== RUN   TestAT09_D126CapturedProvisionalSequence/bare_safety_code_without_reason
=== RUN   TestAT09_D126CapturedProvisionalSequence/historical_tc_alone
=== RUN   TestAT09_D126CapturedProvisionalSequence/lease_cause_overwritten
=== RUN   TestAT09_D126CapturedProvisionalSequence/duplicate_lease_error
=== RUN   TestAT09_D126CapturedProvisionalSequence/expired_provisional
=== RUN   TestAT09_D126CapturedProvisionalSequence/passed_dependent_check
=== NAME  TestAT09_D126CapturedProvisionalSequence
    provisional_test.go:183: replayed 3 provisional reports (1 mixed) and retained 3 serving reports; historical tc evidence unchanged
--- PASS: TestAT09_D126CapturedProvisionalSequence (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/bare_safety_code_without_reason (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/historical_tc_alone (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/lease_cause_overwritten (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/duplicate_lease_error (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/expired_provisional (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/passed_dependent_check (0.00s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	2071.604s
=== RUN   TestAT02PacketObserverScope
--- PASS: TestAT02PacketObserverScope (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
FAIL
make: *** [mk/at-members.mk:13: at-members] Error 1
```

Actual offline output:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.128s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	(cached)
```

Actual hygiene / archive / replay output:

```text
=== RUN   TestCommittedEvidenceHygiene
    evidence_test.go:55: scanned 149 acceptance JSON records plus documentation
--- PASS: TestCommittedEvidenceHygiene (0.12s)
=== RUN   TestD098_Followup7Archive
--- PASS: TestD098_Followup7Archive (0.00s)
=== RUN   TestD098_Followup8Archive
--- PASS: TestD098_Followup8Archive (0.00s)
=== RUN   TestD098_Followup9Archive
--- PASS: TestD098_Followup9Archive (0.01s)
=== RUN   TestAT09_D126CapturedProvisionalSequence
=== RUN   TestAT09_D126CapturedProvisionalSequence/bare_safety_code_without_reason
=== RUN   TestAT09_D126CapturedProvisionalSequence/historical_tc_alone
=== RUN   TestAT09_D126CapturedProvisionalSequence/lease_cause_overwritten
=== RUN   TestAT09_D126CapturedProvisionalSequence/duplicate_lease_error
=== RUN   TestAT09_D126CapturedProvisionalSequence/expired_provisional
=== RUN   TestAT09_D126CapturedProvisionalSequence/passed_dependent_check
=== NAME  TestAT09_D126CapturedProvisionalSequence
    provisional_test.go:183: replayed 3 provisional reports (1 mixed) and retained 3 serving reports; historical tc evidence unchanged
--- PASS: TestAT09_D126CapturedProvisionalSequence (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/bare_safety_code_without_reason (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/historical_tc_alone (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/lease_cause_overwritten (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/duplicate_lease_error (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/expired_provisional (0.00s)
    --- PASS: TestAT09_D126CapturedProvisionalSequence/passed_dependent_check (0.00s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.136s
```

Actual handover output:

```text
=== RUN   TestHandover_Followup16Claims
=== RUN   TestHandover_Followup16Claims/members8_passes_and_AT09_failure
--- PASS: TestHandover_Followup16Claims (0.01s)
    --- PASS: TestHandover_Followup16Claims/members8_passes_and_AT09_failure (0.01s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.013s
```

Actual byte-comparison output:

```text
{"follow-up-8": 32, "follow-up-9": 33, "handover snapshots": 13}
```
