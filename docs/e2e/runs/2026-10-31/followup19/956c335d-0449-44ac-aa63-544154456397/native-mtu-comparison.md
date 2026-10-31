# native-mtu-comparison

F8-02 exact reconnect ordering with endpoint and complete header evidence

Status: pass

Run: 956c335d-0449-44ac-aa63-544154456397

Source: 706d7cf6cec8c00c209dbe029eaa7825921f12ac

Spec: AT09, AT11, F8-02

- matrix4_input: {"input":{"underlay":"ipv4","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix4_transfer: {"after":{"upload_bytes":"68457","download_bytes":"69361","Present":true},"before":{"upload_bytes":"1533","download_bytes":"1638","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:19:35.727837219Z","local_port":37946,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":184,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":67584,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":3},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- matrix4_capture_0_window: {"launch_requested_at":"2026-10-31T14:19:34.432058052Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:36.088898899Z","stop_requested_at":"2026-10-31T14:19:35.884575177Z"}
- matrix4_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
269 packets captured
273 packets received by filter
0 packets dropped by kernel

- matrix4_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456375.724380 fvnb0 Out IP (tos 0x0, ttl 64, id 48944, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [S], cksum 0xa804 (incorrect -> 0x9255), seq 3669307176, win 64480, options [mss 1240,sackOK,TS val 101730750 ecr 0,nop,wscale 10], length 0
1793456375.724572 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [S.], cksum 0xc6be (correct), seq 3715411276, ack 3669307177, win 65160, options [mss 1460,sackOK,TS val 197800310 ecr 101730750,nop,wscale 10], length 0
1793456375.724590 fvnb0 Out IP (tos 0x0, ttl 64, id 48945, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xf3d7), ack 3715411277, win 63, options [nop,nop,TS val 101730750 ecr 197800310], length 0
1793456375.724648 fvnb0 Out IP (tos 0x0, ttl 64, id 48946, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], cksum 0xa828 (incorrect -> 0xa25a), seq 3669307177:3669307221, ack 3715411277, win 63, options [nop,nop,TS val 101730751 ecr 197800310], length 44
1793456375.726574 fvnb0 In  IP (tos 0x0, ttl 63, id 37183, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xf3a7 (correct), ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 0
1793456375.726575 fvnb0 In  IP (tos 0x0, ttl 63, id 37184, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], cksum 0x9a51 (correct), seq 3715411277:3715411280, ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 3
1793456375.726582 fvnb0 Out IP (tos 0x0, ttl 64, id 48947, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xf3a4), ack 3715411280, win 63, options [nop,nop,TS val 101730752 ecr 197800312], length 0
1793456375.726667 fvnb0 Out IP (tos 0x0, ttl 64, id 48948, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669307221:3669313361, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.726684 fvnb0 Out IP (tos 0x0, ttl 64, id 48953, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669313361:3669319501, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.727106 fvnb0 In  IP (tos 0x0, ttl 63, id 37185, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xc3b0 (correct), ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727110 fvnb0 Out IP (tos 0x0, ttl 64, id 48958, offset 0, flags [DF], proto TCP (6), length 12332)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669319501:3669331781, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 12280
1793456375.727120 fvnb0 Out IP (tos 0x0, ttl 64, id 48968, offset 0, flags [DF], proto TCP (6), length 12332)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669331781:3669344061, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 12280
1793456375.727128 fvnb0 In  IP (tos 0x0, ttl 63, id 37186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715411280:3715412286, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727129 fvnb0 In  IP (tos 0x0, ttl 63, id 37187, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715412286:3715413292, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727129 fvnb0 In  IP (tos 0x0, ttl 63, id 37188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715413292:3715414298, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727129 fvnb0 In  IP (tos 0x0, ttl 63, id 37189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715414298:3715415304, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727136 fvnb0 In  IP (tos 0x0, ttl 63, id 37190, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715415304:3715416310, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727136 fvnb0 In  IP (tos 0x0, ttl 63, id 37191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715416310:3715417316, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727137 fvnb0 In  IP (tos 0x0, ttl 63, id 37192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715417316:3715418322, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727137 fvnb0 In  IP (tos 0x0, ttl 63, id 37193, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715418322:3715419328, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727139 fvnb0 Out IP (tos 0x0, ttl 64, id 48978, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x5405), ack 3715415304, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727141 fvnb0 In  IP (tos 0x0, ttl 63, id 37194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715419328:3715420334, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727141 fvnb0 In  IP (tos 0x0, ttl 63, id 37195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715420334:3715421340, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727157 fvnb0 Out IP (tos 0x0, ttl 64, id 48979, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x3c76), ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727229 fvnb0 In  IP (tos 0x0, ttl 63, id 37196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0x92c3 (correct), ack 3669321957, win 66, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727230 fvnb0 In  IP (tos 0x0, ttl 63, id 37197, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0x7aca (correct), ack 3669328097, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727234 fvnb0 Out IP (tos 0x0, ttl 64, id 48980, offset 0, flags [DF], proto TCP (6), length 4964)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669344061:3669348973, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 4912
1793456375.727239 fvnb0 Out IP (tos 0x0, ttl 64, id 48984, offset 0, flags [DF], proto TCP (6), length 12332)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669348973:3669361253, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 12280
1793456375.727246 fvnb0 In  IP (tos 0x0, ttl 63, id 37198, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0x593e (correct), ack 3669336693, win 55, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727249 fvnb0 Out IP (tos 0x0, ttl 64, id 48994, offset 0, flags [DF], proto TCP (6), length 10084)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669361253:3669371285, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 10032
1793456375.727256 fvnb0 In  IP (tos 0x0, ttl 63, id 37199, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0x4fa3 (correct), ack 3669339149, win 58, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727289 fvnb0 In  IP (tos 0x0, ttl 63, id 37200, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], seq 3715421340:3715423796, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 2456
1793456375.727294 fvnb0 Out IP (tos 0x0, ttl 64, id 49003, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xc880), ack 3715423796, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727296 fvnb0 In  IP (tos 0x0, ttl 63, id 37202, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], seq 3715423796:3715425024, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 1228
1793456375.727302 fvnb0 In  IP (tos 0x0, ttl 63, id 37203, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], seq 3715425024:3715427480, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 2456
1793456375.727304 fvnb0 Out IP (tos 0x0, ttl 64, id 49004, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xba16), ack 3715427480, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727308 fvnb0 In  IP (tos 0x0, ttl 63, id 37205, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], seq 3715427480:3715428708, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 1228
1793456375.727309 fvnb0 In  IP (tos 0x0, ttl 63, id 37206, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715428708:3715429936, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 1228
1793456375.727317 fvnb0 In  IP (tos 0x0, ttl 63, id 37207, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], seq 3715429936:3715432392, ack 3669344061, win 56, options [nop,nop,TS val 197800313 ecr 101730753], length 2456
1793456375.727318 fvnb0 Out IP (tos 0x0, ttl 64, id 49005, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xb07e), ack 3715429936, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727319 fvnb0 In  IP (tos 0x0, ttl 63, id 37209, offset 0, flags [DF], proto TCP (6), length 1072)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715432392:3715433412, ack 3669344061, win 56, options [nop,nop,TS val 197800313 ecr 101730753], length 1020
1793456375.727327 fvnb0 In  IP (tos 0x0, ttl 63, id 37210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715433412:3715434418, ack 3669344061, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727328 fvnb0 In  IP (tos 0x0, ttl 63, id 37211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715434418:3715435424, ack 3669347745, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727328 fvnb0 In  IP (tos 0x0, ttl 63, id 37212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715435424:3715436430, ack 3669356341, win 47, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727332 fvnb0 In  IP (tos 0x0, ttl 63, id 37213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715436430:3715437436, ack 3669361253, win 43, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727336 fvnb0 Out IP (tos 0x0, ttl 64, id 49006, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x9725), ack 3715436430, win 61, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727344 fvnb0 In  IP (tos 0x0, ttl 63, id 37214, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xac18 (correct), ack 3669364937, win 40, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727354 fvnb0 In  IP (tos 0x0, ttl 63, id 37215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715437436:3715438442, ack 3669368621, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727356 fvnb0 Out IP (tos 0x0, ttl 64, id 49007, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x8f43), ack 3715438442, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727367 fvnb0 In  IP (tos 0x0, ttl 63, id 37216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715438442:3715439448, ack 3669369849, win 36, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727384 fvnb0 In  IP (tos 0x0, ttl 63, id 37217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715439448:3715440454, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727387 fvnb0 Out IP (tos 0x0, ttl 64, id 49008, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x8767), ack 3715440454, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727397 fvnb0 In  IP (tos 0x0, ttl 63, id 37218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715440454:3715441460, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727405 fvnb0 In  IP (tos 0x0, ttl 63, id 37219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715441460:3715442466, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727408 fvnb0 Out IP (tos 0x0, ttl 64, id 49009, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x7f8b), ack 3715442466, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727418 fvnb0 In  IP (tos 0x0, ttl 63, id 37220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715442466:3715443472, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727439 fvnb0 In  IP (tos 0x0, ttl 63, id 37221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715443472:3715444478, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727442 fvnb0 Out IP (tos 0x0, ttl 64, id 49010, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x77af), ack 3715444478, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727458 fvnb0 In  IP (tos 0x0, ttl 63, id 37222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715444478:3715445484, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727469 fvnb0 In  IP (tos 0x0, ttl 63, id 37223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715445484:3715446490, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727472 fvnb0 Out IP (tos 0x0, ttl 64, id 49011, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x6fd3), ack 3715446490, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727492 fvnb0 In  IP (tos 0x0, ttl 63, id 37224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715446490:3715447496, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727498 fvnb0 In  IP (tos 0x0, ttl 63, id 37225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715447496:3715448502, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727501 fvnb0 Out IP (tos 0x0, ttl 64, id 49012, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x67f7), ack 3715448502, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727516 fvnb0 In  IP (tos 0x0, ttl 63, id 37226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715448502:3715449508, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727545 fvnb0 In  IP (tos 0x0, ttl 63, id 37227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715449508:3715450514, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727548 fvnb0 Out IP (tos 0x0, ttl 64, id 49013, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x601b), ack 3715450514, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727553 fvnb0 In  IP (tos 0x0, ttl 63, id 37228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715450514:3715451520, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727585 fvnb0 In  IP (tos 0x0, ttl 63, id 37229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715451520:3715452526, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727588 fvnb0 Out IP (tos 0x0, ttl 64, id 49014, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x583f), ack 3715452526, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727590 fvnb0 In  IP (tos 0x0, ttl 63, id 37230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715452526:3715453532, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727596 fvnb0 In  IP (tos 0x0, ttl 63, id 37231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715453532:3715454538, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727597 fvnb0 In  IP (tos 0x0, ttl 63, id 37232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715454538:3715455544, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727599 fvnb0 Out IP (tos 0x0, ttl 64, id 49015, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x5063), ack 3715454538, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727607 fvnb0 In  IP (tos 0x0, ttl 63, id 37233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715455544:3715456550, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727610 fvnb0 Out IP (tos 0x0, ttl 64, id 49016, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x4887), ack 3715456550, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727621 fvnb0 In  IP (tos 0x0, ttl 63, id 37234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715456550:3715457556, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727628 fvnb0 In  IP (tos 0x0, ttl 63, id 37235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715457556:3715458562, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727632 fvnb0 Out IP (tos 0x0, ttl 64, id 49017, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x40aa), ack 3715458562, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727647 fvnb0 In  IP (tos 0x0, ttl 63, id 37236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715458562:3715459568, ack 3669371285, win 53, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727652 fvnb0 In  IP (tos 0x0, ttl 63, id 37237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715459568:3715460574, ack 3669371285, win 53, options [nop,nop,TS val 197800314 ecr 101730753], length 1006
1793456375.727656 fvnb0 Out IP (tos 0x0, ttl 64, id 49018, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x38ce), ack 3715460574, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727680 fvnb0 In  IP (tos 0x0, ttl 63, id 37238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715460574:3715461580, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727681 fvnb0 In  IP (tos 0x0, ttl 63, id 37239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715461580:3715462586, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727688 fvnb0 In  IP (tos 0x0, ttl 63, id 37240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715462586:3715463592, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727694 fvnb0 Out IP (tos 0x0, ttl 64, id 49019, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x2d03), ack 3715463592, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727709 fvnb0 In  IP (tos 0x0, ttl 63, id 37241, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715463592:3715464598, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727710 fvnb0 In  IP (tos 0x0, ttl 63, id 37242, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715464598:3715465604, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727716 fvnb0 Out IP (tos 0x0, ttl 64, id 49020, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x2527), ack 3715465604, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727734 fvnb0 In  IP (tos 0x0, ttl 63, id 37243, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715465604:3715466610, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727734 fvnb0 In  IP (tos 0x0, ttl 63, id 37244, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715466610:3715467616, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727740 fvnb0 In  IP (tos 0x0, ttl 63, id 37245, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715467616:3715468622, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727742 fvnb0 Out IP (tos 0x0, ttl 64, id 49021, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x1d4b), ack 3715467616, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727758 fvnb0 In  IP (tos 0x0, ttl 63, id 37246, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715468622:3715469628, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727762 fvnb0 Out IP (tos 0x0, ttl 64, id 49022, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x156f), ack 3715469628, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727765 fvnb0 In  IP (tos 0x0, ttl 63, id 37247, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715469628:3715470634, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727782 fvnb0 In  IP (tos 0x0, ttl 63, id 37248, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715470634:3715471640, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727786 fvnb0 Out IP (tos 0x0, ttl 64, id 49023, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x0d93), ack 3715471640, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727787 fvnb0 In  IP (tos 0x0, ttl 63, id 37249, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715471640:3715472646, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727803 fvnb0 In  IP (tos 0x0, ttl 63, id 37250, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715472646:3715473652, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727806 fvnb0 Out IP (tos 0x0, ttl 64, id 49024, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x05b7), ack 3715473652, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727818 fvnb0 In  IP (tos 0x0, ttl 63, id 37251, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715473652:3715474658, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727829 fvnb0 In  IP (tos 0x0, ttl 63, id 37252, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715474658:3715475664, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727833 fvnb0 Out IP (tos 0x0, ttl 64, id 49025, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xfdda), ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728039 fvnb0 Out IP (tos 0x0, ttl 64, id 49026, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [F.], cksum 0xa7fc (incorrect -> 0xfdd9), seq 3669371285, ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728543 fvnb0 In  IP (tos 0x0, ttl 63, id 37253, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [F.], cksum 0xfdd9 (correct), seq 3715475664, ack 3669371286, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 0
1793456375.728550 fvnb0 Out IP (tos 0x0, ttl 64, id 49027, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xfdd8), ack 3715475665, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4320,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:6d:bb:cd:a6:42","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4323,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"62:1a:d1:c5:55:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.83.128.120","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:33578 172.28.30.21:443
	 cubic wscale:10,10 rto:203 rtt:2.795/5.289 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3551 bytes_acked:3552 bytes_received:5722 segs_out:36 segs_in:35 data_segs_out:20 data_segs_in:29 send 41445438bps lastsnd:627 lastrcv:626 lastack:626 pacing_rate 82890872bps delivery_rate 2896000000bps delivered:21 app_limited busy:46ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.41:33588 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.056/2.05 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3934 bytes_acked:3935 bytes_received:6349 segs_out:38 segs_in:38 data_segs_out:21 data_segs_in:34 send 109696970bps lastsnd:5370 lastrcv:5370 lastack:5370 pacing_rate 219238224bps delivery_rate 2896000000bps delivered:22 app_limited busy:38ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.41:33594 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.161/10.212 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2928 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 22445263bps lastsnd:5371 lastrcv:5371 lastack:5330 pacing_rate 44890520bps delivery_rate 2896000000bps delivered:9 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:79828 minrtt:0.008 snd_wnd:74752
ESTAB 0      0       172.28.30.41:45832 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.061/0.084 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5113 segs_out:16 segs_in:16 data_segs_out:9 data_segs_in:11 send 1899016393bps lastsnd:31557 lastrcv:31556 lastack:1386 pacing_rate 3744323232bps delivery_rate 3861333328bps delivered:10 app_limited rcv_space:14480 rcv_ssthresh:81050 minrtt:0.006 snd_wnd:71680

- matrix4_capture_0_receipt: {"bytes":"36332","sha256":"720bf898456aeb5b16923c7dd2103ec72819858c0a3069d238d834f92f881e29"}
- matrix4_capture_1_window: {"launch_requested_at":"2026-10-31T14:19:34.465638104Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:36.443955833Z","stop_requested_at":"2026-10-31T14:19:36.278448831Z"}
- matrix4_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
398 packets captured
398 packets received by filter
0 packets dropped by kernel

- matrix4_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456375.724458 fvmb0 In  IP (tos 0x0, ttl 64, id 48944, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [S], cksum 0x9255 (correct), seq 3669307176, win 64480, options [mss 1240,sackOK,TS val 101730750 ecr 0,nop,wscale 10], length 0
1793456375.724491 fvup0 Out IP (tos 0x0, ttl 63, id 48944, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [S], cksum 0xb40c (correct), seq 3669307176, win 64480, options [mss 1240,sackOK,TS val 101730750 ecr 0,nop,wscale 10], length 0
1793456375.724523 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [S.], cksum 0x864d (incorrect -> 0xe875), seq 3715411276, ack 3669307177, win 65160, options [mss 1460,sackOK,TS val 197800310 ecr 101730750,nop,wscale 10], length 0
1793456375.724529 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [S.], cksum 0xa804 (incorrect -> 0xc6be), seq 3715411276, ack 3669307177, win 65160, options [mss 1460,sackOK,TS val 197800310 ecr 101730750,nop,wscale 10], length 0
1793456375.724652 fvmb0 In  IP (tos 0x0, ttl 64, id 48945, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xf3d7 (correct), ack 3715411277, win 63, options [nop,nop,TS val 101730750 ecr 197800310], length 0
1793456375.724657 fvup0 Out IP (tos 0x0, ttl 63, id 48945, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x158f (correct), ack 3715411277, win 63, options [nop,nop,TS val 101730750 ecr 197800310], length 0
1793456375.726384 fvmb0 In  IP (tos 0x0, ttl 64, id 48946, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], cksum 0xa25a (correct), seq 3669307177:3669307221, ack 3715411277, win 63, options [nop,nop,TS val 101730751 ecr 197800310], length 44
1793456375.726395 fvup0 Out IP (tos 0x0, ttl 63, id 48946, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], cksum 0xc411 (correct), seq 3669307177:3669307221, ack 3715411277, win 63, options [nop,nop,TS val 101730751 ecr 197800310], length 44
1793456375.726413 fvup0 In  IP (tos 0x0, ttl 64, id 37183, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x155f), ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 0
1793456375.726420 fvmb0 Out IP (tos 0x0, ttl 63, id 37183, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xa7fc (incorrect -> 0xf3a7), ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 0
1793456375.726453 fvup0 In  IP (tos 0x0, ttl 64, id 37184, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], cksum 0x8648 (incorrect -> 0xbc08), seq 3715411277:3715411280, ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 3
1793456375.726470 fvmb0 Out IP (tos 0x0, ttl 63, id 37184, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], cksum 0xa7ff (incorrect -> 0x9a51), seq 3715411277:3715411280, ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 3
1793456375.726605 fvmb0 In  IP (tos 0x0, ttl 64, id 48947, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xf3a4 (correct), ack 3715411280, win 63, options [nop,nop,TS val 101730752 ecr 197800312], length 0
1793456375.726608 fvup0 Out IP (tos 0x0, ttl 63, id 48947, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x155c (correct), ack 3715411280, win 63, options [nop,nop,TS val 101730752 ecr 197800312], length 0
1793456375.726877 fvmb0 In  IP (tos 0x0, ttl 64, id 48948, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669307221:3669313361, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.726878 fvmb0 In  IP (tos 0x0, ttl 64, id 48953, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669313361:3669319501, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.726886 fvup0 Out IP (tos 0x0, ttl 63, id 48948, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669307221:3669313361, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.726888 fvup0 Out IP (tos 0x0, ttl 63, id 48953, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669313361:3669319501, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.726901 fvup0 In  IP (tos 0x0, ttl 64, id 37185, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0xe567), ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.726905 fvmb0 Out IP (tos 0x0, ttl 63, id 37185, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xa7fc (incorrect -> 0xc3b0), ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.726929 fvup0 In  IP (tos 0x0, ttl 64, id 37186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715411280:3715412286, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726932 fvmb0 Out IP (tos 0x0, ttl 63, id 37186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715411280:3715412286, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726949 fvup0 In  IP (tos 0x0, ttl 64, id 37187, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715412286:3715413292, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726952 fvmb0 Out IP (tos 0x0, ttl 63, id 37187, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715412286:3715413292, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726962 fvup0 In  IP (tos 0x0, ttl 64, id 37188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715413292:3715414298, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726964 fvmb0 Out IP (tos 0x0, ttl 63, id 37188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715413292:3715414298, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726972 fvup0 In  IP (tos 0x0, ttl 64, id 37189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715414298:3715415304, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726974 fvmb0 Out IP (tos 0x0, ttl 63, id 37189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715414298:3715415304, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726987 fvup0 In  IP (tos 0x0, ttl 64, id 37190, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715415304:3715416310, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726989 fvmb0 Out IP (tos 0x0, ttl 63, id 37190, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715415304:3715416310, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726996 fvup0 In  IP (tos 0x0, ttl 64, id 37191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715416310:3715417316, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726999 fvmb0 Out IP (tos 0x0, ttl 63, id 37191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715416310:3715417316, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727011 fvup0 In  IP (tos 0x0, ttl 64, id 37192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715417316:3715418322, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727014 fvmb0 Out IP (tos 0x0, ttl 63, id 37192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715417316:3715418322, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727027 fvup0 In  IP (tos 0x0, ttl 64, id 37193, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715418322:3715419328, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727030 fvmb0 Out IP (tos 0x0, ttl 63, id 37193, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715418322:3715419328, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727039 fvup0 In  IP (tos 0x0, ttl 64, id 37194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715419328:3715420334, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727042 fvmb0 Out IP (tos 0x0, ttl 63, id 37194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715419328:3715420334, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727050 fvup0 In  IP (tos 0x0, ttl 64, id 37195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715420334:3715421340, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727052 fvmb0 Out IP (tos 0x0, ttl 63, id 37195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715420334:3715421340, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727156 fvmb0 In  IP (tos 0x0, ttl 64, id 48958, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669319501:3669321957, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727160 fvup0 Out IP (tos 0x0, ttl 63, id 48958, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669319501:3669321957, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727167 fvup0 In  IP (tos 0x0, ttl 64, id 37196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0xb47a), ack 3669321957, win 66, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727171 fvmb0 Out IP (tos 0x0, ttl 63, id 37196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xa7fc (incorrect -> 0x92c3), ack 3669321957, win 66, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727175 fvmb0 In  IP (tos 0x0, ttl 64, id 48960, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669321957:3669328097, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 6140
1793456375.727178 fvup0 Out IP (tos 0x0, ttl 63, id 48960, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669321957:3669328097, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 6140
1793456375.727183 fvmb0 In  IP (tos 0x0, ttl 64, id 48965, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669328097:3669331781, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727185 fvup0 In  IP (tos 0x0, ttl 64, id 37197, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x9c81), ack 3669328097, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727186 fvup0 Out IP (tos 0x0, ttl 63, id 48965, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669328097:3669331781, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727188 fvmb0 Out IP (tos 0x0, ttl 63, id 37197, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xa7fc (incorrect -> 0x7aca), ack 3669328097, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727189 fvmb0 In  IP (tos 0x0, ttl 64, id 48968, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669331781:3669334237, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727191 fvup0 Out IP (tos 0x0, ttl 63, id 48968, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669331781:3669334237, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727194 fvmb0 In  IP (tos 0x0, ttl 64, id 48970, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669334237:3669336693, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727197 fvup0 Out IP (tos 0x0, ttl 63, id 48970, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669334237:3669336693, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727205 fvmb0 In  IP (tos 0x0, ttl 64, id 48972, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669336693:3669339149, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727207 fvup0 In  IP (tos 0x0, ttl 64, id 37198, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x7af5), ack 3669336693, win 55, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727210 fvmb0 Out IP (tos 0x0, ttl 63, id 37198, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xa7fc (incorrect -> 0x593e), ack 3669336693, win 55, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727218 fvup0 Out IP (tos 0x0, ttl 63, id 48972, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669336693:3669339149, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727229 fvup0 In  IP (tos 0x0, ttl 64, id 37199, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x715a), ack 3669339149, win 58, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727233 fvmb0 Out IP (tos 0x0, ttl 63, id 37199, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xa7fc (incorrect -> 0x4fa3), ack 3669339149, win 58, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727237 fvmb0 In  IP (tos 0x0, ttl 64, id 48974, offset 0, flags [DF], proto TCP (6), length 4964)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669339149:3669344061, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 4912
1793456375.727237 fvmb0 In  IP (tos 0x0, ttl 64, id 48978, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x5405 (correct), ack 3715415304, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727238 fvmb0 In  IP (tos 0x0, ttl 64, id 48979, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x3c76 (correct), ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727240 fvup0 Out IP (tos 0x0, ttl 63, id 48974, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669339149:3669344061, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 4912
1793456375.727243 fvup0 Out IP (tos 0x0, ttl 63, id 48978, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x75bc (correct), ack 3715415304, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727244 fvup0 Out IP (tos 0x0, ttl 63, id 48979, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x5e2d (correct), ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727253 fvup0 In  IP (tos 0x0, ttl 64, id 37200, offset 0, flags [DF], proto TCP (6), length 8648)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715421340:3715429936, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 8596
1793456375.727256 fvmb0 Out IP (tos 0x0, ttl 63, id 37200, offset 0, flags [DF], proto TCP (6), length 8648)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715421340:3715429936, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 8596
1793456375.727258 fvup0 In  IP (tos 0x0, ttl 64, id 37207, offset 0, flags [DF], proto TCP (6), length 3528)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715429936:3715433412, ack 3669344061, win 56, options [nop,nop,TS val 197800313 ecr 101730753], length 3476
1793456375.727262 fvmb0 Out IP (tos 0x0, ttl 63, id 37207, offset 0, flags [DF], proto TCP (6), length 3528)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715429936:3715433412, ack 3669344061, win 56, options [nop,nop,TS val 197800313 ecr 101730753], length 3476
1793456375.727266 fvmb0 In  IP (tos 0x0, ttl 64, id 48980, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669344061:3669346517, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727267 fvup0 In  IP (tos 0x0, ttl 64, id 37210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715433412:3715434418, ack 3669344061, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727268 fvup0 Out IP (tos 0x0, ttl 63, id 48980, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669344061:3669346517, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727270 fvmb0 Out IP (tos 0x0, ttl 63, id 37210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715433412:3715434418, ack 3669344061, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727271 fvmb0 In  IP (tos 0x0, ttl 64, id 48982, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669346517:3669347745, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727273 fvup0 Out IP (tos 0x0, ttl 63, id 48982, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669346517:3669347745, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727281 fvup0 In  IP (tos 0x0, ttl 64, id 37211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715434418:3715435424, ack 3669347745, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727284 fvmb0 Out IP (tos 0x0, ttl 63, id 37211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715434418:3715435424, ack 3669347745, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727284 fvmb0 In  IP (tos 0x0, ttl 64, id 48983, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669347745:3669348973, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727286 fvmb0 In  IP (tos 0x0, ttl 64, id 48984, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669348973:3669352657, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727288 fvup0 Out IP (tos 0x0, ttl 63, id 48983, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669347745:3669348973, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727289 fvup0 Out IP (tos 0x0, ttl 63, id 48984, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669348973:3669352657, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727293 fvmb0 In  IP (tos 0x0, ttl 64, id 48987, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669352657:3669356341, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727294 fvup0 Out IP (tos 0x0, ttl 63, id 48987, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669352657:3669356341, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727297 fvmb0 In  IP (tos 0x0, ttl 64, id 48990, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669356341:3669357569, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727298 fvup0 Out IP (tos 0x0, ttl 63, id 48990, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669356341:3669357569, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727299 fvup0 In  IP (tos 0x0, ttl 64, id 37212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715435424:3715436430, ack 3669356341, win 47, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727300 fvmb0 In  IP (tos 0x0, ttl 64, id 48991, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669357569:3669358797, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727302 fvup0 Out IP (tos 0x0, ttl 63, id 48991, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669357569:3669358797, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727302 fvmb0 Out IP (tos 0x0, ttl 63, id 37212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715435424:3715436430, ack 3669356341, win 47, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727305 fvmb0 In  IP (tos 0x0, ttl 64, id 48992, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669358797:3669361253, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727307 fvup0 Out IP (tos 0x0, ttl 63, id 48992, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669358797:3669361253, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727309 fvmb0 In  IP (tos 0x0, ttl 64, id 48994, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669361253:3669362481, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727311 fvup0 Out IP (tos 0x0, ttl 63, id 48994, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669361253:3669362481, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727313 fvup0 In  IP (tos 0x0, ttl 64, id 37213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715436430:3715437436, ack 3669361253, win 43, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727314 fvmb0 In  IP (tos 0x0, ttl 64, id 48995, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669362481:3669363709, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727316 fvup0 Out IP (tos 0x0, ttl 63, id 48995, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669362481:3669363709, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727317 fvmb0 Out IP (tos 0x0, ttl 63, id 37213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715436430:3715437436, ack 3669361253, win 43, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727318 fvmb0 In  IP (tos 0x0, ttl 64, id 48996, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669363709:3669364937, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727319 fvup0 Out IP (tos 0x0, ttl 63, id 48996, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669363709:3669364937, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727327 fvup0 In  IP (tos 0x0, ttl 64, id 37214, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0xcdcf), ack 3669364937, win 40, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727327 fvmb0 In  IP (tos 0x0, ttl 64, id 48997, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669364937:3669368621, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727329 fvup0 Out IP (tos 0x0, ttl 63, id 48997, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669364937:3669368621, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727330 fvmb0 Out IP (tos 0x0, ttl 63, id 37214, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [.], cksum 0xa7fc (incorrect -> 0xac18), ack 3669364937, win 40, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727332 fvmb0 In  IP (tos 0x0, ttl 64, id 49000, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], seq 3669368621:3669369849, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727334 fvup0 Out IP (tos 0x0, ttl 63, id 49000, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669368621:3669369849, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727337 fvup0 In  IP (tos 0x0, ttl 64, id 37215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715437436:3715438442, ack 3669368621, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727339 fvmb0 Out IP (tos 0x0, ttl 63, id 37215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715437436:3715438442, ack 3669368621, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727343 fvmb0 In  IP (tos 0x0, ttl 64, id 49001, offset 0, flags [DF], proto TCP (6), length 1488)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [P.], seq 3669369849:3669371285, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1436
1793456375.727344 fvmb0 In  IP (tos 0x0, ttl 64, id 49003, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xc880 (correct), ack 3715423796, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727346 fvup0 Out IP (tos 0x0, ttl 63, id 49001, offset 0, flags [DF], proto TCP (6), length 1488)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669369849:3669371285, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1436
1793456375.727347 fvup0 Out IP (tos 0x0, ttl 63, id 49003, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xea37 (correct), ack 3715423796, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727349 fvup0 In  IP (tos 0x0, ttl 64, id 37216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715438442:3715439448, ack 3669369849, win 36, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727350 fvmb0 In  IP (tos 0x0, ttl 64, id 49004, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xba16 (correct), ack 3715427480, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727351 fvmb0 In  IP (tos 0x0, ttl 64, id 49005, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xb07e (correct), ack 3715429936, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727353 fvmb0 Out IP (tos 0x0, ttl 63, id 37216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715438442:3715439448, ack 3669369849, win 36, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727353 fvup0 Out IP (tos 0x0, ttl 63, id 49004, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xdbcd (correct), ack 3715427480, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727354 fvup0 Out IP (tos 0x0, ttl 63, id 49005, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xd235 (correct), ack 3715429936, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727358 fvmb0 In  IP (tos 0x0, ttl 64, id 49006, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x9725 (correct), ack 3715436430, win 61, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727360 fvup0 Out IP (tos 0x0, ttl 63, id 49006, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xb8dc (correct), ack 3715436430, win 61, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727366 fvup0 In  IP (tos 0x0, ttl 64, id 37217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715439448:3715440454, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727369 fvmb0 Out IP (tos 0x0, ttl 63, id 37217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715439448:3715440454, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727370 fvmb0 In  IP (tos 0x0, ttl 64, id 49007, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x8f43 (correct), ack 3715438442, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727372 fvup0 Out IP (tos 0x0, ttl 63, id 49007, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xb0fa (correct), ack 3715438442, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727380 fvup0 In  IP (tos 0x0, ttl 64, id 37218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715440454:3715441460, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727383 fvmb0 Out IP (tos 0x0, ttl 63, id 37218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715440454:3715441460, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727390 fvup0 In  IP (tos 0x0, ttl 64, id 37219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715441460:3715442466, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727393 fvmb0 Out IP (tos 0x0, ttl 63, id 37219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715441460:3715442466, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727400 fvmb0 In  IP (tos 0x0, ttl 64, id 49008, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x8767 (correct), ack 3715440454, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727401 fvup0 Out IP (tos 0x0, ttl 63, id 49008, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa91e (correct), ack 3715440454, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727404 fvup0 In  IP (tos 0x0, ttl 64, id 37220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715442466:3715443472, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727406 fvmb0 Out IP (tos 0x0, ttl 63, id 37220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715442466:3715443472, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727418 fvmb0 In  IP (tos 0x0, ttl 64, id 49009, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x7f8b (correct), ack 3715442466, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727420 fvup0 In  IP (tos 0x0, ttl 64, id 37221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715443472:3715444478, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727420 fvup0 Out IP (tos 0x0, ttl 63, id 49009, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa142 (correct), ack 3715442466, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727423 fvmb0 Out IP (tos 0x0, ttl 63, id 37221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715443472:3715444478, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727437 fvup0 In  IP (tos 0x0, ttl 64, id 37222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715444478:3715445484, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727440 fvmb0 Out IP (tos 0x0, ttl 63, id 37222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715444478:3715445484, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727452 fvup0 In  IP (tos 0x0, ttl 64, id 37223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715445484:3715446490, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727454 fvmb0 In  IP (tos 0x0, ttl 64, id 49010, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x77af (correct), ack 3715444478, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727454 fvmb0 Out IP (tos 0x0, ttl 63, id 37223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715445484:3715446490, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727456 fvup0 Out IP (tos 0x0, ttl 63, id 49010, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x9966 (correct), ack 3715444478, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727467 fvup0 In  IP (tos 0x0, ttl 64, id 37224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715446490:3715447496, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727470 fvmb0 Out IP (tos 0x0, ttl 63, id 37224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715446490:3715447496, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727482 fvup0 In  IP (tos 0x0, ttl 64, id 37225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715447496:3715448502, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727482 fvmb0 In  IP (tos 0x0, ttl 64, id 49011, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x6fd3 (correct), ack 3715446490, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727484 fvup0 Out IP (tos 0x0, ttl 63, id 49011, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x918a (correct), ack 3715446490, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727485 fvmb0 Out IP (tos 0x0, ttl 63, id 37225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715447496:3715448502, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727501 fvup0 In  IP (tos 0x0, ttl 64, id 37226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715448502:3715449508, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727504 fvmb0 Out IP (tos 0x0, ttl 63, id 37226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715448502:3715449508, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727515 fvmb0 In  IP (tos 0x0, ttl 64, id 49012, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x67f7 (correct), ack 3715448502, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727517 fvup0 Out IP (tos 0x0, ttl 63, id 49012, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x89ae (correct), ack 3715448502, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727521 fvup0 In  IP (tos 0x0, ttl 64, id 37227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715449508:3715450514, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727524 fvmb0 Out IP (tos 0x0, ttl 63, id 37227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715449508:3715450514, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727535 fvup0 In  IP (tos 0x0, ttl 64, id 37228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715450514:3715451520, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727537 fvmb0 Out IP (tos 0x0, ttl 63, id 37228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715450514:3715451520, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727548 fvup0 In  IP (tos 0x0, ttl 64, id 37229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715451520:3715452526, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727550 fvmb0 Out IP (tos 0x0, ttl 63, id 37229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715451520:3715452526, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727558 fvup0 In  IP (tos 0x0, ttl 64, id 37230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715452526:3715453532, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727560 fvmb0 In  IP (tos 0x0, ttl 64, id 49013, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x601b (correct), ack 3715450514, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727561 fvmb0 Out IP (tos 0x0, ttl 63, id 37230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715452526:3715453532, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727562 fvup0 Out IP (tos 0x0, ttl 63, id 49013, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x81d2 (correct), ack 3715450514, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727568 fvup0 In  IP (tos 0x0, ttl 64, id 37231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715453532:3715454538, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727571 fvmb0 Out IP (tos 0x0, ttl 63, id 37231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715453532:3715454538, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727580 fvup0 In  IP (tos 0x0, ttl 64, id 37232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715454538:3715455544, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727583 fvmb0 Out IP (tos 0x0, ttl 63, id 37232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715454538:3715455544, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727590 fvup0 In  IP (tos 0x0, ttl 64, id 37233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715455544:3715456550, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727592 fvmb0 Out IP (tos 0x0, ttl 63, id 37233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715455544:3715456550, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727601 fvup0 In  IP (tos 0x0, ttl 64, id 37234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715456550:3715457556, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727603 fvmb0 Out IP (tos 0x0, ttl 63, id 37234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715456550:3715457556, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727604 fvmb0 In  IP (tos 0x0, ttl 64, id 49014, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x583f (correct), ack 3715452526, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727607 fvup0 Out IP (tos 0x0, ttl 63, id 49014, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x79f6 (correct), ack 3715452526, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727610 fvup0 In  IP (tos 0x0, ttl 64, id 37235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715457556:3715458562, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727613 fvmb0 Out IP (tos 0x0, ttl 63, id 37235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715457556:3715458562, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727624 fvup0 In  IP (tos 0x0, ttl 64, id 37236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715458562:3715459568, ack 3669371285, win 53, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727626 fvmb0 Out IP (tos 0x0, ttl 63, id 37236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715458562:3715459568, ack 3669371285, win 53, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727630 fvmb0 In  IP (tos 0x0, ttl 64, id 49015, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x5063 (correct), ack 3715454538, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727631 fvup0 Out IP (tos 0x0, ttl 63, id 49015, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x721a (correct), ack 3715454538, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727634 fvmb0 In  IP (tos 0x0, ttl 64, id 49016, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x4887 (correct), ack 3715456550, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727636 fvup0 In  IP (tos 0x0, ttl 64, id 37237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715459568:3715460574, ack 3669371285, win 53, options [nop,nop,TS val 197800314 ecr 101730753], length 1006
1793456375.727637 fvup0 Out IP (tos 0x0, ttl 63, id 49016, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x6a3e (correct), ack 3715456550, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727639 fvmb0 Out IP (tos 0x0, ttl 63, id 37237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715459568:3715460574, ack 3669371285, win 53, options [nop,nop,TS val 197800314 ecr 101730753], length 1006
1793456375.727644 fvmb0 In  IP (tos 0x0, ttl 64, id 49017, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x40aa (correct), ack 3715458562, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727646 fvup0 Out IP (tos 0x0, ttl 63, id 49017, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x6261 (correct), ack 3715458562, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727652 fvup0 In  IP (tos 0x0, ttl 64, id 37238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715460574:3715461580, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727655 fvmb0 Out IP (tos 0x0, ttl 63, id 37238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715460574:3715461580, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727662 fvup0 In  IP (tos 0x0, ttl 64, id 37239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715461580:3715462586, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727664 fvmb0 Out IP (tos 0x0, ttl 63, id 37239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715461580:3715462586, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727673 fvmb0 In  IP (tos 0x0, ttl 64, id 49018, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x38ce (correct), ack 3715460574, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727673 fvup0 In  IP (tos 0x0, ttl 64, id 37240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715462586:3715463592, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727675 fvup0 Out IP (tos 0x0, ttl 63, id 49018, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x5a85 (correct), ack 3715460574, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727676 fvmb0 Out IP (tos 0x0, ttl 63, id 37240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715462586:3715463592, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727683 fvup0 In  IP (tos 0x0, ttl 64, id 37241, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715463592:3715464598, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727685 fvmb0 Out IP (tos 0x0, ttl 63, id 37241, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715463592:3715464598, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727695 fvup0 In  IP (tos 0x0, ttl 64, id 37242, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715464598:3715465604, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727698 fvmb0 Out IP (tos 0x0, ttl 63, id 37242, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715464598:3715465604, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727706 fvup0 In  IP (tos 0x0, ttl 64, id 37243, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715465604:3715466610, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727708 fvmb0 Out IP (tos 0x0, ttl 63, id 37243, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715465604:3715466610, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727709 fvmb0 In  IP (tos 0x0, ttl 64, id 49019, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x2d03 (correct), ack 3715463592, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727716 fvup0 In  IP (tos 0x0, ttl 64, id 37244, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715466610:3715467616, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727718 fvup0 Out IP (tos 0x0, ttl 63, id 49019, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x4eba (correct), ack 3715463592, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727719 fvmb0 Out IP (tos 0x0, ttl 63, id 37244, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715466610:3715467616, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727725 fvup0 In  IP (tos 0x0, ttl 64, id 37245, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715467616:3715468622, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727728 fvmb0 Out IP (tos 0x0, ttl 63, id 37245, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715467616:3715468622, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727734 fvmb0 In  IP (tos 0x0, ttl 64, id 49020, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x2527 (correct), ack 3715465604, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727736 fvup0 Out IP (tos 0x0, ttl 63, id 49020, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x46de (correct), ack 3715465604, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727742 fvup0 In  IP (tos 0x0, ttl 64, id 37246, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715468622:3715469628, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727744 fvmb0 Out IP (tos 0x0, ttl 63, id 37246, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715468622:3715469628, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727751 fvup0 In  IP (tos 0x0, ttl 64, id 37247, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715469628:3715470634, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727754 fvmb0 Out IP (tos 0x0, ttl 63, id 37247, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715469628:3715470634, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727758 fvmb0 In  IP (tos 0x0, ttl 64, id 49021, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x1d4b (correct), ack 3715467616, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727760 fvup0 Out IP (tos 0x0, ttl 63, id 49021, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x3f02 (correct), ack 3715467616, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727762 fvup0 In  IP (tos 0x0, ttl 64, id 37248, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715470634:3715471640, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727765 fvmb0 Out IP (tos 0x0, ttl 63, id 37248, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715470634:3715471640, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727773 fvup0 In  IP (tos 0x0, ttl 64, id 37249, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715471640:3715472646, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727776 fvmb0 Out IP (tos 0x0, ttl 63, id 37249, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715471640:3715472646, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727788 fvup0 In  IP (tos 0x0, ttl 64, id 37250, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715472646:3715473652, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727791 fvmb0 Out IP (tos 0x0, ttl 63, id 37250, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715472646:3715473652, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727798 fvmb0 In  IP (tos 0x0, ttl 64, id 49022, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x156f (correct), ack 3715469628, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727798 fvup0 In  IP (tos 0x0, ttl 64, id 37251, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715473652:3715474658, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727800 fvup0 Out IP (tos 0x0, ttl 63, id 49022, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x3726 (correct), ack 3715469628, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727801 fvmb0 Out IP (tos 0x0, ttl 63, id 37251, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715473652:3715474658, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727809 fvmb0 In  IP (tos 0x0, ttl 64, id 49023, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x0d93 (correct), ack 3715471640, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727809 fvup0 In  IP (tos 0x0, ttl 64, id 37252, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715474658:3715475664, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727811 fvup0 Out IP (tos 0x0, ttl 63, id 49023, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x2f4a (correct), ack 3715471640, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727812 fvmb0 Out IP (tos 0x0, ttl 63, id 37252, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [P.], seq 3715474658:3715475664, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727822 fvmb0 In  IP (tos 0x0, ttl 64, id 49024, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0x05b7 (correct), ack 3715473652, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727824 fvup0 Out IP (tos 0x0, ttl 63, id 49024, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x276e (correct), ack 3715473652, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728056 fvmb0 In  IP (tos 0x0, ttl 64, id 49025, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xfdda (correct), ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728057 fvmb0 In  IP (tos 0x0, ttl 64, id 49026, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [F.], cksum 0xfdd9 (correct), seq 3669371285, ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728068 fvup0 Out IP (tos 0x0, ttl 63, id 49025, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x1f92 (correct), ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728071 fvup0 Out IP (tos 0x0, ttl 63, id 49026, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [F.], cksum 0x1f91 (correct), seq 3669371285, ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728224 fvup0 In  IP (tos 0x0, ttl 64, id 37253, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [F.], cksum 0x8645 (incorrect -> 0x1f91), seq 3715475664, ack 3669371286, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 0
1793456375.728228 fvmb0 Out IP (tos 0x0, ttl 63, id 37253, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37946: Flags [F.], cksum 0xa7fc (incorrect -> 0xfdd9), seq 3715475664, ack 3669371286, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 0
1793456375.728566 fvmb0 In  IP (tos 0x0, ttl 64, id 49027, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37946 > 192.0.3.10.9446: Flags [.], cksum 0xfdd8 (correct), ack 3715475665, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728571 fvup0 Out IP (tos 0x0, ttl 63, id 49027, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x1f90 (correct), ack 3715475665, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4318,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"5a:8d:b5:97:e7:d6","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4319,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"5a:bc:89:da:9c:4a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4322,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"46:08:5f:12:52:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"82:09:3c:35:ad:f2","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.83.207.231","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:32844
	 cubic wscale:10,10 rto:211 rtt:10.724/17.274 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:29889 bytes_acked:29889 bytes_received:15891 segs_out:81 segs_in:106 data_segs_out:78 data_segs_in:78 send 244446102bps lastsnd:3234 lastrcv:3235 lastack:3192 pacing_rate 488858008bps delivery_rate 37449142856bps delivered:79 app_limited busy:542ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.007 snd_wnd:65536
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:48152
	 cubic wscale:10,10 rto:203 rtt:2.949/5.879 ato:40 mss:43520 pmtu:65535 rcvmss:540 advmss:65483 cwnd:10 bytes_sent:7252 bytes_acked:7252 bytes_received:4882 segs_out:9 segs_in:16 data_segs_out:6 data_segs_in:7 send 1180603594bps lastsnd:28145 lastrcv:28150 lastack:12944 pacing_rate 2360806912bps delivery_rate 49737142856bps delivered:7 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78471 minrtt:0.007 snd_wnd:84992
ESTAB 0      0       172.28.30.40:58744 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.612/0.634 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4629 bytes_acked:4630 bytes_received:10418 segs_out:86 segs_in:67 data_segs_out:36 data_segs_in:61 send 189281046bps lastsnd:3246 lastrcv:3246 lastack:3246 pacing_rate 378484784bps delivery_rate 2896000000bps delivered:37 app_limited busy:49ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58756 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.213/0.338 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:7610 bytes_acked:7611 bytes_received:9433 segs_out:63 segs_in:74 data_segs_out:34 data_segs_in:68 send 543849765bps lastsnd:5668 lastrcv:5668 lastack:5668 pacing_rate 1087061576bps delivery_rate 2896000000bps delivered:35 app_limited busy:32ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58772 172.28.30.21:443
	 cubic wscale:10,10 rto:204 rtt:3.54/6.924 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2328 bytes_acked:2329 bytes_received:3439 segs_out:22 segs_in:20 data_segs_out:10 data_segs_in:13 send 32723164bps lastsnd:5669 lastrcv:5669 lastack:5669 pacing_rate 65430152bps delivery_rate 2106181816bps delivered:11 app_limited busy:42ms rcv_space:14480 rcv_ssthresh:82722 minrtt:0.007 snd_wnd:74752
ESTAB 0      0       172.28.30.40:34032 172.28.30.30:9443
	 cubic wscale:10,10 rto:205 rtt:4.435/2.412 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:28121 bytes_acked:28122 bytes_received:24319 segs_out:73 segs_in:44 data_segs_out:49 data_segs_in:31 send 26119504bps lastsnd:3267 lastrcv:3261 lastack:3261 pacing_rate 52236056bps delivery_rate 2106181816bps delivered:50 app_limited busy:97ms rcv_rtt:6 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.008 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:48150
	 cubic wscale:10,10 rto:202 rtt:1.274/2.488 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:18163 bytes_acked:18163 bytes_received:24332 segs_out:28 segs_in:47 data_segs_out:23 data_segs_in:24 send 2861412873bps lastsnd:3261 lastrcv:3267 lastack:3261 pacing_rate 5720019608bps delivery_rate 182272000000bps delivered:24 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:105032 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:32844    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.762/0.045 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:15891 bytes_acked:15892 bytes_received:29889 segs_out:106 segs_in:82 data_segs_out:78 data_segs_in:78 send 3440209974bps lastsnd:3246 lastrcv:3245 lastack:3245 pacing_rate 6880419944bps delivery_rate 26214400000bps delivered:79 app_limited busy:109ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.01 snd_wnd:65536
ESTAB 0      0          127.0.0.1:48150    127.0.0.1:9444
	 cubic wscale:10,10 rto:205 rtt:4.883/2.192 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:24332 bytes_acked:24333 bytes_received:18163 segs_out:47 segs_in:29 data_segs_out:24 data_segs_in:23 send 863993447bps lastsnd:3268 lastrcv:3262 lastack:3262 pacing_rate 1727898424bps delivery_rate 84377600000bps delivered:25 app_limited busy:99ms rcv_rtt:6 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.005 snd_wnd:105472
ESTAB 0      0          127.0.0.1:48152    127.0.0.1:9444
	 cubic wscale:10,10 rto:202 rtt:1.88/2.548 ato:40 mss:39424 pmtu:65535 rcvmss:1156 advmss:65483 cwnd:10 bytes_sent:4882 bytes_acked:4883 bytes_received:7252 segs_out:17 segs_in:10 data_segs_out:7 data_segs_in:6 send 1677617021bps lastsnd:28160 lastrcv:28155 lastack:28155 pacing_rate 3355010968bps delivery_rate 63078400000bps delivered:8 app_limited busy:17ms rcv_rtt:3.578 rcv_space:65495 rcv_ssthresh:71168 minrtt:0.005 snd_wnd:78848

- matrix4_capture_1_receipt: {"bytes":"52870","sha256":"df63e6fb75580dcf49107b07ce357074be25e4539c312f11f780a098d738fd28"}
- matrix4_capture_2_window: {"launch_requested_at":"2026-10-31T14:19:34.487627416Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:36.639810212Z","stop_requested_at":"2026-10-31T14:19:36.564521544Z"}
- matrix4_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
118 packets captured
118 packets received by filter
0 packets dropped by kernel

- matrix4_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456375.724500 eth0  In  IP (tos 0x0, ttl 63, id 48944, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [S], cksum 0xb40c (correct), seq 3669307176, win 64480, options [mss 1240,sackOK,TS val 101730750 ecr 0,nop,wscale 10], length 0
1793456375.724521 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [S.], cksum 0x864d (incorrect -> 0xe875), seq 3715411276, ack 3669307177, win 65160, options [mss 1460,sackOK,TS val 197800310 ecr 101730750,nop,wscale 10], length 0
1793456375.724658 eth0  In  IP (tos 0x0, ttl 63, id 48945, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x158f (correct), ack 3715411277, win 63, options [nop,nop,TS val 101730750 ecr 197800310], length 0
1793456375.726398 eth0  In  IP (tos 0x0, ttl 63, id 48946, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], cksum 0xc411 (correct), seq 3669307177:3669307221, ack 3715411277, win 63, options [nop,nop,TS val 101730751 ecr 197800310], length 44
1793456375.726410 eth0  Out IP (tos 0x0, ttl 64, id 37183, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x155f), ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 0
1793456375.726446 eth0  Out IP (tos 0x0, ttl 64, id 37184, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], cksum 0x8648 (incorrect -> 0xbc08), seq 3715411277:3715411280, ack 3669307221, win 64, options [nop,nop,TS val 197800312 ecr 101730751], length 3
1793456375.726609 eth0  In  IP (tos 0x0, ttl 63, id 48947, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x155c (correct), ack 3715411280, win 63, options [nop,nop,TS val 101730752 ecr 197800312], length 0
1793456375.726891 eth0  In  IP (tos 0x0, ttl 63, id 48948, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669307221:3669313361, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.726894 eth0  In  IP (tos 0x0, ttl 63, id 48953, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669313361:3669319501, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800312], length 6140
1793456375.726900 eth0  Out IP (tos 0x0, ttl 64, id 37185, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0xe567), ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.726928 eth0  Out IP (tos 0x0, ttl 64, id 37186, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715411280:3715412286, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726948 eth0  Out IP (tos 0x0, ttl 64, id 37187, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715412286:3715413292, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726961 eth0  Out IP (tos 0x0, ttl 64, id 37188, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715413292:3715414298, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726971 eth0  Out IP (tos 0x0, ttl 64, id 37189, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715414298:3715415304, ack 3669319501, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726986 eth0  Out IP (tos 0x0, ttl 64, id 37190, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715415304:3715416310, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.726995 eth0  Out IP (tos 0x0, ttl 64, id 37191, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715416310:3715417316, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727010 eth0  Out IP (tos 0x0, ttl 64, id 37192, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715417316:3715418322, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727026 eth0  Out IP (tos 0x0, ttl 64, id 37193, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715418322:3715419328, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727038 eth0  Out IP (tos 0x0, ttl 64, id 37194, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715419328:3715420334, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727049 eth0  Out IP (tos 0x0, ttl 64, id 37195, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715420334:3715421340, ack 3669319501, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727162 eth0  In  IP (tos 0x0, ttl 63, id 48958, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669319501:3669321957, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727166 eth0  Out IP (tos 0x0, ttl 64, id 37196, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0xb47a), ack 3669321957, win 66, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727179 eth0  In  IP (tos 0x0, ttl 63, id 48960, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669321957:3669328097, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 6140
1793456375.727184 eth0  Out IP (tos 0x0, ttl 64, id 37197, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x9c81), ack 3669328097, win 63, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727187 eth0  In  IP (tos 0x0, ttl 63, id 48965, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669328097:3669331781, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727192 eth0  In  IP (tos 0x0, ttl 63, id 48968, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669331781:3669334237, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727197 eth0  In  IP (tos 0x0, ttl 63, id 48970, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669334237:3669336693, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727206 eth0  Out IP (tos 0x0, ttl 64, id 37198, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x7af5), ack 3669336693, win 55, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727222 eth0  In  IP (tos 0x0, ttl 63, id 48972, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669336693:3669339149, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727228 eth0  Out IP (tos 0x0, ttl 64, id 37199, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0x715a), ack 3669339149, win 58, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727246 eth0  In  IP (tos 0x0, ttl 63, id 48974, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669339149:3669344061, ack 3715411280, win 63, options [nop,nop,TS val 101730753 ecr 197800313], length 4912
1793456375.727248 eth0  In  IP (tos 0x0, ttl 63, id 48978, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x75bc (correct), ack 3715415304, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727250 eth0  Out IP (tos 0x0, ttl 64, id 37200, offset 0, flags [DF], proto TCP (6), length 8648)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715421340:3715429936, ack 3669344061, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 8596
1793456375.727252 eth0  In  IP (tos 0x0, ttl 63, id 48979, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x5e2d (correct), ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727257 eth0  Out IP (tos 0x0, ttl 64, id 37207, offset 0, flags [DF], proto TCP (6), length 3528)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715429936:3715433412, ack 3669344061, win 56, options [nop,nop,TS val 197800313 ecr 101730753], length 3476
1793456375.727265 eth0  Out IP (tos 0x0, ttl 64, id 37210, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715433412:3715434418, ack 3669344061, win 57, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727269 eth0  In  IP (tos 0x0, ttl 63, id 48980, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669344061:3669346517, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727273 eth0  In  IP (tos 0x0, ttl 63, id 48982, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669346517:3669347745, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727280 eth0  Out IP (tos 0x0, ttl 64, id 37211, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715434418:3715435424, ack 3669347745, win 54, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727290 eth0  In  IP (tos 0x0, ttl 63, id 48983, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669347745:3669348973, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727291 eth0  In  IP (tos 0x0, ttl 63, id 48984, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669348973:3669352657, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727295 eth0  In  IP (tos 0x0, ttl 63, id 48987, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669352657:3669356341, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727298 eth0  Out IP (tos 0x0, ttl 64, id 37212, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715435424:3715436430, ack 3669356341, win 47, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727299 eth0  In  IP (tos 0x0, ttl 63, id 48990, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669356341:3669357569, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727303 eth0  In  IP (tos 0x0, ttl 63, id 48991, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669357569:3669358797, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727307 eth0  In  IP (tos 0x0, ttl 63, id 48992, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669358797:3669361253, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 2456
1793456375.727312 eth0  Out IP (tos 0x0, ttl 64, id 37213, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715436430:3715437436, ack 3669361253, win 43, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727312 eth0  In  IP (tos 0x0, ttl 63, id 48994, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669361253:3669362481, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727317 eth0  In  IP (tos 0x0, ttl 63, id 48995, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669362481:3669363709, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727320 eth0  In  IP (tos 0x0, ttl 63, id 48996, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669363709:3669364937, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727325 eth0  Out IP (tos 0x0, ttl 64, id 37214, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [.], cksum 0x8645 (incorrect -> 0xcdcf), ack 3669364937, win 40, options [nop,nop,TS val 197800313 ecr 101730753], length 0
1793456375.727330 eth0  In  IP (tos 0x0, ttl 63, id 48997, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669364937:3669368621, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 3684
1793456375.727335 eth0  In  IP (tos 0x0, ttl 63, id 49000, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], seq 3669368621:3669369849, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1228
1793456375.727336 eth0  Out IP (tos 0x0, ttl 64, id 37215, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715437436:3715438442, ack 3669368621, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727348 eth0  Out IP (tos 0x0, ttl 64, id 37216, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715438442:3715439448, ack 3669369849, win 36, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727348 eth0  In  IP (tos 0x0, ttl 63, id 49001, offset 0, flags [DF], proto TCP (6), length 1488)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [P.], seq 3669369849:3669371285, ack 3715421340, win 55, options [nop,nop,TS val 101730753 ecr 197800313], length 1436
1793456375.727349 eth0  In  IP (tos 0x0, ttl 63, id 49003, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xea37 (correct), ack 3715423796, win 60, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727355 eth0  In  IP (tos 0x0, ttl 63, id 49004, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xdbcd (correct), ack 3715427480, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727357 eth0  In  IP (tos 0x0, ttl 63, id 49005, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xd235 (correct), ack 3715429936, win 66, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727361 eth0  In  IP (tos 0x0, ttl 63, id 49006, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xb8dc (correct), ack 3715436430, win 61, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727365 eth0  Out IP (tos 0x0, ttl 64, id 37217, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715439448:3715440454, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727373 eth0  In  IP (tos 0x0, ttl 63, id 49007, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xb0fa (correct), ack 3715438442, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727379 eth0  Out IP (tos 0x0, ttl 64, id 37218, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715440454:3715441460, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727389 eth0  Out IP (tos 0x0, ttl 64, id 37219, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715441460:3715442466, ack 3669371285, win 35, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727402 eth0  In  IP (tos 0x0, ttl 63, id 49008, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa91e (correct), ack 3715440454, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727402 eth0  Out IP (tos 0x0, ttl 64, id 37220, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715442466:3715443472, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727419 eth0  Out IP (tos 0x0, ttl 64, id 37221, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715443472:3715444478, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727421 eth0  In  IP (tos 0x0, ttl 63, id 49009, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0xa142 (correct), ack 3715442466, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727436 eth0  Out IP (tos 0x0, ttl 64, id 37222, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715444478:3715445484, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727451 eth0  Out IP (tos 0x0, ttl 64, id 37223, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715445484:3715446490, ack 3669371285, win 37, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727457 eth0  In  IP (tos 0x0, ttl 63, id 49010, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x9966 (correct), ack 3715444478, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727466 eth0  Out IP (tos 0x0, ttl 64, id 37224, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715446490:3715447496, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727481 eth0  Out IP (tos 0x0, ttl 64, id 37225, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715447496:3715448502, ack 3669371285, win 42, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727485 eth0  In  IP (tos 0x0, ttl 63, id 49011, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x918a (correct), ack 3715446490, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727500 eth0  Out IP (tos 0x0, ttl 64, id 37226, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715448502:3715449508, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727517 eth0  In  IP (tos 0x0, ttl 63, id 49012, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x89ae (correct), ack 3715448502, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727520 eth0  Out IP (tos 0x0, ttl 64, id 37227, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715449508:3715450514, ack 3669371285, win 44, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727532 eth0  Out IP (tos 0x0, ttl 64, id 37228, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715450514:3715451520, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727546 eth0  Out IP (tos 0x0, ttl 64, id 37229, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715451520:3715452526, ack 3669371285, win 45, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727557 eth0  Out IP (tos 0x0, ttl 64, id 37230, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715452526:3715453532, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727562 eth0  In  IP (tos 0x0, ttl 63, id 49013, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x81d2 (correct), ack 3715450514, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727567 eth0  Out IP (tos 0x0, ttl 64, id 37231, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715453532:3715454538, ack 3669371285, win 46, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727579 eth0  Out IP (tos 0x0, ttl 64, id 37232, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715454538:3715455544, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727589 eth0  Out IP (tos 0x0, ttl 64, id 37233, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715455544:3715456550, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727600 eth0  Out IP (tos 0x0, ttl 64, id 37234, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715456550:3715457556, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727608 eth0  In  IP (tos 0x0, ttl 63, id 49014, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x79f6 (correct), ack 3715452526, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727609 eth0  Out IP (tos 0x0, ttl 64, id 37235, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715457556:3715458562, ack 3669371285, win 50, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727622 eth0  Out IP (tos 0x0, ttl 64, id 37236, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715458562:3715459568, ack 3669371285, win 53, options [nop,nop,TS val 197800313 ecr 101730753], length 1006
1793456375.727632 eth0  In  IP (tos 0x0, ttl 63, id 49015, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x721a (correct), ack 3715454538, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727635 eth0  Out IP (tos 0x0, ttl 64, id 37237, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715459568:3715460574, ack 3669371285, win 53, options [nop,nop,TS val 197800314 ecr 101730753], length 1006
1793456375.727638 eth0  In  IP (tos 0x0, ttl 63, id 49016, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x6a3e (correct), ack 3715456550, win 67, options [nop,nop,TS val 101730753 ecr 197800313], length 0
1793456375.727647 eth0  In  IP (tos 0x0, ttl 63, id 49017, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x6261 (correct), ack 3715458562, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727651 eth0  Out IP (tos 0x0, ttl 64, id 37238, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715460574:3715461580, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727661 eth0  Out IP (tos 0x0, ttl 64, id 37239, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715461580:3715462586, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727672 eth0  Out IP (tos 0x0, ttl 64, id 37240, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715462586:3715463592, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727675 eth0  In  IP (tos 0x0, ttl 63, id 49018, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x5a85 (correct), ack 3715460574, win 67, options [nop,nop,TS val 101730754 ecr 197800313], length 0
1793456375.727682 eth0  Out IP (tos 0x0, ttl 64, id 37241, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715463592:3715464598, ack 3669371285, win 56, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727694 eth0  Out IP (tos 0x0, ttl 64, id 37242, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715464598:3715465604, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727704 eth0  Out IP (tos 0x0, ttl 64, id 37243, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715465604:3715466610, ack 3669371285, win 59, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727716 eth0  Out IP (tos 0x0, ttl 64, id 37244, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715466610:3715467616, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727721 eth0  In  IP (tos 0x0, ttl 63, id 49019, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x4eba (correct), ack 3715463592, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727725 eth0  Out IP (tos 0x0, ttl 64, id 37245, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715467616:3715468622, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727737 eth0  In  IP (tos 0x0, ttl 63, id 49020, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x46de (correct), ack 3715465604, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727741 eth0  Out IP (tos 0x0, ttl 64, id 37246, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715468622:3715469628, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727750 eth0  Out IP (tos 0x0, ttl 64, id 37247, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715469628:3715470634, ack 3669371285, win 62, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727761 eth0  Out IP (tos 0x0, ttl 64, id 37248, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715470634:3715471640, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727762 eth0  In  IP (tos 0x0, ttl 63, id 49021, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x3f02 (correct), ack 3715467616, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727772 eth0  Out IP (tos 0x0, ttl 64, id 37249, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715471640:3715472646, ack 3669371285, win 65, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727787 eth0  Out IP (tos 0x0, ttl 64, id 37250, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715472646:3715473652, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727797 eth0  Out IP (tos 0x0, ttl 64, id 37251, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715473652:3715474658, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727800 eth0  In  IP (tos 0x0, ttl 63, id 49022, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x3726 (correct), ack 3715469628, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727808 eth0  Out IP (tos 0x0, ttl 64, id 37252, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [P.], seq 3715474658:3715475664, ack 3669371285, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 1006
1793456375.727811 eth0  In  IP (tos 0x0, ttl 63, id 49023, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x2f4a (correct), ack 3715471640, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.727824 eth0  In  IP (tos 0x0, ttl 63, id 49024, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x276e (correct), ack 3715473652, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728074 eth0  In  IP (tos 0x0, ttl 63, id 49025, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x1f92 (correct), ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728078 eth0  In  IP (tos 0x0, ttl 63, id 49026, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [F.], cksum 0x1f91 (correct), seq 3669371285, ack 3715475664, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0
1793456375.728221 eth0  Out IP (tos 0x0, ttl 64, id 37253, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37946: Flags [F.], cksum 0x8645 (incorrect -> 0x1f91), seq 3715475664, ack 3669371286, win 66, options [nop,nop,TS val 197800314 ecr 101730754], length 0
1793456375.728573 eth0  In  IP (tos 0x0, ttl 63, id 49027, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37946 > 192.0.3.10.9446: Flags [.], cksum 0x1f90 (correct), ack 3715475665, win 67, options [nop,nop,TS val 101730754 ecr 197800314], length 0

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

- matrix4_capture_2_receipt: {"bytes":"14783","sha256":"d16995822698c69472a91e62adb5fb5184eaca6e397d91a1a5294837068703f1"}
- matrix4_origin_endpoint_samples: {"at":"2026-10-31T14:19:35.728093876Z","local_port":9446,"remote_port":37946,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":97,"sacked":0,"snd_cwnd":33,"snd_mss":1228,"snd_wnd":68608,"state":8,"total_retrans":0,"unacked":0}}

- matrix6_input: {"input":{"underlay":"ipv6","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix6_transfer: {"after":{"upload_bytes":"133925","download_bytes":"136408","Present":true},"before":{"upload_bytes":"68457","download_bytes":"69361","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:19:42.864981179Z","local_port":37962,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":202000,"rtt_us":1330,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":105472,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":8},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv6"}}
- matrix6_capture_0_window: {"launch_requested_at":"2026-10-31T14:19:41.556651867Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:43.113343413Z","stop_requested_at":"2026-10-31T14:19:43.031536568Z"}
- matrix6_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
213 packets captured
217 packets received by filter
0 packets dropped by kernel

- matrix6_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456382.856824 fvnb0 Out IP (tos 0x0, ttl 64, id 12082, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [S], cksum 0xa804 (incorrect -> 0xed01), seq 3710504557, win 64480, options [mss 1240,sackOK,TS val 337198911 ecr 0,nop,wscale 10], length 0
1793456382.858582 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [S.], cksum 0xafd3 (correct), seq 3684918615, ack 3710504558, win 65160, options [mss 1460,sackOK,TS val 2285662305 ecr 337198911,nop,wscale 10], length 0
1793456382.858602 fvnb0 Out IP (tos 0x0, ttl 64, id 12083, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xdceb), ack 3684918616, win 63, options [nop,nop,TS val 337198912 ecr 2285662305], length 0
1793456382.858752 fvnb0 Out IP (tos 0x0, ttl 64, id 12084, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], cksum 0xa828 (incorrect -> 0x8b6e), seq 3710504558:3710504602, ack 3684918616, win 63, options [nop,nop,TS val 337198913 ecr 2285662305], length 44
1793456382.860657 fvnb0 In  IP (tos 0x0, ttl 63, id 46090, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xdcbb (correct), ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 0
1793456382.860658 fvnb0 In  IP (tos 0x0, ttl 63, id 46091, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], cksum 0x8365 (correct), seq 3684918616:3684918619, ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 3
1793456382.860669 fvnb0 Out IP (tos 0x0, ttl 64, id 12085, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xdcb7), ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 0
1793456382.860791 fvnb0 Out IP (tos 0x0, ttl 64, id 12086, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710504602:3710510742, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 6140
1793456382.860808 fvnb0 Out IP (tos 0x0, ttl 64, id 12091, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710510742:3710516882, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 6140
1793456382.862673 fvnb0 In  IP (tos 0x0, ttl 63, id 46092, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xbb24 (correct), ack 3710513198, win 60, options [nop,nop,TS val 2285662309 ecr 337198915], length 0
1793456382.862674 fvnb0 In  IP (tos 0x0, ttl 63, id 46093, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684918619:3684919625, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862674 fvnb0 In  IP (tos 0x0, ttl 63, id 46094, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684919625:3684920631, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862675 fvnb0 In  IP (tos 0x0, ttl 63, id 46095, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684920631:3684921637, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862675 fvnb0 In  IP (tos 0x0, ttl 63, id 46096, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684921637:3684922643, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862675 fvnb0 In  IP (tos 0x0, ttl 63, id 46097, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684922643:3684923649, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862688 fvnb0 Out IP (tos 0x0, ttl 64, id 12096, offset 0, flags [DF], proto TCP (6), length 9876)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710516882:3710526706, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 9824
1793456382.862699 fvnb0 Out IP (tos 0x0, ttl 64, id 12104, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710526706:3710534074, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 7368
1793456382.862711 fvnb0 In  IP (tos 0x0, ttl 63, id 46098, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684923649:3684924655, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862713 fvnb0 In  IP (tos 0x0, ttl 63, id 46099, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684924655:3684925661, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862713 fvnb0 In  IP (tos 0x0, ttl 63, id 46100, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684925661:3684926667, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862713 fvnb0 In  IP (tos 0x0, ttl 63, id 46101, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684926667:3684927673, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862714 fvnb0 In  IP (tos 0x0, ttl 63, id 46102, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684927673:3684928679, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862719 fvnb0 Out IP (tos 0x0, ttl 64, id 12110, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x55f1), ack 3684923649, win 59, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862740 fvnb0 Out IP (tos 0x0, ttl 64, id 12111, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x424f), ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862743 fvnb0 Out IP (tos 0x0, ttl 64, id 12112, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710534074:3710541442, ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 7368
1793456382.863916 fvnb0 In  IP (tos 0x0, ttl 63, id 46103, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0x723c (correct), ack 3710521794, win 65, options [nop,nop,TS val 2285662310 ecr 337198917], length 0
1793456382.863917 fvnb0 In  IP (tos 0x0, ttl 63, id 46104, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684928679:3684935721, ack 3710534074, win 57, options [nop,nop,TS val 2285662310 ecr 337198917], length 7042
1793456382.863918 fvnb0 In  IP (tos 0x0, ttl 63, id 46110, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684935721:3684936727, ack 3710538986, win 53, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863929 fvnb0 Out IP (tos 0x0, ttl 64, id 12118, offset 0, flags [DF], proto TCP (6), length 9876)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710541442:3710551266, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 9824
1793456382.863941 fvnb0 Out IP (tos 0x0, ttl 64, id 12126, offset 0, flags [DF], proto TCP (6), length 17452)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710551266:3710568666, ack 3684935721, win 59, options [nop,nop,TS val 337198918 ecr 2285662310], length 17400
1793456382.863955 fvnb0 In  IP (tos 0x0, ttl 63, id 46111, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684936727:3684937733, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863956 fvnb0 In  IP (tos 0x0, ttl 63, id 46112, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684937733:3684938739, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863957 fvnb0 In  IP (tos 0x0, ttl 63, id 46113, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684938739:3684939745, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863957 fvnb0 In  IP (tos 0x0, ttl 63, id 46114, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684939745:3684940751, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863957 fvnb0 In  IP (tos 0x0, ttl 63, id 46115, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684940751:3684941757, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863957 fvnb0 In  IP (tos 0x0, ttl 63, id 46116, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684941757:3684942763, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863958 fvnb0 In  IP (tos 0x0, ttl 63, id 46117, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684942763:3684943769, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863958 fvnb0 In  IP (tos 0x0, ttl 63, id 46118, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684943769:3684944775, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863964 fvnb0 In  IP (tos 0x0, ttl 63, id 46119, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684944775:3684945781, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863965 fvnb0 In  IP (tos 0x0, ttl 63, id 46120, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684945781:3684946787, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863965 fvnb0 In  IP (tos 0x0, ttl 63, id 46121, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684946787:3684947793, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863965 fvnb0 In  IP (tos 0x0, ttl 63, id 46122, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684947793:3684948799, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863965 fvnb0 In  IP (tos 0x0, ttl 63, id 46123, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684948799:3684949805, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863974 fvnb0 Out IP (tos 0x0, ttl 64, id 12141, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x68af), ack 3684949805, win 46, options [nop,nop,TS val 337198918 ecr 2285662310], length 0
1793456382.864887 fvnb0 In  IP (tos 0x0, ttl 63, id 46124, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xc95a (correct), ack 3710543898, win 66, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864888 fvnb0 In  IP (tos 0x0, ttl 63, id 46125, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xb151 (correct), ack 3710550038, win 79, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864888 fvnb0 In  IP (tos 0x0, ttl 63, id 46126, offset 0, flags [DF], proto TCP (6), length 10112)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684949805:3684959865, ack 3710558634, win 97, options [nop,nop,TS val 2285662311 ecr 337198918], length 10060
1793456382.864902 fvnb0 Out IP (tos 0x0, ttl 64, id 12142, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x4157), ack 3684959865, win 56, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.864909 fvnb0 In  IP (tos 0x0, ttl 63, id 46135, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0x4129 (correct), ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864910 fvnb0 In  IP (tos 0x0, ttl 63, id 46136, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684959865:3684966907, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 7042
1793456382.864910 fvnb0 In  IP (tos 0x0, ttl 63, id 46142, offset 0, flags [DF], proto TCP (6), length 2064)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684966907:3684968919, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 2012
1793456382.864916 fvnb0 In  IP (tos 0x0, ttl 63, id 46144, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684968919:3684969925, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864916 fvnb0 In  IP (tos 0x0, ttl 63, id 46145, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684969925:3684970931, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864917 fvnb0 In  IP (tos 0x0, ttl 63, id 46146, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684970931:3684971937, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864917 fvnb0 In  IP (tos 0x0, ttl 63, id 46147, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684971937:3684972943, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864917 fvnb0 In  IP (tos 0x0, ttl 63, id 46148, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684972943:3684973949, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864918 fvnb0 In  IP (tos 0x0, ttl 63, id 46149, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684973949:3684974955, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864918 fvnb0 In  IP (tos 0x0, ttl 63, id 46150, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684974955:3684975961, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864918 fvnb0 In  IP (tos 0x0, ttl 63, id 46151, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684975961:3684976967, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864919 fvnb0 Out IP (tos 0x0, ttl 64, id 12143, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x1e01), ack 3684968919, win 48, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.864923 fvnb0 In  IP (tos 0x0, ttl 63, id 46152, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684976967:3684977973, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864925 fvnb0 In  IP (tos 0x0, ttl 63, id 46153, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684977973:3684978979, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864925 fvnb0 In  IP (tos 0x0, ttl 63, id 46154, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684978979:3684979985, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864925 fvnb0 In  IP (tos 0x0, ttl 63, id 46155, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684979985:3684980991, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864926 fvnb0 In  IP (tos 0x0, ttl 63, id 46156, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684980991:3684981997, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864926 fvnb0 In  IP (tos 0x0, ttl 63, id 46157, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684981997:3684983003, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864936 fvnb0 Out IP (tos 0x0, ttl 64, id 12144, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xe707), ack 3684983003, win 37, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865192 fvnb0 Out IP (tos 0x0, ttl 64, id 12145, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [F.], cksum 0xa7fc (incorrect -> 0xe6f0), seq 3710568666, ack 3684983003, win 59, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865835 fvnb0 In  IP (tos 0x0, ttl 63, id 46158, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [F.], cksum 0xe6c2 (correct), seq 3684983003, ack 3710568667, win 103, options [nop,nop,TS val 2285662312 ecr 337198919], length 0
1793456382.865847 fvnb0 Out IP (tos 0x0, ttl 64, id 12146, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xe6ed), ack 3684983004, win 59, options [nop,nop,TS val 337198920 ecr 2285662312], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4320,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:6d:bb:cd:a6:42","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4323,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"62:1a:d1:c5:55:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":6,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.83.128.120","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:45832 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.061/0.084 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5113 segs_out:16 segs_in:16 data_segs_out:9 data_segs_in:11 send 1899016393bps lastsnd:38579 lastrcv:38578 lastack:8408 pacing_rate 3744323232bps delivery_rate 3861333328bps delivered:10 app_limited rcv_space:14480 rcv_ssthresh:81050 minrtt:0.006 snd_wnd:71680
ESTAB 0      0       172.28.30.41:53996 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.758/2.951 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3775 bytes_acked:3776 bytes_received:6215 segs_out:48 segs_in:42 data_segs_out:24 data_segs_in:36 send 65893060bps lastsnd:217 lastrcv:217 lastack:217 pacing_rate 131786120bps delivery_rate 1930666664bps delivered:25 app_limited busy:49ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.012 snd_wnd:79872
ESTAB 0      0       172.28.30.41:54006 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.773/3.365 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3407 bytes_acked:3408 bytes_received:5996 segs_out:29 segs_in:33 data_segs_out:16 data_segs_in:28 send 65335589bps lastsnd:5298 lastrcv:5298 lastack:5298 pacing_rate 130671176bps delivery_rate 4633600000bps delivered:17 app_limited busy:37ms rcv_space:14480 rcv_ssthresh:85684 minrtt:0.005 snd_wnd:78848
ESTAB 0      0       172.28.30.41:54022 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.237/10.37 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2927 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 22119534bps lastsnd:5298 lastrcv:5299 lastack:5257 pacing_rate 44231672bps delivery_rate 1544533328bps delivered:9 app_limited busy:41ms rcv_space:14480 rcv_ssthresh:79826 minrtt:0.012 snd_wnd:74752

- matrix6_capture_0_receipt: {"bytes":"29378","sha256":"a6b200a1c7a3720a58c045d551e9062c6fc3e7bff47f7c65a31518afeb4bb21f"}
- matrix6_capture_1_window: {"launch_requested_at":"2026-10-31T14:19:41.596422608Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:43.467521194Z","stop_requested_at":"2026-10-31T14:19:43.301287124Z"}
- matrix6_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
297 packets captured
297 packets received by filter
0 packets dropped by kernel

- matrix6_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456382.857707 fvmb0 In  IP (tos 0x0, ttl 64, id 12082, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [S], cksum 0xed01 (correct), seq 3710504557, win 64480, options [mss 1240,sackOK,TS val 337198911 ecr 0,nop,wscale 10], length 0
1793456382.857739 fvup0 Out IP (tos 0x0, ttl 63, id 12082, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [S], cksum 0x0eb9 (correct), seq 3710504557, win 64480, options [mss 1240,sackOK,TS val 337198911 ecr 0,nop,wscale 10], length 0
1793456382.857772 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [S.], cksum 0x864d (incorrect -> 0xd18a), seq 3684918615, ack 3710504558, win 65160, options [mss 1460,sackOK,TS val 2285662305 ecr 337198911,nop,wscale 10], length 0
1793456382.857778 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [S.], cksum 0xa804 (incorrect -> 0xafd3), seq 3684918615, ack 3710504558, win 65160, options [mss 1460,sackOK,TS val 2285662305 ecr 337198911,nop,wscale 10], length 0
1793456382.859665 fvmb0 In  IP (tos 0x0, ttl 64, id 12083, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xdceb (correct), ack 3684918616, win 63, options [nop,nop,TS val 337198912 ecr 2285662305], length 0
1793456382.859666 fvmb0 In  IP (tos 0x0, ttl 64, id 12084, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], cksum 0x8b6e (correct), seq 3710504558:3710504602, ack 3684918616, win 63, options [nop,nop,TS val 337198913 ecr 2285662305], length 44
1793456382.859674 fvup0 Out IP (tos 0x0, ttl 63, id 12083, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0xfea2 (correct), ack 3684918616, win 63, options [nop,nop,TS val 337198912 ecr 2285662305], length 0
1793456382.859677 fvup0 Out IP (tos 0x0, ttl 63, id 12084, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], cksum 0xad25 (correct), seq 3710504558:3710504602, ack 3684918616, win 63, options [nop,nop,TS val 337198913 ecr 2285662305], length 44
1793456382.859701 fvup0 In  IP (tos 0x0, ttl 64, id 46090, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xfe72), ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 0
1793456382.859705 fvmb0 Out IP (tos 0x0, ttl 63, id 46090, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xa7fc (incorrect -> 0xdcbb), ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 0
1793456382.859823 fvup0 In  IP (tos 0x0, ttl 64, id 46091, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], cksum 0x8648 (incorrect -> 0xa51c), seq 3684918616:3684918619, ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 3
1793456382.859831 fvmb0 Out IP (tos 0x0, ttl 63, id 46091, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], cksum 0xa7ff (incorrect -> 0x8365), seq 3684918616:3684918619, ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 3
1793456382.861665 fvmb0 In  IP (tos 0x0, ttl 64, id 12085, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xdcb7 (correct), ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 0
1793456382.861671 fvup0 Out IP (tos 0x0, ttl 63, id 12085, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0xfe6e (correct), ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 0
1793456382.861682 fvmb0 In  IP (tos 0x0, ttl 64, id 12086, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710504602:3710510742, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 6140
1793456382.861686 fvup0 Out IP (tos 0x0, ttl 63, id 12086, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710504602:3710510742, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 6140
1793456382.861692 fvmb0 In  IP (tos 0x0, ttl 64, id 12091, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], seq 3710510742:3710513198, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 2456
1793456382.861695 fvup0 Out IP (tos 0x0, ttl 63, id 12091, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710510742:3710513198, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 2456
1793456382.861703 fvmb0 In  IP (tos 0x0, ttl 64, id 12093, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710513198:3710516882, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 3684
1793456382.861706 fvup0 Out IP (tos 0x0, ttl 63, id 12093, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710513198:3710516882, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 3684
1793456382.861708 fvup0 In  IP (tos 0x0, ttl 64, id 46092, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xdcdb), ack 3710513198, win 60, options [nop,nop,TS val 2285662309 ecr 337198915], length 0
1793456382.861717 fvmb0 Out IP (tos 0x0, ttl 63, id 46092, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xa7fc (incorrect -> 0xbb24), ack 3710513198, win 60, options [nop,nop,TS val 2285662309 ecr 337198915], length 0
1793456382.861749 fvup0 In  IP (tos 0x0, ttl 64, id 46093, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684918619:3684919625, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861752 fvmb0 Out IP (tos 0x0, ttl 63, id 46093, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684918619:3684919625, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861765 fvup0 In  IP (tos 0x0, ttl 64, id 46094, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684919625:3684920631, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861767 fvmb0 Out IP (tos 0x0, ttl 63, id 46094, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684919625:3684920631, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861788 fvup0 In  IP (tos 0x0, ttl 64, id 46095, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684920631:3684921637, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861790 fvmb0 Out IP (tos 0x0, ttl 63, id 46095, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684920631:3684921637, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861803 fvup0 In  IP (tos 0x0, ttl 64, id 46096, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684921637:3684922643, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861806 fvmb0 Out IP (tos 0x0, ttl 63, id 46096, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684921637:3684922643, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861816 fvup0 In  IP (tos 0x0, ttl 64, id 46097, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684922643:3684923649, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861818 fvmb0 Out IP (tos 0x0, ttl 63, id 46097, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684922643:3684923649, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861827 fvup0 In  IP (tos 0x0, ttl 64, id 46098, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684923649:3684924655, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861829 fvmb0 Out IP (tos 0x0, ttl 63, id 46098, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684923649:3684924655, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861839 fvup0 In  IP (tos 0x0, ttl 64, id 46099, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684924655:3684925661, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861842 fvmb0 Out IP (tos 0x0, ttl 63, id 46099, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684924655:3684925661, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861850 fvup0 In  IP (tos 0x0, ttl 64, id 46100, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684925661:3684926667, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861852 fvmb0 Out IP (tos 0x0, ttl 63, id 46100, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684925661:3684926667, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861871 fvup0 In  IP (tos 0x0, ttl 64, id 46101, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684926667:3684927673, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861873 fvmb0 Out IP (tos 0x0, ttl 63, id 46101, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684926667:3684927673, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861880 fvup0 In  IP (tos 0x0, ttl 64, id 46102, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684927673:3684928679, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861882 fvmb0 Out IP (tos 0x0, ttl 63, id 46102, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684927673:3684928679, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862745 fvmb0 In  IP (tos 0x0, ttl 64, id 12096, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], seq 3710516882:3710518110, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 1228
1793456382.862749 fvup0 Out IP (tos 0x0, ttl 63, id 12096, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710516882:3710518110, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 1228
1793456382.862755 fvmb0 In  IP (tos 0x0, ttl 64, id 12097, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], seq 3710518110:3710521794, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 3684
1793456382.862757 fvup0 Out IP (tos 0x0, ttl 63, id 12097, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710518110:3710521794, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 3684
1793456382.862763 fvup0 In  IP (tos 0x0, ttl 64, id 46103, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0x93f3), ack 3710521794, win 65, options [nop,nop,TS val 2285662310 ecr 337198917], length 0
1793456382.862766 fvmb0 Out IP (tos 0x0, ttl 63, id 46103, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xa7fc (incorrect -> 0x723c), ack 3710521794, win 65, options [nop,nop,TS val 2285662310 ecr 337198917], length 0
1793456382.862794 fvmb0 In  IP (tos 0x0, ttl 64, id 12100, offset 0, flags [DF], proto TCP (6), length 4964)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710521794:3710526706, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 4912
1793456382.862794 fvmb0 In  IP (tos 0x0, ttl 64, id 12104, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710526706:3710534074, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 7368
1793456382.862798 fvup0 Out IP (tos 0x0, ttl 63, id 12100, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710521794:3710526706, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 4912
1793456382.862800 fvup0 Out IP (tos 0x0, ttl 63, id 12104, offset 0, flags [DF], proto TCP (6), length 7420)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710526706:3710534074, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 7368
1793456382.862803 fvmb0 In  IP (tos 0x0, ttl 64, id 12110, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0x55f1 (correct), ack 3684923649, win 59, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862803 fvmb0 In  IP (tos 0x0, ttl 64, id 12111, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0x424f (correct), ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862803 fvmb0 In  IP (tos 0x0, ttl 64, id 12112, offset 0, flags [DF], proto TCP (6), length 4964)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], seq 3710534074:3710538986, ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 4912
1793456382.862806 fvup0 Out IP (tos 0x0, ttl 63, id 12110, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x77a8 (correct), ack 3684923649, win 59, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862808 fvup0 Out IP (tos 0x0, ttl 63, id 12111, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x6406 (correct), ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862810 fvup0 Out IP (tos 0x0, ttl 63, id 12112, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710534074:3710538986, ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 4912
1793456382.862823 fvup0 In  IP (tos 0x0, ttl 64, id 46104, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684928679:3684935721, ack 3710534074, win 57, options [nop,nop,TS val 2285662310 ecr 337198917], length 7042
1793456382.862826 fvmb0 Out IP (tos 0x0, ttl 63, id 46104, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684928679:3684935721, ack 3710534074, win 57, options [nop,nop,TS val 2285662310 ecr 337198917], length 7042
1793456382.862833 fvmb0 In  IP (tos 0x0, ttl 64, id 12116, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710538986:3710541442, ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 2456
1793456382.862836 fvup0 Out IP (tos 0x0, ttl 63, id 12116, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710538986:3710541442, ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 2456
1793456382.862838 fvup0 In  IP (tos 0x0, ttl 64, id 46110, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684935721:3684936727, ack 3710538986, win 53, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862843 fvmb0 Out IP (tos 0x0, ttl 63, id 46110, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684935721:3684936727, ack 3710538986, win 53, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862950 fvup0 In  IP (tos 0x0, ttl 64, id 46111, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684936727:3684937733, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862954 fvmb0 Out IP (tos 0x0, ttl 63, id 46111, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684936727:3684937733, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862979 fvup0 In  IP (tos 0x0, ttl 64, id 46112, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684937733:3684938739, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862982 fvmb0 Out IP (tos 0x0, ttl 63, id 46112, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684937733:3684938739, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862990 fvup0 In  IP (tos 0x0, ttl 64, id 46113, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684938739:3684939745, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862993 fvmb0 Out IP (tos 0x0, ttl 63, id 46113, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684938739:3684939745, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863003 fvup0 In  IP (tos 0x0, ttl 64, id 46114, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684939745:3684940751, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863005 fvmb0 Out IP (tos 0x0, ttl 63, id 46114, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684939745:3684940751, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863013 fvup0 In  IP (tos 0x0, ttl 64, id 46115, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684940751:3684941757, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863015 fvmb0 Out IP (tos 0x0, ttl 63, id 46115, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684940751:3684941757, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863024 fvup0 In  IP (tos 0x0, ttl 64, id 46116, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684941757:3684942763, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863027 fvmb0 Out IP (tos 0x0, ttl 63, id 46116, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684941757:3684942763, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863049 fvup0 In  IP (tos 0x0, ttl 64, id 46117, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684942763:3684943769, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863052 fvmb0 Out IP (tos 0x0, ttl 63, id 46117, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684942763:3684943769, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863064 fvup0 In  IP (tos 0x0, ttl 64, id 46118, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684943769:3684944775, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863067 fvmb0 Out IP (tos 0x0, ttl 63, id 46118, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684943769:3684944775, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863074 fvup0 In  IP (tos 0x0, ttl 64, id 46119, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684944775:3684945781, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863077 fvmb0 Out IP (tos 0x0, ttl 63, id 46119, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684944775:3684945781, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863087 fvup0 In  IP (tos 0x0, ttl 64, id 46120, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684945781:3684946787, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863090 fvmb0 Out IP (tos 0x0, ttl 63, id 46120, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684945781:3684946787, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863103 fvup0 In  IP (tos 0x0, ttl 64, id 46121, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684946787:3684947793, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863105 fvmb0 Out IP (tos 0x0, ttl 63, id 46121, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684946787:3684947793, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863197 fvup0 In  IP (tos 0x0, ttl 64, id 46122, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684947793:3684948799, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863200 fvmb0 Out IP (tos 0x0, ttl 63, id 46122, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684947793:3684948799, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863210 fvup0 In  IP (tos 0x0, ttl 64, id 46123, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684948799:3684949805, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863213 fvmb0 Out IP (tos 0x0, ttl 63, id 46123, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684948799:3684949805, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.864116 fvmb0 In  IP (tos 0x0, ttl 64, id 12118, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], seq 3710541442:3710543898, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 2456
1793456382.864122 fvup0 Out IP (tos 0x0, ttl 63, id 12118, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710541442:3710543898, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 2456
1793456382.864131 fvup0 In  IP (tos 0x0, ttl 64, id 46124, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xeb11), ack 3710543898, win 66, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864133 fvmb0 Out IP (tos 0x0, ttl 63, id 46124, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xa7fc (incorrect -> 0xc95a), ack 3710543898, win 66, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864138 fvmb0 In  IP (tos 0x0, ttl 64, id 12120, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], seq 3710543898:3710550038, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 6140
1793456382.864141 fvup0 Out IP (tos 0x0, ttl 63, id 12120, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710543898:3710550038, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 6140
1793456382.864145 fvup0 In  IP (tos 0x0, ttl 64, id 46125, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xd308), ack 3710550038, win 79, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864148 fvmb0 Out IP (tos 0x0, ttl 63, id 46125, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xa7fc (incorrect -> 0xb151), ack 3710550038, win 79, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864150 fvmb0 In  IP (tos 0x0, ttl 64, id 12125, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710550038:3710551266, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 1228
1793456382.864151 fvmb0 In  IP (tos 0x0, ttl 64, id 12126, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], seq 3710551266:3710558634, ack 3684935721, win 59, options [nop,nop,TS val 337198918 ecr 2285662310], length 7368
1793456382.864154 fvup0 Out IP (tos 0x0, ttl 63, id 12125, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710550038:3710551266, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 1228
1793456382.864156 fvup0 Out IP (tos 0x0, ttl 63, id 12126, offset 0, flags [DF], proto TCP (6), length 7420)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710551266:3710558634, ack 3684935721, win 59, options [nop,nop,TS val 337198918 ecr 2285662310], length 7368
1793456382.864164 fvup0 In  IP (tos 0x0, ttl 64, id 46126, offset 0, flags [DF], proto TCP (6), length 10112)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684949805:3684959865, ack 3710558634, win 97, options [nop,nop,TS val 2285662311 ecr 337198918], length 10060
1793456382.864169 fvmb0 Out IP (tos 0x0, ttl 63, id 46126, offset 0, flags [DF], proto TCP (6), length 10112)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684949805:3684959865, ack 3710558634, win 97, options [nop,nop,TS val 2285662311 ecr 337198918], length 10060
1793456382.864178 fvmb0 In  IP (tos 0x0, ttl 64, id 12132, offset 0, flags [DF], proto TCP (6), length 10084)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [P.], seq 3710558634:3710568666, ack 3684935721, win 59, options [nop,nop,TS val 337198918 ecr 2285662310], length 10032
1793456382.864183 fvup0 Out IP (tos 0x0, ttl 63, id 12132, offset 0, flags [DF], proto TCP (6), length 10084)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710558634:3710568666, ack 3684935721, win 59, options [nop,nop,TS val 337198918 ecr 2285662310], length 10032
1793456382.864184 fvmb0 In  IP (tos 0x0, ttl 64, id 12141, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0x68af (correct), ack 3684949805, win 46, options [nop,nop,TS val 337198918 ecr 2285662310], length 0
1793456382.864186 fvup0 Out IP (tos 0x0, ttl 63, id 12141, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x8a66 (correct), ack 3684949805, win 46, options [nop,nop,TS val 337198918 ecr 2285662310], length 0
1793456382.864197 fvup0 In  IP (tos 0x0, ttl 64, id 46135, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0x62e0), ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864199 fvmb0 Out IP (tos 0x0, ttl 63, id 46135, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [.], cksum 0xa7fc (incorrect -> 0x4129), ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864200 fvup0 In  IP (tos 0x0, ttl 64, id 46136, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684959865:3684966907, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 7042
1793456382.864203 fvmb0 Out IP (tos 0x0, ttl 63, id 46136, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684959865:3684966907, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 7042
1793456382.864206 fvup0 In  IP (tos 0x0, ttl 64, id 46142, offset 0, flags [DF], proto TCP (6), length 2064)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684966907:3684968919, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 2012
1793456382.864209 fvmb0 Out IP (tos 0x0, ttl 63, id 46142, offset 0, flags [DF], proto TCP (6), length 2064)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684966907:3684968919, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 2012
1793456382.864220 fvup0 In  IP (tos 0x0, ttl 64, id 46144, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684968919:3684969925, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864222 fvmb0 Out IP (tos 0x0, ttl 63, id 46144, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684968919:3684969925, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864233 fvup0 In  IP (tos 0x0, ttl 64, id 46145, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684969925:3684970931, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864235 fvmb0 Out IP (tos 0x0, ttl 63, id 46145, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684969925:3684970931, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864243 fvup0 In  IP (tos 0x0, ttl 64, id 46146, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684970931:3684971937, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864246 fvmb0 Out IP (tos 0x0, ttl 63, id 46146, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684970931:3684971937, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864255 fvup0 In  IP (tos 0x0, ttl 64, id 46147, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684971937:3684972943, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864258 fvmb0 Out IP (tos 0x0, ttl 63, id 46147, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684971937:3684972943, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864265 fvup0 In  IP (tos 0x0, ttl 64, id 46148, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684972943:3684973949, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864267 fvmb0 Out IP (tos 0x0, ttl 63, id 46148, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684972943:3684973949, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864278 fvup0 In  IP (tos 0x0, ttl 64, id 46149, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684973949:3684974955, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864280 fvmb0 Out IP (tos 0x0, ttl 63, id 46149, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684973949:3684974955, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864288 fvup0 In  IP (tos 0x0, ttl 64, id 46150, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684974955:3684975961, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864290 fvmb0 Out IP (tos 0x0, ttl 63, id 46150, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684974955:3684975961, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864300 fvup0 In  IP (tos 0x0, ttl 64, id 46151, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684975961:3684976967, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864302 fvmb0 Out IP (tos 0x0, ttl 63, id 46151, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684975961:3684976967, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864315 fvup0 In  IP (tos 0x0, ttl 64, id 46152, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684976967:3684977973, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864317 fvmb0 Out IP (tos 0x0, ttl 63, id 46152, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684976967:3684977973, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864326 fvup0 In  IP (tos 0x0, ttl 64, id 46153, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684977973:3684978979, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864329 fvmb0 Out IP (tos 0x0, ttl 63, id 46153, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684977973:3684978979, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864336 fvup0 In  IP (tos 0x0, ttl 64, id 46154, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684978979:3684979985, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864339 fvmb0 Out IP (tos 0x0, ttl 63, id 46154, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684978979:3684979985, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864347 fvup0 In  IP (tos 0x0, ttl 64, id 46155, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684979985:3684980991, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864350 fvmb0 Out IP (tos 0x0, ttl 63, id 46155, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684979985:3684980991, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864359 fvup0 In  IP (tos 0x0, ttl 64, id 46156, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684980991:3684981997, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864361 fvmb0 Out IP (tos 0x0, ttl 63, id 46156, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684980991:3684981997, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864371 fvup0 In  IP (tos 0x0, ttl 64, id 46157, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684981997:3684983003, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864374 fvmb0 Out IP (tos 0x0, ttl 63, id 46157, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [P.], seq 3684981997:3684983003, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.865045 fvmb0 In  IP (tos 0x0, ttl 64, id 12142, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0x4157 (correct), ack 3684959865, win 56, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865047 fvmb0 In  IP (tos 0x0, ttl 64, id 12143, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0x1e01 (correct), ack 3684968919, win 48, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865051 fvup0 Out IP (tos 0x0, ttl 63, id 12142, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x630e (correct), ack 3684959865, win 56, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865054 fvup0 Out IP (tos 0x0, ttl 63, id 12143, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x3fb8 (correct), ack 3684968919, win 48, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865211 fvmb0 In  IP (tos 0x0, ttl 64, id 12144, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xe707 (correct), ack 3684983003, win 37, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865226 fvup0 Out IP (tos 0x0, ttl 63, id 12144, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x08bf (correct), ack 3684983003, win 37, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865242 fvmb0 In  IP (tos 0x0, ttl 64, id 12145, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [F.], cksum 0xe6f0 (correct), seq 3710568666, ack 3684983003, win 59, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865245 fvup0 Out IP (tos 0x0, ttl 63, id 12145, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [F.], cksum 0x08a8 (correct), seq 3710568666, ack 3684983003, win 59, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865362 fvup0 In  IP (tos 0x0, ttl 64, id 46158, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [F.], cksum 0x8645 (incorrect -> 0x087a), seq 3684983003, ack 3710568667, win 103, options [nop,nop,TS val 2285662312 ecr 337198919], length 0
1793456382.865368 fvmb0 Out IP (tos 0x0, ttl 63, id 46158, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37962: Flags [F.], cksum 0xa7fc (incorrect -> 0xe6c2), seq 3684983003, ack 3710568667, win 103, options [nop,nop,TS val 2285662312 ecr 337198919], length 0
1793456382.866016 fvmb0 In  IP (tos 0x0, ttl 64, id 12146, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37962 > 192.0.3.10.9446: Flags [.], cksum 0xe6ed (correct), ack 3684983004, win 59, options [nop,nop,TS val 337198920 ecr 2285662312], length 0
1793456382.866021 fvup0 Out IP (tos 0x0, ttl 63, id 12146, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x08a5 (correct), ack 3684983004, win 59, options [nop,nop,TS val 337198920 ecr 2285662312], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4318,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"5a:8d:b5:97:e7:d6","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4319,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"5a:bc:89:da:9c:4a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4322,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"46:08:5f:12:52:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"82:09:3c:35:ad:f2","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.83.207.231","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:32844
	 cubic wscale:10,10 rto:211 rtt:10.532/16.872 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:36126 bytes_acked:36126 bytes_received:18705 segs_out:96 segs_in:127 data_segs_out:93 data_segs_in:93 send 248902393bps lastsnd:106 lastrcv:107 lastack:65 pacing_rate 497787056bps delivery_rate 52428800000bps delivered:94 app_limited busy:666ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.005 snd_wnd:65536
ESTAB 0      0       172.28.30.40:58744 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.497/0.412 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4846 bytes_acked:4847 bytes_received:10930 segs_out:98 segs_in:75 data_segs_out:40 data_segs_in:69 send 233078471bps lastsnd:175 lastrcv:175 lastack:175 pacing_rate 465454544bps delivery_rate 2896000000bps delivered:41 app_limited busy:51ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58756 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.147/0.197 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:10401 bytes_acked:10402 bytes_received:12465 segs_out:88 segs_in:104 data_segs_out:46 data_segs_in:97 send 788027211bps lastsnd:5593 lastrcv:5593 lastack:5593 pacing_rate 1573378600bps delivery_rate 2896000000bps delivered:47 app_limited busy:33ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58772 172.28.30.21:443
	 cubic wscale:10,10 rto:211 rtt:10.637/17.09 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2650 bytes_acked:2651 bytes_received:3895 segs_out:30 segs_in:28 data_segs_out:14 data_segs_in:18 send 10890289bps lastsnd:4403 lastrcv:4404 lastack:4362 pacing_rate 21779552bps delivery_rate 2106181816bps delivered:15 app_limited busy:125ms rcv_space:14480 rcv_ssthresh:85618 minrtt:0.007 snd_wnd:77824
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:60996
	 cubic wscale:10,10 rto:206 rtt:5.095/10.17 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:3152 bytes_acked:3152 bytes_received:2755 segs_out:4 segs_in:7 data_segs_out:2 data_segs_in:3 send 683336605bps lastsnd:133 lastrcv:140 lastack:92 pacing_rate 1366572624bps delivery_rate 34816000000bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78451 minrtt:0.01 snd_wnd:87040
ESTAB 0      0       172.28.30.40:34032 172.28.30.30:9443
	 cubic wscale:10,10 rto:205 rtt:4.646/4.034 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:31692 bytes_acked:31693 bytes_received:29234 segs_out:84 segs_in:51 data_segs_out:56 data_segs_in:36 send 24933276bps lastsnd:142 lastrcv:135 lastack:135 pacing_rate 49866544bps delivery_rate 2106181816bps delivered:57 app_limited busy:118ms rcv_rtt:6.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.008 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:48150
	 cubic wscale:10,10 rto:201 rtt:0.752/1.471 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:21132 bytes_acked:21132 bytes_received:24766 segs_out:32 segs_in:53 data_segs_out:27 data_segs_in:26 send 4847659574bps lastsnd:277 lastrcv:287 lastack:277 pacing_rate 9695319144bps delivery_rate 182272000000bps delivered:28 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:105032 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:60980
	 cubic wscale:10,10 rto:206 rtt:5.055/10.086 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2832 bytes_acked:2832 bytes_received:4655 segs_out:4 segs_in:8 data_segs_out:2 data_segs_in:4 send 688743818bps lastsnd:321 lastrcv:346 lastack:280 pacing_rate 1377385456bps delivery_rate 29013333328bps delivered:3 app_limited busy:41ms rcv_rtt:1 rcv_space:65483 rcv_ssthresh:78343 minrtt:0.012 snd_wnd:87040
ESTAB 0      0          127.0.0.1:32844    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.696/0.042 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:18705 bytes_acked:18706 bytes_received:36126 segs_out:127 segs_in:97 data_segs_out:93 data_segs_in:93 send 3766436782bps lastsnd:117 lastrcv:116 lastack:116 pacing_rate 7532873560bps delivery_rate 26214400000bps delivered:94 app_limited busy:120ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.01 snd_wnd:65536
ESTAB 0      0          127.0.0.1:48150    127.0.0.1:9444
	 cubic wscale:10,10 rto:207 rtt:6.303/3.658 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:24766 bytes_acked:24767 bytes_received:21132 segs_out:53 segs_in:33 data_segs_out:26 data_segs_in:27 send 669344756bps lastsnd:287 lastrcv:277 lastack:277 pacing_rate 1338609864bps delivery_rate 84377600000bps delivered:27 app_limited busy:120ms rcv_rtt:6 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.005 snd_wnd:105472
ESTAB 0      0          127.0.0.1:60980    127.0.0.1:9444
	 cubic wscale:10,10 rto:204 rtt:3.23/6.412 ato:40 mss:39424 pmtu:65535 rcvmss:1998 advmss:65483 cwnd:10 bytes_sent:4655 bytes_acked:4656 bytes_received:2832 segs_out:8 segs_in:5 data_segs_out:4 data_segs_in:2 send 976445820bps lastsnd:346 lastrcv:321 lastack:321 pacing_rate 1952664936bps delivery_rate 19712000000bps delivered:5 app_limited busy:26ms rcv_space:65495 rcv_ssthresh:86858 minrtt:0.006 snd_wnd:78848
ESTAB 0      0          127.0.0.1:60996    127.0.0.1:9444
	 cubic wscale:10,10 rto:201 rtt:0.041/0.053 ato:40 mss:39424 pmtu:65535 rcvmss:1996 advmss:65483 cwnd:10 bytes_sent:2755 bytes_acked:2756 bytes_received:3152 segs_out:7 segs_in:5 data_segs_out:3 data_segs_in:2 send 76924878049bps lastsnd:143 lastrcv:136 lastack:136 pacing_rate 151085988016bps delivery_rate 24260923072bps delivered:4 app_limited rcv_space:65495 rcv_ssthresh:86642 minrtt:0.013 snd_wnd:78848

- matrix6_capture_1_receipt: {"bytes":"40536","sha256":"5d42018f964229ab4507a4ef364d4acb2af51e1a068a94bb9c2573e83a1bfc27"}
- matrix6_capture_2_window: {"launch_requested_at":"2026-10-31T14:19:41.616481694Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:43.79188943Z","stop_requested_at":"2026-10-31T14:19:43.584677187Z"}
- matrix6_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
77 packets captured
77 packets received by filter
0 packets dropped by kernel

- matrix6_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456382.857747 eth0  In  IP (tos 0x0, ttl 63, id 12082, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [S], cksum 0x0eb9 (correct), seq 3710504557, win 64480, options [mss 1240,sackOK,TS val 337198911 ecr 0,nop,wscale 10], length 0
1793456382.857769 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [S.], cksum 0x864d (incorrect -> 0xd18a), seq 3684918615, ack 3710504558, win 65160, options [mss 1460,sackOK,TS val 2285662305 ecr 337198911,nop,wscale 10], length 0
1793456382.859679 eth0  In  IP (tos 0x0, ttl 63, id 12083, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0xfea2 (correct), ack 3684918616, win 63, options [nop,nop,TS val 337198912 ecr 2285662305], length 0
1793456382.859695 eth0  In  IP (tos 0x0, ttl 63, id 12084, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], cksum 0xad25 (correct), seq 3710504558:3710504602, ack 3684918616, win 63, options [nop,nop,TS val 337198913 ecr 2285662305], length 44
1793456382.859699 eth0  Out IP (tos 0x0, ttl 64, id 46090, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xfe72), ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 0
1793456382.859819 eth0  Out IP (tos 0x0, ttl 64, id 46091, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], cksum 0x8648 (incorrect -> 0xa51c), seq 3684918616:3684918619, ack 3710504602, win 64, options [nop,nop,TS val 2285662307 ecr 337198913], length 3
1793456382.861673 eth0  In  IP (tos 0x0, ttl 63, id 12085, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0xfe6e (correct), ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 0
1793456382.861688 eth0  In  IP (tos 0x0, ttl 63, id 12086, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710504602:3710510742, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 6140
1793456382.861696 eth0  In  IP (tos 0x0, ttl 63, id 12091, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710510742:3710513198, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 2456
1793456382.861704 eth0  Out IP (tos 0x0, ttl 64, id 46092, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xdcdb), ack 3710513198, win 60, options [nop,nop,TS val 2285662309 ecr 337198915], length 0
1793456382.861707 eth0  In  IP (tos 0x0, ttl 63, id 12093, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710513198:3710516882, ack 3684918619, win 63, options [nop,nop,TS val 337198915 ecr 2285662307], length 3684
1793456382.861747 eth0  Out IP (tos 0x0, ttl 64, id 46093, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684918619:3684919625, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861763 eth0  Out IP (tos 0x0, ttl 64, id 46094, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684919625:3684920631, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861786 eth0  Out IP (tos 0x0, ttl 64, id 46095, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684920631:3684921637, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861802 eth0  Out IP (tos 0x0, ttl 64, id 46096, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684921637:3684922643, ack 3710516882, win 57, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861815 eth0  Out IP (tos 0x0, ttl 64, id 46097, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684922643:3684923649, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861826 eth0  Out IP (tos 0x0, ttl 64, id 46098, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684923649:3684924655, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861838 eth0  Out IP (tos 0x0, ttl 64, id 46099, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684924655:3684925661, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861849 eth0  Out IP (tos 0x0, ttl 64, id 46100, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684925661:3684926667, ack 3710516882, win 63, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861869 eth0  Out IP (tos 0x0, ttl 64, id 46101, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684926667:3684927673, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.861879 eth0  Out IP (tos 0x0, ttl 64, id 46102, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684927673:3684928679, ack 3710516882, win 64, options [nop,nop,TS val 2285662309 ecr 337198915], length 1006
1793456382.862750 eth0  In  IP (tos 0x0, ttl 63, id 12096, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710516882:3710518110, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 1228
1793456382.862758 eth0  In  IP (tos 0x0, ttl 63, id 12097, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710518110:3710521794, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 3684
1793456382.862762 eth0  Out IP (tos 0x0, ttl 64, id 46103, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0x93f3), ack 3710521794, win 65, options [nop,nop,TS val 2285662310 ecr 337198917], length 0
1793456382.862812 eth0  In  IP (tos 0x0, ttl 63, id 12100, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710521794:3710526706, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 4912
1793456382.862814 eth0  In  IP (tos 0x0, ttl 63, id 12104, offset 0, flags [DF], proto TCP (6), length 7420)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710526706:3710534074, ack 3684918619, win 63, options [nop,nop,TS val 337198917 ecr 2285662309], length 7368
1793456382.862815 eth0  In  IP (tos 0x0, ttl 63, id 12110, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x77a8 (correct), ack 3684923649, win 59, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862820 eth0  Out IP (tos 0x0, ttl 64, id 46104, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684928679:3684935721, ack 3710534074, win 57, options [nop,nop,TS val 2285662310 ecr 337198917], length 7042
1793456382.862821 eth0  In  IP (tos 0x0, ttl 63, id 12111, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x6406 (correct), ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 0
1793456382.862822 eth0  In  IP (tos 0x0, ttl 63, id 12112, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710534074:3710538986, ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 4912
1793456382.862835 eth0  Out IP (tos 0x0, ttl 64, id 46110, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684935721:3684936727, ack 3710538986, win 53, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862837 eth0  In  IP (tos 0x0, ttl 63, id 12116, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710538986:3710541442, ack 3684928679, win 55, options [nop,nop,TS val 337198917 ecr 2285662309], length 2456
1793456382.862948 eth0  Out IP (tos 0x0, ttl 64, id 46111, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684936727:3684937733, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862977 eth0  Out IP (tos 0x0, ttl 64, id 46112, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684937733:3684938739, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.862989 eth0  Out IP (tos 0x0, ttl 64, id 46113, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684938739:3684939745, ack 3710541442, win 51, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863002 eth0  Out IP (tos 0x0, ttl 64, id 46114, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684939745:3684940751, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863012 eth0  Out IP (tos 0x0, ttl 64, id 46115, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684940751:3684941757, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863023 eth0  Out IP (tos 0x0, ttl 64, id 46116, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684941757:3684942763, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863047 eth0  Out IP (tos 0x0, ttl 64, id 46117, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684942763:3684943769, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863063 eth0  Out IP (tos 0x0, ttl 64, id 46118, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684943769:3684944775, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863073 eth0  Out IP (tos 0x0, ttl 64, id 46119, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684944775:3684945781, ack 3710541442, win 54, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863086 eth0  Out IP (tos 0x0, ttl 64, id 46120, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684945781:3684946787, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863102 eth0  Out IP (tos 0x0, ttl 64, id 46121, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684946787:3684947793, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863196 eth0  Out IP (tos 0x0, ttl 64, id 46122, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684947793:3684948799, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.863209 eth0  Out IP (tos 0x0, ttl 64, id 46123, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684948799:3684949805, ack 3710541442, win 61, options [nop,nop,TS val 2285662310 ecr 337198917], length 1006
1793456382.864125 eth0  In  IP (tos 0x0, ttl 63, id 12118, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710541442:3710543898, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 2456
1793456382.864129 eth0  Out IP (tos 0x0, ttl 64, id 46124, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xeb11), ack 3710543898, win 66, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864142 eth0  In  IP (tos 0x0, ttl 63, id 12120, offset 0, flags [DF], proto TCP (6), length 6192)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710543898:3710550038, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 6140
1793456382.864145 eth0  Out IP (tos 0x0, ttl 64, id 46125, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0xd308), ack 3710550038, win 79, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864157 eth0  In  IP (tos 0x0, ttl 63, id 12125, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710550038:3710551266, ack 3684928679, win 55, options [nop,nop,TS val 337198918 ecr 2285662310], length 1228
1793456382.864159 eth0  In  IP (tos 0x0, ttl 63, id 12126, offset 0, flags [DF], proto TCP (6), length 7420)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], seq 3710551266:3710558634, ack 3684935721, win 59, options [nop,nop,TS val 337198918 ecr 2285662310], length 7368
1793456382.864163 eth0  Out IP (tos 0x0, ttl 64, id 46126, offset 0, flags [DF], proto TCP (6), length 10112)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684949805:3684959865, ack 3710558634, win 97, options [nop,nop,TS val 2285662311 ecr 337198918], length 10060
1793456382.864188 eth0  In  IP (tos 0x0, ttl 63, id 12132, offset 0, flags [DF], proto TCP (6), length 10084)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [P.], seq 3710558634:3710568666, ack 3684935721, win 59, options [nop,nop,TS val 337198918 ecr 2285662310], length 10032
1793456382.864190 eth0  Out IP (tos 0x0, ttl 64, id 46135, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [.], cksum 0x8645 (incorrect -> 0x62e0), ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 0
1793456382.864191 eth0  In  IP (tos 0x0, ttl 63, id 12141, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x8a66 (correct), ack 3684949805, win 46, options [nop,nop,TS val 337198918 ecr 2285662310], length 0
1793456382.864195 eth0  Out IP (tos 0x0, ttl 64, id 46136, offset 0, flags [DF], proto TCP (6), length 7094)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684959865:3684966907, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 7042
1793456382.864204 eth0  Out IP (tos 0x0, ttl 64, id 46142, offset 0, flags [DF], proto TCP (6), length 2064)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684966907:3684968919, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 2012
1793456382.864218 eth0  Out IP (tos 0x0, ttl 64, id 46144, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684968919:3684969925, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864232 eth0  Out IP (tos 0x0, ttl 64, id 46145, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684969925:3684970931, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864242 eth0  Out IP (tos 0x0, ttl 64, id 46146, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684970931:3684971937, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864254 eth0  Out IP (tos 0x0, ttl 64, id 46147, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684971937:3684972943, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864264 eth0  Out IP (tos 0x0, ttl 64, id 46148, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684972943:3684973949, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864277 eth0  Out IP (tos 0x0, ttl 64, id 46149, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684973949:3684974955, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864287 eth0  Out IP (tos 0x0, ttl 64, id 46150, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684974955:3684975961, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864299 eth0  Out IP (tos 0x0, ttl 64, id 46151, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684975961:3684976967, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864314 eth0  Out IP (tos 0x0, ttl 64, id 46152, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684976967:3684977973, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864325 eth0  Out IP (tos 0x0, ttl 64, id 46153, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684977973:3684978979, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864335 eth0  Out IP (tos 0x0, ttl 64, id 46154, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684978979:3684979985, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864346 eth0  Out IP (tos 0x0, ttl 64, id 46155, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684979985:3684980991, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864357 eth0  Out IP (tos 0x0, ttl 64, id 46156, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684980991:3684981997, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.864370 eth0  Out IP (tos 0x0, ttl 64, id 46157, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [P.], seq 3684981997:3684983003, ack 3710568666, win 103, options [nop,nop,TS val 2285662311 ecr 337198918], length 1006
1793456382.865055 eth0  In  IP (tos 0x0, ttl 63, id 12142, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x630e (correct), ack 3684959865, win 56, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865059 eth0  In  IP (tos 0x0, ttl 63, id 12143, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x3fb8 (correct), ack 3684968919, win 48, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865232 eth0  In  IP (tos 0x0, ttl 63, id 12144, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x08bf (correct), ack 3684983003, win 37, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865247 eth0  In  IP (tos 0x0, ttl 63, id 12145, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [F.], cksum 0x08a8 (correct), seq 3710568666, ack 3684983003, win 59, options [nop,nop,TS val 337198919 ecr 2285662311], length 0
1793456382.865361 eth0  Out IP (tos 0x0, ttl 64, id 46158, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37962: Flags [F.], cksum 0x8645 (incorrect -> 0x087a), seq 3684983003, ack 3710568667, win 103, options [nop,nop,TS val 2285662312 ecr 337198919], length 0
1793456382.866023 eth0  In  IP (tos 0x0, ttl 63, id 12146, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37962 > 192.0.3.10.9446: Flags [.], cksum 0x08a5 (correct), ack 3684983004, win 59, options [nop,nop,TS val 337198920 ecr 2285662312], length 0

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

- matrix6_capture_2_receipt: {"bytes":"9999","sha256":"3be18ebbca0d40125ee076473da4ce16da3c54761de5e2ba5cf0a4012e86fdbd"}
- matrix6_origin_endpoint_samples: {"at":"2026-10-31T14:19:35.728093876Z","local_port":9446,"remote_port":37946,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":97,"sacked":0,"snd_cwnd":33,"snd_mss":1228,"snd_wnd":68608,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:19:42.865265238Z","local_port":9446,"remote_port":37962,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":24972,"retrans":0,"rto_us":202000,"rtt_us":1396,"sacked":0,"snd_cwnd":71,"snd_mss":1228,"snd_wnd":60416,"state":8,"total_retrans":0,"unacked":0}}

- matrixDual_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrixDual_transfer: {"after":{"upload_bytes":"201369","download_bytes":"204651","Present":true},"before":{"upload_bytes":"133925","download_bytes":"136408","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:19:49.949874954Z","local_port":35902,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":108,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- matrixDual_ipv6_block: {"native_member":{"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":0,"error":"connect_unavailable"},"positive_control":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0},"scope":"D-123: IPv6 connection must fail; IPv4 bulk must complete. Native refusal may precede tunnel ingress; this does not claim ctv6 was hit. Kernel ctv6 proof is the separate netlab row."}
- matrixDual_capture_0_window: {"launch_requested_at":"2026-10-31T14:19:48.666088082Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:50.210619372Z","stop_requested_at":"2026-10-31T14:19:50.13398692Z"}
- matrixDual_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
292 packets captured
296 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456389.948550 fvnb0 Out IP (tos 0x0, ttl 64, id 48467, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [S], cksum 0xa804 (incorrect -> 0x5a86), seq 4179280516, win 64480, options [mss 1240,sackOK,TS val 1802728036 ecr 0,nop,wscale 10], length 0
1793456389.948747 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [S.], cksum 0x851d (correct), seq 2827345498, ack 4179280517, win 65160, options [mss 1460,sackOK,TS val 622405082 ecr 1802728036,nop,wscale 10], length 0
1793456389.948764 fvnb0 Out IP (tos 0x0, ttl 64, id 48468, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xb235), ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.948830 fvnb0 Out IP (tos 0x0, ttl 64, id 48469, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], cksum 0xa828 (incorrect -> 0x60b9), seq 4179280517:4179280561, ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 44
1793456389.949027 fvnb0 In  IP (tos 0x0, ttl 63, id 15808, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xb208 (correct), ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949039 fvnb0 In  IP (tos 0x0, ttl 63, id 15809, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], cksum 0x58b2 (correct), seq 2827345499:2827345502, ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 3
1793456389.949044 fvnb0 Out IP (tos 0x0, ttl 64, id 48470, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xb206), ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949129 fvnb0 Out IP (tos 0x0, ttl 64, id 48471, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179280561:4179286701, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 6140
1793456389.949140 fvnb0 Out IP (tos 0x0, ttl 64, id 48476, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179286701:4179292841, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 6140
1793456389.949213 fvnb0 In  IP (tos 0x0, ttl 63, id 15810, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0x9a09 (correct), ack 4179286701, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949216 fvnb0 Out IP (tos 0x0, ttl 64, id 48481, offset 0, flags [DF], proto TCP (6), length 8648)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179292841:4179301437, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 8596
1793456389.949223 fvnb0 Out IP (tos 0x0, ttl 64, id 48488, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179301437:4179305121, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949237 fvnb0 In  IP (tos 0x0, ttl 63, id 15811, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827345502:2827346508, ack 4179290385, win 61, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949249 fvnb0 Out IP (tos 0x0, ttl 64, id 48491, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179305121:4179312489, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 7368
1793456389.949261 fvnb0 In  IP (tos 0x0, ttl 63, id 15812, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0x7e24 (correct), ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949262 fvnb0 In  IP (tos 0x0, ttl 63, id 15813, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827346508:2827347514, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949265 fvnb0 Out IP (tos 0x0, ttl 64, id 48497, offset 0, flags [DF], proto TCP (6), length 4964)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179312489:4179317401, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 4912
1793456389.949270 fvnb0 In  IP (tos 0x0, ttl 63, id 15814, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827347514:2827348520, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949277 fvnb0 In  IP (tos 0x0, ttl 63, id 15815, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827348520:2827349526, ack 4179296525, win 56, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949278 fvnb0 Out IP (tos 0x0, ttl 64, id 48501, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x1655), ack 2827348520, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949282 fvnb0 Out IP (tos 0x0, ttl 64, id 48502, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179317401:4179324769, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 7368
1793456389.949289 fvnb0 In  IP (tos 0x0, ttl 63, id 15816, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827349526:2827350532, ack 4179301437, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949292 fvnb0 Out IP (tos 0x0, ttl 64, id 48508, offset 0, flags [DF], proto TCP (6), length 9876)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179324769:4179334593, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 9824
1793456389.949317 fvnb0 In  IP (tos 0x0, ttl 63, id 15817, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0x4347 (correct), ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949328 fvnb0 In  IP (tos 0x0, ttl 63, id 15818, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827350532:2827351538, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949328 fvnb0 In  IP (tos 0x0, ttl 63, id 15819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827351538:2827352544, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949328 fvnb0 In  IP (tos 0x0, ttl 63, id 15820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0x2d0a (correct), ack 4179307577, win 49, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949335 fvnb0 Out IP (tos 0x0, ttl 64, id 48516, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xc375), ack 2827352544, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949337 fvnb0 In  IP (tos 0x0, ttl 63, id 15821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827352544:2827353550, ack 4179312489, win 45, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949340 fvnb0 Out IP (tos 0x0, ttl 64, id 48517, offset 0, flags [DF], proto TCP (6), length 10084)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179334593:4179344625, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 10032
1793456389.949355 fvnb0 In  IP (tos 0x0, ttl 63, id 15822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827353550:2827354556, ack 4179316173, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949367 fvnb0 In  IP (tos 0x0, ttl 63, id 15823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827354556:2827355562, ack 4179317401, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949372 fvnb0 Out IP (tos 0x0, ttl 64, id 48526, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x907c), ack 2827355562, win 60, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949372 fvnb0 In  IP (tos 0x0, ttl 63, id 15824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827355562:2827356568, ack 4179322313, win 38, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949394 fvnb0 In  IP (tos 0x0, ttl 63, id 15825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827356568:2827357574, ack 4179322313, win 41, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949398 fvnb0 In  IP (tos 0x0, ttl 63, id 15826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827357574:2827358580, ack 4179330909, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949398 fvnb0 Out IP (tos 0x0, ttl 64, id 48527, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x88a1), ack 2827357574, win 59, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949402 fvnb0 In  IP (tos 0x0, ttl 63, id 15827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xac00 (correct), ack 4179334593, win 30, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949408 fvnb0 In  IP (tos 0x0, ttl 63, id 15828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827358580:2827359586, ack 4179337049, win 28, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949413 fvnb0 Out IP (tos 0x0, ttl 64, id 48528, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x80c6), ack 2827359586, win 58, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949420 fvnb0 In  IP (tos 0x0, ttl 63, id 15829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827359586:2827360592, ack 4179340733, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949427 fvnb0 In  IP (tos 0x0, ttl 63, id 15830, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0x8297 (correct), ack 4179343189, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949440 fvnb0 In  IP (tos 0x0, ttl 63, id 15831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827360592:2827361598, ack 4179344417, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949441 fvnb0 In  IP (tos 0x0, ttl 63, id 15832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827361598:2827362604, ack 4179344625, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949446 fvnb0 Out IP (tos 0x0, ttl 64, id 48529, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x74fe), ack 2827362604, win 56, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949453 fvnb0 In  IP (tos 0x0, ttl 63, id 15833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827362604:2827363610, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949462 fvnb0 In  IP (tos 0x0, ttl 63, id 15834, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827363610:2827364616, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949466 fvnb0 Out IP (tos 0x0, ttl 64, id 48530, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x6d23), ack 2827364616, win 55, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949471 fvnb0 In  IP (tos 0x0, ttl 63, id 15835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827364616:2827365622, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949481 fvnb0 In  IP (tos 0x0, ttl 63, id 15836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827365622:2827366628, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949485 fvnb0 Out IP (tos 0x0, ttl 64, id 48531, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x6548), ack 2827366628, win 54, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949493 fvnb0 In  IP (tos 0x0, ttl 63, id 15837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827366628:2827367634, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949499 fvnb0 In  IP (tos 0x0, ttl 63, id 15838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827367634:2827368640, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949503 fvnb0 Out IP (tos 0x0, ttl 64, id 48532, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x5d6d), ack 2827368640, win 53, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949508 fvnb0 In  IP (tos 0x0, ttl 63, id 15839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827368640:2827369646, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949515 fvnb0 In  IP (tos 0x0, ttl 63, id 15840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827369646:2827370652, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949519 fvnb0 Out IP (tos 0x0, ttl 64, id 48533, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x5592), ack 2827370652, win 52, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949525 fvnb0 In  IP (tos 0x0, ttl 63, id 15841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827370652:2827371658, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949534 fvnb0 In  IP (tos 0x0, ttl 63, id 15842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827371658:2827372664, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949538 fvnb0 Out IP (tos 0x0, ttl 64, id 48534, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x4db7), ack 2827372664, win 51, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949543 fvnb0 In  IP (tos 0x0, ttl 63, id 15843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827372664:2827373670, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949554 fvnb0 In  IP (tos 0x0, ttl 63, id 15844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827373670:2827374676, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949558 fvnb0 Out IP (tos 0x0, ttl 64, id 48535, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x45dc), ack 2827374676, win 50, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949560 fvnb0 In  IP (tos 0x0, ttl 63, id 15845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827374676:2827375682, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949569 fvnb0 In  IP (tos 0x0, ttl 63, id 15846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827375682:2827376688, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949573 fvnb0 Out IP (tos 0x0, ttl 64, id 48536, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x3e01), ack 2827376688, win 49, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949579 fvnb0 In  IP (tos 0x0, ttl 63, id 15847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827376688:2827377694, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949588 fvnb0 In  IP (tos 0x0, ttl 63, id 15848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827377694:2827378700, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949592 fvnb0 Out IP (tos 0x0, ttl 64, id 48537, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x3626), ack 2827378700, win 48, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949605 fvnb0 In  IP (tos 0x0, ttl 63, id 15849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827378700:2827379706, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949609 fvnb0 In  IP (tos 0x0, ttl 63, id 15850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827379706:2827380712, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949612 fvnb0 Out IP (tos 0x0, ttl 64, id 48538, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x2e4b), ack 2827380712, win 47, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949613 fvnb0 In  IP (tos 0x0, ttl 63, id 15851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827380712:2827381718, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949624 fvnb0 In  IP (tos 0x0, ttl 63, id 15852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827381718:2827382724, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949628 fvnb0 Out IP (tos 0x0, ttl 64, id 48539, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x2670), ack 2827382724, win 46, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949638 fvnb0 In  IP (tos 0x0, ttl 63, id 15853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827382724:2827383730, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949649 fvnb0 In  IP (tos 0x0, ttl 63, id 15854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827383730:2827384736, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949649 fvnb0 In  IP (tos 0x0, ttl 63, id 15855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827384736:2827385742, ack 4179344625, win 42, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949657 fvnb0 Out IP (tos 0x0, ttl 64, id 48540, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x1aa7), ack 2827385742, win 44, options [nop,nop,TS val 1802728038 ecr 622405082], length 0
1793456389.949661 fvnb0 In  IP (tos 0x0, ttl 63, id 15856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827385742:2827386748, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949669 fvnb0 In  IP (tos 0x0, ttl 63, id 15857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827386748:2827387754, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949674 fvnb0 Out IP (tos 0x0, ttl 64, id 48541, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x12cb), ack 2827387754, win 43, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949679 fvnb0 In  IP (tos 0x0, ttl 63, id 15858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827387754:2827388760, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949691 fvnb0 In  IP (tos 0x0, ttl 63, id 15859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827388760:2827389766, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949699 fvnb0 Out IP (tos 0x0, ttl 64, id 48542, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x0af0), ack 2827389766, win 42, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949704 fvnb0 In  IP (tos 0x0, ttl 63, id 15860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827389766:2827390772, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949710 fvnb0 In  IP (tos 0x0, ttl 63, id 15861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827390772:2827391778, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949715 fvnb0 In  IP (tos 0x0, ttl 63, id 15862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827391778:2827392784, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949717 fvnb0 Out IP (tos 0x0, ttl 64, id 48543, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x0315), ack 2827391778, win 41, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949722 fvnb0 In  IP (tos 0x0, ttl 63, id 15863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827392784:2827393790, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949725 fvnb0 Out IP (tos 0x0, ttl 64, id 48544, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xfb39), ack 2827393790, win 40, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949731 fvnb0 In  IP (tos 0x0, ttl 63, id 15864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827393790:2827394796, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949739 fvnb0 In  IP (tos 0x0, ttl 63, id 15865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827394796:2827395802, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949743 fvnb0 Out IP (tos 0x0, ttl 64, id 48545, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xf35e), ack 2827395802, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949757 fvnb0 In  IP (tos 0x0, ttl 63, id 15866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827395802:2827396808, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949759 fvnb0 In  IP (tos 0x0, ttl 63, id 15867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827396808:2827397814, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949762 fvnb0 Out IP (tos 0x0, ttl 64, id 48546, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xeb82), ack 2827397814, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949768 fvnb0 In  IP (tos 0x0, ttl 63, id 15868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827397814:2827398820, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949776 fvnb0 In  IP (tos 0x0, ttl 63, id 15869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827398820:2827399826, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949780 fvnb0 Out IP (tos 0x0, ttl 64, id 48547, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xe3a6), ack 2827399826, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949784 fvnb0 In  IP (tos 0x0, ttl 63, id 15870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827399826:2827400832, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949797 fvnb0 In  IP (tos 0x0, ttl 63, id 15871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827400832:2827401838, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949801 fvnb0 In  IP (tos 0x0, ttl 63, id 15872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827401838:2827402844, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949801 fvnb0 Out IP (tos 0x0, ttl 64, id 48548, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xdbca), ack 2827401838, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949819 fvnb0 In  IP (tos 0x0, ttl 63, id 15873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827402844:2827403850, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949822 fvnb0 In  IP (tos 0x0, ttl 63, id 15874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827403850:2827404856, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949823 fvnb0 Out IP (tos 0x0, ttl 64, id 48549, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xd3ee), ack 2827403850, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949826 fvnb0 In  IP (tos 0x0, ttl 63, id 15875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827404856:2827405862, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949829 fvnb0 Out IP (tos 0x0, ttl 64, id 48550, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xcc12), ack 2827405862, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949838 fvnb0 In  IP (tos 0x0, ttl 63, id 15876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827405862:2827406868, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949848 fvnb0 In  IP (tos 0x0, ttl 63, id 15877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827406868:2827407874, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949857 fvnb0 In  IP (tos 0x0, ttl 63, id 15878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827407874:2827408880, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949860 fvnb0 Out IP (tos 0x0, ttl 64, id 48551, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xc437), ack 2827407874, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949867 fvnb0 In  IP (tos 0x0, ttl 63, id 15879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827408880:2827409886, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949869 fvnb0 Out IP (tos 0x0, ttl 64, id 48552, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xbc5b), ack 2827409886, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950070 fvnb0 Out IP (tos 0x0, ttl 64, id 48553, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [F.], cksum 0xa7fc (incorrect -> 0xbc59), seq 4179344625, ack 2827409886, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950205 fvnb0 In  IP (tos 0x0, ttl 63, id 15880, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [F.], cksum 0xbc3c (correct), seq 2827409886, ack 4179344626, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 0
1793456389.950208 fvnb0 Out IP (tos 0x0, ttl 64, id 48554, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xbc58), ack 2827409887, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4320,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:6d:bb:cd:a6:42","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4323,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"62:1a:d1:c5:55:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":7,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.83.128.120","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:45832 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.061/0.084 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5113 segs_out:16 segs_in:16 data_segs_out:9 data_segs_in:11 send 1899016393bps lastsnd:45675 lastrcv:45674 lastack:15504 pacing_rate 3744323232bps delivery_rate 3861333328bps delivered:10 app_limited rcv_space:14480 rcv_ssthresh:81050 minrtt:0.006 snd_wnd:71680
ESTAB 0      0       172.28.30.41:54044 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.205/0.225 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3551 bytes_acked:3552 bytes_received:5748 segs_out:39 segs_in:34 data_segs_out:20 data_segs_in:30 send 565073171bps lastsnd:514 lastrcv:514 lastack:514 pacing_rate 1128082768bps delivery_rate 1287111104bps delivered:21 app_limited busy:3ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.01 snd_wnd:79872
ESTAB 0      0       172.28.30.41:54054 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.341/2.587 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3779 bytes_acked:3780 bytes_received:6218 segs_out:29 segs_in:35 data_segs_out:17 data_segs_in:30 send 86383296bps lastsnd:5420 lastrcv:5420 lastack:5420 pacing_rate 172670016bps delivery_rate 2574222216bps delivered:18 app_limited busy:31ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.41:54062 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.251/10.347 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2926 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 22060560bps lastsnd:5420 lastrcv:5420 lastack:5379 pacing_rate 44119016bps delivery_rate 2574222216bps delivered:9 app_limited busy:44ms rcv_space:14480 rcv_ssthresh:79824 minrtt:0.009 snd_wnd:74752

- matrixDual_capture_0_receipt: {"bytes":"39420","sha256":"66208e521ce982826266e503559db65cc0cc05c12a1df4fc785a4ff43040e738"}
- matrixDual_capture_1_window: {"launch_requested_at":"2026-10-31T14:19:48.703998643Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:50.562880524Z","stop_requested_at":"2026-10-31T14:19:50.396137865Z"}
- matrixDual_ipv6_uplink_no_leak: true
- matrixDual_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
444 packets captured
444 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456389.948616 fvmb0 In  IP (tos 0x0, ttl 64, id 48467, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [S], cksum 0x5a86 (correct), seq 4179280516, win 64480, options [mss 1240,sackOK,TS val 1802728036 ecr 0,nop,wscale 10], length 0
1793456389.948648 fvup0 Out IP (tos 0x0, ttl 63, id 48467, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [S], cksum 0x7c3d (correct), seq 4179280516, win 64480, options [mss 1240,sackOK,TS val 1802728036 ecr 0,nop,wscale 10], length 0
1793456389.948678 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [S.], cksum 0x864d (incorrect -> 0xa6d4), seq 2827345498, ack 4179280517, win 65160, options [mss 1460,sackOK,TS val 622405082 ecr 1802728036,nop,wscale 10], length 0
1793456389.948684 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [S.], cksum 0xa804 (incorrect -> 0x851d), seq 2827345498, ack 4179280517, win 65160, options [mss 1460,sackOK,TS val 622405082 ecr 1802728036,nop,wscale 10], length 0
1793456389.948794 fvmb0 In  IP (tos 0x0, ttl 64, id 48468, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xb235 (correct), ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.948799 fvup0 Out IP (tos 0x0, ttl 63, id 48468, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xd3ec (correct), ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.948983 fvmb0 In  IP (tos 0x0, ttl 64, id 48469, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], cksum 0x60b9 (correct), seq 4179280517:4179280561, ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 44
1793456389.948987 fvup0 Out IP (tos 0x0, ttl 63, id 48469, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], cksum 0x8270 (correct), seq 4179280517:4179280561, ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 44
1793456389.948995 fvup0 In  IP (tos 0x0, ttl 64, id 15808, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xd3bf), ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.948998 fvmb0 Out IP (tos 0x0, ttl 63, id 15808, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xa7fc (incorrect -> 0xb208), ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949009 fvup0 In  IP (tos 0x0, ttl 64, id 15809, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], cksum 0x8648 (incorrect -> 0x7a69), seq 2827345499:2827345502, ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 3
1793456389.949015 fvmb0 Out IP (tos 0x0, ttl 63, id 15809, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], cksum 0xa7ff (incorrect -> 0x58b2), seq 2827345499:2827345502, ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 3
1793456389.949091 fvmb0 In  IP (tos 0x0, ttl 64, id 48470, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xb206 (correct), ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949094 fvup0 Out IP (tos 0x0, ttl 63, id 48470, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xd3bd (correct), ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949174 fvmb0 In  IP (tos 0x0, ttl 64, id 48471, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179280561:4179281789, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949178 fvup0 Out IP (tos 0x0, ttl 63, id 48471, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179280561:4179281789, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949183 fvmb0 In  IP (tos 0x0, ttl 64, id 48472, offset 0, flags [DF], proto TCP (6), length 4964)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179281789:4179286701, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 4912
1793456389.949186 fvup0 Out IP (tos 0x0, ttl 63, id 48472, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179281789:4179286701, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 4912
1793456389.949190 fvup0 In  IP (tos 0x0, ttl 64, id 15810, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xbbc0), ack 4179286701, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949192 fvmb0 Out IP (tos 0x0, ttl 63, id 15810, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xa7fc (incorrect -> 0x9a09), ack 4179286701, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949195 fvmb0 In  IP (tos 0x0, ttl 64, id 48476, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179286701:4179290385, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949197 fvup0 Out IP (tos 0x0, ttl 63, id 48476, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179286701:4179290385, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949200 fvmb0 In  IP (tos 0x0, ttl 64, id 48479, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179290385:4179292841, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949203 fvup0 Out IP (tos 0x0, ttl 63, id 48479, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179290385:4179292841, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949209 fvup0 In  IP (tos 0x0, ttl 64, id 15811, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827345502:2827346508, ack 4179290385, win 61, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949218 fvmb0 Out IP (tos 0x0, ttl 63, id 15811, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827345502:2827346508, ack 4179290385, win 61, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949227 fvup0 In  IP (tos 0x0, ttl 64, id 15812, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0x9fdb), ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949230 fvmb0 Out IP (tos 0x0, ttl 63, id 15812, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xa7fc (incorrect -> 0x7e24), ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949237 fvup0 In  IP (tos 0x0, ttl 64, id 15813, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827346508:2827347514, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949239 fvmb0 Out IP (tos 0x0, ttl 63, id 15813, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827346508:2827347514, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949242 fvmb0 In  IP (tos 0x0, ttl 64, id 48481, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179292841:4179294069, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949246 fvup0 Out IP (tos 0x0, ttl 63, id 48481, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179292841:4179294069, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949248 fvup0 In  IP (tos 0x0, ttl 64, id 15814, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827347514:2827348520, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949249 fvmb0 In  IP (tos 0x0, ttl 64, id 48482, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179294069:4179296525, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949250 fvmb0 Out IP (tos 0x0, ttl 63, id 15814, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827347514:2827348520, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949252 fvup0 Out IP (tos 0x0, ttl 63, id 48482, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179294069:4179296525, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949257 fvmb0 In  IP (tos 0x0, ttl 64, id 48484, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179296525:4179298981, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949260 fvup0 Out IP (tos 0x0, ttl 63, id 48484, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179296525:4179298981, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949260 fvup0 In  IP (tos 0x0, ttl 64, id 15815, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827348520:2827349526, ack 4179296525, win 56, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949261 fvmb0 Out IP (tos 0x0, ttl 63, id 15815, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827348520:2827349526, ack 4179296525, win 56, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949263 fvmb0 In  IP (tos 0x0, ttl 64, id 48486, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179298981:4179301437, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949265 fvup0 Out IP (tos 0x0, ttl 63, id 48486, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179298981:4179301437, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949269 fvmb0 In  IP (tos 0x0, ttl 64, id 48488, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179301437:4179302665, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949271 fvup0 In  IP (tos 0x0, ttl 64, id 15816, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827349526:2827350532, ack 4179301437, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949272 fvup0 Out IP (tos 0x0, ttl 63, id 48488, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179301437:4179302665, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949273 fvmb0 Out IP (tos 0x0, ttl 63, id 15816, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827349526:2827350532, ack 4179301437, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949274 fvmb0 In  IP (tos 0x0, ttl 64, id 48489, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179302665:4179303893, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949278 fvup0 Out IP (tos 0x0, ttl 63, id 48489, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179302665:4179303893, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949283 fvup0 In  IP (tos 0x0, ttl 64, id 15817, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0x64fe), ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949285 fvup0 In  IP (tos 0x0, ttl 64, id 15818, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827350532:2827351538, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949285 fvmb0 Out IP (tos 0x0, ttl 63, id 15817, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xa7fc (incorrect -> 0x4347), ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949288 fvmb0 Out IP (tos 0x0, ttl 63, id 15818, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827350532:2827351538, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949288 fvmb0 In  IP (tos 0x0, ttl 64, id 48490, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179303893:4179305121, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949289 fvmb0 In  IP (tos 0x0, ttl 64, id 48491, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179305121:4179307577, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949291 fvup0 Out IP (tos 0x0, ttl 63, id 48490, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179303893:4179305121, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949295 fvup0 In  IP (tos 0x0, ttl 64, id 15819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827351538:2827352544, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949296 fvup0 Out IP (tos 0x0, ttl 63, id 48491, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179305121:4179307577, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949298 fvmb0 Out IP (tos 0x0, ttl 63, id 15819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827351538:2827352544, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949300 fvmb0 In  IP (tos 0x0, ttl 64, id 48493, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179307577:4179310033, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949302 fvup0 Out IP (tos 0x0, ttl 63, id 48493, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179307577:4179310033, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949305 fvup0 In  IP (tos 0x0, ttl 64, id 15820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0x4ec1), ack 4179307577, win 49, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949305 fvmb0 In  IP (tos 0x0, ttl 64, id 48495, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179310033:4179312489, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949306 fvmb0 Out IP (tos 0x0, ttl 63, id 15820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xa7fc (incorrect -> 0x2d0a), ack 4179307577, win 49, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949308 fvup0 Out IP (tos 0x0, ttl 63, id 48495, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179310033:4179312489, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949311 fvmb0 In  IP (tos 0x0, ttl 64, id 48497, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179312489:4179314945, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949312 fvup0 In  IP (tos 0x0, ttl 64, id 15821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827352544:2827353550, ack 4179312489, win 45, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949314 fvup0 Out IP (tos 0x0, ttl 63, id 48497, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179312489:4179314945, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949315 fvmb0 Out IP (tos 0x0, ttl 63, id 15821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827352544:2827353550, ack 4179312489, win 45, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949317 fvmb0 In  IP (tos 0x0, ttl 64, id 48499, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179314945:4179316173, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949319 fvup0 Out IP (tos 0x0, ttl 63, id 48499, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179314945:4179316173, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949323 fvmb0 In  IP (tos 0x0, ttl 64, id 48500, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179316173:4179317401, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949324 fvmb0 In  IP (tos 0x0, ttl 64, id 48501, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x1655 (correct), ack 2827348520, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949325 fvup0 In  IP (tos 0x0, ttl 64, id 15822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827353550:2827354556, ack 4179316173, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949327 fvup0 Out IP (tos 0x0, ttl 63, id 48500, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179316173:4179317401, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949327 fvmb0 Out IP (tos 0x0, ttl 63, id 15822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827353550:2827354556, ack 4179316173, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949329 fvup0 Out IP (tos 0x0, ttl 63, id 48501, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x380c (correct), ack 2827348520, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949334 fvmb0 In  IP (tos 0x0, ttl 64, id 48502, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179317401:4179321085, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949336 fvup0 Out IP (tos 0x0, ttl 63, id 48502, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179317401:4179321085, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949339 fvmb0 In  IP (tos 0x0, ttl 64, id 48505, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179321085:4179322313, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949340 fvup0 In  IP (tos 0x0, ttl 64, id 15823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827354556:2827355562, ack 4179317401, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949342 fvup0 Out IP (tos 0x0, ttl 63, id 48505, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179321085:4179322313, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949342 fvmb0 Out IP (tos 0x0, ttl 63, id 15823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827354556:2827355562, ack 4179317401, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949352 fvup0 In  IP (tos 0x0, ttl 64, id 15824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827355562:2827356568, ack 4179322313, win 38, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949354 fvmb0 In  IP (tos 0x0, ttl 64, id 48506, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179322313:4179324769, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949354 fvmb0 Out IP (tos 0x0, ttl 63, id 15824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827355562:2827356568, ack 4179322313, win 38, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949356 fvmb0 In  IP (tos 0x0, ttl 64, id 48508, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179324769:4179328453, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949359 fvup0 Out IP (tos 0x0, ttl 63, id 48506, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179322313:4179324769, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949361 fvup0 Out IP (tos 0x0, ttl 63, id 48508, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179324769:4179328453, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949364 fvup0 In  IP (tos 0x0, ttl 64, id 15825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827356568:2827357574, ack 4179322313, win 41, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949365 fvmb0 In  IP (tos 0x0, ttl 64, id 48511, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179328453:4179330909, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949366 fvmb0 Out IP (tos 0x0, ttl 63, id 15825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827356568:2827357574, ack 4179322313, win 41, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949368 fvup0 Out IP (tos 0x0, ttl 63, id 48511, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179328453:4179330909, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949371 fvmb0 In  IP (tos 0x0, ttl 64, id 48513, offset 0, flags [DF], proto TCP (6), length 3736)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179330909:4179334593, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949372 fvup0 In  IP (tos 0x0, ttl 64, id 15826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827357574:2827358580, ack 4179330909, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949374 fvup0 Out IP (tos 0x0, ttl 63, id 48513, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179330909:4179334593, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949375 fvmb0 Out IP (tos 0x0, ttl 63, id 15826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827357574:2827358580, ack 4179330909, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949378 fvmb0 In  IP (tos 0x0, ttl 64, id 48516, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xc375 (correct), ack 2827352544, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949380 fvup0 Out IP (tos 0x0, ttl 63, id 48516, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xe52c (correct), ack 2827352544, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949382 fvup0 In  IP (tos 0x0, ttl 64, id 15827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xcdb7), ack 4179334593, win 30, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949384 fvmb0 Out IP (tos 0x0, ttl 63, id 15827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xa7fc (incorrect -> 0xac00), ack 4179334593, win 30, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949385 fvmb0 In  IP (tos 0x0, ttl 64, id 48517, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179334593:4179337049, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949387 fvup0 Out IP (tos 0x0, ttl 63, id 48517, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179334593:4179337049, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949391 fvmb0 In  IP (tos 0x0, ttl 64, id 48519, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179337049:4179339505, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949392 fvup0 In  IP (tos 0x0, ttl 64, id 15828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827358580:2827359586, ack 4179337049, win 28, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949394 fvmb0 Out IP (tos 0x0, ttl 63, id 15828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827358580:2827359586, ack 4179337049, win 28, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949394 fvup0 Out IP (tos 0x0, ttl 63, id 48519, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179337049:4179339505, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949397 fvmb0 In  IP (tos 0x0, ttl 64, id 48521, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179339505:4179340733, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949399 fvup0 Out IP (tos 0x0, ttl 63, id 48521, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179339505:4179340733, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949402 fvmb0 In  IP (tos 0x0, ttl 64, id 48522, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179340733:4179343189, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949403 fvup0 In  IP (tos 0x0, ttl 64, id 15829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827359586:2827360592, ack 4179340733, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949405 fvup0 Out IP (tos 0x0, ttl 63, id 48522, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179340733:4179343189, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949405 fvmb0 Out IP (tos 0x0, ttl 63, id 15829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827359586:2827360592, ack 4179340733, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949408 fvmb0 In  IP (tos 0x0, ttl 64, id 48524, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], seq 4179343189:4179344417, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949411 fvup0 In  IP (tos 0x0, ttl 64, id 15830, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xa44e), ack 4179343189, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949411 fvup0 Out IP (tos 0x0, ttl 63, id 48524, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179343189:4179344417, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949412 fvmb0 Out IP (tos 0x0, ttl 63, id 15830, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [.], cksum 0xa7fc (incorrect -> 0x8297), ack 4179343189, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949413 fvmb0 In  IP (tos 0x0, ttl 64, id 48525, offset 0, flags [DF], proto TCP (6), length 260)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [P.], seq 4179344417:4179344625, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 208
1793456389.949413 fvmb0 In  IP (tos 0x0, ttl 64, id 48526, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x907c (correct), ack 2827355562, win 60, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949416 fvup0 Out IP (tos 0x0, ttl 63, id 48525, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179344417:4179344625, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 208
1793456389.949417 fvup0 In  IP (tos 0x0, ttl 64, id 15831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827360592:2827361598, ack 4179344417, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949418 fvup0 Out IP (tos 0x0, ttl 63, id 48526, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xb233 (correct), ack 2827355562, win 60, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949419 fvmb0 Out IP (tos 0x0, ttl 63, id 15831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827360592:2827361598, ack 4179344417, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949424 fvmb0 In  IP (tos 0x0, ttl 64, id 48527, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x88a1 (correct), ack 2827357574, win 59, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949426 fvup0 In  IP (tos 0x0, ttl 64, id 15832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827361598:2827362604, ack 4179344625, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949427 fvup0 Out IP (tos 0x0, ttl 63, id 48527, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xaa58 (correct), ack 2827357574, win 59, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949428 fvmb0 Out IP (tos 0x0, ttl 63, id 15832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827361598:2827362604, ack 4179344625, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949429 fvmb0 In  IP (tos 0x0, ttl 64, id 48528, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x80c6 (correct), ack 2827359586, win 58, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949431 fvup0 Out IP (tos 0x0, ttl 63, id 48528, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa27d (correct), ack 2827359586, win 58, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949437 fvup0 In  IP (tos 0x0, ttl 64, id 15833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827362604:2827363610, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949439 fvmb0 Out IP (tos 0x0, ttl 63, id 15833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827362604:2827363610, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949446 fvup0 In  IP (tos 0x0, ttl 64, id 15834, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827363610:2827364616, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949448 fvmb0 Out IP (tos 0x0, ttl 63, id 15834, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827363610:2827364616, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949456 fvup0 In  IP (tos 0x0, ttl 64, id 15835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827364616:2827365622, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949458 fvmb0 Out IP (tos 0x0, ttl 63, id 15835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827364616:2827365622, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949460 fvmb0 In  IP (tos 0x0, ttl 64, id 48529, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x74fe (correct), ack 2827362604, win 56, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949464 fvup0 Out IP (tos 0x0, ttl 63, id 48529, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x96b5 (correct), ack 2827362604, win 56, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949464 fvup0 In  IP (tos 0x0, ttl 64, id 15836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827365622:2827366628, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949467 fvmb0 Out IP (tos 0x0, ttl 63, id 15836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827365622:2827366628, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949477 fvup0 In  IP (tos 0x0, ttl 64, id 15837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827366628:2827367634, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949478 fvmb0 Out IP (tos 0x0, ttl 63, id 15837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827366628:2827367634, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949481 fvmb0 In  IP (tos 0x0, ttl 64, id 48530, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x6d23 (correct), ack 2827364616, win 55, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949483 fvup0 Out IP (tos 0x0, ttl 63, id 48530, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x8eda (correct), ack 2827364616, win 55, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949484 fvup0 In  IP (tos 0x0, ttl 64, id 15838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827367634:2827368640, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949486 fvmb0 Out IP (tos 0x0, ttl 63, id 15838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827367634:2827368640, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949493 fvup0 In  IP (tos 0x0, ttl 64, id 15839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827368640:2827369646, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949495 fvmb0 Out IP (tos 0x0, ttl 63, id 15839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827368640:2827369646, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949497 fvmb0 In  IP (tos 0x0, ttl 64, id 48531, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x6548 (correct), ack 2827366628, win 54, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949499 fvup0 Out IP (tos 0x0, ttl 63, id 48531, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x86ff (correct), ack 2827366628, win 54, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949500 fvup0 In  IP (tos 0x0, ttl 64, id 15840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827369646:2827370652, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949502 fvmb0 Out IP (tos 0x0, ttl 63, id 15840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827369646:2827370652, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949510 fvup0 In  IP (tos 0x0, ttl 64, id 15841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827370652:2827371658, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949511 fvmb0 Out IP (tos 0x0, ttl 63, id 15841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827370652:2827371658, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949517 fvmb0 In  IP (tos 0x0, ttl 64, id 48532, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x5d6d (correct), ack 2827368640, win 53, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949517 fvup0 In  IP (tos 0x0, ttl 64, id 15842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827371658:2827372664, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949519 fvmb0 Out IP (tos 0x0, ttl 63, id 15842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827371658:2827372664, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949520 fvup0 Out IP (tos 0x0, ttl 63, id 48532, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x7f24 (correct), ack 2827368640, win 53, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949527 fvup0 In  IP (tos 0x0, ttl 64, id 15843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827372664:2827373670, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949529 fvmb0 Out IP (tos 0x0, ttl 63, id 15843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827372664:2827373670, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949534 fvmb0 In  IP (tos 0x0, ttl 64, id 48533, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x5592 (correct), ack 2827370652, win 52, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949535 fvup0 In  IP (tos 0x0, ttl 64, id 15844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827373670:2827374676, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949537 fvmb0 Out IP (tos 0x0, ttl 63, id 15844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827373670:2827374676, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949537 fvup0 Out IP (tos 0x0, ttl 63, id 48533, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x7749 (correct), ack 2827370652, win 52, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949546 fvup0 In  IP (tos 0x0, ttl 64, id 15845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827374676:2827375682, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949548 fvmb0 Out IP (tos 0x0, ttl 63, id 15845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827374676:2827375682, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949554 fvup0 In  IP (tos 0x0, ttl 64, id 15846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827375682:2827376688, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949556 fvmb0 Out IP (tos 0x0, ttl 63, id 15846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827375682:2827376688, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949563 fvup0 In  IP (tos 0x0, ttl 64, id 15847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827376688:2827377694, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949564 fvmb0 Out IP (tos 0x0, ttl 63, id 15847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827376688:2827377694, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949572 fvmb0 In  IP (tos 0x0, ttl 64, id 48534, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x4db7 (correct), ack 2827372664, win 51, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949572 fvup0 In  IP (tos 0x0, ttl 64, id 15848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827377694:2827378700, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949574 fvmb0 Out IP (tos 0x0, ttl 63, id 15848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827377694:2827378700, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949579 fvup0 Out IP (tos 0x0, ttl 63, id 48534, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x6f6e (correct), ack 2827372664, win 51, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949581 fvup0 In  IP (tos 0x0, ttl 64, id 15849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827378700:2827379706, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949583 fvmb0 Out IP (tos 0x0, ttl 63, id 15849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827378700:2827379706, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949589 fvmb0 In  IP (tos 0x0, ttl 64, id 48535, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x45dc (correct), ack 2827374676, win 50, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949589 fvmb0 In  IP (tos 0x0, ttl 64, id 48536, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x3e01 (correct), ack 2827376688, win 49, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949589 fvup0 In  IP (tos 0x0, ttl 64, id 15850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827379706:2827380712, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949591 fvmb0 Out IP (tos 0x0, ttl 63, id 15850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827379706:2827380712, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949592 fvup0 Out IP (tos 0x0, ttl 63, id 48535, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x6793 (correct), ack 2827374676, win 50, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949595 fvup0 Out IP (tos 0x0, ttl 63, id 48536, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x5fb8 (correct), ack 2827376688, win 49, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949597 fvup0 In  IP (tos 0x0, ttl 64, id 15851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827380712:2827381718, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949600 fvmb0 Out IP (tos 0x0, ttl 63, id 15851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827380712:2827381718, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949605 fvmb0 In  IP (tos 0x0, ttl 64, id 48537, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x3626 (correct), ack 2827378700, win 48, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949608 fvup0 Out IP (tos 0x0, ttl 63, id 48537, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x57dd (correct), ack 2827378700, win 48, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949609 fvup0 In  IP (tos 0x0, ttl 64, id 15852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827381718:2827382724, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949610 fvmb0 Out IP (tos 0x0, ttl 63, id 15852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827381718:2827382724, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949617 fvup0 In  IP (tos 0x0, ttl 64, id 15853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827382724:2827383730, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949619 fvmb0 Out IP (tos 0x0, ttl 63, id 15853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827382724:2827383730, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949624 fvmb0 In  IP (tos 0x0, ttl 64, id 48538, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x2e4b (correct), ack 2827380712, win 47, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949625 fvup0 In  IP (tos 0x0, ttl 64, id 15854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827383730:2827384736, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949627 fvmb0 Out IP (tos 0x0, ttl 63, id 15854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827383730:2827384736, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949628 fvup0 Out IP (tos 0x0, ttl 63, id 48538, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x5002 (correct), ack 2827380712, win 47, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949635 fvup0 In  IP (tos 0x0, ttl 64, id 15855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827384736:2827385742, ack 4179344625, win 42, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949637 fvmb0 Out IP (tos 0x0, ttl 63, id 15855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827384736:2827385742, ack 4179344625, win 42, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949646 fvup0 In  IP (tos 0x0, ttl 64, id 15856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827385742:2827386748, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949648 fvmb0 Out IP (tos 0x0, ttl 63, id 15856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827385742:2827386748, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949654 fvup0 In  IP (tos 0x0, ttl 64, id 15857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827386748:2827387754, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949656 fvmb0 Out IP (tos 0x0, ttl 63, id 15857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827386748:2827387754, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949663 fvup0 In  IP (tos 0x0, ttl 64, id 15858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827387754:2827388760, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949665 fvmb0 Out IP (tos 0x0, ttl 63, id 15858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827387754:2827388760, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949670 fvup0 In  IP (tos 0x0, ttl 64, id 15859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827388760:2827389766, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949672 fvmb0 In  IP (tos 0x0, ttl 64, id 48539, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x2670 (correct), ack 2827382724, win 46, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949672 fvmb0 Out IP (tos 0x0, ttl 63, id 15859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827388760:2827389766, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949672 fvmb0 In  IP (tos 0x0, ttl 64, id 48540, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x1aa7 (correct), ack 2827385742, win 44, options [nop,nop,TS val 1802728038 ecr 622405082], length 0
1793456389.949676 fvup0 Out IP (tos 0x0, ttl 63, id 48539, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x4827 (correct), ack 2827382724, win 46, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949677 fvup0 Out IP (tos 0x0, ttl 63, id 48540, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x3c5e (correct), ack 2827385742, win 44, options [nop,nop,TS val 1802728038 ecr 622405082], length 0
1793456389.949681 fvup0 In  IP (tos 0x0, ttl 64, id 15860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827389766:2827390772, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949682 fvmb0 Out IP (tos 0x0, ttl 63, id 15860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827389766:2827390772, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949687 fvmb0 In  IP (tos 0x0, ttl 64, id 48541, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x12cb (correct), ack 2827387754, win 43, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949689 fvup0 Out IP (tos 0x0, ttl 63, id 48541, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x3482 (correct), ack 2827387754, win 43, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949690 fvup0 In  IP (tos 0x0, ttl 64, id 15861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827390772:2827391778, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949692 fvmb0 Out IP (tos 0x0, ttl 63, id 15861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827390772:2827391778, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949700 fvup0 In  IP (tos 0x0, ttl 64, id 15862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827391778:2827392784, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949702 fvmb0 Out IP (tos 0x0, ttl 63, id 15862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827391778:2827392784, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949707 fvup0 In  IP (tos 0x0, ttl 64, id 15863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827392784:2827393790, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949708 fvmb0 Out IP (tos 0x0, ttl 63, id 15863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827392784:2827393790, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949713 fvmb0 In  IP (tos 0x0, ttl 64, id 48542, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x0af0 (correct), ack 2827389766, win 42, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949715 fvup0 In  IP (tos 0x0, ttl 64, id 15864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827393790:2827394796, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949716 fvup0 Out IP (tos 0x0, ttl 63, id 48542, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x2ca7 (correct), ack 2827389766, win 42, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949717 fvmb0 Out IP (tos 0x0, ttl 63, id 15864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827393790:2827394796, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949724 fvup0 In  IP (tos 0x0, ttl 64, id 15865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827394796:2827395802, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949726 fvmb0 Out IP (tos 0x0, ttl 63, id 15865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827394796:2827395802, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949730 fvmb0 In  IP (tos 0x0, ttl 64, id 48543, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0x0315 (correct), ack 2827391778, win 41, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949733 fvup0 Out IP (tos 0x0, ttl 63, id 48543, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x24cc (correct), ack 2827391778, win 41, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949734 fvup0 In  IP (tos 0x0, ttl 64, id 15866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827395802:2827396808, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949735 fvmb0 Out IP (tos 0x0, ttl 63, id 15866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827395802:2827396808, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949738 fvmb0 In  IP (tos 0x0, ttl 64, id 48544, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xfb39 (correct), ack 2827393790, win 40, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949740 fvup0 Out IP (tos 0x0, ttl 63, id 48544, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x1cf1 (correct), ack 2827393790, win 40, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949742 fvup0 In  IP (tos 0x0, ttl 64, id 15867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827396808:2827397814, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949744 fvmb0 Out IP (tos 0x0, ttl 63, id 15867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827396808:2827397814, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949752 fvup0 In  IP (tos 0x0, ttl 64, id 15868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827397814:2827398820, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949753 fvmb0 Out IP (tos 0x0, ttl 63, id 15868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827397814:2827398820, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949759 fvup0 In  IP (tos 0x0, ttl 64, id 15869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827398820:2827399826, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949761 fvmb0 Out IP (tos 0x0, ttl 63, id 15869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827398820:2827399826, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949770 fvup0 In  IP (tos 0x0, ttl 64, id 15870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827399826:2827400832, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949771 fvmb0 Out IP (tos 0x0, ttl 63, id 15870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827399826:2827400832, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949775 fvmb0 In  IP (tos 0x0, ttl 64, id 48545, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xf35e (correct), ack 2827395802, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949777 fvup0 In  IP (tos 0x0, ttl 64, id 15871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827400832:2827401838, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949779 fvup0 Out IP (tos 0x0, ttl 63, id 48545, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x1516 (correct), ack 2827395802, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949779 fvmb0 Out IP (tos 0x0, ttl 63, id 15871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827400832:2827401838, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949781 fvmb0 In  IP (tos 0x0, ttl 64, id 48546, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xeb82 (correct), ack 2827397814, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949783 fvup0 Out IP (tos 0x0, ttl 63, id 48546, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x0d3a (correct), ack 2827397814, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949788 fvup0 In  IP (tos 0x0, ttl 64, id 15872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827401838:2827402844, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949789 fvmb0 Out IP (tos 0x0, ttl 63, id 15872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827401838:2827402844, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949793 fvmb0 In  IP (tos 0x0, ttl 64, id 48547, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xe3a6 (correct), ack 2827399826, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949796 fvup0 Out IP (tos 0x0, ttl 63, id 48547, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x055e (correct), ack 2827399826, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949796 fvup0 In  IP (tos 0x0, ttl 64, id 15873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827402844:2827403850, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949797 fvmb0 Out IP (tos 0x0, ttl 63, id 15873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827402844:2827403850, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949805 fvup0 In  IP (tos 0x0, ttl 64, id 15874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827403850:2827404856, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949807 fvmb0 Out IP (tos 0x0, ttl 63, id 15874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827403850:2827404856, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949812 fvup0 In  IP (tos 0x0, ttl 64, id 15875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827404856:2827405862, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949814 fvmb0 Out IP (tos 0x0, ttl 63, id 15875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827404856:2827405862, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949814 fvmb0 In  IP (tos 0x0, ttl 64, id 48548, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xdbca (correct), ack 2827401838, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949817 fvup0 Out IP (tos 0x0, ttl 63, id 48548, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xfd81 (correct), ack 2827401838, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949823 fvup0 In  IP (tos 0x0, ttl 64, id 15876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827405862:2827406868, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949825 fvmb0 Out IP (tos 0x0, ttl 63, id 15876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827405862:2827406868, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949831 fvup0 In  IP (tos 0x0, ttl 64, id 15877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827406868:2827407874, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949833 fvmb0 Out IP (tos 0x0, ttl 63, id 15877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827406868:2827407874, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949837 fvmb0 In  IP (tos 0x0, ttl 64, id 48549, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xd3ee (correct), ack 2827403850, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949840 fvup0 In  IP (tos 0x0, ttl 64, id 15878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827407874:2827408880, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949840 fvup0 Out IP (tos 0x0, ttl 63, id 48549, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xf5a5 (correct), ack 2827403850, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949843 fvmb0 Out IP (tos 0x0, ttl 63, id 15878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827407874:2827408880, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949850 fvup0 In  IP (tos 0x0, ttl 64, id 15879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827408880:2827409886, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949851 fvmb0 Out IP (tos 0x0, ttl 63, id 15879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [P.], seq 2827408880:2827409886, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949861 fvmb0 In  IP (tos 0x0, ttl 64, id 48550, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xcc12 (correct), ack 2827405862, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949863 fvup0 Out IP (tos 0x0, ttl 63, id 48550, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xedc9 (correct), ack 2827405862, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949872 fvmb0 In  IP (tos 0x0, ttl 64, id 48551, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xc437 (correct), ack 2827407874, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949875 fvup0 Out IP (tos 0x0, ttl 63, id 48551, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xe5ee (correct), ack 2827407874, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949881 fvmb0 In  IP (tos 0x0, ttl 64, id 48552, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xbc5b (correct), ack 2827409886, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949883 fvup0 Out IP (tos 0x0, ttl 63, id 48552, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xde12 (correct), ack 2827409886, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950084 fvmb0 In  IP (tos 0x0, ttl 64, id 48553, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [F.], cksum 0xbc59 (correct), seq 4179344625, ack 2827409886, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950086 fvup0 Out IP (tos 0x0, ttl 63, id 48553, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [F.], cksum 0xde10 (correct), seq 4179344625, ack 2827409886, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950158 fvup0 In  IP (tos 0x0, ttl 64, id 15880, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [F.], cksum 0x8645 (incorrect -> 0xddf3), seq 2827409886, ack 4179344626, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 0
1793456389.950161 fvmb0 Out IP (tos 0x0, ttl 63, id 15880, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.35902: Flags [F.], cksum 0xa7fc (incorrect -> 0xbc3c), seq 2827409886, ack 4179344626, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 0
1793456389.950226 fvmb0 In  IP (tos 0x0, ttl 64, id 48554, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.35902 > 192.0.3.10.9446: Flags [.], cksum 0xbc58 (correct), ack 2827409887, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950230 fvup0 Out IP (tos 0x0, ttl 63, id 48554, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xde0f (correct), ack 2827409887, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4318,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"5a:8d:b5:97:e7:d6","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4319,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"5a:bc:89:da:9c:4a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4322,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"46:08:5f:12:52:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"82:09:3c:35:ad:f2","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.83.207.231","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:32844
	 cubic wscale:10,10 rto:211 rtt:10.633/17.139 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:38205 bytes_acked:38205 bytes_received:19643 segs_out:101 segs_in:134 data_segs_out:98 data_segs_in:98 send 246538136bps lastsnd:2394 lastrcv:2395 lastack:2352 pacing_rate 493047288bps delivery_rate 52428800000bps delivered:99 app_limited busy:709ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.005 snd_wnd:65536
ESTAB 0      0       172.28.30.40:58744 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.411/0.381 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:4951 bytes_acked:4952 bytes_received:11199 segs_out:104 segs_in:79 data_segs_out:42 data_segs_in:73 send 281849148bps lastsnd:824 lastrcv:824 lastack:824 pacing_rate 562500752bps delivery_rate 2896000000bps delivered:43 app_limited busy:51ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58756 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.094/0.115 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:13110 bytes_acked:13111 bytes_received:15675 segs_out:112 segs_in:132 data_segs_out:57 data_segs_in:123 send 1232340426bps lastsnd:5763 lastrcv:5763 lastack:5763 pacing_rate 2464680848bps delivery_rate 2896000000bps delivered:58 app_limited busy:34ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58772 172.28.30.21:443
	 cubic wscale:10,10 rto:208 rtt:7.16/12.42 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2942 bytes_acked:2943 bytes_received:4325 segs_out:34 segs_in:34 data_segs_out:17 data_segs_in:22 send 16178771bps lastsnd:5763 lastrcv:5763 lastack:5763 pacing_rate 32354712bps delivery_rate 2106181816bps delivered:18 app_limited busy:125ms rcv_space:14480 rcv_ssthresh:85754 minrtt:0.007 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:60996
	 cubic wscale:10,10 rto:206 rtt:5.095/10.17 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:3152 bytes_acked:3152 bytes_received:2755 segs_out:4 segs_in:7 data_segs_out:2 data_segs_in:3 send 683336605bps lastsnd:7277 lastrcv:7284 lastack:7236 pacing_rate 1366572624bps delivery_rate 34816000000bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78451 minrtt:0.01 snd_wnd:87040
ESTAB 0      0       172.28.30.40:34032 172.28.30.30:9443
	 cubic wscale:10,10 rto:205 rtt:4.786/3.374 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:36596 bytes_acked:36597 bytes_received:30165 segs_out:92 segs_in:54 data_segs_out:62 data_segs_in:38 send 24203928bps lastsnd:2426 lastrcv:2421 lastack:2421 pacing_rate 48405320bps delivery_rate 2106181816bps delivered:63 app_limited busy:133ms rcv_rtt:6.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.008 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:48150
	 cubic wscale:10,10 rto:201 rtt:0.578/1.127 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:22063 bytes_acked:22063 bytes_received:29592 segs_out:35 segs_in:58 data_segs_out:29 data_segs_in:29 send 6306989619bps lastsnd:2421 lastrcv:2427 lastack:2421 pacing_rate 12594912544bps delivery_rate 182272000000bps delivered:30 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:105032 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:60980
	 cubic wscale:10,10 rto:206 rtt:5.055/10.086 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2832 bytes_acked:2832 bytes_received:4655 segs_out:4 segs_in:8 data_segs_out:2 data_segs_in:4 send 688743818bps lastsnd:7465 lastrcv:7490 lastack:7424 pacing_rate 1377385456bps delivery_rate 29013333328bps delivered:3 app_limited busy:41ms rcv_rtt:1 rcv_space:65483 rcv_ssthresh:78343 minrtt:0.012 snd_wnd:87040
ESTAB 0      0          127.0.0.1:32844    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.728/0.041 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:19643 bytes_acked:19644 bytes_received:38205 segs_out:134 segs_in:102 data_segs_out:98 data_segs_in:98 send 3600879121bps lastsnd:2406 lastrcv:2405 lastack:2405 pacing_rate 7201758240bps delivery_rate 26214400000bps delivered:99 app_limited busy:123ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.01 snd_wnd:65536
ESTAB 0      0          127.0.0.1:48150    127.0.0.1:9444
	 cubic wscale:10,10 rto:206 rtt:5.527/2.817 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:29592 bytes_acked:29593 bytes_received:22063 segs_out:58 segs_in:36 data_segs_out:29 data_segs_in:29 send 763321874bps lastsnd:2427 lastrcv:2421 lastack:2421 pacing_rate 1526505648bps delivery_rate 84377600000bps delivered:30 app_limited busy:126ms rcv_rtt:6 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.005 snd_wnd:105472
ESTAB 0      0          127.0.0.1:60980    127.0.0.1:9444
	 cubic wscale:10,10 rto:204 rtt:3.23/6.412 ato:40 mss:39424 pmtu:65535 rcvmss:1998 advmss:65483 cwnd:10 bytes_sent:4655 bytes_acked:4656 bytes_received:2832 segs_out:8 segs_in:5 data_segs_out:4 data_segs_in:2 send 976445820bps lastsnd:7490 lastrcv:7465 lastack:7465 pacing_rate 1952664936bps delivery_rate 19712000000bps delivered:5 app_limited busy:26ms rcv_space:65495 rcv_ssthresh:86858 minrtt:0.006 snd_wnd:78848
ESTAB 0      0          127.0.0.1:60996    127.0.0.1:9444
	 cubic wscale:10,10 rto:201 rtt:0.041/0.053 ato:40 mss:39424 pmtu:65535 rcvmss:1996 advmss:65483 cwnd:10 bytes_sent:2755 bytes_acked:2756 bytes_received:3152 segs_out:7 segs_in:5 data_segs_out:3 data_segs_in:2 send 76924878049bps lastsnd:7287 lastrcv:7280 lastack:7280 pacing_rate 151085988016bps delivery_rate 24260923072bps delivered:4 app_limited rcv_space:65495 rcv_ssthresh:86642 minrtt:0.013 snd_wnd:78848

- matrixDual_capture_1_receipt: {"bytes":"59046","sha256":"79134af308dcdaa45c2fc0d708856e85023b8de3bd5ae05bb81c21280a795a63"}
- matrixDual_capture_2_window: {"launch_requested_at":"2026-10-31T14:19:48.724588919Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:50.792923175Z","stop_requested_at":"2026-10-31T14:19:50.725657014Z"}
- matrixDual_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
138 packets captured
138 packets received by filter
0 packets dropped by kernel

- matrixDual_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456389.948656 eth0  In  IP (tos 0x0, ttl 63, id 48467, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [S], cksum 0x7c3d (correct), seq 4179280516, win 64480, options [mss 1240,sackOK,TS val 1802728036 ecr 0,nop,wscale 10], length 0
1793456389.948675 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [S.], cksum 0x864d (incorrect -> 0xa6d4), seq 2827345498, ack 4179280517, win 65160, options [mss 1460,sackOK,TS val 622405082 ecr 1802728036,nop,wscale 10], length 0
1793456389.948800 eth0  In  IP (tos 0x0, ttl 63, id 48468, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xd3ec (correct), ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.948988 eth0  In  IP (tos 0x0, ttl 63, id 48469, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], cksum 0x8270 (correct), seq 4179280517:4179280561, ack 2827345499, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 44
1793456389.948994 eth0  Out IP (tos 0x0, ttl 64, id 15808, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xd3bf), ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949006 eth0  Out IP (tos 0x0, ttl 64, id 15809, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], cksum 0x8648 (incorrect -> 0x7a69), seq 2827345499:2827345502, ack 4179280561, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 3
1793456389.949095 eth0  In  IP (tos 0x0, ttl 63, id 48470, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xd3bd (correct), ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949178 eth0  In  IP (tos 0x0, ttl 63, id 48471, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179280561:4179281789, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949187 eth0  In  IP (tos 0x0, ttl 63, id 48472, offset 0, flags [DF], proto TCP (6), length 4964)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179281789:4179286701, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 4912
1793456389.949189 eth0  Out IP (tos 0x0, ttl 64, id 15810, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xbbc0), ack 4179286701, win 64, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949198 eth0  In  IP (tos 0x0, ttl 63, id 48476, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179286701:4179290385, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949204 eth0  In  IP (tos 0x0, ttl 63, id 48479, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179290385:4179292841, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949206 eth0  Out IP (tos 0x0, ttl 64, id 15811, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827345502:2827346508, ack 4179290385, win 61, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949226 eth0  Out IP (tos 0x0, ttl 64, id 15812, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0x9fdb), ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949236 eth0  Out IP (tos 0x0, ttl 64, id 15813, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827346508:2827347514, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949247 eth0  Out IP (tos 0x0, ttl 64, id 15814, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827347514:2827348520, ack 4179292841, win 59, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949247 eth0  In  IP (tos 0x0, ttl 63, id 48481, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179292841:4179294069, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949254 eth0  In  IP (tos 0x0, ttl 63, id 48482, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179294069:4179296525, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949258 eth0  Out IP (tos 0x0, ttl 64, id 15815, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827348520:2827349526, ack 4179296525, win 56, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949261 eth0  In  IP (tos 0x0, ttl 63, id 48484, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179296525:4179298981, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949266 eth0  In  IP (tos 0x0, ttl 63, id 48486, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179298981:4179301437, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949270 eth0  Out IP (tos 0x0, ttl 64, id 15816, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827349526:2827350532, ack 4179301437, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949272 eth0  In  IP (tos 0x0, ttl 63, id 48488, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179301437:4179302665, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949279 eth0  In  IP (tos 0x0, ttl 63, id 48489, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179302665:4179303893, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949282 eth0  Out IP (tos 0x0, ttl 64, id 15817, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0x64fe), ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949284 eth0  Out IP (tos 0x0, ttl 64, id 15818, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827350532:2827351538, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949294 eth0  Out IP (tos 0x0, ttl 64, id 15819, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827351538:2827352544, ack 4179303893, win 52, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949297 eth0  In  IP (tos 0x0, ttl 63, id 48490, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179303893:4179305121, ack 2827345502, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949298 eth0  In  IP (tos 0x0, ttl 63, id 48491, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179305121:4179307577, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949303 eth0  In  IP (tos 0x0, ttl 63, id 48493, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179307577:4179310033, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949304 eth0  Out IP (tos 0x0, ttl 64, id 15820, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0x4ec1), ack 4179307577, win 49, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949309 eth0  In  IP (tos 0x0, ttl 63, id 48495, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179310033:4179312489, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949312 eth0  Out IP (tos 0x0, ttl 64, id 15821, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827352544:2827353550, ack 4179312489, win 45, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949315 eth0  In  IP (tos 0x0, ttl 63, id 48497, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179312489:4179314945, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949320 eth0  In  IP (tos 0x0, ttl 63, id 48499, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179314945:4179316173, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949324 eth0  Out IP (tos 0x0, ttl 64, id 15822, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827353550:2827354556, ack 4179316173, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949330 eth0  In  IP (tos 0x0, ttl 63, id 48500, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179316173:4179317401, ack 2827346508, win 63, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949331 eth0  In  IP (tos 0x0, ttl 63, id 48501, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x380c (correct), ack 2827348520, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949337 eth0  In  IP (tos 0x0, ttl 63, id 48502, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179317401:4179321085, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949339 eth0  Out IP (tos 0x0, ttl 64, id 15823, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827354556:2827355562, ack 4179317401, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949342 eth0  In  IP (tos 0x0, ttl 63, id 48505, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179321085:4179322313, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949350 eth0  Out IP (tos 0x0, ttl 64, id 15824, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827355562:2827356568, ack 4179322313, win 38, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949362 eth0  In  IP (tos 0x0, ttl 63, id 48506, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179322313:4179324769, ack 2827349526, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949363 eth0  Out IP (tos 0x0, ttl 64, id 15825, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827356568:2827357574, ack 4179322313, win 41, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949364 eth0  In  IP (tos 0x0, ttl 63, id 48508, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179324769:4179328453, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949369 eth0  In  IP (tos 0x0, ttl 63, id 48511, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179328453:4179330909, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949371 eth0  Out IP (tos 0x0, ttl 64, id 15826, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827357574:2827358580, ack 4179330909, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949377 eth0  In  IP (tos 0x0, ttl 63, id 48513, offset 0, flags [DF], proto TCP (6), length 3736)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179330909:4179334593, ack 2827350532, win 62, options [nop,nop,TS val 1802728037 ecr 622405082], length 3684
1793456389.949381 eth0  Out IP (tos 0x0, ttl 64, id 15827, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xcdb7), ack 4179334593, win 30, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949383 eth0  In  IP (tos 0x0, ttl 63, id 48516, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xe52c (correct), ack 2827352544, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949388 eth0  In  IP (tos 0x0, ttl 63, id 48517, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179334593:4179337049, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949391 eth0  Out IP (tos 0x0, ttl 64, id 15828, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827358580:2827359586, ack 4179337049, win 28, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949395 eth0  In  IP (tos 0x0, ttl 63, id 48519, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179337049:4179339505, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949400 eth0  In  IP (tos 0x0, ttl 63, id 48521, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179339505:4179340733, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949402 eth0  Out IP (tos 0x0, ttl 64, id 15829, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827359586:2827360592, ack 4179340733, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949406 eth0  In  IP (tos 0x0, ttl 63, id 48522, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179340733:4179343189, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 2456
1793456389.949410 eth0  Out IP (tos 0x0, ttl 64, id 15830, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [.], cksum 0x8645 (incorrect -> 0xa44e), ack 4179343189, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 0
1793456389.949412 eth0  In  IP (tos 0x0, ttl 63, id 48524, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], seq 4179343189:4179344417, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 1228
1793456389.949417 eth0  Out IP (tos 0x0, ttl 64, id 15831, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827360592:2827361598, ack 4179344417, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949420 eth0  In  IP (tos 0x0, ttl 63, id 48525, offset 0, flags [DF], proto TCP (6), length 260)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [P.], seq 4179344417:4179344625, ack 2827353550, win 61, options [nop,nop,TS val 1802728037 ecr 622405082], length 208
1793456389.949421 eth0  In  IP (tos 0x0, ttl 63, id 48526, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xb233 (correct), ack 2827355562, win 60, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949425 eth0  Out IP (tos 0x0, ttl 64, id 15832, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827361598:2827362604, ack 4179344625, win 22, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949428 eth0  In  IP (tos 0x0, ttl 63, id 48527, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xaa58 (correct), ack 2827357574, win 59, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949432 eth0  In  IP (tos 0x0, ttl 63, id 48528, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xa27d (correct), ack 2827359586, win 58, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949436 eth0  Out IP (tos 0x0, ttl 64, id 15833, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827362604:2827363610, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949446 eth0  Out IP (tos 0x0, ttl 64, id 15834, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827363610:2827364616, ack 4179344625, win 23, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949455 eth0  Out IP (tos 0x0, ttl 64, id 15835, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827364616:2827365622, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949464 eth0  Out IP (tos 0x0, ttl 64, id 15836, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827365622:2827366628, ack 4179344625, win 25, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949465 eth0  In  IP (tos 0x0, ttl 63, id 48529, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x96b5 (correct), ack 2827362604, win 56, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949476 eth0  Out IP (tos 0x0, ttl 64, id 15837, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827366628:2827367634, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949483 eth0  Out IP (tos 0x0, ttl 64, id 15838, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827367634:2827368640, ack 4179344625, win 27, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949484 eth0  In  IP (tos 0x0, ttl 63, id 48530, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x8eda (correct), ack 2827364616, win 55, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949492 eth0  Out IP (tos 0x0, ttl 64, id 15839, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827368640:2827369646, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949499 eth0  Out IP (tos 0x0, ttl 64, id 15840, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827369646:2827370652, ack 4179344625, win 29, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949501 eth0  In  IP (tos 0x0, ttl 63, id 48531, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x86ff (correct), ack 2827366628, win 54, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949509 eth0  Out IP (tos 0x0, ttl 64, id 15841, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827370652:2827371658, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949516 eth0  Out IP (tos 0x0, ttl 64, id 15842, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827371658:2827372664, ack 4179344625, win 31, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949521 eth0  In  IP (tos 0x0, ttl 63, id 48532, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x7f24 (correct), ack 2827368640, win 53, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949525 eth0  Out IP (tos 0x0, ttl 64, id 15843, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827372664:2827373670, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949534 eth0  Out IP (tos 0x0, ttl 64, id 15844, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827373670:2827374676, ack 4179344625, win 33, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949538 eth0  In  IP (tos 0x0, ttl 63, id 48533, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x7749 (correct), ack 2827370652, win 52, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949545 eth0  Out IP (tos 0x0, ttl 64, id 15845, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827374676:2827375682, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949553 eth0  Out IP (tos 0x0, ttl 64, id 15846, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827375682:2827376688, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949561 eth0  Out IP (tos 0x0, ttl 64, id 15847, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827376688:2827377694, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949571 eth0  Out IP (tos 0x0, ttl 64, id 15848, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827377694:2827378700, ack 4179344625, win 37, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949581 eth0  Out IP (tos 0x0, ttl 64, id 15849, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827378700:2827379706, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949584 eth0  In  IP (tos 0x0, ttl 63, id 48534, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x6f6e (correct), ack 2827372664, win 51, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949588 eth0  Out IP (tos 0x0, ttl 64, id 15850, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827379706:2827380712, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949596 eth0  In  IP (tos 0x0, ttl 63, id 48535, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x6793 (correct), ack 2827374676, win 50, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949596 eth0  Out IP (tos 0x0, ttl 64, id 15851, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827380712:2827381718, ack 4179344625, win 40, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949597 eth0  In  IP (tos 0x0, ttl 63, id 48536, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x5fb8 (correct), ack 2827376688, win 49, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949608 eth0  Out IP (tos 0x0, ttl 64, id 15852, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827381718:2827382724, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949609 eth0  In  IP (tos 0x0, ttl 63, id 48537, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x57dd (correct), ack 2827378700, win 48, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949616 eth0  Out IP (tos 0x0, ttl 64, id 15853, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827382724:2827383730, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949624 eth0  Out IP (tos 0x0, ttl 64, id 15854, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827383730:2827384736, ack 4179344625, win 42, options [nop,nop,TS val 622405082 ecr 1802728037], length 1006
1793456389.949631 eth0  In  IP (tos 0x0, ttl 63, id 48538, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x5002 (correct), ack 2827380712, win 47, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949635 eth0  Out IP (tos 0x0, ttl 64, id 15855, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827384736:2827385742, ack 4179344625, win 42, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949644 eth0  Out IP (tos 0x0, ttl 64, id 15856, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827385742:2827386748, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949653 eth0  Out IP (tos 0x0, ttl 64, id 15857, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827386748:2827387754, ack 4179344625, win 46, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949662 eth0  Out IP (tos 0x0, ttl 64, id 15858, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827387754:2827388760, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949669 eth0  Out IP (tos 0x0, ttl 64, id 15859, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827388760:2827389766, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949679 eth0  In  IP (tos 0x0, ttl 63, id 48539, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x4827 (correct), ack 2827382724, win 46, options [nop,nop,TS val 1802728037 ecr 622405082], length 0
1793456389.949680 eth0  Out IP (tos 0x0, ttl 64, id 15860, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827389766:2827390772, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728037], length 1006
1793456389.949681 eth0  In  IP (tos 0x0, ttl 63, id 48540, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x3c5e (correct), ack 2827385742, win 44, options [nop,nop,TS val 1802728038 ecr 622405082], length 0
1793456389.949689 eth0  Out IP (tos 0x0, ttl 64, id 15861, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827390772:2827391778, ack 4179344625, win 49, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949690 eth0  In  IP (tos 0x0, ttl 63, id 48541, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x3482 (correct), ack 2827387754, win 43, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949699 eth0  Out IP (tos 0x0, ttl 64, id 15862, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827391778:2827392784, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949706 eth0  Out IP (tos 0x0, ttl 64, id 15863, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827392784:2827393790, ack 4179344625, win 52, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949714 eth0  Out IP (tos 0x0, ttl 64, id 15864, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827393790:2827394796, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949717 eth0  In  IP (tos 0x0, ttl 63, id 48542, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x2ca7 (correct), ack 2827389766, win 42, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949723 eth0  Out IP (tos 0x0, ttl 64, id 15865, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827394796:2827395802, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949733 eth0  Out IP (tos 0x0, ttl 64, id 15866, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827395802:2827396808, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949734 eth0  In  IP (tos 0x0, ttl 63, id 48543, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x24cc (correct), ack 2827391778, win 41, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949741 eth0  In  IP (tos 0x0, ttl 63, id 48544, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x1cf1 (correct), ack 2827393790, win 40, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949741 eth0  Out IP (tos 0x0, ttl 64, id 15867, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827396808:2827397814, ack 4179344625, win 55, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949751 eth0  Out IP (tos 0x0, ttl 64, id 15868, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827397814:2827398820, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949759 eth0  Out IP (tos 0x0, ttl 64, id 15869, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827398820:2827399826, ack 4179344625, win 58, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949769 eth0  Out IP (tos 0x0, ttl 64, id 15870, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827399826:2827400832, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949776 eth0  Out IP (tos 0x0, ttl 64, id 15871, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827400832:2827401838, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949780 eth0  In  IP (tos 0x0, ttl 63, id 48545, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x1516 (correct), ack 2827395802, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949784 eth0  In  IP (tos 0x0, ttl 63, id 48546, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x0d3a (correct), ack 2827397814, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949787 eth0  Out IP (tos 0x0, ttl 64, id 15872, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827401838:2827402844, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949795 eth0  Out IP (tos 0x0, ttl 64, id 15873, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827402844:2827403850, ack 4179344625, win 61, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949797 eth0  In  IP (tos 0x0, ttl 63, id 48547, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0x055e (correct), ack 2827399826, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949804 eth0  Out IP (tos 0x0, ttl 64, id 15874, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827403850:2827404856, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949811 eth0  Out IP (tos 0x0, ttl 64, id 15875, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827404856:2827405862, ack 4179344625, win 64, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949817 eth0  In  IP (tos 0x0, ttl 63, id 48548, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xfd81 (correct), ack 2827401838, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949822 eth0  Out IP (tos 0x0, ttl 64, id 15876, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827405862:2827406868, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949830 eth0  Out IP (tos 0x0, ttl 64, id 15877, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827406868:2827407874, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949839 eth0  Out IP (tos 0x0, ttl 64, id 15878, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827407874:2827408880, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949842 eth0  In  IP (tos 0x0, ttl 63, id 48549, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xf5a5 (correct), ack 2827403850, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949849 eth0  Out IP (tos 0x0, ttl 64, id 15879, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [P.], seq 2827408880:2827409886, ack 4179344625, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 1006
1793456389.949864 eth0  In  IP (tos 0x0, ttl 63, id 48550, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xedc9 (correct), ack 2827405862, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949876 eth0  In  IP (tos 0x0, ttl 63, id 48551, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xe5ee (correct), ack 2827407874, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.949884 eth0  In  IP (tos 0x0, ttl 63, id 48552, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xde12 (correct), ack 2827409886, win 38, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950087 eth0  In  IP (tos 0x0, ttl 63, id 48553, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [F.], cksum 0xde10 (correct), seq 4179344625, ack 2827409886, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0
1793456389.950157 eth0  Out IP (tos 0x0, ttl 64, id 15880, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.35902: Flags [F.], cksum 0x8645 (incorrect -> 0xddf3), seq 2827409886, ack 4179344626, win 67, options [nop,nop,TS val 622405083 ecr 1802728038], length 0
1793456389.950231 eth0  In  IP (tos 0x0, ttl 63, id 48554, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.35902 > 192.0.3.10.9446: Flags [.], cksum 0xde0f (correct), ack 2827409887, win 39, options [nop,nop,TS val 1802728038 ecr 622405083], length 0

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

- matrixDual_capture_2_receipt: {"bytes":"17439","sha256":"791981effd1235c919e4cda141dbfb6307d522b8d284b026e99e41c084755448"}
- matrixDual_origin_endpoint_samples: {"at":"2026-10-31T14:19:35.728093876Z","local_port":9446,"remote_port":37946,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":97,"sacked":0,"snd_cwnd":33,"snd_mss":1228,"snd_wnd":68608,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:19:42.865265238Z","local_port":9446,"remote_port":37962,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":24972,"retrans":0,"rto_us":202000,"rtt_us":1396,"sacked":0,"snd_cwnd":71,"snd_mss":1228,"snd_wnd":60416,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:19:49.950099398Z","local_port":9446,"remote_port":35902,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":58,"sacked":0,"snd_cwnd":21,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}

- matrix1280_input: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- matrix1280_transfer: {"after":{"upload_bytes":"269593","download_bytes":"272998","Present":true},"before":{"upload_bytes":"201369","download_bytes":"204651","Present":true},"bulk":{"endpoint":{"at":"2026-10-31T14:19:56.980234107Z","local_port":37850,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":160,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1},"path_after":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"path_before":{"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}},"selected_after":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"},"selected_before":{"connected":true,"connection_type":"P2P","chosen_family":"ipv4"}}
- matrix1280_ipv6_block: {"native_member":{"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":0,"error":"connect_unavailable"},"positive_control":{"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0},"scope":"D-123: IPv6 connection must fail; IPv4 bulk must complete. Native refusal may precede tunnel ingress; this does not claim ctv6 was hit. Kernel ctv6 proof is the separate netlab row."}
- matrix1280_capture_0_window: {"launch_requested_at":"2026-10-31T14:19:55.694274042Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:57.241113874Z","stop_requested_at":"2026-10-31T14:19:57.167263964Z"}
- matrix1280_capture_0_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
343 packets captured
347 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_0_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456396.978820 fvnb0 Out IP (tos 0x0, ttl 64, id 13955, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [S], cksum 0xa804 (incorrect -> 0xbe5b), seq 533410515, win 64480, options [mss 1240,sackOK,TS val 2675588077 ecr 0,nop,wscale 10], length 0
1793456396.979003 fvnb0 In  IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [S.], cksum 0x671b (correct), seq 2413624814, ack 533410516, win 65160, options [mss 1460,sackOK,TS val 2782928895 ecr 2675588077,nop,wscale 10], length 0
1793456396.979016 fvnb0 Out IP (tos 0x0, ttl 64, id 13956, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x9434), ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979051 fvnb0 Out IP (tos 0x0, ttl 64, id 13957, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], cksum 0xa828 (incorrect -> 0x42b8), seq 533410516:533410560, ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 44
1793456396.979111 fvnb0 In  IP (tos 0x0, ttl 63, id 20705, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0x9407 (correct), ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979215 fvnb0 In  IP (tos 0x0, ttl 63, id 20706, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], cksum 0x3ab1 (correct), seq 2413624815:2413624818, ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 3
1793456396.979220 fvnb0 Out IP (tos 0x0, ttl 64, id 13958, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x9405), ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979290 fvnb0 Out IP (tos 0x0, ttl 64, id 13959, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533410560:533416700, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 6140
1793456396.979304 fvnb0 Out IP (tos 0x0, ttl 64, id 13964, offset 0, flags [DF], proto TCP (6), length 6192)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533416700:533422840, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 6140
1793456396.979387 fvnb0 In  IP (tos 0x0, ttl 63, id 20707, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0x859e (correct), ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979392 fvnb0 Out IP (tos 0x0, ttl 64, id 13969, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533422840:533430208, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 7368
1793456396.979399 fvnb0 In  IP (tos 0x0, ttl 63, id 20708, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413624818:2413625824, ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979417 fvnb0 In  IP (tos 0x0, ttl 63, id 20709, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0x734f (correct), ack 533417928, win 63, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979420 fvnb0 Out IP (tos 0x0, ttl 64, id 13975, offset 0, flags [DF], proto TCP (6), length 7420)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533430208:533437576, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 7368
1793456396.979425 fvnb0 In  IP (tos 0x0, ttl 63, id 20710, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413625824:2413626830, ack 533420384, win 61, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979441 fvnb0 In  IP (tos 0x0, ttl 63, id 20711, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413626830:2413627836, ack 533422840, win 59, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979444 fvnb0 Out IP (tos 0x0, ttl 64, id 13981, offset 0, flags [DF], proto TCP (6), length 9876)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533437576:533447400, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 9824
1793456396.979472 fvnb0 In  IP (tos 0x0, ttl 63, id 20712, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413627836:2413628842, ack 533422840, win 60, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979475 fvnb0 In  IP (tos 0x0, ttl 63, id 20713, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413628842:2413629848, ack 533425296, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979479 fvnb0 In  IP (tos 0x0, ttl 63, id 20714, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0x3d3c (correct), ack 533427752, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979480 fvnb0 Out IP (tos 0x0, ttl 64, id 13989, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xf078), ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979483 fvnb0 In  IP (tos 0x0, ttl 63, id 20715, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413629848:2413630854, ack 533430208, win 57, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979484 fvnb0 Out IP (tos 0x0, ttl 64, id 13990, offset 0, flags [DF], proto TCP (6), length 9876)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533447400:533457224, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 9824
1793456396.979492 fvnb0 In  IP (tos 0x0, ttl 63, id 20716, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413630854:2413631860, ack 533431436, win 56, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979498 fvnb0 In  IP (tos 0x0, ttl 63, id 20717, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0x1d68 (correct), ack 533433892, win 54, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979500 fvnb0 Out IP (tos 0x0, ttl 64, id 13998, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xc23d), ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979503 fvnb0 Out IP (tos 0x0, ttl 64, id 13999, offset 0, flags [DF], proto TCP (6), length 12332)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533457224:533469504, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 12280
1793456396.979523 fvnb0 In  IP (tos 0x0, ttl 63, id 20718, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413631860:2413632866, ack 533435120, win 53, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979526 fvnb0 In  IP (tos 0x0, ttl 63, id 20719, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413632866:2413633872, ack 533436348, win 52, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979532 fvnb0 Out IP (tos 0x0, ttl 64, id 14009, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x8a6a), ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979545 fvnb0 In  IP (tos 0x0, ttl 63, id 20720, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xf8c8 (correct), ack 533441260, win 49, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979547 fvnb0 Out IP (tos 0x0, ttl 64, id 14010, offset 0, flags [DF], proto TCP (6), length 5172)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533469504:533474624, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 5120
1793456396.979567 fvnb0 In  IP (tos 0x0, ttl 63, id 20721, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413633872:2413634878, ack 533444944, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979567 fvnb0 In  IP (tos 0x0, ttl 63, id 20722, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413634878:2413635884, ack 533446172, win 45, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979581 fvnb0 Out IP (tos 0x0, ttl 64, id 14015, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x6e8f), ack 2413635884, win 58, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979583 fvnb0 In  IP (tos 0x0, ttl 63, id 20723, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413635884:2413636890, ack 533447400, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979589 fvnb0 In  IP (tos 0x0, ttl 63, id 20724, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413636890:2413637896, ack 533451084, win 43, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979599 fvnb0 Out IP (tos 0x0, ttl 64, id 14016, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x66b4), ack 2413637896, win 57, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979601 fvnb0 In  IP (tos 0x0, ttl 63, id 20725, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413637896:2413638902, ack 533453540, win 41, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979622 fvnb0 In  IP (tos 0x0, ttl 63, id 20726, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413638902:2413639908, ack 533454768, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979623 fvnb0 In  IP (tos 0x0, ttl 63, id 20727, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa2e1 (correct), ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979630 fvnb0 Out IP (tos 0x0, ttl 64, id 14017, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x5ed9), ack 2413639908, win 56, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979695 fvnb0 In  IP (tos 0x0, ttl 63, id 20728, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413639908:2413640914, ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979696 fvnb0 In  IP (tos 0x0, ttl 63, id 20729, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413640914:2413641920, ack 533460908, win 37, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979696 fvnb0 In  IP (tos 0x0, ttl 63, id 20730, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413641920:2413642926, ack 533462136, win 37, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979697 fvnb0 In  IP (tos 0x0, ttl 63, id 20731, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413642926:2413643932, ack 533465820, win 34, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979697 fvnb0 In  IP (tos 0x0, ttl 63, id 20732, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0x6804 (correct), ack 533468276, win 32, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979704 fvnb0 Out IP (tos 0x0, ttl 64, id 14018, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x56fd), ack 2413641920, win 55, options [nop,nop,TS val 2675588078 ecr 2782928895], length 0
1793456396.979704 fvnb0 In  IP (tos 0x0, ttl 63, id 20733, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413643932:2413644938, ack 533469504, win 31, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979705 fvnb0 In  IP (tos 0x0, ttl 63, id 20734, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413644938:2413645944, ack 533471960, win 29, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979711 fvnb0 In  IP (tos 0x0, ttl 63, id 20735, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0x4761 (correct), ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979716 fvnb0 Out IP (tos 0x0, ttl 64, id 14019, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x4747), ack 2413645944, win 52, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979737 fvnb0 In  IP (tos 0x0, ttl 63, id 20736, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413645944:2413646950, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979740 fvnb0 In  IP (tos 0x0, ttl 63, id 20737, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413646950:2413647956, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979745 fvnb0 Out IP (tos 0x0, ttl 64, id 14020, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x3f6c), ack 2413647956, win 51, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979757 fvnb0 In  IP (tos 0x0, ttl 63, id 20738, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413647956:2413648962, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979769 fvnb0 In  IP (tos 0x0, ttl 63, id 20739, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413648962:2413649968, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979772 fvnb0 In  IP (tos 0x0, ttl 63, id 20740, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413649968:2413650974, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979776 fvnb0 Out IP (tos 0x0, ttl 64, id 14021, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x33a4), ack 2413650974, win 49, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979783 fvnb0 In  IP (tos 0x0, ttl 63, id 20741, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413650974:2413651980, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979796 fvnb0 In  IP (tos 0x0, ttl 63, id 20742, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413651980:2413652986, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979804 fvnb0 In  IP (tos 0x0, ttl 63, id 20743, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413652986:2413653992, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979807 fvnb0 Out IP (tos 0x0, ttl 64, id 14022, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x2bc9), ack 2413652986, win 48, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979823 fvnb0 In  IP (tos 0x0, ttl 63, id 20744, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413653992:2413654998, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979828 fvnb0 Out IP (tos 0x0, ttl 64, id 14023, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x23ee), ack 2413654998, win 47, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979842 fvnb0 In  IP (tos 0x0, ttl 63, id 20745, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413654998:2413656004, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979854 fvnb0 In  IP (tos 0x0, ttl 63, id 20746, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413656004:2413657010, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979861 fvnb0 Out IP (tos 0x0, ttl 64, id 14024, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x1c13), ack 2413657010, win 46, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979865 fvnb0 In  IP (tos 0x0, ttl 63, id 20747, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413657010:2413658016, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979875 fvnb0 In  IP (tos 0x0, ttl 63, id 20748, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413658016:2413659022, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979879 fvnb0 Out IP (tos 0x0, ttl 64, id 14025, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x1438), ack 2413659022, win 45, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979884 fvnb0 In  IP (tos 0x0, ttl 63, id 20749, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413659022:2413660028, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979902 fvnb0 In  IP (tos 0x0, ttl 63, id 20750, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413660028:2413661034, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979907 fvnb0 Out IP (tos 0x0, ttl 64, id 14026, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x0c5d), ack 2413661034, win 44, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979922 fvnb0 In  IP (tos 0x0, ttl 63, id 20751, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413661034:2413662040, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979923 fvnb0 In  IP (tos 0x0, ttl 63, id 20752, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413662040:2413663046, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979929 fvnb0 In  IP (tos 0x0, ttl 63, id 20753, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413663046:2413664052, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979930 fvnb0 Out IP (tos 0x0, ttl 64, id 14027, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x0482), ack 2413663046, win 43, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979945 fvnb0 In  IP (tos 0x0, ttl 63, id 20754, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413664052:2413665058, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979957 fvnb0 In  IP (tos 0x0, ttl 63, id 20755, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413665058:2413666064, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979961 fvnb0 Out IP (tos 0x0, ttl 64, id 14028, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xfca6), ack 2413665058, win 42, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979967 fvnb0 In  IP (tos 0x0, ttl 63, id 20756, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413666064:2413667070, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979972 fvnb0 Out IP (tos 0x0, ttl 64, id 14029, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xf4cb), ack 2413667070, win 41, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979984 fvnb0 In  IP (tos 0x0, ttl 63, id 20757, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413667070:2413668076, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979987 fvnb0 In  IP (tos 0x0, ttl 63, id 20758, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413668076:2413669082, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979990 fvnb0 Out IP (tos 0x0, ttl 64, id 14030, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xecf0), ack 2413669082, win 40, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980005 fvnb0 In  IP (tos 0x0, ttl 63, id 20759, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413669082:2413670088, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980012 fvnb0 In  IP (tos 0x0, ttl 63, id 20760, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413670088:2413671094, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980015 fvnb0 Out IP (tos 0x0, ttl 64, id 14031, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xe515), ack 2413671094, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980024 fvnb0 In  IP (tos 0x0, ttl 63, id 20761, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413671094:2413672100, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980033 fvnb0 In  IP (tos 0x0, ttl 63, id 20762, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413672100:2413673106, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980037 fvnb0 Out IP (tos 0x0, ttl 64, id 14032, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xdd39), ack 2413673106, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980046 fvnb0 In  IP (tos 0x0, ttl 63, id 20763, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413673106:2413674112, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980056 fvnb0 In  IP (tos 0x0, ttl 63, id 20764, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413674112:2413675118, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980060 fvnb0 Out IP (tos 0x0, ttl 64, id 14033, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xd55d), ack 2413675118, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980068 fvnb0 In  IP (tos 0x0, ttl 63, id 20765, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413675118:2413676124, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980079 fvnb0 In  IP (tos 0x0, ttl 63, id 20766, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413676124:2413677130, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980082 fvnb0 Out IP (tos 0x0, ttl 64, id 14034, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xcd81), ack 2413677130, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980093 fvnb0 In  IP (tos 0x0, ttl 63, id 20767, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413677130:2413678136, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980108 fvnb0 In  IP (tos 0x0, ttl 63, id 20768, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413678136:2413679142, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980112 fvnb0 Out IP (tos 0x0, ttl 64, id 14035, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xc5a5), ack 2413679142, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980123 fvnb0 In  IP (tos 0x0, ttl 63, id 20769, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413679142:2413680148, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980134 fvnb0 In  IP (tos 0x0, ttl 63, id 20770, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413680148:2413681154, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980141 fvnb0 Out IP (tos 0x0, ttl 64, id 14036, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xbdc9), ack 2413681154, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980143 fvnb0 In  IP (tos 0x0, ttl 63, id 20771, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413681154:2413682160, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980156 fvnb0 In  IP (tos 0x0, ttl 63, id 20772, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413682160:2413683166, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980162 fvnb0 Out IP (tos 0x0, ttl 64, id 14037, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xb5ed), ack 2413683166, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980167 fvnb0 In  IP (tos 0x0, ttl 63, id 20773, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413683166:2413684172, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980183 fvnb0 In  IP (tos 0x0, ttl 63, id 20774, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413684172:2413685178, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980188 fvnb0 Out IP (tos 0x0, ttl 64, id 14038, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xae11), ack 2413685178, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980192 fvnb0 In  IP (tos 0x0, ttl 63, id 20775, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413685178:2413686184, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980198 fvnb0 In  IP (tos 0x0, ttl 63, id 20776, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413686184:2413687190, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980202 fvnb0 Out IP (tos 0x0, ttl 64, id 14039, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0xa635), ack 2413687190, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980214 fvnb0 In  IP (tos 0x0, ttl 63, id 20777, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413687190:2413688196, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980225 fvnb0 In  IP (tos 0x0, ttl 63, id 20778, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413688196:2413689202, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980230 fvnb0 Out IP (tos 0x0, ttl 64, id 14040, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x9e59), ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980407 fvnb0 Out IP (tos 0x0, ttl 64, id 14041, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [F.], cksum 0xa7fc (incorrect -> 0x9e58), seq 533474624, ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980538 fvnb0 In  IP (tos 0x0, ttl 63, id 20779, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [F.], cksum 0x9e3b (correct), seq 2413689202, ack 533474625, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 0
1793456396.980544 fvnb0 Out IP (tos 0x0, ttl 64, id 14042, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa7fc (incorrect -> 0x9e57), ack 2413689203, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4320,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"92:6d:bb:cd:a6:42","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4323,"ifname":"eth1","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"62:1a:d1:c5:55:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":8,"ifname":"fvnb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvnb0","protocol":"kernel","scope":"link","prefsrc":"100.83.128.120","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.41","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth1","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth1","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
ESTAB 0      0       172.28.30.41:45832 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.061/0.084 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:2178 bytes_acked:2179 bytes_received:5113 segs_out:16 segs_in:16 data_segs_out:9 data_segs_in:11 send 1899016393bps lastsnd:52705 lastrcv:52704 lastack:22534 pacing_rate 3744323232bps delivery_rate 3861333328bps delivered:10 app_limited rcv_space:14480 rcv_ssthresh:81050 minrtt:0.006 snd_wnd:71680
ESTAB 0      0       172.28.30.41:59296 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.166/0.217 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3513 bytes_acked:3514 bytes_received:5509 segs_out:37 segs_in:31 data_segs_out:20 data_segs_in:26 send 591807229bps lastsnd:514 lastrcv:514 lastack:514 pacing_rate 1179174792bps delivery_rate 2183111104bps delivered:21 app_limited busy:3ms rcv_space:12280 rcv_ssthresh:85803 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.41:59308 172.28.30.21:443
	 cubic wscale:10,10 rto:202 rtt:1.412/2.69 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:3860 bytes_acked:3861 bytes_received:6288 segs_out:33 segs_in:39 data_segs_out:19 data_segs_in:32 send 69575071bps lastsnd:5290 lastrcv:5290 lastack:5290 pacing_rate 139150136bps delivery_rate 2183111104bps delivered:20 app_limited busy:42ms rcv_space:12280 rcv_ssthresh:85793 minrtt:0.009 snd_wnd:79872
ESTAB 0      0       172.28.30.41:59312 172.28.30.21:443
	 cubic wscale:10,10 rto:206 rtt:5.358/10.519 ato:40 mss:1228 pmtu:1280 rcvmss:1228 advmss:1228 cwnd:10 bytes_sent:2320 bytes_acked:2321 bytes_received:2927 segs_out:14 segs_in:11 data_segs_out:8 data_segs_in:8 send 18335200bps lastsnd:5291 lastrcv:5291 lastack:5250 pacing_rate 36670392bps delivery_rate 1964800000bps delivered:9 app_limited busy:45ms rcv_space:12280 rcv_ssthresh:78066 minrtt:0.01 snd_wnd:74752

- matrix1280_capture_0_receipt: {"bytes":"45324","sha256":"f56a99cde7c1aa4d4c978253edc4bcf670abbbf51625c438a84bfecd6d4080fd"}
- matrix1280_capture_1_window: {"launch_requested_at":"2026-10-31T14:19:55.733765628Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:57.591679328Z","stop_requested_at":"2026-10-31T14:19:57.424422735Z"}
- matrix1280_ipv6_uplink_no_leak: true
- matrix1280_capture_1_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
534 packets captured
534 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_1_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456396.978890 fvmb0 In  IP (tos 0x0, ttl 64, id 13955, offset 0, flags [DF], proto TCP (6), length 60)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [S], cksum 0xbe5b (correct), seq 533410515, win 64480, options [mss 1240,sackOK,TS val 2675588077 ecr 0,nop,wscale 10], length 0
1793456396.978919 fvup0 Out IP (tos 0x0, ttl 63, id 13955, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [S], cksum 0xe012 (correct), seq 533410515, win 64480, options [mss 1240,sackOK,TS val 2675588077 ecr 0,nop,wscale 10], length 0
1793456396.978948 fvup0 In  IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [S.], cksum 0x864d (incorrect -> 0x88d2), seq 2413624814, ack 533410516, win 65160, options [mss 1460,sackOK,TS val 2782928895 ecr 2675588077,nop,wscale 10], length 0
1793456396.978955 fvmb0 Out IP (tos 0x0, ttl 63, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [S.], cksum 0xa804 (incorrect -> 0x671b), seq 2413624814, ack 533410516, win 65160, options [mss 1460,sackOK,TS val 2782928895 ecr 2675588077,nop,wscale 10], length 0
1793456396.979042 fvmb0 In  IP (tos 0x0, ttl 64, id 13956, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x9434 (correct), ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979046 fvup0 Out IP (tos 0x0, ttl 63, id 13956, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xb5eb (correct), ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979080 fvmb0 In  IP (tos 0x0, ttl 64, id 13957, offset 0, flags [DF], proto TCP (6), length 96)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], cksum 0x42b8 (correct), seq 533410516:533410560, ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 44
1793456396.979084 fvup0 Out IP (tos 0x0, ttl 63, id 13957, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], cksum 0x646f (correct), seq 533410516:533410560, ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 44
1793456396.979091 fvup0 In  IP (tos 0x0, ttl 64, id 20705, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0xb5be), ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979094 fvmb0 Out IP (tos 0x0, ttl 63, id 20705, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0x9407), ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979192 fvup0 In  IP (tos 0x0, ttl 64, id 20706, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], cksum 0x8648 (incorrect -> 0x5c68), seq 2413624815:2413624818, ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 3
1793456396.979199 fvmb0 Out IP (tos 0x0, ttl 63, id 20706, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], cksum 0xa7ff (incorrect -> 0x3ab1), seq 2413624815:2413624818, ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 3
1793456396.979248 fvmb0 In  IP (tos 0x0, ttl 64, id 13958, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x9405 (correct), ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979251 fvup0 Out IP (tos 0x0, ttl 63, id 13958, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xb5bc (correct), ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979345 fvmb0 In  IP (tos 0x0, ttl 64, id 13959, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533410560:533411788, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979348 fvup0 Out IP (tos 0x0, ttl 63, id 13959, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533410560:533411788, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979361 fvmb0 In  IP (tos 0x0, ttl 64, id 13960, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533411788:533414244, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979365 fvup0 Out IP (tos 0x0, ttl 63, id 13960, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533411788:533414244, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979370 fvup0 In  IP (tos 0x0, ttl 64, id 20707, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0xa755), ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979372 fvmb0 Out IP (tos 0x0, ttl 63, id 20707, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0x859e), ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979374 fvmb0 In  IP (tos 0x0, ttl 64, id 13962, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533414244:533416700, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979378 fvup0 Out IP (tos 0x0, ttl 63, id 13962, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533414244:533416700, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979378 fvup0 In  IP (tos 0x0, ttl 64, id 20708, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413624818:2413625824, ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979382 fvmb0 In  IP (tos 0x0, ttl 64, id 13964, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533416700:533417928, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979383 fvmb0 Out IP (tos 0x0, ttl 63, id 20708, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413624818:2413625824, ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979385 fvup0 Out IP (tos 0x0, ttl 63, id 13964, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533416700:533417928, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979388 fvmb0 In  IP (tos 0x0, ttl 64, id 13965, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533417928:533419156, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979391 fvup0 Out IP (tos 0x0, ttl 63, id 13965, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533417928:533419156, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979391 fvup0 In  IP (tos 0x0, ttl 64, id 20709, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x9506), ack 533417928, win 63, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979393 fvmb0 In  IP (tos 0x0, ttl 64, id 13966, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533419156:533420384, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979394 fvmb0 Out IP (tos 0x0, ttl 63, id 20709, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0x734f), ack 533417928, win 63, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979396 fvup0 Out IP (tos 0x0, ttl 63, id 13966, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533419156:533420384, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979399 fvmb0 In  IP (tos 0x0, ttl 64, id 13967, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533420384:533421612, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979401 fvup0 Out IP (tos 0x0, ttl 63, id 13967, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533420384:533421612, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979402 fvup0 In  IP (tos 0x0, ttl 64, id 20710, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413625824:2413626830, ack 533420384, win 61, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979404 fvmb0 In  IP (tos 0x0, ttl 64, id 13968, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533421612:533422840, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979404 fvmb0 Out IP (tos 0x0, ttl 63, id 20710, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413625824:2413626830, ack 533420384, win 61, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979406 fvup0 Out IP (tos 0x0, ttl 63, id 13968, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533421612:533422840, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979417 fvup0 In  IP (tos 0x0, ttl 64, id 20711, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413626830:2413627836, ack 533422840, win 59, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979420 fvmb0 Out IP (tos 0x0, ttl 63, id 20711, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413626830:2413627836, ack 533422840, win 59, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979425 fvmb0 In  IP (tos 0x0, ttl 64, id 13969, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533422840:533424068, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979429 fvup0 Out IP (tos 0x0, ttl 63, id 13969, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533422840:533424068, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979431 fvup0 In  IP (tos 0x0, ttl 64, id 20712, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413627836:2413628842, ack 533422840, win 60, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979434 fvmb0 Out IP (tos 0x0, ttl 63, id 20712, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413627836:2413628842, ack 533422840, win 60, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979435 fvmb0 In  IP (tos 0x0, ttl 64, id 13970, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533424068:533425296, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979438 fvup0 Out IP (tos 0x0, ttl 63, id 13970, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533424068:533425296, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979442 fvmb0 In  IP (tos 0x0, ttl 64, id 13971, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533425296:533427752, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979445 fvup0 Out IP (tos 0x0, ttl 63, id 13971, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533425296:533427752, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979445 fvup0 In  IP (tos 0x0, ttl 64, id 20713, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413628842:2413629848, ack 533425296, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979448 fvmb0 In  IP (tos 0x0, ttl 64, id 13973, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533427752:533428980, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979448 fvmb0 Out IP (tos 0x0, ttl 63, id 20713, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413628842:2413629848, ack 533425296, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979450 fvup0 Out IP (tos 0x0, ttl 63, id 13973, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533427752:533428980, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979454 fvup0 In  IP (tos 0x0, ttl 64, id 20714, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x5ef3), ack 533427752, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979455 fvmb0 In  IP (tos 0x0, ttl 64, id 13974, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533428980:533430208, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979456 fvmb0 Out IP (tos 0x0, ttl 63, id 20714, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0x3d3c), ack 533427752, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979458 fvup0 Out IP (tos 0x0, ttl 63, id 13974, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533428980:533430208, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979461 fvmb0 In  IP (tos 0x0, ttl 64, id 13975, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533430208:533431436, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979464 fvup0 Out IP (tos 0x0, ttl 63, id 13975, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533430208:533431436, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979465 fvup0 In  IP (tos 0x0, ttl 64, id 20715, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413629848:2413630854, ack 533430208, win 57, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979467 fvmb0 Out IP (tos 0x0, ttl 63, id 20715, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413629848:2413630854, ack 533430208, win 57, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979469 fvmb0 In  IP (tos 0x0, ttl 64, id 13976, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533431436:533432664, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979471 fvup0 Out IP (tos 0x0, ttl 63, id 13976, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533431436:533432664, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979475 fvup0 In  IP (tos 0x0, ttl 64, id 20716, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413630854:2413631860, ack 533431436, win 56, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979475 fvmb0 In  IP (tos 0x0, ttl 64, id 13977, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533432664:533433892, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979477 fvup0 Out IP (tos 0x0, ttl 63, id 13977, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533432664:533433892, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979478 fvmb0 Out IP (tos 0x0, ttl 63, id 20716, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413630854:2413631860, ack 533431436, win 56, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979481 fvmb0 In  IP (tos 0x0, ttl 64, id 13978, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533433892:533435120, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979485 fvup0 Out IP (tos 0x0, ttl 63, id 13978, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533433892:533435120, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979485 fvup0 In  IP (tos 0x0, ttl 64, id 20717, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x3f1f), ack 533433892, win 54, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979488 fvmb0 Out IP (tos 0x0, ttl 63, id 20717, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0x1d68), ack 533433892, win 54, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979493 fvmb0 In  IP (tos 0x0, ttl 64, id 13979, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533435120:533436348, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979495 fvup0 In  IP (tos 0x0, ttl 64, id 20718, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413631860:2413632866, ack 533435120, win 53, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979497 fvup0 Out IP (tos 0x0, ttl 63, id 13979, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533435120:533436348, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979497 fvmb0 Out IP (tos 0x0, ttl 63, id 20718, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413631860:2413632866, ack 533435120, win 53, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979501 fvmb0 In  IP (tos 0x0, ttl 64, id 13980, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533436348:533437576, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979501 fvmb0 In  IP (tos 0x0, ttl 64, id 13981, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533437576:533438804, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979504 fvup0 Out IP (tos 0x0, ttl 63, id 13980, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533436348:533437576, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979505 fvup0 In  IP (tos 0x0, ttl 64, id 20719, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413632866:2413633872, ack 533436348, win 52, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979506 fvup0 Out IP (tos 0x0, ttl 63, id 13981, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533437576:533438804, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979507 fvmb0 Out IP (tos 0x0, ttl 63, id 20719, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413632866:2413633872, ack 533436348, win 52, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979510 fvmb0 In  IP (tos 0x0, ttl 64, id 13982, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533438804:533441260, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979512 fvup0 Out IP (tos 0x0, ttl 63, id 13982, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533438804:533441260, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979522 fvmb0 In  IP (tos 0x0, ttl 64, id 13984, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533441260:533443716, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979526 fvup0 Out IP (tos 0x0, ttl 63, id 13984, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533441260:533443716, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979528 fvup0 In  IP (tos 0x0, ttl 64, id 20720, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x1a80), ack 533441260, win 49, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979528 fvmb0 In  IP (tos 0x0, ttl 64, id 13986, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533443716:533444944, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979530 fvmb0 Out IP (tos 0x0, ttl 63, id 20720, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0xf8c8), ack 533441260, win 49, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979531 fvup0 Out IP (tos 0x0, ttl 63, id 13986, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533443716:533444944, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979534 fvmb0 In  IP (tos 0x0, ttl 64, id 13987, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533444944:533446172, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979536 fvup0 Out IP (tos 0x0, ttl 63, id 13987, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533444944:533446172, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979538 fvup0 In  IP (tos 0x0, ttl 64, id 20721, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413633872:2413634878, ack 533444944, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979540 fvmb0 Out IP (tos 0x0, ttl 63, id 20721, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413633872:2413634878, ack 533444944, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979544 fvmb0 In  IP (tos 0x0, ttl 64, id 13988, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533446172:533447400, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979545 fvmb0 In  IP (tos 0x0, ttl 64, id 13989, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xf078 (correct), ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979548 fvup0 In  IP (tos 0x0, ttl 64, id 20722, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413634878:2413635884, ack 533446172, win 45, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979548 fvup0 Out IP (tos 0x0, ttl 63, id 13988, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533446172:533447400, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979550 fvmb0 Out IP (tos 0x0, ttl 63, id 20722, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413634878:2413635884, ack 533446172, win 45, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979550 fvup0 Out IP (tos 0x0, ttl 63, id 13989, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1230 (correct), ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979556 fvmb0 In  IP (tos 0x0, ttl 64, id 13990, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533447400:533449856, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979559 fvup0 Out IP (tos 0x0, ttl 63, id 13990, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533447400:533449856, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979561 fvup0 In  IP (tos 0x0, ttl 64, id 20723, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413635884:2413636890, ack 533447400, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979561 fvmb0 In  IP (tos 0x0, ttl 64, id 13992, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533449856:533451084, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979563 fvmb0 Out IP (tos 0x0, ttl 63, id 20723, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413635884:2413636890, ack 533447400, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979564 fvup0 Out IP (tos 0x0, ttl 63, id 13992, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533449856:533451084, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979569 fvmb0 In  IP (tos 0x0, ttl 64, id 13993, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533451084:533452312, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979570 fvup0 In  IP (tos 0x0, ttl 64, id 20724, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413636890:2413637896, ack 533451084, win 43, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979571 fvup0 Out IP (tos 0x0, ttl 63, id 13993, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533451084:533452312, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979573 fvmb0 Out IP (tos 0x0, ttl 63, id 20724, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413636890:2413637896, ack 533451084, win 43, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979574 fvmb0 In  IP (tos 0x0, ttl 64, id 13994, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533452312:533453540, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979577 fvup0 Out IP (tos 0x0, ttl 63, id 13994, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533452312:533453540, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979582 fvmb0 In  IP (tos 0x0, ttl 64, id 13995, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533453540:533454768, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979584 fvup0 In  IP (tos 0x0, ttl 64, id 20725, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413637896:2413638902, ack 533453540, win 41, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979584 fvup0 Out IP (tos 0x0, ttl 63, id 13995, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533453540:533454768, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979586 fvmb0 Out IP (tos 0x0, ttl 63, id 20725, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413637896:2413638902, ack 533453540, win 41, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979589 fvmb0 In  IP (tos 0x0, ttl 64, id 13996, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533454768:533455996, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979591 fvup0 Out IP (tos 0x0, ttl 63, id 13996, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533454768:533455996, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979594 fvup0 In  IP (tos 0x0, ttl 64, id 20726, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413638902:2413639908, ack 533454768, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979596 fvmb0 In  IP (tos 0x0, ttl 64, id 13997, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533455996:533457224, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979596 fvmb0 Out IP (tos 0x0, ttl 63, id 20726, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413638902:2413639908, ack 533454768, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979598 fvup0 Out IP (tos 0x0, ttl 63, id 13997, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533455996:533457224, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979601 fvmb0 In  IP (tos 0x0, ttl 64, id 13998, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc23d (correct), ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979603 fvup0 In  IP (tos 0x0, ttl 64, id 20727, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0xc498), ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979603 fvup0 Out IP (tos 0x0, ttl 63, id 13998, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xe3f4 (correct), ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979606 fvmb0 Out IP (tos 0x0, ttl 63, id 20727, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0xa2e1), ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979608 fvmb0 In  IP (tos 0x0, ttl 64, id 13999, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533457224:533458452, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979611 fvup0 Out IP (tos 0x0, ttl 63, id 13999, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533457224:533458452, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979613 fvup0 In  IP (tos 0x0, ttl 64, id 20728, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413639908:2413640914, ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979614 fvmb0 In  IP (tos 0x0, ttl 64, id 14000, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533458452:533459680, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979616 fvmb0 Out IP (tos 0x0, ttl 63, id 20728, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413639908:2413640914, ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979616 fvup0 Out IP (tos 0x0, ttl 63, id 14000, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533458452:533459680, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979619 fvmb0 In  IP (tos 0x0, ttl 64, id 14001, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533459680:533460908, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979621 fvup0 Out IP (tos 0x0, ttl 63, id 14001, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533459680:533460908, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979624 fvmb0 In  IP (tos 0x0, ttl 64, id 14002, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533460908:533462136, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979627 fvup0 Out IP (tos 0x0, ttl 63, id 14002, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533460908:533462136, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979631 fvup0 In  IP (tos 0x0, ttl 64, id 20729, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413640914:2413641920, ack 533460908, win 37, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979634 fvmb0 Out IP (tos 0x0, ttl 63, id 20729, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413640914:2413641920, ack 533460908, win 37, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979639 fvmb0 In  IP (tos 0x0, ttl 64, id 14003, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533462136:533464592, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979642 fvup0 Out IP (tos 0x0, ttl 63, id 14003, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533462136:533464592, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979645 fvup0 In  IP (tos 0x0, ttl 64, id 20730, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413641920:2413642926, ack 533462136, win 37, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979648 fvmb0 Out IP (tos 0x0, ttl 63, id 20730, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413641920:2413642926, ack 533462136, win 37, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979648 fvmb0 In  IP (tos 0x0, ttl 64, id 14005, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533464592:533465820, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979651 fvup0 Out IP (tos 0x0, ttl 63, id 14005, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533464592:533465820, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979654 fvmb0 In  IP (tos 0x0, ttl 64, id 14006, offset 0, flags [DF], proto TCP (6), length 2508)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533465820:533468276, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979656 fvup0 In  IP (tos 0x0, ttl 64, id 20731, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413642926:2413643932, ack 533465820, win 34, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979657 fvup0 Out IP (tos 0x0, ttl 63, id 14006, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533465820:533468276, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979658 fvmb0 Out IP (tos 0x0, ttl 63, id 20731, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413642926:2413643932, ack 533465820, win 34, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979663 fvmb0 In  IP (tos 0x0, ttl 64, id 14008, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533468276:533469504, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979664 fvmb0 In  IP (tos 0x0, ttl 64, id 14009, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x8a6a (correct), ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979667 fvup0 Out IP (tos 0x0, ttl 63, id 14008, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533468276:533469504, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979669 fvup0 In  IP (tos 0x0, ttl 64, id 20732, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x89bb), ack 533468276, win 32, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979669 fvup0 Out IP (tos 0x0, ttl 63, id 14009, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xac21 (correct), ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979671 fvmb0 Out IP (tos 0x0, ttl 63, id 20732, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0x6804), ack 533468276, win 32, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979672 fvmb0 In  IP (tos 0x0, ttl 64, id 14010, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533469504:533470732, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979675 fvup0 Out IP (tos 0x0, ttl 63, id 14010, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533469504:533470732, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979677 fvmb0 In  IP (tos 0x0, ttl 64, id 14011, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533470732:533471960, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979678 fvup0 In  IP (tos 0x0, ttl 64, id 20733, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413643932:2413644938, ack 533469504, win 31, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979680 fvup0 Out IP (tos 0x0, ttl 63, id 14011, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533470732:533471960, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979681 fvmb0 Out IP (tos 0x0, ttl 63, id 20733, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413643932:2413644938, ack 533469504, win 31, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979683 fvmb0 In  IP (tos 0x0, ttl 64, id 14012, offset 0, flags [DF], proto TCP (6), length 1280)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], seq 533471960:533473188, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979686 fvup0 Out IP (tos 0x0, ttl 63, id 14012, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533471960:533473188, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979687 fvup0 In  IP (tos 0x0, ttl 64, id 20734, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413644938:2413645944, ack 533471960, win 29, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979688 fvmb0 In  IP (tos 0x0, ttl 64, id 14013, offset 0, flags [DF], proto TCP (6), length 1488)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [P.], seq 533473188:533474624, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1436
1793456396.979690 fvmb0 Out IP (tos 0x0, ttl 63, id 20734, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413644938:2413645944, ack 533471960, win 29, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979691 fvup0 Out IP (tos 0x0, ttl 63, id 14013, offset 0, flags [DF], proto TCP (6), length 1488)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533473188:533474624, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1436
1793456396.979694 fvmb0 In  IP (tos 0x0, ttl 64, id 14015, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x6e8f (correct), ack 2413635884, win 58, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979696 fvup0 In  IP (tos 0x0, ttl 64, id 20735, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x6918), ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979697 fvup0 Out IP (tos 0x0, ttl 63, id 14015, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x9046 (correct), ack 2413635884, win 58, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979698 fvmb0 Out IP (tos 0x0, ttl 63, id 20735, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [.], cksum 0xa7fc (incorrect -> 0x4761), ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979699 fvmb0 In  IP (tos 0x0, ttl 64, id 14016, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x66b4 (correct), ack 2413637896, win 57, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979699 fvmb0 In  IP (tos 0x0, ttl 64, id 14017, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x5ed9 (correct), ack 2413639908, win 56, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979702 fvup0 Out IP (tos 0x0, ttl 63, id 14016, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x886b (correct), ack 2413637896, win 57, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979704 fvup0 Out IP (tos 0x0, ttl 63, id 14017, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x8090 (correct), ack 2413639908, win 56, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979705 fvup0 In  IP (tos 0x0, ttl 64, id 20736, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413645944:2413646950, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979707 fvmb0 Out IP (tos 0x0, ttl 63, id 20736, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413645944:2413646950, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979717 fvmb0 In  IP (tos 0x0, ttl 64, id 14018, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x56fd (correct), ack 2413641920, win 55, options [nop,nop,TS val 2675588078 ecr 2782928895], length 0
1793456396.979720 fvup0 Out IP (tos 0x0, ttl 63, id 14018, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x78b4 (correct), ack 2413641920, win 55, options [nop,nop,TS val 2675588078 ecr 2782928895], length 0
1793456396.979722 fvup0 In  IP (tos 0x0, ttl 64, id 20737, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413646950:2413647956, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979725 fvmb0 Out IP (tos 0x0, ttl 63, id 20737, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413646950:2413647956, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979730 fvmb0 In  IP (tos 0x0, ttl 64, id 14019, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x4747 (correct), ack 2413645944, win 52, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979733 fvup0 Out IP (tos 0x0, ttl 63, id 14019, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x68fe (correct), ack 2413645944, win 52, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979736 fvup0 In  IP (tos 0x0, ttl 64, id 20738, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413647956:2413648962, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979738 fvmb0 Out IP (tos 0x0, ttl 63, id 20738, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413647956:2413648962, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979748 fvup0 In  IP (tos 0x0, ttl 64, id 20739, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413648962:2413649968, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979750 fvmb0 Out IP (tos 0x0, ttl 63, id 20739, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413648962:2413649968, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979758 fvup0 In  IP (tos 0x0, ttl 64, id 20740, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413649968:2413650974, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979759 fvmb0 In  IP (tos 0x0, ttl 64, id 14020, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x3f6c (correct), ack 2413647956, win 51, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979761 fvmb0 Out IP (tos 0x0, ttl 63, id 20740, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413649968:2413650974, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979761 fvup0 Out IP (tos 0x0, ttl 63, id 14020, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x6123 (correct), ack 2413647956, win 51, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979768 fvup0 In  IP (tos 0x0, ttl 64, id 20741, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413650974:2413651980, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979770 fvmb0 Out IP (tos 0x0, ttl 63, id 20741, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413650974:2413651980, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979778 fvup0 In  IP (tos 0x0, ttl 64, id 20742, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413651980:2413652986, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979781 fvmb0 Out IP (tos 0x0, ttl 63, id 20742, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413651980:2413652986, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979788 fvup0 In  IP (tos 0x0, ttl 64, id 20743, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413652986:2413653992, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979791 fvmb0 Out IP (tos 0x0, ttl 63, id 20743, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413652986:2413653992, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979795 fvmb0 In  IP (tos 0x0, ttl 64, id 14021, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x33a4 (correct), ack 2413650974, win 49, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979799 fvup0 Out IP (tos 0x0, ttl 63, id 14021, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x555b (correct), ack 2413650974, win 49, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979804 fvup0 In  IP (tos 0x0, ttl 64, id 20744, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413653992:2413654998, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979807 fvmb0 Out IP (tos 0x0, ttl 63, id 20744, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413653992:2413654998, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979818 fvup0 In  IP (tos 0x0, ttl 64, id 20745, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413654998:2413656004, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979822 fvmb0 In  IP (tos 0x0, ttl 64, id 14022, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x2bc9 (correct), ack 2413652986, win 48, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979825 fvmb0 Out IP (tos 0x0, ttl 63, id 20745, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413654998:2413656004, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979825 fvup0 Out IP (tos 0x0, ttl 63, id 14022, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x4d80 (correct), ack 2413652986, win 48, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979834 fvup0 In  IP (tos 0x0, ttl 64, id 20746, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413656004:2413657010, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979836 fvmb0 Out IP (tos 0x0, ttl 63, id 20746, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413656004:2413657010, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979840 fvmb0 In  IP (tos 0x0, ttl 64, id 14023, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x23ee (correct), ack 2413654998, win 47, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979842 fvup0 Out IP (tos 0x0, ttl 63, id 14023, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x45a5 (correct), ack 2413654998, win 47, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979847 fvup0 In  IP (tos 0x0, ttl 64, id 20747, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413657010:2413658016, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979849 fvmb0 Out IP (tos 0x0, ttl 63, id 20747, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413657010:2413658016, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979859 fvup0 In  IP (tos 0x0, ttl 64, id 20748, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413658016:2413659022, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979861 fvmb0 Out IP (tos 0x0, ttl 63, id 20748, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413658016:2413659022, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979868 fvup0 In  IP (tos 0x0, ttl 64, id 20749, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413659022:2413660028, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979871 fvmb0 Out IP (tos 0x0, ttl 63, id 20749, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413659022:2413660028, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979873 fvmb0 In  IP (tos 0x0, ttl 64, id 14024, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1c13 (correct), ack 2413657010, win 46, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979875 fvup0 Out IP (tos 0x0, ttl 63, id 14024, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x3dca (correct), ack 2413657010, win 46, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979878 fvup0 In  IP (tos 0x0, ttl 64, id 20750, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413660028:2413661034, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979880 fvmb0 Out IP (tos 0x0, ttl 63, id 20750, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413660028:2413661034, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979891 fvmb0 In  IP (tos 0x0, ttl 64, id 14025, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1438 (correct), ack 2413659022, win 45, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979892 fvup0 In  IP (tos 0x0, ttl 64, id 20751, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413661034:2413662040, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979894 fvup0 Out IP (tos 0x0, ttl 63, id 14025, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x35ef (correct), ack 2413659022, win 45, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979894 fvmb0 Out IP (tos 0x0, ttl 63, id 20751, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413661034:2413662040, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979903 fvup0 In  IP (tos 0x0, ttl 64, id 20752, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413662040:2413663046, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979906 fvmb0 Out IP (tos 0x0, ttl 63, id 20752, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413662040:2413663046, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979915 fvup0 In  IP (tos 0x0, ttl 64, id 20753, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413663046:2413664052, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979917 fvmb0 Out IP (tos 0x0, ttl 63, id 20753, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413663046:2413664052, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979918 fvmb0 In  IP (tos 0x0, ttl 64, id 14026, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x0c5d (correct), ack 2413661034, win 44, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979921 fvup0 Out IP (tos 0x0, ttl 63, id 14026, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x2e14 (correct), ack 2413661034, win 44, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979928 fvup0 In  IP (tos 0x0, ttl 64, id 20754, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413664052:2413665058, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979931 fvmb0 Out IP (tos 0x0, ttl 63, id 20754, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413664052:2413665058, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979941 fvup0 In  IP (tos 0x0, ttl 64, id 20755, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413665058:2413666064, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979941 fvmb0 In  IP (tos 0x0, ttl 64, id 14027, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x0482 (correct), ack 2413663046, win 43, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979943 fvmb0 Out IP (tos 0x0, ttl 63, id 20755, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413665058:2413666064, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979943 fvup0 Out IP (tos 0x0, ttl 63, id 14027, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x2639 (correct), ack 2413663046, win 43, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979952 fvup0 In  IP (tos 0x0, ttl 64, id 20756, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413666064:2413667070, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979954 fvmb0 Out IP (tos 0x0, ttl 63, id 20756, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413666064:2413667070, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979963 fvup0 In  IP (tos 0x0, ttl 64, id 20757, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413667070:2413668076, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979965 fvmb0 Out IP (tos 0x0, ttl 63, id 20757, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413667070:2413668076, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979972 fvup0 In  IP (tos 0x0, ttl 64, id 20758, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413668076:2413669082, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979974 fvmb0 Out IP (tos 0x0, ttl 63, id 20758, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413668076:2413669082, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979984 fvup0 In  IP (tos 0x0, ttl 64, id 20759, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413669082:2413670088, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979985 fvmb0 In  IP (tos 0x0, ttl 64, id 14028, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xfca6 (correct), ack 2413665058, win 42, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979986 fvmb0 Out IP (tos 0x0, ttl 63, id 20759, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413669082:2413670088, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979989 fvup0 Out IP (tos 0x0, ttl 63, id 14028, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1e5e (correct), ack 2413665058, win 42, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979992 fvmb0 In  IP (tos 0x0, ttl 64, id 14029, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xf4cb (correct), ack 2413667070, win 41, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979995 fvup0 Out IP (tos 0x0, ttl 63, id 14029, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1683 (correct), ack 2413667070, win 41, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979997 fvup0 In  IP (tos 0x0, ttl 64, id 20760, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413670088:2413671094, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979999 fvmb0 Out IP (tos 0x0, ttl 63, id 20760, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413670088:2413671094, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980009 fvup0 In  IP (tos 0x0, ttl 64, id 20761, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413671094:2413672100, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980011 fvmb0 Out IP (tos 0x0, ttl 63, id 20761, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413671094:2413672100, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980011 fvmb0 In  IP (tos 0x0, ttl 64, id 14030, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xecf0 (correct), ack 2413669082, win 40, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980014 fvup0 Out IP (tos 0x0, ttl 63, id 14030, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x0ea8 (correct), ack 2413669082, win 40, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980019 fvup0 In  IP (tos 0x0, ttl 64, id 20762, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413672100:2413673106, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980021 fvmb0 Out IP (tos 0x0, ttl 63, id 20762, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413672100:2413673106, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980026 fvmb0 In  IP (tos 0x0, ttl 64, id 14031, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xe515 (correct), ack 2413671094, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980028 fvup0 Out IP (tos 0x0, ttl 63, id 14031, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x06cd (correct), ack 2413671094, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980030 fvup0 In  IP (tos 0x0, ttl 64, id 20763, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413673106:2413674112, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980033 fvmb0 Out IP (tos 0x0, ttl 63, id 20763, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413673106:2413674112, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980041 fvup0 In  IP (tos 0x0, ttl 64, id 20764, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413674112:2413675118, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980044 fvmb0 Out IP (tos 0x0, ttl 63, id 20764, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413674112:2413675118, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980049 fvmb0 In  IP (tos 0x0, ttl 64, id 14032, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xdd39 (correct), ack 2413673106, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980052 fvup0 Out IP (tos 0x0, ttl 63, id 14032, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xfef0 (correct), ack 2413673106, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980053 fvup0 In  IP (tos 0x0, ttl 64, id 20765, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413675118:2413676124, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980055 fvmb0 Out IP (tos 0x0, ttl 63, id 20765, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413675118:2413676124, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980063 fvup0 In  IP (tos 0x0, ttl 64, id 20766, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413676124:2413677130, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980065 fvmb0 Out IP (tos 0x0, ttl 63, id 20766, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413676124:2413677130, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980073 fvmb0 In  IP (tos 0x0, ttl 64, id 14033, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xd55d (correct), ack 2413675118, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980076 fvup0 Out IP (tos 0x0, ttl 63, id 14033, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xf714 (correct), ack 2413675118, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980078 fvup0 In  IP (tos 0x0, ttl 64, id 20767, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413677130:2413678136, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980080 fvmb0 Out IP (tos 0x0, ttl 63, id 20767, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413677130:2413678136, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980092 fvup0 In  IP (tos 0x0, ttl 64, id 20768, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413678136:2413679142, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980094 fvmb0 In  IP (tos 0x0, ttl 64, id 14034, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xcd81 (correct), ack 2413677130, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980095 fvmb0 Out IP (tos 0x0, ttl 63, id 20768, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413678136:2413679142, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980097 fvup0 Out IP (tos 0x0, ttl 63, id 14034, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xef38 (correct), ack 2413677130, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980106 fvup0 In  IP (tos 0x0, ttl 64, id 20769, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413679142:2413680148, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980109 fvmb0 Out IP (tos 0x0, ttl 63, id 20769, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413679142:2413680148, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980117 fvup0 In  IP (tos 0x0, ttl 64, id 20770, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413680148:2413681154, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980120 fvmb0 Out IP (tos 0x0, ttl 63, id 20770, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413680148:2413681154, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980128 fvup0 In  IP (tos 0x0, ttl 64, id 20771, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413681154:2413682160, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980130 fvmb0 In  IP (tos 0x0, ttl 64, id 14035, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc5a5 (correct), ack 2413679142, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980130 fvmb0 Out IP (tos 0x0, ttl 63, id 20771, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413681154:2413682160, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980133 fvup0 Out IP (tos 0x0, ttl 63, id 14035, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xe75c (correct), ack 2413679142, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980140 fvup0 In  IP (tos 0x0, ttl 64, id 20772, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413682160:2413683166, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980142 fvmb0 Out IP (tos 0x0, ttl 63, id 20772, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413682160:2413683166, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980151 fvup0 In  IP (tos 0x0, ttl 64, id 20773, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413683166:2413684172, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980154 fvmb0 Out IP (tos 0x0, ttl 63, id 20773, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413683166:2413684172, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980156 fvmb0 In  IP (tos 0x0, ttl 64, id 14036, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xbdc9 (correct), ack 2413681154, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980158 fvup0 Out IP (tos 0x0, ttl 63, id 14036, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xdf80 (correct), ack 2413681154, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980162 fvup0 In  IP (tos 0x0, ttl 64, id 20774, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413684172:2413685178, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980164 fvmb0 Out IP (tos 0x0, ttl 63, id 20774, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413684172:2413685178, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980173 fvup0 In  IP (tos 0x0, ttl 64, id 20775, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413685178:2413686184, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980175 fvmb0 In  IP (tos 0x0, ttl 64, id 14037, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xb5ed (correct), ack 2413683166, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980176 fvmb0 Out IP (tos 0x0, ttl 63, id 20775, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413685178:2413686184, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980178 fvup0 Out IP (tos 0x0, ttl 63, id 14037, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xd7a4 (correct), ack 2413683166, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980183 fvup0 In  IP (tos 0x0, ttl 64, id 20776, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413686184:2413687190, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980186 fvmb0 Out IP (tos 0x0, ttl 63, id 20776, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413686184:2413687190, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980199 fvup0 In  IP (tos 0x0, ttl 64, id 20777, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413687190:2413688196, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980201 fvmb0 In  IP (tos 0x0, ttl 64, id 14038, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xae11 (correct), ack 2413685178, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980202 fvmb0 Out IP (tos 0x0, ttl 63, id 20777, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413687190:2413688196, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980203 fvup0 Out IP (tos 0x0, ttl 63, id 14038, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xcfc8 (correct), ack 2413685178, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980210 fvup0 In  IP (tos 0x0, ttl 64, id 20778, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413688196:2413689202, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980212 fvmb0 Out IP (tos 0x0, ttl 63, id 20778, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [P.], seq 2413688196:2413689202, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980216 fvmb0 In  IP (tos 0x0, ttl 64, id 14039, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0xa635 (correct), ack 2413687190, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980219 fvup0 Out IP (tos 0x0, ttl 63, id 14039, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc7ec (correct), ack 2413687190, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980241 fvmb0 In  IP (tos 0x0, ttl 64, id 14040, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x9e59 (correct), ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980243 fvup0 Out IP (tos 0x0, ttl 63, id 14040, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc010 (correct), ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980430 fvmb0 In  IP (tos 0x0, ttl 64, id 14041, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [F.], cksum 0x9e58 (correct), seq 533474624, ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980433 fvup0 Out IP (tos 0x0, ttl 63, id 14041, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [F.], cksum 0xc00f (correct), seq 533474624, ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980515 fvup0 In  IP (tos 0x0, ttl 64, id 20779, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [F.], cksum 0x8645 (incorrect -> 0xbff2), seq 2413689202, ack 533474625, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 0
1793456396.980524 fvmb0 Out IP (tos 0x0, ttl 63, id 20779, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 100.83.128.120.37850: Flags [F.], cksum 0xa7fc (incorrect -> 0x9e3b), seq 2413689202, ack 533474625, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 0
1793456396.980564 fvmb0 In  IP (tos 0x0, ttl 64, id 14042, offset 0, flags [DF], proto TCP (6), length 52)
    100.83.128.120.37850 > 192.0.3.10.9446: Flags [.], cksum 0x9e57 (correct), ack 2413689203, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980567 fvup0 Out IP (tos 0x0, ttl 63, id 14042, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc00e (correct), ack 2413689203, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0

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
[{"ifindex":1,"ifname":"lo","flags":["LOOPBACK","UP","LOWER_UP"],"mtu":65536,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"loopback","address":"00:00:00:00:00:00","broadcast":"00:00:00:00:00:00"},{"ifindex":2,"link_index":4318,"ifname":"eth0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"5a:8d:b5:97:e7:d6","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":3,"link_index":4319,"ifname":"fvup0","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"htb","operstate":"UP","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"5a:bc:89:da:9c:4a","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":4,"link_index":4322,"ifname":"eth2","flags":["BROADCAST","MULTICAST","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UP","linkmode":"DEFAULT","group":"default","link_type":"ether","address":"46:08:5f:12:52:86","broadcast":"ff:ff:ff:ff:ff:ff","link_netnsid":0},{"ifindex":5,"ifname":"fvfixture","flags":["BROADCAST","NOARP","UP","LOWER_UP"],"mtu":1500,"qdisc":"noqueue","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"ether","address":"82:09:3c:35:ad:f2","broadcast":"ff:ff:ff:ff:ff:ff"},{"ifindex":6,"ifname":"fvmb0","flags":["POINTOPOINT","NOARP","UP","LOWER_UP"],"mtu":1280,"qdisc":"htb","operstate":"UNKNOWN","linkmode":"DEFAULT","group":"default","txqlen":1000,"link_type":"none"}]
[{"dst":"default","gateway":"172.28.30.1","dev":"eth0","flags":[]},{"dst":"100.83.0.0/16","dev":"fvmb0","protocol":"kernel","scope":"link","prefsrc":"100.83.207.231","flags":[]},{"dst":"172.28.30.0/24","dev":"eth0","protocol":"kernel","scope":"link","prefsrc":"172.28.30.40","flags":[]},{"dst":"192.0.3.0/24","dev":"fvup0","protocol":"kernel","scope":"link","prefsrc":"192.0.3.20","flags":[]}]
[{"dst":"2001:db8:30::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fd00::30","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"eth2","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"fe80::/64","dev":"fvfixture","protocol":"kernel","metric":256,"flags":[],"pref":"medium"},{"dst":"default","gateway":"2001:db8:30::1","dev":"eth2","metric":1024,"flags":[],"pref":"medium"}]
State Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
ESTAB 0      0          127.0.0.1:8080     127.0.0.1:32844
	 cubic wscale:10,10 rto:211 rtt:10.495/16.862 ato:40 mss:32768 pmtu:65535 rcvmss:547 advmss:65483 cwnd:10 bytes_sent:40284 bytes_acked:40284 bytes_received:20581 segs_out:106 segs_in:141 data_segs_out:103 data_segs_in:103 send 249779895bps lastsnd:4178 lastrcv:4178 lastack:4137 pacing_rate 499535984bps delivery_rate 52428800000bps delivered:104 app_limited busy:750ms rcv_space:47104 rcv_ssthresh:47104 minrtt:0.005 snd_wnd:65536
ESTAB 0      0       172.28.30.40:58744 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.36/0.338 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:5353 bytes_acked:5354 bytes_received:11779 segs_out:116 segs_in:87 data_segs_out:47 data_segs_in:81 send 321777778bps lastsnd:820 lastrcv:820 lastack:820 pacing_rate 641995144bps delivery_rate 2896000000bps delivered:48 app_limited busy:53ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58756 172.28.30.21:443
	 cubic wscale:10,10 rto:201 rtt:0.177/0.132 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:15885 bytes_acked:15886 bytes_received:18880 segs_out:135 segs_in:159 data_segs_out:69 data_segs_in:148 send 654463277bps lastsnd:5630 lastrcv:5630 lastack:5630 pacing_rate 1306159264bps delivery_rate 2896000000bps delivered:70 app_limited busy:36ms rcv_space:14480 rcv_ssthresh:85803 minrtt:0.008 snd_wnd:79872
ESTAB 0      0       172.28.30.40:58772 172.28.30.21:443
	 cubic wscale:10,10 rto:209 rtt:8.888/14.475 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:3234 bytes_acked:3235 bytes_received:4755 segs_out:39 segs_in:40 data_segs_out:20 data_segs_in:26 send 13033303bps lastsnd:5632 lastrcv:5631 lastack:5631 pacing_rate 26064400bps delivery_rate 2106181816bps delivered:21 app_limited busy:168ms rcv_space:14480 rcv_ssthresh:85754 minrtt:0.007 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:60996
	 cubic wscale:10,10 rto:206 rtt:5.095/10.17 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:3152 bytes_acked:3152 bytes_received:2755 segs_out:4 segs_in:7 data_segs_out:2 data_segs_in:3 send 683336605bps lastsnd:14303 lastrcv:14310 lastack:14262 pacing_rate 1366572624bps delivery_rate 34816000000bps delivered:3 app_limited busy:41ms rcv_space:65483 rcv_ssthresh:78451 minrtt:0.01 snd_wnd:87040
ESTAB 0      0       172.28.30.40:34032 172.28.30.30:9443
	 cubic wscale:10,10 rto:210 rtt:9.298/11.555 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:36841 bytes_acked:36842 bytes_received:30569 segs_out:94 segs_in:56 data_segs_out:63 data_segs_in:39 send 12458593bps lastsnd:4449 lastrcv:4203 lastack:4203 pacing_rate 24914840bps delivery_rate 2106181816bps delivered:64 app_limited busy:174ms rcv_rtt:6.5 rcv_space:14480 rcv_ssthresh:88160 minrtt:0.008 snd_wnd:79872
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:48150
	 cubic wscale:10,10 rto:201 rtt:0.51/0.983 ato:40 mss:45568 pmtu:65535 rcvmss:4118 advmss:65483 cwnd:10 bytes_sent:22467 bytes_acked:22467 bytes_received:29809 segs_out:37 segs_in:60 data_segs_out:30 data_segs_in:30 send 7147921569bps lastsnd:4204 lastrcv:4451 lastack:4204 pacing_rate 14285339208bps delivery_rate 182272000000bps delivered:31 app_limited busy:94ms rcv_space:65483 rcv_ssthresh:105032 minrtt:0.002 snd_wnd:91136
ESTAB 0      0          127.0.0.1:9444     127.0.0.1:60980
	 cubic wscale:10,10 rto:206 rtt:5.055/10.086 ato:40 mss:43520 pmtu:65535 rcvmss:1495 advmss:65483 cwnd:10 bytes_sent:2832 bytes_acked:2832 bytes_received:4655 segs_out:4 segs_in:8 data_segs_out:2 data_segs_in:4 send 688743818bps lastsnd:14491 lastrcv:14516 lastack:14450 pacing_rate 1377385456bps delivery_rate 29013333328bps delivered:3 app_limited busy:41ms rcv_rtt:1 rcv_space:65483 rcv_ssthresh:78343 minrtt:0.012 snd_wnd:87040
ESTAB 0      0          127.0.0.1:32844    127.0.0.1:8080
	 cubic wscale:10,10 rto:201 rtt:0.751/0.062 ato:40 mss:32768 pmtu:65535 rcvmss:848 advmss:65483 cwnd:10 bytes_sent:20581 bytes_acked:20582 bytes_received:40284 segs_out:141 segs_in:107 data_segs_out:103 data_segs_in:103 send 3490599201bps lastsnd:4188 lastrcv:4188 lastack:4188 pacing_rate 6981198400bps delivery_rate 26214400000bps delivered:104 app_limited busy:127ms rcv_rtt:1 rcv_space:60928 rcv_ssthresh:61440 minrtt:0.01 snd_wnd:65536
ESTAB 0      0          127.0.0.1:48150    127.0.0.1:9444
	 cubic wscale:10,10 rto:210 rtt:9.969/10.997 ato:40 mss:52736 pmtu:65535 rcvmss:2205 advmss:65483 cwnd:10 bytes_sent:29809 bytes_acked:29810 bytes_received:22467 segs_out:60 segs_in:38 data_segs_out:30 data_segs_in:30 send 423199920bps lastsnd:4451 lastrcv:4204 lastack:4204 pacing_rate 846368000bps delivery_rate 84377600000bps delivered:31 app_limited busy:168ms rcv_rtt:6 rcv_space:65495 rcv_ssthresh:90187 minrtt:0.005 snd_wnd:105472
ESTAB 0      0          127.0.0.1:60980    127.0.0.1:9444
	 cubic wscale:10,10 rto:204 rtt:3.23/6.412 ato:40 mss:39424 pmtu:65535 rcvmss:1998 advmss:65483 cwnd:10 bytes_sent:4655 bytes_acked:4656 bytes_received:2832 segs_out:8 segs_in:5 data_segs_out:4 data_segs_in:2 send 976445820bps lastsnd:14516 lastrcv:14491 lastack:14491 pacing_rate 1952664936bps delivery_rate 19712000000bps delivered:5 app_limited busy:26ms rcv_space:65495 rcv_ssthresh:86858 minrtt:0.006 snd_wnd:78848
ESTAB 0      0          127.0.0.1:60996    127.0.0.1:9444
	 cubic wscale:10,10 rto:201 rtt:0.041/0.053 ato:40 mss:39424 pmtu:65535 rcvmss:1996 advmss:65483 cwnd:10 bytes_sent:2755 bytes_acked:2756 bytes_received:3152 segs_out:7 segs_in:5 data_segs_out:3 data_segs_in:2 send 76924878049bps lastsnd:14313 lastrcv:14306 lastack:14306 pacing_rate 151085988016bps delivery_rate 24260923072bps delivered:4 app_limited rcv_space:65495 rcv_ssthresh:86642 minrtt:0.013 snd_wnd:78848

- matrix1280_capture_1_receipt: {"bytes":"70454","sha256":"dfe6b780fdef864a4c65fbc51c5be765bf2f1c1cec795fde7245d943efca06bd"}
- matrix1280_capture_2_window: {"launch_requested_at":"2026-10-31T14:19:55.755460793Z","process_stopped":true,"scope":"process bounds; exact first/last packet timestamps and record counts come from the pinned pcap decode","stop_observed_at":"2026-10-31T14:19:58.342201357Z","stop_requested_at":"2026-10-31T14:19:57.837171226Z"}
- matrix1280_capture_2_stats: tcpdump: data link type LINUX_SLL2
tcpdump: listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128 bytes
155 packets captured
155 packets received by filter
0 packets dropped by kernel

- matrix1280_capture_2_headers: reading from file /var/lib/vpn-node/e2e/mtu.pcap, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 128
Warning: interface names might be incorrect
1793456396.978926 eth0  In  IP (tos 0x0, ttl 63, id 13955, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [S], cksum 0xe012 (correct), seq 533410515, win 64480, options [mss 1240,sackOK,TS val 2675588077 ecr 0,nop,wscale 10], length 0
1793456396.978946 eth0  Out IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [S.], cksum 0x864d (incorrect -> 0x88d2), seq 2413624814, ack 533410516, win 65160, options [mss 1460,sackOK,TS val 2782928895 ecr 2675588077,nop,wscale 10], length 0
1793456396.979048 eth0  In  IP (tos 0x0, ttl 63, id 13956, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xb5eb (correct), ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979085 eth0  In  IP (tos 0x0, ttl 63, id 13957, offset 0, flags [DF], proto TCP (6), length 96)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], cksum 0x646f (correct), seq 533410516:533410560, ack 2413624815, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 44
1793456396.979089 eth0  Out IP (tos 0x0, ttl 64, id 20705, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0xb5be), ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979187 eth0  Out IP (tos 0x0, ttl 64, id 20706, offset 0, flags [DF], proto TCP (6), length 55)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], cksum 0x8648 (incorrect -> 0x5c68), seq 2413624815:2413624818, ack 533410560, win 64, options [nop,nop,TS val 2782928895 ecr 2675588077], length 3
1793456396.979251 eth0  In  IP (tos 0x0, ttl 63, id 13958, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xb5bc (correct), ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979349 eth0  In  IP (tos 0x0, ttl 63, id 13959, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533410560:533411788, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979366 eth0  In  IP (tos 0x0, ttl 63, id 13960, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533411788:533414244, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979369 eth0  Out IP (tos 0x0, ttl 64, id 20707, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0xa755), ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979376 eth0  Out IP (tos 0x0, ttl 64, id 20708, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413624818:2413625824, ack 533414244, win 66, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979380 eth0  In  IP (tos 0x0, ttl 63, id 13962, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533414244:533416700, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979386 eth0  In  IP (tos 0x0, ttl 63, id 13964, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533416700:533417928, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979390 eth0  Out IP (tos 0x0, ttl 64, id 20709, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x9506), ack 533417928, win 63, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979392 eth0  In  IP (tos 0x0, ttl 63, id 13965, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533417928:533419156, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979397 eth0  In  IP (tos 0x0, ttl 63, id 13966, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533419156:533420384, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979401 eth0  Out IP (tos 0x0, ttl 64, id 20710, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413625824:2413626830, ack 533420384, win 61, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979403 eth0  In  IP (tos 0x0, ttl 63, id 13967, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533420384:533421612, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979407 eth0  In  IP (tos 0x0, ttl 63, id 13968, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533421612:533422840, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979416 eth0  Out IP (tos 0x0, ttl 64, id 20711, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413626830:2413627836, ack 533422840, win 59, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979429 eth0  In  IP (tos 0x0, ttl 63, id 13969, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533422840:533424068, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979430 eth0  Out IP (tos 0x0, ttl 64, id 20712, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413627836:2413628842, ack 533422840, win 60, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979439 eth0  In  IP (tos 0x0, ttl 63, id 13970, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533424068:533425296, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979443 eth0  Out IP (tos 0x0, ttl 64, id 20713, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413628842:2413629848, ack 533425296, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979446 eth0  In  IP (tos 0x0, ttl 63, id 13971, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533425296:533427752, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979451 eth0  In  IP (tos 0x0, ttl 63, id 13973, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533427752:533428980, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979453 eth0  Out IP (tos 0x0, ttl 64, id 20714, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x5ef3), ack 533427752, win 58, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979459 eth0  In  IP (tos 0x0, ttl 63, id 13974, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533428980:533430208, ack 2413624818, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979463 eth0  Out IP (tos 0x0, ttl 64, id 20715, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413629848:2413630854, ack 533430208, win 57, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979465 eth0  In  IP (tos 0x0, ttl 63, id 13975, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533430208:533431436, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979473 eth0  In  IP (tos 0x0, ttl 63, id 13976, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533431436:533432664, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979474 eth0  Out IP (tos 0x0, ttl 64, id 20716, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413630854:2413631860, ack 533431436, win 56, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979479 eth0  In  IP (tos 0x0, ttl 63, id 13977, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533432664:533433892, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979484 eth0  Out IP (tos 0x0, ttl 64, id 20717, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x3f1f), ack 533433892, win 54, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979486 eth0  In  IP (tos 0x0, ttl 63, id 13978, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533433892:533435120, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979493 eth0  Out IP (tos 0x0, ttl 64, id 20718, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413631860:2413632866, ack 533435120, win 53, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979498 eth0  In  IP (tos 0x0, ttl 63, id 13979, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533435120:533436348, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979504 eth0  Out IP (tos 0x0, ttl 64, id 20719, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413632866:2413633872, ack 533436348, win 52, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979507 eth0  In  IP (tos 0x0, ttl 63, id 13980, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533436348:533437576, ack 2413625824, win 63, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979508 eth0  In  IP (tos 0x0, ttl 63, id 13981, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533437576:533438804, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979513 eth0  In  IP (tos 0x0, ttl 63, id 13982, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533438804:533441260, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979526 eth0  Out IP (tos 0x0, ttl 64, id 20720, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x1a80), ack 533441260, win 49, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979527 eth0  In  IP (tos 0x0, ttl 63, id 13984, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533441260:533443716, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979532 eth0  In  IP (tos 0x0, ttl 63, id 13986, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533443716:533444944, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979537 eth0  Out IP (tos 0x0, ttl 64, id 20721, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413633872:2413634878, ack 533444944, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979537 eth0  In  IP (tos 0x0, ttl 63, id 13987, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533444944:533446172, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979546 eth0  Out IP (tos 0x0, ttl 64, id 20722, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413634878:2413635884, ack 533446172, win 45, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979551 eth0  In  IP (tos 0x0, ttl 63, id 13988, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533446172:533447400, ack 2413627836, win 62, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979553 eth0  In  IP (tos 0x0, ttl 63, id 13989, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1230 (correct), ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979559 eth0  Out IP (tos 0x0, ttl 64, id 20723, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413635884:2413636890, ack 533447400, win 46, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979560 eth0  In  IP (tos 0x0, ttl 63, id 13990, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533447400:533449856, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979564 eth0  In  IP (tos 0x0, ttl 63, id 13992, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533449856:533451084, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979569 eth0  Out IP (tos 0x0, ttl 64, id 20724, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413636890:2413637896, ack 533451084, win 43, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979572 eth0  In  IP (tos 0x0, ttl 63, id 13993, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533451084:533452312, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979578 eth0  In  IP (tos 0x0, ttl 63, id 13994, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533452312:533453540, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979582 eth0  Out IP (tos 0x0, ttl 64, id 20725, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413637896:2413638902, ack 533453540, win 41, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979585 eth0  In  IP (tos 0x0, ttl 63, id 13995, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533453540:533454768, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979592 eth0  Out IP (tos 0x0, ttl 64, id 20726, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413638902:2413639908, ack 533454768, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979592 eth0  In  IP (tos 0x0, ttl 63, id 13996, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533454768:533455996, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979599 eth0  In  IP (tos 0x0, ttl 63, id 13997, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533455996:533457224, ack 2413629848, win 61, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979602 eth0  Out IP (tos 0x0, ttl 64, id 20727, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0xc498), ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 0
1793456396.979604 eth0  In  IP (tos 0x0, ttl 63, id 13998, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xe3f4 (correct), ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979612 eth0  Out IP (tos 0x0, ttl 64, id 20728, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413639908:2413640914, ack 533457224, win 40, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979612 eth0  In  IP (tos 0x0, ttl 63, id 13999, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533457224:533458452, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979617 eth0  In  IP (tos 0x0, ttl 63, id 14000, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533458452:533459680, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979622 eth0  In  IP (tos 0x0, ttl 63, id 14001, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533459680:533460908, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979627 eth0  Out IP (tos 0x0, ttl 64, id 20729, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413640914:2413641920, ack 533460908, win 37, options [nop,nop,TS val 2782928895 ecr 2675588077], length 1006
1793456396.979628 eth0  In  IP (tos 0x0, ttl 63, id 14002, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533460908:533462136, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979643 eth0  In  IP (tos 0x0, ttl 63, id 14003, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533462136:533464592, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979644 eth0  Out IP (tos 0x0, ttl 64, id 20730, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413641920:2413642926, ack 533462136, win 37, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979651 eth0  In  IP (tos 0x0, ttl 63, id 14005, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533464592:533465820, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979655 eth0  Out IP (tos 0x0, ttl 64, id 20731, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413642926:2413643932, ack 533465820, win 34, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979658 eth0  In  IP (tos 0x0, ttl 63, id 14006, offset 0, flags [DF], proto TCP (6), length 2508)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533465820:533468276, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 2456
1793456396.979667 eth0  Out IP (tos 0x0, ttl 64, id 20732, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x89bb), ack 533468276, win 32, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979670 eth0  In  IP (tos 0x0, ttl 63, id 14008, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533468276:533469504, ack 2413631860, win 60, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979671 eth0  In  IP (tos 0x0, ttl 63, id 14009, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xac21 (correct), ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979676 eth0  In  IP (tos 0x0, ttl 63, id 14010, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533469504:533470732, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979678 eth0  Out IP (tos 0x0, ttl 64, id 20733, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413643932:2413644938, ack 533469504, win 31, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979681 eth0  In  IP (tos 0x0, ttl 63, id 14011, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533470732:533471960, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979686 eth0  In  IP (tos 0x0, ttl 63, id 14012, offset 0, flags [DF], proto TCP (6), length 1280)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], seq 533471960:533473188, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1228
1793456396.979687 eth0  Out IP (tos 0x0, ttl 64, id 20734, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413644938:2413645944, ack 533471960, win 29, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979692 eth0  In  IP (tos 0x0, ttl 63, id 14013, offset 0, flags [DF], proto TCP (6), length 1488)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [P.], seq 533473188:533474624, ack 2413633872, win 59, options [nop,nop,TS val 2675588077 ecr 2782928895], length 1436
1793456396.979695 eth0  Out IP (tos 0x0, ttl 64, id 20735, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [.], cksum 0x8645 (incorrect -> 0x6918), ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 0
1793456396.979697 eth0  In  IP (tos 0x0, ttl 63, id 14015, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x9046 (correct), ack 2413635884, win 58, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979704 eth0  Out IP (tos 0x0, ttl 64, id 20736, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413645944:2413646950, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979705 eth0  In  IP (tos 0x0, ttl 63, id 14016, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x886b (correct), ack 2413637896, win 57, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979706 eth0  In  IP (tos 0x0, ttl 63, id 14017, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x8090 (correct), ack 2413639908, win 56, options [nop,nop,TS val 2675588077 ecr 2782928895], length 0
1793456396.979721 eth0  In  IP (tos 0x0, ttl 63, id 14018, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x78b4 (correct), ack 2413641920, win 55, options [nop,nop,TS val 2675588078 ecr 2782928895], length 0
1793456396.979721 eth0  Out IP (tos 0x0, ttl 64, id 20737, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413646950:2413647956, ack 533474624, win 27, options [nop,nop,TS val 2782928896 ecr 2675588077], length 1006
1793456396.979734 eth0  In  IP (tos 0x0, ttl 63, id 14019, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x68fe (correct), ack 2413645944, win 52, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979735 eth0  Out IP (tos 0x0, ttl 64, id 20738, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413647956:2413648962, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979746 eth0  Out IP (tos 0x0, ttl 64, id 20739, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413648962:2413649968, ack 533474624, win 30, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979757 eth0  Out IP (tos 0x0, ttl 64, id 20740, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413649968:2413650974, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979762 eth0  In  IP (tos 0x0, ttl 63, id 14020, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x6123 (correct), ack 2413647956, win 51, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979767 eth0  Out IP (tos 0x0, ttl 64, id 20741, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413650974:2413651980, ack 533474624, win 31, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979777 eth0  Out IP (tos 0x0, ttl 64, id 20742, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413651980:2413652986, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979787 eth0  Out IP (tos 0x0, ttl 64, id 20743, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413652986:2413653992, ack 533474624, win 32, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979800 eth0  In  IP (tos 0x0, ttl 63, id 14021, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x555b (correct), ack 2413650974, win 49, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979803 eth0  Out IP (tos 0x0, ttl 64, id 20744, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413653992:2413654998, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979817 eth0  Out IP (tos 0x0, ttl 64, id 20745, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413654998:2413656004, ack 533474624, win 36, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979826 eth0  In  IP (tos 0x0, ttl 63, id 14022, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x4d80 (correct), ack 2413652986, win 48, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979832 eth0  Out IP (tos 0x0, ttl 64, id 20746, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413656004:2413657010, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979844 eth0  In  IP (tos 0x0, ttl 63, id 14023, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x45a5 (correct), ack 2413654998, win 47, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979846 eth0  Out IP (tos 0x0, ttl 64, id 20747, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413657010:2413658016, ack 533474624, win 38, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979857 eth0  Out IP (tos 0x0, ttl 64, id 20748, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413658016:2413659022, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979867 eth0  Out IP (tos 0x0, ttl 64, id 20749, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413659022:2413660028, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979876 eth0  In  IP (tos 0x0, ttl 63, id 14024, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x3dca (correct), ack 2413657010, win 46, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979878 eth0  Out IP (tos 0x0, ttl 64, id 20750, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413660028:2413661034, ack 533474624, win 40, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979891 eth0  Out IP (tos 0x0, ttl 64, id 20751, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413661034:2413662040, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979895 eth0  In  IP (tos 0x0, ttl 63, id 14025, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x35ef (correct), ack 2413659022, win 45, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979902 eth0  Out IP (tos 0x0, ttl 64, id 20752, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413662040:2413663046, ack 533474624, win 42, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979913 eth0  Out IP (tos 0x0, ttl 64, id 20753, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413663046:2413664052, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979922 eth0  In  IP (tos 0x0, ttl 63, id 14026, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x2e14 (correct), ack 2413661034, win 44, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979927 eth0  Out IP (tos 0x0, ttl 64, id 20754, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413664052:2413665058, ack 533474624, win 44, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979940 eth0  Out IP (tos 0x0, ttl 64, id 20755, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413665058:2413666064, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979946 eth0  In  IP (tos 0x0, ttl 63, id 14027, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x2639 (correct), ack 2413663046, win 43, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979951 eth0  Out IP (tos 0x0, ttl 64, id 20756, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413666064:2413667070, ack 533474624, win 46, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979961 eth0  Out IP (tos 0x0, ttl 64, id 20757, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413667070:2413668076, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979971 eth0  Out IP (tos 0x0, ttl 64, id 20758, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413668076:2413669082, ack 533474624, win 49, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979982 eth0  Out IP (tos 0x0, ttl 64, id 20759, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413669082:2413670088, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.979990 eth0  In  IP (tos 0x0, ttl 63, id 14028, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1e5e (correct), ack 2413665058, win 42, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979996 eth0  In  IP (tos 0x0, ttl 63, id 14029, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x1683 (correct), ack 2413667070, win 41, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.979996 eth0  Out IP (tos 0x0, ttl 64, id 20760, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413670088:2413671094, ack 533474624, win 50, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980008 eth0  Out IP (tos 0x0, ttl 64, id 20761, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413671094:2413672100, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980015 eth0  In  IP (tos 0x0, ttl 63, id 14030, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x0ea8 (correct), ack 2413669082, win 40, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980018 eth0  Out IP (tos 0x0, ttl 64, id 20762, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413672100:2413673106, ack 533474624, win 52, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980029 eth0  Out IP (tos 0x0, ttl 64, id 20763, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413673106:2413674112, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980029 eth0  In  IP (tos 0x0, ttl 63, id 14031, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0x06cd (correct), ack 2413671094, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980040 eth0  Out IP (tos 0x0, ttl 64, id 20764, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413674112:2413675118, ack 533474624, win 55, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980052 eth0  Out IP (tos 0x0, ttl 64, id 20765, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413675118:2413676124, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980053 eth0  In  IP (tos 0x0, ttl 63, id 14032, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xfef0 (correct), ack 2413673106, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980062 eth0  Out IP (tos 0x0, ttl 64, id 20766, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413676124:2413677130, ack 533474624, win 56, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980076 eth0  Out IP (tos 0x0, ttl 64, id 20767, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413677130:2413678136, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980077 eth0  In  IP (tos 0x0, ttl 63, id 14033, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xf714 (correct), ack 2413675118, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980090 eth0  Out IP (tos 0x0, ttl 64, id 20768, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413678136:2413679142, ack 533474624, win 58, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980098 eth0  In  IP (tos 0x0, ttl 63, id 14034, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xef38 (correct), ack 2413677130, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980105 eth0  Out IP (tos 0x0, ttl 64, id 20769, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413679142:2413680148, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980116 eth0  Out IP (tos 0x0, ttl 64, id 20770, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413680148:2413681154, ack 533474624, win 59, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980127 eth0  Out IP (tos 0x0, ttl 64, id 20771, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413681154:2413682160, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980134 eth0  In  IP (tos 0x0, ttl 63, id 14035, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xe75c (correct), ack 2413679142, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980139 eth0  Out IP (tos 0x0, ttl 64, id 20772, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413682160:2413683166, ack 533474624, win 62, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980150 eth0  Out IP (tos 0x0, ttl 64, id 20773, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413683166:2413684172, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980160 eth0  In  IP (tos 0x0, ttl 63, id 14036, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xdf80 (correct), ack 2413681154, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980161 eth0  Out IP (tos 0x0, ttl 64, id 20774, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413684172:2413685178, ack 533474624, win 64, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980172 eth0  Out IP (tos 0x0, ttl 64, id 20775, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413685178:2413686184, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980178 eth0  In  IP (tos 0x0, ttl 63, id 14037, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xd7a4 (correct), ack 2413683166, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980182 eth0  Out IP (tos 0x0, ttl 64, id 20776, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413686184:2413687190, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980198 eth0  Out IP (tos 0x0, ttl 64, id 20777, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413687190:2413688196, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980204 eth0  In  IP (tos 0x0, ttl 63, id 14038, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xcfc8 (correct), ack 2413685178, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980209 eth0  Out IP (tos 0x0, ttl 64, id 20778, offset 0, flags [DF], proto TCP (6), length 1058)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [P.], seq 2413688196:2413689202, ack 533474624, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 1006
1793456396.980220 eth0  In  IP (tos 0x0, ttl 63, id 14039, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc7ec (correct), ack 2413687190, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980244 eth0  In  IP (tos 0x0, ttl 63, id 14040, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc010 (correct), ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980434 eth0  In  IP (tos 0x0, ttl 63, id 14041, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [F.], cksum 0xc00f (correct), seq 533474624, ack 2413689202, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0
1793456396.980511 eth0  Out IP (tos 0x0, ttl 64, id 20779, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.10.9446 > 192.0.3.20.37850: Flags [F.], cksum 0x8645 (incorrect -> 0xbff2), seq 2413689202, ack 533474625, win 67, options [nop,nop,TS val 2782928896 ecr 2675588078], length 0
1793456396.980568 eth0  In  IP (tos 0x0, ttl 63, id 14042, offset 0, flags [DF], proto TCP (6), length 52)
    192.0.3.20.37850 > 192.0.3.10.9446: Flags [.], cksum 0xc00e (correct), ack 2413689203, win 39, options [nop,nop,TS val 2675588078 ecr 2782928896], length 0

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

- matrix1280_capture_2_receipt: {"bytes":"19775","sha256":"a451be5455ef3acebce773931a17488d0d79c8eae3c8ad9d0b9a8d3545efc2b9"}
- matrix1280_origin_endpoint_samples: {"at":"2026-10-31T14:19:35.728093876Z","local_port":9446,"remote_port":37946,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":97,"sacked":0,"snd_cwnd":33,"snd_mss":1228,"snd_wnd":68608,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:19:42.865265238Z","local_port":9446,"remote_port":37962,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":24972,"retrans":0,"rto_us":202000,"rtt_us":1396,"sacked":0,"snd_cwnd":71,"snd_mss":1228,"snd_wnd":60416,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:19:49.950099398Z","local_port":9446,"remote_port":35902,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":58,"sacked":0,"snd_cwnd":21,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}
{"at":"2026-10-31T14:19:56.98045206Z","local_port":9446,"remote_port":37850,"read_bytes":64108,"write_bytes":64387,"complete_lines":64,"partial_line_bytes":0,"error":"eof","tcp_info":{"bytes_acked":64387,"bytes_received":64109,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":14480,"retrans":0,"rto_us":201000,"rtt_us":59,"sacked":0,"snd_cwnd":25,"snd_mss":1228,"snd_wnd":39936,"state":8,"total_retrans":0,"unacked":0}}

- topology: The measured bottleneck is the member's eth0/eth1 MTU. The any-interface pcaps include both underlays and the tunnel, node tunnel/uplink, and controlled origin. No separate bottleneck router exists; no host networking touched.
- D123: IPv4 overlay over the observed native underlay. IPv6Block no-leak is independently proven in docs/poc/G2-f802-mtu1280-2026-10-31.md; no IPv6 overlay success is claimed or required.
