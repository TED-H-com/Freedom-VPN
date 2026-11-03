# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 31](../handover/follow-up-31.md) (2026-11-03 EST):** Through #407: isolation 17 records the first qualified D-140 pass; netlab 27 closes r20 H-2 and live drift. D-144 lists are on main; F30-01 remains routed. r21 returned to agent-core 38 before merge. Historical failures stay failed; nine owner decisions.
**Previous cut — [follow-up 30](../handover/follow-up-30.md):** #397–#403 at b9439411; 23 subtests, seven rows, nine snapshots, 449 checks, six dated documents and 61 findings. See follow-up 31 for timing errata.

## Follow-up 31 acceptance snapshot (2026-11-03 EST)

Cut **#404–#407**, main **3573f396**; no unbriefed PR inside the range and no
later merge at task start. [Evidence and nine owner requests](../handover/follow-up-31.md).

| Lane / PR | Result and boundary |
|---|---|
| Isolation 17 (#404) | 83b457ef archived, 129 roots/subtrees. 0ec14b39 is the first qualified D-140 pass under D-142, no retry. D-104 26/7; holds 8/9 including 30 s fallback. F17-01 pure stale_generation remains an acceptance-coverage gap. AT10 9/0/0, AT12 10/0/0; AT11/AT13 operator blocks. Historical failures stay failed. |
| Netlab 27 (#405) | Exact denied_v4 / host_v4 coverage checked on Apply and periodic reads; r20 H-2 and live drift closed. Quartet PASS/FAIL/PASS/PASS: B1 stays failed, not attributable under D-138; D-120 (iv) not waived. F27-01 → netlab 28, F27-02 → updater 20. |
| Devstack 30 (#406) | D-144 scheduled gate and two expected-blocked rows delivered; F30-01 remains a production defect. TEST ONLY survivor continuation, 8576dfe5 10/0/0; members 17 consumer outside cut. Runtime list loading fixes isolation lint's Go-only copy. |
| Handover 30 (#407) | #397–#403 at b9439411; 23 subtests / seven rows / nine snapshots / 449 checks / six dated documents / 61 findings / nine owner requests. Session quota timing corrected in follow-up 31. |

Review r21 is archived and returned to agent-core 38 before merge. Its five
findings are not yet judged; 37 is held, so r20's five agent-core findings stay
open on main. [72-row register](../handover/README.md#findings-register-at-the-follow-up-31-cutoff).
No scheduled-run result is claimed. D-144 is still provisional.

### Reading nightly results (2026-11-03 EST)

Main records no scheduled-run result; this guide claims none. Read each job's
`$GITHUB_STEP_SUMMARY` and retained records. A red nightly is read, never
re-run by an operator or an agent.

- **e2e, scheduled `stack` (04:30 UTC):** D-144's committed
  `tests/e2e/nightly_expected_blocked.json` lists `member-ipv6` (devstack /
  platform operators; IPv6 disabled, no underlay) and `lease-expiry`
  (agent-core, updater co-owner; F30-01, see [node updates](../handover/operator-guide.md#f30-01)).
  Listed blocks keep the job green but stay `blocked` in acceptance records;
  owner, reason and source appear in the step summary and `nightly-gate.md`.
  Listed passes are **unblocked; remove from expected-blocked list**. Any
  fail/error/non-acceptance test failure, unlisted block, or missing/unreadable
  summary or list fails the job. Download `e2e-stack-records-<run_id>-<attempt>`:
  `nightly` JSON/Markdown records, `acceptance-summary.md` and `nightly-gate.md`.
- **at-isolation, scheduled `accept` (05:40 UTC):**
  `tests/e2e/at/isolation/expected-blocked.json` lists only AT11's `IPv6 platform
  matrix` (acceptance / platform operators) and AT13's `continuous
  crash/NetBird/reboot/updater transitions` (acceptance / operator). Both need
  operator matrices. Exact listed case/row blocks map to green without changing
  acceptance; invalid pass evidence, duplicates, unknown blocks and errors fail.
  Actual reasons are logged. The historical short-expiry block is unlisted
  because status-only logs cannot recover its cause; it stays red until a
  scheduled run names the reason. Listed passes are reported for removal.
  Download `at-isolation-<run_id>-<attempt>` for records and
  `scheduled-job.md` / `scheduled-job.json`; read the step summary too.
- **Both record uploads** run with `if: always()` and retain records **7 days**.
  Local, pull-request and manual acceptance semantics remain unchanged.
- **Other scheduled jobs:** netlab (03:17 UTC), dns (03:15 UTC), NetBird CE
  adapter and G1 lab (04:23 UTC) retain their workflow logs; this cut changes
  neither their behavior nor the absence of a new scheduled result.
