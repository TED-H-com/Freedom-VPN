# forwarding

Member forwarding follows a serving lease

Status: blocked

Run: 5de03570-0436-477d-9847-6f6984432bd8

Source: a7a712aab8a67fcc33514a9ac76cb877a90c8143

Spec: §5, §19 AT10

- nft_table_present: true
- forward_policy: drop
- forward_generation: 0
- baseline_marker: true
- member_accept_rules: 0
- forward_excerpt: type filter hook forward priority mangle; policy drop; | oifname "fvnb0" ct state established,related counter packets 0 bytes 0 accept | iifname "fvnb0" counter packets 0 bytes 0 accept | ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment "fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1" | iifname "fvmb0" counter name "ctbase" drop | oifname "fvmb0" counter name "ctbase" drop
- note: forward chain policy is drop, generation 0, with 0 member accept rules; AT10 is not claimed
- owner: w3-wire-nodes follow-up 11 and w2-agent-core: applied serving policy
