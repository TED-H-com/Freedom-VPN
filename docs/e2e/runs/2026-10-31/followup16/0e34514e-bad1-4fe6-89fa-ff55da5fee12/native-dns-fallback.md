# native-dns-fallback

System resolver fallback, both-down failure and recovery

Status: pass

Run: 0e34514e-bad1-4fe6-89fa-ff55da5fee12

Source: 59903b57c41115f02ccfb0b2bc1d56dc095b5a39

Spec: §12.2, AT21

- healthy: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2}
- primary_down_at: "2026-10-31T06:30:22.82308423Z"
- automatic_secondary: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2503}
- secondary_block_policy: {"authority_ttl":5,"has_soa":true,"available":true,"rcode":3,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":2502}
- both_down_fail_closed: {"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":12001,"error":"lookup_unavailable"}
- recovery: {"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":1}
- recovery_ms: 31
- expected: primary unavailable -> same-policy secondary answers; both unavailable -> resolution fails; recovery answers without public fallback
