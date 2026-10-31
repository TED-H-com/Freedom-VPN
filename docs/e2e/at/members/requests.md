# Follow-up 7 requests — 2026-10-31 UTC (2026-10-30 EDT task)

Current evidence: run `297faa31-2f90-4aab-8cf0-6818629f46c8`, [summary](summary.md), [D-077](D077.md), [ledger](RUN.md). No application, contract, migration or shared lab source was changed.

## Closed: worker RestartPolicy note (devstack follow-up 14, #240)

Owner: w3-devstack-e2e. Actual location: `deploy/devstack/compose.yaml` (the older note named a nonexistent compose path). AT05 records Docker `HostConfig.RestartPolicy.Name=unless-stopped` for worker, control and NetBird on the fresh stack. The scheduled feed readback no longer manually starts a stopped worker. Normal explicit restart for loading/restoring the TEST ONLY feed cadence remains.

## D-077 / D-081 production HTTP result

Owner: w3-wire-members for handler behavior; contracts for wire-schema discrepancies; portal only for rendered-state discrepancies. [D077.md](D077.md) contains the deciding statuses and fence transitions. No D-077 portal rendering is claimed. The D-081 expected result is **409 DEVICE_LIMIT_REACHED**; revision 12's old 422 prose was corrected by #247 before this baseline.

All requested D-077/D-081 HTTP, projection, cursor, fence and native-forwarding assertions passed in the final run. No handler or schema discrepancy was observed. This statement does not cover rendered portal state.

The first attempt's missing bootstrap-admin ID was a harness error, corrected by resolving the actor from the current admin session. It is not a wire-members defect; [attempt 1 remains archived](history/follow-up-7/attempt-1/RUN.md).

[Attempt 2](history/follow-up-7/attempt-2/RUN.md) exposed an inherited usage probe allocating the live meter's next sequence. The agent won that race and the server correctly returned a conflict. The final diagnostic replays a real accepted payload, preserves receipt/sample counts, and asserts no new conflicts; the rejected unknown-device negative runs during a bounded shared-supervisor stop and resumes the same release. This is a harness correction, not a usage-service finding.

## Devstack follow-up 15 request: export a real plan-downgrade fixture

Owner: w3-devstack-e2e, coordinated with w3-wire-members. Current local seam: `tests/e2e/at/members/reauthorization_test.go:replaceFixtureGrant`. The HTTP contract has a grant-create route but no grant-revoke route. To obtain a genuine pending device at its member's allowance, the harness uses production `members.Service.RevokeGrant` with the authenticated admin actor, then HTTP grant creation against an API-created lower-limit plan. Export this bounded fixture through the shared lab API so other lanes do not reproduce its setup. Keep re-authorization on the production HTTP route; never seed or repair device/entitlement projections.

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


## Preserved resolutions and measurement boundaries

- AT02 strict post-withdrawal ingress remains a separate `ObserveMemberIngress` check: revoked source hits `ctunku`, sibling reaches the uplink/origin with its named counter. Fresh values are in AT02.json.
- D-067 is re-measured in AT09.json and summary.md, compared with follow-up 6's 36.976 seconds. Earlier timestamps do not describe this run.
- Fenced native HTTPS is measured before/after explicit re-grant. While fenced, the route is withdrawn and the bounded probe receives no origin response; absent fvmb0 arrival is not a separate injected-ingress enforcement claim.
- Rendered D-077 UI behavior and physical clients are outside this API/CLI observation; no portal finding is inferred from source alone.
