# native-dns-fallback

System resolver fallback, both-down failure and recovery

Status: pass

Run: 99027e5d-ae5e-40d4-b595-7b550f89a52d

Source: b27bb361ae1fee1c42579235b794cb09efe4ed3a

Spec: §12.2, AT21

- healthy: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2}
- primary_down_at: "2026-10-31T19:28:17.29852061Z"
- automatic_secondary: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2503}
- secondary_block_policy: {"authority_serial":0,"dns_trace":{"started":"2026-10-31T19:28:19.96724553Z","completed":"2026-10-31T19:28:22.510714874Z","responder":"secondary","cache_state":"unknown","trace_complete":false,"resolver_events":[{"request_id":"89d118a1-781d-4557-86fe-190706708e3a","stream_id":"1252ba94-1a4b-4169-bfb3-beec97b080a3","sequence":"6","phase":"completed","started":"2026-10-31T19:28:22.504467035Z","completed":"2026-10-31T19:28:22.505567617Z","role":"secondary","source_role":"native","fixture_case":"feed-01","rr_type":1,"protocol":"udp","rcode":3,"ttls":[],"address_classes":[],"authority_serial":100500,"success":true}],"authority_events":[],"authority_history":[],"authority_outstanding_at_query_start":[],"authority_outstanding_at_query_completion":[],"authority_outstanding_at_trace_read":[],"trace_read_at":{"authority":"2026-10-31T19:28:22.602416929Z","primary":"2026-10-31T19:28:22.522152654Z","secondary":"2026-10-31T19:28:22.561569849Z"},"authority_serial_after_query":1001,"authority_read_at":"2026-10-31T19:28:22.625758808Z","answer_ttl_remaining":[]},"authority_ttl":5,"has_soa":true,"available":true,"rcode":3,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":2503}
- both_down_fail_closed: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":12001,"error":"lookup_unavailable"}
- recovery: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2}
- recovery_ms: 40
- expected: primary unavailable -> same-policy secondary answers; both unavailable -> resolution fails; recovery answers without public fallback
