# AT02 device-to-usage trace — Follow-up 7

Run `297faa31-2f90-4aab-8cf0-6818629f46c8`. See [AT02.json](AT02.json) for statuses, timestamps, exact packet/counter snapshots and operation IDs, and [D077.md](D077.md) for the new restore-fence API matrix.

| Stage | Product path | Deciding evidence |
|---|---|---|
| Identity | Shared IdP browser + native device flow | Same public subject, verified production mapping; no seeded identity proof |
| Final slot | Two barrier-synchronized production `ObserveAndAuthorize` calls | Exactly one committed admission and one limit denial |
| Ownership | Member list/delete and owner-gated service read | Another member's actual device is absent/404 and unchanged |
| Slot reuse | Member DELETE, then former race loser admission | Authorized count 3 → 2 → 3 |
| Revoked ingress | `Stack.ObserveMemberIngress` on `fvmb0` | Revoked frame hits `ctunku`, zero forwarded bytes; admitted sibling hits its own counter and reaches origin |
| Restore fence | Migration 00188 row input, running worker | Member/admin flags present; stable filtered keyset; native HTTPS denied |
| Explicit re-grant | Production admin authorize HTTP route | 202 + operation, fence deletion, worker list `authorized` + explicit `false`, counted HTTPS |
| Re-grant negatives | Production HTTP middleware and handler | Member session 401, CSRF 403, owner mismatch 404, revoked/limit 409; fence preserved on rejected grant |
| Replay | Same key and empty body | Same operation and device resource version; no second grant |
| Usage | Real node mTLS + projected device binding | Unknown device rejected while agent stopped; exact real accepted event replayed twice as duplicate, unchanged receipt/samples |

The D-081 pending device is produced by a real plan downgrade. Only fixture grant retirement uses the production internal `RevokeGrant` transaction because neither an HTTP endpoint nor a shared fixture export exists. Grant creation and every D-077 action use HTTP. No device/entitlement/binding/observation row is fabricated.

All traffic is bounded to the assigned lab. Only the rejected unknown-device input is synthetic; the positive replay uses a real metered event. [Follow-up 6](history/follow-up-6/AT02-TRACE.md) and the [interrupted first](history/follow-up-7/attempt-1/RUN.md) and [second](history/follow-up-7/attempt-2/RUN.md) attempts are historical evidence only.
