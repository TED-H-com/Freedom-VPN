# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 30](../handover/follow-up-30.md) (2026-11-03 EST):** Through #403: agent-core 36 refuses same-boot adoption on changed deny / host sets; r20's 2 High / 3 Medium / 1 Low are routed, not yet judged, and the live-drift verifier check stays open with netlab 27. Isolation 16 unqualified on a 3.609 µs overlap; F16-01's fixture disposition merged, its run stays failed. Members 16: AT02 / AT09 fail. The nightly billing failure was a missing runner browser; no scheduled-run result is claimed. D-142–D-144 registered. Nine owner decisions.
**Previous cut — [follow-up 29](../handover/follow-up-29.md):** #391–#396 at d640ce96; 22 subtests, six rows, eight snapshots, 731 checks and 51 findings. Historical failures stay failed.

## Follow-up 30 acceptance snapshot (2026-11-03 EST)

Cut **#397–#403**, main **b9439411**. Nothing was merged after the cutoff at task
start. [Evidence and nine owner requests](../handover/follow-up-30.md).

| Lane / PR | Result and boundary |
|---|---|
| Devstack 28 (#397) | Nonzero ctdenyd (80 packets / 80,000 bytes) survives parsing and changes neither usage nor member presence in ten variants. Coverage only; no leak, no production change, no stack started. |
| Handover 29 (#398) | #391–#396 at d640ce96; 22 subtests / six rows / eight snapshots / 731 checks / four dated documents; 51 findings. Seven owner decisions carried. |
| Isolation 16 (#399) | 0b9e4510 archived (123 roots). 83b457ef on #394: EvaluateDeadline passes, but a 3.609 µs overlap fails D-140 qualification; row unqualified, nothing retried. AT10 9 / 0 / 0, AT12 10 / 0 / 0; AT11 / AT13 operator blocks. Populated holds pass through the TEST ONLY seam; pure stale_generation and the 30 s fallback not exercised. One 120.467 ms install over the guard is an observation. F16-01 routed to netlab. |
| Members 16 (#400) | e80bb592 archived (64 roots). 0ffb355a on #396: 2 pass / 6 blocked / 2 fail; AT02 and AT09 fail, F11-01 first selection misses 5 s. Contention window 17:32:43–17:37:12 UTC confounded. Survivor-hold row not exercised. Measured harness 3a8b6159 is published as 5d9e5f04. |
| Agent-core 36 (#401) | Changed deny / host sets refuse same-boot adoption: baseline, then first full Apply; 0 / 56 and 0 / 12 replies. Signal handling held through Close explains isolation 15's signal 15. A,B,A,B under D-138: no attributable regression. r20 archived: 2 High / 3 Medium / 1 Low routed, none confirmed. |
| Devstack 29 (#402) | Nightly billing failures were a missing Playwright headless shell on the runner. e2e-browser plus workflow install; 31 pass / 2 blocked / 0 fail. Records uploaded with if: always(), 7 days. No scheduled-run result claimed. |
| Netlab 26 (#403) | F16-01 fixture disposition, TEST ONLY: exact timeout read-back kept, each expiry bounded on its own; 4 / 1,000 old rejections, 1,000 / 1,000 accepted. Run 83b457ef stays failed, cause unproven. |

D-142, D-143 and D-144 are registered; D-135, D-136, D-137, D-138, D-140, D-141,
D-142 and D-144 stay provisional, and D-143 is the owner's decision. The
[61-row findings register](../handover/README.md#findings-register-at-the-follow-up-30-cutoff)
keeps every historical failure failed. The node configuration section now
describes the refused adoption, the open live-drift check and r20 H-1. No later
lane result is included in this cut.

### Reading nightly results (2026-11-03 EST)

Main records no scheduled-run result; this index claims none.

- **e2e, scheduled `stack` job (04:30 UTC):** download the run's artifact
  `e2e-stack-records-<run_id>-<attempt>`. It holds the `nightly` scenario
  records (`*.json`, `*.md`) and `acceptance-summary.md` from
  `docs/e2e/runs/<utc-date>/nightly/`, uploaded with `if: always()` and kept
  7 days. Stack state, secrets and cookies are never uploaded (#402).
- **Other scheduled jobs**, as their workflows on main define them: `netlab`
  (03:17 UTC, short network lab), `dns` (03:15 UTC: unit, AdGuard Home
  integration and publish jobs), the NetBird CE adapter and G1 lab (04:23 UTC)
  and at-isolation `accept` (05:40 UTC, AT10–AT13). They upload no records;
  read the GitHub Actions run log.
- **A red nightly is read, never re-run** by an operator or an agent.
- **D-144** decides when a scheduled job fails and lets each job keep a
  committed expected-blocked list. No such list is on main at this cutoff, so
  a blocked row still fails its job; devstack 30 (e2e `stack`) and isolation 17
  (at-isolation `accept`) apply it.

