# member-traffic

Member device and exit traffic

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a1a511-151f-7eb6-b439-8e1b30e9ec05
- plan_revision_id: 01a1a511-1520-722c-ad8e-048b7556ecae
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-11-04T05:39:52Z
- binding_valid_until: 2026-11-04T06:39:43Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- admission_poll_started_at: 2026-11-04T05:39:57.908755953Z
- admission_poll_elapsed: 159.068461ms
- admission_poll_found: true
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-ea3ad7201c918ad1bcbb9ab6
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-ea3ad7201c918ad1bcbb9ab6
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.100.254.103 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.100.254.103/16 brd 100.100.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.100.0.0/16 dev fvnb0 proto kernel scope link src 100.100.254.103
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.100.254.103 dev fvnb0 table local proto kernel scope host src 100.100.254.103
broadcast 100.100.255.255 dev fvnb0 table local proto kernel scope link src 100.100.254.103
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":4,"packets":9,"bytes":792}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":842,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":843,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":844,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":845,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":846,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":847,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":848,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":849,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1002,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1003,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1004,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1005,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":5,"flags":["timeout"],"elem":[{"elem":{"val":"100.100.254.103","timeout":595,"expires":593}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1013,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1014,"comment":"fvpn_class=01a1a510-de0a-7755-84af-ebb5224c2b70:14b6","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1015,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1016,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1017,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1018,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1019,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1020,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1021,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1022,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1023,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1024,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1025,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1026,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"counter":"u_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1027,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1029,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1030,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1031,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1032,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1033,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1034,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1035,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1036,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1037,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1038,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1039,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1042,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=29
rx_bytes=3593
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
05:40:01.069235 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.274431 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.274469 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.274804 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 418
05:40:01.275236 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.275587 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 930
05:40:01.275590 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.276310 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 64
05:40:01.317855 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.317860 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 219
05:40:01.318658 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.318907 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 141
05:40:01.319725 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.320993 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 24
05:40:01.320993 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.321010 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
05:40:01.274224 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.274323 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.274796 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.275025 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 418
05:40:01.275086 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.275470 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 930
05:40:01.275748 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.276422 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 64
05:40:01.317667 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.318380 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 219
05:40:01.318432 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.318580 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 141
05:40:01.320376 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0
05:40:01.320584 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 24
05:40:01.320613 IP 192.0.3.10.443 > 100.100.254.103.34078: tcp 0
05:40:01.321216 IP 100.100.254.103.34078 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
05:40:01.274299 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 0
05:40:01.274317 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 0
05:40:01.274804 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 0
05:40:01.275052 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 418
05:40:01.275078 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 0
05:40:01.275459 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 930
05:40:01.275763 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 0
05:40:01.276437 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 64
05:40:01.317643 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 0
05:40:01.318405 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 219
05:40:01.318425 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 0
05:40:01.318572 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 141
05:40:01.320401 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 0
05:40:01.320565 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 24
05:40:01.320607 IP 192.0.3.10.443 > 192.0.3.20.34078: tcp 0
05:40:01.321226 IP 192.0.3.20.34078 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.100.254.103 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.100.254.103/16 brd 100.100.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.100.0.0/16 dev fvnb0 proto kernel scope link src 100.100.254.103
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.100.254.103 dev fvnb0 table local proto kernel scope host src 100.100.254.103
broadcast 100.100.255.255 dev fvnb0 table local proto kernel scope link src 100.100.254.103
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":4,"packets":9,"bytes":792}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":842,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":843,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":844,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":845,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":846,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":847,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":848,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":849,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1002,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1003,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1004,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a51227d67ce9b02dc3cf0e48ca88","table":"fvpn","handle":1005,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":5,"flags":["timeout"],"elem":[{"elem":{"val":"100.100.254.103","timeout":595,"expires":584}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1013,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1014,"comment":"fvpn_class=01a1a510-de0a-7755-84af-ebb5224c2b70:14b6","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1015,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1016,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1017,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1018,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1019,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1020,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1021,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1022,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1023,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1024,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1025,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1026,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.100.254.103"}},{"counter":"u_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1027,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1029,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1030,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1031,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1032,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1033,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1034,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1035,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1036,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1037,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1038,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1039,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.100.254.103"}},{"counter":"d_01a1a51227d67ce9b02dc3cf0e48ca88"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:14b6"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1042,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.100.254.103 dst=192.0.3.10 sport=34078 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=34078 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=41
rx_bytes=5502
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"9da1a42c-65fb-415d-89bf-a5a8abaf1845","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 94249432621713
wall_unix_ns 1793770806981245569
policy_expires_mono_ns 94835409988462
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 27, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "qos", "detail": "read_at=2026-11-04T05:39:57.141100361Z source=live candidate=5 verified=true {\"generation\":\"5\",\"phase\":\"verify\",\"reads\":16,\"objects\":4,\"truncated\":false,\"sha256\":\"8b4d5f06edf407ff2c59dd33d7772dd4a45edb570094a687b162dc6a0128037c\"}", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:57.157237706Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-11-04T05:39:57.033458+00:00", "expires_at": "2026-11-05T05:39:57.033458+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- provisioning_ready_at: 2026-11-04T05:38:54.501879Z
- populated_lease_at: 2026-11-04T05:39:47.018384Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 52.517
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-11-04T05:39:43.616036764Z","cpu_usage_usec":1650031,"memory_bytes":62443520,"open_fds":104,"processes":12,"conntrack_entries":14},{"at":"2026-11-04T05:39:48.622369914Z","cpu_usage_usec":2226376,"cpu_percent_one_core":11.512318152458551,"memory_bytes":62332928,"open_fds":104,"processes":12,"conntrack_entries":15},{"at":"2026-11-04T05:39:53.618651775Z","cpu_usage_usec":2688696,"cpu_percent_one_core":9.253280996990574,"memory_bytes":62435328,"open_fds":104,"processes":12,"conntrack_entries":19},{"at":"2026-11-04T05:39:58.613553167Z","cpu_usage_usec":3339249,"cpu_percent_one_core":13.0243412020495,"memory_bytes":66383872,"open_fds":117,"processes":14,"conntrack_entries":19},{"at":"2026-11-04T05:40:03.622057235Z","cpu_usage_usec":3941074,"cpu_percent_one_core":12.016062916772698,"memory_bytes":68210688,"open_fds":108,"processes":16,"conntrack_entries":30},{"at":"2026-11-04T05:40:08.617996473Z","cpu_usage_usec":4543264,"cpu_percent_one_core":12.053589351520452,"memory_bytes":64229376,"open_fds":92,"processes":12,"conntrack_entries":34},{"at":"2026-11-04T05:40:13.61774561Z","cpu_usage_usec":5072410,"cpu_percent_one_core":10.583450999253605,"memory_bytes":64770048,"open_fds":92,"processes":12,"conntrack_entries":39},{"at":"2026-11-04T05:40:18.619870079Z","cpu_usage_usec":5594883,"cpu_percent_one_core":10.445021974921993,"memory_bytes":65110016,"open_fds":92,"processes":12,"conntrack_entries":43},{"at":"2026-11-04T05:40:23.618469095Z","cpu_usage_usec":6096350,"cpu_percent_one_core":10.03215097660076,"memory_bytes":66129920,"open_fds":92,"processes":12,"conntrack_entries":48},{"at":"2026-11-04T05:40:28.619675864Z","cpu_usage_usec":6624651,"cpu_percent_one_core":10.563470466261778,"memory_bytes":65617920,"open_fds":90,"processes":12,"conntrack_entries":52}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
