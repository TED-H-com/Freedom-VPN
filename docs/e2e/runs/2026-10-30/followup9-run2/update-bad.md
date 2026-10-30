# update-bad

Tampered artifact fails closed

Status: pass

Run: 65712989-0b05-4117-9a50-689499cd31be

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

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
