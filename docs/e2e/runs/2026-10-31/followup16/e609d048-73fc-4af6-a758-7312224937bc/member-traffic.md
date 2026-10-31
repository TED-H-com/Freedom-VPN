# member-traffic

Member device and exit traffic

Status: pass

Run: e609d048-73fc-4af6-a758-7312224937bc

Source: f771bdb79c0d42b7a3d47090fb52b0d79489d82b

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a1909e-e298-7a3a-8442-acb1fab92b3c
- plan_revision_id: 01a1909e-e299-7587-88b1-d166d2e382e3
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T06:22:40Z
- binding_valid_until: 2026-10-31T07:22:21Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 10.535
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-205af1e6b2c41a9a7096818a
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.115.106.157 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.115.106.157/16 brd 100.115.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.115.0.0/16 dev fvnb0 proto kernel scope link src 100.115.106.157
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.115.106.157 dev fvnb0 table local proto kernel scope host src 100.115.106.157
broadcast 100.115.255.255 dev fvnb0 table local proto kernel scope link src 100.115.106.157
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":126,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":127,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":128,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":129,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":130,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":131,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":132,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":297,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":298,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":299,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":300,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.115.106.157","timeout":3566,"expires":3564}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":357,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":358,"comment":"fvpn_class=01a1909f-9bca-776d-b6f1-2b905f69bc02:4ae6","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.115.106.157"}},{"counter":"u_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":374,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.115.106.157"}},{"counter":"d_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=42
rx_bytes=4471
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
06:22:57.898401 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.005594 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.005611 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.005830 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 418
06:22:58.006688 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.007803 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 929
06:22:58.007809 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.008437 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 64
06:22:58.050282 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.050294 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 219
06:22:58.051886 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.051937 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 141
06:22:58.052704 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.053381 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 24
06:22:58.053381 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.053401 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
06:22:58.004489 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.004586 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.006524 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.006524 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 418
06:22:58.006554 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.007129 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 929
06:22:58.008240 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.008582 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 64
06:22:58.048708 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.051031 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 219
06:22:58.051060 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.051184 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 141
06:22:58.053181 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0
06:22:58.053251 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 24
06:22:58.053269 IP 192.0.3.10.443 > 100.115.106.157.43958: tcp 0
06:22:58.053627 IP 100.115.106.157.43958 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
06:22:58.004564 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 0
06:22:58.004580 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 0
06:22:58.006529 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 0
06:22:58.006531 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 418
06:22:58.006552 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 0
06:22:58.007120 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 929
06:22:58.008250 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 0
06:22:58.008585 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 64
06:22:58.048668 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 0
06:22:58.051044 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 219
06:22:58.051057 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 0
06:22:58.051177 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 141
06:22:58.053194 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 0
06:22:58.053242 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 24
06:22:58.053266 IP 192.0.3.10.443 > 192.0.3.20.43958: tcp 0
06:22:58.053638 IP 192.0.3.20.43958 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.115.106.157 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.115.106.157/16 brd 100.115.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.115.0.0/16 dev fvnb0 proto kernel scope link src 100.115.106.157
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.115.106.157 dev fvnb0 table local proto kernel scope host src 100.115.106.157
broadcast 100.115.255.255 dev fvnb0 table local proto kernel scope link src 100.115.106.157
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":126,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":127,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":128,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":129,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":130,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":131,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":132,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":297,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":298,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":299,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1909fe42976bbb740b6e5f3e41e4b","table":"fvpn","handle":300,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.115.106.157","timeout":3556,"expires":3555}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"comment":"fvpn_class=01a1909f-9bca-776d-b6f1-2b905f69bc02:4ae6","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.115.106.157"}},{"counter":"u_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":422,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":424,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":426,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.115.106.157"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":427,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.115.106.157"}},{"counter":"d_01a1909fe42976bbb740b6e5f3e41e4b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4ae6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":430,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 2 CLOSE src=100.115.106.157 dst=192.0.3.10 sport=43958 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=43958 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=54
rx_bytes=6379
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"7256e576-02c7-4d01-9fb6-b72912af4cee","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2631
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 328d8476-7fb5-4fa6-8a4c-96c35bffd170
mono_ns 11322362010654
wall_unix_ns 1793427784845640505
policy_expires_mono_ns 14879176491408
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
- config_applied: 5
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: passed
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: passed
- check_release_supported: passed
- check_source_binding: passed
- check_storage: passed
- check_tunnel_probe: passed
- stored_lease_matches_applied_and_desired_generation: true
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 7, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:22:54.72832091Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T06:22:44.81306+00:00", "expires_at": "2026-11-01T06:22:44.81306+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- provisioning_ready_at: 2026-10-31T06:21:39.839688Z
- populated_lease_at: 2026-10-31T06:22:24.840371Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 45.001
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T06:22:21.627218447Z","cpu_usage_usec":3843826,"memory_bytes":61194240,"open_fds":84,"processes":11,"conntrack_entries":4},{"at":"2026-10-31T06:22:26.607779781Z","cpu_usage_usec":4407275,"cpu_percent_one_core":11.312961777091129,"memory_bytes":60366848,"open_fds":84,"processes":11,"conntrack_entries":5},{"at":"2026-10-31T06:22:31.564385076Z","cpu_usage_usec":4906624,"cpu_percent_one_core":10.074415255612964,"memory_bytes":61079552,"open_fds":84,"processes":11,"conntrack_entries":9},{"at":"2026-10-31T06:22:36.563219256Z","cpu_usage_usec":5353805,"cpu_percent_one_core":8.94570581655101,"memory_bytes":61968384,"open_fds":84,"processes":11,"conntrack_entries":13},{"at":"2026-10-31T06:22:41.562458455Z","cpu_usage_usec":5781337,"cpu_percent_one_core":8.551941265093285,"memory_bytes":61521920,"open_fds":84,"processes":11,"conntrack_entries":17},{"at":"2026-10-31T06:22:46.563336896Z","cpu_usage_usec":6321423,"cpu_percent_one_core":10.79982259860733,"memory_bytes":63287296,"open_fds":95,"processes":13,"conntrack_entries":17},{"at":"2026-10-31T06:22:51.568018302Z","cpu_usage_usec":6803283,"cpu_percent_one_core":9.628185311103097,"memory_bytes":61689856,"open_fds":84,"processes":11,"conntrack_entries":20},{"at":"2026-10-31T06:22:56.56635764Z","cpu_usage_usec":7312118,"cpu_percent_one_core":10.18008113477949,"memory_bytes":62074880,"open_fds":85,"processes":11,"conntrack_entries":24},{"at":"2026-10-31T06:23:01.573950779Z","cpu_usage_usec":7973446,"cpu_percent_one_core":13.206504235527111,"memory_bytes":66252800,"open_fds":100,"processes":15,"conntrack_entries":35},{"at":"2026-10-31T06:23:06.562624038Z","cpu_usage_usec":8574287,"cpu_percent_one_core":12.044104089519804,"memory_bytes":64024576,"open_fds":84,"processes":11,"conntrack_entries":31},{"at":"2026-10-31T06:23:11.56158644Z","cpu_usage_usec":9029349,"cpu_percent_one_core":9.103129077704954,"memory_bytes":64237568,"open_fds":84,"processes":11,"conntrack_entries":30},{"at":"2026-10-31T06:23:16.563397651Z","cpu_usage_usec":9518100,"cpu_percent_one_core":9.771480357458058,"memory_bytes":64724992,"open_fds":82,"processes":11,"conntrack_entries":34},{"at":"2026-10-31T06:23:21.579057339Z","cpu_usage_usec":10023580,"cpu_percent_one_core":10.078036219430205,"memory_bytes":65818624,"open_fds":82,"processes":11,"conntrack_entries":34},{"at":"2026-10-31T06:23:26.566150591Z","cpu_usage_usec":10533449,"cpu_percent_one_core":10.22377112751049,"memory_bytes":65306624,"open_fds":82,"processes":11,"conntrack_entries":37},{"at":"2026-10-31T06:23:31.564705079Z","cpu_usage_usec":11031666,"cpu_percent_one_core":9.967221547670764,"memory_bytes":65990656,"open_fds":82,"processes":11,"conntrack_entries":29},{"at":"2026-10-31T06:23:36.564729683Z","cpu_usage_usec":11509674,"cpu_percent_one_core":9.560112956596162,"memory_bytes":66039808,"open_fds":82,"processes":11,"conntrack_entries":29}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
