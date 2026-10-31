# exit-dns

Designated TCP and UDP DNS traverse the selected exit

Status: pass

Run: 12d10041-9d33-4fbb-a4b0-9d2a21bce785

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §8.2, §10.3, AT12, AT20

- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.83.104.230 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-6661d07246816bba4d5f8281
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-6661d07246816bba4d5f8281
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.104.230 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.83.104.230/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.104.230
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.104.230 dev fvnb0 table local proto kernel scope host src 100.83.104.230
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.104.230
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1903458597519b428f4418308e67f","table":"fvpn","handle":248,"packets":120,"bytes":10147}},{"counter":{"family":"inet","name":"nd_01a1903458597519b428f4418308e67f","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1903458597519b428f4418308e67f","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1903458597519b428f4418308e67f","table":"fvpn","handle":251,"packets":173,"bytes":11922}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.104.230","timeout":3474,"expires":3473}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":861,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":862,"comment":"fvpn_class=01a19034-32e4-7d4a-8fc5-3d8df444a439:d01","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":863,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":864,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":865,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":866,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":867,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":868,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":869,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":870,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":871,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":872,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":873,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"counter":"u_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":874,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":876,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":877,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":878,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":879,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"counter":"d_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":882,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 3 flow entries have been shown.
ipv4     2 udp      17 119 src=100.83.104.230 dst=192.0.3.10 sport=42584 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=42584 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 78 TIME_WAIT src=100.83.104.230 dst=192.0.3.10 sport=37368 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=37368 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 119 TIME_WAIT src=100.83.104.230 dst=192.0.3.10 sport=40946 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=40946 [ASSURED] mark=0 use=1
- before_member_tunnel: rx_packets=124
rx_bytes=14768
tx_packets=178
tx_bytes=19220
- before_node_tunnel: rx_packets=178
rx_bytes=19220
tx_packets=126
tx_bytes=14892
- before_node_uplink: rx_packets=163
rx_bytes=16247
tx_packets=190
tx_bytes=16010
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:27:09.464097 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.464215 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.464230 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.464491 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 418
04:27:09.464535 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.464798 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 930
04:27:09.464803 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.465484 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 64
04:27:09.506893 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.506897 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 219
04:27:09.507142 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.507143 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 141
04:27:09.507697 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.507988 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 24
04:27:09.507988 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.507995 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:27:09.464134 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.464177 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.464244 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.464503 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 418
04:27:09.464514 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.464777 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 930
04:27:09.464815 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.465500 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 64
04:27:09.506651 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.506920 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 219
04:27:09.506937 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.507022 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 141
04:27:09.507741 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0
04:27:09.507793 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 24
04:27:09.507806 IP 192.0.3.10.443 > 100.83.104.230.33802: tcp 0
04:27:09.508009 IP 100.83.104.230.33802 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
04:27:09.464153 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 0
04:27:09.464172 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 0
04:27:09.464248 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 0
04:27:09.464506 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 418
04:27:09.464511 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 0
04:27:09.464771 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 930
04:27:09.464817 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 0
04:27:09.465504 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 64
04:27:09.506639 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 0
04:27:09.506927 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 219
04:27:09.506935 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 0
04:27:09.507017 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 141
04:27:09.507749 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 0
04:27:09.507783 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 24
04:27:09.507804 IP 192.0.3.10.443 > 192.0.3.20.33802: tcp 0
04:27:09.508012 IP 192.0.3.20.33802 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.104.230 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.83.104.230/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.104.230
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.104.230 dev fvnb0 table local proto kernel scope host src 100.83.104.230
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.104.230
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1903458597519b428f4418308e67f","table":"fvpn","handle":248,"packets":128,"bytes":11666}},{"counter":{"family":"inet","name":"nd_01a1903458597519b428f4418308e67f","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1903458597519b428f4418308e67f","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1903458597519b428f4418308e67f","table":"fvpn","handle":251,"packets":181,"bytes":13035}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.104.230","timeout":3469,"expires":3465}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":887,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":888,"comment":"fvpn_class=01a19034-32e4-7d4a-8fc5-3d8df444a439:d01","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":889,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":890,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":891,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":892,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":893,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":894,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":895,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":896,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":897,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":898,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":899,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"counter":"u_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":900,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":902,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":903,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":904,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":905,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"counter":"d_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":908,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 4 flow entries have been shown.
ipv4     2 udp      17 110 src=100.83.104.230 dst=192.0.3.10 sport=42584 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=42584 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 69 TIME_WAIT src=100.83.104.230 dst=192.0.3.10 sport=37368 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=37368 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 2 CLOSE src=100.83.104.230 dst=192.0.3.10 sport=33802 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=33802 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 110 TIME_WAIT src=100.83.104.230 dst=192.0.3.10 sport=40946 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=40946 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=132
rx_bytes=16624
tx_packets=186
tx_bytes=20660
- after_node_tunnel: rx_packets=186
rx_bytes=20660
tx_packets=134
tx_bytes=16748
- after_node_uplink: rx_packets=172
rx_bytes=17920
tx_packets=199
tx_bytes=17277
- origin_arrival: {"probe_id":"0dd2441d-d653-4dbf-aad5-f30cc8ba1687","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 22069
- admission_bytes_after: 24701
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 2fde9c2a-3d5e-45be-bdbd-b976535ba0ba
mono_ns 4370441342050
wall_unix_ns 1793420832924971098
policy_expires_mono_ns 7839958813289
baseline_ok 1
serving 1
generation 6
reason serving
- udp: {"Transport":"udp","Reply":"rcode=0 answers=1","dns_upload_bytes":"65","dns_download_bytes":"98","internet_upload_bytes":"0","internet_download_bytes":"0"}
- tcp: {"Transport":"tcp","Reply":"rcode=0 answers=1","dns_upload_bytes":"359","dns_download_bytes":"340","internet_upload_bytes":"0","internet_download_bytes":"0"}
- path: origin-network DNS relay -> node filtering resolver; both nu_ and nd_ increased, u_ and d_ held
- native_fallback: not claimed; explicit resolver queries
