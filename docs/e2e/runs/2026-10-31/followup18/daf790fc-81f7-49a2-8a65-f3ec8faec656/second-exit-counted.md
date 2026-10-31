# second-exit-counted

Serving second exit before namespace fault

Status: pass

Run: daf790fc-81f7-49a2-8a65-f3ec8faec656

Source: fc570997ad08facaa963e32d2abb6fdabec8d9fa

Spec: AT09, F8-01

- selected_exit: Networks selected successfully.
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.67.102.246 uid 0
    cache
- before_member_tunnel_address: 10: fvnb0    inet 100.67.102.246/16 brd 100.67.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.67.0.0/16 dev fvnb0 proto kernel scope link src 100.67.102.246
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.67.102.246 dev fvnb0 table local proto kernel scope host src 100.67.102.246
broadcast 100.67.255.255 dev fvnb0 table local proto kernel scope link src 100.67.102.246
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":100,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":101,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":102,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":103,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.67.102.246","timeout":3438,"expires":3434}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":111,"comment":"fvpn_generation=3 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":112,"comment":"fvpn_class=01a1918a-99ad-73fb-9dcc-f5308ad66d65:638a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":113,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":114,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":115,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":116,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":117,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":118,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":119,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":120,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":121,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":122,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":123,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"counter":"u_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":124,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":126,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":127,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":128,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":129,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"counter":"d_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":132,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.22"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=80
rx_bytes=74484
tx_packets=91
tx_bytes=72096
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=19
rx_bytes=2741
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:41:47.660405 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.664803 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.664816 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.665037 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 418
10:41:47.665316 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.665492 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 931
10:41:47.665496 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.666159 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 64
10:41:47.707971 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.707985 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 219
10:41:47.709071 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.709154 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 141
10:41:47.710106 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.711101 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 24
10:41:47.711101 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.711123 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:41:47.664537 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.664637 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.664937 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.665184 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 418
10:41:47.665196 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.665404 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 931
10:41:47.665579 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.666273 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 64
10:41:47.706679 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.708593 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 219
10:41:47.708626 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.708739 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 141
10:41:47.710544 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0
10:41:47.710594 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 24
10:41:47.710608 IP 192.0.3.10.443 > 100.67.102.246.60886: tcp 0
10:41:47.711248 IP 100.67.102.246.60886 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
10:41:47.664612 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 0
10:41:47.664626 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 0
10:41:47.664950 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 0
10:41:47.665186 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 418
10:41:47.665191 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 0
10:41:47.665398 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 931
10:41:47.665580 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 0
10:41:47.666277 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 64
10:41:47.706656 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 0
10:41:47.708610 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 219
10:41:47.708622 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 0
10:41:47.708729 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 141
10:41:47.710555 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 0
10:41:47.710588 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 24
10:41:47.710606 IP 192.0.3.10.443 > 192.0.3.22.60886: tcp 0
10:41:47.711253 IP 192.0.3.22.60886 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.67.102.246 uid 0
    cache
- after_member_tunnel_address: 10: fvnb0    inet 100.67.102.246/16 brd 100.67.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.67.0.0/16 dev fvnb0 proto kernel scope link src 100.67.102.246
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.67.102.246 dev fvnb0 table local proto kernel scope host src 100.67.102.246
broadcast 100.67.255.255 dev fvnb0 table local proto kernel scope link src 100.67.102.246
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":100,"packets":8,"bytes":1520}},{"counter":{"family":"inet","name":"nd_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":101,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":102,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":103,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.67.102.246","timeout":3428,"expires":3426}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":169,"comment":"fvpn_generation=3 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":170,"comment":"fvpn_class=01a1918a-99ad-73fb-9dcc-f5308ad66d65:638a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":171,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":172,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":173,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":174,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":175,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":176,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":177,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":178,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":179,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":180,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":181,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"counter":"u_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":182,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":184,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":185,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":186,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":187,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"counter":"d_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":190,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.22"}}]}}]
- after_conntrack: ipv4     2 tcp      6 2 CLOSE src=100.67.102.246 dst=192.0.3.10 sport=60886 dport=443 src=192.0.3.10 dst=192.0.3.22 sport=443 dport=60886 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=91
rx_bytes=76552
tx_packets=102
tx_bytes=73808
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=29
rx_bytes=4522
tx_packets=23
tx_bytes=2807
- origin_arrival: {"probe_id":"631a3882-4674-44fe-99cc-ecaeb3862745","requests":1,"source_address":"192.0.3.22"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2633
- first_stopped_hop: none after explicit NetBird exit selection
