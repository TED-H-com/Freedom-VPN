# update-good

Good release reaches a terminal updater state

Status: blocked

Spec: §13, §19 AT23, §19 AT25

- batch: 202
- fleet_generation: 1
- target_count: 1
- desired_contains_release: true
- exit_code: 2
- locally_succeeded: false
- rolled_back: false
- HASH_MISMATCH: false
- recovery_required: true
- ISOLATION_BASELINE_MISSING: false
- artifact url address: false
- health_json: {"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"}
- error_code: ROLLBACK_FAILED
- result_last_error: -
- journal_excerpt: stage=recovery_required error_code=ROLLBACK_FAILED error_stage=rollback error_detail=ROLLBACK_FAILED | health_json {"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"}
- note: journal health_json {"dns_policy":"failed","identity_and_state":"passed","internet_only":"failed","processes":"passed","tunnel_probe":"failed"}; error_code ROLLBACK_FAILED; result last_error -. Stock NodeChecks Isolation, DNSPolicy and TunnelProbe return failed with no lab probe. locally_succeeded was not reached. ROLLBACK_FAILED is the rollback decision when no last-good release can be restored

