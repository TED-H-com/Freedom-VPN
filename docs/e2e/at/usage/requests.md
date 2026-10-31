# Requests — Follow-up 4

## Closed: FLOWTABLE_BYPASS (AT14–AT19)

[PR #240](https://github.com/TED-H-com/Freedom-VPN/pull/240) removes Docker DNS NAT before the lab agent starts. [PR #241](https://github.com/TED-H-com/Freedom-VPN/pull/241) adds D-079 loopback classification. Both are in this run; [PR #246](https://github.com/TED-H-com/Freedom-VPN/pull/246) previously established the lab-only result. The old blocked observations remain in [history/followup3/requests.md](history/followup3/requests.md).

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

## Current requests

None. D-067 timings are comparative evidence for agent-core, not a verdict on its in-progress follow-up 20. ENR-05/06 and the restore runbook remain outside this lane.

## D-067 comparison

The previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 20 was not in the image source. Devstack #240 measured **80622 ms** on its separate path.

| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |
|---|---|---|---|---|---|---|
| bring_up | 5021 ms | 5021 ms | 5848 ms | 6837 ms | `01a1902b-8337-7ca0-84e9-c6f100fd21e1` at `2026-10-31T04:15:33.180264Z` | `bdec5e45-20fd-446a-bbe0-9b44736baba0` at `2026-10-31T04:15:38.201795Z` |
| agent_restart | 5025 ms | 5022 ms | 5819 ms | 6135 ms | `01a1902b-bb9a-70fe-bc4d-2127f9d72170` at `2026-10-31T04:15:47.614206Z` | `84f3c278-81d8-4452-a768-b8062b2e3b4b` at `2026-10-31T04:15:52.636805Z` |

Both provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).
