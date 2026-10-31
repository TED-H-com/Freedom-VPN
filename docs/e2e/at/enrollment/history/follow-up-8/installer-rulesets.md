# Installer text ruleset read-back — follow-up 8

Run `f27a81a0-183e-46b4-85a8-304b2f64b635`. Complete `nft list ruleset` text; nft 1.0.9 JSON omits comments.

## Initial serving

```nft
table inet fvpn_install {
	chain owner {
		comment "managed-by=fvpn-node-install"
	}
}
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

	counter d_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	counter nd_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	counter nu_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	counter u_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.116.84.27 timeout 9m59s840ms expires 9m58s162ms }
	}

	set denied_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 0.0.0.0/8, 10.0.0.0/8,
			     100.64.0.0/10, 127.0.0.0/8,
			     169.254.0.0/16, 172.16.0.0/12,
			     192.0.0.0/24, 192.0.2.0/24,
			     192.88.99.0/24, 192.168.0.0/16,
			     198.18.0.0/15, 198.51.100.0/24,
			     203.0.113.0/24, 224.0.0.0/3 }
	}

	set host_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 192.0.11.24 }
	}

	set resolver_v4 {
		type ipv4_addr
		elements = { 192.0.11.53 }
	}

	chain forward {
		type filter hook forward priority mangle; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=3 fvpn_metering=forward_admission_v1"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1923f-1faf-73ff-b026-9fb77ed6ab40:2d29"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 counter name "u_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.116.84.27 tcp sport 53 counter name "nd_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.116.84.27 udp sport 53 counter name "nd_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.116.84.27 counter name "d_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
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
		oifname "fvup0" ip saddr @allowed_v4 snat ip to 192.0.11.24
	}
}
```

```text
v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 38622861345024
wall_unix_ns 1793455085344973897
policy_expires_mono_ns 39222690445274
baseline_ok 1
serving 1
generation 3
reason serving
```

Agent active: `True`; four chains/hookless marker: `True`. Central forwarding: `True` at `2026-10-31T13:58:05.319085Z`.

## After rerun

```nft
table inet fvpn_install {
	chain owner {
		comment "managed-by=fvpn-node-install"
	}
}
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

	counter d_01a1923f3cae7e15b547a08502760a66 {
		packets 12 bytes 716
	}

	counter nd_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	counter nu_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	counter u_01a1923f3cae7e15b547a08502760a66 {
		packets 18 bytes 1100
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.116.84.27 timeout 8m57s11ms expires 8m56s741ms }
	}

	set denied_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 0.0.0.0/8, 10.0.0.0/8,
			     100.64.0.0/10, 127.0.0.0/8,
			     169.254.0.0/16, 172.16.0.0/12,
			     192.0.0.0/24, 192.0.2.0/24,
			     192.88.99.0/24, 192.168.0.0/16,
			     198.18.0.0/15, 198.51.100.0/24,
			     203.0.113.0/24, 224.0.0.0/3 }
	}

	set host_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 192.0.11.24 }
	}

	set resolver_v4 {
		type ipv4_addr
		elements = { 192.0.11.53 }
	}

	chain forward {
		type filter hook forward priority mangle; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=3 fvpn_metering=forward_admission_v1"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1923f-1faf-73ff-b026-9fb77ed6ab40:2d29"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 counter name "u_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.116.84.27 tcp sport 53 counter name "nd_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.116.84.27 udp sport 53 counter name "nd_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.116.84.27 counter name "d_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
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
		oifname "fvup0" ip saddr @allowed_v4 snat ip to 192.0.11.24
	}
}
```

```text
v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 38685693260715
wall_unix_ns 1793455148176889560
policy_expires_mono_ns 39222688627354
baseline_ok 1
serving 1
generation 3
reason serving
```

Agent active: `True`; four chains/hookless marker: `True`. Central forwarding: `True` at `2026-10-31T13:59:08.152357Z`.

## After controlled reinstall and API republication

```nft
table inet fvpn_install {
	chain owner {
		comment "managed-by=fvpn-node-install"
	}
}
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

	counter d_01a1923f3cae7e15b547a08502760a66 {
		packets 18 bytes 1074
	}

	counter nd_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	counter nu_01a1923f3cae7e15b547a08502760a66 {
		packets 0 bytes 0
	}

	counter u_01a1923f3cae7e15b547a08502760a66 {
		packets 27 bytes 1650
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.116.84.27 timeout 9m59s838ms expires 9m58s876ms }
	}

	set denied_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 0.0.0.0/8, 10.0.0.0/8,
			     100.64.0.0/10, 127.0.0.0/8,
			     169.254.0.0/16, 172.16.0.0/12,
			     192.0.0.0/24, 192.0.2.0/24,
			     192.88.99.0/24, 192.168.0.0/16,
			     198.18.0.0/15, 198.51.100.0/24,
			     203.0.113.0/24, 224.0.0.0/3 }
	}

	set host_v4 {
		type ipv4_addr
		flags interval
		auto-merge
		elements = { 192.0.11.24 }
	}

	set resolver_v4 {
		type ipv4_addr
		elements = { 192.0.11.53 }
	}

	chain forward {
		type filter hook forward priority mangle; policy drop;
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=7 fvpn_metering=forward_admission_v1"
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1923f-1faf-73ff-b026-9fb77ed6ab40:2d29"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.116.84.27 counter name "u_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.116.84.27 tcp sport 53 counter name "nd_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.116.84.27 udp sport 53 counter name "nd_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.116.84.27 counter name "d_01a1923f3cae7e15b547a08502760a66" meta priority set 1:2d29 accept
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
		oifname "fvup0" ip saddr @allowed_v4 snat ip to 192.0.11.24
	}
}
```

```text
v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 38950697368199
wall_unix_ns 1793455413180997064
policy_expires_mono_ns 39550526195866
baseline_ok 1
serving 1
generation 7
reason serving
```

Agent active: `True`; four chains/hookless marker: `True`. Central forwarding: `True` at `2026-10-31T14:03:33.157959Z`.
