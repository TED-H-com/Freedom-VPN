# member-traffic

Member device and exit traffic

Status: pass

Run: 2347f2a4-02aa-4f56-94f2-c3bde88b5a6d

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a19024-c299-7e39-b9e1-b832ebbfe5be
- plan_revision_id: 01a19024-c29a-71bc-9bad-919732c2ef8f
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T04:09:51Z
- binding_valid_until: 2026-10-31T05:09:34Z
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
- exit_discovery_seconds: 10.545
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-145394a9ef870ffc1023981d
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.99.222.11 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.99.222.11/16 brd 100.99.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.99.0.0/16 dev fvnb0 proto kernel scope link src 100.99.222.11
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.99.222.11 dev fvnb0 table local proto kernel scope host src 100.99.222.11
broadcast 100.99.255.255 dev fvnb0 table local proto kernel scope link src 100.99.222.11
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":762,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":765,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.99.222.11","timeout":3570,"expires":3568}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":822,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":823,"comment":"fvpn_class=01a19024-7d4e-7e1b-933e-cdd77d31be6f:4c3e","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":824,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":825,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":826,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":827,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":828,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":829,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":830,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":831,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":832,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":833,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":834,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.99.222.11"}},{"counter":"u_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":835,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":837,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":838,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":839,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":840,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.99.222.11"}},{"counter":"d_01a190264c257974b3b77ea2d01b8276"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4c3e"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":843,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=4196
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:10:07.001048 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.004586 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.004598 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.004789 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 418
04:10:07.005069 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.005431 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 927
04:10:07.005434 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.005895 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 64
04:10:07.048208 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.048215 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 219
04:10:07.049187 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.049194 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 141
04:10:07.049758 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.050295 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 24
04:10:07.050309 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.050311 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.050315 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
04:10:07.003746 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.003819 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.004906 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.004907 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 418
04:10:07.004943 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.005291 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 927
04:10:07.005544 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.006247 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 64
04:10:07.046670 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.048549 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 219
04:10:07.048579 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.048671 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 141
04:10:07.050126 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.050153 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 24
04:10:07.050164 IP 192.0.3.10.443 > 100.99.222.11.51904: tcp 0
04:10:07.050464 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
04:10:07.050503 IP 100.99.222.11.51904 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
04:10:07.003798 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 0
04:10:07.003813 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 0
04:10:07.004917 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 0
04:10:07.004919 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 418
04:10:07.004938 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 0
04:10:07.005283 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 927
04:10:07.005549 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 0
04:10:07.006258 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 64
04:10:07.046648 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 0
04:10:07.048564 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 219
04:10:07.048576 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 0
04:10:07.048664 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 141
04:10:07.050132 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 0
04:10:07.050149 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 24
04:10:07.050162 IP 192.0.3.10.443 > 192.0.3.20.51904: tcp 0
04:10:07.050473 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 0
04:10:07.050506 IP 192.0.3.20.51904 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.99.222.11 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.99.222.11/16 brd 100.99.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.99.0.0/16 dev fvnb0 proto kernel scope link src 100.99.222.11
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.99.222.11 dev fvnb0 table local proto kernel scope host src 100.99.222.11
broadcast 100.99.255.255 dev fvnb0 table local proto kernel scope link src 100.99.222.11
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":610,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":611,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":612,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":613,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":614,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":615,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":616,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":762,"packets":8,"bytes":1516}},{"counter":{"family":"inet","name":"nd_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":763,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":764,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190264c257974b3b77ea2d01b8276","table":"fvpn","handle":765,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":896,"comment":"fvpn_generation=0 fvpn_baseline=1 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":897,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"counter":"ctbase"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":898,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"counter":"ctbase"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":899,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":900,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.99.222.11 dst=192.0.3.10 sport=51904 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=51904 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=11
tx_bytes=1760
- after_node_tunnel: rx_packets=11
rx_bytes=1760
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=50
rx_bytes=5978
tx_packets=36
tx_bytes=4317
- origin_arrival: {"probe_id":"c75ffdbe-375c-4757-85ed-a50a242199bb","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2669
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id b63266b8-5ba5-4729-9b93-722e86fd4c21
mono_ns 3351717181529
wall_unix_ns 1793419814200810614
policy_expires_mono_ns 6912399917523
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 26, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:10:34.420876543Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T04:10:14.147178+00:00", "expires_at": "2026-11-01T04:10:14.147178+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
- provisioning_ready_at: 2026-10-31T04:08:22.126352Z
- populated_lease_at: 2026-10-31T04:09:38.404928Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 76.279
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T04:09:34.838667593Z","cpu_usage_usec":3171938,"memory_bytes":57430016,"open_fds":85,"processes":11,"conntrack_entries":5},{"at":"2026-10-31T04:09:39.838981445Z","cpu_usage_usec":3604298,"cpu_percent_one_core":8.646657245866015,"memory_bytes":57597952,"open_fds":85,"processes":11,"conntrack_entries":6},{"at":"2026-10-31T04:09:44.841908814Z","cpu_usage_usec":3964145,"cpu_percent_one_core":7.192728845710332,"memory_bytes":57581568,"open_fds":85,"processes":11,"conntrack_entries":13},{"at":"2026-10-31T04:09:49.843940027Z","cpu_usage_usec":4358616,"cpu_percent_one_core":7.8862162829930345,"memory_bytes":58015744,"open_fds":85,"processes":11,"conntrack_entries":17},{"at":"2026-10-31T04:09:54.842950945Z","cpu_usage_usec":4868554,"cpu_percent_one_core":10.20077788115765,"memory_bytes":58208256,"open_fds":85,"processes":11,"conntrack_entries":17},{"at":"2026-10-31T04:09:59.842660776Z","cpu_usage_usec":7878049,"cpu_percent_one_core":60.19339325134527,"memory_bytes":61255680,"open_fds":84,"processes":11,"conntrack_entries":21},{"at":"2026-10-31T04:10:04.839738381Z","cpu_usage_usec":14489396,"cpu_percent_one_core":132.30426906687993,"memory_bytes":62971904,"open_fds":85,"processes":11,"conntrack_entries":26},{"at":"2026-10-31T04:10:09.845524336Z","cpu_usage_usec":20167361,"cpu_percent_one_core":113.4280420905452,"memory_bytes":65499136,"open_fds":102,"processes":15,"conntrack_entries":37},{"at":"2026-10-31T04:10:14.83944394Z","cpu_usage_usec":20772422,"cpu_percent_one_core":12.115953959598425,"memory_bytes":63213568,"open_fds":86,"processes":11,"conntrack_entries":31},{"at":"2026-10-31T04:10:19.83862313Z","cpu_usage_usec":21160204,"cpu_percent_one_core":7.756913390415998,"memory_bytes":67325952,"open_fds":86,"processes":11,"conntrack_entries":35},{"at":"2026-10-31T04:10:24.840717467Z","cpu_usage_usec":21538822,"cpu_percent_one_core":7.569189513268471,"memory_bytes":62935040,"open_fds":86,"processes":11,"conntrack_entries":35},{"at":"2026-10-31T04:10:29.841166749Z","cpu_usage_usec":21908732,"cpu_percent_one_core":7.397535284110497,"memory_bytes":65130496,"open_fds":86,"processes":11,"conntrack_entries":39},{"at":"2026-10-31T04:10:34.839182621Z","cpu_usage_usec":22290014,"cpu_percent_one_core":7.6286672504588635,"memory_bytes":64634880,"open_fds":86,"processes":11,"conntrack_entries":43}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
