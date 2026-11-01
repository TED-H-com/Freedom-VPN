# AT02 finite grant and device-to-usage trace — Follow-up 12

Run `b98b78ee`, harness `65e18e67`. Full AT02 is **blocked, 150 pass / 0 blocked / 0 fail checks**: F11-03 has no failed Apply to measure. API bodies, audits, device-to-usage joins and all original samples stay in [AT02.json](AT02.json).

| AT | Cohort | Event UTC | Last TCP / UDP reply delta (s) | Target (s) | Samples | Result |
|---|---|---|---:|---:|---:|---|
| AT02 | member-a3 | 2026-10-31T21:59:22.745247458Z | 0.326458120 / 0.326458120 | 5 | 28 | pass |
| AT02 | member-a3 | 2026-10-31T22:01:26Z | -0.085203130 / -0.085203130 | 5 | 193 | pass |
| AT04 | member-paid2 | 2026-10-31T22:09:19.760245577Z | 3.351259294 / 3.351259294 | 60 | 759 | pass |

AT02 has 221/221 successful reads and 442 before/after snapshots joined to the 812-record producer stream: 64 baseline, 364 heartbeat, 25 renewal, 89 Apply, 268 live-verification and two blocked-audit records. No failed Apply occurs; all 89 Applies retain host context. Finite expiry has 193 reads, last reply increments 85.203130 ms before the deadline. These are sampled observation bounds, not exact packet cutoff or kernel commit timestamps. D-134 preserves every pre-D and renewal loss; no fixed outer-guard ceiling is asserted.

The continuous sibling capture separately records 21 losses from 3,976 attempts. D-135 is blocked, while a distinct policy-missing baseline bracket remains F12-01; neither disappears from totals. [Recovery/capture](F12-AT02-recovery.json), [withdrawal records](F12-AT02-withdrawal.json), [complete stream](AT02-enforcement-timing.json), [policy-missing window](F12-01-policy-missing.json).

D-119 retains immediate 3/0 → 2/1 → 3/0 replacement and independent finite expiry to 0/3. Worker completion requires the succeeded parent; one revoke audit per grant remains required. The two D-077 re-grants pass only after populated-grant serving. Historical F10-02 stays failed/unattributed, and F11-03's old expectation stays unchanged in [follow-up 11](history/follow-up-11/AT02.json).
