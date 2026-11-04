# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- kernel_boot_unchanged: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":16,"rules_sha256":"063ab3b25eeabfc9f9f7d4ac571f4063202686575ecc166f606d83c21c845b22","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 1997
- status_stage: enrolled
- heartbeat: v1
boot_id 56e68667-9243-4933-83af-c70d575ff5fe
mono_ns 126325430344823
wall_unix_ns 1793802882978968633
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
- timed_section: node-restart
