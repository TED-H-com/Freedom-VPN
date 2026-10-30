# forwarding

Member forwarding follows a serving lease

Status: blocked

Run: 742a2b14-355f-462e-be89-52ffb2c1993b

Source: 05d70d63890bb181937dabfe01b9e9a680d76086

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 0
- baseline_marker: true
- member_accept_rules: 0
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1" | iifname "fvmb0" counter name "ctbase" drop | oifname "fvmb0" counter name "ctbase" drop | iifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop | oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
- note: forward chain policy is drop, generation 0, with 0 member accept rules; AT10 is not claimed
- owner: w2-agent-core: current member lease and applied serving policy
