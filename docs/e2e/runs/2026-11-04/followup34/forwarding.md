# forwarding

Member forwarding follows a serving lease

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 5
- baseline_marker: false
- member_accept_rules: 9
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=5 fvpn_metering=forward_admission_v1" | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a1a510-de0a-7755-84af-ebb5224c2b70:14b6" | iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop | iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 co
- note: forward chain generation is 5 with member accept rules
