# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: a7c41436-82b3-4665-b75c-e2887f2e1009

Source: d37720e9062b3ff4528ec0213d38c10d1b6309f3

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- kernel_boot_unchanged: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":6,"rules_sha256":"be703714f509474685bab8d27b7c5a71b72781127ec7194eb99545afc5ddce67","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 6366
- status_stage: enrolled
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 18206593355703
wall_unix_ns 1793434669076985277
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
