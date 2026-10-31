# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: be58af04-a09f-41c5-b369-05be49de8c0d

Source: 03839e59d5694b0bacd695c2b1ce23d5fc34ff53

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"b41cd722654313b510451934b2f59fd90f12ac3996dc44312ecfb43a157d3ae8","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 6387
- status_stage: enrolled
- heartbeat: v1
boot_id 9d370541-1281-45e6-8a24-5f6c0627d4ad
mono_ns 2301160461562
wall_unix_ns 1793418763644090988
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
