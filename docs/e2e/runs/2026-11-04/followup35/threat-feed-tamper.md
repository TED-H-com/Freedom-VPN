# threat-feed-tamper

Corrupted resolver response retains node last-good and recovers

Status: pass

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

Spec: §12.5, §19 AT22

- positive_node_delivery: true
- last_good_before: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"063ab3b25eeabfc9f9f7d4ac571f4063202686575ecc166f606d83c21c845b22","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- source_change: approved source adds one synthetic rule; wait for its configured 300-second refresh cadence
- fault_injected: true
- transport_fault: {"corrupted_sha256":"38d61c116b7c02220c1c096cceff13c4c82fddc4dd546ec232ca3093978f0937","document_id":"01a1a5f9-2439-736e-b028-379f510e6ba7","fault_injected":true,"http_status":200,"original_sha256":"5ad06e592cdc0f3448e21ba7d6a88a004cf2c653da67ed7ee7937207ea263ffd","responses":2,"size_bytes":1238}
- rejected_evidence: {"applied_at":null,"content_sha256":"38d61c116b7c02220c1c096cceff13c4c82fddc4dd546ec232ca3093978f0937","document_id":"01a1a5f9-2439-736e-b028-379f510e6ba7","error_code":"HASH_MISMATCH","feeds":[],"generation":"3","outcome":"failed"}
- last_good_during_fault: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"063ab3b25eeabfc9f9f7d4ac571f4063202686575ecc166f606d83c21c845b22","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- last_good_unchanged: true
- recovered_evidence: {"applied_at":"2026-11-04T09:52:20.694346196Z","content_sha256":"5ad06e592cdc0f3448e21ba7d6a88a004cf2c653da67ed7ee7937207ea263ffd","document_id":"01a1a5f9-2439-736e-b028-379f510e6ba7","error_code":null,"feeds":[{"content_sha256":"6fc90b883869a9d03e200a0b014521d0b04c290d2ede553a3c028b53e4e319d5","feed_code":"e2e-feed","rule_count":11}],"generation":"3","outcome":"applied"}
- resolver_after_recovery: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":17,"rules_sha256":"0e0885e01e9f9027c3a206ab0500eb7dfe9631390669cf2db9eeadecb8e387f2","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- timed_section: threat-feed-tamper
