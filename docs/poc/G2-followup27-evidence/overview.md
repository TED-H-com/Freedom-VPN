# Follow-up 27: verify live isolation set coverage

ISO01/ISO02, §8.1/§8.2 and AT13 (§19). The verifier now compares the complete
IPv4 coverage of denied_v4 and host_v4 with the supplied model on serving,
serving-probe and probe-baseline reads. It accepts equivalent merged prefixes
and ranges, and rejects extra, missing, malformed or expiring coverage with
`ISOLATION_POLICY_INVALID`. It requires the interval flag without timeout.
A baseline must have an empty, untainted allowed_v4. A bare baseline must have
neither denial set; its atomic transaction removes previously retained sets
after removing their references. No packet rule or rule ordering changes.

The [source inventory](sources.md) was written before the check. In particular,
HostV4 includes the renderer's implicit SNAT /32, and the agent's TEST ONLY
exemptions are respected. No normal input source changes between Apply and
its saved-model live read. External kernel mutation and restart reconciliation
can expose differences; neither is permission to adopt changed coverage.

## Before and after

[Original unit proof](before-unit.txt) and [real-kernel proof](before-kernel.txt)
ran with every enforcement production file byte-identical to task-start main
`b943941157ab6aeba6b04c79377de686c22bfbb4`. All six safety assertions failed:
adding, removing and replacing an element in each set incorrectly returned
successful verification. Positive controls passed first; only one set changed.
The retained before/after kernel listings are under `before-kernel/`.

[After-kernel results](after-kernel.txt) accept every rendered spec/overlap
control and reject all six mutations. Coverage controls run serving,
serving-probe and probe-baseline paths, including adjacent and overlapping
extra-denial prefixes and a host range formed with the implicit SNAT address.
Each form then transitions to a verified bare baseline. These are mechanism
proofs, not complete assembled AT13 acceptance or physical arm64 evidence.

The owned agent fixture uses the real Apply/VerifyServing/Boot code with the
existing rendered executor. A successful kernel Apply without the matching
retained_applied write leaves the original retained model and rolled-back
configuration equal while kernel contents differ. [Before](before-restart-state.txt),
Boot adopted both mismatches; [after](after-restart.txt), it takes the baseline
without a new full Apply. This represents the durable/kernel state at the crash
boundary; it does not claim a new physical crash experiment. The [periodic
live-read proof](after-periodic.txt) independently shows six mutations withdrawing
service and emptying member admission without a new Apply.

R20's `node/enforce/review_r20_test.go` was adopted verbatim in the first commit
and run before implementation. Its name and safety assertions remain. The
only follow-on changes replace existing fixture sets instead of appending
ambiguous duplicate definitions, remove the now-unused JSON import, and update
the stale TODO. The held test's body is unchanged. Shared owned enforcement
fixtures now include the static sets and empty baseline admission that the
renderer really installs. D063 probe lifetime assertions remain unchanged.
The agent executor converts static CIDR strings to nft JSON prefix/address
objects; it keeps its original decomposition, leaving coverage normalization
to the verifier. It does not call the production interval parser to build its
expected listing. Actual auto-merge output is covered by the kernel tests.

[Unchanged source checks](unchanged-contracts.json) pin the RELATED ICMP source
guard, resolver/probe parsers and checks, rule-order checks, Apply, live-read
caller, model, reload harness and agent-core 35/36 files. No new public model
field, config watcher, hot reload, dependency or tool pin was introduced.

## Measurement and reproduction

The [predeclared comparison](comparison-plan.json) fixes A1,B1,A2,B2 with one
attempt each and no retries. Each fresh topology runs 100 full live verifies
at each of two stated set sizes, then the unchanged 100-reload campaign.
All host admission samples, command attempts, raw measurements and failures
are retained. The 100 ms render-to-installed bound and the outer guard duration
are separate measurements. Host state is context, not a pass criterion.

Run commands through `python3 tests/netlab/scripts/disktmp.py` for unique disk
TMPDIR/GOTMPDIR directories. `run.py` records foreground commands; choose a
fresh receipt name/destination rather than overwrite an existing attempt.
The live scenarios are `TestISO01_F27_LiveSetDrift` and
`TestISO01_F27_RenderedCoverage` under `tests/netlab` with the usual netlab
prerequisites. `TestISO01_F27_VerifyCost` additionally requires
`FVPN_NETLAB_F27_COST=1`. The comparison uses pinned binaries and never swaps
production source in the shared worktree.

## Quartet result: B1 remains failed

All 800 dedicated VerifyServing calls succeeded. The small input uses the 16
spec prefixes and an implicit SNAT /32: the kernel lists 14 denied intervals
and one host element. The expanded TEST ONLY input uses 15 spec prefixes
(the documentation-range exemption), 128 additional disjoint /32s, and 128
host /32s plus SNAT: 141 denied intervals and 129 host elements. Each leg has
100 samples per size. The reload topology uses its unchanged TEST ONLY model:
15 denial prefixes plus one redundant extra /32, and three host /32s already
including SNAT. Exact input models and rendered transactions are retained for
each generation; the reload trace records successful reads, not their full
kernel listings.

| Leg | Spec verify p50 / p99 ms | Expanded verify p50 / p99 ms | Guard max ms | Render-to-installed max ms | Reload verdict |
|---|---:|---:|---:|---:|---|
| A1 | 22.979 / 26.424 | 26.267 / 29.376 | 70.554 | 28.789 | PASS, one accepted UDP boundary loss |
| B1 | 22.904 / 25.799 | 24.354 / 26.624 | 70.183 | 43.585 | FAIL, one TCP timeout; one UDP boundary loss |
| A2 | 23.329 / 28.192 | 24.097 / 26.829 | 69.126 | 48.772 | PASS, zero UDP losses |
| B2 | 23.571 / 25.974 | 25.025 / 30.996 | 68.458 | 34.615 | PASS, one accepted UDP boundary loss |

Every leg completed 101 verified Applies (initial plus 100 reloads); none
exceeded the 100 ms render-to-installed bound. Those bounds do not waive
D-120's TCP clause. [B1's failure record](b1-tcp-failure.json) preserves the
1.501408-second TCP read timeout at sequence 43. Its send timestamp falls in
generation 1064's 25.080203 ms outer guard bracket. There is no TCP packet
capture to establish the drop location or cause; the correlation does not
prove causation. No leg was retried or excluded. Later successful tests do
not change this failed verdict or establish comparison acceptance.

Under D-138, [the summary](comparison-summary.json) pools all legs by arm.
Pooled full-verify medians are A/B 23.147/23.239 ms (spec) and 25.070/24.831 ms
(expanded). Full-verifier differences reverse direction between A1/B1 and
A2/B2, so they do not support a repeatable causal total-latency shift. The
interval between nft JSON completion and tc-read start, less the intervening
precision-read duration, increases in both windows: median +54.578/+51.153
microseconds for spec and +71.093/+78.408 microseconds for expanded. The new
range parsing/comparison provides a code-level explanation for added local
work, but this interval includes both nft parses, other checks and bookkeeping;
it is not an isolated CPU cost. Pooled medians for that interval are A/B
2.051/2.103 ms and 2.383/2.455 ms. One quartet does not establish a broader
performance distribution or explain the B1 TCP failure.

A1 setup waited 1381.145 seconds for a real other-lane isolation run to finish.
Quiet admission then took 30.084, 90.172, 30.086 and 150.269 seconds for
A1/B1/A2/B2. Every admitted six-sample vmstat window met r <= 3 and id >= 80;
the since-boot first row was ignored. Load averages, raw process observations,
rejected windows and post-leg samples are retained. No leg reached the
60-minute quiet deadline. Admission checked actual processes, excluding shell
command-text false matches; the host observations are not test pass criteria.

## Handoff

**2026-11-03 — orchestrator:** the B1-only TCP failure leaves the production
comparison unaccepted. Review its preserved trace and determine the next
controlled diagnostic window; this follow-up's no-retry quartet is finished.
No evidence supports assigning the cause to host contention or to the verifier.
The functional fix and mandatory gates are delivered independently of that
unresolved verdict. Historical evidence and historical failed verdicts remain
unchanged. RELATED ICMP remains the owner's open question, with its source
guard unchanged. Physical arm64 remains unmeasured.
