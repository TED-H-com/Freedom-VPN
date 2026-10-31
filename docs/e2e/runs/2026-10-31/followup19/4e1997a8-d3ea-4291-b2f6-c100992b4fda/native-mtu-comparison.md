# native-mtu-comparison

F8-02 exact reconnect ordering with endpoint and complete header evidence

Status: pass

Run: 4e1997a8-d3ea-4291-b2f6-c100992b4fda

Source: 587e5267b458c180adab656ec7e5346043f143ac

Spec: AT09, AT11, F8-02

- matrix4_input: {"input":{"underlay":"ipv4","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix4_transfer: {"after":{"upload_bytes":"68585","download_bytes":"69822","Present":true},"before":{"upload_bytes":"1453","download_bytes":"1579","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:30:22.307084841Z","local_port":42202,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":125,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":67584,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- matrix4_capture_0_window: {"launch_requested_at":"2026-10-31T14:30:21.028193007Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:22.518316571Z","stop_requested_at":"2026-10-31T14:30:22.461788766Z"}
- matrix4_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
285 packets captured
289 packets received by filter
0 packets dropped by kernel

- matrix4_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457022.305267 fvnb0 Out IP (tos 0x0, ttl 64, id 43710, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [S], cksum 0x2a05 (incorrect -> 0x3ffa), seq 3665029260, win 64480, options [mss 1240,sackOK,TS val 3765990893 ecr 0,nop,wscale 10], length 0
1793457022.305621 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [S.], cksum 0xad14 (correct), seq 2604513529, ack 3665029261, win 65160, options [mss 1460,sackOK,TS val 4280761329 ecr 3765990893,nop,wscale 10], length 0
1793457022.305641 fvnb0 Out IP (tos 0x0, ttl 64, id 43711, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xda2c), ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 0
1793457022.305683 fvnb0 Out IP (tos 0x0, ttl 64, id 43712, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], cksum 0x2a29 (incorrect -> 0xeb50), seq 3665029261:3665029305, ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 44
1793457022.306099 fvnb0 In  IP (tos 0x0, ttl 63, id 58525, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0xd9fe (correct), ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306103 fvnb0 In  IP (tos 0x0, ttl 63, id 58526, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], cksum 0x80a8 (correct), seq 2604513530:2604513533, ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 3
1793457022.306107 fvnb0 Out IP (tos 0x0, ttl 64, id 43713, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xd9fc), ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306240 fvnb0 Out IP (tos 0x0, ttl 64, id 43714, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665029305:3665035445, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 6140
1793457022.306252 fvnb0 Out IP (tos 0x0, ttl 64, id 43719, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665035445:3665041585, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 6140
1793457022.306300 fvnb0 In  IP (tos 0x0, ttl 63, id 58527, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0xd061 (correct), ack 3665031761, win 66, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306311 fvnb0 Out IP (tos 0x0, ttl 64, id 43724, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665041585:3665046497, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306330 fvnb0 In  IP (tos 0x0, ttl 63, id 58528, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604513533:2604514539, ack 3665032989, win 65, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306341 fvnb0 Out IP (tos 0x0, ttl 64, id 43728, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665046497:3665048953, ack 2604514539, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306368 fvnb0 In  IP (tos 0x0, ttl 63, id 58529, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604514539:2604515545, ack 3665041585, win 57, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306369 fvnb0 In  IP (tos 0x0, ttl 63, id 58530, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604515545:2604516551, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306380 fvnb0 Out IP (tos 0x0, ttl 64, id 43730, offset 0, flags [DF], proto TCP (6), length 12332)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665048953:3665061233, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 12280
1793457022.306399 fvnb0 Out IP (tos 0x0, ttl 64, id 43740, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665061233:3665066145, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306413 fvnb0 In  IP (tos 0x0, ttl 63, id 58531, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604516551:2604517557, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306414 fvnb0 In  IP (tos 0x0, ttl 63, id 58532, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x90b7 (correct), ack 3665044041, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306414 fvnb0 In  IP (tos 0x0, ttl 63, id 58533, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604517557:2604518563, ack 3665046497, win 58, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306428 fvnb0 Out IP (tos 0x0, ttl 64, id 43744, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665066145:3665071057, ack 2604517557, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306437 fvnb0 In  IP (tos 0x0, ttl 63, id 58534, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604518563:2604519569, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306437 fvnb0 In  IP (tos 0x0, ttl 63, id 58535, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604519569:2604520575, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306438 fvnb0 In  IP (tos 0x0, ttl 63, id 58536, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604520575:2604521581, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306439 fvnb0 Out IP (tos 0x0, ttl 64, id 43748, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665071057:3665075969, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306444 fvnb0 Out IP (tos 0x0, ttl 64, id 43752, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665075969:3665080881, ack 2604519569, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306452 fvnb0 Out IP (tos 0x0, ttl 64, id 43756, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xf115), ack 2604521581, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306456 fvnb0 In  IP (tos 0x0, ttl 63, id 58537, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604521581:2604522587, ack 3665051409, win 55, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306456 fvnb0 In  IP (tos 0x0, ttl 63, id 58538, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604522587:2604523593, ack 3665052637, win 54, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306460 fvnb0 Out IP (tos 0x0, ttl 64, id 43757, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665080881:3665085793, ack 2604522587, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306466 fvnb0 Out IP (tos 0x0, ttl 64, id 43761, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665085793:3665088249, ack 2604523593, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306477 fvnb0 In  IP (tos 0x0, ttl 63, id 58539, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x4466 (correct), ack 3665057549, win 53, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306479 fvnb0 In  IP (tos 0x0, ttl 63, id 58540, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604523593:2604524599, ack 3665060005, win 51, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306484 fvnb0 Out IP (tos 0x0, ttl 64, id 43763, offset 0, flags [DF], proto TCP (6), length 5172)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665088249:3665093369, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 5120
1793457022.306498 fvnb0 In  IP (tos 0x0, ttl 63, id 58541, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604524599:2604525605, ack 3665062461, win 49, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306498 fvnb0 In  IP (tos 0x0, ttl 63, id 58542, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x1afd (correct), ack 3665066145, win 46, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306513 fvnb0 In  IP (tos 0x0, ttl 63, id 58543, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604525605:2604526611, ack 3665072285, win 41, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306521 fvnb0 In  IP (tos 0x0, ttl 63, id 58544, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604526611:2604527617, ack 3665074741, win 39, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306524 fvnb0 Out IP (tos 0x0, ttl 64, id 43768, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xaca8), ack 2604526611, win 60, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306538 fvnb0 In  IP (tos 0x0, ttl 63, id 58545, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604527617:2604528623, ack 3665077197, win 37, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306543 fvnb0 Out IP (tos 0x0, ttl 64, id 43769, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xa4cd), ack 2604528623, win 59, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306560 fvnb0 In  IP (tos 0x0, ttl 63, id 58546, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604528623:2604529629, ack 3665080881, win 34, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306560 fvnb0 In  IP (tos 0x0, ttl 63, id 58547, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0xbe94 (correct), ack 3665085793, win 30, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306575 fvnb0 In  IP (tos 0x0, ttl 63, id 58548, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604529629:2604530635, ack 3665087021, win 29, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306575 fvnb0 In  IP (tos 0x0, ttl 63, id 58549, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604530635:2604531641, ack 3665091933, win 25, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306578 fvnb0 In  IP (tos 0x0, ttl 63, id 58550, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x9926 (correct), ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306582 fvnb0 Out IP (tos 0x0, ttl 64, id 43770, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x9905), ack 2604531641, win 57, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306593 fvnb0 In  IP (tos 0x0, ttl 63, id 58551, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604531641:2604532647, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306600 fvnb0 In  IP (tos 0x0, ttl 63, id 58552, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604532647:2604533653, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306604 fvnb0 Out IP (tos 0x0, ttl 64, id 43771, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x912a), ack 2604533653, win 56, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306628 fvnb0 In  IP (tos 0x0, ttl 63, id 58553, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604533653:2604534659, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306634 fvnb0 In  IP (tos 0x0, ttl 63, id 58554, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604534659:2604535665, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306637 fvnb0 Out IP (tos 0x0, ttl 64, id 43772, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x894e), ack 2604535665, win 55, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306644 fvnb0 In  IP (tos 0x0, ttl 63, id 58555, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604535665:2604536671, ack 3665093369, win 27, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306656 fvnb0 In  IP (tos 0x0, ttl 63, id 58556, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604536671:2604537677, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306661 fvnb0 Out IP (tos 0x0, ttl 64, id 43773, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x8173), ack 2604537677, win 54, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306666 fvnb0 In  IP (tos 0x0, ttl 63, id 58557, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604537677:2604538683, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306676 fvnb0 In  IP (tos 0x0, ttl 63, id 58558, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604538683:2604539689, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306680 fvnb0 Out IP (tos 0x0, ttl 64, id 43774, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x7997), ack 2604539689, win 53, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306688 fvnb0 In  IP (tos 0x0, ttl 63, id 58559, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604539689:2604540695, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306698 fvnb0 In  IP (tos 0x0, ttl 63, id 58560, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604540695:2604541701, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306703 fvnb0 Out IP (tos 0x0, ttl 64, id 43775, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x71bc), ack 2604541701, win 52, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306711 fvnb0 In  IP (tos 0x0, ttl 63, id 58561, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604541701:2604542707, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306718 fvnb0 In  IP (tos 0x0, ttl 63, id 58562, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604542707:2604543713, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306732 fvnb0 Out IP (tos 0x0, ttl 64, id 43776, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x69e1), ack 2604543713, win 51, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306741 fvnb0 In  IP (tos 0x0, ttl 63, id 58563, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604543713:2604544719, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306741 fvnb0 In  IP (tos 0x0, ttl 63, id 58564, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604544719:2604545725, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306749 fvnb0 Out IP (tos 0x0, ttl 64, id 43777, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x6206), ack 2604545725, win 50, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306760 fvnb0 In  IP (tos 0x0, ttl 63, id 58565, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604545725:2604546731, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306779 fvnb0 In  IP (tos 0x0, ttl 63, id 58566, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604546731:2604547737, ack 3665093369, win 38, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306780 fvnb0 In  IP (tos 0x0, ttl 63, id 58567, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604547737:2604548743, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306791 fvnb0 Out IP (tos 0x0, ttl 64, id 43778, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x563e), ack 2604548743, win 48, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306793 fvnb0 In  IP (tos 0x0, ttl 63, id 58568, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604548743:2604549749, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306802 fvnb0 In  IP (tos 0x0, ttl 63, id 58569, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604549749:2604550755, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306812 fvnb0 In  IP (tos 0x0, ttl 63, id 58570, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604550755:2604551761, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306815 fvnb0 Out IP (tos 0x0, ttl 64, id 43779, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x4e63), ack 2604550755, win 47, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306822 fvnb0 In  IP (tos 0x0, ttl 63, id 58571, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604551761:2604552767, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306828 fvnb0 Out IP (tos 0x0, ttl 64, id 43780, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x4688), ack 2604552767, win 46, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306830 fvnb0 In  IP (tos 0x0, ttl 63, id 58572, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604552767:2604553773, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306843 fvnb0 In  IP (tos 0x0, ttl 63, id 58573, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604553773:2604554779, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306849 fvnb0 Out IP (tos 0x0, ttl 64, id 43781, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x3ead), ack 2604554779, win 45, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306858 fvnb0 In  IP (tos 0x0, ttl 63, id 58574, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604554779:2604555785, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306865 fvnb0 In  IP (tos 0x0, ttl 63, id 58575, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604555785:2604556791, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306872 fvnb0 Out IP (tos 0x0, ttl 64, id 43782, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x36d2), ack 2604556791, win 44, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306875 fvnb0 In  IP (tos 0x0, ttl 63, id 58576, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604556791:2604557797, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306895 fvnb0 In  IP (tos 0x0, ttl 63, id 58577, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604557797:2604558803, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306896 fvnb0 In  IP (tos 0x0, ttl 63, id 58578, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604558803:2604559809, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306903 fvnb0 In  IP (tos 0x0, ttl 63, id 58579, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604559809:2604560815, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306906 fvnb0 Out IP (tos 0x0, ttl 64, id 43783, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x2b0a), ack 2604559809, win 42, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306917 fvnb0 In  IP (tos 0x0, ttl 63, id 58580, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604560815:2604561821, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306924 fvnb0 In  IP (tos 0x0, ttl 63, id 58581, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604561821:2604562827, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306930 fvnb0 Out IP (tos 0x0, ttl 64, id 43784, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x1f42), ack 2604562827, win 40, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306944 fvnb0 In  IP (tos 0x0, ttl 63, id 58582, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604562827:2604563833, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306945 fvnb0 In  IP (tos 0x0, ttl 63, id 58583, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604563833:2604564839, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306954 fvnb0 Out IP (tos 0x0, ttl 64, id 43785, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x1767), ack 2604564839, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306956 fvnb0 In  IP (tos 0x0, ttl 63, id 58584, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604564839:2604565845, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306965 fvnb0 In  IP (tos 0x0, ttl 63, id 58585, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604565845:2604566851, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306974 fvnb0 In  IP (tos 0x0, ttl 63, id 58586, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604566851:2604567857, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306979 fvnb0 Out IP (tos 0x0, ttl 64, id 43786, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x0b9d), ack 2604567857, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307001 fvnb0 In  IP (tos 0x0, ttl 63, id 58587, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604567857:2604568863, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307002 fvnb0 In  IP (tos 0x0, ttl 63, id 58588, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604568863:2604569869, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307005 fvnb0 Out IP (tos 0x0, ttl 64, id 43787, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x03c1), ack 2604569869, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307006 fvnb0 In  IP (tos 0x0, ttl 63, id 58589, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604569869:2604570875, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307020 fvnb0 In  IP (tos 0x0, ttl 63, id 58590, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604570875:2604571881, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307029 fvnb0 In  IP (tos 0x0, ttl 63, id 58591, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604571881:2604572887, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307034 fvnb0 Out IP (tos 0x0, ttl 64, id 43788, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xf7f6), ack 2604572887, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307036 fvnb0 In  IP (tos 0x0, ttl 63, id 58592, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604572887:2604573893, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307050 fvnb0 In  IP (tos 0x0, ttl 63, id 58593, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604573893:2604574899, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307059 fvnb0 In  IP (tos 0x0, ttl 63, id 58594, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604574899:2604575905, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307061 fvnb0 Out IP (tos 0x0, ttl 64, id 43789, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xf01a), ack 2604574899, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307068 fvnb0 In  IP (tos 0x0, ttl 63, id 58595, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604575905:2604576911, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307072 fvnb0 Out IP (tos 0x0, ttl 64, id 43790, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xe83e), ack 2604576911, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307079 fvnb0 In  IP (tos 0x0, ttl 63, id 58596, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604576911:2604577917, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307221 fvnb0 Out IP (tos 0x0, ttl 64, id 43791, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [F.], cksum 0x29fd (incorrect -> 0xe44f), seq 3665093369, ack 2604577917, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307399 fvnb0 In  IP (tos 0x0, ttl 63, id 58597, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [F.], cksum 0xe433 (correct), seq 2604577917, ack 3665093370, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 0
1793457022.307404 fvnb0 Out IP (tos 0x0, ttl 64, id 43792, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xe44e), ack 2604577918, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0

- matrix4_path_0: eth0
2
1500
eth1
3
1500
fvnb0
5
1280
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4418,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:b5:ad:5a:5a:7a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4421,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:13:34:07:c1:06","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.78.2.126","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:49198 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.613/1.162 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5091 segs_out:17 segs_in:14 data_segs_out:9 data_segs_in:10 send 188972268bps lastsnd:37897 lastrcv:37896 lastack:7800 pacing_rate 377790456bps delivery_rate 2896000000bps delivered:10 app_limited busy:6ms rcv_space:14480 rcv_ssthresh:81050 minrtt:0.007 snd_wnd:71680
ESTAB 0      0       172.28.30.41:49816 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.23/0.25 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3573 bytes_acked:3574 bytes_received:5689 segs_out:42 segs_in:33 data_segs_out:21 data_segs_in:28 send 503652174bps lastsnd:443 lastrcv:443 lastack:443 pacing_rate 1005664672bps delivery_rate 2574222216bps delivered:22 app_limited busy:4ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:78848
ESTAB 0      0       172.28.30.41:49824 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.671/2.971 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3757 bytes_acked:3758 bytes_received:6193 segs_out:30 segs_in:36 data_segs_out:17 data_segs_in:30 send 69323758bps lastsnd:5381 lastrcv:5381 lastack:5381 pacing_rate 138574952bps delivery_rate 2574222216bps delivered:18 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49826 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.177/10.246 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2928 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 22375893bps lastsnd:5382 lastrcv:5382 lastack:5341 pacing_rate 44748544bps delivery_rate 2316800000bps delivered:9 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:79828 minrtt:0.009 snd_wnd:74752

- matrix4_capture_0_receipt: {"bytes":"38692","sha256":"381842f3e2abca43995b1019e05be4473009f2c223894b0516185e9f650c423f"}
- matrix4_capture_1_window: {"launch_requested_at":"2026-10-31T14:30:21.06888448Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:22.734678389Z","stop_requested_at":"2026-10-31T14:30:22.693004371Z"}
- matrix4_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
429 packets captured
431 packets received by filter
0 packets dropped by kernel

- matrix4_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457022.305521 fvmb0 In  IP (tos 0x0, ttl 64, id 43710, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [S], cksum 0x3ffa (correct), seq 3665029260, win 64480, options [mss 1240,sackOK,TS val 3765990893 ecr 0,nop,wscale 10], length 0
1793457022.305547 fvup0 Out IP (tos 0x0, ttl 63, id 43710, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [S], cksum 0xe3b1 (correct), seq 3665029260, win 64480, options [mss 1240,sackOK,TS val 3765990893 ecr 0,nop,wscale 10], length 0
1793457022.305574 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [S.], cksum 0x864d (incorrect -> 0x50cc), seq 2604513529, ack 3665029261, win 65160, options [mss 1460,sackOK,TS val 4280761329 ecr 3765990893,nop,wscale 10], length 0
1793457022.305579 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [S.], cksum 0x2a05 (incorrect -> 0xad14), seq 2604513529, ack 3665029261, win 65160, options [mss 1460,sackOK,TS val 4280761329 ecr 3765990893,nop,wscale 10], length 0
1793457022.305888 fvmb0 In  IP (tos 0x0, ttl 64, id 43711, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xda2c (correct), ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 0
1793457022.305890 fvmb0 In  IP (tos 0x0, ttl 64, id 43712, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], cksum 0xeb50 (correct), seq 3665029261:3665029305, ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 44
1793457022.305895 fvup0 Out IP (tos 0x0, ttl 63, id 43711, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x7de4 (correct), ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 0
1793457022.305897 fvup0 Out IP (tos 0x0, ttl 63, id 43712, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], cksum 0x8f08 (correct), seq 3665029261:3665029305, ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 44
1793457022.305915 fvup0 In  IP (tos 0x0, ttl 64, id 58525, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x7db6), ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.305917 fvmb0 Out IP (tos 0x0, ttl 63, id 58525, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x29fd (incorrect -> 0xd9fe), ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306066 fvup0 In  IP (tos 0x0, ttl 64, id 58526, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], cksum 0x8648 (incorrect -> 0x2460), seq 2604513530:2604513533, ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 3
1793457022.306076 fvmb0 Out IP (tos 0x0, ttl 63, id 58526, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], cksum 0x2a00 (incorrect -> 0x80a8), seq 2604513530:2604513533, ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 3
1793457022.306135 fvmb0 In  IP (tos 0x0, ttl 64, id 43713, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xd9fc (correct), ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306138 fvup0 Out IP (tos 0x0, ttl 63, id 43713, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x7db4 (correct), ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306273 fvmb0 In  IP (tos 0x0, ttl 64, id 43714, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665029305:3665031761, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306276 fvup0 Out IP (tos 0x0, ttl 63, id 43714, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665029305:3665031761, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306282 fvup0 In  IP (tos 0x0, ttl 64, id 58527, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x7419), ack 3665031761, win 66, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306284 fvmb0 Out IP (tos 0x0, ttl 63, id 58527, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x29fd (incorrect -> 0xd061), ack 3665031761, win 66, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306287 fvmb0 In  IP (tos 0x0, ttl 64, id 43716, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665031761:3665032989, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306290 fvup0 Out IP (tos 0x0, ttl 63, id 43716, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665031761:3665032989, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306295 fvmb0 In  IP (tos 0x0, ttl 64, id 43717, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665032989:3665035445, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306295 fvmb0 In  IP (tos 0x0, ttl 64, id 43719, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665035445:3665037901, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306298 fvup0 Out IP (tos 0x0, ttl 63, id 43717, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665032989:3665035445, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306299 fvup0 Out IP (tos 0x0, ttl 63, id 43719, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665035445:3665037901, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306304 fvmb0 In  IP (tos 0x0, ttl 64, id 43721, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665037901:3665041585, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306305 fvup0 In  IP (tos 0x0, ttl 64, id 58528, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604513533:2604514539, ack 3665032989, win 65, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306308 fvup0 Out IP (tos 0x0, ttl 63, id 43721, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665037901:3665041585, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306310 fvmb0 Out IP (tos 0x0, ttl 63, id 58528, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604513533:2604514539, ack 3665032989, win 65, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306335 fvup0 In  IP (tos 0x0, ttl 64, id 58529, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604514539:2604515545, ack 3665041585, win 57, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306338 fvmb0 Out IP (tos 0x0, ttl 63, id 58529, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604514539:2604515545, ack 3665041585, win 57, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306349 fvup0 In  IP (tos 0x0, ttl 64, id 58530, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604515545:2604516551, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306351 fvmb0 Out IP (tos 0x0, ttl 63, id 58530, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604515545:2604516551, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306354 fvmb0 In  IP (tos 0x0, ttl 64, id 43724, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665041585:3665044041, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306357 fvup0 Out IP (tos 0x0, ttl 63, id 43724, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665041585:3665044041, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306358 fvup0 In  IP (tos 0x0, ttl 64, id 58531, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604516551:2604517557, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306361 fvmb0 Out IP (tos 0x0, ttl 63, id 58531, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604516551:2604517557, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306365 fvmb0 In  IP (tos 0x0, ttl 64, id 43726, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665044041:3665046497, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306368 fvup0 Out IP (tos 0x0, ttl 63, id 43726, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665044041:3665046497, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306368 fvup0 In  IP (tos 0x0, ttl 64, id 58532, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x346f), ack 3665044041, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306371 fvmb0 Out IP (tos 0x0, ttl 63, id 58532, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x29fd (incorrect -> 0x90b7), ack 3665044041, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306375 fvmb0 In  IP (tos 0x0, ttl 64, id 43728, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665046497:3665048953, ack 2604514539, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306377 fvup0 Out IP (tos 0x0, ttl 63, id 43728, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665046497:3665048953, ack 2604514539, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306380 fvup0 In  IP (tos 0x0, ttl 64, id 58533, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604517557:2604518563, ack 3665046497, win 58, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306382 fvmb0 Out IP (tos 0x0, ttl 63, id 58533, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604517557:2604518563, ack 3665046497, win 58, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306392 fvup0 In  IP (tos 0x0, ttl 64, id 58534, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604518563:2604519569, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306396 fvmb0 Out IP (tos 0x0, ttl 63, id 58534, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604518563:2604519569, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306406 fvup0 In  IP (tos 0x0, ttl 64, id 58535, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604519569:2604520575, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306409 fvmb0 Out IP (tos 0x0, ttl 63, id 58535, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604519569:2604520575, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306417 fvup0 In  IP (tos 0x0, ttl 64, id 58536, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604520575:2604521581, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306419 fvmb0 Out IP (tos 0x0, ttl 63, id 58536, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604520575:2604521581, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306421 fvmb0 In  IP (tos 0x0, ttl 64, id 43730, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665048953:3665051409, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306423 fvup0 Out IP (tos 0x0, ttl 63, id 43730, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665048953:3665051409, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306428 fvmb0 In  IP (tos 0x0, ttl 64, id 43732, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665051409:3665052637, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306429 fvup0 In  IP (tos 0x0, ttl 64, id 58537, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604521581:2604522587, ack 3665051409, win 55, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306431 fvup0 Out IP (tos 0x0, ttl 63, id 43732, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665051409:3665052637, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306432 fvmb0 Out IP (tos 0x0, ttl 63, id 58537, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604521581:2604522587, ack 3665051409, win 55, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306440 fvup0 In  IP (tos 0x0, ttl 64, id 58538, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604522587:2604523593, ack 3665052637, win 54, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306441 fvmb0 In  IP (tos 0x0, ttl 64, id 43733, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665052637:3665057549, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306442 fvmb0 Out IP (tos 0x0, ttl 63, id 58538, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604522587:2604523593, ack 3665052637, win 54, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306443 fvup0 Out IP (tos 0x0, ttl 63, id 43733, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665052637:3665057549, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306447 fvmb0 In  IP (tos 0x0, ttl 64, id 43737, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665057549:3665058777, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306450 fvup0 In  IP (tos 0x0, ttl 64, id 58539, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0xe81d), ack 3665057549, win 53, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306450 fvup0 Out IP (tos 0x0, ttl 63, id 43737, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665057549:3665058777, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306452 fvmb0 Out IP (tos 0x0, ttl 63, id 58539, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x29fd (incorrect -> 0x4466), ack 3665057549, win 53, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306452 fvmb0 In  IP (tos 0x0, ttl 64, id 43738, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665058777:3665060005, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306455 fvup0 Out IP (tos 0x0, ttl 63, id 43738, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665058777:3665060005, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306458 fvmb0 In  IP (tos 0x0, ttl 64, id 43739, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665060005:3665061233, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306458 fvmb0 In  IP (tos 0x0, ttl 64, id 43740, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665061233:3665062461, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306462 fvup0 Out IP (tos 0x0, ttl 63, id 43739, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665060005:3665061233, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306463 fvup0 In  IP (tos 0x0, ttl 64, id 58540, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604523593:2604524599, ack 3665060005, win 51, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306463 fvup0 Out IP (tos 0x0, ttl 63, id 43740, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665061233:3665062461, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306466 fvmb0 Out IP (tos 0x0, ttl 63, id 58540, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604523593:2604524599, ack 3665060005, win 51, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306467 fvmb0 In  IP (tos 0x0, ttl 64, id 43741, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665062461:3665066145, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306470 fvup0 Out IP (tos 0x0, ttl 63, id 43741, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665062461:3665066145, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306474 fvup0 In  IP (tos 0x0, ttl 64, id 58541, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604524599:2604525605, ack 3665062461, win 49, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306476 fvmb0 Out IP (tos 0x0, ttl 63, id 58541, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604524599:2604525605, ack 3665062461, win 49, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306481 fvmb0 In  IP (tos 0x0, ttl 64, id 43744, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665066145:3665071057, ack 2604517557, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306483 fvup0 Out IP (tos 0x0, ttl 63, id 43744, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665066145:3665071057, ack 2604517557, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306484 fvup0 In  IP (tos 0x0, ttl 64, id 58542, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0xbeb4), ack 3665066145, win 46, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306487 fvmb0 In  IP (tos 0x0, ttl 64, id 43748, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665071057:3665072285, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306487 fvmb0 Out IP (tos 0x0, ttl 63, id 58542, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x29fd (incorrect -> 0x1afd), ack 3665066145, win 46, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306490 fvup0 Out IP (tos 0x0, ttl 63, id 43748, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665071057:3665072285, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306494 fvmb0 In  IP (tos 0x0, ttl 64, id 43749, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665072285:3665074741, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306496 fvup0 In  IP (tos 0x0, ttl 64, id 58543, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604525605:2604526611, ack 3665072285, win 41, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306497 fvup0 Out IP (tos 0x0, ttl 63, id 43749, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665072285:3665074741, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306499 fvmb0 Out IP (tos 0x0, ttl 63, id 58543, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604525605:2604526611, ack 3665072285, win 41, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306504 fvmb0 In  IP (tos 0x0, ttl 64, id 43751, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665074741:3665075969, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306505 fvmb0 In  IP (tos 0x0, ttl 64, id 43752, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665075969:3665077197, ack 2604519569, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306507 fvup0 In  IP (tos 0x0, ttl 64, id 58544, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604526611:2604527617, ack 3665074741, win 39, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306508 fvup0 Out IP (tos 0x0, ttl 63, id 43751, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665074741:3665075969, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306509 fvmb0 Out IP (tos 0x0, ttl 63, id 58544, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604526611:2604527617, ack 3665074741, win 39, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306510 fvup0 Out IP (tos 0x0, ttl 63, id 43752, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665075969:3665077197, ack 2604519569, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306515 fvmb0 In  IP (tos 0x0, ttl 64, id 43753, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665077197:3665080881, ack 2604519569, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306515 fvmb0 In  IP (tos 0x0, ttl 64, id 43756, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xf115 (correct), ack 2604521581, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306518 fvup0 Out IP (tos 0x0, ttl 63, id 43753, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665077197:3665080881, ack 2604519569, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306520 fvup0 In  IP (tos 0x0, ttl 64, id 58545, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604527617:2604528623, ack 3665077197, win 37, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306520 fvup0 Out IP (tos 0x0, ttl 63, id 43756, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x94cd (correct), ack 2604521581, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306523 fvmb0 Out IP (tos 0x0, ttl 63, id 58545, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604527617:2604528623, ack 3665077197, win 37, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306530 fvmb0 In  IP (tos 0x0, ttl 64, id 43757, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665080881:3665085793, ack 2604522587, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306532 fvup0 In  IP (tos 0x0, ttl 64, id 58546, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604528623:2604529629, ack 3665080881, win 34, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306532 fvup0 Out IP (tos 0x0, ttl 63, id 43757, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665080881:3665085793, ack 2604522587, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306534 fvmb0 Out IP (tos 0x0, ttl 63, id 58546, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604528623:2604529629, ack 3665080881, win 34, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306535 fvmb0 In  IP (tos 0x0, ttl 64, id 43761, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665085793:3665087021, ack 2604523593, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306537 fvup0 Out IP (tos 0x0, ttl 63, id 43761, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665085793:3665087021, ack 2604523593, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306540 fvmb0 In  IP (tos 0x0, ttl 64, id 43762, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665087021:3665088249, ack 2604523593, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306540 fvup0 In  IP (tos 0x0, ttl 64, id 58547, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x624c), ack 3665085793, win 30, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306540 fvmb0 In  IP (tos 0x0, ttl 64, id 43763, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665088249:3665089477, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306542 fvmb0 Out IP (tos 0x0, ttl 63, id 58547, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x29fd (incorrect -> 0xbe94), ack 3665085793, win 30, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306543 fvup0 Out IP (tos 0x0, ttl 63, id 43762, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665087021:3665088249, ack 2604523593, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306544 fvup0 Out IP (tos 0x0, ttl 63, id 43763, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665088249:3665089477, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306549 fvup0 In  IP (tos 0x0, ttl 64, id 58548, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604529629:2604530635, ack 3665087021, win 29, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306549 fvmb0 In  IP (tos 0x0, ttl 64, id 43764, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], seq 3665089477:3665091933, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306551 fvmb0 Out IP (tos 0x0, ttl 63, id 58548, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604529629:2604530635, ack 3665087021, win 29, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306552 fvup0 Out IP (tos 0x0, ttl 63, id 43764, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665089477:3665091933, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306556 fvmb0 In  IP (tos 0x0, ttl 64, id 43766, offset 0, flags [DF], proto TCP (6), length 1488)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [P.], seq 3665091933:3665093369, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1436
1793457022.306559 fvup0 In  IP (tos 0x0, ttl 64, id 58549, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604530635:2604531641, ack 3665091933, win 25, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306559 fvup0 Out IP (tos 0x0, ttl 63, id 43766, offset 0, flags [DF], proto TCP (6), length 1488)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665091933:3665093369, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1436
1793457022.306561 fvmb0 Out IP (tos 0x0, ttl 63, id 58549, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604530635:2604531641, ack 3665091933, win 25, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306562 fvmb0 In  IP (tos 0x0, ttl 64, id 43768, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xaca8 (correct), ack 2604526611, win 60, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306562 fvmb0 In  IP (tos 0x0, ttl 64, id 43769, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xa4cd (correct), ack 2604528623, win 59, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306564 fvup0 Out IP (tos 0x0, ttl 63, id 43768, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x5060 (correct), ack 2604526611, win 60, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306566 fvup0 Out IP (tos 0x0, ttl 63, id 43769, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x4885 (correct), ack 2604528623, win 59, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306568 fvup0 In  IP (tos 0x0, ttl 64, id 58550, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x3cde), ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306570 fvmb0 Out IP (tos 0x0, ttl 63, id 58550, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [.], cksum 0x29fd (incorrect -> 0x9926), ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306577 fvup0 In  IP (tos 0x0, ttl 64, id 58551, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604531641:2604532647, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306579 fvmb0 Out IP (tos 0x0, ttl 63, id 58551, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604531641:2604532647, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306586 fvup0 In  IP (tos 0x0, ttl 64, id 58552, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604532647:2604533653, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306588 fvmb0 Out IP (tos 0x0, ttl 63, id 58552, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604532647:2604533653, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306596 fvmb0 In  IP (tos 0x0, ttl 64, id 43770, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x9905 (correct), ack 2604531641, win 57, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306598 fvup0 In  IP (tos 0x0, ttl 64, id 58553, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604533653:2604534659, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306598 fvup0 Out IP (tos 0x0, ttl 63, id 43770, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x3cbd (correct), ack 2604531641, win 57, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306600 fvmb0 Out IP (tos 0x0, ttl 63, id 58553, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604533653:2604534659, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306614 fvup0 In  IP (tos 0x0, ttl 64, id 58554, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604534659:2604535665, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306616 fvmb0 Out IP (tos 0x0, ttl 63, id 58554, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604534659:2604535665, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306619 fvmb0 In  IP (tos 0x0, ttl 64, id 43771, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x912a (correct), ack 2604533653, win 56, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306621 fvup0 Out IP (tos 0x0, ttl 63, id 43771, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x34e2 (correct), ack 2604533653, win 56, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306625 fvup0 In  IP (tos 0x0, ttl 64, id 58555, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604535665:2604536671, ack 3665093369, win 27, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306627 fvmb0 Out IP (tos 0x0, ttl 63, id 58555, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604535665:2604536671, ack 3665093369, win 27, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306639 fvup0 In  IP (tos 0x0, ttl 64, id 58556, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604536671:2604537677, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306641 fvmb0 Out IP (tos 0x0, ttl 63, id 58556, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604536671:2604537677, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306651 fvup0 In  IP (tos 0x0, ttl 64, id 58557, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604537677:2604538683, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306653 fvmb0 In  IP (tos 0x0, ttl 64, id 43772, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x894e (correct), ack 2604535665, win 55, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306653 fvmb0 Out IP (tos 0x0, ttl 63, id 58557, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604537677:2604538683, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306656 fvup0 Out IP (tos 0x0, ttl 63, id 43772, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x2d06 (correct), ack 2604535665, win 55, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306660 fvup0 In  IP (tos 0x0, ttl 64, id 58558, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604538683:2604539689, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306662 fvmb0 Out IP (tos 0x0, ttl 63, id 58558, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604538683:2604539689, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306671 fvup0 In  IP (tos 0x0, ttl 64, id 58559, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604539689:2604540695, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306673 fvmb0 Out IP (tos 0x0, ttl 63, id 58559, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604539689:2604540695, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306675 fvmb0 In  IP (tos 0x0, ttl 64, id 43773, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x8173 (correct), ack 2604537677, win 54, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306677 fvup0 Out IP (tos 0x0, ttl 63, id 43773, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x252b (correct), ack 2604537677, win 54, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306682 fvup0 In  IP (tos 0x0, ttl 64, id 58560, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604540695:2604541701, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306684 fvmb0 Out IP (tos 0x0, ttl 63, id 58560, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604540695:2604541701, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306693 fvup0 In  IP (tos 0x0, ttl 64, id 58561, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604541701:2604542707, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306695 fvmb0 In  IP (tos 0x0, ttl 64, id 43774, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x7997 (correct), ack 2604539689, win 53, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306696 fvmb0 Out IP (tos 0x0, ttl 63, id 58561, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604541701:2604542707, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306698 fvup0 Out IP (tos 0x0, ttl 63, id 43774, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x1d4f (correct), ack 2604539689, win 53, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306703 fvup0 In  IP (tos 0x0, ttl 64, id 58562, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604542707:2604543713, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306705 fvmb0 Out IP (tos 0x0, ttl 63, id 58562, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604542707:2604543713, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306715 fvup0 In  IP (tos 0x0, ttl 64, id 58563, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604543713:2604544719, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306717 fvmb0 Out IP (tos 0x0, ttl 63, id 58563, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604543713:2604544719, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306718 fvmb0 In  IP (tos 0x0, ttl 64, id 43775, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x71bc (correct), ack 2604541701, win 52, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306725 fvup0 In  IP (tos 0x0, ttl 64, id 58564, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604544719:2604545725, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306727 fvmb0 Out IP (tos 0x0, ttl 63, id 58564, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604544719:2604545725, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306730 fvup0 Out IP (tos 0x0, ttl 63, id 43775, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x1574 (correct), ack 2604541701, win 52, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306741 fvup0 In  IP (tos 0x0, ttl 64, id 58565, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604545725:2604546731, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306743 fvmb0 Out IP (tos 0x0, ttl 63, id 58565, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604545725:2604546731, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306751 fvup0 In  IP (tos 0x0, ttl 64, id 58566, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604546731:2604547737, ack 3665093369, win 38, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306754 fvmb0 Out IP (tos 0x0, ttl 63, id 58566, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604546731:2604547737, ack 3665093369, win 38, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306756 fvmb0 In  IP (tos 0x0, ttl 64, id 43776, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x69e1 (correct), ack 2604543713, win 51, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306763 fvup0 In  IP (tos 0x0, ttl 64, id 58567, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604547737:2604548743, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306765 fvup0 Out IP (tos 0x0, ttl 63, id 43776, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x0d99 (correct), ack 2604543713, win 51, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306766 fvmb0 Out IP (tos 0x0, ttl 63, id 58567, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604547737:2604548743, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306773 fvmb0 In  IP (tos 0x0, ttl 64, id 43777, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x6206 (correct), ack 2604545725, win 50, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306774 fvup0 In  IP (tos 0x0, ttl 64, id 58568, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604548743:2604549749, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306777 fvmb0 Out IP (tos 0x0, ttl 63, id 58568, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604548743:2604549749, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306778 fvup0 Out IP (tos 0x0, ttl 63, id 43777, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x05be (correct), ack 2604545725, win 50, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306786 fvup0 In  IP (tos 0x0, ttl 64, id 58569, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604549749:2604550755, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306789 fvmb0 Out IP (tos 0x0, ttl 63, id 58569, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604549749:2604550755, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306795 fvup0 In  IP (tos 0x0, ttl 64, id 58570, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604550755:2604551761, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306798 fvmb0 Out IP (tos 0x0, ttl 63, id 58570, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604550755:2604551761, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306807 fvup0 In  IP (tos 0x0, ttl 64, id 58571, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604551761:2604552767, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306809 fvmb0 Out IP (tos 0x0, ttl 63, id 58571, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604551761:2604552767, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306812 fvmb0 In  IP (tos 0x0, ttl 64, id 43778, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x563e (correct), ack 2604548743, win 48, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306815 fvup0 Out IP (tos 0x0, ttl 63, id 43778, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xf9f5 (correct), ack 2604548743, win 48, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306816 fvup0 In  IP (tos 0x0, ttl 64, id 58572, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604552767:2604553773, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306818 fvmb0 Out IP (tos 0x0, ttl 63, id 58572, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604552767:2604553773, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306828 fvup0 In  IP (tos 0x0, ttl 64, id 58573, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604553773:2604554779, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306829 fvmb0 In  IP (tos 0x0, ttl 64, id 43779, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x4e63 (correct), ack 2604550755, win 47, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306831 fvmb0 Out IP (tos 0x0, ttl 63, id 58573, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604553773:2604554779, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306831 fvup0 Out IP (tos 0x0, ttl 63, id 43779, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xf21a (correct), ack 2604550755, win 47, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306839 fvup0 In  IP (tos 0x0, ttl 64, id 58574, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604554779:2604555785, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306841 fvmb0 In  IP (tos 0x0, ttl 64, id 43780, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x4688 (correct), ack 2604552767, win 46, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306842 fvmb0 Out IP (tos 0x0, ttl 63, id 58574, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604554779:2604555785, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306844 fvup0 Out IP (tos 0x0, ttl 63, id 43780, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xea3f (correct), ack 2604552767, win 46, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306850 fvup0 In  IP (tos 0x0, ttl 64, id 58575, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604555785:2604556791, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306853 fvmb0 Out IP (tos 0x0, ttl 63, id 58575, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604555785:2604556791, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306860 fvup0 In  IP (tos 0x0, ttl 64, id 58576, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604556791:2604557797, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306861 fvmb0 In  IP (tos 0x0, ttl 64, id 43781, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x3ead (correct), ack 2604554779, win 45, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306862 fvmb0 Out IP (tos 0x0, ttl 63, id 58576, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604556791:2604557797, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306863 fvup0 Out IP (tos 0x0, ttl 63, id 43781, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xe264 (correct), ack 2604554779, win 45, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306870 fvup0 In  IP (tos 0x0, ttl 64, id 58577, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604557797:2604558803, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306872 fvmb0 Out IP (tos 0x0, ttl 63, id 58577, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604557797:2604558803, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306880 fvup0 In  IP (tos 0x0, ttl 64, id 58578, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604558803:2604559809, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306883 fvmb0 Out IP (tos 0x0, ttl 63, id 58578, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604558803:2604559809, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306889 fvup0 In  IP (tos 0x0, ttl 64, id 58579, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604559809:2604560815, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306892 fvmb0 Out IP (tos 0x0, ttl 63, id 58579, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604559809:2604560815, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306901 fvup0 In  IP (tos 0x0, ttl 64, id 58580, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604560815:2604561821, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306903 fvmb0 Out IP (tos 0x0, ttl 63, id 58580, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604560815:2604561821, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306910 fvup0 In  IP (tos 0x0, ttl 64, id 58581, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604561821:2604562827, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306912 fvmb0 Out IP (tos 0x0, ttl 63, id 58581, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604561821:2604562827, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306921 fvup0 In  IP (tos 0x0, ttl 64, id 58582, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604562827:2604563833, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306923 fvmb0 Out IP (tos 0x0, ttl 63, id 58582, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604562827:2604563833, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306929 fvup0 In  IP (tos 0x0, ttl 64, id 58583, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604563833:2604564839, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306931 fvmb0 Out IP (tos 0x0, ttl 63, id 58583, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604563833:2604564839, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306934 fvmb0 In  IP (tos 0x0, ttl 64, id 43782, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x36d2 (correct), ack 2604556791, win 44, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306934 fvmb0 In  IP (tos 0x0, ttl 64, id 43783, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x2b0a (correct), ack 2604559809, win 42, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306937 fvup0 Out IP (tos 0x0, ttl 63, id 43782, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xda89 (correct), ack 2604556791, win 44, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306939 fvup0 Out IP (tos 0x0, ttl 63, id 43783, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xcec1 (correct), ack 2604559809, win 42, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306940 fvup0 In  IP (tos 0x0, ttl 64, id 58584, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604564839:2604565845, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306942 fvmb0 Out IP (tos 0x0, ttl 63, id 58584, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604564839:2604565845, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306950 fvup0 In  IP (tos 0x0, ttl 64, id 58585, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604565845:2604566851, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306952 fvmb0 Out IP (tos 0x0, ttl 63, id 58585, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604565845:2604566851, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306954 fvmb0 In  IP (tos 0x0, ttl 64, id 43784, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x1f42 (correct), ack 2604562827, win 40, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306958 fvup0 Out IP (tos 0x0, ttl 63, id 43784, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xc2f9 (correct), ack 2604562827, win 40, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306961 fvup0 In  IP (tos 0x0, ttl 64, id 58586, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604566851:2604567857, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306963 fvmb0 Out IP (tos 0x0, ttl 63, id 58586, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604566851:2604567857, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306965 fvmb0 In  IP (tos 0x0, ttl 64, id 43785, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x1767 (correct), ack 2604564839, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306967 fvup0 Out IP (tos 0x0, ttl 63, id 43785, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xbb1e (correct), ack 2604564839, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306971 fvup0 In  IP (tos 0x0, ttl 64, id 58587, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604567857:2604568863, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306973 fvmb0 Out IP (tos 0x0, ttl 63, id 58587, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604567857:2604568863, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306982 fvup0 In  IP (tos 0x0, ttl 64, id 58588, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604568863:2604569869, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306985 fvmb0 Out IP (tos 0x0, ttl 63, id 58588, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604568863:2604569869, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306991 fvmb0 In  IP (tos 0x0, ttl 64, id 43786, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x0b9d (correct), ack 2604567857, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306991 fvup0 In  IP (tos 0x0, ttl 64, id 58589, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604569869:2604570875, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306994 fvmb0 Out IP (tos 0x0, ttl 63, id 58589, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604569869:2604570875, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306995 fvup0 Out IP (tos 0x0, ttl 63, id 43786, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xaf54 (correct), ack 2604567857, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307003 fvup0 In  IP (tos 0x0, ttl 64, id 58590, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604570875:2604571881, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307005 fvmb0 Out IP (tos 0x0, ttl 63, id 58590, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604570875:2604571881, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307012 fvup0 In  IP (tos 0x0, ttl 64, id 58591, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604571881:2604572887, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307014 fvmb0 Out IP (tos 0x0, ttl 63, id 58591, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604571881:2604572887, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307017 fvmb0 In  IP (tos 0x0, ttl 64, id 43787, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0x03c1 (correct), ack 2604569869, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307020 fvup0 Out IP (tos 0x0, ttl 63, id 43787, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xa778 (correct), ack 2604569869, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307022 fvup0 In  IP (tos 0x0, ttl 64, id 58592, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604572887:2604573893, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307024 fvmb0 Out IP (tos 0x0, ttl 63, id 58592, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604572887:2604573893, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307032 fvup0 In  IP (tos 0x0, ttl 64, id 58593, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604573893:2604574899, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307034 fvmb0 Out IP (tos 0x0, ttl 63, id 58593, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604573893:2604574899, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307043 fvup0 In  IP (tos 0x0, ttl 64, id 58594, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604574899:2604575905, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307046 fvmb0 Out IP (tos 0x0, ttl 63, id 58594, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604574899:2604575905, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307048 fvmb0 In  IP (tos 0x0, ttl 64, id 43788, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xf7f6 (correct), ack 2604572887, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307051 fvup0 Out IP (tos 0x0, ttl 63, id 43788, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x9bae (correct), ack 2604572887, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307053 fvup0 In  IP (tos 0x0, ttl 64, id 58595, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604575905:2604576911, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307055 fvmb0 Out IP (tos 0x0, ttl 63, id 58595, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604575905:2604576911, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307065 fvup0 In  IP (tos 0x0, ttl 64, id 58596, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604576911:2604577917, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307068 fvmb0 Out IP (tos 0x0, ttl 63, id 58596, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [P.], seq 2604576911:2604577917, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307087 fvmb0 In  IP (tos 0x0, ttl 64, id 43789, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xf01a (correct), ack 2604574899, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307090 fvup0 Out IP (tos 0x0, ttl 63, id 43789, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x93d2 (correct), ack 2604574899, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307093 fvmb0 In  IP (tos 0x0, ttl 64, id 43790, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xe83e (correct), ack 2604576911, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307095 fvup0 Out IP (tos 0x0, ttl 63, id 43790, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x8bf6 (correct), ack 2604576911, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307237 fvmb0 In  IP (tos 0x0, ttl 64, id 43791, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [F.], cksum 0xe44f (correct), seq 3665093369, ack 2604577917, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307246 fvup0 Out IP (tos 0x0, ttl 63, id 43791, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [F.], cksum 0x8807 (correct), seq 3665093369, ack 2604577917, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307384 fvup0 In  IP (tos 0x0, ttl 64, id 58597, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [F.], cksum 0x8645 (incorrect -> 0x87eb), seq 2604577917, ack 3665093370, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 0
1793457022.307388 fvmb0 Out IP (tos 0x0, ttl 63, id 58597, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.42202: Flags [F.], cksum 0x29fd (incorrect -> 0xe433), seq 2604577917, ack 3665093370, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 0
1793457022.307419 fvmb0 In  IP (tos 0x0, ttl 64, id 43792, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.42202 > 192.0.3.10.9446: Flags [.], cksum 0xe44e (correct), ack 2604577918, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307422 fvup0 Out IP (tos 0x0, ttl 63, id 43792, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x8806 (correct), ack 2604577918, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0

- matrix4_path_1: eth0
2
1500
eth2
4
1500
fvfixture
5
1500
fvmb0
6
1280
fvup0
3
1500
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4416,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:25:c1:29:8d:d8","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4417,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"c6:ee:52:4f:5a:4e","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4420,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:8a:b4:de:17:2f","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ae:1b:a1:20:96:73","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.78.12.82","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51782
	 cubic wscale:10,10 rto:201 rtt:0.444/0.867 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:22989 bytes_acked:22989 bytes_received:32690 segs_out:39 segs_in:62 data_segs_out:31 data_segs_in:31 send 8210450450bps lastsnd:293 lastrcv:298 lastack:293 pacing_rate 16397818384bps delivery_rate 121514666664bps delivered:32 app_limited busy:94ms rcv_rtt:1 rcv_space:65483 rcv_ssthresh:105032 minrtt:0.003 snd_wnd:91136
ESTAB 0      0       172.28.30.40:45976 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.602/0.528 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4718 bytes_acked:4719 bytes_received:9618 segs_out:85 segs_in:66 data_segs_out:38 data_segs_in:60 send 192425249bps lastsnd:5218 lastrcv:5218 lastack:5218 pacing_rate 384770600bps delivery_rate 3309714280bps delivered:39 app_limited busy:53ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45986 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.298/0.452 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:7867 bytes_acked:7868 bytes_received:9715 segs_out:68 segs_in:73 data_segs_out:38 data_segs_in:65 send 388724832bps lastsnd:5549 lastrcv:5549 lastack:5549 pacing_rate 775822512bps delivery_rate 1654857136bps delivered:39 app_limited busy:43ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45992 172.28.30.21:443
	 cubic wscale:10,10 rto:207 rtt:6.182/10.851 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2358 bytes_acked:2359 bytes_received:3466 segs_out:24 segs_in:23 data_segs_out:11 data_segs_in:14 send 18738272bps lastsnd:5549 lastrcv:5548 lastack:5548 pacing_rate 37472752bps delivery_rate 2896000000bps delivered:12 app_limited busy:84ms rcv_space:14480 rcv_ssthresh:82724 minrtt:0.008 snd_wnd:74752
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51796
	 cubic wscale:10,10 rto:203 rtt:2.308/4.59 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:9387 bytes_acked:9387 bytes_received:5962 segs_out:14 segs_in:20 data_segs_out:8 data_segs_in:9 send 1508492201bps lastsnd:5176 lastrcv:5181 lastack:5176 pacing_rate 3016004328bps delivery_rate 34816000000bps delivered:9 app_limited busy:41ms rcv_rtt:1245.38 rcv_space:65483 rcv_ssthresh:78471 minrtt:0.008 snd_wnd:83968
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:55868
	 cubic wscale:10,10 rto:206 rtt:5.079/10.134 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2779 bytes_acked:2779 bytes_received:3998 segs_out:4 segs_in:8 data_segs_out:2 data_segs_in:4 send 685489270bps lastsnd:5372 lastrcv:5384 lastack:5331 pacing_rate 1370877320bps delivery_rate 29013333328bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78342 minrtt:0.012 snd_wnd:87040
ESTAB 0      0          127.0.0.1:36206    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.747/0.045 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:20581 bytes_acked:20582 bytes_received:40284 segs_out:141 segs_in:107 data_segs_out:103 data_segs_in:103 send 3509290495bps lastsnd:285 lastrcv:285 lastack:285 pacing_rate 7016232848bps delivery_rate 29127111104bps delivered:104 app_limited busy:129ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:36842 172.28.30.30:9443
	 cubic wscale:10,10 rto:206 rtt:5.098/4.791 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:39704 bytes_acked:39705 bytes_received:32020 segs_out:102 segs_in:61 data_segs_out:68 data_segs_in:41 send 22722636bps lastsnd:306 lastrcv:301 lastack:301 pacing_rate 45444152bps delivery_rate 2574222216bps delivered:69 app_limited busy:162ms rcv_rtt:6 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.009 snd_wnd:79872
ESTAB 0      0          127.0.0.1:51782    127.0.0.1:9444
	 cubic wscale:10,10 rto:208 rtt:7.164/6.276 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:32690 bytes_acked:32691 bytes_received:22989 segs_out:62 segs_in:40 data_segs_out:31 data_segs_in:31 send 588900056bps lastsnd:307 lastrcv:301 lastack:301 pacing_rate 1177800104bps delivery_rate 38353454544bps delivered:32 app_limited busy:176ms rcv_rtt:12 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.004 snd_wnd:105472
ESTAB 0      0          127.0.0.1:51796    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.101/7.042 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5962 bytes_acked:5963 bytes_received:9387 segs_out:20 segs_in:14 data_segs_out:9 data_segs_in:8 send 516951319bps lastsnd:5186 lastrcv:5181 lastack:5181 pacing_rate 1033754376bps delivery_rate 28672000000bps delivered:10 app_limited busy:71ms rcv_rtt:5 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.01 snd_wnd:78848
ESTAB 0      0          127.0.0.1:55868    127.0.0.1:9444
	 cubic wscale:10,10 rto:202 rtt:1.6/3.153 ato:40 mss:39424 pmtu:65535 rcvmss:1996 advmss:65483 cwnd:10 bytes_sent:3998 bytes_acked:3999 bytes_received:2779 segs_out:8 segs_in:5 data_segs_out:4 data_segs_in:2 send 1971200000bps lastsnd:5388 lastrcv:5376 lastack:5376 pacing_rate 3941784096bps delivery_rate 28672000000bps delivered:5 app_limited busy:13ms rcv_space:65495 rcv_ssthresh:86892 minrtt:0.011 snd_wnd:78848
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:36206
	 cubic wscale:10,10 rto:211 rtt:10.427/16.698 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:40284 bytes_acked:40284 bytes_received:20581 segs_out:106 segs_in:141 data_segs_out:103 data_segs_in:103 send 251408842bps lastsnd:288 lastrcv:288 lastack:247 pacing_rate 502787544bps delivery_rate 29127111104bps delivered:104 app_limited busy:749ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.008 snd_wnd:65536

- matrix4_capture_1_receipt: {"bytes":"57407","sha256":"748cb7d376e98c7a0d9b8ca77ef62953f6cfb03c9bddbf7943f6ac9c02685771"}
- matrix4_capture_2_window: {"launch_requested_at":"2026-10-31T14:30:21.090338703Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:23.063086509Z","stop_requested_at":"2026-10-31T14:30:22.856248253Z"}
- matrix4_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
132 packets captured
132 packets received by filter
0 packets dropped by kernel

- matrix4_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457022.305554 eth0  In  IP (tos 0x0, ttl 63, id 43710, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [S], cksum 0xe3b1 (correct), seq 3665029260, win 64480, options [mss 1240,sackOK,TS val 3765990893 ecr 0,nop,wscale 10], length 0
1793457022.305572 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [S.], cksum 0x864d (incorrect -> 0x50cc), seq 2604513529, ack 3665029261, win 65160, options [mss 1460,sackOK,TS val 4280761329 ecr 3765990893,nop,wscale 10], length 0
1793457022.305899 eth0  In  IP (tos 0x0, ttl 63, id 43711, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x7de4 (correct), ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 0
1793457022.305910 eth0  In  IP (tos 0x0, ttl 63, id 43712, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], cksum 0x8f08 (correct), seq 3665029261:3665029305, ack 2604513530, win 63, options [nop,nop,TS val 3765990894 ecr 4280761329], length 44
1793457022.305914 eth0  Out IP (tos 0x0, ttl 64, id 58525, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x7db6), ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306062 eth0  Out IP (tos 0x0, ttl 64, id 58526, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], cksum 0x8648 (incorrect -> 0x2460), seq 2604513530:2604513533, ack 3665029305, win 64, options [nop,nop,TS val 4280761330 ecr 3765990894], length 3
1793457022.306139 eth0  In  IP (tos 0x0, ttl 63, id 43713, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x7db4 (correct), ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306277 eth0  In  IP (tos 0x0, ttl 63, id 43714, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665029305:3665031761, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306281 eth0  Out IP (tos 0x0, ttl 64, id 58527, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x7419), ack 3665031761, win 66, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306291 eth0  In  IP (tos 0x0, ttl 63, id 43716, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665031761:3665032989, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306300 eth0  In  IP (tos 0x0, ttl 63, id 43717, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665032989:3665035445, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306302 eth0  In  IP (tos 0x0, ttl 63, id 43719, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665035445:3665037901, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306303 eth0  Out IP (tos 0x0, ttl 64, id 58528, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604513533:2604514539, ack 3665032989, win 65, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306309 eth0  In  IP (tos 0x0, ttl 63, id 43721, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665037901:3665041585, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306333 eth0  Out IP (tos 0x0, ttl 64, id 58529, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604514539:2604515545, ack 3665041585, win 57, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306348 eth0  Out IP (tos 0x0, ttl 64, id 58530, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604515545:2604516551, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306357 eth0  Out IP (tos 0x0, ttl 64, id 58531, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604516551:2604517557, ack 3665041585, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306357 eth0  In  IP (tos 0x0, ttl 63, id 43724, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665041585:3665044041, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306367 eth0  Out IP (tos 0x0, ttl 64, id 58532, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x346f), ack 3665044041, win 60, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306369 eth0  In  IP (tos 0x0, ttl 63, id 43726, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665044041:3665046497, ack 2604513533, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306378 eth0  In  IP (tos 0x0, ttl 63, id 43728, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665046497:3665048953, ack 2604514539, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306379 eth0  Out IP (tos 0x0, ttl 64, id 58533, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604517557:2604518563, ack 3665046497, win 58, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306390 eth0  Out IP (tos 0x0, ttl 64, id 58534, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604518563:2604519569, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306405 eth0  Out IP (tos 0x0, ttl 64, id 58535, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604519569:2604520575, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306416 eth0  Out IP (tos 0x0, ttl 64, id 58536, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604520575:2604521581, ack 3665048953, win 56, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306424 eth0  In  IP (tos 0x0, ttl 63, id 43730, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665048953:3665051409, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306428 eth0  Out IP (tos 0x0, ttl 64, id 58537, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604521581:2604522587, ack 3665051409, win 55, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306432 eth0  In  IP (tos 0x0, ttl 63, id 43732, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665051409:3665052637, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306439 eth0  Out IP (tos 0x0, ttl 64, id 58538, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604522587:2604523593, ack 3665052637, win 54, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306444 eth0  In  IP (tos 0x0, ttl 63, id 43733, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665052637:3665057549, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306448 eth0  Out IP (tos 0x0, ttl 64, id 58539, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0xe81d), ack 3665057549, win 53, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306451 eth0  In  IP (tos 0x0, ttl 63, id 43737, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665057549:3665058777, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306456 eth0  In  IP (tos 0x0, ttl 63, id 43738, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665058777:3665060005, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306462 eth0  Out IP (tos 0x0, ttl 64, id 58540, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604523593:2604524599, ack 3665060005, win 51, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306464 eth0  In  IP (tos 0x0, ttl 63, id 43739, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665060005:3665061233, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306465 eth0  In  IP (tos 0x0, ttl 63, id 43740, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665061233:3665062461, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306470 eth0  In  IP (tos 0x0, ttl 63, id 43741, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665062461:3665066145, ack 2604515545, win 63, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306473 eth0  Out IP (tos 0x0, ttl 64, id 58541, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604524599:2604525605, ack 3665062461, win 49, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306482 eth0  Out IP (tos 0x0, ttl 64, id 58542, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0xbeb4), ack 3665066145, win 46, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306486 eth0  In  IP (tos 0x0, ttl 63, id 43744, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665066145:3665071057, ack 2604517557, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306490 eth0  In  IP (tos 0x0, ttl 63, id 43748, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665071057:3665072285, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306495 eth0  Out IP (tos 0x0, ttl 64, id 58543, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604525605:2604526611, ack 3665072285, win 41, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306498 eth0  In  IP (tos 0x0, ttl 63, id 43749, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665072285:3665074741, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306506 eth0  Out IP (tos 0x0, ttl 64, id 58544, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604526611:2604527617, ack 3665074741, win 39, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306511 eth0  In  IP (tos 0x0, ttl 63, id 43751, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665074741:3665075969, ack 2604518563, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306513 eth0  In  IP (tos 0x0, ttl 63, id 43752, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665075969:3665077197, ack 2604519569, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306519 eth0  Out IP (tos 0x0, ttl 64, id 58545, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604527617:2604528623, ack 3665077197, win 37, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306522 eth0  In  IP (tos 0x0, ttl 63, id 43753, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665077197:3665080881, ack 2604519569, win 62, options [nop,nop,TS val 3765990894 ecr 4280761330], length 3684
1793457022.306523 eth0  In  IP (tos 0x0, ttl 63, id 43756, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x94cd (correct), ack 2604521581, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306531 eth0  Out IP (tos 0x0, ttl 64, id 58546, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604528623:2604529629, ack 3665080881, win 34, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306533 eth0  In  IP (tos 0x0, ttl 63, id 43757, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665080881:3665085793, ack 2604522587, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 4912
1793457022.306537 eth0  In  IP (tos 0x0, ttl 63, id 43761, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665085793:3665087021, ack 2604523593, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306539 eth0  Out IP (tos 0x0, ttl 64, id 58547, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x624c), ack 3665085793, win 30, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306546 eth0  In  IP (tos 0x0, ttl 63, id 43762, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665087021:3665088249, ack 2604523593, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306547 eth0  In  IP (tos 0x0, ttl 63, id 43763, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665088249:3665089477, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1228
1793457022.306548 eth0  Out IP (tos 0x0, ttl 64, id 58548, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604529629:2604530635, ack 3665087021, win 29, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306553 eth0  In  IP (tos 0x0, ttl 63, id 43764, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], seq 3665089477:3665091933, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 2456
1793457022.306558 eth0  Out IP (tos 0x0, ttl 64, id 58549, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604530635:2604531641, ack 3665091933, win 25, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306560 eth0  In  IP (tos 0x0, ttl 63, id 43766, offset 0, flags [DF], proto TCP (6), length 1488)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [P.], seq 3665091933:3665093369, ack 2604524599, win 61, options [nop,nop,TS val 3765990894 ecr 4280761330], length 1436
1793457022.306566 eth0  Out IP (tos 0x0, ttl 64, id 58550, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [.], cksum 0x8645 (incorrect -> 0x3cde), ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 0
1793457022.306568 eth0  In  IP (tos 0x0, ttl 63, id 43768, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x5060 (correct), ack 2604526611, win 60, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306568 eth0  In  IP (tos 0x0, ttl 63, id 43769, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x4885 (correct), ack 2604528623, win 59, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306576 eth0  Out IP (tos 0x0, ttl 64, id 58551, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604531641:2604532647, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306585 eth0  Out IP (tos 0x0, ttl 64, id 58552, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604532647:2604533653, ack 3665093369, win 24, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306597 eth0  Out IP (tos 0x0, ttl 64, id 58553, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604533653:2604534659, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306599 eth0  In  IP (tos 0x0, ttl 63, id 43770, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x3cbd (correct), ack 2604531641, win 57, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306613 eth0  Out IP (tos 0x0, ttl 64, id 58554, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604534659:2604535665, ack 3665093369, win 26, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306622 eth0  In  IP (tos 0x0, ttl 63, id 43771, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x34e2 (correct), ack 2604533653, win 56, options [nop,nop,TS val 3765990894 ecr 4280761330], length 0
1793457022.306624 eth0  Out IP (tos 0x0, ttl 64, id 58555, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604535665:2604536671, ack 3665093369, win 27, options [nop,nop,TS val 4280761330 ecr 3765990894], length 1006
1793457022.306638 eth0  Out IP (tos 0x0, ttl 64, id 58556, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604536671:2604537677, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306650 eth0  Out IP (tos 0x0, ttl 64, id 58557, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604537677:2604538683, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306658 eth0  In  IP (tos 0x0, ttl 63, id 43772, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x2d06 (correct), ack 2604535665, win 55, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306659 eth0  Out IP (tos 0x0, ttl 64, id 58558, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604538683:2604539689, ack 3665093369, win 27, options [nop,nop,TS val 4280761331 ecr 3765990894], length 1006
1793457022.306670 eth0  Out IP (tos 0x0, ttl 64, id 58559, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604539689:2604540695, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306678 eth0  In  IP (tos 0x0, ttl 63, id 43773, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x252b (correct), ack 2604537677, win 54, options [nop,nop,TS val 3765990895 ecr 4280761330], length 0
1793457022.306681 eth0  Out IP (tos 0x0, ttl 64, id 58560, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604540695:2604541701, ack 3665093369, win 32, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306692 eth0  Out IP (tos 0x0, ttl 64, id 58561, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604541701:2604542707, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306699 eth0  In  IP (tos 0x0, ttl 63, id 43774, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x1d4f (correct), ack 2604539689, win 53, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306702 eth0  Out IP (tos 0x0, ttl 64, id 58562, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604542707:2604543713, ack 3665093369, win 36, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306714 eth0  Out IP (tos 0x0, ttl 64, id 58563, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604543713:2604544719, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306724 eth0  Out IP (tos 0x0, ttl 64, id 58564, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604544719:2604545725, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306734 eth0  In  IP (tos 0x0, ttl 63, id 43775, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x1574 (correct), ack 2604541701, win 52, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306740 eth0  Out IP (tos 0x0, ttl 64, id 58565, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604545725:2604546731, ack 3665093369, win 37, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306750 eth0  Out IP (tos 0x0, ttl 64, id 58566, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604546731:2604547737, ack 3665093369, win 38, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306762 eth0  Out IP (tos 0x0, ttl 64, id 58567, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604547737:2604548743, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306768 eth0  In  IP (tos 0x0, ttl 63, id 43776, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x0d99 (correct), ack 2604543713, win 51, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306773 eth0  Out IP (tos 0x0, ttl 64, id 58568, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604548743:2604549749, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306779 eth0  In  IP (tos 0x0, ttl 63, id 43777, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x05be (correct), ack 2604545725, win 50, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306785 eth0  Out IP (tos 0x0, ttl 64, id 58569, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604549749:2604550755, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306794 eth0  Out IP (tos 0x0, ttl 64, id 58570, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604550755:2604551761, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306806 eth0  Out IP (tos 0x0, ttl 64, id 58571, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604551761:2604552767, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306815 eth0  Out IP (tos 0x0, ttl 64, id 58572, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604552767:2604553773, ack 3665093369, win 42, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306816 eth0  In  IP (tos 0x0, ttl 63, id 43778, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xf9f5 (correct), ack 2604548743, win 48, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306827 eth0  Out IP (tos 0x0, ttl 64, id 58573, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604553773:2604554779, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306832 eth0  In  IP (tos 0x0, ttl 63, id 43779, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xf21a (correct), ack 2604550755, win 47, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306839 eth0  Out IP (tos 0x0, ttl 64, id 58574, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604554779:2604555785, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306845 eth0  In  IP (tos 0x0, ttl 63, id 43780, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xea3f (correct), ack 2604552767, win 46, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306849 eth0  Out IP (tos 0x0, ttl 64, id 58575, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604555785:2604556791, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306859 eth0  Out IP (tos 0x0, ttl 64, id 58576, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604556791:2604557797, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306864 eth0  In  IP (tos 0x0, ttl 63, id 43781, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xe264 (correct), ack 2604554779, win 45, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306869 eth0  Out IP (tos 0x0, ttl 64, id 58577, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604557797:2604558803, ack 3665093369, win 48, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306879 eth0  Out IP (tos 0x0, ttl 64, id 58578, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604558803:2604559809, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306888 eth0  Out IP (tos 0x0, ttl 64, id 58579, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604559809:2604560815, ack 3665093369, win 51, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306900 eth0  Out IP (tos 0x0, ttl 64, id 58580, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604560815:2604561821, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306909 eth0  Out IP (tos 0x0, ttl 64, id 58581, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604561821:2604562827, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306920 eth0  Out IP (tos 0x0, ttl 64, id 58582, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604562827:2604563833, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306928 eth0  Out IP (tos 0x0, ttl 64, id 58583, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604563833:2604564839, ack 3665093369, win 52, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306939 eth0  Out IP (tos 0x0, ttl 64, id 58584, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604564839:2604565845, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306940 eth0  In  IP (tos 0x0, ttl 63, id 43782, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xda89 (correct), ack 2604556791, win 44, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306941 eth0  In  IP (tos 0x0, ttl 63, id 43783, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xcec1 (correct), ack 2604559809, win 42, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306949 eth0  Out IP (tos 0x0, ttl 64, id 58585, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604565845:2604566851, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306959 eth0  In  IP (tos 0x0, ttl 63, id 43784, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xc2f9 (correct), ack 2604562827, win 40, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306960 eth0  Out IP (tos 0x0, ttl 64, id 58586, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604566851:2604567857, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306969 eth0  In  IP (tos 0x0, ttl 63, id 43785, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xbb1e (correct), ack 2604564839, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.306970 eth0  Out IP (tos 0x0, ttl 64, id 58587, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604567857:2604568863, ack 3665093369, win 56, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306981 eth0  Out IP (tos 0x0, ttl 64, id 58588, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604568863:2604569869, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306990 eth0  Out IP (tos 0x0, ttl 64, id 58589, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604569869:2604570875, ack 3665093369, win 61, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.306996 eth0  In  IP (tos 0x0, ttl 63, id 43786, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xaf54 (correct), ack 2604567857, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307002 eth0  Out IP (tos 0x0, ttl 64, id 58590, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604570875:2604571881, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307011 eth0  Out IP (tos 0x0, ttl 64, id 58591, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604571881:2604572887, ack 3665093369, win 63, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307021 eth0  In  IP (tos 0x0, ttl 63, id 43787, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0xa778 (correct), ack 2604569869, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307021 eth0  Out IP (tos 0x0, ttl 64, id 58592, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604572887:2604573893, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307031 eth0  Out IP (tos 0x0, ttl 64, id 58593, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604573893:2604574899, ack 3665093369, win 64, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307042 eth0  Out IP (tos 0x0, ttl 64, id 58594, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604574899:2604575905, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307052 eth0  In  IP (tos 0x0, ttl 63, id 43788, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x9bae (correct), ack 2604572887, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307053 eth0  Out IP (tos 0x0, ttl 64, id 58595, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604575905:2604576911, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307064 eth0  Out IP (tos 0x0, ttl 64, id 58596, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [P.], seq 2604576911:2604577917, ack 3665093369, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 1006
1793457022.307091 eth0  In  IP (tos 0x0, ttl 63, id 43789, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x93d2 (correct), ack 2604574899, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307096 eth0  In  IP (tos 0x0, ttl 63, id 43790, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x8bf6 (correct), ack 2604576911, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307249 eth0  In  IP (tos 0x0, ttl 63, id 43791, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [F.], cksum 0x8807 (correct), seq 3665093369, ack 2604577917, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0
1793457022.307381 eth0  Out IP (tos 0x0, ttl 64, id 58597, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.42202: Flags [F.], cksum 0x8645 (incorrect -> 0x87eb), seq 2604577917, ack 3665093370, win 66, options [nop,nop,TS val 4280761331 ecr 3765990895], length 0
1793457022.307423 eth0  In  IP (tos 0x0, ttl 63, id 43792, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.42202 > 192.0.3.10.9446: Flags [.], cksum 0x8806 (correct), ack 2604577918, win 39, options [nop,nop,TS val 3765990895 ecr 4280761331], length 0

- matrix4_path_2: eth0
2
1500
lo
1
65536
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ss: not found

- matrix4_capture_2_receipt: {"bytes":"16855","sha256":"c2743a09e0124d7b1d0847547fc261e445c4a4826440b1a66ffe6517a7cab9a0"}
- matrix4_origin_endpoint_samples: {"at":"2026-10-31T14:30:22.307266462Z","local_port":9446,"remote_port":42202,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":86,"sacked":0,"snd_cwnd":25,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}

- matrix6_input: {"input":{"underlay":"ipv6","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix6_transfer: {"after":{"upload_bytes":"136601","download_bytes":"137961","Present":true},"before":{"upload_bytes":"68585","download_bytes":"69822","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:30:29.184892027Z","local_port":40370,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":189,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"}}
- matrix6_capture_0_window: {"launch_requested_at":"2026-10-31T14:30:27.905588648Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:29.402521405Z","stop_requested_at":"2026-10-31T14:30:29.336784131Z"}
- matrix6_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
281 packets captured
285 packets received by filter
0 packets dropped by kernel

- matrix6_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457029.183182 fvnb0 Out IP (tos 0x0, ttl 64, id 52793, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [S], cksum 0x2a05 (incorrect -> 0x61f2), seq 4263915515, win 64480, options [mss 1240,sackOK,TS val 938711169 ecr 0,nop,wscale 10], length 0
1793457029.183368 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [S.], cksum 0xbf51 (correct), seq 1497978720, ack 4263915516, win 65160, options [mss 1460,sackOK,TS val 742638110 ecr 938711169,nop,wscale 10], length 0
1793457029.183388 fvnb0 Out IP (tos 0x0, ttl 64, id 52794, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xec6a), ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183428 fvnb0 Out IP (tos 0x0, ttl 64, id 52795, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], cksum 0x2a29 (incorrect -> 0xfd8e), seq 4263915516:4263915560, ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 44
1793457029.183571 fvnb0 In  IP (tos 0x0, ttl 63, id 47816, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0xec3d (correct), ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 0
1793457029.183572 fvnb0 In  IP (tos 0x0, ttl 63, id 47817, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], cksum 0x92e7 (correct), seq 1497978721:1497978724, ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 3
1793457029.183578 fvnb0 Out IP (tos 0x0, ttl 64, id 52796, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xec3b), ack 1497978724, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183657 fvnb0 Out IP (tos 0x0, ttl 64, id 52797, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263915560:4263921700, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 6140
1793457029.183671 fvnb0 Out IP (tos 0x0, ttl 64, id 52802, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263921700:4263927840, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 6140
1793457029.183725 fvnb0 In  IP (tos 0x0, ttl 63, id 47818, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0xd907 (correct), ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183730 fvnb0 Out IP (tos 0x0, ttl 64, id 52807, offset 0, flags [DF], proto TCP (6), length 8648)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263927840:4263936436, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 8596
1793457029.183738 fvnb0 Out IP (tos 0x0, ttl 64, id 52814, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263936436:4263937664, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183759 fvnb0 In  IP (tos 0x0, ttl 63, id 47819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497978724:1497979730, ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183759 fvnb0 In  IP (tos 0x0, ttl 63, id 47820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0xb858 (correct), ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183759 fvnb0 In  IP (tos 0x0, ttl 63, id 47821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497979730:1497980736, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183759 fvnb0 In  IP (tos 0x0, ttl 63, id 47822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497980736:1497981742, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183766 fvnb0 Out IP (tos 0x0, ttl 64, id 52815, offset 0, flags [DF], proto TCP (6), length 12332)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263937664:4263949944, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 12280
1793457029.183774 fvnb0 Out IP (tos 0x0, ttl 64, id 52825, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263949944:4263952400, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183779 fvnb0 In  IP (tos 0x0, ttl 63, id 47823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497981742:1497982748, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183779 fvnb0 In  IP (tos 0x0, ttl 63, id 47824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497982748:1497983754, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183788 fvnb0 Out IP (tos 0x0, ttl 64, id 52827, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x48ae), ack 1497983754, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.183867 fvnb0 In  IP (tos 0x0, ttl 63, id 47825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497983754:1497984760, ack 4263942576, win 48, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183876 fvnb0 Out IP (tos 0x0, ttl 64, id 52828, offset 0, flags [DF], proto TCP (6), length 19700)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263952400:4263972048, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 19648
1793457029.183887 fvnb0 In  IP (tos 0x0, ttl 63, id 47826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497984760:1497985766, ack 4263946260, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183887 fvnb0 In  IP (tos 0x0, ttl 63, id 47827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0x4f47 (correct), ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183887 fvnb0 In  IP (tos 0x0, ttl 63, id 47828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497985766:1497986772, ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183891 fvnb0 Out IP (tos 0x0, ttl 64, id 52844, offset 0, flags [DF], proto TCP (6), length 7628)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263972048:4263979624, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 7576
1793457029.183898 fvnb0 In  IP (tos 0x0, ttl 63, id 47829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497986772:1497987778, ack 4263952400, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183900 fvnb0 In  IP (tos 0x0, ttl 63, id 47830, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497987778:1497988784, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183906 fvnb0 In  IP (tos 0x0, ttl 63, id 47831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497988784:1497989790, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183909 fvnb0 Out IP (tos 0x0, ttl 64, id 52851, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xcab1), ack 1497988784, win 58, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.183926 fvnb0 In  IP (tos 0x0, ttl 63, id 47832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497989790:1497990796, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183935 fvnb0 In  IP (tos 0x0, ttl 63, id 47833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497990796:1497991802, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183940 fvnb0 Out IP (tos 0x0, ttl 64, id 52852, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xc2d6), ack 1497990796, win 57, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.183987 fvnb0 In  IP (tos 0x0, ttl 63, id 47834, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0x1621 (correct), ack 4263957312, win 41, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.184018 fvnb0 In  IP (tos 0x0, ttl 63, id 47835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497991802:1497992808, ack 4263960996, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184025 fvnb0 Out IP (tos 0x0, ttl 64, id 52853, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xbafb), ack 1497992808, win 56, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184050 fvnb0 In  IP (tos 0x0, ttl 63, id 47836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497992808:1497993814, ack 4263965908, win 34, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184063 fvnb0 In  IP (tos 0x0, ttl 63, id 47837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497993814:1497994820, ack 4263968364, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184064 fvnb0 In  IP (tos 0x0, ttl 63, id 47838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497994820:1497995826, ack 4263969592, win 32, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184072 fvnb0 In  IP (tos 0x0, ttl 63, id 47839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497995826:1497996832, ack 4263970820, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184080 fvnb0 Out IP (tos 0x0, ttl 64, id 52854, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xaf33), ack 1497995826, win 54, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184187 fvnb0 In  IP (tos 0x0, ttl 63, id 47840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497996832:1497997838, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184195 fvnb0 Out IP (tos 0x0, ttl 64, id 52855, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xa758), ack 1497997838, win 53, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184204 fvnb0 In  IP (tos 0x0, ttl 63, id 47841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497997838:1497998844, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184216 fvnb0 In  IP (tos 0x0, ttl 63, id 47842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497998844:1497999850, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184228 fvnb0 Out IP (tos 0x0, ttl 64, id 52856, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x9f7d), ack 1497999850, win 52, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184230 fvnb0 In  IP (tos 0x0, ttl 63, id 47843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497999850:1498000856, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184247 fvnb0 In  IP (tos 0x0, ttl 63, id 47844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498000856:1498001862, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184254 fvnb0 Out IP (tos 0x0, ttl 64, id 52857, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x97a2), ack 1498001862, win 51, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184259 fvnb0 In  IP (tos 0x0, ttl 63, id 47845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498001862:1498002868, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184284 fvnb0 In  IP (tos 0x0, ttl 63, id 47846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498002868:1498003874, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184291 fvnb0 In  IP (tos 0x0, ttl 63, id 47847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498003874:1498004880, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184291 fvnb0 Out IP (tos 0x0, ttl 64, id 52858, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x8fc7), ack 1498003874, win 50, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184302 fvnb0 In  IP (tos 0x0, ttl 63, id 47848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498004880:1498005886, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184308 fvnb0 Out IP (tos 0x0, ttl 64, id 52859, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x87ec), ack 1498005886, win 49, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184316 fvnb0 In  IP (tos 0x0, ttl 63, id 47849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498005886:1498006892, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184347 fvnb0 In  IP (tos 0x0, ttl 63, id 47850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498006892:1498007898, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184353 fvnb0 Out IP (tos 0x0, ttl 64, id 52860, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x8011), ack 1498007898, win 48, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184355 fvnb0 In  IP (tos 0x0, ttl 63, id 47851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498007898:1498008904, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184372 fvnb0 In  IP (tos 0x0, ttl 63, id 47852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498008904:1498009910, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184379 fvnb0 Out IP (tos 0x0, ttl 64, id 52861, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x7836), ack 1498009910, win 47, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184388 fvnb0 In  IP (tos 0x0, ttl 63, id 47853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498009910:1498010916, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184405 fvnb0 In  IP (tos 0x0, ttl 63, id 47854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498010916:1498011922, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184412 fvnb0 Out IP (tos 0x0, ttl 64, id 52862, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x705b), ack 1498011922, win 46, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184423 fvnb0 In  IP (tos 0x0, ttl 63, id 47855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498011922:1498012928, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184436 fvnb0 In  IP (tos 0x0, ttl 63, id 47856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498012928:1498013934, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184442 fvnb0 Out IP (tos 0x0, ttl 64, id 52863, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x6880), ack 1498013934, win 45, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184444 fvnb0 In  IP (tos 0x0, ttl 63, id 47857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498013934:1498014940, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184458 fvnb0 In  IP (tos 0x0, ttl 63, id 47858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498014940:1498015946, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184465 fvnb0 Out IP (tos 0x0, ttl 64, id 52864, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x60a5), ack 1498015946, win 44, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184469 fvnb0 In  IP (tos 0x0, ttl 63, id 47859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498015946:1498016952, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184486 fvnb0 In  IP (tos 0x0, ttl 63, id 47860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498016952:1498017958, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184491 fvnb0 Out IP (tos 0x0, ttl 64, id 52865, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x58ca), ack 1498017958, win 43, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184501 fvnb0 In  IP (tos 0x0, ttl 63, id 47861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498017958:1498018964, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184526 fvnb0 In  IP (tos 0x0, ttl 63, id 47862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498018964:1498019970, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184526 fvnb0 In  IP (tos 0x0, ttl 63, id 47863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498019970:1498020976, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184534 fvnb0 Out IP (tos 0x0, ttl 64, id 52866, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x4d02), ack 1498020976, win 41, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184539 fvnb0 In  IP (tos 0x0, ttl 63, id 47864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498020976:1498021982, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184554 fvnb0 In  IP (tos 0x0, ttl 63, id 47865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498021982:1498022988, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184560 fvnb0 Out IP (tos 0x0, ttl 64, id 52867, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x4527), ack 1498022988, win 40, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184574 fvnb0 In  IP (tos 0x0, ttl 63, id 47866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498022988:1498023994, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184595 fvnb0 In  IP (tos 0x0, ttl 63, id 47867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498023994:1498025000, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184601 fvnb0 Out IP (tos 0x0, ttl 64, id 52868, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x3d4c), ack 1498025000, win 39, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184616 fvnb0 In  IP (tos 0x0, ttl 63, id 47868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498025000:1498026006, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184633 fvnb0 In  IP (tos 0x0, ttl 63, id 47869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498026006:1498027012, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184640 fvnb0 Out IP (tos 0x0, ttl 64, id 52869, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x356f), ack 1498027012, win 39, options [nop,nop,TS val 938711171 ecr 742638111], length 0
1793457029.184659 fvnb0 In  IP (tos 0x0, ttl 63, id 47870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498027012:1498028018, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711170], length 1006
1793457029.184677 fvnb0 In  IP (tos 0x0, ttl 63, id 47871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498028018:1498029024, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184684 fvnb0 Out IP (tos 0x0, ttl 64, id 52870, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x2d92), ack 1498029024, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184700 fvnb0 In  IP (tos 0x0, ttl 63, id 47872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498029024:1498030030, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184718 fvnb0 In  IP (tos 0x0, ttl 63, id 47873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498030030:1498031036, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184724 fvnb0 Out IP (tos 0x0, ttl 64, id 52871, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x25b6), ack 1498031036, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184743 fvnb0 In  IP (tos 0x0, ttl 63, id 47874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498031036:1498032042, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184751 fvnb0 In  IP (tos 0x0, ttl 63, id 47875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498032042:1498033048, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184756 fvnb0 Out IP (tos 0x0, ttl 64, id 52872, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x1dda), ack 1498033048, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184766 fvnb0 In  IP (tos 0x0, ttl 63, id 47876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498033048:1498034054, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184777 fvnb0 In  IP (tos 0x0, ttl 63, id 47877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498034054:1498035060, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184783 fvnb0 Out IP (tos 0x0, ttl 64, id 52873, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x15fe), ack 1498035060, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184790 fvnb0 In  IP (tos 0x0, ttl 63, id 47878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498035060:1498036066, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184803 fvnb0 In  IP (tos 0x0, ttl 63, id 47879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498036066:1498037072, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184809 fvnb0 Out IP (tos 0x0, ttl 64, id 52874, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x0e22), ack 1498037072, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184827 fvnb0 In  IP (tos 0x0, ttl 63, id 47880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498037072:1498038078, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184832 fvnb0 In  IP (tos 0x0, ttl 63, id 47881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498038078:1498039084, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184837 fvnb0 Out IP (tos 0x0, ttl 64, id 52875, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x0646), ack 1498039084, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184842 fvnb0 In  IP (tos 0x0, ttl 63, id 47882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498039084:1498040090, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184854 fvnb0 In  IP (tos 0x0, ttl 63, id 47883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498040090:1498041096, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184860 fvnb0 Out IP (tos 0x0, ttl 64, id 52876, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xfe69), ack 1498041096, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184867 fvnb0 In  IP (tos 0x0, ttl 63, id 47884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498041096:1498042102, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184882 fvnb0 In  IP (tos 0x0, ttl 63, id 47885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498042102:1498043108, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184888 fvnb0 Out IP (tos 0x0, ttl 64, id 52877, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xf68d), ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185070 fvnb0 Out IP (tos 0x0, ttl 64, id 52878, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [F.], cksum 0x29fd (incorrect -> 0xf68c), seq 4263979624, ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185230 fvnb0 In  IP (tos 0x0, ttl 63, id 47886, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [F.], cksum 0xf66f (correct), seq 1498043108, ack 4263979625, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 0
1793457029.185235 fvnb0 Out IP (tos 0x0, ttl 64, id 52879, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xf68b), ack 1498043109, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0

- matrix6_path_0: eth0
2
1500
eth1
3
1500
fvnb0
6
1280
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4418,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:b5:ad:5a:5a:7a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4421,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:13:34:07:c1:06","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":6,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.78.2.126","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:49198 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.613/1.162 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5091 segs_out:17 segs_in:14 data_segs_out:9 data_segs_in:10 send 188972268bps lastsnd:44777 lastrcv:44776 lastack:14680 pacing_rate 377790456bps delivery_rate 2896000000bps delivered:10 app_limited busy:6ms rcv_space:14480 rcv_ssthresh:81050 minrtt:0.007 snd_wnd:71680
ESTAB 0      0       172.28.30.41:49846 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.273/0.323 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3820 bytes_acked:3821 bytes_received:6241 segs_out:47 segs_in:42 data_segs_out:25 data_segs_in:37 send 424322344bps lastsnd:186 lastrcv:186 lastack:186 pacing_rate 847868248bps delivery_rate 4633600000bps delivered:26 app_limited busy:4ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.005 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49862 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.272/2.386 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3512 bytes_acked:3513 bytes_received:6063 segs_out:32 segs_in:34 data_segs_out:18 data_segs_in:29 send 91069182bps lastsnd:5262 lastrcv:5262 lastack:5262 pacing_rate 182013152bps delivery_rate 3861333328bps delivered:19 app_limited busy:33ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.41:49878 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.365/10.551 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2928 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 21591799bps lastsnd:5262 lastrcv:5263 lastack:5221 pacing_rate 43181584bps delivery_rate 1362823528bps delivered:9 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:79828 minrtt:0.017 snd_wnd:74752

- matrix6_capture_0_receipt: {"bytes":"37944","sha256":"dadfda1fcd4b3be99c0d2b819727915f20e04518b95bfc044b3db9bbd73475d9"}
- matrix6_capture_1_window: {"launch_requested_at":"2026-10-31T14:30:27.945917695Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:29.737647619Z","stop_requested_at":"2026-10-31T14:30:29.570106954Z"}
- matrix6_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
459 packets captured
459 packets received by filter
0 packets dropped by kernel

- matrix6_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457029.183263 fvmb0 In  IP (tos 0x0, ttl 64, id 52793, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [S], cksum 0x61f2 (correct), seq 4263915515, win 64480, options [mss 1240,sackOK,TS val 938711169 ecr 0,nop,wscale 10], length 0
1793457029.183289 fvup0 Out IP (tos 0x0, ttl 63, id 52793, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [S], cksum 0x05aa (correct), seq 4263915515, win 64480, options [mss 1240,sackOK,TS val 938711169 ecr 0,nop,wscale 10], length 0
1793457029.183313 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [S.], cksum 0x864d (incorrect -> 0x6309), seq 1497978720, ack 4263915516, win 65160, options [mss 1460,sackOK,TS val 742638110 ecr 938711169,nop,wscale 10], length 0
1793457029.183318 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [S.], cksum 0x2a05 (incorrect -> 0xbf51), seq 1497978720, ack 4263915516, win 65160, options [mss 1460,sackOK,TS val 742638110 ecr 938711169,nop,wscale 10], length 0
1793457029.183432 fvmb0 In  IP (tos 0x0, ttl 64, id 52794, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xec6a (correct), ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183435 fvup0 Out IP (tos 0x0, ttl 63, id 52794, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x9022 (correct), ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183453 fvmb0 In  IP (tos 0x0, ttl 64, id 52795, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], cksum 0xfd8e (correct), seq 4263915516:4263915560, ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 44
1793457029.183456 fvup0 Out IP (tos 0x0, ttl 63, id 52795, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], cksum 0xa146 (correct), seq 4263915516:4263915560, ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 44
1793457029.183460 fvup0 In  IP (tos 0x0, ttl 64, id 47816, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0x8ff5), ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 0
1793457029.183462 fvmb0 Out IP (tos 0x0, ttl 63, id 47816, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0x29fd (incorrect -> 0xec3d), ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 0
1793457029.183546 fvup0 In  IP (tos 0x0, ttl 64, id 47817, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], cksum 0x8648 (incorrect -> 0x369f), seq 1497978721:1497978724, ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 3
1793457029.183554 fvmb0 Out IP (tos 0x0, ttl 63, id 47817, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], cksum 0x2a00 (incorrect -> 0x92e7), seq 1497978721:1497978724, ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 3
1793457029.183592 fvmb0 In  IP (tos 0x0, ttl 64, id 52796, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xec3b (correct), ack 1497978724, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183594 fvup0 Out IP (tos 0x0, ttl 63, id 52796, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x8ff3 (correct), ack 1497978724, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183689 fvmb0 In  IP (tos 0x0, ttl 64, id 52797, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263915560:4263916788, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183691 fvup0 Out IP (tos 0x0, ttl 63, id 52797, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263915560:4263916788, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183700 fvmb0 In  IP (tos 0x0, ttl 64, id 52798, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263916788:4263920472, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 3684
1793457029.183702 fvup0 Out IP (tos 0x0, ttl 63, id 52798, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263916788:4263920472, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 3684
1793457029.183706 fvup0 In  IP (tos 0x0, ttl 64, id 47818, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0x7cbf), ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183708 fvmb0 Out IP (tos 0x0, ttl 63, id 47818, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0x29fd (incorrect -> 0xd907), ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183710 fvmb0 In  IP (tos 0x0, ttl 64, id 52801, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263920472:4263921700, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183710 fvmb0 In  IP (tos 0x0, ttl 64, id 52802, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263921700:4263926612, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 4912
1793457029.183712 fvup0 Out IP (tos 0x0, ttl 63, id 52801, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263920472:4263921700, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183713 fvup0 In  IP (tos 0x0, ttl 64, id 47819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497978724:1497979730, ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183713 fvup0 Out IP (tos 0x0, ttl 63, id 52802, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263921700:4263926612, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 4912
1793457029.183716 fvmb0 Out IP (tos 0x0, ttl 63, id 47819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497978724:1497979730, ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183717 fvmb0 In  IP (tos 0x0, ttl 64, id 52806, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263926612:4263927840, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183719 fvup0 Out IP (tos 0x0, ttl 63, id 52806, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263926612:4263927840, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183725 fvup0 In  IP (tos 0x0, ttl 64, id 47820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0x5c10), ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183726 fvmb0 Out IP (tos 0x0, ttl 63, id 47820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0x29fd (incorrect -> 0xb858), ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183732 fvup0 In  IP (tos 0x0, ttl 64, id 47821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497979730:1497980736, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183734 fvmb0 Out IP (tos 0x0, ttl 63, id 47821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497979730:1497980736, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183741 fvup0 In  IP (tos 0x0, ttl 64, id 47822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497980736:1497981742, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183743 fvmb0 Out IP (tos 0x0, ttl 63, id 47822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497980736:1497981742, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183750 fvup0 In  IP (tos 0x0, ttl 64, id 47823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497981742:1497982748, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183752 fvmb0 Out IP (tos 0x0, ttl 63, id 47823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497981742:1497982748, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183758 fvup0 In  IP (tos 0x0, ttl 64, id 47824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497982748:1497983754, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183760 fvmb0 Out IP (tos 0x0, ttl 63, id 47824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497982748:1497983754, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183760 fvmb0 In  IP (tos 0x0, ttl 64, id 52807, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263927840:4263930296, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183763 fvup0 Out IP (tos 0x0, ttl 63, id 52807, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263927840:4263930296, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183766 fvmb0 In  IP (tos 0x0, ttl 64, id 52809, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263930296:4263931524, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183767 fvup0 Out IP (tos 0x0, ttl 63, id 52809, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263930296:4263931524, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183770 fvmb0 In  IP (tos 0x0, ttl 64, id 52810, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263931524:4263932752, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183772 fvup0 Out IP (tos 0x0, ttl 63, id 52810, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263931524:4263932752, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183775 fvmb0 In  IP (tos 0x0, ttl 64, id 52811, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263932752:4263935208, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183776 fvup0 Out IP (tos 0x0, ttl 63, id 52811, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263932752:4263935208, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183779 fvmb0 In  IP (tos 0x0, ttl 64, id 52813, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263935208:4263936436, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183780 fvup0 Out IP (tos 0x0, ttl 63, id 52813, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263935208:4263936436, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183791 fvmb0 In  IP (tos 0x0, ttl 64, id 52814, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263936436:4263937664, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183792 fvup0 Out IP (tos 0x0, ttl 63, id 52814, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263936436:4263937664, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183800 fvmb0 In  IP (tos 0x0, ttl 64, id 52815, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263937664:4263938892, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183802 fvup0 Out IP (tos 0x0, ttl 63, id 52815, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263937664:4263938892, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183806 fvmb0 In  IP (tos 0x0, ttl 64, id 52816, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263938892:4263941348, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183808 fvup0 Out IP (tos 0x0, ttl 63, id 52816, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263938892:4263941348, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183810 fvmb0 In  IP (tos 0x0, ttl 64, id 52818, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263941348:4263942576, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183811 fvup0 Out IP (tos 0x0, ttl 63, id 52818, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263941348:4263942576, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183824 fvup0 In  IP (tos 0x0, ttl 64, id 47825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497983754:1497984760, ack 4263942576, win 48, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183826 fvmb0 Out IP (tos 0x0, ttl 63, id 47825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497983754:1497984760, ack 4263942576, win 48, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183831 fvmb0 In  IP (tos 0x0, ttl 64, id 52819, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263942576:4263946260, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 3684
1793457029.183834 fvup0 Out IP (tos 0x0, ttl 63, id 52819, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263942576:4263946260, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 3684
1793457029.183837 fvmb0 In  IP (tos 0x0, ttl 64, id 52822, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263946260:4263948716, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183839 fvup0 Out IP (tos 0x0, ttl 63, id 52822, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263946260:4263948716, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183840 fvup0 In  IP (tos 0x0, ttl 64, id 47826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497984760:1497985766, ack 4263946260, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183843 fvmb0 Out IP (tos 0x0, ttl 63, id 47826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497984760:1497985766, ack 4263946260, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183850 fvup0 In  IP (tos 0x0, ttl 64, id 47827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0xf2fe), ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183853 fvmb0 Out IP (tos 0x0, ttl 63, id 47827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0x29fd (incorrect -> 0x4f47), ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183856 fvmb0 In  IP (tos 0x0, ttl 64, id 52824, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263948716:4263949944, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183856 fvmb0 In  IP (tos 0x0, ttl 64, id 52825, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263949944:4263951172, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183858 fvup0 Out IP (tos 0x0, ttl 63, id 52824, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263948716:4263949944, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183859 fvup0 Out IP (tos 0x0, ttl 63, id 52825, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263949944:4263951172, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183860 fvup0 In  IP (tos 0x0, ttl 64, id 47828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497985766:1497986772, ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183861 fvmb0 Out IP (tos 0x0, ttl 63, id 47828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497985766:1497986772, ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183862 fvmb0 In  IP (tos 0x0, ttl 64, id 52826, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263951172:4263952400, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183863 fvup0 Out IP (tos 0x0, ttl 63, id 52826, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263951172:4263952400, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183868 fvup0 In  IP (tos 0x0, ttl 64, id 47829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497986772:1497987778, ack 4263952400, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183870 fvmb0 Out IP (tos 0x0, ttl 63, id 47829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497986772:1497987778, ack 4263952400, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183870 fvmb0 In  IP (tos 0x0, ttl 64, id 52827, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x48ae (correct), ack 1497983754, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.183874 fvup0 Out IP (tos 0x0, ttl 63, id 52827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xec65 (correct), ack 1497983754, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.183882 fvup0 In  IP (tos 0x0, ttl 64, id 47830, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497987778:1497988784, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183884 fvmb0 Out IP (tos 0x0, ttl 63, id 47830, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497987778:1497988784, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183891 fvup0 In  IP (tos 0x0, ttl 64, id 47831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497988784:1497989790, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183893 fvmb0 Out IP (tos 0x0, ttl 63, id 47831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497988784:1497989790, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183901 fvup0 In  IP (tos 0x0, ttl 64, id 47832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497989790:1497990796, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183902 fvmb0 Out IP (tos 0x0, ttl 63, id 47832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497989790:1497990796, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183919 fvmb0 In  IP (tos 0x0, ttl 64, id 52828, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263952400:4263953628, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183920 fvup0 Out IP (tos 0x0, ttl 63, id 52828, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263952400:4263953628, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183921 fvup0 In  IP (tos 0x0, ttl 64, id 47833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497990796:1497991802, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183923 fvmb0 Out IP (tos 0x0, ttl 63, id 47833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497990796:1497991802, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183928 fvmb0 In  IP (tos 0x0, ttl 64, id 52829, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263953628:4263954856, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183930 fvup0 Out IP (tos 0x0, ttl 63, id 52829, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263953628:4263954856, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183940 fvmb0 In  IP (tos 0x0, ttl 64, id 52830, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263954856:4263956084, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183941 fvup0 Out IP (tos 0x0, ttl 63, id 52830, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263954856:4263956084, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183947 fvmb0 In  IP (tos 0x0, ttl 64, id 52831, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263956084:4263957312, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183949 fvup0 Out IP (tos 0x0, ttl 63, id 52831, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263956084:4263957312, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183955 fvmb0 In  IP (tos 0x0, ttl 64, id 52832, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263957312:4263958540, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183957 fvup0 Out IP (tos 0x0, ttl 63, id 52832, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263957312:4263958540, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183963 fvmb0 In  IP (tos 0x0, ttl 64, id 52833, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263958540:4263959768, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183965 fvup0 Out IP (tos 0x0, ttl 63, id 52833, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263958540:4263959768, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183966 fvup0 In  IP (tos 0x0, ttl 64, id 47834, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0xb9d8), ack 4263957312, win 41, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183974 fvmb0 In  IP (tos 0x0, ttl 64, id 52834, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263959768:4263960996, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183974 fvmb0 Out IP (tos 0x0, ttl 63, id 47834, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [.], cksum 0x29fd (incorrect -> 0x1621), ack 4263957312, win 41, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183978 fvup0 Out IP (tos 0x0, ttl 63, id 52834, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263959768:4263960996, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183982 fvmb0 In  IP (tos 0x0, ttl 64, id 52835, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263960996:4263962224, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183984 fvup0 Out IP (tos 0x0, ttl 63, id 52835, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263960996:4263962224, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183987 fvmb0 In  IP (tos 0x0, ttl 64, id 52836, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263962224:4263963452, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183988 fvup0 Out IP (tos 0x0, ttl 63, id 52836, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263962224:4263963452, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183993 fvup0 In  IP (tos 0x0, ttl 64, id 47835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497991802:1497992808, ack 4263960996, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183997 fvmb0 Out IP (tos 0x0, ttl 63, id 47835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497991802:1497992808, ack 4263960996, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184001 fvmb0 In  IP (tos 0x0, ttl 64, id 52837, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263963452:4263965908, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.184003 fvup0 Out IP (tos 0x0, ttl 63, id 52837, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263963452:4263965908, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.184009 fvup0 In  IP (tos 0x0, ttl 64, id 47836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497992808:1497993814, ack 4263965908, win 34, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184009 fvmb0 In  IP (tos 0x0, ttl 64, id 52839, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263965908:4263967136, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184012 fvup0 Out IP (tos 0x0, ttl 63, id 52839, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263965908:4263967136, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184013 fvmb0 Out IP (tos 0x0, ttl 63, id 47836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497992808:1497993814, ack 4263965908, win 34, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184019 fvmb0 In  IP (tos 0x0, ttl 64, id 52840, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263967136:4263968364, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184020 fvup0 Out IP (tos 0x0, ttl 63, id 52840, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263967136:4263968364, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184024 fvmb0 In  IP (tos 0x0, ttl 64, id 52841, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263968364:4263969592, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184025 fvup0 In  IP (tos 0x0, ttl 64, id 47837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497993814:1497994820, ack 4263968364, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184026 fvup0 Out IP (tos 0x0, ttl 63, id 52841, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263968364:4263969592, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184028 fvmb0 Out IP (tos 0x0, ttl 63, id 47837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497993814:1497994820, ack 4263968364, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184033 fvmb0 In  IP (tos 0x0, ttl 64, id 52842, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263969592:4263970820, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184035 fvup0 Out IP (tos 0x0, ttl 63, id 52842, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263969592:4263970820, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184037 fvup0 In  IP (tos 0x0, ttl 64, id 47838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497994820:1497995826, ack 4263969592, win 32, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184039 fvmb0 Out IP (tos 0x0, ttl 63, id 47838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497994820:1497995826, ack 4263969592, win 32, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184050 fvmb0 In  IP (tos 0x0, ttl 64, id 52843, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263970820:4263972048, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184051 fvmb0 In  IP (tos 0x0, ttl 64, id 52844, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263972048:4263973276, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184052 fvup0 Out IP (tos 0x0, ttl 63, id 52843, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263970820:4263972048, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184054 fvup0 Out IP (tos 0x0, ttl 63, id 52844, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263972048:4263973276, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184057 fvup0 In  IP (tos 0x0, ttl 64, id 47839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497995826:1497996832, ack 4263970820, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184059 fvmb0 In  IP (tos 0x0, ttl 64, id 52845, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263973276:4263974504, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184059 fvmb0 Out IP (tos 0x0, ttl 63, id 47839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497995826:1497996832, ack 4263970820, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184060 fvup0 Out IP (tos 0x0, ttl 63, id 52845, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263973276:4263974504, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184069 fvmb0 In  IP (tos 0x0, ttl 64, id 52846, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263974504:4263975732, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184070 fvup0 Out IP (tos 0x0, ttl 63, id 52846, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263974504:4263975732, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184079 fvmb0 In  IP (tos 0x0, ttl 64, id 52847, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263975732:4263976960, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184081 fvup0 Out IP (tos 0x0, ttl 63, id 52847, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263975732:4263976960, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184084 fvmb0 In  IP (tos 0x0, ttl 64, id 52848, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263976960:4263978188, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184085 fvup0 Out IP (tos 0x0, ttl 63, id 52848, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263976960:4263978188, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184090 fvmb0 In  IP (tos 0x0, ttl 64, id 52849, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], seq 4263978188:4263979416, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184091 fvup0 Out IP (tos 0x0, ttl 63, id 52849, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263978188:4263979416, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184094 fvmb0 In  IP (tos 0x0, ttl 64, id 52850, offset 0, flags [DF], proto TCP (6), length 260)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [P.], seq 4263979416:4263979624, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 208
1793457029.184095 fvup0 Out IP (tos 0x0, ttl 63, id 52850, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263979416:4263979624, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 208
1793457029.184102 fvmb0 In  IP (tos 0x0, ttl 64, id 52851, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xcab1 (correct), ack 1497988784, win 58, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184104 fvup0 Out IP (tos 0x0, ttl 63, id 52851, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x6e69 (correct), ack 1497988784, win 58, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184109 fvmb0 In  IP (tos 0x0, ttl 64, id 52852, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xc2d6 (correct), ack 1497990796, win 57, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184111 fvup0 Out IP (tos 0x0, ttl 63, id 52852, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x668e (correct), ack 1497990796, win 57, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184116 fvmb0 In  IP (tos 0x0, ttl 64, id 52853, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xbafb (correct), ack 1497992808, win 56, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184118 fvup0 Out IP (tos 0x0, ttl 63, id 52853, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x5eb3 (correct), ack 1497992808, win 56, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184165 fvmb0 In  IP (tos 0x0, ttl 64, id 52854, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xaf33 (correct), ack 1497995826, win 54, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184166 fvup0 In  IP (tos 0x0, ttl 64, id 47840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497996832:1497997838, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184167 fvup0 Out IP (tos 0x0, ttl 63, id 52854, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x52eb (correct), ack 1497995826, win 54, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184172 fvmb0 Out IP (tos 0x0, ttl 63, id 47840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497996832:1497997838, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184187 fvup0 In  IP (tos 0x0, ttl 64, id 47841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497997838:1497998844, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184191 fvmb0 Out IP (tos 0x0, ttl 63, id 47841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497997838:1497998844, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184199 fvup0 In  IP (tos 0x0, ttl 64, id 47842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497998844:1497999850, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184201 fvmb0 Out IP (tos 0x0, ttl 63, id 47842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497998844:1497999850, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184208 fvmb0 In  IP (tos 0x0, ttl 64, id 52855, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xa758 (correct), ack 1497997838, win 53, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184210 fvup0 Out IP (tos 0x0, ttl 63, id 52855, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x4b10 (correct), ack 1497997838, win 53, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184212 fvup0 In  IP (tos 0x0, ttl 64, id 47843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497999850:1498000856, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184216 fvmb0 Out IP (tos 0x0, ttl 63, id 47843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1497999850:1498000856, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184229 fvup0 In  IP (tos 0x0, ttl 64, id 47844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498000856:1498001862, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184232 fvmb0 Out IP (tos 0x0, ttl 63, id 47844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498000856:1498001862, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184241 fvmb0 In  IP (tos 0x0, ttl 64, id 52856, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x9f7d (correct), ack 1497999850, win 52, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184243 fvup0 Out IP (tos 0x0, ttl 63, id 52856, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x4335 (correct), ack 1497999850, win 52, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184245 fvup0 In  IP (tos 0x0, ttl 64, id 47845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498001862:1498002868, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184247 fvmb0 Out IP (tos 0x0, ttl 63, id 47845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498001862:1498002868, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184263 fvup0 In  IP (tos 0x0, ttl 64, id 47846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498002868:1498003874, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184265 fvmb0 Out IP (tos 0x0, ttl 63, id 47846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498002868:1498003874, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184267 fvmb0 In  IP (tos 0x0, ttl 64, id 52857, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x97a2 (correct), ack 1498001862, win 51, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184268 fvup0 Out IP (tos 0x0, ttl 63, id 52857, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x3b5a (correct), ack 1498001862, win 51, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184275 fvup0 In  IP (tos 0x0, ttl 64, id 47847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498003874:1498004880, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184278 fvmb0 Out IP (tos 0x0, ttl 63, id 47847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498003874:1498004880, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184285 fvup0 In  IP (tos 0x0, ttl 64, id 47848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498004880:1498005886, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184287 fvmb0 Out IP (tos 0x0, ttl 63, id 47848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498004880:1498005886, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184298 fvup0 In  IP (tos 0x0, ttl 64, id 47849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498005886:1498006892, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184301 fvmb0 Out IP (tos 0x0, ttl 63, id 47849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498005886:1498006892, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184308 fvmb0 In  IP (tos 0x0, ttl 64, id 52858, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x8fc7 (correct), ack 1498003874, win 50, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184310 fvup0 Out IP (tos 0x0, ttl 63, id 52858, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x337f (correct), ack 1498003874, win 50, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184321 fvup0 In  IP (tos 0x0, ttl 64, id 47850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498006892:1498007898, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184322 fvmb0 In  IP (tos 0x0, ttl 64, id 52859, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x87ec (correct), ack 1498005886, win 49, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184324 fvup0 Out IP (tos 0x0, ttl 63, id 52859, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x2ba4 (correct), ack 1498005886, win 49, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184326 fvmb0 Out IP (tos 0x0, ttl 63, id 47850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498006892:1498007898, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184340 fvup0 In  IP (tos 0x0, ttl 64, id 47851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498007898:1498008904, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184343 fvmb0 Out IP (tos 0x0, ttl 63, id 47851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498007898:1498008904, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184354 fvup0 In  IP (tos 0x0, ttl 64, id 47852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498008904:1498009910, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184359 fvmb0 Out IP (tos 0x0, ttl 63, id 47852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498008904:1498009910, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184371 fvup0 In  IP (tos 0x0, ttl 64, id 47853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498009910:1498010916, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184375 fvmb0 Out IP (tos 0x0, ttl 63, id 47853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498009910:1498010916, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184381 fvmb0 In  IP (tos 0x0, ttl 64, id 52860, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x8011 (correct), ack 1498007898, win 48, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184384 fvup0 Out IP (tos 0x0, ttl 63, id 52860, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x23c9 (correct), ack 1498007898, win 48, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184389 fvup0 In  IP (tos 0x0, ttl 64, id 47854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498010916:1498011922, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184391 fvmb0 Out IP (tos 0x0, ttl 63, id 47854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498010916:1498011922, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184398 fvmb0 In  IP (tos 0x0, ttl 64, id 52861, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x7836 (correct), ack 1498009910, win 47, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184401 fvup0 Out IP (tos 0x0, ttl 63, id 52861, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x1bee (correct), ack 1498009910, win 47, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184406 fvup0 In  IP (tos 0x0, ttl 64, id 47855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498011922:1498012928, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184410 fvmb0 Out IP (tos 0x0, ttl 63, id 47855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498011922:1498012928, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184421 fvup0 In  IP (tos 0x0, ttl 64, id 47856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498012928:1498013934, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184423 fvmb0 Out IP (tos 0x0, ttl 63, id 47856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498012928:1498013934, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184427 fvmb0 In  IP (tos 0x0, ttl 64, id 52862, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x705b (correct), ack 1498011922, win 46, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184429 fvup0 Out IP (tos 0x0, ttl 63, id 52862, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x1413 (correct), ack 1498011922, win 46, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184431 fvup0 In  IP (tos 0x0, ttl 64, id 47857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498013934:1498014940, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184433 fvmb0 Out IP (tos 0x0, ttl 63, id 47857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498013934:1498014940, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184443 fvup0 In  IP (tos 0x0, ttl 64, id 47858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498014940:1498015946, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184446 fvmb0 Out IP (tos 0x0, ttl 63, id 47858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498014940:1498015946, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184454 fvup0 In  IP (tos 0x0, ttl 64, id 47859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498015946:1498016952, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184454 fvmb0 In  IP (tos 0x0, ttl 64, id 52863, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x6880 (correct), ack 1498013934, win 45, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184456 fvup0 Out IP (tos 0x0, ttl 63, id 52863, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x0c38 (correct), ack 1498013934, win 45, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184456 fvmb0 Out IP (tos 0x0, ttl 63, id 47859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498015946:1498016952, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184467 fvup0 In  IP (tos 0x0, ttl 64, id 47860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498016952:1498017958, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184470 fvmb0 Out IP (tos 0x0, ttl 63, id 47860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498016952:1498017958, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184477 fvmb0 In  IP (tos 0x0, ttl 64, id 52864, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x60a5 (correct), ack 1498015946, win 44, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184479 fvup0 Out IP (tos 0x0, ttl 63, id 52864, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x045d (correct), ack 1498015946, win 44, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184479 fvup0 In  IP (tos 0x0, ttl 64, id 47861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498017958:1498018964, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184481 fvmb0 Out IP (tos 0x0, ttl 63, id 47861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498017958:1498018964, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184493 fvup0 In  IP (tos 0x0, ttl 64, id 47862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498018964:1498019970, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184495 fvmb0 Out IP (tos 0x0, ttl 63, id 47862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498018964:1498019970, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184504 fvmb0 In  IP (tos 0x0, ttl 64, id 52865, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x58ca (correct), ack 1498017958, win 43, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184506 fvup0 Out IP (tos 0x0, ttl 63, id 52865, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xfc81 (correct), ack 1498017958, win 43, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184509 fvup0 In  IP (tos 0x0, ttl 64, id 47863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498019970:1498020976, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184513 fvmb0 Out IP (tos 0x0, ttl 63, id 47863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498019970:1498020976, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184525 fvup0 In  IP (tos 0x0, ttl 64, id 47864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498020976:1498021982, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184528 fvmb0 Out IP (tos 0x0, ttl 63, id 47864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498020976:1498021982, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184535 fvup0 In  IP (tos 0x0, ttl 64, id 47865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498021982:1498022988, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184538 fvmb0 Out IP (tos 0x0, ttl 63, id 47865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498021982:1498022988, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184546 fvmb0 In  IP (tos 0x0, ttl 64, id 52866, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x4d02 (correct), ack 1498020976, win 41, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184548 fvup0 Out IP (tos 0x0, ttl 63, id 52866, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xf0b9 (correct), ack 1498020976, win 41, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184557 fvup0 In  IP (tos 0x0, ttl 64, id 47866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498022988:1498023994, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184560 fvmb0 Out IP (tos 0x0, ttl 63, id 47866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498022988:1498023994, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184572 fvmb0 In  IP (tos 0x0, ttl 64, id 52867, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x4527 (correct), ack 1498022988, win 40, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184574 fvup0 Out IP (tos 0x0, ttl 63, id 52867, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xe8de (correct), ack 1498022988, win 40, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184577 fvup0 In  IP (tos 0x0, ttl 64, id 47867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498023994:1498025000, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184582 fvmb0 Out IP (tos 0x0, ttl 63, id 47867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498023994:1498025000, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184597 fvup0 In  IP (tos 0x0, ttl 64, id 47868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498025000:1498026006, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184601 fvmb0 Out IP (tos 0x0, ttl 63, id 47868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498025000:1498026006, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184614 fvmb0 In  IP (tos 0x0, ttl 64, id 52868, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x3d4c (correct), ack 1498025000, win 39, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184615 fvup0 In  IP (tos 0x0, ttl 64, id 47869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498026006:1498027012, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184616 fvup0 Out IP (tos 0x0, ttl 63, id 52868, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xe103 (correct), ack 1498025000, win 39, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184619 fvmb0 Out IP (tos 0x0, ttl 63, id 47869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498026006:1498027012, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184642 fvup0 In  IP (tos 0x0, ttl 64, id 47870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498027012:1498028018, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711170], length 1006
1793457029.184647 fvmb0 Out IP (tos 0x0, ttl 63, id 47870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498027012:1498028018, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711170], length 1006
1793457029.184651 fvmb0 In  IP (tos 0x0, ttl 64, id 52869, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x356f (correct), ack 1498027012, win 39, options [nop,nop,TS val 938711171 ecr 742638111], length 0
1793457029.184653 fvup0 Out IP (tos 0x0, ttl 63, id 52869, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xd926 (correct), ack 1498027012, win 39, options [nop,nop,TS val 938711171 ecr 742638111], length 0
1793457029.184660 fvup0 In  IP (tos 0x0, ttl 64, id 47871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498028018:1498029024, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184664 fvmb0 Out IP (tos 0x0, ttl 63, id 47871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498028018:1498029024, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184683 fvup0 In  IP (tos 0x0, ttl 64, id 47872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498029024:1498030030, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184687 fvmb0 Out IP (tos 0x0, ttl 63, id 47872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498029024:1498030030, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184695 fvmb0 In  IP (tos 0x0, ttl 64, id 52870, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x2d92 (correct), ack 1498029024, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184697 fvup0 In  IP (tos 0x0, ttl 64, id 47873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498030030:1498031036, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184698 fvup0 Out IP (tos 0x0, ttl 63, id 52870, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xd149 (correct), ack 1498029024, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184702 fvmb0 Out IP (tos 0x0, ttl 63, id 47873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498030030:1498031036, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184724 fvup0 In  IP (tos 0x0, ttl 64, id 47874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498031036:1498032042, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184728 fvmb0 Out IP (tos 0x0, ttl 63, id 47874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498031036:1498032042, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184734 fvmb0 In  IP (tos 0x0, ttl 64, id 52871, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x25b6 (correct), ack 1498031036, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184736 fvup0 Out IP (tos 0x0, ttl 63, id 52871, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xc96d (correct), ack 1498031036, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184738 fvup0 In  IP (tos 0x0, ttl 64, id 47875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498032042:1498033048, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184741 fvmb0 Out IP (tos 0x0, ttl 63, id 47875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498032042:1498033048, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184752 fvup0 In  IP (tos 0x0, ttl 64, id 47876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498033048:1498034054, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184754 fvmb0 Out IP (tos 0x0, ttl 63, id 47876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498033048:1498034054, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184764 fvup0 In  IP (tos 0x0, ttl 64, id 47877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498034054:1498035060, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184766 fvmb0 Out IP (tos 0x0, ttl 63, id 47877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498034054:1498035060, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184771 fvmb0 In  IP (tos 0x0, ttl 64, id 52872, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x1dda (correct), ack 1498033048, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184773 fvup0 Out IP (tos 0x0, ttl 63, id 52872, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xc191 (correct), ack 1498033048, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184775 fvup0 In  IP (tos 0x0, ttl 64, id 47878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498035060:1498036066, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184778 fvmb0 Out IP (tos 0x0, ttl 63, id 47878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498035060:1498036066, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184787 fvup0 In  IP (tos 0x0, ttl 64, id 47879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498036066:1498037072, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184789 fvmb0 Out IP (tos 0x0, ttl 63, id 47879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498036066:1498037072, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184794 fvmb0 In  IP (tos 0x0, ttl 64, id 52873, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x15fe (correct), ack 1498035060, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184796 fvup0 Out IP (tos 0x0, ttl 63, id 52873, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xb9b5 (correct), ack 1498035060, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184803 fvup0 In  IP (tos 0x0, ttl 64, id 47880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498037072:1498038078, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184806 fvmb0 Out IP (tos 0x0, ttl 63, id 47880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498037072:1498038078, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184814 fvup0 In  IP (tos 0x0, ttl 64, id 47881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498038078:1498039084, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184817 fvmb0 Out IP (tos 0x0, ttl 63, id 47881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498038078:1498039084, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184820 fvmb0 In  IP (tos 0x0, ttl 64, id 52874, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x0e22 (correct), ack 1498037072, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184822 fvup0 Out IP (tos 0x0, ttl 63, id 52874, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xb1d9 (correct), ack 1498037072, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184826 fvup0 In  IP (tos 0x0, ttl 64, id 47882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498039084:1498040090, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184828 fvmb0 Out IP (tos 0x0, ttl 63, id 47882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498039084:1498040090, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184838 fvup0 In  IP (tos 0x0, ttl 64, id 47883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498040090:1498041096, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184841 fvmb0 Out IP (tos 0x0, ttl 63, id 47883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498040090:1498041096, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184847 fvmb0 In  IP (tos 0x0, ttl 64, id 52875, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0x0646 (correct), ack 1498039084, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184849 fvup0 Out IP (tos 0x0, ttl 63, id 52875, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xa9fd (correct), ack 1498039084, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184852 fvup0 In  IP (tos 0x0, ttl 64, id 47884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498041096:1498042102, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184855 fvmb0 Out IP (tos 0x0, ttl 63, id 47884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498041096:1498042102, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184865 fvup0 In  IP (tos 0x0, ttl 64, id 47885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498042102:1498043108, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184867 fvmb0 Out IP (tos 0x0, ttl 63, id 47885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [P.], seq 1498042102:1498043108, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184875 fvmb0 In  IP (tos 0x0, ttl 64, id 52876, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xfe69 (correct), ack 1498041096, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184877 fvup0 Out IP (tos 0x0, ttl 63, id 52876, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xa221 (correct), ack 1498041096, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184902 fvmb0 In  IP (tos 0x0, ttl 64, id 52877, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xf68d (correct), ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184904 fvup0 Out IP (tos 0x0, ttl 63, id 52877, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x9a45 (correct), ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185102 fvmb0 In  IP (tos 0x0, ttl 64, id 52878, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [F.], cksum 0xf68c (correct), seq 4263979624, ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185112 fvup0 Out IP (tos 0x0, ttl 63, id 52878, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [F.], cksum 0x9a44 (correct), seq 4263979624, ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185212 fvup0 In  IP (tos 0x0, ttl 64, id 47886, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [F.], cksum 0x8645 (incorrect -> 0x9a27), seq 1498043108, ack 4263979625, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 0
1793457029.185217 fvmb0 Out IP (tos 0x0, ttl 63, id 47886, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.40370: Flags [F.], cksum 0x29fd (incorrect -> 0xf66f), seq 1498043108, ack 4263979625, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 0
1793457029.185262 fvmb0 In  IP (tos 0x0, ttl 64, id 52879, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.40370 > 192.0.3.10.9446: Flags [.], cksum 0xf68b (correct), ack 1498043109, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185274 fvup0 Out IP (tos 0x0, ttl 63, id 52879, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x9a43 (correct), ack 1498043109, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0

- matrix6_path_1: eth0
2
1500
eth2
4
1500
fvfixture
5
1500
fvmb0
6
1280
fvup0
3
1500
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4416,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:25:c1:29:8d:d8","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4417,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"c6:ee:52:4f:5a:4e","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4420,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:8a:b4:de:17:2f","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ae:1b:a1:20:96:73","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.78.12.82","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51782
	 cubic wscale:10,10 rto:201 rtt:0.39/0.759 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:23393 bytes_acked:23393 bytes_received:32907 segs_out:40 segs_in:64 data_segs_out:32 data_segs_in:32 send 9347282051bps lastsnd:2294 lastrcv:2299 lastack:2294 pacing_rate 18664652800bps delivery_rate 121514666664bps delivered:33 app_limited busy:94ms rcv_rtt:1 rcv_space:65483 rcv_ssthresh:105032 minrtt:0.003 snd_wnd:91136
ESTAB 0      0       172.28.30.40:45976 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.453/0.443 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5240 bytes_acked:5241 bytes_received:10446 segs_out:103 segs_in:78 data_segs_out:46 data_segs_in:71 send 255717439bps lastsnd:703 lastrcv:703 lastack:703 pacing_rate 511152784bps delivery_rate 3309714280bps delivered:47 app_limited busy:56ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45986 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.194/0.168 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:10709 bytes_acked:10710 bytes_received:12760 segs_out:94 segs_in:104 data_segs_out:52 data_segs_in:95 send 597113402bps lastsnd:5551 lastrcv:5550 lastack:5550 pacing_rate 1193457816bps delivery_rate 1654857136bps delivered:53 app_limited busy:45ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45992 172.28.30.21:443
	 cubic wscale:10,10 rto:209 rtt:8.254/13.708 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2650 bytes_acked:2651 bytes_received:3874 segs_out:29 segs_in:28 data_segs_out:14 data_segs_in:17 send 14034408bps lastsnd:5552 lastrcv:5551 lastack:5551 pacing_rate 28067112bps delivery_rate 2896000000bps delivered:15 app_limited busy:127ms rcv_space:14480 rcv_ssthresh:85620 minrtt:0.008 snd_wnd:77824
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51796
	 cubic wscale:10,10 rto:203 rtt:2.308/4.59 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:9387 bytes_acked:9387 bytes_received:5962 segs_out:14 segs_in:20 data_segs_out:8 data_segs_in:9 send 1508492201bps lastsnd:12179 lastrcv:12184 lastack:12179 pacing_rate 3016004328bps delivery_rate 34816000000bps delivered:9 app_limited busy:41ms rcv_rtt:1245.38 rcv_space:65483 rcv_ssthresh:78471 minrtt:0.008 snd_wnd:83968
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:55868
	 cubic wscale:10,10 rto:206 rtt:5.079/10.134 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2779 bytes_acked:2779 bytes_received:3998 segs_out:4 segs_in:8 data_segs_out:2 data_segs_in:4 send 685489270bps lastsnd:12375 lastrcv:12387 lastack:12334 pacing_rate 1370877320bps delivery_rate 29013333328bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78342 minrtt:0.012 snd_wnd:87040
ESTAB 0      0          127.0.0.1:36206    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.707/0.046 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:21519 bytes_acked:21520 bytes_received:42363 segs_out:148 segs_in:112 data_segs_out:108 data_segs_in:108 send 3707835926bps lastsnd:2287 lastrcv:2286 lastack:2286 pacing_rate 7409122056bps delivery_rate 29127111104bps delivered:109 app_limited busy:132ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:36842 172.28.30.30:9443
	 cubic wscale:10,10 rto:205 rtt:4.935/3.918 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:39949 bytes_acked:39950 bytes_received:32424 segs_out:104 segs_in:62 data_segs_out:69 data_segs_in:42 send 23473151bps lastsnd:2305 lastrcv:2302 lastack:2302 pacing_rate 46937976bps delivery_rate 2574222216bps delivered:70 app_limited busy:165ms rcv_rtt:6 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.009 snd_wnd:79872
ESTAB 0      0          127.0.0.1:51782    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.789/5.456 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:32907 bytes_acked:32908 bytes_received:23393 segs_out:64 segs_in:41 data_segs_out:32 data_segs_in:32 send 621428782bps lastsnd:2307 lastrcv:2302 lastack:2302 pacing_rate 1242788912bps delivery_rate 38353454544bps delivered:33 app_limited busy:181ms rcv_rtt:12 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.004 snd_wnd:105472
ESTAB 0      0          127.0.0.1:51796    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.101/7.042 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5962 bytes_acked:5963 bytes_received:9387 segs_out:20 segs_in:14 data_segs_out:9 data_segs_in:8 send 516951319bps lastsnd:12189 lastrcv:12184 lastack:12184 pacing_rate 1033754376bps delivery_rate 28672000000bps delivered:10 app_limited busy:71ms rcv_rtt:5 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.01 snd_wnd:78848
ESTAB 0      0          127.0.0.1:55868    127.0.0.1:9444
	 cubic wscale:10,10 rto:202 rtt:1.6/3.153 ato:40 mss:39424 pmtu:65535 rcvmss:1996 advmss:65483 cwnd:10 bytes_sent:3998 bytes_acked:3999 bytes_received:2779 segs_out:8 segs_in:5 data_segs_out:4 data_segs_in:2 send 1971200000bps lastsnd:12390 lastrcv:12378 lastack:12378 pacing_rate 3941784096bps delivery_rate 28672000000bps delivered:5 app_limited busy:13ms rcv_space:65495 rcv_ssthresh:86892 minrtt:0.011 snd_wnd:78848
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:36206
	 cubic wscale:10,10 rto:211 rtt:10.483/16.744 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:42363 bytes_acked:42363 bytes_received:21519 segs_out:111 segs_in:148 data_segs_out:108 data_segs_in:108 send 250065821bps lastsnd:2289 lastrcv:2290 lastack:2248 pacing_rate 500125672bps delivery_rate 29127111104bps delivered:109 app_limited busy:791ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.008 snd_wnd:65536

- matrix6_capture_1_receipt: {"bytes":"61542","sha256":"308aeb17077bcac235826b54cb2a4617bdcd327e36ae755269a0f3be73b4f7d3"}
- matrix6_capture_2_window: {"launch_requested_at":"2026-10-31T14:30:27.96712406Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:30.077475756Z","stop_requested_at":"2026-10-31T14:30:29.859709894Z"}
- matrix6_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
147 packets captured
147 packets received by filter
0 packets dropped by kernel

- matrix6_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457029.183296 eth0  In  IP (tos 0x0, ttl 63, id 52793, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [S], cksum 0x05aa (correct), seq 4263915515, win 64480, options [mss 1240,sackOK,TS val 938711169 ecr 0,nop,wscale 10], length 0
1793457029.183312 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [S.], cksum 0x864d (incorrect -> 0x6309), seq 1497978720, ack 4263915516, win 65160, options [mss 1460,sackOK,TS val 742638110 ecr 938711169,nop,wscale 10], length 0
1793457029.183436 eth0  In  IP (tos 0x0, ttl 63, id 52794, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x9022 (correct), ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183456 eth0  In  IP (tos 0x0, ttl 63, id 52795, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], cksum 0xa146 (correct), seq 4263915516:4263915560, ack 1497978721, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 44
1793457029.183460 eth0  Out IP (tos 0x0, ttl 64, id 47816, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0x8ff5), ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 0
1793457029.183542 eth0  Out IP (tos 0x0, ttl 64, id 47817, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], cksum 0x8648 (incorrect -> 0x369f), seq 1497978721:1497978724, ack 4263915560, win 64, options [nop,nop,TS val 742638110 ecr 938711169], length 3
1793457029.183595 eth0  In  IP (tos 0x0, ttl 63, id 52796, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x8ff3 (correct), ack 1497978724, win 63, options [nop,nop,TS val 938711169 ecr 742638110], length 0
1793457029.183694 eth0  In  IP (tos 0x0, ttl 63, id 52797, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263915560:4263916788, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183703 eth0  In  IP (tos 0x0, ttl 63, id 52798, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263916788:4263920472, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 3684
1793457029.183705 eth0  Out IP (tos 0x0, ttl 64, id 47818, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0x7cbf), ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183712 eth0  Out IP (tos 0x0, ttl 64, id 47819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497978724:1497979730, ack 4263920472, win 65, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183715 eth0  In  IP (tos 0x0, ttl 63, id 52801, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263920472:4263921700, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183716 eth0  In  IP (tos 0x0, ttl 63, id 52802, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263921700:4263926612, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 4912
1793457029.183719 eth0  In  IP (tos 0x0, ttl 63, id 52806, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263926612:4263927840, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638110], length 1228
1793457029.183724 eth0  Out IP (tos 0x0, ttl 64, id 47820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0x5c10), ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183731 eth0  Out IP (tos 0x0, ttl 64, id 47821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497979730:1497980736, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183740 eth0  Out IP (tos 0x0, ttl 64, id 47822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497980736:1497981742, ack 4263927840, win 58, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183750 eth0  Out IP (tos 0x0, ttl 64, id 47823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497981742:1497982748, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183758 eth0  Out IP (tos 0x0, ttl 64, id 47824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497982748:1497983754, ack 4263927840, win 61, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183764 eth0  In  IP (tos 0x0, ttl 63, id 52807, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263927840:4263930296, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183768 eth0  In  IP (tos 0x0, ttl 63, id 52809, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263930296:4263931524, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183773 eth0  In  IP (tos 0x0, ttl 63, id 52810, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263931524:4263932752, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183777 eth0  In  IP (tos 0x0, ttl 63, id 52811, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263932752:4263935208, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183780 eth0  In  IP (tos 0x0, ttl 63, id 52813, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263935208:4263936436, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183793 eth0  In  IP (tos 0x0, ttl 63, id 52814, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263936436:4263937664, ack 1497978724, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183803 eth0  In  IP (tos 0x0, ttl 63, id 52815, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263937664:4263938892, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183809 eth0  In  IP (tos 0x0, ttl 63, id 52816, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263938892:4263941348, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183812 eth0  In  IP (tos 0x0, ttl 63, id 52818, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263941348:4263942576, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183821 eth0  Out IP (tos 0x0, ttl 64, id 47825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497983754:1497984760, ack 4263942576, win 48, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183835 eth0  In  IP (tos 0x0, ttl 63, id 52819, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263942576:4263946260, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 3684
1793457029.183839 eth0  Out IP (tos 0x0, ttl 64, id 47826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497984760:1497985766, ack 4263946260, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183839 eth0  In  IP (tos 0x0, ttl 63, id 52822, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263946260:4263948716, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.183848 eth0  Out IP (tos 0x0, ttl 64, id 47827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0xf2fe), ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183858 eth0  Out IP (tos 0x0, ttl 64, id 47828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497985766:1497986772, ack 4263948716, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183860 eth0  In  IP (tos 0x0, ttl 63, id 52824, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263948716:4263949944, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183861 eth0  In  IP (tos 0x0, ttl 63, id 52825, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263949944:4263951172, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183864 eth0  In  IP (tos 0x0, ttl 63, id 52826, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263951172:4263952400, ack 1497979730, win 63, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183868 eth0  Out IP (tos 0x0, ttl 64, id 47829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497986772:1497987778, ack 4263952400, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183875 eth0  In  IP (tos 0x0, ttl 63, id 52827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xec65 (correct), ack 1497983754, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.183881 eth0  Out IP (tos 0x0, ttl 64, id 47830, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497987778:1497988784, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183890 eth0  Out IP (tos 0x0, ttl 64, id 47831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497988784:1497989790, ack 4263952400, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183900 eth0  Out IP (tos 0x0, ttl 64, id 47832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497989790:1497990796, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183920 eth0  Out IP (tos 0x0, ttl 64, id 47833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497990796:1497991802, ack 4263952400, win 45, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.183922 eth0  In  IP (tos 0x0, ttl 63, id 52828, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263952400:4263953628, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183931 eth0  In  IP (tos 0x0, ttl 63, id 52829, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263953628:4263954856, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183942 eth0  In  IP (tos 0x0, ttl 63, id 52830, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263954856:4263956084, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183949 eth0  In  IP (tos 0x0, ttl 63, id 52831, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263956084:4263957312, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183957 eth0  In  IP (tos 0x0, ttl 63, id 52832, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263957312:4263958540, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183962 eth0  Out IP (tos 0x0, ttl 64, id 47834, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [.], cksum 0x8645 (incorrect -> 0xb9d8), ack 4263957312, win 41, options [nop,nop,TS val 742638111 ecr 938711170], length 0
1793457029.183965 eth0  In  IP (tos 0x0, ttl 63, id 52833, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263958540:4263959768, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183979 eth0  In  IP (tos 0x0, ttl 63, id 52834, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263959768:4263960996, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183985 eth0  In  IP (tos 0x0, ttl 63, id 52835, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263960996:4263962224, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183989 eth0  In  IP (tos 0x0, ttl 63, id 52836, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263962224:4263963452, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.183990 eth0  Out IP (tos 0x0, ttl 64, id 47835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497991802:1497992808, ack 4263960996, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184004 eth0  In  IP (tos 0x0, ttl 63, id 52837, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263963452:4263965908, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 2456
1793457029.184007 eth0  Out IP (tos 0x0, ttl 64, id 47836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497992808:1497993814, ack 4263965908, win 34, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184012 eth0  In  IP (tos 0x0, ttl 63, id 52839, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263965908:4263967136, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184020 eth0  In  IP (tos 0x0, ttl 63, id 52840, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263967136:4263968364, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184024 eth0  Out IP (tos 0x0, ttl 64, id 47837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497993814:1497994820, ack 4263968364, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184027 eth0  In  IP (tos 0x0, ttl 63, id 52841, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263968364:4263969592, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184035 eth0  Out IP (tos 0x0, ttl 64, id 47838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497994820:1497995826, ack 4263969592, win 32, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184035 eth0  In  IP (tos 0x0, ttl 63, id 52842, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263969592:4263970820, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184055 eth0  In  IP (tos 0x0, ttl 63, id 52843, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263970820:4263972048, ack 1497984760, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184055 eth0  In  IP (tos 0x0, ttl 63, id 52844, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263972048:4263973276, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184056 eth0  Out IP (tos 0x0, ttl 64, id 47839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497995826:1497996832, ack 4263970820, win 33, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184061 eth0  In  IP (tos 0x0, ttl 63, id 52845, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263973276:4263974504, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184071 eth0  In  IP (tos 0x0, ttl 63, id 52846, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263974504:4263975732, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184082 eth0  In  IP (tos 0x0, ttl 63, id 52847, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263975732:4263976960, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184086 eth0  In  IP (tos 0x0, ttl 63, id 52848, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263976960:4263978188, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184092 eth0  In  IP (tos 0x0, ttl 63, id 52849, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], seq 4263978188:4263979416, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 1228
1793457029.184097 eth0  In  IP (tos 0x0, ttl 63, id 52850, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [P.], seq 4263979416:4263979624, ack 1497985766, win 60, options [nop,nop,TS val 938711170 ecr 742638111], length 208
1793457029.184105 eth0  In  IP (tos 0x0, ttl 63, id 52851, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x6e69 (correct), ack 1497988784, win 58, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184112 eth0  In  IP (tos 0x0, ttl 63, id 52852, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x668e (correct), ack 1497990796, win 57, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184119 eth0  In  IP (tos 0x0, ttl 63, id 52853, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x5eb3 (correct), ack 1497992808, win 56, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184162 eth0  Out IP (tos 0x0, ttl 64, id 47840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497996832:1497997838, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184168 eth0  In  IP (tos 0x0, ttl 63, id 52854, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x52eb (correct), ack 1497995826, win 54, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184186 eth0  Out IP (tos 0x0, ttl 64, id 47841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497997838:1497998844, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184197 eth0  Out IP (tos 0x0, ttl 64, id 47842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497998844:1497999850, ack 4263979624, win 25, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184211 eth0  Out IP (tos 0x0, ttl 64, id 47843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1497999850:1498000856, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184211 eth0  In  IP (tos 0x0, ttl 63, id 52855, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x4b10 (correct), ack 1497997838, win 53, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184227 eth0  Out IP (tos 0x0, ttl 64, id 47844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498000856:1498001862, ack 4263979624, win 27, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184243 eth0  Out IP (tos 0x0, ttl 64, id 47845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498001862:1498002868, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184244 eth0  In  IP (tos 0x0, ttl 63, id 52856, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x4335 (correct), ack 1497999850, win 52, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184261 eth0  Out IP (tos 0x0, ttl 64, id 47846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498002868:1498003874, ack 4263979624, win 30, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184270 eth0  In  IP (tos 0x0, ttl 63, id 52857, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x3b5a (correct), ack 1498001862, win 51, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184274 eth0  Out IP (tos 0x0, ttl 64, id 47847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498003874:1498004880, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184284 eth0  Out IP (tos 0x0, ttl 64, id 47848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498004880:1498005886, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184297 eth0  Out IP (tos 0x0, ttl 64, id 47849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498005886:1498006892, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184310 eth0  In  IP (tos 0x0, ttl 63, id 52858, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x337f (correct), ack 1498003874, win 50, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184320 eth0  Out IP (tos 0x0, ttl 64, id 47850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498006892:1498007898, ack 4263979624, win 31, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184325 eth0  In  IP (tos 0x0, ttl 63, id 52859, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x2ba4 (correct), ack 1498005886, win 49, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184339 eth0  Out IP (tos 0x0, ttl 64, id 47851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498007898:1498008904, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184353 eth0  Out IP (tos 0x0, ttl 64, id 47852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498008904:1498009910, ack 4263979624, win 36, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184370 eth0  Out IP (tos 0x0, ttl 64, id 47853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498009910:1498010916, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184384 eth0  In  IP (tos 0x0, ttl 63, id 52860, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x23c9 (correct), ack 1498007898, win 48, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184388 eth0  Out IP (tos 0x0, ttl 64, id 47854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498010916:1498011922, ack 4263979624, win 38, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184401 eth0  In  IP (tos 0x0, ttl 63, id 52861, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x1bee (correct), ack 1498009910, win 47, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184405 eth0  Out IP (tos 0x0, ttl 64, id 47855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498011922:1498012928, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184420 eth0  Out IP (tos 0x0, ttl 64, id 47856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498012928:1498013934, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184430 eth0  Out IP (tos 0x0, ttl 64, id 47857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498013934:1498014940, ack 4263979624, win 40, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184430 eth0  In  IP (tos 0x0, ttl 63, id 52862, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x1413 (correct), ack 1498011922, win 46, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184442 eth0  Out IP (tos 0x0, ttl 64, id 47858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498014940:1498015946, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184453 eth0  Out IP (tos 0x0, ttl 64, id 47859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498015946:1498016952, ack 4263979624, win 43, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184457 eth0  In  IP (tos 0x0, ttl 63, id 52863, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x0c38 (correct), ack 1498013934, win 45, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184466 eth0  Out IP (tos 0x0, ttl 64, id 47860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498016952:1498017958, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184478 eth0  Out IP (tos 0x0, ttl 64, id 47861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498017958:1498018964, ack 4263979624, win 44, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184479 eth0  In  IP (tos 0x0, ttl 63, id 52864, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x045d (correct), ack 1498015946, win 44, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184492 eth0  Out IP (tos 0x0, ttl 64, id 47862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498018964:1498019970, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184507 eth0  In  IP (tos 0x0, ttl 63, id 52865, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xfc81 (correct), ack 1498017958, win 43, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184508 eth0  Out IP (tos 0x0, ttl 64, id 47863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498019970:1498020976, ack 4263979624, win 46, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184523 eth0  Out IP (tos 0x0, ttl 64, id 47864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498020976:1498021982, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184534 eth0  Out IP (tos 0x0, ttl 64, id 47865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498021982:1498022988, ack 4263979624, win 49, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184549 eth0  In  IP (tos 0x0, ttl 63, id 52866, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xf0b9 (correct), ack 1498020976, win 41, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184554 eth0  Out IP (tos 0x0, ttl 64, id 47866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498022988:1498023994, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184574 eth0  In  IP (tos 0x0, ttl 63, id 52867, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xe8de (correct), ack 1498022988, win 40, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184576 eth0  Out IP (tos 0x0, ttl 64, id 47867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498023994:1498025000, ack 4263979624, win 50, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184595 eth0  Out IP (tos 0x0, ttl 64, id 47868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498025000:1498026006, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184613 eth0  Out IP (tos 0x0, ttl 64, id 47869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498026006:1498027012, ack 4263979624, win 52, options [nop,nop,TS val 742638111 ecr 938711170], length 1006
1793457029.184617 eth0  In  IP (tos 0x0, ttl 63, id 52868, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xe103 (correct), ack 1498025000, win 39, options [nop,nop,TS val 938711170 ecr 742638111], length 0
1793457029.184641 eth0  Out IP (tos 0x0, ttl 64, id 47870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498027012:1498028018, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711170], length 1006
1793457029.184654 eth0  In  IP (tos 0x0, ttl 63, id 52869, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xd926 (correct), ack 1498027012, win 39, options [nop,nop,TS val 938711171 ecr 742638111], length 0
1793457029.184658 eth0  Out IP (tos 0x0, ttl 64, id 47871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498028018:1498029024, ack 4263979624, win 55, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184681 eth0  Out IP (tos 0x0, ttl 64, id 47872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498029024:1498030030, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184696 eth0  Out IP (tos 0x0, ttl 64, id 47873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498030030:1498031036, ack 4263979624, win 56, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184699 eth0  In  IP (tos 0x0, ttl 63, id 52870, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xd149 (correct), ack 1498029024, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184721 eth0  Out IP (tos 0x0, ttl 64, id 47874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498031036:1498032042, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184737 eth0  In  IP (tos 0x0, ttl 63, id 52871, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xc96d (correct), ack 1498031036, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184737 eth0  Out IP (tos 0x0, ttl 64, id 47875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498032042:1498033048, ack 4263979624, win 58, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184751 eth0  Out IP (tos 0x0, ttl 64, id 47876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498033048:1498034054, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184762 eth0  Out IP (tos 0x0, ttl 64, id 47877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498034054:1498035060, ack 4263979624, win 61, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184773 eth0  In  IP (tos 0x0, ttl 63, id 52872, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xc191 (correct), ack 1498033048, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184774 eth0  Out IP (tos 0x0, ttl 64, id 47878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498035060:1498036066, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184786 eth0  Out IP (tos 0x0, ttl 64, id 47879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498036066:1498037072, ack 4263979624, win 62, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184797 eth0  In  IP (tos 0x0, ttl 63, id 52873, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xb9b5 (correct), ack 1498035060, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184802 eth0  Out IP (tos 0x0, ttl 64, id 47880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498037072:1498038078, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184813 eth0  Out IP (tos 0x0, ttl 64, id 47881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498038078:1498039084, ack 4263979624, win 64, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184822 eth0  In  IP (tos 0x0, ttl 63, id 52874, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xb1d9 (correct), ack 1498037072, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184825 eth0  Out IP (tos 0x0, ttl 64, id 47882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498039084:1498040090, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184837 eth0  Out IP (tos 0x0, ttl 64, id 47883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498040090:1498041096, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184849 eth0  In  IP (tos 0x0, ttl 63, id 52875, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xa9fd (correct), ack 1498039084, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184851 eth0  Out IP (tos 0x0, ttl 64, id 47884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498041096:1498042102, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184863 eth0  Out IP (tos 0x0, ttl 64, id 47885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [P.], seq 1498042102:1498043108, ack 4263979624, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 1006
1793457029.184877 eth0  In  IP (tos 0x0, ttl 63, id 52876, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0xa221 (correct), ack 1498041096, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.184905 eth0  In  IP (tos 0x0, ttl 63, id 52877, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x9a45 (correct), ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185117 eth0  In  IP (tos 0x0, ttl 63, id 52878, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [F.], cksum 0x9a44 (correct), seq 4263979624, ack 1498043108, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0
1793457029.185210 eth0  Out IP (tos 0x0, ttl 64, id 47886, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.40370: Flags [F.], cksum 0x8645 (incorrect -> 0x9a27), seq 1498043108, ack 4263979625, win 67, options [nop,nop,TS val 742638112 ecr 938711171], length 0
1793457029.185278 eth0  In  IP (tos 0x0, ttl 63, id 52879, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.40370 > 192.0.3.10.9446: Flags [.], cksum 0x9a43 (correct), ack 1498043109, win 39, options [nop,nop,TS val 938711171 ecr 742638112], length 0

- matrix6_path_2: eth0
2
1500
lo
1
65536
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ss: not found

- matrix6_capture_2_receipt: {"bytes":"18903","sha256":"306418c4c2d250ca6cafe1af4402b83efee6825e5210e40d3a07b80ca498baa3"}
- matrix6_origin_endpoint_samples: {"at":"2026-10-31T14:30:22.307266462Z","local_port":9446,"remote_port":42202,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":86,"sacked":0,"snd_cwnd":25,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:30:29.185133874Z","local_port":9446,"remote_port":40370,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":63,"sacked":0,"snd_cwnd":26,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}

- matrixDual_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrixDual_transfer: {"after":{"upload_bytes":"204045","download_bytes":"206256","Present":true},"before":{"upload_bytes":"136601","download_bytes":"137961","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:30:36.168669227Z","local_port":56968,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":131,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":65536,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- matrixDual_ipv6_block: {"native_member":{"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":0,"error":"connect_unavailable"},"positive_control":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0},"scope":"D-123: IPv6 connection must fail; IPv4 bulk must complete. Native refusal may precede tunnel ingress; this does not claim ctv6 was hit. Kernel ctv6 proof is the separate netlab row."}
- matrixDual_capture_0_window: {"launch_requested_at":"2026-10-31T14:30:34.882886917Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:36.436244954Z","stop_requested_at":"2026-10-31T14:30:36.361361775Z"}
- matrixDual_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
279 packets captured
283 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457036.166875 fvnb0 Out IP (tos 0x0, ttl 64, id 2335, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [S], cksum 0x2a05 (incorrect -> 0xe674), seq 578216201, win 64480, options [mss 1240,sackOK,TS val 200485323 ecr 0,nop,wscale 10], length 0
1793457036.167057 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [S.], cksum 0x696e (correct), seq 3226301449, ack 578216202, win 65160, options [mss 1460,sackOK,TS val 685186115 ecr 200485323,nop,wscale 10], length 0
1793457036.167074 fvnb0 Out IP (tos 0x0, ttl 64, id 2336, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x9687), ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 0
1793457036.167115 fvnb0 Out IP (tos 0x0, ttl 64, id 2337, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], cksum 0x2a29 (incorrect -> 0xa7ab), seq 578216202:578216246, ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 44
1793457036.167552 fvnb0 In  IP (tos 0x0, ttl 63, id 5168, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x965a (correct), ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 0
1793457036.167793 fvnb0 In  IP (tos 0x0, ttl 63, id 5169, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], cksum 0x3d04 (correct), seq 3226301450:3226301453, ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 3
1793457036.167799 fvnb0 Out IP (tos 0x0, ttl 64, id 2338, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x9657), ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 0
1793457036.167931 fvnb0 Out IP (tos 0x0, ttl 64, id 2339, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578216246:578222386, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 6140
1793457036.167949 fvnb0 Out IP (tos 0x0, ttl 64, id 2344, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578222386:578228526, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 6140
1793457036.168031 fvnb0 In  IP (tos 0x0, ttl 63, id 5170, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x8325 (correct), ack 578221158, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168035 fvnb0 Out IP (tos 0x0, ttl 64, id 2349, offset 0, flags [DF], proto TCP (6), length 8648)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578228526:578237122, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 8596
1793457036.168042 fvnb0 In  IP (tos 0x0, ttl 63, id 5171, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226301453:3226302459, ack 578224842, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168046 fvnb0 Out IP (tos 0x0, ttl 64, id 2356, offset 0, flags [DF], proto TCP (6), length 8648)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578237122:578245718, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 8596
1793457036.168067 fvnb0 In  IP (tos 0x0, ttl 63, id 5172, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226302459:3226303465, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168068 fvnb0 In  IP (tos 0x0, ttl 63, id 5173, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226303465:3226304471, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168076 fvnb0 In  IP (tos 0x0, ttl 63, id 5174, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226304471:3226305477, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168078 fvnb0 Out IP (tos 0x0, ttl 64, id 2363, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x176d), ack 3226304471, win 62, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168093 fvnb0 In  IP (tos 0x0, ttl 63, id 5175, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226305477:3226306483, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168098 fvnb0 In  IP (tos 0x0, ttl 63, id 5176, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226306483:3226307489, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168110 fvnb0 Out IP (tos 0x0, ttl 64, id 2364, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x0ba5), ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168114 fvnb0 In  IP (tos 0x0, ttl 63, id 5177, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x3b9e (correct), ack 578233438, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168115 fvnb0 In  IP (tos 0x0, ttl 63, id 5178, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226307489:3226308495, ack 578238350, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168117 fvnb0 Out IP (tos 0x0, ttl 64, id 2365, offset 0, flags [DF], proto TCP (6), length 14788)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578245718:578260454, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 14736
1793457036.168125 fvnb0 Out IP (tos 0x0, ttl 64, id 2377, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578260454:578262910, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168131 fvnb0 Out IP (tos 0x0, ttl 64, id 2379, offset 0, flags [DF], proto TCP (6), length 9876)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578262910:578272734, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 9824
1793457036.168134 fvnb0 In  IP (tos 0x0, ttl 63, id 5179, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226308495:3226309501, ack 578239578, win 54, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168135 fvnb0 In  IP (tos 0x0, ttl 63, id 5180, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x0d6a (correct), ack 578243262, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168147 fvnb0 In  IP (tos 0x0, ttl 63, id 5181, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226309501:3226310507, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168147 fvnb0 In  IP (tos 0x0, ttl 63, id 5182, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226310507:3226311513, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168150 fvnb0 Out IP (tos 0x0, ttl 64, id 2387, offset 0, flags [DF], proto TCP (6), length 7628)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578272734:578280310, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 7576
1793457036.168156 fvnb0 In  IP (tos 0x0, ttl 63, id 5183, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226311513:3226312519, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168164 fvnb0 Out IP (tos 0x0, ttl 64, id 2394, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x70e0), ack 3226312519, win 58, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168172 fvnb0 In  IP (tos 0x0, ttl 63, id 5184, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226312519:3226313525, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168180 fvnb0 In  IP (tos 0x0, ttl 63, id 5185, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226313525:3226314531, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168184 fvnb0 Out IP (tos 0x0, ttl 64, id 2395, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x6905), ack 3226314531, win 57, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168186 fvnb0 In  IP (tos 0x0, ttl 63, id 5186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226314531:3226315537, ack 578248174, win 50, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168194 fvnb0 In  IP (tos 0x0, ttl 63, id 5187, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0xd910 (correct), ack 578250630, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168204 fvnb0 In  IP (tos 0x0, ttl 63, id 5188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226315537:3226316543, ack 578255542, win 44, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168209 fvnb0 Out IP (tos 0x0, ttl 64, id 2396, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x612a), ack 3226316543, win 56, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168224 fvnb0 In  IP (tos 0x0, ttl 63, id 5189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226316543:3226317549, ack 578259226, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168225 fvnb0 In  IP (tos 0x0, ttl 63, id 5190, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0xa60f (correct), ack 578261682, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168244 fvnb0 In  IP (tos 0x0, ttl 63, id 5191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226317549:3226318555, ack 578262910, win 40, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168245 fvnb0 In  IP (tos 0x0, ttl 63, id 5192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226318555:3226319561, ack 578266594, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168250 fvnb0 Out IP (tos 0x0, ttl 64, id 2397, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x5562), ack 3226319561, win 54, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168259 fvnb0 In  IP (tos 0x0, ttl 63, id 5193, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x8171 (correct), ack 578269050, win 35, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168268 fvnb0 In  IP (tos 0x0, ttl 63, id 5194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226319561:3226320567, ack 578272734, win 32, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168268 fvnb0 In  IP (tos 0x0, ttl 63, id 5195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226320567:3226321573, ack 578273962, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168275 fvnb0 In  IP (tos 0x0, ttl 63, id 5196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x533c (correct), ack 578278874, win 28, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168276 fvnb0 Out IP (tos 0x0, ttl 64, id 2398, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x4d87), ack 3226321573, win 53, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168281 fvnb0 In  IP (tos 0x0, ttl 63, id 5197, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226321573:3226322579, ack 578280102, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168296 fvnb0 In  IP (tos 0x0, ttl 63, id 5198, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226322579:3226323585, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168301 fvnb0 Out IP (tos 0x0, ttl 64, id 2399, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x45ac), ack 3226323585, win 52, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168305 fvnb0 In  IP (tos 0x0, ttl 63, id 5199, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226323585:3226324591, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168313 fvnb0 In  IP (tos 0x0, ttl 63, id 5200, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226324591:3226325597, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168317 fvnb0 Out IP (tos 0x0, ttl 64, id 2400, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x3dd1), ack 3226325597, win 51, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168321 fvnb0 In  IP (tos 0x0, ttl 63, id 5201, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226325597:3226326603, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168329 fvnb0 In  IP (tos 0x0, ttl 63, id 5202, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226326603:3226327609, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168333 fvnb0 Out IP (tos 0x0, ttl 64, id 2401, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x35f6), ack 3226327609, win 50, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168338 fvnb0 In  IP (tos 0x0, ttl 63, id 5203, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226327609:3226328615, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168349 fvnb0 In  IP (tos 0x0, ttl 63, id 5204, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226328615:3226329621, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168354 fvnb0 Out IP (tos 0x0, ttl 64, id 2402, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x2e1b), ack 3226329621, win 49, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168367 fvnb0 In  IP (tos 0x0, ttl 63, id 5205, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226329621:3226330627, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168367 fvnb0 In  IP (tos 0x0, ttl 63, id 5206, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226330627:3226331633, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168372 fvnb0 In  IP (tos 0x0, ttl 63, id 5207, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226331633:3226332639, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168378 fvnb0 Out IP (tos 0x0, ttl 64, id 2403, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x2253), ack 3226332639, win 47, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168401 fvnb0 In  IP (tos 0x0, ttl 63, id 5208, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226332639:3226333645, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168402 fvnb0 In  IP (tos 0x0, ttl 63, id 5209, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226333645:3226334651, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168404 fvnb0 In  IP (tos 0x0, ttl 63, id 5210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226334651:3226335657, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168407 fvnb0 Out IP (tos 0x0, ttl 64, id 2404, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x168b), ack 3226335657, win 45, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168416 fvnb0 In  IP (tos 0x0, ttl 63, id 5211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226335657:3226336663, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168425 fvnb0 In  IP (tos 0x0, ttl 63, id 5212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226336663:3226337669, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168430 fvnb0 Out IP (tos 0x0, ttl 64, id 2405, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x0eb0), ack 3226337669, win 44, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168435 fvnb0 In  IP (tos 0x0, ttl 63, id 5213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226337669:3226338675, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168445 fvnb0 In  IP (tos 0x0, ttl 63, id 5214, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226338675:3226339681, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168456 fvnb0 In  IP (tos 0x0, ttl 63, id 5215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226339681:3226340687, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168457 fvnb0 Out IP (tos 0x0, ttl 64, id 2406, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x06d5), ack 3226339681, win 43, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168479 fvnb0 In  IP (tos 0x0, ttl 63, id 5216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226340687:3226341693, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168479 fvnb0 In  IP (tos 0x0, ttl 63, id 5217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226341693:3226342699, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168482 fvnb0 In  IP (tos 0x0, ttl 63, id 5218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226342699:3226343705, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168485 fvnb0 Out IP (tos 0x0, ttl 64, id 2407, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xf71f), ack 3226343705, win 40, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168500 fvnb0 In  IP (tos 0x0, ttl 63, id 5219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226343705:3226344711, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168502 fvnb0 In  IP (tos 0x0, ttl 63, id 5220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226344711:3226345717, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168504 fvnb0 In  IP (tos 0x0, ttl 63, id 5221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226345717:3226346723, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168513 fvnb0 Out IP (tos 0x0, ttl 64, id 2408, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xeb56), ack 3226346723, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168523 fvnb0 In  IP (tos 0x0, ttl 63, id 5222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226346723:3226347729, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168536 fvnb0 In  IP (tos 0x0, ttl 63, id 5223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226347729:3226348735, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168537 fvnb0 In  IP (tos 0x0, ttl 63, id 5224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226348735:3226349741, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168537 fvnb0 In  IP (tos 0x0, ttl 63, id 5225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226349741:3226350747, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168543 fvnb0 Out IP (tos 0x0, ttl 64, id 2409, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xdb9e), ack 3226350747, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168558 fvnb0 In  IP (tos 0x0, ttl 63, id 5226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226350747:3226351753, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168560 fvnb0 In  IP (tos 0x0, ttl 63, id 5227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226351753:3226352759, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168573 fvnb0 Out IP (tos 0x0, ttl 64, id 2410, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xd3c2), ack 3226352759, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168576 fvnb0 In  IP (tos 0x0, ttl 63, id 5228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226352759:3226353765, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168577 fvnb0 In  IP (tos 0x0, ttl 63, id 5229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226353765:3226354771, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168580 fvnb0 Out IP (tos 0x0, ttl 64, id 2411, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xcbe6), ack 3226354771, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168584 fvnb0 In  IP (tos 0x0, ttl 63, id 5230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226354771:3226355777, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168591 fvnb0 In  IP (tos 0x0, ttl 63, id 5231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226355777:3226356783, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168602 fvnb0 In  IP (tos 0x0, ttl 63, id 5232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226356783:3226357789, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168603 fvnb0 In  IP (tos 0x0, ttl 63, id 5233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226357789:3226358795, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168604 fvnb0 Out IP (tos 0x0, ttl 64, id 2412, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xc40a), ack 3226356783, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168612 fvnb0 Out IP (tos 0x0, ttl 64, id 2413, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xbc2e), ack 3226358795, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168615 fvnb0 In  IP (tos 0x0, ttl 63, id 5234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226358795:3226359801, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168623 fvnb0 In  IP (tos 0x0, ttl 63, id 5235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226359801:3226360807, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168634 fvnb0 Out IP (tos 0x0, ttl 64, id 2414, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xb451), ack 3226360807, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168634 fvnb0 In  IP (tos 0x0, ttl 63, id 5236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226360807:3226361813, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168644 fvnb0 In  IP (tos 0x0, ttl 63, id 5237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226361813:3226362819, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168648 fvnb0 Out IP (tos 0x0, ttl 64, id 2415, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xac75), ack 3226362819, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168650 fvnb0 In  IP (tos 0x0, ttl 63, id 5238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226362819:3226363825, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168660 fvnb0 In  IP (tos 0x0, ttl 63, id 5239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226363825:3226364831, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168663 fvnb0 Out IP (tos 0x0, ttl 64, id 2416, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xa498), ack 3226364831, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168665 fvnb0 In  IP (tos 0x0, ttl 63, id 5240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226364831:3226365837, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168810 fvnb0 Out IP (tos 0x0, ttl 64, id 2417, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [F.], cksum 0x29fd (incorrect -> 0xa0a9), seq 578280310, ack 3226365837, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168917 fvnb0 In  IP (tos 0x0, ttl 63, id 5241, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [F.], cksum 0xa08f (correct), seq 3226365837, ack 578280311, win 64, options [nop,nop,TS val 685186117 ecr 200485325], length 0
1793457036.168921 fvnb0 Out IP (tos 0x0, ttl 64, id 2418, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xa0a8), ack 3226365838, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0

- matrixDual_path_0: eth0
2
1500
eth1
3
1500
fvnb0
7
1280
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4418,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:b5:ad:5a:5a:7a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4421,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:13:34:07:c1:06","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":7,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.78.2.126","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:49198 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.613/1.162 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5091 segs_out:17 segs_in:14 data_segs_out:9 data_segs_in:10 send 188972268bps lastsnd:51821 lastrcv:51820 lastack:21724 pacing_rate 377790456bps delivery_rate 2896000000bps delivered:10 app_limited busy:6ms rcv_space:14480 rcv_ssthresh:81050 minrtt:0.007 snd_wnd:71680
ESTAB 0      0       172.28.30.41:54046 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.228/0.279 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3566 bytes_acked:3567 bytes_received:5712 segs_out:36 segs_in:34 data_segs_out:21 data_segs_in:29 send 508070175bps lastsnd:514 lastrcv:514 lastack:514 pacing_rate 1014471808bps delivery_rate 2896000000bps delivered:22 app_limited busy:3ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.41:54054 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.927/1.783 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3906 bytes_acked:3907 bytes_received:6298 segs_out:36 segs_in:36 data_segs_out:20 data_segs_in:32 send 124962244bps lastsnd:5347 lastrcv:5347 lastack:5347 pacing_rate 249756096bps delivery_rate 2574222216bps delivered:21 app_limited busy:28ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.41:54070 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.353/10.584 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2927 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 21640202bps lastsnd:5347 lastrcv:5347 lastack:5305 pacing_rate 43274336bps delivery_rate 2896000000bps delivered:9 app_limited busy:43ms rcv_space:14480 rcv_ssthresh:79826 minrtt:0.008 snd_wnd:74752

- matrixDual_capture_0_receipt: {"bytes":"37716","sha256":"a67937d225f3f99dbfd279fb75fe67eb30401726196c22989aa7bfd4d04480b3"}
- matrixDual_capture_1_window: {"launch_requested_at":"2026-10-31T14:30:34.918382419Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:36.791545157Z","stop_requested_at":"2026-10-31T14:30:36.624999128Z"}
- matrixDual_ipv6_uplink_no_leak: true
- matrixDual_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
443 packets captured
443 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457036.166946 fvmb0 In  IP (tos 0x0, ttl 64, id 2335, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [S], cksum 0xe674 (correct), seq 578216201, win 64480, options [mss 1240,sackOK,TS val 200485323 ecr 0,nop,wscale 10], length 0
1793457036.166971 fvup0 Out IP (tos 0x0, ttl 63, id 2335, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [S], cksum 0x8a2c (correct), seq 578216201, win 64480, options [mss 1240,sackOK,TS val 200485323 ecr 0,nop,wscale 10], length 0
1793457036.166995 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [S.], cksum 0x864d (incorrect -> 0x0d26), seq 3226301449, ack 578216202, win 65160, options [mss 1460,sackOK,TS val 685186115 ecr 200485323,nop,wscale 10], length 0
1793457036.167000 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [S.], cksum 0x2a05 (incorrect -> 0x696e), seq 3226301449, ack 578216202, win 65160, options [mss 1460,sackOK,TS val 685186115 ecr 200485323,nop,wscale 10], length 0
1793457036.167098 fvmb0 In  IP (tos 0x0, ttl 64, id 2336, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x9687 (correct), ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 0
1793457036.167101 fvup0 Out IP (tos 0x0, ttl 63, id 2336, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x3a3f (correct), ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 0
1793457036.167504 fvmb0 In  IP (tos 0x0, ttl 64, id 2337, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], cksum 0xa7ab (correct), seq 578216202:578216246, ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 44
1793457036.167507 fvup0 Out IP (tos 0x0, ttl 63, id 2337, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], cksum 0x4b63 (correct), seq 578216202:578216246, ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 44
1793457036.167513 fvup0 In  IP (tos 0x0, ttl 64, id 5168, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x3a12), ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 0
1793457036.167515 fvmb0 Out IP (tos 0x0, ttl 63, id 5168, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0x965a), ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 0
1793457036.167532 fvup0 In  IP (tos 0x0, ttl 64, id 5169, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], cksum 0x8648 (incorrect -> 0xe0bb), seq 3226301450:3226301453, ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 3
1793457036.167540 fvmb0 Out IP (tos 0x0, ttl 63, id 5169, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], cksum 0x2a00 (incorrect -> 0x3d04), seq 3226301450:3226301453, ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 3
1793457036.167839 fvmb0 In  IP (tos 0x0, ttl 64, id 2338, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x9657 (correct), ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 0
1793457036.167843 fvup0 Out IP (tos 0x0, ttl 63, id 2338, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x3a0f (correct), ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 0
1793457036.167996 fvmb0 In  IP (tos 0x0, ttl 64, id 2339, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578216246:578221158, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 4912
1793457036.167999 fvup0 Out IP (tos 0x0, ttl 63, id 2339, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578216246:578221158, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 4912
1793457036.168005 fvup0 In  IP (tos 0x0, ttl 64, id 5170, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x26dd), ack 578221158, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168008 fvmb0 Out IP (tos 0x0, ttl 63, id 5170, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0x8325), ack 578221158, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168010 fvmb0 In  IP (tos 0x0, ttl 64, id 2343, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578221158:578222386, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 1228
1793457036.168010 fvmb0 In  IP (tos 0x0, ttl 64, id 2344, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578222386:578224842, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 2456
1793457036.168012 fvup0 Out IP (tos 0x0, ttl 63, id 2343, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578221158:578222386, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 1228
1793457036.168014 fvup0 Out IP (tos 0x0, ttl 63, id 2344, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578222386:578224842, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 2456
1793457036.168019 fvmb0 In  IP (tos 0x0, ttl 64, id 2346, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578224842:578227298, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 2456
1793457036.168020 fvup0 Out IP (tos 0x0, ttl 63, id 2346, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578224842:578227298, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 2456
1793457036.168021 fvup0 In  IP (tos 0x0, ttl 64, id 5171, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226301453:3226302459, ack 578224842, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168024 fvmb0 Out IP (tos 0x0, ttl 63, id 5171, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226301453:3226302459, ack 578224842, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168033 fvup0 In  IP (tos 0x0, ttl 64, id 5172, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226302459:3226303465, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168036 fvmb0 Out IP (tos 0x0, ttl 63, id 5172, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226302459:3226303465, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168049 fvup0 In  IP (tos 0x0, ttl 64, id 5173, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226303465:3226304471, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168050 fvmb0 Out IP (tos 0x0, ttl 63, id 5173, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226303465:3226304471, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168057 fvup0 In  IP (tos 0x0, ttl 64, id 5174, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226304471:3226305477, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168059 fvmb0 Out IP (tos 0x0, ttl 63, id 5174, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226304471:3226305477, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168071 fvup0 In  IP (tos 0x0, ttl 64, id 5175, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226305477:3226306483, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168073 fvmb0 Out IP (tos 0x0, ttl 63, id 5175, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226305477:3226306483, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168074 fvmb0 In  IP (tos 0x0, ttl 64, id 2348, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578227298:578228526, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 1228
1793457036.168075 fvmb0 In  IP (tos 0x0, ttl 64, id 2349, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578228526:578233438, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 4912
1793457036.168077 fvup0 Out IP (tos 0x0, ttl 63, id 2348, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578227298:578228526, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 1228
1793457036.168078 fvup0 Out IP (tos 0x0, ttl 63, id 2349, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578228526:578233438, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 4912
1793457036.168082 fvup0 In  IP (tos 0x0, ttl 64, id 5176, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226306483:3226307489, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168083 fvmb0 Out IP (tos 0x0, ttl 63, id 5176, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226306483:3226307489, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168086 fvmb0 In  IP (tos 0x0, ttl 64, id 2353, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578233438:578237122, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 3684
1793457036.168087 fvup0 Out IP (tos 0x0, ttl 63, id 2353, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578233438:578237122, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 3684
1793457036.168088 fvup0 In  IP (tos 0x0, ttl 64, id 5177, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0xdf55), ack 578233438, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168089 fvmb0 In  IP (tos 0x0, ttl 64, id 2356, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578237122:578238350, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168090 fvmb0 Out IP (tos 0x0, ttl 63, id 5177, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0x3b9e), ack 578233438, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168091 fvup0 Out IP (tos 0x0, ttl 63, id 2356, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578237122:578238350, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168094 fvmb0 In  IP (tos 0x0, ttl 64, id 2357, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578238350:578239578, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168096 fvup0 Out IP (tos 0x0, ttl 63, id 2357, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578238350:578239578, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168096 fvup0 In  IP (tos 0x0, ttl 64, id 5178, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226307489:3226308495, ack 578238350, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168098 fvmb0 Out IP (tos 0x0, ttl 63, id 5178, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226307489:3226308495, ack 578238350, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168100 fvmb0 In  IP (tos 0x0, ttl 64, id 2358, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578239578:578242034, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168102 fvup0 Out IP (tos 0x0, ttl 63, id 2358, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578239578:578242034, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168104 fvmb0 In  IP (tos 0x0, ttl 64, id 2360, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578242034:578243262, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168105 fvup0 In  IP (tos 0x0, ttl 64, id 5179, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226308495:3226309501, ack 578239578, win 54, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168106 fvmb0 Out IP (tos 0x0, ttl 63, id 5179, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226308495:3226309501, ack 578239578, win 54, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168107 fvup0 Out IP (tos 0x0, ttl 63, id 2360, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578242034:578243262, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168110 fvmb0 In  IP (tos 0x0, ttl 64, id 2361, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578243262:578245718, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168112 fvup0 Out IP (tos 0x0, ttl 63, id 2361, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578243262:578245718, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168112 fvup0 In  IP (tos 0x0, ttl 64, id 5180, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0xb121), ack 578243262, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168114 fvmb0 Out IP (tos 0x0, ttl 63, id 5180, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0x0d6a), ack 578243262, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168114 fvmb0 In  IP (tos 0x0, ttl 64, id 2363, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x176d (correct), ack 3226304471, win 62, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168116 fvup0 Out IP (tos 0x0, ttl 63, id 2363, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xbb24 (correct), ack 3226304471, win 62, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168121 fvup0 In  IP (tos 0x0, ttl 64, id 5181, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226309501:3226310507, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168123 fvmb0 Out IP (tos 0x0, ttl 63, id 5181, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226309501:3226310507, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168126 fvmb0 In  IP (tos 0x0, ttl 64, id 2364, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x0ba5 (correct), ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168128 fvup0 Out IP (tos 0x0, ttl 63, id 2364, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xaf5c (correct), ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168130 fvup0 In  IP (tos 0x0, ttl 64, id 5182, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226310507:3226311513, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168131 fvmb0 Out IP (tos 0x0, ttl 63, id 5182, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226310507:3226311513, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168142 fvup0 In  IP (tos 0x0, ttl 64, id 5183, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226311513:3226312519, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168143 fvmb0 Out IP (tos 0x0, ttl 63, id 5183, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226311513:3226312519, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168150 fvup0 In  IP (tos 0x0, ttl 64, id 5184, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226312519:3226313525, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168151 fvmb0 Out IP (tos 0x0, ttl 63, id 5184, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226312519:3226313525, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168159 fvmb0 In  IP (tos 0x0, ttl 64, id 2365, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578245718:578248174, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168159 fvup0 In  IP (tos 0x0, ttl 64, id 5185, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226313525:3226314531, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168161 fvup0 Out IP (tos 0x0, ttl 63, id 2365, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578245718:578248174, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168162 fvmb0 Out IP (tos 0x0, ttl 63, id 5185, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226313525:3226314531, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168166 fvmb0 In  IP (tos 0x0, ttl 64, id 2367, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578248174:578250630, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168168 fvup0 Out IP (tos 0x0, ttl 63, id 2367, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578248174:578250630, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168168 fvup0 In  IP (tos 0x0, ttl 64, id 5186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226314531:3226315537, ack 578248174, win 50, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168170 fvmb0 In  IP (tos 0x0, ttl 64, id 2369, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578250630:578251858, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168170 fvmb0 Out IP (tos 0x0, ttl 63, id 5186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226314531:3226315537, ack 578248174, win 50, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168172 fvup0 Out IP (tos 0x0, ttl 63, id 2369, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578250630:578251858, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168175 fvmb0 In  IP (tos 0x0, ttl 64, id 2370, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578251858:578253086, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168177 fvup0 In  IP (tos 0x0, ttl 64, id 5187, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x7cc8), ack 578250630, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168177 fvup0 Out IP (tos 0x0, ttl 63, id 2370, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578251858:578253086, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168180 fvmb0 Out IP (tos 0x0, ttl 63, id 5187, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0xd910), ack 578250630, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168181 fvmb0 In  IP (tos 0x0, ttl 64, id 2371, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578253086:578255542, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168182 fvup0 Out IP (tos 0x0, ttl 63, id 2371, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578253086:578255542, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168185 fvmb0 In  IP (tos 0x0, ttl 64, id 2373, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578255542:578256770, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168187 fvup0 Out IP (tos 0x0, ttl 63, id 2373, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578255542:578256770, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168187 fvup0 In  IP (tos 0x0, ttl 64, id 5188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226315537:3226316543, ack 578255542, win 44, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168188 fvmb0 Out IP (tos 0x0, ttl 63, id 5188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226315537:3226316543, ack 578255542, win 44, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168190 fvmb0 In  IP (tos 0x0, ttl 64, id 2374, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578256770:578259226, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168192 fvup0 Out IP (tos 0x0, ttl 63, id 2374, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578256770:578259226, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168194 fvmb0 In  IP (tos 0x0, ttl 64, id 2376, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578259226:578260454, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168195 fvup0 Out IP (tos 0x0, ttl 63, id 2376, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578259226:578260454, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168196 fvup0 In  IP (tos 0x0, ttl 64, id 5189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226316543:3226317549, ack 578259226, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168198 fvmb0 In  IP (tos 0x0, ttl 64, id 2377, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578260454:578261682, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168199 fvmb0 Out IP (tos 0x0, ttl 63, id 5189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226316543:3226317549, ack 578259226, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168199 fvup0 Out IP (tos 0x0, ttl 63, id 2377, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578260454:578261682, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168204 fvmb0 In  IP (tos 0x0, ttl 64, id 2378, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578261682:578262910, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168204 fvup0 In  IP (tos 0x0, ttl 64, id 5190, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x49c7), ack 578261682, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168206 fvup0 Out IP (tos 0x0, ttl 63, id 2378, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578261682:578262910, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168206 fvmb0 Out IP (tos 0x0, ttl 63, id 5190, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0xa60f), ack 578261682, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168210 fvmb0 In  IP (tos 0x0, ttl 64, id 2379, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578262910:578265366, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168211 fvup0 Out IP (tos 0x0, ttl 63, id 2379, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578262910:578265366, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168212 fvup0 In  IP (tos 0x0, ttl 64, id 5191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226317549:3226318555, ack 578262910, win 40, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168213 fvmb0 In  IP (tos 0x0, ttl 64, id 2381, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578265366:578266594, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168214 fvmb0 Out IP (tos 0x0, ttl 63, id 5191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226317549:3226318555, ack 578262910, win 40, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168216 fvup0 Out IP (tos 0x0, ttl 63, id 2381, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578265366:578266594, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168218 fvmb0 In  IP (tos 0x0, ttl 64, id 2382, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578266594:578267822, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168220 fvup0 Out IP (tos 0x0, ttl 63, id 2382, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578266594:578267822, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168222 fvup0 In  IP (tos 0x0, ttl 64, id 5192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226318555:3226319561, ack 578266594, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168222 fvmb0 In  IP (tos 0x0, ttl 64, id 2383, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578267822:578269050, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168224 fvup0 Out IP (tos 0x0, ttl 63, id 2383, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578267822:578269050, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168225 fvmb0 Out IP (tos 0x0, ttl 63, id 5192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226318555:3226319561, ack 578266594, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168227 fvmb0 In  IP (tos 0x0, ttl 64, id 2384, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578269050:578271506, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168228 fvup0 Out IP (tos 0x0, ttl 63, id 2384, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578269050:578271506, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168230 fvup0 In  IP (tos 0x0, ttl 64, id 5193, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x2529), ack 578269050, win 35, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168231 fvmb0 In  IP (tos 0x0, ttl 64, id 2386, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578271506:578272734, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168232 fvmb0 Out IP (tos 0x0, ttl 63, id 5193, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0x8171), ack 578269050, win 35, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168233 fvup0 Out IP (tos 0x0, ttl 63, id 2386, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578271506:578272734, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168237 fvmb0 In  IP (tos 0x0, ttl 64, id 2387, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578272734:578273962, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168239 fvup0 Out IP (tos 0x0, ttl 63, id 2387, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578272734:578273962, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168240 fvup0 In  IP (tos 0x0, ttl 64, id 5194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226319561:3226320567, ack 578272734, win 32, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168241 fvmb0 Out IP (tos 0x0, ttl 63, id 5194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226319561:3226320567, ack 578272734, win 32, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168247 fvmb0 In  IP (tos 0x0, ttl 64, id 2388, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578273962:578277646, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 3684
1793457036.168249 fvup0 Out IP (tos 0x0, ttl 63, id 2388, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578273962:578277646, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 3684
1793457036.168251 fvmb0 In  IP (tos 0x0, ttl 64, id 2391, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578277646:578278874, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168252 fvup0 In  IP (tos 0x0, ttl 64, id 5195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226320567:3226321573, ack 578273962, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168254 fvup0 Out IP (tos 0x0, ttl 63, id 2391, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578277646:578278874, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168255 fvmb0 Out IP (tos 0x0, ttl 63, id 5195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226320567:3226321573, ack 578273962, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168256 fvmb0 In  IP (tos 0x0, ttl 64, id 2392, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], seq 578278874:578280102, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168258 fvup0 Out IP (tos 0x0, ttl 63, id 2392, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578278874:578280102, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168260 fvup0 In  IP (tos 0x0, ttl 64, id 5196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0xf6f3), ack 578278874, win 28, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168262 fvmb0 Out IP (tos 0x0, ttl 63, id 5196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [.], cksum 0x29fd (incorrect -> 0x533c), ack 578278874, win 28, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168262 fvmb0 In  IP (tos 0x0, ttl 64, id 2393, offset 0, flags [DF], proto TCP (6), length 260)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [P.], seq 578280102:578280310, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 208
1793457036.168264 fvup0 Out IP (tos 0x0, ttl 63, id 2393, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578280102:578280310, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 208
1793457036.168266 fvmb0 In  IP (tos 0x0, ttl 64, id 2394, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x70e0 (correct), ack 3226312519, win 58, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168267 fvup0 In  IP (tos 0x0, ttl 64, id 5197, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226321573:3226322579, ack 578280102, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168267 fvup0 Out IP (tos 0x0, ttl 63, id 2394, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x1498 (correct), ack 3226312519, win 58, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168268 fvmb0 Out IP (tos 0x0, ttl 63, id 5197, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226321573:3226322579, ack 578280102, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168269 fvmb0 In  IP (tos 0x0, ttl 64, id 2395, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x6905 (correct), ack 3226314531, win 57, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168271 fvup0 Out IP (tos 0x0, ttl 63, id 2395, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x0cbd (correct), ack 3226314531, win 57, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168276 fvmb0 In  IP (tos 0x0, ttl 64, id 2396, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x612a (correct), ack 3226316543, win 56, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168276 fvmb0 In  IP (tos 0x0, ttl 64, id 2397, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x5562 (correct), ack 3226319561, win 54, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168277 fvup0 In  IP (tos 0x0, ttl 64, id 5198, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226322579:3226323585, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168278 fvup0 Out IP (tos 0x0, ttl 63, id 2396, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x04e2 (correct), ack 3226316543, win 56, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168278 fvmb0 Out IP (tos 0x0, ttl 63, id 5198, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226322579:3226323585, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168279 fvup0 Out IP (tos 0x0, ttl 63, id 2397, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xf919 (correct), ack 3226319561, win 54, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168288 fvup0 In  IP (tos 0x0, ttl 64, id 5199, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226323585:3226324591, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168289 fvmb0 Out IP (tos 0x0, ttl 63, id 5199, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226323585:3226324591, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168290 fvmb0 In  IP (tos 0x0, ttl 64, id 2398, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x4d87 (correct), ack 3226321573, win 53, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168291 fvup0 Out IP (tos 0x0, ttl 63, id 2398, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xf13e (correct), ack 3226321573, win 53, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168296 fvup0 In  IP (tos 0x0, ttl 64, id 5200, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226324591:3226325597, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168297 fvmb0 Out IP (tos 0x0, ttl 63, id 5200, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226324591:3226325597, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168305 fvup0 In  IP (tos 0x0, ttl 64, id 5201, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226325597:3226326603, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168307 fvmb0 Out IP (tos 0x0, ttl 63, id 5201, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226325597:3226326603, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168312 fvup0 In  IP (tos 0x0, ttl 64, id 5202, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226326603:3226327609, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168313 fvmb0 In  IP (tos 0x0, ttl 64, id 2399, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x45ac (correct), ack 3226323585, win 52, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168315 fvmb0 Out IP (tos 0x0, ttl 63, id 5202, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226326603:3226327609, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168315 fvup0 Out IP (tos 0x0, ttl 63, id 2399, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xe963 (correct), ack 3226323585, win 52, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168323 fvup0 In  IP (tos 0x0, ttl 64, id 5203, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226327609:3226328615, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168324 fvmb0 Out IP (tos 0x0, ttl 63, id 5203, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226327609:3226328615, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168331 fvup0 In  IP (tos 0x0, ttl 64, id 5204, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226328615:3226329621, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168332 fvmb0 In  IP (tos 0x0, ttl 64, id 2400, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x3dd1 (correct), ack 3226325597, win 51, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168333 fvmb0 Out IP (tos 0x0, ttl 63, id 5204, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226328615:3226329621, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168340 fvup0 In  IP (tos 0x0, ttl 64, id 5205, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226329621:3226330627, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168340 fvup0 Out IP (tos 0x0, ttl 63, id 2400, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xe188 (correct), ack 3226325597, win 51, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168342 fvmb0 Out IP (tos 0x0, ttl 63, id 5205, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226329621:3226330627, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168347 fvmb0 In  IP (tos 0x0, ttl 64, id 2401, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x35f6 (correct), ack 3226327609, win 50, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168348 fvup0 In  IP (tos 0x0, ttl 64, id 5206, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226330627:3226331633, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168350 fvup0 Out IP (tos 0x0, ttl 63, id 2401, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xd9ad (correct), ack 3226327609, win 50, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168350 fvmb0 Out IP (tos 0x0, ttl 63, id 5206, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226330627:3226331633, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168358 fvup0 In  IP (tos 0x0, ttl 64, id 5207, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226331633:3226332639, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168360 fvmb0 Out IP (tos 0x0, ttl 63, id 5207, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226331633:3226332639, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168365 fvup0 In  IP (tos 0x0, ttl 64, id 5208, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226332639:3226333645, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168367 fvmb0 Out IP (tos 0x0, ttl 63, id 5208, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226332639:3226333645, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168374 fvmb0 In  IP (tos 0x0, ttl 64, id 2402, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x2e1b (correct), ack 3226329621, win 49, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168376 fvup0 Out IP (tos 0x0, ttl 63, id 2402, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xd1d2 (correct), ack 3226329621, win 49, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168377 fvup0 In  IP (tos 0x0, ttl 64, id 5209, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226333645:3226334651, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168379 fvmb0 Out IP (tos 0x0, ttl 63, id 5209, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226333645:3226334651, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168387 fvup0 In  IP (tos 0x0, ttl 64, id 5210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226334651:3226335657, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168388 fvmb0 Out IP (tos 0x0, ttl 63, id 5210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226334651:3226335657, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168400 fvup0 In  IP (tos 0x0, ttl 64, id 5211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226335657:3226336663, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168402 fvmb0 Out IP (tos 0x0, ttl 63, id 5211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226335657:3226336663, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168405 fvmb0 In  IP (tos 0x0, ttl 64, id 2403, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x2253 (correct), ack 3226332639, win 47, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168407 fvup0 Out IP (tos 0x0, ttl 63, id 2403, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xc60a (correct), ack 3226332639, win 47, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168409 fvup0 In  IP (tos 0x0, ttl 64, id 5212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226336663:3226337669, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168411 fvmb0 Out IP (tos 0x0, ttl 63, id 5212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226336663:3226337669, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168418 fvup0 In  IP (tos 0x0, ttl 64, id 5213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226337669:3226338675, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168419 fvmb0 Out IP (tos 0x0, ttl 63, id 5213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226337669:3226338675, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168430 fvup0 In  IP (tos 0x0, ttl 64, id 5214, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226338675:3226339681, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168432 fvmb0 Out IP (tos 0x0, ttl 63, id 5214, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226338675:3226339681, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168441 fvup0 In  IP (tos 0x0, ttl 64, id 5215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226339681:3226340687, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168444 fvmb0 Out IP (tos 0x0, ttl 63, id 5215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226339681:3226340687, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168446 fvmb0 In  IP (tos 0x0, ttl 64, id 2404, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x168b (correct), ack 3226335657, win 45, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168447 fvmb0 In  IP (tos 0x0, ttl 64, id 2405, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x0eb0 (correct), ack 3226337669, win 44, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168449 fvup0 Out IP (tos 0x0, ttl 63, id 2404, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xba42 (correct), ack 3226335657, win 45, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168450 fvup0 In  IP (tos 0x0, ttl 64, id 5216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226340687:3226341693, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168450 fvup0 Out IP (tos 0x0, ttl 63, id 2405, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xb267 (correct), ack 3226337669, win 44, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168452 fvmb0 Out IP (tos 0x0, ttl 63, id 5216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226340687:3226341693, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168460 fvup0 In  IP (tos 0x0, ttl 64, id 5217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226341693:3226342699, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168461 fvmb0 Out IP (tos 0x0, ttl 63, id 5217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226341693:3226342699, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168468 fvup0 In  IP (tos 0x0, ttl 64, id 5218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226342699:3226343705, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168469 fvmb0 In  IP (tos 0x0, ttl 64, id 2406, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0x06d5 (correct), ack 3226339681, win 43, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168470 fvmb0 Out IP (tos 0x0, ttl 63, id 5218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226342699:3226343705, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168472 fvup0 Out IP (tos 0x0, ttl 63, id 2406, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xaa8c (correct), ack 3226339681, win 43, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168477 fvup0 In  IP (tos 0x0, ttl 64, id 5219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226343705:3226344711, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168479 fvmb0 Out IP (tos 0x0, ttl 63, id 5219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226343705:3226344711, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168485 fvup0 In  IP (tos 0x0, ttl 64, id 5220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226344711:3226345717, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168486 fvmb0 Out IP (tos 0x0, ttl 63, id 5220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226344711:3226345717, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168492 fvup0 In  IP (tos 0x0, ttl 64, id 5221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226345717:3226346723, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168494 fvmb0 Out IP (tos 0x0, ttl 63, id 5221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226345717:3226346723, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168501 fvup0 In  IP (tos 0x0, ttl 64, id 5222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226346723:3226347729, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168502 fvmb0 Out IP (tos 0x0, ttl 63, id 5222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226346723:3226347729, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168507 fvup0 In  IP (tos 0x0, ttl 64, id 5223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226347729:3226348735, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168509 fvmb0 Out IP (tos 0x0, ttl 63, id 5223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226347729:3226348735, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168516 fvup0 In  IP (tos 0x0, ttl 64, id 5224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226348735:3226349741, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168518 fvmb0 Out IP (tos 0x0, ttl 63, id 5224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226348735:3226349741, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168524 fvup0 In  IP (tos 0x0, ttl 64, id 5225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226349741:3226350747, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168525 fvmb0 Out IP (tos 0x0, ttl 63, id 5225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226349741:3226350747, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168527 fvmb0 In  IP (tos 0x0, ttl 64, id 2407, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xf71f (correct), ack 3226343705, win 40, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168527 fvmb0 In  IP (tos 0x0, ttl 64, id 2408, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xeb56 (correct), ack 3226346723, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168529 fvup0 Out IP (tos 0x0, ttl 63, id 2407, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x9ad7 (correct), ack 3226343705, win 40, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168530 fvup0 Out IP (tos 0x0, ttl 63, id 2408, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x8f0e (correct), ack 3226346723, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168533 fvup0 In  IP (tos 0x0, ttl 64, id 5226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226350747:3226351753, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168535 fvmb0 Out IP (tos 0x0, ttl 63, id 5226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226350747:3226351753, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168544 fvup0 In  IP (tos 0x0, ttl 64, id 5227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226351753:3226352759, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168545 fvmb0 Out IP (tos 0x0, ttl 63, id 5227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226351753:3226352759, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168552 fvup0 In  IP (tos 0x0, ttl 64, id 5228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226352759:3226353765, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168553 fvmb0 Out IP (tos 0x0, ttl 63, id 5228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226352759:3226353765, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168558 fvup0 In  IP (tos 0x0, ttl 64, id 5229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226353765:3226354771, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168560 fvmb0 Out IP (tos 0x0, ttl 63, id 5229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226353765:3226354771, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168566 fvup0 In  IP (tos 0x0, ttl 64, id 5230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226354771:3226355777, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168567 fvmb0 In  IP (tos 0x0, ttl 64, id 2409, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xdb9e (correct), ack 3226350747, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168568 fvmb0 Out IP (tos 0x0, ttl 63, id 5230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226354771:3226355777, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168569 fvup0 Out IP (tos 0x0, ttl 63, id 2409, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x7f56 (correct), ack 3226350747, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168574 fvup0 In  IP (tos 0x0, ttl 64, id 5231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226355777:3226356783, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168576 fvmb0 Out IP (tos 0x0, ttl 63, id 5231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226355777:3226356783, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168583 fvup0 In  IP (tos 0x0, ttl 64, id 5232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226356783:3226357789, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168585 fvmb0 Out IP (tos 0x0, ttl 63, id 5232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226356783:3226357789, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168587 fvmb0 In  IP (tos 0x0, ttl 64, id 2410, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xd3c2 (correct), ack 3226352759, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168588 fvup0 Out IP (tos 0x0, ttl 63, id 2410, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x777a (correct), ack 3226352759, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168590 fvup0 In  IP (tos 0x0, ttl 64, id 5233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226357789:3226358795, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168592 fvmb0 Out IP (tos 0x0, ttl 63, id 5233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226357789:3226358795, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168593 fvmb0 In  IP (tos 0x0, ttl 64, id 2411, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xcbe6 (correct), ack 3226354771, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168594 fvup0 Out IP (tos 0x0, ttl 63, id 2411, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x6f9e (correct), ack 3226354771, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168600 fvup0 In  IP (tos 0x0, ttl 64, id 5234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226358795:3226359801, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168601 fvmb0 Out IP (tos 0x0, ttl 63, id 5234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226358795:3226359801, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168607 fvup0 In  IP (tos 0x0, ttl 64, id 5235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226359801:3226360807, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168609 fvmb0 Out IP (tos 0x0, ttl 63, id 5235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226359801:3226360807, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168615 fvmb0 In  IP (tos 0x0, ttl 64, id 2412, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xc40a (correct), ack 3226356783, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168616 fvup0 In  IP (tos 0x0, ttl 64, id 5236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226360807:3226361813, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168617 fvup0 Out IP (tos 0x0, ttl 63, id 2412, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x67c2 (correct), ack 3226356783, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168618 fvmb0 Out IP (tos 0x0, ttl 63, id 5236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226360807:3226361813, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168624 fvup0 In  IP (tos 0x0, ttl 64, id 5237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226361813:3226362819, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168625 fvmb0 Out IP (tos 0x0, ttl 63, id 5237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226361813:3226362819, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168634 fvup0 In  IP (tos 0x0, ttl 64, id 5238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226362819:3226363825, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168635 fvmb0 Out IP (tos 0x0, ttl 63, id 5238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226362819:3226363825, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168641 fvup0 In  IP (tos 0x0, ttl 64, id 5239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226363825:3226364831, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168642 fvmb0 Out IP (tos 0x0, ttl 63, id 5239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226363825:3226364831, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168648 fvmb0 In  IP (tos 0x0, ttl 64, id 2413, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xbc2e (correct), ack 3226358795, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168648 fvmb0 In  IP (tos 0x0, ttl 64, id 2414, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xb451 (correct), ack 3226360807, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168650 fvup0 Out IP (tos 0x0, ttl 63, id 2413, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x5fe6 (correct), ack 3226358795, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168650 fvup0 In  IP (tos 0x0, ttl 64, id 5240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226364831:3226365837, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168651 fvup0 Out IP (tos 0x0, ttl 63, id 2414, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x5809 (correct), ack 3226360807, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168652 fvmb0 Out IP (tos 0x0, ttl 63, id 5240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [P.], seq 3226364831:3226365837, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168664 fvmb0 In  IP (tos 0x0, ttl 64, id 2415, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xac75 (correct), ack 3226362819, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168667 fvup0 Out IP (tos 0x0, ttl 63, id 2415, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x502d (correct), ack 3226362819, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168676 fvmb0 In  IP (tos 0x0, ttl 64, id 2416, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xa498 (correct), ack 3226364831, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168678 fvup0 Out IP (tos 0x0, ttl 63, id 2416, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x4850 (correct), ack 3226364831, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168831 fvmb0 In  IP (tos 0x0, ttl 64, id 2417, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [F.], cksum 0xa0a9 (correct), seq 578280310, ack 3226365837, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168833 fvup0 Out IP (tos 0x0, ttl 63, id 2417, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [F.], cksum 0x4461 (correct), seq 578280310, ack 3226365837, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168902 fvup0 In  IP (tos 0x0, ttl 64, id 5241, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [F.], cksum 0x8645 (incorrect -> 0x4447), seq 3226365837, ack 578280311, win 64, options [nop,nop,TS val 685186117 ecr 200485325], length 0
1793457036.168905 fvmb0 Out IP (tos 0x0, ttl 63, id 5241, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56968: Flags [F.], cksum 0x29fd (incorrect -> 0xa08f), seq 3226365837, ack 578280311, win 64, options [nop,nop,TS val 685186117 ecr 200485325], length 0
1793457036.168936 fvmb0 In  IP (tos 0x0, ttl 64, id 2418, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56968 > 192.0.3.10.9446: Flags [.], cksum 0xa0a8 (correct), ack 3226365838, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168938 fvup0 Out IP (tos 0x0, ttl 63, id 2418, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x4460 (correct), ack 3226365838, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0

- matrixDual_path_1: eth0
2
1500
eth2
4
1500
fvfixture
5
1500
fvmb0
6
1280
fvup0
3
1500
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4416,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:25:c1:29:8d:d8","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4417,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"c6:ee:52:4f:5a:4e","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4420,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:8a:b4:de:17:2f","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ae:1b:a1:20:96:73","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.78.12.82","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51782
	 cubic wscale:10,10 rto:201 rtt:0.344/0.66 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:23797 bytes_acked:23797 bytes_received:33124 segs_out:41 segs_in:66 data_segs_out:33 data_segs_in:33 send 10597209302bps lastsnd:4386 lastrcv:4392 lastack:4386 pacing_rate 21140645160bps delivery_rate 121514666664bps delivered:34 app_limited busy:94ms rcv_rtt:1 rcv_space:65483 rcv_ssthresh:105032 minrtt:0.003 snd_wnd:91136
ESTAB 0      0       172.28.30.40:45976 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.41/0.362 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5345 bytes_acked:5346 bytes_received:10702 segs_out:109 segs_in:82 data_segs_out:48 data_segs_in:75 send 282536585bps lastsnd:825 lastrcv:825 lastack:825 pacing_rate 564213088bps delivery_rate 3309714280bps delivered:49 app_limited busy:56ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45986 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.114/0.112 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:13544 bytes_acked:13545 bytes_received:16026 segs_out:117 segs_in:133 data_segs_out:66 data_segs_in:122 send 1016140351bps lastsnd:5689 lastrcv:5689 lastack:5689 pacing_rate 2032280696bps delivery_rate 1654857136bps delivered:67 app_limited busy:45ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45992 172.28.30.21:443
	 cubic wscale:10,10 rto:210 rtt:9.603/15.49 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2942 bytes_acked:2943 bytes_received:4304 segs_out:35 segs_in:34 data_segs_out:17 data_segs_in:21 send 12062897bps lastsnd:5690 lastrcv:5690 lastack:5690 pacing_rate 24125792bps delivery_rate 2896000000bps delivered:18 app_limited busy:168ms rcv_space:14480 rcv_ssthresh:85754 minrtt:0.008 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51796
	 cubic wscale:10,10 rto:203 rtt:2.308/4.59 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:9387 bytes_acked:9387 bytes_received:5962 segs_out:16 segs_in:22 data_segs_out:8 data_segs_in:9 send 1508492201bps lastsnd:19276 lastrcv:19281 lastack:4148 pacing_rate 3016004328bps delivery_rate 34816000000bps delivered:9 app_limited busy:41ms rcv_rtt:1245.38 rcv_space:65483 rcv_ssthresh:78471 minrtt:0.008 snd_wnd:83968
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:55868
	 cubic wscale:10,10 rto:206 rtt:5.079/10.134 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2779 bytes_acked:2779 bytes_received:3998 segs_out:6 segs_in:10 data_segs_out:2 data_segs_in:4 send 685489270bps lastsnd:19472 lastrcv:19484 lastack:4150 pacing_rate 1370877320bps delivery_rate 29013333328bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78342 minrtt:0.012 snd_wnd:87040
ESTAB 0      0          127.0.0.1:36206    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.764/0.077 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:22457 bytes_acked:22458 bytes_received:44442 segs_out:155 segs_in:117 data_segs_out:113 data_segs_in:113 send 3431204188bps lastsnd:4378 lastrcv:4377 lastack:4377 pacing_rate 6857920208bps delivery_rate 29127111104bps delivered:114 app_limited busy:137ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:36842 172.28.30.30:9443
	 cubic wscale:10,10 rto:205 rtt:4.989/3.045 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:40194 bytes_acked:40195 bytes_received:32828 segs_out:106 segs_in:63 data_segs_out:70 data_segs_in:43 send 23219082bps lastsnd:4400 lastrcv:4394 lastack:4394 pacing_rate 46435832bps delivery_rate 2574222216bps delivered:71 app_limited busy:171ms rcv_rtt:6 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.009 snd_wnd:79872
ESTAB 0      0          127.0.0.1:51782    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.653/4.365 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:33124 bytes_acked:33125 bytes_received:23797 segs_out:66 segs_in:42 data_segs_out:33 data_segs_in:33 send 634131971bps lastsnd:4400 lastrcv:4394 lastack:4394 pacing_rate 1268240112bps delivery_rate 38353454544bps delivered:34 app_limited busy:187ms rcv_rtt:12 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.004 snd_wnd:105472
ESTAB 0      0          127.0.0.1:51796    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.101/7.042 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5962 bytes_acked:5963 bytes_received:9387 segs_out:22 segs_in:16 data_segs_out:9 data_segs_in:8 send 516951319bps lastsnd:19286 lastrcv:19281 lastack:4154 pacing_rate 1033754376bps delivery_rate 28672000000bps delivered:10 app_limited busy:71ms rcv_rtt:5 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.01 snd_wnd:78848
ESTAB 0      0          127.0.0.1:55868    127.0.0.1:9444
	 cubic wscale:10,10 rto:202 rtt:1.6/3.153 ato:40 mss:39424 pmtu:65535 rcvmss:1996 advmss:65483 cwnd:10 bytes_sent:3998 bytes_acked:3999 bytes_received:2779 segs_out:10 segs_in:7 data_segs_out:4 data_segs_in:2 send 1971200000bps lastsnd:19487 lastrcv:19475 lastack:4153 pacing_rate 3941784096bps delivery_rate 28672000000bps delivered:5 app_limited busy:13ms rcv_space:65495 rcv_ssthresh:86892 minrtt:0.011 snd_wnd:78848
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:36206
	 cubic wscale:10,10 rto:211 rtt:10.498/16.728 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:44442 bytes_acked:44442 bytes_received:22457 segs_out:116 segs_in:155 data_segs_out:113 data_segs_in:113 send 249708516bps lastsnd:4380 lastrcv:4381 lastack:4339 pacing_rate 499393240bps delivery_rate 29127111104bps delivered:114 app_limited busy:832ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.008 snd_wnd:65536

- matrixDual_capture_1_receipt: {"bytes":"59238","sha256":"fc646c20836b9c279b8ca474e03153c44a212a1d3030587548b95f122004cd6d"}
- matrixDual_capture_2_window: {"launch_requested_at":"2026-10-31T14:30:34.938698402Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:37.026705966Z","stop_requested_at":"2026-10-31T14:30:36.949865226Z"}
- matrixDual_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
139 packets captured
139 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457036.166979 eth0  In  IP (tos 0x0, ttl 63, id 2335, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [S], cksum 0x8a2c (correct), seq 578216201, win 64480, options [mss 1240,sackOK,TS val 200485323 ecr 0,nop,wscale 10], length 0
1793457036.166994 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [S.], cksum 0x864d (incorrect -> 0x0d26), seq 3226301449, ack 578216202, win 65160, options [mss 1460,sackOK,TS val 685186115 ecr 200485323,nop,wscale 10], length 0
1793457036.167102 eth0  In  IP (tos 0x0, ttl 63, id 2336, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x3a3f (correct), ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 0
1793457036.167508 eth0  In  IP (tos 0x0, ttl 63, id 2337, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], cksum 0x4b63 (correct), seq 578216202:578216246, ack 3226301450, win 63, options [nop,nop,TS val 200485323 ecr 685186115], length 44
1793457036.167512 eth0  Out IP (tos 0x0, ttl 64, id 5168, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x3a12), ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 0
1793457036.167526 eth0  Out IP (tos 0x0, ttl 64, id 5169, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], cksum 0x8648 (incorrect -> 0xe0bb), seq 3226301450:3226301453, ack 578216246, win 64, options [nop,nop,TS val 685186115 ecr 200485323], length 3
1793457036.167844 eth0  In  IP (tos 0x0, ttl 63, id 2338, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x3a0f (correct), ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 0
1793457036.168000 eth0  In  IP (tos 0x0, ttl 63, id 2339, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578216246:578221158, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 4912
1793457036.168005 eth0  Out IP (tos 0x0, ttl 64, id 5170, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x26dd), ack 578221158, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168015 eth0  In  IP (tos 0x0, ttl 63, id 2343, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578221158:578222386, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 1228
1793457036.168016 eth0  In  IP (tos 0x0, ttl 63, id 2344, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578222386:578224842, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 2456
1793457036.168020 eth0  Out IP (tos 0x0, ttl 64, id 5171, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226301453:3226302459, ack 578224842, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168020 eth0  In  IP (tos 0x0, ttl 63, id 2346, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578224842:578227298, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 2456
1793457036.168032 eth0  Out IP (tos 0x0, ttl 64, id 5172, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226302459:3226303465, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168047 eth0  Out IP (tos 0x0, ttl 64, id 5173, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226303465:3226304471, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168057 eth0  Out IP (tos 0x0, ttl 64, id 5174, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226304471:3226305477, ack 578227298, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168070 eth0  Out IP (tos 0x0, ttl 64, id 5175, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226305477:3226306483, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168081 eth0  In  IP (tos 0x0, ttl 63, id 2348, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578227298:578228526, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186115], length 1228
1793457036.168081 eth0  Out IP (tos 0x0, ttl 64, id 5176, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226306483:3226307489, ack 578227298, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168082 eth0  In  IP (tos 0x0, ttl 63, id 2349, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578228526:578233438, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 4912
1793457036.168087 eth0  Out IP (tos 0x0, ttl 64, id 5177, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0xdf55), ack 578233438, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168088 eth0  In  IP (tos 0x0, ttl 63, id 2353, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578233438:578237122, ack 3226301453, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 3684
1793457036.168091 eth0  In  IP (tos 0x0, ttl 63, id 2356, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578237122:578238350, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168095 eth0  Out IP (tos 0x0, ttl 64, id 5178, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226307489:3226308495, ack 578238350, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168097 eth0  In  IP (tos 0x0, ttl 63, id 2357, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578238350:578239578, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168102 eth0  In  IP (tos 0x0, ttl 63, id 2358, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578239578:578242034, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168104 eth0  Out IP (tos 0x0, ttl 64, id 5179, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226308495:3226309501, ack 578239578, win 54, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168108 eth0  In  IP (tos 0x0, ttl 63, id 2360, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578242034:578243262, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168111 eth0  Out IP (tos 0x0, ttl 64, id 5180, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0xb121), ack 578243262, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168113 eth0  In  IP (tos 0x0, ttl 63, id 2361, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578243262:578245718, ack 3226302459, win 63, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168116 eth0  In  IP (tos 0x0, ttl 63, id 2363, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xbb24 (correct), ack 3226304471, win 62, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168120 eth0  Out IP (tos 0x0, ttl 64, id 5181, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226309501:3226310507, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168129 eth0  In  IP (tos 0x0, ttl 63, id 2364, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xaf5c (correct), ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168129 eth0  Out IP (tos 0x0, ttl 64, id 5182, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226310507:3226311513, ack 578245718, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168141 eth0  Out IP (tos 0x0, ttl 64, id 5183, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226311513:3226312519, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168149 eth0  Out IP (tos 0x0, ttl 64, id 5184, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226312519:3226313525, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168158 eth0  Out IP (tos 0x0, ttl 64, id 5185, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226313525:3226314531, ack 578245718, win 52, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168164 eth0  In  IP (tos 0x0, ttl 63, id 2365, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578245718:578248174, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168167 eth0  Out IP (tos 0x0, ttl 64, id 5186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226314531:3226315537, ack 578248174, win 50, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168169 eth0  In  IP (tos 0x0, ttl 63, id 2367, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578248174:578250630, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168174 eth0  In  IP (tos 0x0, ttl 63, id 2369, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578250630:578251858, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168175 eth0  Out IP (tos 0x0, ttl 64, id 5187, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x7cc8), ack 578250630, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168179 eth0  In  IP (tos 0x0, ttl 63, id 2370, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578251858:578253086, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168183 eth0  In  IP (tos 0x0, ttl 63, id 2371, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578253086:578255542, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168186 eth0  Out IP (tos 0x0, ttl 64, id 5188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226315537:3226316543, ack 578255542, win 44, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168187 eth0  In  IP (tos 0x0, ttl 63, id 2373, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578255542:578256770, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168192 eth0  In  IP (tos 0x0, ttl 63, id 2374, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578256770:578259226, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168195 eth0  Out IP (tos 0x0, ttl 64, id 5189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226316543:3226317549, ack 578259226, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168197 eth0  In  IP (tos 0x0, ttl 63, id 2376, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578259226:578260454, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168200 eth0  In  IP (tos 0x0, ttl 63, id 2377, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578260454:578261682, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168203 eth0  Out IP (tos 0x0, ttl 64, id 5190, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x49c7), ack 578261682, win 41, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168208 eth0  In  IP (tos 0x0, ttl 63, id 2378, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578261682:578262910, ack 3226307489, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168211 eth0  Out IP (tos 0x0, ttl 64, id 5191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226317549:3226318555, ack 578262910, win 40, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168212 eth0  In  IP (tos 0x0, ttl 63, id 2379, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578262910:578265366, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168217 eth0  In  IP (tos 0x0, ttl 63, id 2381, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578265366:578266594, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168221 eth0  In  IP (tos 0x0, ttl 63, id 2382, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578266594:578267822, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168221 eth0  Out IP (tos 0x0, ttl 64, id 5192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226318555:3226319561, ack 578266594, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168225 eth0  In  IP (tos 0x0, ttl 63, id 2383, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578267822:578269050, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168229 eth0  In  IP (tos 0x0, ttl 63, id 2384, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578269050:578271506, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 2456
1793457036.168230 eth0  Out IP (tos 0x0, ttl 64, id 5193, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0x2529), ack 578269050, win 35, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168235 eth0  In  IP (tos 0x0, ttl 63, id 2386, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578271506:578272734, ack 3226308495, win 60, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168239 eth0  Out IP (tos 0x0, ttl 64, id 5194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226319561:3226320567, ack 578272734, win 32, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168240 eth0  In  IP (tos 0x0, ttl 63, id 2387, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578272734:578273962, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168250 eth0  In  IP (tos 0x0, ttl 63, id 2388, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578273962:578277646, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 3684
1793457036.168251 eth0  Out IP (tos 0x0, ttl 64, id 5195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226320567:3226321573, ack 578273962, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168255 eth0  In  IP (tos 0x0, ttl 63, id 2391, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578277646:578278874, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168259 eth0  Out IP (tos 0x0, ttl 64, id 5196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [.], cksum 0x8645 (incorrect -> 0xf6f3), ack 578278874, win 28, options [nop,nop,TS val 685186116 ecr 200485324], length 0
1793457036.168261 eth0  In  IP (tos 0x0, ttl 63, id 2392, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], seq 578278874:578280102, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 1228
1793457036.168264 eth0  In  IP (tos 0x0, ttl 63, id 2393, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [P.], seq 578280102:578280310, ack 3226310507, win 59, options [nop,nop,TS val 200485324 ecr 685186116], length 208
1793457036.168266 eth0  Out IP (tos 0x0, ttl 64, id 5197, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226321573:3226322579, ack 578280102, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168268 eth0  In  IP (tos 0x0, ttl 63, id 2394, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x1498 (correct), ack 3226312519, win 58, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168272 eth0  In  IP (tos 0x0, ttl 63, id 2395, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x0cbd (correct), ack 3226314531, win 57, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168276 eth0  Out IP (tos 0x0, ttl 64, id 5198, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226322579:3226323585, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168280 eth0  In  IP (tos 0x0, ttl 63, id 2396, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x04e2 (correct), ack 3226316543, win 56, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168281 eth0  In  IP (tos 0x0, ttl 63, id 2397, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xf919 (correct), ack 3226319561, win 54, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168287 eth0  Out IP (tos 0x0, ttl 64, id 5199, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226323585:3226324591, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168293 eth0  In  IP (tos 0x0, ttl 63, id 2398, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xf13e (correct), ack 3226321573, win 53, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168295 eth0  Out IP (tos 0x0, ttl 64, id 5200, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226324591:3226325597, ack 578280310, win 27, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168304 eth0  Out IP (tos 0x0, ttl 64, id 5201, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226325597:3226326603, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168312 eth0  Out IP (tos 0x0, ttl 64, id 5202, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226326603:3226327609, ack 578280310, win 30, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168316 eth0  In  IP (tos 0x0, ttl 63, id 2399, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xe963 (correct), ack 3226323585, win 52, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168322 eth0  Out IP (tos 0x0, ttl 64, id 5203, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226327609:3226328615, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168330 eth0  Out IP (tos 0x0, ttl 64, id 5204, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226328615:3226329621, ack 578280310, win 31, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168338 eth0  Out IP (tos 0x0, ttl 64, id 5205, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226329621:3226330627, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168344 eth0  In  IP (tos 0x0, ttl 63, id 2400, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xe188 (correct), ack 3226325597, win 51, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168348 eth0  Out IP (tos 0x0, ttl 64, id 5206, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226330627:3226331633, ack 578280310, win 33, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168351 eth0  In  IP (tos 0x0, ttl 63, id 2401, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xd9ad (correct), ack 3226327609, win 50, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168357 eth0  Out IP (tos 0x0, ttl 64, id 5207, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226331633:3226332639, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168364 eth0  Out IP (tos 0x0, ttl 64, id 5208, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226332639:3226333645, ack 578280310, win 37, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168376 eth0  Out IP (tos 0x0, ttl 64, id 5209, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226333645:3226334651, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168378 eth0  In  IP (tos 0x0, ttl 63, id 2402, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xd1d2 (correct), ack 3226329621, win 49, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168386 eth0  Out IP (tos 0x0, ttl 64, id 5210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226334651:3226335657, ack 578280310, win 39, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168399 eth0  Out IP (tos 0x0, ttl 64, id 5211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226335657:3226336663, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168408 eth0  In  IP (tos 0x0, ttl 63, id 2403, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xc60a (correct), ack 3226332639, win 47, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168408 eth0  Out IP (tos 0x0, ttl 64, id 5212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226336663:3226337669, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168417 eth0  Out IP (tos 0x0, ttl 64, id 5213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226337669:3226338675, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168429 eth0  Out IP (tos 0x0, ttl 64, id 5214, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226338675:3226339681, ack 578280310, win 42, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168440 eth0  Out IP (tos 0x0, ttl 64, id 5215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226339681:3226340687, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168449 eth0  Out IP (tos 0x0, ttl 64, id 5216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226340687:3226341693, ack 578280310, win 45, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168451 eth0  In  IP (tos 0x0, ttl 63, id 2404, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xba42 (correct), ack 3226335657, win 45, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168452 eth0  In  IP (tos 0x0, ttl 63, id 2405, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xb267 (correct), ack 3226337669, win 44, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168459 eth0  Out IP (tos 0x0, ttl 64, id 5217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226341693:3226342699, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168467 eth0  Out IP (tos 0x0, ttl 64, id 5218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226342699:3226343705, ack 578280310, win 48, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168472 eth0  In  IP (tos 0x0, ttl 63, id 2406, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0xaa8c (correct), ack 3226339681, win 43, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168476 eth0  Out IP (tos 0x0, ttl 64, id 5219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226343705:3226344711, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168484 eth0  Out IP (tos 0x0, ttl 64, id 5220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226344711:3226345717, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168491 eth0  Out IP (tos 0x0, ttl 64, id 5221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226345717:3226346723, ack 578280310, win 49, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168500 eth0  Out IP (tos 0x0, ttl 64, id 5222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226346723:3226347729, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168506 eth0  Out IP (tos 0x0, ttl 64, id 5223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226347729:3226348735, ack 578280310, win 51, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168515 eth0  Out IP (tos 0x0, ttl 64, id 5224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226348735:3226349741, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168523 eth0  Out IP (tos 0x0, ttl 64, id 5225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226349741:3226350747, ack 578280310, win 53, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168532 eth0  Out IP (tos 0x0, ttl 64, id 5226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226350747:3226351753, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168535 eth0  In  IP (tos 0x0, ttl 63, id 2407, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x9ad7 (correct), ack 3226343705, win 40, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168536 eth0  In  IP (tos 0x0, ttl 63, id 2408, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x8f0e (correct), ack 3226346723, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168543 eth0  Out IP (tos 0x0, ttl 64, id 5227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226351753:3226352759, ack 578280310, win 55, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168551 eth0  Out IP (tos 0x0, ttl 64, id 5228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226352759:3226353765, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168557 eth0  Out IP (tos 0x0, ttl 64, id 5229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226353765:3226354771, ack 578280310, win 57, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168565 eth0  Out IP (tos 0x0, ttl 64, id 5230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226354771:3226355777, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168571 eth0  In  IP (tos 0x0, ttl 63, id 2409, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x7f56 (correct), ack 3226350747, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168574 eth0  Out IP (tos 0x0, ttl 64, id 5231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226355777:3226356783, ack 578280310, win 59, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168582 eth0  Out IP (tos 0x0, ttl 64, id 5232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226356783:3226357789, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168589 eth0  Out IP (tos 0x0, ttl 64, id 5233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226357789:3226358795, ack 578280310, win 61, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168589 eth0  In  IP (tos 0x0, ttl 63, id 2410, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x777a (correct), ack 3226352759, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168595 eth0  In  IP (tos 0x0, ttl 63, id 2411, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x6f9e (correct), ack 3226354771, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168599 eth0  Out IP (tos 0x0, ttl 64, id 5234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226358795:3226359801, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168606 eth0  Out IP (tos 0x0, ttl 64, id 5235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226359801:3226360807, ack 578280310, win 62, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168615 eth0  Out IP (tos 0x0, ttl 64, id 5236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226360807:3226361813, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168618 eth0  In  IP (tos 0x0, ttl 63, id 2412, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x67c2 (correct), ack 3226356783, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168623 eth0  Out IP (tos 0x0, ttl 64, id 5237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226361813:3226362819, ack 578280310, win 64, options [nop,nop,TS val 685186116 ecr 200485324], length 1006
1793457036.168633 eth0  Out IP (tos 0x0, ttl 64, id 5238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226362819:3226363825, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168640 eth0  Out IP (tos 0x0, ttl 64, id 5239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226363825:3226364831, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168649 eth0  Out IP (tos 0x0, ttl 64, id 5240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [P.], seq 3226364831:3226365837, ack 578280310, win 64, options [nop,nop,TS val 685186117 ecr 200485324], length 1006
1793457036.168653 eth0  In  IP (tos 0x0, ttl 63, id 2413, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x5fe6 (correct), ack 3226358795, win 39, options [nop,nop,TS val 200485324 ecr 685186116], length 0
1793457036.168655 eth0  In  IP (tos 0x0, ttl 63, id 2414, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x5809 (correct), ack 3226360807, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168667 eth0  In  IP (tos 0x0, ttl 63, id 2415, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x502d (correct), ack 3226362819, win 39, options [nop,nop,TS val 200485325 ecr 685186116], length 0
1793457036.168678 eth0  In  IP (tos 0x0, ttl 63, id 2416, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x4850 (correct), ack 3226364831, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168834 eth0  In  IP (tos 0x0, ttl 63, id 2417, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [F.], cksum 0x4461 (correct), seq 578280310, ack 3226365837, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0
1793457036.168901 eth0  Out IP (tos 0x0, ttl 64, id 5241, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56968: Flags [F.], cksum 0x8645 (incorrect -> 0x4447), seq 3226365837, ack 578280311, win 64, options [nop,nop,TS val 685186117 ecr 200485325], length 0
1793457036.168939 eth0  In  IP (tos 0x0, ttl 63, id 2418, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56968 > 192.0.3.10.9446: Flags [.], cksum 0x4460 (correct), ack 3226365838, win 39, options [nop,nop,TS val 200485325 ecr 685186117], length 0

- matrixDual_path_2: eth0
2
1500
lo
1
65536
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ss: not found

- matrixDual_capture_2_receipt: {"bytes":"17751","sha256":"900ac8a74938839c337a2590810fd66dad662634b41b0017a0c1236d2fddf0dd"}
- matrixDual_origin_endpoint_samples: {"at":"2026-10-31T14:30:22.307266462Z","local_port":9446,"remote_port":42202,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":86,"sacked":0,"snd_cwnd":25,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:30:29.185133874Z","local_port":9446,"remote_port":40370,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":63,"sacked":0,"snd_cwnd":26,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:30:36.168846529Z","local_port":9446,"remote_port":56968,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":82,"sacked":0,"snd_cwnd":28,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}

- matrix1280_input: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix1280_transfer: {"after":{"upload_bytes":"271489","download_bytes":"274499","Present":true},"before":{"upload_bytes":"204045","download_bytes":"206256","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:30:43.149892796Z","local_port":56984,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":178,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- matrix1280_ipv6_block: {"native_member":{"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":0,"error":"connect_unavailable"},"positive_control":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0},"scope":"D-123: IPv6 connection must fail; IPv4 bulk must complete. Native refusal may precede tunnel ingress; this does not claim ctv6 was hit. Kernel ctv6 proof is the separate netlab row."}
- matrix1280_capture_0_window: {"launch_requested_at":"2026-10-31T14:30:41.872850679Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:43.39721878Z","stop_requested_at":"2026-10-31T14:30:43.331537022Z"}
- matrix1280_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
337 packets captured
341 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457043.148009 fvnb0 Out IP (tos 0x0, ttl 64, id 14875, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [S], cksum 0x2a05 (incorrect -> 0x962f), seq 1485552438, win 64480, options [mss 1240,sackOK,TS val 2154689859 ecr 0,nop,wscale 10], length 0
1793457043.148319 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [S.], cksum 0xed07 (correct), seq 2577740037, ack 1485552439, win 65160, options [mss 1460,sackOK,TS val 2952145649 ecr 2154689859,nop,wscale 10], length 0
1793457043.148353 fvnb0 Out IP (tos 0x0, ttl 64, id 14876, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x1a21), ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148400 fvnb0 Out IP (tos 0x0, ttl 64, id 14877, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], cksum 0x2a29 (incorrect -> 0x2b45), seq 1485552439:1485552483, ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 44
1793457043.148513 fvnb0 In  IP (tos 0x0, ttl 63, id 31836, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x19f4 (correct), ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 0
1793457043.148521 fvnb0 In  IP (tos 0x0, ttl 63, id 31837, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], cksum 0xc09d (correct), seq 2577740038:2577740041, ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 3
1793457043.148530 fvnb0 Out IP (tos 0x0, ttl 64, id 14878, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x19f2), ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148595 fvnb0 Out IP (tos 0x0, ttl 64, id 14879, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485552483:1485558623, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 6140
1793457043.148610 fvnb0 Out IP (tos 0x0, ttl 64, id 14884, offset 0, flags [DF], proto TCP (6), length 6192)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485558623:1485564763, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 6140
1793457043.148746 fvnb0 In  IP (tos 0x0, ttl 63, id 31838, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x01f4 (correct), ack 1485558623, win 64, options [nop,nop,TS val 2952145650 ecr 2154689859], length 0
1793457043.148751 fvnb0 Out IP (tos 0x0, ttl 64, id 14889, offset 0, flags [DF], proto TCP (6), length 8648)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485564763:1485573359, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 8596
1793457043.148759 fvnb0 Out IP (tos 0x0, ttl 64, id 14896, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485573359:1485577043, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148781 fvnb0 In  IP (tos 0x0, ttl 63, id 31839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577740041:2577741047, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148787 fvnb0 Out IP (tos 0x0, ttl 64, id 14899, offset 0, flags [DF], proto TCP (6), length 12332)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485577043:1485589323, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 12280
1793457043.148805 fvnb0 In  IP (tos 0x0, ttl 63, id 31840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577741047:2577742053, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148813 fvnb0 In  IP (tos 0x0, ttl 63, id 31841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577742053:2577743059, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148820 fvnb0 Out IP (tos 0x0, ttl 64, id 14909, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x7e3e), ack 2577743059, win 62, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148833 fvnb0 In  IP (tos 0x0, ttl 63, id 31842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577743059:2577744065, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148849 fvnb0 In  IP (tos 0x0, ttl 63, id 31843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577744065:2577745071, ack 1485564763, win 63, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148856 fvnb0 Out IP (tos 0x0, ttl 64, id 14910, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x7663), ack 2577745071, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148867 fvnb0 In  IP (tos 0x0, ttl 63, id 31844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577745071:2577746077, ack 1485567219, win 61, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148875 fvnb0 Out IP (tos 0x0, ttl 64, id 14911, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485589323:1485594235, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 4912
1793457043.148882 fvnb0 In  IP (tos 0x0, ttl 63, id 31845, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0xba6c (correct), ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148889 fvnb0 Out IP (tos 0x0, ttl 64, id 14915, offset 0, flags [DF], proto TCP (6), length 7420)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485594235:1485601603, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 7368
1793457043.148931 fvnb0 In  IP (tos 0x0, ttl 63, id 31846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577746077:2577747083, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148932 fvnb0 In  IP (tos 0x0, ttl 63, id 31847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577747083:2577748089, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148932 fvnb0 In  IP (tos 0x0, ttl 63, id 31848, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0xa42f (correct), ack 1485574587, win 55, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148942 fvnb0 Out IP (tos 0x0, ttl 64, id 14921, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x3aa2), ack 2577748089, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148981 fvnb0 In  IP (tos 0x0, ttl 63, id 31849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577748089:2577749095, ack 1485588095, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148981 fvnb0 In  IP (tos 0x0, ttl 63, id 31850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577749095:2577750101, ack 1485589323, win 41, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148989 fvnb0 Out IP (tos 0x0, ttl 64, id 14922, offset 0, flags [DF], proto TCP (6), length 14996)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485601603:1485616547, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 14944
1793457043.149002 fvnb0 In  IP (tos 0x0, ttl 63, id 31851, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x4fa3 (correct), ack 1485594235, win 39, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149014 fvnb0 In  IP (tos 0x0, ttl 63, id 31852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577750101:2577751107, ack 1485600375, win 34, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149022 fvnb0 Out IP (tos 0x0, ttl 64, id 14935, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xf478), ack 2577751107, win 59, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149035 fvnb0 In  IP (tos 0x0, ttl 63, id 31853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577751107:2577752113, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149056 fvnb0 In  IP (tos 0x0, ttl 63, id 31854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577752113:2577753119, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149065 fvnb0 Out IP (tos 0x0, ttl 64, id 14936, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xec9d), ack 2577753119, win 58, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149112 fvnb0 In  IP (tos 0x0, ttl 63, id 31855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577753119:2577754125, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149112 fvnb0 In  IP (tos 0x0, ttl 63, id 31856, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x1993 (correct), ack 1485604059, win 31, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149116 fvnb0 In  IP (tos 0x0, ttl 63, id 31857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577754125:2577755131, ack 1485606515, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149121 fvnb0 Out IP (tos 0x0, ttl 64, id 14937, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xe4c2), ack 2577755131, win 57, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149130 fvnb0 In  IP (tos 0x0, ttl 63, id 31858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577755131:2577756137, ack 1485612655, win 24, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149141 fvnb0 In  IP (tos 0x0, ttl 63, id 31859, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0xe1c6 (correct), ack 1485616339, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149163 fvnb0 In  IP (tos 0x0, ttl 63, id 31860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577756137:2577757143, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149169 fvnb0 Out IP (tos 0x0, ttl 64, id 14938, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xdce7), ack 2577757143, win 56, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149205 fvnb0 In  IP (tos 0x0, ttl 63, id 31861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577757143:2577758149, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149207 fvnb0 In  IP (tos 0x0, ttl 63, id 31862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577758149:2577759155, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149212 fvnb0 Out IP (tos 0x0, ttl 64, id 14939, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xd50c), ack 2577759155, win 55, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149216 fvnb0 In  IP (tos 0x0, ttl 63, id 31863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577759155:2577760161, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149235 fvnb0 In  IP (tos 0x0, ttl 63, id 31864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577760161:2577761167, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149240 fvnb0 Out IP (tos 0x0, ttl 64, id 14940, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xcd31), ack 2577761167, win 54, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149245 fvnb0 In  IP (tos 0x0, ttl 63, id 31865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577761167:2577762173, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149270 fvnb0 In  IP (tos 0x0, ttl 63, id 31866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577762173:2577763179, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149275 fvnb0 Out IP (tos 0x0, ttl 64, id 14941, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xc556), ack 2577763179, win 53, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149287 fvnb0 In  IP (tos 0x0, ttl 63, id 31867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577763179:2577764185, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149305 fvnb0 In  IP (tos 0x0, ttl 63, id 31868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577764185:2577765191, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149310 fvnb0 Out IP (tos 0x0, ttl 64, id 14942, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xbd7b), ack 2577765191, win 52, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149322 fvnb0 In  IP (tos 0x0, ttl 63, id 31869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577765191:2577766197, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149358 fvnb0 In  IP (tos 0x0, ttl 63, id 31870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577766197:2577767203, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149361 fvnb0 In  IP (tos 0x0, ttl 63, id 31871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577767203:2577768209, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149364 fvnb0 Out IP (tos 0x0, ttl 64, id 14943, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xb1b3), ack 2577768209, win 50, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149374 fvnb0 In  IP (tos 0x0, ttl 63, id 31872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577768209:2577769215, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149403 fvnb0 In  IP (tos 0x0, ttl 63, id 31873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577769215:2577770221, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149406 fvnb0 In  IP (tos 0x0, ttl 63, id 31874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577770221:2577771227, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149409 fvnb0 Out IP (tos 0x0, ttl 64, id 14944, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xa9d8), ack 2577770221, win 49, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149420 fvnb0 In  IP (tos 0x0, ttl 63, id 31875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577771227:2577772233, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149425 fvnb0 Out IP (tos 0x0, ttl 64, id 14945, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0xa1fd), ack 2577772233, win 48, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149449 fvnb0 In  IP (tos 0x0, ttl 63, id 31876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577772233:2577773239, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149464 fvnb0 In  IP (tos 0x0, ttl 63, id 31877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577773239:2577774245, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149469 fvnb0 Out IP (tos 0x0, ttl 64, id 14946, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x9a22), ack 2577774245, win 47, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149482 fvnb0 In  IP (tos 0x0, ttl 63, id 31878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577774245:2577775251, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149498 fvnb0 In  IP (tos 0x0, ttl 63, id 31879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577775251:2577776257, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149503 fvnb0 Out IP (tos 0x0, ttl 64, id 14947, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x9247), ack 2577776257, win 46, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149535 fvnb0 In  IP (tos 0x0, ttl 63, id 31880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577776257:2577777263, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149536 fvnb0 In  IP (tos 0x0, ttl 63, id 31881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577777263:2577778269, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149542 fvnb0 Out IP (tos 0x0, ttl 64, id 14948, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x8a6c), ack 2577778269, win 45, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149574 fvnb0 In  IP (tos 0x0, ttl 63, id 31882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577778269:2577779275, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149574 fvnb0 In  IP (tos 0x0, ttl 63, id 31883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577779275:2577780281, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149580 fvnb0 Out IP (tos 0x0, ttl 64, id 14949, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x8291), ack 2577780281, win 44, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149589 fvnb0 In  IP (tos 0x0, ttl 63, id 31884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577780281:2577781287, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149608 fvnb0 In  IP (tos 0x0, ttl 63, id 31885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577781287:2577782293, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149614 fvnb0 Out IP (tos 0x0, ttl 64, id 14950, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x7ab6), ack 2577782293, win 43, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149620 fvnb0 In  IP (tos 0x0, ttl 63, id 31886, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577782293:2577783299, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149636 fvnb0 In  IP (tos 0x0, ttl 63, id 31887, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577783299:2577784305, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149641 fvnb0 Out IP (tos 0x0, ttl 64, id 14951, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x72da), ack 2577784305, win 42, options [nop,nop,TS val 2154689861 ecr 2952145650], length 0
1793457043.149656 fvnb0 In  IP (tos 0x0, ttl 63, id 31888, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577784305:2577785311, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149673 fvnb0 In  IP (tos 0x0, ttl 63, id 31889, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577785311:2577786317, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149678 fvnb0 Out IP (tos 0x0, ttl 64, id 14952, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x6afe), ack 2577786317, win 41, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149691 fvnb0 In  IP (tos 0x0, ttl 63, id 31890, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577786317:2577787323, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149724 fvnb0 In  IP (tos 0x0, ttl 63, id 31891, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577787323:2577788329, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149725 fvnb0 In  IP (tos 0x0, ttl 63, id 31892, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577788329:2577789335, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149729 fvnb0 Out IP (tos 0x0, ttl 64, id 14953, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x5f36), ack 2577789335, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149741 fvnb0 In  IP (tos 0x0, ttl 63, id 31893, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577789335:2577790341, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149781 fvnb0 In  IP (tos 0x0, ttl 63, id 31894, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577790341:2577791347, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149781 fvnb0 In  IP (tos 0x0, ttl 63, id 31895, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577791347:2577792353, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149786 fvnb0 Out IP (tos 0x0, ttl 64, id 14954, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x536c), ack 2577792353, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149800 fvnb0 In  IP (tos 0x0, ttl 63, id 31896, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577792353:2577793359, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149801 fvnb0 In  IP (tos 0x0, ttl 63, id 31897, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577793359:2577794365, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149806 fvnb0 Out IP (tos 0x0, ttl 64, id 14955, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x4b90), ack 2577794365, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149808 fvnb0 In  IP (tos 0x0, ttl 63, id 31898, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577794365:2577795371, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149817 fvnb0 In  IP (tos 0x0, ttl 63, id 31899, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577795371:2577796377, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149822 fvnb0 Out IP (tos 0x0, ttl 64, id 14956, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x43b4), ack 2577796377, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149826 fvnb0 In  IP (tos 0x0, ttl 63, id 31900, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577796377:2577797383, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149837 fvnb0 In  IP (tos 0x0, ttl 63, id 31901, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577797383:2577798389, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149841 fvnb0 Out IP (tos 0x0, ttl 64, id 14957, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x3bd8), ack 2577798389, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149847 fvnb0 In  IP (tos 0x0, ttl 63, id 31902, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577798389:2577799395, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149852 fvnb0 In  IP (tos 0x0, ttl 63, id 31903, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577799395:2577800401, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149856 fvnb0 Out IP (tos 0x0, ttl 64, id 14958, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x33fc), ack 2577800401, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149870 fvnb0 In  IP (tos 0x0, ttl 63, id 31904, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577800401:2577801407, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149870 fvnb0 In  IP (tos 0x0, ttl 63, id 31905, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577801407:2577802413, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149875 fvnb0 Out IP (tos 0x0, ttl 64, id 14959, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x2c20), ack 2577802413, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149884 fvnb0 In  IP (tos 0x0, ttl 63, id 31906, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577802413:2577803419, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149884 fvnb0 In  IP (tos 0x0, ttl 63, id 31907, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577803419:2577804425, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149889 fvnb0 Out IP (tos 0x0, ttl 64, id 14960, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x2444), ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150055 fvnb0 Out IP (tos 0x0, ttl 64, id 14961, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [F.], cksum 0x29fd (incorrect -> 0x2443), seq 1485616547, ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150155 fvnb0 In  IP (tos 0x0, ttl 63, id 31908, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [F.], cksum 0x2426 (correct), seq 2577804425, ack 1485616548, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 0
1793457043.150159 fvnb0 Out IP (tos 0x0, ttl 64, id 14962, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x29fd (incorrect -> 0x2442), ack 2577804426, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0

- matrix1280_path_0: eth0
2
1280
eth1
3
1280
fvnb0
8
1280
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4418,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"8a:b5:ad:5a:5a:7a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4421,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:13:34:07:c1:06","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":8,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.78.2.126","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:49198 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.613/1.162 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5091 segs_out:17 segs_in:14 data_segs_out:9 data_segs_in:10 send 188972268bps lastsnd:58772 lastrcv:58771 lastack:28675 pacing_rate 377790456bps delivery_rate 2896000000bps delivered:10 app_limited busy:6ms rcv_space:14480 rcv_ssthresh:81050 minrtt:0.007 snd_wnd:71680
ESTAB 0      0       172.28.30.41:53656 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.196/0.236 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3439 bytes_acked:3440 bytes_received:5477 segs_out:32 segs_in:31 data_segs_out:18 data_segs_in:26 send 501224490bps lastsnd:484 lastrcv:484 lastack:484 pacing_rate 998627696bps delivery_rate 4912000000bps delivered:19 app_limited busy:2ms rcv_space:12280 rcv_ssthresh:85803 minrtt:0.004 snd_wnd:79872
ESTAB 0      0       172.28.30.41:53672 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.51/2.856 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3735 bytes_acked:3736 bytes_received:6210 segs_out:29 segs_in:34 data_segs_out:16 data_segs_in:30 send 65059603bps lastsnd:5221 lastrcv:5221 lastack:5221 pacing_rate 130076128bps delivery_rate 2183111104bps delivered:17 app_limited busy:36ms rcv_space:12280 rcv_ssthresh:85793 minrtt:0.009 snd_wnd:78848
ESTAB 0      0       172.28.30.41:53682 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.22/10.315 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2928 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 18819923bps lastsnd:5221 lastrcv:5221 lastack:5180 pacing_rate 37635336bps delivery_rate 2183111104bps delivered:9 app_limited busy:43ms rcv_space:12280 rcv_ssthresh:78068 minrtt:0.009 snd_wnd:74752

- matrix1280_capture_0_receipt: {"bytes":"44652","sha256":"365ca6547c5d0528b834f29c0174330391cf1e700e886aa0ef8b070c8f4ed2ef"}
- matrix1280_capture_1_window: {"launch_requested_at":"2026-10-31T14:30:41.900794451Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:44.091050085Z","stop_requested_at":"2026-10-31T14:30:44.047305702Z"}
- matrix1280_ipv6_uplink_no_leak: true
- matrix1280_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
500 packets captured
502 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457043.148218 fvmb0 In  IP (tos 0x0, ttl 64, id 14875, offset 0, flags [DF], proto TCP (6), length 60)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [S], cksum 0x962f (correct), seq 1485552438, win 64480, options [mss 1240,sackOK,TS val 2154689859 ecr 0,nop,wscale 10], length 0
1793457043.148244 fvup0 Out IP (tos 0x0, ttl 63, id 14875, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [S], cksum 0x39e7 (correct), seq 1485552438, win 64480, options [mss 1240,sackOK,TS val 2154689859 ecr 0,nop,wscale 10], length 0
1793457043.148268 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [S.], cksum 0x864d (incorrect -> 0x90bf), seq 2577740037, ack 1485552439, win 65160, options [mss 1460,sackOK,TS val 2952145649 ecr 2154689859,nop,wscale 10], length 0
1793457043.148273 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [S.], cksum 0x2a05 (incorrect -> 0xed07), seq 2577740037, ack 1485552439, win 65160, options [mss 1460,sackOK,TS val 2952145649 ecr 2154689859,nop,wscale 10], length 0
1793457043.148376 fvmb0 In  IP (tos 0x0, ttl 64, id 14876, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x1a21 (correct), ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148379 fvup0 Out IP (tos 0x0, ttl 63, id 14876, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xbdd8 (correct), ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148422 fvmb0 In  IP (tos 0x0, ttl 64, id 14877, offset 0, flags [DF], proto TCP (6), length 96)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], cksum 0x2b45 (correct), seq 1485552439:1485552483, ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 44
1793457043.148433 fvup0 Out IP (tos 0x0, ttl 63, id 14877, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], cksum 0xcefc (correct), seq 1485552439:1485552483, ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 44
1793457043.148446 fvup0 In  IP (tos 0x0, ttl 64, id 31836, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xbdab), ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 0
1793457043.148450 fvmb0 Out IP (tos 0x0, ttl 63, id 31836, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x29fd (incorrect -> 0x19f4), ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 0
1793457043.148492 fvup0 In  IP (tos 0x0, ttl 64, id 31837, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], cksum 0x8648 (incorrect -> 0x6455), seq 2577740038:2577740041, ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 3
1793457043.148499 fvmb0 Out IP (tos 0x0, ttl 63, id 31837, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], cksum 0x2a00 (incorrect -> 0xc09d), seq 2577740038:2577740041, ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 3
1793457043.148555 fvmb0 In  IP (tos 0x0, ttl 64, id 14878, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x19f2 (correct), ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148576 fvup0 Out IP (tos 0x0, ttl 63, id 14878, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xbda9 (correct), ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148658 fvmb0 In  IP (tos 0x0, ttl 64, id 14879, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485552483:1485553711, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148665 fvup0 Out IP (tos 0x0, ttl 63, id 14879, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485552483:1485553711, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148676 fvmb0 In  IP (tos 0x0, ttl 64, id 14880, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485553711:1485558623, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 4912
1793457043.148680 fvup0 Out IP (tos 0x0, ttl 63, id 14880, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485553711:1485558623, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 4912
1793457043.148685 fvmb0 In  IP (tos 0x0, ttl 64, id 14884, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485558623:1485559851, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148688 fvup0 Out IP (tos 0x0, ttl 63, id 14884, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485558623:1485559851, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148690 fvmb0 In  IP (tos 0x0, ttl 64, id 14885, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485559851:1485561079, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148693 fvup0 Out IP (tos 0x0, ttl 63, id 14885, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485559851:1485561079, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148698 fvup0 In  IP (tos 0x0, ttl 64, id 31838, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xa5ab), ack 1485558623, win 64, options [nop,nop,TS val 2952145650 ecr 2154689859], length 0
1793457043.148701 fvmb0 In  IP (tos 0x0, ttl 64, id 14886, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485561079:1485563535, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 2456
1793457043.148705 fvup0 Out IP (tos 0x0, ttl 63, id 14886, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485561079:1485563535, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 2456
1793457043.148712 fvmb0 Out IP (tos 0x0, ttl 63, id 31838, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x29fd (incorrect -> 0x01f4), ack 1485558623, win 64, options [nop,nop,TS val 2952145650 ecr 2154689859], length 0
1793457043.148729 fvmb0 In  IP (tos 0x0, ttl 64, id 14888, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485563535:1485564763, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148731 fvup0 Out IP (tos 0x0, ttl 63, id 14888, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485563535:1485564763, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148747 fvup0 In  IP (tos 0x0, ttl 64, id 31839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577740041:2577741047, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148753 fvmb0 Out IP (tos 0x0, ttl 63, id 31839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577740041:2577741047, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148769 fvup0 In  IP (tos 0x0, ttl 64, id 31840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577741047:2577742053, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148774 fvmb0 Out IP (tos 0x0, ttl 63, id 31840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577741047:2577742053, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148791 fvup0 In  IP (tos 0x0, ttl 64, id 31841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577742053:2577743059, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148795 fvmb0 Out IP (tos 0x0, ttl 63, id 31841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577742053:2577743059, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148807 fvup0 In  IP (tos 0x0, ttl 64, id 31842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577743059:2577744065, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148810 fvmb0 Out IP (tos 0x0, ttl 63, id 31842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577743059:2577744065, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148820 fvmb0 In  IP (tos 0x0, ttl 64, id 14889, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485564763:1485565991, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148824 fvup0 Out IP (tos 0x0, ttl 63, id 14889, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485564763:1485565991, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148827 fvup0 In  IP (tos 0x0, ttl 64, id 31843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577744065:2577745071, ack 1485564763, win 63, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148829 fvmb0 In  IP (tos 0x0, ttl 64, id 14890, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485565991:1485567219, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148830 fvup0 Out IP (tos 0x0, ttl 63, id 14890, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485565991:1485567219, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148831 fvmb0 Out IP (tos 0x0, ttl 63, id 31843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577744065:2577745071, ack 1485564763, win 63, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148839 fvmb0 In  IP (tos 0x0, ttl 64, id 14891, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485567219:1485570903, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148842 fvup0 Out IP (tos 0x0, ttl 63, id 14891, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485567219:1485570903, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148847 fvup0 In  IP (tos 0x0, ttl 64, id 31844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577745071:2577746077, ack 1485567219, win 61, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148851 fvmb0 Out IP (tos 0x0, ttl 63, id 31844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577745071:2577746077, ack 1485567219, win 61, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148861 fvup0 In  IP (tos 0x0, ttl 64, id 31845, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0x5e24), ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148863 fvmb0 In  IP (tos 0x0, ttl 64, id 14894, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485570903:1485573359, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.148864 fvmb0 In  IP (tos 0x0, ttl 64, id 14896, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485573359:1485574587, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148864 fvmb0 Out IP (tos 0x0, ttl 63, id 31845, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x29fd (incorrect -> 0xba6c), ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148877 fvup0 In  IP (tos 0x0, ttl 64, id 31846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577746077:2577747083, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148881 fvmb0 Out IP (tos 0x0, ttl 63, id 31846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577746077:2577747083, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148883 fvup0 Out IP (tos 0x0, ttl 63, id 14894, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485570903:1485573359, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.148888 fvup0 Out IP (tos 0x0, ttl 63, id 14896, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485573359:1485574587, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148894 fvup0 In  IP (tos 0x0, ttl 64, id 31847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577747083:2577748089, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148898 fvmb0 Out IP (tos 0x0, ttl 63, id 31847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577747083:2577748089, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148903 fvmb0 In  IP (tos 0x0, ttl 64, id 14897, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485574587:1485577043, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.148904 fvmb0 In  IP (tos 0x0, ttl 64, id 14899, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485577043:1485580727, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148909 fvup0 Out IP (tos 0x0, ttl 63, id 14897, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485574587:1485577043, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.148911 fvup0 In  IP (tos 0x0, ttl 64, id 31848, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0x47e7), ack 1485574587, win 55, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148913 fvup0 Out IP (tos 0x0, ttl 63, id 14899, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485577043:1485580727, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148915 fvmb0 Out IP (tos 0x0, ttl 63, id 31848, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x29fd (incorrect -> 0xa42f), ack 1485574587, win 55, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148921 fvmb0 In  IP (tos 0x0, ttl 64, id 14902, offset 0, flags [DF], proto TCP (6), length 7420)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485580727:1485588095, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 7368
1793457043.148925 fvup0 Out IP (tos 0x0, ttl 63, id 14902, offset 0, flags [DF], proto TCP (6), length 7420)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485580727:1485588095, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 7368
1793457043.148935 fvmb0 In  IP (tos 0x0, ttl 64, id 14908, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485588095:1485589323, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148935 fvmb0 In  IP (tos 0x0, ttl 64, id 14909, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x7e3e (correct), ack 2577743059, win 62, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148936 fvmb0 In  IP (tos 0x0, ttl 64, id 14910, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x7663 (correct), ack 2577745071, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148937 fvup0 In  IP (tos 0x0, ttl 64, id 31849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577748089:2577749095, ack 1485588095, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148940 fvup0 Out IP (tos 0x0, ttl 63, id 14908, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485588095:1485589323, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148941 fvmb0 Out IP (tos 0x0, ttl 63, id 31849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577748089:2577749095, ack 1485588095, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148943 fvup0 Out IP (tos 0x0, ttl 63, id 14909, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x21f6 (correct), ack 2577743059, win 62, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148945 fvup0 Out IP (tos 0x0, ttl 63, id 14910, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x1a1b (correct), ack 2577745071, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148956 fvup0 In  IP (tos 0x0, ttl 64, id 31850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577749095:2577750101, ack 1485589323, win 41, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148957 fvmb0 In  IP (tos 0x0, ttl 64, id 14911, offset 0, flags [DF], proto TCP (6), length 4964)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485589323:1485594235, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 4912
1793457043.148957 fvmb0 In  IP (tos 0x0, ttl 64, id 14915, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485594235:1485595463, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148960 fvmb0 Out IP (tos 0x0, ttl 63, id 31850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577749095:2577750101, ack 1485589323, win 41, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148961 fvup0 Out IP (tos 0x0, ttl 63, id 14911, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485589323:1485594235, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 4912
1793457043.148964 fvup0 Out IP (tos 0x0, ttl 63, id 14915, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485594235:1485595463, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148971 fvmb0 In  IP (tos 0x0, ttl 64, id 14916, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485595463:1485596691, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148974 fvup0 Out IP (tos 0x0, ttl 63, id 14916, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485595463:1485596691, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148975 fvup0 In  IP (tos 0x0, ttl 64, id 31851, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xf35a), ack 1485594235, win 39, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148979 fvmb0 In  IP (tos 0x0, ttl 64, id 14917, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485596691:1485600375, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148980 fvmb0 Out IP (tos 0x0, ttl 63, id 31851, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x29fd (incorrect -> 0x4fa3), ack 1485594235, win 39, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148983 fvup0 Out IP (tos 0x0, ttl 63, id 14917, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485596691:1485600375, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148988 fvmb0 In  IP (tos 0x0, ttl 64, id 14920, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485600375:1485601603, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148989 fvmb0 In  IP (tos 0x0, ttl 64, id 14921, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x3aa2 (correct), ack 2577748089, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148992 fvup0 Out IP (tos 0x0, ttl 63, id 14920, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485600375:1485601603, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148995 fvup0 Out IP (tos 0x0, ttl 63, id 14921, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xde59 (correct), ack 2577748089, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148995 fvup0 In  IP (tos 0x0, ttl 64, id 31852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577750101:2577751107, ack 1485600375, win 34, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148999 fvmb0 Out IP (tos 0x0, ttl 63, id 31852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577750101:2577751107, ack 1485600375, win 34, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149014 fvup0 In  IP (tos 0x0, ttl 64, id 31853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577751107:2577752113, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149019 fvmb0 Out IP (tos 0x0, ttl 63, id 31853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577751107:2577752113, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149036 fvup0 In  IP (tos 0x0, ttl 64, id 31854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577752113:2577753119, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149040 fvmb0 Out IP (tos 0x0, ttl 63, id 31854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577752113:2577753119, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149049 fvmb0 In  IP (tos 0x0, ttl 64, id 14922, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485601603:1485602831, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149052 fvup0 Out IP (tos 0x0, ttl 63, id 14922, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485601603:1485602831, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149056 fvup0 In  IP (tos 0x0, ttl 64, id 31855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577753119:2577754125, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149056 fvmb0 In  IP (tos 0x0, ttl 64, id 14923, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485602831:1485604059, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149061 fvmb0 Out IP (tos 0x0, ttl 63, id 31855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577753119:2577754125, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149062 fvup0 Out IP (tos 0x0, ttl 63, id 14923, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485602831:1485604059, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149066 fvmb0 In  IP (tos 0x0, ttl 64, id 14924, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485604059:1485606515, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.149069 fvup0 Out IP (tos 0x0, ttl 63, id 14924, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485604059:1485606515, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.149074 fvup0 In  IP (tos 0x0, ttl 64, id 31856, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xbd4a), ack 1485604059, win 31, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149078 fvmb0 Out IP (tos 0x0, ttl 63, id 31856, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x29fd (incorrect -> 0x1993), ack 1485604059, win 31, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149087 fvmb0 In  IP (tos 0x0, ttl 64, id 14926, offset 0, flags [DF], proto TCP (6), length 3736)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485606515:1485610199, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.149090 fvup0 Out IP (tos 0x0, ttl 63, id 14926, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485606515:1485610199, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.149090 fvup0 In  IP (tos 0x0, ttl 64, id 31857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577754125:2577755131, ack 1485606515, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149093 fvmb0 In  IP (tos 0x0, ttl 64, id 14929, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485610199:1485611427, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149094 fvmb0 Out IP (tos 0x0, ttl 63, id 31857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577754125:2577755131, ack 1485606515, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149095 fvup0 Out IP (tos 0x0, ttl 63, id 14929, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485610199:1485611427, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149100 fvmb0 In  IP (tos 0x0, ttl 64, id 14930, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485611427:1485612655, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149103 fvup0 Out IP (tos 0x0, ttl 63, id 14930, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485611427:1485612655, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149107 fvmb0 In  IP (tos 0x0, ttl 64, id 14931, offset 0, flags [DF], proto TCP (6), length 2508)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485612655:1485615111, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.149110 fvup0 In  IP (tos 0x0, ttl 64, id 31858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577755131:2577756137, ack 1485612655, win 24, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149110 fvup0 Out IP (tos 0x0, ttl 63, id 14931, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485612655:1485615111, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.149114 fvmb0 In  IP (tos 0x0, ttl 64, id 14933, offset 0, flags [DF], proto TCP (6), length 1280)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], seq 1485615111:1485616339, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149114 fvmb0 Out IP (tos 0x0, ttl 63, id 31858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577755131:2577756137, ack 1485612655, win 24, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149116 fvup0 Out IP (tos 0x0, ttl 63, id 14933, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485615111:1485616339, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149120 fvmb0 In  IP (tos 0x0, ttl 64, id 14934, offset 0, flags [DF], proto TCP (6), length 260)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [P.], seq 1485616339:1485616547, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 208
1793457043.149121 fvmb0 In  IP (tos 0x0, ttl 64, id 14935, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xf478 (correct), ack 2577751107, win 59, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149125 fvup0 Out IP (tos 0x0, ttl 63, id 14934, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485616339:1485616547, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 208
1793457043.149125 fvup0 In  IP (tos 0x0, ttl 64, id 31859, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0x857e), ack 1485616339, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149127 fvup0 Out IP (tos 0x0, ttl 63, id 14935, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x9830 (correct), ack 2577751107, win 59, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149129 fvmb0 Out IP (tos 0x0, ttl 63, id 31859, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [.], cksum 0x29fd (incorrect -> 0xe1c6), ack 1485616339, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149132 fvmb0 In  IP (tos 0x0, ttl 64, id 14936, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xec9d (correct), ack 2577753119, win 58, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149136 fvup0 Out IP (tos 0x0, ttl 63, id 14936, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x9055 (correct), ack 2577753119, win 58, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149141 fvmb0 In  IP (tos 0x0, ttl 64, id 14937, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xe4c2 (correct), ack 2577755131, win 57, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149144 fvup0 Out IP (tos 0x0, ttl 63, id 14937, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x887a (correct), ack 2577755131, win 57, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149144 fvup0 In  IP (tos 0x0, ttl 64, id 31860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577756137:2577757143, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149148 fvmb0 Out IP (tos 0x0, ttl 63, id 31860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577756137:2577757143, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149163 fvup0 In  IP (tos 0x0, ttl 64, id 31861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577757143:2577758149, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149166 fvmb0 Out IP (tos 0x0, ttl 63, id 31861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577757143:2577758149, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149183 fvup0 In  IP (tos 0x0, ttl 64, id 31862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577758149:2577759155, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149185 fvmb0 In  IP (tos 0x0, ttl 64, id 14938, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xdce7 (correct), ack 2577757143, win 56, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149186 fvmb0 Out IP (tos 0x0, ttl 63, id 31862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577758149:2577759155, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149188 fvup0 Out IP (tos 0x0, ttl 63, id 14938, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x809f (correct), ack 2577757143, win 56, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149198 fvup0 In  IP (tos 0x0, ttl 64, id 31863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577759155:2577760161, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149201 fvmb0 Out IP (tos 0x0, ttl 63, id 31863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577759155:2577760161, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149215 fvup0 In  IP (tos 0x0, ttl 64, id 31864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577760161:2577761167, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149219 fvmb0 Out IP (tos 0x0, ttl 63, id 31864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577760161:2577761167, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149230 fvup0 In  IP (tos 0x0, ttl 64, id 31865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577761167:2577762173, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149234 fvmb0 Out IP (tos 0x0, ttl 63, id 31865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577761167:2577762173, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149235 fvmb0 In  IP (tos 0x0, ttl 64, id 14939, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xd50c (correct), ack 2577759155, win 55, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149238 fvup0 Out IP (tos 0x0, ttl 63, id 14939, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x78c4 (correct), ack 2577759155, win 55, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149251 fvup0 In  IP (tos 0x0, ttl 64, id 31866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577762173:2577763179, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149253 fvmb0 In  IP (tos 0x0, ttl 64, id 14940, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xcd31 (correct), ack 2577761167, win 54, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149254 fvmb0 Out IP (tos 0x0, ttl 63, id 31866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577762173:2577763179, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149257 fvup0 Out IP (tos 0x0, ttl 63, id 14940, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x70e9 (correct), ack 2577761167, win 54, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149269 fvup0 In  IP (tos 0x0, ttl 64, id 31867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577763179:2577764185, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149273 fvmb0 Out IP (tos 0x0, ttl 63, id 31867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577763179:2577764185, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149288 fvup0 In  IP (tos 0x0, ttl 64, id 31868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577764185:2577765191, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149292 fvmb0 Out IP (tos 0x0, ttl 63, id 31868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577764185:2577765191, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149303 fvup0 In  IP (tos 0x0, ttl 64, id 31869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577765191:2577766197, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149308 fvmb0 Out IP (tos 0x0, ttl 63, id 31869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577765191:2577766197, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149316 fvmb0 In  IP (tos 0x0, ttl 64, id 14941, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xc556 (correct), ack 2577763179, win 53, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149319 fvup0 Out IP (tos 0x0, ttl 63, id 14941, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x690e (correct), ack 2577763179, win 53, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149322 fvup0 In  IP (tos 0x0, ttl 64, id 31870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577766197:2577767203, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149325 fvmb0 Out IP (tos 0x0, ttl 63, id 31870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577766197:2577767203, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149337 fvup0 In  IP (tos 0x0, ttl 64, id 31871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577767203:2577768209, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149338 fvmb0 In  IP (tos 0x0, ttl 64, id 14942, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xbd7b (correct), ack 2577765191, win 52, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149340 fvmb0 Out IP (tos 0x0, ttl 63, id 31871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577767203:2577768209, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149341 fvup0 Out IP (tos 0x0, ttl 63, id 14942, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x6133 (correct), ack 2577765191, win 52, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149355 fvup0 In  IP (tos 0x0, ttl 64, id 31872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577768209:2577769215, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149358 fvmb0 Out IP (tos 0x0, ttl 63, id 31872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577768209:2577769215, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149369 fvup0 In  IP (tos 0x0, ttl 64, id 31873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577769215:2577770221, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149373 fvmb0 Out IP (tos 0x0, ttl 63, id 31873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577769215:2577770221, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149377 fvmb0 In  IP (tos 0x0, ttl 64, id 14943, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xb1b3 (correct), ack 2577768209, win 50, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149379 fvup0 Out IP (tos 0x0, ttl 63, id 14943, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x556b (correct), ack 2577768209, win 50, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149387 fvup0 In  IP (tos 0x0, ttl 64, id 31874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577770221:2577771227, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149390 fvmb0 Out IP (tos 0x0, ttl 63, id 31874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577770221:2577771227, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149403 fvup0 In  IP (tos 0x0, ttl 64, id 31875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577771227:2577772233, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149407 fvmb0 Out IP (tos 0x0, ttl 63, id 31875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577771227:2577772233, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149423 fvmb0 In  IP (tos 0x0, ttl 64, id 14944, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xa9d8 (correct), ack 2577770221, win 49, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149425 fvup0 Out IP (tos 0x0, ttl 63, id 14944, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x4d90 (correct), ack 2577770221, win 49, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149431 fvup0 In  IP (tos 0x0, ttl 64, id 31876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577772233:2577773239, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149434 fvmb0 Out IP (tos 0x0, ttl 63, id 31876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577772233:2577773239, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149446 fvup0 In  IP (tos 0x0, ttl 64, id 31877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577773239:2577774245, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149449 fvmb0 Out IP (tos 0x0, ttl 63, id 31877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577773239:2577774245, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149464 fvup0 In  IP (tos 0x0, ttl 64, id 31878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577774245:2577775251, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149468 fvmb0 Out IP (tos 0x0, ttl 63, id 31878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577774245:2577775251, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149479 fvup0 In  IP (tos 0x0, ttl 64, id 31879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577775251:2577776257, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149483 fvmb0 Out IP (tos 0x0, ttl 63, id 31879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577775251:2577776257, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149484 fvmb0 In  IP (tos 0x0, ttl 64, id 14945, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0xa1fd (correct), ack 2577772233, win 48, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149486 fvup0 Out IP (tos 0x0, ttl 63, id 14945, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x45b5 (correct), ack 2577772233, win 48, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149493 fvmb0 In  IP (tos 0x0, ttl 64, id 14946, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x9a22 (correct), ack 2577774245, win 47, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149494 fvup0 Out IP (tos 0x0, ttl 63, id 14946, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x3dda (correct), ack 2577774245, win 47, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149501 fvup0 In  IP (tos 0x0, ttl 64, id 31880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577776257:2577777263, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149504 fvmb0 Out IP (tos 0x0, ttl 63, id 31880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577776257:2577777263, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149516 fvmb0 In  IP (tos 0x0, ttl 64, id 14947, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x9247 (correct), ack 2577776257, win 46, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149517 fvup0 In  IP (tos 0x0, ttl 64, id 31881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577777263:2577778269, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149517 fvup0 Out IP (tos 0x0, ttl 63, id 14947, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x35ff (correct), ack 2577776257, win 46, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149520 fvmb0 Out IP (tos 0x0, ttl 63, id 31881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577777263:2577778269, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149536 fvup0 In  IP (tos 0x0, ttl 64, id 31882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577778269:2577779275, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149540 fvmb0 Out IP (tos 0x0, ttl 63, id 31882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577778269:2577779275, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149551 fvup0 In  IP (tos 0x0, ttl 64, id 31883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577779275:2577780281, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149554 fvmb0 In  IP (tos 0x0, ttl 64, id 14948, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x8a6c (correct), ack 2577778269, win 45, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149555 fvmb0 Out IP (tos 0x0, ttl 63, id 31883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577779275:2577780281, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149556 fvup0 Out IP (tos 0x0, ttl 63, id 14948, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x2e24 (correct), ack 2577778269, win 45, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149569 fvup0 In  IP (tos 0x0, ttl 64, id 31884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577780281:2577781287, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149572 fvmb0 Out IP (tos 0x0, ttl 63, id 31884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577780281:2577781287, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149583 fvup0 In  IP (tos 0x0, ttl 64, id 31885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577781287:2577782293, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149587 fvmb0 Out IP (tos 0x0, ttl 63, id 31885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577781287:2577782293, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149596 fvmb0 In  IP (tos 0x0, ttl 64, id 14949, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x8291 (correct), ack 2577780281, win 44, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149597 fvup0 Out IP (tos 0x0, ttl 63, id 14949, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x2649 (correct), ack 2577780281, win 44, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149600 fvup0 In  IP (tos 0x0, ttl 64, id 31886, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577782293:2577783299, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149605 fvmb0 Out IP (tos 0x0, ttl 63, id 31886, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577782293:2577783299, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149616 fvup0 In  IP (tos 0x0, ttl 64, id 31887, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577783299:2577784305, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149619 fvmb0 Out IP (tos 0x0, ttl 63, id 31887, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577783299:2577784305, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149626 fvmb0 In  IP (tos 0x0, ttl 64, id 14950, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x7ab6 (correct), ack 2577782293, win 43, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149627 fvup0 Out IP (tos 0x0, ttl 63, id 14950, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x1e6e (correct), ack 2577782293, win 43, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149636 fvup0 In  IP (tos 0x0, ttl 64, id 31888, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577784305:2577785311, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149639 fvmb0 Out IP (tos 0x0, ttl 63, id 31888, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577784305:2577785311, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149656 fvup0 In  IP (tos 0x0, ttl 64, id 31889, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577785311:2577786317, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149659 fvmb0 Out IP (tos 0x0, ttl 63, id 31889, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577785311:2577786317, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149674 fvup0 In  IP (tos 0x0, ttl 64, id 31890, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577786317:2577787323, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149678 fvmb0 Out IP (tos 0x0, ttl 63, id 31890, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577786317:2577787323, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149692 fvup0 In  IP (tos 0x0, ttl 64, id 31891, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577787323:2577788329, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149693 fvmb0 In  IP (tos 0x0, ttl 64, id 14951, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x72da (correct), ack 2577784305, win 42, options [nop,nop,TS val 2154689861 ecr 2952145650], length 0
1793457043.149694 fvmb0 In  IP (tos 0x0, ttl 64, id 14952, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x6afe (correct), ack 2577786317, win 41, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149695 fvup0 Out IP (tos 0x0, ttl 63, id 14951, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x1692 (correct), ack 2577784305, win 42, options [nop,nop,TS val 2154689861 ecr 2952145650], length 0
1793457043.149696 fvmb0 Out IP (tos 0x0, ttl 63, id 31891, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577787323:2577788329, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149696 fvup0 Out IP (tos 0x0, ttl 63, id 14952, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x0eb6 (correct), ack 2577786317, win 41, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149707 fvup0 In  IP (tos 0x0, ttl 64, id 31892, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577788329:2577789335, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149711 fvmb0 Out IP (tos 0x0, ttl 63, id 31892, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577788329:2577789335, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149724 fvup0 In  IP (tos 0x0, ttl 64, id 31893, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577789335:2577790341, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149728 fvmb0 Out IP (tos 0x0, ttl 63, id 31893, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577789335:2577790341, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149739 fvup0 In  IP (tos 0x0, ttl 64, id 31894, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577790341:2577791347, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149740 fvmb0 In  IP (tos 0x0, ttl 64, id 14953, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x5f36 (correct), ack 2577789335, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149742 fvup0 Out IP (tos 0x0, ttl 63, id 14953, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x02ee (correct), ack 2577789335, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149743 fvmb0 Out IP (tos 0x0, ttl 63, id 31894, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577790341:2577791347, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149757 fvup0 In  IP (tos 0x0, ttl 64, id 31895, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577791347:2577792353, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149761 fvmb0 Out IP (tos 0x0, ttl 63, id 31895, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577791347:2577792353, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149775 fvup0 In  IP (tos 0x0, ttl 64, id 31896, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577792353:2577793359, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149779 fvmb0 Out IP (tos 0x0, ttl 63, id 31896, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577792353:2577793359, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149786 fvup0 In  IP (tos 0x0, ttl 64, id 31897, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577793359:2577794365, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149789 fvmb0 Out IP (tos 0x0, ttl 63, id 31897, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577793359:2577794365, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149794 fvup0 In  IP (tos 0x0, ttl 64, id 31898, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577794365:2577795371, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149796 fvmb0 Out IP (tos 0x0, ttl 63, id 31898, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577794365:2577795371, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149796 fvmb0 In  IP (tos 0x0, ttl 64, id 14954, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x536c (correct), ack 2577792353, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149798 fvup0 Out IP (tos 0x0, ttl 63, id 14954, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xf723 (correct), ack 2577792353, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149804 fvup0 In  IP (tos 0x0, ttl 64, id 31899, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577795371:2577796377, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149806 fvmb0 Out IP (tos 0x0, ttl 63, id 31899, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577795371:2577796377, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149811 fvup0 In  IP (tos 0x0, ttl 64, id 31900, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577796377:2577797383, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149812 fvmb0 Out IP (tos 0x0, ttl 63, id 31900, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577796377:2577797383, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149820 fvup0 In  IP (tos 0x0, ttl 64, id 31901, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577797383:2577798389, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149821 fvmb0 Out IP (tos 0x0, ttl 63, id 31901, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577797383:2577798389, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149827 fvup0 In  IP (tos 0x0, ttl 64, id 31902, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577798389:2577799395, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149829 fvmb0 Out IP (tos 0x0, ttl 63, id 31902, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577798389:2577799395, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149833 fvmb0 In  IP (tos 0x0, ttl 64, id 14955, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x4b90 (correct), ack 2577794365, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149835 fvup0 Out IP (tos 0x0, ttl 63, id 14955, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xef47 (correct), ack 2577794365, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149836 fvup0 In  IP (tos 0x0, ttl 64, id 31903, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577799395:2577800401, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149837 fvmb0 Out IP (tos 0x0, ttl 63, id 31903, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577799395:2577800401, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149838 fvmb0 In  IP (tos 0x0, ttl 64, id 14956, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x43b4 (correct), ack 2577796377, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149839 fvup0 Out IP (tos 0x0, ttl 63, id 14956, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xe76b (correct), ack 2577796377, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149844 fvup0 In  IP (tos 0x0, ttl 64, id 31904, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577800401:2577801407, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149846 fvmb0 Out IP (tos 0x0, ttl 63, id 31904, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577800401:2577801407, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149852 fvmb0 In  IP (tos 0x0, ttl 64, id 14957, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x3bd8 (correct), ack 2577798389, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149853 fvup0 In  IP (tos 0x0, ttl 64, id 31905, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577801407:2577802413, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149854 fvmb0 Out IP (tos 0x0, ttl 63, id 31905, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577801407:2577802413, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149855 fvup0 Out IP (tos 0x0, ttl 63, id 14957, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xdf8f (correct), ack 2577798389, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149860 fvup0 In  IP (tos 0x0, ttl 64, id 31906, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577802413:2577803419, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149862 fvmb0 Out IP (tos 0x0, ttl 63, id 31906, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577802413:2577803419, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149866 fvmb0 In  IP (tos 0x0, ttl 64, id 14958, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x33fc (correct), ack 2577800401, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149868 fvup0 Out IP (tos 0x0, ttl 63, id 14958, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xd7b3 (correct), ack 2577800401, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149869 fvup0 In  IP (tos 0x0, ttl 64, id 31907, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577803419:2577804425, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149871 fvmb0 Out IP (tos 0x0, ttl 63, id 31907, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [P.], seq 2577803419:2577804425, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149885 fvmb0 In  IP (tos 0x0, ttl 64, id 14959, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x2c20 (correct), ack 2577802413, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149887 fvup0 Out IP (tos 0x0, ttl 63, id 14959, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xcfd7 (correct), ack 2577802413, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149899 fvmb0 In  IP (tos 0x0, ttl 64, id 14960, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x2444 (correct), ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149901 fvup0 Out IP (tos 0x0, ttl 63, id 14960, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xc7fb (correct), ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150070 fvmb0 In  IP (tos 0x0, ttl 64, id 14961, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [F.], cksum 0x2443 (correct), seq 1485616547, ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150073 fvup0 Out IP (tos 0x0, ttl 63, id 14961, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [F.], cksum 0xc7fa (correct), seq 1485616547, ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150141 fvup0 In  IP (tos 0x0, ttl 64, id 31908, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [F.], cksum 0x8645 (incorrect -> 0xc7dd), seq 2577804425, ack 1485616548, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 0
1793457043.150143 fvmb0 Out IP (tos 0x0, ttl 63, id 31908, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.78.2.126.56984: Flags [F.], cksum 0x29fd (incorrect -> 0x2426), seq 2577804425, ack 1485616548, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 0
1793457043.150186 fvmb0 In  IP (tos 0x0, ttl 64, id 14962, offset 0, flags [DF], proto TCP (6), length 52)
    100.78.2.126.56984 > 192.0.3.10.9446: Flags [.], cksum 0x2442 (correct), ack 2577804426, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150188 fvup0 Out IP (tos 0x0, ttl 63, id 14962, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xc7f9 (correct), ack 2577804426, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0

- matrix1280_path_1: eth0
2
1500
eth2
4
1500
fvfixture
5
1500
fvmb0
6
1280
fvup0
3
1500
lo
1
65536
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4416,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:25:c1:29:8d:d8","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4417,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"c6:ee:52:4f:5a:4e","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4420,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"ea:8a:b4:de:17:2f","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"ae:1b:a1:20:96:73","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.78.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.78.12.82","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51782
	 cubic wscale:10,10 rto:201 rtt:0.267/0.506 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:24605 bytes_acked:24605 bytes_received:33558 segs_out:43 segs_in:70 data_segs_out:35 data_segs_in:35 send 13653333333bps lastsnd:1679 lastrcv:1686 lastack:1679 pacing_rate 27281122544bps delivery_rate 121514666664bps delivered:36 app_limited busy:94ms rcv_rtt:1 rcv_space:65483 rcv_ssthresh:105032 minrtt:0.003 snd_wnd:91136
ESTAB 0      0       172.28.30.40:45976 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.396/0.343 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5457 bytes_acked:5458 bytes_received:10958 segs_out:115 segs_in:86 data_segs_out:50 data_segs_in:79 send 292525253bps lastsnd:1670 lastrcv:1670 lastack:1670 pacing_rate 584681384bps delivery_rate 3309714280bps delivered:51 app_limited busy:57ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.006 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45986 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.073/0.077 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:16321 bytes_acked:16322 bytes_received:19354 segs_out:143 segs_in:164 data_segs_out:78 data_segs_in:151 send 1586849315bps lastsnd:5910 lastrcv:5910 lastack:5910 pacing_rate 3146757208bps delivery_rate 1654857136bps delivered:79 app_limited busy:46ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.40:45992 172.28.30.21:443
	 cubic wscale:10,10 rto:215 rtt:14.21/19.815 ato:47 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3264 bytes_acked:3265 bytes_received:4760 segs_out:42 segs_in:42 data_segs_out:21 data_segs_in:26 send 8152006bps lastsnd:5883 lastrcv:5883 lastack:5842 pacing_rate 16303576bps delivery_rate 2896000000bps delivered:22 app_limited busy:250ms rcv_space:14480 rcv_ssthresh:85754 minrtt:0.008 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:51796
	 cubic wscale:10,10 rto:203 rtt:2.308/4.59 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:9387 bytes_acked:9387 bytes_received:5962 segs_out:16 segs_in:22 data_segs_out:8 data_segs_in:9 send 1508492201bps lastsnd:26573 lastrcv:26578 lastack:11445 pacing_rate 3016004328bps delivery_rate 34816000000bps delivered:9 app_limited busy:41ms rcv_rtt:1245.38 rcv_space:65483 rcv_ssthresh:78471 minrtt:0.008 snd_wnd:83968
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:55868
	 cubic wscale:10,10 rto:206 rtt:5.079/10.134 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2779 bytes_acked:2779 bytes_received:3998 segs_out:6 segs_in:10 data_segs_out:2 data_segs_in:4 send 685489270bps lastsnd:26770 lastrcv:26782 lastack:11448 pacing_rate 1370877320bps delivery_rate 29013333328bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78342 minrtt:0.012 snd_wnd:87040
ESTAB 0      0          127.0.0.1:36206    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.753/0.047 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:24333 bytes_acked:24334 bytes_received:48600 segs_out:169 segs_in:127 data_segs_out:123 data_segs_in:123 send 3481328021bps lastsnd:1669 lastrcv:1668 lastack:1668 pacing_rate 6955728024bps delivery_rate 29127111104bps delivered:124 app_limited busy:145ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.009 snd_wnd:65536
ESTAB 0      0       172.28.30.40:36842 172.28.30.30:9443
	 cubic wscale:10,10 rto:206 rtt:5.048/2.347 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:40684 bytes_acked:40685 bytes_received:33636 segs_out:110 segs_in:65 data_segs_out:72 data_segs_in:45 send 22947702bps lastsnd:1693 lastrcv:1686 lastack:1686 pacing_rate 45895400bps delivery_rate 2574222216bps delivered:73 app_limited busy:181ms rcv_rtt:6 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.009 snd_wnd:79872
ESTAB 0      0          127.0.0.1:51782    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.393/3.09 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:33558 bytes_acked:33559 bytes_received:24605 segs_out:70 segs_in:44 data_segs_out:35 data_segs_in:35 send 659921789bps lastsnd:1694 lastrcv:1687 lastack:1687 pacing_rate 1319662952bps delivery_rate 38353454544bps delivered:36 app_limited busy:198ms rcv_rtt:12 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.004 snd_wnd:105472
ESTAB 0      0          127.0.0.1:51796    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.101/7.042 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:5962 bytes_acked:5963 bytes_received:9387 segs_out:22 segs_in:16 data_segs_out:9 data_segs_in:8 send 516951319bps lastsnd:26584 lastrcv:26579 lastack:11452 pacing_rate 1033754376bps delivery_rate 28672000000bps delivered:10 app_limited busy:71ms rcv_rtt:5 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.01 snd_wnd:78848
ESTAB 0      0          127.0.0.1:55868    127.0.0.1:9444
	 cubic wscale:10,10 rto:202 rtt:1.6/3.153 ato:40 mss:39424 pmtu:65535 rcvmss:1996 advmss:65483 cwnd:10 bytes_sent:3998 bytes_acked:3999 bytes_received:2779 segs_out:10 segs_in:7 data_segs_out:4 data_segs_in:2 send 1971200000bps lastsnd:26785 lastrcv:26773 lastack:11451 pacing_rate 3941784096bps delivery_rate 28672000000bps delivered:5 app_limited busy:13ms rcv_space:65495 rcv_ssthresh:86892 minrtt:0.011 snd_wnd:78848
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:36206
	 cubic wscale:10,10 rto:211 rtt:10.497/16.882 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:48600 bytes_acked:48600 bytes_received:24333 segs_out:126 segs_in:169 data_segs_out:123 data_segs_in:123 send 249732304bps lastsnd:1672 lastrcv:1673 lastack:1631 pacing_rate 499422976bps delivery_rate 29127111104bps delivered:124 app_limited busy:914ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.008 snd_wnd:65536

- matrix1280_capture_1_receipt: {"bytes":"65723","sha256":"7cee916282215a193081329ad8c984a09e8229b0c5427294009da8b7fb4963de"}
- matrix1280_capture_2_window: {"launch_requested_at":"2026-10-31T14:30:41.920201837Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:30:44.698508134Z","stop_requested_at":"2026-10-31T14:30:44.614518087Z"}
- matrix1280_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
138 packets captured
138 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793457043.148251 eth0  In  IP (tos 0x0, ttl 63, id 14875, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [S], cksum 0x39e7 (correct), seq 1485552438, win 64480, options [mss 1240,sackOK,TS val 2154689859 ecr 0,nop,wscale 10], length 0
1793457043.148266 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [S.], cksum 0x864d (incorrect -> 0x90bf), seq 2577740037, ack 1485552439, win 65160, options [mss 1460,sackOK,TS val 2952145649 ecr 2154689859,nop,wscale 10], length 0
1793457043.148380 eth0  In  IP (tos 0x0, ttl 63, id 14876, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xbdd8 (correct), ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148436 eth0  In  IP (tos 0x0, ttl 63, id 14877, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], cksum 0xcefc (correct), seq 1485552439:1485552483, ack 2577740038, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 44
1793457043.148445 eth0  Out IP (tos 0x0, ttl 64, id 31836, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xbdab), ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 0
1793457043.148487 eth0  Out IP (tos 0x0, ttl 64, id 31837, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], cksum 0x8648 (incorrect -> 0x6455), seq 2577740038:2577740041, ack 1485552483, win 64, options [nop,nop,TS val 2952145649 ecr 2154689859], length 3
1793457043.148583 eth0  In  IP (tos 0x0, ttl 63, id 14878, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xbda9 (correct), ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 0
1793457043.148668 eth0  In  IP (tos 0x0, ttl 63, id 14879, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485552483:1485553711, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148681 eth0  In  IP (tos 0x0, ttl 63, id 14880, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485553711:1485558623, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 4912
1793457043.148689 eth0  In  IP (tos 0x0, ttl 63, id 14884, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485558623:1485559851, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148690 eth0  Out IP (tos 0x0, ttl 64, id 31838, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xa5ab), ack 1485558623, win 64, options [nop,nop,TS val 2952145650 ecr 2154689859], length 0
1793457043.148694 eth0  In  IP (tos 0x0, ttl 63, id 14885, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485559851:1485561079, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148706 eth0  In  IP (tos 0x0, ttl 63, id 14886, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485561079:1485563535, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 2456
1793457043.148732 eth0  In  IP (tos 0x0, ttl 63, id 14888, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485563535:1485564763, ack 2577740041, win 63, options [nop,nop,TS val 2154689859 ecr 2952145649], length 1228
1793457043.148744 eth0  Out IP (tos 0x0, ttl 64, id 31839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577740041:2577741047, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148768 eth0  Out IP (tos 0x0, ttl 64, id 31840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577741047:2577742053, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148789 eth0  Out IP (tos 0x0, ttl 64, id 31841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577742053:2577743059, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148805 eth0  Out IP (tos 0x0, ttl 64, id 31842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577743059:2577744065, ack 1485564763, win 59, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148825 eth0  Out IP (tos 0x0, ttl 64, id 31843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577744065:2577745071, ack 1485564763, win 63, options [nop,nop,TS val 2952145650 ecr 2154689859], length 1006
1793457043.148827 eth0  In  IP (tos 0x0, ttl 63, id 14889, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485564763:1485565991, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148831 eth0  In  IP (tos 0x0, ttl 63, id 14890, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485565991:1485567219, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148843 eth0  In  IP (tos 0x0, ttl 63, id 14891, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485567219:1485570903, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148846 eth0  Out IP (tos 0x0, ttl 64, id 31844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577745071:2577746077, ack 1485567219, win 61, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148859 eth0  Out IP (tos 0x0, ttl 64, id 31845, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0x5e24), ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148875 eth0  Out IP (tos 0x0, ttl 64, id 31846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577746077:2577747083, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148892 eth0  Out IP (tos 0x0, ttl 64, id 31847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577747083:2577748089, ack 1485570903, win 58, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148894 eth0  In  IP (tos 0x0, ttl 63, id 14894, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485570903:1485573359, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.148898 eth0  In  IP (tos 0x0, ttl 63, id 14896, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485573359:1485574587, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148909 eth0  Out IP (tos 0x0, ttl 64, id 31848, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0x47e7), ack 1485574587, win 55, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148915 eth0  In  IP (tos 0x0, ttl 63, id 14897, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485574587:1485577043, ack 2577740041, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.148917 eth0  In  IP (tos 0x0, ttl 63, id 14899, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485577043:1485580727, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148927 eth0  In  IP (tos 0x0, ttl 63, id 14902, offset 0, flags [DF], proto TCP (6), length 7420)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485580727:1485588095, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 7368
1793457043.148936 eth0  Out IP (tos 0x0, ttl 64, id 31849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577748089:2577749095, ack 1485588095, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148947 eth0  In  IP (tos 0x0, ttl 63, id 14908, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485588095:1485589323, ack 2577741047, win 63, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148949 eth0  In  IP (tos 0x0, ttl 63, id 14909, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x21f6 (correct), ack 2577743059, win 62, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148953 eth0  In  IP (tos 0x0, ttl 63, id 14910, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x1a1b (correct), ack 2577745071, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.148955 eth0  Out IP (tos 0x0, ttl 64, id 31850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577749095:2577750101, ack 1485589323, win 41, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148965 eth0  In  IP (tos 0x0, ttl 63, id 14911, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485589323:1485594235, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 4912
1793457043.148968 eth0  In  IP (tos 0x0, ttl 63, id 14915, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485594235:1485595463, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148972 eth0  Out IP (tos 0x0, ttl 64, id 31851, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xf35a), ack 1485594235, win 39, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.148975 eth0  In  IP (tos 0x0, ttl 63, id 14916, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485595463:1485596691, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148984 eth0  In  IP (tos 0x0, ttl 63, id 14917, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485596691:1485600375, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.148993 eth0  Out IP (tos 0x0, ttl 64, id 31852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577750101:2577751107, ack 1485600375, win 34, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.148996 eth0  In  IP (tos 0x0, ttl 63, id 14920, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485600375:1485601603, ack 2577746077, win 61, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.148997 eth0  In  IP (tos 0x0, ttl 63, id 14921, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xde59 (correct), ack 2577748089, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149012 eth0  Out IP (tos 0x0, ttl 64, id 31853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577751107:2577752113, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149034 eth0  Out IP (tos 0x0, ttl 64, id 31854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577752113:2577753119, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149053 eth0  In  IP (tos 0x0, ttl 63, id 14922, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485601603:1485602831, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149054 eth0  Out IP (tos 0x0, ttl 64, id 31855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577753119:2577754125, ack 1485601603, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149063 eth0  In  IP (tos 0x0, ttl 63, id 14923, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485602831:1485604059, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149070 eth0  In  IP (tos 0x0, ttl 63, id 14924, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485604059:1485606515, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.149073 eth0  Out IP (tos 0x0, ttl 64, id 31856, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0xbd4a), ack 1485604059, win 31, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149089 eth0  Out IP (tos 0x0, ttl 64, id 31857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577754125:2577755131, ack 1485606515, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149091 eth0  In  IP (tos 0x0, ttl 63, id 14926, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485606515:1485610199, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 3684
1793457043.149097 eth0  In  IP (tos 0x0, ttl 63, id 14929, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485610199:1485611427, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149104 eth0  In  IP (tos 0x0, ttl 63, id 14930, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485611427:1485612655, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149108 eth0  Out IP (tos 0x0, ttl 64, id 31858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577755131:2577756137, ack 1485612655, win 24, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149112 eth0  In  IP (tos 0x0, ttl 63, id 14931, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485612655:1485615111, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 2456
1793457043.149117 eth0  In  IP (tos 0x0, ttl 63, id 14933, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], seq 1485615111:1485616339, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 1228
1793457043.149123 eth0  Out IP (tos 0x0, ttl 64, id 31859, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [.], cksum 0x8645 (incorrect -> 0x857e), ack 1485616339, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 0
1793457043.149128 eth0  In  IP (tos 0x0, ttl 63, id 14934, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [P.], seq 1485616339:1485616547, ack 2577749095, win 60, options [nop,nop,TS val 2154689860 ecr 2952145650], length 208
1793457043.149130 eth0  In  IP (tos 0x0, ttl 63, id 14935, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x9830 (correct), ack 2577751107, win 59, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149137 eth0  In  IP (tos 0x0, ttl 63, id 14936, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x9055 (correct), ack 2577753119, win 58, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149142 eth0  Out IP (tos 0x0, ttl 64, id 31860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577756137:2577757143, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149145 eth0  In  IP (tos 0x0, ttl 63, id 14937, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x887a (correct), ack 2577755131, win 57, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149161 eth0  Out IP (tos 0x0, ttl 64, id 31861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577757143:2577758149, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149181 eth0  Out IP (tos 0x0, ttl 64, id 31862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577758149:2577759155, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149190 eth0  In  IP (tos 0x0, ttl 63, id 14938, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x809f (correct), ack 2577757143, win 56, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149196 eth0  Out IP (tos 0x0, ttl 64, id 31863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577759155:2577760161, ack 1485616547, win 23, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149214 eth0  Out IP (tos 0x0, ttl 64, id 31864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577760161:2577761167, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149229 eth0  Out IP (tos 0x0, ttl 64, id 31865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577761167:2577762173, ack 1485616547, win 27, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149240 eth0  In  IP (tos 0x0, ttl 63, id 14939, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x78c4 (correct), ack 2577759155, win 55, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149249 eth0  Out IP (tos 0x0, ttl 64, id 31866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577762173:2577763179, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149259 eth0  In  IP (tos 0x0, ttl 63, id 14940, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x70e9 (correct), ack 2577761167, win 54, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149268 eth0  Out IP (tos 0x0, ttl 64, id 31867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577763179:2577764185, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149287 eth0  Out IP (tos 0x0, ttl 64, id 31868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577764185:2577765191, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149301 eth0  Out IP (tos 0x0, ttl 64, id 31869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577765191:2577766197, ack 1485616547, win 29, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149321 eth0  Out IP (tos 0x0, ttl 64, id 31870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577766197:2577767203, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149322 eth0  In  IP (tos 0x0, ttl 63, id 14941, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x690e (correct), ack 2577763179, win 53, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149336 eth0  Out IP (tos 0x0, ttl 64, id 31871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577767203:2577768209, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149342 eth0  In  IP (tos 0x0, ttl 63, id 14942, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x6133 (correct), ack 2577765191, win 52, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149353 eth0  Out IP (tos 0x0, ttl 64, id 31872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577768209:2577769215, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149368 eth0  Out IP (tos 0x0, ttl 64, id 31873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577769215:2577770221, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149385 eth0  Out IP (tos 0x0, ttl 64, id 31874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577770221:2577771227, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149386 eth0  In  IP (tos 0x0, ttl 63, id 14943, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x556b (correct), ack 2577768209, win 50, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149402 eth0  Out IP (tos 0x0, ttl 64, id 31875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577771227:2577772233, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149426 eth0  In  IP (tos 0x0, ttl 63, id 14944, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x4d90 (correct), ack 2577770221, win 49, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149429 eth0  Out IP (tos 0x0, ttl 64, id 31876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577772233:2577773239, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149445 eth0  Out IP (tos 0x0, ttl 64, id 31877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577773239:2577774245, ack 1485616547, win 33, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149462 eth0  Out IP (tos 0x0, ttl 64, id 31878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577774245:2577775251, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149478 eth0  Out IP (tos 0x0, ttl 64, id 31879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577775251:2577776257, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149489 eth0  In  IP (tos 0x0, ttl 63, id 14945, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x45b5 (correct), ack 2577772233, win 48, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149495 eth0  In  IP (tos 0x0, ttl 63, id 14946, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x3dda (correct), ack 2577774245, win 47, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149499 eth0  Out IP (tos 0x0, ttl 64, id 31880, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577776257:2577777263, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149515 eth0  Out IP (tos 0x0, ttl 64, id 31881, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577777263:2577778269, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149518 eth0  In  IP (tos 0x0, ttl 63, id 14947, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x35ff (correct), ack 2577776257, win 46, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149535 eth0  Out IP (tos 0x0, ttl 64, id 31882, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577778269:2577779275, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149549 eth0  Out IP (tos 0x0, ttl 64, id 31883, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577779275:2577780281, ack 1485616547, win 42, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149557 eth0  In  IP (tos 0x0, ttl 63, id 14948, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x2e24 (correct), ack 2577778269, win 45, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149567 eth0  Out IP (tos 0x0, ttl 64, id 31884, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577780281:2577781287, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149582 eth0  Out IP (tos 0x0, ttl 64, id 31885, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577781287:2577782293, ack 1485616547, win 48, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149599 eth0  In  IP (tos 0x0, ttl 63, id 14949, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x2649 (correct), ack 2577780281, win 44, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149599 eth0  Out IP (tos 0x0, ttl 64, id 31886, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577782293:2577783299, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149615 eth0  Out IP (tos 0x0, ttl 64, id 31887, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577783299:2577784305, ack 1485616547, win 49, options [nop,nop,TS val 2952145650 ecr 2154689860], length 1006
1793457043.149628 eth0  In  IP (tos 0x0, ttl 63, id 14950, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x1e6e (correct), ack 2577782293, win 43, options [nop,nop,TS val 2154689860 ecr 2952145650], length 0
1793457043.149635 eth0  Out IP (tos 0x0, ttl 64, id 31888, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577784305:2577785311, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149654 eth0  Out IP (tos 0x0, ttl 64, id 31889, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577785311:2577786317, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149672 eth0  Out IP (tos 0x0, ttl 64, id 31890, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577786317:2577787323, ack 1485616547, win 49, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149690 eth0  Out IP (tos 0x0, ttl 64, id 31891, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577787323:2577788329, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689860], length 1006
1793457043.149698 eth0  In  IP (tos 0x0, ttl 63, id 14951, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x1692 (correct), ack 2577784305, win 42, options [nop,nop,TS val 2154689861 ecr 2952145650], length 0
1793457043.149700 eth0  In  IP (tos 0x0, ttl 63, id 14952, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x0eb6 (correct), ack 2577786317, win 41, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149705 eth0  Out IP (tos 0x0, ttl 64, id 31892, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577788329:2577789335, ack 1485616547, win 54, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149722 eth0  Out IP (tos 0x0, ttl 64, id 31893, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577789335:2577790341, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149737 eth0  Out IP (tos 0x0, ttl 64, id 31894, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577790341:2577791347, ack 1485616547, win 55, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149743 eth0  In  IP (tos 0x0, ttl 63, id 14953, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0x02ee (correct), ack 2577789335, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149756 eth0  Out IP (tos 0x0, ttl 64, id 31895, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577791347:2577792353, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149774 eth0  Out IP (tos 0x0, ttl 64, id 31896, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577792353:2577793359, ack 1485616547, win 57, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149785 eth0  Out IP (tos 0x0, ttl 64, id 31897, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577793359:2577794365, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149793 eth0  Out IP (tos 0x0, ttl 64, id 31898, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577794365:2577795371, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149799 eth0  In  IP (tos 0x0, ttl 63, id 14954, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xf723 (correct), ack 2577792353, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149803 eth0  Out IP (tos 0x0, ttl 64, id 31899, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577795371:2577796377, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149810 eth0  Out IP (tos 0x0, ttl 64, id 31900, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577796377:2577797383, ack 1485616547, win 59, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149819 eth0  Out IP (tos 0x0, ttl 64, id 31901, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577797383:2577798389, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149826 eth0  Out IP (tos 0x0, ttl 64, id 31902, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577798389:2577799395, ack 1485616547, win 64, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149835 eth0  Out IP (tos 0x0, ttl 64, id 31903, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577799395:2577800401, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149837 eth0  In  IP (tos 0x0, ttl 63, id 14955, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xef47 (correct), ack 2577794365, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149840 eth0  In  IP (tos 0x0, ttl 63, id 14956, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xe76b (correct), ack 2577796377, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149843 eth0  Out IP (tos 0x0, ttl 64, id 31904, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577800401:2577801407, ack 1485616547, win 65, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149851 eth0  Out IP (tos 0x0, ttl 64, id 31905, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577801407:2577802413, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149856 eth0  In  IP (tos 0x0, ttl 63, id 14957, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xdf8f (correct), ack 2577798389, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149859 eth0  Out IP (tos 0x0, ttl 64, id 31906, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577802413:2577803419, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149868 eth0  Out IP (tos 0x0, ttl 64, id 31907, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [P.], seq 2577803419:2577804425, ack 1485616547, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 1006
1793457043.149869 eth0  In  IP (tos 0x0, ttl 63, id 14958, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xd7b3 (correct), ack 2577800401, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149887 eth0  In  IP (tos 0x0, ttl 63, id 14959, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xcfd7 (correct), ack 2577802413, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.149902 eth0  In  IP (tos 0x0, ttl 63, id 14960, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xc7fb (correct), ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150073 eth0  In  IP (tos 0x0, ttl 63, id 14961, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [F.], cksum 0xc7fa (correct), seq 1485616547, ack 2577804425, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0
1793457043.150139 eth0  Out IP (tos 0x0, ttl 64, id 31908, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.56984: Flags [F.], cksum 0x8645 (incorrect -> 0xc7dd), seq 2577804425, ack 1485616548, win 67, options [nop,nop,TS val 2952145651 ecr 2154689861], length 0
1793457043.150189 eth0  In  IP (tos 0x0, ttl 63, id 14962, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.56984 > 192.0.3.10.9446: Flags [.], cksum 0xc7f9 (correct), ack 2577804426, win 39, options [nop,nop,TS val 2154689861 ecr 2952145651], length 0

- matrix1280_path_2: eth0
2
1500
lo
1
65536
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ip: not found
sh: 1: ss: not found

- matrix1280_capture_2_receipt: {"bytes":"17439","sha256":"4033fea3bbe2e5f976a3f63644ef036caf822563a1635b77431afc4d2755de07"}
- matrix1280_origin_endpoint_samples: {"at":"2026-10-31T14:30:22.307266462Z","local_port":9446,"remote_port":42202,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":86,"sacked":0,"snd_cwnd":25,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:30:29.185133874Z","local_port":9446,"remote_port":40370,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":63,"sacked":0,"snd_cwnd":26,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:30:36.168846529Z","local_port":9446,"remote_port":56968,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":82,"sacked":0,"snd_cwnd":28,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:30:43.150084579Z","local_port":9446,"remote_port":56984,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":64,"sacked":0,"snd_cwnd":16,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}

- topology: The measured bottleneck is the member's eth0/eth1 MTU. The any-interface pcaps include both underlays and the tunnel, node tunnel/uplink, and controlled origin. No separate bottleneck router exists; no host networking touched.
- D123: IPv4 overlay over the observed native underlay. IPv6Block no-leak is independently proven in docs/poc/G2-f802-mtu1280-2026-10-31.md; no IPv6 overlay success is claimed or required.
