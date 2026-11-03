# Follow-up 46 declaration withheld — 2026-11-03

**No measurement declaration was made. No reservation or measured leg ran.**
Item 3 requires the current runner hash to pass complete dry A and B legs
before any reservation. Three rounds exhausted that limit without a complete
pair; the unrelated-container and suite-process stops remain mandatory.

The [dry record](followup46-dry-legs-2026-11-03.md) preserves each attempt.
The [identity record](followup46-declaration-status-2026-11-03.json) preserves
the verified arms, fixtures, clock binaries, runner/test/analysis hashes and
exact standalone invocation. It is an identity/status record, not authorization
to run timing or a claim that this runner passed the full dry gate.

A remains `567080be1969bf22782617e78c7605c2c92a52a2`; B remains
`38420dadef6ae9d0236288eef533bc1ddee0da53`. Production equality to #429 was
verified with 45's exact command and empty output in the audit. Prefix remains
`fvpn-w2-agent-core-f46`; all fresh evidence is in `.agent/f46/timing/`.

The unexecuted intended windows were A1,B1,A2,B2 and A3,B3,A4,B4, full
node-test-netlab per leg, at least ten minutes apart, thirty minutes maximum
per reservation. No less-than-seven-minute start, thirty-second expiry stop,
sixty-second drain or ninety-minute reservation wait was exercised. There were
zero reservation requests, zero reservation refusals, zero reservation wait
seconds and no release receipts because no reservation existed.

Unchanged rules remain frozen in the runner: zero clock offsets/aligned
brackets; foreign CPU above 0.5 cores marks intervals and their top five
processes; all five distributions exclude overlapping spans; a leg needs at
least half of its whole-serving-Step population retained; other suite/container,
unreadable state and intervals above five seconds stop/disqualify. Ancestors
are excluded only from suite matching and remain foreign CPU. Admission before
each command is silent to both frozen data streams and separately receipted.
D087/D104 failures, stopped/failed operations and D-138 attribution stay as in
45. No arm has two pooled legs, so no comparison can be drawn.

The dry attempts used the exact launch below, with F46_RUN equal to dry1,
dry2 and dry3 respectively. A future session needs new authorized dry rounds
and a successful full pair before a fresh declaration; do not resume timing
from these status records.

```sh
python3 -B tests/netlab/agent/followup46_timing.py run > .agent/f46/timing/driver-"$F46_RUN".stdout 2> .agent/f46/timing/driver-"$F46_RUN".stderr
```

### Requests to orchestrator

- **2026-11-03 — D-133/D-150 dry admission:** arrange a later follow-up with
  a quiet interval long enough for the full dry A/B pair, including other
  lanes' waiting shells under the existing suite-process rule. This session
  used all three dry rounds and may not reserve. Do not weaken qualification
  or infer a production fix from these interruptions.
