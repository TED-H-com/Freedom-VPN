# Follow-up 12 external blockers

This page retains older owner context below; current Follow-up 12 hand-offs are in the dated section of [requests.md](requests.md). F11-02 closes on the fresh populated-grant measurement. F11-03 stays blocked on no failed Apply observed. F11-01/F8-02 await devstack 22 and were not rerun. F12-01–F12-04 retain the policy-missing, reader-capacity, negative-cache and native SERVFAIL findings.

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
([immutable capture](history/follow-up-9/AT09-restart.json)). F9-01 is closed
under D-126 / D-128. **D-129 closes F10-04:** follow-up 10's unchanged sequence
qualified at generation 29 (serving 15.751594344 s, first counted byte
21.357052110 s, three provisional reports). No mutation forces generation 30.
The complete AT09 case still requires every platform/QUIC selector.

F8-02's historical native failure remains failed. [Devstack 20's instrumented
capture](../../runs/2026-10-31/followup20/capture-cce1a688/README.md) supplies
10,010/64,064 bytes in 12,011 ms at MTU 1280 on attempt 1 of 3; 19 reverse
outer IPv4 datagrams of length 1,340 are absent at the member, while all 17 of
length 1,132 arrive; no PTB. Netlab 21 has since reproduced the receive-link MTU mechanism on both underlays; devstack 22 owns the next native captures. Members bulk
rows remain as measured; a later pass is no fix proof.

F10-01 is closed by netlab 20 #336's immutable source pin. Devstack 20 #343
answers the F10-02 reader-diagnostic and F10-03 query-trace requests. The
historical failed sample and premature positive answer remain failed and
unattributed. Fresh measurements and any new dated request appear in
[summary.md](summary.md) and [requests.md](requests.md). Identity-freeze
behavior is not independently recertified by this lane.

[Follow-up 7 final evidence](history/follow-up-7/RUN.md) is preserved byte-identically for D-098. [Follow-up 8 diagnostic attempt 1](history/follow-up-8/attempt-1/RUN.md) preserves all initial failures and their harness corrections.

F4-08 / F4-09: never defined; retired 2026-10-31. The follow-up 4 ledger is F4-01 … F4-07 and F4-10.

D-098: [follow-up 8](history/follow-up-8/RUN.md) run `69f92f49` is archived byte-identically with [32-file checksums](history/follow-up-8/SHA256SUMS). Handover already asserts its own immutable followup16 snapshots; no external pin is changed.

D-098: [follow-up 9](history/follow-up-9/RUN.md) run `11e305fe` has all 33 root
files preserved byte-identically with [SHA256SUMS](history/follow-up-9/SHA256SUMS).
All 13 follow-up 8 handover snapshots still match. No outside-owned file changed.

D-098: [follow-up 10](history/follow-up-10/RUN.md), run `2be2d32f`, preserves
all 35 root files byte-identically with [SHA256SUMS](history/follow-up-10/SHA256SUMS).
Follow-up 8/9 archives and handover snapshots remain unchanged. The interrupted
[follow-up 11 attempt](history/follow-up-11/attempt-1/RUN.md) is separate.

Historical follow-up 11: AT02 fails its D-077 readiness and admitted-sibling controls
(F11-02/F11-03); AT01 retains the new initial missing-hop blocker (F11-01).
AT09 recovers at generation 30 but fails the dual/1280 row at 13,013/64,064 bytes
in 12,010 ms over a selected IPv6 underlay. Its five unavailable selectors are
unchanged. That run had nine pending request groups: six carried groups plus the
three new dated groups. Passing finite-expiry reads and before-TTL DNS do not
pass either failed complete case.
