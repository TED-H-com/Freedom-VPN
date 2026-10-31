# Requests — Follow-up 5

## Closed: FLOWTABLE_BYPASS (AT14–AT19)

[PR #240](https://github.com/TED-H-com/Freedom-VPN/pull/240) removes Docker DNS NAT before the lab agent starts. [PR #241](https://github.com/TED-H-com/Freedom-VPN/pull/241) adds D-079 loopback classification. Both are in this run; [PR #246](https://github.com/TED-H-com/Freedom-VPN/pull/246) previously established the lab-only result. The old blocked observations remain in [history/followup3/requests.md](history/followup3/requests.md).

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

## Current requests

- Devstack documentation (nonblocking): update RESPONSES.md usage step 4 after D-118. ControlledReinstall starts the replacement agent for bootstrap, so the usage DB/WAL hashes may change before it returns quiescent. Original canonical event bytes, origin ACK and history remain the preservation invariant. See [diagnostic eba86d44](history/follow-up-5/eba86d44/README.md). Outside-owned file left unchanged.
No product blocker. AT20 is the usage lane ENR-05 extension; it is not the DNS case numbered AT20 in §19. Installer hardware/platform and restore-runbook campaigns remain with their owners.

## D-067 comparison

The previous blocked run measured lease issuance because serving never applied. This run records issuance and serving read-back separately; the original <=65000 ms bound remains on serving read-back. Agent-core follow-up 25 is in the image source. Devstack #240 measured **80622 ms** on its separate path.

| Cycle | Follow-up 3 ready → lease issued | This run ready → lease issued | Ready → serving read-back | Ready → counted flow | Ready observation | Populated lease |
|---|---|---|---|---|---|---|
| bring_up | 5021 ms | 4914 ms | 5801 ms | 7652 ms | `01a191e1-582e-7e13-a799-4d2750b7a18f` at `2026-10-31T12:13:46.932902Z` | `f234dd5d-df9e-4b9e-b6cf-d1f3907d1f3e` at `2026-10-31T12:13:51.847759Z` |
| agent_restart | 5025 ms | 4983 ms | 5874 ms | 6166 ms | `01a191e1-9354-7997-86df-ad948e519d75` at `2026-10-31T12:14:02.073854Z` | `09769a21-7ad9-4fd4-bd9d-b9f9f1eb36df` at `2026-10-31T12:14:07.056909Z` |

Both provisional observations require `AWAITING_MEMBER_GRANTS` on `lease`, `qos`, and `source_binding`, exactly one `errors[]` entry (`stage=lease`), and zero `LEASE_EXPIRED`. Full lease/observation IDs and checks are in [AT16.json](AT16.json).

### For the enrollment lane

Same node 01a191e7-e2d9-7d9f-bdcd-87b79739916f reenrolled 01a191e7-e888-72d1-826b-026b025dafc5 -> 01a191e9-4151-77d6-bbc0-2406573cfe54 through ControlledReinstall. Member period 2026-10: before 33750 + held 7782 = after 41532 bytes; 1 original events retained once, 1 agent batches returned the identical stored canonical result, origin ACK 2, independent current ACK 0. Meter interval 10s <= central bound 10s. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history. See [AT20.json](AT20.json) for the original CLI, controlled reinstall outcome, immutable payloads, central samples, stored batch replay, independent ACKs, configuration and refusal audit. This follows the devstack follow-up 18 path in [RESPONSES.md](../../RESPONSES.md#usage--2026-10-31). The systemd installer node is quiesced in documented unit order; pause-agent applies only to ordinary supervised nodes. No ledger rows or retirement timestamps are edited.
