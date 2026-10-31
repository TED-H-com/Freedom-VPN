# G2 follow-up 14 — 2026-10-31

This is a mechanism campaign on the `w1-netlab` lane, based on `c6e3eca`
(PRs #262, #271 and #272 included). It does not replace the assembled isolation
run `bf671f3a-c6ac-4824-9107-425c4c5a5f02` or promote its blocked AT10/AT12 cases.

## D-096: real NetBird ownership and recovery

Measured at **2026-10-31T06:51:18Z**, source `f004b78`, amd64:

| Measurement | Result |
| --- | --- |
| Clean Apply recovery, 10 cycles | median **54.4495 ms**, worst **78.581 ms**, link-up command start to verified topology; upper bounds including reads |
| Address/routes while down | one IPv4 address retained; route entries 4 → 1 → 4 after Apply; retained hashes matched without harness repair |
| Spontaneous raises | **0/10 cycles**, 50 seconds total, nominal 20 ms polling; shorter inter-sample raises are not excluded |
| Deliberate NetBird reconnect raises | **10/10** while quarantine remained pending |
| Reconnect window, median | **1.802286–1.868795 s** lower/upper bounds |
| Reconnect window, worst | **1.930080–2.007971 s** lower/upper bounds |
| Link-up to real ingress readiness | median 766.296 ms, worst 896.360 ms; included in the window, followed by a one-second harness hold |
| Packet evidence during reconnect | **0 captured redirected frames**, **1,071 input-drop packets** across all 10 windows (103–118 each) |
| Quarantine packet evidence | zero redirected frames in all ten down intervals; ten positive controls captured 200 frames before rejection |
| Capture completeness | 294 total redirected frames including fixture setup outside the measured windows; zero kernel capture drops; final capture count matches recorded headers |

**Recommendation:** close D-096 for this pinned native IPv4 path with NetBird
retaining address/route ownership and the agent waiting for verified restoration.
The window is real, but the measured duration includes a controlled harness
pause and is not a production bound. D-100 needs no expectation override.
The topology checker passed in every forced open window; serving verification
rejected the retained nft baseline with `GENERATION_MISMATCH`, and the next
agent Step's current link evidence changed admin-up true → false. The last
emitted heartbeat stayed non-serving. Scope and pending measurements follow.

The fixture uses the client and combined server pinned by `deploy/dev/netbird`
(0.82.0, the same image digests), Ubuntu 24.04 tooling, and kernel WireGuard.
Both peers, management, signal and STUN run inside one `--network none`
privileged lab container and its `fv-member` namespace. There is no third-party
traffic path. Agent control and the capability prober use the existing test rig (the actual
NetBird version and kernel backend are independently read from its CLI), with real
`SystemPlane`, nft/tc, `MemberPath`, heartbeat and observation code. This is not
an assembled control/worker/portal acceptance run.

The client owns `fvnb0`. Its flags match the assembled node: IPv4 overlay,
`--disable-dns --disable-ipv6 --disable-firewall`; Linux network monitoring is
explicitly false. The lab owns only the underlay and isolated uplink.
Quarantine and Apply run through the agent. The harness removes only its own
injected clsact/redirect; it never repairs the member address or routes or
raises the member link after quarantine.

The test measures ten five-second quarantine intervals at nominal 20 ms link
polling, followed by ten deliberate real `netbird down` / `netbird up` cycles.
The harness first waits up to ten seconds for a real ingress packet after
reconnect, then holds the link open for one second before the next agent Step.
The measured window therefore includes peer handshake readiness and this
intentional delay; it is not a production re-containment deadline. A canceled control context on that Step exercises deferred re-containment
without authorizing a clean Apply; the next uncanceled Step performs recovery.
During the open interval the harness invokes the actual `SystemPlane.MemberPath`
checker read-only against the engine's retained expectation and separately
invokes `VerifyServing`. These are diagnostic reads, not an extra engine Step.
The logged heartbeat is the last emitted non-serving heartbeat from rejection;
no fresh heartbeat is emitted during the deliberate pause/canceled Step. The
post-Step quarantine before/after link read is current engine evidence.

The recorded link-up-command-start to topology durations are **upper bounds**:
they include the link command itself,
post-raise nft/tc verification, the subsequent agent live read and `MemberPath`.
They are not daemon-reconciliation latency, and do not attribute each route
restoration to the daemon rather than kernel reactivation of its retained
configuration. IPv4 addresses and connected routes
are sampled while down, so retained configuration can be distinguished from
configuration the daemon had to recreate. The agent learns one address hash
and four route hashes on the initial real peer path.

The complete per-cycle output is retained in `G2-followup14-evidence.txt`.
Lower bounds run from the first completed up sample to the down-command start;
upper bounds run from the last down-sample start to down-command completion.
The raw aggregate line truncates the half-microsecond median; the table uses
the ten per-cycle integers.

### Packet evidence and its limits

A real member sends a fixed TEST ONLY UDP datagram (source port 40000,
destination port 39997, 42-byte IPv4 packet) to the node's NetBird overlay
address every 10 ms. An explicitly planted `matchall`/`mirred` ingress redirect
is the positive control. The uplink capture selects its IPv4 header and UDP
port and records only header summaries/counts, never payloads. Capture shutdown
checks the final tcpdump capture count against recorded headers and requires
zero kernel capture drops. A layer-3
WireGuard packet redirected directly onto the Ethernet uplink has **no Ethernet
header**; it is an unmetered raw frame, not proof of a usable Ethernet/IP
session reaching an application. The capture accounts for that layout rather
than incorrectly declaring zero packets from an Ethernet `udp` BPF filter.

The whole reconnect interval is captured, beginning before the CLI disconnect.
A diagnostic attempt was rejected when one window had zero real ingress; the
final fixture waits for an actual input-drop increment before its one-second
hold, so handshake silence cannot count as a no-bypass pass.
After reconnect, the node's `ctin` input-drop counter provides the real ingress
positive control. NetBird's interface recreation removes the old clsact/filter;
the retained nft baseline rejects that traffic. This does not prove protection
against a surviving/reinserted redirect, an uplink filter, userspace routing,
IPv6 or a privileged process racing the verifier.

### Operator recommendation and D-100

For this pinned **native IPv4 NetBird-owned member interface**, keep NetBird as
the address/route owner and let the agent wait for D-096 read-back. Do not add
route/address writes to `node/enforce` or clear topology expectations to force
recovery. Ordinary down/up retains the IPv4 address; routes become available again without
harness repair in this fixture. A deliberate
NetBird reconnect recreates it. An installer-owned transit interface or IPv6
route that disappears is a different ownership case: its topology owner must
reconcile it while the agent remains non-serving. The earlier G2 veth repair
must not be described as work NetBird performed.

A NetBird reconnect is an administrative link raise even while the quarantine
marker remains. Operators must treat heartbeat non-serving and a retained
quarantine marker as policy state, not a continuous physical-link interlock.
Next-step re-containment and the independent nft baseline are separate facts.
Do not restart NetBird as an unguarded recovery shortcut during an unresolved
foreign tc fault.

D-100 needs no out-of-band override. Retain its clean, verified newer published
generation requirement for changed topology expectations. Document the topology
owner and the reconnect interval explicitly; an unchanged topology becoming
available again is recovery against the retained expectation, not a refresh.
IPv6, network-monitor=true, actual arm64 hardware, relay paths and continuous
protection against a surviving redirect remain **待實測**.

## ISO-F6-01: rejected tc state

Ten disposable-interface/capacity Apply cycles in the ordinary lab, and ten
interface add/remove cycles through the real agent and NetBird client, exercise
the reported transition. No foreign tc state is planted during those cases.
The separately injected redirect is labeled and is not a reproduction of the
original fault. The original observation `f2cfdff8-3e46-49e8-86f1-b38128ea6cd4`
lost its rejected object; its post-rollback noqueue listing cannot establish
which path created that object. A passing reproduction attempt cannot close
that finding.

`ServingEvidence.TCSummary` now identifies the candidate generation, read phase
(`load`, `pre_apply`, `verify`), full read/object counts, an RFC 8785 digest of the retained bounded
sanitized rows and total counts, truncation, and the deciding rejected attachment/object.
The library captures this before quarantine or rollback, including initial tc
load failures and class-listing failures. `TC` retains at most 16 attachments,
four objects each and four action kinds per object; tokens are at most 32 ASCII
characters. Selectors, action parameters, command output and counters are not
serialized. The rejected row is placed first for the current agent's existing
report-only `errors[stage=tc_evidence]` transport, even if it was outside the
normal bounded sample.

The capacity update in the original case was later than the rejection
(04:40:52.423892Z versus 04:40:51.80844254Z), so it is not identified as the
trigger. The library cycle changes capacity as an additional transition; the
real-client interface cycle does not publish a newer generation.

A separate **agent recovery defect is reproduced** in
`TestISO_F6_RetainedFailureBlocksNewGeneration` (owned
`node/agent/followup12_test.go`). After a real library tc rejection, remove the
foreign fixture and publish generation 3 while applied generation/lease remain
2. Five steps and five seconds later: zero new Apply calls, zero lease renewals,
`needRenew=false`, retained `TC_FOREIGN_FILTER` plus `policy_missing`. The
original assembled run used a restart to recover; this focused fixture does
not substitute for that recorded action. The latch also appeared after a
real-client reconnect when the diagnostic harness changed publication to a
newer generation. It is the subsequent latch, not proof of the original tc
object's creator.

Exact agent-core 23 path:

1. `engine.go:noteApply` retains `applyError` from the tc rejection.
2. `sync.go:needRenew` suppresses generation-driven renewal when `applyError`
   exists; its generation comparison is against applied generation, which is
   still the old generation.
3. `engine.go:enforce` omits the error only from its candidate `Decide` call.
   `decide.go:leaseGenerationStale` still yields `policy_missing`, so that
   candidate installs the baseline rather than running a serving Apply.
4. `engine.go:ackApplied` refuses the candidate generation while the actual
   `e.applyError` is non-nil. `sync.go:renewAppliedLease` also returns early on
   that error. No clean Apply is reached to clear the retained error.

This lane leaves those production files unchanged. Agent-core must break the
recovery dependency without clearing genuine fault evidence or acknowledging
unverified serving. A failed historical TC read must remain identifiable as
historical while a new-generation recovery attempt is pending.

Verification still checks the full live read. Truncation is only reporting;
no filter, qdisc or unreadable attachment becomes allowed. Unit tests prove
late rejected-object retention, deterministic ordering, decimal generation
precision, redaction, load-error retention and continued quarantine.

## ISO-F6-02: product transport handoff

`ServingEvidence.ReportDetail()` returns a deterministic summary within the
existing 500-character `NodeCheck.detail` / `NodeError.message` limit. The lab
Apply/verify read-back exposes the bounded `tc` rows and `tc_summary` directly.
The real agent already forwards failed TC rows through report-only errors, so
prioritizing the deciding row improves that existing surface immediately.

A successful live read is still discarded by `node/agent/serving.go`'s
`verifyServingModel`; `engine.go` also does not retain successful Apply evidence
for `observe.go`. Those production files are outside this lane. Agent-core 23
must retain the last actual read with its candidate generation and read time,
attach `ReportDetail()` to the **qos check detail**, and preserve the actual
check status/error. Do not invent a successful NodeError, label a rollback
snapshot as the rejection, or label an old Apply snapshot as a fresh read.
Failed reports should use the rejected read plus its original error code.

For contracts revision 14, the proposed typed addition is optional nullable
`ObservationReport.tc_evidence`, referencing a new `TCReadbackEvidence` object
with `additionalProperties: false`:

| Field | Type and bound |
| --- | --- |
| `candidate_generation` | uint64 decimal string, existing 20-digit/max-uint64 convention; not an ACK |
| `read_at` | UTC RFC 3339 date-time with `Z`; actual verifier read time |
| `phase` | enum `load`, `pre_apply`, `verify` |
| `verified` | boolean; actual complete serving-verifier outcome |
| `error_code` | nullable existing `ErrorCode` |
| `reads_total`, `objects_total` | integer, 0..2147483647 |
| `truncated` | boolean; any omission from retained rows/actions |
| `sha256` | lowercase hex string, exactly 64 characters, RFC 8785 digest of retained bounded rows and total counts |
| `attachments` | array, maxItems 16, `TCReadAttachment` |
| `rejected` | nullable `TCReadAttachment`, at most one deciding object |

`TCReadAttachment`: required `interface`, `attachment`, `objects`; optional
`error`. All strings use `^[a-zA-Z0-9_:-]{1,32}$` after producer redaction;
`objects` has maxItems 4 (maxItems 1 in `rejected`). Each object has required
`kind`, optional `handle`, `parent`, boolean `root`, and `actions` with maxItems 4
of the same bounded token type. `error` is the enum
`read_failed`, `invalid_json`, `invalid_listing`; never raw stderr. The parent
observation's report ID supplies the observation linkage. A null/absent field
means no retained read, never an empty successful scan.

This is a schema proposal, not a shipped contract or a claim that observation
`4a1dc037-c2c0-4955-90a5-c19a2928e998` has been repaired retroactively. No contract
or agent production file was changed here.

## Running the campaign

```sh
FVPN_LAB_PREFIX=fvpn-w1-netlab-real make -f mk/netlab.mk netlab-real-netbird
FVPN_LAB_PREFIX=fvpn-w1-netlab-check bash tests/netlab/scripts/gates.sh lab
make -f mk/netlab.mk netlab-lint netlab-unit netlab-integration netlab-build
```

Both launcher modes wait while two `gates.sh` campaigns are active,
keeps privileged scratch in the container's disk overlay, and removes its
container and images on exit. All Go test scratch is on disk. The build target
cross-compiles the lab binary for arm64; this is not an arm64 hardware run.
