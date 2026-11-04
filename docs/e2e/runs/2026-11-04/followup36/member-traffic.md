# member-traffic

Member device and exit traffic

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a1a6fb-300d-7c68-9001-cd890a2cde77
- plan_revision_id: 01a1a6fb-300e-71a9-8fe7-82bb782e0a16
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-11-04T14:35:13Z
- binding_valid_until: 2026-11-04T15:34:54Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- admission_poll_started_at: 2026-11-04T14:35:18.708871732Z
- admission_poll_elapsed: 91.187352ms
- admission_poll_found: true
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 1.140
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-ced1962bb6ee02b08744a841
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.66.15.199 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.66.15.199/16 brd 100.66.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.66.0.0/16 dev fvnb0 proto kernel scope link src 100.66.15.199
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.66.15.199 dev fvnb0 table local proto kernel scope host src 100.66.15.199
broadcast 100.66.255.255 dev fvnb0 table local proto kernel scope link src 100.66.15.199
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":4,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":808,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":809,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":810,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":811,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":812,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":813,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":814,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":815,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":968,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":969,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":970,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":971,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":5,"flags":["timeout"],"elem":[{"elem":{"val":"100.66.15.199","timeout":594,"expires":592}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":979,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":980,"comment":"fvpn_class=01a1a6fa-fcf2-7104-98fe-f966095b4f94:5a0d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":981,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":982,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":983,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":984,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":985,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":986,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":987,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":988,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":989,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":990,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":991,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":992,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"counter":"u_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":993,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":995,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":996,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":997,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":998,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":999,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1000,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1001,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1002,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1003,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1004,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1005,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1008,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- before_member_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_tunnel: rx_packets=0
rx_bytes=0
tx_packets=0
tx_bytes=0
- before_node_uplink: rx_packets=30
rx_bytes=3703
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:35:22.150017 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.154519 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.154529 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.154710 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 418
14:35:22.155828 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.155828 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 929
14:35:22.155840 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.156288 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 64
14:35:22.198664 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.198674 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 219
14:35:22.200915 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.200915 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 141
14:35:22.201701 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.202413 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 24
14:35:22.202413 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.202424 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:35:22.154337 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.154410 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.154899 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.155314 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 418
14:35:22.155335 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.155582 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 929
14:35:22.156182 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.156403 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 64
14:35:22.197677 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.199827 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 219
14:35:22.199871 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.199966 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 141
14:35:22.202118 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0
14:35:22.202152 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 24
14:35:22.202169 IP 192.0.3.10.443 > 100.66.15.199.44034: tcp 0
14:35:22.202542 IP 100.66.15.199.44034 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
14:35:22.154392 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 0
14:35:22.154404 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 0
14:35:22.154907 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 0
14:35:22.155322 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 418
14:35:22.155331 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 0
14:35:22.155575 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 929
14:35:22.156190 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 0
14:35:22.156406 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 64
14:35:22.197647 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 0
14:35:22.199842 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 219
14:35:22.199862 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 0
14:35:22.199959 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 141
14:35:22.202122 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 0
14:35:22.202145 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 24
14:35:22.202166 IP 192.0.3.10.443 > 192.0.3.20.44034: tcp 0
14:35:22.202545 IP 192.0.3.20.44034 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.66.15.199 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.66.15.199/16 brd 100.66.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.66.0.0/16 dev fvnb0 proto kernel scope link src 100.66.15.199
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.66.15.199 dev fvnb0 table local proto kernel scope host src 100.66.15.199
broadcast 100.66.255.255 dev fvnb0 table local proto kernel scope link src 100.66.15.199
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":4,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":808,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":809,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":810,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":811,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":812,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":813,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":814,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":815,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":968,"packets":8,"bytes":1518}},{"counter":{"family":"inet","name":"nd_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":969,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":970,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1a6fc45297358801f7c91938f2b65","table":"fvpn","handle":971,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":5,"flags":["timeout"],"elem":[{"elem":{"val":"100.66.15.199","timeout":594,"expires":583}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":979,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":980,"comment":"fvpn_class=01a1a6fa-fcf2-7104-98fe-f966095b4f94:5a0d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":981,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":982,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":983,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":984,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":985,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":986,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":987,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":988,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":989,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":990,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":991,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":992,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.66.15.199"}},{"counter":"u_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":993,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":995,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":996,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":997,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":998,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":999,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1000,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"related"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"type"}},"right":"destination-unreachable"}},{"match":{"op":"==","left":{"payload":{"protocol":"icmp","field":"code"}},"right":"frag-needed"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1001,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1002,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1003,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1004,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"match":{"op":"==","left":{"\u0026":[{"payload":{"protocol":"tcp","field":"flags"}},"syn"]},"right":"syn"}},{"match":{"op":"\u003e","left":{"tcp option":{"name":"maxseg","field":"size"}},"right":1160}},{"mangle":{"key":{"tcp option":{"name":"maxseg","field":"size"}},"value":1160}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":1005,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.66.15.199"}},{"counter":"d_01a1a6fc45297358801f7c91938f2b65"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5a0d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":1008,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.66.15.199 dst=192.0.3.10 sport=44034 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=44034 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=42
rx_bytes=5611
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"6847d715-7073-4942-bcb8-2f05bccc1a43","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2631
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 126370593768613
wall_unix_ns 1793802928142392508
policy_expires_mono_ns 126955558671251
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 26, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "qos", "detail": "read_at=2026-11-04T14:35:28.095423435Z source=live candidate=5 verified=true {\"generation\":\"5\",\"phase\":\"verify\",\"reads\":16,\"objects\":4,\"truncated\":false,\"sha256\":\"616e059222cfe63f1aaac2180d3ab9fec7a990afffc878dac76c9c0566ec5cfa\"}", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:35:28.101172441Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-11-04T14:35:18.172015+00:00", "expires_at": "2026-11-05T14:35:18.172015+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- provisioning_ready_at: 2026-11-04T14:34:14.765916Z
- populated_lease_at: 2026-11-04T14:34:58.133378Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 43.367
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-11-04T14:34:54.415668845Z","cpu_usage_usec":1581337,"memory_bytes":62070784,"open_fds":104,"processes":12,"conntrack_entries":14},{"at":"2026-11-04T14:34:59.41805401Z","cpu_usage_usec":2165547,"cpu_percent_one_core":11.678628908615837,"memory_bytes":63254528,"open_fds":104,"processes":12,"conntrack_entries":16},{"at":"2026-11-04T14:35:04.417617127Z","cpu_usage_usec":2648955,"cpu_percent_one_core":9.66900484476872,"memory_bytes":62447616,"open_fds":104,"processes":12,"conntrack_entries":20},{"at":"2026-11-04T14:35:09.417564756Z","cpu_usage_usec":3132489,"cpu_percent_one_core":9.670781293697427,"memory_bytes":62754816,"open_fds":104,"processes":12,"conntrack_entries":25},{"at":"2026-11-04T14:35:14.415495477Z","cpu_usage_usec":3636292,"cpu_percent_one_core":10.080231762380205,"memory_bytes":62193664,"open_fds":92,"processes":12,"conntrack_entries":31},{"at":"2026-11-04T14:35:19.418257601Z","cpu_usage_usec":4268502,"cpu_percent_one_core":12.637218886883858,"memory_bytes":63832064,"open_fds":93,"processes":12,"conntrack_entries":31},{"at":"2026-11-04T14:35:24.416895629Z","cpu_usage_usec":4880427,"cpu_percent_one_core":12.241834607192725,"memory_bytes":68419584,"open_fds":108,"processes":16,"conntrack_entries":41},{"at":"2026-11-04T14:35:29.414739821Z","cpu_usage_usec":5462331,"cpu_percent_one_core":11.643100057649816,"memory_bytes":63991808,"open_fds":92,"processes":12,"conntrack_entries":45},{"at":"2026-11-04T14:35:34.418342726Z","cpu_usage_usec":6007923,"cpu_percent_one_core":10.903982797172029,"memory_bytes":65179648,"open_fds":92,"processes":12,"conntrack_entries":49},{"at":"2026-11-04T14:35:39.419508265Z","cpu_usage_usec":6507660,"cpu_percent_one_core":9.992410691127096,"memory_bytes":67371008,"open_fds":99,"processes":13,"conntrack_entries":53}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
- timed_section: member-traffic-forwarding-usage
