# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 956c335d-0449-44ac-aa63-544154456397

Source: 706d7cf6cec8c00c209dbe029eaa7825921f12ac

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
- binding_valid_from: 2026-10-31T14:19:06Z
- binding_valid_until: 2026-10-31T15:18:58Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-a58fddce006ca5009c5edc9a
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-a58fddce006ca5009c5edc9a
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.128.120 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.83.128.120/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.128.120
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.128.120 dev fvnb0 table local proto kernel scope host src 100.83.128.120
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.128.120
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":191,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":192,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":193,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":194,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.128.120","timeout":597,"expires":597}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":199,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":200,"comment":"fvpn_class=01a19253-f80e-76f7-98eb-d84e5237c3f3:3288","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":201,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":202,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":203,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":204,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":205,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":206,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":207,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":208,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":209,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":210,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":211,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"counter":"u_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":212,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":214,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":215,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":216,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":217,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"counter":"d_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":220,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=28
rx_bytes=3472
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:19:10.301675 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:11.348654 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:12.373643 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:13.396649 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:14.421655 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893139 IP 192.0.3.10.443 > 100.83.128.120.39882: tcp 0
14:19:15.893155 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893158 IP 192.0.3.10.443 > 100.83.128.120.39882: tcp 0
14:19:15.893161 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0

9 packets captured
9 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:19:15.893019 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893019 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893019 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893114 IP 192.0.3.10.443 > 100.83.128.120.39882: tcp 0
14:19:15.893117 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893118 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893131 IP 192.0.3.10.443 > 100.83.128.120.39882: tcp 0
14:19:15.893182 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0
14:19:15.893182 IP 100.83.128.120.39882 > 192.0.3.10.443: tcp 0

9 packets captured
9 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
14:19:15.893094 IP 192.0.3.20.39882 > 192.0.3.10.443: tcp 0
14:19:15.893108 IP 192.0.3.10.443 > 192.0.3.20.39882: tcp 0
14:19:15.893121 IP 192.0.3.20.39882 > 192.0.3.10.443: tcp 0
14:19:15.893123 IP 192.0.3.20.39882 > 192.0.3.10.443: tcp 0
14:19:15.893128 IP 192.0.3.10.443 > 192.0.3.20.39882: tcp 0
14:19:15.893185 IP 192.0.3.20.39882 > 192.0.3.10.443: tcp 0
14:19:15.893186 IP 192.0.3.20.39882 > 192.0.3.10.443: tcp 0

7 packets captured
7 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.128.120 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.83.128.120/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.128.120
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.128.120 dev fvnb0 table local proto kernel scope host src 100.83.128.120
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.128.120
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":191,"packets":2,"bytes":120}},{"counter":{"family":"inet","name":"nd_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":192,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":193,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":194,"packets":7,"bytes":380}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.128.120","timeout":593,"expires":588}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":228,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":229,"comment":"fvpn_class=01a19253-f80e-76f7-98eb-d84e5237c3f3:3288","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":230,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":231,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":232,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":233,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":234,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":235,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":236,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":237,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":238,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":239,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":240,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"counter":"u_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":241,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":243,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":244,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":245,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":246,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"counter":"d_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":249,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 8 CLOSE src=100.83.128.120 dst=192.0.3.10 sport=39882 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=39882 mark=0 use=1
- after_member_tunnel: rx_packets=4
rx_bytes=316
tx_packets=9
tx_bytes=936
- after_node_tunnel: rx_packets=9
rx_bytes=936
tx_packets=6
tx_bytes=556
- after_node_uplink: rx_packets=33
rx_bytes=3856
tx_packets=20
tx_bytes=1912
- origin_arrival: {"probe_id":"74c89830-e4e1-4450-a070-66fa262148bc","requests":0,"source_address":""}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: false
- admission_bytes_before: 0
- admission_bytes_after: 0
- first_stopped_hop: see packet captures and rule counter deltas; do not infer a lease defect
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 39890915489068
wall_unix_ns 1793456353399118024
policy_expires_mono_ns 40483949271771
baseline_ok 1
serving 1
generation 5
reason serving
- counted_attempt: 2
- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.83.128.120 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-a58fddce006ca5009c5edc9a
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-a58fddce006ca5009c5edc9a
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.128.120 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.83.128.120/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.128.120
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.128.120 dev fvnb0 table local proto kernel scope host src 100.83.128.120
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.128.120
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":191,"packets":2,"bytes":120}},{"counter":{"family":"inet","name":"nd_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":192,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":193,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":194,"packets":7,"bytes":380}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.128.120","timeout":588,"expires":585}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":257,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":258,"comment":"fvpn_class=01a19253-f80e-76f7-98eb-d84e5237c3f3:3288","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":259,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":260,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":261,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":262,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":263,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":264,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":265,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":266,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":267,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":268,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":269,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"counter":"u_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":270,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":272,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":273,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":274,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":275,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"counter":"d_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":278,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: ipv4     2 tcp      6 4 CLOSE src=100.83.128.120 dst=192.0.3.10 sport=39882 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=39882 mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- before_member_tunnel: rx_packets=4
rx_bytes=316
tx_packets=9
tx_bytes=936
- before_node_tunnel: rx_packets=9
rx_bytes=936
tx_packets=6
tx_bytes=556
- before_node_uplink: rx_packets=34
rx_bytes=3898
tx_packets=21
tx_bytes=1954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:19:22.373171 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.373618 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.373632 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.373756 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 418
14:19:22.374077 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.374077 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 929
14:19:22.374083 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.374613 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 64
14:19:22.416875 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.416885 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 219
14:19:22.417277 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.417277 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 141
14:19:22.417939 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.418390 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 24
14:19:22.418403 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.418409 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.418413 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:19:22.373536 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.373589 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.373658 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.373777 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 418
14:19:22.373786 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.374052 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 929
14:19:22.374102 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.374783 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 64
14:19:22.416680 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.417120 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 219
14:19:22.417138 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.417234 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 141
14:19:22.418145 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.418378 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 24
14:19:22.418393 IP 192.0.3.10.443 > 100.83.128.120.55978: tcp 0
14:19:22.418423 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0
14:19:22.418573 IP 100.83.128.120.55978 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
14:19:22.373562 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 0
14:19:22.373583 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 0
14:19:22.373662 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 0
14:19:22.373780 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 418
14:19:22.373784 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 0
14:19:22.374045 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 929
14:19:22.374111 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 0
14:19:22.374787 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 64
14:19:22.416653 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 0
14:19:22.417128 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 219
14:19:22.417136 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 0
14:19:22.417229 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 141
14:19:22.418153 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 0
14:19:22.418370 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 24
14:19:22.418390 IP 192.0.3.10.443 > 192.0.3.20.55978: tcp 0
14:19:22.418426 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 0
14:19:22.418587 IP 192.0.3.20.55978 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.128.120 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.83.128.120/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.128.120
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.128.120 dev fvnb0 table local proto kernel scope host src 100.83.128.120
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.128.120
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":191,"packets":10,"bytes":1638}},{"counter":{"family":"inet","name":"nd_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":192,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":193,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192541516702a9b74c2fc8f224e66","table":"fvpn","handle":194,"packets":16,"bytes":1533}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.128.120","timeout":578,"expires":576}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":315,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"comment":"fvpn_class=01a19253-f80e-76f7-98eb-d84e5237c3f3:3288","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":317,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":318,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":320,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":321,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.128.120"}},{"counter":"u_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.128.120"}},{"counter":"d_01a192541516702a9b74c2fc8f224e66"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3288"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 2 CLOSE src=100.83.128.120 dst=192.0.3.10 sport=55978 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=55978 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=12
rx_bytes=2172
tx_packets=18
tx_bytes=2456
- after_node_tunnel: rx_packets=18
rx_bytes=2456
tx_packets=14
tx_bytes=2412
- after_node_uplink: rx_packets=43
rx_bytes=5635
tx_packets=30
tx_bytes=3233
- origin_arrival: {"probe_id":"1c24969c-174a-4c2a-b760-f0ef1c4c7e0f","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 500
- admission_bytes_after: 3171
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 39905910902301
wall_unix_ns 1793456368394531224
policy_expires_mono_ns 40483949271771
baseline_ok 1
serving 1
generation 5
reason serving
