# member-traffic

Member device and exit traffic

Status: pass

Run: 14231a83-130d-40fb-a9b4-0d3851520b67

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18c63-9e7f-716e-ba00-7b422e8985cb
- plan_revision_id: 01a18c63-9e7f-7446-9f5b-407c7a8e8f92
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T10:40:09Z
- binding_valid_until: 2026-10-30T11:39:49Z
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

  - ID: fvpn-exit-e5f30c67c242b366d6956762
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-e5f30c67c242b366d6956762
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.111.6.238 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.111.6.238/16 brd 100.111.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.111.0.0/16 dev fvnb0 proto kernel scope link src 100.111.6.238
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.111.6.238 dev fvnb0 table local proto kernel scope host src 100.111.6.238
broadcast 100.111.255.255 dev fvnb0 table local proto kernel scope link src 100.111.6.238
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":396,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":397,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":398,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":399,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.111.6.238","timeout":3577,"expires":3574}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":404,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":405,"comment":"fvpn_class=01a18c64-f55a-714c-a895-bf42de1578ac:7bab","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":406,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":407,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":408,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.111.6.238"}},{"counter":"u_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":422,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.111.6.238"}},{"counter":"d_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=39
rx_bytes=4187
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
10:40:14.738396 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.742808 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.742819 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.742992 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 418
10:40:14.743040 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.743318 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 930
10:40:14.743329 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.743961 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 64
10:40:14.784412 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.784421 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 219
10:40:14.784688 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.784693 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 141
10:40:14.785326 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.785605 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 24
10:40:14.785606 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.785619 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:40:14.742714 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.742779 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.742839 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.743016 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 418
10:40:14.743027 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.743291 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 930
10:40:14.743469 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.743993 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 64
10:40:14.784207 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.784534 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 219
10:40:14.784563 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.784672 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 141
10:40:14.785369 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0
10:40:14.785461 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 24
10:40:14.785495 IP 192.0.3.10.443 > 100.111.6.238.34504: tcp 0
10:40:14.785786 IP 100.111.6.238.34504 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
10:40:14.742762 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 0
10:40:14.742774 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 0
10:40:14.742843 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 0
10:40:14.743019 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 418
10:40:14.743025 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 0
10:40:14.743285 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 930
10:40:14.743476 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 0
10:40:14.743996 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 64
10:40:14.784192 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 0
10:40:14.784547 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 219
10:40:14.784559 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 0
10:40:14.784666 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 141
10:40:14.785377 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 0
10:40:14.785440 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 24
10:40:14.785487 IP 192.0.3.10.443 > 192.0.3.20.34504: tcp 0
10:40:14.785790 IP 192.0.3.20.34504 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.111.6.238 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.111.6.238/16 brd 100.111.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.111.0.0/16 dev fvnb0 proto kernel scope link src 100.111.6.238
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.111.6.238 dev fvnb0 table local proto kernel scope host src 100.111.6.238
broadcast 100.111.255.255 dev fvnb0 table local proto kernel scope link src 100.111.6.238
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":396,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":397,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":398,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18c6545187052a4cf78b6cda863ee","table":"fvpn","handle":399,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.111.6.238","timeout":3567,"expires":3566}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":489,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":490,"comment":"fvpn_class=01a18c64-f55a-714c-a895-bf42de1578ac:7bab","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":491,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":492,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":493,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":494,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":495,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":496,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":497,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":498,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":499,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":500,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":501,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.111.6.238"}},{"counter":"u_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":502,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":504,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":505,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":506,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.111.6.238"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":507,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.111.6.238"}},{"counter":"d_01a18c6545187052a4cf78b6cda863ee"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7bab"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":510,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.111.6.238 dst=192.0.3.10 sport=34504 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=34504 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=50
rx_bytes=6009
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"15f3b9e5-208e-4e97-adc2-09c9c0c35a1e","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 2798153d-6853-4098-81bc-57c9ae011ee4
mono_ns 711860607037956
wall_unix_ns 1793356821410216721
policy_expires_mono_ns 715428284493563
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 6, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T10:40:46.729667098Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T10:40:21.351699+00:00", "expires_at": "2026-10-31T10:40:21.351699+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
