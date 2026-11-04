# update-bad

Tampered artifact fails closed

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §13.4, §19 AT25, §19 AT27

- batch: 202
- fleet_generation: 3
- target_count: 3
- desired_contains_release: true
- running_candidate_executable: false
- exit_code: 1
- locally_succeeded: false
- rolled_back: false
- HASH_MISMATCH: true
- recovery_required: false
- ISOLATION_BASELINE_MISSING: false
- artifact url address: false
- health_json: -
- health_attestation: -
- error_code: -
- result_last_error: -
- journal_excerpt: fvpn-updater: updater: HASH_MISMATCH: artifact
- note: tampered bytes were rejected
- updater_invocations: [{"invocation_id":"invocation.YQEMZXKV","started_at":"2026-11-04T14:45:49.043793385Z","ended_at":"2026-11-04T14:45:59.988545964Z","pid":186378,"exit_code":0,"output":"{\"time\":\"2026-11-04T14:45:49.495505375Z\",\"level\":\"INFO\",\"msg\":\"update lock wait\",\"command\":\"run\",\"waited_ms\":446,\"acquired\":true,\"budget_ms\":10000}\n"},{"invocation_id":"invocation.hHQ2yCIu","started_at":"2026-11-04T14:46:07.995705033Z","ended_at":"2026-11-04T14:46:11.002748759Z","pid":195558,"exit_code":1,"output":"fvpn-updater: updater: HASH_MISMATCH: artifact\n"}]
- timed_section: update-bad
