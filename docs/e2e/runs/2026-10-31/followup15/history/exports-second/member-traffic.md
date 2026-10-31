# member-traffic

Member device and exit traffic

Status: pass

Run: 630b9f07-62d2-4db1-9737-9060732db1f1

Source: 3c25ce9daa081874988c9b09bc8c26e66d3c1b46

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a1901d-0003-74f2-a60f-f9e65dea3838
- plan_revision_id: 01a1901d-0003-7721-88a3-a7bc4bfd38d7
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T04:00:36Z
- binding_valid_until: 2026-10-31T05:00:27Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 15.873
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-ee0251c1e884a659c42425cd
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.68.156.235 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.68.156.235/16 brd 100.68.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.68.0.0/16 dev fvnb0 proto kernel scope link src 100.68.156.235
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.68.156.235 dev fvnb0 table local proto kernel scope host src 100.68.156.235
broadcast 100.68.255.255 dev fvnb0 table local proto kernel scope link src 100.68.156.235
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":248,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":251,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.68.156.235","timeout":3570,"expires":3568}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"comment":"fvpn_class=01a1901d-b2e6-792c-ada7-db6717cb69e7:629e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.156.235"}},{"counter":"u_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":347,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":349,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":350,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.156.235"}},{"counter":"d_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=34
rx_bytes=3791
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
04:01:00.357332 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.561809 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.561824 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.562018 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 418
04:01:00.562058 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.562328 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 930
04:01:00.562331 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.562919 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 64
04:01:00.604788 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.604798 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 219
04:01:00.605128 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.605283 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 141
04:01:00.605872 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.606001 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 24
04:01:00.606010 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.606119 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.606122 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:01:00.561701 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.561787 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.561839 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.562036 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 418
04:01:00.562046 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.562311 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 930
04:01:00.562343 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.563192 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 64
04:01:00.604671 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.605081 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 219
04:01:00.605102 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.605251 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 141
04:01:00.605939 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.605982 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 24
04:01:00.605996 IP 192.0.3.10.443 > 100.68.156.235.44244: tcp 0
04:01:00.606023 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0
04:01:00.606134 IP 100.68.156.235.44244 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
04:01:00.561765 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 0
04:01:00.561781 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 0
04:01:00.561843 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 0
04:01:00.562038 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 418
04:01:00.562044 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 0
04:01:00.562301 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 930
04:01:00.562345 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 0
04:01:00.563195 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 64
04:01:00.604642 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 0
04:01:00.605089 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 219
04:01:00.605099 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 0
04:01:00.605242 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 141
04:01:00.605949 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 0
04:01:00.605977 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 24
04:01:00.605994 IP 192.0.3.10.443 > 192.0.3.20.44244: tcp 0
04:01:00.606025 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 0
04:01:00.606136 IP 192.0.3.20.44244 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.68.156.235 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.68.156.235/16 brd 100.68.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.68.0.0/16 dev fvnb0 proto kernel scope link src 100.68.156.235
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.68.156.235 dev fvnb0 table local proto kernel scope host src 100.68.156.235
broadcast 100.68.255.255 dev fvnb0 table local proto kernel scope link src 100.68.156.235
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":248,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1901dd4407ce28a64973d9eacd480","table":"fvpn","handle":251,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.68.156.235","timeout":3565,"expires":3560}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"comment":"fvpn_class=01a1901d-b2e6-792c-ada7-db6717cb69e7:629e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.156.235"}},{"counter":"u_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.156.235"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.156.235"}},{"counter":"d_01a1901dd4407ce28a64973d9eacd480"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:629e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: ipv4     2 tcp      6 3 CLOSE src=100.68.156.235 dst=192.0.3.10 sport=44244 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=44244 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=10
tx_bytes=1668
- after_node_tunnel: rx_packets=10
rx_bytes=1668
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=43
rx_bytes=5464
tx_packets=24
tx_bytes=2861
- origin_arrival: {"probe_id":"391baf2c-bfcc-4264-8456-1a695f4259dd","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2672
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id c08a27f2-bb2d-4d14-92e5-b1a13608b2dc
mono_ns 2800172452941
wall_unix_ns 1793419262656082374
policy_expires_mono_ns 6365402939604
baseline_ok 1
serving 1
generation 6
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
- config_desired: 6
- config_applied: 6
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 4, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:00:57.543506889Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T04:00:41.69912+00:00", "expires_at": "2026-11-01T04:00:41.69912+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
- provisioning_ready_at: 2026-10-31T03:59:49.772832Z
- populated_lease_at: 2026-10-31T04:00:31.347828Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 41.575
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T04:00:27.823340455Z","cpu_usage_usec":3573794,"memory_bytes":58150912,"open_fds":84,"processes":11,"conntrack_entries":4},{"at":"2026-10-31T04:00:32.82503406Z","cpu_usage_usec":4017295,"cpu_percent_one_core":8.867016555285376,"memory_bytes":58200064,"open_fds":84,"processes":11,"conntrack_entries":4},{"at":"2026-10-31T04:00:37.82565005Z","cpu_usage_usec":4421502,"cpu_percent_one_core":8.083144172804198,"memory_bytes":58011648,"open_fds":84,"processes":11,"conntrack_entries":9},{"at":"2026-10-31T04:00:42.828143669Z","cpu_usage_usec":4884049,"cpu_percent_one_core":9.246328635846682,"memory_bytes":60665856,"open_fds":84,"processes":11,"conntrack_entries":10},{"at":"2026-10-31T04:00:47.82993037Z","cpu_usage_usec":5309377,"cpu_percent_one_core":8.503521349980094,"memory_bytes":60231680,"open_fds":84,"processes":11,"conntrack_entries":14},{"at":"2026-10-31T04:00:52.833146334Z","cpu_usage_usec":5732638,"cpu_percent_one_core":8.45977873123048,"memory_bytes":59551744,"open_fds":84,"processes":11,"conntrack_entries":18},{"at":"2026-10-31T04:00:57.825845475Z","cpu_usage_usec":6134686,"cpu_percent_one_core":8.052718352251299,"memory_bytes":59826176,"open_fds":84,"processes":11,"conntrack_entries":23},{"at":"2026-10-31T04:01:02.827611282Z","cpu_usage_usec":6678898,"cpu_percent_one_core":10.880397463599198,"memory_bytes":62861312,"open_fds":100,"processes":15,"conntrack_entries":35},{"at":"2026-10-31T04:01:07.826729957Z","cpu_usage_usec":7219324,"cpu_percent_one_core":10.810425499650696,"memory_bytes":61661184,"open_fds":84,"processes":11,"conntrack_entries":35},{"at":"2026-10-31T04:01:12.825521Z","cpu_usage_usec":7598720,"cpu_percent_one_core":7.589755137520358,"memory_bytes":63934464,"open_fds":82,"processes":11,"conntrack_entries":39},{"at":"2026-10-31T04:01:17.823858349Z","cpu_usage_usec":7935817,"cpu_percent_one_core":6.744182644403581,"memory_bytes":76886016,"open_fds":82,"processes":11,"conntrack_entries":38},{"at":"2026-10-31T04:01:22.83085355Z","cpu_usage_usec":8294559,"cpu_percent_one_core":7.164816134202642,"memory_bytes":63602688,"open_fds":82,"processes":11,"conntrack_entries":42},{"at":"2026-10-31T04:01:27.824770805Z","cpu_usage_usec":8685359,"cpu_percent_one_core":7.825520128687033,"memory_bytes":62132224,"open_fds":82,"processes":11,"conntrack_entries":46},{"at":"2026-10-31T04:01:32.82508667Z","cpu_usage_usec":9057196,"cpu_percent_one_core":7.436270228500855,"memory_bytes":61452288,"open_fds":82,"processes":11,"conntrack_entries":50},{"at":"2026-10-31T04:01:37.827533399Z","cpu_usage_usec":9446027,"cpu_percent_one_core":7.772816404938072,"memory_bytes":60555264,"open_fds":82,"processes":11,"conntrack_entries":52},{"at":"2026-10-31T04:01:42.825456553Z","cpu_usage_usec":9821166,"cpu_percent_one_core":7.50589771873071,"memory_bytes":63508480,"open_fds":82,"processes":11,"conntrack_entries":56},{"at":"2026-10-31T04:01:47.82666077Z","cpu_usage_usec":10216508,"cpu_percent_one_core":7.904936148301261,"memory_bytes":66199552,"open_fds":82,"processes":11,"conntrack_entries":61}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
