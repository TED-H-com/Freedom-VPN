# update-bad

Tampered artifact fails closed

Status: pass

Spec: §13.4, §19 AT25, §19 AT27

- batch: 202
- fleet_generation: 2
- target_count: 1
- desired_contains_release: true
- exit_code: 1
- locally_succeeded: false
- rolled_back: false
- HASH_MISMATCH: true
- recovery_required: true
- ISOLATION_BASELINE_MISSING: false
- artifact url address: false
- health_json: {"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"}
- error_code: ROLLBACK_FAILED
- result_last_error: -
- journal_excerpt: stage=recovery_required error_code=ROLLBACK_FAILED error_stage=rollback error_detail=ROLLBACK_FAILED | health_json {"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"} | outbox state=failed status=pending last_error= | fvpn-updater: updater: HASH_MISMATCH: content changed during resume
- note: tampered bytes were rejected

