# member-traffic

Member device and exit traffic

Status: pass

Run: 926791b9-36ef-4051-9d44-cd7eafdc8792

Source: 9e493e8f6407dd7f091963fb7e133a11123305c3

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a1900a-fb42-78ef-ae12-96061a03bdc1
- plan_revision_id: 01a1900a-fb42-7dea-9e87-1bbb741b5235
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T03:40:58Z
- binding_valid_until: 2026-10-31T04:40:48Z
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
- exit_discovery_seconds: 14.839
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-3cd2e829d415af47f76a0064
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.82.60.13 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.82.60.13/16 brd 100.82.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.82.0.0/16 dev fvnb0 proto kernel scope link src 100.82.60.13
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.82.60.13 dev fvnb0 table local proto kernel scope host src 100.82.60.13
broadcast 100.82.255.255 dev fvnb0 table local proto kernel scope link src 100.82.60.13
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":248,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":251,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.82.60.13","timeout":3571,"expires":3570}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"comment":"fvpn_class=01a1900b-b48f-7b00-94e6-0b5c2ccd4db5:e09a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.82.60.13"}},{"counter":"u_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":347,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":349,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":350,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.82.60.13"}},{"counter":"d_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=4266
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
03:41:18.932709 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.936182 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.936193 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.936381 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 418
03:41:18.936609 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.936805 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 930
03:41:18.936810 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.937403 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 64
03:41:18.978769 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.978776 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 219
03:41:18.978862 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.979129 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 141
03:41:18.979704 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.980007 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 24
03:41:18.980008 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.980017 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
03:41:18.936099 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.936162 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.936206 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.936581 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 418
03:41:18.936591 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.936783 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 930
03:41:18.936822 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.937434 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 64
03:41:18.978664 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.978820 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 219
03:41:18.978844 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.979101 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 141
03:41:18.979740 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
03:41:18.979879 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 24
03:41:18.979892 IP 192.0.3.10.443 > 100.82.60.13.49686: tcp 0
03:41:18.980038 IP 100.82.60.13.49686 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
03:41:18.936145 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 0
03:41:18.936156 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 0
03:41:18.936215 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 0
03:41:18.936584 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 418
03:41:18.936587 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 0
03:41:18.936776 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 930
03:41:18.936824 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 0
03:41:18.937436 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 64
03:41:18.978645 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 0
03:41:18.978830 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 219
03:41:18.978840 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 0
03:41:18.979088 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 141
03:41:18.979745 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 0
03:41:18.979871 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 24
03:41:18.979890 IP 192.0.3.10.443 > 192.0.3.20.49686: tcp 0
03:41:18.980041 IP 192.0.3.20.49686 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.82.60.13 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.82.60.13/16 brd 100.82.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.82.0.0/16 dev fvnb0 proto kernel scope link src 100.82.60.13
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.82.60.13 dev fvnb0 table local proto kernel scope host src 100.82.60.13
broadcast 100.82.255.255 dev fvnb0 table local proto kernel scope link src 100.82.60.13
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":248,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1900bdc8979c9bd6c3b1b6a8de1e4","table":"fvpn","handle":251,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.82.60.13","timeout":3566,"expires":3561}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"comment":"fvpn_class=01a1900b-b48f-7b00-94e6-0b5c2ccd4db5:e09a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.82.60.13"}},{"counter":"u_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.82.60.13"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.82.60.13"}},{"counter":"d_01a1900bdc8979c9bd6c3b1b6a8de1e4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:e09a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.82.60.13 dst=192.0.3.10 sport=49686 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=49686 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=50
rx_bytes=5981
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"c5a2ffcb-5f30-441f-bafb-ef114e7da6dd","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id d18a6860-c966-44f9-a861-a6c2ba515938
mono_ns 1619311115769
wall_unix_ns 1793418081794744947
policy_expires_mono_ns 5186178487285
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 4, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T03:41:16.678910621Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T03:41:01.419922+00:00", "expires_at": "2026-11-01T03:41:01.419922+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- provisioning_ready_at: 2026-10-31T03:40:10.552886Z
- populated_lease_at: 2026-10-31T03:40:51.184752Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 40.632
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T03:40:48.60626608Z","cpu_usage_usec":3585220,"memory_bytes":61718528,"open_fds":84,"processes":10,"conntrack_entries":7},{"at":"2026-10-31T03:40:53.605230922Z","cpu_usage_usec":4039648,"cpu_percent_one_core":9.090442008753778,"memory_bytes":60301312,"open_fds":86,"processes":10,"conntrack_entries":11},{"at":"2026-10-31T03:40:58.602876885Z","cpu_usage_usec":4423091,"cpu_percent_one_core":7.672472256714756,"memory_bytes":60362752,"open_fds":86,"processes":10,"conntrack_entries":17},{"at":"2026-10-31T03:41:03.603920357Z","cpu_usage_usec":4884226,"cpu_percent_one_core":9.220775675752813,"memory_bytes":62144512,"open_fds":86,"processes":10,"conntrack_entries":17},{"at":"2026-10-31T03:41:08.603451654Z","cpu_usage_usec":5230382,"cpu_percent_one_core":6.923769038263909,"memory_bytes":61530112,"open_fds":86,"processes":10,"conntrack_entries":22},{"at":"2026-10-31T03:41:13.602196963Z","cpu_usage_usec":5616318,"cpu_percent_one_core":7.720657407872748,"memory_bytes":61530112,"open_fds":88,"processes":10,"conntrack_entries":28},{"at":"2026-10-31T03:41:18.602282352Z","cpu_usage_usec":6084198,"cpu_percent_one_core":9.357440195507829,"memory_bytes":65372160,"open_fds":105,"processes":14,"conntrack_entries":32},{"at":"2026-10-31T03:41:23.603625233Z","cpu_usage_usec":6521106,"cpu_percent_one_core":8.735813768334193,"memory_bytes":66215936,"open_fds":104,"processes":14,"conntrack_entries":39},{"at":"2026-10-31T03:41:28.601727753Z","cpu_usage_usec":6980039,"cpu_percent_one_core":9.182144587142242,"memory_bytes":63397888,"open_fds":88,"processes":10,"conntrack_entries":43},{"at":"2026-10-31T03:41:33.598916586Z","cpu_usage_usec":7371325,"cpu_percent_one_core":7.830122356314807,"memory_bytes":63959040,"open_fds":85,"processes":10,"conntrack_entries":47},{"at":"2026-10-31T03:41:38.601650581Z","cpu_usage_usec":7744879,"cpu_percent_one_core":7.466997053478155,"memory_bytes":64237568,"open_fds":84,"processes":10,"conntrack_entries":51},{"at":"2026-10-31T03:41:43.600355841Z","cpu_usage_usec":8128967,"cpu_percent_one_core":7.6837496916151435,"memory_bytes":68808704,"open_fds":83,"processes":10,"conntrack_entries":55},{"at":"2026-10-31T03:41:48.607013117Z","cpu_usage_usec":8534807,"cpu_percent_one_core":8.106007214542958,"memory_bytes":63537152,"open_fds":82,"processes":10,"conntrack_entries":42},{"at":"2026-10-31T03:41:53.606518549Z","cpu_usage_usec":8900336,"cpu_percent_one_core":7.3113031873189485,"memory_bytes":65847296,"open_fds":82,"processes":10,"conntrack_entries":46},{"at":"2026-10-31T03:41:58.602400175Z","cpu_usage_usec":9283440,"cpu_percent_one_core":7.668396264759697,"memory_bytes":70254592,"open_fds":81,"processes":10,"conntrack_entries":50},{"at":"2026-10-31T03:42:03.604145858Z","cpu_usage_usec":9631473,"cpu_percent_one_core":6.958230627016866,"memory_bytes":66383872,"open_fds":81,"processes":10,"conntrack_entries":54},{"at":"2026-10-31T03:42:08.602712313Z","cpu_usage_usec":9990718,"cpu_percent_one_core":7.186960566276997,"memory_bytes":66596864,"open_fds":80,"processes":10,"conntrack_entries":59}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
