# update-bad

Tampered artifact fails closed

Status: pass

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

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
- updater_invocations: [{"invocation_id":"invocation.uoern8lE","started_at":"2026-11-04T09:52:28.264063978Z","ended_at":"2026-11-04T09:52:37.470593496Z","pid":178784,"exit_code":0,"output":""},{"invocation_id":"invocation.yIuYeknH","started_at":"2026-11-04T09:52:43.478106458Z","ended_at":"2026-11-04T09:52:46.494609725Z","pid":186533,"exit_code":1,"output":"fvpn-updater: updater: HASH_MISMATCH: artifact\n"}]
- timed_section: update-bad
