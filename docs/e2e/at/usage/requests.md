# Requests — Follow-up 6

## Closed: FLOWTABLE_BYPASS (AT14–AT19)

[PR #240](https://github.com/TED-H-com/Freedom-VPN/pull/240) removes Docker DNS NAT before the lab agent starts. [PR #241](https://github.com/TED-H-com/Freedom-VPN/pull/241) adds D-079 loopback classification. Both are in this run; [PR #246](https://github.com/TED-H-com/Freedom-VPN/pull/246) previously established the lab-only result. The old blocked observations remain in [history/followup3/requests.md](history/followup3/requests.md).

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

## Current requests

Follow-up 5 request 1 closed by [devstack 19](../../RESPONSES.md#usage--2026-10-31-follow-up-19): retained canonical payloads and origin sequence/hash/ACK are the adoption predicate. [Its eight of eight rows](../../runs/2026-10-31/followup19/usage-retention.md) are independent local retention evidence; this run adds before/after attribution in [AT20.json](AT20.json). No assertion required a file-equality fixture change.
No new requests. AT20 is the usage lane ENR-05 extension; it is not the DNS case numbered AT20 in §19. Installer hardware/platform and restore-runbook campaigns remain with their owners.

## D-067 comparison

The previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 25 is in the image source. Devstack #240 measured **80622 ms** on its separate path.

| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |
|---|---|---|---|---|---|---|
| bring_up | 5021 ms | 4906 ms | 5786 ms | 6925 ms | `01a192ae-cfa2-70c4-84b9-d23904520b8d` at `2026-10-31T15:58:12.39275Z` | `45cdabe0-75f3-49b9-b647-dd0050c41bac` at `2026-10-31T15:58:17.298767Z` |
| agent_restart | 5025 ms | 6006 ms | 6414 ms | 6712 ms | `01a192af-0ae2-7046-b817-84e3ed580db3` at `2026-10-31T15:58:27.55995Z` | `2d00acc0-0d5a-4be2-aba3-1bbed614fcf7` at `2026-10-31T15:58:33.56613Z` |

Both provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).

### For the enrollment lane

Same node 01a192b4-46ef-7509-96da-98696f249696 reenrolled 01a192b4-6629-7435-a77f-2b0290fe4560 -> 01a192b5-8940-78fe-b6f0-b266ba63d5b8 through ControlledReinstall. Member period 2026-10: before 33775 + held 7780 = after 41555 bytes; 1 original events retained once, 1 agent batches returned the identical stored canonical result, origin ACK 2, independent current ACK 1. Meter interval 10s <= central bound 10s. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history. D-130 fresh unknown gap accepted at ACK 1, followed by counted event 2 at ACK 2: measured 1066 upload + 1513 download = 2579 bytes, exactly attributed to the original member/device/period once. All stored event/sample/gap endpoints equal the canonical UTC microsecond projection; same-key replays retain stored ACK 1 and ACK 2 results without changing attribution. See [AT20.json](AT20.json) for the original CLI, controlled reinstall outcome, immutable payloads, central samples, stored batch replay, independent ACKs, configuration and refusal audit. This follows the devstack follow-up 19 corrected path in [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31). The systemd installer node is quiesced in documented unit order; pause-agent applies only to ordinary supervised nodes. No ledger rows or retirement timestamps are edited.
