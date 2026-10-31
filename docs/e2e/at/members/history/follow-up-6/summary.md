# Follow-up 6 acceptance results

Current run: `80db103b-622c-42b1-946d-4628daaa213d` on `w4-members / 24000`, product base `99ab668` (includes #193 D-067, #213 revision 11, #216 agent follow-up 18, #219 devstack follow-up 13), harness base `159d617`.
**Three complete cases passed; seven are blocked.** Every executed subcheck in this run passed (384 passing subchecks, 0 failed subchecks, 0 retained failed prior checks).

- **AT02 strict post-withdrawal ingress is proved with the devstack follow-up 13 export (#219):** after real member DELETE of `member-a2` (`01a18efa-99e8-75d7-9a6c-47f05b5c55e9`, `100.104.160.29`) and convergence to generation `11`, `labstack.Stack.ObserveMemberIngress` on `fvmb0` (`tun-mirred`, link type `65534`, source port `40000`, dest port `9` -> `192.0.4.10`) proves that the revoked source increments `ctunku` by `+1` packet (`+29` bytes) with `0` forwarded bytes, `0` device counter changes (`u_01a18efa99e875d79a6c47f05b5c55e9` stays at `10` packets / `2317` bytes), and empty `fvup0` and `origin` packet captures, while the admitted sibling (`member`, `01a18ef9-ae93-7a1c-b934-20bed403bf2f`, `100.104.242.187`) increments `u_01a18ef9ae937a1cb93420bed403bf2f` by `+1` packet (`+29` bytes, `31 -> 32`), `ctunku` by `+0`, reaches `fvup0` and `origin`, and increments `d_01a18ef9ae937a1cb93420bed403bf2f` by `+1` packet (`+57` bytes, `22 -> 23`) on the origin's ICMP response while all other device counters hold steady.
- **AT09 is re-recorded in D-067 / revision 11 terms (#193, #213, #216):** during the `node2` cold-restart provisional window on unchanged desired generation `23`, provisional zero-grant leases (`1`-minute TTL) are reported with `forwarding_block_reasons` containing `"awaiting_member_grants"` (never `"lease_expired"`), `lease`, `qos` and `source_binding` checks carry `error_code: "AWAITING_MEMBER_GRANTS"`, and `errors[]` carries `stage: "lease"`, `code: "AWAITING_MEMBER_GRANTS"` with exact `stage=lease` count `1`. Serving readiness is restored on generation `23` in **`36.976 s`** and the first counted member byte arrives in **`41.695 s`** with no policy change or admin action. AT09 remains `blocked` only on F4-03 (forced relay/QUIC/PMTU and physical second NAT).

| AT | Status | Deciding observation | Evidence |
|---|---|---|---|
| AT01 | blocked | Counted Linux forwarding before/after pause; four native platforms and network-change scripts remain. | [AT01.json](AT01.json), [AT01.md](AT01.md) |
| AT02 | passed | One race winner; 3→2→3 slots; strict `fvmb0` ingress injection proves revoked source hits `ctunku` (+1 pkt, 0 forwarded bytes) while sibling hits `u_<hex>` (+1 pkt) and reaches `fvup0`/`origin`; sibling/replacement counted HTTPS flows continue. | [AT02.json](AT02.json), [AT02.md](AT02.md) |
| AT03 | passed | Ten identical deliveries and six orders preserve one exact paid period; paid native device carries counted traffic. | [AT03.json](AT03.json), [AT03.md](AT03.md) |
| AT04 | blocked | TCP refund/expiry/suspension stops on both exits; member UDP and rendered suspended/expired screens remain unproved. | [AT04.json](AT04.json), [AT04.md](AT04.md) |
| AT05 | blocked | Two installer runtimes do not prove preinstalled/cloud-init boot or physical arm64. | [AT05.json](AT05.json), [AT05.md](AT05.md) |
| AT08 | passed | A/B forwarding isolation and post-restart disable hold; operation polling observes final node ACK. | [AT08.json](AT08.json), [AT08.md](AT08.md) |
| AT09 | blocked | Cold restart reports revision 11 `awaiting_member_grants` + `AWAITING_MEMBER_GRANTS` (`stage=lease` count 1, never `lease_expired`) and restores generation-`23` 7-grant lease and counted `fvmb0` TCP in 41.695 s; relay/QUIC/PMTU and physical NAT remain (F4-03). | [AT09.json](AT09.json), [AT09.md](AT09.md) |
| AT20 | blocked | Typed resolver delivery applies; native DNS and full RR/cache/rebinding matrix remain unproved. | [AT20.json](AT20.json), [AT20.md](AT20.md) |
| AT21 | blocked | Two filtering resolvers and corruption/last-good recovery work; native DisableDNS prevents an automatic fallback claim. | [AT21.json](AT21.json), [AT21.md](AT21.md) |
| AT22 | blocked | Privacy/audit/feed subsets pass; complete URL/file/raw-usage/backup retention and commercial-source approval remain. | [AT22.json](AT22.json), [AT22.md](AT22.md) |

AT01/AT05/AT09/AT21's Windows, macOS, iOS, Android, physical arm64, second NAT
and Private DNS/DoH rows remain **needs operator run** in
[OPERATORS.md](OPERATORS.md). Linux observations and cross-builds do not pass them.

Previous Follow-up 5 records are archived under
[history/follow-up-5](history/follow-up-5/RUN.md), Follow-up 4 under
[history/follow-up-4](history/follow-up-4/RUN.md), and Follow-up 3 under
[history/follow-up-3](history/follow-up-3/RUN.md). [Requests](requests.md) give each
resolved and open owner/location.

TCP stop latencies (seconds after the actual rights change/deadline):

| Event | First exit | Second exit |
|---|---:|---:|
| Full refund | 5.512 | 5.722 |
| Real paid expiry | 1.529 | 1.374 |
| Manual suspension | 3.965 | 4.006 |

Every monitor first received an origin response and reused its TCP connection.
A later successful paid event did not clear suspension. New-connection denial
was measured after each event. The 60-second comparison is the acceptance test
target; this is not an unmeasured performance guarantee.
