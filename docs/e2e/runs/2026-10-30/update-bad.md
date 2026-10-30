# update-bad

Tampered artifact fails closed

Status: pass

Run: 5852f09a-30e8-45bc-8983-b5f212e33640

Source: 5eb716e1a36ced17b8013646c9282c92746c81da

Spec: §13.4, §19 AT25, §19 AT27

- batch: 202
- fleet_generation: 2
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

