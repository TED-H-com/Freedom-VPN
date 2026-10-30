# member-traffic

Member device and exit traffic

Status: pass

Run: 273496b3-9897-43b0-b289-a571d1d14561

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18c4d-18de-7e09-abf8-3cb95bd3fccd
- plan_revision_id: 01a18c4d-18df-706b-9602-cb13a80c0462
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T10:15:18Z
- binding_valid_until: 2026-10-30T11:15:11Z
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

  - ID: fvpn-exit-d0a27efe1385622f38ae9881
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-d0a27efe1385622f38ae9881
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.95.176.119 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.95.176.119/16 brd 100.95.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.95.0.0/16 dev fvnb0 proto kernel scope link src 100.95.176.119
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.95.176.119 dev fvnb0 table local proto kernel scope host src 100.95.176.119
broadcast 100.95.255.255 dev fvnb0 table local proto kernel scope link src 100.95.176.119
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":358,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":359,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":360,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":361,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.95.176.119","timeout":3588,"expires":3585}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"comment":"fvpn_class=01a18c4e-6a0f-76b9-a0d5-742e852df12e:186c","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":374,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.95.176.119"}},{"counter":"u_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":379,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":382,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":383,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.95.176.119"}},{"counter":"d_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":387,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=3923
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:15:26.997438 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.005642 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.005661 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.005971 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 418
10:15:28.006620 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.007122 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 930
10:15:28.007126 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.007811 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 64
10:15:28.049403 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.049426 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 219
10:15:28.050040 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.050040 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 141
10:15:28.050802 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.051543 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 24
10:15:28.051543 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.051555 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
10:15:28.005530 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.005618 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.005788 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.006427 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 418
10:15:28.006438 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.007012 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 930
10:15:28.007602 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.008123 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 64
10:15:28.049229 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.049731 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 219
10:15:28.049756 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.049892 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 141
10:15:28.051461 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
10:15:28.051516 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 24
10:15:28.051528 IP 192.0.3.10.443 > 100.95.176.119.43512: tcp 0
10:15:28.051836 IP 100.95.176.119.43512 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
10:15:28.005593 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 0
10:15:28.005610 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 0
10:15:28.005792 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 0
10:15:28.006430 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 418
10:15:28.006436 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 0
10:15:28.007004 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 930
10:15:28.007605 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 0
10:15:28.008129 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 64
10:15:28.049203 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 0
10:15:28.049740 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 219
10:15:28.049750 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 0
10:15:28.049886 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 141
10:15:28.051469 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 0
10:15:28.051510 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 24
10:15:28.051525 IP 192.0.3.10.443 > 192.0.3.20.43512: tcp 0
10:15:28.051843 IP 192.0.3.20.43512 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.95.176.119 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.95.176.119/16 brd 100.95.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.95.0.0/16 dev fvnb0 proto kernel scope link src 100.95.176.119
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.95.176.119 dev fvnb0 table local proto kernel scope host src 100.95.176.119
broadcast 100.95.255.255 dev fvnb0 table local proto kernel scope link src 100.95.176.119
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":358,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":359,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":360,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18c4e85677427a80bdb58e1f4013f","table":"fvpn","handle":361,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.95.176.119","timeout":3577,"expires":3576}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":451,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":452,"comment":"fvpn_class=01a18c4e-6a0f-76b9-a0d5-742e852df12e:186c","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":453,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":454,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":455,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":456,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":457,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":458,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":459,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":460,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":461,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":462,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":463,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.95.176.119"}},{"counter":"u_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":464,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":466,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":467,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":468,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.95.176.119"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":469,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.95.176.119"}},{"counter":"d_01a18c4e85677427a80bdb58e1f4013f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:186c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":472,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 3 CLOSE src=100.95.176.119 dst=192.0.3.10 sport=43512 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=43512 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=48
rx_bytes=5832
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"68937531-4e8c-4d21-9d7d-961fac7c05da","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 2fff4f99-0aef-4bbb-83e4-4946536f4dec
mono_ns 710372884370758
wall_unix_ns 1793355333687549615
policy_expires_mono_ns 713950832813135
baseline_ok 1
serving 1
generation 6
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
- config_desired: 6
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 6, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:15:38.723188455Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T10:15:33.624058+00:00", "expires_at": "2026-10-31T10:15:33.624058+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
