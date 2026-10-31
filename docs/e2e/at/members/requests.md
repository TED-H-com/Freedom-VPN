# Members requests and external blockers

Current full-case status and evidence: [summary.md](summary.md). Follow-up 9 run
`11e305fe` is immutable in [follow-up 9](history/follow-up-9/RUN.md), with its
[33-file checksum index](history/follow-up-9/SHA256SUMS) (D-098). Historical
run `69f92f49` and its requests are immutable in [follow-up 8](history/follow-up-8/requests.md),
with [32-file checksum index](history/follow-up-8/SHA256SUMS) (D-098).

## Follow-up 10 — 2026-10-31 (D-102, D-126, D-128)

Baseline `d3e9a0f1` includes agent-core 27 (#325), wire-nodes 32 (#321) and
netlab 19 (#318). **F9-01 is closed as a predicate mismatch under D-126**;
see the producer/central answers in its section below. The replay accepts the
three original provisional observations, including the mixed safety report,
and rejects unexplained safety codes. It does not rewrite the archived failed
run or prove live recovery. The fresh full mutation sequence is recorded in
[RUN.md](RUN.md); full-case status continues to include resource blockers.

**Fresh run `2be2d32f`: 2 pass / 6 blocked / 2 fail; 560 / 5 / 2 subchecks.**
AT09 has 38 pass / 5 blocked / 0 fail checks, with live mixed-report acceptance
and recovery, but the full sequence produced generation 29, not 30 (F10-04).
F10-02 (AT02 unreadable sample) and F10-03 (AT20 before-TTL cache result) remain
failed. F10-01 is the external pin correction. The new dated sections below
retain captures and owners; none of these findings erases older evidence.

F8-02 stays unattributed. Netlab 19's answer and devstack 19's exact-sequence
decode ownership are recorded below. The dated F4-03 / F4-04 / F4-05 / F4-06 /
F4-07 / F4-10 resource requests and AT20–AT22 DNS expectations are unchanged.

## Follow-up 9 — 2026-10-31 (D-102)

Baseline `7e044c71` includes agent-core 25 (#312), devstack 18 (#313),
dns-publish 8 (#309), and netlab 18. The original full mutation sequence
adopts `RestartNodeNamespace90`; no standalone scenario pass changes AT09.
Run `11e305fe-6360-4dbd-95b1-67274aa4988c`, harness `49fbb48e`, finishes **3 pass / 6 blocked / 1 fail**. AT09 remains failed on F9-01 below. F8-01 recovers in the original generation-30 sequence; the native MTU-1280 row passes this time, while the historical F8-02 cause remains unattributed.

### D-119 — finite expiry and billing replacement

Resolved by orchestrator, 2026-10-31: retire the inherited “3/0 → 2/1 deadline transition” wording.

Plan v1.0 has no scheduled plan-change contract. A finite
grant's deadline means expiry (0/3 when no other grant covers the member).
A period-boundary lower plan is a billing event that calls `ReplaceFiniteGrant`
then, immediately revoking and creating the replacement. AT02 retains natural
expiry → 0/3 and immediate replacement → 3/0 → 2/1 → 3/0. No new production
schedule, fallback grant or overlapping different-plan grant is fabricated.

### F8-01 — closed by full mutation sequence recovery, 2026-10-31

Owners: **w2-agent-core / w3-devstack-e2e**, with **w3-wire-nodes** for lease
readback. [Agent-core 25](../../../node/2026-10-31-d104-d106-agent.md) keeps
recovery fail-closed and renews against valid desired generation while an Apply
error is retained. The [devstack 18 evidence index](../../runs/2026-10-31/followup18/README.md)
reports unchanged generation 3 recovery in 5.96 s with the initial
`TC_FOREIGN_FILTER` retained. It explicitly does not close the original
generation-30 failure. This lane records the complete 90-second window,
boot/final nft/tc, heartbeats, observations, agent log, generation equality,
first fresh serving receipt and first counted byte without repair.

**Follow-up 9 result (2026-10-31):** desired/applied generation **30 → 30**; namespace restart
`2026-10-31T12:06:31.816656225Z`; first fresh serving receipt
`12:06:47.545258Z` (**15.728602 s**); first counted byte
`12:06:53.190469871Z` (**21.373814 s**). A new generation-30 lease at
`12:06:47.408641Z` has seven member grants. The capture retains **88 heartbeat
samples over 90.337054 s**, boot/final nft/tc, every central observation and the
agent log, including the initial `TC_FOREIGN_FILTER`. No generation, topology
or state repair occurred. [Complete historical capture](history/follow-up-9/AT09-restart.json).
This closes the original non-recovery failure, not the full AT09 case.

### F8-02 — MTU 1280 bulk transfer incomplete

Owner: **w3-devstack-e2e / native fixture investigation**; wire-nodes and netlab
supply path evidence. The archived failure remains **11,011 / 64,064 bytes in
12,011 ms**, direct IPv4 selected before/after, counters +66,296/+45,963.
[Devstack 18](../../runs/2026-10-31/followup18/README.md) reproduced partial
transfers with and without reconnect and with either underlay family; measured
SYN MSS 1240/1460 and no fragmentation-needed/PTB in its bounded captures.
It cannot assign the exact loss/MSS/fragmentation or fixture read/echo cause.
[Netlab 18](../../../poc/G2-f802-mtu1280-2026-10-31.md) completes IPv4 at 1280
and proves IPv4 fragmentation-needed errors traverse nft/tc in both directions.
Neither independent result attributes the original native failure.

**Follow-up 9 result (2026-10-31):** dual underlay / MTU 1280 completes **64,064 / 64,064 bytes in 1 ms**, with direct IPv4 selected before and after; device counters increase **67,652 upload / 68,139 download bytes**. All three preceding 1500 rows also complete. [Archived NATIVE.md](history/follow-up-9/NATIVE.md) and [AT09.json](history/follow-up-9/AT09.json) retain every family/path and counter. This is a passing row in the full mutation sequence, not proof of a root-cause fix or an overall AT09 pass.

**D-123 (2026-10-31): IPv6Block is the v1.0 overlay mode.** The rendered
`ctv6` drops in both directions prove the netlab IPv6 no-leak control; no
end-to-end IPv6 MTU/PMTU row exists or is claimed. Native family selectors
control underlay availability; all bulk payloads here use the IPv4 origin.
Keep F8-02 unattributed between NetBird reconnect behavior and fixture echo/read
behavior until the same row has SYN/MSS, ICMP errors, interface MTUs and
application read counts sufficient to decide. Do not call it an IPv6 problem.

**Netlab 19 answer (2026-10-31, #318):**
[G2 F8-02 attribution](../../../poc/G2-f802-native-attribution-2026-10-31.md)
finds no single transport cause: the failed and successful IPv4 captures have
unchanged 1240/1460 SYN/SYN-ACK MSS, and published SYN-only summaries cannot
resolve TCP retransmission, outer fragmentation or endpoint read/write stalls.
No central/nft/tc cause or production workaround is established. **Devstack 19
owns the exact-sequence decode/capture**: full payload-free TCP and outer
headers, capture completeness and endpoint read/write/partial-line counts,
with the original 12-second deadline and 64,064-byte requirement. The original
11,011-byte failure remains failed; the follow-up 9 MTU-1280 pass is no fix proof.

### F9-01 — mixed provisional lease and tc-rejection observation, 2026-10-31

**Closed 2026-10-31 under D-126 / D-128.**
[Agent-core 27's hand-off](../../../node/followup27-members-2026-10-31.md)
and [PR #325](https://github.com/TED-H-com/Freedom-VPN/pull/325) confirm the
existing deliberate synchronous single-decision composition, **no reporter
race and no producer fix**. `12:06:32.527615Z` is receiver time; the captured
body's `observed_at` is `12:06:32.521093013Z`. Reasons accumulate; a retained
enforcement failure overrides the three dependent checks, while the sole
lease-stage error retains `AWAITING_MEMBER_GRANTS` and its provisional message.

[Wire-nodes 32, PR #321](https://github.com/TED-H-com/Freedom-VPN/pull/321)
proves acceptance, blocked publication, stored-ACK replay and value-unchanged
storage/projection of this exact report (JSON-value equality, identical JCS
and SHA-256). Under D-128, JSONB need not preserve whitespace/key order;
`SubmitObservation` remains 202 Accepted. No schema or HTTP change is needed.

The retained revision-11 predicate now permits each failed lease/qos/source_binding
check to carry AWAITING, or the active enforcement safety code only with
`safety_check_failed`, failed safety and matching apply/preflight stage evidence.
Historical tc/quarantine rows alone cannot justify that override. Clean
provisional checks still require AWAITING, including the lease detail; exactly
one stage=lease error must retain the provisional cause; `lease_expired` stays
rejected. The offline test replays the entire six-report captured sequence and
its JCS values, accepts all three provisional rows, and rejects the mixed row
without the safety reason as well as missing active-stage evidence, overwritten
or duplicate lease causes, expiry and a passed dependent check. The captured tc
rejection remains in the immutable archive and is never converted to safety
success. Full AT09 acceptance remains separate from recovery or replay success.

**Historical follow-up 9 failure/request (superseded by the answer above):**

Owners: **w2-agent-core / w3-wire-nodes**, with orchestrator contract review.
The [archived restart capture](history/follow-up-9/AT09-restart.json) contains three non-serving observations. At
`2026-10-31T12:06:32.527615Z`, reasons include both `safety_check_failed` and
`awaiting_member_grants`; safety is failed, and lease/qos/source_binding checks
carry `TC_FOREIGN_FILTER`. Its sole stage=lease error still carries
`AWAITING_MEMBER_GRANTS`. The neighboring reports at `12:06:32.473389Z` and
`12:06:37.574574Z` carry the expected AWAITING codes on all three checks.
There is no `lease_expired` report, and serving subsequently recovers.

The retained revision-11 assertion requires AWAITING codes for every observed
provisional report, so **`D-067 revision 11 provisional lease observation` fails**.
The composite **`same-generation D-067 restart recovery and lease sequence`**
check also fails through that predicate, despite matching generation, a new
populated lease, serving and counted traffic. These are the run's only two
failed checks; AT09 remains **fail**, with 36 pass / 5 blocked / 2 fail subchecks.

Please settle the intended check-code precedence when a real boot tc rejection
coexists with a provisional lease, against [node D-067 wording](../../../node/README.md)
and [central read-model semantics](../../../nodes/README.md). If the TC override
is intended, publish the mixed-state contract before narrowing this assertion;
otherwise correct the observation. The evidence alone does not establish
which owner should change. No expectation is weakened or old rejection erased
here, and this reporting gap is not presented as a renewed F8-01 outage.

### F4-05 / F4-07 — DNS wording retired; other gaps retained

DNS-wording part retired **2026-10-31**, per [dns-publish follow-up 8,
BACKEND.md §2.1.1–§2.1.4](../../../dns/BACKEND.md). Native assertions cite:

| Queries | Exact filtered response | Backend row / five-second profile |
|---|---|---|
| A direct / blocked alias / subdomain | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-A scopes D / C / W; F8-TTL5 |
| AAAA | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-AAAA; F8-TTL5 |
| Direct blocked CNAME | RCODE 0 NODATA, zero answers, SOA TTL 5 | F8-CNAME; F8-TTL5 |
| SVCB | RCODE 0 NODATA, zero answers, SOA TTL 5 | F8-SVCB; F8-TTL5 |
| HTTPS | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-HTTPS; F8-TTL5 |
| AT21 fallback A, AT22 blocked A sentinel / last-good | RCODE 3 NXDOMAIN, zero answers, SOA TTL 5 | F8-A (sentinel W); F8-TTL5 |

Every query retains observed RCODE, answer types, SOA presence/TTL; cache state
is separate. Positive and deliberately absent upstream controls are not
synthetic filtering rows. TXT/MX/NS (not queried here) are NODATA under F8-TXT /
F8-MX / F8-NS; no type used by this matrix lacks a table row. The AT22 unique
sentinel now sits under the already configured `lab.example.invalid` wildcard
block, so it is a genuine filtering control rather than an upstream NXDOMAIN.

Remaining owners: **devstack/native integration** for positive IDNA wire data
and production automatic nameserver distribution; **operator/platform** for
Private DNS/browser DoH. ConfigureNativeDNS distribution remains TEST ONLY.
No complete AT20–AT22 status is promoted by resolver-only measurements.

### F4-03 / F4-04 / F4-06 / F4-10 — resources still required

- **Platform/operator resources:** AT01 native Android/iOS/Windows/macOS,
  AT09 QUIC, physical second NAT/roaming/PMTU; exact existing procedures in
  [OPERATORS.md](OPERATORS.md) and [PROCEDURES.md](PROCEDURES.md).
- **Portal/devstack/operator:** AT04 rendered expired/suspended screens.
- **Packaging/devstack/operator:** AT05 preinstalled/cloud-init provenance and
  physical arm64. Cross-compilation supplies no physical-device claim.
- **Devstack/usage/database/backup:** AT22 raw-usage/rollup/file/backup retention
  and representative URL privacy. Commercial feed approval remains business
  work. Guarded audit purge and TEST ONLY feed last-good do not close these.

### Retired identifiers — 2026-10-31

F4-08 / F4-09: never defined; retired 2026-10-31. The follow-up 4 ledger is
F4-01 … F4-07 and F4-10. No replacement definition or owner is invented.
The separate worker RestartPolicy issue remains closed by #240.

### D-098 handover pins

The members8 claim test already reads immutable `tests/handover/testdata/`
snapshots, so its claims remain true without editing outside this lane.
Handover may adopt the now-published follow-up-8 archive and SHA256SUMS in its
own lane; no outside pin change is required for this rerun.

### Harness correction retained — 2026-10-31

[Interrupted attempt 1](history/follow-up-9/attempt-1/RUN.md) keeps all three
stale audit-selector failures and the read-only diagnostic. Current production
uses `member.grant_revoke`; the owned query now matches it. The final fresh run
passes all 122 AT02 checks without reducing the single-audit replay assertion.
No request to another lane is needed for this correction.

## Follow-up 10 pin request — 2026-10-31 (D-098, D-102)

### F10-01 — netlab attribution source pin still names mutable AT09

Owner: **netlab / orchestrator**. This is an evidence-index correction, not a
new runtime failure. `docs/poc/G2-followup19-evidence/source-pins.json` points
its members input at `docs/e2e/at/members/AT09.json`, while SHA-256
`c89ba7b98285a0883889c83455ed24368e1b73859978fe11a1f008012d09ea3c`
identifies the original failure in
[history/follow-up-8/AT09.json](history/follow-up-8/AT09.json). The root file
already held follow-up 9 at this task's start. Please change that pin's path
to the immutable archive, retaining its digest, and update the original-members
link in `docs/poc/G2-f802-native-attribution-2026-10-31.md` likewise.
Both files are outside this lane and remain untouched; only this correction
is deferred. The follow-up 8 archive and all 13 handover snapshots still match.
The original failure's attribution and devstack 19 capture request are unchanged.

## Follow-up 10 new failure — 2026-10-31 (D-102)

### F10-02 — AT02 finite-expiry withdrawal has an unreadable counter sample

Owners: **members harness / devstack counter reader**, with **agent-core / netlab**
if the missing read is traced to enforcement. Run `2be2d32f-bfd9-46f5-bf29-afaa6617ae94`,
harness `731be5a7`, retains **AT02 fail: 121 pass / 0 blocked / 1 fail**.
[`persistent withdrawal member-a3`](F10-02-withdrawal.json) has 234 samples;
the one at `2026-10-31T13:35:45.333735552Z` has `read_ok=false` and counters
`Present=false`, while the transport status is running without an error.
The adjacent reads succeed. The current capture does not retain the counter
reader's error category, so its root cause is not attributed.

The finite deadline is `13:36:39Z`. Last TCP and UDP reply increments are
observed at `13:36:38.987828874Z` (12.171126 ms before expiry); senders stay
active, and final counters remain stable at 32,447 upload / 26,893 download
bytes. Genuine expiry to 0/3, its three audit events, restoration to 3/0 and
counted traffic pass. These observations do not establish a post-expiry leak,
but do not repair the missing read: `finishPersistent` requires every sampled
read to succeed, so **the failure is retained without a retry or relaxation**.

Please expose the bounded counter-read error category and correlate the failed
read with the original enforcement/read timestamps before attributing it.
[AT02.json](AT02.json) keeps the complete mutation evidence; the linked capture
copies the failed check and all 234 samples by JSON value. No product or
persistent-withdrawal assertion was changed in this follow-up.

### F10-03 — AT20 positive cache changes before the retained TTL, 2026-10-31

Owners: **devstack/native DNS fixture / dns-publish**, with the members harness
for query-path evidence. Run `2be2d32f`, harness `731be5a7`, retains **AT20 fail:
80 pass / 0 blocked / 1 fail**. The unchanged `positive/negative cache before
TTL` assertion fails; the per-type DNS filtering/RCODE/SOA/TTL rows still pass.
[Cache capture](F10-03-cache.json), [complete AT20 evidence](AT20.json).

The initial positive query at `13:52:11Z` returns the old `other_ipv4` answer
with TTL 10 (2,502 ms query duration). After the authority-only change, at
`13:52:12.96280019Z` the positive query already returns `private_ipv4`, TTL 10,
`expected_answer=false` (1 ms). The negative control remains NXDOMAIN with
SOA TTL 1. At `13:52:25.057177644Z`, both after-TTL expectations pass.
No cache flush or DNS expectation change was made.

The native two-resolver path does not identify the responding resolver per
query in this capture; the 2,502 ms initial latency does not by itself prove
fallback, cache bypass or a product TTL defect. Please correlate resolver
selection/cache receipts and the authority phase timestamps before assigning
cause. Preserve this failed before-TTL row. Positive IDNA, production resolver
distribution and platform requests remain open independently.

### F10-04 — full sequence yielded generation 29, not requested 30, 2026-10-31

Owner: **members / devstack / orchestrator qualification review**. The complete
unchanged mutation order ran on baseline `d3e9a0f1`, harness `731be5a7`, run
`2be2d32f`. `RestartNodeNamespace90` observed desired and applied **29 → 29**,
not the requested generation 30. No extra mutation, state/topology repair or
generation bump was made to force the number. [Complete capture](AT09-restart.json)
and the `same-generation D-067 restart recovery and lease sequence` row in
[AT09.json](AT09.json) retain the actual values.

Recovery passed at generation 29: serving in 15.751594344 s, first counted byte
in 21.357052110 s; three provisional reports satisfy D-126 without lease_expired.
All 38 executed AT09 checks pass, five selector rows remain blocked, and the
complete case stays blocked. This is **not exact generation-30 qualification**.
The original generation-30 F8-01 closure remains the immutable follow-up 9 run,
and the offline D-126 test replays that original generation-30 sequence.

Please reconcile the generation-producing history for a deterministic exact-30
reproduction. The reason this
unchanged full sequence produced one fewer generation is not attributed by the
capture. No passing subset or recovery result closes the full case, and no
second run is used to replace the two new failures elsewhere in this run.
