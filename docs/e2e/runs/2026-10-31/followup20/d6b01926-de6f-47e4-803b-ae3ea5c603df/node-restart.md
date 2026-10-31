# node-restart

Enrolled node resumes without an installer token

Status: pass

Run: d6b01926-de6f-47e4-803b-ae3ea5c603df

Source: f438edffc2b4e0df4b1a15bd4d7e47f0ca546037

Spec: §5, §6, F2-05

- installer_token_absent_before: true
- resume_within_watchdog_deadline: true
- nft_table_readable: true
- new_observation_same_incarnation: true
- kernel_boot_unchanged: true
- resolver_after_restart: {"admin_http":200,"query_log_enabled":false,"query_log_file_enabled":false,"rule_count":6,"rules_sha256":"5f3a6ebb984772d42bc5df16e45cfce3577112984c58e3408b068ae075520e07","statistics_enabled":false,"synthetic_answer_count":0,"synthetic_blocked":true}
- restart_elapsed_ms: 5276
- status_stage: enrolled
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 47225966887689
wall_unix_ns 1793463688450516701
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 14
reason recovery_required
- same_node_and_incarnation: true
- enrollment_rows_unchanged: true
- installer_token_absent_after: true
