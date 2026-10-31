# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: abd7ce3a-7b3f-4fa3-8120-c11635971e2c

Source: 0b9fb5ebbff0d1618679b8eed6925b7aa9cc7daf

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
- binding_valid_from: 2026-10-31T10:17:00Z
- binding_valid_until: 2026-10-31T11:16:53Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-f8d496cea714b8e5e8752136
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-f8d496cea714b8e5e8752136
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.119.146.150 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.119.146.150/16 brd 100.119.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.119.0.0/16 dev fvnb0 proto kernel scope link src 100.119.146.150
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.119.146.150 dev fvnb0 table local proto kernel scope host src 100.119.146.150
broadcast 100.119.255.255 dev fvnb0 table local proto kernel scope link src 100.119.146.150
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":170,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":171,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":172,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":173,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":174,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":175,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":176,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":290,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":291,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":292,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":293,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.119.146.150","timeout":3589,"expires":3588}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":298,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":299,"comment":"fvpn_class=01a19176-52a3-75bf-a381-1aa55b6b803f:b492","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":300,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":301,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":302,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":303,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":304,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":305,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":306,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":307,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":308,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":309,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":310,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.146.150"}},{"counter":"u_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":311,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":313,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":314,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":315,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.146.150"}},{"counter":"d_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=24
rx_bytes=3231
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:17:04.817251 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.821286 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.821296 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.821460 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 418
10:17:04.821738 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.821928 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 931
10:17:04.821931 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.822427 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 64
10:17:04.863258 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.863268 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 219
10:17:04.864331 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.864950 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 141
10:17:04.865618 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.866311 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 24
10:17:04.866312 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.866329 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:17:04.821083 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.821159 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.821420 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.821591 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 418
10:17:04.821609 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.821807 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 931
10:17:04.822020 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.822531 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 64
10:17:04.862662 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.863830 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 219
10:17:04.863858 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.863969 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 141
10:17:04.865790 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0
10:17:04.865925 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 24
10:17:04.865942 IP 192.0.3.10.443 > 100.119.146.150.58402: tcp 0
10:17:04.866464 IP 100.119.146.150.58402 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
10:17:04.821143 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 0
10:17:04.821154 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 0
10:17:04.821432 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 0
10:17:04.821598 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 418
10:17:04.821605 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 0
10:17:04.821801 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 931
10:17:04.822022 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 0
10:17:04.822533 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 64
10:17:04.862644 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 0
10:17:04.863844 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 219
10:17:04.863854 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 0
10:17:04.863964 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 141
10:17:04.865804 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 0
10:17:04.865918 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 24
10:17:04.865939 IP 192.0.3.10.443 > 192.0.3.24.58402: tcp 0
10:17:04.866470 IP 192.0.3.24.58402 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.119.146.150 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.119.146.150/16 brd 100.119.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.119.0.0/16 dev fvnb0 proto kernel scope link src 100.119.146.150
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.119.146.150 dev fvnb0 table local proto kernel scope host src 100.119.146.150
broadcast 100.119.255.255 dev fvnb0 table local proto kernel scope link src 100.119.146.150
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":170,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":171,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":172,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":173,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":174,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":175,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":176,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":290,"packets":8,"bytes":1520}},{"counter":{"family":"inet","name":"nd_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":291,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":292,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a191766ee9742e9f4c47e812bd7e29","table":"fvpn","handle":293,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.119.146.150","timeout":3585,"expires":3580}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"comment":"fvpn_class=01a19176-52a3-75bf-a381-1aa55b6b803f:b492","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.146.150"}},{"counter":"u_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.146.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.146.150"}},{"counter":"d_01a191766ee9742e9f4c47e812bd7e29"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b492"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":348,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.119.146.150 dst=192.0.3.10 sport=58402 dport=443 src=192.0.3.10 dst=192.0.3.24 sport=443 dport=58402 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=34
rx_bytes=4947
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"b1f9165b-c9c1-43f3-a732-f782b09c3040","requests":1,"source_address":"192.0.3.24"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2633
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 25369736545569
wall_unix_ns 1793441832220174548
policy_expires_mono_ns 28950674318714
baseline_ok 1
serving 1
generation 5
reason serving
