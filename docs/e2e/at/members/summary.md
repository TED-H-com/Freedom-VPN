# Follow-up 23 results — 2026-11-04

Run `6a25221b-1dcf-4f4d-b866-cd04f57f58f6`, measured source `8bde2b6acc0b116515148eeb0ff4448c65453a92`, baseline/current main at preflight `c148cf349df219124e457113ce90c010016a308e`. **3 PASS / 7 BLOCKED / 0 FAIL; 706 passed / 8 blocked / 0 failed subchecks**. No predicate changed, no case retried. [All original subchecks](F23-subchecks.md), [result index](F23-results.json), [verification receipts](F23-verification.md).

D-098 was committed first as `2d4d37a8`: [122 original root records](history/follow-up-22/INDEX.md), including unchanged gzip and SHA-256 pins. Run `3eb3960d` remains **3 / 7 / 0 cases; 702 / 8 / 0 subchecks**, measured on `55f2765d`; the orchestrator accepted that baseline without a later-main rerun. Run `5715939e` and every earlier archive/note retain their original outcomes.

| Case | Requirements (§19 acceptance) | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 31 / 0 / 0 |
| [AT01](AT01.json) | APP01 ID01 MEM01 | blocked | 23 / 0 / 0 |
| [AT02](AT02.json.gz) | MEM01 MEM02 MEM03 SEC01 | pass | 193 / 0 / 0 |
| [AT03](AT03.json) | BILL01 BILL03 ENT01 | pass | 98 / 0 / 0 |
| [AT04](AT04.json) | BILL02 ENT02 MEM03 | blocked | 113 / 0 / 0 |
| [AT08](AT08.json) | NODE02 NODE03 EXIT01 EXIT03 | pass | 36 / 0 / 0 |
| [AT20](AT20.json) | DNS01 DNS02 ISO04 | blocked | 89 / 0 / 0 |
| [AT21](AT21.json) | DNS03 DNS04 APP01 | blocked | 13 / 0 / 0 |
| [AT22](AT22.json) | SEC02 DNS01 DNS04 | blocked | 40 / 0 / 0 |
| [AT09](AT09.json) | EXIT02 APP01 NFR01 | blocked | 70 / 8 / 0 |

AT01 and AT08 initial transfers are counted. The old AT01 response reached member underlay ingress with an IPv6 loopback source; the first unconfirmed boundary is ingress → WireGuard accounting. Native control observations arrive about every 30 s because the regular-node config omits the installer-only 5 s override. The new stale row is dual/IPv4, age 22.606093 s, and remains blocked under the unchanged 15 s bound. Per-submission outcomes remain unavailable from existing agent logs. [Full dated analysis](followup23-2026-11-04.md).

AT02's single closed recovery: Apply 544 → renewal 549 → populated lease → serving Apply 550; deny-all **179.481443 ms / seven losses**. AT09 indexes **25 BLOCKED, 26 PASS, 27 PASS**; D-067 provisional predicate unexercised. No retained F48-01 signature; **zero node_write conflicts** across 30 receipts. D-133 overlap occurs only in the recorded teardown window; no retry. Cleanup leaves zero owned Docker inventory.

# Follow-up 22 results — 2026-11-04

Run `3eb3960d-c411-4648-8d77-bba40b436ae3`, measured source `8a6dcf34248f035a3a460fcdee3e1e19e80c9137`, baseline `55f2765dac6e1c6871045ee6182d6cc23ccf8a33` (agent-core 49 #454 and 50 #461). **3 PASS / 7 BLOCKED / 0 FAIL; 702 passed / 8 blocked / 0 failed subchecks**. Every predicate remains unchanged; every observed case and subcheck is retained in [the result index](F22-results.json) and [the complete subcheck list](F22-subchecks.md).

This is the single permitted retry after [attempt 1](F22-attempt1.json) stopped in this lane's admission parser before any acceptance check. The parser correction is `8a6dcf34`, with offline tests. Attempt 1 has no acceptance run UUID or case result. No failed case or overlap was retried. Both attempts' cleanup inventories are retained.

D-098 was committed first: [98 original root records and byte pins](history/follow-up-21/INDEX.md), including unchanged gzip, preserve run `5715939e` at **2 PASS / 8 BLOCKED / 0 FAIL; 705 / 8 / 0**. Run 2 and every earlier archive retain their outcomes. The [G2 relative alias](../poc/G2-followup34-2026-11-03.md) resolves the three frozen run-2 links; no original is rewritten.

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](AT05.json) | NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 31 / 0 / 0 |
| [AT01](AT01.json) | APP01 ID01 MEM01 | blocked | 19 / 0 / 0 |
| [AT02](AT02.json.gz) | MEM01 MEM02 MEM03 SEC01 | pass | 193 / 0 / 0 |
| [AT03](AT03.json) | BILL01 BILL03 ENT01 | pass | 98 / 0 / 0 |
| [AT04](AT04.json) | BILL02 ENT02 MEM03 | blocked | 113 / 0 / 0 |
| [AT08](AT08.json) | NODE02 NODE03 EXIT01 EXIT03 | pass | 36 / 0 / 0 |
| [AT20](AT20.json) | DNS01 DNS02 ISO04 | blocked | 89 / 0 / 0 |
| [AT21](AT21.json) | DNS03 DNS04 APP01 | blocked | 13 / 0 / 0 |
| [AT22](AT22.json) | SEC02 DNS01 DNS04 | blocked | 40 / 0 / 0 |
| [AT09](AT09.json) | EXIT02 APP01 NFR01 | blocked | 70 / 8 / 0 |


## AT02 — D-157 in the recorded closed recovery

AT02 passes all **193** original checks. Its unchanged D-135/D-136 classifier identifies **one** verified closed recovery, producer `759a30f5-5100-4368-bd6e-263fde797c04`. [Original joined sequence](F22-AT02-closed-recovery-trace.json), [loss accounting](F22-AT02-recovery.json), [complete original accounting](F18-AT02-recovery.json.gz).

| Event | Original sequence / time | Result |
|---|---|---|
| Previous renewal | 585; request `08:28:33.292550804Z` | Generation 13, zero members |
| Verified closed Apply | 586; monotonic `[104355768797947,104355907779743)` ns; end anchor `08:28:33.456502139Z` | Verified deny-all; Apply duration 138.981796 ms, six lost probes |
| Next renewal | 591; request `08:28:33.524221152Z`, monotonic `104355975597809` ns | **67.818066 ms after closed Apply ends**, 231.670381 ms after request 585 |
| Populated lease | Issued `08:28:33.526081953Z`; `d9f2451c-7731-4e1d-bec2-ae3094af6966` | Generation 13, one member |
| Verified serving Apply | 592; monotonic `[104356017218047,104356113147228)` ns; end anchor `08:28:33.661854145Z` | Same populated lease; succeeded and verified |

There is exactly **one renewal request** and **zero baseline writes** between closed Apply 586 and serving Apply 592. Deny-all after the closed Apply lasts **205.367485 ms**, with **eight** lost probes; closed Apply plus deny-all is **344.349281 ms / 14 losses**. End anchors are wall/monotonic mapping receipts, not exact kernel commit times. The actual install subinterval and every command boundary remain in the original trace.

This observed early request, before D-067's next grant-less slot, is the behavior introduced by agent-core 50 **#461 / D-157**. The run shows one early request for its one classified closed recovery. It does not retain the accepted clean report's HTTP receipt or central readiness commit time, so it cannot independently prove the earliest useful request time, every internal eligibility condition, or behavior after an empty early answer. The historical 5715939e measurement stays **4,968.068672 ms / 241 deny-all losses** (closed Apply plus wait: 248 losses); no old loss is removed. No timing-only attribution is made for other intervals or for #454.

The complete sibling capture retains **6,777 sends / 1,014 losses**, no send errors or unplaced packets. **985** lost sends have no node-ingress observation during interface-down intervals; **29** have node ingress without egress. Three recovery brackets contain 9, 10 and 995 losses, with durations **221.733530 ms**, **255.639505 ms** and **20,358.258641 ms**. Five losses fall in completed guard intervals and 1,009 outside them; none lie outside both baseline/recovery brackets and completed guards. All three failed Applies are bracketed. These are observed classifications; the 100 ms install budget is not an outage ceiling and the accounting's cause remains `unproven`.

## AT08 — F21-AT08 evidence gap supplied

The first selected-route transfer is now counted: **one origin receipt**, upload **317,593 → 319,858 bytes** and download **22,727 → 25,140 bytes**, generation 62. Its original record starts at `08:52:34Z`; first counted-byte receipt is `08:52:42.523054514Z`. AT08 passes **36** subchecks, including the added capture completeness check. Origin receipt plus both counter increments remain the only forwarding success predicate. [Original transfer, both peer views and headers](F22-AT08-forwarding.json), [capture pins](F22-AT08-capture/SHA256SUMS).

| View | Before transfer | After transfer |
|---|---|---|
| Member sees selected node | `08:52:41.784800298Z`: found, idle, transport unknown; handshake absent; rx/tx 0/0 B; latency 0 ns; route `0.0.0.0/0` included | `08:52:42.630905552Z`: connected/direct; handshake age 325,797,792 ns; rx/tx 2,908/2,852 B; latency 170,761 ns; selected route included |
| Node sees member | `08:52:41.879690141Z`: found, idle, transport unknown; handshake absent; rx/tx 0/0 B; latency 0 ns; no route prefixes | `08:52:42.723733691Z`: connected/direct; handshake age 418,424,836 ns; rx/tx 2,852/3,056 B; latency 120,894 ns; no route prefixes |

The node view is available through the existing container hook; no shared export changed. Only validated route prefixes and allowlisted peer metrics persist, with byte/duration values as decimal strings. Keys, endpoint addresses, FQDNs and identity fields are discarded; the full native status is never retained. Offline tests exercise realistic nested identity fields, missing selected peers and malformed input.

The original `at08-initial-selection` capture contains **176 member / 268 node / 56 origin** packets, with **zero kernel drops, truncation, decoder errors or unparsed records**, and all capture processes stopped. The bounded TCP flow's client port 20033 yields **16 / 32 / 16** decoded headers at those three roles. These are the original transfer's packets, with no repeated request.

This resolves F21-AT08's missing instrumentation, not the cause of its old failure. Run 5715939e still records five member-interface packets, none at the node/origin and unchanged counters. Its selected peer state was not retained. The successful new row supplies no causal attribution to #454, #461 or another merged change; it does not close F11-01.

## Native MTU — D-137, D-146 and unchanged D-151 qualification

[Every original native row and qualification](F22-native-MTU.json) retains bytes, MSS, reverse datagrams, rule reads and all original pass/blocked predicates. Payloads remain IPv4 over the selected native underlay under D-123; no end-to-end IPv6 certification is claimed.

| AT09 index | Original row | Bytes / milliseconds | Observed status |
|---:|---|---:|---|
| 35 | native IPv4 MTU 1500 | 64,064 / 2 | PASS |
| 38 | native IPv6 MTU 1500 | 64,064 / 1 | PASS |
| 41 | native dual MTU 1500 | 64,064 / 1 | PASS |
| 44 / 45 / 46 | dual receive-only 1280; upload; D-146 | 64,064 / 2 download; full upload | PASS / PASS / qualified |
| 49 / 50 / 51 | IPv6 receive-only 1280; upload; D-146 | 64,064 / 1 download; full upload | BLOCKED / PASS / unqualified |
| 54 | D-137 known-PMTU 1280 | 64,064 / 1 | PASS |
| 57 / 58 / 59 | IPv4 receive-only 1280; upload; D-146 | 64,064 / 1 download; full upload | PASS / PASS / qualified |
| 73 | routed IPv6 1280 | 64,064 / 2 | PASS |
| 76 | routed IPv4 1280 | 64,064 / 1 | PASS |

For all three receive-only rows, member SYN MSS **1240 → 1160** and origin SYN-ACK MSS **1460 → 1160** pass, with no PTB at any of the three capture points. All reverse datagrams fit 1280 and arrive: dual **75/75**, maximum IP length **1260**; IPv6 **74/74**, maximum **1152**; IPv4 **76/76**, maximum **1132**. Both byte predicates pass. Dual and IPv4 have successful combined serving reads and qualify. The IPv6 row's only failed qualification check remains `serving_rules_before_new_handshake`.

**F21-D146: new later-read evidence.** The IPv6 combined serving read again returns no row (`serving_read_ok=false`). The second read at **09:06:41.098555Z**, explicitly marked **later**, finds desired/applied/kernel/observation generation **75**, the matching current incarnation, `member_forwarding_enabled=true` and `forwarding_block_reasons=[]`. Latest current and latest any-incarnation observations are the same report, received **09:06:18.705372Z**: **22.393183 s old**, exceeding the strict **15 s** bound. Its sole failed condition is `observation_age_below_15_seconds`. This explains the later snapshot; it does not turn the original combined query into a success or prove an identical preceding snapshot. No waiting, retry, age-bound change or D-151 predicate change was made.

**Historical indexes 49 and 51 remain indeterminate.** Run 5715939e retained only the failed combined read, not its failed conjuncts. Neighboring successful observation receipts cannot establish the intervening row/state. [Retrospective extraction and pins](F22-F21-D146-retrospective.json). The diagnosis is in this lane's own read code; no shared-helper fault is inferred. Any policy for observation freshness is for the orchestrator/owner, not a predicate fix here.

UDP/QUIC without PTB remain limited. Native QUIC and the four physical client platforms are still blocked owner-resource rows; relay-to-direct continuation passes with the original sockets. There is no physical arm64, roaming or complete PMTU claim.

## AT09 restart, watchdogs and writes

AT09 stays **70 passed / 8 blocked / 0 failed subchecks**. [Original restart records](F22-AT09-restart.json):

- **Index 25 BLOCKED:** zero grant-less leases and zero provisional observations; populated from the first renewal, seven grants. D-067's provisional restart predicate is **not exercised**, never satisfied vacuously.
- **Index 26 PASS:** desired/applied **65 → 65**; serving-ready **5.989982876 s**, first counted byte **12.622674621 s**, from resume `09:03:48.027513124Z`. Four post-restart reports are retained; the first populated nonserving report has `HEALTH_CHECK_FAILED` at `tunnel_probe`, then forwarding recovers.
- **Index 27 PASS:** all four post-restart reports have no `TC_FOREIGN_FILTER`. The namespace restart removes old kernel objects; it does not test retaining an existing namespace.

[Watchdog/agent excerpts with original line numbers and file hashes](F22-restart-log-review.json) retain snapshots between every case and final teardown. Initial node and node2 watchdog logs, lines **1–3**, show `heartbeat_unreadable` decisions and successful baseline writes at **08:21:47.849175570Z** and **08:31:57.757940597Z**. AT02's agent restart/resume has `serving rules adopted`, generation **26**, at **08:31:55.347Z**, agent log **line 2**; the retained watchdog file has no later decision after its initial startup. AT08's disabled-node container restart leaves an empty exported watchdog log. AT09 node2 watchdog **line 1** records `baseline_unreadable` at **09:03:48.711558383Z**, and **lines 2–3** record its successful write; this is the full namespace restart.

No retained snapshot contains `baseline_unverified` followed by `GENERATION_MISMATCH`; no F48-01 signature is observed. The warm adoption receipt is consistent with the retained-rule path, but these records do not establish that #454 alone changed an outcome or provide complete overwritten intra-case logs. F48-01 is not independently closed by this run.

**Every `node_write`:** [30 distinct original receipts](F22-node-writes.json), each with exactly one attempt and HTTP **202**; **zero 412 conflicts**, zero exhausted retries. All receipt locations and attempts remain in the index.

## Differences, limits and carried ownership

[Status/check inventory comparison to 5715939e](F22-status-changes.json) keeps both runs intact. AT08 changes BLOCKED → PASS and adds one native-capture check. AT01's first bounded transfer changes counted true → false; its conditional sleep/wake continuation is not reached, leaving four fewer passed checks. Thus the total is **702**, versus **705** previously; no predicate was removed. AT01 still has no origin receipt and unchanged counters. Its peer views begin idle and end connected/direct but without a reported WireGuard handshake. The native cause remains unproven. UUID-only capture labels change with fresh resources. All other aligned subcheck statuses remain unchanged; numeric timings are observations, not causal attribution.

F11-01 remains **OPEN and upstream-dependent under D-148**, with no product change. F4-03, F4-04, F4-05/F4-07, F4-06 and F4-10 remain their existing owner-resource groups and are not re-asked.

The retry measured the assigned **55f2765d** baseline including agent-core 49/50. At retry preflight, `main` had advanced to **0d944880** with netlab follow-up 35 #462. [Exact baseline drift](F22-baseline-drift.json) records the five netlab code paths and documentation difference. This is **not** claimed as a run of that newer main; no further live retry is authorized. Orchestrator disposition is requested.

The only observed second-attempt overlap is **09:08:22.475474Z–09:08:52.654917Z**, after acceptance ended at **09:08:08.816545Z**. That teardown window is **D-133 confounded**; no measured case envelope intersects it. Five-second sampling cannot exclude shorter activity. All **308** timed-section bounds and **6,852** live-tool/admission checks are retained; no held check or admission sleep occurs inside a timed section. No overlap or failed case was retried.

---

Historical text below keeps its outcomes; superseded root evidence links resolve to the byte-pinned archive.

# Follow-up 21 results — 2026-11-03

Run `5715939e-3943-48f1-8bd5-c869bf7c91c2`, measured source `dbfccc6fc7d076ec73f4f8575eb27efe39337a23`, baseline `9e70951872105139213ed39c808704fdab315112` (agent-core 48, #447). One assembled run, no rerun: **2 PASS / 8 BLOCKED / 0 FAIL; 705 passed / 8 blocked / 0 failed subchecks**. Live make exit **2**, because blocked cases remain non-green. [Every case and subcheck](history/follow-up-21/F21-subchecks.md), [machine results](history/follow-up-21/F21-results.json).

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](history/follow-up-21/AT05.json) | NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 31 / 0 / 0 |
| [AT01](history/follow-up-21/AT01.json) | APP01 ID01 MEM01 | blocked | 23 / 0 / 0 |
| [AT02](history/follow-up-21/AT02.json.gz) | MEM01 MEM02 MEM03 SEC01 | pass | 193 / 0 / 0 |
| [AT03](history/follow-up-21/AT03.json) | BILL01 BILL03 ENT01 | pass | 98 / 0 / 0 |
| [AT04](history/follow-up-21/AT04.json) | BILL02 ENT02 MEM03 | blocked | 113 / 0 / 0 |
| [AT08](history/follow-up-21/AT08.json) | NODE02 NODE03 EXIT01 EXIT03 | blocked | 35 / 0 / 0 |
| [AT20](history/follow-up-21/AT20.json) | DNS01 DNS02 ISO04 | blocked | 89 / 0 / 0 |
| [AT21](history/follow-up-21/AT21.json) | DNS03 DNS04 APP01 | blocked | 13 / 0 / 0 |
| [AT22](history/follow-up-21/AT22.json) | SEC02 DNS01 DNS04 | blocked | 40 / 0 / 0 |
| [AT09](history/follow-up-21/AT09.json) | EXIT02 APP01 NFR01 | blocked | 70 / 8 / 0 |

AT09 index **25 BLOCKED / not exercised**: zero grant-less leases and zero provisional observations after restart; one populated-lease non-serving report passes the held-lease safety predicate. Index **26 PASS**: generation 67 → 67, populated from the first renewal, serving ready in **5.961887299 s**, counted recovery in **12.525243315 s**. New index **27 PASS**: no `TC_FOREIGN_FILTER` in four post-restart reports. The observed non-serving report contains only the tunnel-probe safety error; its lease/QoS/source-binding checks pass and `tc_evidence` is null. D-067's restart provisional predicate is **not exercised**, never promoted to pass. [Original restart checks](history/follow-up-21/F21-AT09-restart.json).

The retained AT09 watchdog decision at **04:05:48.281765307Z** is `cause=baseline_unreadable`, followed by its successful baseline write. It is not F48-01's `baseline_unverified` successor-heartbeat signature; the retained agent logs contain no `GENERATION_MISMATCH`. Namespace recreation removes kernel state and does not prove warm adoption. Earlier overwritten logs cannot exclude a prior same-boot race; F48-01 is not closed. [Exact log lines and pins](history/follow-up-21/F21-restart-log-review.json).

AT02 indexes **190 PASS** (accounting) and **191 PASS** (sibling recovery). Every one of **6,165 sends**, with **0 send errors**, is retained; **985** are absent at node ingress during verified interface-down bounds, **252** reach ingress without egress, and all **1,237 losses** count. Three failed Applies are bracketed; no loss lies outside the measured baseline/guard accounting. The two complete recovery brackets are:

| Baseline → serving Apply | Cause | Duration | Losses | Recovery observed |
|---|---|---:|---:|---|
| 405 → 409 | policy_missing | 218.616067 ms | 8 | yes |
| 542 → 564 | foreign_tc_quarantine (injected stimulus) | 25.118880429 s | 1,229 | yes |

Closed-recovery Apply **558** takes **133.039428 ms / 7 losses**, followed by **4.968068672 s / 241 losses** of deny-all until serving Apply 564; the whole interval is **5.101108100 s / 248 losses**. No loss is waived and no outage ceiling is invented. There is **no baseline write** between 558 and 564. Renewals 557→563 start **5.000559819 s** apart, 559,819 ns after the five-second grant-less slot. Failed Applies 545/549/553 each have one recorded quarantine baseline (546/550/554), followed by verification/heartbeat. [All brackets and closed-recovery accounting](history/follow-up-21/F21-AT02-recovery.json), [original sequences 540–566](history/follow-up-21/F21-AT02-closed-recovery-trace.json), [complete original measurement](history/follow-up-21/F18-AT02-recovery.json.gz).

| Index | Native row | Status | Bytes / ms |
|---:|---|---|---:|
| 35 | native ipv4 MTU 1500 transfer | pass | 64064 / 1 |
| 38 | native ipv6 MTU 1500 transfer | pass | 64064 / 1 |
| 41 | native dual MTU 1500 transfer | pass | 64064 / 1 |
| 44 | native dual MTU 1280 transfer | pass | 64064 / 2 |
| 49 | native ipv6 MTU 1280 transfer | blocked | 64064 / 1 |
| 54 | D-137 supported known-PMTU 1280 transfer | pass | 64064 / 2 |
| 57 | native ipv4 MTU 1280 transfer | pass | 64064 / 1 |
| 73 | D-137 routed ipv6 1280 transfer | pass | 64064 / 2 |
| 76 | D-137 routed ipv4 1280 transfer | pass | 64064 / 1 |

D-146 receive-only dual/IPv4-selected and IPv4-only rows qualify; IPv6 remains **BLOCKED** at indexes 49 and 51. All three transfer 64,064 payload bytes with 64 exact echoes and pass upload; SYN MSS **1240→1160** and SYN-ACK **1460→1160** are observed in each. Reverse datagrams are **72/72, max 1132 B** (dual), **73/73, max 1280 B** (IPv6), and **73/73, max 1132 B** (IPv4-only); none is missing and no PTB appears at any capture point. The IPv6 rule snapshot is available at generation 76, but `serving_read_ok=false` and `serving=false` at 04:08:38.411158320–04:08:38.494516965Z. Its `serving_rules_before_new_handshake` qualification fails. The records do not identify which serving-query condition failed; packet success cannot replace it. [Original MTU rows and qualifications](history/follow-up-21/F21-native-MTU.json), [IPv6 original analysis](history/follow-up-18-captures/5715939e-3943-48f1-8bd5-c869bf7c91c2/matrix-ipv6-1280-d146/analysis.json).

D-137 known-PMTU and routed rows retain their observed passes. No new D-147 inner-ICMP-from-denied-source qualification is inferred from these outer native paths. Receive-only UDP/QUIC without PTB remain limitations under D-146; smaller-MSS control, real platforms, other encapsulations and physical arm64 remain unexercised. Historical F8-02 stays failed.

**Node writes:** **30 distinct receipts, 0 conflict receipts, 0 HTTP-412 attempts**. Every command has one recorded attempt; no conflict retry was exercised. [All attempts and their case locations](history/follow-up-21/F21-node-writes.json).

**Changed rows and attribution:** AT02's two corrected accounting predicates and AT09's lease-sequence/blocked-not-exercised handling are the follow-up 20 (#445) evaluator changes; run 2 had already passed the corrected AT02 offline replay and remains historically failed. The fresh absent-foreign-filter observation and the retained deny-all/no-extra-baseline/first-slot renewal trace are the measured paths addressed by #447 H-1/M-1/L-1. They do not prove a timing improvement or a full r28 control-outage scenario. AT01's first selection now counts (HTTP 200); its cause is unproven and F11-01 stays OPEN/upstream-dependent under D-148. AT08 changes from pass to blocked: its initial selected-route transfer has five member-interface packets, none at node/origin, zero origin requests and unchanged byte counters; a later member-B transfer counts. IPv6 D-146 changes from pass to blocked solely on the recorded serving-read qualification. Neither changed row identifies a merged cause. [AT08 original forwarding](history/follow-up-21/F21-AT08-forwarding.json).

D-133: startup/admission and teardown overlaps are retained as confounded, with **no retry**. No external-suite sample intersects the acceptance checks; five-second sampling cannot exclude shorter activity and supplies no full CPU/A-B qualification. [Run, waits and host state](history/follow-up-21/RUN.md), [process monitor](history/follow-up-21/F21-process-monitor.json), [admission windows](history/follow-up-21/F21-admission.json). All owner-resource groups remain unchanged and are not re-asked.

---

Historical text below retains its outcomes; links to superseded root evidence now resolve to the byte-pinned archive.

# Follow-up 18 acceptance results — run 2

Run 2 `50117c68-d1b3-431d-a2be-cd94d27a2447`, measured harness `4eb5d7d32c449da7689e749fd8ecb208657937dd`, baseline `27ff63928222c9642e4f5e180cf225108327a1e5`. **2 PASS / 6 BLOCKED / 2 FAIL; 699 passed / 5 blocked / 4 failed subchecks.**

| Case | Requirements | Status | Pass / blocked / fail |
|---|---|---|---:|
| [AT05](history/follow-up-18-run2/AT05.json) | NODE01 NODE04 ENR01 ENR04 PKG01 | blocked | 31 / 0 / 0 |
| [AT01](history/follow-up-18-run2/AT01.json) | APP01 ID01 MEM01 | blocked | 19 / 0 / 0 |
| [AT02](history/follow-up-18-run2/AT02.json.gz) | MEM01 MEM02 MEM03 SEC01 | fail | 191 / 0 / 2 |
| [AT03](history/follow-up-18-run2/AT03.json) | BILL01 BILL03 ENT01 | pass | 98 / 0 / 0 |
| [AT04](history/follow-up-18-run2/AT04.json) | BILL02 ENT02 MEM03 | blocked | 113 / 0 / 0 |
| [AT08](history/follow-up-18-run2/AT08.json) | NODE02 NODE03 EXIT01 EXIT03 | pass | 35 / 0 / 0 |
| [AT20](history/follow-up-18-run2/AT20.json) | DNS01 DNS02 ISO04 | blocked | 89 / 0 / 0 |
| [AT21](history/follow-up-18-run2/AT21.json) | DNS03 DNS04 APP01 | blocked | 13 / 0 / 0 |
| [AT22](history/follow-up-18-run2/AT22.json) | SEC02 DNS01 DNS04 | blocked | 40 / 0 / 0 |
| [AT09](history/follow-up-18-run2/AT09.json) | EXIT02 APP01 NFR01 | fail | 70 / 5 / 2 |

| Receive-only row | Selected family | Transfer / upload / evidence | Payload bytes / ms | Member SYN MSS sent → received | Origin SYN-ACK MSS sent → received | Reverse sent / received / max sent / max received IP bytes |
|---|---|---|---:|---|---|---:|
| [native dual MTU 1280](history/follow-up-18-captures/50117c68-d1b3-431d-a2be-cd94d27a2447/matrix-dual-1280-d146/analysis.json) | ipv4 | pass / pass / pass | 64064 / 1 | 1240 → 1160 | 1460 → 1160 | 73 / 73 / 1132 / 1132 |
| [native ipv6 MTU 1280](history/follow-up-18-captures/50117c68-d1b3-431d-a2be-cd94d27a2447/matrix-ipv6-1280-d146/analysis.json) | ipv6 | pass / pass / pass | 64064 / 1 | 1240 → 1160 | 1460 → 1160 | 74 / 74 / 1152 / 1152 |
| [native ipv4 MTU 1280](history/follow-up-18-captures/50117c68-d1b3-431d-a2be-cd94d27a2447/matrix-ipv4-1280-d146/analysis.json) | ipv4 | pass / pass / pass | 64064 / 1 | 1240 → 1160 | 1460 → 1160 | 72 / 72 / 1132 / 1132 |

The existing dual and IPv6 inputs and original byte/counter/family predicate are unchanged. The additional IPv4-only 1280 row makes IPv4 selection explicit independently of the dual row. The upload check uses the original bulk connection and requires 64,064 payload bytes after subtracting its 44-byte MEMBER greeting, with all 64 exact echoes proving acceptance at the origin. D-146 removes the D-137 downgrade only when every same-run qualification condition passes; an original failed transfer stays failed. Missing evidence or an advertisement already at/below 1160 cannot qualify the clamp. All three original analyses qualify: every selected reverse datagram matches at sender and member ingress, and both PTB types are absent at member, node and origin. Reverse payloads remain the fixture's 1006-byte echo writes; the endpoint MSS rewrites are the direct evidence that the clamp operated.

Receive-only UDP and QUIC without PTB remain documented limitations. The smaller-MSS control is **not exercised**: the unchanged fixture has no TCP_MAXSEG input; [Netlab 34's 1000 → 1000 control](../../../poc/G2-followup34-2026-11-03.md) stands. Clamp operation at 1280 on relay, real platforms, other encapsulations, IPv6 overlay and physical arm64 remain unverified. D-123 IPv6Block is unchanged.

AT09 records serving readiness **6.021246702 s**, counted recovery **12.548980167 s**, and desired generation **66 → 66**. [Every original AT09 subcheck](history/follow-up-18-run2/F18-AT09-subchecks.md), including D-067 and the combined restart, retains its measured status; [original restart records](history/follow-up-18-run2/F18-AT09-restart.json). These timing values alone identify no particular merged change from agent-core 37–43 (#429). Any narrower attribution requires the original records. Namespace destruction/recreation does not prove warm-kernel adoption. The D-067 record has 2 provisional observations, stage=lease error counts [0, 0], forwarding-block reasons [['safety_check_failed'], ['safety_check_failed']], and saw_lease_expired=False. Captured errors include [('apply', 'TC_FOREIGN_FILTER'), ('initial_quarantine', 'TC_FOREIGN_FILTER'), ('probe', 'HEALTH_CHECK_FAILED'), ('quarantine', 'TC_FOREIGN_FILTER'), ('tc_evidence', 'TC_FOREIGN_FILTER')]. These are original failure observations, not a causal attribution to a merged change.

AT02 retains **6250 sends**, **0 send errors**, **983 absent node ingresses**, and **259 ingress-without-egress losses**. All **1242** losses count. [Every original send](history/follow-up-18-run2/F18-AT02-every-send.json), [original recovery](history/follow-up-18-run2/F18-AT02-recovery.json.gz), [reservation joins](history/follow-up-18-run2/F18-pre-apply-reservations.json.gz). D-135/D-136 remain provisional; no loss waiver or finite outage ceiling is introduced. The original evaluator records measurement_valid=False, failed_apply_in_capture=3, failed_apply_bracketed=0, and cause_status=unproven.

AT01's original first selection is counted=False, HTTP status 0, origin receipt=False, with 0 origin requests. Its joined first-SYN transit is 5.654543001–5.654549869 s. [Original headers and observations](history/follow-up-18-run2/F18-loopback-and-first-selection.json) retain cause_status=unproven. F11-01 remains OPEN and upstream-dependent; later success closes neither F11-01 nor historical F14-01.

Orchestrator note 1 (2026-11-03 15:58 EST) authorized this second run after archiving run 1. Commit `4eb5d7d3` adds only the missing production retire phase to the reader and preserves its other validation. [Closed-list audit and regression tests](history/follow-up-18-run2/F18-reader-audit.md). D-146, all timeouts and the case order are unchanged. No third run is authorized or performed.

Run 1 `118271c1`, measured harness `b4317bd4`, remains **0 PASS / 3 BLOCKED / 7 FAIL; 442 / 0 / 35 subchecks**. [Archive commit 71dc60ac](history/follow-up-18-run1/INDEX.md) retains 72 byte-identical root files, gzip and pins, plus the original captures. The [retire rejection and 16 exact export-prefix joins](history/follow-up-18-run1/F18-lifecycle-phase-compatibility.json) prove the reader defect where joined; they do not explain every independent failure. Its [finite-grant restoration gap](history/follow-up-18-run1/F18-AT09-precondition.json), restart failures and unexercised D-146 rows remain recorded.

D-098 first commit `50f4403b` archives run 81b638e8's 77 root files byte-identically, including gzip and sidecars: [archive](history/follow-up-17/INDEX.md), [pins](history/follow-up-17/SHA256SUMS). No handover claims test reads a moved root file, so no D-132 pointer edit was needed. Earlier archive bytes remain unchanged. Historical e80bb592, 0ffb355a and 81b638e8 outcomes stay unchanged; their F8-02 statuses remain failed. [Historical MSS/small-write analysis](history/follow-up-18-run2/F18-historical-81b638e8-MSS.json) explains the 64 separate 1006-byte echo segments and 1132/1152-byte outer datagrams; it changes no historical verdict.

[All failed checks](history/follow-up-18-run2/F18-results.json), [D-146 evidence](history/follow-up-18-run2/F18-D146.json), [grant restoration and prerequisites](history/follow-up-18-run2/F18-AT09-precondition.json), [qualification](history/follow-up-18-run2/F18-qualification.json), [hold joins](history/follow-up-18-run2/F18-populated-hold-joins.json), [run/gates](history/follow-up-18-run2/RUN.md), [carried requests](history/follow-up-18-run2/requests.md). F11-01 stays **OPEN and upstream-dependent (D-148)**; no product change.

## 2026-11-03 — Follow-up 19 post-run-50117c68 early-return restoration note

Changed **after** run `50117c68` (`50117c68-d1b3-431d-a2be-cd94d27a2447`) in commits `6186fd60` / `306a1969` (`306a19693d694a0d4f26565705e471f8f0ee9448`). No assembled run was performed in Follow-up 19; all committed run records, including run 1's retained archive (`118271c1`) and run 2's root records (`50117c68`), remain byte-identical.

- **Finite-grant early-return restoration (`tests/e2e/at/members/finite_grant_test.go`, `tests/e2e/at/members/grant_restore.go`):** every early return in `finiteExpiry` after the case may have changed member `A`'s grant (after failed `replaceFiniteGrant` and after failed `traceForwarding`) now restores member `A` (`l.a`) to 3 authorized devices and 0 pending devices through the product's public API (`POST /api/v1/admin/members/{id}/grants` on `l.planRev` for `now + 1h`). After failed `replaceFiniteGrant`, the 75-second grant on `l.planRev` may or may not be in place (prior `l.planRev` grant still live, no live grant after revoke, or 75-second `l.planRev` grant installed before projection timeout); after failed `traceForwarding`, the 75-second `l.planRev` grant is live until its `valid_until`, or expired (`0 authorized, 3 pending`) if readiness timed out past `valid_until`. On every path, `members.Service.Grant` in `internal/members/service.go` (lines 460–466) accepts `POST /api/v1/admin/members/{id}/grants` on `l.planRev` (`202 Accepted`) because its active-grant overlap check rejects only `plan_revision_id <> $2` (different plan revisions) and ignores expired/revoked grants, while `entitlement.Compute` in `internal/entitlement/engine.go` (lines 171–229) and `reconcileDevices` in `internal/entitlement/devices.go` (lines 57–99) merge overlapping same-plan grants or activate the new grant and re-admit any pending devices back to `3 authorized, 0 pending`. The original failed check stays failed with its name, expectation, and text unchanged; early-return restoration is recorded under `"finite expiry early-return restoration HTTP"` and `"finite expiry early-return restored 3/0"`; and the normal path still restores once with its existing `"finite expiry restoration HTTP"` and `"finite expiry restored 3/0"` checks unchanged.
- **Cross-case early-return audit (`tests/e2e/at/members/reauthorization_test.go`, `tests/e2e/at/members/populated_hold_test.go`, `tests/e2e/at/members/devices_billing_test.go`, `tests/e2e/at/members/sibling_live_test.go`, `tests/e2e/at/members/persistent_test.go`):** early returns in `reauthorizationLimit` (after failed lower-plan `replaceFiniteGrant` or failed `fenceInsert`), `reauthorization` (after failed `fenceInsert` when a fence is present), `populatedHolds` (cleanup when member `B`'s device remains pending after suspension), `entitlements` (early return when `l.paid == nil` while `node2` is published), and `startSibling` (early return after starting `c.native` and `c.stopSamples`) now restore shared state before later cases run. Synthetic offline unit tests in `tests/e2e/at/members/grant_restore_test.go` verify every early-return path and the normal path without a stack.
