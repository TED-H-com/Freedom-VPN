# Follow-up 4 acceptance results

Current run: `d89eda32-84ef-4582-b669-053d5d9abadd` on `w4-members / 24000`, product base `e0a84fb`.
**Three complete cases passed; seven are blocked.** The primary second suite
reported AT09 as failed; its failed recovery assertion remains in `prior_checks`
and the untouched attempt-2 archive. The partial readback classifies that
complete case as blocked on F4-01, never passed. Current records contain
380 passing subchecks and 1 retained failed prior check.

| AT | Status | Deciding observation | Evidence |
|---|---|---|---|
| AT01 | blocked | Counted Linux forwarding before/after pause; four native platforms and network-change scripts remain. | [AT01.json](AT01.json), [AT01.md](AT01.md) |
| AT02 | passed | One race winner; 3→2→3 slots; revoked/loser denied, sibling/replacement counted, unchanged agent PID and coherent health. | [AT02.json](AT02.json), [AT02.md](AT02.md) |
| AT03 | passed | Ten identical deliveries and six orders preserve one exact paid period; paid native device carries counted traffic. | [AT03.json](AT03.json), [AT03.md](AT03.md) |
| AT04 | blocked | TCP refund/expiry/suspension stops on both exits; member UDP and rendered suspended/expired screens remain unproved. | [AT04.json](AT04.json), [AT04.md](AT04.md) |
| AT05 | blocked | Two installer runtimes do not prove preinstalled/cloud-init boot or physical arm64. | [AT05.json](AT05.json), [AT05.md](AT05.md) |
| AT08 | passed | A/B forwarding isolation and post-restart disable hold; operation polling observes final node ACK. | [AT08.json](AT08.json), [AT08.md](AT08.md) |
| AT09 | blocked | Restart receives an empty generation-23 lease; no serving recovery in 90 seconds, node baseline drops observed traffic (F4-01). | [AT09.json](AT09.json), [AT09.md](AT09.md) |
| AT20 | blocked | Typed resolver delivery applies; native DNS and full RR/cache/rebinding matrix remain unproved. | [AT20.json](AT20.json), [AT20.md](AT20.md) |
| AT21 | blocked | Two filtering resolvers and corruption/last-good recovery work; native DisableDNS prevents an automatic fallback claim. | [AT21.json](AT21.json), [AT21.md](AT21.md) |
| AT22 | blocked | Privacy/audit/feed subsets pass; complete URL/file/raw-usage/backup retention and commercial-source approval remain. | [AT22.json](AT22.json), [AT22.md](AT22.md) |

AT01/AT05/AT09/AT21's Windows, macOS, iOS, Android, physical arm64, second NAT
and Private DNS/DoH rows remain **needs operator run** in
[OPERATORS.md](OPERATORS.md). Linux observations and cross-builds do not pass them.

Both fresh attempts, the first failed AT21 command, the early partial withdrawal
sample, and both restart probes remain in [RUN.md](RUN.md) and
[history/follow-up-4](history/follow-up-4/attempt-1/attempt.json). Follow-up 3 is
archived under [history/follow-up-3](history/follow-up-3/RUN.md). No failed prior
result is silently replaced. [Requests](requests.md) give each owner/location.

TCP stop latencies (seconds after the actual rights change/deadline):

| Event | First exit | Second exit |
|---|---:|---:|
| Full refund | 5.511 | 5.673 |
| Real paid expiry | 2.124 | 1.148 |
| Manual suspension | 3.966 | 4.006 |

Every monitor first received an origin response and reused its TCP connection.
A later successful paid event did not clear suspension. New-connection denial
was measured after each event. The 60-second comparison is the acceptance test
target; this is not an unmeasured performance guarantee.
