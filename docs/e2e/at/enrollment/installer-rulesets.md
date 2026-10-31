# Installer text ruleset read-back — follow-up 9

Run `891ee9b8-bf01-4583-a47a-9542c8b04c8a`. Full text read-back preserves ownership comments.

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

	counter d_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	counter nd_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	counter nu_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	counter u_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.102.180.216 timeout 9m59s838ms expires 9m58s104ms }
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
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1927f-18e6-7941-929a-b43046014a22:df9a"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 counter name "u_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.102.180.216 tcp sport 53 counter name "nd_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.102.180.216 udp sport 53 counter name "nd_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.102.180.216 counter name "d_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
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
mono_ns 42815146974899
wall_unix_ns 1793459277630603811
policy_expires_mono_ns 43414960024440
baseline_ok 1
serving 1
generation 3
reason serving
```

Agent active: `True`; four chains/hookless marker: `True`. Central forwarding: `True` at `2026-10-31T15:07:57.604594Z`.

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

	counter d_01a1927f3566703fb6390260ae3de34e {
		packets 12 bytes 716
	}

	counter nd_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	counter nu_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	counter u_01a1927f3566703fb6390260ae3de34e {
		packets 18 bytes 1100
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.102.180.216 timeout 8m56s415ms expires 8m56s155ms }
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
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1927f-18e6-7941-929a-b43046014a22:df9a"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 counter name "u_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.102.180.216 tcp sport 53 counter name "nd_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.102.180.216 udp sport 53 counter name "nd_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.102.180.216 counter name "d_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
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
mono_ns 42878562930204
wall_unix_ns 1793459341046559145
policy_expires_mono_ns 43414961024227
baseline_ok 1
serving 1
generation 3
reason serving
```

Agent active: `True`; four chains/hookless marker: `True`. Central forwarding: `True` at `2026-10-31T15:09:01.023115Z`.

## After controlled reinstall

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

	counter d_01a1927f3566703fb6390260ae3de34e {
		packets 18 bytes 1074
	}

	counter nd_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	counter nu_01a1927f3566703fb6390260ae3de34e {
		packets 0 bytes 0
	}

	counter u_01a1927f3566703fb6390260ae3de34e {
		packets 27 bytes 1650
	}

	set allowed_v4 {
		type ipv4_addr
		flags timeout
		elements = { 100.102.180.216 timeout 9m59s836ms expires 9m58s920ms }
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
		ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1927f-18e6-7941-929a-b43046014a22:df9a"
		iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
		iifname "fvmb0" oifname "fvup0" ct state invalid counter name "ctinv" drop
		iifname "fvup0" oifname "fvmb0" ct state invalid counter name "ctinv" drop
		iifname "fvmb0" oifname "fvup0" ip saddr != @allowed_v4 counter name "ctunku" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 ip daddr @resolver_v4 tcp dport 53 counter name "nu_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 ip daddr @resolver_v4 udp dport 53 counter name "nu_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvmb0" oifname "fvup0" ip daddr @resolver_v4 counter name "ctbadns" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @denied_v4 counter name "ctdeny" drop
		iifname "fvmb0" oifname "fvup0" ip daddr @host_v4 counter name "cthost" drop
		iifname "fvmb0" oifname "fvup0" ip saddr 100.102.180.216 counter name "u_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip daddr != @allowed_v4 counter name "ctunkd" drop
		iifname "fvup0" oifname "fvmb0" ct state != { established, related } counter name "ctnodn" drop
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.102.180.216 tcp sport 53 counter name "nd_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip saddr @resolver_v4 ip daddr 100.102.180.216 udp sport 53 counter name "nd_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
		iifname "fvup0" oifname "fvmb0" ip daddr 100.102.180.216 counter name "d_01a1927f3566703fb6390260ae3de34e" meta priority set 1:df9a accept
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
mono_ns 43141544476131
wall_unix_ns 1793459604028104990
policy_expires_mono_ns 43741375072391
baseline_ok 1
serving 1
generation 7
reason serving
```

Agent active: `True`; four chains/hookless marker: `True`. Central forwarding: `True` at `2026-10-31T15:13:24.00315Z`.
