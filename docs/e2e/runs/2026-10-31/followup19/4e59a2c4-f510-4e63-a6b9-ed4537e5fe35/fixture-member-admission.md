# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 4e59a2c4-f510-4e63-a6b9-ed4537e5fe35

Source: c75ad1957773af8f9b9e71036be357eeb491f943

Spec: §8, AT02, AT12

- grant: 202
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T13:53:47Z
- binding_valid_until: 2026-10-31T14:53:32Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-be10bf6db872060c6b13be63
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-be10bf6db872060c6b13be63
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.72.217.174 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.72.217.174/16 brd 100.72.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.72.0.0/16 dev fvnb0 proto kernel scope link src 100.72.217.174
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.72.217.174 dev fvnb0 table local proto kernel scope host src 100.72.217.174
broadcast 100.72.255.255 dev fvnb0 table local proto kernel scope link src 100.72.217.174
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":170,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":171,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":172,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":173,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":174,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":175,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":176,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":334,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":335,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":336,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":337,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.72.217.174","timeout":596,"expires":594}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"comment":"fvpn_class=01a1923c-ad2b-7522-ad48-1cc2de192b36:59ca","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":347,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":348,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":349,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":350,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":353,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":354,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.72.217.174"}},{"counter":"u_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":357,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":358,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.72.217.174"}},{"counter":"d_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=25
rx_bytes=3318
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
13:53:54.157684 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.161682 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.161693 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.161876 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 418
13:53:54.162240 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.162505 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 930
13:53:54.162508 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.162972 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 64
13:53:54.204726 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.204730 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 219
13:53:54.205897 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.205897 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 141
13:53:54.206887 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.208603 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 24
13:53:54.208603 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.208612 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
13:53:54.161445 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.161523 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.161816 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.162140 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 418
13:53:54.162154 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.162431 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 930
13:53:54.162575 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.163042 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 64
13:53:54.203657 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.205367 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 219
13:53:54.205393 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.205480 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 141
13:53:54.208106 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0
13:53:54.208333 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 24
13:53:54.208346 IP 192.0.3.10.443 > 100.72.217.174.34516: tcp 0
13:53:54.208902 IP 100.72.217.174.34516 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
13:53:54.161505 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 0
13:53:54.161516 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 0
13:53:54.161824 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 0
13:53:54.162144 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 418
13:53:54.162150 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 0
13:53:54.162423 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 930
13:53:54.162578 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 0
13:53:54.163045 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 64
13:53:54.203642 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 0
13:53:54.205379 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 219
13:53:54.205390 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 0
13:53:54.205474 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 141
13:53:54.208120 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 0
13:53:54.208324 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 24
13:53:54.208344 IP 192.0.3.10.443 > 192.0.3.24.34516: tcp 0
13:53:54.208912 IP 192.0.3.24.34516 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.72.217.174 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.72.217.174/16 brd 100.72.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.72.0.0/16 dev fvnb0 proto kernel scope link src 100.72.217.174
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.72.217.174 dev fvnb0 table local proto kernel scope host src 100.72.217.174
broadcast 100.72.255.255 dev fvnb0 table local proto kernel scope link src 100.72.217.174
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":170,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":171,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":172,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":173,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":174,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":175,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":176,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":334,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":335,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":336,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1923ce8ad78569f3629c4740ac06a","table":"fvpn","handle":337,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.72.217.174","timeout":591,"expires":586}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"comment":"fvpn_class=01a1923c-ad2b-7522-ad48-1cc2de192b36:59ca","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":374,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":379,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":380,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":382,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":383,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.72.217.174"}},{"counter":"u_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":386,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":387,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":388,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.72.217.174"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.72.217.174"}},{"counter":"d_01a1923ce8ad78569f3629c4740ac06a"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:59ca"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":392,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- after_conntrack: ipv4     2 tcp      6 2 CLOSE src=100.72.217.174 dst=192.0.3.10 sport=34516 dport=443 src=192.0.3.10 dst=192.0.3.24 sport=443 dport=34516 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=35
rx_bytes=5033
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"0e04e658-4524-4c8e-af2c-25dfe46eb777","requests":1,"source_address":"192.0.3.24"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 38374011332876
wall_unix_ns 1793454836494961734
policy_expires_mono_ns 38965239315266
baseline_ok 1
serving 1
generation 5
reason serving
