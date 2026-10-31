# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: 04de29d3-e56b-4a32-b678-21462845ddb6

Source: fb5b8fcee72bd71b3bd0a4536af9049df4055096

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- kernel_boot_unchanged: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":6,"rules_sha256":"66f554f1229e8c110a6a79c860e2ddf8d42fd835fd7423a91038fe456f29785b","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 5249
- status_stage: enrolled
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 39727578918102
wall_unix_ns 1793456190062547103
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
