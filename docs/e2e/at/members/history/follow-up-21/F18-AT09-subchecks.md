# Follow-up 18 AT09 subchecks

Run `50117c68-d1b3-431d-a2be-cd94d27a2447`, measured harness `4eb5d7d32c449da7689e749fd8ecb208657937dd`. All original statuses below come from [AT09](history/follow-up-18-run2/AT09.json). Only D-146 qualifies the former receive-only TCP downgrade; no other predicate changes.

| Index | Original check | Observed status |
|---:|---|---|
| 0 | AT09 member OIDC reauthentication | pass |
| 1 | finite capacity and classification | pass |
| 2 | publication command published | pass |
| 3 | publication progress published | pass |
| 4 | publication operation published | pass |
| 5 | finite capacity and classification | pass |
| 6 | publication command published | pass |
| 7 | publication progress published | pass |
| 8 | publication operation published | pass |
| 9 | populated-grant serving readiness | pass |
| 10 | real default-route semantics | pass |
| 11 | member sees both published exits | pass |
| 12 | populated-grant serving readiness | pass |
| 13 | populated-grant serving readiness | pass |
| 14 | bounded member transfer observation | pass |
| 15 | populated-grant serving readiness | pass |
| 16 | bounded member transfer observation | pass |
| 17 | agent transition selected namespace stop | pass |
| 18 | selected exit offline behavior | pass |
| 19 | second exit fresh serving after restart | pass |
| 20 | populated-grant serving readiness | pass |
| 21 | populated-grant serving readiness | pass |
| 22 | bounded member transfer observation | pass |
| 23 | agent transition selected namespace restart and 90-second observation | pass |
| 24 | second exit complete 90-second restart capture | pass |
| 25 | D-067 revision 11 provisional lease observation | fail |
| 26 | same-generation D-067 restart recovery and lease sequence | fail |
| 27 | typed DNS evidence | pass |
| 28 | issued resolver document mTLS fetch | pass |
| 29 | current desired-state DNS selection | pass |
| 30 | populated-grant serving readiness | pass |
| 31 | bounded member transfer observation | pass |
| 32 | native ipv4 MTU 1500 input | pass |
| 33 | native capture completeness matrix-ipv4-1500 | pass |
| 34 | native ipv4 MTU 1500 transfer | pass |
| 35 | native ipv6 MTU 1500 input | pass |
| 36 | native capture completeness matrix-ipv6-1500 | pass |
| 37 | native ipv6 MTU 1500 transfer | pass |
| 38 | native dual MTU 1500 input | pass |
| 39 | native capture completeness matrix-dual-1500 | pass |
| 40 | native dual MTU 1500 transfer | pass |
| 41 | native dual MTU 1280 input | pass |
| 42 | native capture completeness matrix-dual-1280 | pass |
| 43 | native dual MTU 1280 transfer | pass |
| 44 | native dual MTU 1280 upload | pass |
| 45 | native dual MTU 1280 D-146 clamp evidence | pass |
| 46 | native ipv6 MTU 1280 input | pass |
| 47 | native capture completeness matrix-ipv6-1280 | pass |
| 48 | native ipv6 MTU 1280 transfer | pass |
| 49 | native ipv6 MTU 1280 upload | pass |
| 50 | native ipv6 MTU 1280 D-146 clamp evidence | pass |
| 51 | D-137 selected-peer 1280 constraint | pass |
| 52 | native capture completeness known-pmtu1280 | pass |
| 53 | D-137 supported known-PMTU 1280 transfer | pass |
| 54 | native ipv4 MTU 1280 input | pass |
| 55 | native capture completeness matrix-ipv4-1280 | pass |
| 56 | native ipv4 MTU 1280 transfer | pass |
| 57 | native ipv4 MTU 1280 upload | pass |
| 58 | native ipv4 MTU 1280 D-146 clamp evidence | pass |
| 59 | native linux quic | blocked |
| 60 | native android tcp | blocked |
| 61 | native ios tcp | blocked |
| 62 | native windows tcp | blocked |
| 63 | native macos tcp | blocked |
| 64 | relay IPv4 precondition | pass |
| 65 | forced native relay | pass |
| 66 | persistent TCP/UDP positive member | pass |
| 67 | restore direct transport | pass |
| 68 | relay to direct same sockets | pass |
| 69 | D-137 routed hop setup | pass |
| 70 | D-137 routed ipv6 selection | pass |
| 71 | native capture completeness routed-ipv6-1280 | pass |
| 72 | D-137 routed ipv6 1280 transfer | pass |
| 73 | D-137 routed ipv4 selection | pass |
| 74 | native capture completeness routed-ipv4-1280 | pass |
| 75 | D-137 routed ipv4 1280 transfer | pass |
| 76 | r18 M-3 accepted hold and central ID join | pass |

The original JSON retains each expected value, actual measurement, command and timestamp. [D-146 joins](history/follow-up-18-run2/F18-D146.json) and [restart rows](history/follow-up-18-run2/F18-AT09-restart.json) are additive indexes.

Historical e80bb592, 0ffb355a and 81b638e8 outcomes stay unchanged. UDP/QUIC receive-only no-PTB remains a limitation; D-123, D-135 and D-136 remain in force.
