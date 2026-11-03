# Follow-up 42: r25 judgments — 2026-11-03

D-141: judge before fixing. Production starts at `8cc27609`; r25's proof
file from `1afecb29` was imported unchanged, alone, as `bf68757f`. The
[before-fail receipt](followup42-r25-before-2026-11-03.txt) reproduces all
six failing subtests and all three passing held tests. No fixture edits.

| Finding or sub-trace | Verdict | Reachable | Introduced by | Fix |
| --- | --- | --- | --- | --- |
| H-1a Step entry, device/member sibling | Confirmed | A Step starts at the earlier binding deadline | 41 adds this entry call; underlying whole-node withdrawal is pre-existing | Re-decide; keep survivors until ordinary Apply on this tick |
| H-1b verifyCurrentServing, device/member sibling | Confirmed; review attribution corrected | Deadline before/during a live read, before evaluate/enforce | Pre-existing on main | Only withdraw when the fresh decision cannot serve; otherwise continue the Step |
| H-1b retained live read, device/member sibling | Confirmed | Deadline crossed in locked retained verification | 41 | Re-decide after read; if model changed, fall through to ordinary Apply |
| H-1 watchdog, device/member sibling | Confirmed additional consumer | Between earlier expiry and next heartbeat, including unlocked control I/O | Pre-existing | Heartbeat carries last admitted binding deadline, capped by all node limits; kernel expires each earlier element |
| H-1 false dash/empty reason | Confirmed consequence | Withdrawal followed by a serving decision but early return before Apply | Pre-existing verify path; 41 entry/retain variants | Remove erroneous withdrawal/return; use current blocking reasons when cached decision has none |
| M-1 post-Apply switch | Confirmed | Ruleset nudge rotates epoch after intent write | 41 retention predicate | Accept proven same counting interval; durably advance intent before retaining |
| M-1 sample discontinuity | Confirmed | Sampler rotates epoch with unchanged generation/bindings | 41 retention predicate | Same interval proof and durable-then-memory intent update |

`main` at `792e3e41` sets `policyExpires` to the minimum binding deadline
(`decide.go`, `if deadline < policyExpires`). `verifyCurrentServing` checks
`expireServing` before and after its live read. That helper withdraws, then
re-decides; Step returns without enforce. Thus sibling outage and the
contradictory report predate 37–41. Follow-up 41's per-device kernel timeouts
remove any need to withdraw other devices at the earlier deadline.

Watch.Once acquires the node lock, reads the heartbeat, tests `policy_expired`,
and installs baseline after a second check. Step's control pass holds no node
lock. A fresh heartbeat whose earliest binding expires during that pass can
therefore close the entire node; lock serialization does not prevent it.

Planned deadline separation: retain `policyExpires`/Decision.PolicyExpires as
the earliest installed boundary for nextStepDelay and retention comparison.
Derive the whole-node deadline from the installed model's last admitted
binding, intersecting each device with its member and node caps. Heartbeat
PolicyExpires (including boot/adoption and its diagnostic record) uses that
whole-node limit. Missing deadlines still fail closed. At an early boundary,
expireServing re-decides without withdrawing survivors; retained verification
must re-decide after elapsed local reads. Reports still describe actual
forwarding, with existing reasons preserved and refreshed if absent.

The existing identity-change accounting handoff also installs baseline.
For the narrow same-generation, expired-only removal case, first verify the
installed rules, pause and checkpoint all outgoing counters, then persist
the surviving intent and Apply. Expired kernel elements cannot increment
those retired counters; surviving identities remain unchanged. Other changes
retain the existing baseline/final-sample boundary. Failures still withdraw.
This is ordinary Apply, not intersection serving or a hitless claim.

M-1's intent and head already use SameCountingInterval on restart (40).
Retention introduced strict equality in 41. Refreshing at retention after
proving continuity handles both rotation producers without changing the
sampler's concurrency model. Persist the new intent before publishing it in
memory; failed persistence must withdraw with a meter storage reason. A final
sample, boot change or closed epoch cannot satisfy the continuity proof.

F32-01 is confirmed by inspection: two bytes modulo a nanosecond span yield
approximately fixed minus twelve seconds at a 120-second lead, re-drawn on
each check. Replace this with uniform integer-millisecond jitter, cached by
lease identity. Test endpoints, midpoint, reuse, next lease and read failure.
The nominal lead is 108–132 seconds at default; the jittered upper bound is
1.1 times renewLead's capped base (0.55 of the lease window when that cap
applies, subject to the existing one-second base floor).

What held: renderer input and pending-gate comparison, kernel device bounds,
r23 restart accounting, prior test coverage and both fixture corrections.
The usage lab retry remains unchanged: its exact counter/ledger equality and
accept-rule counter make forwarded-but-uncounted traffic visible.

## Requests to orchestrator — 2026-11-03

Schedule the owed D-133 A,B,A,B for 37–42 after review in a quiet slot.
No timing measurement or reservation is part of this follow-up. The only
node/enforce change will be the expressly approved DeadlineMono comment.
