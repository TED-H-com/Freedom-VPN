# usage

Sampled usage visible with exact bytes

Status: blocked

Spec: §10, §19 AT14, §19 AT15

- member_usage: 200
- batch_status: 200
- replay_status: 200
- replay_results_equal: true
- batch_outcome: rejected VALIDATION_FAILED binding_not_found
- member_usage_after_batch: 200
- member_total_bytes: 0
- exact_bytes: false
- note: member usage is served and the idempotent replay returned the same results; exact byte totals were not claimed

