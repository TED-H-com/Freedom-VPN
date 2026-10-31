# forced-relay

Member forced onto the relay path and restored to direct

Status: pass

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §8, AT31

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
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":762,"packets":14,"bytes":1874}},{"counter":{"family":"inet","name":"nd_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":765,"packets":18,"bytes":1703}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.99.222.11","timeout":3540,"expires":3535}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1011,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1012,"comment":"fvpn_class=01a19024-7d4e-7e1b-933e-cdd77d31be6f:4c3e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1013,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1014,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1015,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1016,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1017,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1018,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1019,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1020,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1021,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1022,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1023,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"counter":"u_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1024,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1026,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1027,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1028,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1029,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"counter":"d_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1032,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 tcp      6 117 TIME_WAIT src=100.99.222.11 dst=192.0.3.10 sport=55168 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=55168 [ASSURED] mark=0 use=1
ipv4     2 udp      17 27 src=100.99.222.11 dst=192.0.3.10 sport=41998 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=41998 mark=0 use=1
- before_member_tunnel: rx_packets=17
rx_bytes=2720
tx_packets=22
tx_bytes=2852
- before_node_tunnel: rx_packets=22
rx_bytes=2852
tx_packets=18
tx_bytes=2812
- before_node_uplink: rx_packets=56
rx_bytes=6420
tx_packets=45
tx_bytes=4993
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:10:40.411211 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.411348 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.411357 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.411559 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 418
04:10:40.411600 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.411876 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 929
04:10:40.411880 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.412332 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 64
04:10:40.452933 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.452943 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 219
04:10:40.453157 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.453158 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 141
04:10:40.454111 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.454313 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 24
04:10:40.454313 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.454328 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
04:10:40.411266 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.411318 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.411370 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.411574 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 418
04:10:40.411585 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.411850 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 929
04:10:40.411895 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.412363 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 64
04:10:40.452679 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.452988 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 219
04:10:40.453006 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.453117 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 141
04:10:40.454149 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
04:10:40.454190 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 24
04:10:40.454205 IP 192.0.3.10.443 > 100.99.222.11.40142: tcp 0
04:10:40.454346 IP 100.99.222.11.40142 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
04:10:40.411291 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 0
04:10:40.411313 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 0
04:10:40.411374 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 0
04:10:40.411577 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 418
04:10:40.411583 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 0
04:10:40.411843 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 929
04:10:40.411897 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 0
04:10:40.412365 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 64
04:10:40.452650 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 0
04:10:40.452996 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 219
04:10:40.453004 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 0
04:10:40.453110 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 141
04:10:40.454154 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 0
04:10:40.454183 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 24
04:10:40.454202 IP 192.0.3.10.443 > 192.0.3.20.40142: tcp 0
04:10:40.454349 IP 192.0.3.20.40142 > 192.0.3.10.443: tcp 0

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
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":762,"packets":22,"bytes":3392}},{"counter":{"family":"inet","name":"nd_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":765,"packets":26,"bytes":2816}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.99.222.11","timeout":3530,"expires":3526}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1063,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1064,"comment":"fvpn_class=01a19024-7d4e-7e1b-933e-cdd77d31be6f:4c3e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1065,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1066,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1067,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1068,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1069,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1070,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1071,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1072,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1073,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1074,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1075,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"counter":"u_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1076,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1078,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1079,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1080,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1081,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"counter":"d_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1084,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 108 TIME_WAIT src=100.99.222.11 dst=192.0.3.10 sport=55168 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=55168 [ASSURED] mark=0 use=1
ipv4     2 udp      17 18 src=100.99.222.11 dst=192.0.3.10 sport=41998 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=41998 mark=0 use=1
ipv4     2 tcp      6 2 CLOSE src=100.99.222.11 dst=192.0.3.10 sport=40142 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=40142 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 3 flow entries have been shown.
- after_member_tunnel: rx_packets=25
rx_bytes=4576
tx_packets=30
tx_bytes=4292
- after_node_tunnel: rx_packets=30
rx_bytes=4292
tx_packets=26
tx_bytes=4668
- after_node_uplink: rx_packets=65
rx_bytes=8092
tx_packets=54
tx_bytes=6260
- origin_arrival: {"probe_id":"a4dbbb14-9815-4fa5-a836-8f501890a925","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 3577
- admission_bytes_after: 6208
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id b63266b8-5ba5-4729-9b93-722e86fd4c21
mono_ns 3382123957652
wall_unix_ns 1793419844607586608
policy_expires_mono_ns 6912399917523
baseline_ok 1
serving 1
generation 6
reason serving
- relayed_before: 0
- direct_before: 1
- relayed_forced: 1
- direct_forced: 0
- relay_authenticated_udp: 1
- relay_tcp: 1
- relayed_restored: 0
- direct_restored: 1
- same_sockets_continue_on_direct: true
- table: inet e2e_relay in the member; inet fvpn unchanged
