# Run usage acceptance

```bash
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
# Before stack startup and the live suite, wait while two other suites are active.
pgrep -af 'gates.sh|mk/at-' || true
make -f mk/at-usage.mk at-usage-lint at-usage-test at-usage-build
bash tests/e2e/at/usage/stack.sh up
FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/at-usage.mk at-usage
bash tests/e2e/at/usage/stack.sh down
```

The lane wrapper calls the shared driver and scopes image tags to the owned prefix. `e2e.PrepareServingBaseline` builds/signs/registers real amd64 and arm64 releases. Member group, plan `exit_group_ids`, publication `group_ids`, grants and native admission all use product APIs. A real serving heartbeat and counted forward flow precede acceptance.

PostgreSQL access to the live stack is read-only. The existing mechanism fixtures use `internal/db/dbtest` to create/drop isolated databases on the assigned lab PostgreSQL when `FVPN_TEST_DATABASE_URL` is configured; their SQL fixture setup is unchanged from Follow-up 3. Audited usage adjustment and ledger recovery use `fvpn-usage-admin`. AT15 deliberately submits signed mTLS batch fixtures after pausing the agent. D-067 retains the prior mTLS applying-observation setup; its provisional/serving observations are real agent reports.

Follow-up 3's synthetic ACK workaround is removed: Fair-Use convergence now waits for actual agent apply and kernel tc read-back. Counted traffic is sampled/uploaded by the agent, including both incarnations. AT14 uses the shared ingress injector plus a temporary node-netns private-destination /32 route over fvup0 (removed immediately) to exercise ctdeny; no default route or foreign nft table is changed. The same component checks remain; deterministic month-boundary/clock fixtures do not change the host clock.

# Usage acceptance — Follow-up 6

Run `71b1a160`; harness `7ccaabade8efc5074cfde5dcae84c6b4a27fd8c0` (dirty `false`); image build `7ccaabade8efc5074cfde5dcae84c6b4a27fd8c0`; origin/main product source `fb5e002549f79e8ef61bebcf42cce1678ab65cb1`. Stack `fvpn-w4-usage-*`, port base `48000`.

| Case | Status | Evidence | Live result |
|---|---|---|---|
| AT14 | pass | [AT14.json](AT14.json) / [AT14.md](AT14.md) | Live serving exit 01a192ae-1201-70bf-b380-a6dfdbc2724c (incarnation 01a192ae-1361-7d54-a603-22b10f2ac4a6, exit 01a192ae-1201-70c0-9ca3-8470b302d19d) metered admitted HTTPS traffic for memberA device 01a192ae-6f45-72b9-ad7c-e02298bacbcc and uploaded 3 UsageEvents to PostgreSQL with GET /api/v1/member/usage reporting exact total_bytes=23415. |
| AT15 | pass | [AT15.json](AT15.json) / [AT15.md](AT15.md) | Live mTLS /api/v1/agent/usage-checkpoint and /api/v1/agent/usage-batches on serving node 01a192ae-1201-70bf-b380-a6dfdbc2724c (incarnation 01a192ae-1361-7d54-a603-22b10f2ac4a6, initial real agent seq=5) verified Ed25519 recovery signature, out-of-order seq {6,8} -> gap fill seq 7 advancing highest_contiguous_seq to 8, batch replay and duplicate retry with zero double-counted bytes, payload_sha256 conflict audit (usage_event_conflicts=1), rejected seq 10 (>2^53 decimal string 9007199254740993) reporting blocked_seq=10 then advancing to 11, 409/403 error guards, and live fvpn-usage-admin freeze/begin-recovery/complete-recovery rotating ledger_epoch 3b3d68b0-fc62-4d15-9d83-f0aea976d52e -> 01a192b8-b497-7a3e-80a7-af6da7effaf8 with replay_unavailable reconciliation to seq 12. |
| AT16 | pass | [AT16.json](AT16.json) / [AT16.md](AT16.md) | Live node fvpn-w4-usage-node verified sampler restart continuity (event_seq 1->2), D-067 provisional-lease recovery across bring_up (5786ms/6925ms) and agent_restart (6414ms/6712ms), and live fail-closed withdrawal via fvpn-watchdog. |
| AT17 | pass | [AT17.json](AT17.json) / [AT17.md](AT17.md) | Live GET /api/v1/member/usage and ?period_id=2026-10&granularity=day returned real counted memberA traffic totals (upload=13032, download=18163, total=31195) within half-open UTC month bounds [2026-10-01T00:00:00Z, 2026-11-01T00:00:00Z), while central and node mechanisms verified 00:00:00Z UTC month-boundary splitting, late-arriving previous-month event attribution within the 24h watermark, and >5s wall-clock rollback protection. |
| AT18 | pass | [AT18.json](AT18.json) / [AT18.md](AT18.md) | Live serving exit fvpn-w4-usage-node installed kernel tc HTB + fq_codel on fvmb0 and fvup0 with host reserve class 1:fffe (100Mbit) and per-member classes at effective_rate = min(plan=80M/40M, tier0=100M/50M, exit=60M/30M) = 60Mbit down / 30Mbit up, shaped concurrent memberA (+1106/+1513 B) and memberB (+1066/+1515 B) flows, and preserved mTLS control-plane responsiveness (HTTP 200). |
| AT19 | pass | [AT19.json](AT19.json) / [AT19.md](AT19.md) | Live multi-node test enrolled second exit 01a192b0-2fb8-719f-9818-d04e4237ed6f (node2) and non-carrying node 01a192b1-8199-7ef2-8edf-1bcb771d17a6, routed memberA traffic across both carrying exits (aggregating 28614 bytes across 2 exits), verified shadow mode (60Mbit on node, 15Mbit on node2), proved Transition 1 (policy.activated mode=enforce: carrying gens node 7->8, node2 4->5, non-carrying 0->0; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in 6533 ms via observation 01a192b1-bd70-76df-b761-30054c9a5e34; replay no double bump) and Transition 2 (fup.member_generation_changed via usage threshold crossing: carrying gens node 9->10, node2 6->7, non-carrying 0->0; Tier 2 5Mbit/2Mbit tc classes on both exits; fair_use.applying true->false in 54453 ms via observation 01a192b2-bdcf-7c42-9349-1b50d8c8f913; replay no double bump), and verified ENR-03 per-incarnation retained usage across node2 re-enrollment (01a192b0-311f-7e4f-b1ad-d6b7f7de5a94 -> 01a192b2-f8a4-7eda-bf06-f55aba2ea13d). |
| AT20 | pass | [AT20.json](AT20.json) / [AT20.md](AT20.md) | Same node 01a192b4-46ef-7509-96da-98696f249696 reenrolled 01a192b4-6629-7435-a77f-2b0290fe4560 -> 01a192b5-8940-78fe-b6f0-b266ba63d5b8 through ControlledReinstall. Member period 2026-10: before 33775 + held 7780 = after 41555 bytes; 1 original events retained once, 1 agent batches returned the identical stored canonical result, origin ACK 2, independent current ACK 1. Meter interval 10s <= central bound 10s. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history. D-130 fresh unknown gap accepted at ACK 1, followed by counted event 2 at ACK 2: measured 1066 upload + 1513 download = 2579 bytes, exactly attributed to the original member/device/period once. All stored event/sample/gap endpoints equal the canonical UTC microsecond projection; same-key replays retain stored ACK 1 and ACK 2 results without changing attribution. |

## D-067 comparison

The previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 25 is in the image source. Devstack #240 measured **80622 ms** on its separate path.

| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |
|---|---|---|---|---|---|---|
| bring_up | 5021 ms | 4906 ms | 5786 ms | 6925 ms | `01a192ae-cfa2-70c4-84b9-d23904520b8d` at `2026-10-31T15:58:12.39275Z` | `45cdabe0-75f3-49b9-b647-dd0050c41bac` at `2026-10-31T15:58:17.298767Z` |
| agent_restart | 5025 ms | 6006 ms | 6414 ms | 6712 ms | `01a192af-0ae2-7046-b817-84e3ed580db3` at `2026-10-31T15:58:27.55995Z` | `2d00acc0-0d5a-4be2-aba3-1bbed614fcf7` at `2026-10-31T15:58:33.56613Z` |

Both provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).

## Fair-Use convergence

| Trigger | Clearing observation | Trigger → applying false | Tier 2 installed on both exits |
|---|---|---|---|
| policy_activated_enforce_tier2 | `01a192b1-bd70-76df-b761-30054c9a5e34` | 6533 ms | true |
| fup_member_generation_changed_tier0_to_tier2 | `01a192b2-bdcf-7c42-9349-1b50d8c8f913` | 54453 ms | true |

The usage-threshold timer starts with real traffic and includes the normal sample/upload cadence; the activation timer starts with policy activation. Follow-up 3 used synthetic batch/ACK fixtures (5387 / 6151 ms), so those timings are not identical paths.

## ENR-03 and UTC boundary

Node `01a192b0-2fb8-719f-9818-d04e4237ed6f`: retired incarnation `01a192b0-311f-7e4f-b1ad-d6b7f7de5a94`, new incarnation `01a192b2-f8a4-7eda-bf06-f55aba2ea13d`; old usage fixed `true`; member total unchanged at reinstall `true`; new traffic accumulated `true`. Member totals: `28614` → `28614` → `31195` bytes. Detailed per-incarnation bytes and timestamps are in [AT19.json](AT19.json). [AT17.json](AT17.json) re-runs the UTC split (1001/2003 bytes → October 500/1001 and November 501/1002), late events and clock rollback.

Prior runs: [Follow-up 5](history/follow-up-5/ARCHIVE.md), [Follow-up 4](history/follow-up-4/ARCHIVE.md), [Follow-up 3](history/followup3/summary.md), [Follow-up 2](history/followup2/summary.md), [initial](history/initial/).

## Which serving fix engaged

no foreign rules; Docker DNS NAT absent; #240 lab cleanup made this node serve; D-079 present but not exercised. The first real serving observation `01a192ae-e3bf-70a8-a75c-e79f4a504873` (received `2026-10-31T15:58:17.547466Z`) is preserved in each case's `first_serving_apply.observation`, alongside `serving 1`, the admitted-device counter deltas and the ruleset read-back. absent: ObservationReport has no foreign-rule classification field on this tree; classification above is enforce.LoopbackEvidence over the live read-back.

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

	counter d_01a192ae6f4572b9ad7ce02298bacbcc {
		packets 8 bytes 1514
	}

	counter d_01a192aeaa4e7791b0c77b3d5974b528 {
		packets 0 bytes 0
	}

	counter nd_01a192ae6f4572b9ad7ce02298bacbcc {
		packets 0 bytes 0
	}

	counter nd_01a192aeaa4e7791b0c77b3d5974b528 {
		packets 0 bytes 0
	}

	counter nu_01a192ae6f4572b9ad7ce02298bacbcc {
		packets 0 bytes 0
	}

	counter nu_01a192aeaa4e7791b0c77b3d5974b528 {
		packets 0 bytes 0
	}

	counter u_01a192ae6f4572b9ad7ce02298bacbcc {
		packets 8 bytes 1066
	}

	counter u_01a192aeaa4e7791b0c77b3d5974b528 {
		packets 0 bytes 0
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.68.183.201 timeout 9m54s805ms expires 9m52s817ms, 100.68.243.111 timeout 9m54s805ms expires 9m52s817ms }
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
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=6 fvpn_metering=forward_admission_v1"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a192ad-e9a0-79d7-9490-c3c5320fcacf:955e"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a192ad-ec27-74ff-891f-09d9c0668310:87c5"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.68.243.111 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a192ae6f4572b9ad7ce02298bacbcc" meta priority set 1:955e accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.68.243.111 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a192ae6f4572b9ad7ce02298bacbcc" meta priority set 1:955e accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.68.183.201 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a192aeaa4e7791b0c77b3d5974b528" meta priority set 1:87c5 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.68.183.201 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a192aeaa4e7791b0c77b3d5974b528" meta priority set 1:87c5 accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.68.243.111 counter name "u_01a192ae6f4572b9ad7ce02298bacbcc" meta priority set 1:955e accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.68.183.201 counter name "u_01a192aeaa4e7791b0c77b3d5974b528" meta priority set 1:87c5 accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.68.243.111 tcp sport 53 counter name "nd_01a192ae6f4572b9ad7ce02298bacbcc" meta priority set 1:955e accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.68.243.111 udp sport 53 counter name "nd_01a192ae6f4572b9ad7ce02298bacbcc" meta priority set 1:955e accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.68.243.111 counter name "d_01a192ae6f4572b9ad7ce02298bacbcc" meta priority set 1:955e accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.68.183.201 tcp sport 53 counter name "nd_01a192aeaa4e7791b0c77b3d5974b528" meta priority set 1:87c5 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.68.183.201 udp sport 53 counter name "nd_01a192aeaa4e7791b0c77b3d5974b528" meta priority set 1:87c5 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.68.183.201 counter name "d_01a192aeaa4e7791b0c77b3d5974b528" meta priority set 1:87c5 accept
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

Same node 01a192b4-46ef-7509-96da-98696f249696 reenrolled 01a192b4-6629-7435-a77f-2b0290fe4560 -> 01a192b5-8940-78fe-b6f0-b266ba63d5b8 through ControlledReinstall. Member period 2026-10: before 33775 + held 7780 = after 41555 bytes; 1 original events retained once, 1 agent batches returned the identical stored canonical result, origin ACK 2, independent current ACK 1. Meter interval 10s <= central bound 10s. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history. D-130 fresh unknown gap accepted at ACK 1, followed by counted event 2 at ACK 2: measured 1066 upload + 1513 download = 2579 bytes, exactly attributed to the original member/device/period once. All stored event/sample/gap endpoints equal the canonical UTC microsecond projection; same-key replays retain stored ACK 1 and ACK 2 results without changing attribution. See [AT20.json](AT20.json) for the original CLI, controlled reinstall outcome, immutable payloads, central samples, stored batch replay, independent ACKs, configuration and refusal audit. This follows the devstack follow-up 19 corrected path in [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31). The systemd installer node is quiesced in documented unit order; pause-agent applies only to ordinary supervised nodes. No ledger rows or retirement timestamps are edited.

### D-130 scope and known gaps

AT20 observes the fresh unknown-continuity gap at ACK 1 before producing counted event 2. Its `d130_*` observations retain the actual agent payloads/batches, original member/device/period, exact kernel direction counters, member totals before/after/replay and read-only event/sample/gap interval projections plus the three installed finite `<=` constraints. Same-key replays of both saved batches preserve the stored results, including ACK 1 after the stream reaches 2. Newly minted intervals must be UTC whole microseconds with width at least 1 µs. See [agent D-130](../../../node/usage-meter.md#interval-precision-and-compatibility-d-130) and [central D-130](../../../usage/2026-10-31-d130-interval-precision.md).

This harness builds the current node source; already sealed pre-D-130 1 ns retry/recovery belongs to enrollment 9 and is not claimed here. Enrollment 9's controlled_reinstall qualification and devstack 20's F10 diagnostics are not re-measured. Physical arm64 remains unmeasured (amd64/arm64 cross-builds only). AT17 UTC boundary/clock and power-loss mechanisms retain their deterministic fixture scope, not a real clock-change or physical power-loss claim.

D-118: [devstack 19 retained 8/8 canonical origin payload/hash pairs](../../runs/2026-10-31/followup19/usage-retention.md), with origin ACKs 4→4 and 3→4. [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31-follow-up-19) answers follow-up 5 request 1. Our [AT20.json](AT20.json) records `held_events_before_reenrollment`, `reinstall_original_payload_bytes_equal`, `handed_off_events`, origin ACKs and `member_period_before` / `member_period_after` / `member_period_after_replay`. DB/WAL hashes may change during adoption; canonical payloads, origin sequence/hash and the independent origin ACK establish retention. Whole-file equality remains only a pre-adoption quiescence check. Existing assertions are unchanged.
