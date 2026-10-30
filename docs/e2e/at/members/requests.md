# Follow-up 2 requests to orchestrator

The current records are from the assigned `w4-members / 24000` stack. All
business state is projected by the product. Group/plan/publication commands use
D-057; releases use `e2e.PrepareServingBaseline`; payment truth uses the lab's
authenticated provider hooks. No application or shared lab source was changed.

## F2-01 — current member lease and counted forwarding

Owner: **w2-agent-core**, `node/agent/sync.go:needRenew` and `renew` (with the
central lease owner if generation change needs a new issuance contract).
Blocks complete AT01, AT02, AT03, AT04, AT08 and AT09, and member traffic parts
of AT20–AT22.

Inherit the diagnosis and earlier full forwarding attempt from
[devstack follow-up 7](../../followup7-verification.md), run
`dc2f6353-89ec-434c-bc68-567066d9fe51`. That run had desired/applied generation
5/5, an authorized binding and rate policy, but only a generation-1 lease with
zero members. This suite does not repeat deadline/control-stop faults without
a positive counted flow.

Reproduce with `at-members`: prepare signed boot/support, create A/B groups and
API plans, publish to A, give A a finite grant and join its native user peer.
Read AT01/AT02's current desired/applied generations, lease generation/member
count and nft counter inventory. AT08 records operation steps independently:
agent-policy and route readback can succeed while data-path confirmation stays
incomplete. A 202, a route, or a device row is not an applied traffic pass.

Refresh member-bearing serving authorization after relevant desired/member
changes. Re-run until an A device's real controlled-origin response increments
its node `u_<device>`/`d_<device>` counters. Then prove loser/B denial, existing
TCP/UDP termination, network move/wake and exit switching. Do not forge a lease,
add a binding row or weaken publication's data-path step to obtain green tests.

## B4 resolved — provider payment truth and Portal

AT03/AT04 now exercise real checkout sessions, paid invoice reconciliation,
ten identical deliveries, all six created/paid/failed orders on independent
subjects/checkouts, cancellation, cumulative 250/1000 TEST ONLY refunds, real
paid-period expiry and payment after manual suspension. A forged success return
and invalid signature grant nothing. The built Portal return page is rendered
by the shared browser probe. Exact results and revisions are in the JSON.

The provider hook in `tests/e2e/cmd/fvpn-e2e-payment/truth.go:server.hook`
currently emits paid/updated/deleted/refund/checkout events, but not
`customer.subscription.created` or `invoice.payment_failed`. The runner reuses
its real checkout-bound event objects and signs those two event inputs with the
generated TEST ONLY webhook key. Provider truth is still read through the real
adapter; no financial projection is seeded. Optional lab improvement: support
those two event types in `server.hook` so this input helper can be removed.

## B5 resolved setup; publication still subject to F2-01

The signed boot/support setup, required capacities, group creation, plan
activation, exact-subject native admission and quota race no longer need SQL
fixtures. `at-members-readback` also records rename/regroup/disable/restart and
restoration through the publication API. Its operation evidence distinguishes
pending/failed required steps from applied results. The second node is not
enrolled while the inherited positive forwarding prerequisite is unresolved;
this is not a claim that the shared lab lacks a second-node hook.

## B6/B7 — current resolver evidence supersedes the inherited missing reference

Owners for any remaining delivery mismatch: **w3-wire-nodes**,
`internal/nodes/httpwire/document_inputs.go:ResolveDocumentInputs`, and
**w2-agent-core / DNS**, `node/agent/resolver.go:applyResolver`,
`node/dns/applier.go:Apply`.

The earlier follow-up 7 missing-reference/404/null-evidence result is retained
as historical evidence, not presumed to describe this run. AT20/AT22 read the
typed `NodeDetail.dns_policy_evidence`, match its issued document and fetch that
document with the node's actual mTLS identity. They compare canonical digest
and report `applied` only when the product supplies it. Host-local DNS probes
are expressly distinct from the member's normal resolver path.

Two setup constraints were discovered and addressed without product changes:

- An empty profile invokes `node/dns/applier.go:probe`'s `example.com` allowed
  probe, but `deploy/devstack/coredns/Corefile` answers NXDOMAIN outside its
  controlled zone. The initial attempt reported resolver application failure.
  The final baseline instead includes an API allow rule for the existing
  positive `feed-01.example.invalid` fixture. No external DNS traffic is used.
- `ResolveDocumentInputs` requires a single active DNS definition. The harness
  creates immutable revisions of that definition instead of accumulating
  ambiguous active profiles. Defining how multiple plan DNS profiles select a
  node document remains an owner design question; no unsupported selector is
  invented here.

The D-061 AdGuard sidecar shares the node network namespace. Restarting the node
also requires restarting that owned sidecar to attach its new namespace; the
runner records both actions. It does not treat an unavailable old namespace as
a member fallback result.

## F2-04 — remaining lab and operator coverage

Owner: **devstack**, shared hooks in `tests/e2e/stack_exports.go`,
`tests/e2e/native_exports.go`, `deploy/devstack/entrypoint-member.sh` and
`deploy/devstack/coredns/` (requests only; no edits in this task).

- AT05: provide preinstalled/cloud-init/operator-installer boot flavours with
  package provenance; reservations alone are not three boot tests.
- AT09: after F2-01, exercise both existing node hooks, native route selection,
  forced direct/relay and controlled TCP/UDP/QUIC/PMTU endpoints. Do not infer
  physical NAT or native-platform behavior from a container.
- AT20/AT21: native entrypoint still uses `--disable-dns`; there is one filtering
  resolver plus an authoritative CoreDNS, not two independent filtering
  resolvers. Supply a normal member DNS path, second resolver, policy-sync fault
  hook and positive CNAME/AAAA/HTTPS/SVCB/rebinding fixtures. Current direct
  loopback faults cannot prove native controlled fallback/public-DNS exclusion.
- AT22: extend the retained fixture inventory to service files, successful member
  URL traffic, raw usage/rollup eligibility and backup/log expiration. Current
  evidence covers disabled resolver history/statistics, inspected service logs,
  metrics, audit, an in-memory database dump, bounded audit purge and scheduled
  feed last-good. It does not certify every retention surface.

Windows/macOS/iOS/Android, physical arm64, second physical NAT and candidate
production hardware remain **needs operator run** under [OPERATORS.md](OPERATORS.md).
Commercial feed approval is a business decision; the licensed source exercised
here is explicitly self-authored TEST ONLY data.

## F2-05 — enrolled node restart waits for another token

Owner: **devstack**, `deploy/devstack/entrypoint-node.sh`, the unconditional
`waiting_token`/enrollment block before watchdog, agent and NetBird startup.
Observed in AT08's restart test, on the same node incarnation used by the other
cases. The durable identity file exists, but `/var/lib/vpn-node/e2e/status.json`
says `waiting_token` and `nft list table inet fvpn` fails because no agent
reinstalled its baseline. Persisted `publication_intent=disabled` alone does not
prove a rebooted, connected node enforced disable.

Reproduce: enroll normally, issue disable through PUT publication, then
`docker restart fvpn-w4-members-node`; reattach its AdGuard sidecar and inspect
status, existing identity, daemon readiness and nft table. The entrypoint always
requires `enrollment.token`, which the enrollment path consumes. Add an existing
identity boot branch that starts the normal watchdog/agent/NetBird supervision
without a second token or a new incarnation. Re-run disable/reboot and verify
fresh observations plus real denial after the positive-flow gate is fixed.

Documented partial workaround: `e2e.Stack.Service(ctx, "start", "/usr/local")`
starts the agent so independent resolver-document checks can continue. It does
not recreate NetBird/watchdog supervision or pass reboot recovery. In this run
that restart also obtained a later member-bearing lease; it does not fix renewal
after desired/member changes during uninterrupted operation. The original
empty-member lease evidence remains in AT01/AT02 and AT08's prior checks.
