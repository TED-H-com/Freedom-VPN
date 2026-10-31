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

Exact call path, rechecked on follow-up 16 (base `e498225`):

1. `engine.go:noteApply` retains `applyError` from the tc rejection.
2. `sync.go:needRenew` suppresses generation-driven renewal when `applyError`
   exists; its generation comparison is against applied generation, which is
   still the old generation.
3. `engine.go:enforce` omits the error only from its candidate `Decide` call.
   `decide.go:leaseGenerationStale` still yields `policy_missing`, so that
   candidate installs the baseline rather than running a serving Apply.
4. `serving.go:auditBlocked` still calls `VerifyServing`, but only a new
   `TC_FOREIGN_FILTER` updates `applyError`; clean tc plus the retained nft
   baseline returns `GENERATION_MISMATCH` and never clears the old error.
   `verifyCurrentServing` returns early when already non-serving.
5. `engine.go:ackApplied` refuses the candidate generation while the actual
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

## ISO-F6-02 / D-106: product transport handoff (follow-up 16)

Contracts revision 14 shipped `ObservationReport.tc_evidence` in #289. The
producer entry point is now:

```go
if tc := evidence.TCReadbackEvidence(); tc != nil {
    report.TcEvidence.Set(*tc)
}
```

Call it on the `ServingEvidence` returned by `VerifyServing` or on
`ApplyResult.Evidence`, including error returns. The helper returns
`*api.TCReadbackEvidence`; nil means no retained read. Capture this typed value
before serializing or discarding `ServingEvidence`: its transport snapshot is
private, intentionally separate from the public diagnostic JSON. Each call
returns a detached copy. Mutating the copy, public `TC` rows, or quarantine
annotations cannot rewrite the retained snapshot.

The snapshot is captured after the complete verifier finishes reading and
before any rollback or quarantine. `read_at` is the actual UTC completion time,
not model `Now`, observation emission time, or conversion time. Conversion
performs no I/O and never refreshes it. `verified` describes the complete
nft/tc/QoS serving verifier. Clean tc with a failed nft rule/deadline/quarantine
check remains false. Ownership-only `load`/`pre_apply` reads cannot be verified.
It is not an Apply/engine success flag: later quarantine release, conntrack,
member-path or lease checks may still fail; callers must honor the returned
error and existing serving gates. No verifier was weakened.

| Read-back element | `TCReadbackEvidence` field and semantics |
| --- | --- |
| `TCSummary.Generation` | `candidate_generation`: exact `decstr.U64` decimal string through max uint64; never an ACK |
| Read completion sample | `read_at`: UTC RFC 3339 with `Z`, retained before rollback |
| `TCSummary.Phase` | `phase`: `load`, `pre_apply`, or `verify` |
| Complete `verifyServing` result | `verified`: true only for the full verifier's nil error, not merely an empty foreign-filter scan |
| Returned `CheckError.Reason` | `error_code`: the matching existing HTTP enum, including `TC_FOREIGN_FILTER`; null for success or an unknown non-CheckError, never raw error text |
| `TCSummary.Reads`, `.Objects` | `reads_total`, `objects_total`: counts before row selection, capped at 2147483647 with `truncated=true` on saturation |
| `TCSummary.Truncated` | `truncated`: any omitted rows/objects/actions, including siblings displaced by the deciding row |
| `TCSummary.SHA256` | `sha256`: unchanged netlab canonical digest below |
| Bounded `TC` | `attachments`: at most 16 retained attachment reads, populated/error rows before empty ones |
| `TCReadEvidence.Interface`, `.Attachment` | `attachments[].interface`, `.attachment`: member/uplink interface and qdisc/root/ingress/egress/class/parent handle |
| `TCReadEvidence.Error` | optional attachment `error`: only `read_failed`, `invalid_json`, `invalid_listing` |
| `TCObject` | `objects[]`: at most 4, carrying `kind`, optional `handle`, `parent`, true `root`, and at most 4 ordered action kinds |
| `TCSummary.Rejected` | nullable `rejected`: deciding attachment with at most 1 object; same deciding row first in `attachments`; null when no TC decision exists |

Tokens match `^[a-zA-Z0-9_:-]{1,32}$`; invalid, empty required, or credential-like
tokens become `redacted`. Optional empty handles/parents remain omitted. No raw
selectors, action parameters, stderr, routes, destinations, payloads or counters
enter the contract. Reporting bounds never shorten the verifier's scan.

D-106 has **no foreign-object count field**. `objects_total` counts all observed
objects, not foreign objects. A verified full read proves zero foreign objects;
a rejected read identifies its deciding object, not an exhaustive foreign-object
count (the verifier can stop on a read failure). Do not relabel total objects as
foreign or invent an extra field in the closed schema.

`sha256` is RFC 8785 SHA-256 over exactly:

`{reads: reads_total, objects: objects_total, truncated: truncated, tc: attachments}`.
Counts are JSON integers, truncated is a boolean and tc is the bounded array. Array order and optional
member presence are significant. Timestamp, generation, outcome and rejection
are outside this existing diagnostic digest; it is not a signature or a digest
of omitted raw kernel data.

The generated Go schema/round-trip test regenerates the verified, rejected and
truncated vectors from producer rows (including the omitted seventeenth row),
then matches the corresponding ObservationReport examples. Digests:

| Example | Netlab = contract = ObservationReport example SHA-256 |
| --- | --- |
| verified | `058a980c795256f70ca7fe4490730c9b7fb6cd05a7e99d0f37e3cb34200ebcb2` |
| rejected | `c57355cb80736e4023b422916bb7c002e1bb771dd134042dbcc1e9b1bc0862bf` |
| truncated | `a41f0ebfd4d2ee92f8e250d99adae9f6d449cec76cadb420605ce121f31cc017` |

`node contracts/ts/verify-vectors.ts` independently verifies these canonical
projections. `ServingEvidence.ReportDetail()` and the existing
`errors[stage=tc_evidence]` rejection transport remain available and unchanged.
Success belongs in `tc_evidence`, never a manufactured NodeError. Agent-core
follow-up 25 must retain successful Apply/live-read evidence from
`engine.go`/`serving.go` for `observe.go`; those production files are unchanged
here. Keep a historical failure's original time/generation separate from the
latest read, and do not label a post-rollback baseline snapshot as the original
rejection. Null/absent means no retained read, never an empty successful scan.

## ISO-F6-01 follow-up 16 recovery proof and isolation follow-up 8 assertions

The original follow-up 7 run `041e9e48-79a6-45dd-a19d-8b5a63290655` removed its
injection at 06:29:32.889Z, requested generation 32, and remained applied at 31
with `TC_FOREIGN_FILTER` / `policy_missing`. The extra generation 33 and assisted
route restoration at 06:31:25Z did not establish recovery. This is consistent
with the agent dependency above, not proof of a cached library verdict.

`TestISO_F6_01_ForeignRemovalNewGeneration` reproduces the sequence with real
kernel tc/nft in the privileged lab: generation 31 serves; injected
clsact/mirred is rejected; dirty Apply quarantines; removing only the filter
still rejects the foreign clsact; removing both yields a fresh
`GENERATION_MISMATCH` against the baseline, with no rejected TC object; clean
Apply generation 32 verifies, clears the marker and raises the member link;
a subsequent public `VerifyServing` also verifies. The dated retained trace is
`G2-tc-evidence-2026-10-31/TestISO_F6_01_ForeignRemovalNewGeneration/trace.json`.
The final member probe is measured **after explicit lab-owned veth route/IPv6
reconciliation**. This is library mechanism evidence, not assembled agent
serving or a claim that NetBird restored this lab veth.

The unprivileged same-executor regression also proves three fresh ingress
reads during the clean Apply (ownership, staged verification, post-link-raise
verification), and that the original rejection snapshot survives unchanged.
The existing `TestISO_F6_RetainedFailureBlocksNewGeneration` still reproduces
five steps/five seconds with desired 3, applied/lease 2, zero new serving Apply
calls, zero renewals, and `needRenew=false`. No production agent file changed.

Isolation follow-up 8 must assert all of the following before closing ISO-F6-01:

1. Capture the initial serving generation, injected object/attachment and actual
   rejected read/time, plus quarantine and non-serving heartbeat evidence.
2. Remove **all** injected foreign objects (an empty filter block with foreign
   clsact remaining is not clean). Retain fresh read-back proving absence.
3. Publish the clean newer generation after rejection. Show the engine actually
   reaches Apply, with the matching lease generation and pending/quarantine
   transitions; a 202 or repeated blocked audit is insufficient.
4. Retain successful Apply and fresh live `tc_evidence` with the newer candidate
   generation, actual UTC read times, verified=true, null error/rejection and a
   recomputable digest. Keep historical failed reads explicitly historical.
5. Prove restored member-path/topology, marker clearance, serving heartbeat and
   observation, applied generation, and member packet/counter positive control.
   Record the address/route owner and any harness assistance. A clean library
   result alone does not close the assembled recovery defect.
6. Keep the failure control: with the foreign object still present, a newer
   generation stays non-serving, retains `TC_FOREIGN_FILTER`, and never deletes
   the foreign object as part of enforcement.

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
