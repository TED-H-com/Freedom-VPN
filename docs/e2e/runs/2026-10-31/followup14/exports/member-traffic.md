# member-traffic

Member device and exit traffic

Status: pass

Run: 0a687f3a-2351-4234-b43d-882109267967

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18f86-f209-726b-a54f-5eda99ca2d30
- plan_revision_id: 01a18f86-f209-7421-ae92-51262b3b1e39
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T01:16:43Z
- binding_valid_until: 2026-10-31T02:16:36Z
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
- exit_discovery_seconds: 41.982
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-2b6993ff32683b94bbe2292d
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.108.52.81 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.108.52.81/16 brd 100.108.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.108.0.0/16 dev fvnb0 proto kernel scope link src 100.108.52.81
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.108.52.81 dev fvnb0 table local proto kernel scope host src 100.108.52.81
broadcast 100.108.255.255 dev fvnb0 table local proto kernel scope link src 100.108.52.81
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":232,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.108.52.81","timeout":3546,"expires":3544}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":445,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":446,"comment":"fvpn_class=01a18f87-af2a-7971-888e-69146ac15661:8e23","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":447,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":448,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":449,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":450,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":451,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":452,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":453,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":454,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":455,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":456,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":457,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.108.52.81"}},{"counter":"u_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":458,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":460,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":461,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":462,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":463,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.108.52.81"}},{"counter":"d_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":466,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
01:17:33.191923 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.195872 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.195885 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.196052 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 418
01:17:33.196096 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.196460 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 929
01:17:33.196464 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.197040 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 64
01:17:33.239491 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.239503 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 219
01:17:33.239885 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.240370 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 141
01:17:33.241012 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.241097 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 24
01:17:33.241097 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.241108 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
01:17:33.195780 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.195853 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.195898 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.196070 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 418
01:17:33.196078 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.196432 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 929
01:17:33.196711 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.197186 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 64
01:17:33.239216 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.239828 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 219
01:17:33.239868 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.240127 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 141
01:17:33.241028 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
01:17:33.241062 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 24
01:17:33.241079 IP 192.0.3.10.443 > 100.108.52.81.46120: tcp 0
01:17:33.241133 IP 100.108.52.81.46120 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
01:17:33.195835 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 0
01:17:33.195847 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 0
01:17:33.195901 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 0
01:17:33.196072 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 418
01:17:33.196076 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 0
01:17:33.196422 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 929
01:17:33.196714 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 0
01:17:33.197189 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 64
01:17:33.239194 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 0
01:17:33.239837 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 219
01:17:33.239861 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 0
01:17:33.240118 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 141
01:17:33.241032 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 0
01:17:33.241056 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 24
01:17:33.241076 IP 192.0.3.10.443 > 192.0.3.20.46120: tcp 0
01:17:33.241135 IP 192.0.3.20.46120 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.108.52.81 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.108.52.81/16 brd 100.108.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.108.0.0/16 dev fvnb0 proto kernel scope link src 100.108.52.81
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.108.52.81 dev fvnb0 table local proto kernel scope host src 100.108.52.81
broadcast 100.108.255.255 dev fvnb0 table local proto kernel scope link src 100.108.52.81
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":115,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":116,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":117,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":118,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":119,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":120,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":121,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":229,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18f87caaa75d1ad40637b1d2ac537","table":"fvpn","handle":232,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.108.52.81","timeout":3536,"expires":3535}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":497,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":498,"comment":"fvpn_class=01a18f87-af2a-7971-888e-69146ac15661:8e23","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":499,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":500,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":501,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":502,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":503,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":504,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":505,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":506,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":507,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":508,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":509,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.108.52.81"}},{"counter":"u_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":510,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":512,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":513,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":514,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.108.52.81"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":515,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.108.52.81"}},{"counter":"d_01a18f87caaa75d1ad40637b1d2ac537"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:8e23"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":518,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.108.52.81 dst=192.0.3.10 sport=46120 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=46120 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=47
rx_bytes=5744
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"8e27a3dc-17c9-4dbb-b803-e77073b5c8cd","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2631
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id e8646b71-d1de-4f92-bd58-5678c4986182
mono_ns 764499419129782
wall_unix_ns 1793409460222308925
policy_expires_mono_ns 768035811062722
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
- config_applied: 6
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 5, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T01:17:14.838932902Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-31T01:16:49.266538+00:00", "expires_at": "2026-11-01T01:16:49.266538+00:00", "generation": "6", "member_count": 1}, "applied_generation": "6", "desired_generation": "6"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 6
- desired_publication_intent: published
- provisioning_ready_at: 2026-10-31T01:15:55.955069Z
- populated_lease_at: 2026-10-31T01:16:37.639652Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 41.685
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-10-31T01:16:36.304470863Z","cpu_usage_usec":3717639,"memory_bytes":57212928,"open_fds":84,"processes":10,"conntrack_entries":7},{"at":"2026-10-31T01:16:41.305363552Z","cpu_usage_usec":4106376,"cpu_percent_one_core":7.773352162806228,"memory_bytes":56213504,"open_fds":84,"processes":10,"conntrack_entries":9},{"at":"2026-10-31T01:16:46.30225745Z","cpu_usage_usec":4511263,"cpu_percent_one_core":8.102773608262034,"memory_bytes":57438208,"open_fds":84,"processes":10,"conntrack_entries":9},{"at":"2026-10-31T01:16:51.301385052Z","cpu_usage_usec":4971432,"cpu_percent_one_core":9.204986082289643,"memory_bytes":58580992,"open_fds":86,"processes":10,"conntrack_entries":11},{"at":"2026-10-31T01:16:56.302484504Z","cpu_usage_usec":5349712,"cpu_percent_one_core":7.563936762919626,"memory_bytes":59289600,"open_fds":86,"processes":10,"conntrack_entries":17},{"at":"2026-10-31T01:17:01.30214878Z","cpu_usage_usec":5736396,"cpu_percent_one_core":7.7341993112659155,"memory_bytes":58404864,"open_fds":86,"processes":10,"conntrack_entries":22},{"at":"2026-10-31T01:17:06.302037986Z","cpu_usage_usec":6131304,"cpu_percent_one_core":7.898335017625989,"memory_bytes":59404288,"open_fds":88,"processes":10,"conntrack_entries":28},{"at":"2026-10-31T01:17:11.301442398Z","cpu_usage_usec":6512147,"cpu_percent_one_core":7.6177674101712585,"memory_bytes":58052608,"open_fds":88,"processes":10,"conntrack_entries":32},{"at":"2026-10-31T01:17:16.303120978Z","cpu_usage_usec":6891562,"cpu_percent_one_core":7.585753341231295,"memory_bytes":61538304,"open_fds":97,"processes":11,"conntrack_entries":37},{"at":"2026-10-31T01:17:21.301286116Z","cpu_usage_usec":7274245,"cpu_percent_one_core":7.656469713066132,"memory_bytes":60108800,"open_fds":85,"processes":10,"conntrack_entries":41},{"at":"2026-10-31T01:17:26.303507697Z","cpu_usage_usec":7649097,"cpu_percent_one_core":7.493710423060926,"memory_bytes":59961344,"open_fds":84,"processes":10,"conntrack_entries":45},{"at":"2026-10-31T01:17:31.303748485Z","cpu_usage_usec":8038782,"cpu_percent_one_core":7.793324692186804,"memory_bytes":61341696,"open_fds":83,"processes":10,"conntrack_entries":48},{"at":"2026-10-31T01:17:36.303401282Z","cpu_usage_usec":8562348,"cpu_percent_one_core":10.472047185239772,"memory_bytes":64053248,"open_fds":91,"processes":14,"conntrack_entries":46},{"at":"2026-10-31T01:17:41.301804328Z","cpu_usage_usec":9081284,"cpu_percent_one_core":10.382035926760276,"memory_bytes":62861312,"open_fds":74,"processes":10,"conntrack_entries":40},{"at":"2026-10-31T01:17:46.301023638Z","cpu_usage_usec":9488078,"cpu_percent_one_core":8.137150518407644,"memory_bytes":62709760,"open_fds":74,"processes":10,"conntrack_entries":44},{"at":"2026-10-31T01:17:51.301577033Z","cpu_usage_usec":9893091,"cpu_percent_one_core":8.099363570539376,"memory_bytes":63406080,"open_fds":74,"processes":10,"conntrack_entries":48},{"at":"2026-10-31T01:17:56.30131362Z","cpu_usage_usec":10304747,"cpu_percent_one_core":8.23355376501958,"memory_bytes":64237568,"open_fds":76,"processes":10,"conntrack_entries":54}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
