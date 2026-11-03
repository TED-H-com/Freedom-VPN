# Follow-up 38: r21 attribution and Boot corrections

Follow-up 37 (`7b3d7bf8`) was reviewed before merge. The branch retains its
commits and first merges main `b9439411`. Before timing, it also merges
`8a775315` (isolation 17 and netlab 27), keeping both comparison arms on
that same updated base. Requirements: MTR03/MTR04,
§10.2, §10.5–§10.7, §10.9; ISO01/ISO02, §16.3; UPD04 and AT13/AT16. Physical
arm64 and real-network acceptance remain unmeasured; cross-builds are required.

## Judgments and before-fail evidence

The unmodified r21 proofs fail at 37 for H-1, H-2, both M-1 variants, M-2 and
L-1; the five held-question tests pass. They also compile at pre-37 main
`05b2c2f2` with the r20 fixture helpers. H-1 does not misattribute there; H-2
already does. M-2's entry-canceled test stops at main's failed runtime load,
so it alone cannot classify later stages.

| Finding | Judgment / production reachability | Origin |
| --- | --- | --- |
| H-1 | Confirmed: crash after kernel Apply, before retained persistence, without a sample | 37 regression |
| H-2 | Confirmed: counter absent or nil-bound at the head, then failed settlement and rebinding | Pre-existing; 37 narrowed the restart window |
| M-1 open | Confirmed: first new traffic conflicts with the old open head | Pre-existing live/restart rebinding gap; 37 reduced its size |
| M-1 closed | Confirmed: new baseline inherits retired identities | 37 regression |
| M-2(1) | Confirmed: uploader and incarnation-lock cancellation boundaries | Uploader bug pre-existed 37 (portable late-TERM before-fail); 37 detached the lock wait and exposed the entry-canceled path |
| M-2(2) | Confirmed: canceled updater recovery latches failure | Pre-existing |
| M-2(3) | Confirmed: sequential local budgets and independent adoption cleanup | 37's two-phase design; cleanup and post-TERM Run-to-Step mechanisms pre-existed |
| M-2(4) | Confirmed: runtime/baseline failure can exit zero and leave stale heartbeat | Pre-existing |
| L-1 | Confirmed diagnostic inconsistency after error clear | 37's orphan reason; main retained the actual error |

The original 37 receipt recorded these concrete failures: H-1 charged 8192
bytes to revision 1 instead of 4096; H-2 charged 4096 old-owner bytes to
revision 2; M-1 omitted the first 1024 new-revision bytes into gaps; M-2 sent
two uncanceled uploads and reported `recovery_required`, while the command
returned zero for a failed Boot; L-1 reported `member_path_unconfigured`
while the decision was `not_published`. The original M-1 values are evidence
of that fixture's failure, not accepted production traffic sequencing.
The converted M-1 receipts and portable late-TERM proof provide the origin
judgments above. M-2's lock and budget items additionally follow directly
from the old context and sequential-deadline call paths; the replacement
regressions exercise every blocked stage.

D-141 changes to the proofs:

- M-1 originally advanced the source before Apply, while baseline prohibited
  forwarding. Its closed-epoch helper also omitted retiring bindings and
  created an unrelated gap. The converted test performs a bound final sample
  after baseline and admits new bytes inside the fake Apply. On `05b2c2f2`
  the converted closed-epoch case passes; on `7b3d7bf8` both cases fail.
- r20's retained-ahead guard remains, but its assertion that retired bindings
  remain installed after settlement is replaced with a closed epoch and empty
  binding snapshot. That assertion encoded 37's M-1 regression.
- r21 Q2 now expects no additional power-loss gap after an already-finalized
  interval. Separate r20 and r21 open-epoch reboot proofs still require the
  power-loss gap and prohibit attribution across boots.
- L-1 tests the decision/heartbeat agreement and preserves the real topology
  problem through an unpublished generation, a later failed serving attempt,
  and eventual repair. It does not require discarding the pending evidence.

## Attribution mechanism

Before adoption can replace kernel rules, Boot captures the installed
marker and the persisted desired document, independently of a subsequent
control refresh. The desired document is accounting provenance even after
its serving validity expires; it never authorizes forwarding. A missing or
unverifiable document or unexplained live generation supplies no invented
owner. The attribution identity is `(device, member, binding_revision)`.

For a still-open same-boot epoch, only identities shared by the retained
model and the persisted candidate explain pending bytes. A baseline marker
on a second restart does not erase the earlier candidate: the first Boot
could have crashed after withdrawing but before committing settlement.
Changed identities produce mapping-conflict gaps. The meter keeps the
unchanged devices' records in those events; it never converts all devices'
bytes into an empty gap merely because one device changed.

After verified baseline, a final sample closes the interval. The retired
snapshot is cleared. A new epoch uses an unbound baseline, opened durably
before the next Apply admits traffic. Already-closed epochs retain their
clean close; new-boot continuity emits the existing power-loss gap for an
open interval. Neither inherits retired bindings.

A held or failed settlement keeps forwarding off. Step retries before
refreshing persisted desired state; enforce also checks this invariant.
Retries are scheduled no slower than the ten-second sample interval. The
H-2 proofs use a real two-second wall-clock hold, both missing and nil-bound
counter heads, and prove no intervening Apply or desired refresh. Recovery
of a persistent source/storage failure requires successful settlement; no
availability promise substitutes for accounting evidence.

The existing meter still records a conflict gap for an interval that actually
straddles a rebinding without a clean final-sample boundary. This change does
not claim exact attribution for that interval. Refused restarts now have a
verified baseline/final-sample boundary, so their first new traffic has no
such gap. A proposed extra checkpoint on every live binding change was dropped:
the real control-pass fixture already withdrew and closed before rebinding,
so it did not establish a reachable additional defect (D-141).

## TERM and diagnostics

Boot owns one ten-second deadline covering runtime loading, recovery,
adoption, cleanup, baseline, ledger migration/reconciliation and settlement.
Control/probe I/O and lock waits additionally follow the signal context.
The uploader uses a direct child of the process caller rather than a local
or per-Step budget, and is not started when that caller is canceled. Signal
cancellation propagates directly to request contexts; it does not wait for an
asynchronous cancellation callback. Recovery runs under the
local budget; an interrupted recovery is left unlatched, not reported as
`recovery_required`.

Boot runs its local operations under a bounded worker, as Close already did.
If a synchronous call ignores cancellation, Boot returns `ErrBoot` at the
deadline. Close waits for that worker before touching its stores, within its
own three-second allowance. Ownership stays with unfinished work until it
completes or the process exits. The actual ten-second-plus-three-second
blocking test and scaled stage tests cover the bound, including adoption's
nested cleanup. No independent cleanup phase can extend it. Run also checks
the signal context immediately after successful Boot, before entering Step.
An entry-canceled proof failed on the initial 38 implementation; a portable
late-TERM variant failed on pre-37 main. Each performed one extra blocked
audit in a new Step budget.
That transition is now stopped, preserving the process-level Boot/Close bound.

Runtime-load failure attempts baseline and publishes a failure heartbeat.
Baseline failure publishes `BaselineOK=false` and returns `ErrBoot`. The
command preserves these failures under TERM; successful baseline completion
followed by TERM remains exit zero. A permanently blocked worker cannot
safely publish a concurrent heartbeat: it returns failure, and the prior
heartbeat ages out under the existing watchdog contract.

The member-path pending flag survives diagnostic cleanup. Its heartbeat
reason overrides the decision only while a current member-path error exists.
A non-serving generation can therefore report its actual decision while the
next serving Apply still rechecks the durable topology expectation.

## Validation and timing

The hand-off records all gate output, preflight attempts and timing results.
The serving Step/Apply path changes, so D-133/D-138 requires a frozen A,B,A,B
comparison, with A at merged follow-up 37 and B at this production change.
Full `node-test-netlab` legs retain all failures and timing distributions.
No threshold, test in `followup12_test.go`, or unowned enforcement path is
changed. Any quiet-host D087/D104 failure is retained and handed off without
rerunning that item to obtain a pass.

### Final gate result and admission limit

Final production source: `37577472e00e280b4f1a1b8f0f9de0b9de47f444`.
Node lint, unit and integration tests, amd64/arm64 builds, unchanged updater
and installer consumers, R16–R21 regressions, handover, root `make go-lint`,
whole-module vet, evidence hygiene and plain `go test ./...` pass. The final
node gate took 226.202 s; integration reported 109.433 s. The final whole-module
run took 120.212 s. No dependency changed.

The full production `node-test-netlab` run passed all 13 scenarios, including
D087/D104 and F36 restart, with `FVPN_LAB_PREFIX=fvpn-w2-agent-core-f38`.
It started at 09:06:16.980 UTC after a qualified entry preflight. Its exit
sample at 09:10:36.461 UTC had three competing processes and one r=4 interval,
so **it is excluded from a qualified timing comparison**. Both the successful
suite result and the failed exit qualification remain evidence.

The fixed admission window was 07:43–09:13 UTC on 2026-11-03. Its final
in-flight preflight ended at 09:13:14; it admitted no run. All 56 preflights
are retained: 53 rejected entries, two qualified entries, and one rejected
exit. One qualified entry belongs to the interrupted earlier-source run
below. Actual rejected-preflight waiting, including the intervening polling
pauses, was 3129.537 s (52 min 9.537 s). A separate node-gate admission wait
was 240.316 s. Source integration, two necessary TERM refinements, gates and
the admitted suite consumed the remainder; the original deadline was never
extended. The long blocked interval was the members assembled run; the earlier
wait also encountered netlab's run and other test processes.

An earlier production attempt on `3f9389df` was interrupted during its first
ISO-F6 test to strengthen direct TERM propagation. It reached no D087/D104
case. Its log and binary/image pins are retained; it has no completed suite
or comparison result. Its owned container and image were explicitly removed.
The later Run-to-Step refinement interrupted only an admission wait.

A is tree `1e2cca44f1b029c39140b38f5696515264feec55`, reproducible with
`git merge-tree --write-tree bb686b365b19bf6e271139ef12f9aca20e0b7c6b 8a775315f232c9142d8a884b5be5d7d7b88addb0`.
B is the final production commit above. Their archives match all 17,045 and
17,049 tracked Git blobs, respectively, with zero mismatches. Both include
netlab 27's merged verifier; neither changes `followup12_test.go` locally.

| Planned comparison leg | Result |
| --- | --- |
| A1 | Not admitted: competing tests and runnable-task spikes at both remaining preflights |
| B1 | Not run: A1 was not admitted before the fixed deadline |
| A2 | Not run |
| B2 | Not run |

**D-133/D-138 remains incomplete.** No regression or no-regression judgment is
made from this single production run. The
[validation companion](followup38-validation-2026-11-03.json) contains every
preflight, source/archive/binary/image pin, before-fail receipt pin, completed
guard, failed operation, packet summary and gate result. Raw receipts remain
in the worktree's ignored hand-off directory. GOCACHE was queried as
`~/.cache/go-build`; all scratch used each archive's disk `.agent/tmp`.

Descriptive production measurements, excluded from a comparative claim:

| Metric, 136 completed guards | p50 ms | p90 ms | p99 ms | max ms |
| --- | ---: | ---: | ---: | ---: |
| Outer guard | 31.014 | 40.865 | 73.065 | 74.561 |
| Install command | 14.945 | 24.122 | 35.739 | 42.864 |
| Render to installed | 15.036 | 24.213 | 35.826 | 42.948 |

The stream retains 25 unsuccessful operations and no failed tests. Renewal
had 11 losses among 1518 probes; all 11 ingress timestamps fall inside recorded
guards, with zero capture drops. This is a correlation, not a zero-loss claim.
Expiry had 97 positive packets and 15 drops, no positive egress at/after D,
last positive egress D−100.317068 ms, and first drop sent D−63.799963 ms.
D087/D104's packet predicate passed. Physical arm64 remains unmeasured.

### Request to orchestrator — 2026-11-03

Provide a new quiet-host window for the complete frozen A,B,A,B comparison
under D-133/D-138. The full production lab gate is passed; timing qualification
is the remaining item. No unowned code change is requested. Final inventory
at 09:13:14 UTC contains zero owned containers, images, networks or volumes.

## Correction — follow-up 39, 2026-11-03 (r22)

The preceding follow-up 38 record is preserved as delivered. r22 disproved
several conclusions; this section supersedes those conclusions without
rewriting the earlier evidence.

- The discarded live-transition checkpoint was needed. A same-pass lease
  refresh can authorize the next desired generation without withdrawal, and
  an asynchronous sampler wake is not a committed boundary. A nil-bound or
  absent counter head can consequently charge an intermediate binding's
  bytes to the next member. Main `8a775315` reproduces this on the serving
  path before any crash. Follow-up 39 withdraws, commits the outgoing final
  sample, and opens an unbound epoch before any identity-set change. The
  unchanged lease-renewal path does not take that extra withdrawal.
- The promise that failed settlement blocks Apply was only implemented for
  restart settlement. A failed runtime hand-off did not set the pending
  gate. Follow-up 39 uses the gate for both paths, including source errors
  and wall-clock holds. FinalSample pauses only after verified withdrawal;
  recovery proves the periodic Run loop actually samples again. r22's
  separate claim that successful Apply skipped ResumeSampling is rejected:
  the delivered code called it unconditionally.
- Intersecting retained and desired identities was too restrictive and did
  not identify the actual candidate generations. The atomic live marker can
  prove a newer desired was never installed; an unpublished document cannot
  admit members; omitting a device supplies no counting rule for it. The
  latest retained model can belong to an already-closed epoch, and the first
  Apply can count before any retained model exists. Follow-up 39 persists
  accounting intent **before** Apply, scoped to boot and usage epoch. Its
  identities describe all traffic possible in that epoch; a stale model
  from a prior epoch is irrelevant. A fetched document alone is never an
  intent. Unique legacy candidate identities are retained when the committed
  head supports them. An absent/nil-bound legacy head without scoped evidence
  can conceal older add/remove/rebind traffic and remains an honest gap.
  Unchanged bound devices retain their records alongside that gap. Thus an
  unconditional claim that every single-generation-looking delta is exact
  would still be too strong without historical provenance.
- Retry scheduling did not guarantee recovery: ensureUsage latched success
  before meter construction. Follow-up 39 releases partial initialization
  and retries; the portable main proof confirms the latch pre-existed 37.
- Adoption performed meter reconciliation and started uploads before its
  final decision. The prepared meter is now read-only and requires an open,
  continuous epoch; bindings and upload activation wait until all gates
  succeed. r22's original M-2 false-gap fixture is rejected under D-141: its
  manual close used nil bindings and created the asserted gap before Boot,
  while also leaving serving rules live. A real open-epoch late-drift proof
  replaces it and checks that no usage effect precedes the decision.
- A blocked worker need not leave the last serving heartbeat until it ages
  out. Boot now publishes conservative state before blocking stages, retains
  the old receipt privately for adoption verification, publishes the verified
  baseline immediately before ledger work, and publishes after probe cleanup
  and on errors. Heartbeat writes have a 100 ms cap inside the existing Boot
  deadline, with a cancellation check before atomic publication. The timeout
  caller does not race the worker with another writer. A baseline Boot receipt
  reports kernel generation zero; r20's old generation-two expectation was
  an acknowledged-generation assertion and is corrected accordingly.
- Failed probe cleanup previously both left a stale BaselineOK receipt and
  marked the nonce cleared. It now reports failure, keeps the issuance
  uncleared, and retries clear without reinstalling or extending the timeout.
  A verified ordinary baseline can complete that pending clear. The kernel
  authorizes the failed-clear exception only until its installed element
  timeout, bounded by the original issuance deadline and 300-second maximum.

The actual Boot/Close allowance remains 10+3 seconds inside the shipped
`deploy/node/systemd/fvpn-agent.service` TimeoutStopSec=15s. There is no
main.go defaultStopTimeout or twenty-second shipped unit. Static datapath
configuration is validated by LoadConfig before production Boot; the raw-error
path is retained as an ErrBoot/heartbeat guard for invalid direct callers.

The previous full production lab pass and incomplete comparison remain facts
about follow-up 38. Follow-up 39 owes a new full production gate and a declared,
reserved A(main), B(final production), A, B comparison. Its qualification uses
foreign process/container exclusion and at most 0.5 foreign CPU cores per
interval; vmstat is recorded but does not decide admission.
