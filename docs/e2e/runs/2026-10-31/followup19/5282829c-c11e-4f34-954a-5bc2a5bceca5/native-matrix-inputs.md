# native-matrix-inputs

Declared protocol, MTU and real-platform inputs

Status: fail

Run: 5282829c-c11e-4f34-954a-5bc2a5bceca5

Source: 5dc018aca482183cb5ca26889bbba54f8db036bc

Spec: §8.3, AT09, AT11

- linux_tcp: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- linux_udp: {"input":{"underlay":"dual","mtu":1280,"protocol":"udp","platform":"linux"},"available":true,"reason":""}
- linux_quic: {"input":{"underlay":"dual","mtu":1280,"protocol":"quic","platform":"linux"},"available":false,"reason":"requires a pinned QUIC client/server at the controlled origin; UDP echo is not QUIC"}
- android_tcp: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"android"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- android_udp: {"input":{"underlay":"dual","mtu":1280,"protocol":"udp","platform":"android"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- android_quic: {"input":{"underlay":"dual","mtu":1280,"protocol":"quic","platform":"android"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- ios_tcp: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"ios"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- ios_udp: {"input":{"underlay":"dual","mtu":1280,"protocol":"udp","platform":"ios"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- ios_quic: {"input":{"underlay":"dual","mtu":1280,"protocol":"quic","platform":"ios"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- windows_tcp: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"windows"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- windows_udp: {"input":{"underlay":"dual","mtu":1280,"protocol":"udp","platform":"windows"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- windows_quic: {"input":{"underlay":"dual","mtu":1280,"protocol":"quic","platform":"windows"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- macos_tcp: {"input":{"underlay":"dual","mtu":1280,"protocol":"tcp","platform":"macos"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- macos_udp: {"input":{"underlay":"dual","mtu":1280,"protocol":"udp","platform":"macos"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- macos_quic: {"input":{"underlay":"dual","mtu":1280,"protocol":"quic","platform":"macos"},"available":false,"reason":"requires the named real platform and operator client; no emulation by the Linux container"}
- mtu_1500_bulk: {"endpoint":{"at":"2026-10-31T14:23:59.452240463Z","local_port":47376,"remote_port":9446,"read_bytes":64387,"write_bytes":64108,"complete_lines":64,"partial_line_bytes":0,"error":"none","tcp_info":{"bytes_acked":64109,"bytes_received":64387,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":124,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":66560,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1}
- mtu_1500_counters_before: {"upload_bytes":"200897","download_bytes":"202972","Present":true}
- mtu_1500_counters_after: {"upload_bytes":"267977","download_bytes":"271111","Present":true}
- mtu_1280_bulk: {"endpoint":{"at":"2026-10-31T14:24:11.73542441Z","local_port":47388,"remote_port":9446,"read_bytes":12075,"write_bytes":64108,"complete_lines":12,"partial_line_bytes":0,"error":"timeout","tcp_info":{"bytes_acked":64109,"bytes_received":12075,"lost":0,"notsent_bytes":0,"rcv_mss":1006,"rcv_space":12280,"retrans":0,"rto_us":201000,"rtt_us":162,"sacked":0,"snd_cwnd":63,"snd_mss":1228,"snd_wnd":36864,"state":1,"total_retrans":0,"unacked":0}},"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":12012,"milliseconds":12011,"error":"bulk_echo_incomplete"}
- mtu_1280_counters_before: {"upload_bytes":"267977","download_bytes":"271111","Present":true}
- mtu_1280_counters_after: {"upload_bytes":"334017","download_bytes":"321084","Present":true}
- scope: selector contract plus bounded TCP transfer at two MTUs; QUIC and non-Linux platforms are explicitly unavailable, not acceptance passes; ICMP black-hole and physical second NAT remain operator inputs
