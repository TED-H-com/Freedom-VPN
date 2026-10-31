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
miss; no exchange supports only a labeled cache-hit inference. No upstream
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
