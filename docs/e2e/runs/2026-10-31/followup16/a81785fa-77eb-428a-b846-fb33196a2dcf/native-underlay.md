# native-underlay

Real member/node IPv4, IPv6 and dual underlay selectors

Status: pass

Run: a81785fa-77eb-428a-b846-fb33196a2dcf

Source: ec84d71f45ce3622b11a8a7e55f67c9c1c432d29

Spec: §8.3, AT09, AT11

- v4_reachable: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- v6_reachable: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":0,"milliseconds":0}
- ipv4_input: {"input":{"underlay":"ipv4","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv4_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- ipv4_member_bulk: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1}
- ipv4_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv4":2}}
- ipv6_input: {"input":{"underlay":"ipv6","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- ipv6_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- ipv6_member_bulk: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1}
- ipv6_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- dual_input: {"input":{"underlay":"dual","mtu":1500,"protocol":"tcp","platform":"linux"},"available":true,"reason":""}
- dual_transport: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- dual_member_bulk: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":true,"bytes":64064,"milliseconds":1}
- dual_transport_after_bulk: {"path":{"Relayed":0,"Direct":1,"Forced":false},"candidate_families":{"ipv6":2}}
- scope: peer UDP family availability is selected; TCP control retained; NetBird's chosen path is observed, not prescribed; IPv6 overlay and platform leak prevention are unclaimed
