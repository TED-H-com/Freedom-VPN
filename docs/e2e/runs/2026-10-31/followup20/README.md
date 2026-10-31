# Follow-up 20 — shared fixture exports and attribution (TEST ONLY)

D-098 was the first commit: [168 SHA-256 pins](../../../followup19-archive.sha256)
cover every file in the follow-up 19 run root. Its private disk copy is
byte-identical. Follow-up 16, 17, 18 and 19 committed roots remain untouched.
Original failed runs below are retained as failed. This campaign changes only
shared harness/deployment files and evidence, without changing product code.

## D-087: real future-grant transition

[Final run 79e59c20](79e59c20-f289-4679-9b37-01f060f2d376/member-grants-fixed-generation.md)
passes 7 / 0 / 0 (pass / blocked / fail) on source `c9d45bdd`.
The real control runtime uses only wire-nodes 33's documented lease-eligibility
reader option. The [consumer sequence](../../../scenarios/harness-exports.md#follow-up-20--d-087-f10-02-and-f10-03-test-only)
exports `SetMemberGrantsReady`, original wire events and invariant snapshots.

The fresh node is published, unfrozen, ready/passed, with desired bindings,
a working meter and counted native traffic. Initial projectors run until their
outputs are stable for 65 seconds. The supported TEST ONLY offline window is
60 seconds, configured before enrollment and never changed during the flip.
Generation **6**, the desired envelope and ETag, bindings, readiness, publication,
freeze flag and entitlement/device outputs remain byte-identical across all
three snapshots. No transport hold, unpublish, mapping freeze, synthetic report
or re-signed lease is used.

| Measured event | UTC |
|---|---|
| Future grants withheld | 16:10:54.742828263 |
| Natural empty renewal request begins | 16:10:58.999195816 |
| Node accepts empty lease | 16:10:59.005571531 |
| Qualifying provisional POST accepted (202) | 16:10:59.118606996 |
| Future grants released | 16:11:00.077699515 |
| Populated renewal request begins | 16:11:04.047641248 |
| Node accepts populated lease | 16:11:04.058221366 |
| Distinct serving POST accepted (202) | 16:11:04.190340903 |

The provisional report has exactly `awaiting_member_grants`, no expiry/safety
reason, three dependent `AWAITING_MEMBER_GRANTS` checks and exactly one lease
error. Its lease ID joins to the real empty signed response. The subsequent
serving report joins to a new populated response; true → accepted serving is
**4,112 ms**. A subsequent native **64,064-byte** transfer completes and both
admission counters increase. All **47 original JSON body SHA-256 hashes** verify
across **53 wire events**. The **103 original node timing records** are contiguous;
both selected renewals are explicitly newly accepted leases. Server completion
is distinguished from node acceptance and native counted forwarding.

The earlier [5ab1f11c run](5ab1f11c-a36b-4b77-bc70-85d902c93da6/member-grants-fixed-generation.md)
also passes 7 / 0 / 0 at generation 7 (3,980 ms); it predates original-body and
node-timing capture. The [b4d2d001 attempt](b4d2d001-a781-4df5-80d2-f7aad09f5b55/member-grants-fixed-generation.md)
stays 6 / 0 / 1: initial FUP projection changed generation 5→6 during the
window. The runner now waits for stability and decodes the actual `signed.members`
field. This exports the prerequisite for isolation 10; it does not close assembled
AT10 or replace its historical durations.

## F10-02: retained reader and enforcement diagnostics

[Final run 977efa83](977efa83-e3bd-4ced-b548-dbc9763ed266/finite-expiry-reader-diagnostics.md)
passes 2 / 0 / 0, retaining **247 samples** through deadline +16 seconds.
Every nft command has category, stage, exit status, wall and same-boot monotonic
bounds, before/after heartbeat and completed enforcement records. All **494**
surrounding records join exactly to the **242-record** original detached trace.
The session-status reader has its own diagnostic, so a future failed combined
read can be assigned to the reader that failed. Missing counters are distinct
from a failed read; no retry or assertion relaxation is added.

Observed category: **none (247/247)**. Last TCP/UDP reply increments were at
16:13:34.880812395Z, **119.187605 ms before** expiry at 16:13:35Z. Senders stayed
active and final counters were stable/removed. Host load from the owned namespace
was 16.45 / 13.65 / 13.03. The earlier [218e18c9 run](218e18c9-50ee-4050-a60e-a96e188b0167/finite-expiry-reader-diagnostics.md)
retains 214 successful samples through expiry +6 seconds without the later full
enforcement trace. No unreadable sample was reproduced. Historical load was not
retained, so comparable load is unproven; F10-02's original missing error cannot
be reconstructed and its failed every-read-success result remains unchanged.

## Regular release layout and recovery checkpoint

[Run d6b01926](d6b01926-de6f-47e4-803b-ae3ea5c603df/update-good.md)
passes **11 / 0 / 0**, actual exit **0**, on source `f438edff`.
Regular and installer artifacts both place the agent/updater/watchdog at the
release root; the installer option adds the installer and original units.
The supervisor distinguishes bootstrap from release-root executable identity.
Shared recovery configs name `/var/lib/vpn-node/state/meter.checkpoint`.
The legacy synthetic four-field file is removed; the running agent must answer
the real fresh D-125 barrier before the updater stops it.

Twenty supervisor stop/start cycles and idempotent repeated start pass. The good
update exits **0**, records `locally_succeeded`, runs the exact root candidate,
and passes **5/5 health checks**, including the issued real tunnel probe.
The [tampered artifact](d6b01926-de6f-47e4-803b-ae3ea5c603df/update-bad.md)
is rejected; the [node restart](d6b01926-de6f-47e4-803b-ae3ea5c603df/node-restart.md)
resumes without enrollment, preserves identity and enrollment rows, and passes
DNS/readback checks. Product executable/journal verification is unchanged.
This supplies updates 11's regular-fixture prerequisite, not full AT23–AT27
qualification. Its recipient-owned `live_test.go` root-path assertions and
`lab_test.go` legacy checkpoint configuration still need adoption by that lane.

## F10-03: two-resolver query attribution

The broader [cce1a688 run](cce1a688-6af5-483c-bd67-2d167c51c04b/dns-negative.md)
is **15 / 0 / 1**, actual exit **1**, source `98a4b460`; DNS passes and the MTU
row below fails. All **36 DNS matrix queries** retain complete resolver/authority
traces. Query payloads and full qnames are not logged. The two actual filtering
resolvers retain the same applied document; no resolver is chosen by the test,
no cache is flushed, and all previous positive/negative TTL assertions remain.

| Positive cache query (resolver completion UTC) | Responder | Returned class | TTL | Authority exchange | Authority SOA sampled after query |
|---|---|---|---:|---|---:|
| Initial 16:28:47.762244225 | primary | other_ipv4 | 10 | observed | 1001 |
| Before TTL 16:28:48.970141306 | primary | other_ipv4 | 9 | none observed | 1002 |
| After TTL 16:29:01.359396477 | primary | private_ipv4 | 10 | observed | 1002 |

The middle answer is an **inferred cache hit**, retaining the old class about
1.208 seconds after the initial response while the authority is already changed.
The after-TTL fetch returns the new class about 13.597 seconds after the first
response. The negative control retains NXDOMAIN, SOA serial 1001 and TTL 4
before expiry while the authority is at serial 1002; afterward it resolves.
A positive cached record has no SOA: the separately sampled serial is never
claimed as its cached serial. Negative policy answers retain `cache_state=unknown`
where the harness cannot separate local policy from cache. Full events, time
bounds and native query duration remain available for consumer interpretation.

The [fallback row](cce1a688-6af5-483c-bd67-2d167c51c04b/native-dns-fallback.md)
identifies an actual secondary response during primary failure. That trace is
marked incomplete because the intentionally stopped primary cannot be read.
The main cache measurement has all three trace files readable. This run does
not reproduce F10-03 and cannot retrospectively attribute its missing responder.
The dated [D-102 answer](../../../RESPONSES.md#members--dns-publish--2026-10-31-follow-up-20-f10-03-d-102)
coordinates inference/serial wording with dns-publish 9 and members 11.

## Instrumented broader MTU-1280 failure

Attempt **1 of at most 3** captured the requested failure, so no further attempts
were run. The [original transfer record](cce1a688-6af5-483c-bd67-2d167c51c04b/native-matrix-inputs.md)
retains **10,010 / 64,064 bytes in 12,011 ms**; its 1500 control completes
64,064 bytes in 2 ms. Both underlay selections remain direct IPv4. All original
inputs/reconnect ordering and the 12-second criterion are retained. The IPv6
refusal/no-uplink-leak controls pass independently.

The [six pinned payload-free decodes and endpoint analysis](capture-cce1a688/README.md)
localize the first missing 1,228-byte TCP range after node tunnel egress and
before member capture. Later data is SACKed around the persistent gap. Nineteen
captured reverse outer packets of length 1,340 are missing at the member;
seventeen of length 1,132 arrive. The origin socket receives the whole upload,
then develops write backpressure with response bytes unacknowledged. All files
end on complete records, decode without header errors, and match tcpdump's
captured counts; filter/capture differences remain explicit. The exact
kernel/virtual-link drop reason is not established. No MTU/MSS workaround is
made. Per-I/O tracing can change segmentation, but does not always eliminate
this failure; historical 11,011/12,012/22,022-byte failures remain unchanged.
