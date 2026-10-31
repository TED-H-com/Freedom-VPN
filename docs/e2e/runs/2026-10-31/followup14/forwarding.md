# forwarding

Member forwarding follows a serving lease

Status: pass

Run: 4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 6
- baseline_marker: false
- member_accept_rules: 6
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=6 fvpn_metering=forward_admission_v1" | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a18f68-28c1-70e4-8c7f-700ebb321b0b:4971" | iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop | iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 co
- note: forward chain generation is 6 with member accept rules
