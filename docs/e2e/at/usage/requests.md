# Requests — Follow-up 7

## Closed: FLOWTABLE_BYPASS (AT14–AT19)

[PR #240](https://github.com/TED-H-com/Freedom-VPN/pull/240) removes Docker DNS NAT before the lab agent starts. [PR #241](https://github.com/TED-H-com/Freedom-VPN/pull/241) adds D-079 loopback classification. Both are in this run; [PR #246](https://github.com/TED-H-com/Freedom-VPN/pull/246) previously established the lab-only result. The old blocked observations remain in [history/followup3/requests.md](history/followup3/requests.md).

## Which serving fix engaged

no foreign rules; Docker DNS NAT absent; #240 lab cleanup made this node serve; D-079 present but not exercised. The first real serving observation `01a1962e-9736-7bab-8275-94e126217cbf` (received `2026-11-01T08:16:38.207535Z`) is preserved in each case's `first_serving_apply.observation`, alongside `serving 1`, the admitted-device counter deltas and the ruleset read-back. absent: ObservationReport has no foreign-rule classification field on this tree; classification above is enforce.LoopbackEvidence over the live read-back.

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

	counter d_01a1962e21a6775a8fba146baf20d5ab {
		packets 8 bytes 1513
	}

	counter d_01a1962e5e197246b6f4b22c46702df1 {
		packets 0 bytes 0
	}

	counter nd_01a1962e21a6775a8fba146baf20d5ab {
		packets 0 bytes 0
	}

	counter nd_01a1962e5e197246b6f4b22c46702df1 {
		packets 0 bytes 0
	}

	counter nu_01a1962e21a6775a8fba146baf20d5ab {
		packets 0 bytes 0
	}

	counter nu_01a1962e5e197246b6f4b22c46702df1 {
		packets 0 bytes 0
	}

	counter u_01a1962e21a6775a8fba146baf20d5ab {
		packets 8 bytes 1066
	}

	counter u_01a1962e5e197246b6f4b22c46702df1 {
		packets 0 bytes 0
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.76.35.174 timeout 9m54s834ms expires 9m52s972ms, 100.76.143.104 timeout 9m54s834ms expires 9m52s972ms }
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
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1962d-a6e8-731a-abd1-f5afd32576fb:642a"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1962d-a772-7bd0-87eb-506a7e5a948c:ac52"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.76.143.104 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1962e21a6775a8fba146baf20d5ab" meta priority set 1:642a accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.76.143.104 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1962e21a6775a8fba146baf20d5ab" meta priority set 1:642a accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.76.35.174 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1962e5e197246b6f4b22c46702df1" meta priority set 1:ac52 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.76.35.174 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1962e5e197246b6f4b22c46702df1" meta priority set 1:ac52 accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.76.143.104 counter name "u_01a1962e21a6775a8fba146baf20d5ab" meta priority set 1:642a accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.76.35.174 counter name "u_01a1962e5e197246b6f4b22c46702df1" meta priority set 1:ac52 accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.76.143.104 tcp sport 53 counter name "nd_01a1962e21a6775a8fba146baf20d5ab" meta priority set 1:642a accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.76.143.104 udp sport 53 counter name "nd_01a1962e21a6775a8fba146baf20d5ab" meta priority set 1:642a accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.76.143.104 counter name "d_01a1962e21a6775a8fba146baf20d5ab" meta priority set 1:642a accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.76.35.174 tcp sport 53 counter name "nd_01a1962e5e197246b6f4b22c46702df1" meta priority set 1:ac52 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.76.35.174 udp sport 53 counter name "nd_01a1962e5e197246b6f4b22c46702df1" meta priority set 1:ac52 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.76.35.174 counter name "d_01a1962e5e197246b6f4b22c46702df1" meta priority set 1:ac52 accept
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
| bring_up | 5021 ms | 4922 ms | 5860 ms | 6786 ms | `01a1962e-8369-7e46-9266-9587ee3a8beb` at `2026-11-01T08:16:33.135196Z` | `cabe9999-9cd1-4352-b2c7-f376b063a640` at `2026-11-01T08:16:38.057453Z` |
| agent_restart | 5025 ms | 4919 ms | 5944 ms | 6248 ms | `01a1962e-bab9-7866-9156-69121f5a2a6e` at `2026-11-01T08:16:47.29578Z` | `fe4c7cdd-bfd1-4809-b675-66f94f2807f1` at `2026-11-01T08:16:52.215652Z` |

Both provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).

### For the enrollment lane

Same node 01a19633-b721-7028-a6db-023c4b337ade reenrolled 01a19633-bcf7-7dfb-a64e-50036b5891db -> 01a19635-0720-7af1-b709-67d12e572248 through ControlledReinstall. Member period 2026-11: before 33681 + held 7736 = after 41417 bytes; 1 original events retained once, 1 agent batches returned the identical stored canonical result, origin ACK 2, independent current ACK 1. Meter interval 10s <= central bound 10s. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history. D-130 fresh unknown gap accepted at ACK 1, followed by counted event 2 at ACK 2: measured 1066 upload + 1514 download = 2580 bytes, exactly attributed to the original member/device/period once. All stored event/sample/gap endpoints equal the canonical UTC microsecond projection; same-key replays retain stored ACK 1 and ACK 2 results without changing attribution. See [AT20.json](AT20.json) for the original CLI, controlled reinstall outcome, immutable payloads, central samples, stored batch replay, independent ACKs, configuration and refusal audit. This follows the devstack follow-up 19 corrected path in [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31). The systemd installer node is quiesced in documented unit order; pause-agent applies only to ordinary supervised nodes. No ledger rows or retirement timestamps are edited.
