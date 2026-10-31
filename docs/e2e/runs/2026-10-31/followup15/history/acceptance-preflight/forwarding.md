# forwarding

Member forwarding follows a serving lease

Status: pass

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 6
- baseline_marker: false
- member_accept_rules: 6
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=6 fvpn_metering=forward_admission_v1" | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a19015-a203-7629-af08-191fa4d268f6:15b0" | iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop | iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 co
- note: forward chain generation is 6 with member accept rules
