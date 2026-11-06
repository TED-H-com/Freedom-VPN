# Central serving-read diagnostics — 2026-11-06

Central accepts the existing D-164 `serving_read` field on `ObservationReport`
and returns the selected diagnostic on every `NodeDetail` path. The five fields
remain `read_at`, `source`, `candidate_generation`, `verified`, and `diagnostic`.
Both verified and rejected reads retain their opaque text. A read is diagnostic
evidence; it grants no serving permission, acknowledges no applied generation,
and changes no health, safety, forwarding, lease, freeze, or recovery decision.
There is no added freshness or source/check-detail consistency requirement.

This implements Freedom VPN's observation persistence and detail projection.
The existing [NetBird 0.82.0 adapter](../netbird/ADAPTER.md) continues to own
upstream peer, route, group, policy, and nameserver operations. Its upstream
observed-state reads do not identify which Freedom VPN observation updated
`latest_checks`; this projection uses the existing Central transaction instead.

## Selection, presence, and reads

`SubmitObservation` retains its node-row lock and newer-or-equal `observed_at`
selection rule. The same `UPDATE nodes` writes `latest_checks`, the retained
`serving_read`, its incarnation binding, and `resource_version`. Equal timestamps
therefore follow actual serialized acceptance order, including when receipt
clocks tie or move backward. Older reports and original-ACK replays do not
replace the selection. `read_at` never selects an observation.

The common detail query reads checks, diagnostic, and version in one statement.
This covers ordinary `GetNode`, transactional mutation responses, idempotent
mutation projections, and the admin GET wrapper that also adds DNS evidence.
ETags retain their existing resource-version format and conditional behavior.
The unrelated DNS-evidence selector is unchanged.

The report payload and canonical hash keep absence, explicit null, and a value
distinct. Changing any of these under an existing report ID yields the existing
`409 ACK_CONFLICT`; identical canonical content replays the original ACK.
Projection alone maps both absence and null to explicit `serving_read: null`.
It never searches backward for an older non-null diagnostic. JSONB retains JSON
values, not original wire whitespace or object-key order.

The projection is bound to the node and incarnation by a composite foreign key.
When enrollment changes the current incarnation, detail returns null until a
qualifying report from that incarnation updates the selection. An initial report
older than the retained `last_observed_at` does not qualify. The value is stored
on the node, so observation retention cannot leave a dangling selected-report
link. Existing observation and incarnation history are not rewritten or deleted.

HTTP schema validation and typed service validation preserve full decimal-string
uint64 generations, false verdicts, and diagnostics up to 500 Unicode characters.
Persisted values are decoded strictly: missing fields, invalid types or dates,
and malformed objects cause a read error rather than a successful null result.

## Migration and coordinated deployment

Migration `00195_node_serving_read.sql` adds the projection and locks `nodes`
before inspecting legacy observations. For each node it considers **all**
accepted current-incarnation reports at `last_observed_at`. Backfill succeeds
only if their projected values agree; missing and explicit null agree for this
purpose. It does not infer acceptance order from receipt timestamps, UUIDs, or
matching checks. Nodes with no matching observation receive null.

Different tied legacy values stop the migration with an explicit ambiguity
error. The transaction rolls back the schema and version changes. On the old
Central version, accept a new valid current-incarnation observation with an
`observed_at` strictly later than the retained selection, then quiesce writers
again and retry. Do not rewrite immutable reports or their hashes to make a
migration pass. Invalid legacy object shapes also fail the migration. The SQL
shape guard checks UTC timestamp syntax; an impossible calendar date can pass
that guard but fails the strict detail decoder. Investigate that corrupt legacy
record and obtain a fresh qualifying observation before resuming normal use.

Up increments every existing node's `resource_version` once, including nodes
with no observation, because explicit null changes their detail representation.
Down also increments every existing node once before removing the projection.
Both preserve the normal monotonic-version trigger and invalidate cached strong
ETags. Neither direction changes immutable observation payloads or hashes.

Follow the existing [migration-before-start lifecycle](../control/README.md#configuration-and-startup):

1. Stop observation ingress and drain **all** old Central observation writers.
   Keep them stopped through migration and restart; the migration's table lock
   protects its transaction only.
2. Apply migration 00195 using the migration identity and verify success.
3. Start the updated Central processes, then resume observation ingress.

Mixed old/new observation writers are unsupported: an old binary can update
checks without the new projection after migration commits. For rollback, stop
ingress and drain updated Central readers and writers, run Down to 00194, start
the old binaries, and then resume ingress. Running updated readers against the
downgraded schema is unsupported. Retained report payloads survive rollback.

## Local proof and remaining handoff

Implementation was based on `3f960eeb71b670e6938349bdb5aebfe8111d7be8`.
On 2026-11-06, the full node package suite passed with the existing isolated
PostgreSQL test-database fixture:

```sh
CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 -tags integration ./internal/nodes/...
```

Run tests with a unique disk-backed `TMPDIR`/`GOTMPDIR`; integration requires the
test-only admin DSN in `FVPN_TEST_DATABASE_URL`. Fixtures create and drop their
own databases. Do not point them at a production database.

The D-164 tests prove real validated HTTP and typed-service acceptance/rejection,
canonical compatibility with the pre-existing D-126 absent-field fixture,
presence-sensitive ACKs and hashes, full uint64 preservation, equal-time ordering
with reversed IDs and regressed/tied receipt clocks, older reports, concurrent
detail reads, frozen identity, reinstall isolation, and mutation-response ETags.
Real migration tests cover agreeing/ambiguous legacy evidence, malformed data,
null source rejection, cross-node incarnation rejection, immutable row
preservation, failed-Up rollback, and Up/Down version invalidation.

This is the Central phase only. Consumer parsing/display, then producer
activation, then a fresh owner-run acceptance campaign remain in that order.
The local tests do not close F17, establish a new live AT result, or replace
the orchestrator's complete product gates before merge.
