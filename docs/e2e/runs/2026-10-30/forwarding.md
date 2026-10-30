# forwarding

Member forwarding follows a serving lease

Status: blocked

Run: 3a129919-af3f-4bc9-b54a-8075eae8af45

Source: 79259af1facc1faa798fbb83cc20aa4e74539d1f

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 5
- baseline_marker: false
- member_accept_rules: 6
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=5 fvpn_metering=forward_admission_v1" | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a18da2-15c1-7e7c-8bc8-18726057c99a:be5f" | iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop | iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 co
- note: forward chain policy is drop, generation 5, with 6 member accept rules; AT10 is not claimed
- owner: w3-devstack-e2e: native client exit selection; see route and NetBird network evidence
