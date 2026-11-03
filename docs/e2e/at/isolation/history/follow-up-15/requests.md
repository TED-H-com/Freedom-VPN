# Isolation follow-up 15 requests — 2026-11-01

Three rows: **two carried operator assignments**, without renewed asks, and
**one new D-140 qualification follow-up**. Run
`0b9e4510-776f-4c51-9e66-500404ce9f7e`; [full evidence](summary.md).

1. **Carried — AT11, existing platform/operator assignment.** Native dual-stack
   and member-platform route/capture matrix remains blocked. No new request.
2. **Carried — AT13, existing acceptance/operator assignment.** Full continuous
   crash, NetBird restart, kernel reboot, updater and physical-arm64 matrix
   remains blocked. Four parent wait receipts do not close it. No new request.
3. **New — orchestrator / isolation next follow-up, D-140 qualification.**
   The unchanged evaluator passes this trace, but the requested schedule
   noninterleaving condition fails: packet172 ingress precedes packet171 egress
   by **3.821 µs**, both before the first drop. The 0.6 ms measured spacing
   exceeded node transit but omitted the measured **3.675698 ms** latency spread.
   [All times and the failed qualification](boundary-review.json) are retained.
   Final source includes the spread in its formula and would choose **4.3 ms**
   from this calibration; that correction is **not assembled-qualified**.
   Schedule the next independent sample under D-140; this task's one-run limit
   precludes a retry. No product change, evaluator change or waiver is requested.
   Historical **isolation F14-01** remains failed, distinct from **members F14-01**.
   The four historical outside-guard renewal losses (15–18, 30→31) stay counted.

**Populated hold: not exercised — devstack 27's fixture pending.** This is the
already assigned next-follow-up dependency, not a renewed request here. No hold
report occurred in this run; no alarm/escalation/active-stop evidence is inferred.

Devstack26's D-102 stop hand-off is consumed: 15-second direct-agent budget,
25-second whole-container and clone grace, pre-stop process identity, matching
parent receipts copied with `docker cp`, and separate PID1 outcomes. The terminal
source's actual signal15 is retained; fast exit is not assumed cooperative.
`.sigkill.json` would be attempted escalation only. Historical forced exits stay
unchanged. Step cancellation timestamps remain unavailable.

No change outside owned paths was needed. The handover claims test already reads
immutable archives, so the D-132 one-file exception was not used. Large-JSON links
were re-pointed only in owned living Markdown. Earlier archives remain unchanged.

Historical requests and outcomes: [follow-up14](history/follow-up-14/requests.md),
[follow-up13](history/follow-up-13/requests.md). F13-01 retains netlab23's typed
`duplicate_chain` decision and unproven origin; no read failure becomes proof of
frozen/removed counters or closed forwarding.
