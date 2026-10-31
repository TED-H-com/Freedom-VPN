# Shared harness exports (follow-up 12)

Run `TestE2E_HarnessExports` on a fresh prepared stack. Records go to
`runs/2026-10-30/followup12/<prefix>/`; they do not replace full-suite evidence.
Every record carries the source commit and one run ID. This task uses `e2e`
and then the temporary `e2e-exports` prefix, sequentially on its own base 18000.

The checks require real signed boot/candidate releases and API-created policy,
enrollment, publication and grants. No projector-owned rows are written.

| Check | Required observation |
|---|---|
| update-good | Shared probe setup and its non-secret notes; real updater issuance with `configured_probe_issued` true; timed exception elements, authenticated origin replies, nonzero probe counters, unchanged member counters, elements removed |
| update-bad | Real tampered candidate rejected before health; no challenge requirement for this earlier failure |
| export-native-streams | Two independent native joins; bounded final per-stream inventories, mode 0600/root, one recognized device-flow key line, successful deletion; foreign container refused |
| export-enrollment | Two harness keys race one code; one success/one rejection; successful response deliberately discarded; byte-identical replay completes one-time mTLS bootstrap and native join; complete correlated peer readback count one |
| export-clone | Consistent stopped-source snapshot, clone created before boot, original resumed; all three public identity hashes equal; both real entrypoints run |
| clone-conflict | Admin `identity_conflict` is `{mapping_frozen: true, conflict_count ≥ 1}` after the clone presents from the second IPv4 /24; exactly one `node.identity.conflict` audit; provisioning `failed` and member forwarding off; revoke plus entrypoint re-enroll of the same node, with the clone stopped, yields `{mapping_frozen: false, conflict_count: 0, last_conflict_at: null}` once the new incarnation is observed |

The clone fixture is a fault in disposable storage, not a new authorized exit.
A successful clone setup does not prove AT07 safety. Complete incarnation
history still needs the ENR-03 admin read surface. The enrollment protocol
fixture uses a waiting node slot and does not fabricate agent SQLite state or
claim node readiness; ordinary serving nodes continue to use `EnrollAt`.
`AttachClonePresentationNetwork` creates that second /24 and can pin the clone
netns. `StartClonedIdentityFixture` calls it. Records for this follow-up are
under `followup14/exports/`. `clone-conflict` reads the admin node and audit
list. It does not insert conflict rows. A container restart that deletes the
local agent database is the entrypoint enroll path for the replacement
incarnation, not ENR-06.

## Member-ingress injection (follow-up 13)

`Stack.InjectMemberIngress` and `Stack.ObserveMemberIngress` are the devstack
side of `InjectMemberIngress` in package `tests/netlab`. The acceptance
scenario calls `ObserveMemberIngress` after a real device withdrawal. Members
(AT02) and isolation (AT13) call the same exports; they do not exec
`fvpn-netlab` themselves.

| Export | What the caller gets |
|---|---|
| `InjectMemberIngress(ctx, "fvmb0", source, dest, 9)` | `MemberIngressResult`. Empty mechanism, so type 65534 selects `tun-mirred`. `deltas["ctunku"].packets` is 1 for a source that is not admitted. `deltas["u_<32 hex>"]` is the admitted device's upload counter. |
| `ObserveMemberIngress(ctx, source, dest)` | `MemberIngressProof`: that result, plus `fvup0` and origin-namespace tcpdump text for UDP port 9. |

The hook copies a linux/amd64 `fvpn-netlab` into the node container, reads
`nft -j list table inet fvpn`, and removes `fv-inj0` before it returns. It
does not restore a revoked peer, group, or binding. The node container is
already privileged, so this hook adds no capability.

## Follow-up 15

`EstablishedTransports`, `MemberTransportStatus`, `DeviceCounters`, `RelayPath`,
`DNSPathObservation`, `UsageSnapshot`, `InstallerOutcome` and `InstallerHashes`
are exported through `labstack`. See [RUN.md](../RUN.md) for invocations,
cleanup and limitations. The exports suite includes established transports,
forced relay, exit DNS, usage route hold/release, real usage backlog/restart,
and the dedicated systemd installer wrappers. Its scenario outcomes retain
actual product failures; wrapper presence does not imply an AT pass.

## Follow-up 20 — D-087, F10-02 and F10-03 (TEST ONLY)

Prepare a **dedicated** runtime with `FVPN_E2E_GRANT_SEAM=1` on `e2e-up`.
This builds `tests/e2e/cmd/fvpn-e2e-control`, composes the ordinary control
Domains, and replaces only the nodes-domain constructor with
`httpwire.WithTestOnlyGrantEligibilityReaderFactory(grants.ReaderFactory)`.
`grants` is one `nodestest.NewMemberGrantEligibility()` instance. Production
control remains the default binary; the switch is not a product route or
configuration. Its adapter listens on control-container loopback only.

The `labstack.Stack` methods are:

```go
flip, err := stack.SetMemberGrantsReady(ctx, false)
events, err := stack.MemberGrantEvidence(ctx)
invariants, err := stack.MemberGrantInvariants(ctx, nodeID)
```

The flag applies to future renewals for every node in that dedicated runtime.
It does not revoke an existing lease. Configure the supported TEST ONLY fleet
offline window **before enrollment** (this campaign uses 60 seconds), then
leave it unchanged. Wait for initial grant/FUP projection to settle and retain
a passed baseline, working meter, real tunnel probe and counted native flow.
Our runner requires 65 seconds of identical snapshots before arming false;
it neither stops the projector nor manufactures a generation.

Capture sequence for isolation 10:

1. Retain `MemberGrantInvariants`: desired envelope/ETag, generation, bindings,
   ready/passed state, publication, freeze flag, entitlement and device outputs.
2. Arm false and await a natural renewal. Retain the actual signed response.
   An older populated lease can remain active until that renewal.
3. Require a real HTTP-accepted ready/passed observation with forwarding false,
   a lease ID joined to the empty response, exactly `awaiting_member_grants`,
   `AWAITING_MEMBER_GRANTS` on lease/qos/source_binding, and exactly one error,
   with that code and `stage=lease`. `lease_expired` or retained safety failure
   does not qualify.
4. Only after the receipt, arm true. Await a distinct accepted serving report
   joined to a new populated signed lease, then independently count native
   forwarding. Compare every invariant against the pre-flip snapshot.

`MemberGrantEvidence` retains request start / handler completion, HTTP status,
Retry-After, desired ETag, parsed bodies and original JSON wire strings with
SHA-256 digests. Completion is server-side; consumption is established by the
subsequent accepted report naming the lease. No transport hold, mapping freeze,
unpublish, re-signed response or synthetic observation is used. This supplies
the measured central prerequisite; isolation owns assembled AT10 qualification
and its historical durations. See [follow-up 20 evidence](../runs/2026-10-31/followup20/README.md).

`DeviceByteCounters` now returns `read_diagnostic` even on failure: nft command
start/completion, stage, category (`none`, `nft_read`, `parse`, `timeout`,
`permission`, `transient_io`, or a container/helper read failure), exit code,
and heartbeat values read immediately before/after the command. The successful
read includes `kernel_generation`. Before enrollment, call
`EnableEnforcementTiming(ctx)` to enable the product's existing detached
TEST ONLY observer. `EnforcementTiming(ctx)` exports all original operations
and renewals. Each counter sample retains its boot ID, decimal-string
CLOCK_MONOTONIC read start/end and the latest completed enforcement records
before/after the nft read. Join these to the full trace by boot/run/sequence;
reject zero clocks, sequence gaps and missing records. This is a command bracket,
not a fabricated exact kernel commit time. Timing capture writes after the
operation; no callback is added inside the enforcement guard. A missing device counter is a successful
read with `Present=false`; it is never silently substituted for an error.
`EstablishedStatus` independently retains its read category and timestamps.
Existing AT callers that serialize these shared structs retain both diagnoses;
they need not relax `read_ok` or retry a failed sample. Heartbeat times correlate
enforcement state; they are **not** exact nft commit timestamps.

`ProbeNativeDNS` adds `dns_trace`: native query bounds, matching responder,
resolver receipts, authority exchanges, answer TTL remaining, and a separately
sampled authority SOA serial/time. Both native relay endpoints and an uncached
relay immediately before the owned authority capture metadata for a closed
synthetic case list. Full query names, answer addresses, payloads and credentials
are not recorded. AdGuard query logging/statistics remain disabled. No cache is
flushed, resolver chosen by the harness, or query assertion relaxed.
A unique metadata match identifies the responding relay. Multiple matching
responses remain explicitly ambiguous. An observed upstream exchange proves a
fetch; no exchange supports only a labeled cache-hit inference, subject to
the whole-sequence outstanding-request rules below. No upstream
receipt can mean native-stub caching or an unobserved path; it is not labeled
an AdGuard cache hit. The authority SOA is sampled after the native response,
not retroactively attached to a positive cached answer that contains no SOA.

All regular release bundles now place `fvpn-agent`, `fvpn-updater` and
`fvpn-watchdog` directly at the release root. `InstallerLayout` additionally
includes the installer and its original systemd units. Bootstrap continues to
use `/usr/local/bin`; switched releases use
`/opt/vpn-node/releases/<release_id>/fvpn-agent`. The supervisor verifies that
exact executable and retains idempotent process ownership. Shared updater and
recovery configs use `/var/lib/vpn-node/state/meter.checkpoint`, created by the
real agent's D-125 barrier. The regular update scenario leaves the agent alive
to answer that barrier; the updater owns subsequent stop/start. The fixture
no longer creates the unused legacy four-field checkpoint.

## Follow-up 21 — installer input and DNS lifetime correlation (TEST ONLY)

The shared installer spec explicitly sets `allow_loopback=true` only for its
existing authenticated `https://127.0.0.1:9444` proxy. `installArgs` passes
`--allow-loopback` only when that boolean and that exact origin agree. An unset
boolean or another origin adds no flag. Production installer defaults and all
installer-generated reporting fields are untouched. Consume `RerunInstaller`
normally; no `ProbeInstallerExit` reporting workaround is required.

`dns_trace.authority_history` is cumulative for the closed fixture case / RR
type across the whole cache-change sequence, including requests that started
before the current query. Each relay appends `started` before dispatch and
`completed` after response/error, joined by random `request_id`; `stream_id`
and decimal-string `sequence` detect discontinuities. Later snapshots join late
completions to the same original start. `authority_events` now includes **all**
completed matching requests in that stream, not only starts inside this query.
Do not discard earlier events when comparing snapshots.

The three `authority_outstanding_at_*` arrays identify requests outstanding at
query start, query completion, and trace read. `trace_read_at` bounds each file
snapshot separately; the snapshots are not simultaneous. A request may still
complete after the final snapshot. `trace_complete=true` means the inspected
lifecycle stream is readable and contiguous, **not** that all requests have
finished or no older request can update the cache. Missing/partial records,
orphan completions, duplicate completions or unreadable stopped relays
make attribution incomplete. `trace_streams` retains every process stream ID
for each role, including deliberate restarts during fallback setup. Continuity
is checked from sequence one within each stream; a new stream is never
silently merged with an old one. Completeness does not assert no restart or
cache clear: compare stream IDs and policy state across the query sequence. The old completion-only `cce1a688` records remain
unchanged; their `trace_complete` cannot retroactively establish start history.

Adopt [BACKEND §2.1.5](../../dns/BACKEND.md#215-f10-03--positive-cache-and-ttl-measured-2026-10-31):

- An authority request proves a fetch, not fallback or which response supplied
  the current answer. `fetch_authority_request_observed` replaces the older
  overly strong miss label. Source role must match the responding resolver.
- A uniquely different responder establishes different independent caches.
  Call it fallback only with primary failure/timeout and secondary receipts.
- Same responder, old class, decreasing TTL and no new or overlapping fetch
  supports `inferred_hit_no_authority_exchange`; this is no private cache flag.
- An earlier fetch still outstanding during the response yields
  `unknown_earlier_authority_request_inflight`. A completion between queries is
  retained in the next history even if the final query starts no fetch.
  F9-INFLIGHT shows why changed class inside retained TTL cannot be assigned to
  fallback or declared an unexplained cache defect without that full history.
- Missing history/responder, cache-key or policy stability leaves cause
  ambiguous. Policy-generated negative answers can remain `unknown`; separately
  sampled authority SOA never becomes a positive cached A record's serial.

The fixture deliberately retains **4096 bytes (4 KiB)**, its historical override,
for comparability; it is not the pin's 4194304-byte (4 MiB) default. No silent
alignment was made and no eviction cause is inferred. The focused runner retains
both resolvers' live `GET /control/dns_info` before/after: size, TTL overrides,
optimistic caching, upstream mode, timeout, fallback count, blocking mode/TTL.
The pin's empty API upstream mode means load balancing. The fixture relays
also retain their existing 2-second exchange deadline; the backend's 10-second
upstream timeout is a separate limit. These traces do not reproduce the
backend's deliberately delayed 3.5-second experiment or observe its private
cache insertion. A successful send is relay completion, not an AdGuard cache
write receipt. No TTL override, cache clear, timeout change or assertion
relaxation masks the product's F9-INFLIGHT limitation.

TTL is a maximum lifetime, not a minimum retention guarantee. The historical
F10-03 and the later passing follow-up 20 run retain their verdicts; no product
TTL fix is claimed. The [members fixture analysis](followup21-members.md)
separately identifies F11-01's unobserved transport segment and F11-03's original
guarded installation failures without changing either acceptance row.

## Follow-up 22 — selected native path capture (TEST ONLY)

Members 12 can use the following through `labstack.Stack` on the stack returned
by `MemberTransportStack(memberContainer, selectedNodeSlot)`. The selected node
and member must already have joined. Start before the **first** route selection;
no positive traffic prerequisite is inserted by this export.

```go
capture, err := stack.StartNativeCapture(ctx)
// Check err. All three tcpdump processes are ready before this returns.
err = capture.Sample(ctx, "before-selection")
// Perform the existing route selection, unchanged; sample during its polling.
err = capture.Sample(ctx, "after-selection")
// Perform the original bounded transfer; retain counters and origin receipt.
err = capture.Sample(ctx, "after-transfer")
evidence, err := capture.Stop(cleanupContext, privateDiskDirectory)
```

`Sample` and `Stop` are serial methods on one handle; the original selection or
transfer may run concurrently with sampling. Always stop, including on failure,
with a noncancelled bounded cleanup context. Use a unique ignored disk directory
under `.agent/tmp`. `StartNativeCapture` builds/copies a small read-only observer
before starting capture; this overhead is observable and can affect first-use
timing. It neither selects a route nor reconnects, waits for serving, or repairs
transport. Existing `ApplyNativeInputs` still lowers **only member** eth0/eth1.

Each namespace capture is `any` with SLL2 interface index/direction: member
underlay and `fvnb0`; selected node underlay, `fvnb0`, `fvmb0` and `fvup0`; origin.
The closed filter is ICMP/ICMPv6, UDP (including fragments) and TCP 443/9446.
Raw full-snaplen pcaps are private (mode 0600 export); 32 MiB capture buffers
reduce observer drops. Retained capture counts, received-by-filter counts,
kernel drops, readiness/stop bounds and complete byte/SHA receipts are explicit.
Unknown statistics are `-1`/`statistics_ok=false`, never zero. Any kernel drop,
header/file truncation or decoded-count mismatch invalidates loss attribution.
Raw receipts identify ignored files; they are not public retained-tree pins.

`Samples[].points` includes independently bracketed command reads at member,
node and origin: link names/indexes/MTUs; origin, overlay and selected endpoint
route-get results; connection type; selected local/remote ICE endpoint family,
port and address token. WireGuard public allowed-IP/endpoint views are matched
by the selected peer overlay address; peer keys are used only in memory. No
private-key dump is read. `wireguard_endpoint` and `wireguard_route` distinguish
an actual outer peer endpoint from a local userspace proxy. Missing peers and
failed commands remain visible and cannot establish transport readiness.

All clock pairs retain boot ID, UTC wall sample, and decimal CLOCK_MONOTONIC
bounds around that sample. Every command and capture lifecycle has its own
bounds. Samples are **not simultaneous**, nor exact kernel route-commit times.
Pcap timestamps remain kernel wall timestamps, not hardware or native monotonic
packet timestamps. Only derive monotonic packet intervals against the retained
clock pairs when the same boot and measured wall/monotonic offset are stable;
state that interpolation assumption and its measured error. Packet hashes join
outer datagrams across endpoints; a time-adjacent encrypted datagram alone does
not prove which inner packet it carries.

Decode with `tests/e2e/decode_capture.py --stable-addresses --out <archive>` and
the three pcaps. It emits headers, opaque full-IP-packet hashes and `addr-…`
tokens matching snapshots, never payload/MAC/native keys. Preserve original
private receipts and publicly pin only immutable run/archive files (D-132).

`ConstrainSelectedNativePeer1280(ctx)` is a **separate control**, returning a
cleanup function. It requires the connected selected sender's P2P IPv6 endpoint
to be this fixture's member `::41`, then adds one new `/128` route with MTU 1280
on the route-read-back underlay interface. It refuses an existing route rather
than replacing it. Cleanup deletes precisely that route. Sample before/after
and prove the actual route metric; no global interface/tunnel MTU or MSS changes.
This represents prior knowledge of a 1280 path. It does not emulate a router,
create PTB or prove reactive ICMP PMTU discovery. Keep the original asymmetric
receive-link failure and the explicit-path result as separate evidence.
