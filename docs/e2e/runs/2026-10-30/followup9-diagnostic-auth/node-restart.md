# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: 376e5f84-535e-4ddd-902b-985852d4691b

Source: 724ab19fd46a974a66d138ac0cddcd7f803bafb7

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"b41cd722654313b510451934b2f59fd90f12ac3996dc44312ecfb43a157d3ae8","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 6358
- status_stage: enrolled
- heartbeat: v1
boot_id cd4a18ce-0aec-4d41-aa94-40873659b462
mono_ns 697972970918596
wall_unix_ns 1793342933774097894
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
