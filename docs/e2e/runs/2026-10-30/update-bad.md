# update-bad

Tampered artifact fails closed

Status: pass

Run: 6bf270e4-abb1-43ff-93eb-18d282fc7c2a

Source: 5477de091f449047193fdc9114cfca4ced5b36e6

Spec: §13.4, §19 AT25, §19 AT27

- batch: 202
- fleet_generation: 2
- target_count: 1
- desired_contains_release: true
- exit_code: 1
- locally_succeeded: false
- rolled_back: false
- HASH_MISMATCH: true
- recovery_required: false
- ISOLATION_BASELINE_MISSING: false
- artifact url address: false
- health_json: -
- error_code: -
- result_last_error: -
- journal_excerpt: fvpn-updater: updater: HASH_MISMATCH: content changed during resume
- note: tampered bytes were rejected

