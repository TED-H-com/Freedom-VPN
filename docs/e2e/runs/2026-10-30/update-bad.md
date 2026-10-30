# update-bad

Tampered artifact fails closed

Status: pass

Run: 3a129919-af3f-4bc9-b54a-8075eae8af45

Source: 79259af1facc1faa798fbb83cc20aa4e74539d1f

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
