# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: d6aaa798-6c39-49f1-8fa9-5f84f696ac91

Source: a9ad1c471720a2df27f7d4aaa3a0ae1a565fa2cd

Spec: §8, AT02, AT12

- grant: 202
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T14:05:00Z
- binding_valid_until: 2026-10-31T15:04:45Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-71dbdc53a34a97993043e8cd
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-71dbdc53a34a97993043e8cd
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.64.184.43 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.64.184.43/16 brd 100.64.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.64.0.0/16 dev fvnb0 proto kernel scope link src 100.64.184.43
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.64.184.43 dev fvnb0 table local proto kernel scope host src 100.64.184.43
broadcast 100.64.255.255 dev fvnb0 table local proto kernel scope link src 100.64.184.43
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":148,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":149,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":150,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":151,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":152,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":153,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":154,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":312,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":313,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":314,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":315,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.64.184.43","timeout":596,"expires":595}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":320,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":321,"comment":"fvpn_class=01a19246-f179-7972-8f38-af09f7104c66:6d59","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.64.184.43"}},{"counter":"u_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.64.184.43"}},{"counter":"d_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=23
rx_bytes=3164
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:05:07.041700 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.045717 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.045729 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.045906 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 418
14:05:07.046138 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.046389 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 928
14:05:07.046393 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.046844 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 64
14:05:07.089513 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.089530 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 219
14:05:07.090470 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.090845 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 141
14:05:07.091576 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.092124 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 24
14:05:07.092124 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.092158 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
14:05:07.045536 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.045611 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.045854 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.046033 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 418
14:05:07.046045 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.046321 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 928
14:05:07.046507 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.047077 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 64
14:05:07.088665 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.090119 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 219
14:05:07.090132 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.090235 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 141
14:05:07.091857 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
14:05:07.091889 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 24
14:05:07.091903 IP 192.0.3.10.443 > 100.64.184.43.52858: tcp 0
14:05:07.092271 IP 100.64.184.43.52858 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
14:05:07.045594 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 0
14:05:07.045604 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 0
14:05:07.045858 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 0
14:05:07.046036 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 418
14:05:07.046042 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 0
14:05:07.046314 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 928
14:05:07.046509 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 0
14:05:07.047079 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 64
14:05:07.088643 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 0
14:05:07.090124 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 219
14:05:07.090130 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 0
14:05:07.090228 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 141
14:05:07.091861 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 0
14:05:07.091882 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 24
14:05:07.091900 IP 192.0.3.10.443 > 192.0.3.24.52858: tcp 0
14:05:07.092282 IP 192.0.3.24.52858 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.64.184.43 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.64.184.43/16 brd 100.64.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.64.0.0/16 dev fvnb0 proto kernel scope link src 100.64.184.43
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.64.184.43 dev fvnb0 table local proto kernel scope host src 100.64.184.43
broadcast 100.64.255.255 dev fvnb0 table local proto kernel scope link src 100.64.184.43
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":148,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":149,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":150,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":151,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":152,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":153,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":154,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":312,"packets":8,"bytes":1517}},{"counter":{"family":"inet","name":"nd_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":313,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":314,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192472e6c73ffb0ce065412fb82ee","table":"fvpn","handle":315,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.64.184.43","timeout":586,"expires":586}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":379,"comment":"fvpn_class=01a19246-f179-7972-8f38-af09f7104c66:6d59","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":380,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":382,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":383,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":385,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":386,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":387,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":388,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":390,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.64.184.43"}},{"counter":"u_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":391,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":393,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":394,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":395,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.64.184.43"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":396,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.64.184.43"}},{"counter":"d_01a192472e6c73ffb0ce065412fb82ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d59"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":399,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.64.184.43 dst=192.0.3.10 sport=52858 dport=443 src=192.0.3.10 dst=192.0.3.24 sport=443 dport=52858 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=34
rx_bytes=4984
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"463bf568-5171-435c-99e1-876b720f7b15","requests":1,"source_address":"192.0.3.24"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2630
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 39051894524712
wall_unix_ns 1793455514378153658
policy_expires_mono_ns 39638462131127
baseline_ok 1
serving 1
generation 5
reason serving
