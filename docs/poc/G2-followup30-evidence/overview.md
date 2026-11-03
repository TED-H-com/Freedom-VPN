# Follow-up 30 — live NFLOG validated; seven guard drops attributed

Read the [orchestrator’s interpretation of PR #415](https://github.com/TED-H-com/Freedom-VPN/pull/415)
and its roadmap row first. F28-01 stays closed as the prior test-isolation defect;
follow-up 28’s gate 2 stays FAIL. **F27-01 remains unresolved and B1 remains FAIL.**

- **Collector: PASS live.** Eight expected packets match exactly once, one unlogged
  negative control is independently counted, receiver 8 / 8 / 0 drops, every
  record decoded with microsecond pcap precision and a kernel timestamp.
  [Declaration](collector-declaration.md), [result and qualifications](collector-outcome.md).
  Two slots were used: the first stopped at preflight; the second completed the
  collector. Cleanup was completed separately without rerunning it.
- **One guard window, no retry.** Seven missing TCP node egresses have seven
  exact guard-drop matches, all upload `iifname` rules inside install brackets.
  There were 101 verified Applies, 118 successful reads and seven retransmissions;
  the longest read was 416.740 ms. This demonstrates the candidate mechanism in
  this run, but does not recover B1’s 1.501408-second failure.
  [Declaration](guard-declaration.md), [outcome/cost/proposal](guard-outcome.md),
  [complete drop ledger](drop-ledger.md), [deterministic replay](replay.json).
- **Guard window UNQUALIFIED.** A four-second interval overlaps its beginning
  and reports 2.478498 foreign cores. No retry, status promotion or quiet-host
  performance claim. The observed diagnostic cost does not isolate overhead
  against an uninstrumented control.
- **Final suite assertions PASS; host qualification UNQUALIFIED.** The ordinary
  full suite passed in 292.536 s with logging disabled, including AT10 and AT12.
  Its continuous monitor recorded eleven foreign-CPU breaches, maximum 4.310365
  cores, over 74 covering intervals. No other lane suite or lab container was
  observed; maximum check interval was 4.103481 s. Its verdict is not presented
  as an entirely green qualified gate. [Receipt](final-gate2-complete.json),
  [qualification](final-gate2-complete-suite-qualification.json).
- **All consumers and gates 1, 3–9 PASS without production/consumer edits.**
  [Gate summary](final-gates-summary.json). Earlier evidence-pin validation errors,
  cleanup errors, denied reservations and the corrected monitor placement remain
  retained. [Monitor correction](gate2-monitor-correction.md).

No production or ordinary test source changed. D-120’s probe, 1.5-second timeout,
UDP observation and analyzer are unchanged. The only proposed guard improvement
is a separate design and D-133 comparison; none is implemented. Physical arm64
remains unmeasured; the amd64 and arm64 builds pass.

D-098 verifies **1,506 payload files and five manifests** from follow-ups 25–29
at [start](d098-initial.json) and [completion](d098-final.json), including both
forms of every prior gzip. Follow-up 29’s manifest remains
`d9f9e7d539a884c113a461646448539343c1da803cf9dc378c4e0ab59dbddefa`.
New evidence is confined to this directory. [Publication mappings](publication.json)
retain lossless gzip hashes; [location redactions](location-publication.json)
remove residual home-relative admission locations without changing measured
values. All raw packet captures stayed in disposable lab scratch.

[Admission summary](admission-summary.json) records all reservations, waits and
checks. `fvpn-pg` / `postgres:17-alpine` is recorded as shared infrastructure;
its CPU always counts as foreign. The initial collector preflight’s first six
checks also caught another Go test. CPU accounting retains PID/start-time
identity, all excluded class totals and disappearing PIDs; exited processes’
final ticks cannot be recovered between `/proc` samples. No host networking or
admission script was modified. [Final cleanup](final-cleanup-inventory.json)
proves zero owned containers, images, networks, volumes or task scratch.

## Requests to orchestrator — 2026-11-03

1. **`docs/ROADMAP.md` (outside owned paths):** record finite live collector
   validation and seven exact guard-drop matches, with both CPU qualifications
   failed, no B1 explanation and all historical statuses unchanged.
2. **Launch coordination in the orchestrator’s private repository:** the
   reservations excluded competing suites but did not prevent non-suite CPU
   bursts. The guard overlap was dominated by a foreign `git` process; the full
   suite also recorded other non-suite processes. Review the retained CPU
   intervals before another qualified comparison. No threshold relaxation or
   additional diagnostic attempt is requested by this delivery.
