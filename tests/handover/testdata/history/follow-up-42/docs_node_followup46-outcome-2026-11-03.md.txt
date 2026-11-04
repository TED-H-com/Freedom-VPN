# Follow-up 46 outcome — 2026-11-03

**No timing comparison: all three dry rounds stopped on foreign activity.**
No complete dry A/B pair passed, no measurement declaration was made, and no
reservation or measured leg ran. Both arms have zero pooled legs and empty
pooled distributions for every requested metric. No B-only failure or
attributable shift is established. No production or node/enforce change.

The runner and tests implement both requested repairs: admission receipts go
only to each leg's file, with no admission bytes on either Docker stream;
suite matching excludes the runner and all ancestors while CPU classification
continues to count ancestor/session CPU as foreign. Unrelated processes with
identical command text still disqualify. Sixty-five synthetic tests pass,
including 24 follow-up 46 tests that also pass from another working directory.
The old wrapper reproduces the exact combined-stream corruption; the new one
preserves true/JSON/multiline bytes and real tool diagnostics. Refusal exits
125, records its receipt, and stops the leg even if a consumer ignores exit.

The [audit](followup46-harness-audit-2026-11-03.md) lists the frozen output
readers in both arms, capture modes, expectations, and every helper call site.
A and B archives, every tree blob, fixtures and overlays match prior pins.
45's exact production-Go diff against #429 returned exit 0 and empty output.
All earlier tools and records remain unchanged.

## Dry gate

| Round | A | B | Disposition |
| --- | --- | --- | --- |
| 1 | Preflight refused: foreign lane containers | Not started | Admission itself was free; monitoring correctly stopped |
| 2 | 1/13 scenarios; 88.405 s; unrelated suite shell detected | Not started | 61 retained / 4 excluded whole Steps; no scenario failure |
| 3 | Preflight refused: unrelated gate process | Not started | Both inventories had been clear immediately before launch |

All three rounds used runner SHA-256
`adfcf5d93d8b8e30d3cae3c27d71aea5dec5b8688b9b9c49b9d3131a18788063` and the
same standalone invocation. No harness patch or qualification exception was
introduced between attempts. The partial A leg proves the frozen combined
consumer and clock path can reach scenarios, but it does not pass the required
full dry gate. Its 13 producer diagnostic failures remain in the JSON, not
silently discarded. No dry packet expiry or renewal predicate was obtained.

There are five marked intervals across the dry attempts, each recorded with
UTC, monotonic endpoints and the five largest foreign processes (pid, comm,
ticks). A-dry2's three clock samples passed zero-offset/aligned-bracket checks.
All qualification samples, source/binary/fixture identities, complete and
excluded span counts, failed operations and raw evidence hashes remain in the
[dry record](followup46-dry-legs-2026-11-03.json). Dry statistics are diagnostic
only, never timing evidence, and no dry leg is pooled.

The [declaration status](followup46-declaration-status-2026-11-03.md) records
why declaration was withheld. All eight intended measured legs remain
unstarted. Reservation requests, refusals, wait seconds and releases are all
zero. There is no first or second window to qualify, and no third reservation
was attempted. The [results](followup46-timing-results-2026-11-03.md) and their
JSON companion retain the empty pooled metrics separately from dry diagnostics.

## Ordinary main functional gate

The ordinary, uninstrumented main lab passed **13/13** required scenarios in
232.009 seconds on `7cfeb4c98727ee0893d7a10cd67210353c4ffa3b`. All 20,490
archive tree blobs were verified. It used the follow-up 46 Docker prefix and
no overlay; it ran after the dry item stopped and with no timing reservation.
The optional D116 installer case was skipped. This gate is functional only.

Expiry passed: 97 positives, 15 drops, last positive 100.591 ms before the
deadline, first drop 63.852 ms before it, and zero positive egresses at/after
expiry. Renewal observed two actual renewals and delivered 1,518/1,518 probes,
zero losses; its zero-loss predicate passed. These are results from this
controlled run, not a field-network or physical-arm64 claim. The functional
receipt preserves all 26 producer diagnostic failures from the complete
fault-injection stream alongside the passing scenarios; no failure was removed.
See [functional evidence](followup46-production-lab-2026-11-03.json).

## Coverage and limits

This follow-up changes only TEST ONLY Python measurement tools and records.
Functional and review gates cover ISO01/ISO02, SEC01 and lease deadlines
(§16.3), metering MTR01–MTR04 (§10.3–§10.7), and AT10/AT13/AT16/AT29 (§19).
D-133/D-138/D-150 attribution is still open: no arm has two pooled legs, no
window comparison exists and no code explanation can establish a measured
shift. Four-second PID/start sampling still cannot reconstruct short-lived
processes that begin and exit entirely between samples. Physical arm64 and
field-network measurements remain later work; amd64/arm64 builds are checked.

### Requests to orchestrator

- **2026-11-03 — D-133/D-150:** authorize a later dry-pair follow-up with a
  sufficiently quiet interval, including admission-waiting shells that match
  the unchanged suite-process rule. This follow-up reached its three-round
  limit. Do not reserve or continue timing from its withheld declaration.
  No product fix is inferred; preserve both arms and the qualification rules.
