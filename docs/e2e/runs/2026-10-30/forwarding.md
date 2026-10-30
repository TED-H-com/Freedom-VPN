# forwarding

Member forwarding follows a serving lease

Status: blocked

Run: 6bf270e4-abb1-43ff-93eb-18d282fc7c2a

Source: 5477de091f449047193fdc9114cfca4ced5b36e6

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 0
- baseline_marker: true
- member_accept_rules: 0
- forward_excerpt: type filter hook forward priority mangle; policy drop; | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1" | iifname "fvmb0" counter name "ctbase" drop | oifname "fvmb0" counter name "ctbase" drop | iifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop | oifname "fvmb0" meta nfproto ipv6 counter name "ctv6" drop
- note: forward chain policy is drop, generation 0, with 0 member accept rules; AT10 is not claimed

