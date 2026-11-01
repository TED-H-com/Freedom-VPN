# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 9bb2098e-1c6f-48a8-9fbd-fd99bd9fc49e

Source: 1d396065c432485875add24a00761608afb293cc

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
- binding_valid_from: 2026-11-01T08:29:11Z
- binding_valid_until: 2026-11-01T09:29:02Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.125.32.183 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-f0b947dedea4ff1658ef78bf
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-f0b947dedea4ff1658ef78bf
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.125.32.183 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.125.32.183/16 brd 100.125.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.125.0.0/16 dev fvnb0 proto kernel scope link src 100.125.32.183
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.125.32.183 dev fvnb0 table local proto kernel scope host src 100.125.32.183
broadcast 100.125.255.255 dev fvnb0 table local proto kernel scope link src 100.125.32.183
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":235,"packets":9,"bytes":2513}},{"counter":{"family":"inet","name":"nd_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":238,"packets":11,"bytes":2357}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.125.32.183","timeout":591,"expires":591}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":272,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":273,"comment":"fvpn_class=01a19639-f2dc-76b3-9fb3-52426ab94969:cae1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":274,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":275,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":276,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":277,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":278,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":279,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":280,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":281,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":282,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":283,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":284,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.125.32.183"}},{"counter":"u_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":285,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":287,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":288,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":289,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":290,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.125.32.183"}},{"counter":"d_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":293,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: ipv4     2 tcp      6 8 CLOSE src=100.125.32.183 dst=192.0.3.10 sport=20001 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=20001 [ASSURED] mark=0 use=1
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
- before_member_tunnel: rx_packets=12
rx_bytes=3100
tx_packets=12
tx_bytes=2932
- before_node_tunnel: rx_packets=12
rx_bytes=2932
tx_packets=13
tx_bytes=3248
- before_node_uplink: rx_packets=44
rx_bytes=6580
tx_packets=26
tx_bytes=4093
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
08:29:21.899632 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.900108 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.900137 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.900361 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 418
08:29:21.900610 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.900949 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 928
08:29:21.900959 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.901536 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 64
08:29:21.942870 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.942885 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 219
08:29:21.943392 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.943392 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 141
08:29:21.944036 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.944399 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 24
08:29:21.944399 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.944430 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
08:29:21.899845 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.899905 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.900339 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.900475 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 418
08:29:21.900489 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.900749 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 928
08:29:21.901089 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.901667 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 64
08:29:21.942689 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.943109 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 219
08:29:21.943132 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.943194 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 141
08:29:21.944246 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0
08:29:21.944267 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 24
08:29:21.944278 IP 192.0.3.10.443 > 100.125.32.183.44006: tcp 0
08:29:21.944447 IP 100.125.32.183.44006 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
08:29:21.899874 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 0
08:29:21.899899 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 0
08:29:21.900347 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 0
08:29:21.900478 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 418
08:29:21.900486 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 0
08:29:21.900742 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 928
08:29:21.901096 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 0
08:29:21.901670 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 64
08:29:21.942658 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 0
08:29:21.943119 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 219
08:29:21.943128 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 0
08:29:21.943189 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 141
08:29:21.944249 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 0
08:29:21.944264 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 24
08:29:21.944276 IP 192.0.3.10.443 > 192.0.3.20.44006: tcp 0
08:29:21.944449 IP 192.0.3.20.44006 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.125.32.183 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.125.32.183/16 brd 100.125.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.125.0.0/16 dev fvnb0 proto kernel scope link src 100.125.32.183
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.125.32.183 dev fvnb0 table local proto kernel scope host src 100.125.32.183
broadcast 100.125.255.255 dev fvnb0 table local proto kernel scope link src 100.125.32.183
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":235,"packets":17,"bytes":4030}},{"counter":{"family":"inet","name":"nd_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1963a17937e23ab9072fe343745fd","table":"fvpn","handle":238,"packets":19,"bytes":3470}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.125.32.183","timeout":586,"expires":582}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":301,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":302,"comment":"fvpn_class=01a19639-f2dc-76b3-9fb3-52426ab94969:cae1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":303,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":304,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":305,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":306,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":307,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":308,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":309,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":310,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":311,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":312,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":313,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.125.32.183"}},{"counter":"u_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":314,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":317,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":318,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.125.32.183"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.125.32.183"}},{"counter":"d_01a1963a17937e23ab9072fe343745fd"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:cae1"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 tcp      6 0 CLOSE src=100.125.32.183 dst=192.0.3.10 sport=20001 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=20001 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 2 CLOSE src=100.125.32.183 dst=192.0.3.10 sport=44006 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=44006 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=20
rx_bytes=4956
tx_packets=20
tx_bytes=4372
- after_node_tunnel: rx_packets=20
rx_bytes=4372
tx_packets=21
tx_bytes=5104
- after_node_uplink: rx_packets=52
rx_bytes=8209
tx_packets=34
tx_bytes=5318
- origin_arrival: {"probe_id":"ab878c63-62e0-4666-b84a-6d7c1f64bac5","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 4870
- admission_bytes_after: 7500
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 105303229688291
wall_unix_ns 1793521765713317303
policy_expires_mono_ns 105889506252541
baseline_ok 1
serving 1
generation 5
reason serving
