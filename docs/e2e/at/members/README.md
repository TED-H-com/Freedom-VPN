# Wave 4 member acceptance — Follow-up 2

This suite exercises the real assigned devstack. Whole ATs remain non-green
while their traffic, topology or operator prerequisites are missing. Successful
subchecks are recorded separately; a 202 never counts as applied forwarding.
See [RUN.md](RUN.md), [requests.md](requests.md) and each JSON record.

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
# Supplemental evidence on those same images and node incarnation:
make -f mk/at-members.mk at-members-readback
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
```

Use a fresh stack. `PrepareServingBaseline` builds/signs both architectures and
activates nonempty release support before enrollment. Groups, plan policies,
finite grants, revocation, capacities and publication use product APIs. Native
peers use shared Dex login and `StartNativePeer`. Two real-adapter goroutines
race for the final device slot. All business projections are product-owned
(D-050); the only direct SQL inputs left are synthetic old/young audit rows.

Billing uses authenticated provider snapshots tied to actual checkout IDs,
then exact signed webhook bytes and reconciliation polling. Each event-order
permutation uses an independent subject/checkout. Created/failed event inputs
reuse the provider's object and the generated TEST ONLY signing key because
those two event kinds are not yet emitted by the shared hook. No invoice,
subscription, paid interval or entitlement row is hand-written.

The baseline DNS profile allows the existing controlled `feed-01.example.invalid`
fixture; it does not depend on a public DNS probe. Later policies revise the
same active DNS definition. Typed node evidence and actual node-mTLS document
fetches distinguish delivery from application. Loopback resolver probes are
labeled separately from native member DNS, which remains disabled by the lab
entrypoint. The inherited lease gate blocks traffic claims.

`at-members-readback` records current projections, independent A/B publication
commands, disable/restart persistence, NetBird route semantics, and resolver
application. It then runs a real 300-second feed-refresh failure experiment on
a self-authored TEST ONLY source and verifies last-good preservation. It restores
the feed bytes and generated DNS cadence settings. The target is a **partial**
path; its successful exit does not pass a full AT.

JSON records include UTC times, commands, expected/actual results, source and
binary/image digests, release artifact hashes, policy revisions and node
incarnation. `complete:false` records are incremental and remain blocked until
the case finishes. Raw logs, dumps, provider bodies, signatures and credentials
are never public evidence.

| AT | Full acceptance | Evidence | Outstanding scope |
|---|---|---|---|
| AT01 | blocked | [JSON](AT01.json), [page](AT01.md) | counted traffic; four platforms |
| AT02 | blocked | [JSON](AT02.json), [page](AT02.md), [trace](AT02-TRACE.md) | winner/loser real gateway control |
| AT03 | blocked | [JSON](AT03.json), [page](AT03.md) | paid device's gateway authorization |
| AT04 | blocked | [JSON](AT04.json), [page](AT04.md) | long-flow stop on every node; rendered state |
| AT05 | blocked | [JSON](AT05.json), [page](AT05.md) | three boot sources; physical arm64 |
| AT08 | blocked | [JSON](AT08.json), [page](AT08.md) | applied group forwarding and disable |
| AT09 | blocked | [JSON](AT09.json), [page](AT09.md) | two-exit flow baseline, transport/MTU, second NAT |
| AT20 | blocked | [JSON](AT20.json), [page](AT20.md) | native DNS, complete RR/cache/rebinding matrix |
| AT21 | blocked | [JSON](AT21.json), [page](AT21.md) | second filtering resolver; native DNS/DoH/DoT |
| AT22 | blocked | [JSON](AT22.json), [page](AT22.md) | member URL traffic, full file/usage/backup retention |

Platform, arm64 and physical NAT rows stay **needs operator run** under
[OPERATORS.md](OPERATORS.md). [PROCEDURES.md](PROCEDURES.md) specifies the full
remaining acceptance sequence; [BLOCKERS.md](BLOCKERS.md) is historical.

```sh
make -f mk/at-members.mk at-members-lint at-members-test at-members-build
make go-lint
go vet -tags e2e ./tests/...
```

Targets use disk scratch and pinned staticcheck. The workflow checks PRs and
runs the assigned stack on manual dispatch, installs the shared Portal browser
probe, removes only its prefix and uploads only public evidence.
