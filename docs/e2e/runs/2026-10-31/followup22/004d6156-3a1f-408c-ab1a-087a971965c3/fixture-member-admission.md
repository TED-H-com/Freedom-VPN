# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 004d6156-3a1f-408c-ab1a-087a971965c3

Source: ede844d49b1d87477c107f85efda9e06e8811a64

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
- binding_valid_from: 2026-10-31T21:43:56Z
- binding_valid_until: 2026-10-31T22:43:38Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.106.182.203 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-800f059f657c105f4baa170b
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-800f059f657c105f4baa170b
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.106.182.203 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.106.182.203/16 brd 100.106.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.106.0.0/16 dev fvnb0 proto kernel scope link src 100.106.182.203
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.106.182.203 dev fvnb0 table local proto kernel scope host src 100.106.182.203
broadcast 100.106.255.255 dev fvnb0 table local proto kernel scope link src 100.106.182.203
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":82,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":83,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":84,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":85,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":86,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":87,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":88,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":246,"packets":7,"bytes":2411}},{"counter":{"family":"inet","name":"nd_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":247,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":248,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":249,"packets":11,"bytes":2357}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.106.182.203","timeout":594,"expires":592}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":283,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":284,"comment":"fvpn_class=01a193eb-111e-7ecc-83cc-2b853c764f9e:b99d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":285,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":286,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":287,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":288,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":290,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":291,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":292,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":293,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":294,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":295,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.106.182.203"}},{"counter":"u_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":296,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":298,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":299,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":300,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":301,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.106.182.203"}},{"counter":"d_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":304,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: ipv4     2 tcp      6 8 CLOSE src=100.106.182.203 dst=192.0.3.10 sport=20001 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=20001 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- before_member_tunnel: rx_packets=10
rx_bytes=2964
tx_packets=14
tx_bytes=3120
- before_node_tunnel: rx_packets=14
rx_bytes=3120
tx_packets=11
tx_bytes=3056
- before_node_uplink: rx_packets=38
rx_bytes=6254
tx_packets=26
tx_bytes=4093
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
21:44:05.307534 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:06.325660 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:07.349647 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.373655 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.373873 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.373894 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.374112 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 418
21:44:08.374169 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.374432 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 928
21:44:08.374436 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.375009 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 64
21:44:08.415706 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.415715 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 219
21:44:08.416061 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.416061 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 141
21:44:08.416746 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.416955 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 24
21:44:08.416955 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.416974 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0

19 packets captured
19 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
21:44:05.308092 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:05.308159 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:06.325701 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:06.326576 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:06.326599 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:07.350464 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:07.350545 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.373769 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.373827 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.373935 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.374132 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 418
21:44:08.374148 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.374406 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 928
21:44:08.374451 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.375025 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 64
21:44:08.415671 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.415830 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 219
21:44:08.415847 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.416038 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 141
21:44:08.416893 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0
21:44:08.416927 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 24
21:44:08.416940 IP 192.0.3.10.443 > 100.106.182.203.52378: tcp 0
21:44:08.416991 IP 100.106.182.203.52378 > 192.0.3.10.443: tcp 0

23 packets captured
23 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
21:44:05.308124 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0
21:44:05.308152 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:06.325678 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:06.326587 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0
21:44:06.326596 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:07.350501 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0
21:44:07.350538 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:08.373796 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0
21:44:08.373820 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:08.373968 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0
21:44:08.374135 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 418
21:44:08.374144 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:08.374397 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 928
21:44:08.374454 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0
21:44:08.375028 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 64
21:44:08.415649 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:08.415837 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 219
21:44:08.415845 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:08.416026 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 141
21:44:08.416898 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0
21:44:08.416921 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 24
21:44:08.416937 IP 192.0.3.10.443 > 192.0.3.20.52378: tcp 0
21:44:08.416993 IP 192.0.3.20.52378 > 192.0.3.10.443: tcp 0

23 packets captured
23 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.106.182.203 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.106.182.203/16 brd 100.106.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.106.0.0/16 dev fvnb0 proto kernel scope link src 100.106.182.203
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.106.182.203 dev fvnb0 table local proto kernel scope host src 100.106.182.203
broadcast 100.106.255.255 dev fvnb0 table local proto kernel scope link src 100.106.182.203
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":82,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":83,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":84,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":85,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":86,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":87,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":88,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":246,"packets":19,"bytes":4168}},{"counter":{"family":"inet","name":"nd_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":247,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":248,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a193eb56d17673b853b621c607ad6a","table":"fvpn","handle":249,"packets":22,"bytes":3650}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.106.182.203","timeout":584,"expires":583}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"comment":"fvpn_class=01a193eb-111e-7ecc-83cc-2b853c764f9e:b99d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":347,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":348,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":349,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":350,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":353,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.106.182.203"}},{"counter":"u_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":354,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":356,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":357,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":358,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.106.182.203"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.106.182.203"}},{"counter":"d_01a193eb56d17673b853b621c607ad6a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b99d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 tcp      6 5 CLOSE src=100.106.182.203 dst=192.0.3.10 sport=52378 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=52378 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 0 CLOSE src=100.106.182.203 dst=192.0.3.10 sport=20001 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=20001 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=18
rx_bytes=4820
tx_packets=25
tx_bytes=4848
- after_node_tunnel: rx_packets=25
rx_bytes=4848
tx_packets=23
tx_bytes=5296
- after_node_uplink: rx_packets=51
rx_bytes=8221
tx_packets=38
tx_bytes=5582
- origin_arrival: {"probe_id":"a5388d3b-2974-47bf-8194-c406353f8ee7","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 4768
- admission_bytes_after: 7818
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 66589879884197
wall_unix_ns 1793483052363513563
policy_expires_mono_ns 67173923136432
baseline_ok 1
serving 1
generation 5
reason serving
