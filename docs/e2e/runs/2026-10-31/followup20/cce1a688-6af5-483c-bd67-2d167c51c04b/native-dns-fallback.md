# native-dns-fallback

System resolver fallback, both-down failure and recovery

Status: pass

Run: cce1a688-6af5-483c-bd67-2d167c51c04b

Source: 98a4b4607cef592a4b00f79aa623c15682cc6ef4

Spec: §12.2, AT21

- healthy: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":5003}
- primary_down_at: "2026-10-31T16:26:58.011749705Z"
- automatic_secondary: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2505}
- secondary_block_policy: {"authority_serial":0,"dns_trace":{"started":"2026-10-31T16:27:00.786569225Z","completed":"2026-10-31T16:27:03.331567102Z","responder":"secondary","cache_state":"unknown","trace_complete":false,"resolver_events":[{"started":"2026-10-31T16:27:03.327041977Z","completed":"2026-10-31T16:27:03.327518544Z","role":"secondary","source_role":"native","fixture_case":"feed-01","rr_type":1,"protocol":"udp","rcode":3,"ttls":[],"address_classes":[],"authority_serial":100500,"success":true}],"authority_events":[],"authority_serial_after_query":1001,"authority_read_at":"2026-10-31T16:27:03.45077687Z","answer_ttl_remaining":[]},"authority_ttl":5,"has_soa":true,"available":true,"rcode":3,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":2502}
- both_down_fail_closed: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":12001,"error":"lookup_unavailable"}
- recovery: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":1}
- recovery_ms: 33
- expected: primary unavailable -> same-policy secondary answers; both unavailable -> resolution fails; recovery answers without public fallback
