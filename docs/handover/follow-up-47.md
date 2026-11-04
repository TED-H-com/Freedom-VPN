# Follow-up 47 — #474–#477; scheduled harness fixes, D-162, first live wait, r31

Verified against main **1c39a0e73825dbcf911dfd2bec6e75aed64c979b**, through
**agent-core 52 #477 / 341d099a**, on **2026-11-04 EST**. Follow-up 46 ended
at **#473 / 52b93845**. Exactly **four merges and four final roadmap rows**:
**#474, #475, #476, #477**; no unbriefed PR merged inside the range.

**Running at briefing, status only:** **isolation 25**, taking F24-01, the
D-098 archive of run df9dbad7 and one assembled run in scheduled mode;
**members 24**, launched with this cut for the dual 1280 rows under D-162,
reading agent-core 52's two records in the native window; **review r32**, a
post-merge code review of agent-core 52 by **agy/gemini-4-argon-high**.
No delivered or merged PR was present at the initial read-only check. Later
results are **after the cutoff** and do not enter this cut. No date is promised.

**D-098 / D-109 / D-132:** `TestHandover_Followup47Claims` pins the final
roadmap rows, PR excerpts, frozen producer documents, archived run records,
review provenance and main Go test names. All lane-document quotations use
committed frozen copies. No claims read living members, isolation, secops or
usage evidence roots. Follow-up 46's rewritten inputs now read frozen **#474
bytes**; its assertions and every earlier claims test remain unchanged.

## Handover 46 (#474): previous cut, counts and session

[PR #474](https://github.com/TED-H-com/Freedom-VPN/pull/474) records the
**#469–#473 cut, up to 52b93845**. On main,
`TestHandover_Followup46Claims` has **15 subtests** (nine direct and six claim
groups), **five final roadmap rows and PR excerpts**, **21 byte-identical
frozen inputs** (nine #472 documents and twelve producer records), **23 Go
test declarations**, **18 frozen source readings**, **ten archived artifacts**
and **59 exact JSON values**, decoded as `json.Number`. Its **39 earlier
claims functions** stay exact after reversing only four input redirects.
Its register contains **65 decisions** (D-162 added, D-158 updated, 63 exact)
and **133 findings rows** (10 updated, none added, 123 exact). These are
#474's verified counts, not this cut's changes.

The session, attributed to #474's body: **codex gpt-6-astra**, launched
**09:05 EST** on **2026-11-04** on #473's roadmap row **52b93845**, delivered
**09:32**; **6 commits, 48 owned files, +56,377/−299**. Final gates waited
**13 minutes**, **09:16–09:29**, while isolation 24's suite ran. Docs, claims,
frozen snapshots and digest values only; no reservation, Docker or live measurement.

## Isolation 24 (#475): two scheduled harness causes fixed

[PR #475](https://github.com/TED-H-com/Freedom-VPN/pull/475) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-47/docs_e2e_at_isolation_followup24-2026-11-04.md.txt)
record the failed **11-04 scheduled job 41836274140 on d2cb5446**, with two
harness causes: the member-expiry plan missed the unchanged window, and the
populated-holds review's sole unexercised gap made the job red. The records
of that run and **41640196805** are retained in the lane's history. Run
**328ab9e0's 203 files** are archived byte-identically under **history/follow-up-23/**.
The original scheduled failure stays failed; the next scheduled run is pending.

**D-141 diagnosis corrects the orchestrator's reading.** Generations **29
and 30**, at **A + 2.1 s** and **A + 3.1 s**, already carried the finite member.
Generation **31**, at **A + 19.1 s**, was the first to carry its device binding,
which the wait requires. Most of that interval precedes binding publication.
No record connects it to agent-core 51 **#470**, and no specification,
contract or decision bounds how soon a node renews after a new grant.
**There is no product finding.** The 11-03 run retained no fresh anchor and
is not replayed as one.

**The plan changes, the predicate is byte-identical:** **D = floor(A + 44 s)**
replaces **floor(A + 55 s)**. The binding-bearing lease must issue after
D − 35 s and be seen before D − 5 s, with the member's signed deadline equal
to D and an accepted serving node. Offline proofs replay A + 19.1 s, local
A + 10.5 s / A + 17.1 s and synthetic 10–34 s renewals; slower setup still
blocks. These are TEST ONLY values.

The orchestrator's note, quoted exactly from #475's frozen PR excerpt:

> **The orchestrator's note: the low end has little margin.** A binding-bearing lease issued up to about 2 s before D − 35 s, so between about A + 6 s and A + 9 s, would miss, and its successor would be seen too late. The binding-bearing leases on record were issued at A + 10.5 s, A + 11.0 s (this run), A + 17.1 s and A + 19.1 s. If one lands in that gap, the row blocks; it cannot pass wrongly.

**D-163 is implemented in the scheduled summary.** The explicit
`tests/e2e/at/isolation/expected-not-exercised.json` has one entry beside
`expected-blocked.json`: populated-holds / `pure_stale_generation`, F17-01.
Only this same run's incomplete review with that sole gap, all other items
passing, no join problem and no error can map exit 1 to **expected not
exercised**. Every other error stays red. The review still writes `incomplete`
and exits 1; outputs, acceptance records and D-144's row mapping remain.
A passing listed item is **no longer needed**. `run.py` deletes any earlier
`populated-holds-review.json` before invoking the review. The list follows
D-144: an item is listed only once the runner shows it.

Run **df9dbad7**, measured source **56d81411**, in scheduled mode: **27 pass,
2 blocked, 0 fail**, all **29 statuses match 328ab9e0**. The blocks are
**AT11's IPv6 platform matrix** and **AT13's full continuous fault matrix**,
the two operator matrices. **AT13 member expiry passes** under the new plan:
the qualifying generation **31** was issued at **A + 11.0 s**, so the window
was open at the first sample. All **five requests timed out**, bytes stayed
flat at **2,671**, and all **six reads showed the node serving**. The final
probe finished **3.8 s** before the sampled node lease expired.

**Whole-node expiry's packet verdict passes**, but **D-142 qualification is
not established**. **F24-01**: the added-probe schedule is empty and written
as **null**, and the unchanged reviewer crashes at **`sorted(None)`**. No
boundary review was written. Thus the run's own **scheduled verdict is fail,
for F24-01 alone**. No earlier root review stands in for it; isolation 25 is
running at cutoff. This is separate from the original scheduled job's two causes.

The **D-120 lab passes under D-161** on complete checked captures: **65 TCP
reads, zero TCP timeouts, zero UDP losses**, **nine lost transmissions**, each
attributed to a guard; **longest TCP stall 209.6 ms**. It adds a second pass
to F22-01, still answered by #469. **15 node writes**, each a **first-attempt
202**, with zero conflicts; command acceptance remains separate from applied
completion. Restarts retain **one watchdog `baseline_unverified` decision**,
baseline completion 34 ms later, and **no `GENERATION_MISMATCH`**. The lane
makes **no closure claim** about the startup/watchdog race.

Main proofs include `TestAT13_MemberScheduleRunnerAndFollowup23Replay`,
`TestAT13_MemberScheduleJitterAndEarlyRenewal`,
`TestAT13_MemberScheduleStrictWaitBoundaries` and
`TestD144_NightlyReviewAndSuiteAdmission`; the latter invokes the Python
scheduled-summary tests, including D-163's positive and rejected variants.

## Devstack 36 (#476): D-162 and the first live D-158 wait

[PR #476](https://github.com/TED-H-com/Freedom-VPN/pull/476) and the
[frozen analysis](../../tests/handover/testdata/history/follow-up-47/docs_e2e_runs_2026-11-04_followup36_analysis.md.txt)
record **D-162, TEST ONLY**: both prepared configs carry **heartbeat, poll
and observation at 5 s**. The installer node's values are unchanged and
pinned by `TestD162_InstallerCadenceUnchanged`;
`TestD162_PreparedAgentCadence` pins both prepared configs and the unchanged
**30 s product default**. Every predicate stays, including **15 s freshness**.

Run **c17589fa** on **f6edbda9**: **32 pass, 1 blocked (`member-ipv6`), 0 fail**.
Make exits **2**; the unchanged nightly evaluator **passes, exit 0**. Every
scenario status matches follow-up 35. From the stack database: **123 accepted
regular-node observations / 122 intervals**, **min 0.081 s, median 5.162 s,
max 10.255 s**. The run's last **622 s had no accepted receipt**; the last
came **1.7 s before lease-expiry stopped the control plane**. This setting
is not a promise of exact periodic receipts. **The 15 s reads did not run**,
so this run **does not re-score F22-D146**; members 24 owns the dual 1280 run.

**If-Match:** three node writes, each a **first-attempt 202**, zero 412s.
The consumer audit finds one missing retry outside devstack:
**tests/e2e/at/isolation/assist.py:22**, an operator-assist capacity PATCH
with a single GET/If-Match attempt, **never executed in this run**. It is
routed to isolation after follow-up 25; its assisted classification must remain.

**update-good exits 0 after 446 ms**. Its one output line is the wait record:
**`waited_ms: 446`, `acquired: true`, `budget_ms: 10000`**. Lock samples show
the running agent holding **update.lock within the wait**. This is the
**first devstack run to record D-158's wait**; follow-up 35 was uncontended.
**Stale-then-live contention is unobserved**; no individual metadata sequence
is retained or staged. **update-bad exits 1**, **`HASH_MISMATCH: artifact`**,
with **no wait record**. **lease-expiry passes**. F30-01 stays closed by #468;
this measurement adds evidence without changing that closure's attribution.

This is the **first run with agent-core 51, updater 26 and D-162 together**.
The lane **credits no change to any one of them**. AT29's guest-clock row
remains operator-run; member-ipv6 remains listed.

## Agent-core 52 (#477): exact binding predicate and observation records

[PR #477](https://github.com/TED-H-com/Freedom-VPN/pull/477) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-47/docs_node_followup52-2026-11-04.md.txt)
record **diagnosis first under D-141**, committed on the unchanged base.
r31 **case A**, an isolation error in the report, is **reachable and
persistent**; historical isolation containment can survive full Steps.
**Case B**, a failed meter path with an armed recovery, **is not reached
through normal Steps**: enforcement withdraws and disarms first.
**Case C**, a stale lease, **is reachable for one enforcement pass**, including
after rejected adoption; if still stale the next Step refuses retention.
Zero bindings alone do not start a closed recovery.

**r31 M-1 is fixed** by **`canBindRecoveryReport`**, extracted exactly from
the existing binding rules. Both binding and the dedup exception use it.
At unchanged state with a report pending, a new report is queued only when
an **armed, unbound recovery can bind it**; otherwise ordinary dedup holds.
**Unchanged:** report body, **32-entry outbox**, cadence, retry, backoff,
Retry-After, agent-core 51's unchanged-state binding, **D-067, D-135/D-136,
D-155/D-156 and D-157**. The ordinary no-waiting-recovery dedup path builds
no report and reads no update journal.

r31's proofs are on main: updater byte-identical; agent with only the two
changes recorded in the dated note—three `TODO` labels to `Regression`, and
case C's existing **30 s stale-lease alarm** crossed and asserted before its
baseline. **Every imported assertion is kept; all three M-1 subtests fail
on unfixed code**. Own full-Step proofs cover the persistent latch for
**210 Steps / 35 default 30 s observation intervals**, and transient stale
and pending-revoke cases for **35 TEST ONLY 1 s intervals** after recovery
ends. Main includes `TestD157_F52_IsolationLatchStepsOutlast32Intervals`,
`TestD157_F52_StaleLeaseStepsOutlast32Intervals`,
`TestD157_F52_PendingRevokeStepsOutlast32Intervals` and
`TestD157_F52_CatchupBindsWaitingRecovery`.

Two stable records are documented in the
[frozen Observations section](../../tests/handover/testdata/history/follow-up-47/docs_node_README.md.txt):
**`observation enqueued`**, once after durable queueing; and **`observation
submission`**, once when each submission call finishes. The
[operator paragraph](operator-guide.md#observation-records) quotes the field
list and scope: **per report per process**, **`attempt_scope=process`**,
correlated by **boot_id / process_id**, and by report ID across restarts.
The durable uncertainty counter is unchanged. The frozen dated note excludes
report bodies, URLs, response error messages, credentials, tokens and keys;
deduplicated enqueue, idle Step and waiting retry slots emit neither record.
Main logging proofs are `TestF52_ObservationSubmissionRecords`,
`TestF52_ObservationEnqueueReasons` and
`TestF52_ObservationRecordsDoNotExposeErrorText`.

Its ordinary lab passes **13 of 13**, and both records appear in the retained
lab log. **No assembled live claim**: members' next assembled run reads the
records; devstack can read the same format. No date is promised.

## Review r31: raw archive and dispositions

The [raw archive](../reviews/r31-recovery-lock-fixes-review.md) is **302
lines**, SHA-256 **46300735f2a56cfcab37d22b862153efd2f2895e6e25c244f7af938ca5ec7aaa**,
from branch **r31-recovery-lock-fixes-review**, head
**0e33b9fe3ef62244aab7a35eb1119ef6a7036621**, base
**fd0f249418926e875928d4e6799ac2fc05c6ba9f**. This post-merge review of
**#470 and #471** is by **agy/gemini-4-argon-high**, committed by **Ted H**.
Findings: **0 Critical, 0 High, 1 Medium, 0 Low**. It is now a **D-098 input**.

Its fidelity table marks **r30 H-1, H-2, M-1, M-2** and all **four clarified
D-158 clauses** implemented. r30 M-3's exception is incomplete: **r31 M-1**,
now **fixed by #477**. **Updater needs nothing; no owner question**.
Both proof files are on main through #477: `node/agent/review_r31_test.go`
and `node/updater/review_r31_test.go`. The updater matches the branch's full
digest **00414c8f141bab43caed86eb683d28c0baa50fb3ee27d833711316a6c4e7b132**.
The agent differences and assertion preservation are as recorded above.
[The r31 dispositions](../reviews/README.md#r31-dispositions-at-the-follow-up-47-cutoff-2026-11-04-est)
state r30's status at this cut. Both r30 disposition sections stay byte-identical.

## Decisions, register and operator boundary

**66 decisions:** **D-163 alone is added**, provisional, orchestrator,
**2026-11-04 08:5x EST**, owner may overturn; implemented by **#475**.
Under D-144 the job is green only when known owned gaps remain. A carried
review gap no run can exercise must not make the job red every night, and
stays visible in the summary under D-163's exact mapping. **D-162's row is
updated** to implemented by **#476**, with the measured cadence and no
F22-D146 re-score. **The other 64 rows are byte-identical** to #474; main
had no other new decision after D-162.

**136 findings rows: 9 updated, 3 added, 124 byte-identical** to #474.
New: **Isolation F24-01**, **r31 M-1**, and the **11-04 scheduled job
41836274140**. Updated: members F22-D146, Isolation F17-01 and F22-01,
devstack F30-01 and r30's five findings. Every other row, **Isolation F18-01
included**, is unchanged. No PR in this cut claims its first scheduled
exercise. Members F22-AT01 and F11-01 stay open under **D-148**. Every
historical failure stays failed.

Seven entrypoints point here. The operator guide preserves devstack 35's
F30-01 paragraph except its outdated live-wait sentence, then adds devstack
36's record from its dated analysis. AT02's paragraph and the TCP reload-guard
anchor stay exact. Observation records get their own anchor. Ops records
D-163 as the scheduled job's second list. **Nothing else in this cut changes
what an operator or member sees**: D-162 is TEST ONLY; r31 M-1 restores
ordinary deduplication. Changed owned documents have refreshed digest values
only in `docs/e2e/at/secops/scope-review.json`.

### Requests to orchestrator

**2026-11-04 EST:** follow-up 46's requests are answered by **devstack 36
#476**, **r31** and the observation records in **#477**. F22-AT01's boundary
stays with D-148's owner, already reported upstream. Carry isolation 25's
F24-01/archive/scheduled run, members 24's dual 1280 rows and native-window
record reading, and review r32 as running at cutoff, without importing results.
Route **tests/e2e/at/isolation/assist.py:22** to isolation **after follow-up
25** for the bounded retry and receipts, preserving assisted classification;
it was never executed by devstack 36. This outside-owned implementation item
stays with its owner. No outside-owned pin needs changing; the rest is delivered.
