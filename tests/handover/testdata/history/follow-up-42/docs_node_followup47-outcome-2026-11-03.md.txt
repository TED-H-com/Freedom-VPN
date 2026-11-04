# Follow-up 47 outcome — 2026-11-03

Timing stopped on a frozen A scenario failure. A-dry1 ran all thirteen
scenarios, passed twelve, and failed TestLabExternalMeterReset. The reservation
was released promptly. B, the second dry reservation, the declaration and both
measured windows did not run. Both arms have zero pooled legs; no comparison,
B-only failure or attributable shift is established. No production change.

The first reservation request was granted after 0.060 seconds at
2026-11-04T00:47:00.202151Z (2026-11-03 EST); release succeeded at
00:52:12.151923Z. After the required 60 seconds, preflight was clear in 4.026
seconds. No activity wait, reservation refusal or running qualification breach
occurred. All clock samples matched and admission receipts and Step spans were
written. The same committed runner hash ran the dry leg. All five marks and
five-metric retained/excluded distributions are preserved as dry diagnostics,
never timing evidence. The producer failure count is not a scenario count.

Dry expiry passed: 97 positives, 15 drops, last positive -100.267 ms, first drop
-63.827 ms, no positive egress at/after expiry. Renewal delivered 1503/1518;
all fifteen losses had ingress within the outer guard, so zero-loss is false.
Two actual renewals were observed. The frozen D087/D104 scenario passed; the full analyzer output is retained.

The uninstrumented ordinary main lab passed 13/13 on `6548329f8f82a01de99d46392e70018a7cdbedcd`.
Its expiry passed with 97 positives/15 drops, last positive -100.281 ms,
first drop -63.766 ms and no late positive. Renewal delivered 1518/1518 with
zero loss. This is a functional gate only. D116 is optional and skipped.

The first plain full-module run failed an unchanged R23 checkpoint-read/restart
subtest with a heartbeat deadline. Its earlier review, targeted recheck and
plain full-module retry passed; the initial failure is retained, without a
code change or root-cause claim.

The runner, tests, archive/blob/overlay/fixture checks and exact empty B/#429
production diff are complete. All earlier tools and records are unchanged.
The declaration-status record explicitly withholds a timing declaration.

### Requests to orchestrator

- **2026-11-03 — D-133/D-150:** route the frozen A TestLabExternalMeterReset
  scenario failure for a later investigation before authorizing another timing
  attempt. It asserted at scenario_az_external_test.go:88 after 12.393 seconds:
  open reason after_final_sample, counter bytes 165612, discontinuity 0,
  generation 3. Main's ordinary lab passed the same scenario in 1.466 seconds.
  No B observation exists, so this is not a B-only failure or an attributable
  shift. This follow-up is stopped by item 3's scenario-failure rule; unused
  dry/measured reservations are not permission to resume. Preserve the frozen
  arms and evidence; any production fix belongs to a later follow-up.

- **2026-11-03 — R23 gate follow-up:** retain the intermittent boot heartbeat
  deadline at node/agent/followup40_test.go:228 for later investigation. The
  earlier R20–R26 gate, targeted recheck and one plain full-module retry passed.
  The original full-module exit 1 is preserved; this runner-only task changes
  neither production code nor the existing test.
