# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 8576dfe5-aa76-417c-9c12-f104cef47fdb

Source: 38d10b3b450b41a67ae5223d05e82c05c659c429

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
- binding_valid_from: 2026-11-03T06:51:51Z
- binding_valid_until: 2026-11-03T07:51:35Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-9c0455746af9085a477b5de2
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-9c0455746af9085a477b5de2
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.88.216.12 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.88.216.12/16 brd 100.88.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.88.0.0/16 dev fvnb0 proto kernel scope link src 100.88.216.12
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.88.216.12 dev fvnb0 table local proto kernel scope host src 100.88.216.12
broadcast 100.88.255.255 dev fvnb0 table local proto kernel scope link src 100.88.216.12
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":78,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":248,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":251,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.88.216.12","timeout":598,"expires":596}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":256,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":257,"comment":"fvpn_class=01a1a02d-7200-7f22-b4ba-9e1c79ce69ee:e42a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":258,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":259,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":260,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":261,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":262,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":263,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":264,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":265,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":266,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":267,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":268,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.88.216.12"}},{"counter":"u_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":269,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":271,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":272,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":273,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":274,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":275,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":276,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":277,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.88.216.12"}},{"counter":"d_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":280,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=37
rx_bytes=4196
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
06:51:55.829497 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.034409 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.034428 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.034617 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 418
06:51:56.035197 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.035611 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 930
06:51:56.035618 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.036194 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 64
06:51:56.077140 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.077149 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 219
06:51:56.077945 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.077946 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 141
06:51:56.078517 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.079330 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 24
06:51:56.079330 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.079346 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
06:51:56.033920 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.034010 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.034565 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.034732 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 418
06:51:56.034743 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.035121 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 930
06:51:56.036104 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.036276 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 64
06:51:56.076718 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.077552 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 219
06:51:56.077574 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.077638 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 141
06:51:56.079084 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
06:51:56.079111 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 24
06:51:56.079121 IP 192.0.3.10.443 > 100.88.216.12.52852: tcp 0
06:51:56.079572 IP 100.88.216.12.52852 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
06:51:56.033987 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 0
06:51:56.034003 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 0
06:51:56.034568 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 0
06:51:56.034735 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 418
06:51:56.034740 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 0
06:51:56.035114 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 930
06:51:56.036106 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 0
06:51:56.036278 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 64
06:51:56.076705 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 0
06:51:56.077561 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 219
06:51:56.077570 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 0
06:51:56.077633 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 141
06:51:56.079091 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 0
06:51:56.079107 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 24
06:51:56.079119 IP 192.0.3.10.443 > 192.0.3.20.52852: tcp 0
06:51:56.079579 IP 192.0.3.20.52852 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.88.216.12 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.88.216.12/16 brd 100.88.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.88.0.0/16 dev fvnb0 proto kernel scope link src 100.88.216.12
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.88.216.12 dev fvnb0 table local proto kernel scope host src 100.88.216.12
broadcast 100.88.255.255 dev fvnb0 table local proto kernel scope link src 100.88.216.12
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":78,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":248,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a02db10f783ca690dabd41bbd19b","table":"fvpn","handle":251,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.88.216.12","timeout":588,"expires":588}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":320,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":321,"comment":"fvpn_class=01a1a02d-7200-7f22-b4ba-9e1c79ce69ee:e42a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.88.216.12"}},{"counter":"u_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.88.216.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.88.216.12"}},{"counter":"d_01a1a02db10f783ca690dabd41bbd19b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e42a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 3 CLOSE src=100.88.216.12 dst=192.0.3.10 sport=52852 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=52852 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=47
rx_bytes=5976
tx_packets=23
tx_bytes=2807
- origin_arrival: {"probe_id":"ec4d104d-fb98-46c2-9952-ed7fe45673d6","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 12165501369429
wall_unix_ns 1793688723049992927
policy_expires_mono_ns 12753920561608
baseline_ok 1
serving 1
generation 5
reason serving
