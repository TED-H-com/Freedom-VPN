# member-traffic

Member device and exit traffic

Status: pass

Run: fa5650f3-d238-496e-b7d4-718ce705fefb

Source: a36bb84f1dafc7526c021177026b4d558e273cd0

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18e45-f72a-783a-b03a-a66a1fc13021
- plan_revision_id: 01a18e45-f72a-7b36-a628-7f57bcaf92ba
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T19:26:07Z
- binding_valid_until: 2026-10-30T20:26:00Z
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
- exit_discovery_seconds: 19.961
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-083ebce94ff75e42bbf00eea
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.122.204.105 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.122.204.105/16 brd 100.122.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.122.0.0/16 dev fvnb0 proto kernel scope link src 100.122.204.105
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.122.204.105 dev fvnb0 table local proto kernel scope host src 100.122.204.105
broadcast 100.122.255.255 dev fvnb0 table local proto kernel scope link src 100.122.204.105
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":248,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":251,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.122.204.105","timeout":3568,"expires":3567}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"comment":"fvpn_class=01a18e46-b1a8-7899-9ba3-1abb1ad88d00:115e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.204.105"}},{"counter":"u_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.204.105"}},{"counter":"d_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=3878
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
19:26:33.749275 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.758290 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.758309 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.758592 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 418
19:26:34.759123 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.759605 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 928
19:26:34.759612 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.760256 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 64
19:26:34.801481 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.801489 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 219
19:26:34.802224 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.802224 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 141
19:26:34.802803 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.803772 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 24
19:26:34.803772 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.803784 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
19:26:34.757930 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.758013 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.758443 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.758953 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 418
19:26:34.758965 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.759575 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 928
19:26:34.759712 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.760560 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 64
19:26:34.801336 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.801909 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 219
19:26:34.801971 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.802044 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 141
19:26:34.803540 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0
19:26:34.803569 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 24
19:26:34.803580 IP 192.0.3.10.443 > 100.122.204.105.43320: tcp 0
19:26:34.804267 IP 100.122.204.105.43320 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
19:26:34.757992 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 0
19:26:34.758007 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 0
19:26:34.758446 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 0
19:26:34.758956 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 418
19:26:34.758962 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 0
19:26:34.759565 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 928
19:26:34.759714 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 0
19:26:34.760562 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 64
19:26:34.801325 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 0
19:26:34.801936 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 219
19:26:34.801961 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 0
19:26:34.802038 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 141
19:26:34.803547 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 0
19:26:34.803566 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 24
19:26:34.803578 IP 192.0.3.10.443 > 192.0.3.20.43320: tcp 0
19:26:34.804275 IP 192.0.3.20.43320 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.122.204.105 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.122.204.105/16 brd 100.122.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.122.0.0/16 dev fvnb0 proto kernel scope link src 100.122.204.105
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.122.204.105 dev fvnb0 table local proto kernel scope host src 100.122.204.105
broadcast 100.122.255.255 dev fvnb0 table local proto kernel scope link src 100.122.204.105
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":248,"packets":8,"bytes":1517}},{"counter":{"family":"inet","name":"nd_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18e46cf0c7627abfba3064f9422cc","table":"fvpn","handle":251,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.122.204.105","timeout":3563,"expires":3559}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":386,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":387,"comment":"fvpn_class=01a18e46-b1a8-7899-9ba3-1abb1ad88d00:115e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":388,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":390,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":391,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":392,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":393,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":394,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":395,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":396,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":397,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":398,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.204.105"}},{"counter":"u_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":399,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":401,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":402,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":403,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.204.105"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":404,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.204.105"}},{"counter":"d_01a18e46cf0c7627abfba3064f9422cc"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:115e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":407,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 3 CLOSE src=100.122.204.105 dst=192.0.3.10 sport=43320 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=43320 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=46
rx_bytes=5591
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"8f6f4076-aaef-4ada-9e98-94b69078d0f3","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2630
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id b7aee795-db52-49a0-8e61-bdbefc5d2055
mono_ns 743436254368827
wall_unix_ns 1793388397057547629
policy_expires_mono_ns 746999306903877
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 4, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:26:26.896810319Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T19:26:11.674103+00:00", "expires_at": "2026-10-31T19:26:11.674103+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-30T19:26:00.059508029Z","cpu_usage_usec":1005980,"memory_bytes":54439936,"open_fds":86,"processes":10,"conntrack_entries":127},{"at":"2026-10-30T19:26:05.061133739Z","cpu_usage_usec":1117213,"cpu_percent_one_core":2.22393690470693,"memory_bytes":55468032,"open_fds":86,"processes":10,"conntrack_entries":134},{"at":"2026-10-30T19:26:10.060372138Z","cpu_usage_usec":1176744,"cpu_percent_one_core":1.1908013831048347,"memory_bytes":56455168,"open_fds":86,"processes":10,"conntrack_entries":77},{"at":"2026-10-30T19:26:15.061720034Z","cpu_usage_usec":1304633,"cpu_percent_one_core":2.557090661545133,"memory_bytes":54927360,"open_fds":86,"processes":10,"conntrack_entries":77},{"at":"2026-10-30T19:26:20.060986032Z","cpu_usage_usec":1365623,"cpu_percent_one_core":1.2199790934189054,"memory_bytes":56885248,"open_fds":86,"processes":10,"conntrack_entries":84},{"at":"2026-10-30T19:26:25.059788304Z","cpu_usage_usec":1426149,"cpu_percent_one_core":1.2108100442185286,"memory_bytes":55443456,"open_fds":86,"processes":10,"conntrack_entries":78},{"at":"2026-10-30T19:26:30.059263689Z","cpu_usage_usec":1487067,"cpu_percent_one_core":1.2184878474004128,"memory_bytes":54796288,"open_fds":86,"processes":10,"conntrack_entries":82},{"at":"2026-10-30T19:26:35.060439761Z","cpu_usage_usec":1641608,"cpu_percent_one_core":3.0900931655901123,"memory_bytes":61726720,"open_fds":104,"processes":14,"conntrack_entries":72},{"at":"2026-10-30T19:26:40.063124132Z","cpu_usage_usec":1704767,"cpu_percent_one_core":1.262502195143984,"memory_bytes":61374464,"open_fds":104,"processes":14,"conntrack_entries":79},{"at":"2026-10-30T19:26:45.060466485Z","cpu_usage_usec":1823382,"cpu_percent_one_core":2.373561617782563,"memory_bytes":59662336,"open_fds":85,"processes":10,"conntrack_entries":79},{"at":"2026-10-30T19:26:50.06777475Z","cpu_usage_usec":1885235,"cpu_percent_one_core":1.2352544865739357,"memory_bytes":59400192,"open_fds":84,"processes":10,"conntrack_entries":83},{"at":"2026-10-30T19:26:55.060157788Z","cpu_usage_usec":1948223,"cpu_percent_one_core":1.2616820368261177,"memory_bytes":59502592,"open_fds":83,"processes":10,"conntrack_entries":70},{"at":"2026-10-30T19:27:00.058827426Z","cpu_usage_usec":2007927,"cpu_percent_one_core":1.194397796288213,"memory_bytes":60461056,"open_fds":82,"processes":10,"conntrack_entries":77},{"at":"2026-10-30T19:27:05.063736802Z","cpu_usage_usec":2074942,"cpu_percent_one_core":1.3389852835569107,"memory_bytes":61460480,"open_fds":83,"processes":10,"conntrack_entries":87},{"at":"2026-10-30T19:27:10.061861277Z","cpu_usage_usec":2133996,"cpu_percent_one_core":1.1815231952581573,"memory_bytes":61235200,"open_fds":83,"processes":10,"conntrack_entries":66},{"at":"2026-10-30T19:27:15.061743352Z","cpu_usage_usec":2198104,"cpu_percent_one_core":1.282190240456821,"memory_bytes":60940288,"open_fds":82,"processes":10,"conntrack_entries":70},{"at":"2026-10-30T19:27:20.060645502Z","cpu_usage_usec":2267925,"cpu_percent_one_core":1.3967266792769688,"memory_bytes":62578688,"open_fds":74,"processes":10,"conntrack_entries":59}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
