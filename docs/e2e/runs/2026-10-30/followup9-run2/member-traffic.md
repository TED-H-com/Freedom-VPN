# member-traffic

Member device and exit traffic

Status: pass

Run: 65712989-0b05-4117-9a50-689499cd31be

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18bb2-bdcd-78c0-8f83-4d2cbf64b5cd
- plan_revision_id: 01a18bb2-bdcd-7a81-97c7-adfa5ece2657
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T07:26:56Z
- binding_valid_until: 2026-10-30T08:26:36Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-2f0cef9d2e48f617d4665a65
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-2f0cef9d2e48f617d4665a65
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.102.46.45 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.102.46.45/16 brd 100.102.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.102.0.0/16 dev fvnb0 proto kernel scope link src 100.102.46.45
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.102.46.45 dev fvnb0 table local proto kernel scope host src 100.102.46.45
broadcast 100.102.255.255 dev fvnb0 table local proto kernel scope link src 100.102.46.45
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":374,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":375,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":376,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":377,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.102.46.45","timeout":3578,"expires":3576}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":382,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":383,"comment":"fvpn_class=01a18bb4-0fe1-79dd-be7b-fd061441a7bb:90a2","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":385,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":386,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":387,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":388,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":390,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":391,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":392,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":393,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":394,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.46.45"}},{"counter":"u_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":395,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":397,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":398,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":399,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":400,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.46.45"}},{"counter":"d_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":403,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=40
rx_bytes=4229
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
07:27:00.678697 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.683076 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.683089 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.683267 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 418
07:27:00.683308 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.683572 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 929
07:27:00.683575 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.684100 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 64
07:27:00.724299 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.724323 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 219
07:27:00.724605 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.724605 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 141
07:27:00.725139 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.725273 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 24
07:27:00.725273 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.725288 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
07:27:00.682991 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.683057 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.683100 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.683284 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 418
07:27:00.683295 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.683556 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 929
07:27:00.683587 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.684139 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 64
07:27:00.724205 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.724365 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 219
07:27:00.724380 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.724459 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 141
07:27:00.725161 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0
07:27:00.725180 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 24
07:27:00.725191 IP 192.0.3.10.443 > 100.102.46.45.40984: tcp 0
07:27:00.725375 IP 100.102.46.45.40984 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
07:27:00.683039 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 0
07:27:00.683050 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 0
07:27:00.683103 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 0
07:27:00.683287 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 418
07:27:00.683292 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 0
07:27:00.683549 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 929
07:27:00.683595 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 0
07:27:00.684141 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 64
07:27:00.724190 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 0
07:27:00.724370 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 219
07:27:00.724378 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 0
07:27:00.724450 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 141
07:27:00.725163 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 0
07:27:00.725176 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 24
07:27:00.725189 IP 192.0.3.10.443 > 192.0.3.20.40984: tcp 0
07:27:00.725377 IP 192.0.3.20.40984 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.102.46.45 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.102.46.45/16 brd 100.102.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.102.0.0/16 dev fvnb0 proto kernel scope link src 100.102.46.45
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.102.46.45 dev fvnb0 table local proto kernel scope host src 100.102.46.45
broadcast 100.102.255.255 dev fvnb0 table local proto kernel scope link src 100.102.46.45
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":374,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":375,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":376,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18bb45e9a797eab2f0c250ad090b3","table":"fvpn","handle":377,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.102.46.45","timeout":3572,"expires":3567}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":408,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"comment":"fvpn_class=01a18bb4-0fe1-79dd-be7b-fd061441a7bb:90a2","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.102.46.45"}},{"counter":"u_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":423,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":424,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.46.45"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":426,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.102.46.45"}},{"counter":"d_01a18bb45e9a797eab2f0c250ad090b3"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:90a2"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":429,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.102.46.45 dst=192.0.3.10 sport=40984 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=40984 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=51
rx_bytes=6050
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"d1f188d7-ad56-4752-9483-2350f50eb5f8","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2631
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id ec66f1e0-796e-4a2e-91a2-623ce4e464c1
mono_ns 700262263898479
wall_unix_ns 1793345223067077272
policy_expires_mono_ns 703835205879053
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 5, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:27:03.017054879Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T07:26:57.916342+00:00", "expires_at": "2026-10-31T07:26:57.916342+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
