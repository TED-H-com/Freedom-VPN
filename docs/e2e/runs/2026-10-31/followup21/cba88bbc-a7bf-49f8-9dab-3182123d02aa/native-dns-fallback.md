# native-dns-fallback

System resolver fallback, both-down failure and recovery

Status: pass

Run: cba88bbc-a7bf-49f8-9dab-3182123d02aa

Source: 5bc416660913b06cabfab680a24b32c9b541fed2

Spec: §12.2, AT21

- healthy: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2}
- primary_down_at: "2026-10-31T19:42:05.424968573Z"
- automatic_secondary: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2504}
- secondary_block_policy: {"authority_serial":0,"dns_trace":{"trace_streams":{"authority":["f889f8bc-ec08-4fc2-8b11-78f1909335a3"],"secondary":["64157ebe-7c77-4ea0-87c1-6e472c5e9572"]},"started":"2026-10-31T19:42:08.091864236Z","completed":"2026-10-31T19:42:10.637085463Z","responder":"secondary","cache_state":"unknown","trace_complete":false,"resolver_events":[{"request_id":"2a35b25e-d772-45e7-80e5-c1e63042291f","stream_id":"64157ebe-7c77-4ea0-87c1-6e472c5e9572","sequence":"6","phase":"completed","started":"2026-10-31T19:42:10.631743454Z","completed":"2026-10-31T19:42:10.632673404Z","role":"secondary","source_role":"native","fixture_case":"feed-01","rr_type":1,"protocol":"udp","rcode":3,"ttls":[],"address_classes":[],"authority_serial":100500,"success":true}],"authority_events":[],"authority_history":[],"authority_outstanding_at_query_start":[],"authority_outstanding_at_query_completion":[],"authority_outstanding_at_trace_read":[],"trace_read_at":{"authority":"2026-10-31T19:42:10.72983231Z","primary":"2026-10-31T19:42:10.64903914Z","secondary":"2026-10-31T19:42:10.689927194Z"},"authority_serial_after_query":1001,"authority_read_at":"2026-10-31T19:42:10.75228434Z","answer_ttl_remaining":[]},"authority_ttl":5,"has_soa":true,"available":true,"rcode":3,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":2502}
- both_down_fail_closed: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":false,"rcode":-1,"types":[],"ttls":[],"address_classes":[],"expected_answer":false,"connected":false,"bytes":0,"milliseconds":12001,"error":"lookup_unavailable"}
- recovery: {"authority_serial":0,"authority_ttl":0,"has_soa":false,"available":true,"rcode":-1,"types":[],"ttls":[],"address_classes":["other_ipv6","other_ipv4"],"expected_answer":true,"connected":false,"bytes":0,"milliseconds":2}
- recovery_ms: 40
- expected: primary unavailable -> same-policy secondary answers; both unavailable -> resolution fails; recovery answers without public fallback
