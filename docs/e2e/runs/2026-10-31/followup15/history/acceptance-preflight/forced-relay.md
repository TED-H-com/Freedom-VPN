# forced-relay

Member forced onto the relay path and restored to direct

Status: pass

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §8, AT31

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
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":762,"packets":14,"bytes":1874}},{"counter":{"family":"inet","name":"nd_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":765,"packets":18,"bytes":1703}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.124.98.98","timeout":3540,"expires":3535}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1011,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1012,"comment":"fvpn_class=01a19015-a203-7629-af08-191fa4d268f6:15b0","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1013,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1014,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1015,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1016,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1017,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1018,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1019,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1020,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1021,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1022,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1023,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"counter":"u_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1024,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1026,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1027,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1028,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1029,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"counter":"d_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1032,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 tcp      6 117 TIME_WAIT src=100.124.98.98 dst=192.0.3.10 sport=56430 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=56430 [ASSURED] mark=0 use=1
ipv4     2 udp      17 27 src=100.124.98.98 dst=192.0.3.10 sport=38197 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=38197 mark=0 use=1
- before_member_tunnel: rx_packets=17
rx_bytes=2720
tx_packets=22
tx_bytes=2852
- before_node_tunnel: rx_packets=22
rx_bytes=2852
tx_packets=18
tx_bytes=2812
- before_node_uplink: rx_packets=57
rx_bytes=6490
tx_packets=45
tx_bytes=4993
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
03:54:26.192961 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.193083 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.193092 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.193238 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 418
03:54:26.193402 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.193773 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 929
03:54:26.193775 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.194210 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 64
03:54:26.234836 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.234846 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 219
03:54:26.235196 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.235196 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 141
03:54:26.235852 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.235944 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 24
03:54:26.235952 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.235955 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.235958 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
03:54:26.193009 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.193052 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.193370 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.193370 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 418
03:54:26.193386 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.193752 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 929
03:54:26.193833 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.194233 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 64
03:54:26.234653 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.235024 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 219
03:54:26.235070 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.235168 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 141
03:54:26.235884 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.235925 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 24
03:54:26.235936 IP 192.0.3.10.443 > 100.124.98.98.43656: tcp 0
03:54:26.235993 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
03:54:26.235993 IP 100.124.98.98.43656 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
03:54:26.193032 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 0
03:54:26.193047 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 0
03:54:26.193373 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 0
03:54:26.193374 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 418
03:54:26.193384 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 0
03:54:26.193741 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 929
03:54:26.193836 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 0
03:54:26.194235 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 64
03:54:26.234637 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 0
03:54:26.235046 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 219
03:54:26.235065 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 0
03:54:26.235150 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 141
03:54:26.235895 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 0
03:54:26.235920 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 24
03:54:26.235935 IP 192.0.3.10.443 > 192.0.3.20.43656: tcp 0
03:54:26.235996 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 0
03:54:26.235997 IP 192.0.3.20.43656 > 192.0.3.10.443: tcp 0

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
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":762,"packets":22,"bytes":3392}},{"counter":{"family":"inet","name":"nd_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1901771d372d9a49302d8eaaa1a6e","table":"fvpn","handle":765,"packets":27,"bytes":2856}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.124.98.98","timeout":3530,"expires":3527}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1063,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1064,"comment":"fvpn_class=01a19015-a203-7629-af08-191fa4d268f6:15b0","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1065,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1066,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1067,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1068,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1069,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1070,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1071,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1072,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1073,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1074,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1075,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.124.98.98"}},{"counter":"u_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1076,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1078,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1079,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1080,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1081,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.124.98.98"}},{"counter":"d_01a1901771d372d9a49302d8eaaa1a6e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:15b0"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1084,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 2 CLOSE src=100.124.98.98 dst=192.0.3.10 sport=43656 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=43656 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 108 TIME_WAIT src=100.124.98.98 dst=192.0.3.10 sport=56430 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=56430 [ASSURED] mark=0 use=1
ipv4     2 udp      17 18 src=100.124.98.98 dst=192.0.3.10 sport=38197 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=38197 mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 3 flow entries have been shown.
- after_member_tunnel: rx_packets=25
rx_bytes=4576
tx_packets=31
tx_bytes=4372
- after_node_tunnel: rx_packets=31
rx_bytes=4372
tx_packets=26
tx_bytes=4668
- after_node_uplink: rx_packets=65
rx_bytes=8120
tx_packets=54
tx_bytes=6272
- origin_arrival: {"probe_id":"bc07c920-4b37-4e06-a1ed-d630b36e4949","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 3577
- admission_bytes_after: 6248
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 9d370541-1281-45e6-8a24-5f6c0627d4ad
mono_ns 2407910013820
wall_unix_ns 1793418870393642952
policy_expires_mono_ns 5938159816800
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
