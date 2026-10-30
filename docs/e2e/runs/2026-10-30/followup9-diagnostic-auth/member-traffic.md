# member-traffic

Member device and exit traffic

Status: pass

Run: 376e5f84-535e-4ddd-902b-985852d4691b

Source: 724ab19fd46a974a66d138ac0cddcd7f803bafb7

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18b90-ce66-7e67-84df-0dd13456fc3b
- plan_revision_id: 01a18b90-ce67-70e1-aef9-79bc784a91ff
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T06:49:53Z
- binding_valid_until: 2026-10-30T07:49:34Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-f6e1194da0971a688b582332
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-f6e1194da0971a688b582332
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.65.79.12 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.65.79.12/16 brd 100.65.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.65.0.0/16 dev fvnb0 proto kernel scope link src 100.65.79.12
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.65.79.12 dev fvnb0 table local proto kernel scope host src 100.65.79.12
broadcast 100.65.255.255 dev fvnb0 table local proto kernel scope link src 100.65.79.12
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":374,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":375,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":376,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":377,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.65.79.12","timeout":3580,"expires":3578}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":382,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":383,"comment":"fvpn_class=01a18b92-2bbe-76bd-8dfe-2b8e70083ee4:a998","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":384,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":385,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":386,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":387,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":388,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":389,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":390,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":391,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":392,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":393,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":394,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.65.79.12"}},{"counter":"u_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":395,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":397,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":398,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":399,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":400,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.65.79.12"}},{"counter":"d_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":403,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=4047
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
06:49:57.438132 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.542979 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.542994 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.543190 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 418
06:49:57.543733 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.544012 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 930
06:49:57.544019 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.544627 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 64
06:49:57.585652 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.585664 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 219
06:49:57.586065 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.586310 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 141
06:49:57.586915 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.587137 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 24
06:49:57.587155 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.587373 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.587381 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
06:49:57.542823 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.542911 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.543129 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.543612 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 418
06:49:57.543624 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.543888 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 930
06:49:57.544107 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.544733 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 64
06:49:57.585320 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.585927 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 219
06:49:57.585953 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.586042 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 141
06:49:57.587081 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.587110 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 24
06:49:57.587122 IP 192.0.3.10.443 > 100.65.79.12.43420: tcp 0
06:49:57.587175 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0
06:49:57.587566 IP 100.65.79.12.43420 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
06:49:57.542889 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 0
06:49:57.542904 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 0
06:49:57.543132 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 0
06:49:57.543616 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 418
06:49:57.543621 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 0
06:49:57.543881 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 930
06:49:57.544118 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 0
06:49:57.544735 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 64
06:49:57.585305 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 0
06:49:57.585939 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 219
06:49:57.585950 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 0
06:49:57.586033 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 141
06:49:57.587091 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 0
06:49:57.587107 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 24
06:49:57.587120 IP 192.0.3.10.443 > 192.0.3.20.43420: tcp 0
06:49:57.587177 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 0
06:49:57.587576 IP 192.0.3.20.43420 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.65.79.12 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.65.79.12/16 brd 100.65.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.65.0.0/16 dev fvnb0 proto kernel scope link src 100.65.79.12
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.65.79.12 dev fvnb0 table local proto kernel scope host src 100.65.79.12
broadcast 100.65.255.255 dev fvnb0 table local proto kernel scope link src 100.65.79.12
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":374,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":375,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":376,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18b9275ad76bd9ba94336e6b9d754","table":"fvpn","handle":377,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.65.79.12","timeout":3575,"expires":3570}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":408,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"comment":"fvpn_class=01a18b92-2bbe-76bd-8dfe-2b8e70083ee4:a998","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.65.79.12"}},{"counter":"u_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":423,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":424,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.65.79.12"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":426,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.65.79.12"}},{"counter":"d_01a18b9275ad76bd9ba94336e6b9d754"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:a998"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":429,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.65.79.12 dst=192.0.3.10 sport=43420 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=43420 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=10
tx_bytes=1668
- after_node_tunnel: rx_packets=10
rx_bytes=1668
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=47
rx_bytes=5827
tx_packets=35
tx_bytes=4275
- origin_arrival: {"probe_id":"d60b454c-e1b9-4889-850e-37ae1652a93a","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2672
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id cd4a18ce-0aec-4d41-aa94-40873659b462
mono_ns 698039097189208
wall_unix_ns 1793342999900368163
policy_expires_mono_ns 701614117696172
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 5, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T06:49:59.85357598Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T06:49:54.764844+00:00", "expires_at": "2026-10-31T06:49:54.764844+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
