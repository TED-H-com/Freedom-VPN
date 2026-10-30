# member-traffic

Member device and exit traffic

Status: pass

Run: 76f0725c-2264-4021-a553-60f6c921e32b

Source: 1b2f358880d38ae643865edc2bb6f19756748456

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18dda-0ef0-7d9d-9e40-b29a047b208c
- plan_revision_id: 01a18dda-0ef1-7051-b354-65ed20fb48c7
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T17:29:06Z
- binding_valid_until: 2026-10-30T18:28:47Z
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
- exit_discovery_seconds: 7.400
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-06e77ca6b138d5adfa608c71
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.73.141.175 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.73.141.175/16 brd 100.73.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.73.0.0/16 dev fvnb0 proto kernel scope link src 100.73.141.175
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.73.141.175 dev fvnb0 table local proto kernel scope host src 100.73.141.175
broadcast 100.73.255.255 dev fvnb0 table local proto kernel scope link src 100.73.141.175
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":396,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":397,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":398,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":399,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.73.141.175","timeout":3572,"expires":3568}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":430,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":431,"comment":"fvpn_class=01a18dd9-dcea-7957-8e9d-432d77899c54:eba6","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":432,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":433,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":434,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":435,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":436,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":437,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":438,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":439,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":440,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":441,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":442,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.141.175"}},{"counter":"u_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":443,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":445,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":446,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":447,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":448,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.141.175"}},{"counter":"d_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":451,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=41
rx_bytes=4336
tx_packets=23
tx_bytes=2870
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
17:29:19.906040 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.011062 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.011084 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.011400 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 418
17:29:20.011932 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.012071 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 930
17:29:20.012076 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.012714 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 64
17:29:20.053410 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.053433 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 219
17:29:20.053866 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.053867 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 141
17:29:20.054595 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.054746 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 24
17:29:20.054746 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.054765 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
17:29:20.010800 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.010900 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.011099 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.011546 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 418
17:29:20.011558 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.012053 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 930
17:29:20.012089 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.012831 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 64
17:29:20.053221 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.053632 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 219
17:29:20.053673 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.053807 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 141
17:29:20.054635 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
17:29:20.054692 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 24
17:29:20.054712 IP 192.0.3.10.443 > 100.73.141.175.46694: tcp 0
17:29:20.054810 IP 100.73.141.175.46694 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
17:29:20.010874 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 0
17:29:20.010894 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 0
17:29:20.011102 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 0
17:29:20.011548 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 418
17:29:20.011556 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 0
17:29:20.012043 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 930
17:29:20.012092 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 0
17:29:20.012845 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 64
17:29:20.053197 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 0
17:29:20.053651 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 219
17:29:20.053668 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 0
17:29:20.053800 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 141
17:29:20.054644 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 0
17:29:20.054684 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 24
17:29:20.054709 IP 192.0.3.10.443 > 192.0.3.20.46694: tcp 0
17:29:20.054812 IP 192.0.3.20.46694 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.73.141.175 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.73.141.175/16 brd 100.73.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.73.0.0/16 dev fvnb0 proto kernel scope link src 100.73.141.175
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.73.141.175 dev fvnb0 table local proto kernel scope host src 100.73.141.175
broadcast 100.73.255.255 dev fvnb0 table local proto kernel scope link src 100.73.141.175
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":396,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":397,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":398,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18ddbaba47d9a8de0d297f487df99","table":"fvpn","handle":399,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.73.141.175","timeout":3562,"expires":3559}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":482,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":483,"comment":"fvpn_class=01a18dd9-dcea-7957-8e9d-432d77899c54:eba6","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":484,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":485,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":486,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":487,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":488,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":489,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":490,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":491,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":492,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":493,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":494,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.73.141.175"}},{"counter":"u_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":495,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":497,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":498,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":499,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.141.175"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":500,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.73.141.175"}},{"counter":"d_01a18ddbaba47d9a8de0d297f487df99"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:eba6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":503,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.73.141.175 dst=192.0.3.10 sport=46694 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=46694 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=51
rx_bytes=6051
tx_packets=33
tx_bytes=4179
- origin_arrival: {"probe_id":"d4a08190-b8e0-4f74-88ed-538a5ae559c3","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id e1542154-508e-4eda-ac37-55f8fd3853cd
mono_ns 736404258185798
wall_unix_ns 1793381365061364687
policy_expires_mono_ns 739966417450123
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 5, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:14.866182003Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T17:29:09.767633+00:00", "expires_at": "2026-10-31T17:29:09.767633+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-30T17:28:47.160489216Z","cpu_usage_usec":856994,"memory_bytes":53264384,"open_fds":87,"processes":10,"conntrack_entries":134},{"at":"2026-10-30T17:28:52.160597851Z","cpu_usage_usec":972434,"cpu_percent_one_core":2.3087498377922744,"memory_bytes":54083584,"open_fds":87,"processes":10,"conntrack_entries":100},{"at":"2026-10-30T17:28:57.162498832Z","cpu_usage_usec":1028158,"cpu_percent_one_core":1.114056439974936,"memory_bytes":53985280,"open_fds":87,"processes":10,"conntrack_entries":94},{"at":"2026-10-30T17:29:02.16352712Z","cpu_usage_usec":1088226,"cpu_percent_one_core":1.2011129819867958,"memory_bytes":54910976,"open_fds":87,"processes":10,"conntrack_entries":101},{"at":"2026-10-30T17:29:07.161152879Z","cpu_usage_usec":1146912,"cpu_percent_one_core":1.174277603606373,"memory_bytes":54951936,"open_fds":87,"processes":10,"conntrack_entries":96},{"at":"2026-10-30T17:29:12.159427648Z","cpu_usage_usec":6454694,"cpu_percent_one_core":106.19228124311226,"memory_bytes":59142144,"open_fds":86,"processes":10,"conntrack_entries":104},{"at":"2026-10-30T17:29:17.160671374Z","cpu_usage_usec":12604465,"cpu_percent_one_core":122.96483308800057,"memory_bytes":58863616,"open_fds":86,"processes":10,"conntrack_entries":108},{"at":"2026-10-30T17:29:22.163049601Z","cpu_usage_usec":18932165,"cpu_percent_one_core":126.4938337898295,"memory_bytes":62767104,"open_fds":102,"processes":14,"conntrack_entries":89},{"at":"2026-10-30T17:29:27.158903174Z","cpu_usage_usec":22616978,"cpu_percent_one_core":73.7574259564873,"memory_bytes":59711488,"open_fds":89,"processes":10,"conntrack_entries":96}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
