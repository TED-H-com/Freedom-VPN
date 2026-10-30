# Follow-up 2 — counted assembled isolation

All four full product ATs remain **blocked**. Thirteen measured subchecks pass,
including cross-member isolation, publication withdrawal, revocation and finite
member lease expiry. The four independent mechanism controls pass. The aggregate
run exits **2**, after writing every product record.

Live run: `422cdd6d-090b-4bb7-b92b-d4883259ac87`, source
`11698fc4d7205a5c902592660ccb346d7f4db750` (clean code, main baseline `be2976c`).
Product UTC window: `2026-10-30T17:29:04Z`–`2026-10-30T17:45:37Z`.
Image IDs, incarnation, generation and per-check times are in each JSON record.
The final desired generation was 27.

| Product AT | Status / evidence | Counted positive control in the same record | Result and remaining gap |
|---|---|---|---|
| AT10 | blocked — [JSON](AT10.json), [Markdown](AT10.md) | A1, 0 → 2632 bytes for the interface/generation case; absent for the six earlier destination cases | Interface change plus generation 13 → 14: 3976 attempts in 60.01 s, zero delivered. Initial RFC1918, metadata, internal public-prefix and deprecated-6to4 probes lacked an advertised exit and stay blocked. |
| AT11 | blocked — [JSON](AT11.json), [Markdown](AT11.md) | A1, 5264 → 7895 bytes for the platform precondition; absent for early metadata/6to4 probes | No controlled IPv6/platform path. Initial destination negatives cannot pass without their positive. |
| AT12 | blocked — [JSON](AT12.json), [Markdown](AT12.md) | All ten checks have a same-device origin control; cross-member A1 10526 → 13199, B1 0 → 2672 bytes | Eight pass: both same-member directions, both cross-member directions, three management ports and A-only publication withdrawing B while A stays counted. DNS traverses the member underlay; stale/spoofed ingress remains unmeasured. |
| AT13 | blocked — [JSON](AT13.json), [Markdown](AT13.md) | All seven checks have controls; revoked A2 2673 → 5305, expiry member 0 → 2631 bytes | Four pass: disable, TCP/UDP revoke, restart recovery and finite-member expiry. Post-revoke ingress attribution, full continuous fault matrix and short whole-node expiry remain blocked. |

The early blocker was serving convergence. An accepted generation-8 observation
reported lease `STALE_GENERATION` against a newly issued generation-9 lease.
Later leases had no members while desired generation 12 held three bindings and
two member policies; the forward chain was baseline generation 0 and the exit
was withdrawn. The counted path recovered at generation 13 before the capacity
PATCH. Both the [earlier state](node-state-before-publication-probes.json) and
[recovered state](node-state-after-recovery.json) are retained. Later success
does not erase the six early missing-positive outcomes.

Revocation had healthy established TCP and UDP exchanges beforehand, then both
stopped, conntrack read-back showed zero entries, the new serving generation
applied, and A1 remained counted. No post-revoke apply failure was observed.
The native route can withdraw before a revoked client's next node-ingress packet;
this limits independent kernel attribution, as recorded in #170.

For finite-member expiry, the actual signed deadline was `17:45:14Z`; control and
worker stopped at `17:44:50Z`. Five later requests retained the exit route, each
appeared on node ingress, none appeared on the uplink, and admission bytes stayed
2631. This subcheck was re-evaluated offline after correcting the singular
tcpdump-summary parser. The [original live AT13](history/follow-up-2-live/AT13.json)
is preserved; the root record includes the evaluator hash and review time.
The live gate and the corrected aggregate both remain blocked. Whole-node short
expiry is independently blocked by `GET /admin/fleet-config` returning
501 `NOT_IMPLEMENTED`.

| Mechanism control | Status | Assembled result it supports |
|---|---|---|
| [lab-AT10](lab-AT10.json) | pass | AT10 destination and interface/apply isolation; 20 regenerations, 86.6 attempts/s, positive origin retained |
| [lab-AT11](lab-AT11.json) | pass | AT11 IPv6 block and special-address mechanisms; does not prove real member-platform paths |
| [lab-AT12](lab-AT12.json) | pass | AT12 peer/source/DNS isolation; directional captures retain the outbound-capture artefact distinction from #69 |
| [lab-AT13](lab-AT13.json) | pass | AT13 conntrack revocation and baseline/crash mechanisms; does not replace assembled lifecycle or ingress proof |

Area lint, whole-module `make go-lint`, ordinary Go tests, amd64/arm64 builds,
and capture-parser/review tests passed. The live run reported mechanism success
in 19.481 s and product blocking after 993.46 s. After the parser correction,
targeted unit/overlay checks and offline review were rerun; the live stack was
not rerun. All assigned containers, images, networks and volumes were removed.

See [requests](requests.md) for owners and [RUN.md](RUN.md) for reproduction.
The #69 run is under [history/follow-up-1](history/follow-up-1/follow-up-1.md).
Earlier dirty-harness diagnostics are preserved separately and do not promote
the current product outcomes. No enforcement bypass is claimed.
