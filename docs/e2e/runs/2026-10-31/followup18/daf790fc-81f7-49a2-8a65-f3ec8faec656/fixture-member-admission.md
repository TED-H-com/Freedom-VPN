# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: daf790fc-81f7-49a2-8a65-f3ec8faec656

Source: fc570997ad08facaa963e32d2abb6fdabec8d9fa

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
- binding_valid_from: 2026-10-31T10:39:14Z
- binding_valid_until: 2026-10-31T11:39:02Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-a3b16fa43269564f00d0db0a
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-a3b16fa43269564f00d0db0a
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.67.102.246 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.67.102.246/16 brd 100.67.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.67.0.0/16 dev fvnb0 proto kernel scope link src 100.67.102.246
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.67.102.246 dev fvnb0 table local proto kernel scope host src 100.67.102.246
broadcast 100.67.255.255 dev fvnb0 table local proto kernel scope link src 100.67.102.246
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":235,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":238,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.67.102.246","timeout":3585,"expires":3583}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":243,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":244,"comment":"fvpn_class=01a1918a-99ad-73fb-9dcc-f5308ad66d65:638a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":245,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":246,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":247,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":248,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":249,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":250,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":251,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":252,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":253,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":254,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":255,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"counter":"u_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":256,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":258,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":259,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":260,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":261,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"counter":"d_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":264,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
rx_bytes=3559
tx_packets=14
tx_bytes=1540
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:39:19.441971 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.445875 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.445884 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.446041 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 418
10:39:19.446295 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.446750 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 930
10:39:19.446754 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.447473 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 64
10:39:19.489534 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.489538 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 219
10:39:19.490757 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.490935 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 141
10:39:19.491531 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.492150 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 24
10:39:19.492150 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.492158 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
10:39:19.445710 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.445765 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.445994 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.446157 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 418
10:39:19.446168 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.446425 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 930
10:39:19.446896 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.447794 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 64
10:39:19.488673 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.490380 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 219
10:39:19.490391 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.490463 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 141
10:39:19.491977 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0
10:39:19.492010 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 24
10:39:19.492022 IP 192.0.3.10.443 > 100.67.102.246.48818: tcp 0
10:39:19.492645 IP 100.67.102.246.48818 > 192.0.3.10.443: tcp 0

16 packets captured
16 packets received by filter
0 packets dropped by kernel
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
10:39:19.445749 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 0
10:39:19.445759 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 0
10:39:19.445997 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 0
10:39:19.446160 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 418
10:39:19.446166 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 0
10:39:19.446418 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 930
10:39:19.446900 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 0
10:39:19.447798 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 64
10:39:19.488651 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 0
10:39:19.490384 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 219
10:39:19.490389 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 0
10:39:19.490457 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 141
10:39:19.491983 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 0
10:39:19.492006 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 24
10:39:19.492019 IP 192.0.3.10.443 > 192.0.3.20.48818: tcp 0
10:39:19.492649 IP 192.0.3.20.48818 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.67.102.246 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.67.102.246/16 brd 100.67.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.67.0.0/16 dev fvnb0 proto kernel scope link src 100.67.102.246
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.67.102.246 dev fvnb0 table local proto kernel scope host src 100.67.102.246
broadcast 100.67.255.255 dev fvnb0 table local proto kernel scope link src 100.67.102.246
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":235,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":236,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":237,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1918ac8b471b799fbb8e6c34c1781","table":"fvpn","handle":238,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.67.102.246","timeout":3575,"expires":3575}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":301,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":302,"comment":"fvpn_class=01a1918a-99ad-73fb-9dcc-f5308ad66d65:638a","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":303,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":304,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":305,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":306,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":307,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":308,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":309,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":310,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":311,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":312,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":313,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.67.102.246"}},{"counter":"u_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":314,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":317,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":318,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.67.102.246"}},{"counter":"d_01a1918ac8b471b799fbb8e6c34c1781"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:638a"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.67.102.246 dst=192.0.3.10 sport=48818 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=48818 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=2036
tx_packets=10
tx_bytes=1680
- after_node_tunnel: rx_packets=10
rx_bytes=1680
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=40
rx_bytes=5381
tx_packets=24
tx_bytes=2849
- origin_arrival: {"probe_id":"efdae146-ec77-4ec6-9c54-1769128dd091","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 26703439683130
wall_unix_ns 1793443165923311969
policy_expires_mono_ns 30279578858310
baseline_ok 1
serving 1
generation 5
reason serving
