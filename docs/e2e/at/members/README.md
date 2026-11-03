# Members acceptance evidence

**2 PASS / 6 BLOCKED / 2 FAIL**, with **679 passed / 6 blocked / 7 failed subchecks**. One assembled ten-case run; live make exit **2**. No assembled run was repeated after the reboot.

Run `0ffb355a-945b-4198-a551-3794120a686c`, measured harness `3a8b61594b447d164bb7c9481545e207f8688ec2`, baseline `d640ce96d9145e57a60e04e5875e049070f30b96`.

**D-133 contention:** the orchestrator reported CPU-heavy handover gates at **2026-11-01 17:32:43–17:37:12 UTC**. Every timing, failure and anomaly overlapping that window is **contention-confounded**, with `cause_status=unproven`; measured values and verdicts stand. [Complete time and step index](history/follow-up-16/F16-contention.json) includes 46 AT02 steps completed in the window and cumulative references in later records. Finite-expiry withdrawal/restoration, the second re-grant, SIGTERM/usage stop/resume, node2 preparation, alarm recovery and sibling removal overlap. The 4,724-send stream, initial AT01 selection, sole-member/retained-safety holds and fresh AT09 measurements are outside this reported window.

[Results](summary.md), [run ledger](RUN.md), [diagnostics](DIAGNOSTICS.md), [owner assignments](requests.md).

D-098 is the first separate commit `56eb3c0b`: all **64 root files** of e80bb592, including gzip bytes and sidecars, are byte-identical in [follow-up 15](history/follow-up-15/RUN.md), with [SHA-256 pins](history/follow-up-15/SHA256SUMS). Every earlier archive is untouched. Handover claim tests read frozen evidence and explicitly reject living root pins; no D-132 one-file exception was needed. The plain [AT02.md index](AT02.md) is retained for Markdown links.
