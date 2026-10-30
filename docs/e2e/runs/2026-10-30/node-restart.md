# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: 14231a83-130d-40fb-a9b4-0d3851520b67

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"b41cd722654313b510451934b2f59fd90f12ac3996dc44312ecfb43a157d3ae8","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 6343
- status_stage: enrolled
- heartbeat: v1
boot_id 2798153d-6853-4098-81bc-57c9ae011ee4
mono_ns 711788955965341
wall_unix_ns 1793356749759144121
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
