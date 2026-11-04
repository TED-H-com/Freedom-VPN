# D-147 RELATED fragmentation-needed — 2026-11-03

Fresh mechanism run `1d746f9a-e2df-4baf-af8d-f2d3074c8cbf` on committed source `3e12f913418e6ee87cee5d43e4dcb93d86331237`: **all four source-set rows pass**, independently. Started `2026-11-03T22:06:39Z`, ended `2026-11-03T22:07:03Z`. Production baseline is `27ff63928222c9642e4f5e180cf225108327a1e5` (netlab 34 / #432). This is mechanism evidence for §8.2–§8.3, §10.1–§10.2, §11.4 and §19 AT13 (ISO01, ISO02, MTR01); it does not close the full native AT13 fault matrix.

[Raw record](history/follow-up-19-lab-final/lab-AT13-d147.json), [header-only captures and deltas](history/follow-up-19-lab-final/lab-AT13-d147-detail.json), [installed rules](history/follow-up-19-lab-final/lab-AT13-d147-ruleset.nft), [immutable SHA-256 manifest](history/follow-up-19-lab-final/archive.json). Root [lab-AT13-d147.json](lab-AT13-d147.json) may subsequently identify the assembled run's separate execution.

## Topology and production path

`fv-a1` member `100.64.1.1` uploads through the normal forwarding node (`fv-fh` → `fv-ih`, SNAT `192.0.2.37`) to router `fv-inet`, then origin `203.0.113.34:8080`. The router/origin veth uses `192.0.2.41/30` and `192.0.2.42/30`. Router egress MTU is 1000; origin egress remains 1500, so SYN-ACK advertises a larger MSS and the upload genuinely requires router fragmentation-needed errors. A preferred-source return route selects each row’s outer source. TCP MTU probing is disabled; each row starts without a cached PMTU and learns 1000 from the error. The receiver reads and verifies all 64,064 payload bytes before replying `ok:64064`.

**TEST ONLY:** the existing lab exempts `203.0.113.0/24` from `denied_v4` so documentation addresses can act as isolated Internet endpoints. The new model adds the resolver-only and host-only addresses to their exact production sets, plus the overlap host prefix. The model’s host prefixes represent protected uplink destinations, not a claim about automatic discovery of those prefixes. All four memberships are independently read from the installed nft JSON; no row is inferred from another.

`fvpn-netlab apply` calls the normal production Apply and live verifier. Generation 4200 admits three addresses. The new forward view finds exactly **27 admitted-member accepts, 9 D-147 accepts and 6 non-verdict D-146 MSS clamps**. The old `MemberAccept` predicate is unchanged. There is no ad hoc accept, rule mutation, altered deadline, or changed production implementation.

## Positive rows and exact admission usage

Every row completes 64,064 bytes, learns PMTU **1000**, reports socket MSS **948**, and carries exactly **10** fragmentation-needed errors / **5,760 IP bytes** to the member. Device `019a0000-0000-7000-8000-0000000000a1` uses download counter `d_019a00000000700080000000000000a1`, priority `0x0001002f` / tc class `1:2f`.

| Source set (verified exact membership) | Source | Upload packets / IP bytes | Download packets / IP bytes | Download TCP IP bytes | tc upload / download bytes | Verdict |
| --- | --- | --- | --- | --- | --- | --- |
| resolver_v4 | `203.0.113.53` | 84 / 80,007 | 44 / 7,601 | 1,841 | 81,183 / 8,217 | pass |
| denied_v4 | `192.168.34.2` | 84 / 80,007 | 36 / 7,185 | 1,425 | 81,183 / 7,689 | pass |
| host_v4 | `203.0.113.33` | 84 / 80,007 | 38 / 7,289 | 1,529 | 81,183 / 7,821 | pass |
| denied_v4 + host_v4 | `192.168.34.1` | 84 / 80,007 | 50 / 7,913 | 2,153 | 81,183 / 8,613 | pass |

The independent node-egress captures (`node_up_egress_fv_ih`, `node_down_egress_fv_fh`) match both packets and IP bytes exactly against counter deltas. Download is TCP IP bytes plus the 5,760 ICMP IP bytes once. Each tc leaf has the same packet count and `IP bytes + 14 × packets`, with zero drops. The overlap row’s first matching accept therefore accounts once. Different ACK counts across rows are measured separately, never normalized or inferred.

Captures also retain `router_fv_ii` and `member_fv_wg0`, ICMP type/code/MTU, and per-packet IP-header lengths. Every capture records tcpdump’s completion count and zero kernel drops; the parser rejects incomplete headers, frames, unsupported link types and mismatched decode counts. Raw payload pcaps are deleted inside the lab; only header evidence is published.

## Three negatives per source

| Source | Genuine time-exceeded | Unsolicited type 3/code 4 | Member upload to source | Upload usage for denied upload |
| --- | --- | --- | --- | --- |
| `203.0.113.53` | `ctdenyd`: 1 packet / 112 bytes | `ctinv`: 1 packet / 60 bytes | `ctbadns`: 1 packet / 84 bytes | 0 packets / 0 bytes |
| `192.168.34.2` | `ctdenyd`: 1 packet / 112 bytes | `ctinv`: 1 packet / 60 bytes | `ctdeny`: 1 packet / 84 bytes | 0 packets / 0 bytes |
| `203.0.113.33` | `ctdenyd`: 1 packet / 112 bytes | `ctinv`: 1 packet / 60 bytes | `cthost`: 1 packet / 84 bytes | 0 packets / 0 bytes |
| `192.168.34.1` | `ctdenyd`: 1 packet / 112 bytes | `ctinv`: 1 packet / 60 bytes | `ctdeny`: 1 packet / 84 bytes | 0 packets / 0 bytes |

Time-exceeded uses TTL 3 through the member’s routing namespace, node and router. Captures prove it reached the node and disappeared before member egress, with zero download usage. The unsolicited valid-checksum error quotes an untracked tuple; it reaches the node, dies at `ctinv`, and never reaches the member. Each upload ping reaches node ingress, increments its exact production destination-deny counter, and is absent at node egress and the router. Both Internet and DNS-service upload counters remain zero.

**Literal brief mismatch (F19-01):** the requested `ctdeny` drop is observed for denied-only and overlap. Resolver-only is dropped earlier by `ctbadns`; host-only is dropped by `cthost`. A source only in resolver/host cannot also match `denied_v4` without violating the requested exclusive membership. The literal `ctdeny` subcase for those two exclusive sets is **not exercised**; the actual policy denial is separately verified above. No predicate or production order is weakened. The orchestrator/netlab owner must confirm these named counters as the intended expectation or define a different authorized requirement. This is a brief/production-contract mismatch, not evidence of forwarding leakage.

## Existing rows and failure preservation

All existing `tests/netlab/at/scenario_test.go` code remains byte-identical to main. AT10–AT13 reran unchanged in the same fresh lab session on `3e12f913`: protected destinations/interface changes, IPv6 and metadata, member/DNS isolation, admission/revocation, SIGKILL and baseline. The assembled run additionally retains the existing renewal/expiry and probe restrictions; its outcomes are separate.

| Record | Run | Verdict |
| --- | --- | --- |
| [lab-AT10](history/follow-up-19-lab-final/lab-AT10.json) | `363283a9-9222-4308-9262-62c8b8fa7a3b` | pass |
| [lab-AT11](history/follow-up-19-lab-final/lab-AT11.json) | `df9d963f-05ae-4f59-baf8-3b59f74292da` | pass |
| [lab-AT12](history/follow-up-19-lab-final/lab-AT12.json) | `be06c4f0-4365-41b7-8d3b-63db87227196` | pass |
| [lab-AT13](history/follow-up-19-lab-final/lab-AT13.json) | `aac8deb8-62c6-45e8-aad8-9e887a5740b1` | pass |

1. [Inherited records](history/follow-up-19-inherited-lab/archive.json): existing AT10–AT13 passed on `9493950a` (runs `b2702151`, `e5bdeacd`, `4dd132e5`, `e60f954b`). Failed D-147 run `58fb86e8` remains **failed**. All four router positives delivered 10 errors and learned PMTU 1000, but the fixture used `services.py`, whose HTTP handler reads at most 2,048 bytes and closes. Resolver upload failed with Broken pipe in 1.67 ms; other sources reset. The resolver upload denial also checked nonexistent `ctdns` instead of production `ctbadns`. The inherited run reports build `9493950a` and `source_dirty=false`, but the Python runner was embedded from uncommitted changes. That build label is not reliable provenance; the inherited source snapshot is retained without rewriting the historical record.

2. [Collector attempt 1](history/follow-up-19-lab-attempt-1/archive.json) on `150c090f` remains **failed**: requiring all counters exposed another inherited spelling, `ctnodown` instead of `ctnodn`. The four existing rows passed. DNS accounting names were also corrected from nonexistent `du_`/`dd_` to production `nu_`/`nd_`.

3. [Reused-lab attempt 2](history/follow-up-19-lab-attempt-2/archive.json) on `486d75d1` remains **failed**: every positive D-147 subcase passed, but old AT10 streams added 2 upload packets / 111 bytes during the resolver negative. The denial itself reached `ctbadns` (1 / 84); its zero-usage predicate correctly failed. AT10 also saw an old failure in its append-only stream log. Process inspection subsequently found two generations each of AT10 and AT13 stream processes. Recreating the lab removed that contamination; no existing row or verdict was edited.

Every continuation lab record was built from committed code. The final focused run used a new lab container, and it was removed before the assembled invocation. The inherited four stray tcpdump processes were removed by the first lab teardown.

## Reader compatibility and sessions

[Reader audit](follow-up-19-reader-audit.json): `retire` is now accepted as a production command phase. A successful Apply with nft install then tc retire parses; an unknown phase still raises. All 23 synthetic reader tests pass. Command kinds, operations, timed/untimed record sources and stages were checked against `node/` on main; no other lifecycle mismatch was found. The unrelated fixture counter corrections are explicitly listed.

Orchestrator note 1 is preserved: separate D-147 test/record; old AT10/AT13 code unchanged; `memberAcceptRule` unchanged for every `fvmb0` line; new lab/MSS/PMTU counts are separate; `forward_view` in the assembled snapshot is evidence only. Python is invoked with `-B` or `PYTHONDONTWRITEBYTECODE=1`, and no bytecode is committed.

The first session used agy Gemini 4 Argon through 16:58 EST; a content-filter interruption after reading long lab output caused the orchestrator’s D-143 handoff to codex/gpt-6-astra. This continuation retained the branch, both prior commits and all work in progress. D-098 archive commit `f5a5a425` preserves all 140 follow-up-18 root files byte-identically with SHA-256 pins; earlier archives and handover claim paths remain untouched.

## Assembled run

The single assembled invocation and final handoffs are recorded in [summary](summary.md), [run procedure](RUN.md), and [requests](requests.md). Mechanism success never promotes blocked native-platform or full-fault rows.
