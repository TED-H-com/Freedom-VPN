# Follow-up 4 requests to orchestrator

Use the [current AT table](summary.md), [run ledger](RUN.md) and per-AT JSON
for the observed boundary. No application, contract, migration or shared lab
source was changed. No D-050 projection was seeded or repaired.

## F4-01 — empty member lease after second-exit restart

Owner: **w2-agent-core**, coordinated with **w3-wire-nodes** for lease issuance.
Locations: `node/agent/sync.go:needRenew`, `node/agent/decide.go:Decide`,
`internal/leases/leases.go:RenewTx` (`includeMembers`). AT: AT09.

Reproduce on fresh `w4-members / 24000`: run the complete suite through paid
member admission and both published exits; select node2 and prove its counted
transfer. Stop `node2`, `node2-adguard` and `node2-netns`, then start the holder,
resolver and node in that order. Do not alter any identity, policy or projection.
Poll a fresh accepted serving observation for 90 seconds, then issue the bounded
member transfer. The same node incarnation resumes, but the recovery assertion
fails and the member's five TCP attempts reach node2's `fvmb0` and its
generation-zero baseline. No origin receipt or device byte increment follows;
the primary exit still carries counted traffic.

Run `d89eda32-84ef-4582-b669-053d5d9abadd` records generation 23 with seven member
grants before the fault. At `17:37:24.662928Z`, immediately after restart, central
issues another generation-23 lease with **zero member grants**, expiring
`2026-10-31T17:37:24.662928Z`. Repeated accepted observations report forwarding
false and `lease_expired`. AT20's independent DNS policy change subsequently
issues generation 24 at `17:39:36.696084Z`, again with seven grants, and forwarding
becomes true. This later change is not an AT09 repair or a passing recovery.
The [JSON](AT09.json) retains the failed primary check in `prior_checks`, the
original packet trace, and the immutable lease/observation timeline. The full
primary record is under [attempt 2](history/follow-up-4/attempt-2/AT09.json).

The source reading points to a same-generation refresh gap: `RenewTx` correctly
omits member grants while the node is not ready, but `needRenew` considers a
fresh same-generation node lease sufficient until its renewal lead. `Decide`
uses `lease_expired` when nonempty desired bindings yield no admitted model
bindings, even though the node lease deadline is in the future. This is an
inference from the trace and source, not a claim about an unobserved internal
branch.

Request: investigate bounded lease refresh after restart/readiness when a
published populated desired generation receives an empty grant list. Restore
service through normal verified leases without requiring an unrelated policy
change; keep fail-closed entitlement/binding checks and accurate block reasons.
Add a counted restart regression with unchanged desired generation. Do not
force readiness, forge a grant, shorten a production timeout or clear state.

## F4-03 — remaining transport/platform matrix

Owner: **w3-devstack-e2e and the platform/operator campaign**.
Locations: `tests/e2e/second_exit.go:StartSecondExit`,
`tests/e2e/cmd/fvpn-e2e-origin/probe_udp.go`, and `docs/e2e/RUN.md`.
ATs: AT01, AT04, AT09.

Reproduce: prepare the signed baseline, enroll/publish both actual exits, admit
paid native peers and run the owned harness. HTTPS traverses both exits with
selected-node SNAT and device counter evidence. The available UDP/4443 hook is
an authenticated updater-only exception probe; it is not a member echo/QUIC
fixture and must not be counted as member traffic. No controlled forced-relay,
QUIC or PMTU matrix is exported.

Request: add a bounded member TCP/UDP/QUIC/PMTU fixture and a documented
forced-relay transport fault through the shared lab. Keep the updater probe
separate. Run the exact four-platform/network-change and physical second-NAT
scripts in [OPERATORS.md](OPERATORS.md). Persistent TCP refund, expiry and
suspension observations are retained independently; they cannot pass the UDP
or physical-network assertions.

## F4-04 — member UDP withdrawal on every exit

Owner: **w3-devstack-e2e**. Location:
`tests/e2e/cmd/fvpn-e2e-origin/probe_udp.go`; AT04.
The existing hook serves the updater exception only. Reproduce with the real
paid cohorts and refund/expiry/suspension events in AT04: persistent TCP can be
measured, but there is no sanctioned member UDP echo peer. Add the bounded member
fixture requested in F4-03, then run existing and new UDP flows through both
published exits. Do not reuse or count the non-member updater probe. Extend `deploy/devstack/probe-portal.mjs` with rendered suspended/expired state readback, coordinated with the portal owner; the current probe establishes the billing return only.

## F4-05 — native DNS configuration and automatic controlled fallback

Owner: **w3-devstack-e2e**, then the native-client integration owner if required.
Locations: `deploy/devstack/entrypoint-member.sh`,
`tests/e2e/native_exports.go:JoinNativeMember`, `tests/e2e/second_exit.go`.
ATs: AT20, AT21.

Reproduce: read only `DisableDNS` from the running member's
`/var/lib/netbird/default.json`; it is true. Pause each of `adguard` and
`node2-adguard` in turn: its loopback DNS endpoint becomes unavailable while
the other continues to apply the synthetic block. Typed node evidence and the
node-mTLS document hash establish resolver delivery separately.

Request: export a native member DNS configuration path using the product's
issued resolver document and both authorized filtering resolvers. Then measure
automatic fallback, no public fallback and recovery. Explicit loopback queries
are not native-client failover. The synchronization corruption hook is tested
separately; Private DNS/DoH remain operator runs with member guidance.

## F4-06 — complete privacy and retention surfaces

Owner: **w3-devstack-e2e**, with usage/database/backup owners for retention inputs.
Locations: `tests/e2e/backup_scenario_test.go`, `docs/e2e/scenarios/backup.md`,
`tests/e2e/at/members/dns_privacy_test.go:privacy` and
`privacy_readback_test.go:secondPrivacy`.
AT: AT22.

Reproduce: run the synthetic DNS sentinel, disabled querylog/statistics
readback, log/metrics/audit/pg_dump scans and guarded audit purge. The extra
resolver has its own config/log inventory. The scheduled partial target
publishes a self-authored licensed feed, observes the real failed refresh and
last-good delivery. These checks do not age representative raw-usage/rollup,
file and backup artifacts through every configured retention deadline, or
cover a representative member URL history workload.

Request: provide sanctioned retention inputs for those remaining surfaces and
a bounded member URL privacy sample through shared hooks; do not backdate or
repair projector-owned rows. Record the selected commercial feed's actual
license separately: the TEST ONLY self-authored feed is not commercial-source
approval. Full AT22 stays blocked despite passed subsets.

## F4-07 — controlled DNS RR/cache/rebinding matrix

Owner: **w3-devstack-e2e**, with DNS/netlab owners reviewing fixture semantics.
Locations: `deploy/devstack/coredns/example.invalid.zone`,
`tests/e2e/at/members/dns_privacy_test.go:dnsRules`.
AT: AT20.

Reproduce: publish canonical rules through the API and inspect the node's typed
applied document, then issue the recorded A/AAAA/CNAME/HTTPS/SVCB queries. API
normalization rejection and resolver application are measured, but the shared
zone does not provide the complete positive/negative CNAME, HTTPS/SVCB, cache
expiry and rebinding sequence. Native DNS is additionally blocked by F4-05.

Request: extend the controlled zone and sanctioned rebinding fixture with exact
answers/TTLs, including label-boundary controls and an allowed domain that
changes to an isolated private target. Prove that DNS allowlisting never opens
that network. Never query unrelated Internet domains.

## F4-10 — source flavours and physical architecture

Owner: **packaging/devstack and the operator campaign**.
Locations: `deploy/devstack/entrypoint-node.sh`,
`tests/e2e/stack_exports.go:EnrollAt`, [OPERATORS.md](OPERATORS.md).
AT: AT05.

Two real nodes use the same installer entrypoint. Installer/preinstalled token
reservation and one-time replay contracts pass, but a preinstalled reservation
is not a preinstalled boot and a second installer is not cloud-init. Supply the
other two boot-source fixtures/provenance and run physical arm64. The signed
arm64 artifact and arm64 probe cross-build are not a hardware run.

## Closed prior findings and preserved harness findings

- F3-02: current images contain conntrack. AT02 requires a continuous agent PID,
  coherent serving health, revoked-source denial/empty conntrack and counted
  sibling/replacement traffic. The historical failure remains archived.
- F3-01: selection waits for the advertised route and explicitly selects the
  network matching the intended node's peer. `skip_auto_apply` remains true.
- D-056/D-061: typed resolver evidence, canonical mTLS document hashes and actual
  block answers replace the former missing-delivery diagnosis. Null feed
  attribution is valid; only blank/whitespace notes are rejected.
- Attempt 1 retains its early `partially_succeeded` withdrawal sample and the
  subsequent node ACK in `AT08-late-operation.json`. `executeWithdraw` explicitly
  permits this transition. Attempt 2 retains progress states and waits within
  the original bound for succeeded.
- Attempt 1's restart probe sampled the fail-closed startup before fresh health.
  Stopping only `node2` also leaves its kernel namespace held by two containers.
  Attempt 2 stops all three namespace holders and requires a new serving
  observation before its recovery transfer. No product state is repaired.
- Attempt 1 AT21 retains HTTP 403 after recent-auth expiry. Attempt 2 performs
  real OIDC login per case; no auth window or limiter configuration changed.

The strict post-revoke node-ingress measurement remains the **devstack** owner's
separate [recorded limitation](../../runs/2026-10-30/followup10-run3/device-revoke.md):
route withdrawal prevents that later packet from reaching the node. This harness
does not restore a revoked route or relabel upstream withdrawal as a kernel drop.
