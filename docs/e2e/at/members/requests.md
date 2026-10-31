# Members requests and external blockers

Current full-case status and evidence: [summary.md](summary.md). Historical
run `69f92f49` and its requests are immutable in [follow-up 8](history/follow-up-8/requests.md),
with [32-file checksum index](history/follow-up-8/SHA256SUMS) (D-098).

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

**Fresh result:** desired/applied generation **30 → 30**; namespace restart
`2026-10-31T12:06:31.816656225Z`; first fresh serving receipt
`12:06:47.545258Z` (**15.728602 s**); first counted byte
`12:06:53.190469871Z` (**21.373814 s**). A new generation-30 lease at
`12:06:47.408641Z` has seven member grants. The capture retains **88 heartbeat
samples over 90.337054 s**, boot/final nft/tc, every central observation and the
agent log, including the initial `TC_FOREIGN_FILTER`. No generation, topology
or state repair occurred. [Complete capture](AT09-restart.json).
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

**Fresh result:** dual underlay / MTU 1280 completes **64,064 / 64,064 bytes in 1 ms**, with direct IPv4 selected before and after; device counters increase **67,652 upload / 68,139 download bytes**. All three preceding 1500 rows also complete. [NATIVE.md](NATIVE.md) and [AT09.json](AT09.json) retain every family/path and counter. This is a passing row in the full mutation sequence, not proof of a root-cause fix or an overall AT09 pass.

**D-123 (2026-10-31): IPv6Block is the v1.0 overlay mode.** The rendered
`ctv6` drops in both directions prove the netlab IPv6 no-leak control; no
end-to-end IPv6 MTU/PMTU row exists or is claimed. Native family selectors
control underlay availability; all bulk payloads here use the IPv4 origin.
Keep F8-02 unattributed between NetBird reconnect behavior and fixture echo/read
behavior until the same row has SYN/MSS, ICMP errors, interface MTUs and
application read counts sufficient to decide. Do not call it an IPv6 problem.

### F9-01 — mixed provisional lease and tc-rejection observation, 2026-10-31

Owners: **w2-agent-core / w3-wire-nodes**, with orchestrator contract review.
The complete restart capture contains three non-serving observations. At
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
