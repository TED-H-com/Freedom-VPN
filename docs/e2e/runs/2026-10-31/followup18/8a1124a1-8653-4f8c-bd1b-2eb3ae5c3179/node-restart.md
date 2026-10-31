# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: 8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179

Source: a0089b8386c749100d9bd19b8002dd4d666d065c

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- kernel_boot_unchanged: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":6,"rules_sha256":"73652507367e68c9b5c0779fce0f4a09646379cc4b010ee4fa437c21b6441f40","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 5255
- status_stage: enrolled
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 27425463868679
wall_unix_ns 1793443887947497917
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
