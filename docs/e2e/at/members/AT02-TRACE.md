# AT02 device-to-usage binding trace

Status: **blocked** for live evidence. The assigned stack fails before startup.
This is a source trace at the source commit in AT02.json, not a new packet/DB
observation. Earlier [usage evidence](../../runs/2026-10-29/usage.json) records
`rejected VALIDATION_FAILED binding_not_found`, total 0. Do not label that
historical result as a new run.

| Stage | Current code | What it establishes |
|---|---|---|
| Exit peer observation | `apps/worker/jobs_nodes.go: NodeJobs`, `node.peer.observe` | Reads node observation, calls `EnsureNodePeer`, updates `node_incarnations.deployment_id/peer_id`, marks `peer_readback`. Creates no member device/binding. |
| Member joins lab | `tests/e2e/scenarios_test.go: memberNetBird`; `deploy/devstack/entrypoint-member.sh` | Setup-key peer in staging. `netbird_up` is connectivity, not verified member ownership. |
| Discovery | `apps/worker/jobs_members.go`, `devices.discover` → `internal/devices/netbird_adapter.go: Discover` | Skips blank `UserID`, ephemeral peers and missing `netbird_user_bindings.mapping_proof`; existing devices reconcile separately. |
| Verified admission | `internal/devices/service.go: ObserveAndAuthorize`, `AuthorizeDevice` | Re-observes peer; requires cross-client proof, valid finite entitlement and final-slot lock. Atomically inserts `member_devices` and finite `device_bindings`, operation, audit, outbox. |
| External application | `internal/devices/worker.go: SyncDevice` | Re-observes identity/IP, applies owned memberships, reads back and fences by resource/entitlement revision. `nodes` step still needs real ACK. |
| Lab upload | `tests/e2e/scenarios_test.go: replayUsageBatch` | Generates a new unrelated device UUID; hard-codes binding revision 1 and a five-minute lookback. |
| Ingest guard | `internal/usage/ingest.go: validateEventBindingsInTx` | Looks up device and revision in `device_bindings`; missing row gives `binding_not_found`. Also checks member equality and entire half-open event interval. |

There are two independent fixture gaps: no verified member admission is driven,
and the upload never selects the actual binding even if one exists. Fixing the
member path alone leaves the random device UUID invalid. Fixing that UUID alone
can still produce `interval_outside_binding_window` if a newly authorized
binding is newer than the synthetic five-minute interval.

After B1/B2 are fixed, the runner should execute these read-only probes through
its own stack DB connection. Pass `peer` from NetBird readback (never a guessed
name or overlay IP); `deployment` from this stack's members settings. The SQL
below reports only counts/booleans, not identities or secrets:

```sql
BEGIN TRANSACTION READ ONLY;
SELECT count(*) AS node_peer_bindings
FROM node_incarnations WHERE deployment_id=:'deployment' AND peer_id=:'peer';
SELECT count(*) AS member_devices,
       count(*) FILTER (WHERE status='authorized') AS authorized_devices
FROM member_devices WHERE deployment_id=:'deployment' AND peer_id=:'peer';
SELECT count(*) AS verified_member_mappings
FROM netbird_user_bindings n
JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id
WHERE n.deployment_id=:'deployment' AND n.external_user_id=:'user'
  AND n.mapping_proof<>'' AND i.stable_claim_ref<>'' AND i.binding_method<>''
  AND n.verified_at<=clock_timestamp() AND i.verified_at<=clock_timestamp();
SELECT count(*) AS finite_device_bindings
FROM device_bindings b JOIN member_devices d ON d.id=b.device_id
WHERE d.deployment_id=:'deployment' AND d.peer_id=:'peer'
  AND isfinite(b.valid_from) AND isfinite(b.valid_until)
  AND b.valid_until>b.valid_from;
SELECT count(*) AS matching_usage_bindings
FROM device_bindings
WHERE device_id=:'device_id'::uuid AND binding_revision=:'binding_revision'::numeric
  AND member_id=:'member_id'::uuid
  AND valid_from<=:'interval_start'::timestamptz
  AND valid_until>=:'interval_end'::timestamptz;
COMMIT;
```

Use only `docker exec -i fvpn-w4-members-postgres psql -U fvpn -d fvpn` on this
task's stack, with `-v ON_ERROR_STOP=1` and those locally obtained variables.
Never use the shared admin DSN to run these lab probes. Record the observed
NetBird peer's `UserID`-present boolean, device/binding counts, operation step
statuses and rejected field/reason. Publish pseudonyms for correlating peers.

Expected admitted member: node-peer count 0, verified mapping 1, authorized
device 1, matching finite usage binding 1. Node-peer count 1 only proves the
exit path. The quota loser must have no authorized device and must fail actual
gateway traffic. A setup-key peer with blank user must remain unadmitted.

Owner fix: [B2/B3](BLOCKERS.md). Existing lower-level regression commands (not
substitutes for live AT02) are:

```sh
# Private disk TMPDIR/GOTMPDIR; dbtest creates and drops its own database.
go test -tags integration -count=1 -p 1 -parallel 2 -v ./internal/devices \
  -run 'TestAT01_VerifiedCrossClientMappingAndReadBack|TestAT02_LastSlotRaceIsolationAndRevoke'
```
