# Follow-up 34: central serving-lease generation holds

2026-11-01 — r18 M-3 / agent-core 32 request 2; §5.1, §5.2, §15.3,
§16.3, ISO02 / SEC01, AT08 / AT10 / AT29 supporting central evidence.

## Recognition and publication

`policy_missing` together with either a `lease` check carrying
`STALE_GENERATION` or an `errors[]` entry with that code and `stage=lease`
is the reported hold marker. It does not compare the applied ACK to the desired
generation. In particular, an ACK equal to desired does not confirm serving.

`PublicationDataPathFailure` returns the existing fixed cause `stale_generation`
for a corroborated hold, before generic `check_failed` / `forwarding_disabled`.
The cause still also describes an ordinary lagging ACK; its meaning is not
changed. The new incident table distinguishes the serving-lease incident from
that older case. No contract or existing admin field changes meaning.

The reported lease ID must resolve to the same node's current incarnation. Its
persisted `policy_generation` must agree with the signed envelope's lease ID,
node ID, incarnation ID and generation. A missing, unknown or foreign-incarnation
lease returns `lease_unconfirmed`; it cannot borrow evidence from the latest
issued lease. A hold marker remains a hold even with a missing ACK or an ACK
equal to the lease generation; neither condition proves the hold ended.
Issuing a newer matching lease does not prove the node served it.
The ordinary populated publication gate also checks the stored lease generation
against the ACK, as well as the existing clamped expiry.

Every unrelated failed check wins, regardless of array order or duplicate name.
Only the lease's `STALE_GENERATION` and the agent's exact consequential qos /
source_binding `VALIDATION_FAILED` checks belong to the hold. A retained
`TC_FOREIGN_FILTER`, isolation/meter failure, dependency failure or independent
error still returns a safety failure (`check_failed` or `isolation_unconfirmed`).
The hold incident is nevertheless retained beside those failures. A contradictory
`member_forwarding_enabled=true` with a hold marker normalizes to false.

D-075's `node_incarnations.stale_generation_grace_at` logic is unchanged. That
one-shot baseline-readiness grace is neither the hold clock nor proof of a
working member path. The baseline `agent_policy` ACK can complete independently;
the publication data-path gate cannot complete on that ACK alone.

## Durable evidence and escalation

Migration `00194_node_generation_holds.sql` adds:

- `node_generation_holds`: one open incident per node/incarnation, first and last
  central receipt times, first observation time, latest report ID, escalation
  time, terminal time and fixed terminal reason. Closed rows remain history.
- `node_generation_hold_reports`: every incident-associated report's ID, kernel
  boot ID, reported lease ID, applied and signed lease generations, reported
  steps / elapsed milliseconds / alarm, and summary validity. The immutable
  `node_observations` row preserves the complete original report and node time.

The latest sample is joined through `(node_id, incarnation_id, latest_report_id)`;
the complete history is joined by `hold_id`. All generation/counter numbers use
exact numeric storage and decimal strings in JSON/audit. They never pass through
floating-point conversion. Process restarts can retain the same kernel boot ID;
central does not need to detect a process restart to retain the incident.
An absent applied ACK stays SQL NULL / JSON null in the new diagnostic record.

Only this exact lease-stage summary is parsed:

```text
Serving lease generation hold: applied=<decimal> lease=<decimal> steps=<decimal> elapsed_ms=<decimal> alarm=<bool>.
```

Parsing requires canonical uint64 decimal values and `true`/`false`. Missing,
malformed or duplicate summaries leave all three reported counters unknown
(SQL NULL), with `summary_valid=false`; they never clear the hold. A syntactically
valid summary with conflicting generations or zero steps retains its reported
counters, but is marked invalid. Original text remains in the observation, never
copied into the audit. Central neither accumulates process steps nor changes a
reported `alarm=false` into `true`.

The first receipt queues `node.generation_hold.check` for 30 seconds later.
A valid reported `alarm=true` escalates immediately on ingestion; otherwise the
worker escalates when central age reaches 30 seconds, even without another
heartbeat. Periodic `node.reconcile` also checks open incidents and repairs a
missed/dead-lettered deadline check. Execution can be delayed by worker backlog;
30 seconds is the diagnostic threshold, not an availability or delivery SLA.

The existing admin audit endpoint receives exactly one
`node.generation_hold.escalated` event per incident, via
`operations.RecordGenerationHoldAlarm`. Its bounded evidence includes the
incident/report/boot/incarnation/lease IDs, receipt times, generations and
nullable reported counters. Its `succeeded` result means the escalation was
recorded; it makes no claim that forwarding recovered. Audit insertion and
`escalated_at` commit in the same locked transaction. Retries, report replay,
central restart and concurrent replicas cannot multiply the escalation. The
worker's normal claim fence applies.

A delayed alarm report associated with the active incident also escalates it
without replacing a newer report's counters. The audit's `alarm_report_id`
identifies that trigger; it is null when only central age triggered escalation.

## Ending an incident

A newer or equal-time report without the hold marker that names a corroborated
matching-generation lease ends the mismatch. Mere absence of the marker,
an unknown lease, issuance of a new lease, or a new desired generation does not.
An older observation cannot end or reopen an incident; its immutable receipt
remains available. Associated older hold samples never replace the current
sample. `last_seen_at` uses central receipt time, independent of D-075.

Identity revocation/retirement, incarnation replacement and identity freeze end
an incident. Enrollment revocation and conflict-freeze transactions and the
worker's discovery-freeze transaction close it immediately. Reconciliation also
handles terminal state discovered later. Frozen heartbeats cannot create a new
incident.

If all members of the reported signed lease are suspended, closed or explicitly
admin-suspended, the incident ends with `members_revoked`. The member worker
checks this before external synchronization, and periodic node reconciliation
also handles it. One member's removal cannot erase an incident still affecting
survivors. An empty lease grant set supplies no positive evidence of revocation.
Grant withdrawal that leaves members eligible does not end this node-wide hold.
These checks only close diagnostic incidents; they grant no entitlement.

## Validation and consumer hand-offs

Central regression tests:

```sh
go test -count=1 ./internal/nodes -run TestAT29_M3
go test -count=1 -tags integration ./internal/nodes ./apps/worker -run TestAT29_M3
```

Use the worktree's disk `.agent/tmp` as TMPDIR/GOTMPDIR. The integration fixtures
use dbtest-created databases. Issuer-row fixtures are labeled TEST ONLY; the
worker test exercises actual observation ingestion, durable scheduling and the
claim fence. These are central proofs, not kernel traffic or physical arm64
measurements. The existing nodes build gate cross-compiles amd64 and arm64.

Dated requests to orchestrator (outside this lane's owned paths):

1. **2026-11-01 — isolation / AT08, AT10, AT29:** extend the assembled acceptance
   fixtures under `tests/` and `docs/e2e/at/isolation/` to retain the accepted
   `policy_missing` + lease-stage `STALE_GENERATION` report with applied==desired,
   its exact lease ID and signed generation, and the central incident/audit IDs.
   Assert `stale_generation` before generic serving failures, baseline remains
   installed, counted forwarding remains closed, and no publication operation
   completes from the ACK. Repeat with an unrelated internet/meter/retained
   enforcement failure in both array orders: that failure must win while the
   same incident remains. Restart the process with reset counters, then reboot
   with a different kernel boot ID; first_seen and escalation count must persist.
   Check one escalation after an alarm or central 30-second age, including no
   new heartbeat, then matching-lease recovery and a genuinely new incident.
2. **2026-11-01 — members / AT13, AT29:** in the owned member acceptance fixtures,
   assert suspension/closure of the sole leased member closes the incident
   without any serving recovery claim; revoking one member while a sibling
   remains eligible preserves the incident and forwarding hold. Assert retained
   source-binding/TC safety failures keep priority over `stale_generation`.
   Please confirm the node-wide all-affected-members interpretation; no typed
   per-member hold identity exists in the current observation contract.
3. **2026-11-01 — contracts / agent-core:** consider an optional typed hold
   diagnostic in `contracts/openapi.yaml` and the generated clients, carrying
   exact decimal generations/steps/elapsed_ms and a boolean alarm. If process
   identity is needed, distinguish it explicitly from the existing kernel
   `boot_id`. This would remove dependence on a human summary. This lane makes
   no contract change; the current parser and unknown-counter behavior ship now.
