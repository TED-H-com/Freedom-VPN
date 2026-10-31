# node-restart

Enrolled node resumes without an installer token

Status: fail

Run: f1a17b4e-c671-4186-9a6e-b5b7f0033d37

Source: 6e3667f57a55c2757644b2840d4a6763435f5009

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: false
- nft_table_readable: true
- new_observation_same_incarnation: false
- kernel_boot_unchanged: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":0,"rules_sha256":"4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945","statistics_enabled":false,"synthetic_answer_count":2,"synthetic_blocked":false}
- restart_elapsed_ms: 35205
- status_stage: enrolled
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 17486834381976
wall_unix_ns 1793433949318011692
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason not_published
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
