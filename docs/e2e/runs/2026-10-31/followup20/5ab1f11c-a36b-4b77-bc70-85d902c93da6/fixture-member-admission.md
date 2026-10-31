# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 5ab1f11c-a36b-4b77-bc70-85d902c93da6

Source: ab436054042dbb4f141c7cdd79aec4d636a894f2

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
- binding_valid_from: 2026-10-31T15:52:48Z
- binding_valid_until: 2026-10-31T16:52:30Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-1d740cb20db9b961f5c980a3
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-1d740cb20db9b961f5c980a3
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.96.24.194 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.96.24.194/16 brd 100.96.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.96.0.0/16 dev fvnb0 proto kernel scope link src 100.96.24.194
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.96.24.194 dev fvnb0 table local proto kernel scope host src 100.96.24.194
broadcast 100.96.255.255 dev fvnb0 table local proto kernel scope link src 100.96.24.194
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":257,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":258,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":259,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":260,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.96.24.194","timeout":59,"expires":58}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":265,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":266,"comment":"fvpn_class=01a192a9-98b6-7bf5-90ea-faeb7d38ef9c:ab3c","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":267,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":268,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":269,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":270,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":271,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":272,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":273,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":274,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":275,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":276,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":277,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.24.194"}},{"counter":"u_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":278,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":280,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":281,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":282,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":283,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.24.194"}},{"counter":"d_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":286,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=3489
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
15:52:56.280936 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.284969 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.284980 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.285190 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 418
15:52:56.285573 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.285755 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 929
15:52:56.285759 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.286284 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 64
15:52:56.330444 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.330463 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 219
15:52:56.332545 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.332545 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 141
15:52:56.333664 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.334636 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 24
15:52:56.334657 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.334662 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.334665 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
15:52:56.284748 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.284831 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.285119 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.285344 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 418
15:52:56.285359 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.285628 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 929
15:52:56.285883 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.289014 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 64
15:52:56.329666 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.332046 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 219
15:52:56.332080 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.332371 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 141
15:52:56.333973 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.334041 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 24
15:52:56.334060 IP 192.0.3.10.443 > 100.96.24.194.55750: tcp 0
15:52:56.335176 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0
15:52:56.335176 IP 100.96.24.194.55750 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
15:52:56.284811 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 0
15:52:56.284824 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 0
15:52:56.285126 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 0
15:52:56.285349 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 418
15:52:56.285355 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 0
15:52:56.285620 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 929
15:52:56.285886 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 0
15:52:56.289019 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 64
15:52:56.329645 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 0
15:52:56.332063 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 219
15:52:56.332075 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 0
15:52:56.332362 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 141
15:52:56.333983 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 0
15:52:56.334034 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 24
15:52:56.334057 IP 192.0.3.10.443 > 192.0.3.20.55750: tcp 0
15:52:56.335185 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 0
15:52:56.335187 IP 192.0.3.20.55750 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.96.24.194 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.96.24.194/16 brd 100.96.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.96.0.0/16 dev fvnb0 proto kernel scope link src 100.96.24.194
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.96.24.194 dev fvnb0 table local proto kernel scope host src 100.96.24.194
broadcast 100.96.255.255 dev fvnb0 table local proto kernel scope link src 100.96.24.194
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":71,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":72,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":73,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":74,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":75,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":76,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":77,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":257,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":258,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":259,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192a9dd317600bfd7ed7cf89fb9e4","table":"fvpn","handle":260,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.96.24.194","timeout":49,"expires":49}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"comment":"fvpn_class=01a192a9-98b6-7bf5-90ea-faeb7d38ef9c:ab3c","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.24.194"}},{"counter":"u_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.24.194"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.24.194"}},{"counter":"d_01a192a9dd317600bfd7ed7cf89fb9e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:ab3c"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.96.24.194 dst=192.0.3.10 sport=55750 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=55750 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=11
tx_bytes=1760
- after_node_tunnel: rx_packets=11
rx_bytes=1760
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=39
rx_bytes=5310
tx_packets=25
tx_bytes=2903
- origin_arrival: {"probe_id":"9e6f2bac-6489-4cc9-8bbd-0cefa3efcdce","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2671
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 45520890870381
wall_unix_ns 1793461983374499236
policy_expires_mono_ns 45570842475026
baseline_ok 1
serving 1
generation 6
reason serving
