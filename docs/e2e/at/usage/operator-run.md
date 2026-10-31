# Run usage acceptance

```bash
make -f mk/at-usage.mk at-usage-lint at-usage-test at-usage-build
bash tests/e2e/at/usage/stack.sh up
: "${FVPN_TEST_DATABASE_URL:?set the assigned lab PostgreSQL admin DSN first}"
FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/at-usage.mk at-usage
bash tests/e2e/at/usage/stack.sh down
```

For a complete run, set `FVPN_TEST_DATABASE_URL` to the assigned lab PostgreSQL admin DSN (loopback port `53432`, user/database `fvpn`, password from `.agent/e2e-state-w4-usage/secrets/postgres.password`, `sslmode=disable`). Keep the password out of command output and evidence. The live run above supplied that DSN; all central mechanism fixtures ran without skips.

The lane wrapper calls the shared driver and scopes image tags to the owned prefix. `e2e.PrepareServingBaseline` builds/signs/registers real amd64 and arm64 releases. Member group, plan `exit_group_ids`, publication `group_ids`, grants and native admission all use product APIs. A real serving heartbeat and counted forward flow precede acceptance.

PostgreSQL access to the live stack is read-only. The existing mechanism fixtures use `internal/db/dbtest` to create/drop isolated databases on the assigned lab PostgreSQL when `FVPN_TEST_DATABASE_URL` is configured; their SQL fixture setup is unchanged from Follow-up 3. Audited usage adjustment and ledger recovery use `fvpn-usage-admin`. AT15 deliberately submits signed mTLS batch fixtures after pausing the agent. D-067 retains the prior mTLS applying-observation setup; its provisional/serving observations are real agent reports.

Follow-up 3's synthetic ACK workaround is removed: Fair-Use convergence now waits for actual agent apply and kernel tc read-back. Counted traffic is sampled/uploaded by the agent, including both incarnations. AT14 uses the shared ingress injector plus a temporary node-netns private-destination /32 route over fvup0 (removed immediately) to exercise ctdeny; no default route or foreign nft table is changed. The same component checks remain; deterministic month-boundary/clock fixtures do not change the host clock.

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

## Which serving fix engaged

no foreign rules; Docker DNS NAT absent; #240 lab cleanup made this node serve; D-079 present but not exercised. The first real serving observation `01a1902b-971e-7ce4-a46c-2bc3cc0ab272` (received `2026-10-31T04:15:38.275502Z`) is preserved in each case's `first_serving_apply.observation`, alongside `serving 1`, the admitted-device counter deltas and the ruleset read-back. absent: ObservationReport has no foreign-rule classification field on this tree; classification above is enforce.LoopbackEvidence over the live read-back.

```nft
table inet fvpn {
	counter ctbase {
		packets 0 bytes 0
	}

	counter ctv6 {
		packets 0 bytes 0
	}

	counter ctin {
		packets 0 bytes 0
	}

	counter ctbadns {
		packets 0 bytes 0
	}

	counter ctdeny {
		packets 0 bytes 0
	}

	counter cthost {
		packets 0 bytes 0
	}

	counter ctinv {
		packets 0 bytes 0
	}

	counter ctnodn {
		packets 0 bytes 0
	}

	counter ctunkd {
		packets 0 bytes 0
	}

	counter ctunku {
		packets 0 bytes 0
	}

	counter d_01a1902b274071d985262f83f7a4ee27 {
		packets 8 bytes 1515
	}

	counter d_01a1902b62647c7c9a6f8d2fdd1c612f {
		packets 0 bytes 0
	}

	counter nd_01a1902b274071d985262f83f7a4ee27 {
		packets 0 bytes 0
	}

	counter nd_01a1902b62647c7c9a6f8d2fdd1c612f {
		packets 0 bytes 0
	}

	counter nu_01a1902b274071d985262f83f7a4ee27 {
		packets 0 bytes 0
	}

	counter nu_01a1902b62647c7c9a6f8d2fdd1c612f {
		packets 0 bytes 0
	}

	counter u_01a1902b274071d985262f83f7a4ee27 {
		packets 8 bytes 1066
	}

	counter u_01a1902b62647c7c9a6f8d2fdd1c612f {
		packets 0 bytes 0
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.75.87.195 timeout 23h59m10s expires 23h59m8s167ms, 100.75.122.40 timeout 23h59m12s expires 23h59m10s167ms }
	}

	set denied_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 0.0.0.0/8, 10.0.0.0/8,
			     100.64.0.0/10, 127.0.0.0/8,
			     169.254.0.0/16, 192.0.2.0/12,
			     192.0.0.0/24, 192.0.2.0/24,
			     192.88.99.0/24, 192.168.0.0/16,
			     198.18.0.0/15, 198.51.100.0/24,
			     203.0.113.0/24, 224.0.0.0/3 }
	}

	set host_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 198.51.100.20 }
	}

	set resolver_v4 {
		type ipv4_addr
		elements = { 192.0.2.40 }
	}

	chain forward {
		type filter hook forward priority mangle; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=6 fvpn_metering=forward_admission_v1"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1902a-6e6e-7385-9a68-1c16632ec598:2363"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1902a-74b9-769c-8733-dd5e439e8af4:a745"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.75.87.195 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1902b274071d985262f83f7a4ee27" meta priority set 1:2363 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.75.87.195 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1902b274071d985262f83f7a4ee27" meta priority set 1:2363 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.75.122.40 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1902b62647c7c9a6f8d2fdd1c612f" meta priority set 1:a745 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.75.122.40 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1902b62647c7c9a6f8d2fdd1c612f" meta priority set 1:a745 accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.75.87.195 counter name "u_01a1902b274071d985262f83f7a4ee27" meta priority set 1:2363 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.75.122.40 counter name "u_01a1902b62647c7c9a6f8d2fdd1c612f" meta priority set 1:a745 accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.75.87.195 tcp sport 53 counter name "nd_01a1902b274071d985262f83f7a4ee27" meta priority set 1:2363 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.75.87.195 udp sport 53 counter name "nd_01a1902b274071d985262f83f7a4ee27" meta priority set 1:2363 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.75.87.195 counter name "d_01a1902b274071d985262f83f7a4ee27" meta priority set 1:2363 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.75.122.40 tcp sport 53 counter name "nd_01a1902b62647c7c9a6f8d2fdd1c612f" meta priority set 1:a745 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.75.122.40 udp sport 53 counter name "nd_01a1902b62647c7c9a6f8d2fdd1c612f" meta priority set 1:a745 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.75.122.40 counter name "d_01a1902b62647c7c9a6f8d2fdd1c612f" meta priority set 1:a745 accept
	}

	chain input {
		type filter hook input priority mangle; policy accept;
		iifname "fvmb0" counter name "ctin" drop
		ip saddr 100.64.0.0/10 counter name "ctin" drop
	}

	chain output {
		type filter hook output priority mangle; policy accept;
	}

	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;
		oifname "fvup0" ip saddr @allowed_v4 snat ip to 198.51.100.20
	}
}
```

## Post-run verification

Live run `351d59aa` passed all six cases (`TestATUsage_Live` 461.12 s; package 467.921 s). Area lint/test/build, root `make go-lint`, `go vet ./...`, formatting, whitespace and committed-evidence hygiene checks passed. Linux amd64 and arm64 test binaries cross-compiled; no physical arm64 run is claimed.

The shared driver tore the stack down. Final `fvpn-w4-usage-*` inventory: **0 containers, 0 networks, 0 volumes, 0 image tags**. No push or PR.
