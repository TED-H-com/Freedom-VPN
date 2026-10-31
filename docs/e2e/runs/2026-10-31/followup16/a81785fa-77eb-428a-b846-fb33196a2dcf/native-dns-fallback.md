# native-dns-fallback

System resolver fallback, both-down failure and recovery

Status: pass

Run: a81785fa-77eb-428a-b846-fb33196a2dcf

Source: ec84d71f45ce3622b11a8a7e55f67c9c1c432d29

Spec: §12.2, AT21

- healthy: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2}
- primary_down_at: "2026-10-31T06:48:30.635223321Z"
- automatic_secondary: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2502}
- secondary_block_policy: {"authority_ttl":5,"has_soa":true,"available":true,"rcode":3,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":2504}
- both_down_fail_closed: {"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":12000,"error":"lookup_unavailable"}
- recovery: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":1}
- recovery_ms: 44
- expected: primary unavailable -> same-policy secondary answers; both unavailable -> resolution fails; recovery answers without public fallback
