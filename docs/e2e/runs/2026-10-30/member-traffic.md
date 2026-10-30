# member-traffic

Member device and exit traffic

Status: pass

Run: 22c14cd9-39fc-444b-a7c5-c04cf3ffbff6

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18bc3-3d38-7480-a5ef-b0f6a3ea3727
- plan_revision_id: 01a18bc3-3d38-764c-87f8-e1d5e5f97bdc
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T07:44:53Z
- binding_valid_until: 2026-10-30T08:44:38Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-19d4da50367ad609033ca17c
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-19d4da50367ad609033ca17c
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.104.75.121 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.104.75.121/16 brd 100.104.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.104.0.0/16 dev fvnb0 proto kernel scope link src 100.104.75.121
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.104.75.121 dev fvnb0 table local proto kernel scope host src 100.104.75.121
broadcast 100.104.255.255 dev fvnb0 table local proto kernel scope link src 100.104.75.121
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":355,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":356,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":357,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":358,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.104.75.121","timeout":3583,"expires":3581}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"comment":"fvpn_class=01a18bc4-945c-7d07-991b-e8b0ba1a9c30:9971","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":374,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.104.75.121"}},{"counter":"u_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":379,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":380,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.104.75.121"}},{"counter":"d_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=36
rx_bytes=3960
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
07:44:57.933673 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.040824 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.040848 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.041138 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 418
07:44:58.041596 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.041949 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 929
07:44:58.041953 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.042654 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 64
07:44:58.084009 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.084020 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 219
07:44:58.084643 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.085075 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 141
07:44:58.086140 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.087153 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 24
07:44:58.087154 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.087167 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
07:44:58.040468 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.040543 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.040963 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.041418 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 418
07:44:58.041444 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.041848 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 929
07:44:58.042106 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.042802 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 64
07:44:58.083445 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.084233 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 219
07:44:58.084265 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.084455 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 141
07:44:58.086581 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0
07:44:58.086767 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 24
07:44:58.086783 IP 192.0.3.10.443 > 100.104.75.121.56150: tcp 0
07:44:58.087590 IP 100.104.75.121.56150 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
07:44:58.040523 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 0
07:44:58.040537 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 0
07:44:58.040974 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 0
07:44:58.041429 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 418
07:44:58.041440 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 0
07:44:58.041840 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 929
07:44:58.042118 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 0
07:44:58.042813 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 64
07:44:58.083434 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 0
07:44:58.084247 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 219
07:44:58.084260 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 0
07:44:58.084445 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 141
07:44:58.086594 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 0
07:44:58.086756 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 24
07:44:58.086780 IP 192.0.3.10.443 > 192.0.3.20.56150: tcp 0
07:44:58.087604 IP 192.0.3.20.56150 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.104.75.121 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.104.75.121/16 brd 100.104.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.104.0.0/16 dev fvnb0 proto kernel scope link src 100.104.75.121
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.104.75.121 dev fvnb0 table local proto kernel scope host src 100.104.75.121
broadcast 100.104.255.255 dev fvnb0 table local proto kernel scope link src 100.104.75.121
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":355,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":356,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":357,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18bc4cfe87c4c89c889b37cfec3f7","table":"fvpn","handle":358,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.104.75.121","timeout":3578,"expires":3573}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":390,"comment":"fvpn_class=01a18bc4-945c-7d07-991b-e8b0ba1a9c30:9971","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":391,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":392,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":393,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":394,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":395,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":396,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":397,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":398,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":399,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":400,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":401,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.104.75.121"}},{"counter":"u_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":402,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":404,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":405,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":406,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.104.75.121"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":407,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.104.75.121"}},{"counter":"d_01a18bc4cfe87c4c89c889b37cfec3f7"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:9971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.104.75.121 dst=192.0.3.10 sport=56150 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=56150 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=48
rx_bytes=5851
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"ef9c25a6-39c0-40c6-a863-08b226040d2f","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2631
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 83f3da3a-ee81-4d33-963d-a7d5c7c8ff33
mono_ns 701339583719785
wall_unix_ns 1793346300386898649
policy_expires_mono_ns 704917695273311
baseline_ok 1
serving 1
generation 5
reason serving
- persistent_boot_recovery_config_owner_mode: 0:600
- admin_node: 200
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: ready
- lifecycle_state: registered
- publication_intent: published
- publication_observed: active
- config_desired: 5
- config_applied: 3
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: not_run
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: not_run
- check_release_supported: passed
- check_source_binding: not_run
- check_storage: passed
- check_tunnel_probe: passed
- stored_lease_matches_applied_and_desired_generation: true
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 6, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:45:05.504384813Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T07:45:05.482819+00:00", "expires_at": "2026-10-31T07:45:05.482819+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
