# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 9b3a89e7-fa86-426c-9737-ce55bf66aa77

Source: a1796b0d2be7f0ef068f66347fccb11d55ed58a4

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
- binding_valid_from: 2026-10-31T22:17:55Z
- binding_valid_until: 2026-10-31T23:17:40Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.80.106.226 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-916af475ac3784a32b735ad1
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-916af475ac3784a32b735ad1
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.80.106.226 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.80.106.226/16 brd 100.80.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.80.0.0/16 dev fvnb0 proto kernel scope link src 100.80.106.226
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.80.106.226 dev fvnb0 table local proto kernel scope host src 100.80.106.226
broadcast 100.80.255.255 dev fvnb0 table local proto kernel scope link src 100.80.106.226
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":257,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":258,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":259,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":260,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.80.106.226","timeout":587,"expires":586}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"comment":"fvpn_class=01a1940a-37b8-766a-ac06-3599dc50c266:3c30","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.80.106.226"}},{"counter":"u_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.80.106.226"}},{"counter":"d_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=1
rx_bytes=148
tx_packets=3
tx_bytes=388
- before_node_tunnel: rx_packets=2
rx_bytes=296
tx_packets=4
tx_bytes=480
- before_node_uplink: rx_packets=30
rx_bytes=3666
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
23 packets captured
23 packets received by filter
0 packets dropped by kernel
22:18:10.676443 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:11.700654 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:12.724840 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.749647 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.750864 IP 192.0.3.10.443 > 100.80.106.226.20001: tcp 0
22:18:13.750864 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.750864 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.750883 IP 100.80.106.226.20001 > 192.0.3.10.443: tcp 0
22:18:13.750894 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.750897 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.751213 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 418
22:18:13.751483 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.751930 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 930
22:18:13.751935 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.752425 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 64
22:18:13.796239 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.796260 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 219
22:18:13.796584 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.796584 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 141
22:18:13.797171 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.797368 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 24
22:18:13.797368 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.797390 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
22:18:13.750698 IP 100.80.106.226.20001 > 192.0.3.10.443: tcp 0
22:18:13.750698 IP 100.80.106.226.20001 > 192.0.3.10.443: tcp 0
22:18:13.750698 IP 100.80.106.226.20001 > 192.0.3.10.443: tcp 0
22:18:13.750698 IP 100.80.106.226.20001 > 192.0.3.10.443: tcp 0
22:18:13.750698 IP 100.80.106.226.20001 > 192.0.3.10.443: tcp 0
22:18:13.750698 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.750698 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.750698 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.750754 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.750830 IP 192.0.3.10.443 > 100.80.106.226.20001: tcp 0
22:18:13.750835 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.750838 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.751090 IP 100.80.106.226.20001 > 192.0.3.10.443: tcp 0
22:18:13.751090 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.751090 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.751449 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 418
22:18:13.751468 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.751909 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 930
22:18:13.752116 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.754169 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 64
22:18:13.795665 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.796402 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 219
22:18:13.796424 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.796506 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 141
22:18:13.797193 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0
22:18:13.797219 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 24
22:18:13.797231 IP 192.0.3.10.443 > 100.80.106.226.47864: tcp 0
22:18:13.797408 IP 100.80.106.226.47864 > 192.0.3.10.443: tcp 0

28 packets captured
28 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
22 packets captured
22 packets received by filter
0 packets dropped by kernel
22:18:13.750795 IP 192.0.3.20.20001 > 192.0.3.10.443: tcp 0
22:18:13.750796 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 0
22:18:13.750797 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 0
22:18:13.750824 IP 192.0.3.10.443 > 192.0.3.20.20001: tcp 0
22:18:13.750833 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 0
22:18:13.750836 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 0
22:18:13.751098 IP 192.0.3.20.20001 > 192.0.3.10.443: tcp 0
22:18:13.751101 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 0
22:18:13.751102 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 0
22:18:13.751457 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 418
22:18:13.751465 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 0
22:18:13.751902 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 930
22:18:13.752123 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 0
22:18:13.754178 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 64
22:18:13.795642 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 0
22:18:13.796413 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 219
22:18:13.796422 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 0
22:18:13.796499 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 141
22:18:13.797198 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 0
22:18:13.797215 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 24
22:18:13.797229 IP 192.0.3.10.443 > 192.0.3.20.47864: tcp 0
22:18:13.797411 IP 192.0.3.20.47864 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.80.106.226 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.80.106.226/16 brd 100.80.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.80.0.0/16 dev fvnb0 proto kernel scope link src 100.80.106.226
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.80.106.226 dev fvnb0 table local proto kernel scope host src 100.80.106.226
broadcast 100.80.255.255 dev fvnb0 table local proto kernel scope link src 100.80.106.226
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":257,"packets":10,"bytes":1639}},{"counter":{"family":"inet","name":"nd_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":258,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":259,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1940a751070cdb0abf3f6db577ab1","table":"fvpn","handle":260,"packets":18,"bytes":1685}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.80.106.226","timeout":582,"expires":577}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":353,"comment":"fvpn_class=01a1940a-37b8-766a-ac06-3599dc50c266:3c30","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":354,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":356,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":357,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":358,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.80.106.226"}},{"counter":"u_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.80.106.226"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.80.106.226"}},{"counter":"d_01a1940a751070cdb0abf3f6db577ab1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3c30"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 tcp      6 5 CLOSE src=100.80.106.226 dst=192.0.3.10 sport=20001 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=20001 mark=0 use=1
ipv4     2 tcp      6 5 CLOSE src=100.80.106.226 dst=192.0.3.10 sport=47864 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=47864 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=13
rx_bytes=2320
tx_packets=22
tx_bytes=2920
- after_node_tunnel: rx_packets=21
rx_bytes=2828
tx_packets=16
tx_bytes=2652
- after_node_uplink: rx_packets=42
rx_bytes=5594
tx_packets=27
tx_bytes=3075
- origin_arrival: {"probe_id":"b0af7cc6-0c98-45c6-81a8-5bd6166906bb","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 3324
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 68631012624460
wall_unix_ns 1793485093496253528
policy_expires_mono_ns 69213286524404
baseline_ok 1
serving 1
generation 5
reason serving
