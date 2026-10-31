# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 04de29d3-e56b-4a32-b678-21462845ddb6

Source: fb5b8fcee72bd71b3bd0a4536af9049df4055096

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
- binding_valid_from: 2026-10-31T14:13:03Z
- binding_valid_until: 2026-10-31T15:12:47Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-0e0370d3611ef5cc579d2c2a
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-0e0370d3611ef5cc579d2c2a
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.119.193.232 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.119.193.232/16 brd 100.119.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.119.0.0/16 dev fvnb0 proto kernel scope link src 100.119.193.232
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.119.193.232 dev fvnb0 table local proto kernel scope host src 100.119.193.232
broadcast 100.119.255.255 dev fvnb0 table local proto kernel scope link src 100.119.193.232
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":159,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":160,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":161,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":162,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":163,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":164,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":165,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":411,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":412,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":413,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":414,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.119.193.232","timeout":596,"expires":595}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"comment":"fvpn_class=01a1924e-4e61-7626-bb8b-147951a6220d:eba7","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":422,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":423,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":424,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":426,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":427,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":428,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":429,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":430,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":431,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.193.232"}},{"counter":"u_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":432,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":434,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":435,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":436,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":437,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.193.232"}},{"counter":"d_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":440,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=26
rx_bytes=3408
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:13:09.629819 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.833222 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.833237 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.833484 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 418
14:13:09.833551 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.833877 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 930
14:13:09.833883 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.834501 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 64
14:13:09.874804 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.874815 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 219
14:13:09.875361 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.875570 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 141
14:13:09.876250 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.876877 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 24
14:13:09.876877 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.876888 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
14:13:09.832923 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.833005 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.833264 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.833522 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 418
14:13:09.833533 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.833856 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 930
14:13:09.833898 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.834531 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 64
14:13:09.874660 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.874942 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 219
14:13:09.874954 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.875190 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 141
14:13:09.876649 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
14:13:09.876723 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 24
14:13:09.876743 IP 192.0.3.10.443 > 100.119.193.232.46550: tcp 0
14:13:09.877083 IP 100.119.193.232.46550 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
14:13:09.832986 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 0
14:13:09.832999 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 0
14:13:09.833267 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 0
14:13:09.833525 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 418
14:13:09.833530 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 0
14:13:09.833850 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 930
14:13:09.833902 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 0
14:13:09.834533 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 64
14:13:09.874641 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 0
14:13:09.874947 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 219
14:13:09.874951 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 0
14:13:09.875185 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 141
14:13:09.876666 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 0
14:13:09.876713 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 24
14:13:09.876740 IP 192.0.3.10.443 > 192.0.3.24.46550: tcp 0
14:13:09.877087 IP 192.0.3.24.46550 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.119.193.232 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.119.193.232/16 brd 100.119.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.119.0.0/16 dev fvnb0 proto kernel scope link src 100.119.193.232
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.119.193.232 dev fvnb0 table local proto kernel scope host src 100.119.193.232
broadcast 100.119.255.255 dev fvnb0 table local proto kernel scope link src 100.119.193.232
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":159,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":160,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":161,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":162,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":163,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":164,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":165,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":411,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":412,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":413,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1924e8bf57b59bc65ece94dd391ca","table":"fvpn","handle":414,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.119.193.232","timeout":591,"expires":586}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":448,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":449,"comment":"fvpn_class=01a1924e-4e61-7626-bb8b-147951a6220d:eba7","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":450,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":451,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":452,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":453,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":454,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":455,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":456,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":457,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":458,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":459,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":460,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.119.193.232"}},{"counter":"u_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":461,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":463,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":464,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":465,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.193.232"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":466,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.119.193.232"}},{"counter":"d_01a1924e8bf57b59bc65ece94dd391ca"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba7"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":469,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.119.193.232 dst=192.0.3.10 sport=46550 dport=443 src=192.0.3.10 dst=192.0.3.24 sport=443 dport=46550 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=36
rx_bytes=5123
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"e9d340b9-f5cf-47c9-a812-d5726551bd3e","requests":1,"source_address":"192.0.3.24"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 39529347909576
wall_unix_ns 1793455991831538457
policy_expires_mono_ns 40121164948462
baseline_ok 1
serving 1
generation 5
reason serving
