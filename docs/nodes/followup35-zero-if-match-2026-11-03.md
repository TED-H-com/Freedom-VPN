# Follow-up 35 — zero If-Match precondition (2026-11-03)

Status: parser change stopped by the task's explicit precondition. No runtime,
contract, contract-version, schema or existing test behavior is changed.

## Finding and provisional ruling

Devstack follow-up 32, F19-02, found that `If-Match: "0"` returns HTTP 412
`RESOURCE_VERSION_CONFLICT` without `details.current_resource_version`.
`nodes.ParseIfMatch` rejects zero before the resource lookup. The existing
[OpenAPI contract](../../contracts/openapi.yaml) accepts zero in the `IfMatch`
pattern and specifies the current version for a mismatch. See the
[consumer evidence](../e2e/runs/2026-11-03/followup32/resume.md) and
[response record](../e2e/RESPONSES.md).

The orchestrator's provisional ruling was to treat zero as an ordinary version
that matches no resource, return the ordinary conflict detail, and leave the
contract and its version unchanged. Before changing the parser, the task
requires proving that no resource guarded by this parser can have version zero;
if one can, stop the implementation and report it. This is an API01 concurrency
issue under Project Spec §14.5, §14.6 and §15.1.

## Precondition evidence

All six production call sites ultimately compare **nodes.resource_version**.
Publication also mutates `exits`, but compares the node ETag.

- Migration `00002_nodes.sql` defines both `nodes.resource_version` and
  `exits.resource_version` as `NUMERIC(20,0) NOT NULL DEFAULT 1`, with
  `CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615)`.
  The default is positive; the constraint explicitly permits zero.
- Migration `00004_guards.sql` installs `monotonic_fields('resource_version')`
  for both tables on **UPDATE** only. It prevents decreasing an existing
  positive version to zero, but does not reject an INSERT starting at zero.
  No subsequent migration tightens these columns or adds such an INSERT guard.
- The repository's production INSERT paths are
  `internal/enrollment/service.go`, `CreateEnrollment`: `INSERT INTO
  nodes(id,notes)` and `INSERT INTO exits(id,node_id,display_name,region)`.
  Both omit the version and therefore start at 1. No production explicit-zero
  INSERT, COPY path, or reset to zero was found. This limits ordinary application
  reachability, but does not establish the required database invariant.
- `internal/db/version.go`, `UpdateWithVersion`, accepts the decimal string
  `"0"` and uses it in the ordinary `WHERE resource_version=$2::numeric`
  comparison. If a node was inserted at zero, accepting the header would allow
  that comparison to succeed and advance the node to 1.

An explicit zero insert is therefore a counterexample to the precondition,
not evidence that a deployed node currently has that version. No shared or
production resource data was inspected.

An isolated `dbtest.New` database with all current migrations confirmed this
on 2026-11-03. The local diagnostic `TestAPI01_ZeroVersionPreconditionProbe`
passed in 1.07 seconds:

```text
Current migrated schema accepted explicit inserts: nodes=0 exits=0
GetNode exposes resource_version="0" and ETag="0" for the inserted node
UpdateWithVersion(expected=0) matched the node and advanced its version to 1;
probe transaction rolled back
```

The probe used explicit `INSERT INTO nodes(id,resource_version) VALUES($1,0)`
and the corresponding exit insert. It exercised the existing `GetNode` and
`db.UpdateWithVersion` functions without changing `ParseIfMatch`. The update
was rolled back and dbtest dropped the isolated database without a cleanup
error. This diagnostic proves the precondition failure; it is not the deferred
route-level regression suite or evidence that a zero-header HTTP write
currently succeeds.

## Conditional routes and all parser callers

| HTTP route | Parser caller and comparison | Follow-up disposition |
| --- | --- | --- |
| `PATCH /api/v1/admin/nodes/{node_id}` | `nodes.UpdateNode` uses `db.UpdateWithVersion` on the node; conflicts use `versionConflictError`. | Audited; implementation and zero-conflict regression test deferred. |
| `POST /api/v1/admin/nodes/{node_id}/revoke` | `httpwire.Handlers.RevokeNode` parses the header and passes the version to `enrollment.revokeNode`, which compares the locked node row. The alternate `nodes.RevokeNode` path parses in `nodes.revokeOrRetireNode` and uses `UpdateWithVersion`. | Audited; implementation and zero-conflict regression test deferred. |
| `DELETE /api/v1/admin/nodes/{node_id}` | `httpwire.Handlers.RetireNode` likewise delegates the comparison to `enrollment.revokeNode`; alternate `nodes.RetireNode` uses `nodes.revokeOrRetireNode`. | Audited; implementation and zero-conflict regression test deferred. |
| `PUT /api/v1/admin/nodes/{node_id}/publication` | `httpwire.Handlers.SetNodePublication` calls `exits.SetPublicationTx`, which parses and compares the locked node row. | Audited; implementation and zero-conflict regression test deferred. |

The sixth call site, `exits.SetPublication`, initially parses only for
validation. For a new idempotency key it delegates to `SetPublicationTx`, which
parses again and performs the comparison. For an already accepted key it
returns the stored operation and current ETag, following the existing replay
path. Currently zero is rejected at the initial parse, with empty details. No
validation-only bypass or additional HTTP route was found. Its zero handling
remains deferred with the shared parser; the existing contract discrepancy
remains open rather than being described as fixed.

## Consumer audit

`tests/e2e/node_write.go` records a decimal `current_resource_version` when
provided and always re-reads after an exact 412 `RESOURCE_VERSION_CONFLICT`.
`tests/e2e/publish_test.go` injects `"0"` on the first capacity and publication
attempt in the forced proof. `tests/e2e/publisher_proof_test.go` explicitly
accepts both empty details and a supplied version, with
`TestAPI01_PublisherProofOptionalConflictVersion` covering both. No consumer
requiring empty details was found. Its comment currently describes the
zero-header rejection; devstack owns updating that comment if the behavior
changes. No consumer files or historical evidence were edited.

## Requests to orchestrator

- **2026-11-03 — Zero-version invariant blocks the parser change.**
  `nodes.resource_version` permits an initial zero, which is the comparison
  target of every guarded route. `exits.resource_version` also permits zero.
  The isolated migrated-database probe above confirmed both the insert and
  successful ordinary version comparison.
  Decide whether to establish a strictly positive node-version invariant with
  an authorized forward migration and a policy for any existing zero rows, or
  revise the provisional premise to allow zero to match an actual zero-version
  resource. The schema owner must decide; this task does not authorize a new
  migration or a changed response contract. After resolution, implement the
  parser change and all four route-level no-write/same-key retry tests.

## 2026-11-03 — Resume 1: revised ruling and implementation

The orchestrator withdrew the no-zero precondition. Project Spec §14.5 and the
existing OpenAPI contract assign zero no special meaning and impose no minimum
stored version. A client that reads ETag `"0"` may send `If-Match: "0"` and write
that same version. Against a positive current version, `"0"` is an ordinary
mismatch: HTTP 412 `RESOURCE_VERSION_CONFLICT`, with the current decimal-string
version in `details.current_resource_version`. The earlier counterexample and
stop record above remain accurate history; this revised ruling resolves their
blocking request. No positive-version invariant is introduced.

`nodes.ParseIfMatch` now accepts zero through its existing decimal parser. All
six audited callers retain their existing comparisons and transaction paths.
Missing/empty headers still produce 428; other malformed values retain their
previous parser responses. Non-zero comparisons, the schema, production's
default version of 1, the contract and its version are unchanged.

For a fresh command, `exits.SetPublication` now accepts `"0"` at its initial
validation and delegates to `SetPublicationTx` for the locked node comparison.
A zero mismatch returns the current version and rolls back its idempotency
reservation. As with other well-formed versions, an already accepted key/body
returns the original operation through the existing idempotent replay path.
The alternate node service revoke/retire entry points likewise use their
ordinary conditional updates.

Added coverage under API01 (Project Spec §14.5, §14.6):

- `TestAPI01_ParseIfMatchZero` accepts `"0"` and preserves surrounding whitespace
  handling. `TestParseIfMatchAndFormatETag` retains every other existing bad
  value and the empty-header 428 check, and adds weak/list tags, leading zeros,
  unquoted/empty values, non-integers and overflow as malformed regressions.
- `TestAPI01_HTTPZeroIfMatchConflictAndSameKeyRetry` covers
  `PATCH /api/v1/admin/nodes/{node_id}`,
  `POST /api/v1/admin/nodes/{node_id}/revoke`,
  `DELETE /api/v1/admin/nodes/{node_id}`, and
  `PUT /api/v1/admin/nodes/{node_id}/publication` through the real HTTP command
  boundary. Each positive-version resource rejects both zero and a non-zero
  stale version with the exact current version. Complete node/exit/identity
  row snapshots and operation/audit/outbox/idempotency counts stay unchanged;
  credentials remain usable. Changing only the header to the correct ETag
  accepts the same key/body once, with one operation, audit, outbox event and
  idempotency record and the intended state change.
- `TestAPI01_HTTPIfMatchZeroRowMatches` inserts a test node at zero, reads its
  ETag and PATCHes notes using `"0"`. The response is 200, the version becomes
  1, and replay performs no additional write. A version-1 case runs the same
  checks and advances to 2. These explicit inserts are TEST ONLY fixtures.
- `TestAPI01_StandaloneZeroIfMatchConflictAndSameKeyRetry` covers the alternate
  node revoke/retire service entry points and the `exits.SetPublication` wrapper:
  zero conflict details, no durable effects, same-key retry and accepted-key
  replay. Together with the HTTP tests this exercises all six parser callers.

Consumer behavior remains compatible: devstack re-reads and retries with the
same key/body, and its proof accepts either empty or populated conflict details.
No consumer requiring empty details was found. No files under `tests/e2e/`
changed. Its proof-helper comment still describes the old zero rejection;
devstack owns that comment update.

### Resume 1 request to orchestrator

- **2026-11-03 — Devstack documentation follow-up (nonblocking).** Update the
  comment above `publisherTrace32` in `tests/e2e/publisher_proof_test.go`, which
  says zero is rejected before lookup. The existing test intentionally covers
  both error-detail shapes and needs no behavioral change to consume this fix.
  Historical devstack evidence remains unchanged.
