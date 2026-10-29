# Requests to orchestrator

All live cases are blocked by B1. B2–B7 are source-inspection findings and
requirements for the next run, not runtime defects reproduced on this branch.
No application, lab, contract or base migration was changed.

## B1 — assigned port base rejected (all assigned ATs)

Owner: devstack task. `tests/e2e/cmd/fvpn-e2e-prepare/layout.go`, `parseLayout`;
`mk/e2e.mk`; orchestrator's stack assignment.

Reproduce with exactly:

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=19000 make -f mk/e2e.mk e2e-up
```

Observed: exit 2, `port base must be 18000 plus a multiple of 6000`. Preparation
exits before `run`, secret generation, image builds and compose startup. The
accepted sequence is documented in `docs/e2e/README.md`; 19000 is outside it.
This is **needs devstack follow-up 2**, despite both variable names being present.
The acceptance task did not select another slot. Owner must reconcile the
binding assignment with collision-free ports/subnets for concurrent stacks.

Expected: the explicitly assigned prefix/base prepares its own state and starts
without overlapping another task. Then rerun the fresh stack and the live ATs.

## B2 — member fixture and usage record cannot establish device ownership (AT01–AT04)

Owners: devstack (`tests/e2e/scenarios_test.go`, `memberNetBird`,
`replayUsageBatch`; `deploy/devstack/entrypoint-member.sh`), with members owner
for the verified cross-client proof contract. Details and executable read-only
queries: [AT02-TRACE.md](AT02-TRACE.md).

`memberNetBird` creates a setup key; member entrypoint uses `--setup-key-file`.
Discovery is for native user peers with a verified identity mapping. A connected
machine peer does not imply a `member_devices` or `device_bindings` row.
Independently, `replayUsageBatch` uses `DeviceId: uuid.New()` with binding revision
1; the usage lookup has no corresponding row. This explains the earlier
`binding_not_found` evidence without weakening usage validation.

Expected devstack follow-up 3: create the member via the second real IdP client,
verify its cross-client mapping, let the normal discovery/admission path create
device/binding rows, and wait for external readback and node ACK. Select the
actual `(device_id, binding_revision)` and an interval contained by the binding
when uploading usage. Do not insert a random binding merely to satisfy ingest.
If follow-up 3 uses a TEST ONLY binding seed, label it explicitly and keep AT01
cross-client/native-client coverage blocked. Do not make `node.peer.observe`
authorize a member: it handles node identity, not member discovery.

## B3 — second OAuth client and admission-race hook absent (AT01, AT02)

Owner: devstack. `tests/e2e/cmd/fvpn-e2e-idp/main.go` (`server`, authorize/token
handlers) stores a single `clientID`/redirect and rejects other clients.
`fvpn-e2e-prepare/main.go` configures `portal-e2e`; the member uses a setup key.

Expected: two registered clients with distinct audiences, authenticated
native-member login, independently verified stable-claim linkage, and at least
three members. Provide a documented test hook to drive two
`devices.Service.ObserveAndAuthorize` calls behind a goroutine start barrier
using real observed peers. There is no public device-authorize HTTP endpoint;
do not invent one. Discovery's sequential loop alone is not a simultaneous
last-slot test. Expose a supported session/DB/NetBird runner extension seam;
`lab`, `openLab`, `runner`, `login`, `call` are currently package-private.

## B4 — provider has no authoritative invoice/refund fixtures (AT03, AT04)

Owner: devstack payment fixture. `tests/e2e/cmd/fvpn-e2e-payment/main.go`,
`server.route`, returns empty lists for subscriptions/invoices/refunds and 404
for unsupported details. It supports checkout and portal creation only.

Expected: authenticated TEST ONLY hooks for authoritative subscription, invoice,
line-item and refund snapshots, deterministic event bytes/signing, replay and
ordering, and cancellation/refund transitions. The real billing adapter must
read this state. Injecting webhook payloads alone cannot prove reconciliation.
No public provider or third-party payment traffic is authorized for this test.

## B5 — serving node and binding follow-ups pending (AT01–AT04, AT08–AT21)

Owners: wire-nodes follow-up 5 (`apps/worker/jobs_nodes.go`, `NodeJobs`) and
devstack follow-up 3 (`tests/e2e/scenarios_test.go`, enrollment/member setup).
On this revision `NodeJobs` has no `node.publication_changed` registration;
the lab runner never submits node publication. The orchestrator already owns
these fixes; this task did not reimplement them.

Expected: publish through `PUT /api/v1/admin/nodes/{node_id}/publication`, retain
202/pending until route readback, matching node generation/ACK and actual
forwarding are observed. Supply a controlled echo and sustained TCP/UDP endpoint
reachable only through the member path. A baseline drop chain, `fvnb0` accepts,
healthy heartbeat or NetBird `Connected` is not positive forwarding evidence.

## B6 — topology and platform hooks missing (AT05, AT09–AT21)

Owner: devstack (`deploy/devstack/compose.yaml`, prepare and entrypoints).
The current compose has one node and one AdGuard Home with CoreDNS upstream;
CoreDNS is not a second filtering resolver with the same profile. Member
entrypoint has `--disable-dns --disable-ipv6`. Prior gateway logs report relay
WebSocket 404; this run did not reach a gateway.

Expected documented hooks: three join-source node variants, two independent
exits, two controlled filtering resolvers, normal member DNS, network-change
and resolver/policy-sync faults, controlled A/AAAA/CNAME/HTTPS/SVCB/rebinding
zones, TCP/UDP/QUIC/PMTU fixtures and a functioning forced-relay path. Missing
physical arm64, real clients and physical second NAT remain operator work.

## B7 — candidate privacy/feed/retention fixture not supplied (AT22)

Owners: devstack and DNS/data operators. `docs/dns/FEEDS.md` explicitly leaves
commercial feed approval pending and `docs/dns/PUBLISH.md` says publication pins
feed metadata but does not fetch/render feed bodies. `SourceRegistry.Ingest`
last-good unit coverage is not proof of the running worker's update pipeline.

Expected: candidate source/license inventory, controlled feed transport hook,
observable failed refresh preserving the previous hash, generated browsing
sentinels, metrics/log inventories and an isolated backup/retention fixture.
Only the approved `purge_retained` targets may be exercised. Financial retention
must stay a required decision. Do not invent a production license approval,
refund rule, grace period or DNS TTL to make acceptance green.
