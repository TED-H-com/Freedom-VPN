# forwarding

Member forwarding follows a serving lease

Status: pass

Run: 273496b3-9897-43b0-b289-a571d1d14561

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 6
- baseline_marker: false
- member_accept_rules: 6
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=6 fvpn_metering=forward_admission_v1" | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a18c4e-6a0f-76b9-a0d5-742e852df12e:186c" | iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop | iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 co
- note: forward chain generation is 6 with member accept rules
