# AT02 device-to-usage binding trace — Follow-up 3

The native admission gap is resolved in this run. [AT01](AT01.json) proves the
shared issuer mapping; [AT02](AT02.json) proves the quota transaction, ownership
gates, slot reuse and receipt validation. Complete AT02 remains **blocked** after revocation by missing lab conntrack
(F3-02). A and the final-slot winner have real counted forwarding; the loser
has no selected exit and no origin response. Database authorization alone is
not forwarding.

| Stage | Product path | Live evidence |
|---|---|---|
| Two OAuth clients | Shared `tests/e2e/internal/idplogin`; embedded Dex `netbird-dashboard` and `netbird-cli` | Portal login and real CLI device flow compare equal public subjects in memory. No token/subject is published. |
| Identity discovery | `apps/worker/jobs_members.go: memberJob`, `devices.discover` → `internal/devices/netbird_adapter.go: Discover` → `identity.go: bindUser` | Production `netbird_user_bindings` writer creates `shared_issuer_public_subject:<issuer>` proof. No email linkage or seeded proof. |
| Member admission | `internal/devices/service.go: ObserveAndAuthorize`, `AuthorizeDevice` | Re-observes actual native peer; verifies mapping and finite entitlement; locks member; commits device, binding, operation, audit and outbox together. |
| Last-slot race | Two real contenders in reused lab member images; own worker paused only during the barrier | A has two of three devices. Two goroutines start together: one commits, one returns `ErrLimit`; count becomes three. Loser has no device row or paid group. |
| Cross-owner access | HTTP list/delete and `devices.Service.Get` | A's list omits B's existing device; deleting it returns 404 without identifiers; service read is not found with an empty result. B remains authorized. |
| Slot reuse | Member DELETE, then real-adapter admission | Count changes 3 → 2 → 3; former loser is admitted. No projection is hand-written. |
| External authorization | `internal/devices/worker.go: SyncDevice` | Worker resumes and observations are read separately. Desired node generation advances, and desired/applied generations are recorded with current-generation leases; post-revoke apply failure is separate. |
| Node identity | `apps/worker/jobs_nodes.go`, `node.peer.observe` | This node identity path is separate from the native member identity/device writer; never substitute a node peer for a member device. |
| Usage negative control | Actual node mTLS, deliberately unknown device | Per-event `VALIDATION_FAILED / binding_not_found`. This remains the correct rejection. |
| Usage positive control | Actual projected device/revision and interval wholly inside its binding | `accepted`, then `duplicate`; exactly one receipt and one sample. Synthetic bytes test ingest, not metering. |

The primary runner reuses the lab's shared `StartNativePeer` hook, immutable running-node image and `user-auth`
entrypoint for A1, A2, A3, A4 and B1. Extra clients have only the member image's
NET_ADMIN/NET_RAW capabilities and tun device, no host ports or new network.
All names use `fvpn-w4-members-*`; cleanup removes every extra client.

Read-only trace commands run against this task's isolated lab database:

```sql
SELECT count(*) FROM netbird_user_bindings n
JOIN identity_bindings i ON i.id=n.identity_binding_id
WHERE n.external_user_id=i.subject
  AND n.mapping_proof='shared_issuer_public_subject:'||i.issuer;
SELECT status,count(*) FROM member_devices GROUP BY status;
SELECT count(*) FROM device_bindings
WHERE valid_from<=clock_timestamp() AND valid_until>clock_timestamp();
SELECT desired_config_generation,applied_config_generation FROM nodes
WHERE current_incarnation_id IS NOT NULL;
SELECT count(*) FROM usage_events;
SELECT count(*) FROM usage_samples;
```

Use `docker exec fvpn-w4-members-postgres psql -U fvpn -d fvpn -X` only.
Never migrate or query the shared admin database for these checks. The
positive/negative packet path is in [forwarding-observation.md](forwarding-observation.md).
Post-revoke recovery awaits [F3-02](requests.md); original transaction and usage
checks remain intact.
