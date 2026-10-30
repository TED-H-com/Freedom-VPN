# Requests to orchestrator

These requests refer to observed live prerequisites in the per-AT records.
Application, lab, contract and migration files remain unchanged by this task.
The legal slot `w4-members / 24000` starts successfully. The earlier slot
assignment error is resolved; it is not a devstack defect. Publication worker
follow-up 5 is included in the tested source.

## B2 — native member admission and usage binding (AT01–AT04)

Owners: devstack (`tests/e2e/scenarios_test.go: memberNetBird, replayUsageBatch`,
`deploy/devstack/entrypoint-member.sh`); members owner for identity verification.
See [AT02-TRACE.md](AT02-TRACE.md) and [AT02.json](AT02.json).

Reproduction: fresh assigned stack, Portal login A, finite grant, documented
member setup-key handoff, NetBird `/api/peers` readback, then two simultaneous
`devices.Service.ObserveAndAuthorize` calls using the real `NetBirdAdapter`.
The setup-key peer has no `user_id`; verified NetBird mappings, member devices
and device bindings remain absent. A real-node mTLS usage event with an explicitly
unknown device reproduces `VALIDATION_FAILED / binding_not_found`.

This is the correct rejection, not an ingest defect. `node.peer.observe` in
`apps/worker/jobs_nodes.go` updates **node** incarnation `deployment_id/peer_id`.
It must not authorize member peers. `devices.discover` in
`apps/worker/jobs_members.go` skips machine peers and unverified users; verified
admission in `internal/devices/service.go` creates `member_devices` and finite
`device_bindings` together under the member lock.

Expected devstack follow-up 3: native member SSO, trusted cross-client linkage,
normal discovery/admission, paid-group readback and node ACK. Upload with the
actual device ID/revision and an interval entirely inside the binding. The
current shared `replayUsageBatch` generates a random device ID and a five-minute
lookback; even a corrected ID can fail `interval_outside_binding_window`.
Never insert a fake binding just to make usage green. If a TEST ONLY identity
seed is offered as a limited workaround, label it and keep native mapping AT01
blocked. Two verified candidate peers and B's existing device are still required
for the true final-slot race, cross-owner delete, revoke-slot reuse and gateway
winner/loser positive controls.

## B3 — second OAuth client (AT01, AT02)

Owner: devstack. `netbird-server` 0.82.0 exposes two static Dex clients and
rejects a third registration. The portal and the CLI share
`https://netbird.example.invalid/oauth2`. The portal client is
`netbird-dashboard`; the CLI client is `netbird-cli`. The public `sub` is the
same for both and is the NetBird user id. See `docs/e2e/README.md`. A probe of
the test IdP with `client_id=netbird-native` is not the portal path.

Stable-claim proof across those two clients, plus multiple real user peers,
is still required for AT01. `e2e-member` and `e2e-member-b` exist after
bootstrap. The acceptance run admits one member container. Document a reusable
session/NetBird/DB setup seam; the shared runner's `lab`, `runner`, `login`
and `call` are private. No new public device-authorize HTTP endpoint is
requested.

## B4 — authoritative payment and Portal fixtures (AT03, AT04)

Owner: devstack payment fixture,
`tests/e2e/cmd/fvpn-e2e-payment/main.go: server.route`.

Reproduction: send the same correctly signed invoice event ten times and all
six created/paid/failed orders. The real inbox deduplicates; bad signatures
receive 401 and create no inbox row. However, the controlled provider's
`GET /v1/subscriptions/{id}` returns 404, and subscription/invoice/refund list
endpoints contain no records. No successful paid reconciliation is possible.
The control origin also returns 404 for `/billing/return`.

Provide authenticated TEST ONLY snapshot/event hooks for subscription, invoices,
line items, cancellation and full/partial refunds. The billing adapter must
read authoritative provider state; webhook payloads are not payment truth.
Serve the built Portal through a documented stack hook to test return-page and
member-screen behavior. Current finite grants, real clock expiry and persistent
manual suspension are partial evidence; they do not prove successful-payment
precedence or flow termination on every node.

## B5 — valid serving baseline and publication application (AT01–AT05, AT08, AT09)

Owners: devstack follow-up 3 (`tests/e2e/harness.go: insertReleaseSupport`,
`tests/e2e/scenarios_test.go` setup); nodes owner if readiness still fails after
valid release/config fixtures. Publication follow-up 5 is already present.

Reproduction: enroll the actual installer node, classify it to group A and
call `PUT /admin/nodes/{id}/publication` with the current ETag. The running API
rejects publication with `422 NODE_NOT_SERVICEABLE`; NetBird has no exit route
and no successful member forwarding baseline exists. Raw forward-chain accept
rules include NetBird control traffic; their count is not member authorization. Disable is accepted and its intent
survives a node-container restart, while external application remains pending.

The lab has no active release-support policy or installed candidate serving
release. The API correctly rejects an empty `serving_release_ids` array (422).
The shared runner currently bypasses that contract with a direct `{}` policy
insert. Supply a valid signed candidate release and support policy through the
real interfaces, plus required isolation/DNS/meter health and member binding.
Do not weaken readiness or treat 202, NetBird Connected, an online heartbeat,
or a drop-only chain as proof of serving. Preserve pending versus observed
operation state and include real controlled TCP/UDP forwarding readback.

## B6 — DNS application and missing topology (AT05, AT09, AT20, AT21)

Owners: devstack (`deploy/devstack/compose.yaml`, prepare, node/member
entrypoints), node-agent/DNS wiring (`node/agent`, `node/dns`).

Reproduction: create canonical rules, create/activate a DNS profile, inspect
`dns_resolver_documents` and real AdGuard `/control/filtering/status`, and query
the actual resolver. Activation is accepted but `user_rules` stays empty; the
requested blocked controlled name still returns A/NOERROR after bounded
convergence. The policy operation reports desired_state_compile succeeded;
that central operation is not a resolver ACK. Provide the
node-agent document/application hook and authenticated resolver targets from
`docs/dns/PUBLISH.md`; require readback, synthetic probe and matching ACK.

Observed topology: one enrolled exit, one filtering AdGuard resolver, one
CoreDNS upstream; member starts with `--disable-dns --disable-ipv6`. Pausing
either resolver/upstream causes timeouts and unpause restores responses; this
is not two-filtering-resolver failover. There is no successful exit baseline.

Provide documented hooks for three boot-source flavours, two exits, two
filtering resolvers, native DNS, network move and policy-sync faults, controlled
A/AAAA/CNAME/HTTPS/SVCB/rebinding zones, long TCP/UDP flows, QUIC/PMTU and forced
relay/direct baselines. CoreDNS's root NXDOMAIN template keeps all test names
inside the lab. Physical arm64, platforms and second physical NAT remain
operator work, with scripts in OPERATORS.md. Prior-run relay errors are not
claimed as reproduced here.

## B7 — privacy inventory, feeds and retention coverage (AT22)

Owners: devstack, DNS and retention operators. Live checks show disabled query
logging/statistics, no DNS sentinel in inspected service logs/API/audit/database
dump, and correct guarded audit purge of old versus young synthetic rows.
There are zero active threat-feed definitions. No licensed feed refresh can be
faulted, and no successful member URL-flow privacy baseline exists.

Supply approved source/license inventory, controlled feed transport and a
last-good failure hook. `docs/dns/PUBLISH.md` explicitly says feed metadata is
pinned but feed domain bodies are not fetched/rendered. Component tests do not
prove the running refresh pipeline. Provide retained-file/backups inventory,
raw-usage/rollup purge fixtures and candidate log retention behavior. Keep
financial retention and commercial feed approval explicit business decisions.

## B8 — configured metrics endpoint is unavailable (AT22)

Owner: control composition/observability (`apps/control/server.go`,
`internal/config.Config.MetricsAddress`), with devstack configuration owner.
Reproduction: execute the bounded HTTP probe inside the running control
container against its configured loopback `127.0.0.1:9091/metrics`; it fails.
`MetricsAddress` is configured by prepare, but no control listener consumes it
in this source. Wire the intended endpoint or document the supported metrics
surface, then inspect actual metrics for synthetic qnames/URLs. An unavailable
endpoint is not privacy evidence.
