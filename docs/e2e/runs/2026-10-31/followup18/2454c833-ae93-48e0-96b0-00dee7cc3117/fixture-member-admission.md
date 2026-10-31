# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 2454c833-ae93-48e0-96b0-00dee7cc3117

Source: 114d31df188dfa71b43c5884588e8f8f9ea958b2

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
- binding_valid_from: 2026-10-31T10:31:28Z
- binding_valid_until: 2026-10-31T11:31:08Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-421c2a1a2d0ba381bbebed3e
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-421c2a1a2d0ba381bbebed3e
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.122.232.86 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.122.232.86/16 brd 100.122.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.122.0.0/16 dev fvnb0 proto kernel scope link src 100.122.232.86
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.122.232.86 dev fvnb0 table local proto kernel scope host src 100.122.232.86
broadcast 100.122.255.255 dev fvnb0 table local proto kernel scope link src 100.122.232.86
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":279,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":280,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":281,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":282,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.122.232.86","timeout":3575,"expires":3574}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":287,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":288,"comment":"fvpn_class=01a19183-5ffd-777e-b767-ddac60e1a6f2:4859","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":290,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":291,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":292,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":293,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":294,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":295,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":296,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":297,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":298,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":299,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.232.86"}},{"counter":"u_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":300,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":302,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":303,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":304,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":305,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.232.86"}},{"counter":"d_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":308,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=30
rx_bytes=3703
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:31:34.176357 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.180538 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.180551 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.180716 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 418
10:31:34.180967 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.181233 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 928
10:31:34.181237 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.181799 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 64
10:31:34.223116 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.223120 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 219
10:31:34.224184 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.224184 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 141
10:31:34.224895 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.225611 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 24
10:31:34.225611 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.225633 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
10:31:34.180139 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.180201 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.180675 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.180857 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 418
10:31:34.180869 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.181136 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 928
10:31:34.181325 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.181890 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 64
10:31:34.222663 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.223431 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 219
10:31:34.223439 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.223492 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 141
10:31:34.225297 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
10:31:34.225320 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 24
10:31:34.225330 IP 192.0.3.10.443 > 100.122.232.86.35796: tcp 0
10:31:34.226172 IP 100.122.232.86.35796 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
10:31:34.180184 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 0
10:31:34.180195 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 0
10:31:34.180680 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 0
10:31:34.180861 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 418
10:31:34.180866 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 0
10:31:34.181130 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 928
10:31:34.181327 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 0
10:31:34.181893 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 64
10:31:34.222641 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 0
10:31:34.223434 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 219
10:31:34.223437 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 0
10:31:34.223488 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 141
10:31:34.225300 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 0
10:31:34.225317 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 24
10:31:34.225328 IP 192.0.3.10.443 > 192.0.3.20.35796: tcp 0
10:31:34.226179 IP 192.0.3.20.35796 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.122.232.86 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.122.232.86/16 brd 100.122.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.122.0.0/16 dev fvnb0 proto kernel scope link src 100.122.232.86
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.122.232.86 dev fvnb0 table local proto kernel scope host src 100.122.232.86
broadcast 100.122.255.255 dev fvnb0 table local proto kernel scope link src 100.122.232.86
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":279,"packets":8,"bytes":1517}},{"counter":{"family":"inet","name":"nd_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":280,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":281,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19183acff7cfb9b3450673b0b4d24","table":"fvpn","handle":282,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.122.232.86","timeout":3570,"expires":3566}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":317,"comment":"fvpn_class=01a19183-5ffd-777e-b767-ddac60e1a6f2:4859","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":318,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":320,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":321,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.122.232.86"}},{"counter":"u_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.232.86"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.122.232.86"}},{"counter":"d_01a19183acff7cfb9b3450673b0b4d24"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4859"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.122.232.86 dst=192.0.3.10 sport=35796 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=35796 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=40
rx_bytes=5416
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"f882b595-c08f-43df-a135-e67b09cc98f8","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2630
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 26234482228878
wall_unix_ns 1793442696965857876
policy_expires_mono_ns 29806061289414
baseline_ok 1
serving 1
generation 5
reason serving
