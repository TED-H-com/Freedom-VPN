# Run usage acceptance

```bash
make -f mk/at-usage.mk at-usage-lint at-usage-test at-usage-build
bash tests/e2e/at/usage/stack.sh up
FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/at-usage.mk at-usage
bash tests/e2e/at/usage/stack.sh down
```

The lane wrapper calls the shared driver and scopes image tags to the owned prefix. `e2e.PrepareServingBaseline` builds/signs/registers real amd64 and arm64 releases. Member group, plan `exit_group_ids`, publication `group_ids`, grants and native admission all use product APIs. A real serving heartbeat and counted forward flow precede acceptance.

PostgreSQL access to the live stack is read-only. The existing mechanism fixtures use `internal/db/dbtest` to create/drop isolated databases on the assigned lab PostgreSQL when `FVPN_TEST_DATABASE_URL` is configured; their SQL fixture setup is unchanged from Follow-up 3. Audited usage adjustment and ledger recovery use `fvpn-usage-admin`. AT15 deliberately submits signed mTLS batch fixtures after pausing the agent. D-067 retains the prior mTLS applying-observation setup; its provisional/serving observations are real agent reports.

Follow-up 3's synthetic ACK workaround is removed: Fair-Use convergence now waits for actual agent apply and kernel tc read-back. Counted traffic is sampled/uploaded by the agent, including both incarnations. AT14 uses the shared ingress injector plus a temporary node-netns private-destination /32 route over fvup0 (removed immediately) to exercise ctdeny; no default route or foreign nft table is changed. The same component checks remain; deterministic month-boundary/clock fixtures do not change the host clock.

# Usage acceptance — Follow-up 5

Run `170ea3e3`; harness `3a59beb47e8bc311e063c525fdae4ea17067d134` (dirty `false`); image build `3a59beb47e8bc311e063c525fdae4ea17067d134`; origin/main product source `7e044c71ecf1d07f63e89721bd213db6611a6a47`. Stack `fvpn-w4-usage-*`, port base `48000`.

| Case | Status | Evidence | Live result |
|---|---|---|---|
| AT14 | pass | [AT14.json](AT14.json) / [AT14.md](AT14.md) | Live serving exit 01a191e0-7f82-714e-bc9c-0407ab79ba7a (incarnation 01a191e0-8191-7d82-8822-0497f4dbee7c, exit 01a191e0-7f82-714f-a9fc-e3f77eb7fe43) metered admitted HTTPS traffic for memberA device 01a191e0-f90c-7809-a6b2-2f5cefdcad1d and uploaded 3 UsageEvents to PostgreSQL with GET /api/v1/member/usage reporting exact total_bytes=23425. |
| AT15 | pass | [AT15.json](AT15.json) / [AT15.md](AT15.md) | Live mTLS /api/v1/agent/usage-checkpoint and /api/v1/agent/usage-batches on serving node 01a191e0-7f82-714e-bc9c-0407ab79ba7a (incarnation 01a191e0-8191-7d82-8822-0497f4dbee7c, initial real agent seq=6) verified Ed25519 recovery signature, out-of-order seq {7,9} -> gap fill seq 8 advancing highest_contiguous_seq to 9, batch replay and duplicate retry with zero double-counted bytes, payload_sha256 conflict audit (usage_event_conflicts=1), rejected seq 11 (>2^53 decimal string 9007199254740993) reporting blocked_seq=11 then advancing to 12, 409/403 error guards, and live fvpn-usage-admin freeze/begin-recovery/complete-recovery rotating ledger_epoch 84a1f7a0-b640-4dcf-8c5a-d654c53bd71a -> 01a191e9-8ad1-74a3-a555-9702d0a82214 with replay_unavailable reconciliation to seq 13. |
| AT16 | pass | [AT16.json](AT16.json) / [AT16.md](AT16.md) | Live node fvpn-w4-usage-node verified sampler restart continuity (event_seq 1->2), D-067 provisional-lease recovery across bring_up (5801ms/7652ms) and agent_restart (5874ms/6166ms), and live fail-closed withdrawal via fvpn-watchdog. |
| AT17 | pass | [AT17.json](AT17.json) / [AT17.md](AT17.md) | Live GET /api/v1/member/usage and ?period_id=2026-10&granularity=day returned real counted memberA traffic totals (upload=12992, download=18177, total=31169) within half-open UTC month bounds [2026-10-01T00:00:00Z, 2026-11-01T00:00:00Z), while central and node mechanisms verified 00:00:00Z UTC month-boundary splitting, late-arriving previous-month event attribution within the 24h watermark, and >5s wall-clock rollback protection. |
| AT18 | pass | [AT18.json](AT18.json) / [AT18.md](AT18.md) | Live serving exit fvpn-w4-usage-node installed kernel tc HTB + fq_codel on fvmb0 and fvup0 with host reserve class 1:fffe (100Mbit) and per-member classes at effective_rate = min(plan=80M/40M, tier0=100M/50M, exit=60M/30M) = 60Mbit down / 30Mbit up, shaped concurrent memberA (+1066/+1515 B) and memberB (+1066/+1515 B) flows, and preserved mTLS control-plane responsiveness (HTTP 200). |
| AT19 | pass | [AT19.json](AT19.json) / [AT19.md](AT19.md) | Live multi-node test enrolled second exit 01a191e2-b50c-75a4-8d56-b480eea714db (node2) and non-carrying node 01a191e3-fbf0-7701-a4a8-3fe1a3d6fce4, routed memberA traffic across both carrying exits (aggregating 28587 bytes across 2 exits), verified shadow mode (60Mbit on node, 15Mbit on node2), proved Transition 1 (policy.activated mode=enforce: carrying gens node 10->11, node2 4->5, non-carrying 0->0; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in 56181 ms via observation 01a191e5-0403-79f5-874a-dcb629a09ae4; replay no double bump) and Transition 2 (fup.member_generation_changed via usage threshold crossing: carrying gens node 12->13, node2 6->7, non-carrying 0->0; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in 66466 ms via observation 01a191e6-4790-77a3-a45f-7e21b499c56f; replay no double bump), and verified ENR-03 per-incarnation retained usage across node2 re-enrollment (01a191e2-b648-7824-b7cb-b747c222f2c1 -> 01a191e6-9633-7ccc-bae2-e65610e0e145). |
| AT20 | pass | [AT20.json](AT20.json) / [AT20.md](AT20.md) | Same node 01a191e7-e2d9-7d9f-bdcd-87b79739916f reenrolled 01a191e7-e888-72d1-826b-026b025dafc5 -> 01a191e9-4151-77d6-bbc0-2406573cfe54 through ControlledReinstall. Member period 2026-10: before 33750 + held 7782 = after 41532 bytes; 1 original events retained once, 1 agent batches returned the identical stored canonical result, origin ACK 2, independent current ACK 0. Meter interval 10s <= central bound 10s. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history. |

## D-067 comparison

The previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 25 is in the image source. Devstack #240 measured **80622 ms** on its separate path.

| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |
|---|---|---|---|---|---|---|
| bring_up | 5021 ms | 4914 ms | 5801 ms | 7652 ms | `01a191e1-582e-7e13-a799-4d2750b7a18f` at `2026-10-31T12:13:46.932902Z` | `f234dd5d-df9e-4b9e-b6cf-d1f3907d1f3e` at `2026-10-31T12:13:51.847759Z` |
| agent_restart | 5025 ms | 4983 ms | 5874 ms | 6166 ms | `01a191e1-9354-7997-86df-ad948e519d75` at `2026-10-31T12:14:02.073854Z` | `09769a21-7ad9-4fd4-bd9d-b9f9f1eb36df` at `2026-10-31T12:14:07.056909Z` |

Both provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).

## Fair-Use convergence

| Trigger | Clearing observation | Trigger → applying false | Tier 2 installed on both exits |
|---|---|---|---|
| policy_activated_enforce_tier2 | `01a191e5-0403-79f5-874a-dcb629a09ae4` | 56181 ms | true |
| fup_member_generation_changed_tier0_to_tier2 | `01a191e6-4790-77a3-a45f-7e21b499c56f` | 66466 ms | true |

The usage-threshold timer starts with real traffic and includes the normal sample/upload cadence; the activation timer starts with policy activation. Follow-up 3 used synthetic batch/ACK fixtures (5387 / 6151 ms), so those timings are not identical paths.

## ENR-03 and UTC boundary

Node `01a191e2-b50c-75a4-8d56-b480eea714db`: retired incarnation `01a191e2-b648-7824-b7cb-b747c222f2c1`, new incarnation `01a191e6-9633-7ccc-bae2-e65610e0e145`; old usage fixed `true`; member total unchanged at reinstall `true`; new traffic accumulated `true`. Member totals: `28587` → `28587` → `31169` bytes. Detailed per-incarnation bytes and timestamps are in [AT19.json](AT19.json). [AT17.json](AT17.json) re-runs the UTC split (1001/2003 bytes → October 500/1001 and November 501/1002), late events and clock rollback.

Prior runs: [Follow-up 4](history/follow-up-4/ARCHIVE.md), [Follow-up 3](history/followup3/summary.md), [Follow-up 2](history/followup2/summary.md), [initial](history/initial/).

## Which serving fix engaged

no foreign rules; Docker DNS NAT absent; #240 lab cleanup made this node serve; D-079 present but not exercised. The first real serving observation `01a191e1-6bef-728e-b55c-11c3801cc3b0` (received `2026-10-31T12:13:51.990089Z`) is preserved in each case's `first_serving_apply.observation`, alongside `serving 1`, the admitted-device counter deltas and the ruleset read-back. absent: ObservationReport has no foreign-rule classification field on this tree; classification above is enforce.LoopbackEvidence over the live read-back.

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

	counter d_01a191e0f90c7809a6b22f5cefdcad1d {
		packets 8 bytes 1514
	}

	counter d_01a191e133d97c2c93f8fd22a736f77f {
		packets 0 bytes 0
	}

	counter nd_01a191e0f90c7809a6b22f5cefdcad1d {
		packets 0 bytes 0
	}

	counter nd_01a191e133d97c2c93f8fd22a736f77f {
		packets 0 bytes 0
	}

	counter nu_01a191e0f90c7809a6b22f5cefdcad1d {
		packets 0 bytes 0
	}

	counter nu_01a191e133d97c2c93f8fd22a736f77f {
		packets 0 bytes 0
	}

	counter u_01a191e0f90c7809a6b22f5cefdcad1d {
		packets 8 bytes 1066
	}

	counter u_01a191e133d97c2c93f8fd22a736f77f {
		packets 0 bytes 0
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.96.150.23 timeout 9m54s833ms expires 9m52s99ms, 100.96.210.17 timeout 9m54s833ms expires 9m52s99ms }
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
		elements = { 198.51.100.53 }
	}

	chain forward {
		type filter hook forward priority mangle; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=8 fvpn_metering=forward_admission_v1"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a191e0-5a47-777e-af02-a3161664463f:475b"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a191e0-5aba-7dd0-9eec-14e48c1a9ea1:d1d3"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.96.210.17 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a191e0f90c7809a6b22f5cefdcad1d" meta priority set 1:475b accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.96.210.17 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a191e0f90c7809a6b22f5cefdcad1d" meta priority set 1:475b accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.96.150.23 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a191e133d97c2c93f8fd22a736f77f" meta priority set 1:d1d3 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.96.150.23 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a191e133d97c2c93f8fd22a736f77f" meta priority set 1:d1d3 accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.96.210.17 counter name "u_01a191e0f90c7809a6b22f5cefdcad1d" meta priority set 1:475b accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.96.150.23 counter name "u_01a191e133d97c2c93f8fd22a736f77f" meta priority set 1:d1d3 accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.96.210.17 tcp sport 53 counter name "nd_01a191e0f90c7809a6b22f5cefdcad1d" meta priority set 1:475b accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.96.210.17 udp sport 53 counter name "nd_01a191e0f90c7809a6b22f5cefdcad1d" meta priority set 1:475b accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.96.210.17 counter name "d_01a191e0f90c7809a6b22f5cefdcad1d" meta priority set 1:475b accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.96.150.23 tcp sport 53 counter name "nd_01a191e133d97c2c93f8fd22a736f77f" meta priority set 1:d1d3 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.96.150.23 udp sport 53 counter name "nd_01a191e133d97c2c93f8fd22a736f77f" meta priority set 1:d1d3 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.96.150.23 counter name "d_01a191e133d97c2c93f8fd22a736f77f" meta priority set 1:d1d3 accept
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

### For the enrollment lane

Same node 01a191e7-e2d9-7d9f-bdcd-87b79739916f reenrolled 01a191e7-e888-72d1-826b-026b025dafc5 -> 01a191e9-4151-77d6-bbc0-2406573cfe54 through ControlledReinstall. Member period 2026-10: before 33750 + held 7782 = after 41532 bytes; 1 original events retained once, 1 agent batches returned the identical stored canonical result, origin ACK 2, independent current ACK 0. Meter interval 10s <= central bound 10s. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history. See [AT20.json](AT20.json) for the original CLI, controlled reinstall outcome, immutable payloads, central samples, stored batch replay, independent ACKs, configuration and refusal audit. This follows the devstack follow-up 18 path in [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31). The systemd installer node is quiesced in documented unit order; pause-agent applies only to ordinary supervised nodes. No ledger rows or retirement timestamps are edited.
