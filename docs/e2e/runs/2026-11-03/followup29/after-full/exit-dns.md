# exit-dns

Designated TCP and UDP DNS traverse the selected exit

Status: pass

Run: 91eb0372-e42b-4698-a137-7cce53b263fa

Source: f52fc9381405d4aafd064dc0ef055977de97f418

Spec: §8.2, §10.3, AT12, AT20

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
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":786,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":787,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":788,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":789,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":790,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":791,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":792,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":793,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":913,"packets":80,"bytes":7350}},{"counter":{"family":"inet","name":"nd_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":914,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":915,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":916,"packets":111,"bytes":8088}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.116.9.160","timeout":564,"expires":563}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1437,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1438,"comment":"fvpn_class=01a19fc2-259b-7d6c-8757-3e03573886a4:1ccd","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1439,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1440,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1441,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1442,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1443,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1444,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1445,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1446,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1447,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1448,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1449,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"counter":"u_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1450,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1452,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1453,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1454,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1455,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1456,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1457,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1458,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"counter":"d_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1461,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: ipv4     2 udp      17 119 src=100.116.9.160 dst=192.0.3.10 sport=55820 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=55820 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 119 TIME_WAIT src=100.116.9.160 dst=192.0.3.10 sport=50334 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=50334 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 87 TIME_WAIT src=100.116.9.160 dst=192.0.3.10 sport=43608 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=43608 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 3 flow entries have been shown.
- before_member_tunnel: rx_packets=83
rx_bytes=10380
tx_packets=112
tx_bytes=12644
- before_node_tunnel: rx_packets=112
rx_bytes=12644
tx_packets=86
tx_bytes=10656
- before_node_uplink: rx_packets=123
rx_bytes=12890
tx_packets=139
tx_bytes=12722
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:56:58.108283 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.108803 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.108821 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.109090 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 418
04:56:58.109465 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.109822 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 931
04:56:58.109830 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.110367 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 64
04:56:58.150902 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.150921 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 219
04:56:58.151656 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.151656 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 141
04:56:58.152276 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.152581 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 24
04:56:58.152595 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.152909 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.152917 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:56:58.108446 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.108507 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.108980 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.109247 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 418
04:56:58.109257 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.109520 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 931
04:56:58.109978 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.110519 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 64
04:56:58.150652 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.151189 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 219
04:56:58.151207 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.151287 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 141
04:56:58.152505 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.152555 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 24
04:56:58.152568 IP 192.0.3.10.443 > 100.116.9.160.51946: tcp 0
04:56:58.152795 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0
04:56:58.153141 IP 100.116.9.160.51946 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
04:56:58.108475 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 0
04:56:58.108500 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 0
04:56:58.108983 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 0
04:56:58.109250 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 418
04:56:58.109254 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 0
04:56:58.109512 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 931
04:56:58.109981 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 0
04:56:58.110522 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 64
04:56:58.150638 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 0
04:56:58.151196 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 219
04:56:58.151204 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 0
04:56:58.151280 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 141
04:56:58.152517 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 0
04:56:58.152551 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 24
04:56:58.152565 IP 192.0.3.10.443 > 192.0.3.20.51946: tcp 0
04:56:58.152802 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 0
04:56:58.153148 IP 192.0.3.20.51946 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
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
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":786,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":787,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":788,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":789,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":790,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":791,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":792,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":793,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":913,"packets":88,"bytes":8870}},{"counter":{"family":"inet","name":"nd_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":914,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":915,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":916,"packets":120,"bytes":9241}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.116.9.160","timeout":559,"expires":555}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1469,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1470,"comment":"fvpn_class=01a19fc2-259b-7d6c-8757-3e03573886a4:1ccd","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1471,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1472,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1473,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1474,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1475,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1476,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1477,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1478,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1479,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1480,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1481,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"counter":"u_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1482,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1484,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1485,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1486,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1487,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1488,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1489,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1490,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"counter":"d_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1493,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 4 flow entries have been shown.
ipv4     2 udp      17 110 src=100.116.9.160 dst=192.0.3.10 sport=55820 dport=9445 src=192.0.3.10 dst=192.0.3.20 sport=9445 dport=55820 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 110 TIME_WAIT src=100.116.9.160 dst=192.0.3.10 sport=50334 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=50334 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 2 CLOSE src=100.116.9.160 dst=192.0.3.10 sport=51946 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=51946 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 78 TIME_WAIT src=100.116.9.160 dst=192.0.3.10 sport=43608 dport=9446 src=192.0.3.10 dst=192.0.3.20 sport=9446 dport=43608 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=91
rx_bytes=12236
tx_packets=121
tx_bytes=14164
- after_node_tunnel: rx_packets=121
rx_bytes=14164
tx_packets=94
tx_bytes=12512
- after_node_uplink: rx_packets=131
rx_bytes=14522
tx_packets=148
tx_bytes=14001
- origin_arrival: {"probe_id":"e64effac-ed08-4d68-b3bf-41e0ede360cd","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 15438
- admission_bytes_after: 18111
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 5263538662454
wall_unix_ns 1793681821087285964
policy_expires_mono_ns 5823356842668
baseline_ok 1
serving 1
generation 6
reason serving
- udp: {"Transport":"udp","Reply":"rcode=0 answers=1","dns_upload_bytes":"65","dns_download_bytes":"98","internet_upload_bytes":"0","internet_download_bytes":"0"}
- tcp: {"Transport":"tcp","Reply":"rcode=0 answers=1","dns_upload_bytes":"307","dns_download_bytes":"340","internet_upload_bytes":"0","internet_download_bytes":"0"}
- path: origin-network DNS relay -> node filtering resolver; both nu_ and nd_ increased, u_ and d_ held
- native_fallback: not claimed; explicit resolver queries
