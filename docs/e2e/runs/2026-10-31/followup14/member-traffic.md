# member-traffic

Member device and exit traffic

Status: pass

Run: 4e3e9487-fe0f-4d88-a396-d5d5fa4a8e1d

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18f68-7060-7566-a4b8-9755e5b57ac6
- plan_revision_id: 01a18f68-7060-7703-8b19-c61b33936857
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T00:44:09Z
- binding_valid_until: 2026-10-31T01:43:58Z
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
- exit_discovery_seconds: 27.322
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-54d21662c2b6d4c65592871f
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.96.195.16 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.96.195.16/16 brd 100.96.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.96.0.0/16 dev fvnb0 proto kernel scope link src 100.96.195.16
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.96.195.16 dev fvnb0 table local proto kernel scope host src 100.96.195.16
broadcast 100.96.255.255 dev fvnb0 table local proto kernel scope link src 100.96.195.16
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":621,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":622,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":623,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":624,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":625,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":626,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":627,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":754,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":755,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":756,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":757,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.96.195.16","timeout":3561,"expires":3558}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":892,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":893,"comment":"fvpn_class=01a18f68-28c1-70e4-8c7f-700ebb321b0b:4971","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":894,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":895,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":896,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":897,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":898,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":899,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":900,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":901,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":902,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":903,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":904,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.195.16"}},{"counter":"u_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":905,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":907,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":908,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":909,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":910,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.195.16"}},{"counter":"d_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":913,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
00:44:40.618033 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.922980 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.922995 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.923201 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 418
00:44:40.923323 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.923690 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 929
00:44:40.923694 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.924527 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 64
00:44:40.967722 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.967741 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 219
00:44:40.967985 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.968847 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 141
00:44:40.969607 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.969883 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 24
00:44:40.969899 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.969901 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.969906 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
00:44:40.922864 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.922958 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.923245 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.923245 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 418
00:44:40.923274 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.923652 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 929
00:44:40.923708 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.924546 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 64
00:44:40.965324 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.967801 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 219
00:44:40.967830 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.968083 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 141
00:44:40.969799 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.969860 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 24
00:44:40.969876 IP 192.0.3.10.443 > 100.96.195.16.41514: tcp 0
00:44:40.970029 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
00:44:40.970039 IP 100.96.195.16.41514 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
00:44:40.922936 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 0
00:44:40.922951 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 0
00:44:40.923250 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 0
00:44:40.923252 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 418
00:44:40.923271 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 0
00:44:40.923639 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 929
00:44:40.923717 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 0
00:44:40.924548 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 64
00:44:40.965295 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 0
00:44:40.967816 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 219
00:44:40.967828 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 0
00:44:40.968061 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 141
00:44:40.969809 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 0
00:44:40.969852 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 24
00:44:40.969873 IP 192.0.3.10.443 > 192.0.3.20.41514: tcp 0
00:44:40.970032 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 0
00:44:40.970040 IP 192.0.3.20.41514 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.96.195.16 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.96.195.16/16 brd 100.96.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.96.0.0/16 dev fvnb0 proto kernel scope link src 100.96.195.16
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.96.195.16 dev fvnb0 table local proto kernel scope host src 100.96.195.16
broadcast 100.96.255.255 dev fvnb0 table local proto kernel scope link src 100.96.195.16
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":621,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":622,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":623,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":624,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":625,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":626,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":627,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":754,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":755,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":756,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18f69f88879dc8b059b95e7ccae28","table":"fvpn","handle":757,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.96.195.16","timeout":3551,"expires":3550}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":977,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":978,"comment":"fvpn_class=01a18f68-28c1-70e4-8c7f-700ebb321b0b:4971","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":979,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":980,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":981,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":982,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":983,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":984,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":985,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":986,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":987,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":988,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":989,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.195.16"}},{"counter":"u_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":990,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":992,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":993,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":994,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.195.16"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":995,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.195.16"}},{"counter":"d_01a18f69f88879dc8b059b95e7ccae28"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:4971"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":998,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 3 CLOSE src=100.96.195.16 dst=192.0.3.10 sport=41514 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=41514 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=10
tx_bytes=1668
- after_node_tunnel: rx_packets=10
rx_bytes=1668
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=50
rx_bytes=5980
tx_packets=36
tx_bytes=4317
- origin_arrival: {"probe_id":"2036c06d-92a1-4784-928b-751e940f571f","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2671
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 9221a278-3d9e-4513-9f0f-fb079ef24c1e
mono_ns 762526777513992
wall_unix_ns 1793407487580693007
policy_expires_mono_ns 766078049627069
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 26, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:44:57.839534932Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T00:44:47.499637+00:00", "expires_at": "2026-11-01T00:44:47.499637+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
- provisioning_ready_at: 2026-10-31T00:42:39.665608Z
- populated_lease_at: 2026-10-31T00:44:00.287374Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 80.622
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T00:43:58.701471664Z","cpu_usage_usec":3356453,"memory_bytes":53235712,"open_fds":85,"processes":10,"conntrack_entries":6},{"at":"2026-10-31T00:44:03.70108593Z","cpu_usage_usec":3797050,"cpu_percent_one_core":8.812619865422235,"memory_bytes":53473280,"open_fds":87,"processes":10,"conntrack_entries":10},{"at":"2026-10-31T00:44:08.701411174Z","cpu_usage_usec":4186874,"cpu_percent_one_core":7.795972881319238,"memory_bytes":54534144,"open_fds":87,"processes":10,"conntrack_entries":17},{"at":"2026-10-31T00:44:13.70262332Z","cpu_usage_usec":4648507,"cpu_percent_one_core":9.23042227611194,"memory_bytes":54915072,"open_fds":87,"processes":10,"conntrack_entries":18},{"at":"2026-10-31T00:44:18.700465733Z","cpu_usage_usec":5018967,"cpu_percent_one_core":7.4123985789625575,"memory_bytes":55263232,"open_fds":87,"processes":10,"conntrack_entries":22},{"at":"2026-10-31T00:44:23.69997696Z","cpu_usage_usec":10425514,"cpu_percent_one_core":108.14151133018346,"memory_bytes":58974208,"open_fds":86,"processes":10,"conntrack_entries":26},{"at":"2026-10-31T00:44:28.698625965Z","cpu_usage_usec":17042130,"cpu_percent_one_core":132.36808572439463,"memory_bytes":60100608,"open_fds":86,"processes":10,"conntrack_entries":30},{"at":"2026-10-31T00:44:33.701438393Z","cpu_usage_usec":23760930,"cpu_percent_one_core":134.30045792634303,"memory_bytes":59326464,"open_fds":101,"processes":12,"conntrack_entries":37},{"at":"2026-10-31T00:44:38.70154758Z","cpu_usage_usec":30410213,"cpu_percent_one_core":132.98275600236406,"memory_bytes":60440576,"open_fds":88,"processes":10,"conntrack_entries":41},{"at":"2026-10-31T00:44:43.700048095Z","cpu_usage_usec":35633204,"cpu_percent_one_core":104.49115658438619,"memory_bytes":61849600,"open_fds":102,"processes":14,"conntrack_entries":46},{"at":"2026-10-31T00:44:48.700263865Z","cpu_usage_usec":36198053,"cpu_percent_one_core":11.296492511162175,"memory_bytes":61661184,"open_fds":87,"processes":10,"conntrack_entries":42},{"at":"2026-10-31T00:44:53.701448246Z","cpu_usage_usec":36580724,"cpu_percent_one_core":7.651607516287651,"memory_bytes":61120512,"open_fds":86,"processes":10,"conntrack_entries":47}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
