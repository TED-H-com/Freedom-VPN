# forwarding

Member forwarding follows a serving lease

Status: blocked

Run: 0e34514e-bad1-4fe6-89fa-ff55da5fee12

Source: 59903b57c41115f02ccfb0b2bc1d56dc095b5a39

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 5
- baseline_marker: false
- member_accept_rules: 6
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=5 fvpn_metering=forward_admission_v1" | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_class=01a190a6-49a7-70d8-937f-80b4210102a7:1e89" | iifname "fvmb0" oifname "fvup0" meta nfproto ipv6 counter name "ctv6" drop | iifname "fvup0" oifname "fvmb0" meta nfproto ipv6 co
- note: forward chain policy is drop, generation 5, with 6 member accept rules; AT10 is not claimed
- owner: w2-agent-core / w1-netlab: compare named drop/admission counters and tunnel captures in member-traffic; lease generation alone is not the gate
