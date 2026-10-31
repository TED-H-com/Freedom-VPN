# native-mtu-comparison

F8-02 standalone versus reconnect matrix with PMTU and MSS capture

Status: fail

Run: daf790fc-81f7-49a2-8a65-f3ec8faec656

Source: fc570997ad08facaa963e32d2abb6fdabec8d9fa

Spec: AT09, AT11, F8-02

- standalone1500_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- standalone1500_transfer: {"after":{"upload_bytes":"68297","download_bytes":"69450","Present":true},"before":{"upload_bytes":"1113","download_bytes":"1519","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":2},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- standalone1500_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
277 packets captured
281 packets received by filter
0 packets dropped by kernel

- standalone1500_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443168.154880 fvnb0 Out IP (tos 0x0, ttl 64, id 14097, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.48222 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0x26d4), seq 2652344711, win 64480, options [mss 1240,sackOK,TS val 3717835106 ecr 0,nop,wscale 10], length 0
1793443168.155037 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.48222: Flags [S.], cksum 0x5290 (correct), seq 3753722902, ack 2652344712, win 65160, options [mss 1460,sackOK,TS val 1480873110 ecr 3717835106,nop,wscale 10], length 0

- standalone1500_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:35760 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.025/0.026 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5114 segs_out:15 segs_in:15 data_segs_out:9 data_segs_in:11 send 4633600000bps lastsnd:20561 lastrcv:20560 lastack:20560 pacing_rate 9221094520bps delivery_rate 1930666664bps delivered:10 app_limited busy:1ms rcv_space:14480 rcv_ssthresh:81052 minrtt:0.005 snd_wnd:71680
ESTAB 0      0       172.28.30.41:48406 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.152/0.159 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3647 bytes_acked:3648 bytes_received:7687 segs_out:48 segs_in:38 data_segs_out:22 data_segs_in:33 send 762105263bps lastsnd:591 lastrcv:591 lastack:591 pacing_rate 1524210520bps delivery_rate 4633600000bps delivered:23 app_limited busy:3ms rcv_space:14480 rcv_ssthresh:85793 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.41:48412 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.417/2.672 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3816 bytes_acked:3817 bytes_received:6187 segs_out:33 segs_in:34 data_segs_out:17 data_segs_in:29 send 81750176bps lastsnd:9180 lastrcv:9179 lastack:9179 pacing_rate 163485928bps delivery_rate 2316800000bps delivered:18 app_limited busy:43ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.41:48420 172.28.30.21:443
	 cubic wscale:10,10 rto:205 rtt:4.75/9.187 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4364 bytes_acked:4365 bytes_received:5447 segs_out:26 segs_in:23 data_segs_out:16 data_segs_in:17 send 24387368bps lastsnd:9079 lastrcv:9133 lastack:9079 pacing_rate 48770880bps delivery_rate 2896000000bps delivered:17 app_limited busy:43ms rcv_space:14480 rcv_ssthresh:85788 minrtt:0.008 snd_wnd:79872

- standalone1500_capture_0_receipt: {"bytes":"37372","sha256":"f64cdaa40b98ebd151ce45de41b38c0ba054559b36b1733c93a164ed06db1398"}
- standalone1500_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
419 packets captured
419 packets received by filter
0 packets dropped by kernel

- standalone1500_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443168.154952 fvmb0 In  IP (tos 0x0, ttl 64, id 14097, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.48222 > 192.0.3.10.9446: Flags [S], cksum 0x26d4 (correct), seq 2652344711, win 64480, options [mss 1240,sackOK,TS val 3717835106 ecr 0,nop,wscale 10], length 0
1793443168.154971 fvup0 Out IP (tos 0x0, ttl 63, id 14097, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.48222 > 192.0.3.10.9446: Flags [S], cksum 0x2ef9 (correct), seq 2652344711, win 64480, options [mss 1240,sackOK,TS val 3717835106 ecr 0,nop,wscale 10], length 0
1793443168.154993 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.48222: Flags [S.], cksum 0x864d (incorrect -> 0x5ab5), seq 3753722902, ack 2652344712, win 65160, options [mss 1460,sackOK,TS val 1480873110 ecr 3717835106,nop,wscale 10], length 0
1793443168.154999 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.48222: Flags [S.], cksum 0x8e72 (incorrect -> 0x5290), seq 3753722902, ack 2652344712, win 65160, options [mss 1460,sackOK,TS val 1480873110 ecr 3717835106,nop,wscale 10], length 0

- standalone1500_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.767/0.054 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:15891 bytes_acked:15892 bytes_received:29889 segs_out:106 segs_in:82 data_segs_out:78 data_segs_in:78 send 3417783572bps lastsnd:3116 lastrcv:3116 lastack:3116 pacing_rate 6834453312bps delivery_rate 32768000000bps delivered:79 app_limited busy:106ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.223/0.273 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4412 bytes_acked:4413 bytes_received:9823 segs_out:73 segs_in:56 data_segs_out:32 data_segs_in:51 send 519461883bps lastsnd:3122 lastrcv:3122 lastack:3122 pacing_rate 1037760352bps delivery_rate 2896000000bps delivered:33 app_limited busy:6ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:77824
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.005/1.884 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5032 bytes_acked:5033 bytes_received:6468 segs_out:40 segs_in:47 data_segs_out:23 data_segs_in:41 send 115263682bps lastsnd:9475 lastrcv:9474 lastack:9474 pacing_rate 230326824bps delivery_rate 4633600000bps delivered:24 app_limited busy:40ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.993/3.408 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4750 bytes_acked:4751 bytes_received:5117 segs_out:28 segs_in:26 data_segs_out:17 data_segs_in:18 send 58123432bps lastsnd:9428 lastrcv:9374 lastack:9374 pacing_rate 116210416bps delivery_rate 2896000000bps delivered:18 app_limited busy:44ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:210 rtt:9.026/5.757 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:23640 bytes_acked:23641 bytes_received:18690 segs_out:46 segs_in:27 data_segs_out:22 data_segs_in:24 send 444724130bps lastsnd:3153 lastrcv:3133 lastack:3133 pacing_rate 889435936bps delivery_rate 44600888888bps delivered:23 app_limited busy:144ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:36092    127.0.0.1:9444
	 cubic wscale:10,10 rto:204 rtt:3.856/3.958 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5422 bytes_acked:5423 bytes_received:8230 segs_out:17 segs_in:10 data_segs_out:8 data_segs_in:7 send 817925311bps lastsnd:13030 lastrcv:13025 lastack:13025 pacing_rate 1635532504bps delivery_rate 39424000000bps delivered:9 app_limited busy:40ms rcv_rtt:5 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.008 snd_wnd:78848
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36092
	 cubic wscale:10,10 rto:203 rtt:2.603/5.188 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:8230 bytes_acked:8230 bytes_received:5422 segs_out:9 segs_in:17 data_segs_out:7 data_segs_in:8 send 1337533615bps lastsnd:13029 lastrcv:13034 lastack:13029 pacing_rate 2674681896bps delivery_rate 58026666664bps delivered:8 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78471 minrtt:0.006 snd_wnd:84992
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:202 rtt:1.126/2.201 ato:40 mss:45568 pmtu:65535 rcvmss:3179 advmss:65483 cwnd:10 bytes_sent:18690 bytes_acked:18690 bytes_received:23640 segs_out:26 segs_in:46 data_segs_out:24 data_segs_in:22 send 3237513321bps lastsnd:3138 lastrcv:3158 lastack:3138 pacing_rate 6475026640bps delivery_rate 121514666664bps delivered:25 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:99450 minrtt:0.003 snd_wnd:91136
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:211 rtt:10.508/16.828 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:29889 bytes_acked:29889 bytes_received:15891 segs_out:81 segs_in:106 data_segs_out:78 data_segs_in:78 send 249470879bps lastsnd:3125 lastrcv:3125 lastack:3084 pacing_rate 498923952bps delivery_rate 29127111104bps delivered:79 app_limited busy:543ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:206 rtt:5.844/7.45 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:28070 bytes_acked:28071 bytes_received:25826 segs_out:78 segs_in:48 data_segs_out:51 data_segs_in:33 send 19822040bps lastsnd:3161 lastrcv:3141 lastack:3141 pacing_rate 39644072bps delivery_rate 1448000000bps delivered:52 app_limited busy:105ms rcv_rtt:5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.016 snd_wnd:79872

- standalone1500_capture_1_receipt: {"bytes":"55670","sha256":"b0623f01865cd4940c50f3eacfa0769564a33c34c2c2287e7f808044498a59e3"}
- standalone1280_input: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- standalone1280_transfer: {"after":{"upload_bytes":"136677","download_bytes":"137589","Present":true},"before":{"upload_bytes":"68297","download_bytes":"69450","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- standalone1280_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
331 packets captured
331 packets received by filter
0 packets dropped by kernel

- standalone1280_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443169.844672 eth1  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::88a6:94ff:fec9:250d > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::a048:9dff:fea8:4507
	  source link-address option (1), length 8 (1): 8a:a6:94:c9:25:0d
1793443169.844698 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::a048:9dff:fea8:4507 > fe80::88a6:94ff:fec9:250d: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::a048:9dff:fea8:4507, Flags [solicited]
1793443170.385162 fvnb0 Out IP (tos 0x0, ttl 64, id 8213, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.48238 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0x16d6), seq 3244539949, win 64480, options [mss 1240,sackOK,TS val 3959773426 ecr 0,nop,wscale 10], length 0
1793443170.385324 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.48238: Flags [S.], cksum 0x7d88 (correct), seq 2684061078, ack 3244539950, win 65160, options [mss 1460,sackOK,TS val 3590036010 ecr 3959773426,nop,wscale 10], length 0

- standalone1280_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:35760 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.025/0.026 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5114 segs_out:15 segs_in:15 data_segs_out:9 data_segs_in:11 send 4633600000bps lastsnd:22783 lastrcv:22782 lastack:22782 pacing_rate 9221094520bps delivery_rate 1930666664bps delivered:10 app_limited busy:1ms rcv_space:14480 rcv_ssthresh:81052 minrtt:0.005 snd_wnd:71680
ESTAB 0      0       172.28.30.41:48406 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.152/0.159 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3647 bytes_acked:3648 bytes_received:7687 segs_out:48 segs_in:38 data_segs_out:22 data_segs_in:33 send 762105263bps lastsnd:2812 lastrcv:2812 lastack:2812 pacing_rate 1524210520bps delivery_rate 4633600000bps delivered:23 app_limited busy:3ms rcv_space:14480 rcv_ssthresh:85793 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.41:48412 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.417/2.672 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3816 bytes_acked:3817 bytes_received:6187 segs_out:33 segs_in:34 data_segs_out:17 data_segs_in:29 send 81750176bps lastsnd:11401 lastrcv:11400 lastack:11400 pacing_rate 163485928bps delivery_rate 2316800000bps delivered:18 app_limited busy:43ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.41:48420 172.28.30.21:443
	 cubic wscale:10,10 rto:205 rtt:4.75/9.187 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4364 bytes_acked:4365 bytes_received:5447 segs_out:26 segs_in:23 data_segs_out:16 data_segs_in:17 send 24387368bps lastsnd:11300 lastrcv:11354 lastack:11300 pacing_rate 48770880bps delivery_rate 2896000000bps delivered:17 app_limited busy:43ms rcv_space:14480 rcv_ssthresh:85788 minrtt:0.008 snd_wnd:79872

- standalone1280_capture_0_receipt: {"bytes":"44127","sha256":"528d2f923f078a5bfdde1e571ba13e356f5686415519ab914eb830748e80ff66"}
- standalone1280_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
528 packets captured
530 packets received by filter
0 packets dropped by kernel

- standalone1280_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443169.844656 eth2  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::88a6:94ff:fec9:250d > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::a048:9dff:fea8:4507
	  source link-address option (1), length 8 (1): 8a:a6:94:c9:25:0d
1793443169.844701 eth2  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::a048:9dff:fea8:4507 > fe80::88a6:94ff:fec9:250d: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::a048:9dff:fea8:4507, Flags [solicited]
1793443170.385226 fvmb0 In  IP (tos 0x0, ttl 64, id 8213, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.48238 > 192.0.3.10.9446: Flags [S], cksum 0x16d6 (correct), seq 3244539949, win 64480, options [mss 1240,sackOK,TS val 3959773426 ecr 0,nop,wscale 10], length 0
1793443170.385247 fvup0 Out IP (tos 0x0, ttl 63, id 8213, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.48238 > 192.0.3.10.9446: Flags [S], cksum 0x1efb (correct), seq 3244539949, win 64480, options [mss 1240,sackOK,TS val 3959773426 ecr 0,nop,wscale 10], length 0
1793443170.385269 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.48238: Flags [S.], cksum 0x864d (incorrect -> 0x85ad), seq 2684061078, ack 3244539950, win 65160, options [mss 1460,sackOK,TS val 3590036010 ecr 3959773426,nop,wscale 10], length 0
1793443170.385274 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.48238: Flags [S.], cksum 0x8e72 (incorrect -> 0x7d88), seq 2684061078, ack 3244539950, win 65160, options [mss 1460,sackOK,TS val 3590036010 ecr 3959773426,nop,wscale 10], length 0

- standalone1280_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.788/0.056 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:16829 bytes_acked:16830 bytes_received:31968 segs_out:113 segs_in:87 data_segs_out:83 data_segs_in:83 send 3326700508bps lastsnd:215 lastrcv:215 lastack:215 pacing_rate 6646021232bps delivery_rate 32768000000bps delivered:84 app_limited busy:110ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.223/0.273 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4412 bytes_acked:4413 bytes_received:9823 segs_out:73 segs_in:56 data_segs_out:32 data_segs_in:51 send 519461883bps lastsnd:5216 lastrcv:5216 lastack:5216 pacing_rate 1037760352bps delivery_rate 2896000000bps delivered:33 app_limited busy:6ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:77824
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.005/1.884 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5032 bytes_acked:5033 bytes_received:6468 segs_out:40 segs_in:47 data_segs_out:23 data_segs_in:41 send 115263682bps lastsnd:11569 lastrcv:11568 lastack:11568 pacing_rate 230326824bps delivery_rate 4633600000bps delivered:24 app_limited busy:40ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.993/3.408 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4750 bytes_acked:4751 bytes_received:5117 segs_out:28 segs_in:26 data_segs_out:17 data_segs_in:18 send 58123432bps lastsnd:11522 lastrcv:11468 lastack:11468 pacing_rate 116210416bps delivery_rate 2896000000bps delivered:18 app_limited busy:44ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:210 rtt:9.411/5.088 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:23857 bytes_acked:23858 bytes_received:19094 segs_out:48 segs_in:28 data_segs_out:23 data_segs_in:25 send 426530656bps lastsnd:244 lastrcv:232 lastack:232 pacing_rate 853038648bps delivery_rate 44600888888bps delivered:24 app_limited busy:156ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:36092    127.0.0.1:9444
	 cubic wscale:10,10 rto:204 rtt:3.856/3.958 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5422 bytes_acked:5423 bytes_received:8230 segs_out:17 segs_in:10 data_segs_out:8 data_segs_in:7 send 817925311bps lastsnd:15124 lastrcv:15119 lastack:15119 pacing_rate 1635532504bps delivery_rate 39424000000bps delivered:9 app_limited busy:40ms rcv_rtt:5 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.008 snd_wnd:78848
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36092
	 cubic wscale:10,10 rto:203 rtt:2.603/5.188 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:8230 bytes_acked:8230 bytes_received:5422 segs_out:9 segs_in:17 data_segs_out:7 data_segs_in:8 send 1337533615bps lastsnd:15123 lastrcv:15128 lastack:15123 pacing_rate 2674681896bps delivery_rate 58026666664bps delivered:8 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78471 minrtt:0.006 snd_wnd:84992
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:201 rtt:0.987/1.927 ato:40 mss:45568 pmtu:65535 rcvmss:3179 advmss:65483 cwnd:10 bytes_sent:19094 bytes_acked:19094 bytes_received:23857 segs_out:27 segs_in:48 data_segs_out:25 data_segs_in:23 send 3693454914bps lastsnd:237 lastrcv:249 lastack:237 pacing_rate 7380366944bps delivery_rate 121514666664bps delivered:26 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:99450 minrtt:0.003 snd_wnd:91136
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:211 rtt:10.405/16.694 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:31968 bytes_acked:31968 bytes_received:16829 segs_out:86 segs_in:113 data_segs_out:83 data_segs_in:83 send 251940413bps lastsnd:224 lastrcv:224 lastack:183 pacing_rate 503856608bps delivery_rate 29127111104bps delivered:84 app_limited busy:584ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:207 rtt:6.579/7.058 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:28315 bytes_acked:28316 bytes_received:26230 segs_out:80 segs_in:49 data_segs_out:52 data_segs_in:34 send 17607539bps lastsnd:252 lastrcv:240 lastack:240 pacing_rate 35213064bps delivery_rate 1448000000bps delivered:53 app_limited busy:117ms rcv_rtt:5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.016 snd_wnd:79872

- standalone1280_capture_1_receipt: {"bytes":"70223","sha256":"0a592015dafa720d32a8a2f339644880217c678544f0f05657f02b321f62e1a4"}
- matrix4_input: {"input":{"underlay":"ipv4","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix4_transfer: {"after":{"upload_bytes":"204433","download_bytes":"205624","Present":true},"before":{"upload_bytes":"136677","download_bytes":"137589","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":2},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- matrix4_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
275 packets captured
280 packets received by filter
0 packets dropped by kernel

- matrix4_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443176.500645 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 16) fe80::a048:9dff:fea8:4507 > ff02::2: [icmp6 sum ok] ICMP6, router solicitation, length 16
	  source link-address option (1), length 8 (1): a2:48:9d:a8:45:07
1793443176.914229 fvnb0 Out IP (tos 0x0, ttl 64, id 29091, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.50776 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0xaee2), seq 2168988271, win 64480, options [mss 1240,sackOK,TS val 3042495362 ecr 0,nop,wscale 10], length 0
1793443176.914911 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.50776: Flags [S.], cksum 0x376f (correct), seq 2929157115, ack 2168988272, win 65160, options [mss 1460,sackOK,TS val 2190139072 ecr 3042495362,nop,wscale 10], length 0

- matrix4_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:35760 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.025/0.026 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5114 segs_out:15 segs_in:15 data_segs_out:9 data_segs_in:11 send 4633600000bps lastsnd:29202 lastrcv:29201 lastack:29201 pacing_rate 9221094520bps delivery_rate 1930666664bps delivered:10 app_limited busy:1ms rcv_space:14480 rcv_ssthresh:81052 minrtt:0.005 snd_wnd:71680
ESTAB 0      0       172.28.30.41:49976 172.28.30.21:443
	 cubic wscale:10,10 rto:204 rtt:3.562/6.791 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3461 bytes_acked:3462 bytes_received:5461 segs_out:32 segs_in:32 data_segs_out:19 data_segs_in:25 send 32521056bps lastsnd:462 lastrcv:462 lastack:462 pacing_rate 65028416bps delivery_rate 3861333328bps delivered:20 app_limited busy:45ms rcv_space:14480 rcv_ssthresh:85793 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49988 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.976/1.878 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3890 bytes_acked:3891 bytes_received:6371 segs_out:35 segs_in:40 data_segs_out:19 data_segs_in:35 send 118688525bps lastsnd:5228 lastrcv:5228 lastack:5228 pacing_rate 237255504bps delivery_rate 3861333328bps delivered:20 app_limited busy:31ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.41:50002 172.28.30.21:443
	 cubic wscale:10,10 rto:205 rtt:4.533/8.994 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2428 bytes_acked:2429 bytes_received:3078 segs_out:16 segs_in:13 data_segs_out:9 data_segs_in:9 send 25554820bps lastsnd:351 lastrcv:5227 lastack:351 pacing_rate 51102592bps delivery_rate 2574222216bps delivered:10 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:79828 minrtt:0.009 snd_wnd:77824

- matrix4_capture_0_receipt: {"bytes":"37256","sha256":"ce60b42ac7aae095fece3658c2e9bc97c4a36df92daf217c5d6593c7cc6ce9a2"}
- matrix4_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
447 packets captured
450 packets received by filter
0 packets dropped by kernel

- matrix4_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443176.500681 eth2  M   IP6 (hlim 255, next-header ICMPv6 (58) payload length: 16) fe80::a048:9dff:fea8:4507 > ff02::2: [icmp6 sum ok] ICMP6, router solicitation, length 16
	  source link-address option (1), length 8 (1): a2:48:9d:a8:45:07
1793443176.914808 fvmb0 In  IP (tos 0x0, ttl 64, id 29091, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.50776 > 192.0.3.10.9446: Flags [S], cksum 0xaee2 (correct), seq 2168988271, win 64480, options [mss 1240,sackOK,TS val 3042495362 ecr 0,nop,wscale 10], length 0
1793443176.914830 fvup0 Out IP (tos 0x0, ttl 63, id 29091, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.50776 > 192.0.3.10.9446: Flags [S], cksum 0xb707 (correct), seq 2168988271, win 64480, options [mss 1240,sackOK,TS val 3042495362 ecr 0,nop,wscale 10], length 0
1793443176.914855 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.50776: Flags [S.], cksum 0x864d (incorrect -> 0x3f94), seq 2929157115, ack 2168988272, win 65160, options [mss 1460,sackOK,TS val 2190139072 ecr 3042495362,nop,wscale 10], length 0
1793443176.914860 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.50776: Flags [S.], cksum 0x8e72 (incorrect -> 0x376f), seq 2929157115, ack 2168988272, win 65160, options [mss 1460,sackOK,TS val 2190139072 ecr 3042495362,nop,wscale 10], length 0

- matrix4_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.793/0.07 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:17767 bytes_acked:17768 bytes_received:34047 segs_out:120 segs_in:92 data_segs_out:88 data_segs_in:88 send 3305725095bps lastsnd:1737 lastrcv:1736 lastack:1736 pacing_rate 6606243496bps delivery_rate 32768000000bps delivered:89 app_limited busy:115ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.262/0.307 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4629 bytes_acked:4630 bytes_received:10313 segs_out:84 segs_in:63 data_segs_out:36 data_segs_in:58 send 442137405bps lastsnd:717 lastrcv:717 lastack:717 pacing_rate 881750712bps delivery_rate 2896000000bps delivered:37 app_limited busy:7ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:77824
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.259/0.433 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:7888 bytes_acked:7889 bytes_received:9765 segs_out:62 segs_in:77 data_segs_out:37 data_segs_in:69 send 447258687bps lastsnd:5515 lastrcv:5515 lastack:5515 pacing_rate 891505528bps delivery_rate 4633600000bps delivered:38 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:210 rtt:9.184/15.308 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5226 bytes_acked:5227 bytes_received:5677 segs_out:37 segs_in:35 data_segs_out:22 data_segs_in:24 send 12613240bps lastsnd:939 lastrcv:637 lastack:637 pacing_rate 25224760bps delivery_rate 2896000000bps delivered:23 app_limited busy:127ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:212 rtt:11.449/7.892 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:24074 bytes_acked:24075 bytes_received:19498 segs_out:50 segs_in:29 data_segs_out:24 data_segs_in:26 send 350605293bps lastsnd:1779 lastrcv:1753 lastack:1753 pacing_rate 701179960bps delivery_rate 44600888888bps delivered:25 app_limited busy:182ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:36092    127.0.0.1:9444
	 cubic wscale:10,10 rto:204 rtt:3.856/3.958 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5422 bytes_acked:5423 bytes_received:8230 segs_out:19 segs_in:11 data_segs_out:8 data_segs_in:7 send 817925311bps lastsnd:21662 lastrcv:21657 lastack:21657 pacing_rate 1635532504bps delivery_rate 39424000000bps delivered:9 app_limited busy:40ms rcv_rtt:5 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.008 snd_wnd:78848
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36092
	 cubic wscale:10,10 rto:203 rtt:2.603/5.188 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:8230 bytes_acked:8230 bytes_received:5422 segs_out:10 segs_in:18 data_segs_out:7 data_segs_in:8 send 1337533615bps lastsnd:21661 lastrcv:21666 lastack:6176 pacing_rate 2674681896bps delivery_rate 58026666664bps delivered:8 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78471 minrtt:0.006 snd_wnd:84992
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:201 rtt:0.866/1.688 ato:40 mss:45568 pmtu:65535 rcvmss:3179 advmss:65483 cwnd:10 bytes_sent:19498 bytes_acked:19498 bytes_received:24074 segs_out:28 segs_in:50 data_segs_out:26 data_segs_in:24 send 4209515012bps lastsnd:1758 lastrcv:1784 lastack:1758 pacing_rate 8417814976bps delivery_rate 121514666664bps delivered:27 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:99450 minrtt:0.003 snd_wnd:91136
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:211 rtt:10.466/16.683 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:34047 bytes_acked:34047 bytes_received:17767 segs_out:91 segs_in:120 data_segs_out:88 data_segs_in:88 send 250472005bps lastsnd:1745 lastrcv:1746 lastack:1704 pacing_rate 500926056bps delivery_rate 29127111104bps delivered:89 app_limited busy:625ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:209 rtt:8.907/9.95 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:28560 bytes_acked:28561 bytes_received:26634 segs_out:82 segs_in:50 data_segs_out:53 data_segs_in:35 send 13005501bps lastsnd:1786 lastrcv:1761 lastack:1761 pacing_rate 26009176bps delivery_rate 1448000000bps delivered:54 app_limited busy:142ms rcv_rtt:5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.016 snd_wnd:79872

- matrix4_capture_1_receipt: {"bytes":"59959","sha256":"65430b58fde018ca50458519498d112782e77729a4685f612bd45dc2a30213b0"}
- matrix6_input: {"input":{"underlay":"ipv6","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix6_transfer: {"after":{"upload_bytes":"271045","download_bytes":"273555","Present":true},"before":{"upload_bytes":"204433","download_bytes":"205624","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":205},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"}}
- matrix6_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
271 packets captured
275 packets received by filter
0 packets dropped by kernel

- matrix6_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443183.157655 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::a048:9dff:fea8:4507 > 2001:db8:30::1: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has 2001:db8:30::1
	  source link-address option (1), length 8 (1): a2:48:9d:a8:45:07
1793443183.157704 eth1  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) 2001:db8:30::1 > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is 2001:db8:30::1, Flags [router, solicited]
1793443183.463806 fvnb0 Out IP (tos 0x0, ttl 64, id 57171, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.50778 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0xf048), seq 3923554633, win 64480, options [mss 1240,sackOK,TS val 3754715191 ecr 0,nop,wscale 10], length 0
1793443183.667186 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.50778: Flags [S.], cksum 0x9db6 (correct), seq 1109133914, ack 3923554634, win 65160, options [mss 1460,sackOK,TS val 1011336767 ecr 3754715191,nop,wscale 10], length 0

- matrix6_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":6,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:35760 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.025/0.026 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5114 segs_out:16 segs_in:16 data_segs_out:9 data_segs_in:11 send 4633600000bps lastsnd:35943 lastrcv:35942 lastack:5457 pacing_rate 9221094520bps delivery_rate 1930666664bps delivered:10 app_limited busy:1ms rcv_space:14480 rcv_ssthresh:81052 minrtt:0.005 snd_wnd:71680
ESTAB 0      0       172.28.30.41:49452 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.273/0.25 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3797 bytes_acked:3798 bytes_received:6208 segs_out:50 segs_in:41 data_segs_out:25 data_segs_in:36 send 424322344bps lastsnd:187 lastrcv:187 lastack:187 pacing_rate 846706256bps delivery_rate 4633600000bps delivered:26 app_limited busy:4ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49456 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.895/3.592 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3310 bytes_acked:3311 bytes_received:5814 segs_out:25 segs_in:27 data_segs_out:14 data_segs_in:22 send 61129288bps lastsnd:5440 lastrcv:5440 lastack:5440 pacing_rate 122218264bps delivery_rate 2896000000bps delivered:15 app_limited busy:29ms rcv_space:14480 rcv_ssthresh:85793 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49466 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.103/10.152 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2166 bytes_acked:2167 bytes_received:2984 segs_out:14 segs_in:11 data_segs_out:7 data_segs_in:8 send 22700372bps lastsnd:5440 lastrcv:5440 lastack:5399 pacing_rate 45398520bps delivery_rate 1544533328bps delivered:8 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:79828 minrtt:0.009 snd_wnd:74752

- matrix6_capture_0_receipt: {"bytes":"36948","sha256":"4f510f466d4599c0390ccdc7eff469801315cbd603ee77e4e660110b56a6ace3"}
- matrix6_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
393 packets captured
393 packets received by filter
0 packets dropped by kernel

- matrix6_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443183.666735 fvmb0 In  IP (tos 0x0, ttl 64, id 57171, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.50778 > 192.0.3.10.9446: Flags [S], cksum 0xf048 (correct), seq 3923554633, win 64480, options [mss 1240,sackOK,TS val 3754715191 ecr 0,nop,wscale 10], length 0
1793443183.666760 fvup0 Out IP (tos 0x0, ttl 63, id 57171, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.50778 > 192.0.3.10.9446: Flags [S], cksum 0xf86d (correct), seq 3923554633, win 64480, options [mss 1240,sackOK,TS val 3754715191 ecr 0,nop,wscale 10], length 0
1793443183.666785 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.50778: Flags [S.], cksum 0x864d (incorrect -> 0xa5db), seq 1109133914, ack 3923554634, win 65160, options [mss 1460,sackOK,TS val 1011336767 ecr 3754715191,nop,wscale 10], length 0
1793443183.666791 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.50778: Flags [S.], cksum 0x8e72 (incorrect -> 0x9db6), seq 1109133914, ack 3923554634, win 65160, options [mss 1460,sackOK,TS val 1011336767 ecr 3754715191,nop,wscale 10], length 0

- matrix6_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.781/0.061 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:19643 bytes_acked:19644 bytes_received:38205 segs_out:134 segs_in:102 data_segs_out:98 data_segs_in:98 send 3356517286bps lastsnd:3213 lastrcv:3212 lastack:3212 pacing_rate 6705521976bps delivery_rate 32768000000bps delivered:99 app_limited busy:124ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.272/0.312 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4734 bytes_acked:4735 bytes_received:10569 segs_out:89 segs_in:67 data_segs_out:38 data_segs_in:62 send 425882353bps lastsnd:791 lastrcv:791 lastack:791 pacing_rate 850201832bps delivery_rate 2896000000bps delivered:39 app_limited busy:8ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:77824
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.176/0.219 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:10612 bytes_acked:10613 bytes_received:12604 segs_out:86 segs_in:102 data_segs_out:49 data_segs_in:92 send 658181818bps lastsnd:5603 lastrcv:5603 lastack:5603 pacing_rate 1311705584bps delivery_rate 4633600000bps delivered:50 app_limited busy:43ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:207 rtt:6.203/10.916 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5574 bytes_acked:5575 bytes_received:5957 segs_out:42 segs_in:40 data_segs_out:25 data_segs_in:27 send 18674835bps lastsnd:5603 lastrcv:5603 lastack:5603 pacing_rate 37348160bps delivery_rate 2896000000bps delivered:26 app_limited busy:127ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:212 rtt:11.09/5.017 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:27489 bytes_acked:27490 bytes_received:22590 segs_out:56 segs_in:33 data_segs_out:26 data_segs_in:30 send 361954914bps lastsnd:3098 lastrcv:3087 lastack:3087 pacing_rate 723869032bps delivery_rate 44600888888bps delivered:27 app_limited busy:202ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:36092    127.0.0.1:9444
	 cubic wscale:10,10 rto:205 rtt:4.285/3.827 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5962 bytes_acked:5963 bytes_received:9386 segs_out:21 segs_in:12 data_segs_out:9 data_segs_in:8 send 736037340bps lastsnd:3238 lastrcv:3231 lastack:3231 pacing_rate 1471859992bps delivery_rate 39424000000bps delivered:10 app_limited busy:47ms rcv_rtt:5.25 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.008 snd_wnd:78848
ESTAB 0      0          127.0.0.1:59158    127.0.0.1:9444
	 cubic wscale:10,10 rto:203 rtt:2.529/5.018 ato:40 mss:39424 pmtu:65535 rcvmss:1995 advmss:65483 cwnd:10 bytes_sent:4006 bytes_acked:4007 bytes_received:2777 segs_out:8 segs_in:5 data_segs_out:4 data_segs_in:2 send 1247101621bps lastsnd:3496 lastrcv:3476 lastack:3476 pacing_rate 2493833456bps delivery_rate 24260923072bps delivered:5 app_limited busy:20ms rcv_space:65495 rcv_ssthresh:86893 minrtt:0.004 snd_wnd:78848
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36092
	 cubic wscale:10,10 rto:203 rtt:2.278/4.54 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:9386 bytes_acked:9386 bytes_received:5962 segs_out:11 segs_in:20 data_segs_out:8 data_segs_in:9 send 1528358209bps lastsnd:3235 lastrcv:3242 lastack:3235 pacing_rate 3055710360bps delivery_rate 58026666664bps delivered:9 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78471 minrtt:0.006 snd_wnd:83968
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:201 rtt:0.51/0.998 ato:40 mss:45568 pmtu:65535 rcvmss:3198 advmss:65483 cwnd:10 bytes_sent:22590 bytes_acked:22590 bytes_received:27489 segs_out:32 segs_in:56 data_segs_out:30 data_segs_in:26 send 7147921569bps lastsnd:3092 lastrcv:3103 lastack:3092 pacing_rate 14288838800bps delivery_rate 182272000000bps delivered:31 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:99450 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:59158
	 cubic wscale:10,10 rto:206 rtt:5.06/10.097 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2777 bytes_acked:2777 bytes_received:4006 segs_out:4 segs_in:8 data_segs_out:2 data_segs_in:4 send 688063241bps lastsnd:3480 lastrcv:3500 lastack:3439 pacing_rate 1375922536bps delivery_rate 31650909088bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78342 minrtt:0.011 snd_wnd:87040
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:211 rtt:10.488/16.772 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:38205 bytes_acked:38205 bytes_received:19643 segs_out:101 segs_in:134 data_segs_out:98 data_segs_in:98 send 249946606bps lastsnd:3221 lastrcv:3222 lastack:3180 pacing_rate 499851504bps delivery_rate 29127111104bps delivered:99 app_limited busy:707ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:207 rtt:6.122/7.35 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:34484 bytes_acked:34485 bytes_received:31620 segs_out:96 segs_in:58 data_segs_out:63 data_segs_in:40 send 18921921bps lastsnd:3105 lastrcv:3095 lastack:3095 pacing_rate 37843840bps delivery_rate 1448000000bps delivered:64 app_limited busy:151ms rcv_rtt:5.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.013 snd_wnd:79872

- matrix6_capture_1_receipt: {"bytes":"52794","sha256":"16bed71e2c9ca0a1c565c0a68373e8059bceadf59552a957bf3e7c9f006596dc"}
- matrixDual_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrixDual_transfer: {"after":{"upload_bytes":"337865","download_bytes":"341746","Present":true},"before":{"upload_bytes":"271045","download_bytes":"273555","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"}}
- matrixDual_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
283 packets captured
287 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443189.812782 eth1  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::8c8c:34ff:fe9d:d24 > 2001:db8:30::41: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has 2001:db8:30::41
	  source link-address option (1), length 8 (1): 8e:8c:34:9d:0d:24
1793443189.812797 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) 2001:db8:30::41 > fe80::8c8c:34ff:fe9d:d24: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is 2001:db8:30::41, Flags [solicited]
1793443190.145041 fvnb0 Out IP (tos 0x0, ttl 64, id 57444, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.47890 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0xa567), seq 936959863, win 64480, options [mss 1240,sackOK,TS val 3890035238 ecr 0,nop,wscale 10], length 0
1793443190.145358 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.47890: Flags [S.], cksum 0x2507 (correct), seq 1831053098, ack 936959864, win 65160, options [mss 1460,sackOK,TS val 38495794 ecr 3890035238,nop,wscale 10], length 0

- matrixDual_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":7,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:35760 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.025/0.026 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5114 segs_out:16 segs_in:16 data_segs_out:9 data_segs_in:11 send 4633600000bps lastsnd:42435 lastrcv:42434 lastack:11949 pacing_rate 9221094520bps delivery_rate 1930666664bps delivered:10 app_limited busy:1ms rcv_space:14480 rcv_ssthresh:81052 minrtt:0.005 snd_wnd:71680
ESTAB 0      0       172.28.30.41:49478 172.28.30.21:443
	 cubic wscale:10,10 rto:203 rtt:2.797/5.273 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3625 bytes_acked:3626 bytes_received:5778 segs_out:40 segs_in:38 data_segs_out:22 data_segs_in:31 send 41415803bps lastsnd:478 lastrcv:478 lastack:478 pacing_rate 82805696bps delivery_rate 2896000000bps delivered:23 app_limited busy:44ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49484 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.133/2.132 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3853 bytes_acked:3854 bytes_received:6257 segs_out:34 segs_in:36 data_segs_out:19 data_segs_in:31 send 102241836bps lastsnd:5198 lastrcv:5198 lastack:5198 pacing_rate 204370928bps delivery_rate 2896000000bps delivered:20 app_limited busy:35ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49498 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.099/0.126 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2006 bytes_acked:2007 bytes_received:2772 segs_out:12 segs_in:9 data_segs_out:6 data_segs_in:7 send 1170101010bps lastsnd:5200 lastrcv:5200 lastack:5200 pacing_rate 2328442208bps delivery_rate 2316800000bps delivered:7 app_limited busy:1ms rcv_space:14480 rcv_ssthresh:76932 minrtt:0.01 snd_wnd:71680

- matrixDual_capture_0_receipt: {"bytes":"38324","sha256":"2943763c56fa70bdc92e5e07f9782848814e5e5096ffba460da414f956f009d9"}
- matrixDual_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
413 packets captured
413 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443190.145253 fvmb0 In  IP (tos 0x0, ttl 64, id 57444, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.47890 > 192.0.3.10.9446: Flags [S], cksum 0xa567 (correct), seq 936959863, win 64480, options [mss 1240,sackOK,TS val 3890035238 ecr 0,nop,wscale 10], length 0
1793443190.145283 fvup0 Out IP (tos 0x0, ttl 63, id 57444, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.47890 > 192.0.3.10.9446: Flags [S], cksum 0xad8c (correct), seq 936959863, win 64480, options [mss 1240,sackOK,TS val 3890035238 ecr 0,nop,wscale 10], length 0
1793443190.145312 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.47890: Flags [S.], cksum 0x864d (incorrect -> 0x2d2c), seq 1831053098, ack 936959864, win 65160, options [mss 1460,sackOK,TS val 38495794 ecr 3890035238,nop,wscale 10], length 0
1793443190.145318 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.47890: Flags [S.], cksum 0x8e72 (incorrect -> 0x2507), seq 1831053098, ack 936959864, win 65160, options [mss 1460,sackOK,TS val 38495794 ecr 3890035238,nop,wscale 10], length 0

- matrixDual_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.694/0.07 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:21519 bytes_acked:21520 bytes_received:42363 segs_out:147 segs_in:112 data_segs_out:108 data_segs_in:108 send 3777291066bps lastsnd:3 lastrcv:2 lastack:2 pacing_rate 7546426768bps delivery_rate 32768000000bps delivered:109 app_limited busy:132ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.232/0.249 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4846 bytes_acked:4847 bytes_received:10860 segs_out:94 segs_in:72 data_segs_out:40 data_segs_in:67 send 499310345bps lastsnd:757 lastrcv:756 lastack:756 pacing_rate 995937664bps delivery_rate 2896000000bps delivered:41 app_limited busy:10ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:77824
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.156/0.18 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:13306 bytes_acked:13307 bytes_received:15744 segs_out:104 segs_in:124 data_segs_out:61 data_segs_in:112 send 742564103bps lastsnd:5515 lastrcv:5514 lastack:5514 pacing_rate 1483939144bps delivery_rate 4633600000bps delivered:62 app_limited busy:46ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:210 rtt:9.161/15.155 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5706 bytes_acked:5707 bytes_received:6081 segs_out:46 segs_in:44 data_segs_out:27 data_segs_in:29 send 12644908bps lastsnd:5516 lastrcv:5516 lastack:5516 pacing_rate 25287744bps delivery_rate 2896000000bps delivered:28 app_limited busy:168ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:212 rtt:11.347/3.291 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:27923 bytes_acked:27924 bytes_received:23398 segs_out:60 segs_in:35 data_segs_out:28 data_segs_in:32 send 353756940bps lastsnd:32 lastrcv:19 lastack:19 pacing_rate 707490496bps delivery_rate 44600888888bps delivered:29 app_limited busy:227ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:36092    127.0.0.1:9444
	 cubic wscale:10,10 rto:205 rtt:4.285/3.827 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5962 bytes_acked:5963 bytes_received:9386 segs_out:21 segs_in:12 data_segs_out:9 data_segs_in:8 send 736037340bps lastsnd:9884 lastrcv:9877 lastack:9877 pacing_rate 1471859992bps delivery_rate 39424000000bps delivered:10 app_limited busy:47ms rcv_rtt:5.25 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.008 snd_wnd:78848
ESTAB 0      0          127.0.0.1:59158    127.0.0.1:9444
	 cubic wscale:10,10 rto:203 rtt:2.529/5.018 ato:40 mss:39424 pmtu:65535 rcvmss:1995 advmss:65483 cwnd:10 bytes_sent:4006 bytes_acked:4007 bytes_received:2777 segs_out:8 segs_in:5 data_segs_out:4 data_segs_in:2 send 1247101621bps lastsnd:10142 lastrcv:10122 lastack:10122 pacing_rate 2493833456bps delivery_rate 24260923072bps delivered:5 app_limited busy:20ms rcv_space:65495 rcv_ssthresh:86893 minrtt:0.004 snd_wnd:78848
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36092
	 cubic wscale:10,10 rto:203 rtt:2.278/4.54 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:9386 bytes_acked:9386 bytes_received:5962 segs_out:11 segs_in:20 data_segs_out:8 data_segs_in:9 send 1528358209bps lastsnd:9881 lastrcv:9888 lastack:9881 pacing_rate 3055710360bps delivery_rate 58026666664bps delivered:9 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78471 minrtt:0.006 snd_wnd:83968
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:201 rtt:0.395/0.761 ato:40 mss:45568 pmtu:65535 rcvmss:3198 advmss:65483 cwnd:10 bytes_sent:23398 bytes_acked:23398 bytes_received:27923 segs_out:34 segs_in:60 data_segs_out:32 data_segs_in:28 send 9228962025bps lastsnd:24 lastrcv:37 lastack:24 pacing_rate 18457924048bps delivery_rate 182272000000bps delivered:33 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:99450 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:59158
	 cubic wscale:10,10 rto:206 rtt:5.06/10.097 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2777 bytes_acked:2777 bytes_received:4006 segs_out:4 segs_in:8 data_segs_out:2 data_segs_in:4 send 688063241bps lastsnd:10126 lastrcv:10146 lastack:10085 pacing_rate 1375922536bps delivery_rate 31650909088bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78342 minrtt:0.011 snd_wnd:87040
ESTAB 0      290        127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:207 rtt:6.173/10.979 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:42363 bytes_acked:42073 bytes_received:21519 segs_out:111 segs_in:147 data_segs_out:108 data_segs_in:108 send 424662239bps lastsnd:11 lastrcv:12 lastack:12 pacing_rate 849272880bps delivery_rate 29127111104bps delivered:108 app_limited busy:760ms unacked:1 rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:208 rtt:7.47/6.493 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:34974 bytes_acked:34975 bytes_received:32428 segs_out:100 segs_in:60 data_segs_out:65 data_segs_in:42 send 15507363bps lastsnd:40 lastrcv:28 lastack:28 pacing_rate 31011088bps delivery_rate 1448000000bps delivered:66 app_limited busy:174ms rcv_rtt:5.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.013 snd_wnd:79872

- matrixDual_capture_1_receipt: {"bytes":"55142","sha256":"708e86a263ed495e0871edbb112ef5f7d02d5302b0380a31e85c4c6911d4cd02"}
- matrix1280_input: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix1280_transfer: {"after":{"upload_bytes":"403861","download_bytes":"357533","Present":true},"before":{"upload_bytes":"337865","download_bytes":"341746","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":2002,"milliseconds":12010,"error":"bulk_echo_incomplete"},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"}}
- matrix1280_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
223 packets captured
228 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443196.806540 fvnb0 Out IP (tos 0x0, ttl 64, id 55023, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.35316 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0xe24b), seq 3308675523, win 64480, options [mss 1240,sackOK,TS val 25036054 ecr 0,nop,wscale 10], length 0
1793443197.812653 fvnb0 Out IP (tos 0x0, ttl 64, id 55024, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.35316 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0xde5c), seq 3308675523, win 64480, options [mss 1240,sackOK,TS val 25037061 ecr 0,nop,wscale 10], length 0
1793443197.813649 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.35316: Flags [S.], cksum 0x3ec7 (correct), seq 2110808769, ack 3308675524, win 65160, options [mss 1460,sackOK,TS val 1725716609 ecr 25036054,nop,wscale 10], length 0
1793443199.540698 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::a048:9dff:fea8:4507 > fe80::88a6:94ff:fec9:250d: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::88a6:94ff:fec9:250d
	  source link-address option (1), length 8 (1): a2:48:9d:a8:45:07
1793443199.540740 eth1  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::88a6:94ff:fec9:250d > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::88a6:94ff:fec9:250d, Flags [solicited]
1793443200.052809 eth1  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::8c8c:34ff:fe9d:d24 > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::a048:9dff:fea8:4507
	  source link-address option (1), length 8 (1): 8e:8c:34:9d:0d:24
1793443200.052831 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::a048:9dff:fea8:4507 > fe80::8c8c:34ff:fe9d:d24: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::a048:9dff:fea8:4507, Flags [solicited]
1793443204.661693 eth1  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::88a6:94ff:fec9:250d > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::a048:9dff:fea8:4507
	  source link-address option (1), length 8 (1): 8a:a6:94:c9:25:0d
1793443204.661724 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::a048:9dff:fea8:4507 > fe80::88a6:94ff:fec9:250d: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::a048:9dff:fea8:4507, Flags [solicited]

- matrix1280_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":8,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State      Recv-Q Send-Q  Local Address:Port  Peer Address:PortProcess
ESTAB      0      0        172.28.30.41:35760 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.025/0.026 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5114 segs_out:17 segs_in:17 data_segs_out:9 data_segs_in:11 send 4633600000bps lastsnd:61103 lastrcv:61102 lastack:408 pacing_rate 9221094520bps delivery_rate 1930666664bps delivered:10 app_limited busy:1ms rcv_space:14480 rcv_ssthresh:81052 minrtt:0.005 snd_wnd:71680
ESTAB      0      0        172.28.30.41:48764 172.28.30.21:443
	 cubic wscale:10,10 rto:203 rtt:2.92/5.329 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3573 bytes_acked:3574 bytes_received:5737 segs_out:39 segs_in:37 data_segs_out:21 data_segs_in:30 send 33643836bps lastsnd:277 lastrcv:277 lastack:277 pacing_rate 67273264bps delivery_rate 2806857136bps delivered:22 app_limited busy:46ms rcv_space:12280 rcv_ssthresh:85803 minrtt:0.007 snd_wnd:79872
ESTAB      0      0        172.28.30.41:48778 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.995/1.886 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3986 bytes_acked:3987 bytes_received:6371 segs_out:39 segs_in:42 data_segs_out:22 data_segs_in:35 send 98733668bps lastsnd:17327 lastrcv:17327 lastack:2320 pacing_rate 197293832bps delivery_rate 1786181816bps delivered:23 app_limited busy:38ms rcv_space:12280 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB      0      0        172.28.30.41:48780 172.28.30.21:443
	 cubic wscale:10,10 rto:205 rtt:4.474/8.874 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:2428 bytes_acked:2429 bytes_received:3078 segs_out:16 segs_in:13 data_segs_out:9 data_segs_in:9 send 21957979bps lastsnd:12359 lastrcv:17327 lastack:12359 pacing_rate 43913504bps delivery_rate 1786181816bps delivered:10 app_limited busy:42ms rcv_space:12280 rcv_ssthresh:78068 minrtt:0.009 snd_wnd:77824
FIN-WAIT-2 0      0      100.67.102.246:35316   192.0.3.10:9446


- matrix1280_capture_0_receipt: {"bytes":"30468","sha256":"14cf7751df24fd39c86d694cc20917eb26fb2f083fd6959bb5c464ebb154d93f"}
- matrix1280_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
320 packets captured
327 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443196.807248 fvmb0 In  IP (tos 0x0, ttl 64, id 55023, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.35316 > 192.0.3.10.9446: Flags [S], cksum 0xe24b (correct), seq 3308675523, win 64480, options [mss 1240,sackOK,TS val 25036054 ecr 0,nop,wscale 10], length 0
1793443196.807270 fvup0 Out IP (tos 0x0, ttl 63, id 55023, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.35316 > 192.0.3.10.9446: Flags [S], cksum 0xea70 (correct), seq 3308675523, win 64480, options [mss 1240,sackOK,TS val 25036054 ecr 0,nop,wscale 10], length 0
1793443196.807297 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.35316: Flags [S.], cksum 0x864d (incorrect -> 0x4ada), seq 2110808769, ack 3308675524, win 65160, options [mss 1460,sackOK,TS val 1725715603 ecr 25036054,nop,wscale 10], length 0
1793443196.807303 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.35316: Flags [S.], cksum 0x8e72 (incorrect -> 0x42b5), seq 2110808769, ack 3308675524, win 65160, options [mss 1460,sackOK,TS val 1725715603 ecr 25036054,nop,wscale 10], length 0
1793443197.812796 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.35316: Flags [S.], cksum 0x864d (incorrect -> 0x46ec), seq 2110808769, ack 3308675524, win 65160, options [mss 1460,sackOK,TS val 1725716609 ecr 25036054,nop,wscale 10], length 0
1793443197.812812 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.35316: Flags [S.], cksum 0x8e72 (incorrect -> 0x3ec7), seq 2110808769, ack 3308675524, win 65160, options [mss 1460,sackOK,TS val 1725716609 ecr 25036054,nop,wscale 10], length 0
1793443197.813275 fvmb0 In  IP (tos 0x0, ttl 64, id 55024, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.35316 > 192.0.3.10.9446: Flags [S], cksum 0xde5c (correct), seq 3308675523, win 64480, options [mss 1240,sackOK,TS val 25037061 ecr 0,nop,wscale 10], length 0
1793443197.813291 fvup0 Out IP (tos 0x0, ttl 63, id 55024, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.35316 > 192.0.3.10.9446: Flags [S], cksum 0xe681 (correct), seq 3308675523, win 64480, options [mss 1240,sackOK,TS val 25037061 ecr 0,nop,wscale 10], length 0
1793443197.813305 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.35316: Flags [S.], cksum 0x864d (incorrect -> 0x46ec), seq 2110808769, ack 3308675524, win 65160, options [mss 1460,sackOK,TS val 1725716609 ecr 25036054,nop,wscale 10], length 0
1793443197.813309 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.35316: Flags [S.], cksum 0x8e72 (incorrect -> 0x3ec7), seq 2110808769, ack 3308675524, win 65160, options [mss 1460,sackOK,TS val 1725716609 ecr 25036054,nop,wscale 10], length 0
1793443199.540714 eth2  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::a048:9dff:fea8:4507 > fe80::88a6:94ff:fec9:250d: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::88a6:94ff:fec9:250d
	  source link-address option (1), length 8 (1): a2:48:9d:a8:45:07
1793443199.540737 eth2  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::88a6:94ff:fec9:250d > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::88a6:94ff:fec9:250d, Flags [solicited]
1793443204.661669 eth2  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::88a6:94ff:fec9:250d > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::a048:9dff:fea8:4507
	  source link-address option (1), length 8 (1): 8a:a6:94:c9:25:0d
1793443204.661726 eth2  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::a048:9dff:fea8:4507 > fe80::88a6:94ff:fec9:250d: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::a048:9dff:fea8:4507, Flags [solicited]

- matrix1280_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.757/0.06 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:24333 bytes_acked:24334 bytes_received:48600 segs_out:169 segs_in:127 data_segs_out:123 data_segs_in:123 send 3462932629bps lastsnd:3634 lastrcv:3633 lastack:3633 pacing_rate 6920151784bps delivery_rate 32768000000bps delivered:124 app_limited busy:144ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.338/0.363 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5443 bytes_acked:5444 bytes_received:11887 segs_out:113 segs_in:85 data_segs_out:48 data_segs_in:80 send 342721893bps lastsnd:3642 lastrcv:3642 lastack:3642 pacing_rate 685190384bps delivery_rate 2896000000bps delivered:49 app_limited busy:14ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.121/0.114 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:16089 bytes_acked:16090 bytes_received:19068 segs_out:130 segs_in:157 data_segs_out:74 data_segs_in:142 send 957355372bps lastsnd:17628 lastrcv:17628 lastack:2246 pacing_rate 1906831272bps delivery_rate 4633600000bps delivered:75 app_limited busy:49ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:213 rtt:12.861/18.928 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:6182 bytes_acked:6183 bytes_received:6641 segs_out:55 segs_in:53 data_segs_out:32 data_segs_in:35 send 9007076bps lastsnd:7854 lastrcv:7854 lastack:7813 pacing_rate 18013448bps delivery_rate 2896000000bps delivered:33 app_limited busy:251ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:212 rtt:11.612/1.818 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:28574 bytes_acked:28575 bytes_received:24610 segs_out:66 segs_in:38 data_segs_out:31 data_segs_in:35 send 345683775bps lastsnd:3668 lastrcv:3656 lastack:3656 pacing_rate 691352664bps delivery_rate 44600888888bps delivered:32 app_limited busy:264ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:36092    127.0.0.1:9444
	 cubic wscale:10,10 rto:205 rtt:4.285/3.827 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5962 bytes_acked:5963 bytes_received:9386 segs_out:23 segs_in:14 data_segs_out:9 data_segs_in:8 send 736037340bps lastsnd:28536 lastrcv:28529 lastack:13511 pacing_rate 1471859992bps delivery_rate 39424000000bps delivered:10 app_limited busy:47ms rcv_rtt:5.25 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.008 snd_wnd:78848
ESTAB 0      0          127.0.0.1:59158    127.0.0.1:9444
	 cubic wscale:10,10 rto:203 rtt:2.529/5.018 ato:40 mss:39424 pmtu:65535 rcvmss:1995 advmss:65483 cwnd:10 bytes_sent:4006 bytes_acked:4007 bytes_received:2777 segs_out:10 segs_in:7 data_segs_out:4 data_segs_in:2 send 1247101621bps lastsnd:28793 lastrcv:28773 lastack:13510 pacing_rate 2493833456bps delivery_rate 24260923072bps delivered:5 app_limited busy:20ms rcv_space:65495 rcv_ssthresh:86893 minrtt:0.004 snd_wnd:78848
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36092
	 cubic wscale:10,10 rto:203 rtt:2.278/4.54 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:9386 bytes_acked:9386 bytes_received:5962 segs_out:13 segs_in:22 data_segs_out:8 data_segs_in:9 send 1528358209bps lastsnd:28533 lastrcv:28540 lastack:13515 pacing_rate 3055710360bps delivery_rate 58026666664bps delivered:9 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78471 minrtt:0.006 snd_wnd:83968
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:201 rtt:0.268/0.511 ato:40 mss:45568 pmtu:65535 rcvmss:3198 advmss:65483 cwnd:10 bytes_sent:24610 bytes_acked:24610 bytes_received:28574 segs_out:37 segs_in:66 data_segs_out:35 data_segs_in:31 send 13602388060bps lastsnd:3661 lastrcv:3673 lastack:3661 pacing_rate 27141479752bps delivery_rate 182272000000bps delivered:36 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:99450 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:59158
	 cubic wscale:10,10 rto:206 rtt:5.06/10.097 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2777 bytes_acked:2777 bytes_received:4006 segs_out:6 segs_in:10 data_segs_out:2 data_segs_in:4 send 688063241bps lastsnd:28778 lastrcv:28798 lastack:13516 pacing_rate 1375922536bps delivery_rate 31650909088bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78342 minrtt:0.011 snd_wnd:87040
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:211 rtt:10.533/16.906 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:48600 bytes_acked:48600 bytes_received:24333 segs_out:126 segs_in:169 data_segs_out:123 data_segs_in:123 send 248878762bps lastsnd:3642 lastrcv:3643 lastack:3601 pacing_rate 497757520bps delivery_rate 29127111104bps delivered:124 app_limited busy:914ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:209 rtt:8.843/4.859 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:35709 bytes_acked:35710 bytes_received:33640 segs_out:106 segs_in:63 data_segs_out:68 data_segs_in:45 send 13099627bps lastsnd:3676 lastrcv:3664 lastack:3664 pacing_rate 26198880bps delivery_rate 1448000000bps delivered:69 app_limited busy:209ms rcv_rtt:5.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.013 snd_wnd:79872

- matrix1280_capture_1_receipt: {"bytes":"43097","sha256":"69b79aed6ec9ce8b0d5654fa78fdadf4a37e37908d5f0f8517f26efb8d35e722"}
- ipv6_1280_input: {"input":{"underlay":"ipv6","mtu":1280,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv6_1280_transfer: {"after":{"upload_bytes":"469405","download_bytes":"416008","Present":true},"before":{"upload_bytes":"403861","download_bytes":"358865","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":10010,"milliseconds":12011,"error":"bulk_echo_incomplete"},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"}}
- ipv6_1280_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
259 packets captured
263 packets received by filter
0 packets dropped by kernel

- ipv6_1280_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443215.413652 eth1  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::a048:9dff:fea8:4507 > 2001:db8:30::1: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has 2001:db8:30::1
	  source link-address option (1), length 8 (1): a2:48:9d:a8:45:07
1793443215.413686 eth1  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) 2001:db8:30::1 > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is 2001:db8:30::1, Flags [router, solicited]
1793443215.427964 fvnb0 Out IP (tos 0x0, ttl 64, id 45104, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.37446 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0x3ecc), seq 2057858330, win 64480, options [mss 1240,sackOK,TS val 889433077 ecr 0,nop,wscale 10], length 0
1793443215.572298 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.37446: Flags [S.], cksum 0x4eb0 (correct), seq 3909650624, ack 2057858331, win 65160, options [mss 1460,sackOK,TS val 872171201 ecr 889433077,nop,wscale 10], length 0

- ipv6_1280_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":9,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State      Recv-Q Send-Q  Local Address:Port  Peer Address:PortProcess
ESTAB      0      0        172.28.30.41:38214 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.79/2.925 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3835 bytes_acked:3836 bytes_received:6224 segs_out:47 segs_in:42 data_segs_out:26 data_segs_in:36 send 54882682bps lastsnd:190 lastrcv:190 lastack:190 pacing_rate 109742368bps delivery_rate 2456000000bps delivered:27 app_limited busy:45ms rcv_space:12280 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB      0      0        172.28.30.41:38226 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.613/3.095 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3466 bytes_acked:3467 bytes_received:5932 segs_out:32 segs_in:32 data_segs_out:17 data_segs_in:25 send 60905146bps lastsnd:17314 lastrcv:17314 lastack:2307 pacing_rate 121810288bps delivery_rate 2183111104bps delivered:18 app_limited busy:42ms rcv_space:12280 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB      0      0        172.28.30.41:38230 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.068/10.084 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:2166 bytes_acked:2167 bytes_received:2984 segs_out:15 segs_in:12 data_segs_out:7 data_segs_in:8 send 19384373bps lastsnd:17315 lastrcv:17315 lastack:2268 pacing_rate 38762048bps delivery_rate 1964800000bps delivered:8 app_limited busy:41ms rcv_space:12280 rcv_ssthresh:78068 minrtt:0.01 snd_wnd:74752
FIN-WAIT-2 0      0      100.67.102.246:35316   192.0.3.10:9446

FIN-WAIT-2 0      0      100.67.102.246:37446   192.0.3.10:9446


- ipv6_1280_capture_0_receipt: {"bytes":"36154","sha256":"4b8e4f3133014fc84e8556da4a6e901b0ae0735db08e3ed82d69ea4b18359cbc"}
- ipv6_1280_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
373 packets captured
379 packets received by filter
0 packets dropped by kernel

- ipv6_1280_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443215.571733 fvmb0 In  IP (tos 0x0, ttl 64, id 45104, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.37446 > 192.0.3.10.9446: Flags [S], cksum 0x3ecc (correct), seq 2057858330, win 64480, options [mss 1240,sackOK,TS val 889433077 ecr 0,nop,wscale 10], length 0
1793443215.571758 fvup0 Out IP (tos 0x0, ttl 63, id 45104, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.37446 > 192.0.3.10.9446: Flags [S], cksum 0x46f1 (correct), seq 2057858330, win 64480, options [mss 1240,sackOK,TS val 889433077 ecr 0,nop,wscale 10], length 0
1793443215.571781 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.37446: Flags [S.], cksum 0x864d (incorrect -> 0x56d5), seq 3909650624, ack 2057858331, win 65160, options [mss 1460,sackOK,TS val 872171201 ecr 889433077,nop,wscale 10], length 0
1793443215.571787 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.37446: Flags [S.], cksum 0x8e72 (incorrect -> 0x4eb0), seq 3909650624, ack 2057858331, win 65160, options [mss 1460,sackOK,TS val 872171201 ecr 889433077,nop,wscale 10], length 0

- ipv6_1280_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.815/0.105 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:28085 bytes_acked:28086 bytes_received:56916 segs_out:197 segs_in:147 data_segs_out:143 data_segs_in:143 send 3216490798bps lastsnd:2245 lastrcv:2244 lastack:2244 pacing_rate 6430022992bps delivery_rate 32768000000bps delivered:144 app_limited busy:161ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.348/0.359 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5758 bytes_acked:5759 bytes_received:12611 segs_out:129 segs_in:95 data_segs_out:54 data_segs_in:90 send 332873563bps lastsnd:2251 lastrcv:2251 lastack:2251 pacing_rate 665508072bps delivery_rate 2896000000bps delivered:55 app_limited busy:17ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.069/0.062 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:18932 bytes_acked:18933 bytes_received:22082 segs_out:159 segs_in:190 data_segs_out:89 data_segs_in:171 send 1678840580bps lastsnd:17606 lastrcv:17606 lastack:2552 pacing_rate 3351609400bps delivery_rate 4633600000bps delivered:90 app_limited busy:50ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:213 rtt:12.627/18.537 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:6560 bytes_acked:6561 bytes_received:6947 segs_out:62 segs_in:61 data_segs_out:36 data_segs_in:39 send 9173992bps lastsnd:1464 lastrcv:1464 lastack:1423 pacing_rate 18346888bps delivery_rate 2896000000bps delivered:37 app_limited busy:293ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:211 rtt:10.792/1.785 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:32652 bytes_acked:32653 bytes_received:26753 segs_out:76 segs_in:43 data_segs_out:36 data_segs_in:40 send 371949592bps lastsnd:2275 lastrcv:2262 lastack:2262 pacing_rate 743881952bps delivery_rate 44600888888bps delivered:37 app_limited busy:311ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:201 rtt:0.145/0.261 ato:40 mss:45568 pmtu:65535 rcvmss:3210 advmss:65483 cwnd:10 bytes_sent:26753 bytes_acked:26753 bytes_received:32652 segs_out:42 segs_in:76 data_segs_out:40 data_segs_in:36 send 25140965517bps lastsnd:2267 lastrcv:2280 lastack:2267 pacing_rate 50238621872bps delivery_rate 182272000000bps delivered:41 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:100352 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:211 rtt:10.532/16.883 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:56916 bytes_acked:56916 bytes_received:28085 segs_out:146 segs_in:197 data_segs_out:143 data_segs_in:143 send 248902393bps lastsnd:2253 lastrcv:2254 lastack:2212 pacing_rate 497763424bps delivery_rate 29127111104bps delivered:144 app_limited busy:1079ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:209 rtt:8.616/4.113 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:39949 bytes_acked:39950 bytes_received:35783 segs_out:119 segs_in:69 data_segs_out:76 data_segs_in:50 send 13444754bps lastsnd:2283 lastrcv:2270 lastack:2270 pacing_rate 26887944bps delivery_rate 1448000000bps delivered:77 app_limited busy:250ms rcv_rtt:5.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.013 snd_wnd:79872

- ipv6_1280_capture_1_receipt: {"bytes":"51593","sha256":"4ff5ac110f5dfec59da68f040e748301ed92f11d216d6bc7c89d9a07b64dc086"}
- ipv4_1280_input: {"input":{"underlay":"ipv4","mtu":1280,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv4_1280_transfer: {"after":{"upload_bytes":"537213","download_bytes":"485583","Present":true},"before":{"upload_bytes":"469405","download_bytes":"417340","Present":true},"bulk":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":2},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- ipv4_1280_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
334 packets captured
339 packets received by filter
0 packets dropped by kernel

- ipv4_1280_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443234.027042 fvnb0 Out IP (tos 0x0, ttl 64, id 31710, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.55090 > 192.0.3.10.9446: Flags [S], cksum 0x8e72 (incorrect -> 0x277f), seq 4196169297, win 64480, options [mss 1240,sackOK,TS val 205332081 ecr 0,nop,wscale 10], length 0
1793443234.027565 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.55090: Flags [S.], cksum 0x3607 (correct), seq 59143023, ack 4196169298, win 65160, options [mss 1460,sackOK,TS val 2857946260 ecr 205332081,nop,wscale 10], length 0

- ipv4_1280_path_0: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3090,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"aa:9b:9a:b8:49:95","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3093,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"a2:48:9d:a8:45:07","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":10,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.67.102.246","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State      Recv-Q Send-Q  Local Address:Port  Peer Address:PortProcess
ESTAB      0      0        172.28.30.41:57918 172.28.30.21:443
	 cubic wscale:10,10 rto:203 rtt:2.838/5.283 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3573 bytes_acked:3574 bytes_received:5738 segs_out:41 segs_in:36 data_segs_out:21 data_segs_in:30 send 34615927bps lastsnd:464 lastrcv:464 lastack:464 pacing_rate 69231848bps delivery_rate 1964800000bps delivered:22 app_limited busy:46ms rcv_space:12280 rcv_ssthresh:85803 minrtt:0.01 snd_wnd:79872
ESTAB      0      0        172.28.30.41:57934 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.626/3.105 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3749 bytes_acked:3750 bytes_received:6127 segs_out:30 segs_in:33 data_segs_out:17 data_segs_in:27 send 60418204bps lastsnd:5215 lastrcv:5215 lastack:5215 pacing_rate 120771416bps delivery_rate 1964800000bps delivered:18 app_limited busy:43ms rcv_space:12280 rcv_ssthresh:85793 minrtt:0.009 snd_wnd:79872
ESTAB      0      0        172.28.30.41:57944 172.28.30.21:443
	 cubic wscale:10,10 rto:205 rtt:4.49/8.891 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:2428 bytes_acked:2429 bytes_received:3077 segs_out:16 segs_in:13 data_segs_out:9 data_segs_in:9 send 21879733bps lastsnd:346 lastrcv:5215 lastack:346 pacing_rate 43757024bps delivery_rate 1637333328bps delivered:10 app_limited busy:41ms rcv_space:12280 rcv_ssthresh:78066 minrtt:0.012 snd_wnd:77824
FIN-WAIT-2 0      0      100.67.102.246:35316   192.0.3.10:9446

FIN-WAIT-2 0      0      100.67.102.246:37446   192.0.3.10:9446


- ipv4_1280_capture_0_receipt: {"bytes":"44388","sha256":"2f66fa0e576f68e19d74796c2c30f44327e72c108660de63cf00ede08588efdc"}
- ipv4_1280_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
509 packets captured
510 packets received by filter
0 packets dropped by kernel

- ipv4_1280_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793443234.027463 fvmb0 In  IP (tos 0x0, ttl 64, id 31710, offset 0, flags [DF], proto TCP (6), length 60)
    100.67.102.246.55090 > 192.0.3.10.9446: Flags [S], cksum 0x277f (correct), seq 4196169297, win 64480, options [mss 1240,sackOK,TS val 205332081 ecr 0,nop,wscale 10], length 0
1793443234.027491 fvup0 Out IP (tos 0x0, ttl 63, id 31710, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.55090 > 192.0.3.10.9446: Flags [S], cksum 0x2fa4 (correct), seq 4196169297, win 64480, options [mss 1240,sackOK,TS val 205332081 ecr 0,nop,wscale 10], length 0
1793443234.027519 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.55090: Flags [S.], cksum 0x864d (incorrect -> 0x3e2c), seq 59143023, ack 4196169298, win 65160, options [mss 1460,sackOK,TS val 2857946260 ecr 205332081,nop,wscale 10], length 0
1793443234.027525 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.67.102.246.55090: Flags [S.], cksum 0x8e72 (incorrect -> 0x3607), seq 59143023, ack 4196169298, win 65160, options [mss 1460,sackOK,TS val 2857946260 ecr 205332081,nop,wscale 10], length 0
1793443234.356737 eth2  In  IP6 (hlim 255, next-header ICMPv6 (58) payload length: 32) fe80::a048:9dff:fea8:4507 > fe80::88a6:94ff:fec9:250d: [icmp6 sum ok] ICMP6, neighbor solicitation, length 32, who has fe80::88a6:94ff:fec9:250d
	  source link-address option (1), length 8 (1): a2:48:9d:a8:45:07
1793443234.356758 eth2  Out IP6 (hlim 255, next-header ICMPv6 (58) payload length: 24) fe80::88a6:94ff:fec9:250d > fe80::a048:9dff:fea8:4507: [icmp6 sum ok] ICMP6, neighbor advertisement, length 24, tgt is fe80::88a6:94ff:fec9:250d, Flags [solicited]

- ipv4_1280_path_1: [{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":3088,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"6e:7d:42:7f:01:db","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":3089,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"6a:6f:d2:4f:34:40","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":3092,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:a6:94:c9:25:0d","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ce:91:d3:79:70:f1","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.67.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.67.5.252","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:44702    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.783/0.093 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:29023 bytes_acked:29024 bytes_received:58995 segs_out:204 segs_in:152 data_segs_out:148 data_segs_in:148 send 3347943806bps lastsnd:3709 lastrcv:3709 lastack:3709 pacing_rate 6691614544bps delivery_rate 32768000000bps delivered:149 app_limited busy:165ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.008 snd_wnd:65536
ESTAB 0      0       172.28.30.40:50274 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.336/0.332 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5863 bytes_acked:5864 bytes_received:12867 segs_out:135 segs_in:99 data_segs_out:56 data_segs_in:94 send 344761905bps lastsnd:3715 lastrcv:3715 lastack:3715 pacing_rate 687988120bps delivery_rate 2896000000bps delivered:57 app_limited busy:18ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50284 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.108/0.122 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:21672 bytes_acked:21673 bytes_received:25322 segs_out:184 segs_in:219 data_segs_out:101 data_segs_in:198 send 1072592593bps lastsnd:5386 lastrcv:5386 lastack:5386 pacing_rate 2132842344bps delivery_rate 4633600000bps delivered:102 app_limited busy:52ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.40:50296 172.28.30.21:443
	 cubic wscale:10,10 rto:208 rtt:7.511/12.487 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:7006 bytes_acked:7007 bytes_received:7481 segs_out:68 segs_in:68 data_segs_out:40 data_segs_in:44 send 15422713bps lastsnd:5386 lastrcv:517 lastack:517 pacing_rate 30844400bps delivery_rate 2896000000bps delivered:41 app_limited busy:294ms rcv_space:14480 rcv_ssthresh:85800 minrtt:0.004 snd_wnd:79872
ESTAB 0      0          127.0.0.1:36090    127.0.0.1:9444
	 cubic wscale:10,10 rto:212 rtt:11.013/1.782 ato:40 mss:50176 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:32869 bytes_acked:32870 bytes_received:27157 segs_out:78 segs_in:44 data_segs_out:37 data_segs_in:41 send 364485608bps lastsnd:3739 lastrcv:3726 lastack:3726 pacing_rate 728913296bps delivery_rate 44600888888bps delivered:38 app_limited busy:324ms rcv_rtt:14 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.009 snd_wnd:100352
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:36090
	 cubic wscale:10,10 rto:201 rtt:0.128/0.228 ato:40 mss:45568 pmtu:65535 rcvmss:3210 advmss:65483 cwnd:10 bytes_sent:27157 bytes_acked:27157 bytes_received:32869 segs_out:43 segs_in:78 data_segs_out:41 data_segs_in:37 send 28480000000bps lastsnd:3731 lastrcv:3744 lastack:3731 pacing_rate 56573268664bps delivery_rate 182272000000bps delivered:42 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:100352 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:44702
	 cubic wscale:10,10 rto:211 rtt:10.564/17.02 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:58995 bytes_acked:58995 bytes_received:29023 segs_out:151 segs_in:204 data_segs_out:148 data_segs_in:148 send 248148429bps lastsnd:3717 lastrcv:3717 lastack:3675 pacing_rate 496267496bps delivery_rate 29127111104bps delivered:149 app_limited busy:1121ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:43022 172.28.30.30:9443
	 cubic wscale:10,10 rto:210 rtt:9.081/4.013 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:40194 bytes_acked:40195 bytes_received:36187 segs_out:121 segs_in:70 data_segs_out:77 data_segs_in:51 send 12756304bps lastsnd:3747 lastrcv:3734 lastack:3734 pacing_rate 25512608bps delivery_rate 1448000000bps delivered:78 app_limited busy:263ms rcv_rtt:5.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.013 snd_wnd:79872

- ipv4_1280_capture_1_receipt: {"bytes":"67414","sha256":"ca9390d8bfa02febe5b2e9403b60d0f2263a0baacafe435b582eb27f23aaa71d"}
