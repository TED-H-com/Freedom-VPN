# update-bad

Tampered artifact fails closed

Status: pass

Run: 0a687f3a-2351-4234-b43d-882109267967

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

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
