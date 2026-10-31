# exit-dns

Designated TCP and UDP DNS traverse the selected exit

Status: pass

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §8.2, §10.3, AT12, AT20

- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.99.222.11 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-145394a9ef870ffc1023981d
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-145394a9ef870ffc1023981d
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.99.222.11 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.99.222.11/16 brd 100.99.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.99.0.0/16 dev fvnb0 proto kernel scope link src 100.99.222.11
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.99.222.11 dev fvnb0 table local proto kernel scope host src 100.99.222.11
broadcast 100.99.255.255 dev fvnb0 table local proto kernel scope link src 100.99.222.11
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":762,"packets":78,"bytes":7225}},{"counter":{"family":"inet","name":"nd_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":765,"packets":110,"bytes":8016}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.99.222.11","timeout":3508,"expires":3505}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1167,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1168,"comment":"fvpn_class=01a19024-7d4e-7e1b-933e-cdd77d31be6f:4c3e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1169,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1170,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1171,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1172,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1173,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1174,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1175,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1176,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1177,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1178,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1179,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"counter":"u_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1180,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1182,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1183,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1184,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1185,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"counter":"d_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1188,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 3 flow entries have been shown.
ipv4     2 tcp      6 88 TIME_WAIT src=100.99.222.11 dst=192.0.3.10 sport=55168 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=55168 [ASSURED] mark=0 use=1
ipv4     2 udp      17 119 src=100.99.222.11 dst=192.0.3.10 sport=34667 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=34667 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 119 TIME_WAIT src=100.99.222.11 dst=192.0.3.10 sport=56902 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=56902 [ASSURED] mark=0 use=1
- before_member_tunnel: rx_packets=81
rx_bytes=10368
tx_packets=114
tx_bytes=12804
- before_node_tunnel: rx_packets=114
rx_bytes=12804
tx_packets=83
tx_bytes=10492
- before_node_uplink: rx_packets=121
rx_bytes=12709
tx_packets=138
tx_bytes=12636
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:11:09.513907 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.514062 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.514073 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.514207 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 418
04:11:09.514249 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.514596 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 929
04:11:09.514600 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.515131 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 64
04:11:09.556139 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.556148 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 219
04:11:09.556485 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.557047 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 141
04:11:09.557608 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.558061 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 24
04:11:09.558061 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.558079 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:11:09.513956 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.514015 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.514200 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.514225 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 418
04:11:09.514234 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.514579 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 929
04:11:09.514614 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.515148 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 64
04:11:09.555679 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.556439 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 219
04:11:09.556465 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.556657 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 141
04:11:09.557627 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0
04:11:09.557868 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 24
04:11:09.557882 IP 192.0.3.10.443 > 100.99.222.11.53968: tcp 0
04:11:09.558203 IP 100.99.222.11.53968 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
04:11:09.513986 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 0
04:11:09.514010 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 0
04:11:09.514203 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 0
04:11:09.514228 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 418
04:11:09.514231 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 0
04:11:09.514573 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 929
04:11:09.514617 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 0
04:11:09.515152 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 64
04:11:09.555658 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 0
04:11:09.556451 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 219
04:11:09.556462 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 0
04:11:09.556651 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 141
04:11:09.557637 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 0
04:11:09.557862 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 24
04:11:09.557880 IP 192.0.3.10.443 > 192.0.3.20.53968: tcp 0
04:11:09.558207 IP 192.0.3.20.53968 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.99.222.11 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.99.222.11/16 brd 100.99.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.99.0.0/16 dev fvnb0 proto kernel scope link src 100.99.222.11
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.99.222.11 dev fvnb0 table local proto kernel scope host src 100.99.222.11
broadcast 100.99.255.255 dev fvnb0 table local proto kernel scope link src 100.99.222.11
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":762,"packets":86,"bytes":8743}},{"counter":{"family":"inet","name":"nd_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":765,"packets":118,"bytes":9129}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.99.222.11","timeout":3498,"expires":3497}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1219,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1220,"comment":"fvpn_class=01a19024-7d4e-7e1b-933e-cdd77d31be6f:4c3e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1221,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1222,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1223,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1224,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1225,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1226,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1227,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1228,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1229,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1230,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1231,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"counter":"u_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1232,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1234,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1235,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1236,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1237,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"counter":"d_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1240,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 4 flow entries have been shown.
ipv4     2 tcp      6 79 TIME_WAIT src=100.99.222.11 dst=192.0.3.10 sport=55168 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=55168 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 2 CLOSE src=100.99.222.11 dst=192.0.3.10 sport=53968 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=53968 [ASSURED] mark=0 use=1
ipv4     2 udp      17 110 src=100.99.222.11 dst=192.0.3.10 sport=34667 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=34667 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 110 TIME_WAIT src=100.99.222.11 dst=192.0.3.10 sport=56902 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=56902 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=89
rx_bytes=12224
tx_packets=122
tx_bytes=14244
- after_node_tunnel: rx_packets=122
rx_bytes=14244
tx_packets=91
tx_bytes=12348
- after_node_uplink: rx_packets=129
rx_bytes=14339
tx_packets=146
tx_bytes=13861
- origin_arrival: {"probe_id":"5e49d397-f909-4c79-88e3-7a3f31529e67","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 15241
- admission_bytes_after: 17872
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id b63266b8-5ba5-4729-9b93-722e86fd4c21
mono_ns 3413716537709
wall_unix_ns 1793419876200166808
policy_expires_mono_ns 6912399917523
baseline_ok 1
serving 1
generation 6
reason serving
- udp: {"Transport":"udp","Reply":"rcode=0 answers=1","dns_upload_bytes":"65","dns_download_bytes":"98","internet_upload_bytes":"0","internet_download_bytes":"0"}
- tcp: {"Transport":"tcp","Reply":"rcode=0 answers=1","dns_upload_bytes":"307","dns_download_bytes":"340","internet_upload_bytes":"0","internet_download_bytes":"0"}
- path: origin-network DNS relay -> node filtering resolver; both nu_ and nd_ increased, u_ and d_ held
- native_fallback: not claimed; explicit resolver queries
