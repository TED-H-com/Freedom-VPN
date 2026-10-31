# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: b4d2d001-a781-4df5-80d2-f7aad09f5b55

Source: de2d40329db5e3a2d9ab0525afd5a7f6016f095e

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
- binding_valid_from: 2026-10-31T15:48:46Z
- binding_valid_until: 2026-10-31T16:48:39Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-0a694e0bc7d381319a7d1be9
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-0a694e0bc7d381319a7d1be9
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.102.126.207 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.102.126.207/16 brd 100.102.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.102.0.0/16 dev fvnb0 proto kernel scope link src 100.102.126.207
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.102.126.207 dev fvnb0 table local proto kernel scope host src 100.102.126.207
broadcast 100.102.255.255 dev fvnb0 table local proto kernel scope link src 100.102.126.207
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":213,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":214,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":215,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":216,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.102.126.207","timeout":59,"expires":57}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":221,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":222,"comment":"fvpn_class=01a192a6-10f3-70be-87d1-12d7ba73d462:3a0d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":223,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":224,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":225,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":226,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":227,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":228,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":229,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":230,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":231,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":232,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":233,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.126.207"}},{"counter":"u_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":234,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":236,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":237,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":238,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":239,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.126.207"}},{"counter":"d_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":242,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=28
rx_bytes=3472
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
15:48:50.646415 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.851152 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.851179 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.851445 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 418
15:48:50.852095 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.852505 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 929
15:48:50.852515 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.853143 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 64
15:48:50.894232 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.894256 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 219
15:48:50.895253 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.895573 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 141
15:48:50.896356 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.897100 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 24
15:48:50.897100 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.897122 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
15:48:50.850994 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.851110 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.851467 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.851790 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 418
15:48:50.851813 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.852105 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 929
15:48:50.853408 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.853409 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 64
15:48:50.893681 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.894608 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 219
15:48:50.894685 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.894948 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 141
15:48:50.896624 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
15:48:50.896849 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 24
15:48:50.896865 IP 192.0.3.10.443 > 100.102.126.207.48984: tcp 0
15:48:50.897280 IP 100.102.126.207.48984 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
15:48:50.851081 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 0
15:48:50.851103 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 0
15:48:50.851480 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 0
15:48:50.851800 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 418
15:48:50.851810 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 0
15:48:50.852097 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 929
15:48:50.853431 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 0
15:48:50.853436 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 64
15:48:50.893651 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 0
15:48:50.894651 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 219
15:48:50.894678 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 0
15:48:50.894939 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 141
15:48:50.896658 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 0
15:48:50.896839 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 24
15:48:50.896862 IP 192.0.3.10.443 > 192.0.3.20.48984: tcp 0
15:48:50.897290 IP 192.0.3.20.48984 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.102.126.207 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.102.126.207/16 brd 100.102.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.102.0.0/16 dev fvnb0 proto kernel scope link src 100.102.126.207
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.102.126.207 dev fvnb0 table local proto kernel scope host src 100.102.126.207
broadcast 100.102.255.255 dev fvnb0 table local proto kernel scope link src 100.102.126.207
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":213,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":214,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":215,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192a62cc1763bafebf4b7e215c1ac","table":"fvpn","handle":216,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.102.126.207","timeout":49,"expires":49}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":279,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":280,"comment":"fvpn_class=01a192a6-10f3-70be-87d1-12d7ba73d462:3a0d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":281,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":282,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":283,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":284,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":285,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":286,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":287,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":288,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":290,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":291,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.126.207"}},{"counter":"u_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":292,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":294,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":295,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":296,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.126.207"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":297,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.126.207"}},{"counter":"d_01a192a62cc1763bafebf4b7e215c1ac"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:3a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":300,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 3 CLOSE src=100.102.126.207 dst=192.0.3.10 sport=48984 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=48984 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=40
rx_bytes=5380
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"671965dd-7d67-45c9-ba43-426abe12f4dd","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2631
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 45274734691463
wall_unix_ns 1793461737218320384
policy_expires_mono_ns 45324699780999
baseline_ok 1
serving 1
generation 5
reason serving
