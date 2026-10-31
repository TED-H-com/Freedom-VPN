# Usage acceptance — Follow-up 4

Run `351d59aa`; harness `463ec28805449e4cd71e66589e2ecac51d21b241` (dirty `false`); image build `3d2576dbf20617169723323c526f3700868406e3`; origin/main product source `e8a5e046d3617e1f914a02629ecad940ee7fc1ef`. Stack `fvpn-w4-usage-*`, port base `48000`.

| Case | Status | Evidence | Live result |
|---|---|---|---|
| AT14 | pass | [AT14.json](AT14.json) / [AT14.md](AT14.md) | Live serving exit 01a1902a-a422-7ac1-9163-a0eaac36d1af (incarnation 01a1902a-a4e5-7069-8c48-199ea651549a, exit 01a1902a-a422-7ac2-8236-e3615867fe5f) metered admitted HTTPS traffic for memberA device 01a1902b-2740-71d9-8526-2f83f7a4ee27 and uploaded 3 UsageEvents to PostgreSQL with GET /api/v1/member/usage reporting exact total_bytes=23260. |
| AT15 | pass | [AT15.json](AT15.json) / [AT15.md](AT15.md) | Live mTLS /api/v1/agent/usage-checkpoint and /api/v1/agent/usage-batches on serving node 01a1902a-a422-7ac1-9163-a0eaac36d1af (incarnation 01a1902a-a4e5-7069-8c48-199ea651549a, initial real agent seq=5) verified Ed25519 recovery signature, out-of-order seq {6,8} -> gap fill seq 7 advancing highest_contiguous_seq to 8, batch replay and duplicate retry with zero double-counted bytes, payload_sha256 conflict audit (usage_event_conflicts=1), rejected seq 10 (>2^53 decimal string 9007199254740993) reporting blocked_seq=10 then advancing to 11, 409/403 error guards, and live fvpn-usage-admin freeze/begin-recovery/complete-recovery rotating ledger_epoch 313800c1-c690-425c-a7d6-d87606ebb62a -> 01a19031-4298-7aa0-9813-9eb352e475ac with replay_unavailable reconciliation to seq 12. |
| AT16 | pass | [AT16.json](AT16.json) / [AT16.md](AT16.md) | Live node fvpn-w4-usage-node verified sampler restart continuity (event_seq 1->2), D-067 provisional-lease recovery across bring_up (5848ms/6837ms) and agent_restart (5819ms/6135ms), and live fail-closed withdrawal via fvpn-watchdog. |
| AT17 | pass | [AT17.json](AT17.json) / [AT17.md](AT17.md) | Live GET /api/v1/member/usage and ?period_id=2026-10&granularity=day returned real counted memberA traffic totals (upload=12872, download=18168, total=31040) within half-open UTC month bounds [2026-10-01T00:00:00Z, 2026-11-01T00:00:00Z), while central and node mechanisms verified 00:00:00Z UTC month-boundary splitting, late-arriving previous-month event attribution within the 24h watermark, and >5s wall-clock rollback protection. |
| AT18 | pass | [AT18.json](AT18.json) / [AT18.md](AT18.md) | Live serving exit fvpn-w4-usage-node installed kernel tc HTB + fq_codel on fvmb0 and fvup0 with host reserve class 1:fffe (100Mbit) and per-member classes at effective_rate = min(plan=80M/40M, tier0=100M/50M, exit=60M/30M) = 60Mbit down / 30Mbit up, shaped concurrent memberA (+1106/+1514 B) and memberB (+1106/+1515 B) flows, and preserved mTLS control-plane responsiveness (HTTP 200). |
| AT19 | pass | [AT19.json](AT19.json) / [AT19.md](AT19.md) | Live multi-node test enrolled second exit 01a1902c-df9f-7494-bfd3-5c4f56f6f07e (node2) and non-carrying node 01a1902e-3aa3-7393-8619-91be8b750813, routed memberA traffic across both carrying exits (aggregating 28459 bytes across 2 exits), verified shadow mode (60Mbit on node, 15Mbit on node2), proved Transition 1 (policy.activated mode=enforce: carrying gens node 8->10, node2 4->5, non-carrying 0->0; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in 7032 ms via observation 01a1902e-934f-7e82-a9fd-0a3bc7dd6693; replay no double bump) and Transition 2 (fup.member_generation_changed via usage threshold crossing: carrying gens node 11->12, node2 6->7, non-carrying 0->0; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in 29864 ms via observation 01a1902f-8382-7102-898d-c247a2cb5b37; replay no double bump), and verified ENR-03 per-incarnation retained usage across node2 re-enrollment (01a1902c-e118-7aea-a535-b182bd653cb9 -> 01a1902f-c18b-7863-b1e9-4e63bbad5a9e). |

## D-067 comparison

The previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 20 was not in the image source. Devstack #240 measured **80622 ms** on its separate path.

| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |
|---|---|---|---|---|---|---|
| bring_up | 5021 ms | 5021 ms | 5848 ms | 6837 ms | `01a1902b-8337-7ca0-84e9-c6f100fd21e1` at `2026-10-31T04:15:33.180264Z` | `bdec5e45-20fd-446a-bbe0-9b44736baba0` at `2026-10-31T04:15:38.201795Z` |
| agent_restart | 5025 ms | 5022 ms | 5819 ms | 6135 ms | `01a1902b-bb9a-70fe-bc4d-2127f9d72170` at `2026-10-31T04:15:47.614206Z` | `84f3c278-81d8-4452-a768-b8062b2e3b4b` at `2026-10-31T04:15:52.636805Z` |

Both provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).

## Fair-Use convergence

| Trigger | Clearing observation | Trigger → applying false | Tier 2 installed on both exits |
|---|---|---|---|
| policy_activated_enforce_tier2 | `01a1902e-934f-7e82-a9fd-0a3bc7dd6693` | 7032 ms | true |
| fup_member_generation_changed_tier0_to_tier2 | `01a1902f-8382-7102-898d-c247a2cb5b37` | 29864 ms | true |

The usage-threshold timer starts with real traffic and includes the normal sample/upload cadence; the activation timer starts with policy activation. Follow-up 3 used synthetic batch/ACK fixtures (5387 / 6151 ms), so those timings are not identical paths.

## ENR-03 and UTC boundary

Node `01a1902c-df9f-7494-bfd3-5c4f56f6f07e`: retired incarnation `01a1902c-e118-7aea-a535-b182bd653cb9`, new incarnation `01a1902f-c18b-7863-b1e9-4e63bbad5a9e`; old usage fixed `true`; member total unchanged at reinstall `true`; new traffic accumulated `true`. Member totals: `28459` → `28459` → `31040` bytes. Detailed per-incarnation bytes and timestamps are in [AT19.json](AT19.json). [AT17.json](AT17.json) re-runs the UTC split (1001/2003 bytes → October 500/1001 and November 501/1002), late events and clock rollback.

Prior runs: [Follow-up 3](history/followup3/summary.md), [Follow-up 2](history/followup2/summary.md), [initial](history/initial/).

## Post-run verification

Live run `351d59aa` passed all six cases (`TestATUsage_Live` 461.12 s; package 467.921 s). Area lint/test/build, root `make go-lint`, `go vet ./...`, formatting, whitespace and committed-evidence hygiene checks passed. Linux amd64 and arm64 test binaries cross-compiled; no physical arm64 run is claimed.

The shared driver tore the stack down. Final `fvpn-w4-usage-*` inventory: **0 containers, 0 networks, 0 volumes, 0 image tags**. No push or PR.
