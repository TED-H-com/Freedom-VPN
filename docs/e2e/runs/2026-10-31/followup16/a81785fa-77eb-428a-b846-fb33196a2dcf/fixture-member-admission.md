# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: a81785fa-77eb-428a-b846-fb33196a2dcf

Source: ec84d71f45ce3622b11a8a7e55f67c9c1c432d29

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
- binding_valid_from: 2026-10-31T06:47:31Z
- binding_valid_until: 2026-10-31T07:47:21Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 20.056
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-eded1126562565889f08f566
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.117.189.189 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.117.189.189/16 brd 100.117.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.117.0.0/16 dev fvnb0 proto kernel scope link src 100.117.189.189
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.117.189.189 dev fvnb0 table local proto kernel scope host src 100.117.189.189
broadcast 100.117.255.255 dev fvnb0 table local proto kernel scope link src 100.117.189.189
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":137,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":138,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":139,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":140,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":141,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":142,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":143,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":270,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":271,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":272,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":273,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.117.189.189","timeout":3567,"expires":3565}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":382,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":383,"comment":"fvpn_class=01a190b6-8014-7e5f-9d92-47b3616ff0bf:1366","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":385,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":386,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":387,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":388,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":390,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":391,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":392,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":393,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":394,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.117.189.189"}},{"counter":"u_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":395,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":397,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":398,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":399,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":400,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.117.189.189"}},{"counter":"d_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":403,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=41
rx_bytes=4575
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
06:47:57.342248 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.448799 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.448816 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.449038 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 418
06:47:57.450330 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.450331 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 930
06:47:57.450358 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.451005 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 64
06:47:57.493807 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.493829 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 219
06:47:57.496284 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.496284 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 141
06:47:57.497313 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.498309 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 24
06:47:57.498309 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.498325 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
06:47:57.447381 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.447489 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.449336 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.449337 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 418
06:47:57.449392 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.449809 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 930
06:47:57.450906 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.451711 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 64
06:47:57.492679 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.494510 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 219
06:47:57.494557 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.494841 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 141
06:47:57.497719 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
06:47:57.497892 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 24
06:47:57.497911 IP 192.0.3.10.443 > 100.117.189.189.37412: tcp 0
06:47:57.498508 IP 100.117.189.189.37412 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
06:47:57.447463 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 0
06:47:57.447481 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 0
06:47:57.449354 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 0
06:47:57.449356 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 418
06:47:57.449387 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 0
06:47:57.449800 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 930
06:47:57.450918 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 0
06:47:57.451735 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 64
06:47:57.492655 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 0
06:47:57.494531 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 219
06:47:57.494550 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 0
06:47:57.494825 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 141
06:47:57.497738 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 0
06:47:57.497882 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 24
06:47:57.497908 IP 192.0.3.10.443 > 192.0.3.20.37412: tcp 0
06:47:57.498518 IP 192.0.3.20.37412 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.117.189.189 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.117.189.189/16 brd 100.117.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.117.0.0/16 dev fvnb0 proto kernel scope link src 100.117.189.189
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.117.189.189 dev fvnb0 table local proto kernel scope host src 100.117.189.189
broadcast 100.117.255.255 dev fvnb0 table local proto kernel scope link src 100.117.189.189
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":137,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":138,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":139,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":140,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":141,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":142,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":143,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":270,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":271,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":272,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190b6a4d67398a0beb27f22ae4b51","table":"fvpn","handle":273,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.117.189.189","timeout":3562,"expires":3557}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":408,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"comment":"fvpn_class=01a190b6-8014-7e5f-9d92-47b3616ff0bf:1366","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.117.189.189"}},{"counter":"u_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":423,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":424,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.117.189.189"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":426,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.117.189.189"}},{"counter":"d_01a190b6a4d67398a0beb27f22ae4b51"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1366"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":429,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.117.189.189 dst=192.0.3.10 sport=37412 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=37412 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=51
rx_bytes=6290
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"f765e4db-4a7c-4718-a62c-b918871a13b2","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 8ff72e70-3ca2-46ba-b26a-105319ece2d7
mono_ns 12817287600697
wall_unix_ns 1793429279771230044
policy_expires_mono_ns 16379395397623
baseline_ok 1
serving 1
generation 5
reason serving
