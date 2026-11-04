# Follow-up 42 — #439–#445; D-150 closes without a comparison

Verified against `main` **5a942aa1d6782b333983bbd230e97f44963e4d94**, the
final roadmap commit through **#445**, on **2026-11-03 EST**. Immediately
after follow-up 41's cutoff **#438 / c359d4a9**, the range contains exactly
**#439, #440, #441, #442, #443, #444 and #445**: seven merges and seven final
roadmap updates. No unbriefed PR merged in the range. Main and the public PR
listing were at #445 when checked at task start.

**Running at briefing, status only:** secops **15** (AT29 manual grant
withdrawal through `revokeMemberGrant`, six shared node writes, one live AT29
transport run); agent-core **48** (r28 H-1, r27 H-1/M-1/L-1, the R23 boot
heartbeat flake, and D-150's frozen-A code reading); and the orchestrator's
review **r29** (F30-01 design diagnosis). No delivered or merged PR for these
follow-ups was present at task start. Any later delivery is **after the
cutoff**; none of its results enters this cut. **r29 is not archived here.**

**D-098 / D-109 / D-132:** `TestHandover_Followup42Claims` pins the seven final
roadmap rows and PR excerpts, #440 frozen inputs, dated producer records,
immutable isolation history, the two raw review archives and Go test names on
main. No claim reads a living members, isolation, secops or usage evidence
root. Four reads in follow-up 41 move to frozen #440 bytes; all 35 earlier
claims functions retain their assertions. Review-branch proofs are explicitly
distinguished from declarations on main.

## Handover 41 (#440)

[PR #440](https://github.com/TED-H-com/Freedom-VPN/pull/440) records the
previous cut, **#435 to #438, up to c359d4a9**. The actual function and
manifests confirm **14 subtests**, **four final roadmap rows and PR excerpts**,
**ten frozen #435 inputs**, **101 immutable records**, **36 Go test
declarations**, and **seven frozen producer prose copies**. The register has
**55 decisions and 105 findings rows**: three updated, three added and 99
unchanged. The **34 earlier claims functions** remained exact after reversing
the four input redirects. These are #440's counts, not this cut's counts.

The session account is attributed to #440's body: codex **gpt-6-astra**,
launched **18:59 EST** on **2026-11-03**, delivered **19:37**, **6 commits and
44 owned files**, **+5,286/−292**. Its scope was documentation, claims,
fixtures, snapshots and digest values; no reservation.

## Agent-core 46 (#439) and 47 (#444)

[PR #439](https://github.com/TED-H-com/Freedom-VPN/pull/439) and its
[dated outcome](../node/followup46-outcome-2026-11-03.md) record D-150's
**third attempt**, with frozen **A 567080be / B 38420dad**. Admission receipts
are kept out of both Docker streams consumed by the frozen harness; ancestors
are excluded from the suite check, while their CPU remains foreign. The driver
launches standalone. Three dry rounds stopped on foreign activity: foreign
containers, a waiting shell's command text, and a merge-queue gate. B never
started. **No reservation, declaration or measured leg; no comparison.**
The ordinary main lab passed **13/13**, functional evidence only.

[PR #444](https://github.com/TED-H-com/Freedom-VPN/pull/444) and its
[dated outcome](../node/followup47-outcome-2026-11-03.md) record the
**fourth attempt** on the same frozen arms. The suite-program ruling is
implemented and tested: judge a suite by its **argv program**, not command-line
text. Actual make suites, go test and interpreters running gates.sh count;
an inline waiting shell does not, but its suite descendants do.

**One dry reservation:** A ran all **13 scenarios, 12 passed**.
`TestLabExternalMeterReset` found no recovery after **12.393 s**; main's
ordinary lab passes that scenario in **1.466 s**. Under the brief's
scenario-failure rule, this stopped the timing item. B, the second dry
reservation, the declaration and both measured windows did not run. The
reservation was released. Both arms have **zero pooled legs**; A's dry
statistics remain diagnostics only.

Main's ordinary lab passed **13/13 as a functional gate**. One full-module
R23 checkpoint-read/restart subtest failed on a **boot heartbeat deadline**
(`node/agent/followup40_test.go:228`), then passed a targeted recheck and a
full retry. The initial failure remains recorded, with no cause inferred.

**Neither attempt is a comparison, a B-only failure or an attributable shift.**
D-153 closes D-150; nothing is claimed about agent-core 37–43's timing and
no further A/B attempt is scheduled. Agent-core 48's frozen-A code reading
is for the record only.

## Devstack 33 (#441)

[PR #441](https://github.com/TED-H-com/Freedom-VPN/pull/441) and the
[dated record](../e2e/runs/2026-11-03/followup33/README.md) export
`Stack.WriteNode` to AT packages, with typed receipts and `NodeFieldsMatch`,
`NodePublished` and `NodeRevoked` read-back predicates. Only exact 412
`RESOURCE_VERSION_CONFLICT` triggers a re-read. Already-intended keeps the
actual 412 and invents no operation; otherwise fresh ETag, same key/body,
three writes and ten seconds apply. Operation success and observed state
remain separate from command acceptance.

`ReplaceFiniteGrant` restores what it revoked after a later failure: withdraw
any replacement with a lost create response, then recreate each original
plan/validity through the public API and verify operation success/live state.
The original error remains, joined with any recovery failure. Normal success
restores nothing. `PrepareNativeRouter1280` rolls back completed setup steps
in reverse order, with bounded per-step receipts. **Offline only; no live
run.** Consumer live proof belongs to their next authorized assembled runs.

## Isolation 20 (#442), F20-01 and r28

[PR #442](https://github.com/TED-H-com/Freedom-VPN/pull/442) and the
[dated hand-off](../e2e/at/isolation/followup20-2026-11-03.md) retain the
D-098 archive of **113 byte-identical follow-up 19 root records**. Six node
writes use the shared helper, with exact predicates, receipts and an AST audit.
Early-abort diagnostics report unavailable stages with the current run ID,
without exceptions or borrowed evidence. **F19-01 is answered by D-152**;
no rule ordering or acceptance requirement changed.

The one assembled run **c52bea6c**, measured source **037f4418**, records
**26 pass, 2 blocked, 1 fail**: AT10 **9/0/0**, AT11 **2/1/0**, AT12
**10/0/0**, AT13 **5/1/1** (pass/blocked/fail). F19-02 is answered on the
publisher side and exercised here: both shared publisher writes returned 202
on attempt one with serving read-back. Fourteen retained logical node writes
have fourteen attempts, no 412. An extra successful setup publication lost
its trace, so this is not a complete run-wide conflict audit. That and the
operation collector's borrowed run-ID defect were fixed offline after the
run; no missing live evidence is reconstructed.

**F20-01 remains failed:** AT13's short whole-node expiry installed a
`safety_check_failed` baseline **40.99 s before the lease deadline**.
The [raw r28 archive](../reviews/r28-expiry-probe-review.md) diagnoses it as
**H-1, a product defect**: the tunnel probe used the Step's control budget,
which the hung desired-state request had spent, so it failed without running.
Agent-core **48** owns the fix, running at the cutoff. The operator-visible
consequence and spec requirement are stated once in the
[operator guide](operator-guide.md#r28-h-1).

**r28 M-1:** `memberExpiry` recorded a pass in the same run while the node
had withdrawn all forwarding under H-1. The orchestrator's ruling preserves
that recorded pass unchanged; **it is not evidence for AT13 per-member expiry
under a serving node**. Isolation's next run after agent-core 48 decides it.
**F20-01's failure stays failed.**

## Wire-nodes 35 (#443), under D-154

[PR #443](https://github.com/TED-H-com/Freedom-VPN/pull/443) and the
[dated stop and resume](../nodes/followup35-zero-if-match-2026-11-03.md)
record the first session's correct stop: the brief assumed no resource could
hold version 0, but an explicit insert can. After the revised ruling,
`ParseIfMatch` no longer rejects zero before lookup. A row at version 0
matches **`If-Match: "0"`**, is written and advances normally; every other
row gets the ordinary **412 RESOURCE_VERSION_CONFLICT** with
**details.current_resource_version**. No schema change, positive-version
invariant or contract change. Missing/empty still gets 428; malformed tags
retain their existing parser responses.

`TestAPI01_HTTPZeroIfMatchConflictAndSameKeyRetry` covers capacity PATCH,
revoke POST, retire DELETE and publication PUT, proving no effects on
conflict and same-key/body retry. `TestAPI01_HTTPIfMatchZeroRowMatches`
proves zero→one and replay without another write.
`TestAPI01_StandaloneZeroIfMatchConflictAndSameKeyRetry` covers alternate
service entry points, including publication's validation-only parse; together
the tests cover all six callers. Parser tests also cover zero and malformed
values.

**Nonblocking hand-off:** the comment above `publisherTrace32` in
`tests/e2e/publisher_proof_test.go` still describes the old rejection.
Devstack owns its update; the two-shape proof stays compatible.

## Members 20 (#445), answering r27 M-2 and M-3

[PR #445](https://github.com/TED-H-com/Freedom-VPN/pull/445) and the
[dated note](../e2e/at/members/F20-r27-predicates.md) record r27's proofs
imported first. The two TODO proofs were adjusted only to exercise the
repaired predicates with their stated scenario facts: M-2 calls the repaired
predicate; M-3 already called `siblingRecovery` and explicitly marks its
verified Applies and written serving heartbeat. Both Held proofs remain
byte-identical.

**M-2 / AT09:** judge from the restart, under each observation's held lease.
When no grant-less lease follows, **D-067 is blocked / not exercised**, never
passed. Either legitimate renewal sequence is recorded. A new
no-foreign-filter subcheck fails on run 2's evidence until agent-core fixes
r27 H-1.

**M-3 / AT02:** apply D-135 and D-136 as confirmed, applied by the
orchestrator, with no new decision. Both rollback and quarantine link to the
failed Apply. Keep a deny-all Apply inside the bracket; close only on verified
sibling-serving recovery. Account for all sends, including interface-down
losses. Check 190 judges accounting and 191 recovery; no zero-loss criterion
or outage ceiling. Run 2's offline replay records **three recovered brackets
and 1,242 losses**. **It is a proof, not a verdict. Every historical run stays
failed.**

Five node writes use `WriteNode`, exact predicates and embedded receipts, with
an AST audit; expected-error tests stay unchanged. The permitted **grant
fallback is kept**. No assembled run here; the next run waits for agent-core
48's r27 fixes.

## Decisions, review archives and boundaries

The [decisions table](README.md#follow-up-42-decisions-register-2026-11-03-est)
goes from **55 to 57**. **D-153** (20:4x EST) closes D-150 with no comparison:
D-133 owes attribution for a failed live timing gate, and none exists. Attempts
44–46 had harness/admission failures; 47 had a frozen-arm scenario failure.
Ordinary main 13/13 is functional evidence only. **D-154** (19:4x EST) gives
zero the ordinary §14.5 version comparison, replacing the first ruling's
false no-zero premise. Both are **2026-11-03 orchestrator rulings,
provisional; the owner may overturn**. D-150 keeps its substance as history
and adds 47's argv-program ruling. **D-151 and D-152 are unchanged**; no
other decision existed after D-152 on main at this cutoff.

The [review index](../reviews/README.md) archives **r27 / dfde8d25**, **221
lines**, SHA-256 `3f4cc753495fe24edb1bc48a33ab8748231c14414d3056fcc7c33e4ca30abb10`,
and **r28 / 34a8d61f**, **231 lines**, SHA-256
`0af3db0ca803a0e37497618d9dca852df667d2e9a5a643f72a56676b07e593d3`, byte-for-byte.
r27's reviewer committed proofs in **62146ff1**, not its report; the
orchestrator committed the report in **dfde8d25**. r28's base was isolation
20's delivered **9be409f8**. These archives are D-098 inputs from now on.

r27 has **0 Critical, 1 High, 3 Medium, 1 Low**: H-1/M-1/L-1 go to
agent-core 48; M-2/M-3 are answered by #445; the owner's question is answered
by confirmed D-135/D-136 as applied by the orchestrator, no new decision.
r28 has **0 Critical, 1 High, 1 Medium**: H-1 goes to agent-core 48, M-1 to
isolation afterward; its optional NetBird management-loss note is recorded
with no action now.

The orchestrator reports verification on **main dd052c60**: r28's TODO proof
fails, three Held proofs pass. Changing the probe context alone makes the
TODO green but fails **TestR21_M2_OneBootBudgetAllStages/adoption_probe**.
Agent-core 48 must preserve both. Handover records this supplied verification;
it does not claim a new execution or a merged fix.

The [findings register](README.md#findings-register-at-the-follow-up-42-cutoff)
has **113 rows: seven updated, eight added, 98 unchanged**. F20-01 had no row
in #440, so it is entered alongside the seven requested review/flake/comment
rows. F30-01's row is byte-identical: known behaviour, no merged fix; r29 is
running. Every historical failure stays failed. The one new known operator
behaviour is [r28 H-1](operator-guide.md#r28-h-1); nothing else in this cut
changes what operators or members see. No date is promised for anything not
on main. No scheduled acceptance, physical arm64 or timing claim is added.

### Requests to orchestrator

**2026-11-03 EST:**

1. **Agent-core 48:** r28 H-1 (`node/agent/engine.go`, with the R21 boot-budget
   guard); r27 H-1/M-1/L-1 (`node/agent/serving.go`, `engine.go`, `sync.go`);
   the R23 flake (`node/agent/followup40_test.go`); frozen A meter-reset code
   reading for D-153's record only. Running; no further A/B scheduled.
2. **Isolation:** after agent-core 48, repair `memberExpiry` in
   `tests/e2e/at/isolation/driver.go.txt` and run the next assembled AT13
   check. Preserve the historical pass and F20-01 failure as recorded.
3. **Members:** next assembled run after agent-core 48; offline predicate
   proof does not replace historical verdicts.
4. **Devstack:** update the stale `publisherTrace32` comment in
   `tests/e2e/publisher_proof_test.go` for D-154 (nonblocking).
5. **AT owners:** secops 15's six writes and AT29 transport run are running;
   enrollment, usage and updates still own shared `WriteNode` adoption.
6. **Orchestrator:** r29's F30-01 design diagnosis remains running, outside
   this cut and unarchived. Carry the existing resource/platform gaps.

These are outside-owned hand-offs. No outside-owned pin needs changing;
no item is stopped. All remaining work is routed without importing its
post-cutoff results.
