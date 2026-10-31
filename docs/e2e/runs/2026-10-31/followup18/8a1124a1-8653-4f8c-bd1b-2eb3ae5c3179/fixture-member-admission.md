# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179

Source: a0089b8386c749100d9bd19b8002dd4d666d065c

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
- binding_valid_from: 2026-10-31T10:48:31Z
- binding_valid_until: 2026-10-31T11:48:23Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-68d74b4a88706eb94552df10
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-68d74b4a88706eb94552df10
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.68.143.1 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.68.143.1/16 brd 100.68.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.68.0.0/16 dev fvnb0 proto kernel scope link src 100.68.143.1
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.68.143.1 dev fvnb0 table local proto kernel scope host src 100.68.143.1
broadcast 100.68.255.255 dev fvnb0 table local proto kernel scope link src 100.68.143.1
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":159,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":160,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":161,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":162,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":163,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":164,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":165,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":334,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":335,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":336,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":337,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.68.143.1","timeout":3588,"expires":3586}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"comment":"fvpn_class=01a19193-2b52-7583-bcc1-3a52bfcd3219:7e87","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":347,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":348,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":349,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":350,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":353,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":354,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.143.1"}},{"counter":"u_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":357,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":358,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.143.1"}},{"counter":"d_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=22
rx_bytes=3000
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:48:37.696829 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.901716 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.901729 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.902034 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 418
10:48:37.902182 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.902577 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 930
10:48:37.902581 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.903192 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 64
10:48:37.943745 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.943752 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 219
10:48:37.943829 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.943960 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 141
10:48:37.944695 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.944768 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 24
10:48:37.944781 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.944784 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.944788 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:48:37.901600 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.901688 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.901923 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.902148 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 418
10:48:37.902164 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.902436 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 930
10:48:37.902760 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.903401 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 64
10:48:37.943659 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.943792 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 219
10:48:37.943816 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.943942 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 141
10:48:37.944721 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.944748 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 24
10:48:37.944761 IP 192.0.3.10.443 > 100.68.143.1.52256: tcp 0
10:48:37.944816 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0
10:48:37.944817 IP 100.68.143.1.52256 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
10:48:37.901668 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 0
10:48:37.901682 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 0
10:48:37.901930 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 0
10:48:37.902155 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 418
10:48:37.902161 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 0
10:48:37.902427 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 930
10:48:37.902783 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 0
10:48:37.903411 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 64
10:48:37.943640 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 0
10:48:37.943804 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 219
10:48:37.943813 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 0
10:48:37.943934 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 141
10:48:37.944725 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 0
10:48:37.944744 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 24
10:48:37.944758 IP 192.0.3.10.443 > 192.0.3.24.52256: tcp 0
10:48:37.944820 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 0
10:48:37.944821 IP 192.0.3.24.52256 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.68.143.1 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.68.143.1/16 brd 100.68.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.68.0.0/16 dev fvnb0 proto kernel scope link src 100.68.143.1
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.68.143.1 dev fvnb0 table local proto kernel scope host src 100.68.143.1
broadcast 100.68.255.255 dev fvnb0 table local proto kernel scope link src 100.68.143.1
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":159,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":160,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":161,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":162,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":163,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":164,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":165,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":334,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":335,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":336,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a191934a2e7d3388c967effa3ab59e","table":"fvpn","handle":337,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.68.143.1","timeout":3578,"expires":3578}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":400,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":401,"comment":"fvpn_class=01a19193-2b52-7583-bcc1-3a52bfcd3219:7e87","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":402,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":403,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":404,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":405,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":406,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":407,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":408,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.68.143.1"}},{"counter":"u_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.143.1"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.68.143.1"}},{"counter":"d_01a191934a2e7d3388c967effa3ab59e"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:7e87"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.24"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.68.143.1 dst=192.0.3.10 sport=52256 dport=443 src=192.0.3.10 dst=192.0.3.24 sport=443 dport=52256 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=10
tx_bytes=1668
- after_node_tunnel: rx_packets=10
rx_bytes=1668
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=32
rx_bytes=4715
tx_packets=25
tx_bytes=2903
- origin_arrival: {"probe_id":"c26f69dc-8cdf-4fc4-acb7-e49b1e2ce4b7","requests":1,"source_address":"192.0.3.24"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2672
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 27261894669193
wall_unix_ns 1793443724378298152
policy_expires_mono_ns 30841150340522
baseline_ok 1
serving 1
generation 6
reason serving
