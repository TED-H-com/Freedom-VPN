# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 5282829c-c11e-4f34-954a-5bc2a5bceca5

Source: 5dc018aca482183cb5ca26889bbba54f8db036bc

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
- binding_valid_from: 2026-10-31T14:23:25Z
- binding_valid_until: 2026-10-31T15:23:13Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 1.098
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-5b3bc1f984dec65a27ec0897
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.75.239.94 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.75.239.94/16 brd 100.75.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.75.0.0/16 dev fvnb0 proto kernel scope link src 100.75.239.94
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.75.239.94 dev fvnb0 table local proto kernel scope host src 100.75.239.94
broadcast 100.75.255.255 dev fvnb0 table local proto kernel scope link src 100.75.239.94
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":213,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":214,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":215,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":216,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.75.239.94","timeout":599,"expires":596}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":221,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":222,"comment":"fvpn_class=01a19257-db45-748c-8a71-ce4fdd6ef660:b5a2","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":223,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":224,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":225,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":226,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":227,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":228,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":229,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":230,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":231,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":232,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":233,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.75.239.94"}},{"counter":"u_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":234,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":236,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":237,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":238,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":239,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.75.239.94"}},{"counter":"d_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":242,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=33
rx_bytes=3792
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:23:30.242360 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.349406 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.349439 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.349891 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 418
14:23:30.350969 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.351968 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 930
14:23:30.351984 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.353356 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 64
14:23:30.395081 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.395087 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 219
14:23:30.395825 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.396239 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 141
14:23:30.396824 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.397935 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 24
14:23:30.397935 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.397946 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:23:30.348712 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.348826 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.350275 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.350275 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 418
14:23:30.350351 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.351301 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 930
14:23:30.352493 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.354116 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 64
14:23:30.394671 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.395541 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 219
14:23:30.395558 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.395818 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 141
14:23:30.397522 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0
14:23:30.397556 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 24
14:23:30.397570 IP 192.0.3.10.443 > 100.75.239.94.34650: tcp 0
14:23:30.398025 IP 100.75.239.94.34650 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
14:23:30.348790 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 0
14:23:30.348814 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 0
14:23:30.350297 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 0
14:23:30.350304 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 418
14:23:30.350343 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 0
14:23:30.351284 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 930
14:23:30.352501 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 0
14:23:30.354119 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 64
14:23:30.394649 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 0
14:23:30.395548 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 219
14:23:30.395555 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 0
14:23:30.395812 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 141
14:23:30.397527 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 0
14:23:30.397550 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 24
14:23:30.397568 IP 192.0.3.10.443 > 192.0.3.20.34650: tcp 0
14:23:30.398034 IP 192.0.3.20.34650 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.75.239.94 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.75.239.94/16 brd 100.75.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.75.0.0/16 dev fvnb0 proto kernel scope link src 100.75.239.94
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.75.239.94 dev fvnb0 table local proto kernel scope host src 100.75.239.94
broadcast 100.75.255.255 dev fvnb0 table local proto kernel scope link src 100.75.239.94
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":213,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":214,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":215,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192580aa877fba1f9c2c1fde03908","table":"fvpn","handle":216,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.75.239.94","timeout":589,"expires":588}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":279,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":280,"comment":"fvpn_class=01a19257-db45-748c-8a71-ce4fdd6ef660:b5a2","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":281,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":282,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":283,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":284,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":285,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":286,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":287,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":288,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":290,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":291,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.75.239.94"}},{"counter":"u_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":292,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":294,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":295,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":296,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.75.239.94"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":297,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.75.239.94"}},{"counter":"d_01a192580aa877fba1f9c2c1fde03908"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b5a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":300,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.75.239.94 dst=192.0.3.10 sport=34650 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=34650 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=44
rx_bytes=5614
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"9c5e5229-6fed-44b8-a484-c78b0017c8c6","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 40153971301111
wall_unix_ns 1793456616454929952
policy_expires_mono_ns 40743417511747
baseline_ok 1
serving 1
generation 5
reason serving
