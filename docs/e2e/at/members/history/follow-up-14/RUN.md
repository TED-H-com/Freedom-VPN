# Follow-up 14 run ledger

Run `258ba504-6ddf-4888-8633-9cfee8da6eea`, clean compiled harness `e9a215f3c26484d92c0293d6070810a4b1d1e56d`, product baseline `8f3055b9fbbbc1e1522e57ce69d6eede5da9fa56` (devstack 23 #367, agent-core 31 #369, devstack 24 #370).

`go env GOCACHE` returned `/data/ted-h-cache/go-build`; no cache override or purge. Assigned prefix `w4-members`, port base `24000`, two resolvers.

**2 PASS / 5 BLOCKED / 3 FAIL**, with **628 passed / 6 blocked / 5 failed subchecks**. One assembled ten-case run; live make exit **2**.

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | §19.1 §19 AT05 NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 30 / 0 / 0 |
| [AT01](AT01.json) | §19.1 §19 AT01 APP01 ID01 MEM01 | blocked | 22 / 0 / 0 |
| [AT02](AT02.json) | §19.1 §19 AT02 MEM01 MEM02 MEM03 SEC01 | fail | 137 / 0 / 2 |
| [AT03](AT03.json) | §19.1 §19 AT03 BILL01 BILL03 ENT01 | pass | 95 / 0 / 0 |
| [AT04](AT04.json) | §19.1 §19 AT04 BILL02 ENT02 MEM03 | fail | 116 / 0 / 1 |
| [AT08](AT08.json) | §19.1 §19 AT08 NODE02 NODE03 EXIT01 EXIT03 | pass | 34 / 0 / 0 |
| [AT20](AT20.json) | §19.1 §19 AT20 DNS01 DNS02 ISO04 | blocked | 88 / 0 / 0 |
| [AT21](AT21.json) | §19.1 §19 AT21 DNS03 DNS04 APP01 | blocked | 12 / 0 / 0 |
| [AT22](AT22.json) | §19.1 §19 AT22 SEC02 DNS01 DNS04 | blocked | 39 / 0 / 0 |
| [AT09](AT09.json) | §19.1 §19 AT09 EXIT02 APP01 NFR01 | fail | 55 / 6 / 2 |

D-098 was the first, separate commit `cc04d0c2`: all **61 root files** from run 4182dd3d are byte-identical in [follow-up 13](history/follow-up-13/RUN.md), with [SHA-256 pins](history/follow-up-13/SHA256SUMS). The retained AT09-restart.json still belongs to run 7edb263f. Follow-up-13-captures, follow-up-13-stream and every earlier archive are untouched. Handover claims were checked: tests already read frozen history files; no outside-owned D-132 exception was needed.

**Harness qualification limit:** the compiled harness adds the routed hop before the original eth0-only relay hold. The added member interface can bypass that hold; the original relay row fails with `relayed=1 direct=1`. This is a confounded harness result, not proof of a product relay regression. Commit `682d4434` moves the routed rows after the relay check. The correction passes offline gates but is **not live-qualified**; no second assembled run was made.

**D-136, corrected:** a verified deny-all candidate is used only while an Apply error is retained. The ordinary stale-lease path installs the node-wide baseline, ACKs, renews, then Applies, verifies and heartbeats. No finite outage ceiling follows. F12-01 is labeled **policy_missing baseline during generation 11→13; stale-lease path consistent with code; no failed Apply**. `cause_status=unproven` until the desired input is joined. Netlab 21 request 3 is settled by D-135; this is a cross-reference, not a renewed request.

AT02 retains **96 successful Applies, zero failed Applies**, with validated `commands` on all 96. Missing historical fields read **unavailable**; original run/sequence identities and historical replay remain intact. F11-03 stays blocked on no failed Apply observed. All **58 baseline calls** are paired to successful recovery: 55 outside the sibling capture and three fully covered. The continuous stream has **3,941 attempts / 3,941 ingresses / 3,920 egresses / 21 losses**, **14 inside completed guards / seven outside**. Nothing is excluded or waived. Baseline 410 → Apply 414 spans **215.189032 ms**, with eight losses (one inside / seven outside); nested baselines 411/412 overlap and are never summed as unique losses. Renewal 413 moves lease 11→13; live 415 and heartbeat 416 follow. First post-Apply egress is **15.068322 ms** later, not a claimed outage ceiling or first recovery packet. [Accounting and exact records](F14-AT02-recovery.json), [transition](F14-AT02-policy-missing.json).

**F14-01:** the first D-077 re-grant has populated generation-10 readiness and an intended-device producer join, but its original forwarding attempt fails: five member packets, no observed node/origin ingress, zero origin requests and unchanged device counters. The capture stops attribution between member inner egress and node inner ingress; no detailed native capture exists for this attempt. Cause remains unproven and is not assumed identical to F11-01. [Original failed row and preceding readiness](F14-01-regrant-forwarding.json).

**F13-01 fresh sample:** primary member-expiry1 has one exit-0 `nft_parse` / `parse` rejection among 266 withdrawal samples. Raw output and exact parser error are retained privately (0600); public receipt: **34,273 bytes / one line**, SHA-256 `2e13a93c901408906614345c7e2be3457751c7877835b991d115285f284ccc3a`. The raw output is valid JSON with four duplicated, identical chain definitions, including handles. `node/enforce/parse_nft.go` rejects repeated chain identity. This is a parser compatibility candidate for netlab, not a proven parser defect: why the snapshot contains duplicates and whether they are allowed remain unproven. Successful Apply 942 overlaps the 18.668818 ms read; that overlap does not establish cause. Active senders, stable counters and the observed 0.148039349 s last reply increment cannot repair an incomplete read. No retry or predicate relaxation. Both historical F13-01 failed rows stay failed and cannot be attributed using this new raw sample. [Safe receipts, exact failed tick and producer window](F14-F13-01-private-receipts.json).

AT09’s known-PMTU row transfers **64,064 bytes**, retains route MTU 1280 before/after traffic and exact removal, with **16 delivered reverse IPv6 fragment pairs**. Routed IPv6 transfers **64,064 bytes in 1 ms**: one 1,360-byte router ingress causes a type-2/code-0 PTB with MTU 1280; its identical packet digest appears at router and sender. The sender route changes from no MTU metric to learned **1280**, and all 20 subsequent fragment packets arrive. Routed IPv4 transfers **64,064 bytes in 2 ms**: 42 oversized **DF=0** datagrams become 84 fragments in 42 groups, all received, with no PTB. These are separate D-137 mechanisms over IPv4 overlay payloads; D-123 IPv6Block still applies. [Original rows and mechanism evidence](F14-D137-native-paths.json).

Receive-only 1280 remains a **limitation, never PASS**: dual-family selection transfers 64,064 bytes but is BLOCKED; IPv6 returns only **12,012 bytes in 12,011 ms** and stays FAIL. Nineteen reverse 1,360-byte packets are absent at member ingress, with no PTB; the first missing TCP range is retained. Historical F8-02 rows remain failed.

**F12-04:** capture begins before the original readiness/RR query and preceding matrix mutations. The one original RR query succeeds (wire ID **62890**, RCODE 0, **1 ms**). Both resolvers retain complete lifecycle histories, outstanding sets and reply digests; the secondary is genuinely idle, not missing. Exact primary reply bytes join primary → node → member tunnel; the local stub response has the original wire ID. All five points are complete, zero-drop (**1,030 / 1,670 / 150 / 278 / 6** records for member/node/origin/primary/secondary). The positive IDNA probe also succeeds over UDP (**1 ms**) and TCP (**2 ms**). Historical F12-04 remains **unattributed**; this success does not explain its 5,001 ms SERVFAIL. [Original query, mutations and lifecycle](F14-F12-04-original-dns.json), [header analysis](history/follow-up-14-captures/258ba504-6ddf-4888-8633-9cfee8da6eea/native-dns-delivery-analysis.json). Devstack 24’s capture and lifecycle handoff is consumed; no additional independent positive query is requested.

F12-03’s quiet window lasts **6.006144294 s**, followed by fresh negative SOA TTL **5** and a before-TTL query inside the receipt-plus-four-second conservative boundary. The unchanged combined assertion passes. [Original cache evidence](F14-AT20-cache.json). Historical F12-03 and F10-03 stay failed/unattributed. F12-02’s large-stream reader remains accepted with additive lifecycle records; AT02 alone exports **20,533,943 bytes / 858 records**.

AT01 keeps its own **BLOCKED** result: initial selection and sleep/wake transfers are counted; first selection has one SYN and complete zero-drop captures (**274 / 322 / 80** member/node/origin records). The captured type-2 response is IPv4 loopback; this run has no matching IPv6 `::1` response to diagnose. Devstack 24 located historical F11-01 before member IPv6 prerouting, with no repair on main. This fresh success proves neither historical cause nor repair. [Original first-selection analysis](history/follow-up-14-captures/258ba504-6ddf-4888-8633-9cfee8da6eea/initial-selection-analysis.json).

| Original action | Recorded duration | Stored lease evidence |
|---|---:|---|
| AT02 supervisor stop | 0.467624715 s | unchanged closed-storage readback |
| AT02 supervisor resume | 0.471370722 s | immediate snapshot renewed; new producer first retains the prior lease before renewal |
| AT08 disabled-node docker restart | 10.110052517 s | old process renews during stop; new producer first retains that prior-run lease before renewal |
| AT09 selected namespace stop | 5.437043416 s | unchanged closed-storage readback |
| AT09 namespace restart + observation | 90.470845850 s | includes the required 90-second capture; new producer first retains prior-run lease before renewal |

[Original command timestamps, stored readbacks and joined producer windows](F14-agent-transitions.json).
These bounded observations do not prove repeated-restart reliability or an exact graceful-shutdown duration.
AT09 keeps desired/applied **30→30**, fresh serving **6.111632611 s** and first counted byte
**11.624517030 s** after the owner capture start. Original provisional and tc errors remain
in [restart evidence](F14-AT09-restart.json); recovery does not erase them.

Startup `2026-11-01T09:57:50.549514+00:00`; assembled teardown completed `2026-11-01T10:37:15.397373+00:00`. Admission initially saw two other suites and waited; pre-start, post-start and live-target checks accepted with at most one other suite. Sequence AT05 → AT01 → AT02 → AT03 → AT04 → AT08 → AT20 → AT21 → AT22 → AT09. No second assembled attempt, manufactured failed Apply, predicate relaxation, or host network change.

All gates ran sequentially in the foreground. Area lint, package tests, whole-module `make go-lint`, `go vet ./...`, tracked gofmt, diff checks, four helpers for linux/amd64 + linux/arm64, historical replay and committed-evidence hygiene exited **0**. Live make exited **2**. Area lint was repeated after the ordering correction. The final evidence hygiene/diff check is repeated after this documentation update. No dependency changed.

```sh
make -f mk/at-members.mk at-members-lint
python3 tests/e2e/at/members/disktmp.py python3 -B tests/e2e/at/members/assembled.py
python3 tests/e2e/at/members/disktmp.py go test ./tests/e2e/at/members/...
python3 tests/e2e/at/members/disktmp.py make go-lint
python3 tests/e2e/at/members/disktmp.py go vet ./...
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
make -f mk/at-members.mk at-members-build
python3 tests/e2e/at/members/disktmp.py go test -count=1 -v -run '^(TestCommittedEvidenceHygiene|TestD098_.*|TestAT02_F14.*|TestAT02_F13.*|TestAT04_F13.*)$' ./tests/e2e/at/members
```

Actual live case output:

```text
--- FAIL: TestATMembers_Live (2289.53s)
    --- FAIL: TestATMembers_Live/AT05 (15.39s)
    --- FAIL: TestATMembers_Live/AT01 (75.04s)
    --- FAIL: TestATMembers_Live/AT02 (426.92s)
    --- PASS: TestATMembers_Live/AT03 (199.81s)
    --- FAIL: TestATMembers_Live/AT04 (612.31s)
    --- PASS: TestATMembers_Live/AT08 (110.38s)
    --- FAIL: TestATMembers_Live/AT20 (81.59s)
    --- FAIL: TestATMembers_Live/AT21 (32.07s)
    --- FAIL: TestATMembers_Live/AT22 (368.76s)
    --- FAIL: TestATMembers_Live/AT09 (361.33s)
```

Teardown returned 0; final owned inventory is **0 containers / 0 networks / 0 volumes / 0 images**. Intermediate network-in-use notices were followed by cleanup. Owned stack state was deleted using a self-removing `fvpn-w4-members-cleanup` container, existing public PostgreSQL image, no network and only the owned state mount. Shared PostgreSQL, other lanes, host networking and toolchain caches were untouched. No sudo, tmpfs or shell cleanup traps. Private pcaps and parser output/error files intentionally remain in ignored disk scratch.

Eleven captures / 37 points are complete and zero-drop. [108-file header archive](history/follow-up-14-captures/258ba504-6ddf-4888-8633-9cfee8da6eea/SHA256SUMS); [nine dated requests](requests.md). No push, PR, merge or tag.
