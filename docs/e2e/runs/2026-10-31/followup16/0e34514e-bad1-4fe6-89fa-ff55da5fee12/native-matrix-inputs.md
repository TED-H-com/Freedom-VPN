# native-matrix-inputs

Declared protocol, MTU and real-platform inputs

Status: blocked

Run: 0e34514e-bad1-4fe6-89fa-ff55da5fee12

Source: 59903b57c41115f02ccfb0b2bc1d56dc095b5a39

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
