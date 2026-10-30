# update-bad

Tampered artifact fails closed

Status: pass

Run: fa5650f3-d238-496e-b7d4-718ce705fefb

Source: a36bb84f1dafc7526c021177026b4d558e273cd0

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
