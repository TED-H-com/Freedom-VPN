# native-underlay

Real member/node IPv4, IPv6 and dual underlay selectors

Status: pass

Run: e609d048-73fc-4af6-a758-7312224937bc

Source: f771bdb79c0d42b7a3d47090fb52b0d79489d82b

Spec: §8.3, AT09, AT11

- v4_reachable: {"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- v6_reachable: {"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- ipv4_input: {"input":{"underlay":"ipv4","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv4_transport: {"path":{"Relayed":1,"Direct":0,"Forced":false},"candidate_families":{}}
- ipv4_member_bulk: {"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":3}
- ipv6_input: {"input":{"underlay":"ipv6","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv6_transport: {"path":{"Relayed":1,"Direct":0,"Forced":false},"candidate_families":{}}
- ipv6_member_bulk: {"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":5}
- dual_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- dual_transport: {"path":{"Relayed":1,"Direct":0,"Forced":false},"candidate_families":{}}
- dual_member_bulk: {"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":4}
- scope: peer UDP family availability is selected; TCP control retained; NetBird's chosen path is observed, not prescribed; IPv6 overlay and platform leak prevention are unclaimed
