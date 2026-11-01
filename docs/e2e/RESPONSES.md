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
