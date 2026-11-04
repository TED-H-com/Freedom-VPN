# member-traffic

Member device and exit traffic

Status: pass

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a1a5ee-b2a2-77a7-a28f-a4711f8e93d2
- plan_revision_id: 01a1a5ee-b2a2-794f-abdb-ac69483f3346
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-11-04T09:42:11Z
- binding_valid_until: 2026-11-04T10:41:51Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- admission_poll_started_at: 2026-11-04T09:42:15.980665695Z
- admission_poll_elapsed: 87.945153ms
- admission_poll_found: true
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 1.134
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-55b03110e959b31f91f228e8
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.67.204.50 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.67.204.50/16 brd 100.67.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.67.0.0/16 dev fvnb0 proto kernel scope link src 100.67.204.50
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.67.204.50 dev fvnb0 table local proto kernel scope host src 100.67.204.50
broadcast 100.67.255.255 dev fvnb0 table local proto kernel scope link src 100.67.204.50
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":4,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":842,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":843,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":844,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":845,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":846,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":847,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":848,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":849,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1002,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1003,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1004,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1005,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":5,"flags":["timeout"],"elem":[{"elem":{"val":"100.67.204.50","timeout":595,"expires":593}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1013,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1014,"comment":"fvpn_class=01a1a5ee-7d9a-7a5c-8b57-62a8b40ea056:10c2","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1015,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1016,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1017,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1018,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1019,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1020,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1021,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1022,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1023,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1024,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1025,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1026,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"counter":"u_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1027,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1029,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1030,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1031,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1032,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1033,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1034,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1035,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1036,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1037,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1038,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1039,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1042,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=33
rx_bytes=3967
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
09:42:19.407708 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.514701 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.514739 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.514977 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 418
09:42:19.517063 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.517843 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 930
09:42:19.517854 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.518517 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 64
09:42:19.560632 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.560642 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 219
09:42:19.562987 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.562987 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 141
09:42:19.563993 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.566597 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 24
09:42:19.566597 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.566617 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
09:42:19.513655 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.513744 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.515551 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.516175 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 418
09:42:19.516195 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.516609 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 930
09:42:19.518940 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.518954 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 64
09:42:19.559670 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.561758 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 219
09:42:19.561816 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.561902 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 141
09:42:19.565407 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0
09:42:19.565469 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 24
09:42:19.565487 IP 192.0.3.10.443 > 100.67.204.50.34384: tcp 0
09:42:19.567600 IP 100.67.204.50.34384 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
09:42:19.513722 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 0
09:42:19.513737 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 0
09:42:19.515559 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 0
09:42:19.516182 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 418
09:42:19.516191 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 0
09:42:19.516601 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 930
09:42:19.518948 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 0
09:42:19.518957 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 64
09:42:19.559642 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 0
09:42:19.561782 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 219
09:42:19.561805 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 0
09:42:19.561891 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 141
09:42:19.565426 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 0
09:42:19.565462 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 24
09:42:19.565483 IP 192.0.3.10.443 > 192.0.3.20.34384: tcp 0
09:42:19.567617 IP 192.0.3.20.34384 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.67.204.50 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.67.204.50/16 brd 100.67.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.67.0.0/16 dev fvnb0 proto kernel scope link src 100.67.204.50
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.67.204.50 dev fvnb0 table local proto kernel scope host src 100.67.204.50
broadcast 100.67.255.255 dev fvnb0 table local proto kernel scope link src 100.67.204.50
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":4,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":842,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":843,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":844,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":845,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":846,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":847,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":848,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":849,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1002,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1003,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1004,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a5effe067ef6874ac0e676ee0857","table":"fvpn","handle":1005,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":5,"flags":["timeout"],"elem":[{"elem":{"val":"100.67.204.50","timeout":595,"expires":584}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1013,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1014,"comment":"fvpn_class=01a1a5ee-7d9a-7a5c-8b57-62a8b40ea056:10c2","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1015,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1016,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1017,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1018,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1019,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1020,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1021,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1022,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1023,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1024,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1025,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1026,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.204.50"}},{"counter":"u_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1027,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1029,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1030,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1031,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1032,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1033,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1034,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1035,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1036,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1037,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1038,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1039,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.204.50"}},{"counter":"d_01a1a5effe067ef6874ac0e676ee0857"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:10c2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1042,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.67.204.50 dst=192.0.3.10 sport=34384 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=34384 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=44
rx_bytes=5789
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"b99ebf2b-792d-4efa-b866-768650be0e5a","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 108787814201638
wall_unix_ns 1793785345362826673
policy_expires_mono_ns 109373696591597
baseline_ok 1
serving 1
generation 5
reason serving
- persistent_boot_recovery_config_owner_mode: 0:600
- admin_node: 200
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: ready
- lifecycle_state: registered
- publication_intent: published
- publication_observed: active
- config_desired: 5
- config_applied: 5
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: passed
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: passed
- check_release_supported: passed
- check_source_binding: passed
- check_storage: passed
- check_tunnel_probe: passed
- stored_lease_matches_applied_and_desired_generation: true
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 27, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "qos", "detail": "read_at=2026-11-04T09:42:15.507587993Z source=live candidate=5 verified=true {\"generation\":\"5\",\"phase\":\"verify\",\"reads\":16,\"objects\":4,\"truncated\":false,\"sha256\":\"406e3b17ccd7bcfaa586e929858d9a2c6319cc6b34360b1ec90f36d5392b0912\"}", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:42:15.525302753Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-11-04T09:42:15.397452+00:00", "expires_at": "2026-11-05T09:42:15.397452+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- provisioning_ready_at: 2026-11-04T09:41:06.680364Z
- populated_lease_at: 2026-11-04T09:41:55.407462Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 48.727
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-11-04T09:41:51.646653099Z","cpu_usage_usec":1610519,"memory_bytes":61812736,"open_fds":104,"processes":12,"conntrack_entries":14},{"at":"2026-11-04T09:41:56.647292189Z","cpu_usage_usec":2196631,"cpu_percent_one_core":11.720741878214609,"memory_bytes":61046784,"open_fds":104,"processes":12,"conntrack_entries":17},{"at":"2026-11-04T09:42:01.646869928Z","cpu_usage_usec":2662293,"cpu_percent_one_core":9.314026590036386,"memory_bytes":61870080,"open_fds":104,"processes":12,"conntrack_entries":21},{"at":"2026-11-04T09:42:06.650713867Z","cpu_usage_usec":3153478,"cpu_percent_one_core":9.816153460976274,"memory_bytes":62054400,"open_fds":104,"processes":12,"conntrack_entries":26},{"at":"2026-11-04T09:42:11.654077454Z","cpu_usage_usec":3654099,"cpu_percent_one_core":10.005688998911443,"memory_bytes":64077824,"open_fds":100,"processes":13,"conntrack_entries":31},{"at":"2026-11-04T09:42:16.648464945Z","cpu_usage_usec":4297756,"cpu_percent_one_core":12.887606361338294,"memory_bytes":64135168,"open_fds":90,"processes":11,"conntrack_entries":31},{"at":"2026-11-04T09:42:21.65156675Z","cpu_usage_usec":4929237,"cpu_percent_one_core":12.621789933774894,"memory_bytes":65929216,"open_fds":108,"processes":16,"conntrack_entries":42},{"at":"2026-11-04T09:42:26.645192095Z","cpu_usage_usec":5512437,"cpu_percent_one_core":11.678889778624352,"memory_bytes":63115264,"open_fds":92,"processes":12,"conntrack_entries":47},{"at":"2026-11-04T09:42:31.649962852Z","cpu_usage_usec":6056250,"cpu_percent_one_core":10.865892293655758,"memory_bytes":63967232,"open_fds":92,"processes":12,"conntrack_entries":50},{"at":"2026-11-04T09:42:36.650623088Z","cpu_usage_usec":6570423,"cpu_percent_one_core":10.282102277184183,"memory_bytes":63610880,"open_fds":92,"processes":12,"conntrack_entries":54}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
- timed_section: member-traffic-forwarding-usage
