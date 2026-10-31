# Follow-up 10 external blockers

The full-case outcomes and failed checks are in [summary.md](summary.md). Fixture availability closes only the named missing fixture; it does not certify a consuming case or a physical platform. [requests.md](requests.md) retains owner actions.

| Request | Landed change / fresh recheck | Remaining boundary and owner |
|---|---|---|
| F4-03 | #266 exports member persistent TCP/UDP and forced relay; #284 exports native IPv4/IPv6/dual and MTU 1500/1280. AT01/AT04/AT09 consume them. | QUIC and Android/iOS/Windows/macOS explicitly unavailable; physical second NAT, roaming and ICMP-black-hole PMTU remain. Devstack and platform/operator campaign. |
| F4-04 | #266 closes missing member UDP fixture; AT04 measures refund, real paid expiry and suspension on both exits, including persistent senders. | Rendered suspended/expired member screens remain portal/devstack work. Fresh API state is not rendered UI evidence. |
| F4-05 | #284 supplies exact issued two-resolver list, enabled native DNS, OS stub, primary-down fallback, both-down failure and recovery. AT20/AT21 record every result. | Nameserver distribution is TEST ONLY. Production automatic distribution and Private DNS/browser DoH platform coverage remain devstack/native integration/operator work. |
| F4-06 | AT22 rechecks native sentinel privacy, both resolver configurations, logs, metrics, audit/dump surfaces, guarded audit retention and real scheduled feed last-good. #266/#284 probes export metadata only. | Representative raw-usage/rollup/file/backup retention and URL privacy inputs remain devstack/usage/database/backup work. Commercial feed approval remains a business decision. |
| F4-07 | #284 supplies the RR, TTL/cache authority phases and private A/AAAA rebinding controls; AT20 consumes them through the native stub. | DNS wording retired 2026-10-31: [backend §2.1.1–§2.1.4](../../../dns/BACKEND.md) confirms per-type RCODE, SOA and TTL. Positive IDNA wire fixture remains absent; owner devstack. |
| F4-10 | #279 installer startup/seed lifecycle and #282 enrollment evidence are on the baseline; AT05 repeats reservation, replay and actual installer enrollment. | Preinstalled/cloud-init boot provenance and physical arm64 remain packaging/devstack/operator work. Cross-build is separate. |

The separate worker RestartPolicy note is closed by #240 and fresh AT05 readback. D-050/D-081 now use exported ReplaceFiniteGrant (#284); the duplicate local production-service adapter is removed. Immediate 3/0→2/1 replacement and grant-only deadline expiry to 0/3 are separate measured transitions. D-119 (2026-10-31): retire the inherited “3/0 → 2/1 deadline transition” wording. Plan v1.0 has no scheduled plan-change contract: natural expiry yields 0/3 with no other grant; billing issues an immediate replacement through `ReplaceFiniteGrant` at the period boundary, producing 3/0 → 2/1 → 3/0 in the harness.

F8-01 remains closed by follow-up 9's original generation-30 evidence
([immutable capture](history/follow-up-9/AT09-restart.json)). F9-01 closes under
D-126 / D-128: agent-core 27 confirms deliberate composition, wire-nodes 32
proves JSON-value preservation, and this lane replays the exact old sequence.
The fresh run accepts the same mixed shape and recovers at unchanged **29**,
so **F10-04 keeps exact generation-30 qualification open**. The full AT09 case
remains blocked with 38 pass / 5 blocked / 0 fail checks. All four bulk rows
pass; dual/1280 completes 64,064 bytes in 1 ms, with direct IPv4 selected.
F8-02 stays unattributed per [netlab 19](../../../poc/G2-f802-native-attribution-2026-10-31.md);
devstack 19 owns the exact-sequence decode/capture. A later pass is no fix proof.

The fresh run also retains **F10-02 (AT02 unreadable withdrawal sample)** and
**F10-03 (AT20 positive cache changed before TTL)** as failures. F10-01 requests
the outside-owned netlab pin's immutable archive path. All four new items,
exact captures and owners are in [requests.md](requests.md). Identity-freeze
behavior is not independently recertified by this lane.

[Follow-up 7 final evidence](history/follow-up-7/RUN.md) is preserved byte-identically for D-098. [Follow-up 8 diagnostic attempt 1](history/follow-up-8/attempt-1/RUN.md) preserves all initial failures and their harness corrections.

F4-08 / F4-09: never defined; retired 2026-10-31. The follow-up 4 ledger is F4-01 … F4-07 and F4-10.

D-098: [follow-up 8](history/follow-up-8/RUN.md) run `69f92f49` is archived byte-identically with [32-file checksums](history/follow-up-8/SHA256SUMS). Handover already asserts its own immutable followup16 snapshots; no external pin is changed.

D-098: [follow-up 9](history/follow-up-9/RUN.md) run `11e305fe` has all 33 root
files preserved byte-identically with [SHA256SUMS](history/follow-up-9/SHA256SUMS).
All 13 follow-up 8 handover snapshots still match. No outside-owned file changed.
