# Follow-up 9 measured outcome

AT06 **662 pass / 0 blocked / 0 failed**; AT07 **48 pass / 0 blocked / 0 failed**.

Retired usage: **908 + 1816 = 2724 bytes**, original ACK **3**. The fresh stream independently advances **1 → 2**, attributing **550 upload + 358 download = 908 bytes** to its own incarnation, member, device and binding revision. Historical payloads, hashes, samples and totals remain unchanged.

| Row | Status | Evidence |
|---|---|---|
| installer_rerun | pass | Same identity, stopped file equality, restored serving and counted traffic. |
| controlled_reinstall | pass | Same Node, new incarnation, retired certificate denial, fresh serving, gap ACK 1 and counted event ACK 2. |
| counted_unacked_usage | pass | Held CLI records plus persistent TCP/UDP receipts and admission counters. |
| identity_ledger_preserved | pass | Original canonical payloads/hashes/ACK survive production adoption and original-stream upload. |
| historical_usage_retained | pass | Four named AT20 assertions, once-only samples and distinct retired/current totals. |


Gap sequence 1's actual stored interval: `2026-10-31T15:11:08.150045Z` → `2026-10-31T15:11:08.150046Z` (1 µs).
The saved current production receipts select the current incarnation and report
accepted events at ACK 1, then ACK 2. The original stream remains separately
ACKed at 3; all four named AT20 assertions pass in this run.

AT06.json and AT07.json are unchanged raw runner output. Their prior_checks retain
every failed attempt. requests.md is the request-document snapshot taken by the
runner; the [final requests](../../requests.md) and [gap resolution](../../current-stream-gap.md)
supersede its historical follow-up-8 wording. [First follow-up-9 failure](../6c05c918-0819-4ea2-94f9-1d7e12e84c35/REVIEW.md)
remains failed evidence; only its documented harness assumption is resolved by this run.
