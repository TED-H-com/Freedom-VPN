# forced-relay

Member forced onto the relay path and restored to direct

Status: pass

Run: 91eb0372-e42b-4698-a137-7cce53b263fa

Source: f52fc9381405d4aafd064dc0ef055977de97f418

Spec: §8, AT31

- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.116.9.160 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-73ecc0f49c1a6843b498c2cc
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-73ecc0f49c1a6843b498c2cc
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.116.9.160 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.116.9.160/16 brd 100.116.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.116.0.0/16 dev fvnb0 proto kernel scope link src 100.116.9.160
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.116.9.160 dev fvnb0 table local proto kernel scope host src 100.116.9.160
broadcast 100.116.255.255 dev fvnb0 table local proto kernel scope link src 100.116.9.160
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":786,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":787,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":788,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":789,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":790,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":791,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":792,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":793,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":913,"packets":14,"bytes":1877}},{"counter":{"family":"inet","name":"nd_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":914,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":915,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":916,"packets":17,"bytes":1663}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.116.9.160","timeout":594,"expires":594}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1245,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1246,"comment":"fvpn_class=01a19fc2-259b-7d6c-8757-3e03573886a4:1ccd","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1247,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1248,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1249,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1250,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1251,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1252,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1253,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1254,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1255,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1256,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1257,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"counter":"u_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1258,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1260,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1261,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1262,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1263,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1264,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1265,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1266,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"counter":"d_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1269,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 udp      17 26 src=100.116.9.160 dst=192.0.3.10 sport=38091 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=38091 mark=0 use=1
ipv4     2 tcp      6 117 TIME_WAIT src=100.116.9.160 dst=192.0.3.10 sport=43608 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=43608 [ASSURED] mark=0 use=1
- before_member_tunnel: rx_packets=18
rx_bytes=2636
tx_packets=18
tx_bytes=2500
- before_node_tunnel: rx_packets=18
rx_bytes=2500
tx_packets=19
tx_bytes=2784
- before_node_uplink: rx_packets=56
rx_bytes=6451
tx_packets=44
tx_bytes=4939
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
04:56:27.976550 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:27.977635 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:27.977658 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:27.977827 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 418
04:56:27.977899 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:27.978177 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 931
04:56:27.978181 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:27.978755 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 64
04:56:28.020984 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:28.020991 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 219
04:56:28.021261 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:28.021262 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 141
04:56:28.021816 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:28.022656 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 24
04:56:28.022656 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:28.022668 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:56:27.976764 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:27.976844 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:27.977692 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:27.977855 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 418
04:56:27.977873 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:27.978163 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 931
04:56:27.978311 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:27.978783 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 64
04:56:28.020661 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:28.021126 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 219
04:56:28.021138 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:28.021223 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 141
04:56:28.022254 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0
04:56:28.022326 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 24
04:56:28.022358 IP 192.0.3.10.443 > 100.116.9.160.37834: tcp 0
04:56:28.023070 IP 100.116.9.160.37834 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
04:56:27.976793 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 0
04:56:27.976838 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 0
04:56:27.977703 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 0
04:56:27.977859 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 418
04:56:27.977868 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 0
04:56:27.978153 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 931
04:56:27.978320 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 0
04:56:27.978786 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 64
04:56:28.020641 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 0
04:56:28.021130 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 219
04:56:28.021136 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 0
04:56:28.021216 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 141
04:56:28.022262 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 0
04:56:28.022307 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 24
04:56:28.022350 IP 192.0.3.10.443 > 192.0.3.20.37834: tcp 0
04:56:28.023075 IP 192.0.3.20.37834 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.116.9.160 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.116.9.160/16 brd 100.116.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.116.0.0/16 dev fvnb0 proto kernel scope link src 100.116.9.160
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.116.9.160 dev fvnb0 table local proto kernel scope host src 100.116.9.160
broadcast 100.116.255.255 dev fvnb0 table local proto kernel scope link src 100.116.9.160
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":786,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":787,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":788,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":789,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":790,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":791,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":792,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":793,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":913,"packets":22,"bytes":3397}},{"counter":{"family":"inet","name":"nd_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":914,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":915,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":916,"packets":25,"bytes":2776}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.116.9.160","timeout":589,"expires":585}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1277,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1278,"comment":"fvpn_class=01a19fc2-259b-7d6c-8757-3e03573886a4:1ccd","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1279,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1280,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1281,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1282,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1283,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1284,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1285,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1286,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1287,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1288,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"counter":"u_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1290,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1292,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1293,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1294,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1295,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1296,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1297,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1298,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"counter":"d_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1301,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 3 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.116.9.160 dst=192.0.3.10 sport=37834 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=37834 [ASSURED] mark=0 use=1
ipv4     2 udp      17 18 src=100.116.9.160 dst=192.0.3.10 sport=38091 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=38091 mark=0 use=1
ipv4     2 tcp      6 108 TIME_WAIT src=100.116.9.160 dst=192.0.3.10 sport=43608 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=43608 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=26
rx_bytes=4492
tx_packets=26
tx_bytes=3940
- after_node_tunnel: rx_packets=26
rx_bytes=3940
tx_packets=27
tx_bytes=4640
- after_node_uplink: rx_packets=64
rx_bytes=8083
tx_packets=52
tx_bytes=6164
- origin_arrival: {"probe_id":"b8b711ca-f3c8-453c-9b54-118f128a06f4","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 3540
- admission_bytes_after: 6173
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 5233514908065
wall_unix_ns 1793681791063531749
policy_expires_mono_ns 5823356842668
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
