# update-bad

Tampered artifact fails closed

Status: pass

Run: d6b01926-de6f-47e4-803b-ae3ea5c603df

Source: f438edffc2b4e0df4b1a15bd4d7e47f0ca546037

Spec: §13.4, §19 AT25, §19 AT27

- batch: 202
- fleet_generation: 3
- target_count: 1
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
