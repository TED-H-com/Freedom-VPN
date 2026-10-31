# native-underlay

Real member/node IPv4, IPv6 and dual underlay selectors

Status: pass

Run: 5282829c-c11e-4f34-954a-5bc2a5bceca5

Source: 5dc018aca482183cb5ca26889bbba54f8db036bc

Spec: §8.3, AT09, AT11

- v4_reachable: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- v6_reachable: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- ipv4_input: {"input":{"underlay":"ipv4","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv4_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- ipv4_member_bulk: {"endpoint":{"at":"2026-10-31T14:23:49.872648687Z","local_port":56336,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":578,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":4}
- ipv4_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- ipv6_input: {"input":{"underlay":"ipv6","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv6_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- ipv6_member_bulk: {"endpoint":{"at":"2026-10-31T14:23:54.53974287Z","local_port":56348,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1228,"rcv_space":12280,"retrans":0,"rto_us":202000,"rtt_us":1049,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":7}
- ipv6_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- dual_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- dual_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- dual_member_bulk: {"endpoint":{"at":"2026-10-31T14:23:59.16989359Z","local_port":47366,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":115,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":68608,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1}
- dual_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- scope: peer UDP family availability is selected; TCP control retained; NetBird's chosen path is observed, not prescribed; IPv6 overlay and platform leak prevention are unclaimed
