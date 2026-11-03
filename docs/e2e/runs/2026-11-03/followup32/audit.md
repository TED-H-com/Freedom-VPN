# Node conditional-write audit — 2026-11-03

Baseline: `3f7dbc254ba1ce001628af8151a4d81c6954412d` (isolation 19 / PR #434).
Search: `rg -n 'If-Match|IfMatch|if-match|RESOURCE_VERSION_CONFLICT|StatusPreconditionFailed' tests/e2e --glob '*.go'`, plus isolation's copied `driver.go.txt`.
Line numbers below refer to the follow-up 32 source unless marked baseline.
No AT-lane file was edited.

Accepted observations increment `nodes.resource_version` in
`internal/nodes/observations.go:302` and `:404`. They also update operation and
exit versions (`:780`, `:791`, `:809`, `:819`, `:826`, `:1025`). No owned
conditional write targets an operation or exit directly. Public node writes
use the node ETag. A stopped agent reduces the opportunity for this race but
does not make the node version immutable: an already in-flight report or the
central liveness sweep can still advance it.

| Owned site | Write | Can an accepted observation invalidate the read ETag? | Action |
|---|---|---|---|
| `tests/e2e/publish_test.go:88` (baseline 80) | node capacity PATCH | Yes | Shared `writeNode`; all four desired capacity fields must match on re-read. |
| `tests/e2e/publish_test.go:105` (baseline 93) | node publication PUT | Yes | Same helper; published intent and exact requested group membership must match. Continue the existing operation and observed-publication gates. |
| `tests/e2e/followup18_native_test.go:226` (baseline 225) | retained node capacity PATCH | Yes; agent is stopped but in-flight reports/liveness are possible | Same helper; existing first-offer/read-back predicates retained. |
| `tests/e2e/followup27_live_test.go:95` (baseline 94) | populated hold capacity PATCH | Yes | Same helper; original hold/report predicates retained. |
| `tests/e2e/followup30_live_test.go:231` (baseline 225) | survivor hold capacity PATCH | Yes | Same helper; original hold/report predicates retained. |
| `tests/e2e/scenarios_test.go:832` (baseline 832) | node revoke POST | Yes | Same helper; already revoked ends writes, then original lifecycle polling. Exhausted conflict remains blocked. |
| `tests/e2e/exports_live_test.go:353` (baseline 352) | clone reconciliation revoke POST | Yes; primary agent can report | Same helper; already revoked ends writes, then original re-enrollment. |
| `tests/e2e/installer_exports_test.go:274` (baseline 272) | installer node revoke POST | Yes | Same helper using the existing browser session; already revoked satisfies this write. |
| `tests/e2e/followup20_live_test.go:49` | fleet-config PUT | No | Unchanged. Observations do not mutate fleet configuration. |
| `tests/e2e/serving.go:225` | policy activation POST | No | Unchanged. Policy definition version is not the node report version. |
| `tests/e2e/feed_scenario_test.go:68` | policy activation POST | No | Unchanged. |
| `tests/e2e/followup30_live_test.go:285` (baseline 290) | member suspension PATCH | No, for accepted node observations | Unchanged. Member versions can separately change under admin/entitlement work (`internal/entitlement/store.go:124,151`); node report completion changes the member operation, not `members.resource_version`. |

`http_session.go` is an exact extraction of `session`, `reply`, `session.call`
and `errorCode` from the tagged harness so ordinary non-stack stub tests call
the actual HTTP client. Its behavior and every other `session.call` /
`Stack.Call` caller are unchanged. `node_write.go` is the only retry loop added.
Synthetic stub write sites in `node_write_test.go` deliberately induce conflicts;
they are tests of the helper, not further live conditional-write sites.
No owned pre-existing test expected a node 412 as its successful result.

## Contract and key trace

- OpenAPI `IfMatch` (`contracts/openapi.yaml:2788`) requires the last-read strong
  ETag. `ResourceVersionConflict` (`:2990`) instructs re-reading and provides
  `details.current_resource_version`. NodeDetail `usage_outbox` (`:3876`)
  explicitly documents observations advancing the shared node ETag.
- OpenAPI `IdempotencyKey` (`:2762`) and spec §14.5 scope a UUID to principal,
  method, normalized route and payload; same key and payload return the same
  resource or operation. `If-Match` is not part of the validated body.
- `internal/httpapi/commands.go:142-181` hashes the body, reserves the key and
  executes the handler in the same transaction. Handler errors return before
  `CompleteIdempotent`; `internal/db/tx.go:67-84` rolls back. `SetResult` only
  accepts durable 2xx results (`commands.go:65-69`). A 412 is **not stored**.
- Capacity is routed through `internal/nodes/httpwire/handlers.go:240`;
  publication through `internal/nodes/httpwire/state.go:210`. Both use that
  command boundary (`internal/nodes/commandtx/transaction.go:47-58` suppresses
  nested domain key reservations inside the HTTP transaction). Therefore all attempts for one logical command reuse its
  UUID and identical body, changing only `If-Match` after a successful GET.
  No new key is needed. No transport or other-status retry is authorized.

Three writes (initial plus at most two retries) allow two ordinary version races
without hiding persistent contention. One ten-second context covers writes and
re-reads together, shorter than the ordinary 30-second HTTP timeout. There is no
backoff or run-level retry. Every conflict gets one re-read, including the last
allowed write; matching desired values can end that command successfully.
Otherwise the third conflict or the time bound stops as blocked. Failed GETs,
missing/invalid fresh ETags, other 412 codes and every other status stop.

For already-intended publication the harness reads the latest real `node.publish`
operation ID for that node, using its existing read-only DB connection. It then
runs the unchanged operation-success and `publication_observed=active` checks.
No operation is fabricated. An unavailable operation leaves publication blocked.
This distinction is necessary because NodeDetail does not expose an operation ID.

### Requests to orchestrator

2026-11-03: route the following read-only findings to their AT owners. Every
listed node write can race with an accepted observation. These are follow-up
requests, not a change to acceptance predicates or `expected-blocked.json`.

| Owner/site (baseline lines) | Existing behavior / requested owner action |
|---|---|
| isolation `tests/e2e/at/isolation/driver.go.txt:625` | Publication PUT is single-shot. Adopt bounded exact-code conflict handling with intended-value read-back and complete receipts. |
| isolation `driver.go.txt:822,1306,2450,2850,3008` | Node capacity PATCH sites are single-shot. Apply the same rule without relaxing hold/fault assertions. The fleet-config PUT at 1443 is unrelated and unchanged. |
| secops `tests/e2e/at/secops/counted_capacity_test.go:104` | Capacity PATCH is single-shot. |
| secops `followup4_live_test.go:54`, `followup5_live_test.go:56`, `followup7_live_test.go:101` | Capacity PATCH setup sites are single-shot. |
| secops `inputs_test.go:335` | Publication PUT is single-shot. |
| secops `live_test.go:671` | Node revoke POST is single-shot. |
| enrollment `tests/e2e/at/enrollment/freeze_test.go:68` | Closure handles capacity PATCH and publication PUT, single-shot. |
| enrollment `handoff_test.go:427` | Installer publication closure handles capacity PATCH and publication PUT, single-shot. |
| enrollment `live_test.go:888` | Revoke helper has two attempts; it retries transport errors and any 412. Narrow to exact-code conflicts only; retain every attempt, a total deadline, and intended-value checks. |
| members `tests/e2e/at/members/identity_nodes_test.go:173` | Generic node mutation uses three attempts on any 412, new keys, no total bound or complete per-attempt receipts. Covers publication/capacity and node lifecycle callers. Adopt exact-code/read-back/record rules. |
| usage `tests/e2e/at/usage/live_test.go:592` | Generic node mutation uses four attempts on any 412, new keys, no total bound/complete receipts. Same owner action. |
| updates `tests/e2e/at/updates/installer_test.go:411` | Node retirement DELETE is single-shot. Preserve retirement confirmation while handling a genuine version conflict. |

Expected-error cases remain unchanged: secops `live_test.go:550` deliberately
expects a stale member ETag 412; `deadline_live_test.go:30` deliberately expects
a stale fleet-config 412, alongside missing-header 428 and same-key replay
checks. Neither should be made to pass by retrying. Other policy activation,
fleet configuration and member writes in AT directories are not node-observation
version races; their owning lanes retain all behavior.

Isolation 19 itself remains its original single blocked run. After merging the
publisher fix, isolation owns its requested fresh acceptance run and all 12
checks. This lane's serving proof cannot close those checks.
