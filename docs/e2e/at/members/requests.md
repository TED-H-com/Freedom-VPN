# Members requests and external blockers

Current full-case status and evidence: [summary.md](summary.md). Historical
run `69f92f49` and its requests are immutable in [follow-up 8](history/follow-up-8/requests.md),
with [32-file checksum index](history/follow-up-8/SHA256SUMS) (D-098).

## Follow-up 9 — 2026-10-31 (D-102)

Baseline `7e044c71` includes agent-core 25 (#312), devstack 18 (#313),
dns-publish 8 (#309), and netlab 18. The original full mutation sequence
adopts `RestartNodeNamespace90`; no standalone scenario pass changes AT09.
Current F8-01/F8-02 verdicts remain failed pending this lane's full live rerun.

### D-119 — finite expiry and billing replacement

Resolved by orchestrator, 2026-10-31: retire the inherited “3/0 → 2/1 deadline
transition” wording. Plan v1.0 has no scheduled plan-change contract. A finite
grant's deadline means expiry (0/3 when no other grant covers the member).
A period-boundary lower plan is a billing event that calls `ReplaceFiniteGrant`
then, immediately revoking and creating the replacement. AT02 retains natural
expiry → 0/3 and immediate replacement → 3/0 → 2/1 → 3/0. No new production
schedule, fallback grant or overlapping different-plan grant is fabricated.

### F8-01 — original full mutation sequence restart

Owners: **w2-agent-core / w3-devstack-e2e**, with **w3-wire-nodes** for lease
readback. [Agent-core 25](../../../node/2026-10-31-d104-d106-agent.md) keeps
recovery fail-closed and renews against valid desired generation while an Apply
error is retained. The [devstack 18 evidence index](../../runs/2026-10-31/followup18/README.md)
reports unchanged generation 3 recovery in 5.96 s with the initial
`TC_FOREIGN_FILTER` retained. It explicitly does not close the original
generation-30 failure. This lane records the complete 90-second window,
boot/final nft/tc, heartbeats, observations, agent log, generation equality,
first fresh serving receipt and first counted byte without repair.

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
Neither independent result closes the original native row.

**D-123 (2026-10-31): IPv6Block is the v1.0 overlay mode.** The rendered
`ctv6` drops in both directions prove the netlab IPv6 no-leak control; no
end-to-end IPv6 MTU/PMTU row exists or is claimed. Native family selectors
control underlay availability; all bulk payloads here use the IPv4 origin.
Keep F8-02 unattributed between NetBird reconnect behavior and fixture echo/read
behavior until the same row has SYN/MSS, ICMP errors, interface MTUs and
application read counts sufficient to decide. Do not call it an IPv6 problem.

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

The members8 claim test already reads immutable `docs/handover/fixtures/`
snapshots, so its claims remain true without editing outside this lane.
Handover may adopt the now-published follow-up-8 archive and SHA256SUMS in its
own lane; no outside pin change is required for this rerun.
