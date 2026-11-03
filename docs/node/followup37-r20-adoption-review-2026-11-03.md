# Follow-up 37: review r20 findings on same-boot adoption and Boot signals

Baseline: `05b2c2f2` (main after #401, agent-core follow-up 36).
Review: r20 (`r20-adoption-signal-review`, proofs at commit `639eda6b`).
It covers MTR03/MTR04 (§10.5–§10.7), ISO01/ISO02 and §16.3 restart
authorization, D-064 probe exceptions, and the restart mechanism in AT13
(§19). No historical acceptance result changes. H-2 (kernel verification of
`denied_v4`/`host_v4` contents) belongs to netlab 27 and is not addressed
here.

Each finding was judged on the unchanged baseline first. r20's two
agent-core proof files were brought over; on `05b2c2f2` all five finding
proofs failed and the four "held" proofs passed. Where r20's test built a
state production cannot produce, it was rewritten so it is reachable.

## H-1: refused adoption and pending usage — confirmed, reachable, fixed

`preserveServingOnStop` deliberately leaves the serving rules in the kernel
across a same-boot restart. Packets keep incrementing the per-device
counters until the successor acts. If the successor refuses adoption (for
example a changed deny/host set, since follow-up 36), `loadRuntime` restored
the installed model but not the meter bindings. The first non-serving Step's
hand-off, or a first Apply that removes a device or rotates its binding
revision, then read the delta with no matching binding: a `mapping conflict`
gap with no records.

**Rule:** the meter's bindings describe what the kernel counters are bound
to, which is the retained installed model until the first full Apply
rebinds them.

**Mechanism:**

- `loadRuntime` sets the meter bindings from `retained_applied` together
  with the installed model.
- After a refused adoption, Boot installs its baseline (member admission
  stops), opens the ledger, and `settleRetainedUsage` takes one sample of
  the open epoch against those bindings, before the first Step can hand off
  or rebind.
- A boot-ID change needs no special case: the meter's continuity check
  closes the old epoch as a power-loss gap without attributing any delta,
  before any binding is consulted.
- Adoption itself is unchanged; it still attributes the delta through its
  own sampler.

Proofs (`node/agent/review_r20_test.go`):

| Test | Scenario | Baseline result | Result |
| --- | --- | --- | --- |
| `TestR20_H1_…/unpublished_first_step` | r20's scenario: refusal, then unpublished | 4096 attributed, one empty gap | 8192 at revision 1, no gap |
| `…/rebinding_first_apply` | refusal, then Apply rotating revision 1→2 | 4096 attributed, one empty gap | 8192 at revision 1, 0 at revision 2, no gap |
| `…/two_crashes_same_boot` | process 2 refuses, settles, is killed; process 3 refuses again | 4096, one empty gap | exactly 8192, no gap |
| `…/boot_changed` | new boot ID with counters that look continuous | one empty power-loss gap | unchanged: one empty power-loss gap, only process 1's 4096 |

Limits: a settlement sample that fails (clock hold, source failure) leaves
the delta pending under the same bindings; a non-serving hand-off still
attributes it, and a rebinding Apply records an honest gap rather than a
misattribution. A crash between `plane.Apply` and the `retained_applied`
write (r20 H-2 window B) can leave counters already sampled under the newer
bindings; the next refusal then records a gap, not a misattribution.

## M-1: TERM during Boot — confirmed, reachable, fixed (local completion)

Follow-up 36 installs the run command's signal context before `Boot`. A
TERM during adoption's lock wait or evaluation made adoption return false,
and Boot then ran `Baseline` with the canceled context. The real nft
executor fails on a canceled context, so the baseline was not installed,
yet Boot overwrote the serving heartbeat with `Serving=false
BaselineOK=false` and the process exited 0, leaving the previous rules
live under a heartbeat that denied them.

**Choice:** Boot completes its local work under a bounded local budget,
rather than leaving the prior heartbeat untouched. This is the fail-closed
reading: a stop request during startup cannot prove the retained rules
still match this process's configuration, so member forwarding ends at a
verified baseline whose heartbeat describes it. Leaving the old heartbeat
would keep possibly refused rules forwarding until watchdog staleness, with
no agent running. The cost is availability: the next start repeats Boot
and the first full Apply.

`bootLocal` is `newLocalBudget(…, 10 s)` with the control budget bound to
the caller. Lock waits, the probe and resolver refresh, and control requests
still end on TERM; `loadRuntime`/freeze persistence and, after adoption,
the baseline, ledger open, settlement and heartbeat do not. A canceled
caller also skips the start-up health answer. Ten seconds equals Step's
maximum local allowance, so with Close's three seconds the agent stays
inside the 13 s cooperative budget and `TimeoutStopSec=15s`.

`TestR20_M1_TermDuringBootInstallsBaselineAndExitsZero` cancels the run
context at each stage: while another holder keeps the node lock busy, inside
the probe, and on entry to `Baseline` (whose plane honors cancellation like
nft). It runs the real `Engine.Run` and applies the run command's
exit rule.

| Stage | Baseline | After |
| --- | --- | --- |
| lock wait | exit 0, heartbeat `BaselineOK=false`, baseline not installed | exit 0, `Serving=false BaselineOK=true`, generation 0 rules |
| evaluation | same as lock wait | same as lock wait |
| baseline | same as lock wait | same as lock wait |

## M-2: check order in adoption — confirmed, reachable, fixed

The deny/host set comparison and the installation-sample clock checks need
only the retained model, the stored lease and the preliminary decision.
They ran after the node lock (up to 2.01 s) and after the live probe and
resolver refresh, while the old rules kept forwarding. They now run before
both; the set comparison is repeated after evaluation. The retained-ahead
generation guard (L-1) is also there.

During adoption this process holds the exclusive node lock. Evaluation's
D-064 check opened a second descriptor, met its own `flock`, and reported
the updater as holding the lock: a live health challenge then installed a
probe baseline over the retained rules and adoption failed. `updateLockHeld`
now returns false while this process holds the node lock; the updater
cannot hold an exclusive lock that we hold.

Proofs: `TestR20_M2_RefusalPrecedesLockAndProbe` (changed sets and a
mismatched installation sample, node lock busy: baseline took about 2.07 s
and refused after the wait; now it refuses without the lock wait or a probe)
and `TestR20_M2_AdoptionLockIsNotUpdaterLock` (baseline installed a probe
baseline and refused; now unchanged rules are adopted, no probe baseline).

## M-3: stale apply error and shutdown heartbeat — confirmed, fixed; r20's proof rewritten

r20's proof rewound the monotonic clock two hours between processes; that
state is unreachable. The defects are reachable without it:

- **Heartbeat:** when `Close` cannot preserve serving (lease or policy
  deadline passed, updater barrier), its hand-off installs the baseline but
  left the serving heartbeat. It now writes the heartbeat after a hand-off
  that withdrew serving. A hand-off that failed before its baseline keeps the
  old heartbeat, which still describes the kernel.
  `TestR20_M3_ShutdownHandoffRecordsWithdrawal`: the baseline left
  `Serving=true` over generation 0 rules.
- **Apply error:** any failed live read during adoption (rules replaced or
  tc/link drift while no agent ran, inside the heartbeat staleness window)
  latched `applyError`. Only a serving Apply cleared it, so a later
  unpublished or draining generation was never acknowledged. Boot now clears
  an error set by that refused adoption once its own baseline is installed
  and verified. Quarantine (foreign tc) keeps its error. The rejected read
  stays in the retained serving-read evidence and the log.
  `TestR20_M3_RefusedAdoptionErrorDoesNotBlockNonServingAck` (rules replaced
  out of band, no clock rewind): the baseline acknowledged generation 2 of 3
  with safety failed. `TestR20_M3_AdoptionQuarantineKeepsError` guards
  quarantine.

## L-1 — empty deny list and generation ahead

- **Empty `DeniedV4`:** unreachable as installed state. `enforce` rejects an
  empty deny list (`denied_v4 is empty`), so no verified Apply retains one.
  A candidate can be empty (TEST ONLY exemptions covering every spec
  prefix), but it is then compared with a non-empty installed list. The
  comparison now treats a non-nil empty list as unequal on either side,
  kept as a guard: `TestR20_L1_EmptyDeniedV4NeverEqual`.
- **Generation ahead:** harmless. `syncDesired` persists desired generation
  N+1 before Apply, so a crash between the `retained_applied` write and the
  ACK leaves desired ahead of `applied_snapshot` and adoption's first check
  refuses. An explicit `installed.Generation == appliedGen` guard keeps that
  invariant local. `TestR20_L1_RetainedAheadOfAckRefusesAdoption` reproduces
  enforce's durable write order and a kill: adoption is refused on the
  baseline too, and the test also requires the meter bindings that describe
  the installed N+1 counters (H-1).

## Scope and timing

No serving Step or Apply code changed. Changes are in Boot, adoption,
`updateLockHeld` (probe exceptions while the node lock is held, only during
adoption), shutdown's withdrawal branch, and `loadRuntime`'s meter bindings.
A same-host A,B,A,B comparison is therefore not owed; the full privileged
netlab gate was rerun. Physical arm64 remains unmeasured; both architectures
are cross-built.

## Correction dated 2026-11-03 — follow-up 38, review r21

The original text above remains the historical follow-up 37 record. The
following claims were disproved; they must not be used as current guarantees.

- **H-1 attribution rule and crash-window limit:** `retained_applied` can lag
  an Apply already counting for a different member/revision. No sampler has to
  run between Apply and persistence. Restoring only the retained bindings can
  bill the old member for the new member's bytes. The claimed “next refusal
  then records a gap” was false. Follow-up 38 uses the persisted desired
  document captured before refresh, retained identities and a live installed
  generation read. Only identities common to every plausible generation are
  attributed; changed/unknown identities gap individually. Unchanged devices
  retain their records, even in an event whose overall quality is `gap`.
- **r21 H-2; H-1 failed-settlement limit:** a missing counter head or a nil-bound head
  does not cause a mapping conflict. On a held settlement, the subsequent
  Apply could bill old bytes to the new binding. This mechanism predates 37;
  37 narrowed its restart window. Follow-up 38 blocks desired refresh and
  Apply until a final settlement closes the pending epoch. A short clock hold
  retries with forwarding off; a persistent read/write failure keeps it off.
- **r21 M-1; H-1 mechanism and proof coverage:** the retained identities explain only
  pending bytes. Leaving their epoch open caused an avoidable post-baseline
  rebinding gap; restoring them into an already-closed epoch introduced a
  regression. Follow-up 38 closes after settlement, clears retired identities,
  and prepares a fresh unbound epoch before Apply. Its tests admit new traffic
  inside Apply. The original `rebinding_first_apply` test observed only the
  old bytes and could not prove attribution of the first new traffic.
- **TERM control-I/O claim:** `ensureUsage` could start its uploader under a
  background context after TERM, and its incarnation lock wait used the local
  context. Follow-up 38 binds the uploader lifecycle and that lock wait to the
  caller. Recovery uses the shared local budget; cancellation alone supplies
  no evidence for `recovery_required`.
- **TERM time bound:** 37 allocated two separate ten-second phases; adoption
  cleanup could also allocate an independent allowance. The claimed 13-second
  bound was false. Follow-up 38 owns one ten-second Boot deadline, including
  recovery, adoption cleanup, ledger construction and settlement. A bounded
  worker also handles an executor or filesystem call that ignores cancellation;
  Close waits for its ownership to end within Close's separate three seconds.
  An uncompleted worker retains its stores until completion or process exit.
  Run also checks TERM after successful Boot before entering Step; the old
  transition could start another audit budget after the completed baseline.
  That transition predates 37 and has a portable before-fail regression.
- **TERM success/exit claim:** the old command could return zero after failed
  runtime load or baseline establishment, and load failure left the prior
  heartbeat. Follow-up 38 writes a failure heartbeat when that local write can
  run, returns a distinct Boot error and retains nonzero status under TERM.
  A stop after successful baseline completion still returns zero. A blocked
  Boot worker cannot safely publish a concurrent heartbeat; it returns failure
  at its deadline, leaving watchdog staleness as the existing containment path.
- **r21 L-1; M-3 diagnostic cleanup:** clearing the adoption error could leave
  `memberPathPending` overriding an unrelated non-serving decision. Follow-up
  38 retains the topology evidence for a later serving check and scopes the
  heartbeat override to an actual current member-path error.
- **Scope/timing:** 37's “no Step/Apply change” describes its source edits, not
  proof that its startup changes had no later attribution effect. Follow-up
  38 changes Step/Apply: pending settlement precedes refresh, and a fresh
  post-final-sample baseline clears retired attribution before Apply. Its timing qualification
  and remaining limits are recorded in the follow-up 38 design note.
