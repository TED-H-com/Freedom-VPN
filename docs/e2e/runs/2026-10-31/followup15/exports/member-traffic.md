# member-traffic

Member device and exit traffic

Status: pass

Run: 12d10041-9d33-4fbb-a4b0-9d2a21bce785

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a19033-8075-7ddf-abd1-839d5a4375e0
- plan_revision_id: 01a19033-8076-70f1-8a39-0e28e616ccaa
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T04:25:12Z
- binding_valid_until: 2026-10-31T05:25:02Z
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
- exit_discovery_seconds: 15.803
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-6661d07246816bba4d5f8281
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.104.230 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.83.104.230/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.104.230
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.104.230 dev fvnb0 table local proto kernel scope host src 100.83.104.230
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.104.230
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1903458597519b428f4418308e67f","table":"fvpn","handle":248,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1903458597519b428f4418308e67f","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1903458597519b428f4418308e67f","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1903458597519b428f4418308e67f","table":"fvpn","handle":251,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.104.230","timeout":3571,"expires":3569}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"comment":"fvpn_class=01a19034-32e4-7d4a-8fc5-3d8df444a439:d01","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"counter":"u_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":347,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":349,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":350,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"counter":"d_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=36
rx_bytes=3923
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
04:25:33.732073 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.736256 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.736267 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.736457 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 418
04:25:33.736954 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.736962 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 929
04:25:33.736964 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.737416 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 64
04:25:33.779281 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.779292 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 219
04:25:33.779709 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.779963 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 141
04:25:33.780697 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.781238 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 24
04:25:33.781248 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.781254 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.781257 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
04:25:33.736046 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.736103 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.736408 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.736623 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 418
04:25:33.736632 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.736818 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 929
04:25:33.737044 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.737506 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 64
04:25:33.778659 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.779559 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 219
04:25:33.779575 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.779823 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 141
04:25:33.781032 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.781071 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 24
04:25:33.781086 IP 192.0.3.10.443 > 100.83.104.230.35604: tcp 0
04:25:33.781644 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
04:25:33.781644 IP 100.83.104.230.35604 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
04:25:33.736090 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 0
04:25:33.736098 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 0
04:25:33.736412 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 0
04:25:33.736625 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 418
04:25:33.736630 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 0
04:25:33.736810 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 929
04:25:33.737045 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 0
04:25:33.737513 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 64
04:25:33.778641 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 0
04:25:33.779566 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 219
04:25:33.779573 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 0
04:25:33.779814 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 141
04:25:33.781035 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 0
04:25:33.781064 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 24
04:25:33.781083 IP 192.0.3.10.443 > 192.0.3.20.35604: tcp 0
04:25:33.781655 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 0
04:25:33.781657 IP 192.0.3.20.35604 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.83.104.230 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.83.104.230/16 brd 100.83.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.83.0.0/16 dev fvnb0 proto kernel scope link src 100.83.104.230
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.83.104.230 dev fvnb0 table local proto kernel scope host src 100.83.104.230
broadcast 100.83.255.255 dev fvnb0 table local proto kernel scope link src 100.83.104.230
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1903458597519b428f4418308e67f","table":"fvpn","handle":248,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a1903458597519b428f4418308e67f","table":"fvpn","handle":249,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1903458597519b428f4418308e67f","table":"fvpn","handle":250,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1903458597519b428f4418308e67f","table":"fvpn","handle":251,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.83.104.230","timeout":3566,"expires":3561}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"comment":"fvpn_class=01a19034-32e4-7d4a-8fc5-3d8df444a439:d01","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.83.104.230"}},{"counter":"u_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":373,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":378,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.83.104.230"}},{"counter":"d_01a1903458597519b428f4418308e67f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:d01"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":381,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.83.104.230 dst=192.0.3.10 sport=35604 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=35604 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=11
tx_bytes=1760
- after_node_tunnel: rx_packets=11
rx_bytes=1760
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=46
rx_bytes=5637
tx_packets=25
tx_bytes=2903
- origin_arrival: {"probe_id":"6a295410-699e-4805-ac83-fdaec3eb7497","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2671
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 2fde9c2a-3d5e-45be-bdbd-b976535ba0ba
mono_ns 4273839108408
wall_unix_ns 1793420736322737663
policy_expires_mono_ns 7839959785973
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 4, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T04:25:31.188608886Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T04:25:15.891687+00:00", "expires_at": "2026-11-01T04:25:15.891687+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- provisioning_ready_at: 2026-10-31T04:24:24.851452Z
- populated_lease_at: 2026-10-31T04:25:05.649334Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 40.798
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T04:25:02.388193417Z","cpu_usage_usec":3618569,"memory_bytes":57311232,"open_fds":84,"processes":11,"conntrack_entries":6},{"at":"2026-10-31T04:25:07.389860288Z","cpu_usage_usec":4052388,"cpu_percent_one_core":8.673488482715864,"memory_bytes":57606144,"open_fds":84,"processes":11,"conntrack_entries":6},{"at":"2026-10-31T04:25:12.409878246Z","cpu_usage_usec":4489516,"cpu_percent_one_core":8.707697933697311,"memory_bytes":62550016,"open_fds":97,"processes":13,"conntrack_entries":11},{"at":"2026-10-31T04:25:17.389871043Z","cpu_usage_usec":4996056,"cpu_percent_one_core":10.171500655686591,"memory_bytes":58888192,"open_fds":84,"processes":11,"conntrack_entries":11},{"at":"2026-10-31T04:25:22.390136828Z","cpu_usage_usec":5420697,"cpu_percent_one_core":8.492368571163862,"memory_bytes":60104704,"open_fds":84,"processes":11,"conntrack_entries":15},{"at":"2026-10-31T04:25:27.390330774Z","cpu_usage_usec":5817086,"cpu_percent_one_core":7.927472499683716,"memory_bytes":60018688,"open_fds":84,"processes":11,"conntrack_entries":19},{"at":"2026-10-31T04:25:32.386301433Z","cpu_usage_usec":6207574,"cpu_percent_one_core":7.816058713166274,"memory_bytes":60874752,"open_fds":85,"processes":11,"conntrack_entries":23},{"at":"2026-10-31T04:25:37.38985986Z","cpu_usage_usec":6739028,"cpu_percent_one_core":10.62152081870753,"memory_bytes":63995904,"open_fds":100,"processes":15,"conntrack_entries":33},{"at":"2026-10-31T04:25:42.3859789Z","cpu_usage_usec":7232508,"cpu_percent_one_core":9.877266655359756,"memory_bytes":61706240,"open_fds":84,"processes":11,"conntrack_entries":33},{"at":"2026-10-31T04:25:47.387911846Z","cpu_usage_usec":7601435,"cpu_percent_one_core":7.3756886384298195,"memory_bytes":65576960,"open_fds":83,"processes":11,"conntrack_entries":37},{"at":"2026-10-31T04:25:52.391076967Z","cpu_usage_usec":7993570,"cpu_percent_one_core":7.837738521842401,"memory_bytes":61235200,"open_fds":82,"processes":11,"conntrack_entries":40},{"at":"2026-10-31T04:25:57.390652965Z","cpu_usage_usec":8402317,"cpu_percent_one_core":8.175633296973837,"memory_bytes":61743104,"open_fds":82,"processes":11,"conntrack_entries":41},{"at":"2026-10-31T04:26:02.389586121Z","cpu_usage_usec":8795309,"cpu_percent_one_core":7.861517402534359,"memory_bytes":62365696,"open_fds":82,"processes":11,"conntrack_entries":45},{"at":"2026-10-31T04:26:07.387893126Z","cpu_usage_usec":9198188,"cpu_percent_one_core":8.06030921263909,"memory_bytes":63135744,"open_fds":82,"processes":11,"conntrack_entries":48},{"at":"2026-10-31T04:26:12.38728845Z","cpu_usage_usec":9570002,"cpu_percent_one_core":7.437179416780204,"memory_bytes":63029248,"open_fds":82,"processes":11,"conntrack_entries":52},{"at":"2026-10-31T04:26:17.388765754Z","cpu_usage_usec":9960595,"cpu_percent_one_core":7.809552583346083,"memory_bytes":65519616,"open_fds":82,"processes":11,"conntrack_entries":57},{"at":"2026-10-31T04:26:22.388860392Z","cpu_usage_usec":10346772,"cpu_percent_one_core":7.723393814691233,"memory_bytes":64253952,"open_fds":82,"processes":11,"conntrack_entries":33}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
