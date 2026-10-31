# Follow-up 16 fixture evidence

Each directory contains one fresh-stack run's immutable, redacted JSON records
and Markdown renderings. Records carry their source commit and run UUID. The
campaign uses the owner-assigned `e2e` / `18000` slot and the two-resolver input.
It does not run ENR-06 installer wrappers. Fixture results do not pass a whole
AT or a platform which was not executed.

## Final assembled run

Run `a81785fa-77eb-428a-b846-fb33196a2dcf`, source
`ec84d71f45ce3622b11a8a7e55f67c9c1c432d29`: **16 pass / 0 blocked / 0 fail**,
286.77 seconds. These are fixture/scenario results, not whole-AT conclusions.

| Fixture | Deciding observation from this run | Export |
|---|---|---|
| Finite grant transition | Three real native peers; production revoke/admin grant plus worker projection changes authorized/pending from 3/0 to 2/1, then restores 3/0. Revisions 1→3→5; both same-key replays verified; expired input rejected before revocation. | [finite-grant-downgrade](a81785fa-77eb-428a-b846-fb33196a2dcf/finite-grant-downgrade.md) |
| Native IPv4/IPv6/dual | Node reachable on both underlays. All three member transfers return 64,064 bytes. Native status chooses direct IPv4 for IPv4, direct IPv6 for IPv6 and dual; candidate families agree before and after transfer. | [native-underlay](a81785fa-77eb-428a-b846-fb33196a2dcf/native-underlay.md) |
| MTU/protocol/platform inputs | 64,064 bytes complete at MTU 1500 (2 ms) and 1280 (1 ms), with both device counters increasing. QUIC and non-Linux inputs report unavailable; these are declaration checks, not execution passes. | [native-matrix-inputs](a81785fa-77eb-428a-b846-fb33196a2dcf/native-matrix-inputs.md) |
| Native two-resolver configuration | Native status lists exactly the two issued resolvers, the system uses the native stub, and both independent nodes report the same applied document hash. | [native-dns-config](a81785fa-77eb-428a-b846-fb33196a2dcf/native-dns-config.md) |
| Native fallback | Primary unavailable: secondary answers in 2,502 ms and returns the block policy in 2,504 ms. Both unavailable: lookup fails at 12,000 ms. Recovery observation takes 44 ms. | [native-dns-fallback](a81785fa-77eb-428a-b846-fb33196a2dcf/native-dns-fallback.md) |
| DNS RR negatives | All 28 TCP/UDP cases meet their recorded expected RCODE/answer sets. Blocked direct CNAME/SVCB return empty NODATA; other blocked query types return NXDOMAIN. Existing authority records provide positive controls. | [dns-negative](a81785fa-77eb-428a-b846-fb33196a2dcf/dns-negative.md) |
| DNS caches and rebinding | After authority changes, old positive answer remains with TTL 9 and negative answer with SOA TTL 3; both change after expiry. Rebinding returns private A/AAAA. Private listeners work from the node, while member connections fail (IPv4 lookup/connect bounded at 12,010 ms; IPv6 connect unavailable). | [dns-negative](a81785fa-77eb-428a-b846-fb33196a2dcf/dns-negative.md) |
| Retained enrolling population | A real offline agent attempt persists the pending identity. Central state is enrolling/pending with no bound incarnation when the directory snapshot is stopped. | [fleet-retained](a81785fa-77eb-428a-b846-fb33196a2dcf/fleet-retained.md) |
| Mid-run third-node boot | Same key and pending incarnation are bound centrally after boot. First ready observation: `2026-10-31T06:51:06.144071494Z`. | [fleet-midrun-boot](a81785fa-77eb-428a-b846-fb33196a2dcf/fleet-midrun-boot.md) |

Third target: node `01a190b5-9e08-7e47-86bf-62b05bc269bb`, incarnation
`01a190b5-a243-75d3-8862-07cc45fac4c8`; retained at `2026-10-31T06:46:38.334635246Z`,
booted at `2026-10-31T06:50:30.217170485Z`. Eligibility is explicitly the first
admin API ready observation; fleet membership/update assignment are separate.

The [signed baseline](a81785fa-77eb-428a-b846-fb33196a2dcf/fixture-baseline.md)
exports manifest/signature digests and both architecture artifact digests.
The [native member admission](a81785fa-77eb-428a-b846-fb33196a2dcf/fixture-member-admission.md)
records the authenticated controlled-origin flow and device counters. All timing
claims refer only to this run; they are not latency or throughput guarantees.

## Earlier attempts

These are retained diagnostic results, not substituted for the final run.

| Run | Source | Outcome and deciding observation |
|---|---|---|
| [e609d048](e609d048-73fc-4af6-a758-7312224937bc/native-matrix-inputs.md) | `f771bdb79c0d42b7a3d47090fb52b0d79489d82b` | 12 pass / 3 blocked / 3 fail, 266.92 seconds. All family reachability/transfer cases and finite downgrade/restoration passed. The MTU-1280 transfer stopped at 10,010 of 64,064 bytes after 12,005 ms. Native DNS had not persisted its enable flag; the original snapshot was already centrally registered. Those two fixture setup mistakes were subsequently corrected. |
| [089c9592](089c9592-3d5b-4cf4-b0aa-300a31431569/admin-login.md) | `dd6307bd8a9d2f4fb78804ae16009ec804b9a45f` | Incomplete setup, 4.24 seconds; only admin login recorded. The campaign started before the node entrypoint was running. The owned partial bridge was removed, and the hook now waits for the entrypoint's pre-enrollment stage. This is not a completed one-pass campaign. |
| [0e34514e](0e34514e-bad1-4fe6-89fa-ff55da5fee12/native-dns-fallback.md) | `59903b57c41115f02ccfb0b2bc1d56dc095b5a39` | 12 pass / 5 blocked / 1 fail, 262.06 seconds. Native fallback answered in 2,503 ms; both-down resolution failed at 12,001 ms. The initial one-shot member flow ran before ICE converged, blocking family/MTU probes. DNS negatives exposed the authority's long SOA TTL, misplaced positive listener setup, and direct CNAME/SVCB NODATA semantics. The finite transition passed. This run's old registered/applying snapshot resumed the same identity, but did not prove a central enrolling snapshot. |

The final implementation bounds initial counted-flow readiness, starts private
positive controls independently of underlay success, uses a five-second SOA RR
TTL and negative minimum, and records empty NODATA explicitly for direct blocked
CNAME/SVCB. The third snapshot now retains a real failed offline enrollment
attempt before token consumption, leaving central state enrolling/pending.

## Reproduction and limits

Use the exact [shared fixture invocation](../../../README.md#follow-up-16-shared-fixtures)
and the owner's [port/prefix allocation](../../../RUN.md). Inspect the scenario
JSON for expected versus actual fields and timings. No query history, private
keys, enrollment tokens or credential-bearing HTTP bodies are exported.

Real QUIC tooling, non-Linux platforms, physical second NAT/roaming, ICMP black
holes, IPv6 overlay leak prevention, real arm64 runtime and full AT24/AT26 fleet
campaign assertions remain outside these fixture results. QUIC and non-Linux
selectors explicitly report unavailable. The signed baseline includes amd64
and arm64 artifacts, but this native runtime is Linux amd64.

Validation: e2e lint (both vet/staticcheck tag variants), ordinary and e2e-tagged
unit suites, committed-evidence hygiene, root `make go-lint`, linux/amd64 and
linux/arm64 builds, shell syntax and `git diff --check` passed. Final exact-prefix
teardown inventory: **0 containers / 0 networks / 0 volumes / 0 images**.
