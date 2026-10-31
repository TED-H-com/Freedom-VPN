# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: 99027e5d-ae5e-40d4-b595-7b550f89a52d

Source: b27bb361ae1fee1c42579235b794cb09efe4ed3a

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
- binding_valid_from: 2026-10-31T19:27:52Z
- binding_valid_until: 2026-10-31T20:27:38Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-a7ff7a0808cbf7a3456de725
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-a7ff7a0808cbf7a3456de725
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.96.69.67 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.96.69.67/16 brd 100.96.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.96.0.0/16 dev fvnb0 proto kernel scope link src 100.96.69.67
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.96.69.67 dev fvnb0 table local proto kernel scope host src 100.96.69.67
broadcast 100.96.255.255 dev fvnb0 table local proto kernel scope link src 100.96.69.67
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":257,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":258,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":259,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":260,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.96.69.67","timeout":595,"expires":593}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":265,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":266,"comment":"fvpn_class=01a1936e-8fb9-7a85-b5c1-ebe0f26867d2:5239","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":267,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":268,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":269,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":270,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":271,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":272,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":273,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":274,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":275,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":276,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":277,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.69.67"}},{"counter":"u_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":278,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":280,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":281,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":282,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":283,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.69.67"}},{"counter":"d_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":286,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=3719
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
19:28:00.481482 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.485819 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.485833 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.486032 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 418
19:28:00.486223 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.486420 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 931
19:28:00.486424 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.487069 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 64
19:28:00.530165 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.530175 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 219
19:28:00.532123 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.532123 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 141
19:28:00.532990 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.534157 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 24
19:28:00.534175 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.534179 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.534183 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
19:28:00.485580 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.485668 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.485898 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.486137 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 418
19:28:00.486147 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.486355 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 931
19:28:00.486501 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.487153 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 64
19:28:00.528815 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.531525 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 219
19:28:00.531588 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.531701 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 141
19:28:00.533692 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.533772 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 24
19:28:00.533788 IP 192.0.3.10.443 > 100.96.69.67.34306: tcp 0
19:28:00.534340 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0
19:28:00.534351 IP 100.96.69.67.34306 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
19:28:00.485648 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 0
19:28:00.485661 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 0
19:28:00.485907 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 0
19:28:00.486140 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 418
19:28:00.486143 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 0
19:28:00.486347 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 931
19:28:00.486503 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 0
19:28:00.487157 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 64
19:28:00.528781 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 0
19:28:00.531556 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 219
19:28:00.531580 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 0
19:28:00.531693 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 141
19:28:00.533704 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 0
19:28:00.533755 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 24
19:28:00.533786 IP 192.0.3.10.443 > 192.0.3.20.34306: tcp 0
19:28:00.534344 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 0
19:28:00.534353 IP 192.0.3.20.34306 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.96.69.67 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.96.69.67/16 brd 100.96.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.96.0.0/16 dev fvnb0 proto kernel scope link src 100.96.69.67
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.96.69.67 dev fvnb0 table local proto kernel scope host src 100.96.69.67
broadcast 100.96.255.255 dev fvnb0 table local proto kernel scope link src 100.96.69.67
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":257,"packets":8,"bytes":1520}},{"counter":{"family":"inet","name":"nd_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":258,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":259,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1936ec43e72aeb995bad8876d444f","table":"fvpn","handle":260,"packets":9,"bytes":1153}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.96.69.67","timeout":585,"expires":584}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"comment":"fvpn_class=01a1936e-8fb9-7a85-b5c1-ebe0f26867d2:5239","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.96.69.67"}},{"counter":"u_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":339,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":340,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.69.67"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.96.69.67"}},{"counter":"d_01a1936ec43e72aeb995bad8876d444f"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5239"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":344,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.96.69.67 dst=192.0.3.10 sport=34306 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=34306 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=11
tx_bytes=1760
- after_node_tunnel: rx_packets=11
rx_bytes=1760
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=42
rx_bytes=5435
tx_packets=25
tx_bytes=2903
- origin_arrival: {"probe_id":"549555d7-4e1d-45a7-b2a2-39509787980a","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2673
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 58424651646926
wall_unix_ns 1793474887135276048
policy_expires_mono_ns 59009939355002
baseline_ok 1
serving 1
generation 5
reason serving
