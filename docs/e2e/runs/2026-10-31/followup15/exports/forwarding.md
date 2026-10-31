# forwarding

Member forwarding follows a serving lease

Status: pass

Run: 12d10041-9d33-4fbb-a4b0-9d2a21bce785

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 5
- baseline_marker: false
- member_accept_rules: 6
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=5 fvpn_metering=forward_admission_v1" | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a19034-32e4-7d4a-8fc5-3d8df444a439:d01" | iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop | iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 cou
- note: forward chain generation is 5 with member accept rules
