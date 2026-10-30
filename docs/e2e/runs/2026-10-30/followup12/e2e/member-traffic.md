# member-traffic

Member device and exit traffic

Status: pass

Run: 8b97ae7f-ea01-4fba-9a34-930e330684ea

Source: a36bb84f1dafc7526c021177026b4d558e273cd0

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18e40-183d-791e-bf44-d48537bef816
- plan_revision_id: 01a18e40-183d-7be5-b49b-05d5746ffed1
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T19:19:49Z
- binding_valid_until: 2026-10-30T20:19:34Z
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
- exit_discovery_seconds: 10.530
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-f38941dfa39734a3614a597e
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.73.50.215 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.73.50.215/16 brd 100.73.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.73.0.0/16 dev fvnb0 proto kernel scope link src 100.73.50.215
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.73.50.215 dev fvnb0 table local proto kernel scope host src 100.73.50.215
broadcast 100.73.255.255 dev fvnb0 table local proto kernel scope link src 100.73.50.215
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":267,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":268,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":269,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":270,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.73.50.215","timeout":3572,"expires":3570}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"comment":"fvpn_class=01a18e40-cf9f-72e6-92fa-c1f26aae26e7:5b94","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.50.215"}},{"counter":"u_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.50.215"}},{"counter":"d_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":348,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=35
rx_bytes=3918
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
19:20:04.569870 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.674823 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.674850 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.675089 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 418
19:20:04.675750 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.675895 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 931
19:20:04.675904 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.676519 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 64
19:20:04.718531 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.718543 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 219
19:20:04.719175 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.719301 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 141
19:20:04.720087 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.720765 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 24
19:20:04.720765 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.720790 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
19:20:04.674677 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.674787 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.674973 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.675581 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 418
19:20:04.675602 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.675869 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 931
19:20:04.675959 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.676939 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 64
19:20:04.718226 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.719003 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 219
19:20:04.719041 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.719137 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 141
19:20:04.720558 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
19:20:04.720604 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 24
19:20:04.720619 IP 192.0.3.10.443 > 100.73.50.215.53204: tcp 0
19:20:04.721330 IP 100.73.50.215.53204 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
19:20:04.674758 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 0
19:20:04.674778 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 0
19:20:04.674980 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 0
19:20:04.675589 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 418
19:20:04.675599 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 0
19:20:04.675857 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 931
19:20:04.675966 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 0
19:20:04.676946 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 64
19:20:04.718194 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 0
19:20:04.719019 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 219
19:20:04.719037 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 0
19:20:04.719130 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 141
19:20:04.720571 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 0
19:20:04.720600 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 24
19:20:04.720616 IP 192.0.3.10.443 > 192.0.3.20.53204: tcp 0
19:20:04.721354 IP 192.0.3.20.53204 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.73.50.215 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.73.50.215/16 brd 100.73.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.73.0.0/16 dev fvnb0 proto kernel scope link src 100.73.50.215
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.73.50.215 dev fvnb0 table local proto kernel scope host src 100.73.50.215
broadcast 100.73.255.255 dev fvnb0 table local proto kernel scope link src 100.73.50.215
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":267,"packets":8,"bytes":1520}},{"counter":{"family":"inet","name":"nd_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":268,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":269,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18e41084b7ed5a247537e1ae54c85","table":"fvpn","handle":270,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.73.50.215","timeout":3567,"expires":3561}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":353,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":354,"comment":"fvpn_class=01a18e40-cf9f-72e6-92fa-c1f26aae26e7:5b94","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":356,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":357,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":358,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.50.215"}},{"counter":"u_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.50.215"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.50.215"}},{"counter":"d_01a18e41084b7ed5a247537e1ae54c85"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5b94"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":374,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.73.50.215 dst=192.0.3.10 sport=53204 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=53204 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=46
rx_bytes=5704
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"ae4b10fa-a32d-4ff7-91d9-0a91cdc13b9c","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2633
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 30f2013b-2375-4e2a-bdd4-07b9ba1575a1
mono_ns 743050952177244
wall_unix_ns 1793388011755355938
policy_expires_mono_ns 746613763440463
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 4, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:20:01.588938479Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T19:19:51.436484+00:00", "expires_at": "2026-10-31T19:19:51.436484+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-30T19:19:34.505985235Z","cpu_usage_usec":1009735,"memory_bytes":56442880,"open_fds":86,"processes":10,"conntrack_entries":134},{"at":"2026-10-30T19:19:39.512949481Z","cpu_usage_usec":1120750,"cpu_percent_one_core":2.217211758376115,"memory_bytes":57237504,"open_fds":86,"processes":10,"conntrack_entries":141},{"at":"2026-10-30T19:19:44.507298713Z","cpu_usage_usec":1176851,"cpu_percent_one_core":1.1232894896605818,"memory_bytes":56705024,"open_fds":86,"processes":10,"conntrack_entries":72},{"at":"2026-10-30T19:19:49.512373083Z","cpu_usage_usec":1238259,"cpu_percent_one_core":1.2269148360326962,"memory_bytes":57749504,"open_fds":86,"processes":10,"conntrack_entries":79},{"at":"2026-10-30T19:19:54.510526572Z","cpu_usage_usec":1372126,"cpu_percent_one_core":2.678329112833694,"memory_bytes":54927360,"open_fds":86,"processes":10,"conntrack_entries":87},{"at":"2026-10-30T19:19:59.509223387Z","cpu_usage_usec":1436551,"cpu_percent_one_core":1.288835918327245,"memory_bytes":58748928,"open_fds":86,"processes":10,"conntrack_entries":91},{"at":"2026-10-30T19:20:04.508601812Z","cpu_usage_usec":1575880,"cpu_percent_one_core":2.7869264567624725,"memory_bytes":61403136,"open_fds":103,"processes":14,"conntrack_entries":63},{"at":"2026-10-30T19:20:09.513566334Z","cpu_usage_usec":1655013,"cpu_percent_one_core":1.5810901286544623,"memory_bytes":61067264,"open_fds":102,"processes":14,"conntrack_entries":79},{"at":"2026-10-30T19:20:14.507444982Z","cpu_usage_usec":1767814,"cpu_percent_one_core":2.2587853640611733,"memory_bytes":60100608,"open_fds":85,"processes":10,"conntrack_entries":70},{"at":"2026-10-30T19:20:19.508923525Z","cpu_usage_usec":1835967,"cpu_percent_one_core":1.3626570505912896,"memory_bytes":61100032,"open_fds":86,"processes":10,"conntrack_entries":80},{"at":"2026-10-30T19:20:24.508484049Z","cpu_usage_usec":1899866,"cpu_percent_one_core":1.2780923381816829,"memory_bytes":60948480,"open_fds":83,"processes":10,"conntrack_entries":67},{"at":"2026-10-30T19:20:29.511449557Z","cpu_usage_usec":1964511,"cpu_percent_one_core":1.2921336334745723,"memory_bytes":62169088,"open_fds":82,"processes":10,"conntrack_entries":74},{"at":"2026-10-30T19:20:34.510801746Z","cpu_usage_usec":2028276,"cpu_percent_one_core":1.2754652520840837,"memory_bytes":61652992,"open_fds":81,"processes":10,"conntrack_entries":78},{"at":"2026-10-30T19:20:39.508312383Z","cpu_usage_usec":2092764,"cpu_percent_one_core":1.290402456025828,"memory_bytes":62009344,"open_fds":81,"processes":10,"conntrack_entries":82},{"at":"2026-10-30T19:20:44.509089341Z","cpu_usage_usec":2160304,"cpu_percent_one_core":1.3505901296388112,"memory_bytes":62844928,"open_fds":74,"processes":10,"conntrack_entries":55},{"at":"2026-10-30T19:20:49.52310469Z","cpu_usage_usec":2227637,"cpu_percent_one_core":1.3428957694241794,"memory_bytes":64393216,"open_fds":73,"processes":10,"conntrack_entries":62},{"at":"2026-10-30T19:20:54.506895765Z","cpu_usage_usec":2296572,"cpu_percent_one_core":1.3831839850951215,"memory_bytes":62210048,"open_fds":73,"processes":10,"conntrack_entries":66}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
