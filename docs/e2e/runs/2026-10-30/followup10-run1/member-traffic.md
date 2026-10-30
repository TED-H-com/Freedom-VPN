# member-traffic

Member device and exit traffic

Status: pass

Run: cfda038f-dc18-4371-8047-a4d34954066f

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18c36-a3cd-7014-b111-94c1df1fc885
- plan_revision_id: 01a18c36-a3cd-71ff-b87d-e896db7d7bf8
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T09:51:00Z
- binding_valid_until: 2026-10-30T10:50:40Z
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

  - ID: fvpn-exit-9c2942818bd6a7bf88bc99f1
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-9c2942818bd6a7bf88bc99f1
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.78.129.64 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.78.129.64/16 brd 100.78.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.78.0.0/16 dev fvnb0 proto kernel scope link src 100.78.129.64
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.78.129.64 dev fvnb0 table local proto kernel scope host src 100.78.129.64
broadcast 100.78.255.255 dev fvnb0 table local proto kernel scope link src 100.78.129.64
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":396,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":397,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":398,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":399,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.78.129.64","timeout":3578,"expires":3577}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":404,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":405,"comment":"fvpn_class=01a18c37-f8d6-716e-aff6-4602c2ee1619:b928","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":406,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":407,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":408,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":409,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.129.64"}},{"counter":"u_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":422,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.129.64"}},{"counter":"d_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
09:51:04.452233 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.557764 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.557786 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.558021 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 418
09:51:04.559070 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.559070 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 930
09:51:04.559083 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.559702 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 64
09:51:04.600750 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.600759 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 219
09:51:04.601307 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.601308 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 141
09:51:04.601892 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.602203 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 24
09:51:04.602204 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.602220 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
09:51:04.557546 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.557642 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.558078 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.558474 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 418
09:51:04.558494 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.558783 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 930
09:51:04.559697 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.560014 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 64
09:51:04.600197 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.601022 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 219
09:51:04.601074 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.601134 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 141
09:51:04.602069 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0
09:51:04.602098 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 24
09:51:04.602108 IP 192.0.3.10.443 > 100.78.129.64.60980: tcp 0
09:51:04.602642 IP 100.78.129.64.60980 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
09:51:04.557617 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 0
09:51:04.557635 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 0
09:51:04.558089 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 0
09:51:04.558481 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 418
09:51:04.558491 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 0
09:51:04.558776 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 930
09:51:04.559719 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 0
09:51:04.560023 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 64
09:51:04.600189 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 0
09:51:04.601046 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 219
09:51:04.601067 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 0
09:51:04.601128 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 141
09:51:04.602078 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 0
09:51:04.602096 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 24
09:51:04.602106 IP 192.0.3.10.443 > 192.0.3.20.60980: tcp 0
09:51:04.602645 IP 192.0.3.20.60980 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.78.129.64 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.78.129.64/16 brd 100.78.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.78.0.0/16 dev fvnb0 proto kernel scope link src 100.78.129.64
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.78.129.64 dev fvnb0 table local proto kernel scope host src 100.78.129.64
broadcast 100.78.255.255 dev fvnb0 table local proto kernel scope link src 100.78.129.64
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":225,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":226,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":227,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":228,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":229,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":230,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":231,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":396,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":397,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":398,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a18c3843ba7711977ffa1a0a03f4f4","table":"fvpn","handle":399,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.78.129.64","timeout":3573,"expires":3569}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":430,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":431,"comment":"fvpn_class=01a18c37-f8d6-716e-aff6-4602c2ee1619:b928","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":432,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":433,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":434,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":435,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":436,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":437,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":438,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":439,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":440,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":441,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":442,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.129.64"}},{"counter":"u_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":443,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":445,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":446,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":447,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.129.64"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":448,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.129.64"}},{"counter":"d_01a18c3843ba7711977ffa1a0a03f4f4"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:b928"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":451,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.78.129.64 dst=192.0.3.10 sport=60980 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=60980 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=50
rx_bytes=6009
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"ed78874b-e29f-4e10-99c1-b48937a97153","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 2bae3e01-6ed9-49fa-bd37-bb95506e6fa4
mono_ns 708907013079706
wall_unix_ns 1793353867816258640
policy_expires_mono_ns 712480057299451
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 5, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T09:51:07.769838396Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T09:51:02.675529+00:00", "expires_at": "2026-10-31T09:51:02.675529+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
