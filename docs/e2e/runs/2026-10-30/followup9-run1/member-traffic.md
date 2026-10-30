# member-traffic

Member device and exit traffic

Status: pass

Run: 8fb30ba4-b012-42d9-ba79-4ecb46adff53

Source: 943e2a15aac3a45990abdbcdd25957376fa997e0

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18ba2-40cc-703e-810c-d1ab165bb6ce
- plan_revision_id: 01a18ba2-40cc-72d8-b83d-bb6e3918eedc
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T07:08:41Z
- binding_valid_until: 2026-10-30T08:08:34Z
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

  - ID: fvpn-exit-4722f1d69ebadb63461b0908
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-4722f1d69ebadb63461b0908
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.92.59.47 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.92.59.47/16 brd 100.92.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.92.0.0/16 dev fvnb0 proto kernel scope link src 100.92.59.47
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.92.59.47 dev fvnb0 table local proto kernel scope host src 100.92.59.47
broadcast 100.92.255.255 dev fvnb0 table local proto kernel scope link src 100.92.59.47
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":317,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":318,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":319,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":320,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.92.59.47","timeout":3591,"expires":3587}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"comment":"fvpn_class=01a18ba3-8e21-7e37-a767-a1662fe952e4:6d0a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.92.59.47"}},{"counter":"u_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.92.59.47"}},{"counter":"d_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=37
rx_bytes=3965
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
07:08:47.539403 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.644236 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.644248 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.644544 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 418
07:08:47.645408 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.646344 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 931
07:08:47.646348 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.646928 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 64
07:08:47.689293 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.689298 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 219
07:08:47.690331 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.690331 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 141
07:08:47.691205 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.692342 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 24
07:08:47.692343 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.692351 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
07:08:47.643645 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.643749 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.644280 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.644655 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 418
07:08:47.644665 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.645071 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 931
07:08:47.646371 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.647032 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 64
07:08:47.688210 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.689444 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 219
07:08:47.689472 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.689647 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 141
07:08:47.691255 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0
07:08:47.691394 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 24
07:08:47.691407 IP 192.0.3.10.443 > 100.92.59.47.60682: tcp 0
07:08:47.692387 IP 100.92.59.47.60682 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
07:08:47.643729 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 0
07:08:47.643742 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 0
07:08:47.644283 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 0
07:08:47.644657 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 418
07:08:47.644663 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 0
07:08:47.645064 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 931
07:08:47.646378 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 0
07:08:47.647034 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 64
07:08:47.688194 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 0
07:08:47.689454 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 219
07:08:47.689466 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 0
07:08:47.689640 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 141
07:08:47.691258 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 0
07:08:47.691388 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 24
07:08:47.691404 IP 192.0.3.10.443 > 192.0.3.20.60682: tcp 0
07:08:47.692390 IP 192.0.3.20.60682 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.92.59.47 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.92.59.47/16 brd 100.92.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.92.0.0/16 dev fvnb0 proto kernel scope link src 100.92.59.47
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.92.59.47 dev fvnb0 table local proto kernel scope host src 100.92.59.47
broadcast 100.92.255.255 dev fvnb0 table local proto kernel scope link src 100.92.59.47
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":317,"packets":8,"bytes":1520}},{"counter":{"family":"inet","name":"nd_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":318,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":319,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18ba3a90a7136affaeb497a47af9b","table":"fvpn","handle":320,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.92.59.47","timeout":3580,"expires":3578}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"comment":"fvpn_class=01a18ba3-8e21-7e37-a767-a1662fe952e4:6d0a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":379,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":380,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":382,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":383,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":385,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":386,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":387,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":388,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.92.59.47"}},{"counter":"u_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":390,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":392,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":393,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":394,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.92.59.47"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":395,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.92.59.47"}},{"counter":"d_01a18ba3a90a7136affaeb497a47af9b"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:6d0a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":398,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 2 CLOSE src=100.92.59.47 dst=192.0.3.10 sport=60682 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=60682 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=49
rx_bytes=5875
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"b0e12105-d4ee-495a-bbab-70415d83852b","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2633
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 65d820ba-62c2-4516-a4a3-d77d391bf9f8
mono_ns 699172839737338
wall_unix_ns 1793344133642916100
policy_expires_mono_ns 702753412531861
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
- config_applied: 3
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: not_run
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: not_run
- check_release_supported: passed
- check_source_binding: not_run
- check_storage: passed
- check_tunnel_probe: passed
- stored_lease_matches_applied_and_desired_generation: true
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 6, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T07:09:03.853417438Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T07:08:58.741972+00:00", "expires_at": "2026-10-31T07:08:58.741972+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
