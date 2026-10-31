# member-traffic

Member device and exit traffic

Status: blocked

Run: 0e34514e-bad1-4fe6-89fa-ff55da5fee12

Source: 59903b57c41115f02ccfb0b2bc1d56dc095b5a39

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a190a5-5908-7adb-828d-0bd00f71d6d0
- plan_revision_id: 01a190a5-5908-7cab-b20d-6f568f3a39ea
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T06:29:52Z
- binding_valid_until: 2026-10-31T07:29:39Z
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
- exit_discovery_seconds: 11.579
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-a83b56eea2114196f75daf8d
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.98.222.110 uid 0
    cache
- before_member_tunnel_address: 5: fvnb0    inet 100.98.222.110/16 brd 100.98.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.98.0.0/16 dev fvnb0 proto kernel scope link src 100.98.222.110
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.98.222.110 dev fvnb0 table local proto kernel scope host src 100.98.222.110
broadcast 100.98.255.255 dev fvnb0 table local proto kernel scope link src 100.98.222.110
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":137,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":138,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":139,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":140,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":141,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":142,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":143,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":289,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":290,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":291,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":292,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.98.222.110","timeout":3572,"expires":3569}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":349,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":350,"comment":"fvpn_class=01a190a6-49a7-70d8-937f-80b4210102a7:1e89","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":352,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":353,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":354,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":355,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":356,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":357,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":358,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.98.222.110"}},{"counter":"u_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.98.222.110"}},{"counter":"d_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=32
rx_bytes=3852
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
06:30:10.759283 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:11.765645 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:12.789646 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:13.813646 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:14.837646 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:15.863042 IP 192.0.3.10.443 > 100.98.222.110.36932: tcp 0
06:30:15.863056 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0

7 packets captured
7 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
06:30:15.862196 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:15.862196 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:15.862196 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:15.862196 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:15.862196 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0
06:30:15.862293 IP 192.0.3.10.443 > 100.98.222.110.36932: tcp 0
06:30:15.863359 IP 100.98.222.110.36932 > 192.0.3.10.443: tcp 0

7 packets captured
7 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
3 packets captured
3 packets received by filter
0 packets dropped by kernel
06:30:15.862269 IP 192.0.3.20.36932 > 192.0.3.10.443: tcp 0
06:30:15.862287 IP 192.0.3.10.443 > 192.0.3.20.36932: tcp 0
06:30:15.863367 IP 192.0.3.20.36932 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.98.222.110 uid 0
    cache
- after_member_tunnel_address: 5: fvnb0    inet 100.98.222.110/16 brd 100.98.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.98.0.0/16 dev fvnb0 proto kernel scope link src 100.98.222.110
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.98.222.110 dev fvnb0 table local proto kernel scope host src 100.98.222.110
broadcast 100.98.255.255 dev fvnb0 table local proto kernel scope link src 100.98.222.110
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":137,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":138,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":139,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":140,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":141,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":142,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":143,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":289,"packets":1,"bytes":60}},{"counter":{"family":"inet","name":"nd_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":290,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":291,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a190a67eb378ca83d984f4fe452239","table":"fvpn","handle":292,"packets":6,"bytes":340}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.98.222.110","timeout":3562,"expires":3561}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":401,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":402,"comment":"fvpn_class=01a190a6-49a7-70d8-937f-80b4210102a7:1e89","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":403,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":404,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":405,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":406,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":407,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":408,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.98.222.110"}},{"counter":"u_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.98.222.110"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.98.222.110"}},{"counter":"d_01a190a67eb378ca83d984f4fe452239"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1e89"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":422,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 7 CLOSE src=100.98.222.110 dst=192.0.3.10 sport=36932 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=36932 mark=0 use=1
- after_member_tunnel: rx_packets=3
rx_bytes=220
tx_packets=8
tx_bytes=856
- after_node_tunnel: rx_packets=8
rx_bytes=856
tx_packets=5
tx_bytes=460
- after_node_uplink: rx_packets=34
rx_bytes=3968
tx_packets=17
tx_bytes=1710
- origin_arrival: {"probe_id":"449d1d1c-64f7-4d97-b751-c5480f4bfa0f","requests":0,"source_address":""}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: false
- admission_bytes_before: 0
- admission_bytes_after: 0
- first_stopped_hop: see packet captures and rule counter deltas; do not infer a lease defect
- heartbeat: v1
boot_id 578320a0-dfdb-4b77-a0a4-7951b9a304c6
mono_ns 11754729853720
wall_unix_ns 1793428217213482978
policy_expires_mono_ns 15316890389591
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 7, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T06:30:07.340987573Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T06:29:57.213747+00:00", "expires_at": "2026-11-01T06:29:57.213747+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- traffic: not claimed; no admission counter increment. Inspect route selection, tunnel captures and named rule counters.
- provisioning_ready_at: 2026-10-31T06:28:57.188341Z
- populated_lease_at: 2026-10-31T06:29:42.446257Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 45.258
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T06:29:39.289072531Z","cpu_usage_usec":4112004,"memory_bytes":59830272,"open_fds":84,"processes":11,"conntrack_entries":4},{"at":"2026-10-31T06:29:44.293532624Z","cpu_usage_usec":4659030,"cpu_percent_one_core":10.930769550248867,"memory_bytes":60538880,"open_fds":84,"processes":11,"conntrack_entries":5},{"at":"2026-10-31T06:29:49.293957862Z","cpu_usage_usec":5135927,"cpu_percent_one_core":9.537128890077009,"memory_bytes":58724352,"open_fds":84,"processes":11,"conntrack_entries":10},{"at":"2026-10-31T06:29:54.29448236Z","cpu_usage_usec":5639562,"cpu_percent_one_core":10.071643488626702,"memory_bytes":60661760,"open_fds":84,"processes":11,"conntrack_entries":15},{"at":"2026-10-31T06:29:59.295662839Z","cpu_usage_usec":6261599,"cpu_percent_one_core":12.437803486835532,"memory_bytes":60334080,"open_fds":84,"processes":11,"conntrack_entries":15},{"at":"2026-10-31T06:30:04.292313901Z","cpu_usage_usec":6789235,"cpu_percent_one_core":10.559792818288258,"memory_bytes":65925120,"open_fds":91,"processes":12,"conntrack_entries":19},{"at":"2026-10-31T06:30:09.290224027Z","cpu_usage_usec":7297796,"cpu_percent_one_core":10.175473091330254,"memory_bytes":60710912,"open_fds":85,"processes":11,"conntrack_entries":23},{"at":"2026-10-31T06:30:14.293861625Z","cpu_usage_usec":7927309,"cpu_percent_one_core":12.581106998069208,"memory_bytes":64487424,"open_fds":100,"processes":15,"conntrack_entries":35}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
- owner: w2-agent-core / w1-netlab: compare named drop/admission counters and tunnel captures in member-traffic; lease generation alone is not the gate
