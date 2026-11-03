# Follow-up 18 AT09 subchecks

Run `118271c1-616e-46a2-a744-d67b00fb94fb`, measured harness `b4317bd4e4ad178609b71d3fd4fbb986cb9947db`. All original statuses below come from [AT09](AT09.json). Only D-146 qualifies the former receive-only TCP downgrade; no other predicate changes.

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
| 11 | member sees both published exits | fail |
| 12 | populated-grant serving readiness | fail |
| 13 | bounded member transfer observation | pass |
| 14 | unauthorized real transfer | pass |
| 15 | bounded member transfer observation | pass |
| 16 | unauthorized real transfer | pass |
| 17 | agent transition selected namespace stop | pass |
| 18 | selected exit offline behavior | pass |
| 19 | second exit fresh serving after restart | fail |
| 20 | populated-grant serving readiness | pass |
| 21 | bounded member transfer observation | pass |
| 22 | unauthorized real transfer | pass |
| 23 | agent transition selected namespace restart and 90-second observation | pass |
| 24 | second exit complete 90-second restart capture | fail |
| 25 | D-067 revision 11 provisional lease observation | fail |
| 26 | same-generation D-067 restart recovery and lease sequence | fail |
| 27 | typed DNS evidence | pass |
| 28 | issued resolver document mTLS fetch | pass |
| 29 | current desired-state DNS selection | pass |
| 30 | bounded member transfer observation | pass |
| 31 | unauthorized real transfer | pass |
| 32 | r18 M-3 accepted hold and central ID join | fail |

The original JSON retains each expected value, actual measurement, command and timestamp. [D-146 joins](F18-D146.json) and [restart rows](F18-AT09-restart.json) are additive indexes.

Historical e80bb592, 0ffb355a and 81b638e8 outcomes stay unchanged. UDP/QUIC receive-only no-PTB remains a limitation; D-123, D-135 and D-136 remain in force.
