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

## isolation / wire-nodes / agent-core — 2026-10-31 (follow-up 20, D-102)

> D-087 / D-131: export the clean false → natural empty renewal → accepted
> provisional report → true → populated lease / serving transition.

The seam is now wired into a dedicated TEST ONLY composition of real control,
using exactly `WithTestOnlyGrantEligibilityReaderFactory(grants.ReaderFactory)`
and one `nodestest.NewMemberGrantEligibility()` instance. Prepare with
`FVPN_E2E_GRANT_SEAM=1`; the [export contract](scenarios/harness-exports.md#follow-up-20--d-087-f10-02-and-f10-03-test-only)
documents `SetMemberGrantsReady`, `MemberGrantEvidence` and
`MemberGrantInvariants`, including the required receipt ordering. The production
binary, desired compiler and entitlement projector are unchanged.

[Run 5ab1f11c](runs/2026-10-31/followup20/5ab1f11c-a36b-4b77-bc70-85d902c93da6/member-grants-fixed-generation.md)
passes 7/0/0. Generation 7, desired envelope/ETag, bindings, ready/passed state,
publication, freeze flag and entitlement/device outputs remain byte-identical.
The real empty and populated signed leases join to distinct accepted reports;
true → serving receipt is 3,980 ms, followed by a complete 64,064-byte native
transfer and increasing admission counters. No generation bump or state repair
is made to drive the transition. This supplies isolation 10's central trigger;
it does not replace isolation's assembled AT10 result or historical durations.

The final [original-wire capture 79e59c20](runs/2026-10-31/followup20/79e59c20-f289-4679-9b37-01f060f2d376/member-grants-fixed-generation.md)
also passes 7/0/0, at fixed generation 6. All 47 captured original JSON body
hashes verify across 53 events. Both post-arm accepted reports remain ready /
passed at generation 6. The 103 original detached node timing records have no
sequence gaps and identify both renewals as newly accepted leases. True →
accepted serving receipt is 4,112 ms. The baseline generation differs between
fresh fixtures; neither is artificially forced to a particular number.

The [first attempt](runs/2026-10-31/followup20/b4d2d001-a781-4df5-80d2-f7aad09f5b55/member-grants-fixed-generation.md)
remains failed: normal initial FUP projection changed rate-policy generation
1→0 and desired generation 5→6 during the window. The runner now lets initial
outputs settle for 65 seconds before arming, without stopping either compiler.
It also corrects its lease response decoder to the actual `signed.members`
field. No failed evidence is rewritten.

## members — 2026-10-31 (follow-up 20, F10-02 / F10-04, D-102)

> Retain the counter-reader error category and correlate every sample with
> enforcement/read timestamps; preserve the original failed finite-expiry row.

The shared reader now retains category, stage, exit code, start/completion,
kernel generation and both surrounding heartbeat reads. The session-status
reader independently records its own failure category; consumers can distinguish
which half caused `read_ok=false`. Missing counters remain distinct from failed
reads. `EnableEnforcementTiming` before enrollment also enables the existing
agent observer: each read carries same-boot monotonic bounds and before/after
enforcement sequence records, with the full detached trace exported separately.
No retry or assertion relaxation is introduced. The exact fields and
category limits are in the [export contract](scenarios/harness-exports.md#follow-up-20--d-087-f10-02-and-f10-03-test-only).

[Run 218e18c9](runs/2026-10-31/followup20/218e18c9-50ee-4050-a60e-a96e188b0167/finite-expiry-reader-diagnostics.md)
retains 214 finite-expiry samples: all counter categories are `none`, both readers
succeed, and all heartbeat reads are present. Last observed TCP/UDP increments
are 182.021266 ms before the finite deadline. Senders stay active, final counters
are stable/removed, and sampling continues six seconds after expiry. This is a
reader-diagnostic reproduction, not a rerun of the complete members AT02 row.
Container-visible host load is 8.94 / 10.61 / 11.69; the historical run has no
matching retained load measurement, so comparable host load is unproven.
The original 13:35:45.333735552Z error was not retained and cannot be recovered
retrospectively. F10-02 remains historically failed and unattributed.

The extended [timing-enabled run 977efa83](runs/2026-10-31/followup20/977efa83-e3bd-4ced-b548-dbc9763ed266/finite-expiry-reader-diagnostics.md)
passes 2/0/0: 247 samples through deadline +16 seconds, all categories `none`.
All 247 same-boot monotonic nft-read brackets and all 494 surrounding producer
records join exactly to the 242-record full enforcement trace. Host load was
16.45 / 13.65 / 13.03. No unreadable counter sample was reproduced. This preserves
the original every-read-success criterion and gives members 11 the timestamps
needed to attribute a future failure; it does not repair the historical sample.

F10-04: the shared hooks preserve the generation the product actually issues.
This task adds no mutation to force generation 30. Members' generation-29 full
sequence and original generation-30 recovery remain distinct records. Exact-30
history reconciliation belongs to the orchestrator and the generation-producing
owners; neither this clean D-087 run nor another passing subset closes it.

## updates — 2026-10-31 (follow-up 20, F10-01, D-102)

> Align regular release artifacts, supervisor and shared assertions with the
> installer release-root contract; use the actual product meter checkpoint.

Regular bundles now contain root `fvpn-agent`, `fvpn-updater`, `fvpn-watchdog`
and `release.json`. `InstallerLayout` adds the installer binary and original
units. The supervisor separately handles `/usr/local/bin` bootstrap and exact
`/opt/vpn-node/releases/<id>/fvpn-agent` candidate ownership. Shared executable
assertions use the root path; product executable/journal validation is intact.
Both shared recovery configurations use `/var/lib/vpn-node/state/meter.checkpoint`
(the default from `node/agent/config.go`, documented in `docs/node/usage-upload.md`).
The fixture no longer manufactures `e2e/meter.json`. A regular update keeps the
agent alive for the genuine fresh D-125 checkpoint; the updater owns stop/start.

The regular [run d6b01926](runs/2026-10-31/followup20/d6b01926-de6f-47e4-803b-ae3ea5c603df/update-good.md)
passes **11/0/0**, exit **0**: twenty supervisor cycles, exact root candidate
identity, apply exit 0, five passing health checks with the issued real probe,
tampered-artifact rejection, and restart without enrollment. This is the
regular shared fixture rerun; updates retains ownership of full AT23–AT27.

Updates 11 must consume this layout in its own `tests/e2e/at/updates/live_test.go`
(candidate-running and tampered-artifact absence checks still name `bin/`),
then run the complete AT23–AT27 sequence. The initial checkout's recipient
`lab_test.go` also had the legacy checkpoint, but updates 10 has since corrected
it on main; no duplicate request is made for that file. Recipient adapters were
not edited here. D-102 keeps this answer in RESPONSES.md and their registers
with that lane. Updates 10's separate F10-02 generated `result_url` request is
installer-owned (`packaging/installer/config.go`); this regular-fixture pass
does not close it, and no packaging file is changed here.

## security-ops / enrollment / usage — 2026-10-31 (follow-up 20, D-102)

The merged secops 13 register reports SECOPS-21 closed using its own restore
run and the shared first-offer export. No new export discrepancy is reported.
Independently clocked disposable guests and full candidate-hardware capacity
remain operator resources; this lane never changes the shared clock or claims
physical arm64 acceptance. Manual grant revocation remains the wire-members /
contracts API item; D-087 future eligibility suppression is not a substitute.
Enrollment 9 reports zero new requests. Usage's D-118 canonical-event/origin-ACK
clarification was already answered in follow-up 19; the DB/WAL file hashes are
not preservation invariants across real adoption. No recipient register changes.

## isolation — 2026-10-31 (follow-up 20, remaining follow-up 9 requests, D-102)

The D-087 export above supplies the requested clean central trigger. The new
ISO-F6-03 baseline-writer question remains with `node/agent/engine.go` and
`node/watchdog/watch.go`; the shared detached Engine trace cannot identify a
concurrent watchdog writer that the product does not emit. No synthetic writer
or timestamp is added here. The 96 early drops, 65 subsequent positives and
41 final expiry drops remain the isolation lane's failed evidence. Its three
renewal losses retain their measured command brackets without a tolerance.
Physical platforms, OS reboot and the full continuous fault matrix still need
their designated operators. The native dual-underlay fixture measures IPv4
overlay traffic and IPv6Block refusal; it cannot qualify a member-platform
matrix. Recipient request registers and immutable follow-up 8 pins stay intact.

## netlab / members — 2026-10-31 (follow-up 20, F8-02 instrumented failure, D-102)

The requested broader fixture now has an **instrumented failed transfer**:
[run cce1a688](runs/2026-10-31/followup20/cce1a688-6af5-483c-bd67-2d167c51c04b/native-matrix-inputs.md),
**10,010 / 64,064 bytes in 12,011 ms** at MTU 1280; the preceding 1500 control
passes 64,064 bytes in 2 ms. We bounded the campaign at three fresh attempts,
stopping after this first captured failure. No further attempt was needed.
The original reconnect/input ordering, deadline, IPv6Block controls and bulk
criterion are unchanged; no global MTU or MSS adjustment was made.

[All six complete decodes and analysis](runs/2026-10-31/followup20/capture-cce1a688/README.md)
retain member/node/origin header records, original pcap pins, tcpdump statistics,
process windows, link MTUs, endpoint counts and TCP_INFO. The first missing TCP
range is [1784148912,1784150140). Origin and node tunnel egress retain its
retransmissions; the member keeps ACKing that gap while SACKing later data.
Nineteen captured reverse outer IPv4 datagrams of length 1,340 are absent at
member; all seventeen of length 1,132 arrive. No fragmentation-needed/PTB is
captured. The origin has received the full upload at its socket, then develops
write backpressure while response bytes remain unacknowledged. This is stronger
than a SYN-only or endpoint-count hypothesis; it does not name an exact kernel
or virtual-link drop reason. Receive-link MTU rejection remains an inference.
Capture/filter differences and GSO/GRO limits are explicit in the analysis.

Per-I/O tracing can alter segmentation, but this attempt proves instrumentation
does not always remove the failure. It neither reproduces the exact historical
12,012/11,011-byte boundary nor changes those failed records. Netlab 20/21 notes
were not present in this checkout or main when checked; this hand-off supplies
the requested failure evidence for netlab 21's interpretation. IPv4 overlay is
measured over the selected native underlay; no IPv6 service or ctv6 hit is invented.

## members / dns-publish — 2026-10-31 (follow-up 20, F10-03, D-102)

`ProbeNativeDNS` now returns per-query `dns_trace`: query bounds, resolver-role
receipts, authority exchanges, TTL remaining and separately sampled authority
SOA serial/time. All three metadata relays retain only closed synthetic case
labels and response classes, not full qnames, answer addresses or payloads.
Resolver query logging/statistics remain disabled. No cache flush or forced
responder was added. See the [export contract](scenarios/harness-exports.md#follow-up-20--d-087-f10-02-and-f10-03-test-only).

The [cce1a688 DNS record](runs/2026-10-31/followup20/cce1a688-6af5-483c-bd67-2d167c51c04b/dns-negative.md)
passes the unchanged assertions, with **36/36 query traces complete**. Initial
positive: primary, `other_ipv4`, TTL **10**, observed authority exchange,
authority serial **1001** after the query. Before TTL: primary again, old class,
TTL **9**, no authority exchange, authority serial **1002** after the query.
After TTL: primary, `private_ipv4`, TTL **10**, observed exchange, serial **1002**.
The negative before-TTL response retains SOA serial **1001** and TTL **4** while
the separately queried authority is already **1002**. Primary-down fallback
also identifies a real secondary response; its trace is explicitly incomplete
because the stopped primary container cannot be read during that query.

Wording for dns-publish 9 / members 11: a unique metadata match identifies the
observed responder; multiple matches stay ambiguous. An authority exchange is
positive evidence of an upstream fetch. No exchange supports an **inferred**
cache hit for a positive answer, not a private AdGuard cache flag. Negative
policy answers can be locally generated, so their cache state stays unknown
unless separately attributable from the retained receipts. A positive cached A
record has no SOA; `authority_serial_after_query` must never be represented as
that cached record's serial. Native probe milliseconds and the broader wrapper
bounds are distinct; trace collection itself adds elapsed time.

F10-03's historical 2,502-ms initial query and premature changed answer remain
failed and unattributed: this passing run cannot recover its missing responder
identity or prove a product TTL fix. Members 11 can retain these fields through
the existing shared result serialization, keeping its before-TTL assertion
unchanged. No recipient harness, request register or historical row was edited.

## usage / updates — 2026-10-31 (follow-up 20, newly merged registers, D-102)

Read-only reconciliation against main `838e8eb2`: usage 6 explicitly closes
its prior canonical-event retention request and reports no new requests. Its
fresh D-130 gap and counted-event result is its own acceptance, not a new
shared fixture measurement. Updates 10's newly merged register is answered by
the F10-01 layout/result section above. Its actual apply configuration already
uses the genuine checkpoint and keeps the agent alive; only its two executable
path assertions still need the release-root adoption. F10-02 generated installer
reporting remains with its named packaging owner. No other lane's register was
edited, and this worktree was not rebased onto those newly merged changes.

## updates — 2026-10-31 (follow-up 21, F11-01, D-102)

The shared `installerSpec` now supplies the explicit TEST ONLY loopback opt-in
for its existing `https://127.0.0.1:9444` authenticated proxy. `installArgs`
requires both that boolean and the exact local origin before adding
`--allow-loopback`. Production defaults and generated reporting fields are
unchanged. [Fresh shared-wrapper run 33b9621a](runs/2026-10-31/followup21/33b9621a-41d7-49fb-bc27-1c3f05e4ab44/installer-loopback-wrapper.md)
passes 3/0/0: actual install exit **0**, status `installed`, generated
`result_url=https://127.0.0.1:9444/api/v1/agent/update-results`. The generated
configuration SHA-256 stays
`14a8975a8a36df9558b9b957e07613476e43de351281eadaae4657e860f06164`.
No-client-certificate POST returns **403**; the same deliberately invalid empty
receipt with the generated identity/server CA reaches central validation (**400**).
This verifies the authenticated endpoint and unchanged configuration, not a
new successful receipt delivery. Updates 11 retains the real automatic-delivery
proof. The [run overview](runs/2026-10-31/followup21/README.md) also preserves the
initial setup failure from sharing a stack with the third-node fixture.

F10-01 needs no further fixture change: updates 11 consumed the regular
release-root correction and measured installer 27/27 and regular AT23–AT27
5/5 without the former reporting adapter. Its F10-02 generated-reporting proof
also stands. Updates F11-02's obsolete offline-install paragraph belongs in
`docs/updates/node-updater.md`, outside this lane: updater/installer should
explain initial/same-release retained verified archives and distinguish the
uncached fallback. No artifact-origin change is warranted by a rejection that
did not occur; the supplied exact-origin/private-address interpretation is
accepted. This answer does not claim private-mirror deployment coverage.

## dns-publish / members — 2026-10-31 (follow-up 21, F9-INFLIGHT, D-102)

The shared trace now writes a request start before dispatch, retains outstanding
requests and joins later completions by identity across the entire closed-case
cache-change sequence. The [export contract](scenarios/harness-exports.md#follow-up-21--installer-input-and-dns-lifetime-correlation-test-only)
defines cumulative history, three outstanding sets, receipt continuity and
per-role snapshot bounds. Current-query `trace_complete` cannot rule out an
earlier outstanding request. A reply between queries remains visible in the
next snapshot even if the last query starts no fetch.

[BACKEND §2.1.5](../dns/BACKEND.md#215-f10-03--positive-cache-and-ttl-measured-2026-10-31)
is adopted: fetch alone is not fallback; different uniquely matched responders
establish independent caches; failure/timeout plus secondary receipts establish
fallback. Same-resolver changed class without a new fetch requires inspecting
older outstanding requests **and their completion times**. Missing history
keeps the cause ambiguous. Hit wording stays inferred, negative policy synthesis
stays distinct, and post-query authority SOA is not a cached positive serial.
The pin's F9-INFLIGHT replacement limitation remains a product finding.

[Final DNS run cba88bbc](runs/2026-10-31/followup21/cba88bbc-a7bf-49f8-9dab-3182123d02aa/dns-history-integrity.md)
passes **13/0/0**, with **36/36** complete query traces, **85** distinct authority
requests and **70** identities retained across query snapshots. The positive
sequence remains primary fetch / inferred primary hit (TTL 9) / post-expiry
primary fetch. This validates the export; it does not reproduce F9-INFLIGHT.

The fixture explicitly retains its historical **4 KiB (`4096`) cache** for
comparability, not the pin's 4 MiB default. Both resolvers' actual cache/mode/
timeout values are read before and after the focused run. The relay's 2-second
exchange deadline and AdGuard's live upstream timeout are distinct. No cache,
timeout, policy or TTL assertion is changed, and no TTL-fix claim is made.
`cce1a688`, the F10-03 historical failure and follow-ups 16–20 are untouched.
The [114-file D-098 pin set](followup20-archive.sha256) was committed first.

## members / agent-core / netlab — 2026-10-31 (follow-up 21, F11-01–03, D-102)

F11-01's [fixture analysis](scenarios/followup21-members.md#f11-01-the-first-unobserved-segment)
places the first unobserved segment after member inner `fvnb0` egress and
before node observable inner ingress. Five member packets, zero matching
node/origin packets, generation-5 zero admission/diagnostic counters and no
origin request are retained. The fixture's inner TCP filter cannot observe the
encrypted outer path, handshake, relay or link drops; it cannot assign the
missing transport sub-hop. This is not a generation-0 baseline or proven node
enforcement drop. Netlab 21's transport ownership remains; AT01 stays blocked.

F11-03's [original input/timing extract](followup21-members.json) and
[health-check analysis](scenarios/followup21-members.md#f11-03-what-the-failed-health-result-actually-asserts)
retain the exact two injection inputs and twelve generation-13 Applies for
agent-core 29. All eight failed Applies have `install_ok=false`, 184–304 ms
render/install brackets against the unchanged 100 ms bound, unreleased guard
and no verification records. `HEALTH_CHECK_FAILED/stage=apply` is the sanitized
Apply error, not an observed failed tunnel challenge. Capture/injection work
can add load, but historical command error/exit, resource pressure and injection
monotonic bounds were not retained; causation is not established. The fixture
cannot recreate the original full model/transaction from a later stack. No
health criterion, fixture timing or positive sibling assertion is changed;
the correct revoked-source drop does not pass the failed sibling control.

F11-02 is a members readiness prerequisite: a current-generation unexpired
lease with zero grants and `AWAITING_MEMBER_GRANTS` is not serving. Future D-077
positive probes must wait for an accepted serving report joined to a populated
current-generation lease and the intended device's binding, then independently
assert counted forwarding. Keep the original generation-10 failed transfer.
The shared hooks do not manufacture grants or force generations. Members owns
its `forwardingGate` in `tests/e2e/at/members/forwarding_test.go`; no recipient
harness change or substitute passing rerun is made here. F10-02/F10-03 historical
failures remain unattributed, while members 11's fresh diagnostics remain passes.

## updater — 2026-10-31 (follow-up 21, F18-01, D-102)

[The negative scenario contract](scenarios/updater-negative.md) adopts updater
18's stopped-producer interpretation: apply/replay/recover **1/1/2**, sealed
`CHECKPOINT_FAILED`, blocked forwarding, unchanged baseline/pointer and receipt,
no candidate success. The old 13/0/1 campaign stays failed and archived. Future
negative D-125 coverage must require those facts separately from the unchanged
positive upgrade assertions. No updater behavior or assertion is relaxed.

## netlab / members — 2026-10-31 (follow-up 22, F8-02, D-102)

> **2026-10-31 — native transport owner, F8-02:**

The original members IPv4/1500 → IPv6/1500 → dual/1500 → dual/1280 mutation
and down/up sequence now retains member underlay/tunnel, selected node
underlay/tunnel/uplink and origin captures. Both fresh campaigns' original
four rows pass; selected family and packet sizing are reported per row. The
member-only MTU mutation is unchanged. Devstack 20 `cce1a688` and members 11
IPv6 history remain untouched. Supplemental asymmetric IPv6 rows retain the
failure: first run **10,010 / 64,064 bytes, 12,007 ms**; final run **16,016 /
64,064 bytes, 12,011 ms**. Process exits stay **1**; criteria were not weakened.

[Final capture analysis](runs/2026-10-31/followup22/9b3a89e7-fa86-426c-9737-ce55bf66aa77/analysis.md)
finds all **15** reverse outer IPv6 packets of **1,360 bytes** absent at member
underlay, while every smaller selected-endpoint packet arrives. First absent
TCP range **[3005476689,3005477917)** is at origin and node tunnel egress,
not member tunnel ingress. All 52 upload fragment pairs arrive. The first
campaign independently lost all **23** oversized reverse packets. Captures
have zero drops/truncations. Loss is beyond enforcement; receive-link MTU
rejection is an inference, not a recorded kernel drop reason.

A separate fixture-only selected sender `/128` route with **MTU 1280** makes
the path constraint explicit. Same peer, no reconnect or global MTU/MSS change.
Route readbacks prove installation and exact removal. The final three controls
receive **64,064 bytes in 1 ms** each; controls 1 and 3 deliver respectively
**7 and 44 reverse fragment pairs**, joined across the underlay, with complete
current-bulk TCP byte coverage. This strengthens the first campaign's separate
old-flow fragment receipt and new-flow success. **No PTB is seen at node,
tunnel or any other capture point**: sender fragmentation follows known route
MTU, not measured reactive PMTU discovery. A route constraint was chosen to
isolate sender sizing without introducing router/ICMP behavior.

This member-only topology is an **asymmetric receive-link fixture artifact**;
it has no router or sender signal and does not certify a real platform link.
If a real known-1280 receiver is established, peer MTU/packetization belongs
to pinned NetBird/WireGuard integration (`deploy/devstack/Dockerfile.node`),
not an enforcement MSS clamp. No production default or repair is inferred.

D-098 was first, commit `716f8144`: [63 immutable pins](followup21-archive.sha256).
The DNS third-node / installer `.24` collision remains handled by **separate
reset stacks**; slot layout is unchanged.

## members / netlab — 2026-10-31 (follow-up 22, F11-01, D-102)

> **2026-10-31 — members 12 and native transport owner, F11-01:**

`StartNativeCapture`, `Sample`, `Stop` and selected-peer readbacks are available
through `labstack` / `MemberTransportStack`. The [export contract](scenarios/harness-exports.md#follow-up-22--selected-native-path-capture-test-only)
specifies ready/stop and per-command monotonic bounds, route/link/ICE/relay/
WireGuard endpoint readbacks, private pcap receipts and payload-free joins.
Start before the existing route selection; sample during selection and the
original transfer. It does not reconnect, select an exit, wait for serving or
repair transport. Members 12 owns adoption in
`tests/e2e/at/members/forwarding_test.go`; that file is unchanged.

The [final fresh reproduction](runs/2026-10-31/followup22/9b3a89e7-fa86-426c-9737-ce55bf66aa77/analysis.md)
has **five SYNs**, sequence **1392714787**, source port 20001: member tunnel
packets **8, 144, 155, 160, 163**. None reaches node inner ingress or origin.
The HTTP attempt fails after **5,004 ms**; origin requests = 0, node upload /
download remain **0 → 0**, generation **5**, heartbeat serving=true. It runs
before the ordinary counted-flow retry. No node health repair is applied.

The first SYN's WireGuard initiation (member lo packet 9) has the same message
hash/index as node lo packet 133. Node eth2 packet 134 responds; the identical
response reaches member eth1 packet 137. Its IPv6 **source is loopback**, token
`addr-eff8e7ca506627fe`, distinct from the selected node endpoint. No structurally
valid WireGuard type-4 transport-data message appears at member (including lo)
or node during the request. Readbacks transition from local endpoint/pending
to IPv6 P2P connected while all subsequent inner SYNs still fail. The first
unconfirmed data step in this reproduction is **member inner tunnel egress →
observable WireGuard transport-data emission**, before node inner ingress.
The handshake response reaching member underlay does not prove kernel/session
acceptance. This is not evidence of an enforcement fault.

All three captures are complete (167 / 172 / 30 records), zero drops/truncation;
metadata reads now succeed. Every packet has an empirical monotonic interval
with explicit wall-clock stability assumptions. Node's later handshake tail
is after the member capture stopped and cannot prove receiver loss/recovery.
The loopback-source response and endpoint transition are handed to the
**native NetBird/WireGuard integration owner** (NetBird 0.82.0 binary pin in
`deploy/devstack/Dockerfile.node`; `node/agent/probe.go` only observes status).
Please trace handshake/session acceptance before assigning a fault or repair.

The first campaign's one-SYN success and missing origin/WireGuard readbacks
remain retained as history. The final run reproduces the failure pattern on
a fresh stack; it does not replay all members 11 history or identify those
historical packet identities. Members 12 should compare its own original
sequence using these exports before transferring a root-cause conclusion.

## members 12 — 2026-11-01 (follow-up 23, D-102, provisional D-137)

**Request 1 / F8-02 / AT09:** `PrepareNativeRouter1280`, `Select` and
`StartCapture` are exported through `labstack` for the prepared canonical
node/member pair. The [contract](scenarios/harness-exports.md#follow-up-23--routed-pmtu-native-debug-and-dns-delivery-test-only)
and [dated native analysis](runs/2026-11-01/followup23/a26d9b0f-bd1b-48df-9546-4105f9b3f328/analysis.md)
show both router-link ends at 1280, sender/router ingress at 1500, ICMP enabled,
and exact peer routes without an MTU metric. Both families receive **64,064
bytes in 1 ms**. IPv6 emits **one PTB**, received by the sender, whose PMTU
becomes **1280**; **84 fragments** arrive. IPv4's **34** oversized outer datagrams
have **DF=0** and become **68** arriving router fragments; **zero frag-needed**,
no learned PMTU reduction. This is transit fragmentation, not IPv4 DF discovery.

The original receive-only IPv4 row fails again at **10,010 bytes / 12,011 ms**;
the separate IPv6 row fails at **10,010 bytes / 12,006 ms**, both without PTB.
Those rows and historical F8-02 stay failed. The existing known-PMTU control is
unchanged. Per provisional D-137, members may qualify the tested sender-learnable
scope and report the silent receive-only black hole explicitly. No global MTU,
MSS clamp, complete platform AT09 pass or QUIC qualification is introduced.
Members owns adoption in `tests/e2e/at/members/forwarding_test.go`; no members
acceptance assertion or register was edited.

**Request 7 / F11-01:** the [same native analysis](runs/2026-11-01/followup23/a26d9b0f-bd1b-48df-9546-4105f9b3f328/analysis.md#f11-01-pinned-proxy--ipv6-p2p-transition)
reproduces five SYNs, **5,003 ms** timeout, zero origin requests and zero node
byte increments. NetBird 0.82.0's debug log records direct IPv6 endpoint setup
then retained relay proxy redirection. Member loopback initiation **8** joins
node loopback **164**; node response **167** joins member underlay **178**, with
source `::1`. The member emits no WireGuard type-4 data during that request.
The pinned upstream's `peer.Conn` → `loopback.ProxyWrapper.RedirectAs` → raw
IPv6 injection path supplies the captured remote-source/loopback-destination
initiation. This names the **native NetBird/WireGuard integration / upstream
NetBird** path and owner. It does not observe the kernel's exact rejection or
handshake-acceptance reason. No product transport, enforcement or health file
changed. Historical members packet IDs and verdicts remain their own evidence.
The exported debug decoder publishes allowlisted metadata; raw logs stay private.

**Request 11 / F12-03:** no negative-cache fixture, cache setting or assertion
change. The retained initial SOA TTL **2**, followed **1.228672687 seconds** later
by the same-primary positive refetch, still lies within whole-second expiry
uncertainty. The authority's older completion is retained, with no outstanding
exchange or switch to another resolver established. Existing lifecycle history,
stream IDs, read times, outstanding arrays and `authority_ttl` supply the quiet
window evidence; no additional quiet-window helper was found necessary here.
Members/dns-publish own choosing a repeatable initial quiet sample and a measured
interval safely below its remaining TTL after accounting for rounding and
command overhead. A later TTL-zero or too-short sample cannot be silently retried
to replace this failed combined row. No negative TTL extension or cache flush
is exported, and the original F12-03 result stays failed.

**Request 5 / F4-05 / F4-07:** the positive IDNA fixture is delivered:
`ProbeNativeIDNA(ctx, tcp)` converts the closed U-label, checks its A-label,
and probes it through the native stub. The [dated wire evidence](runs/2026-11-01/followup23/9bb2098e-1c6f-48a8-9fbd-fd99bd9fc49e/analysis.md)
records positive A answers over **UDP and TCP, 1 ms each**, TTL **5 / 4**,
matching the controlled origin. Both replies join primary egress, node forwarding,
member tunnel ingress and local-stub output. The original capture gate failure
(singular tcpdump statistics) remains failed; a separate same-capture readback
proves one secondary packet / zero drops, and the parser regression is fixed.
No traffic retry was used. Production automatic nameserver distribution remains
outside this TEST ONLY `ConfigureNativeDNS` integration; operator/platform
Private DNS, browser DoH and Unicode UI checks remain outstanding.

**Request 12 / F12-04:** `StartNativeDNSCapture`, opaque resolver query/reply
hashes, wire IDs and the closed DNS-header decoder now observe the previously
missing relay-send → node → member tunnel → local-stub segment. The [exact joins](runs/2026-11-01/followup23/9bb2098e-1c6f-48a8-9fbd-fd99bd9fc49e/dns-delivery-analysis.json)
show the fresh RR reply at primary packet **4**, node **14/15**, member tunnel
**15**, and a reserialized local response **16**. This new query succeeds in
**2 ms**; it does **not** reproduce or close the historical **5,001 ms SERVFAIL**.
The historic responder and cause remain **unattributed / unproven** because those
interfaces were not captured then. Members owns using the export around its
original AT20 query and preceding mutations; a successful relay socket write
is still not proof of client delivery or processing. No AdGuard/cache/transport
cause is inferred, and no query is repeated to replace the failed row.

## members 13 — 2026-11-01 (follow-up 24, D-102)

**Request 7 / F11-01:** [follow-up 24's immutable analysis and owner options](runs/2026-11-01/followup24/analysis.md)
retain each first-selection attempt separately. The IPv6 observation places the
`::1`-source response at member underlay ingress, followed by a header-error
counter increment and no inet prerouting/input traversal. The successful
one-SYN request already has type-4 traffic before that response. This explains
why a dropped handover response can coexist with members 13's success; it does
not turn every first-selection timeout into this one cause. Read the exact
packet/hash, trace and handshake bounds in the analysis. NetBird 0.82.0 and
`node/agent/probe.go` are unchanged. Pin, daemon-setting, IPv4 preference and
node-side proposals need the owner decision recorded there; none is deployed.

**Request 9 / F12-04:** this is **members' original AT20 case**. #367 already
exports `StartNativeDNSCapture`, `Sample`, `Stop`, private streaming export,
`decode_capture.py --stable-addresses --dns-headers`, and reply-digest/wire-ID
joins through the [DNS capture contract](scenarios/harness-exports.md#follow-up-23--routed-pmtu-native-debug-and-dns-delivery-test-only).
There is no missing shared helper identified and no further independent positive
query needed from devstack. Members must instrument its original query and
preceding mutations in `tests/e2e/at/members/native_dns_test.go` and
`dns_attribution_test.go`, after two-resolver configuration and native join.
Start before the original query, retain that query's result once, sample both
sides, stop/export all points even on failure, and retain complete resolver
lifecycle histories/outstanding sets. Keep raw pcaps/debug logs private.

For a devstack delivery diagnosis, hand back the immutable run/capture IDs,
public header files plus capture receipts/statistics, original query wire ID and
result, both resolver query/reply digests, and their completion times. Those are
the concrete inputs devstack needs from members to locate a missing relay-send →
node → member tunnel → local-stub boundary or a native error response. A successful
socket write still proves neither client receipt nor parsing. The historical
5,001 ms SERVFAIL remains unattributed; no new F12-04 query ran in follow-up 24.

**Request 10 / F13-01:** the shared reader now preserves exit-0 parse failures
without retry. [The contract](scenarios/harness-exports.md#follow-up-24--private-parse-evidence-and-ipv6-input-observation-test-only)
defines `parse_failure` classification, digest, byte size, line count and storage/
export status. Exact command bytes **and exact parser error text** are retained
in private mode-0600 files and automatically exported by `DeviceByteCounters`
to `.agent/tmp/counterread/<id>/` before container teardown. Error text stays
private because the parser can embed nft names/values. The original devstack-20
categories, command exit, timing/heartbeat fields and failed read remain intact.
Synthetic empty/truncated/interleaved/semantic failures exercise the real parser
and one exit-0 read; success/failure of private export is also covered.

Members should preserve the returned diagnostic on a non-nil error and retain
those private files on its next original withdrawal run. Both F13-01 rows stay
failed; no replacement sample or weakened predicate is supplied. The old raw
bytes/errors cannot be recovered, and no fresh production parser rejection is
claimed here. There is no evidence-backed parser defect to assign to netlab yet.
F10-02 may be an earlier member of this broad class but stays historically
unattributed. `node/enforce/parse_nft.go` and members' request register are untouched.

**Request 1 / D-137 and request 5 / positive IDNA:** these were already delivered
by **#367**, which merged while members 13 ran its older baseline. Adopt
`PrepareNativeRouter1280` → `Select` / `StartCapture` using the [routed-hop evidence](runs/2026-11-01/followup23/a26d9b0f-bd1b-48df-9546-4105f9b3f328/analysis.md)
(IPv6 PTB/learned 1280; IPv4 DF=0/transit fragmentation). Adopt `ProbeNativeIDNA`
with the [positive UDP/TCP wire evidence](runs/2026-11-01/followup23/9bb2098e-1c6f-48a8-9fbd-fd99bd9fc49e/analysis.md).
The receive-only black-hole rows remain limitations, and production nameserver
provisioning/platform Private DNS/DoH/UI checks remain separate. No new router,
IDNA fixture, TTL or cache work is added here. All follow-up 23 pins remain intact.

## updates 13 — 2026-11-01 (follow-up 25, D-102)

> # Updates requests — follow-up 13, 2026-11-01

**F13-01 / missed first AT23 exit:** the TEST ONLY lab parent now persists the
actual wait status independently of the 1 ms `/proc` observer. In
`tests/e2e/at/updates/stopobserve/main.go`, retain the current
`/var/lib/vpn-node/e2e/agent-process-id` before invoking the original service
stop command. After it returns, read
`/var/lib/vpn-node/e2e/agent-exits/<retained-id>.json`. Match its PID to the
pre-stop PID and use `start_ticks` plus the unique ID to avoid PID reuse. The
public stop waits for the completed receipt. Read this same retained ID after
a subsequent start; the current ID will then refer to a different process.

The [export contract](scenarios/harness-exports.md#follow-up-25--persistent-agent-wait-status-test-only)
defines nullable `exit_code` versus numeric `signal`, raw `wait_status`, and
UTC `wait_observed_at`. This time is the parent's wait completion observation,
not the kernel's exact exit instant. A supervisor SIGKILL attempt additionally
has `<id>.sigkill.json`, with `requested_at` and `kill_accepted`; actual termination
still comes from the wait receipt. Missing evidence is unavailable, never an
inferred successful exit. Repeated stop of an already stopped process preserves
the old receipt and must not be counted as a new exit.

The [live graceful receipt](runs/2026-11-01/followup25/d513a413-4597-423c-9dd3-7c0669f03eca/agent-exit-graceful.json)
and [live SIGKILL receipt](runs/2026-11-01/followup25/d513a413-4597-423c-9dd3-7c0669f03eca/agent-exit-sigkill.json)
were read after the agent was reaped and remained byte-identical after repeated
stop. Unit tests additionally distinguish normal exit 137 from SIGKILL and
retain receipts across restarts. The agent's command, stop signals, five-second
escalation threshold and product shutdown limits are unchanged. Your observer
file remains untouched for lane-owned adoption. Historical run 759d244a's first
exit stays unobserved; this export cannot reconstruct it.

## netlab 23 / members / isolation — 2026-11-01 (follow-up 26, D-102)

> counterread's `duplicate_chain` kind (netlab 23, #377, request 1)

The shared `counterread` helper now matches `*enforce.DuplicateChainError`
through `errors.As` and publishes `parser_error_kind="duplicate_chain"`,
`class="ambiguous_listing"`. Exact and differing duplicate definitions have
separate real-parser tests; syntax and type errors keep their own kinds.
`stage="nft_parse"`, failed read, absent counter observation, private raw/error
retention, SHA/length/line receipts and single-attempt behavior are unchanged.
The parser error string is never published. This classification does not prove
the origin of F13-01 or change netlab's 0/3,300 reproduction result. Existing
members/isolation failed rows remain failed. No parser tolerance or retry was
added.

## updates 13 / isolation 13 — 2026-11-01 (follow-up 26, D-102)

> # Updates requests — follow-up 13, 2026-11-01
>
> ## Follow-up 13 requests — 2026-11-01

The TEST ONLY supervisor now allows the shipped **15-second** agent stop
budget; PID1 forwards container TERM/INT through that same supervised stop
path. Compose and secondary/clone node containers allow **25 seconds**.
Shared namespace-restart, clone-source and retained-fleet stop wrappers also
allow the agent its full budget. The handler keeps the waiter and watchdog
alive through agent completion. It is signal-forwarding behavior, not a shell
scratch-cleanup trap. Automatically running agents resume on container boot;
a deliberate service pause stays paused.

Consumers include updates' `tests/e2e/at/updates/stopobserve/main.go`, isolation's
whole-node and clone lifecycle rows in `tests/e2e/at/isolation/driver.go.txt`,
and its direct-agent sampler in `observer_main.go`. Capture the launch ID,
PID and start ticks before the original stop. After a stopped container, read
`agent-exits/<id>.json` with `docker cp` or its private bind-mounted state;
`docker exec` cannot run there. After a service-only stop, the existing read
sequence still applies. The matching `.sigkill.json` is an attempted escalation;
the actual exit code or signal comes from the parent wait receipt.

The [follow-up 26 results](runs/2026-11-01/followup26/analysis.md) retain ordinary
exit-0 and deliberately SIGSTOP-held/SIGKILL proofs. Container exit 0 records
completion of its shutdown handler, **not** agent exit 0; forced agent signal 9
remains explicit. Later successful measurements do not replace updates' missed
first exit or isolation's historical 10.087/20.153-second exit-137 rows. No
product shutdown deadline, updater assertion or historical row is changed.

## members 14 — 2026-11-01 (follow-up 26, D-102)

> # Follow-up 14 requests — 2026-11-01

**Request 6 / F11-01 and F14-01:** the [bounded campaign](runs/2026-11-01/followup26/analysis.md)
exports central desired/lease/report histories before teardown and distinguishes
admission, traced first selection, and the D-077 re-grant population. The old
`fixture-member-admission` failure reused the original request result; it did
not prove that serving admission had failed before measurement. The new
prerequisite is independent and keeps a non-serving node as its own failed row.

The old node nft observer can itself trigger `ISOLATION_POLICY_INVALID`:
its broad IPv6 UDP `nftrace` rules are unscoped foreign `mangle` statements.
The exact captured listing first rejects a netdev ingress rule; the isolated
inet prerouting rule also fails the unchanged verifier. Normal observation now
traces only the member; the explicit historical
reproduction flag is diagnostic. The retained original IPv6 generation-5,
serving=true five-SYN failures remain distinct from the generation-0 closed-node
samples. Historical central causes cannot be retroactively recovered.

F14-01 uses native capture from before fencing through withdrawal, explicit
re-authorization, serving readiness, route selection and the first re-grant
request. Both peers' status/endpoint/handshake samples and packet hops are kept.
These fresh, reduced populations do not replay all of run 258ba504's preceding
mutations. A successful first selection or re-grant does not close either
historical finding. The original F14-01 report stays `cause_status: unproven`.

Fresh [af718646 re-grant evidence](runs/2026-11-01/followup26/af718646-166a-4a3f-af55-3f7a3ee8e17b/regrant-analysis.md)
does reproduce a five-second deadline failure after generation-7 readiness,
with unchanged counters and serving intact. Native capture shows a loopback
endpoint → IPv6 handover, a loopback-source type-2 response and one member
IPv6 header error, then a later completed handshake. The five SYNs reach the
node only after **5.671 seconds**; late origin TCP packets do not constitute an
HTTP receipt. No failed Apply or baseline follows this re-grant's serving gate.
This localizes the fresh delay to native handshake/data delivery, separately
from the first-selection node-tracer closure. NetBird/native transport and
members owners should assess the transition against 258ba504's full population;
the historical cause and Ted's transport choice remain open.

## isolation 14 / members 15 — 2026-11-01 (follow-up 27, D-102)

> ## Follow-up 14 requests — 2026-11-01
>
> # Follow-up 15 owner assignments — 2026-11-01

**Isolation request 4 / wire-nodes 34 request 1, and members' populated-hold
rows:** the TEST ONLY transport seam is available through `labstack.Stack`.
[Exact arming/release sequence and export fields](scenarios/harness-exports.md#follow-up-27--bounded-populated-generation-hold-test-only).
The matching lease arrives only on `POST /api/v1/agent/lease-renewals`, not in
the desired-state envelope. After the real capacity command, the proxy forwards
desired g+1 and all observations, returning prompt 503s only for successful
renewal responses whose signed generation matches that new desired document.
It never edits, saves for replay, or synthesizes a signed lease or report.
At release it forwards the next genuine central-issued matching response.

Rebuild the ordinary node image; establish the real populated serving lease;
save its report/node/incarnation/lease IDs and generation string; call
`ArmMatchingLeaseHold(ctx, "node", generation, 90*time.Second)` once; perform
the genuine node PATCH with the current ETag and idempotency key. Poll
`MatchingLeaseHold` and `GenerationHoldSnapshot` for accepted applied g+1 /
old lease g and its exact incident. After the required observation window,
call `ReleaseMatchingLeaseHold`, then join the forwarded lease, accepted
matching report and incident ending. `LocalLeaseReadback` supplies the stored
lease digest inside the node. Use a fresh arm UUID for each later incident.
The 90-second example is a TEST ONLY duration; the measured proof uses 45.
The bounded hold does not require stopping the agent or holding desired state.

[Final fresh proof and all attempts](runs/2026-11-01/followup27/hold-fixture.md):
**13 / 0 / 0**, two members, applied 8 / signed lease 7, incident **50.000298 s**.
The node's Step-3 report has `elapsed_ms=9891`, `alarm=true`. Central escalates
once at incident age **9.887144 s**, so **the reported alarm triggered it**;
central's 30-second age fallback did not. A later alarm sample adds no second
audit. The matching-generation report ends the same incident
`generation_matched`, and the actual local lease body equals the forwarded
central response. The two earlier original runner failures stay failed (host
permission, then helper build directory); separate read-only supplements retain
their exact local-byte proof. No product defect was demonstrated.

The final joins are:

- Node `01a197a3-6e35-7ad4-b222-6a74240ce7b5`, incarnation
  `01a197a3-7148-769d-8dc7-65242aac1194`;
  arm `141d6ebb-f2f0-48bf-a540-5ad27e7c2595`.
- Old populated lease `6bcc30f4-b55b-4111-9e62-08c94c0906a9`, generation 7;
  incident `01a197a5-5b5d-76c7-a866-c359da0703a8`.
- Hold reports `01a197a5-5b57-79bb-a1bb-0db65382757c` (Step 1),
  `01a197a5-81f4-7284-8fc3-5e2c8712547f` (Step 3/alarm), and
  `01a197a6-0aab-706a-8a3a-949162e94f16` (Step 10/alarm).
- Single audit `01a197a5-81fd-7151-b4ab-d77111a39d0b`;
  released generation-8 lease `d6075d26-1b48-4bfc-b0fa-4b93f4b9fd2c`;
  matching report `01a197a6-1ea5-78a5-845e-01ac2ab5e520`.

[The immutable joins](runs/2026-11-01/followup27/hold-joins.json) include first /
last seen, end reason, every accepted hold sample, all lease/member windows and
the audit. Join samples by node/incarnation/report and incident, and audit by
its `incident_id`. Recovery is a distinct accepted report, not a fabricated
extra hold sample. An agent-only supervised stop keeps the proxy armed while
local counters reset; restarting the proxy itself releases the test seam.

Isolation and members still own their original unrelated-failure precedence,
active-stop retention/counter reset, matching-recovery acceptance, distinct
later incident, and members' sole-member `members_revoked` rows. The fixture
proof does not pass those combinations. A sole-member row must use its own
one-member populated lease and original real member revocation; one revoked
device or one revoked member among surviving members is insufficient.

## 2026-11-01 — Follow-up 27 response to members 15's serving-node samples (D-102)

Request: measure F15-F11-01's first selection after proven serving and
F15-AT02's 249 missing ingresses / 5.0975 s gap, per IPv4 and IPv6, with the
corrected member-only observer. Keep archived F14-01 `cause_status: unproven`.

[All attempts and attribution limits](runs/2026-11-01/followup27/analysis.md)
and [machine-readable joins](runs/2026-11-01/followup27/campaign.json) are ready.
The predeclared **four fresh stacks per family** each retain first contact,
first selection, and a 150-second stream. IPv4: 4/4 first selections, 29,300
sends, zero missing node ingress; IPv6: 4/4, 29,319, zero missing node ingress.
The largest consecutive ingress gap is 27.761 ms. The 93 IPv4 and 90 IPv6
packets with ingress but no origin receipt remain losses; sender completion
is not AT02 acceptance. No five-SYN failure or five-second ingress gap reproduced.

The IPv6 joins distinguish three events:

- `0876ef42`: eight `::1`-source **type-4 data** messages at both underlay ends,
  with eight member header errors. Valid-source data resumes on the same
  receiver index without an intervening handshake; member→node ingress continues.
- `ae1a2829`: one `::1`-source **type-2 response** and one header error, arriving
  after the concurrent node-originated handshake had already carried the first
  request. The next member initiation is 15.600070 s later. The sustained stream
  starts after that handshake, so the preceding idle interval is not a
  continuous-forwarding proof.
- `1ecee8fe`: another such response after the successful first request, with one
  header error. No further member initiation is captured; the next initiation
  is the node's normal rekey about 120 s later.

Each run's `initiation-origins.json` supplies the direction-qualified origin
join. It supersedes an earlier convenience list that also counted incoming
loopback copies; original packets, timestamps and measured outcomes are intact.
Full route tables, endpoint/status samples, both-end handshake hashes and every
socket-write→ingress interval remain in the pinned archive. Private receipts,
zero capture drops/read errors and member-only tracing were checked. No node
ruleset observer was added.

These fresh rows do not attribute a common cause to historical F15-F11-01,
F15-AT02 or F14-01; all three historical files remain byte-identical. Members
16's original send-time capture and the consumers' full loss attribution remain
theirs. Ted retains the F11-01 repair choice. No transport, NAT, kernel, member
validation, probe behavior or NetBird pin changed. The populated-hold arming
sequence and exact consumer joins are in the preceding Follow-up 27 reply.

## members 16 — 2026-11-03 (follow-up 30, D-102)

> # Follow-up 16 owner assignments — 2026-11-03
>
> **2026-11-03 — devstack, survivor hold observation:** `tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go` releases a consumed arm on `generation_superseded`, as its D-102 contract says. In this run the real sibling-member suspension advances the generation and releases before an accepted post-command survivor hold report. Please provide or specify a bounded TEST ONLY sequence that can retain that eligible survivor through the next generation without editing/replaying signed documents.

`Stack.ArmMatchingLeaseHoldSurvivor(ctx, slot, baseGeneration, survivorMemberID, duration)` is now exported through `labstack.Stack` as an opt-in, TEST ONLY continuation in the D-102 seam ([exact contract and fields](scenarios/harness-exports.md#follow-up-30--bounded-survivor-hold-continuation-in-the-d-102-seam-test-only)). Existing `ArmMatchingLeaseHold` callers keep `survivor_member_id=""` and `max_continued_generations=0`, so the default D-102 behavior and JSON receipts stay byte-identical.

- **Option and bounds:** `ArmMatchingLeaseHoldSurvivor` validates a non-nil `survivorMemberID` UUID and writes `survivor_member_id` plus `max_continued_generations=1` into `/var/lib/vpn-node/e2e/lease-hold/active/arm.json`. Time remains strictly bounded by the original `HoldSeconds` (`1..120s`, anchored once at `hold_started_at` and never reset on continuation) and the 3-minute `ExpiresAt` arm deadline. Generation continuation is bounded to at most **1** advance (`G1 -> G2`) after initial hold target `G1`.
- **What it withholds and releases on:** During `G1`, successful 200 renewal responses for `G1` return prompt HTTP `503` (`LEASE_HOLD_TEST_ONLY`) as before. When the real sibling suspension advances desired state to `G2 > G1`, the proxy forwards desired state `G2` unchanged; on the first 200 renewal response for `G2`, it inspects the signed `members` array (read-only) without editing, replaying, or synthesizing any signed document. If the signed `G2` lease contains **only** `survivor_member_id`, the proxy records `continued_generation=G2`, `continued_at`, and a `continuation_started` event, and withholds only `G2` renewal responses within the original `hold_until` window. It immediately releases and forwards unchanged on:
  - `survivor_mismatch` (if the `G2` lease has 0 members, still contains the suspended sibling, or contains any member other than `survivor_member_id`, so sole-member `members_revoked` closure is never masked);
  - `generation_superseded` (any second advance `G3 > G2`);
  - `explicit_release` (`ReleaseMatchingLeaseHold`), `hold_deadline_elapsed`, `arm_expired_before_desired`, `invalid_lease_envelope`, or `proxy_restart`.
- **What a consumer must record:**
  1. `arm` (`id`, `base_generation`, `hold_seconds`, `survivor_member_id`, `max_continued_generations`, `armed_at`, `expires_at`), `retained_local_lease` (`lease_id`, `body_sha256`), and the two member IDs (`survivor_member_id`, `sibling_member_id`).
  2. Initial node command (`status=202`, `desired_generation`), `incident_id`, `entry_hold_report_id` (Step 1), `alarm_hold_report_id` (Step ≥ 3, `reported_alarm=true`), and `escalation_audit_id` (`node.generation_hold.escalated`, count `1`).
  3. Sibling suspension mutation (`started_at`, `completed_at`, `status=202`) and the accepted `post_command_hold_report_id` received after `completed_at` while `ended_at IS NULL`, at `applied_generation = continued_generation > initial_target_generation`, joined to the retained lease containing the non-suspended survivor and the now-suspended sibling.
  4. Proxy receipt (`continued_generation`, `continued_at`, `continuation_started`, `withheld` on `continued_generation`, `released`, and `matching_lease_forwarded` on `continued_generation`), `recovery_survivor_lease_id` (matching `LocalLeaseReadback` SHA-256 byte-for-byte), and `recovery_serving_report_id` closing the same incident `generation_matched`.
- **Fresh stack proof ([`docs/e2e/runs/2026-11-03/followup30/README.md`](runs/2026-11-03/followup30/README.md), run `8576dfe5-aa76-417c-9c12-f104cef47fdb`, 1 attempt, `10 / 0 / 0`):**
  - Node `01a1a02d-29a3-74ce-b6bf-f63f739267c4`, incarnation `01a1a02d-2d37-786c-a359-090bc44c43a9`, survivor `01a1a02d-7200-7f22-b4ba-9e1c79ce69ee`, sibling `01a1a02d-df51-7e7d-8e43-afd3884056b0`, retained generation-`8` two-member lease `0e45897f-2b45-48dd-9956-4cfe2c987274`, arm `c81a7351-ce93-451b-b988-68aedb4fbb90`.
  - Incident `01a1a02f-3e1d-7521-836c-c994dfc8190a`; entry hold report `01a1a02f-3e18-75c9-a1ca-df9ecde0daab` (Step 1, applied `9` / lease `8`); alarm hold report `01a1a02f-64cf-7760-b675-b7f64dbb77d3` (Step 3, `reported_alarm=true`, `elapsed_ms=9916`); single escalation audit `01a1a02f-64d5-7ee1-93c3-951a99af5e9a`.
  - Real sibling suspension (`operation_id=01a1a02f-6675-713d-98b1-22190987579d`, completed `2026-11-03T06:53:43.415287241Z`) advanced desired state to generation `11`; proxy recorded `continuation_started` on generation `11` at `2026-11-03T06:53:47.941470275Z` and withheld generation-`11` survivor-only lease `53dd9d93-50d3-4d7e-bce7-72a5ba53888e`.
  - Accepted post-command survivor hold report `01a1a02f-785d-79f2-a660-7a99fc16c580` (Step 4, `applied_generation=11`, `lease_generation=8`, `reported_alarm=true`, received `2026-11-03T06:53:48.003245+00:00` while incident remained open).
  - Explicit release at `2026-11-03T06:53:52.925435523Z` forwarded generation-`11` survivor lease `b36b4be7-84c2-4728-8e36-7049d5523c48` (matching local durable lease `bec6929bb25334fc7a4082e2cb6a908718207e3ebb8a85a064fe935067c7c2ee`), and recovery serving report `01a1a02f-8c27-7587-b6fe-96f8b865a6db` closed the incident `generation_matched` at `2026-11-03T06:53:53.070299+00:00`.

## members 17 — 2026-11-03 (follow-up 31, source-binding feasibility and archive navigation)

**Source-address populated hold: not exercised; hook unavailable.** This item
stopped at the requested feasibility condition. No new Stack option, bounds,
state mutation or release behavior is advertised. Existing D-102 callers are
unchanged. There were zero stack attempts or injections, and no accepted-report
or central-classification evidence was generated for this row.

The node computes `bindingOK` in `Decide` from signed binding/lease model
construction. Model errors and zero admitted bindings fail qos too. Live nft
or member-path failures become a retained Apply error, which overrides lease,
qos and source_binding together. During the hold, `leaseOK=false` already fails
both source_binding and qos; the blocked audit promotes TC errors, not an
independent nft source-rule failure. A TC-free namespace mutation therefore
cannot satisfy the requested independent source-binding failure and passing
restoration. See the [complete source trace and rejected mechanisms](runs/2026-11-03/followup31/source-binding.md).

**2026-11-03 hand-off to orchestrator / w2-agent-core:** supply a genuine,
cause-specific source-binding verifier that runs against a defined binding
expectation while forwarding is held, and distinguish its result from shared
Apply/qos and lease-dependent statuses. Specify the checked node state and
restoration contract; route contract or central changes to their owners.
This requires changes outside this lane. No report-field override or synthetic
signed document is a substitute.

Once that exists, the devstack hook must be opt-in and scoped to one node role
(and member/device where applicable), expire at a hard maximum duration, release
on explicit removal and test/handle cleanup, and verify idempotent removal by
state read-back. Consumers must retain commands, exits, before/after state and
UTC bounds for injection and removal, including every failed attempt; join the
node/incarnation, populated lease/member IDs, hold arm, incident, accepted report
and audit IDs; and record central's actual classification. These are future
acceptance requirements, **not an implemented API**. The separate source-address
row remains open for members' next follow-up after the owner prerequisite.

**Archive navigation: repaired.** Added
[`docs/e2e/at/poc/G1-app-checklist.md`](at/poc/G1-app-checklist.md) as a relative
symlink to `../../../poc/G1-app-checklist.md`. It opens the real checklist and
preserves every byte-pinned archive. All **14** archived `OPERATORS.md` files
carrying this link in members commit
`1019e9591c68a3d8f9323dd51259d94d1e5e5bf8` (follow-ups 3–16) resolve to
`docs/poc/G1-app-checklist.md`; the
[per-file verification](runs/2026-11-03/followup31/archive-links.md) records each
source file, source digest and successful target/byte comparison.

## 2026-11-03 — isolation 19 F19-02: bounded node ETag conflict handling

The shared publisher now re-reads the node after an exact HTTP 412
`RESOURCE_VERSION_CONFLICT` on its capacity PATCH or publication PUT. If the
re-read has all requested capacity values, or published intent and the exact
requested group, that command succeeds without another write. Otherwise it
retries with the new ETag. Each command has at most three write attempts and a
ten-second total deadline, including re-reads. This permits two transient races
while keeping persistent contention blocked. Other statuses, other 412 codes,
failed re-reads and transport errors stop immediately. No run is retried and
nothing is added to `expected-blocked.json`.

The contract and `internal/nodes/observations.go` show that accepted reports
advance the node version. Isolation's stored check timestamps include
`2026-11-03T22:10:56.673939768Z`, in the same second as its blocked PATCH record.
A report arriving between the GET and PATCH is a plausible explanation, but
that record lacks the sent ETag, returned current version and individual request
times needed to identify the competing write. This is not proof of host
interference or a product defect. Isolation's later independent PUT returned
202 and remained running; that separate evidence is preserved.

Retries keep the original UUID key and unchanged body. OpenAPI `IdempotencyKey`
and spec §14.5 identify the same logical command by its validated payload;
`If-Match` is not in that hash. The server rolls back both the failed command
and its key reservation on a 412, and stores only durable successful results.
A new key is unnecessary. The [source audit](runs/2026-11-03/followup32/audit.md)
traces this boundary and lists every owned/AT conditional-write site.

Consumer records now contain a `node_write` observation for each command. Its
JSON has ordered `attempts`, method/path, sent `if_match`, UUID key, response
status/error code, decimal-string `current_resource_version` when supplied,
UTC start/finish times, and the nested re-read's method/path/status/error/ETag
and times. `outcome=already_intended` explicitly identifies success without a
second write. Bounds and the terminal outcome/status are included. This
command receipt does not imply applied state: all downstream publication
predicates remain intact, including real operation success and active observed
publication. The other owned node helpers use the same implementation;
unrelated calls keep their existing behavior.

`FVPN_E2E_PUBLISH_STALE_ETAG=1` is an off-by-default TEST ONLY switch: it sends
`"0"` on the first capacity and first publication attempt, provoking real API
412s without modifying product state or observations. Subsequent writes use
the actual re-read ETags. The [follow-up 32 evidence](runs/2026-11-03/followup32/README.md)
retains the synthetic gates and both fresh-stack attempts, including the
ordinary path with the switch unset. AT-owned sites are read-only hand-offs in
the audit's dated Requests to orchestrator; isolation reruns after merge.

Follow-up 32 validation result: the ordinary fresh-stack run
`c52b760c-c289-4220-a722-6c5a059819b9` passed all eight checks, including active
publication and populated serving. It had zero conflicts. The forced run
`e22872b7-1112-4082-8e2b-235275f4a659` failed before enrollment because the
local admission wrapper polluted Docker's JSON stdout with `free`; its original
failure is retained and the wrapper was corrected. Under the explicit no-run-
retry rule it was not repeated. **Real-API 412→successful retry validation
remains not exercised**, although all synthetic conflict cases passed. A later
orchestrator-authorized forced proof is still needed; F19-02 is not fully closed.

2026-11-03 resumed validation: **Resolved by forced invocation
`95fb7eab-6795-48df-b3f2-ed014710d292`**, which passed all eight checks. The
first PATCH and PUT each sent `"0"`, received real HTTP 412
`RESOURCE_VERSION_CONFLICT`, re-read once, and retried once with ETags `"7"`
and `"8"` respectively. Both retries returned 202 with the same key and exact
body digest; each key had one durable accepted operation. Real operation
success, active publication, member admission/counting and populated serving
all passed. Fresh ordinary invocation `3ad796d4-b669-42d2-aadf-46e60146822a`
also passed all eight checks with zero conflicts. First resumed invocation
`af726b13-9f9c-4675-b81f-edd78d3a5c24` remains blocked: publication recovered,
but the owned proof helper incorrectly demanded a version detail when the
API's zero-header rejection supplied empty details. The proof was fixed and
covered by a stub regression under the resume's explicit retry authorization;
no product code or downstream predicate changed. The
[resume records](runs/2026-11-03/followup32/resume.md) retain every invocation,
request/re-read time, fresh ETag, same-key/body receipt, clean-wrapper dry check
and diagnostic. Original records and earlier “not exercised” wording are
preserved as history; the required real-API proof is now exercised. This
zero-header injection does not establish the historical observation race.
Isolation still owns its post-merge acceptance run.

## 2026-11-03 — AT owners: exported node conflict handling (follow-up 33)

`labstack.Stack.WriteNode(ctx, browser, method, writePath, readPath, body,
etag, intended, successStatuses...)` is now available through
`tests/e2e/internal/labstack` with `-tags e2e`. Supply the AT caller's own
`BrowserSession` (cookie client and CSRF), `/api/v1/` paths and the initial
GET's ETag. The caller's client must use `*http.Transport` or nil; a custom
RoundTripper is refused because it could introduce unrecorded retries.

```go
result := stack.WriteNode(ctx, browser, "PATCH", nodePath, nodePath,
    capacityBody, read.Header.Get("ETag"), labstack.NodeFieldsMatch(capacityBody), 202)
// Embed result.Receipt even when result.Accepted(202) is false.
```

Predicates: `NodeFieldsMatch(want)` compares requested fields exactly (including
nested capacity and decimal strings); exclude command-only fields such as
`reason`. `NodePublished(groups...)` requires published intent and the exact
unordered group membership, refusing duplicates. `NodeRevoked` checks revoked
lifecycle. Other lifecycle writes can supply `NodeFieldsMatch` with their
intended lifecycle, or a `func([]byte) bool` decoding the GET body and requiring
all intended values. A nil predicate is refused.

The exported call shares `writeNode`: only exact `412 RESOURCE_VERSION_CONFLICT`
re-reads; already-intended ends writes; otherwise the fresh strong ETag is used
with the same UUID key and frozen JSON body. Three writes and ten seconds cover
all attempts/re-reads, including a final re-read after the third conflict. Other
statuses/codes, transport errors, failed reads and invalid fresh ETags stop.
Private HTTP/1 connections prevent implicit replay; redirects are not followed.
`Accepted(202)` includes already-intended; `Reply` remains the real 412 in that
case. It supplies no new operation. Retain the consumer's real operation lookup
and observed-state checks; command acceptance is not applied-state completion.

Embed `NodeWriteReceipt` (`result.Receipt`): UTC `started_at`/`finished_at`,
`max_attempts`, `limit_seconds`, `status`, `outcome`, ordered `attempts` with
method/path, sent `if_match`, `idempotency_key`, exact request `body_sha256`,
response status/code, optional current-version detail, transport-error flag,
and nested `reread`. Each re-read has its own times, method/path, status/code,
returned ETag and `already_intended`; its sent If-Match is empty and its request
body digest is SHA-256 of empty bytes. Bodies/cookies/CSRF are never in receipts.

The [follow-up 32 audit](runs/2026-11-03/followup32/audit.md) sites are now
adoptable in each owner's follow-up (line numbers are the audit baseline):

| Owner | Sites |
|---|---|
| secops | Capacity PATCH: `counted_capacity_test.go:104`, `followup4_live_test.go:54`, `followup5_live_test.go:56`, `followup7_live_test.go:101`; publication PUT: `inputs_test.go:335`; revoke POST: `live_test.go:671`. |
| enrollment | Capacity/publication closures: `freeze_test.go:68`, `handoff_test.go:427`; revoke helper: `live_test.go:888`. |
| members | Generic node mutations: `identity_nodes_test.go:173`. |
| usage | Generic node mutations: `live_test.go:592`. |
| updates | Retirement DELETE: `installer_test.go:411`; retain retirement read-back. |
| isolation | Existing overlaid driver can keep calling `writeNode`; six sites are assigned to isolation 20. |

No AT file changed. Expected stale-version negative tests remain negative.
Offline cookie/CSRF stubs cover retry, already-intended, three-write exhaustion,
other-code 412 and redirect refusal. Live proof belongs to each consumer's next
authorized assembled run; no stack or Docker command ran in this follow-up.

2026-11-04 adoption on base `33f0ef8e`: `Stack.WriteNode` has merged for members ([#445](https://github.com/TED-H-com/Freedom-VPN/pull/445)), security-ops ([#448](https://github.com/TED-H-com/Freedom-VPN/pull/448)), enrollment ([#450](https://github.com/TED-H-com/Freedom-VPN/pull/450)), updates ([#451](https://github.com/TED-H-com/Freedom-VPN/pull/451)) and usage ([#453](https://github.com/TED-H-com/Freedom-VPN/pull/453)); isolation ([#442](https://github.com/TED-H-com/Freedom-VPN/pull/442)) uses the same shared `writeNode` directly in its overlay. These are merged adoptions, not new live-proof claims.

## 2026-11-03 — members 19: finite-grant early-return recovery (follow-up 33)

`ReplaceFiniteGrant` keeps its signature, successful fields and expired-input
pre-revoke refusal. It now retains every successfully revoked grant's plan
revision and original `[valid_from, valid_until)` before replay verification.
Any later error triggers recovery through the public grants API, using the
original validity rather than extending the fixture. Recovery has a separate
two-minute context so cancellation of the failed transition cannot skip it.

If the replacement committed before its response/replay/projection failed,
read-only lookup by this command's key, administrator and member identifies
only its own still-live grant. Recovery withdraws that replacement through the
public revoke API before restoring the originals. This also handles a different
replacement plan, which would otherwise prevent the original plan's restoration.
Every original grant gets one restoration POST; a failed restoration does not
skip the others. A grant that has expired in the meantime cannot be recreated
with its original end; the failed API result is reported, never extended.

`GrantTransition.Restorations` records the revoked grant ID, original plan and
validity, new grant/operation IDs, HTTP status/code, accepted flag, outcome and
UTC times. `outcome=restored` requires operation success and public member-grant
read-back of the live, unrevoked grant with exactly that plan and validity.
`ReplacementRevocations` retains acceptance and operation receipts for any
withdrawal of the replacement; acceptance alone does not claim worker completion.
The original failure remains the returned error. If recovery fails too, the
returned joined error contains both failures and all available result receipts.

Members' existing secondary restoration is **harmless for same-plan service
eligibility**, but creates another grant/audit and may extend validity to its own
one-hour end. `members.Service.Grant` rejects overlapping grants only on a
different plan revision; `entitlement.Compute` merges overlapping same-plan
spans. Callers are `finiteExpiry` in `finite_grant_test.go` via
`runFiniteExpiry`/`restoreFiniteGrantOnEarlyReturn`, and `reauthorizationLimit`
in `reauthorization_test.go` via `restoreLimitGrantOnEarlyReturn` in
`grant_restore.go`. Their extra recovery can remain as a fallback, especially
when the shared recovery reports failure. Owners may skip a duplicate POST
when all shared restoration receipts are `restored`, while preserving their
own device/fence verification and recovery after later failures. The normal
expiry restoration and successful downgrade's later return to the higher plan
are still needed. No caller was changed.

Offline tests exercise create, replay and projection failures, first-revoke
replay failure, partial revocation, committed replacement recovery, joined
restoration failure with continued attempts, and success without restoration.
Live worker/API proof remains for the consumers' next authorized assembled run.

## 2026-11-03 — members 19: partial native-router rollback (follow-up 33)

`PrepareNativeRouter1280` retains its normal ten steps and `e2e-down` cleanup.
A failed setup step rolls back only completed steps, in reverse order: delete
peer routes, disconnect attached networks, remove the router container and
remove its network. Router-only forwarding sysctls are undone by container
removal. Each inverse gets a detached ten-second bound; failure or cancellation
of one does not stop the remaining inverses.

The error preserves the original setup cause through `errors.Is`/`Unwrap` and
includes all rollback receipts in its text, so members' existing `safeError`
already retains them. `errors.As` to `*labstack.NativeRouterSetupError` exposes
`Step` and `Rollback` (setup-step index, inverse command, outcome, UTC bounds).
The sysctl row records its container removal's outcome. A failed inverse stays
visible for later cleanup; it is never treated as successful removal.

An injected runner tests failure at every one of the ten steps, the exact
reverse commands, continued rollback when every inverse fails, caller
cancellation, and success with no rollback. These are offline proofs only;
no live network or platform acceptance is claimed.

2026-11-03 follow-up 33 validation: all required offline gates passed, including
unchanged node-write tests, new cookie/CSRF export and recovery stubs, the
isolation overlay and all AT packages under `-tags e2e`. Linux arm64 compilation
also passed (no hardware run). See the [gate ledger](runs/2026-11-03/followup33/checks.md).
Live adoption and assembled proof remain with the consumer owners.

## 2026-11-04 — r29 L-2 and wire-nodes 35 (follow-up 34)

One full acceptance invocation on base `33f0ef8e`, measured code `ab0899e6`,
including updater 24 (#446), agent-core 48 (#447) and 49 (#454): run
`af4c130b-5029-401b-bd0f-e8afaad3d37f`, **31 pass / 2 blocked / 0 fail**, make
exit 2. No rerun. The [analysis and all scenario records](runs/2026-11-04/followup34/analysis.md)
retain D-133 overlap and the three setup-sub-make admission omissions.

**r29 L-2:** `lease-expiry` passed. Its admission poll found counters in
243.761994 ms; control stopped, the earlier desired-policy deadline D was
reached, established replies and bytes stayed quiet with active senders, and
control restarted. Its live quarantine entry is removed; `member-ipv6` remains
byte-identical. Before removal, the nightly evaluator lists lease-expiry under
Unblocked. Both [before](runs/2026-11-04/followup34/nightly-gate-before.md) and
[after](runs/2026-11-04/followup34/nightly-gate-after.md) verdicts are **fail**
because update-good is an unexpected blocked row. No workflow was dispatched.
Historical E2E gate tests use a new byte-identical frozen list; their existing
assertions, handover's frozen fixtures and follow-up 29 records are preserved.

**F30-01 remains open for candidate proof.** update-good exited **3 (ExitBusy)**,
with no candidate executable or health challenge; update-bad passed with
**1 / HASH_MISMATCH**. PID 876 stayed throughout the observed update/control
window. There was no candidate boot deferral or clearance to timestamp. A
passing lease row after a skipped candidate update cannot establish the
requested #446 + #454 recovery proof. The observed exit status follows #446's
mapping; the cause of this invocation's contention is not attributed to a merge.

**Wire-nodes 35:** publisherTrace32's comment now describes D-154: TEST ONLY
If-Match `"0"` is parsed and compared, yielding ordinary 412
RESOURCE_VERSION_CONFLICT with details.current_resource_version on these
nonzero-version rows. The predicate and its tests are unchanged, accepting both
shapes because retained forced run `95fb7eab` predates #443 and has empty details.

**Security-ops 15 / AT29:** the repeated guest-clock request is already answered
by the [2026-10-31 response](#security-ops--2026-10-31) and
[operator-run procedure](at29-guest-clock.md). It remains an operator-run row;
this follow-up adds no guest provision, host clock change or simulated pass.

### Requests to orchestrator

2026-11-04 — updater / agent-core: investigate update-good's exit 3 before
candidate start and arrange a separately authorized candidate acceptance proof.
Handover/status: keep F30-01 open until that path is observed; lease-expiry's
quarantine removal alone is insufficient in this run. Preserve the confounded
window and admission omissions in review. No out-of-scope path was edited.

## 2026-11-04 — updater 25 / D-158 and D-160 (follow-up 35)

**F30-01 is closed by updater 24, agent-core 49 and updater 25**, with run
`8a436a87-5efe-4c18-a7ec-72be170cb69b` on code `9fbff94c` and baseline
`5c21e670`: **32 pass / 1 blocked / 0 fail**, acceptance make exit 2, unchanged
nightly evaluator **PASS** (exit 0). No scenario or full run was retried. See
[analysis and all records](runs/2026-11-04/followup35/analysis.md).

Each updater-loop invocation now retains complete output, UTC start/end, PID and
exit status. Both scenario records contain the cumulative archive; the second
retains the first unchanged. The offline two-invocation proof passes.
`update-good` exited **0**, ran the exact candidate and recorded local success;
its complete output is empty and it wrote **no lock-wait record**. `update-bad`
exited **1**, retained the exact `HASH_MISMATCH: artifact` error line and also
wrote no wait record. Neither invocation encountered a failed first acquisition,
so this run supplies no wait interval or holder during a wait. It cannot assign
the changed good-update outcome to updater 25's ten-second budget.

The lock/heartbeat trace identifies candidate PID **181291** booting under
parent updater PID **178784** at **09:52:32.026315972 UTC**, followed by
`recovery_required` while the parent still holds the lock. Parent ownership
ends between samples **09:52:37.440685257–37.654699088**. The agent log bounds
recovery clearance by **09:52:42.112**; its first serving heartbeat is written
**09:52:42.902971300** and observed **09:52:42.906726464**, generation 95.
The analysis explains the runtime-contract interpretation of deferral and
clearance and distinguishes it from an unlogged recovery-child completion time.

`lease-expiry` then passes: countedFlow succeeds after a **1.293275146 s**
readiness poll, control stops, the installed desired-policy deadline is reached,
member accept rules become zero, and active TCP/UDP senders produce no additional
replies or counted bytes in the closed-path checks. Control restarts successfully.
The analysis retains D, all fault/restart timestamps, counters and timing limits.

D-160 records **30 admitted sections**, **zero admission checks inside them**,
and all section start/end times. Host preflight waited **660 s** before launch;
no foreign overlap was observed in the five-second run samples. Follow-up 34's
2029 recorded acceptance-time admission checks were all free: its only recorded
wait began after acceptance ended. The old three setup-sub-make omissions are
not rewritten or excused. Current recursive makes are admitted.

All required final gates pass. Teardown and inventory prove zero owned Docker
containers, images, networks and volumes. The initial record-format hygiene and
EOF-whitespace corrections are retained with their failing and passing receipts.
AT29 guest-clock rollback stays [operator-run](at29-guest-clock.md), and
`member-ipv6` remains the sole unchanged nightly-list entry. No workflow was
triggered, and no other lane's path was changed.

### Requests to orchestrator

2026-11-04 — Carry the above F30-01 closure and run ID into finding tracking and
handover, credited to updater 24, agent-core 49 and updater 25. Preserve the
absence of contended acquisition in this run; do not call it a measured proof
that the new wait budget caused the status change. Keep the existing operator
AT29 and member IPv6 dispositions. No additional outside-path fix is requested.


## 2026-11-04 — members 23 request 1, D-162 and updater 26 guidance (follow-up 36)

**Members 23 request 1 is implemented under provisional D-162:** the shared
prepared-config map now explicitly sets TEST ONLY `observation_seconds: 5`.
Both `agent-run.json` and `agent-enroll.json` carry heartbeat/poll/observation
5/5/5 seconds. Offline tests check the serialized fields, production loading,
the unchanged 30-second default without the override, and the unchanged
installer transformation. Every acceptance predicate, including 15 seconds,
remains unchanged. No AT file or historical evidence was edited.

Run **`c17589fa-7e48-4dee-b574-7e29aca1effa`**, measured code `f6edbda9` on
baseline `52b93845`, observed **32 pass / 1 blocked / 0 fail**, make exit 2.
The unchanged scheduled-job evaluator returned **PASS (exit 0)**. Every status
matches follow-up 35; no scenario or full run was retried. This is the first
run with agent-core 51, updater 26 and D-162 together. See the
[analysis, consumer tables and all records](runs/2026-11-04/followup36/analysis.md).

Before teardown, the owned database returned **123 accepted regular-node
observations** in the acceptance window: 122 successive intervals, minimum
**0.081075 s**, median **5.162159 s**, maximum **10.254596 s**. The final
**622.437456 s** has no new accepted receipt, covering the control-loss and
brief restart/revocation tail; that tail is explicitly separate from intervals
between two rows. The ordinary full acceptance path executed **no** regular-node
15-second observation query; the opt-in consumers were skipped. This does not
re-score or close members' dual-1280 finding. All three exercised node writers
(capacity, publication and revocation) returned 202 on their first attempt,
with **zero 412s**. Owned consumers already have the bounded reread/retry and
retain every attempt; their offline conflict tests passed.

**Updater 26 guidance:** each invocation's complete output and wait record are
retained, with the good invocation unchanged inside the bad-update archive.
`update-good` exited **0** with one wait record: **446 ms**, `acquired=true`,
`budget_ms=10000`. Lock samples identify original agent **PID 894** holding
`update.lock` at **14:45:49.155531064** and **14:45:49.367913641 UTC**, followed
by updater **PID 186378** at **14:45:49.584005669**. Candidate **PID 189738**
starts under its parent's lock at **14:45:54.577655841**, reports deferred
recovery at **14:45:54.789907759**, clears by **14:46:04.707**, and first writes
a serving heartbeat at **14:46:05.436664574**, sampled **14:46:05.451947393**.
`update-bad` exits **1 / HASH_MISMATCH**, with no wait record. No contention
was staged. **Stale-then-live contention is unobserved**: sampled ownership
and the terminal wait summary do not retain individual acquisition answers.
The changed wait observation cannot be assigned to one of the combined changes.

`lease-expiry` passed after candidate service: the counted-flow poll succeeds
in **2.343563389 s**, control stops, the earlier desired-policy deadline D is
reached, member accept rules go **558 → 0**, and active established senders
produce no additional replies or counted bytes in the quiet samples. Control
restarts successfully. The analysis retains exact deadlines, timestamps,
counters and observational limits. AT29 guest-clock rollback remains
[operator-run](at29-guest-clock.md); `member-ipv6` stays listed.

All required offline gates pass. D-160 records 30 admitted sections, 2031 wrapped
commands and zero checks inside sections. No foreign overlap was sampled.
Admission waits totaled **32 minutes**: 21 before the first gate, zero in
acceptance preflight, 10 after acceptance before the database export, and one
before the final documentation check.
Teardown completed and inventory proves zero owned Docker containers, images,
networks and volumes. No workflow was triggered.

### Requests to orchestrator

2026-11-04 — Route the analysis's complete source-audit table to all AT owners.
Isolation's **`tests/e2e/at/isolation/assist.py:22`** remains a one-attempt
GET/If-Match capacity PATCH: adopt the shared bounded retry with an offline
conflict test and retain every attempt, preserving its explicit assisted-run
classification. Normal AT node-write wrappers already retry. Members should
retain the strict bound and assess F22-D146 only in a separately authorized
owner campaign. The analysis gives dated file/line hand-offs for members,
enrollment, security-ops, usage, updates and isolation; no outside-path item
was edited. Preserve the operator AT29 and listed member IPv6 dispositions.


## 2026-11-04 — isolation 26 D-141 clone control-path diagnosis and shared fixture fix (follow-up 37)

Fix round 1 answers isolation 26's request to review inherited clone receiver
wiring in its [Requests to orchestrator](at/isolation/followup26-2026-11-04.md#requests-to-orchestrator).
The [corrected diagnosis and consumer inventory](runs/2026-11-04/followup37/analysis.md)
uses committed baseline `b921c75f` evidence and preserves every historical AT verdict.

The static mismatch is supported: the source's observer configures 9445
(baseline `tests/e2e/at/isolation/observer_main.go:125`), and `candidateConfig`
inherits that runtime URL (baseline `tests/e2e/harness.go:551-566`). The clone
copies config and durable data but not `/usr/local/bin`'s observer/wrapper;
its original image serves 9444 (baseline `tests/e2e/second_exit.go:98`,
`tests/e2e/second_exit.go:121-142`, `tests/e2e/cmd/fvpn-e2e-nodeconfig/proxy.go:140`).
The copied runtime config therefore points to an unserved port under this code
path. No actual clone `connection refused` observation has been identified.

The committed inventory has **30 identity-freeze files**, including three stream
copies, not 30 independent runs. F7 has 4 total / 3 clone-observation rows;
F8–F15 including both F10 attempts have 3 / 2, with maximum endpoints 1,
conflicts 1 and ever-frozen true. Later retained captures have 90 / 90, maximum
endpoints 1, conflicts 0 and ever-frozen false. Every status is
`requires receiver correlation`. Read-error rows carrying `clone_observation`
count as such, not as successful reads. F19's capture is F18's identical blob;
F19's own run stopped before clone creation. Stream, later `-run` and root copies
are not independent experiments. The analysis records each external ID/build join.

`freezeIdentity` writes an auxiliary consumer-latch record rather than scoring
`x.records` (baseline `tests/e2e/at/isolation/driver.go.txt:2673-2678`,
`tests/e2e/at/isolation/driver.go.txt:2700-2703`). Distinct post-port-change stop
reviews are F16, F17 and F18 only; F19–F24 and root reuse F18. Recorded clone
exit code 0 establishes a stop outcome, not control transport. Identified
agent/watchdog log retention reads only the source; clone fingerprints/stages
and stop receipts are separate retained data.

Earlier devstack samples are separate: run `12d10041-9d33-4fbb-a4b0-9d2a21bce785`
in `runs/2026-10-31/followup15/exports/export-clone.json` records endpoints 1,
conflicts 0 and frozen false at `2026-10-31T04:28:50Z`; its separate
`clone-conflict.json` records detail strings `conflict_count:"1"` and
`mapping_frozen:"true"` at `2026-10-31T04:30:11Z`. Enrollment's AT07 sample is
also from `2026-10-31`. Neither is a new live validation of this fix.

The authorized correction keeps the existing etc-config SNAT updates, preserves
the portal enrollment URL in both clone and fresh modes, and changes only
`control_url` in copied release configs to `https://127.0.0.1:9444`. Runtime etc
config also uses that local URL. JSON is reserialized with non-target values
unchanged; identity and durable files remain byte-for-byte unchanged. Copied
release SNAT is an explicit known gap. The correction is committed in `a650a4e3`.
The single D141 test requires `-tags e2e` and passed in this session with compact
and formatted config cases, complete JSON-value comparison, durable-byte checks
and malformed-input refusal. Default no-tag tests do not execute it. No prior
report output is treated as verification.

### Requests to orchestrator

2026-11-04 — After merge, request a separately authorized isolation run to validate
the clone transport and consumer latch. Request clone agent/watchdog retention
around the actual `archiveReceiver` / `agent-stop-sample` stop boundary (baseline
`tests/e2e/at/isolation/driver.go.txt:2341-2357`). No live pass is claimed here.
Carry the copied-release-SNAT gap, AT29 guest-clock rollback as
[operator-run](at29-guest-clock.md), and `member-ipv6` as listed in
`tests/e2e/nightly_expected_blocked.json`.
