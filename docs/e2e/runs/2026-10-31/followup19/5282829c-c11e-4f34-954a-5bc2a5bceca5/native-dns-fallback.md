# native-dns-fallback

System resolver fallback, both-down failure and recovery

Status: pass

Run: 5282829c-c11e-4f34-954a-5bc2a5bceca5

Source: 5dc018aca482183cb5ca26889bbba54f8db036bc

Spec: §12.2, AT21

- healthy: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2}
- primary_down_at: "2026-10-31T14:24:13.710480998Z"
- automatic_secondary: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":7502}
- secondary_block_policy: {"authority_ttl":5,"has_soa":true,"available":true,"rcode":3,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":2502}
- both_down_fail_closed: {"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":12000,"error":"lookup_unavailable"}
- recovery: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":1}
- recovery_ms: 42
- expected: primary unavailable -> same-policy secondary answers; both unavailable -> resolution fails; recovery answers without public fallback
