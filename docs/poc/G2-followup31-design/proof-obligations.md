# C1 proof obligations and implementation test plan

These are tests for a later, separately authorized implementation. No new tests,
ordinary test changes or executable sketch are part of follow-up 31. The
[candidate specification](candidates.md) defines C1; the
[current ledger](current-guarantees.md) supplies G1–G10. Keep all existing tests
and failure expectations, including D-120's negative cases. A mock executor can
prove call order and arithmetic; it cannot establish nft commit or packet
semantics. Kernel claims require the privileged lab and platform evidence.

## Blocking obligations before implementation

**P0 / G5–G6:** the owner must decide the closure contract. Inject a kill after
install commit and before the budget check while the previous lease still has
time remaining. A drops every member; C1 permits eligible traffic until old
expiry or independent closure. This is an expected distinguishing observation,
not a regression test whose old assertion can silently be deleted. If the owner
requires A's result, C1 is rejected without performance measurement. The same
decision covers late-commit and rejected-install baseline-pending continuation
under G4/G8. Retain both outcomes and the decision with the future implementation.

**P1 / provenance and rollout:** agent-core supplies a trustworthy old verified
model, absolute deadlines, active bank, assignment and boot identity under the
shared lock. Fresh live checks must reject drift. Every reader and baseline
writer, watchdog, updater/installer and rollback release must understand the
new format before activation. Unknown format → member closure. This is not
solved by a renderer unit test or a new `verified=true` argument.

**P2 / kernel proof:** unchanged old elements must preserve their actual expiry
through an unrelated batch; a base-chain `return` must still traverse the next
base chain; failed token deletion must abort the whole release; a missing old
set must reject the whole delayed install. Confirm on the real supported
kernel/backend/nft versions, including arm64. There is no implementation
acceptance until these questions are measured.

**P3 / time and lock budget:** enumerate the existing caller/read/command timeouts
and failure paths for C1. No additional retry loop may hold the lock. Demonstrate
the declared bound on extra queued-revoke work in [C1](candidates.md) and test a
revoke arriving during every stage. If a required budget is missing, return to
design review; do not invent a new production duration or equate context
cancellation with kernel completion. A stopped lock holder remains expiry-bound.

## Unit and executor-integration assertions

Suggested test names are descriptive future names, not claims that tests exist.
Use deterministic synthetic model times and fake command barriers; avoid
scheduler-sensitive unit assertions. All scratch lives in disk `.agent/tmp`.

| Obligation / suggested test | Observable assertions and negative injections |
|---|---|
| G1 `TestAT29_HandoverDeadlineIntersection` | For each admitted identity and both directions, derive `E_old <= D_old <= D_new` before release and `E_new <= D_new` after release. Boundary cases: zero/missing anchor, expired lease/policy/node/binding, validity not begun, exactly 100/101 ms remaining, 1 ms rounding, timeout ceiling, overflow, negative elapsed and wall rollback. Missing or ambiguous comparison excludes the address; no zero-duration element becomes permanent. |
| G1/G6 `TestAT29_LateInstallNeverReleases` | At 100 ms exact success retains current inclusive predicate; at 100 ms+1 ns, negative sample, clock error, command error and canceled context, no release command occurs. Exercise an executor that commits then returns an error, and one that returns success after deadline. Record failed timing and rollback, never a verified result. |
| G2 `TestAT13_HandoverIdentityAndRevoke` | Removed/shortened/rebound IP, changed device/member/peer/revision/class and newly added IP cannot enter eligibility. Same address under a new identity is insufficient. Candidate allowset excludes revokes; upload and download admission precede every established/related accept and DNS exception. Conntrack failure does not reinstall old policy; retries preserve the new restriction. |
| G3 `TestAT13_HandoverNoVerifiedBase` | Boot/baseline/quarantine/unknown prior model, wrong boot, unknown bank, failed previous Apply and non-nil probe choose unconditional closure. Same-boot adoption never accepts a guard/token/incomplete hand-over or a merely well-formed generation marker. Initial migration remains guarded. |
| G4/G8 `TestAT13_HandoverFailureRollback` | Fail every preflight read, Render, tc command, install, release, precision read, nft/TC verifier, agent reverify and durable applied write. Assert rejection evidence, no ACK/published success, node-wide baseline attempt and no restoration of A. Fail baseline reads/batch/tc cleanup too; expose partial containment and original error, not a successful cleanup claim. |
| G5 `TestAT29_HandoverCrashCuts` | State-machine executor enumerates every cut in the matrix below. Show which bank is reachable, which deadlines constrain it, whether closure is proven and whether a successful ACK exists. Preserve the P0 distinguishing result explicitly. |
| G7 `TestISO01_HandoverVerifierExactLayout` | Tamper active-bank identity, old/eligible/token references, generation, timeout, flags, exact deny/host interval coverage, input drops, NAT, counter/class binding and rule order. Duplicate/unknown chains/sets, extra permits, absent/tainted set elements, offload, marks, foreign hook or tc redirect fail. A precision read checks the explicitly expected active bank. Unreferenced old bank cannot be selected as active by naming order. |
| G8 `TestAT13_HandoverStaleReleaseFence` | Successful release removes its own unique token once. Duplicate, older-generation and same-generation retry release fail as whole transactions if token is absent. Baseline deletes all hand-over state atomically. Delayed install referencing a deleted old bank cannot recreate it. An old failure must not erase a newer guard. |
| G9 `TestISO01_ISO02_HandoverRules` | Eligible IPv4 still passes candidate deny/host/resolver/source-binding and direction checks; IPv6, member-to-member, member-to-host, unknown source and wrong uplink remain denied. No established blanket exemption. `return` is not modeled as a final packet accept. Global rule/path/probe changes choose unconditional fallback before extra fast-path work. |
| G10 `TestMTR01_FUP02_HandoverContinuity` | Old bank is never flushed/updated/copied; existing counters and class IDs remain. Exactly one usage counter increment per admitted packet, none for guard drops, DNS separate. No changed tc tree/rate in C1; restrictive or changed QoS uses the current path. Retired counters are sampled before retirement; no reset hidden by bank rotation. |
| G4/G7 caller integration, agent-core | Preserve desired/applied separation, failed-candidate reason, verified conntrack-error special case, retry-bound withdrawal, full reverify-before-ACK, meter epoch handling and cancellation-independent cleanup. Fast-path selection must not turn an unverified error into slow-path success. |

The pure deadline/property tests should enumerate small sets of identities,
old/new deadlines and transaction outcomes; test the conjunction against a
separate predicate, not just compare generated text with itself. Include a
regression that the old-bank TTL is identical before/after an unrelated render,
then confirm its **remaining** lifetime decreases in the kernel. Assert exact
commands for release fencing and no runtime old-bank updates in any branch.

## Kernel crash/failure injection matrix

Use child processes **inside the owned lab**, controlled pipes/barriers and
kernel/read-back evidence. Keep separate records for SIGKILL, SIGSTOP/resume,
context cancellation and abrupt container/process restart. Kill both the agent
alone and agent plus watchdog; inspect whether an orphaned `nft` command can
still commit. Never stop or inject faults into host services. Test with a finite
old deadline and a later candidate deadline so accidental extension is visible.

| Injection cut | Required observation (conditional on P0 decision for intersection cuts) |
|---|---|
| Before eligibility; after every prior-state/TC read | No new timeout or candidate authority; old expiry keeps decreasing. Invalid provenance cannot select C1. |
| Before/after each tc command, including mid-batch error | No partial nft transaction; retain current partial-tc limitation and caller closure. Foreign objects are not deleted. |
| After render, before `nft` starts / before transaction commit | Old generation only. Release never occurs without a successful budget check. |
| After install commits, before command return | Guard+candidate+old bank present together. Revoked/ineligible packets absent at egress; eligible ones bounded by old expiry. |
| Install commit delayed beyond 100 ms and beyond old/new deadlines | No candidate-only forwarding; expired old gate permits nothing. Continuing actor reports failure and closes; dead actor cannot extend old lifetime. Do not infer a late commit from late process return alone: retain actual state and packet brackets. |
| After return, before/after budget branch | Same eligibility cap before release; rejected result has no release. Preserve all failed-Apply samples. |
| Before/during release, lost ACK and stale duplicate release | Exactly guard-present or safe candidate state. Later baseline and unique-token absence prevent delayed reopening. |
| After release, before/during nft, precision and tc verification | Current post-release failure exposure remains visible. Failed verification triggers baseline/quarantine; no successful ACK. |
| Before/during baseline transaction; before/after tc cleanup | Successful baseline cuts all member admissions atomically and invalidates all tokens/banks. Failed baseline remains explicitly unproven; no old-policy restore. |
| After verification, during conntrack cleanup, retained-model write, applied snapshot/heartbeat | Revoked flows remain denied; counters bind to the actual installed generation; partial durable state is not enough to adopt an interrupted transition. |
| Concurrent watchdog/renewal/new generation; lock-holder stop | Shared lock serialization, refreshed heartbeat recheck, stale-release fence and kernel expiry survive. A stopped lock holder does not gain a new watchdog deadline. |
| Same-boot restart, reboot, simulated clock loss, supported suspend/resume | Full live adoption only for completed known state; otherwise baseline/fresh authority. Do not manufacture clock trust or claim suspend acceptance from cross-compilation. |

For atomicity inject a syntactically/semantically invalid command at the end of
the install, release and baseline batches. The entire transaction must abort.
Test token absence *after* a batch has been queued but before it can commit,
using a lab-only barrier mechanism whose position is documented. A shell sleep
before launching `nft` proves launch delay, not an in-kernel commit stall.
If the supported tool/kernel cannot expose that cut, mark it unmeasured and
retain the source-level argument; do not silently mark the kill matrix complete.

## Netlab campaigns for the later follow-up

1. **Unchanged AT10 campaign:** generation 1000 plus 100 reloads through 1100,
   including A2 removals/re-additions and the existing topology event; same
   persistent TCP socket, 1.5 s read timeout, one outstanding UDP datagram,
   200 ms cadence, capture shutdown/successor protocol and D-120 analyzer.
   Inspect the actual C1 eligibility of A1 on every generation, not only the
   total number of successful reads. A1 must have zero observed node-ingress-
   without-egress losses in eligible windows for the hitless claim. A2's required
   revocation/new-admission closure is not an A1 continuity failure exemption.
2. **D-104/AT29 expiry:** run continuous sequenced TCP/UDP in both directions,
   with origin/node/responder headers and monotonic-to-capture anchors. Test all
   deadline minima independently, old deadline earlier than new, a shorter new
   deadline, and expiry in each install/release/verifier cut. Kill both userland
   enforcers. Assert no admission at/after each applicable deadline under the
   unchanged D-104 predicate; report all pre-D losses and [D−101 ms,D] behavior.
   Correlate queued pre-D packets separately from post-D admissions. Millisecond
   listings alone are not packet proof; no new timestamp tolerance.
3. **AT13 revocation and ISO01/ISO02:** established and new flows, upload and
   download, two independent members plus an unchanged sibling; revoke A2 while
   A1 continues. Include shorter deadlines, peer/address reuse, global denies,
   DNS/non-DNS resolver traffic, host/member/metadata targets, IPv6 and forged
   sources. Assert no revoked node egress after the candidate install cutoff,
   no later cutoff than paired A, and unchanged control traffic. Inject failed
   conntrack deletes and retry exhaustion. The original AT13 five-second
   observation and assertions remain; they are not replaced by the 60-second
   system target.
4. **Failure and recovery:** execute the cut matrix with stable counter reads,
   quarantine/foreign tc cases, failed baseline and later valid recovery. Measure
   full baseline→recovery outage, not only completed guards. New-generation
   recovery must reauthenticate authority; never jump back to old admission.
5. **Consumers and platforms:** real Engine renewal/revocation/restart/ACK,
   metering, updater and installer tests are cross-lane hand-offs. Cross-build
   linux/amd64 and linux/arm64 with CGO disabled, then run the kernel proofs on
   native amd64 and real arm64 with version/backend receipts. Missing hardware
   is an explicit acceptance gap, not an inferred pass.

Raw packet payloads never enter production telemetry or public evidence. Use
synthetic lab traffic, header-only correlation, decimal-string nanoseconds,
content-free counter deltas and hashed scripts. Retain all failures, capture
drops, missing evidence and negative controls. Performance acceptance is solely
the separately qualified [D-133 declaration](d133-comparison.md); passing this
matrix cannot explain historical B1's missing packet/timer trace.
