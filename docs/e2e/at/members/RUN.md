# Follow-up 11 run ledger

Run `7edb263f-1f6e-4f94-9e8e-26099f96a699`, harness `33abb2fd2c2fc95826c6e69eb434045522d6f7ef`, product baseline `96ec0630e4834e13a3a14d65553ba8108bb5da55` (devstack 20 #343, agent-core 28 #344, netlab 20 #336; dns-publish 8 #309). Assigned `w4-members / 24000`, two resolvers. The full mutation order is unchanged.

**2 complete cases pass / 6 blocked / 2 fail; 558 pass / 5 blocked / 3 fail subchecks.**

D-098 was first: commit `48933c7b` archives all **35** follow-up 10 root files byte-identically from baseline `96ec0630`, with SHA256SUMS. Follow-up 8/9 archives (32/33 files), all 146 handover files, the netlab history pin and unchanged D-126 provenance text still match. D-132's living-root/history boundary is preserved.

**F10-02 new finite-expiry row passes:** 186/186 reads, counter/status categories all `none`; last TCP/UDP increments at `18:34:36.591993718Z`, 408.006282 ms before the `18:34:37Z` deadline. Across both withdrawal rows, 214 samples and 428 before/after snapshots join 798 producer records with no gaps, zero clocks or missing records. The full stream includes 109 baseline, 323 heartbeat, 27 renewal, 93 apply, 218 live and 28 blocked-audit records. Existing `finishPersistent` is byte-identical; no retry or relaxation. Historical `13:35:45.333735552Z` remains failed and unattributed. Full AT02 fails **121/0/2** on separate F11-02/F11-03 checks.

**F10-03 new cache row passes:** AT20 is blocked **81/0/0**, with 39/39 complete traces uniquely matching primary. Initial positive is `other_ipv4`, TTL 10, observed authority exchange, separately sampled authority serial 1001. Before TTL it remains `other_ipv4`, TTL 9, with no authority exchange (inferred hit only), authority sample 1002. After TTL it is `private_ipv4`, TTL 10, with observed exchange, authority sample 1002. Each positive probe reports 1 ms; broader query-call brackets are approximately 42.558/33.577/40.232 ms, followed by separate trace collection. Negative before-TTL retains NXDOMAIN/SOA TTL 1 and resolver-receipt SOA serial 1001 while the authority is 1002; cache state stays unknown. No cached-A serial or private AdGuard flag is claimed. The historical 2,502-ms query and premature answer stay failed and unattributed. Dns-publish 9 is not in the run baseline.

**F10-04 is closed under D-129:** follow-up 10's unchanged generation-29 recovery (15.751594344 s serving, 21.357052110 s first byte, three provisional reports) qualifies at its yielded generation. Devstack 20 confirms it preserves product-issued generations and adds no mutation to force 30. The original generation-30 F8-01 closure remains immutable follow-up 9.

**Complete fresh AT09 fails 37/5/1.** The unchanged sequence yields desired/applied **30 → 30**, serves in **6.032389860 s** and first counts bytes in **11.399224550 s**. All three provisional reports pass D-126, including the mixed TC shape; the initial tc rejection is retained. The 90.451283001-second capture has 88 heartbeat samples and six observations, plus boot/final nft/tc and agent log. Agent-core 28's same-boot adoption/bounded verification changes are on this baseline; the different timing is measured, not causally attributed from ownership. All five unavailable QUIC/platform selectors remain blocked with their exact resource reasons. No forced generation, state repair or topology repair.

**F8-02 recurs:** dual/1280 returns **13,013/64,064 bytes in 12,010 ms**, direct P2P **IPv6 underlay** before/after, IPv4 payload. Endpoint write/read counts 64,108/13,081, 13 complete lines, zero partial-line bytes, timeout; client bytes_acked 64,109, notsent/retrans zero. Device deltas +66,216/+38,241. All three MTU-1500 rows pass 64,064 bytes in 2/1/1 ms. This is distinct from devstack 20's `cce1a688` IPv4 capture: 10,010/64,064 bytes in 12,011 ms, attempt 1 of 3; 19 reverse 1,340-byte outer datagrams absent, all 17 1,132-byte datagrams arrive, no PTB. Netlab 21 owns interpretation; no per-datagram capture or exact drop reason is claimed for this new members row.

| Case / evidence | Requirements | Full status | Pass / blocked / fail |
|---|---|---|---:|
| [AT01](AT01.json) | §19.1 §19 AT01 APP01 ID01 MEM01 | blocked | 16 / 0 / 0 |
| [AT02](AT02.json) | §19.1 §19 AT02 MEM01 MEM02 MEM03 SEC01 | fail | 121 / 0 / 2 |
| [AT03](AT03.json) | §19.1 §19 AT03 BILL01 BILL03 ENT01 | pass | 92 / 0 / 0 |
| [AT04](AT04.json) | §19.1 §19 AT04 BILL02 ENT02 MEM03 | blocked | 99 / 0 / 0 |
| [AT05](AT05.json) | §19.1 §19 AT05 NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 |
| [AT08](AT08.json) | §19.1 §19 AT08 NODE02 NODE03 EXIT01 EXIT03 | pass | 31 / 0 / 0 |
| [AT09](AT09.json) | §19.1 §19 AT09 EXIT02 APP01 NFR01 | fail | 37 / 5 / 1 |
| [AT20](AT20.json) | §19.1 §19 AT20 DNS01 DNS02 ISO04 | blocked | 81 / 0 / 0 |
| [AT21](AT21.json) | §19.1 §19 AT21 DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 |
| [AT22](AT22.json) | §19.1 §19 AT22 SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 |

Live interval: `2026-10-31T18:26:25.973239+00:00` → `2026-10-31T19:01:52.414257+00:00`; package **2126.014 s**, make exit **2**. All later listed gates ran sequentially in the foreground; no test scratch used tmpfs.

The first launch was an orchestration error: two other suites appeared during stack startup; the shell printed that admission check but still launched. The owned process group was stopped/terminated during AT05 setup, before AT02/AT20, and its incomplete AT05 evidence is archived under `history/follow-up-11/attempt-1/`. Its stack was torn down and reset. The final runner waited until one other suite remained and rechecked after startup; the complete run began from clean commit `33abb2fd`. AT02 and AT20 each ran once. No interrupted result was hidden or promoted.

After all ten cases completed, the live target's trailing `cmd/probe` build reported missing files in the shared Go build cache. The actor/cause of that disappearance is not established. The live target stays exit 2; its complete measured failures remain. Remaining gates used a private disk GOCACHE under `.agent/tmp/f11-go-cache` without purging or altering the shared cache; member/probe tests and both architectures subsequently pass. This is not a live rerun.

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
| committed-evidence hygiene, D-098 archives, D-126 replay, timing negatives | 0 |
| immutable handover members8 claim | 0 |
| baseline byte comparisons / owned Docker inventory | 0 |

Owned Docker inventory is **0 containers / 0 networks / 0 volumes / 0 images**. Teardown exited 0. Private stack state was deleted using the pinned cleanup image with `--pull=never`, `--network none`, only the assigned mount and an owned self-removing container. Generated stack builds and interrupted release/test scratch were removed. Shared `fvpn-pg`, other lanes' objects and host networking were untouched. No sudo or shell cleanup traps.

Count actual suites with `pgrep -af 'gates.sh|mk/at-'`; wait while two or more other suites run, and recheck after startup. Use disk scratch and the assigned prefix/port; do not reuse another lane's stack.

```sh
make -f mk/at-members.mk at-members-lint
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 FVPN_E2E_TWO_RESOLVERS=true E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
mkdir -p .agent/tmp/f11-go-cache
export GOCACHE="$PWD/.agent/tmp/f11-go-cache"
python3 tests/e2e/at/members/disktmp.py go test ./tests/e2e/at/members/...
python3 tests/e2e/at/members/disktmp.py make go-lint
python3 tests/e2e/at/members/disktmp.py go vet ./...
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
make -f mk/at-members.mk at-members-build
python3 tests/e2e/at/members/disktmp.py go test -count=1 -v -run '^(TestCommittedEvidenceHygiene|TestD098_Followup(7|8|9|10)Archive|TestAT09_D126CapturedProvisionalSequence|TestAT02_CounterTimingRejectsUnattributableSamples)$' ./tests/e2e/at/members
```

Actual live case output:

```text
--- FAIL: TestATMembers_Live (2125.84s)
    --- FAIL: TestATMembers_Live/AT05 (15.42s)
    --- FAIL: TestATMembers_Live/AT01 (56.50s)
    --- FAIL: TestATMembers_Live/AT02 (451.47s)
    --- PASS: TestATMembers_Live/AT03 (177.96s)
    --- FAIL: TestATMembers_Live/AT04 (596.83s)
    --- PASS: TestATMembers_Live/AT08 (108.99s)
    --- FAIL: TestATMembers_Live/AT20 (72.80s)
    --- FAIL: TestATMembers_Live/AT21 (36.95s)
    --- FAIL: TestATMembers_Live/AT22 (368.29s)
    --- FAIL: TestATMembers_Live/AT09 (234.64s)
```

Live trailing build failure (local paths redacted):

```text
$GOMODCACHE/golang.org/x/net@v0.62.0/idna/idna.go:19:2: could not import fmt (open $GOCACHE/ef/efb97225890f490e10576186f1fe19094c0bcfbfc5b8d293abb6d6240771784a-d: no such file or directory)
$GOMODCACHE/golang.org/x/net@v0.62.0/idna/idna.go:20:2: could not import strings (open $GOCACHE/3b/3b6f158ac04ae1e899e2f7a17165370c1251ebc42c5ea993c52103991420a11f-d: no such file or directory)
FAIL github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe [build failed]
```

Actual offline output:

```text
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.267s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members/cmd/probe	0.002s
```

Actual hygiene / replay tail:

```text
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
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/members	0.273s
```

Actual immutable handover output:

```text
=== RUN   TestHandover_Followup16Claims
=== RUN   TestHandover_Followup16Claims/members8_passes_and_AT09_failure
--- PASS: TestHandover_Followup16Claims (0.01s)
    --- PASS: TestHandover_Followup16Claims/members8_passes_and_AT09_failure (0.01s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	0.012s
```

[Summary](summary.md), [native/DNS rows](NATIVE.md), [nine dated request groups](requests.md). Lint, vet, formatting, diff and cross-build logs had empty stdout and exit 0. No dependency or product source changed.
