# Follow-up 30 — restart set equality, r20, an unqualified sample and the nightly browser

Verified against `main` **b943941157ab6aeba6b04c79377de686c22bfbb4**,
through **#403** on **2026-11-03 EST**. This cut is **#397–#403**:
devstack 28, handover 29, isolation 16, members 16, agent-core 36,
devstack 29 and netlab 26. Nothing was merged after the cutoff at task start.
Agent-core 37, devstack 30, isolation 17 and netlab 27 were running; members 17
may follow. (During this run's host-admission wait, **#404** isolation 17 and
**#405** netlab 27 merged onto `origin/main` after `#403`.) Their results are outside this cut. Handover ran no new live
acceptance.

**D-098 / D-109 / D-132:** `TestHandover_Followup30Claims` pins
[seven final roadmap rows](../../tests/handover/testdata/followup30-main.json),
[nine frozen handover snapshots](../../tests/handover/testdata/followup30-snapshots.json)
and [dated evidence / archive manifests](../../tests/handover/testdata/followup30-evidence.json),
including **449 archive and run file checks**. Go test names are checked on
main. Isolation's run 83b457ef, members' run 0ffb355a and usage's run b8a6e9e1
root records are living: cite them by roadmap row / PR only, never as
claims-test inputs. Agent-core 36's dated hand-off is pinned only through its
own sections; agent-core 37 adds a dated correction section to that file later.
No roadmap row was edited.

## Previous cut — handover 29 (#398)

[Follow-up 29](follow-up-29.md) covers **#391–#396 at d640ce96**.
On main, `TestHandover_Followup29Claims` has **22 subtests / 6 final roadmap
rows / 8 frozen snapshots / 731 archive and run file checks / 4 dated evidence
documents**, and its findings register has **51 rows**. Each count is verified
against main, not taken from the delivery message. Its seven owner requests
carry forward below. Its restart warning in the operator guide is revised in
this cut; its frozen copies under `tests/handover/testdata/history/follow-up-30/`
are byte-identical to main at the cutoff.

## Devstack 28 (#397) — coverage, no repair

Netlab 24's request 2 is closed as coverage.
`TestAT14_CtdenydExcludedFromMemberUsage` shows a nonzero `ctdenyd`
(80 packets / 80,000 bytes) surviving `enforce.ParseState` exactly. It changes
neither upload / download usage nor member presence across five fixture shapes,
with and without it (ten variants). No leak was found, no production code
changed and no stack was started. The brief's `e2e-test` gate would have started
one; that was the orchestrator's error, and the lane's gate substitution was
accepted (MTR01 / MTR05, §10.1 / §19 AT14).

## Isolation 16 (#399) — evaluator pass, qualification failure, F16-01

**D-098 first:** run 0b9e4510 is archived, **123 root files and subtrees**, in
the [follow-up 15 archive](../e2e/at/isolation/history/follow-up-15/archive.json).
`TestD098_Followup15ArchivePins` names the owner check; handover re-checks every
entry, including both gzip forms.

Run **83b457ef on #394's tree** took the D-140 sample at a **2.2 ms spacing**,
chosen before the run from 103 same-run calibration packets. The byte-identical
`EvaluateDeadline` passes: last positive **D−99.797 ms**, first drop
**D−97.671 ms**. The no-interleaving qualification **fails once**: packet 162
(latency 3.151 ms) left **3.609 µs** after packet 163's ingress, with both
positives at D−106.3 ms. The whole-node row stays **unqualified**, and nothing
was retried. Isolation 15's sample and isolation F14-01 stay failed.
Isolation's request 3, the next D-140 sample, is decided as **D-142** below.

**F16-01:** the shared D-104 precision fixture
(`tests/netlab/scripts/deadline.py`) failed before its packet window, on one
2 ms `expires` step. It was routed to netlab; netlab 26's disposition is below.

Populated stale-lease holds pass through devstack 27's TEST ONLY seam, with two
explicit incidents. **Not exercised:** pure `stale_generation` and the 30 s
fallback. **AT10 PASS 9 / 0 / 0 (ISO-F6-01 6 / 6)** and **AT12 PASS 10 / 0 / 0**;
AT11 / AT13 stay blocked on operator rows. **One install exceeded the 100 ms
guard (120.467 ms; rollback and recovery).** It is an observation; no fix is
routed. Run 83b457ef's root files are living until isolation 17 archives them;
they are cited by roadmap row / PR only (D-132). Scope: ISO01 / ISO02,
§8.1 / §8.2 / §16.3 / §19 AT10–AT13.

## Members 16 (#400) — run 0ffb355a

**D-098 first:** run e80bb592's **64 root files** are byte-identical under the
[follow-up 15 archive](../e2e/at/members/history/follow-up-15/SHA256SUMS);
`TestD098_Followup15Archive` names the owner check.

Run **0ffb355a on #396's tree: 2 pass (AT03, AT08) / 6 blocked / 2 fail**, with
**679 / 6 / 7 subchecks**. **AT02** fails the provisional D-135 / D-136 check.
AT01's F11-01 first selection **still misses the 5 s deadline**. **AT09 fails
four subchecks**, reviewed, with no predicate relaxed.

**Contention:** the orchestrator's chain gates overlapped the run from
**17:32:43 to 17:37:12 UTC on 11-01**. Every timing observation inside that
window is marked contention-confounded under D-133, with no cause assigned.

Populated holds alarm and escalate through devstack 27's seam. **The
survivor-hold row is not exercised:** the seam released its arm on
`generation_superseded` before an accepted survivor report arrived. A bounded
continuation is devstack 30's. The host powered off after the run and its
cleanup; the resume finished the joins and gates with no new run. This is
stated only as far as the #400 row records it. F11-01 / F14-01 stay unproven.
Run 0ffb355a's root files are living; they are cited by roadmap row / PR only
(D-132).

**The rewritten harness commit.** Before publication, the orchestrator removed
a home-directory path from one JSON line in the lane's branch history; the
final tree is identical. The measured harness commit **`3a8b6159`**, named in
79 evidence files, was published as **`5d9e5f04`**. Two other commits were
renamed with it: **`cbd59ccb` → `876c0558`** and **`7852fba6` → `f97da33d`**.
The old IDs exist only as text in members' evidence; they are not in the
published history. Wherever handover cites the measured harness, it states this
mapping: **measured harness `3a8b6159` = published `5d9e5f04`**. Members 17
adds the mapping to members' own run notes.

## Agent-core 36 (#401) — restart set equality and signal lifetime

The [dated hand-off](../node/followup36-restart-and-signals-2026-11-01.md)
is pinned only through agent-core 36's own sections.

**Restart.** Same-boot adoption (`adoptStoredServing`, `node/agent/restart.go`)
now refuses unless the retained installed model and the candidate have equal
`DeniedV4` (nil means the spec defaults), `ExtraDeniedV4` and `HostV4` sets,
compared as masked IPv4 prefixes. **Any growth, shrink or replacement takes
Boot's ordinary baseline, then the first full Apply.** Equal sets, reordered or
duplicated, still adopt. Agent-core 35's known-gap assertion is flipped. A busy
node lock and a real control timeout both keep the baseline, never the old
sets. The lab scenario restarts an established UDP flow on a changed set:
**0 / 56 replies during the baseline and 0 / 12 after Apply**. Proof names:
`TestAT13_F36_RestartSetChanges`, `TestAT13_F36_ChangedSetsControlTimeout`,
`TestAT13_F36_RetainedModelContainsSets`,
`TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth` and
`TestLabF36_AT13_RestartChangedSetsStopsEstablishedFlow`.

**Signals.** `signal.NotifyContext` is installed before open / Boot / Start,
and its stop runs after `Close` returns. Isolation 15's signal-15 receipt is
explained by the old window: both old windows reproduce signal-15 exits before
the fix, and both exit 0 after it (`TestAT13_F36_SignalLifetime`). The
durability claim, "the pending counter delta is recovered exactly once on the
same boot", **holds only when that boot adopts**, as the #401 row records
(`TestAT13_F36_InterruptedCloseDurability`). That limit is r20's H-1 below.

**Timing (D-133 attribution, judged under D-138).** Window 1 expired with no
admitted legs. Window 2 ran a clean A,B,A,B quartet: **four of four full suites
pass, and eight of eight entry / exit samples qualify**. Verdict: **no
attributable regression**. The earlier D087/D104 failures are attributed to host
contention; this is comparison attribution, not a controlled reproduction.
Every earlier failure stays recorded.

**Naming.** The lane's documents call the verifier's content check
"netlab 26". It is **netlab 27**: netlab 26 went to isolation's F16-01.
Scope: ISO01 / ISO02, MTR03 / MTR04, §8.1 / §8.2 / §10.5–§10.7 / §16.3 / §19 AT13.

## Review r20 — archived, merged with routed fixes

The [raw archive](../reviews/r20-adoption-signal-review.md) is byte-for-byte:
**32,207 bytes**, SHA-256
`af15fb7c42e86eb7269fd9e8bd1396a730cb5ba51643e52e7f6f1c34738a0f1a`. It carries no
host path, username or secret. agy `gemini-4-argon-high` reviewed agent-core
36's `7d34c885` before the merge and found **2 High, 3 Medium and 1 Low**. Each
finding has a failing proof test on the review branch
`r20-adoption-signal-review` (`639eda6b`). r20 recommended fixing before the
merge.

**Disposition: merged with routed fixes.** The orchestrator merged agent-core
36 anyway: it closes a security gap on main, and what r20 found that 36
introduced fails closed or is recorded, never silent. The #401 row records this.
**H-1, M-1, M-2, M-3 (pre-existing) and L-1 go to agent-core 37. H-2 goes to
netlab 27**, which has four concrete requirements from it. Agent-core 37
judges each finding before fixing it, so **none is recorded as confirmed**.

## Devstack 29 (#402) — the nightly browser

**The failure:** the nightly `e2e` `stack` job failed `portal-billing-return`
and `billing` on every scheduled run from 10-30 to 11-02. The cause is
**`browser_error`**: a fresh `ubuntu-24.04` runner has no Playwright headless
shell (the revision is pinned by the Portal lockfile), and the job never
installed one. This host passed only because the revision was in its cache. It
was reproduced with an empty browser cache and a fresh stack and Portal build.

**The fix:** an `e2e-browser` target (`make -f mk/e2e.mk e2e-browser`), which
`e2e-test` runs after `e2e-up` (no sudo; a no-op when cached). The job gains
Node 24 and a headless-shell install with its system dependencies.
**The proof:** before-fail **4 pass / 2 fail**; after-focus **6 / 0**; one full
`e2e-test` on another empty browser cache, **31 pass / 2 blocked / 0 fail**.
No assertion changed.

**Diagnosability:** the probe names its failing stage; records carry the
scrubbed cause (`TestSEC02_ProbeFailureDetailRedactsCookiesAndPaths`); runs write
`acceptance-summary.md`; the nightly uploads its `nightly` records with
`if: always()`, kept 7 days. The dated run evidence is
`docs/e2e/runs/2026-11-03/followup29/` (D-098), with
[the analysis](../e2e/runs/2026-11-03/followup29/analysis.md).

**Its requests:** the nightly signal, decided as **D-144** below; and confirming
the next scheduled run, which is the orchestrator's check and is not on main.
**No run result is claimed.** Scope: §9.2 / §9.3 / §18.4 / §19 AT03 / AT04, SEC02.

## Netlab 26 (#403) — the cutoff: isolation 16's F16-01

The per-follow-up [evidence tree](../poc/G2-followup26-evidence/overview.md)
is pinnable under D-098 like follow-up 25's: final manifest
[retention-pins.json](../poc/G2-followup26-evidence/retention-pins.json),
SHA-256 `118ab1c02d17ab0ea68d46ceb8d2c76263e0cfafd2c420709bcd98b4f1db8200`.

**F16-01:** isolation 16's assembled run failed the shared D-104 precision
fixture before its packet window. Line 35 of `tests/netlab/scripts/deadline.py`
required the eight remaining expiries to step by exactly 1 ms, and one listing
had a 2 ms step. **The mechanism, shown:** on the lab kernel, each set element
takes its own expiration base when it is allocated, before commit; and a dump
samples the clock per element, and nft sorts the reply by address. The evidence:
upstream v7.0 source, normal trial 758's raw age decrease, two padded controls,
and two mid-listing crossings in 2,000 dumps of one unchanged set.

**The disposition (TEST ONLY).** Kept: the exact 1001–1008 ms timeout read-back,
which is the 1 ms quantum proof, and the text read-back. Replaced: the
adjacent-expiry equality, by a check that each element is live and no greater
than its configured expiry. Unchanged: every D-104 and D-140 predicate, the
deadline window, guards, cadence, packet checks and production code.

**Proof.** The campaign ran once, with no retry: the old assertion rejected
**4 / 1,000** normal listings, and the replacement accepted **1,000 / 1,000**.
Before-fail / after-pass on isolation 16's eight pairs. All 19 historical
listings pass both; coarse 2 / 4 / 10 / 1000 ms listings fail both. The full lab
passes in **227.651 s** (`TestD104_AT13_DeadlineContinuity` unchanged).

**What stays failed:** F16-01's exact shape was not recaptured. Run 83b457ef
stays failed, and its individual cause stays unproven. F16-01 is recorded as
**"fixture disposition merged; the original run stays failed"**. Its one request
was advisory, about the orchestrator's host-admission check, and is closed on
the orchestrator's side. Scope: ISO01 / ISO02, §8.1 / §8.2 / §19 AT13.

## The nightly scheduled jobs

**Main records no scheduled-run result, so none is claimed.** On main, five
workflows define scheduled jobs: `e2e` (`stack`), `at-isolation` (`accept`),
`dns`, the NetBird CE adapter and G1 lab, and `netlab`. Only the `e2e` `stack`
job uploads records: its `nightly` records and `acceptance-summary.md`, with
`if: always()`, kept 7 days (devstack 29's change). The others leave their
GitHub Actions run logs only.

**D-144 is registered, but no expected-blocked list is on main at this
cutoff.** Devstack 30 (the e2e `stack` job) and isolation 17 (the at-isolation
`accept` job) apply it; neither had merged at task start. Until a list is
merged, a blocked row still fails the job. The
[operations index](../ops/README.md#reading-nightly-results-2026-11-03-est)
says where an operator reads results.

## Decisions and findings

No decision was added after D-141 on main before this cut. This cut registers
**D-142**, **D-143** and **D-144** in the
[decisions register](README.md#follow-up-30-decisions-register-2026-11-03-est).
**D-135, D-136, D-137, D-138, D-140, D-141, D-142 and D-144 stay provisional.
D-143 is Ted's decision.**

The [findings register](README.md#findings-register-at-the-follow-up-30-cutoff)
has **61 rows**: the previous 51, plus r20's six findings (routed, not yet
judged), isolation 16's unqualified D-140 sample, isolation F16-01, isolation
16's 120.467 ms install and members 16's run. Agent-core 35's same-boot adoption
gap is **closed for the restart path** by agent-core 36; the verifier's content
check for `denied_v4` / `host_v4` stays open, routed to netlab 27 (r20 H-2). The
live-drift reading is still unconfirmed, now routed to netlab 27. Isolation 15's
signal-15 receipt is explained and fixed by agent-core 36; the receipt stays
recorded as it was. Every other row of follow-up 29's register is unchanged.
Every historical failure stays failed.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation**.

2026-11-01 — **Owner decision 2: D-137 hardening option**; keep the failed
receive-only MTU-1280 limitation without upgrade or waiver.

2026-11-01 — **Owner decision 3: D-138 overturn option**.

2026-11-01 — **Owner decision 4: D-140 overturn option**; unchanged evaluator,
failed qualification and historical F14-01 retained.

2026-11-01 — **Owner decision 5: D-141 overturn option**, retaining its witness,
qualifying-share, promotion-cost and design-finding prerequisites.

2026-11-01 — **Owner decision 6: F11-01 repair direction and upstream report**,
including devstack 27's `::1` data and handshake-response evidence. Members 16's
5.222 s first SYN adds no cause; F11-01 / F14-01 stay unproven.

2026-11-01 — **Owner question 7: RELATED ICMP from spec-denied outer sources**
under §8.1 / §8.2 / §19 AT13. Keep the fail-closed source guard pending an owner
decision; no exception, defect disposition or fix is promised.

2026-11-03 — **Owner decision 8: D-142 overturn option**; isolation 15's and
16's samples stay failed qualification either way.

2026-11-03 — **Owner decision 9: D-144 overturn option**; acceptance status,
pull-request and local runs are unchanged either way.

Already routed, not new requests (2026-11-03): r20 H-1, M-1, M-2, M-3 and L-1 →
agent-core 37; r20 H-2 and the live-drift reading → netlab 27; the survivor-hold
continuation → devstack 30; the next D-140 sample → isolation 17 under D-142;
D-144's expected-blocked lists → devstack 30 and isolation 17. These items stop
with their owners. No outside-owned pin needed changing. **Nine owner
requests**, with no additional implementation request from this lane.
