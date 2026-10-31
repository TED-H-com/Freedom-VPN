# Follow-up 5 acceptance results

Current run: `16aa2d1b-d46f-4e88-a509-a8a9c9076006` on `w4-members / 24000`, product base `dde4aa6` (merged #193, D-067), harness base `d6f49fd`.
**Three complete cases passed; seven are blocked.** Every executed subcheck in this run passed (382 passing subchecks, 0 failed subchecks, 0 retained failed prior checks).

**F4-01 is resolved by PR #193 (D-067):** after stopping and starting `node2`, `node2-adguard` and `node2-netns`, central initially issues provisional zero-grant leases on unchanged desired generation `25` while `node2` is not yet `ready`, and `fvpn-agent` renews on its bounded provisional backoff schedule until readiness restores the 7-grant generation-`25` lease and counted member TCP traffic through `fvmb0` (`39.236 s` from container start to serving-ready, `43.982 s` to first counted member byte) with no policy change or admin action. AT09 remains `blocked` only on the unexportable forced-relay/QUIC/PMTU and physical second-NAT matrix (F4-03).

| AT | Status | Deciding observation | Evidence |
|---|---|---|---|
| AT01 | blocked | Counted Linux forwarding before/after pause; four native platforms and network-change scripts remain. | [AT01.json](AT01.json), [AT01.md](AT01.md) |
| AT02 | passed | One race winner; 3→2→3 slots; revoked/loser denied, sibling/replacement counted, unchanged agent PID and coherent health. | [AT02.json](AT02.json), [AT02.md](AT02.md) |
| AT03 | passed | Ten identical deliveries and six orders preserve one exact paid period; paid native device carries counted traffic. | [AT03.json](AT03.json), [AT03.md](AT03.md) |
| AT04 | blocked | TCP refund/expiry/suspension stops on both exits; member UDP and rendered suspended/expired screens remain unproved. | [AT04.json](AT04.json), [AT04.md](AT04.md) |
| AT05 | blocked | Two installer runtimes do not prove preinstalled/cloud-init boot or physical arm64. | [AT05.json](AT05.json), [AT05.md](AT05.md) |
| AT08 | passed | A/B forwarding isolation and post-restart disable hold; operation polling observes final node ACK. | [AT08.json](AT08.json), [AT08.md](AT08.md) |
| AT09 | blocked | Cold restart restores same-generation (25) 7-grant lease and counted fvmb0 TCP in 43.982 s (D-067 #193 resolves F4-01); relay/QUIC/PMTU and physical NAT remain (F4-03). | [AT09.json](AT09.json), [AT09.md](AT09.md) |
| AT20 | blocked | Typed resolver delivery applies; native DNS and full RR/cache/rebinding matrix remain unproved. | [AT20.json](AT20.json), [AT20.md](AT20.md) |
| AT21 | blocked | Two filtering resolvers and corruption/last-good recovery work; native DisableDNS prevents an automatic fallback claim. | [AT21.json](AT21.json), [AT21.md](AT21.md) |
| AT22 | blocked | Privacy/audit/feed subsets pass; complete URL/file/raw-usage/backup retention and commercial-source approval remain. | [AT22.json](AT22.json), [AT22.md](AT22.md) |

AT01/AT05/AT09/AT21's Windows, macOS, iOS, Android, physical arm64, second NAT
and Private DNS/DoH rows remain **needs operator run** in
[OPERATORS.md](OPERATORS.md). Linux observations and cross-builds do not pass them.

Previous Follow-up 4 records are archived under
[history/follow-up-4](history/follow-up-4/RUN.md) and Follow-up 3 under
[history/follow-up-3](history/follow-up-3/RUN.md). [Requests](requests.md) give each
resolved and open owner/location.

TCP stop latencies (seconds after the actual rights change/deadline):

| Event | First exit | Second exit |
|---|---:|---:|
| Full refund | 6.012 | 6.224 |
| Real paid expiry | 1.700 | 1.225 |
| Manual suspension | 3.964 | 4.008 |

Every monitor first received an origin response and reused its TCP connection.
A later successful paid event did not clear suspension. New-connection denial
was measured after each event. The 60-second comparison is the acceptance test
target; this is not an unmeasured performance guarantee.
