# Follow-up 44: measured clock prerequisite failure — 2026-11-03

The post-reservation probe measured a different time namespace inside the lab
container. Runner namespace: `time:[4026531834]`; container namespace:
`time:[4026534102]`. Both report zero monotonic and boottime offsets.

All three calls to the frozen B `timeanchor.Mono()` lie between direct kernel
CLOCK_MONOTONIC reads, and those reads lie inside the runner's host brackets.
The arithmetic is aligned in these observations. Nevertheless, this does not
satisfy the declaration's **no separate time namespace** prerequisite. The
check correctly rejects all three samples. No clock offset is invented or
subtracted, no namespace is changed and no timing leg is requalified.

| Probe | Host before (ns) | Kernel before (ns) | timeanchor.Mono (ns) | Kernel after (ns) | Host after (ns) |
| --- | --- | --- | --- | --- | --- |
| 1 | 54656085115928 | 54656163355032 | 54656163355812 | 54656163355970 | 54656164674952 |
| 2 | 54656164817684 | 54656244768742 | 54656244769228 | 54656244769369 | 54656246106257 |
| 3 | 54656246213724 | 54656325802883 | 54656325803553 | 54656325803703 | 54656326884706 |

The [JSON receipt](followup44-clock-prerequisite-2026-11-03.json) retains all
UTC/monotonic readings, offsets, namespace identities, source/probe binary
identity and raw file hashes. This was outside both reservations, during the
initial ordinary lab setup; the extra timing-prerequisite check aborted that
setup before any scenario. It is not a ninth timing leg or functional gate
pass. The subsequent ordinary functional gate omits this timing-only check.

2026-11-03 request to orchestrator: resolve the measured namespace prerequisite
before any newly declared timing follow-up. The two authorized reservations
are exhausted. This lane changes neither Docker's host configuration nor the
clock/namespace qualification rule.
