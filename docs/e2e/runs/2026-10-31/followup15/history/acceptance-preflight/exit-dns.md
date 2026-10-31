# exit-dns

Designated TCP and UDP DNS traverse the selected exit

Status: fail

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §8.2, §10.3, AT12, AT20

- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.124.98.98 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-5027557e54f08768976d53fa
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-5027557e54f08768976d53fa
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.124.98.98 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.124.98.98/16 brd 100.124.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.124.0.0/16 dev fvnb0 proto kernel scope link src 100.124.98.98
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.124.98.98 dev fvnb0 table local proto kernel scope host src 100.124.98.98
broadcast 100.124.255.255 dev fvnb0 table local proto kernel scope link src 100.124.98.98
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":762,"packets":120,"bytes":10144}},{"counter":{"family":"inet","name":"nd_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":765,"packets":174,"bytes":11962}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.124.98.98","timeout":3499,"expires":3495}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1219,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1220,"comment":"fvpn_class=01a19015-a203-7629-af08-191fa4d268f6:15b0","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1221,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1222,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1223,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1224,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1225,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1226,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1227,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1228,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1229,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1230,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1231,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"counter":"u_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1232,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1234,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1235,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1236,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1237,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"counter":"d_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1240,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: ipv4     2 tcp      6 119 TIME_WAIT src=100.124.98.98 dst=192.0.3.10 sport=45466 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=45466 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 78 TIME_WAIT src=100.124.98.98 dst=192.0.3.10 sport=56430 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=56430 [ASSURED] mark=0 use=1
ipv4     2 udp      17 119 src=100.124.98.98 dst=192.0.3.10 sport=33221 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=33221 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 3 flow entries have been shown.
- before_member_tunnel: rx_packets=123
rx_bytes=14736
tx_packets=178
tx_bytes=19268
- before_node_tunnel: rx_packets=178
rx_bytes=19268
tx_packets=125
tx_bytes=14860
- before_node_uplink: rx_packets=164
rx_bytes=16286
tx_packets=202
tx_bytes=17478
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
03:55:05.467546 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.468170 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.468181 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.468303 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 418
03:55:05.468523 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.469074 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 927
03:55:05.469078 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.469481 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 64
03:55:05.510198 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.510210 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 219
03:55:05.510631 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.510829 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 141
03:55:05.511382 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.511609 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 24
03:55:05.511620 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.511768 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.511771 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
03:55:05.467795 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.467841 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.468326 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.468482 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 418
03:55:05.468488 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.468802 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 927
03:55:05.469093 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.469513 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 64
03:55:05.509770 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.510543 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 219
03:55:05.510589 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.510674 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 141
03:55:05.511402 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.511568 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 24
03:55:05.511581 IP 192.0.3.10.443 > 100.124.98.98.49228: tcp 0
03:55:05.511786 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
03:55:05.511786 IP 100.124.98.98.49228 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
03:55:05.467818 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 0
03:55:05.467836 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 0
03:55:05.468328 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 0
03:55:05.468484 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 418
03:55:05.468487 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 0
03:55:05.468793 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 927
03:55:05.469095 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 0
03:55:05.469516 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 64
03:55:05.509746 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 0
03:55:05.510565 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 219
03:55:05.510584 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 0
03:55:05.510668 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 141
03:55:05.511413 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 0
03:55:05.511561 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 24
03:55:05.511579 IP 192.0.3.10.443 > 192.0.3.20.49228: tcp 0
03:55:05.511789 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 0
03:55:05.511790 IP 192.0.3.20.49228 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.124.98.98 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.124.98.98/16 brd 100.124.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.124.0.0/16 dev fvnb0 proto kernel scope link src 100.124.98.98
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.124.98.98 dev fvnb0 table local proto kernel scope host src 100.124.98.98
broadcast 100.124.255.255 dev fvnb0 table local proto kernel scope link src 100.124.98.98
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":762,"packets":128,"bytes":11660}},{"counter":{"family":"inet","name":"nd_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":765,"packets":183,"bytes":13115}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.124.98.98","timeout":3489,"expires":3487}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1271,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1272,"comment":"fvpn_class=01a19015-a203-7629-af08-191fa4d268f6:15b0","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1273,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1274,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1275,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1276,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1277,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1278,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1279,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1280,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1281,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1282,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1283,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"counter":"u_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1284,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1286,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1287,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1288,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"counter":"d_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1292,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 110 TIME_WAIT src=100.124.98.98 dst=192.0.3.10 sport=45466 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=45466 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 69 TIME_WAIT src=100.124.98.98 dst=192.0.3.10 sport=56430 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=56430 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 2 CLOSE src=100.124.98.98 dst=192.0.3.10 sport=49228 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=49228 [ASSURED] mark=0 use=1
ipv4     2 udp      17 110 src=100.124.98.98 dst=192.0.3.10 sport=33221 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=33221 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 4 flow entries have been shown.
- after_member_tunnel: rx_packets=131
rx_bytes=16592
tx_packets=187
tx_bytes=20788
- after_node_tunnel: rx_packets=187
rx_bytes=20788
tx_packets=133
tx_bytes=16716
- after_node_uplink: rx_packets=172
rx_bytes=17914
tx_packets=211
tx_bytes=18757
- origin_arrival: {"probe_id":"2165fed8-0da2-4c0a-93de-bb536743b8ed","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 22106
- admission_bytes_after: 24775
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 9d370541-1281-45e6-8a24-5f6c0627d4ad
mono_ns 2448529976004
wall_unix_ns 1793418911013605079
policy_expires_mono_ns 5938159816800
baseline_ok 1
serving 1
generation 6
reason serving
- udp: {"Transport":"udp","Reply":"fvpn-e2e-member-session: dns read","dns_upload_bytes":"0","dns_download_bytes":"0","internet_upload_bytes":"0","internet_download_bytes":"0"}
- error: designated DNS query: fvpn-e2e-member-session: dns read
