# member-traffic

Member device and exit traffic

Status: pass

Run: 91eb0372-e42b-4698-a137-7cce53b263fa

Source: f52fc9381405d4aafd064dc0ef055977de97f418

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a19fc2-5f1e-723c-a6c2-d66297ab5feb
- plan_revision_id: 01a19fc2-5f1e-7526-b3be-b86319eef363
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-11-03T04:55:38Z
- binding_valid_until: 2026-11-03T05:55:31Z
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

  - ID: fvpn-exit-73ecc0f49c1a6843b498c2cc
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: Networks selected successfully.
- netbird_networks_after_selection: Available Networks:

  - ID: fvpn-exit-73ecc0f49c1a6843b498c2cc
    Network: 0.0.0.0/0
    Status: Selected
- before_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.116.9.160 uid 0
    cache
- before_member_tunnel_address: 3: fvnb0    inet 100.116.9.160/16 brd 100.116.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- before_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.116.0.0/16 dev fvnb0 proto kernel scope link src 100.116.9.160
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.116.9.160 dev fvnb0 table local proto kernel scope host src 100.116.9.160
broadcast 100.116.255.255 dev fvnb0 table local proto kernel scope link src 100.116.9.160
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- before_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":786,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":787,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":788,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":789,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":790,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":791,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":792,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":793,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":913,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nd_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":914,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":915,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":916,"packets":0,"bytes":0}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.116.9.160","timeout":597,"expires":596}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":921,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":922,"comment":"fvpn_class=01a19fc2-259b-7d6c-8757-3e03573886a4:1ccd","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":923,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":924,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":925,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":926,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":927,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":928,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":929,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":930,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":931,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":932,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":933,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"counter":"u_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":934,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":936,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":937,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":938,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":939,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":940,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":941,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":942,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"counter":"d_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":945,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
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
tx_packets=25
tx_bytes=2954
- capture_member: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvnb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
04:55:43.176573 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.381856 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.381891 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.382104 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 418
04:55:43.382833 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.382919 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 930
04:55:43.382923 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.383489 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 64
04:55:43.426012 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.426023 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 219
04:55:43.426393 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.426792 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 141
04:55:43.427499 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.428311 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 24
04:55:43.428311 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.428326 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
- capture_node_ingress: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvmb0, link-type RAW (Raw IP), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
04:55:43.381229 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.381359 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.382202 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.382469 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 418
04:55:43.382479 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.382733 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 930
04:55:43.383188 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.383647 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 64
04:55:43.425643 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.426245 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 219
04:55:43.426257 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.426351 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 141
04:55:43.427846 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
04:55:43.428064 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 24
04:55:43.428080 IP 192.0.3.10.443 > 100.116.9.160.57144: tcp 0
04:55:43.428386 IP 100.116.9.160.57144 > 192.0.3.10.443: tcp 0
- capture_node_uplink: tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on fvup0, link-type EN10MB (Ethernet), snapshot length 96 bytes
16 packets captured
16 packets received by filter
0 packets dropped by kernel
04:55:43.381323 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 0
04:55:43.381351 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 0
04:55:43.382205 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 0
04:55:43.382472 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 418
04:55:43.382477 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 0
04:55:43.382726 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 930
04:55:43.383191 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 0
04:55:43.383650 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 64
04:55:43.425631 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 0
04:55:43.426249 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 219
04:55:43.426255 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 0
04:55:43.426342 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 141
04:55:43.427852 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 0
04:55:43.428053 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 24
04:55:43.428077 IP 192.0.3.10.443 > 192.0.3.20.57144: tcp 0
04:55:43.428390 IP 192.0.3.20.57144 > 192.0.3.10.443: tcp 0
- after_member_route: 192.0.3.10 dev fvnb0 table netbird src 100.116.9.160 uid 0
    cache
- after_member_tunnel_address: 3: fvnb0    inet 100.116.9.160/16 brd 100.116.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
- after_member_routes: default dev fvnb0 table netbird
default via 172.28.30.1 dev eth0
100.116.0.0/16 dev fvnb0 proto kernel scope link src 100.116.9.160
172.28.30.0/24 dev eth0 proto kernel scope link src 172.28.30.41
local 100.116.9.160 dev fvnb0 table local proto kernel scope host src 100.116.9.160
broadcast 100.116.255.255 dev fvnb0 table local proto kernel scope link src 100.116.9.160
local 127.0.0.0/8 dev lo table local proto kernel scope host src 127.0.0.1
local 127.0.0.1 dev lo table local proto kernel scope host src 127.0.0.1
broadcast 127.255.255.255 dev lo table local proto kernel scope link src 127.0.0.1
local 172.28.30.41 dev eth0 table local proto kernel scope host src 172.28.30.41
broadcast 172.28.30.255 dev eth0 table local proto kernel scope link src 172.28.30.41
- after_node_nft: [{"counter":{"family":"inet","name":"ctbase","table":"fvpn","handle":1,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctv6","table":"fvpn","handle":2,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctin","table":"fvpn","handle":3,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctbadns","table":"fvpn","handle":786,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdeny","table":"fvpn","handle":787,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctdenyd","table":"fvpn","handle":788,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"cthost","table":"fvpn","handle":789,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctinv","table":"fvpn","handle":790,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctnodn","table":"fvpn","handle":791,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunkd","table":"fvpn","handle":792,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"ctunku","table":"fvpn","handle":793,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"d_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":913,"packets":8,"bytes":1519}},{"counter":{"family":"inet","name":"nd_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":914,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"nu_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":915,"packets":0,"bytes":0}},{"counter":{"family":"inet","name":"u_01a19fc34a8f7d06b1f86a09f66e47c8","table":"fvpn","handle":916,"packets":8,"bytes":1113}},{"set":{"family":"inet","name":"allowed_v4","table":"fvpn","type":"ipv4_addr","handle":4,"flags":["timeout"],"elem":[{"elem":{"val":"100.116.9.160","timeout":592,"expires":587}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":953,"comment":"fvpn_generation=5 fvpn_metering=forward_admission_v1","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":954,"comment":"fvpn_class=01a19fc2-259b-7d6c-8757-3e03573886a4:1ccd","expr":[{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"192.0.2.255"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"192.0.2.255"}}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":955,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":956,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"nfproto"}},"right":"ipv6"}},{"counter":"ctv6"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":957,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":958,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"in","left":{"ct":{"key":"state"}},"right":"invalid"}},{"counter":"ctinv"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":959,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"counter":"ctunku"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":960,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":961,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"dport"}},"right":53}},{"counter":"nu_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":962,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@resolver_v4"}},{"counter":"ctbadns"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":963,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@denied_v4"}},{"counter":"ctdeny"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":964,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@host_v4"}},{"counter":"cthost"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":965,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"100.116.9.160"}},{"counter":"u_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":966,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"@allowed_v4"}},{"counter":"ctunkd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":968,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"!=","left":{"ct":{"key":"state"}},"right":{"set":["established","related"]}}},{"counter":"ctnodn"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":969,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"tcp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":970,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"match":{"op":"==","left":{"payload":{"protocol":"udp","field":"sport"}},"right":53}},{"counter":"nd_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":971,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@resolver_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":972,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@denied_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":973,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@host_v4"}},{"counter":"ctdenyd"},{"drop":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"forward","handle":974,"expr":[{"match":{"op":"==","left":{"meta":{"key":"iifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvmb0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"daddr"}},"right":"100.116.9.160"}},{"counter":"d_01a19fc34a8f7d06b1f86a09f66e47c8"},{"mangle":{"key":{"meta":{"key":"priority"}},"value":"1:1ccd"}},{"accept":null}]}},{"rule":{"family":"inet","table":"fvpn","chain":"postrouting","handle":977,"expr":[{"match":{"op":"==","left":{"meta":{"key":"oifname"}},"right":"fvup0"}},{"match":{"op":"==","left":{"payload":{"protocol":"ip","field":"saddr"}},"right":"@allowed_v4"}},{"snat":{"family":"ip","addr":"192.0.3.20"}}]}}]
- after_conntrack: conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 3 CLOSE src=100.116.9.160 dst=192.0.3.10 sport=57144 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=57144 [ASSURED] mark=0 use=1
- after_member_tunnel: rx_packets=10
rx_bytes=1980
tx_packets=9
tx_bytes=1588
- after_node_tunnel: rx_packets=9
rx_bytes=1588
tx_packets=11
tx_bytes=2128
- after_node_uplink: rx_packets=46
rx_bytes=5638
tx_packets=35
tx_bytes=4263
- origin_arrival: {"probe_id":"1b818954-ed23-4549-a033-27e8b92a7de8","requests":1,"source_address":"192.0.3.20"}
- snat_counter_limit: product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence
- controlled_origin_http_200_and_counted: true
- admission_bytes_before: 0
- admission_bytes_after: 2632
- first_stopped_hop: none after explicit NetBird exit selection
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 5188489597433
wall_unix_ns 1793681746038220957
policy_expires_mono_ns 5780873373286
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
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 27, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "qos", "detail": "read_at=2026-11-03T04:55:41.126647689Z source=live candidate=5 verified=true {\"generation\":\"5\",\"phase\":\"verify\",\"reads\":16,\"objects\":4,\"truncated\":false,\"sha256\":\"bfc249fb6a5605216d7847b6c04c3c9bd150786d952192fb3594cc692ba06a5c\"}", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-03T04:55:41.037386216Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-11-03T04:55:41.011896+00:00", "expires_at": "2026-11-04T04:55:41.011896+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- provisioning_ready_at: 2026-11-03T04:54:49.948941Z
- populated_lease_at: 2026-11-03T04:55:36.004224Z
- latest_lease_member_count: 1
- ready_to_populated_lease_seconds: 46.055
- node_telemetry: {"cadence_seconds":5,"errors":0,"samples":[{"at":"2026-11-03T04:55:31.88034296Z","cpu_usage_usec":1595353,"memory_bytes":61882368,"open_fds":104,"processes":12,"conntrack_entries":4},{"at":"2026-11-03T04:55:36.884420586Z","cpu_usage_usec":2184296,"cpu_percent_one_core":11.769261870359323,"memory_bytes":62464000,"open_fds":104,"processes":12,"conntrack_entries":17},{"at":"2026-11-03T04:55:41.880424817Z","cpu_usage_usec":2813552,"cpu_percent_one_core":12.595185490346317,"memory_bytes":62226432,"open_fds":105,"processes":12,"conntrack_entries":17},{"at":"2026-11-03T04:55:46.88768278Z","cpu_usage_usec":3484574,"cpu_percent_one_core":13.400987226109883,"memory_bytes":69877760,"open_fds":120,"processes":16,"conntrack_entries":26},{"at":"2026-11-03T04:55:51.88343458Z","cpu_usage_usec":4088522,"cpu_percent_one_core":12.089231494647112,"memory_bytes":63447040,"open_fds":92,"processes":12,"conntrack_entries":31},{"at":"2026-11-03T04:55:56.882914803Z","cpu_usage_usec":4621909,"cpu_percent_one_core":10.668849084474115,"memory_bytes":66732032,"open_fds":100,"processes":13,"conntrack_entries":36},{"at":"2026-11-03T04:56:01.883610162Z","cpu_usage_usec":5153861,"cpu_percent_one_core":10.637560615297621,"memory_bytes":67739648,"open_fds":101,"processes":13,"conntrack_entries":40},{"at":"2026-11-03T04:56:06.883656542Z","cpu_usage_usec":5687320,"cpu_percent_one_core":10.669081033604332,"memory_bytes":67268608,"open_fds":100,"processes":13,"conntrack_entries":44},{"at":"2026-11-03T04:56:11.882902702Z","cpu_usage_usec":6222345,"cpu_percent_one_core":10.702113536253632,"memory_bytes":69234688,"open_fds":90,"processes":12,"conntrack_entries":48},{"at":"2026-11-03T04:56:16.884375371Z","cpu_usage_usec":6760599,"cpu_percent_one_core":10.761910253677726,"memory_bytes":65888256,"open_fds":90,"processes":12,"conntrack_entries":49},{"at":"2026-11-03T04:56:21.882437049Z","cpu_usage_usec":7369749,"cpu_percent_one_core":12.187724747001411,"memory_bytes":67788800,"open_fds":94,"processes":12,"conntrack_entries":51}]}
- node_telemetry_scope: 5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.
