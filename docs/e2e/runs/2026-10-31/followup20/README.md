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
