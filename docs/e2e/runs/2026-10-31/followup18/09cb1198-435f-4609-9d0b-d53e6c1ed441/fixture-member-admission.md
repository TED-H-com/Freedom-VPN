# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 09cb1198-435f-4609-9d0b-d53e6c1ed441

Source: 84b6a342970ffc651aafda81d485451529902233

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
- binding_valid_from: 2026-10-31T10:25:27Z
- binding_valid_until: 2026-10-31T11:25:22Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-0631054e00307c4a0b1be192
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-0631054e00307c4a0b1be192
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.119.232.150 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.119.232.150/16 brd 100.119.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.119.0.0/16 dev fvnb0 proto kernel scope link src 100.119.232.150
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.119.232.150 dev fvnb0 table local proto kernel scope host src 100.119.232.150
broadcast 100.119.255.255 dev fvnb0 table local proto kernel scope link src 100.119.232.150
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":159,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":160,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":161,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":162,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":163,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":164,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":165,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":279,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":280,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":281,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":282,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.119.232.150","timeout":3590,"expires":3589}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":287,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":288,"comment":"fvpn_class=01a1917e-1662-7571-8a9a-60da4e86666d:a753","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":290,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":291,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":292,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":293,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":294,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":295,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":296,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":297,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":298,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":299,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.232.150"}},{"counter":"u_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":300,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":302,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":303,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":304,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":305,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.232.150"}},{"counter":"d_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":308,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=23
rx_bytes=3164
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:25:33.604480 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.610620 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.610634 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.610835 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 418
10:25:33.612038 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.612235 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 930
10:25:33.612239 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.612831 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 64
10:25:33.655370 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.655377 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 219
10:25:33.656290 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.656293 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 141
10:25:33.657071 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.657552 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 24
10:25:33.657564 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.657566 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.657570 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:25:33.610414 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.610483 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.610716 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.611961 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 418
10:25:33.611971 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.612154 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 930
10:25:33.612285 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.612890 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 64
10:25:33.654663 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.655923 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 219
10:25:33.655951 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.656072 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 141
10:25:33.657409 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.657450 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 24
10:25:33.657465 IP 192.0.3.10.443 > 100.119.232.150.46966: tcp 0
10:25:33.657653 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0
10:25:33.657661 IP 100.119.232.150.46966 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
10:25:33.610466 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 0
10:25:33.610478 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 0
10:25:33.610724 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 0
10:25:33.611963 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 418
10:25:33.611968 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 0
10:25:33.612148 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 930
10:25:33.612287 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 0
10:25:33.612892 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 64
10:25:33.654644 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 0
10:25:33.655937 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 219
10:25:33.655948 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 0
10:25:33.656053 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 141
10:25:33.657412 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 0
10:25:33.657442 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 24
10:25:33.657462 IP 192.0.3.10.443 > 192.0.3.24.46966: tcp 0
10:25:33.657655 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 0
10:25:33.657662 IP 192.0.3.24.46966 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.119.232.150 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.119.232.150/16 brd 100.119.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.119.0.0/16 dev fvnb0 proto kernel scope link src 100.119.232.150
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.119.232.150 dev fvnb0 table local proto kernel scope host src 100.119.232.150
broadcast 100.119.255.255 dev fvnb0 table local proto kernel scope link src 100.119.232.150
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":159,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":160,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":161,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":162,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":163,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":164,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":165,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":279,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":280,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":281,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1917e2af07cda883c736214db2cc1","table":"fvpn","handle":282,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.119.232.150","timeout":3585,"expires":3580}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":317,"comment":"fvpn_class=01a1917e-1662-7571-8a9a-60da4e86666d:a753","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":318,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":320,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":321,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.232.150"}},{"counter":"u_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.232.150"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.232.150"}},{"counter":"d_01a1917e2af07cda883c736214db2cc1"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a753"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- after_conntrack: ipv4     2 tcp      6 2 CLOSE src=100.119.232.150 dst=192.0.3.10 sport=46966 dport=443 src=192.0.3.10 dst=192.0.3.24 sport=443 dport=46966 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=11
tx_bytes=1760
- after_node_tunnel: rx_packets=11
rx_bytes=1760
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=33
rx_bytes=4879
tx_packets=25
tx_bytes=2903
- origin_arrival: {"probe_id":"7ec80cb6-848f-44df-8275-9a937291385b","requests":1,"source_address":"192.0.3.24"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2672
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 25878565029412
wall_unix_ns 1793442341048658332
policy_expires_mono_ns 29459539589634
baseline_ok 1
serving 1
generation 5
reason serving
