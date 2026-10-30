# Requests to orchestrator — Follow-up 1

Observed on fresh `w4-members / 24000`, product source
`629766c18d1bdd63a336972f95ae2b41734a242b` (full binary VCS revisions and image digests are in each AT JSON).
No application, contract, lab or migration files were changed. D-050 projections
were never hand-written.

## Resolved B2/B3 — native admission, identity and usage

The shared Dex login helper and D-052 writer now work live. Portal and CLI map
to the same subject; real native devices project automatically. Final-slot
race: one commit and one `ErrLimit`; cross-owner existing-device deletion: 404;
revoke/re-admit: 3 → 2 → 3. Real-binding usage: accepted then duplicate, one
receipt/sample. See [AT02 trace](AT02-TRACE.md). These are no longer requests for
an identity or member binding writer. Complete AT01/AT02 still need B5's traffic
baseline and the platform runs.

## B4 — authoritative payment snapshots and served Portal (AT03/AT04)

Owner: devstack, `tests/e2e/cmd/fvpn-e2e-payment/main.go: server.route`,
`createCheckout`, `getCheckout`; Portal serving hook.

Reproduce with `at-members`: real checkout and idempotent replay return 201;
ten identical signed paid deliveries create one inbox key; all 18 permutations
of created/paid/failed are accepted. Invalid signature returns 401 without a row.
Signed cancellation, deletion, refund and paid-after-suspension notifications
also return 202. Yet `GET /v1/subscriptions/{id}` returns 404 and invoice/refund
lists are empty. No paid period exists to reconcile; pending entitlement
revision stays 1. `/billing/return` is 404 and cannot test a rendered return page.

Supply authenticated TEST ONLY snapshot/event hooks linking an actual checkout
to its subscription, paid invoice, lines, cancellations and full/partial
refunds. The worker must read provider truth and project the interval; accepting
a webhook is not payment proof. Serve the built Portal through a documented
hook. Real short-grant expiry and suspension surviving another grant pass as
partial checks; payment precedence and TCP/UDP flow-stop latency remain blocked.

## B5 — valid boot release and serving baseline (AT01/02/04/05/08/09)

Owner: devstack, `tests/e2e/harness.go: insertReleaseSupport`, `setBootRelease`,
`tests/e2e/scenarios_test.go: releases`; node peer observation separately owned
by `w3-wire-nodes` follow-up 10 (D-055).

Reproduction: fresh `e2e-up`, enroll using the documented token-file hook,
classify the node through PATCH, then PUT publication with current ETag. The API
returns 422 `NODE_NOT_SERVICEABLE`. Actual node state is `applying`, safety
`failed`, applied generation 0; native device operations do advance the desired
generation. There are no NetBird routes. A real node-mTLS
`GET /agent/desired-state` returns **503 `DEPENDENCY_UNAVAILABLE`, “Release
support policy required.”** Active release-support policy count is zero.
`internal/nodes/httpwire/state.go: CompileControlState` enforces this precondition.

Expose a reusable setup hook that packages/registers the actual signed boot
release, configures its identity and activates a valid non-empty release-support
policy through the API. The private shared helper's direct `{}` policy insert
violates the HTTP contract and is not an acceptance workaround. No readiness
rows should be forced. This run does not disprove #75/#77 readiness fixes:
it never obtains a valid policy/ACK baseline.

Node incarnation `peer_id` also remains null. D-055 must provide the agent's
real peer observation; do not guess it or use it to authorize member devices.
The effect of the null peer cannot be isolated before the policy gate is fixed.
Disable returns 202 and its intent survives restart, but applied withdrawal and
group A/B isolation require a previously serving positive control.

## B6 — resolver document delivery and topology (AT05/09/20/21/22)

Owners: **`w3-contracts-r5`, then nodes/agent delivery follow-ups**;
devstack (`deploy/devstack/compose.yaml`, prepare and entrypoint hooks).

Reproduction: create canonical rules, activate a DNS profile, read the central
`dns_resolver_documents`, then read actual AdGuard filtering status and query
it. Central compilation succeeds, but the requested rule is absent from AdGuard.
The controlled blocked name still resolves. Node delivery is not yet supplied;
central policy-operation success is not a resolver ACK. Provide the signed
resolver document/credential/application path, readback and synthetic probe.

The lab has one installer node, one filtering AdGuard resolver plus its CoreDNS
upstream, and member `--disable-dns --disable-ipv6`. Pausing each DNS service
causes failures and restoration recovers responses; this is not two-resolver
failover. Provide three source flavours, a second exit and filtering resolver,
controlled CNAME/HTTPS/SVCB/rebinding zones, native DNS, long TCP/UDP flows,
QUIC/PMTU and forced direct/relay hooks. Platform clients, physical arm64 and a
second physical NAT remain **needs operator run**, with scripts in OPERATORS.md.

## B7 — remaining privacy and retention coverage (AT22)

Owners: devstack, DNS delivery and retention operators. Query history and
statistics are disabled, the DNS sentinel is absent from inspected APIs,
service logs, audit and a memory-only database dump, and guarded audit retention
deletes only the expired synthetic row. These checks do not prove URL privacy
without real exit traffic, retained-file scans, raw-usage/rollup purge and backup
expiry. Supply those fixtures and the candidate configuration inventory.

The D-054 feed content pipeline is now exercised against the existing controlled
origin with a self-authored TEST ONLY license record. It is not missing. The
300-second test cadence uses normal generated settings and real time, with
settings restored afterwards. Evidence records bootstrap pin, scheduled invalid
update and immutable last-good state. No commercial source approval is inferred
from a synthetic feed. Resolver delivery remains B6.

## B8 — configured metrics listener absent (AT22)

Owner: `apps/control/server.go`, `internal/config.Config.MetricsAddress`, devstack.
The bounded HTTP probe inside the control container cannot reach the configured
loopback `/metrics` listener on port 9091. Wire that listener or document the
supported metrics surface. An unavailable endpoint cannot prove privacy.

## B9 — nullable feed attribution rejects a referencing profile (AT22)

Owners: policies/DNS (`internal/policies/store.go: buildAndValidateDNSProfileTx`,
`internal/dnspolicy/sources.go: ListSource.ValidateMetadata`), contract owner if
an orchestrator decision is needed.

The threat-feed contract permits `attribution: null`. Creating/activating that
feed succeeds and the running worker verifies its pinned content. Creating a
DNS profile referencing it rejects the blank `AttributionNotes` passed into the
compiler registry. Reproduce using the explicit null-attribution probe in
`at-members-readback`'s feed inspection. Support the contract's null value when
the license needs no attribution, or request an explicit contract revision.
An explicit TEST ONLY attribution is the documented positive-control workaround;
no immutable policy or observation row is edited.
