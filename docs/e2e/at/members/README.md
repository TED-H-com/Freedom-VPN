# Member acceptance

Current evidence is [Follow-up 12](RUN.md): eight fresh cases plus explicitly preserved follow-up 11 AT01/AT09 records. [Summary](summary.md) separates these scopes. The full target exits nonzero for blocked or failed cases. No acceptance claim follows merely from an API response, fixture availability or diagnostic label.

Use the assigned `FVPN_E2E_PREFIX=w4-members` / `FVPN_E2E_PORT_BASE=24000` and a fresh two-resolver stack. `assembled.py` hard-stops if two other integration suites are active, checks again after startup, runs the foreground live target, and tears down in Python `finally`. It sets Follow-up 12's explicit AT01/AT09 skip mode, retaining the necessary identity setup inside AT02. The regular make target retains all ten cases. [Run commands and checks](RUN.md).

Disk scratch uses `disktmp.py` under `.agent/tmp`; use `go env GOCACHE` rather than an assumed home path. No shell cleanup traps or tmpfs. The primary scenario deadline is 50 minutes, Go limit 55 minutes.

Positive forwarding requires populated signed member grants, a fresh accepted serving report, the intended binding and the original same-boot producer serving record, then a separate origin receipt, SNAT and device-counter assertion. [AT02](AT02.md) defines the exact predicate. The corrected reader streams cumulative producer JSONL through the existing export hook; its live qualification is pending, while this run's export-limit failures remain recorded. [D-077 API index](D077.md), [forwarding boundaries](forwarding-observation.md).

D-135 measures the admitted sibling's failed-Apply outage from baseline through the next successful Apply; every loss stays counted. D-134's expiry window does not waive pre-deadline/renewal loss. Agent-core host samples support a conditional diagnostic label, never CPU causation or a verdict waiver. [Full original producer stream](AT02-enforcement-timing.json), [recovery capture](F12-AT02-recovery.json).

AT20 retains actual resolver, TTL, settings and whole-sequence lifecycle evidence under F9-HIT/F9-OTHER/F9-ROUND/F9-INFLIGHT. Distribution, fallback and same-resolver cache behavior are separate. No cache assertion is changed to pass a row. [AT20](AT20.md), [cache sequence](F12-AT20-cache.json).

[Requests](requests.md) and [blockers](BLOCKERS.md) keep owner boundaries. [Operators](OPERATORS.md) and [procedures](PROCEDURES.md) retain platform/physical campaigns. [NATIVE.md](NATIVE.md) is preserved follow-up 11 evidence, not a fresh Follow-up 12 run. D-129 keeps generation-29 recovery valid; it never passes all AT09 selectors.

The 43-file [follow-up 11 archive](history/follow-up-11/SHA256SUMS) is byte-identical; its interrupted attempt and all older history stay unchanged. Historical F10-02/F10-03 failures remain failed/unattributed. D-132 allows evidence pins only under `history/`; root documents are living evidence.
