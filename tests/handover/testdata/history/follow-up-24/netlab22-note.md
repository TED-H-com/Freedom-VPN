# Follow-up 22 — executor lifecycle and D-135, 2026-10-31

TEST ONLY diagnostic evidence under D-133. D-134's 100 ms reserve, 1 ms
rounding, [D−101 ms,D] expiry window and measured atomic guard remain unchanged.
There is no new ceiling, tolerance, recovery SLA, or availability exemption.
D-135 remains provisional pending the owner's confirmation. This assessment
supports that decision; it does not implement a partial-restoration design.

## D-098 first

Commit `8a13b606` archives 132 follow-up 21 files, including its note, under
[G2-followup21-evidence/archive](G2-followup21-evidence/archive/archive.json).
Each file was compared byte-for-byte with source commit
`8f74bc8b884e3a9a69924f791c7938e3dea09908` before copying; lengths and SHA-256
are pinned. Original files/pins and every older archive remain unchanged.
The new [consulted inputs](G2-followup22-evidence/inputs/archive.json) freeze
agent-core 29's hand-off and the three latest AT roadmap rows independently.

## Executor evidence and observability

`WithCommandLifecycleForTest(ctx)` opts an Apply into an additive `commands`
array in the existing `ServingEvidence.TimingEvidence()` snapshot. The ordinary
agent producer already serializes that snapshot without a schema or sequence
change. Its production caller still needs the explicit opt-in under its existing
`test_mode && test_timing_path` gate; that caller is outside this lane.
The netlab trace opts in locally. No diagnostic work runs in the disabled
executor path: it calls the original `cmd.Run`, with no new samples, proc reads,
resource collection, serialization, cancellation watcher, or allocations.
The additional disabled checks are context lookups/branches, not literally
zero CPU instructions. Existing timing rows and D-106 digest inputs are intact.

Every real command retains a fixed kind and phase (prepare, shape, install,
release, verify, revoke, rollback or quarantine), context state before/after
and at cancellation, start attempt/return, successful child-start upper bound,
cancellation-callback entry, Wait return, start/command status, exit/signal,
child user/system CPU and child voluntary/involuntary context switches.
Context enum values are `none`, `canceled`, `deadline_exceeded`, `other`;
`unavailable` is used for an unobserved cancellation callback.
Counters and nanoseconds are decimal strings; missing data has explicit
availability flags. A failed Start has no invented child, Wait, exit or CPU.
The original CommandContext kill callback, errors, returned bytes and deadlines
are preserved. No argv, command output, selectors or cgroup paths enter these
records. The detached snapshot copies nested cgroup arrays too.

Per-command before/after resource snapshots have CLOCK_MONOTONIC read brackets,
`cpu.max`, `cpu.weight`, `cpu.stat`, `cpu.pressure` for the resolved current
cgroup v2 and its visible ancestors (depth only, at most 64). No unknown field
means unlimited entitlement or zero pressure. No cgroup/affinity/priority
setting is written. Measurements themselves add overhead when enabled; the
A/B comparison must retain that cost.

Fields that cannot be exactly observed here are **not approximated as events**:

- Go exposes its deadline as `time.Time`; conversion to CLOCK_MONOTONIC is an
  enclosing interval, not a claimed exact deadline nanosecond. A wall-only
  deadline is explicitly unavailable: the public value cannot reconstruct the
  monotonic timer target after a wall-clock correction.
- `exec.Start` gives no child-creation/first-scheduled timestamp. The attempt
  and successful return bracket creation; `child_start_exact.available=false`.
- Cancellation is observed at exec's callback, not the instant the deadline
  fires. A canceled-before-Start command has no callback observation.
- Rusage switch counts belong to the exited child. They do not reconstruct
  runnable-delay intervals or scheduler switches/timestamps for parent threads.
- A cgroup namespace may hide host ancestors. `host_ancestors_available=false`
  even when every visible ancestor was read; v1/unsupported mappings are unknown.
- An executor starts after rendering. Its resource samples do not claim to be
  instantaneous render-time host samples, kernel insertion times, or historical
  measurements. Exact scheduler causality requires another instrument.

The TEST ONLY trace additionally retains the exact input Model, the final
render Model plus otherwise-private timeout reserve, and actual NFT transaction
strings. Hashing and JSON output happen after Apply, outside every guard.
These fixture-only selectors are separate from TimingEvidence and D-106.

## D-135 enforcement statement (assessment, no change)

[`Apply:68`](../../node/enforce/apply.go#L68) shapes first, renders current deadlines,
installs the candidate plus [`guardedNFT`](../../node/enforce/deadline_install.go)
in one nft transaction, checks the original 100 ms render→installed budget,
then releases and verifies. Install error/overrun never runs release. A normal
post-release verification error calls `installBaseline`; foreign tc or retained
quarantine takes the stronger `Quarantine` path. The special verified
conntrack-delete failure keeps the verified serving generation and is not
baseline rollback.

[`installBaseline:301`](../../node/enforce/apply.go#L301) deliberately passes only Datapath to
[`installBaselineModel:305`](../../node/enforce/apply.go#L305).
There is no previous Model or previous-generation restoration. It refreshes
live state, calls [`RenderBaseline:260`](../../node/enforce/render_nft.go#L260), and
installs one atomic NFT batch: generation 0, empty `allowed_v4`, member forward
drops in both directions, preserved counters, and control input/output still
available. The ordinary rollback has no synthetic probe exception. Owned HTB
roots are removed only when live ownership reads permit; foreign tc is not
erased. This implements §8.2 / §10.3 / §11.5 / §16.3 conservatively.

The guarantee is conditional on successful atomic installation: no partially
installed nft generation, no old revoked authorization restored, no reported
successful Apply for a failed candidate. It does **not** claim an atomic
cross-subsystem nft+tc transaction or infallible cleanup. Baseline installation
errors are ignored by Apply while the original failure is returned. If a
candidate committed behind its guard and baseline fails, that guard stays
closed; a post-release verifier failure plus failed baseline needs the caller's
independent withdrawal/quarantine. A pre-install failure is not evidence that
baseline committed. The agent's subsequent cleanup and readbacks remain
necessary; an error alone is never proof of kernel state.

A per-member restoration would require fresh valid signed authorization for
all restored members and a monotonic intersection with current revocations,
lease/policy expiry and identity/publication state. It would also need proven
node-wide isolation, unambiguous source/member/class ownership, correct meter
continuity, verified QoS and current live member-path checks. Shared-path,
unknown-ownership or global-policy failures cannot be assumed member-local.
Its atomically committed generation/readback/ACK semantics would have to
represent that subset truthfully, with crash/late-commit and repeated-retry
proofs and measured sibling losses. Replaying the previous rules cannot meet
those obligations merely because the sibling was once admitted. This lists
requirements for a separate D-133 design, not a proposed implementation.

## Measured recovery brackets and the AT consumers

Follow-up 21 deliberately failed one verifier read after guard release and
held baseline for **200 ms** before retry. Its three baseline-start→successful
recovery-release-end outer brackets were **304.815081 / 280.395264 /
285.296508 ms**; definitely closed intervals were **272.051181 / 253.222256 /
257.972529 ms**. The failed Applies' completed guards were only **34.435027 /
28.329661 / 24.020724 ms**. All 173 losses remain counted (150 outside completed
guards). These are measured fault/hold/retry windows, not a production maximum.
The exact original brackets remain in the D-098 archive.

[Agent-core 29 S2](G2-followup22-evidence/inputs/followup29-install-contention-2026-10-31.md)
is a different withdrawal: Apply 335 **and** the following live verifier
succeeded, then `member_path_unconfigured` caused baseline commands
[64186458578812,64186551257218] and [64186561126808,64186634462698] ns.
The admitted packet arrived at 64186799139348 ns without egress through
64187304013373 ns. No recovery endpoint was retained for that packet; this is
not a recovered 100 ms install overrun, and the missing underlying path-read
error cannot be called a timeout. None of the interventions through 384 workers
reproduced the historical install overrun. Historical cause stays **unproven**.

The [latest roadmap rows](G2-followup22-evidence/inputs/roadmap-rows.txt) add:

- Members 12: all 89 AT02 Applies succeeded, so F11-03 is **blocked: no failed
  Apply observed**. The 3,976 attempts/21 losses retain 13 inside completed
  guards and 8 outside. Those eight lie in a new 236.970 ms successful
  `policy_missing` baseline→successful Apply bracket containing nine losses.
  Its `cause_status=unproven` (F12-01); it is not install rollback or evidence for a
  new availability guarantee.
- Isolation 11: 952 Applies, 945 completed guards, zero overruns. Seven failed
  Applies are intended foreign-tc rejections **before install** and retain
  their timing/host/status. Historical 49 failed installs have host and verifier
  observations **unavailable**, `cause_status=unproven`, and all 279 historical
  renewal losses counted. A current `not_applicable` cause status on an Apply
  with no overrun does not resolve a historical cause. Current 18/1,396 renewal
  losses all overlap completed guards, without waiving them.
- Updates 12: regular AT23–AT27 5/5 and installer 27/27 pass, including physical
  software rollback. This does not measure or change enforcement's sibling
  restoration or its maximum outage. No new enforcement failure bracket is
  supplied by that roadmap row.

A diagnostic label such as “fail-closed install overrun with recorded host
contention” requires both joined observations and keeps `cause_status=unproven`
unless separate causal evidence exists. Missing samples stay unavailable;
`not_applicable` is not a historical exoneration. D-135 acceptance counts and
brackets every sibling outage; it never converts baseline into a completed
guard or a failed admitted-source control into AT02 success.

## Validation and hand-offs

Runtime comparison, full distributions and gates follow below. A complete
quiet-host comparison is required for acceptance; a retained suite failure is
not converted to a pass by later attempts.
The agent opt-in belongs in `node/agent/engine.go` under its existing measurement
gate; production defaults must stay off. No native F8-02 work belongs here.

## Comparison protocol and retained harness correction

A is the frozen, unchanged `main` archive at
`1b25757bf4651be80f48b78150eab988832c7d2a`; its complete runtime/module/lab
inputs are byte-identical to task-start main `8f74bc8b` (see
[source equivalence](G2-followup22-evidence/source-equivalence.json)).
The live main ref advanced during this task; the actual archive/binary receipt
is authoritative, and the initial live-ref metadata is retained with a correction.
B's final runtime source is `c6994eff`. The earlier `77b489f2` candidate
is retained separately and excluded from the final comparison: review found
that wall-only context deadlines cannot expose their timer's monotonic target.
A focused regression failed before the fix; that case is now unavailable.
The complete quartet restarts on the corrected observer, with unchanged
enforcement behavior, bound and reserve. Both [source captures](G2-followup22-evidence/sources)
are byte-pinned. The common Python collector runs Apply/hold with unchanged
arguments/input and saves the synthetic Model and result **after** child exit.
Both sources use GOMAXPROCS=4, GOFLAGS=-p=4, CGO_ENABLED=0 and GOTOOLCHAIN=local.
No deadline, QoS window, predicate or release budget is adjusted.

The first A-only attempt is rejected as a comparison: the original collector
renamed the executable to `fvpn-netlab-real`, breaking the existing
`pidof fvpn-netlab` kill check after the 101-reload timing check passed.
The full live suite finished with that failure. It was stopped before the
next suite; cleanup removed all owned Docker objects. That attempt's live
artifacts and 101-guard trace are retained, but its post-exit collector files
were not exported before teardown, so it is **not** a full-suite distribution.
No A/B conclusion is drawn from it. The corrected collector preserves the
executable basename in a separate directory and includes the hold result.
The complete A/B/A/B restarts with the same corrected collector in every leg.

Each leg runs the ordinary live-netlab suite and the ordinary agent suite.
A1 and B1 additionally run the D-116 trace-agent fixture. The ordinary suites
preserve the existing configuration; the trace-agent run exercises the new opt-in through the real
Engine and its existing producer. Ordinary agent measurements remain disabled
at the command layer until the agent owner adds its context opt-in. Results
for these populations must remain separate rather than hide that distinction.

Before startup and again before traffic, qualification is `vmstat 5 7`: all
six **interval** samples have r≤3 and id≥80, and the exact pgrep suite filter is
empty before/after sampling. The since-boot row is retained separately because
it is not a current interval. Load averages, D-state thread counts and monotonic
sample bounds are retained; load average is not a qualification criterion.
Postflight uses the same check. Rejected preflights are retained and retried
on a 60-second cadence within the 90-minute window. No third-party traffic or
host network mutation is involved.

The preserved host receipts retain raw pgrep output privately. Public receipts
retain exact matched counts and empty successful checks; nonempty command text
is withheld. Display-only whitespace in text logs and extra EOF newlines are normalized, with
private originals pinned in the normalization ledger. Source/Model/transaction bytes and native producer timing rows are
not changed by that normalization. SHA-256 on the fixture's exported Model and
transaction denotes the exact exported bytes, not a new signed-document or
D-106 identity contract. An absent `commands` array is unobserved, not evidence
that no child commands ran (ordinary callers and fake executors may not emit it).

The fresh rollback fixture names the independent denied source `revoked`; it
is absent from every captured candidate, so this fixture measures continued
denial through rollback/recovery, not a new revocation transition. The ordinary
live suite retains its existing revocation-transition tests separately.

An excluded final-source B1 live attempt failed unchanged D-120: the last
retained datagram, 65, was a boundary loss in generation 1093, with outer
27.772695 ms and render→installed 17.938669 ms. There was no following control
datagram, so predicate (iii) failed. Its postflight also failed host qualification
(minimum idle 77%); both failures and the full packet/Apply evidence remain
under `runtime/unqualified/B1-live-1`. The repeat is due to host qualification,
not a retry-until-pass rule. This does not establish an install overrun or a
cause attributable to the observer, and the missing control is not waived.

After repeated postflight rejections, the driver was paused outside a running
suite to prioritize the required work within the same 90-minute window. The
fresh two-source fixture runs within B1, before A2/B2; redundant optional
trace-agent repeats on A2/B2 are omitted. The required live/agent A,B,A,B order,
frozen binaries, qualification, all completed runs and failures are unchanged.

The first fresh rollback attempt used UDP source port 40022, while the existing
header decoder recognizes the synthetic sequence only at port 40021. Raw
captures show traffic, but the summary's sequence-based counts were all zero
and its assertion correctly failed. That attempt is retained with its original
raw captures and failed host postflight. The fixture was corrected to use the
existing 40021 format; no decoder, predicate, enforcement source or frozen A/B
binary changed. The final fixture source and before/after SHA-256 receipt are
retained separately from the compiled product source.

## Fresh D-135 rollback capture

The corrected B1 fixture passes with qualified pre/post samples. Three known
post-release verifier read failures each install the actual generation-0
baseline, hold for 200 ms, and recover on the next successful Apply. The
baseline-start→recovery-release-end outer brackets are **312.900556 /
323.540915 / 319.004140 ms**; definitely closed intervals are **288.116630 /
294.025595 / 291.445125 ms**. Each baseline/recovery outer bracket contains
59 admitted losses. Across the complete capture: **425 admitted ingresses,
240 egresses, 185 losses** (18 overlapping completed guards, **167 outside**);
the denied source has **424 ingresses and zero egresses**. None is waived.
All four captures decode completely with zero kernel drops; successful
admitted forwarding is observed after each recovery. These are measurements
of an injected fault plus an explicit hold, not a recovery SLA or a recreation
of the historical overrun.

The exact synthetic input/render Models, their SHA-256 pins, every rendered
NFT transaction and hash, injection observation bounds, both source headers,
node/origin headers and raw-pcap receipts are retained. All seven Applies and
all three failed-Apply brackets appear in the diagnostic analysis.

The B1 trace-agent fixture joins **23** exact TimingEvidence snapshots to the
existing native producer by monotonic Apply start, preserving run/boot IDs and
sequence. Its final canceled Apply distinguishes **one started child killed by
signal 9** (observed cancellation and CPU/Wait data) from **eight canceled
start attempts** (no invented child, CPU, exit or Wait). This observation does
not diagnose the historical install overrun.

## Final comparison status and distributions

**Blocked: host not quiet — the full B2 agent suite did not run before the
90-minute qualification window expired. The Apply-path product change is
not accepted until the required comparison completes in a resume.**

The live A,B,A,B quartet passes with qualified pre/post samples. The ordinary
agent comparison has A1, B1 and A2 only: B1/A2 pass; unchanged-main A1 fails
`TestLab04_RestartKeepsVerifiedLease` at `scenario_test.go:140` ("restart lost
the valid stored lease"). It remains red, with its original stream and log;
later passes do not establish its cause. No assertion was edited.
The original final-source window is [91425922603476,96825922603476] ns;
restarts did not extend it. The driver stops at 5404.460 seconds after finishing
the already admitted B2 live suite and its postflight. This does not claim
that the host was continuously busy: earlier suites found quiet intervals,
but rejected qualifications and repeats consumed the bounded window.

[Host summary](G2-followup22-evidence/host-summary.json) retains each selected
preflight/ready/postflight load average, six-interval vmstat summary, D-state
thread counts and monotonic bounds; complete accepted and rejected receipts
are under `runtime/`. Load average is never a criterion. These are bracketed
host samples, not a reconstructed scheduler history.

Nearest-rank percentiles below use **every completed guard** in each named
population. Full vectors and every failed Apply (including excluded attempts)
are under [analysis](G2-followup22-evidence/analysis), with a compact
[failed-Apply index](G2-followup22-evidence/failed-apply-index.json). A failure
after release can have a completed guard; a pre-install rejection cannot.
The populations stay separate. The missing B2-agent population is unavailable.

| Population | Applies / guards / failed | Outer p50 / p90 / p99 / max (ms) | Install p50 / p90 / p99 / max (ms) | Render→installed p50 / p90 / p99 / max (ms) |
|---|---:|---|---|---|
| A1-live | 138 / 137 / 2 | 30.218767 / 40.484455 / 49.285496 / 52.533127 | 14.962917 / 23.011785 / 29.310054 / 29.634719 | 15.041772 / 23.133039 / 29.435089 / 29.766487 |
| A1-agent | 138 / 135 / 3 | 29.651227 / 40.919860 / 50.623540 / 50.939721 | 14.806233 / 22.007141 / 29.981965 / 30.841340 | 14.887575 / 22.171828 / 30.098342 / 30.909662 |
| A1-agent-trace | 23 / 23 / 0 | 32.979477 / 39.498207 / 45.955402 / 45.955402 | 15.576078 / 21.648830 / 23.034500 / 23.034500 | 15.665614 / 21.706932 / 23.143003 / 23.143003 |
| B1-live | 138 / 137 / 2 | 30.293684 / 40.866305 / 50.882349 / 52.860299 | 16.145908 / 22.541100 / 26.909355 / 28.090277 | 16.229710 / 22.642165 / 26.999323 / 28.182530 |
| B1-agent | 138 / 135 / 3 | 30.077815 / 40.732984 / 46.061634 / 46.214033 | 14.557328 / 21.596318 / 26.373941 / 27.350680 | 14.643810 / 21.675187 / 26.611570 / 27.436425 |
| B1-agent-trace | 23 / 23 / 1 | 34.292599 / 44.810593 / 54.110261 / 54.110261 | 15.985031 / 27.201435 / 29.934308 / 29.934308 | 16.055255 / 27.297278 / 30.039110 / 30.039110 |
| A2-live | 138 / 137 / 2 | 27.563664 / 37.740059 / 44.365668 / 45.983480 | 14.132060 / 21.596473 / 26.995971 / 28.893651 | 14.232089 / 21.736268 / 27.065263 / 29.024620 |
| A2-agent | 138 / 135 / 3 | 30.030849 / 41.108341 / 46.615981 / 48.194191 | 15.307332 / 22.609308 / 27.925440 / 28.968468 | 15.394795 / 22.687553 / 28.201652 / 29.027697 |
| B2-live | 138 / 137 / 2 | 31.189132 / 40.956407 / 50.161712 / 56.630295 | 16.747348 / 23.494631 / 28.269935 / 29.835215 | 16.850169 / 23.578644 / 28.359817 / 29.930400 |
| B1-diagnostic | 7 / 7 / 3 | 26.049546 / 34.692832 / 34.692832 / 34.692832 | 13.334974 / 18.753833 / 18.753833 / 18.753833 | 13.411820 / 18.837484 / 18.837484 / 18.837484 |

No selected run has an install overrun. These observations do not identify the
historical cause or establish a ceiling. The full live-suite population has
137 completed guards per leg; command measurement is enabled for 102 of B's
137 guards through the existing trace paths. Ordinary agent command
measurement remains off; its existing timing/host producer supplies all Apply
rows. The additional B1 trace-agent and rollback fixture exercise the opt-in.

The final command audit verifies **932 byte hashes / 232 traced Applies /
12,717 command records**, including 23 exact native-producer joins. Child
CPU/switch observations are available for all 12,709 started children; the
eight failed starts remain unavailable. All command cgroup lookups resolve
the visible v2 hierarchy; hidden host ancestors and exact child-start remain
unavailable. All owned containers, images, networks and volumes are absent
in the final cleanup receipt.

Remaining hand-offs: agent-core must enable the context only under its existing
measurement gate in `node/agent/engine.go`, investigate the unchanged-main
restart assertion with the agent-lab owner, and coordinate a quiet-host resume
for the incomplete comparison. F8-02 native work remains outside this lane.

## Final gates

Lane lint/unit/build (including linux/amd64 and linux/arm64) pass after the
fixture correction. Unchanged agent/updater consumers, committed-evidence
hygiene, handover, root `make go-lint`, root vet, tracked gofmt and diff checks
all exit 0 in the required foreground order. The actual output is retained
under `gates/`, with commands/exits in [validation results](G2-followup22-evidence/validation-results.json).
An optional race invocation was unavailable with required CGO_ENABLED=0; no
race-pass is claimed. All 132 archived follow-up 21 files, four consulted
inputs, and 98 captured source files match their recorded git source bytes
(the three roadmap rows match the exact ordered extraction). Older evidence
and its pins are unchanged. The fixture's only later source change is the
recorded synthetic UDP port correction.

These successful gates do not erase the qualified A1 agent-lab failure or
complete the missing B2 agent leg. The final status remains unaccepted pending
the quiet-host resume and the reported owner hand-offs.

## Resume (2026-11-01): fresh agent comparison

This section records a new qualification window and new evidence. Everything
above, including the first window's unchanged-main A1 failure, rejected host
samples, excluded attempts and incomplete agent quartet, remains unchanged.
The accepted live quartet and rollback capture were not repeated. Requests 1
and 2 belong to agent-core follow-up 31; no caller or assertion was changed.

The fixed window is **2026-11-01 06:22:00–07:52:00 UTC**, corresponding to
01:22–02:52 EST. Setup and retries did not extend it. The fresh ignored runtime
directory uses the retained driver with only the recorded resume changes:
new paths, frozen extraction for both arms, new leg names and the agent-only
loop. See the [driver delta](G2-followup22-resume-evidence/runtime/driver-delta.json),
[window receipt](G2-followup22-resume-evidence/runtime/window.json), and
[predeclared comparison rule](G2-followup22-resume-evidence/runtime/protocol.json).

The immutable sources are A `1b25757bf4651be80f48b78150eab988832c7d2a` and
B `c6994eff40d9a98fffcf36b53dd3356f0360a31d`. Every extracted regular file
matches its pinned Git object after the builds: 7,607 A files and 7,586 B
files. Agent/runtime/module/lab inputs common to both arms are unchanged.
The [source pins](G2-followup22-resume-evidence/source-pins.json) refer to the
unchanged first-window source captures, and the new source receipts pin all
eight built executables. The retained driver's build from nested archive
directories embeds the enclosing worktree's common `c67977b7` VCS metadata
in both arms; it does not describe the executable's extracted source tree.
The full per-file Git-object audit, binary hashes and
[build-info receipt](G2-followup22-resume-evidence/runtime/build-info.json)
make that distinction explicit.

Qualification is unchanged: exact competing-suite filter, empty before/after
process checks, six current `vmstat 5 7` intervals with r≤3 and idle≥80,
preflight, after-startup check, before-traffic qualification and postflight.
The since-boot row is retained separately. Load average and D-state thread
counts are recorded, never used as qualification criteria. Rejected
qualification samples and all attempts rejected at postflight are retained.
The [host summary](G2-followup22-resume-evidence/host-summary.json) links every
selected receipt and preserves its monotonic bounds, load averages and
D-state counts; full receipts are under `runtime/`.

### D-133 verdict

**Blocked: host not quiet. The Apply-path change remains NOT ACCEPTED.**
The fresh ordinary-agent/trace-agent comparison is incomplete: R-A1, R-B1
and R-A2 have qualified results for both populations; neither R-B2 population
has a qualified completed result. R-B2's two full-suite attempts both passed
their tests but failed postflight (r=5 / idle=77, then r=2 / idle=79). A third
attempt's preflight started before the cutoff and finished after it; startup
completed, but the ready gate refused admission. No traffic or suite ran on
that third attempt. The driver stopped at 5,424.187 seconds from the fixed
window start. Its [blocked receipt](G2-followup22-resume-evidence/runtime/blocked.json)
and full [driver log](G2-followup22-resume-evidence/gates/runtime-driver.log)
retain that boundary; no restart extended the window.

The selected unchanged-main **R-A2-agent** fails
`TestLab04_RestartKeepsVerifiedLease`, `scenario_test.go:140`, “restart lost
the valid stored lease.” Its postflight qualifies (r≤1, idle≥85), so this is
retained as the A2 result, without a retry to obtain a pass. R-A1, R-B1 and
all three selected trace runs pass. No observed B-only test failure occurs.
The A2 failure is additional evidence for existing request 2 / agent-core
follow-up 31; it does not count against B. The first-window A1 failure stays
unchanged. Both failures' causes remain unproven.

The comparison rule was recorded before measurement: each B p50/p90/p99/max
must lie inside the inclusive interval between the corresponding A1 and A2
statistics, separately for ordinary-agent and trace-agent populations, with
no tolerance. Both A controls apply to B2 as well; no extra A3 was added.
Of the **24 available R-B1 comparisons**, **13 are inside**, **9 above** and
**2 below** their A interval. For example, ordinary-agent outer p90 is
41.546862 ms against A's 39.459996–39.544397 ms; trace-agent install p90 is
23.702304 ms against 19.009878–19.077435 ms. The full component comparisons
are in [comparison.json](G2-followup22-resume-evidence/comparison.json).
These incomplete measurements do not establish that the instrumentation
caused a timing change, and no acceptance tolerance, bound or ceiling was
introduced. Even the available B1 components do not satisfy the recorded
spread rule. The live quartet remains the previously accepted evidence.

### Distributions, host receipts and failures

Nearest-rank percentiles use every completed guard in each population;
outer is release-end minus install-start, install is installed-by minus
install-start, and render→installed is installed-by minus render-start.
The complete vectors and every failed Apply are retained in
[analysis](G2-followup22-resume-evidence/analysis). Counts are Apply rows /
completed guards / failed Applies; an expected failed Apply need not mean a
Go test failed. Ordinary agent and trace-agent results stay separate.

| Population | Applies / guards / failed Applies | Outer p50 / p90 / p99 / max (ms) | Install p50 / p90 / p99 / max (ms) | Render→installed p50 / p90 / p99 / max (ms) |
|---|---:|---|---|---|
| R-A1-agent | 140 / 137 / 3 | 28.396579 / 39.459996 / 46.354671 / 48.094211 | 14.925346 / 22.735286 / 27.029495 / 28.411582 | 15.029781 / 22.816232 / 27.092490 / 28.625166 |
| R-A1-agent-trace | 23 / 23 / 0 | 32.059055 / 38.202672 / 45.186285 / 45.186285 | 16.167351 / 19.009878 / 19.788954 / 19.788954 | 16.327309 / 19.069095 / 19.862109 / 19.862109 |
| R-B1-agent | 136 / 134 / 2 | 30.875969 / 41.546862 / 49.166242 / 52.345324 | 15.529396 / 23.468851 / 28.079370 / 28.242359 | 15.631968 / 23.558062 / 28.150005 / 28.315393 |
| R-B1-agent-trace | 23 / 23 / 0 | 30.608828 / 41.265926 / 45.609007 / 45.609007 | 14.961846 / 23.702304 / 25.551837 / 25.551837 | 15.039199 / 23.772157 / 25.776912 / 25.776912 |
| R-A2-agent | 137 / 134 / 3 | 30.147310 / 39.544397 / 48.341351 / 52.903495 | 16.091714 / 21.816662 / 25.783009 / 30.391279 | 16.179747 / 21.888375 / 25.846919 / 30.557965 |
| R-A2-agent-trace | 23 / 23 / 0 | 28.842469 / 41.488060 / 47.655221 / 47.655221 | 14.830113 / 19.077435 / 25.770801 / 25.770801 | 14.936035 / 19.162909 / 25.933116 / 25.933116 |

R-B2 ordinary-agent and trace-agent qualified distributions are unavailable.
The following **excluded** attempts all pass their Go tests but fail host
postflight. They are retained in full and are never substituted into the
selected distributions or comparison.

| Excluded attempt | Applies / guards / failed Applies | Outer p50 / p90 / p99 / max (ms) | Install p50 / p90 / p99 / max (ms) | Render→installed p50 / p90 / p99 / max (ms) |
|---|---:|---|---|---|
| R-A1-agent-1 | 139 / 136 / 3 | 28.845579 / 37.027421 / 44.617007 / 45.129485 | 15.315226 / 21.003293 / 26.343820 / 27.124737 | 15.397552 / 21.111321 / 26.418289 / 27.199730 |
| R-A2-agent-1 | 138 / 135 / 3 | 30.694382 / 40.529043 / 44.776711 / 49.791757 | 15.477613 / 22.973962 / 26.659117 / 26.760203 | 15.565880 / 23.202518 / 26.840803 / 26.870347 |
| R-A2-agent-2 | 138 / 135 / 3 | 28.021930 / 38.958687 / 45.331703 / 47.108681 | 14.978964 / 22.068972 / 26.165132 / 28.102026 | 15.063930 / 22.175956 / 26.245490 / 28.187844 |
| R-A2-agent-3 | 138 / 135 / 3 | 28.615615 / 39.894376 / 49.571862 / 50.486981 | 14.655341 / 21.013509 / 27.205850 / 31.596107 | 14.792110 / 21.354152 / 27.273180 / 31.678285 |
| R-A2-agent-4 | 137 / 134 / 3 | 29.842554 / 39.389685 / 46.486180 / 47.105169 | 14.768794 / 21.784935 / 28.652136 / 30.327764 | 14.851142 / 21.953651 / 29.212848 / 30.410649 |
| R-B1-agent-1 | 136 / 133 / 3 | 29.083617 / 37.078696 / 43.586815 / 44.605847 | 14.666054 / 20.403493 / 27.565183 / 29.081573 | 14.754913 / 20.472972 / 27.651956 / 29.158870 |
| R-B1-agent-2 | 138 / 135 / 3 | 30.737996 / 41.290209 / 47.500830 / 48.517143 | 15.240709 / 23.171915 / 27.560505 / 31.568721 | 15.326305 / 23.260248 / 27.621886 / 31.642759 |
| R-B2-agent-1 | 138 / 135 / 3 | 29.095756 / 38.181285 / 47.308497 / 48.987480 | 14.275860 / 21.289623 / 25.669554 / 26.234737 | 14.423014 / 21.526843 / 26.132114 / 26.308499 |
| R-B2-agent-2 | 138 / 135 / 3 | 31.059104 / 40.294691 / 48.226112 / 51.824759 | 15.194174 / 22.812486 / 27.113221 / 30.832942 | 15.266787 / 22.883811 / 27.198031 / 30.912537 |

Across the full resume, **69 host samples** are retained; **32 fail
qualification**. All competing-suite checks are empty. Nine completed
attempts fail postflight: A1 once, B1 twice, A2 four times and B2 twice.
[Qualification summary](G2-followup22-resume-evidence/qualification-summary.json)
indexes them. Selected preflight/ready/postflight summaries follow; each
cell is maximum r / minimum idle across its six current intervals. Full
load averages, D-state counts and monotonic bounds remain in the host summary
and raw receipts, including rejected samples.

| Population | Preflight r / idle | Before traffic r / idle | Postflight r / idle | D-state range across these receipts | One-minute load range across these receipts |
|---|---:|---:|---:|---:|---:|
| R-A1-agent | 0 / 80 | 2 / 87 | 0 / 87 | 9–12 | 10.14–10.92 |
| R-A1-agent-trace | 0 / 80 | 0 / 84 | 0 / 82 | 6–13 | 9.05–10.52 |
| R-B1-agent | 2 / 83 | 2 / 82 | 0 / 87 | 9–13 | 10.16–11.98 |
| R-B1-agent-trace | 1 / 85 | 1 / 81 | 1 / 84 | 11–13 | 10.10–11.65 |
| R-A2-agent | 0 / 85 | 0 / 86 | 1 / 85 | 5–8 | 7.62–9.29 |
| R-A2-agent-trace | 1 / 83 | 2 / 84 | 1 / 86 | 6–10 | 8.87–9.83 |

The [failed-Apply index](G2-followup22-resume-evidence/failed-apply-index.json)
contains **35 exact brackets**, including all excluded attempts. There are
29 pre-install failures and six install-command failures. Install-command
causality remains `unproven`; other rows use `not_applicable` only for the
install-overrun cause field. There is no measured install-budget overrun in
these completed-guard populations. No loss, failure or rejected sample is
waived, and none diagnoses the historical overrun.

The qualified B1 trace retains **23 Applies / 92 verified byte hashes /
1,024 command records**, joined exactly to the native producer's TimingEvidence
with run/boot/sequence intact. Every child in that trace starts and succeeds;
child CPU/switch and visible cgroup samples are available. Exact child-start
and hidden ancestors remain unavailable. The 413 selected ordinary-agent
Apply rows have command measurement off, preserving the unchanged caller.
See the [command audit](G2-followup22-resume-evidence/command-audit.json).
The exact TEST ONLY input/render Model and rendered NFT bytes remain in the
trace records and their native producer join. No code, digest input,
sequence rule, reserve or behavioral setting changed in this resume.

The agent collector's missing direct-helper output directory produces the
same non-fatal copy receipt as the retained driver; native agent producer
and trace outputs are retained. The trace fixture's absent watchdog log is
reported by the existing fixture and is not a test failure. Actual Go test
exit/status remains explicit in every selected or excluded run.

### Resume retention and gates

New evidence is under `G2-followup22-resume-evidence/`, alongside the untouched
first window. [Retention pins](G2-followup22-resume-evidence/retention-pins.json)
cover every new evidence file; the normalization ledger preserves original
hashes for public path/display normalization. Producer/Apply JSONL rows are
retained byte-identically. The note's original 24,444 bytes and the entire
first-window evidence tree are unchanged, checked against `c67977b7`.
The [preservation receipt](G2-followup22-resume-evidence/runtime/preservation.json)
records that boundary. No older archive changed.

All owned Docker containers, images, networks and volumes are absent in the
[cleanup receipt](G2-followup22-resume-evidence/runtime/cleanup.json).
No shared service, host networking or unrelated workload was touched.
Existing requests 1 and 2 stay with agent-core follow-up 31. Request 3 remains
open because the required qualified B2 populations are missing; no new
outside-path request is added. F8-02 native work remains outside this lane.

The required non-privileged gates ran in foreground order after the lab:
lane lint/unit/build (linux/amd64 and linux/arm64), unchanged agent/updater
consumers, committed-evidence hygiene, handover, root `make go-lint`, root
`go vet ./...`, tracked gofmt and diff checks. Tests, builds, lint, vet and
formatting pass. The first staged diff check identified significant blank
context spaces in the retained driver patch. Its exact original diff text
is now wrapped in JSON, with its raw hash preserved; the staged replay passes.
The initial check failure and replay remain recorded rather than overwritten.
See [gate results](G2-followup22-resume-evidence/gates/final-results.json),
[replay results](G2-followup22-resume-evidence/gates/replay-results.json),
and full logs under `gates/`. Final pins and staged evidence hygiene are
rechecked after retention. The postrun source/binary audit again verifies
both immutable arms before removing the five resume build/scratch directories.
These non-privileged passes do not complete the missing qualified B2 leg.
