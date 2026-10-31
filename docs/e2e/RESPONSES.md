# Devstack responses

Under D-102, each lane owns its `requests.md`. Devstack appends dated answers
here, with one `## <lane> — <date>` block per answer and the original request
heading quoted verbatim. Earlier answers stay where they were written. New
requests go in the dated hand-off report for the orchestrator to route.

## members — 2026-10-31

> ## Devstack follow-up 15 request: export a real plan-downgrade fixture

`Stack.ReplaceFiniteGrant(ctx, admin, memberID, lowerPlanRevision, validUntil)`
is available through `labstack`. It resolves the actor from the current admin
session, calls production `members.Service.RevokeGrant`, creates the replacement
through the public admin grant API, verifies same-key replay, and waits for the
worker operation plus entitlement/device projection. It exports partial results
on error and never writes projector-owned rows. It also restores a higher plan.
The fixture rejects primary subscriptions and expired/excessive deadlines before
revocation. Reauthorization remains on the existing HTTP route.

[Exact invocation and evidence fields](README.md#finite-plan-grant-transition-members-request-3).
[Follow-up 16 runs](runs/2026-10-31/followup16/README.md) retain downgrade,
restoration and failed-input results from real native peers.

## members — 2026-10-31

> ## F4-03 — remaining transport/platform matrix

`PrepareNativeUnderlay`, `ApplyNativeInputs` and `ObserveNativeUnderlay` expose
IPv4/IPv6/dual peer transport availability and MTU 1280–1500 on the real native
member/node. Transport selection is observed, not asserted. The campaign uses
bounded 64,064-byte TCP transfers and device counters. Explicit `quic` and
Android/iOS/Windows/macOS inputs return unavailable with their missing real
execution prerequisite; neither UDP echo nor Linux substitutes for those rows.
Physical NAT/roaming and the complete PMTU/QUIC/platform campaign remain open.

[Input effects and boundaries](README.md#native-underlay-and-protocolplatform-inputs).
[Actual measurements, including failures](runs/2026-10-31/followup16/README.md).

## members — 2026-10-31

> ## F4-05 — native DNS configuration and automatic controlled fallback

Prepare with `FVPN_E2E_TWO_RESOLVERS=true`, enroll the independent second exit,
then call `ConfigureNativeDNS`. It checks identical applied policy-document
hashes, reads the product-issued resolver addresses, distributes them through a
TEST ONLY peer-restricted native nameserver group, enables native DNS across a
down/up cycle, and checks both native nameservers and the system stub. The
campaign measures ordinary system resolution after primary transport loss,
both-down failure and recovery. It does not claim product automatic nameserver
distribution, nor Private DNS/DoH platform results.

[Invocation and timing exports](README.md#native-two-resolver-dns-and-negative-answers).
[Observed outcomes](runs/2026-10-31/followup16/README.md).

## members — 2026-10-31

> ## F4-07 — controlled DNS RR/cache/rebinding matrix

The controlled authority now supplies A/AAAA/CNAME/HTTPS/SVCB positives,
blocked-name and label-boundary controls, a 10-second positive cache record,
a 5-second negative SOA, and an allowlisted name changing to private A/AAAA.
`SetDNSFixturePhase("private")` changes authority only; caches are not flushed.
The runner observes authority convergence separately, tests before/after TTL,
and uses independently listening private targets behind the exit. TCP/UDP
exports record expected RCODE sets, answer presence, TTLs, address classes and
connection results. Blocked direct CNAME/SVCB permit only empty NODATA or
NXDOMAIN; they are not reported as uniformly NXDOMAIN. Other blocked types and
CNAME-chain A require NXDOMAIN. Query contents are not exported.

[Fixture contract](README.md#native-two-resolver-dns-and-negative-answers).
[Per-run evidence and qualifications](runs/2026-10-31/followup16/README.md).
Full AT20/AT21 still require the consuming lane's assertions and platform work.

## updates — 2026-10-31

> ## Remaining full-§19 scope items outside this focused follow-up

`RetainEnrollingThirdNode` and `BootEnrollingThirdNode` export a third target for
the pre-existing enrolling population. A real failed offline agent enrollment
persists the key/CSR/request/pending incarnation while the API-created node
remains centrally `enrolling/pending`, with no bound incarnation. Its mid-run
boot resumes that same identity and original request. Exports include node ID,
pending incarnation, central null-incarnation evidence, key digest, timestamps,
and the first ready observation after central binding. No projector rows or
installer wrappers are used. The consumer still owns the batch snapshot state,
job assignment, update inheritance and D-089 assertions.

[Exact lifecycle invocation](README.md#retained-third-enrolling-fleet-target).
[Run IDs and deciding observations](runs/2026-10-31/followup16/README.md).
