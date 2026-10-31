# native-underlay

Real member/node IPv4, IPv6 and dual underlay selectors

Status: pass

Run: cce1a688-6af5-483c-bd67-2d167c51c04b

Source: 98a4b4607cef592a4b00f79aa623c15682cc6ef4

Spec: §8.3, AT09, AT11

- v4_reachable: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- v6_reachable: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- ipv4_input: {"input":{"underlay":"ipv4","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv4_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- ipv4_member_bulk: {"authority_serial":0,"endpoint":{"at":"2026-10-31T16:26:21.627754473Z","local_port":58918,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":202000,"rtt_us":1190,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":5}
- ipv4_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- ipv6_input: {"input":{"underlay":"ipv6","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv6_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- ipv6_member_bulk: {"authority_serial":0,"endpoint":{"at":"2026-10-31T16:26:28.347808977Z","local_port":36418,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":43186,"retrans":0,"rto_us":5156000,"rtt_us":707745,"sacked":0,"snd_cwnd":49,"snd_mss":1228,"snd_wnd":67584,"state":1,"total_retrans":2,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":2063}
- ipv6_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- dual_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- dual_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- dual_member_bulk: {"authority_serial":0,"endpoint":{"at":"2026-10-31T16:26:33.007054427Z","local_port":36432,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":414,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":65536,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":5}
- dual_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- scope: peer UDP family availability is selected; TCP control retained; NetBird's chosen path is observed, not prescribed; IPv6 overlay and platform leak prevention are unclaimed
