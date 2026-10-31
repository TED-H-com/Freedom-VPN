# native-matrix-inputs

Declared protocol, MTU and real-platform inputs

Status: fail

Run: e609d048-73fc-4af6-a758-7312224937bc

Source: f771bdb79c0d42b7a3d47090fb52b0d79489d82b

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
- mtu_1500_bulk: {"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":2}
- mtu_1500_counters_before: {"upload_bytes":"201637","download_bytes":"200735","Present":true}
- mtu_1500_counters_after: {"upload_bytes":"268561","download_bytes":"266950","Present":true}
- mtu_1280_bulk: {"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":10010,"milliseconds":12005}
- mtu_1280_counters_before: {"upload_bytes":"268561","download_bytes":"266950","Present":true}
- mtu_1280_counters_after: {"upload_bytes":"334333","download_bytes":"317209","Present":true}
- scope: selector contract plus bounded TCP transfer at two MTUs; QUIC and non-Linux platforms are explicitly unavailable, not acceptance passes; ICMP black-hole and physical second NAT remain operator inputs
