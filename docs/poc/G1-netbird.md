# Gate G1 — pinned NetBird CE machine evidence

Date: 2026-10-29. **Machine subset passed; G1 remains open for native apps and
identity integration.** This run proves real local-user login and scoped exit
selection on Linux clients. It does not establish Windows/macOS/iOS/Android
support, production shared-issuer mapping or G2 forwarding safety.

## Version and topology

NetBird server/client **0.82.0**, dashboard **2.96.0**, nginx **1.30.0-alpine**;
exact OCI index digests and component licenses are in
[VERSIONS](../netbird/VERSIONS.md). The combined server includes management,
signal, relay, STUN and embedded Dex. The run used Linux amd64 containers and
Go 1.27.1. The adapter also cross-compiles for Linux arm64; no real arm64
execution is claimed. Public documentation and pinned source review are
recorded in [SOURCES](../netbird/SOURCES.md).

```text
host 127.0.0.1:18080 TCP / 127.0.0.1:13478 UDP
                     |
             fvpn-w1-netbird-gateway
              /                    \
     dashboard                  combined server
                                   |
       Docker internal network, no Internet egress
             /             |             |          \
      exit (one-off)   member-a (SSO)  member-b (SSO) replay
```

Only the gateway joins the separate frontend bridge to expose loopback ports.
All upstreams are fixed to our stack. TEST ONLY passwords, relay/store secrets,
owner PAT and service PAT are generated locally, mode 0600, never committed or
printed. NetBird database access is exclusively through its own services.
All resource names use the task prefix. Client and server volumes are separate.

The account is IPv4-only (`ipv6_enabled_groups=[]`), with telemetry, geolocation,
auto-update and JWT/user auto-group propagation disabled. Before any peer joins,
bootstrap removes the All-to-All default policy. A newly created user has no
paid group. The node's one-off key assigns only `fvpn-unpublished-nodes`.

## Reproduction

From a clean worktree with the prerequisites in the
[deployment README](../../deploy/dev/netbird/README.md):

```sh
export FVPN_NB_PREFIX=fvpn-w1-netbird
make -f mk/netbird.mk netbird-up netbird-bootstrap
python3 -B deploy/dev/netbird/bootstrap.py
make -f mk/netbird.mk netbird-test netbird-vet netbird-test-integration
mkdir -p .agent/netbird-tmp
TMPDIR="$PWD/.agent/netbird-tmp" GOOS=linux GOARCH=arm64 go build ./internal/netbird/...
TMPDIR="$PWD/.agent/netbird-tmp" GOOS=linux GOARCH=amd64 go build ./internal/netbird/...
docker compose -f deploy/dev/netbird/compose.yml --profile clients config --quiet
make -f mk/netbird.mk netbird-reset
docker ps -a --filter name=fvpn-w1-netbird
```

`netbird-reset` calls Compose `down -v --rmi all` and removes generated secret
state. It targets only this prefix. Run tests with a disk-backed TMPDIR; the Make
targets use `.agent/netbird-tmp/`. The integration suite skips when called directly
without its API URL/token-file environment variables. CLI evidence additionally
requires the explicitly enabled disposable Compose clients.

## What the machine proved

| Check | Observed result |
|---|---|
| Bootstrap rerun | Same owner/service users; existing service token accepted; default policy remains absent. |
| Real user login | Both ordinary local users completed embedded Dex device authorization. API peer `user_id` matched the created user's opaque ID. No member setup keys were used. |
| Default deny | Before authorization, member A could not ping the node or another member over overlay. The control channel remained connected. This is narrow overlay evidence, not exhaustive host isolation. |
| Machine enrollment | Node joined with one-off, 24-hour, usage-limit-one key; replay container's second use was explicitly rejected. Retry returned metadata without secret replay. Node initially had no published exit. |
| Exit API | Read-back showed `network=0.0.0.0/0`, exactly one peer, no `peer_groups`, `skip_auto_apply=true`, `masquerade=false`. |
| Exit client behavior | A in the paid distribution group saw the route; B in the other paid group did not. A's `netbird routes select <stable network ID>` changed status to Selected. |
| Positive/negative overlay control | Authorized A could ping the node; out-of-group B could not. The positive control rules out a missing ping binary as the explanation for negative probes. |
| Revocation/withdrawal | Removing A's paid membership removed its route. Reauthorizing restored visibility; withdrawing the exit removed the route again. |
| Repeated operations | Repeated publication and authorization retained counts of five groups, one policy and one route in the fresh-stack run. Unit tests also count mutation requests and require zero writes on no-op repeats. |
| Lost response | Faulting HTTP transport forwarded a real route POST, drained/closed the successful upstream response, and returned an injected transport error. Retry found the stable route and confirmed it by GET; count stayed one. |
| Observation | Fresh peer/group/route/policy/nameserver reads reflected actual upstream objects, independent of desired state. |
| DNS/ops API | Two resolver addresses, paid-only nameserver distribution, TCP/UDP 53 exceptions and an explicit admin TCP port were created and read back. Resolver identities used the two existing lab peers as TEST ONLY stand-ins; no DNS service or failover was tested. Exceptions were removed after recording. |
| Architecture/checks | Unit suite, full Go suite, vet, formatting and Linux amd64/arm64 builds passed. Compose configuration validated. |

Representative scrubbed CLI state during the run:

```text
Daemon version: 0.82.0
CLI version: 0.82.0
Management: Connected
Signal: Connected
Relays: 2/2 Available
Interface type: Kernel

netbird routes list   # member A after distribution, before manual selection
  - ID: TEST-EXIT-NETWORK-ID
    Network: 0.0.0.0/0
    Status: Not Selected

netbird routes list   # member A after routes select
  - ID: TEST-EXIT-NETWORK-ID
    Network: 0.0.0.0/0
    Status: Selected
```

The network ID above replaces the generated stable label. Read-back fixtures
retain API field shapes with all identifiers/metadata scrubbed; see
[`internal/netbird/testdata`](../../internal/netbird/testdata/README.md).
`Relays: 2/2 Available` reports service reachability, **not proof that a tested
flow traversed a relay**. No external traffic, MTU/throughput claim, SNAT claim
or resolver-query log was produced. Raw authentication logs are private and
are removed on reset.

Final fresh-stack integration output (identifiers and secrets were never printed):

```text
=== RUN   TestAT08_LiveAdapterReadBack
    integration_test.go:155: AT05 subset: one-off node enrollment succeeded; second use rejected; secret not replayed
    integration_test.go:185: AT02/AT08 subsets: ungrouped real user peer cannot reach node or member overlay; default policy absent
    integration_test.go:220: AT08/AT09 subsets: IPv4 exit, single peer, skip_auto_apply=true, masquerade=false; A sees and selects; B does not see
    integration_test.go:221: Overlay ICMP positive control: A reaches exit; out-of-group B is denied
    integration_test.go:232: Repeated publication/authorization object counts stable: groups=5 policies=1 routes=1
    integration_test.go:239: Authorization removal and exit withdrawal each removed client route
    integration_test.go:265: Lost POST response reconciled by stable name and fresh GET; one route remains; observed state matches upstream
    integration_test.go:271: DNS/ops API read-back: two resolver addresses; paid-group distribution; TCP/UDP 53 and explicit admin port only (resolver service not tested)
--- PASS: TestAT08_LiveAdapterReadBack (5.50s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/internal/netbird	7.578s
```

## Requirement / acceptance boundary

| Spec / requirements | Coverage here | Still needed |
|---|---|---|
| §2, §7.3, §9.1; ID01, APP01; AT01 | Real Linux local-user → peer association | Four native apps, Portal/shared issuer, verified cross-client subject proof, production admin MFA |
| §9.4; MEM01–MEM03, ENT01; AT02 | Peer-owner mismatch rejection, separate mapping/grant, per-device revocation, safe concurrent group writes | Central quota row locks, simultaneous last-slot race and cross-member HTTP authorization |
| §6.1–§6.4; ENR04; AT05 subset | Unpublished one-off machine enrollment and replay rejection | Three hardware/source installation traces, mTLS enrollment, incarnation proof, real arm64 |
| §7.1–§7.2, §14.5; EXIT01–EXIT03, API01; AT08 | API read-back, group restriction, Linux route selection/removal, no-duplicate retry | Node ACK, Internet-only forwarding, established-flow revocation, restart/lease behavior and UI pending/applied integration |
| §8.3; EXIT02, APP01, NFR01; AT09 subset | Manual Linux exit selection, source-preserving route flags | Two-exit native app switching, direct/relay traffic, dual NAT, TCP/UDP/QUIC, PMTU/MSS and throughput |
| §12.2; ISO04, DNS03 | Exact DNS/ops policy and nameserver API distribution | Real resolvers, privacy, per-device profiles, platform fallback and Android Private DNS |
| §15.2–§15.3, §17.4; DATA01 | Binding interface, memory uniqueness, fresh observations and conflict handling | DB implementation, durable outbox, restore drills and generation fencing |
| §16.1–§16.2; SEC01/SEC02 subset | Central-only PAT, no-secret errors, one-time secret behavior, no traffic/DNS logs | Production OIDC/MFA, operational secret rotation and full platform/security acceptance |

## Remaining 待實測

Windows and macOS 0.82.0; Android app 0.6.2; iOS app 0.4.1 build 2 must each
complete self-hosted URL setup, login, VPN permissions, correct member/device
mapping, exact exit label/display, manual A/B selection, withdrawal and revoke,
sleep/wake, network change and reconnect. The
[per-platform checklist](G1-app-checklist.md) records every screen, expected
result and space for actual results/screenshots. iOS TestFlight availability
is not an assertion about App Store distribution.

Direct versus forced relay, real NAT pairs, large packets/PMTU, TCP MSS, QUIC,
IPv6 leak prevention, DNS resolver failover and Android Private DNS remain open.
The HTTP dev gateway deliberately does not expose QUIC relay. Changing an exit
must not be assumed to move DNS to the exit's country. Native explicit route
selection can persist across route withdrawal/republication; integration resets
the local selection preference before its initial-selection assertion.

The product cannot promise a full endpoint kill switch from server-side
isolation. G2 must verify the native packet path, isolation, SNAT placement,
metering and established-flow revocation. The embedded dev IdP is not declared
a shared Portal issuer; matching email is never accepted as identity proof.
