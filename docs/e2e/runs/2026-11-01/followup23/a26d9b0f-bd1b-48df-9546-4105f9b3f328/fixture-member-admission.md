# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: pass

Run: a26d9b0f-bd1b-48df-9546-4105f9b3f328

Source: c26303b57317c893ca169a87a662b0d0bd238daa

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
- binding_valid_from: 2026-11-01T08:18:08Z
- binding_valid_until: 2026-11-01T09:17:48Z
- binding: projected
- counted_attempt: 1
- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.113.52.80 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-72a06ba7362f16519ff3561e
    Network: 0.0.0.0/0
    Status: Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-72a06ba7362f16519ff3561e
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.113.52.80 uid 0
    cache
- before_member_tunnel_address: 4: fvnb0    inet 100.113.52.80/16 brd 100.113.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.113.0.0/16 dev fvnb0 proto kernel scope link src 100.113.52.80
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.113.52.80 dev fvnb0 table local proto kernel scope host src 100.113.52.80
broadcast 100.113.255.255 dev fvnb0 table local proto kernel scope link src 100.113.52.80
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":279,"packets":2,"bytes":120}},{"counter":{"family":"inet","name":"nd_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":280,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":281,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":282,"packets":7,"bytes":380}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.113.52.80","timeout":591,"expires":586}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":316,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":317,"comment":"fvpn_class=01a1962f-a826-7e71-bbfd-8f12178ec74a:5127","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":318,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":319,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":320,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":321,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":322,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":323,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":324,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":325,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":326,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":327,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":328,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.113.52.80"}},{"counter":"u_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":329,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":331,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":332,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":333,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":334,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.113.52.80"}},{"counter":"d_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":337,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- before_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 8 CLOSE src=100.113.52.80 dst=192.0.3.10 sport=20001 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=20001 mark=0 use=1
- before_member_tunnel: rx_packets=4
rx_bytes=316
tx_packets=9
tx_bytes=936
- before_node_tunnel: rx_packets=9
rx_bytes=936
tx_packets=6
tx_bytes=556
- before_node_uplink: rx_packets=36
rx_bytes=4194
tx_packets=19
tx_bytes=1838
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
08:18:22.609213 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.609955 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.609979 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.610113 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 418
08:18:22.610881 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.610881 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 930
08:18:22.610889 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.611503 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 64
08:18:22.652717 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.652736 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 219
08:18:22.653010 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.653227 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 141
08:18:22.653789 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.653870 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 24
08:18:22.653886 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.653989 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.653994 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
17 packets captured
17 packets received by filter
0 packets dropped by kernel
08:18:22.609543 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.609615 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.610281 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.610282 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 418
08:18:22.610313 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.610792 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 930
08:18:22.611237 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.611718 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 64
08:18:22.652673 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.652970 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 219
08:18:22.652994 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.653102 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 141
08:18:22.653808 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.653838 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 24
08:18:22.653852 IP 192.0.3.10.443 > 100.113.52.80.50370: tcp 0
08:18:22.653992 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
08:18:22.654008 IP 100.113.52.80.50370 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
08:18:22.609578 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 0
08:18:22.609609 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 0
08:18:22.610286 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 0
08:18:22.610287 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 418
08:18:22.610310 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 0
08:18:22.610784 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 930
08:18:22.611240 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 0
08:18:22.611721 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 64
08:18:22.652648 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 0
08:18:22.652980 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 219
08:18:22.652992 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 0
08:18:22.653095 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 141
08:18:22.653812 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 0
08:18:22.653833 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 24
08:18:22.653850 IP 192.0.3.10.443 > 192.0.3.20.50370: tcp 0
08:18:22.653995 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 0
08:18:22.654010 IP 192.0.3.20.50370 > 192.0.3.10.443: tcp 0

17 packets captured
17 packets received by filter
0 packets dropped by kernel
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.113.52.80 uid 0
    cache
- after_member_tunnel_address: 4: fvnb0    inet 100.113.52.80/16 brd 100.113.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.113.0.0/16 dev fvnb0 proto kernel scope link src 100.113.52.80
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.113.52.80 dev fvnb0 table local proto kernel scope host src 100.113.52.80
broadcast 100.113.255.255 dev fvnb0 table local proto kernel scope link src 100.113.52.80
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":93,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":94,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":95,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":96,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":97,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":98,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":99,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":279,"packets":10,"bytes":1639}},{"counter":{"family":"inet","name":"nd_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":280,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":281,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a1962ff6c279e8908b4df2281559bb","table":"fvpn","handle":282,"packets":16,"bytes":1533}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.113.52.80","timeout":599,"expires":596}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":410,"comment":"fvpn_generation=6 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":411,"comment":"fvpn_class=01a1962f-a826-7e71-bbfd-8f12178ec74a:5127","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":412,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":413,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":414,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":415,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":416,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":417,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":418,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":419,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":420,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":421,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":422,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.113.52.80"}},{"counter":"u_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":423,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":425,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":426,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":427,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.113.52.80"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":428,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.113.52.80"}},{"counter":"d_01a1962ff6c279e8908b4df2281559bb"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:5127"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":431,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 2 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.113.52.80 dst=192.0.3.10 sport=50370 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=50370 [ASSURED] mark=0 use=1
ipv4     2 tcp      6 0 CLOSE src=100.113.52.80 dst=192.0.3.10 sport=20001 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=20001 mark=0 use=1
- after_member_tunnel: rx_packets=12
rx_bytes=2172
tx_packets=18
tx_bytes=2456
- after_node_tunnel: rx_packets=18
rx_bytes=2456
tx_packets=14
tx_bytes=2412
- after_node_uplink: rx_packets=44
rx_bytes=5825
tx_packets=28
tx_bytes=3117
- origin_arrival: {"probe_id":"8bfd084d-0176-4396-8729-324f2370ada4","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 500
- admission_bytes_after: 3172
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 104644433345606
wall_unix_ns 1793521106916974664
policy_expires_mono_ns 105244154676054
baseline_ok 1
serving 1
generation 6
reason serving
