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
underlay and `fvnb0`; selected node underlay, its native `fvmb0` tunnel and `fvup0`; origin.
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
port and address token. `relay_endpoint`/`relay_route` resolve a selected relay
address only inside the controlled fixture; an absent ICE endpoint is not proof
of missing relay transport. The native `uses_kernel_interface` flag and link
kind are read back. WireGuard public allowed-IP/endpoint views are matched
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
For first-selection forensics, optional `--wireguard-headers` adds only message
type, public sender/receiver indexes, decimal counter, length, loopback address
classification and an opaque message hash. It checks fixed WireGuard layouts
and reserved bytes on UDP port 51820; it exports no ephemeral key, MAC or
ciphertext. Classification does not authenticate a handshake or prove kernel
acceptance. The message hash can join a handshake through a local proxy even
when its outer IP header changes. Retain the decoder revision for post-run
derivations. Start/stop are bracketed separately per role; a sender tail after
the receiver stopped cannot establish loss. `analyze_native_capture.py` joins
headers with the retained readbacks and derives the stated empirical packet
clock bounds. Run it before pinning a new immutable evidence subtree.

`ConstrainSelectedNativePeer1280(ctx)` is a **separate control**, returning a
cleanup function. It requires the connected selected sender's P2P IPv6 endpoint
to be this fixture's member `::41`, then adds one new `/128` route with MTU 1280
on the route-read-back underlay interface. It refuses an existing route rather
than replacing it. Cleanup deletes precisely that route. Sample before/after
and prove the actual route metric; no global interface/tunnel MTU or MSS changes.
This represents prior knowledge of a 1280 path. It does not emulate a router,
create PTB or prove reactive ICMP PMTU discovery. Keep the original asymmetric
receive-link failure and the explicit-path result as separate evidence.

## Follow-up 23 — routed PMTU, native debug and DNS delivery (TEST ONLY)

D-137 is provisional. §8.3 / AT09's 1280 scope is a sender-learnable path;
historical receive-only black holes stay failed. No global MTU or MSS clamp.

```go
hop, err := stack.PrepareNativeRouter1280(ctx)
// Canonical member + primary node after PrepareNativeUnderlay; check err.
err = hop.Select(ctx, "ipv6") // or ipv4; does not reconnect or send traffic
capture, err := hop.StartCapture(ctx)
err = capture.Sample(ctx, "before-reconnect")
// Run the existing native down/up, selected-family readback and bounded bulk.
err = capture.Sample(ctx, "before-bulk")
// Preserve 64,064 bytes / original 12-second deadline and independent counters.
err = capture.Sample(ctx, "after-bulk")
evidence, err := capture.Stop(cleanupContext, privateDiskDirectory)
```

The router is an owned `node-router` container on the existing 1500-byte node
underlays. Its new bridge/member link has MTU 1280 at both ends; each legal port
slot reserves a distinct documentation-range /28 and IPv6 subnet. Only exact
fixture peer routes are added; no configured route MTU. IPv4/IPv6 forwarding is
enabled inside that container; its fresh namespace has no ICMP-blocking filter.
`Select` restores old member links to 1500, then applies 180-second family/UDP
filters that force native candidates through the new link. The member-facing
interface must be the canonical new eth2; the API refuses a noncanonical
member/node pair before mutation. One hop per fresh stack. Always run e2e-down
on exit, including partial setup errors; it removes routes with the namespaces
and all owned router objects. Do not reuse the fixture as a production router.

Capture includes a fourth `router` point. All samples/stop completeness rules
from follow-up 22 apply. Decode and run `analyze_router_capture.py <new-run>`
before pinning. Read sender `peer_route` before/after and correlate PTB at router
and sender. IPv4 DF=false plus matching router fragments is transit fragmentation,
not learned PMTU. The [native run](../runs/2026-11-01/followup23/a26d9b0f-bd1b-48df-9546-4105f9b3f328/analysis.md)
records both families. `ApplyNativeInputs` and `ConstrainSelectedNativePeer1280`
retain their original semantics; the routed rows are separate controls.

The pinned daemon supports `netbird debug log level debug` without reconnect.
Enable node debug **after enrollment starts its daemon**, and member debug
before native join. Export `netbird-service.log` privately. Run
`decode_native_debug.py --record <record.json> --out <new-archive.json> <logs>`
to retain only allowlisted timestamp/source/event/family/address-token fields.
It exports no arbitrary log text, peer keys, DNS names or session identifiers.
Status and all-interface captures remain independent evidence. Debug logging
and capture setup add observable timing overhead; they do not prove reproducibility
without instrumentation or a private kernel acceptance reason.

```go
capture, err := stack.StartNativeDNSCapture(ctx)
// Check readiness; run the original query once, retaining its result.
result, err := stack.ProbeNativeDNS(ctx, "", "rr.example.invalid", 1, false, expected)
evidence, err := capture.Stop(cleanupContext, privateDiskDirectory)
```

This adds primary/secondary relay namespace captures to member/node/origin.
It requires `ConfigureNativeDNS`'s two resolvers and joined peers. The packet
filter now also includes TCP/53. `--dns-headers` decodes only the closed synthetic
case list: wire ID, QR/RCODE, counts, RR type, length and opaque message digest.
It exports no qname or answer RDATA; arbitrary names are omitted. TCP DNS is
decoded only when one complete length-prefixed message is in a captured segment;
no TCP stream reassembly or missing-message inference is implied.

Resolver lifecycle events now carry `query_sha256`, `reply_sha256`, `wire_id`
and `reply_bytes`; the probe carries `dns_wire_id`. Exact hashes join successful
relay writes to packets without relying on adjacent timestamps. A successful
relay `completed` event means **socket write completed**, not native receipt,
client parsing or cache insertion. `resolver_observation_boundary` states this
explicitly. Missing matches and ambiguous responders remain unattributed.
Older receipt schemas and F12-04's original failure are unchanged.

`ProbeNativeIDNA(ctx, tcp)` converts the closed U-label `bücher.example.invalid`
with `idna.Lookup`, checks `xn--bcher-kva.example.invalid`, then uses the native
stub for A over UDP or TCP. The controlled authority has a positive A record
at the controlled origin; case mapping is `idna_positive`. This tests wire
A-label behavior and fixture conversion, not a browser/platform's Unicode UI.
It changes no block response, cache TTL, cache size or nameserver distribution.

F12-03 requires no new zone/cache fixture. Existing `authority_history`, all
three `authority_outstanding_at_*` arrays, per-stream IDs/read times and the
response's `authority_ttl` remain the quiet-window evidence. A retained SOA TTL
of 2 followed 1.229 seconds later is inside whole-second expiry uncertainty.
Members/dns-publish own a quiet setup and a window with adequate remaining TTL
and measurement margin; this lane exports no cache-clear or TTL extension.

## Follow-up 24 — private parse evidence and IPv6 input observation (TEST ONLY)

`DeviceByteCounters` retains the existing stage/category/exit contract. An exit-0
`nft_parse` / `parse` sample adds `read_diagnostic.parse_failure`:

- `id`: a unique sample directory, independent of device identity.
- `class`: `empty`, `truncated`, `interleaved`, or `unexpected_format`.
- `parser_error_kind`: a content-free error category.
- `sha256`, decimal-string `bytes`, and `lines`: exact combined command bytes.
  Lines count LF terminators plus a nonempty unterminated last line; empty is zero.
- `private_retained`, `retention_error`, `private_exported`, `export_error`:
  storage/export success is separate from the failed read.

The helper writes exact bytes to `output.raw` and exact parser error text to
`parser-error.txt`, mode 0600, in a mode-0700 UUID directory under the owned
container's `/var/lib/vpn-node/e2e/counterread`. The stack method automatically
exports both to `.agent/tmp/counterread/<id>/` before returning the original
read error. It verifies the output's digest/size; export has a separate bounded
five-second cleanup context and never repeats `nft`. Consumers must preserve
the returned diagnostic even when the Go error is non-nil. Direct helper users
must export both files before teardown. Keep these files private: parser errors
can include nft object names or values, so exact error text is **also private**.
Public evidence includes only the safe diagnostic, never raw nft data or the
error's arbitrary text. Preserve failed rows; missing evidence is not a zero.

Classification is structural, not causal: whitespace-only is empty; parser EOF
is truncated; explicit warning/error lines mixed with an otherwise complete
JSON document are interleaved; other syntax/type/semantic rejections are
unexpected format. Unrecognized contamination remains unexpected format.
`CombinedOutput` is unchanged, so the reader does not claim which descriptor
produced interleaved bytes. No parser retry, normalization or enforcement change.
F10-02 and F13-01's historical samples cannot be retroactively classified.

The opt-in `TestE2E_Followup24Native` uses a fresh prepared stack and one bounded
first-selection request. It enables native debug without reconnect, starts the
existing all-interface member/node/origin captures, and installs temporary
`fvtrace_<id>` tables in **owned containers only**. Netdev ingress, inet
prerouting, input and output count loopback-source/destination IPv6 UDP and mark
IPv6 UDP for nftrace. All rules continue; no accept/drop override, NAT, endpoint
change or health override is installed. The tables and monitors are removed
before teardown; raw trace (which can include packet bytes), logs and pcaps stay
in `.agent/tmp/followup24/<run-id>/`. `decode_kernel_trace.py` never copies rule
expressions or payload fields. Unknown lines fail derived analysis.

`FVPN_E2E_KERNEL_OBSERVE=1` adds namespace `/proc/net/snmp6`, absolute `nstat -asz`
and IPv6 UDP conntrack readbacks to capture samples. Only numeric counters and
loopback tuple address tokens/ports are exported. WireGuard endpoints and latest
handshakes match the selected peer by allowed IP; no `wg ... dump`, key, secret
or unrelated peer row is exported. Each command retains its clock bounds and
read status. An unavailable counter/conntrack read is not evidence of zero.
A two-second passive tail catches handover after an already completed request;
it does not retry or alter that request's result. Optional
`FVPN_E2E_FOLLOWUP24_FAMILY=ipv6` labels a separate family-constrained diagnostic
using `ApplyNativeInputs` before enrollment. It is not an original dual-path run
or a production preference change.

Run `analyze_kernel_capture.py --run <new-run> --private <private-run>` once,
before pinning. It verifies exact raw receipts, complete captures, monitor stop
and table removal. nft trace IDs join hook traversal, not pcap ciphertext; packet
shape/count and bounded namespace counter changes are corroboration. Source-code
interpretation remains separate from an observed kernel drop tracepoint.

## Follow-up 25 — persistent agent wait status (TEST ONLY)

The lab service manager still owns start, TERM, its 25 × 0.2-second wait,
SIGKILL escalation and restart. A small parent (`fvpn-e2e-nodeconfig agent-wait`)
now starts the identical agent command and calls Wait. `agent.pid` is still the
agent PID. Read `/var/lib/vpn-node/e2e/agent-process-id` **before** stopping or
killing it. After `agent-service.sh stop` returns, read
`/var/lib/vpn-node/e2e/agent-exits/<id>.json`. The public stop waits for that
receipt as well as process disappearance. Repeated stops preserve it; restarts
create a new ID and never overwrite earlier receipts. Keep the ID, not a PID
alone, across a restart.

The mode-0600, atomically published JSON contains `id`, `pid`, `/proc` identity
`start_ticks`, `started_at` (before launch), `wait_observed_at` (UTC RFC3339Nano,
when the parent returned from Wait), raw `wait_status`, nullable `exit_code` and
nullable numeric `signal`, and `core_dump`. Exit zero has `exit_code:0` and
`signal:null`; SIGKILL has `exit_code:null` and `signal:9`. Exit 137 is a distinct
normal exit. The observation timestamp is an upper observation bound on exit,
not an exact kernel event timestamp. No log parsing or zombie polling is needed.

If the supervisor reaches its existing SIGKILL branch it also writes
`agent-exits/<id>.sigkill.json`: PID, signal 9, `requested_at`, and
`kill_accepted`. This records a signal attempt, independently of the child's
actual wait result. An externally injected SIGKILL has no supervisor escalation
receipt. A missing wait receipt is unavailable evidence, never inferred exit 0.
Killing the whole container or the waiter can prevent publication; this export
covers an agent stop while its lab supervisor remains alive. It adds no product
API or production supervisor behavior.


## Follow-up 25 — native diagnostic arms (TEST ONLY)

`TestE2E_Followup25Native` accepts explicit `FVPN_E2E_FOLLOWUP25_ARM=kernel` or
`userspace` and `FVPN_E2E_FOLLOWUP25_FAMILY=ipv4` or `ipv6`, on a fresh reset
stack. The userspace arm sets a one-use node marker before enrollment; only the
lab daemon command receives `NB_WG_KERNEL_DISABLED=true`. The marker is consumed
and cleaned up. This never changes the default fixture or NetBird Dockerfile
pin, and is not a production preference or architecture recommendation.

The selected-peer observer retains IPv4 `/proc/net/snmp` counters alongside
SNMP6 and nstat. `native_latest_handshake_at`, when present, is the native
daemon's separately reported handshake timestamp. It never fills the independent
`latest_handshake_unix_seconds` or `wireguard_endpoint` after a failed `wg` read.
Unknown or zero daemon timestamps remain absent. Public keys remain in memory
only for the selected-peer join. Per-run command success, clock bounds and
missing samples remain visible. See the [complete bounded campaign](../runs/2026-11-01/followup25/analysis.md)
for every attempt, partial observation gap, original failure and scope limit.

## Follow-up 26 — serving prerequisites, central history and stop budgets (TEST ONLY)

`TestE2E_Followup26Native` is a fresh kernel-only successor. Set
`FVPN_E2E_FOLLOWUP26=1` and an explicit `FVPN_E2E_FOLLOWUP26_FAMILY=ipv4|ipv6`.
It keeps every original request and exports `central-record.json` before
teardown: all immutable desired generation/signing times, issued lease and
member windows, and every accepted report's generation, forwarding, block
reasons, checks and errors (code/stage/time only). Free-form messages, signed
envelopes and keys are excluded. Later snapshots use one repeatable-read,
read-only transaction. The detached producer stream has a private byte/hash
receipt and content-free baseline, lease, heartbeat and operation timing rows.

The fixture's new 90-second serving gate joins the latest accepted report to
its populated unexpired same-generation lease and the intended live authorized
binding. It also requires matching local desired and kernel generations and a
fresh serving heartbeat. A failed gate gets its own failed admission row; no
first-selection request is launched. Readiness is independent of the original
request's origin/counter assertion. Default measurement rechecks readiness after
observer setup. Historical rows remain unchanged.

The historical node tracer is **not passive** with respect to verification:
its unscoped prerouting `meta nftrace set 1` is a foreign `mangle` statement,
rejected by the unchanged node verifier. Normal kernel tracing now marks only
member packets. `FVPN_E2E_REPRODUCE_TRACE26=1` explicitly reenables the old node
tracer for retained diagnostics; its rows are confounded experiments, never
transport acceptance. The exact private listing replay identifies the tracer
as the deciding rejection separately from the actual central report. F14-01
D-077 captures start before fencing and use passive pcap/session observation
without nft tracing, followed by a fresh serving gate and one original request.

The supervisor starts a monotonic **15-second** cooperative budget after TERM
delivery, then records its existing SIGKILL sidecar and actual parent wait
receipt. PID1 handles TERM/INT by calling the supervised stop path; the waiter
and watchdog outlive agent shutdown. Ordinary compose and secondary/clone
containers allow **25 seconds**. Shared namespace/fleet/clone stop wrappers do
not cut the agent budget short. Container-stop intent restores an automatically
running agent on the next boot; an explicit service pause stays paused.

Before stopping, retain `agent-process-id`, PID and start ticks. After a whole
container stop use `docker cp <container>:/var/lib/vpn-node/e2e/agent-exits/<id>.json -`
(the tar stream) or the private bind-mounted state; `docker exec` is unavailable.
The receipt distinguishes exit code from signal. The container's exit code 0
means its handler completed, and is not evidence that the agent exited 0: the
forced test has container 0 and agent signal 9. The sidecar's `term_requested_at`
is the supervisor's signal-request bound and `waited_ms` measures the monotonic
post-delivery wait. Historical exit 137 rows cannot be reconstructed or replaced.
