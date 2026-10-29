# Adding domain handlers

Implement only the methods your domain owns from
`internal/httpapi/gen.StrictServerInterface`. All request/response models alias
`pkg/api`, so no model conversion is needed. Add one domain constructor line to
`apps/control/handlers.go`'s `Domains()` list, following the
[composition seam](COMPOSITION.md). The domain returns handler sets and any
request middleware; only the shared server calls `httpapi.Compose`/`httpapi.New`. Overlapping methods cause a
startup error. Unimplemented methods keep the transitional INTERNAL_ERROR/501
fallback. Wave 3 domain wiring must remove every fallback; no contract change is
needed for this transition.

A command handler must also implement `httpapi.ReplayHandler`. The strict command
middleware opens `db.InTx`, reserves the key, and invokes the handler only for a
new request. `httpapi.Transaction(ctx)` returns that transaction; do not nest
another transaction or perform an external call. Record the result with
`httpapi.SetResult` before returning a generated success response. Missing result
or a returned error rolls back all local effects. Command handlers without Replay
fail closed. Replay runs after the reservation transaction commits and must read
current persisted state under the original caller's authorization.

This worked adapter is suitable for a domain service that performs a node command
in the caller's transaction. The service checks If-Match under a row lock, writes
intent, creates an operation, and writes audit/outbox rows atomically:

```go
package nodes

import (
    "context"

    "github.com/TED-H-com/Freedom-VPN/internal/auth"
    "github.com/TED-H-com/Freedom-VPN/internal/db"
    "github.com/TED-H-com/Freedom-VPN/internal/httpapi"
    "github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
    "github.com/TED-H-com/Freedom-VPN/internal/operations"
    "github.com/TED-H-com/Freedom-VPN/pkg/api"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

type Service interface {
    Update(context.Context, pgx.Tx, uuid.UUID, api.NodePatch, string) (api.Operation, error)
}
type Handlers struct {
    Service Service
    Operations operations.Store
}
func (h Handlers) UpdateNode(ctx context.Context, req gen.UpdateNodeRequestObject) (gen.UpdateNodeResponseObject, error) {
    tx, err := httpapi.Transaction(ctx)
    if err != nil { return nil, err }
    op, err := h.Service.Update(ctx, tx, req.NodeId, *req.Body, req.Params.IfMatch)
    if err != nil { return nil, err }
    err = httpapi.SetResult(ctx, db.IdempotencyResult{
        ResourceType: "node", ResourceID: &req.NodeId,
        OperationID: &op.OperationId, ResponseStatus: 202,
    })
    if err != nil { return nil, err }
    principal, _ := auth.FromContext(ctx)
    return gen.UpdateNode202JSONResponse{
        OperationAcceptedJSONResponse: operations.Accepted(op, principal.Role),
    }, nil
}
func (h Handlers) Replay(ctx context.Context, operationID string, ref db.IdempotencyRecord) (any, error) {
    principal, _ := auth.FromContext(ctx)
    op, err := h.Operations.Get(ctx, *ref.OperationID, principal)
    if err != nil { return nil, err }
    return gen.UpdateNode202JSONResponse{
        OperationAcceptedJSONResponse: operations.Accepted(op, principal.Role),
    }, nil
}
```

Return `nodes.Handlers{Service: service, Operations: operations.Store{Pool: pool}}`
in the domain build's `Built.Handlers`, and register the domain once in `Domains()`. A multi-command handler switches on the generated
method name passed to Replay (for example `UpdateNode`) and returns that method's
response type. The service must use the session Member ID or verified machine
identity from `auth.FromContext`, never trust IDs supplied by a member. Resource
lookups must include ownership in SQL; absence and another member's row both map
to RESOURCE_NOT_FOUND. Admin/Member role and MFA checks already ran.

## Idempotency and preconditions

Every browser command requires a UUID Idempotency-Key; Agent POSTs may supply one.
Keys are scoped by principal, method and normalized route including canonical UUID
path parameters. Omitting the resource identifier would incorrectly replay another
resource's command. The key is hashed; validated raw JSON is canonicalized through
`pkg/canonjson` and SHA-256. Property order and whitespace do not change the hash;
unknown fields and duplicate JSON keys fail before reservation. Decimal strings
never pass through floating point. A bodyless command hashes `{}`.

The record holds references/status only. `BeginIdempotent` cannot commit alone:
the deferred database guard requires `response_status` and a resource or operation
ID in the same transaction (DB F-06). The middleware calls `CompleteIdempotent`
after the handler supplies `SetResult`; never commit the transaction in a handler.
A completed HTTP command may reference an operation that is still pending. A committed pending operation is the
same operation on every retry, with its current status re-read. Concurrent first
uses serialize through the database uniqueness constraint. Different payloads
return 409; expired keys can create a new command. Retention defaults to seven
days and is configurable; durable domain uniqueness survives key expiry.

Enrollment Replay must recheck the proof/credential binding and return the same
enrollment with `secret_available=false`, `token=null`, and no reconstructed
installer secret. The framework additionally strips the token and forces the flag
false on enrollment replays. The original token must never enter the idempotency record.
Other replay handlers likewise re-read resources and recheck ownership. These are
domain hooks because the framework cannot know resource semantics or reproduce a
one-time secret safely. Do not cache serialized response bodies.

`httpapi.ETag(resourceVersion)` creates a strong decimal-string ETag.
`CheckIfMatch(header, current)` returns 428 when absent and 412 with
`details.current_resource_version` when stale, weak or wildcard. Call it inside
the mutation transaction under a row lock, or combine it with `db.UpdateWithVersion`.
Every UPDATE of a versioned domain resource must strictly increase its version,
even metadata-only updates (DB F-14). The route boundary checks required header presence; only the domain can compare
the current resource version. `NotModified` implements weak/list/wildcard
If-None-Match matching for safe GETs. Agent signed-envelope ETags must cover the
whole envelope, including renewed signatures/timestamps (§14.7); do not use only
its generation. A 304 never extends a lease.

Schema-expressible rules are enforced at the boundary. Domain semantic validation,
for example clear_capacity versus capacity or policy tier ordering, belongs to
the owning service before its transactional writes. Return safe apierr values;
never expose SQL, driver or provider errors in error messages/details.

Agent handlers without Idempotency-Key own their normal domain transaction. When
an agent key is supplied, reuse `Transaction(ctx)` instead; do not open an inner
transaction. The same service can accept a transaction parameter from either path.

### Agent receipt exception (D-029 follow-up)

`SetResult` normally requires an `OperationID` for every 202 operator command.
The only receipt exception is an authenticated Agent submitting
`POST /api/v1/agent/observations` or `POST /api/v1/agent/update-results`. These are
called `submitNodeObservation` / `submitNodeUpdateResult` in the wiring task;
the current generated strict methods are `SubmitObservation` / `SubmitUpdateResult`
(OpenAPI operation IDs `submitObservation` / `submitUpdateResult`). For these two
methods, a keyed command may set `ResponseStatus: 202` with `OperationID: nil` and
a nonnil `ResourceID` pointing to the durable observation/result receipt. Store
it in `Transaction(ctx)` and implement `Replay` to re-read that receipt under the
certificate identity, returning the same generated 202 receipt response. The
middleware commits the reference and response status together and never invents
an operation. All other 202 commands, including any operator principal, retain
the operation requirement. Unkeyed Agent submissions use the domain transaction
and return their generated receipt directly; `SetResult` requires a keyed command
context. This exception does not authorize submission through the public listener.
