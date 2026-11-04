# Follow-up 41 — #435–#438; harness fixes and a second D-150 attempt

Verified against `main` **c359d4a9447fded8b035d9a084ae24999850dbf0**, the
final roadmap commit through **#438**, on **2026-11-03 EST**. The range starts
immediately after #434 / **3f7dbc25** and contains exactly **#435, #436, #437
and #438**: four merges and four final roadmap updates. No unbriefed PR merged
inside the range. Main and the public PR listing were at this cutoff when
checked at task start.

**Running at briefing, status only:** isolation **20**, launched on #438's
merge, moves its six node writes to the shared helper, cleans up early-abort
diagnostics and runs one assembled run; agent-core **46** makes D-150's third
attempt; the orchestrator's read-only **r27** diagnoses members 18's four
failing subchecks (AT02's two and AT09's two). No delivered or merged PR for
these follow-ups was present at task start. Any later delivery is **after the
cutoff**; none of their results enters this cut. **r27 is not archived in this
cut.** The existing [review index](../reviews/README.md) is unchanged.

**D-098 / D-109 / D-132:** `TestHandover_Followup41Claims` pins the four final
roadmap rows and public PR excerpts, exact #435 inputs, dated agent-core 45
records, devstack's dated/run-ID records and Go test declarations on main.
Members 19's three dated notes are frozen from the cutoff into the owned
`history/follow-up-41/` archive, with complete source hashes. No claims test
reads a living members, isolation or usage evidence root. Earlier claims keep
every assertion; four reads in follow-up 40 move to frozen #435 bytes.

## Handover 40 (#435)

[PR #435](https://github.com/TED-H-com/Freedom-VPN/pull/435) records the
previous cut, **#430 to #434, up to 3f7dbc25**. The actual claims function and
manifests on main confirm **14 subtests**, **five final roadmap rows and PR
excerpts**, **ten frozen #430 inputs**, **76 immutable records and 11 frozen
run snapshots**, members' **ten-case and 77-subcheck inventory**, and **84 Go
test declarations**. Its register has **55 decisions and 102 findings rows**.
These counts are checked against the retained function, manifests and records.

The session account is attributed to #435's body: the first session received
the brief from before the orchestrator's last amendment and was stopped after
**1.5 minutes with a clean tree**. A fresh session launched at **17:45 EST**
and delivered at **18:06**, with **5 commits and 46 owned files**. Admission
waited **329.2 s in total**, and the **suite wait was 0 s**.

## Agent-core 45 (#436)

[PR #436](https://github.com/TED-H-com/Freedom-VPN/pull/436) and the
[dated outcome](../node/followup45-outcome-2026-11-03.md) record D-150's
measurement after the merge, retried with 43's frozen arms: **A 567080be**,
**B 38420dad**. B's production equality to #429 was re-proved. The corrected
clock condition **passes the dry preflight and all three A3 samples**:
different time namespaces, zero monotonic/boottime offsets on both sides,
and aligned host/probe brackets.

**No comparison:** no leg reached a scenario, so each arm has **zero pooled
legs**. The [results](../node/followup45-timing-results-2026-11-03.json) retain
empty distributions and absent quantiles, not successful measured operations.

| Window | What happened | Boundary |
|---|---|---|
| 1 | A1 was refused: the suite-process check matched the session's own parent shell, whose declaration-writing command text carried a suite-pattern literal. | No scenario; B1/A2/B2 not started. |
| 2 | A3 was admitted. Then make exited 2 after **14.0 s**, before any scenario. The frozen harness reads Docker's combined stdout/stderr and expects exactly `true`; the admission text made it `free`, then `true`. | No scenario; B3/A4/B4 not started. |

**Both are harness failures, not product or clock failures.** Both reservations
were released; no third was taken. The dry preflight had separate streams and
missed the exact combined-output consumer. The runner and standalone-launch
requests are routed to 46; its results remain outside this cut.

**Functional evidence only:** main's ordinary production lab passed **all 13
scenarios**, D087 / D104 included, on **3f7dbc25**, including netlab 34's
D-146/D-147 rules. The [receipt](../node/followup45-production-lab-2026-11-03.json)
records the expiry predicate true: **97 positives and 15 drops**, with **zero
positive egresses at or after expiry**. Renewal delivered **1,517 of 1,518
probes**; the **one loss is inside the guards**, zero outside. The **zero-loss
renewal predicate is false**, reported rather than asserted; 13/13 does not
establish loss-free renewal and supplies no pooled timing leg.

**D-150 status only:** attempted in **44 and 45 with no comparison**; **46
repeats it**. Its substance and all other measurement rules stay unchanged.

## Members 19 (#437)

[PR #437](https://github.com/TED-H-com/Freedom-VPN/pull/437) answers members
18's grant-restoration request. The
[frozen dated note](../../tests/handover/testdata/history/follow-up-41/members19-summary.md.txt)
records `finiteExpiry`'s two early returns: after a failed finite-grant
replacement and after a failed forwarding trace. Both restore member A to
**3 authorized / 0 pending**, through **POST /api/v1/admin/members/{id}/grants**
on the **same plan revision**.

The named product rules are unchanged: `members.Service.Grant` rejects an
overlapping grant only on a different plan revision; `entitlement.Compute`
merges same-plan grants; `reconcileDevices` re-admits previously authorized
eligible devices. The original failed check keeps its **name, expectation and
text**. The normal path still restores **once**, with its **two existing
checks unchanged**. Early-return restoration has its own two checks:
`finite expiry early-return restoration HTTP` and
`finite expiry early-return restored 3/0`.

The audit fixed **six more sites** in the members lane: `reauthorizationLimit`
after the lower-plan grant and after the fence; `reauthorization` after the
fence; `populatedHolds` for member B's device; the entitlements case's node2
publication when no paid member exists; and `startSibling`'s capture and sampler.
Two shared-helper sites are outside that lane: `Stack.ReplaceFiniteGrant`
revokes before replacing, and `Stack.PrepareNativeRouter1280` does not roll
back setup that fails midway. They remain open, routed to devstack.

Offline tests pass **with and without the e2e tag**:
`TestAT02_FiniteExpiryRestoresOnEveryEarlyReturnAndOnceOnNormalPath` (six cases)
and `TestAT02_AuditedEarlyReturnRestorations` (five). **No assembled run.**
Every committed run record is untouched, including 118271c1 and 50117c68;
**AT02's and AT09's recorded failures are unchanged**. r27's diagnosis is
status only. The next authorized live run exercises the normal path; cleanup
paths are exercised live only if their triggering failures occur.

## Devstack 32 (#438)

[PR #438](https://github.com/TED-H-com/Freedom-VPN/pull/438) answers F19-02 on
the shared publisher side. Its [dated audit](../e2e/runs/2026-11-03/followup32/audit.md)
records the rule: **only HTTP 412 RESOURCE_VERSION_CONFLICT** triggers a node
re-read. If intended values already match, the command succeeds **without
another write**. Otherwise it retries with the fresh **ETag**, the **same
Idempotency-Key and body**. Bounds: **three writes, ten seconds per command**,
including re-reads. Other statuses, other 412 codes, failed re-reads and
transport errors stop; implicit HTTP replays and redirects are disabled.

The same key is valid because its reservation shares the action's transaction:
a **412 rolls both back**, and **only durable successes are stored**. Receipts
retain every attempt and re-read, UTC times, sent If-Match, status/code and
body SHA-256 (added during the resume). A supplied current version is recorded
**when present**; the re-read supplies the retry ETag. Publication still needs
operation success and observed active read-back; a 202 alone is not completion.

| Converted site | Node write |
|---|---|
| publish_test | Capacity PATCH and publication PUT (two sites). |
| followup18_native_test | Retained-node capacity PATCH. |
| followup27_live_test | Populated-hold capacity PATCH. |
| followup30_live_test | Survivor-hold capacity PATCH. |
| scenarios_test | Node revoke POST. |
| exports_live_test | Clone-reconciliation revoke POST. |
| installer_exports_test | Installer-node revoke POST. |

These are **eight converted sites**. **Four unchanged writes** cannot race
with node observations: followup20's fleet-config PUT, serving's and
feed_scenario's policy-activation POSTs, and followup30's member-suspension
PATCH. The audit also routes AT-owned sites to isolation, secops, enrollment,
members, usage and updates; isolation's six are assigned to isolation 20.

Every live invocation is retained in the
[dated original and appended record](../e2e/runs/2026-11-03/followup32/README.md)
and [resume](../e2e/runs/2026-11-03/followup32/resume.md):

| Invocation | Mode | Outcome |
|---|---|---|
| e22872b7 | Forced | Died before enrollment: admission `free` contaminated Docker JSON output; zero publisher writes or real 412s. Its one completed scenario is only a partial summary; the invocation failed. |
| c52b760c | Ordinary | **8/8**, two 202 writes, zero conflicts. |
| af726b13 | Forced, after the orchestrator's resume | Both writes recovered 412→GET→202, but the lane's own proof check wrongly required a version detail. **5 pass / 1 blocked**, downstream member/serving not run. The proof was fixed; this run stays blocked. |
| 95fb7eab | Forced | **8/8**. Capacity PATCH and publication PUT each received **one real 412, one re-read and one same-key, same-body retry returning 202**. One durable accepted operation row per key; publication succeeded and observed active, member admission/counting and populated serving passed. |
| 3ad796d4 | Ordinary | **8/8**, no conflict; each write accepted once with 202, downstream predicates passed. |

**Limit:** the forced header is **"0"**, rejected **before any version lookup**
with empty details. The separate diagnostic was one rejected PATCH, not a
sixth publisher invocation. The proof correction accepts current-version
details when present; synthetic tests cover both empty and populated details.
This proves handling of the real status/code; **isolation 19's observation
race is not reproduced**, and the nonzero-version lookup branch is not
exercised by the forced proof. Its competing write remains unproven.
**Isolation 20's assembled run is pending** at this cutoff.

## Decisions and findings

**No new decision; 55 decisions.** No decision was added after **D-152** on
main at the cutoff. Only **D-150's status** changes; **D-151 and D-152 stay as
they are**, along with all other decision substance.

The [findings register](README.md#findings-register-at-the-follow-up-41-cutoff)
has **105 rows: three updated, three new, 99 unchanged**. Updated: D-150's
measurement in the Netlab F27-01 / B1 row; members 18 grant restoration answered
by #437; F19-02 answered on the publisher side by #438, assembled run pending.
New: shared e2e helper early returns; AT-owned node-write sites; zero If-Match
contract detail. Every other row is byte-identical. **Every historical failure
stays failed**, including members AT02/AT09 and both publisher failures; the
12 historical isolation checks stay blocked.

Nothing in this cut changes what an operator or a member sees. It is test
harness, records and a timing attempt. Operator/member procedures and the
existing D-146/D-147 guidance remain unchanged; seven entrypoints now point
here. No date is promised for anything not on main. No scheduled acceptance,
platform qualification, physical arm64 evidence, C1 implementation or timing
comparison is added.

### Requests to orchestrator

**2026-11-03 EST:**

1. **Devstack:** members 19's shared-helper cleanup request, specifically
   `tests/e2e/grant_fixture.go` (`Stack.ReplaceFiniteGrant`) and
   `tests/e2e/native_underlay.go` (`Stack.PrepareNativeRouter1280`). Open.
2. **AT owners:** route #438's node-write audit to isolation, secops, enrollment,
   members, usage and updates; retain expected-error cases and acceptance
   predicates. Isolation's six sites and its assembled run are in isolation
   20, running. All AT changes are outside this lane's paths.
3. **Contract and node-API owners:** resolve the zero-header/detail mismatch in
   `contracts/openapi.yaml` and `internal/nodes/nodes.go`: pattern allows "0",
   412 describes current-version details, parser rejects zero before lookup
   with empty details. Open; no policy or production value invented here.
4. **Agent-core 46:** retain 45's already routed runner/standalone-launch work
   and third D-150 attempt. r27 remains the orchestrator's read-only diagnosis,
   not an adopted review or acceptance result in this cut.

These are routing hand-offs, not edits outside the owned paths. No
outside-owned pin needs changing and no item is stopped. The existing owner
resource and acceptance gaps remain with their recorded owners.
