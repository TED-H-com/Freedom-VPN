# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: cce1a688-6af5-483c-bd67-2d167c51c04b

Source: 98a4b4607cef592a4b00f79aa623c15682cc6ef4

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
- binding_valid_from: 2026-10-31T16:25:49Z
- binding_valid_until: 2026-10-31T17:25:30Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_discovery_seconds: 1.095
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-e6ca6283a8e57ffdc0161107
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.120.193.23 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.120.193.23/16 brd 100.120.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.120.0.0/16 dev fvnb0 proto kernel scope link src 100.120.193.23
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.120.193.23 dev fvnb0 table local proto kernel scope host src 100.120.193.23
broadcast 100.120.255.255 dev fvnb0 table local proto kernel scope link src 100.120.193.23
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":312,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":313,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":314,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":315,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.120.193.23","timeout":590,"expires":588}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":320,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":321,"comment":"fvpn_class=01a192c7-cf9a-70ee-a377-58d35ef5f703:78f9","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":330,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.120.193.23"}},{"counter":"u_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":335,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":336,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":338,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.120.193.23"}},{"counter":"d_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":341,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=4130
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16:26:01.923743 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.928795 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.928807 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.929007 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 418
16:26:01.929722 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.929732 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 931
16:26:01.929735 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.930362 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 64
16:26:01.974641 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.974648 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 219
16:26:01.975456 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.975550 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 141
16:26:01.976223 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.976610 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 24
16:26:01.976610 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.976623 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16:26:01.927805 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.927897 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.928918 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.929086 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 418
16:26:01.929097 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.929384 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 931
16:26:01.929789 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.931638 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 64
16:26:01.972673 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.975281 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 219
16:26:01.975313 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.975444 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 141
16:26:01.976474 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0
16:26:01.976511 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 24
16:26:01.976524 IP 192.0.3.10.443 > 100.120.193.23.51830: tcp 0
16:26:01.976749 IP 100.120.193.23.51830 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16:26:01.927878 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 0
16:26:01.927891 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 0
16:26:01.928925 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 0
16:26:01.929089 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 418
16:26:01.929094 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 0
16:26:01.929377 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 931
16:26:01.929792 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 0
16:26:01.931649 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 64
16:26:01.972653 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 0
16:26:01.975293 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 219
16:26:01.975306 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 0
16:26:01.975435 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 141
16:26:01.976479 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 0
16:26:01.976506 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 24
16:26:01.976521 IP 192.0.3.10.443 > 192.0.3.20.51830: tcp 0
16:26:01.976753 IP 192.0.3.20.51830 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.120.193.23 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.120.193.23/16 brd 100.120.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.120.0.0/16 dev fvnb0 proto kernel scope link src 100.120.193.23
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.120.193.23 dev fvnb0 table local proto kernel scope host src 100.120.193.23
broadcast 100.120.255.255 dev fvnb0 table local proto kernel scope link src 100.120.193.23
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":312,"packets":8,"bytes":1520}},{"counter":{"family":"inet","name":"nd_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":313,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":314,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a192c8181075c99df74174e1c56784","table":"fvpn","handle":315,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.120.193.23","timeout":599,"expires":599}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"comment":"fvpn_class=01a192c7-cf9a-70ee-a377-58d35ef5f703:78f9","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":422,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":423,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":424,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":426,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.120.193.23"}},{"counter":"u_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":427,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":429,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":430,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":431,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.120.193.23"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":432,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.120.193.23"}},{"counter":"d_01a192c8181075c99df74174e1c56784"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:78f9"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":435,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.120.193.23 dst=192.0.3.10 sport=51830 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=51830 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=46
rx_bytes=5846
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"419f16a7-5a50-4c79-951f-e67770fa74cc","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2633
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 47506689809009
wall_unix_ns 1793463969173437885
policy_expires_mono_ns 48106380218954
baseline_ok 1
serving 1
generation 6
reason serving
