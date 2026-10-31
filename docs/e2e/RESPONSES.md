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
The [complete follow-up 17 run](runs/2026-10-31/followup17/README.md) passed
kernel/heartbeat/updater-lock equality and the ordinary process restart check;
installer apply still returned durable `failed/final_sample`, health null.

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
The [live plumbing row](runs/2026-10-31/followup17/a7c41436-82b3-4665-b75c-e2887f2e1009/capture-and-boot.md)
passed a real 204-byte origin capture, a complete 3,145,728-byte disk export and
explicit diagnostic overflow. These are helper checks, not a rerun of the
isolation lane's full acceptance matrix.

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
`apps/portal/e2e/dev-stack.mjs` directly. The 15 audit rows below group concrete
writes in that simulator; their numbering is not a row-for-row match to the
absent table. No portal
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

## enrollment — 2026-10-31

> # Enrollment acceptance requests — follow-up 5

ENR-06 now reaches successful initial installation and an `already_current`
rerun on the fixed NetBird socket/runtime layout. Digests serialize with
`*_sha256` names and retain `absent`; `current_target` is symlink text.
`InstallerOutcome.Services` captures active consumers before sampling and
those started by the installer. Call `RestoreInstallerServices(ctx,
"installer-node", out.Services)` after comparing quiescent files, including
when the wrapper returns an error. It restores captured active services with
read-back and leaves an inactive updater oneshot inactive. The shared runtime
and NetBird socket survive sampling. [Contract and example](installer-layout.md).

ENR-05's `HoldUsageAcknowledgements`, `UsageLedgerSnapshot` /
`UsageLedgerSnapshotBytes`, `MemberTransportStack` and `CountedMemberTraffic`
now accept `installer-node`. The hold is on the authenticated installer proxy
at loopback 9444; the CLI uses `/opt/vpn-node/current/fvpn-agent` and
`/etc/vpn-node/agent.json`. `ConfigureInstallerServing` supplies the real
AdGuard backend, and the focused campaign composes publication, native member
join, persistent TCP/UDP, counters and held usage without writing observations
or ledger rows.

The live attempt exposed an additional **product blocker**: the installer
creates an empty `inet fvpn owner` chain, but enforcement requires exactly
four chains and reports `HOOK_PRIORITY_MISMATCH`. Publication and the counted
member prerequisite do not complete. Nonzero held usage and restored traffic
therefore remain unproved. The harness leaves the marker and verifier intact;
the owner is installer/enforcement (`packaging/installer/isolation.go` and
`node/enforce/verify.go`), outside this lane's paths.
[Actual marker and agent failure](runs/2026-10-31/followup17/a7c41436-82b3-4665-b75c-e2887f2e1009/installer-owner-chain.md).

Authorized old-incarnation backlog delivery remains a separate unresolved
usage/installer protocol. Disk preservation cannot establish once-only central
totals or permission to send retired events under the new certificate. The
full journaled installer upgrade also remains failed at the genuine missing
meter checkpoint; no replacement checkpoint is supplied here.

In [run a7c41436](runs/2026-10-31/followup17/README.md), controlled reinstall
also exited 0 (`reenrolled`): same Node ID, different incarnation, retired mode
0700, retained identity/bound-state digests, unchanged usage DB/WAL digests,
and HTTP 403 for the retired certificate on both direct and proxy paths.
Service restoration succeeded. Nonzero historical usage and forwarding after
restoration are not claimed from those file/service results. The complete
campaign remains exit 1 (8 pass / 1 blocked / 3 fail).

## usage — 2026-10-31

> # Requests — Follow-up 4
>
> Usage — the D-105 hand-off path

The current usage request file says “None”; the second quoted line is the
orchestrator follow-up 18 assignment. The shared exports already support the
controlled same-node path. For the requested usage AT20 hand-off case (lane
label; accounting contract §10.5–§10.8), use this sequence:

1. Use `PrepareInstallerNode`, `RegisterInstallerReleases`, real publication
   and `MemberTransportStack(member, "installer-node")` to establish counted
   traffic. Hold `HoldUsageAcknowledgements(ctx, "installer-node", true)`;
   wait for nonzero `UnackedEvents[].Records` and save the CLI snapshot's
   `NodeIncarnationID`, `LedgerEpoch`, `LastEventSeq`, `AckedThrough`, event
   sequence/hash and byte totals. A nonempty payload is not sufficient evidence
   of nonzero member bytes.
2. Save `InstallerServices`; stop `fvpn-updater.timer`, `fvpn-updater.service`,
   `fvpn-watchdog.service`, then `fvpn-agent.service` inside the installer
   container. This is the same quiescence order as `RerunInstaller`. Keep the
   usage hold throughout. Take two `UsageLedgerSnapshotBytes` calls and require
   byte equality. Do not use the ordinary node's supervisor hook in systemd.
3. Through `Stack.Call`, GET `/api/v1/admin/nodes/{id}` and POST
   `/api/v1/admin/nodes/{id}/revoke` with its ETag in `If-Match`, a new
   `Idempotency-Key`, and `{"reason":"TEST ONLY controlled reinstall"}`.
   Expect 202, then GET `/api/v1/admin/nodes/{id}/incarnations` until the origin
   is `retired` with `retired_at`. Wait for the returned operation's worker
   effects when checking external peer revocation. No SQL retirement shortcut.
4. POST `/api/v1/admin/enrollments` with `delivery:"installer"` and the **same
   node_id**, extract the secret token privately, call
   `HoldInstallerFirstPoll(ctx, true)`, then
   `ControlledReinstall(ctx, "installer-node", token)`. Require `reenrolled`,
   unchanged Node ID, different `InstallerOutcome.Incarnation`, matching retired identity/bound-state digests and retained origin canonical
   payloads, event sequence/hash and origin ACK. D-118 runs during replacement
   bootstrap: usage DB/WAL hashes may change before this wrapper returns.
   File equality after adoption is not the retention predicate.
   This wrapper leaves the services quiescent for sampling. Never rename or
   relabel the surviving events as the new incarnation.
5. Read the incarnations API again: it exports `incarnation_id`, `current`,
   `state`, `retired_at`, `retire_reason` and retained usage per incarnation.
   `InstallerOutcome.Incarnation` and the node's public `identity/node.json`
   identify the new presenter. The CLI snapshot identifies the **current ledger
   stream**, its `acked_through`, event sequence and payload hashes. After the
   D-105 store transition it describes the new stream only. Save the origin
   snapshot before reenrollment; the new stream's ACK is not the retired ACK. The API's usage `through` is a timestamp,
   not a sequence ACK.
6. `ConfigureInstallerServing`, release the first-poll barrier, and
   `RestoreInstallerServices` with the state saved **before** the manual stop
   (the reinstall wrapper sees those already-stopped units). Then release
   `HoldUsageAcknowledgements`. D-105 must use the current certificate and
   top-level current `node_incarnation_id`, with `origin_incarnation_id` set
   to the retired origin; immutable events retain the old incarnation and
   hashes. Save the real batch ACK and read-only `usage_checkpoints` row keyed
   by `(node_id, incarnation_id)` to distinguish origin and current streams.
   The central `highest_contiguous_seq` belongs to that selected stream; no ACK may be
   borrowed from the new incarnation. Replay must leave origin totals unchanged.

For the **local retired ACK**, stop the services again and use the existing
`ExportContainerFile` to copy `state/usage.db` and its WAL (when present) into
one private scratch directory, keeping the filenames paired. No process may
write between the two exports. Open that coherent pair read-only and query:

```sql
SELECT incarnation_id, CAST(acked_through AS TEXT) AS acked_through,
       CAST(last_seq AS TEXT) AS last_seq, ledger_epoch, final_code
FROM retired_streams ORDER BY rowid;
```

This is evidence extraction, not retirement by SQL. The CLI does not yet expose
retired tables. `RetiredStreams`, `RetiredUnacked` and `HandoffRejections` are
product package readers, documented in [usage-store](../node/usage-store.md#retired-stream-hand-off-d-105-enr-05);
the shared fixture does not call an internal product transition to simulate
adoption. Existing CLI/file/API exports suffice for this hand-off path.

D-118 is now implemented in agent-core (#320), under the node lock and
`usage.db.lock`, and executes during replacement bootstrap. The wrapper can
therefore return after stream adoption has already changed the DB/WAL. Compare
immutable canonical origin payload bytes, origin event sequence and hash, and
the origin stream's retained ACK; keep the new stream's ACK separate. Usage AT
follow-up 5 (#324) consumed the controlled retirement/reinstall path. These
claims do not retroactively change the archived follow-up 18 file measurements.
The historical `8a1124a1` seed `HASH_MISMATCH` is retained in
[its diagnostic](runs/2026-10-31/followup18/8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179/post-upgrade-reenroll-diagnostic.md);
the combined-source seed result belongs to follow-up 19.

## portal / wire-nodes — 2026-10-31

> ### D-093 harness audit — 2026-10-31, Follow-up 18
>
> D093-12

This lane supports option (a), **a compiled TEST ONLY fixture fetcher**, if the
worker/release owners choose it. This is a view, not an implemented product
change. `internal/releases/verify.go` rejects local addresses and bypasses
proxy environment settings; `apps/worker/jobs_nodes.go` unconditionally creates
`NewHTTPSFetcher`. Merely registering a release over HTTP cannot bypass that
worker decision. The portal's SQL `verified` remains a fixture shortcut today.

Use a dedicated build tag such as `e2e_fixture_fetcher`, with mutually exclusive
factory files (`//go:build e2e_fixture_fetcher` / `!e2e_fixture_fetcher`). Keep the
strict default factory in every untagged build. The tagged implementation
should still run the genuine manifest validation, origin allowlist, signature,
length/hash verification and operation/read-back workflow; replace only the
bounded TLS transport to a selected owned fixture origin. Refuse redirects and
unlisted origins; never implement “always verified”. A corrupt artifact must
produce a failed verification through the same worker job.

For adoption, `mk/e2e.mk` would expose an explicit dev-only worker build mode,
not append the tag to global `GOFLAGS`. `deploy/devstack/up.sh` would pass it
only to `go build ./apps/worker/cmd/fvpn-worker`; the image context would carry
the worker binary plus source commit, `go version -m` build settings (including
`-tags`), binary SHA-256 and immutable image ID. Start the container by that
recorded ID, then compare `docker inspect .Image` and the running container's
`/usr/local/bin/service` hash with the receipt. Document the mode and both positive and
corrupt-artifact runs in `docs/e2e`. Production packaging/CI must use clean
untagged builds and fail if their build metadata includes this tag; those
outside-owned gates need the worker/release owners, not an e2e-only assertion.
The portal's non-Docker worker launcher also needs its own explicit tag and
binary receipt when adopting the factory.

An honest existing alternative is to point browser tests at this real Docker
stack's API-created releases and isolated fixture origin, which the current
worker verifies. This requires migrating the portal fixture setup and cannot
retroactively close its current localhost/SQL row. No public artifact hosting
or production SSRF relaxation is proposed. [All fifteen audit rows reconciled](portal-d093-reconciliation-2026-10-31.md).

## security-ops — 2026-10-31

> # Requests to orchestrator — Follow-up 12 (2026-10-31 UTC)

`labstack.Stack` now exports `ArmFirstDesiredOffer`, `ReleaseDesiredOffer`, and
`FirstDesiredOffer`. Rebuild the secops stack so its node image contains the
new `fvpn-e2e-nodeconfig` proxy. Stop the ordinary node with the supervised
`Stack.Service(ctx, "stop", "")`, arm the capture for that slot, then resume
with `Stack.Service(ctx, "start", selectedRelease)`; read `selectedRelease`
from `e2e/agent-selected` before stopping (normally `/usr/local`).
Arming returns a capture UUID and holds actual desired GETs with 503 while
observations can proceed. Perform the restore/retained-generation observation
ordering required by D-099, then release the hold. Do not issue diagnostic
GETs through loopback 9444 while armed: the first response must go to the
retained agent. Separate direct-control probes remain separate evidence.

Poll `FirstDesiredOffer` and require the matching capture UUID:

- `first.generation`, exact `body_sha256`, byte count, `delivered_at`, and
  `write_complete`, plus `http_status`: the first actual response flushed at the agent proxy,
  never a compiler-row inference. A 304 or error is retained with no generation;
  later 200 responses cannot overwrite it or turn it into a proven first offer.
- `local.accepted_generation`, `applied_generation`, `envelope_sha256`, and
  sample time: a read-only SQLite query of `state/agent.db` while processing
  the agent's subsequent observation. It is retained only for the exact first
  generation and digest. A successful write without this match is insufficient.
- `ack.report_id`, `applied_generation`, HTTP status and receipt time: the
  corresponding observation really reached control and got 200/202 while the
  local store matched. Join `report_id` to the central observation if the
  consuming lane also needs durable server-row evidence. A later generation's
  ACK cannot satisfy the first offer.

The HTTP-side marker lives solely in the test proxy; it is necessary because
SQL compilation and separate probes do not prove delivery. Local acceptance
is sampled, so a first generation superseded before the next observation stays
**unproved**, not inferred accepted. The capture does not freeze subsequent
compilation to manufacture a matching ACK. The focused live hold/release test
is not itself PostgreSQL-restore acceptance; SECOPS-21 still needs the secops
lane's greater-than-retained and restore guards. The live supplement
[5f74d7c3](runs/2026-10-31/followup18/5f74d7c3-6404-461d-9f47-296bd69958c2/first-desired-offer.md)
passes with first delivered generation 4 > retained 3, exact local digest and
applied generation 4, and a 202 receipt joined to the central report. A real
capacity PATCH/read-back supplies a new document in this test. Waiting for
compilation or generation advancement while GETs are held is invalid: both
occur on the agent GET. The earlier first-304 and polling mistakes are archived.

## members — 2026-10-31

> ### F8-01 — second-exit namespace restart does not regain serving

`RestartNodeNamespace90(ctx, slot, nodeID)` is exported through `labstack` for
`node` and `node2`. It stops the agent container, AdGuard and namespace holder,
then starts namespace, AdGuard and agent in order. It keeps durable state and
does not change generations or repair tc. The result contains before/after
central generations, the first boot nft/tc read-back, timestamped heartbeat
samples over the full first 90 seconds, all central observations in that
window, first fresh forwarding observation if any, final nft/tc and agent log.
It returns partial evidence on an error. Use only an established serving node;
check generation equality and a fresh serving observation before claiming
recovery. Agent-core follow-up 25 owns any product fix. The shared host kernel
boot ID remains unchanged; this is namespace/process restart evidence.
The current isolated second-exit fixture recovers at unchanged generation 3 in
5.96 seconds, despite capturing an initial tc missing-link rejection; all 90
seconds are retained. The original generation-30/full-mutation failure is not
closed by that smaller reproduction. [Live evidence](runs/2026-10-31/followup18/daf790fc-81f7-49a2-8a65-f3ec8faec656/namespace-restart-90.md).

## members — 2026-10-31

> ### F8-02 — MTU 1280 bulk transfer incomplete

The follow-up 18 native runner executes standalone dual 1500/1280 without
reconnect, then the consumer's IPv4/IPv6/dual 1500 → dual 1280 reconnect matrix,
plus forced IPv6 and IPv4 availability at 1280. It records transfer bytes,
counters and native candidate families, captures TCP SYN MSS and ICMP/ICMPv6
MTU messages on member and exit, and exports complete bounded pcaps with
size/digest receipts under private `.agent/tmp`. Packet contents are not
committed. [Measured verdicts and limits](runs/2026-10-31/followup18/README.md)
remain separate from the archived members failure. A successful rerun alone
cannot establish that reconnect caused that earlier failure.

## usage — 2026-10-31 (follow-up 19)

> Update docs/e2e/RESPONSES.md, usage follow-up 18 step 4 and its stale activation note

Corrected above: D-118 executes during bootstrap, so DB/WAL equality is no
longer required after adoption. Preserve canonical payloads and origin
sequence/hash/ACK, separately from the replacement stream and its ACK.

The two real reinstall runs retain [eight exact canonical origin events](runs/2026-10-31/followup19/usage-retention.md),
with origin sequence/hash and ACK before/after adoption plus coherent snapshot
SHA pins. The serving candidate run retains ACK 4; the stopped-agent run's
origin ACK advances 3→4. Neither is inferred from the replacement stream.

## DNS-publish — 2026-10-31 (follow-up 19)

> Devstack follow-up 16, request 1

`fixtureDNSNegative` now requires exactly RCODE 0 and zero answer types for
[BACKEND F8-CNAME/F8-SVCB](../dns/BACKEND.md), with the TEST ONLY profile's
F8-TTL5 synthetic SOA TTL 5. A/AAAA/HTTPS cite F8-A/F8-AAAA/F8-HTTPS; alias A
and wildcard A cite F8-A scopes C/W. Each synthetic lookup waits six seconds
for a previous five-second cache entry to expire, then records lookup elapsed
time and allows only the whole-second TTL decrement within that measured
lookup (plus one rounding second). Lookups taking five seconds cannot prove
the TTL. NODATA is not NXDOMAIN. The absent authority name, positive cache
expiry and rebinding controls keep their separate assertions.

The combined-source [live DNS row](runs/2026-10-31/followup19/5282829c-c11e-4f34-954a-5bc2a5bceca5/dns-negative.md)
passes on UDP and TCP: both exact NODATA rows have RCODE 0, zero answers and
SOA TTL 5, and the matching NXDOMAIN rows have TTL 5. Synthetic lookup times
are 31–46 ms after the explicit cache-expiry wait. The whole fixture campaign
is 15/0/1 because its separate MTU-1280 transfer stalls at 12,012 bytes; that
failure is retained and is not a DNS result.

## installer — 2026-10-31 (follow-up 19)

> Installer 5 evidence scope

[Installer follow-up 5](../node/install-followup5-2026-10-31.md) is isolated
installer-side evidence; it does not replace ENR-05. The real serving acceptance
remains devstack 18's scoped native traffic/ledger proof and the separately
recorded follow-up 19 combined-source upgrade rerun.

## isolation / agent-core — 2026-10-31 (follow-up 19)

> D-087 clean provisional recipe — agent-core + devstack.

The corresponding assembled central switch is **not available on this base**.
`internal/nodes/httpwire/state.go:RenewServingLease` constructs
`leases.NewService` directly with the production DB entitlement reader.
`internal/leases/leases.go` requires published + registered + ready + unfrozen
state before including grants. `SetMemberGrantsReady` and `/member-grants`
exist only in fake control/labcontrol; the assembled devstack runs the real
central HTTP handler. Holding a lease response produces a transport error;
it cannot create an authentic empty signed lease. Unpublish, mapping freeze,
changing entitlements, editing projector rows or re-signing a stripped response
would not be the requested clean fixed-generation trigger.

This item stops at the ownership boundary. Requested owner change:
`internal/nodes/httpwire/state.go` and `internal/nodes/httpwire/runtime.go` need a
TEST ONLY injectable grant-eligibility reader/factory, defaulting to the exact
production reader, that can withhold future grants without changing desired
bindings/generation or ready/published state. `leases.WithEntitlementReader`
already exists; the HTTP handler does not expose that seam. No production
bypass or synthetic accepted observation was added in this lane.

Once that owner seam exists, the exported harness sequence must be: arm false
on a clean unfrozen published generation with bindings; let the existing
populated lease renew naturally (false does not revoke it); retain the genuine
empty signed lease and every actual observation; require ready/passed, exactly
`awaiting_member_grants`, the three AWAITING checks and one lease error; release
true only after the qualifying report is accepted; require a populated lease,
a distinct accepted serving report, unchanged generation/bindings and counted
native traffic. The agent-core 27 labcontrol proof remains producer evidence,
not an executed central/devstack trigger. Isolation's assembled AT10 remains
unproved here.

## orchestrator — 2026-10-31 (follow-up 19)

> D-124 fixture

On source `33034048`, `tests/e2e/cmd/fvpn-e2e-prepare/agent_config_test.go`
contains only config/hosts tests and never calls `agent.Decide`. No file in
that command directory calls it. The referenced model
`internal/nodes/publication_evidence_test.go` already sets both desired and
lease future expiries. No unrelated fixture or assertion was changed. Route
the actual stale fixture path if another file was intended.

## updater / installer — 2026-10-31 (follow-up 19 measured result)

> Updater 17 request 1 / installer 6 request 1

The unchanged serving fixture passes **14/0/0**, initial apply **0**, on the
combined source: [d6aaa798](runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.md).
The real member allow-set is withdrawn while mTLS control returns 200; the
five-field checkpoint's barrier UUID equals the persisted latch under its
original boot/deadline. The actual meter epoch closes with `final_sample`,
and the sealed receipt matches its durable watermark 4 / ACK 3. This is a
quiet final sample, not a newly generated post-withdrawal byte event. The
candidate pointer/executable switch, all five health checks and canonical
receipt hash pass. Installer check uses `updater_journal` and keeps install
history. [Candidate revoke/re-enrollment](runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-reenroll.md)
seeds successfully with no HASH_MISMATCH or tree repair, retains canonical
origin events and retired identity, restores services and refuses the old
certificate (403 through both paths).

The old stopped-agent sequence separately yields **13/0/1**:
[04de29d3](runs/2026-10-31/followup19/04de29d3-e56b-4a32-b678-21462845ddb6/installer-quiescent-candidate.md).
A stopped agent cannot respond to D-125's new latch. Initial apply and sealed
replay return **1**, recover returns **2**, pointer stays unchanged, forwarding
stays blocked, and coherent journal and receipt bytes remain identical.
Its later seed succeeds on the unchanged initial tree; that is not candidate
seed acceptance. The serving run supplies that proof. All prior devstack 18
artifacts remain unchanged, with the byte-identical archive and pins linked
from the [run overview](runs/2026-10-31/followup19/README.md).

## netlab — 2026-10-31 (follow-up 19)

> Netlab 19 request 2: exact F8-02 failed/control decode

All four requested raw pcaps exist here and match their exact byte/SHA receipts.
[Full payload-free decode and completeness limits](runs/2026-10-31/followup19/capture-decode/README.md)
retain TCP sequence/ACK/SACK/window/options/length and outer IP fragmentation
headers for every packet. The failed 22,022-byte flow's member ACK stops at
10731373; the missing 1,228-byte inner segment is retransmitted at node tunnel
egress but absent at member. Associated outer IPv4 packets are unfragmented
1,340-byte packets on the node's 1500 link, absent at the 1280 member link.
Receive-link MTU rejection is a bounded inference, not a measured kernel drop
reason; the failed pair has no retained tcpdump drop statistics or endpoint
samples. The original 11,011-byte members row has no matching capture here.

A [fresh same-order reconnect capture](runs/2026-10-31/followup19/capture-fresh/README.md)
retains all 12 member/node/origin pcaps with header decodes, actual MTUs,
process bounds, capture counts, and endpoint read/write/line/error/TCP_INFO.
It passes all four 64,064-byte transfers under the unchanged 12-second bound.
Its 1280 reverse path sends 1,006-byte TCP / 1,132-byte outer packets; all
arrive. In the opposite direction, 52 fragment pairs match at both ends.
This successful control does not clear either historical failure or support
a global MTU/MSS change. Both dual rows require IPv6 refusal without uplink
probe leakage plus IPv4 completion; underlay family is not overlay IPv6
service. There is no separate bottleneck router in this topology. No product
or NetBird MTU, MSS or policy was changed.

The intervening DNS fixture campaign also retains a genuine 12,012-byte
MTU-1280 timeout with 12 complete echoes and no partial final line. One
[additional captured reconnect attempt](runs/2026-10-31/followup19/capture-repeat/README.md)
again completes all four transfers. No instrumented failure was captured;
per-I/O tracing can affect scheduling/segmentation. These controls do not
erase the failures or establish the exact kernel drop reason.
