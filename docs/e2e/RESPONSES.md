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

## updates — 2026-10-31

> # Requests to orchestrator — follow-up 9

F9-06: the ordinary lab entrypoint now reads the kernel boot ID, matching the
packaged updater. Remove the acceptance lane's random-to-kernel write adapter;
an optional read-only equality assertion remains useful. Container restarts
keep this ID and are process-restart evidence, not fresh-kernel evidence.

`BuildInstallerRelease` builds real root-layout amd64/arm64 archives with the
unchanged packaged units and the lab release signer. `PrepareInstallerNode`
uses these for its older/newer fixtures. `RegisterInstallerReleases` uploads
them, calls the admin release API and waits for worker verification. Activate
support policy separately. `HoldInstallerFirstPoll` withholds only desired-state
GETs while allowing genuine bootstrap. `HoldInstallerApply` exposes the runtime
systemd condition barrier; releasing either does not start a unit.
`InstallerServices` / `RestoreInstallerServices` make the sampling lifecycle
explicit. [Exact paths, invocation and limits](installer-layout.md).

The missing genuine `state/meter.checkpoint` remains a product prerequisite.
These exports never supply one and do not promote an updater process exit 0
over a failed durable result. Candidate restart/health remains unverified until
the barrier is supplied by agent-core and the real apply succeeds.

## isolation — 2026-10-31

> ## Follow-up 7 requests — 2026-10-31

The origin image now includes `tcpdump` via an origin-only image build argument.
`Stack.ExportContainerFile` streams raw capture bytes to a private mode-0600 disk
file, atomically publishes on success, and returns size/SHA-256. The diagnostic
`lab.docker` explicitly errors above 1 MiB; it no longer returns clipped success.
The owned isolation receiver adapter calls the shared stream export. Stop or
quiesce the writer before final capture, then redact the private file before
committing it. A read failure cannot replace a previously complete destination.
[Shared export contract](installer-layout.md#complete-capture-exports).

This does not close AT11 hardware/platform coverage or AT13's continuous
crash/NetBird/OS-reboot/updater matrix. Its process restart assertion now expects
the same kernel ID and a fresh serving heartbeat.

## security-ops — 2026-10-31

> # Requests to orchestrator — Follow-up 11 (2026-10-31 UTC; live execution pending)

AT29 guest-clock rollback is an **operator-run row**. No VM is provisioned by
this Docker harness and no shared-kernel container clock is changed. The
[operator procedure](at29-guest-clock.md) requires an independently clocked
disposable Ubuntu guest, isolated control/member/origin, genuine signed finite
deadlines, guest-only wall-clock rollback with the agent killed, independent
monotonic timing and continuous TCP/UDP receipts, then recovery ACK. It remains
blocked until those real traces exist. There is no simulated clock pass.

## portal — 2026-10-31

> **Portal D-093 audit read** (portal follow-up 17)

I read `docs/portal/DEV.md` on this checkout. It documents D-101's simulated
kernel/QoS/usage/updater evidence but does **not** contain the requested 15-value
D-093 table. The classification below therefore audits the current
`apps/portal/e2e/dev-stack.mjs` directly against the 15 groups in the request;
it does not claim to have read a table that is absent from this base. No portal
file is changed. Live values below mean the real agent/control/worker paths in
the assembled privileged devstack, not the portal's mTLS reporter simulator.

| Harness-written group | Real evidence available now | Extra lab or fixture boundary |
|---|---|---|
| 1. `internet_only=passed` | Actual agent baseline/ruleset verification and accepted check, with nft read-back | Adversarial bypass, spoof, private-destination and continuous-fault coverage needs the network lab; a passed check alone does not prove the matrix. |
| 2. `meter_path=passed` | Actual nft device admission counters, sampler and check | Fragments, offload/bypass and failure attribution need netlab stimuli; no constant passes. |
| 3. `tunnel_probe=passed` | Agent probe outcome/detail plus real native NetBird path and independent origin receipt | Full NAT/platform/QUIC matrix is unmeasured. A status/handshake string does not establish end-to-end traffic. |
| 4. `provisioning_state=ready` | Product observations and node API projection after actual bootstrap/apply | Do not force ready while awaiting grants or failing safety. |
| 5. `safety_status=passed` | Actual agent decision and accepted central observation | Fault rows require real missing policy, lease, isolation or meter conditions. |
| 6. `data_health=healthy` | Actual datapath probe outcome and API projection | Unknown/degraded rows need corresponding probe behavior; not arbitrary UI constants. |
| 7. Applied config/policy/binding revisions copied from desired | Real application ACKs, observed generation and signed desired state independently | Copying desired into applied is never evidence. |
| 8. Forwarding boolean/block reasons | Agent decision, signed lease, current rules and counted persistent TCP/UDP | Deadline/crash/freeze tests need the network lab; wall-clock rollback needs the operator guest above. |
| 9. NetBird peer/version/backend | Real enrolled peer mapping, native status and kernel interface | Fake peer IDs and the portal's fixed version remain simulator values. |
| 10. Usage epoch/sequence/ACK/outbox counters | `UsageLedgerSnapshot` and agent usage observations, including nonzero held outbox and release | Exact canned counts/ages/byte totals remain display fixtures. |
| 11. DNS applied outcome, digest/generation and feed counts | Real signed document fetch/apply, AdGuard read-back, node evidence and controlled DNS queries | Fixed feed digest/count 1420 is fixture-only; negative/cache/rebind/transport matrix needs controlled lab authority. |
| 12. Current/last-good release IDs | Installed tree and durable updater status, accepted product observations | Must not advance last-good from install, stage or a process exit alone. |
| 13. Updater state/stage/result/health/checkpoint | Genuine updater journal and failure results; regular devstack update scenarios exercise real binaries | The ordinary lab's `e2e/meter.json` is still a TEST ONLY checkpoint adapter. Fully journaled installer success is blocked by the absent genuine meter checkpoint; fixed all-passed health and synthetic seq 10/9 are fixture-only. |
| 14. Direct-SQL `release_verifications` | `BuildRelease` / `BuildInstallerRelease` → `RegisterRelease`, real downloads/signatures and worker verification read-back | Replace SQL seeding with that public workflow for live claims. |
| 15. Direct-SQL `usage_rollups` | Real counted member bytes → durable events → accepted batches → worker rollup/member usage reads | The exact >2^53 totals and canned estimated/gap flags are fixture-only display/precision cases. Do not generate petabytes or mutate live projector rows to obtain them. |

The real paths can replace simulator inputs in a separate portal integration
run; they cannot retroactively make the existing browser fixture rows live-node
evidence. The shared stack uses TEST ONLY IdP/payment fixtures; their external
provider behavior and physical arm64/mobile/desktop coverage are not established
by the live-node checks above.
