# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 4e1997a8-d3ea-4291-b2f6-c100992b4fda

Source: 587e5267b458c180adab656ec7e5346043f143ac

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
- binding_valid_from: 2026-10-31T14:29:51Z
- binding_valid_until: 2026-10-31T15:29:39Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 2.157
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-d363fe88af0c2f548e794102
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.78.2.126 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.78.2.126/16 brd 100.78.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.78.0.0/16 dev fvnb0 proto kernel scope link src 100.78.2.126
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.78.2.126 dev fvnb0 table local proto kernel scope host src 100.78.2.126
broadcast 100.78.255.255 dev fvnb0 table local proto kernel scope link src 100.78.2.126
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":235,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":238,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.78.2.126","timeout":599,"expires":596}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":243,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":244,"comment":"fvpn_class=01a1925d-bcc0-792a-8084-d87f775efd41:2a8d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":245,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":246,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":247,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":248,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":249,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":250,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":251,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":252,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":253,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":254,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":255,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"counter":"u_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":256,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":258,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":259,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":260,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":261,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"counter":"d_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":264,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=3596
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:29:56.725511 IP 100.78.2.126.44384 > 192.0.3.10.443: tcp 0
14:29:57.749670 IP 100.78.2.126.44384 > 192.0.3.10.443: tcp 0
14:29:58.772674 IP 100.78.2.126.44384 > 192.0.3.10.443: tcp 0
14:29:59.796658 IP 100.78.2.126.44384 > 192.0.3.10.443: tcp 0
14:30:00.820665 IP 100.78.2.126.44384 > 192.0.3.10.443: tcp 0

5 packets captured
5 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
0 packets captured
0 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
0 packets captured
0 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.78.2.126 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.78.2.126/16 brd 100.78.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.78.0.0/16 dev fvnb0 proto kernel scope link src 100.78.2.126
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.78.2.126 dev fvnb0 table local proto kernel scope host src 100.78.2.126
broadcast 100.78.255.255 dev fvnb0 table local proto kernel scope link src 100.78.2.126
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":235,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":238,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.78.2.126","timeout":589,"expires":587}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":301,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":302,"comment":"fvpn_class=01a1925d-bcc0-792a-8084-d87f775efd41:2a8d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":303,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":304,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":305,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":306,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":307,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":308,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":309,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":310,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":311,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":312,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":313,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"counter":"u_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":314,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":317,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":318,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"counter":"d_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.
- after_member_tunnel: rx_packets=1
rx_bytes=148
tx_packets=3
tx_bytes=388
- after_node_tunnel: rx_packets=2
rx_bytes=296
tx_packets=4
tx_bytes=480
- after_node_uplink: rx_packets=30
rx_bytes=3703
tx_packets=14
tx_bytes=1540
- origin_arrival: {"probe_id":"1346ae43-a413-4ed3-bd3c-2a7bc59d7cf5","requests":0,"source_address":""}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: false
- admission_bytes_before: 0
- admission_bytes_after: 0
- first_stopped_hop: see packet captures and rule counter deltas; do not infer a lease defect
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 40540449460228
wall_unix_ns 1793457002933089175
policy_expires_mono_ns 41129480000751
baseline_ok 1
serving 1
generation 5
reason serving
- counted_attempt: 2
- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.78.2.126 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-d363fe88af0c2f548e794102
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-d363fe88af0c2f548e794102
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.78.2.126 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.78.2.126/16 brd 100.78.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.78.0.0/16 dev fvnb0 proto kernel scope link src 100.78.2.126
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.78.2.126 dev fvnb0 table local proto kernel scope host src 100.78.2.126
broadcast 100.78.255.255 dev fvnb0 table local proto kernel scope link src 100.78.2.126
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":235,"packets":1,"bytes":60}},{"counter":{"family":"inet","name":"nd_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":238,"packets":6,"bytes":340}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.78.2.126","timeout":584,"expires":584}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"comment":"fvpn_class=01a1925d-bcc0-792a-8084-d87f775efd41:2a8d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":342,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"counter":"u_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":343,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":345,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":346,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":347,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":348,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"counter":"d_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":351,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 9 CLOSE src=100.78.2.126 dst=192.0.3.10 sport=44384 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=44384 mark=0 use=1
- before_member_tunnel: rx_packets=4
rx_bytes=424
tx_packets=10
tx_bytes=1040
- before_node_tunnel: rx_packets=9
rx_bytes=948
tx_packets=7
tx_bytes=756
- before_node_uplink: rx_packets=32
rx_bytes=3819
tx_packets=17
tx_bytes=1710
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:30:08.847466 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.848333 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.848356 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.848484 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 418
14:30:08.849117 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.849641 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 930
14:30:08.849648 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.850184 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 64
14:30:08.891137 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.891151 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 219
14:30:08.891562 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.891980 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 141
14:30:08.892574 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.893139 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 24
14:30:08.893140 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.893157 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
14:30:08.847830 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.847889 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.848603 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.848603 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 418
14:30:08.848637 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.849272 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 930
14:30:08.850103 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.850293 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 64
14:30:08.890701 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.891287 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 219
14:30:08.891336 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.891441 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 141
14:30:08.892860 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0
14:30:08.892897 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 24
14:30:08.892913 IP 192.0.3.10.443 > 100.78.2.126.36034: tcp 0
14:30:08.893413 IP 100.78.2.126.36034 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
14:30:08.847858 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 0
14:30:08.847884 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 0
14:30:08.848608 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 0
14:30:08.848610 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 418
14:30:08.848634 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 0
14:30:08.849265 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 930
14:30:08.850106 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 0
14:30:08.850295 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 64
14:30:08.890667 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 0
14:30:08.891308 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 219
14:30:08.891329 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 0
14:30:08.891435 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 141
14:30:08.892865 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 0
14:30:08.892891 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 24
14:30:08.892910 IP 192.0.3.10.443 > 192.0.3.20.36034: tcp 0
14:30:08.893419 IP 192.0.3.20.36034 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.78.2.126 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.78.2.126/16 brd 100.78.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.78.0.0/16 dev fvnb0 proto kernel scope link src 100.78.2.126
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.78.2.126 dev fvnb0 table local proto kernel scope host src 100.78.2.126
broadcast 100.78.255.255 dev fvnb0 table local proto kernel scope link src 100.78.2.126
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":235,"packets":9,"bytes":1579}},{"counter":{"family":"inet","name":"nd_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1925deeb674e98349dc16ef41db1f","table":"fvpn","handle":238,"packets":14,"bytes":1453}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.78.2.126","timeout":579,"expires":575}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":359,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":360,"comment":"fvpn_class=01a1925d-bcc0-792a-8084-d87f775efd41:2a8d","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":361,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":362,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":363,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":364,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":365,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":366,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":367,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":368,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":369,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":370,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":371,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.78.2.126"}},{"counter":"u_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":372,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":374,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":375,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":376,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":377,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.78.2.126"}},{"counter":"d_01a1925deeb674e98349dc16ef41db1f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:2a8d"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":380,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 tcp      6 1 CLOSE src=100.78.2.126 dst=192.0.3.10 sport=44384 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=44384 mark=0 use=1
ipv4     2 tcp      6 2 CLOSE src=100.78.2.126 dst=192.0.3.10 sport=36034 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=36034 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=12
rx_bytes=2280
tx_packets=18
tx_bytes=2480
- after_node_tunnel: rx_packets=17
rx_bytes=2388
tx_packets=15
tx_bytes=2612
- after_node_uplink: rx_packets=40
rx_bytes=5450
tx_packets=25
tx_bytes=2935
- origin_arrival: {"probe_id":"a13fdf56-7962-449c-9b6f-2712cbff6126","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 400
- admission_bytes_after: 3032
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 40550147411837
wall_unix_ns 1793457012631040764
policy_expires_mono_ns 41129480000751
baseline_ok 1
serving 1
generation 5
reason serving
