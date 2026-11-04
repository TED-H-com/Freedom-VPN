# update-good

Good release reaches a terminal updater state

Status: blocked

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §13, §19 AT23, §19 AT25

- batch: 202
- fleet_generation: 2
- target_count: 3
- desired_contains_release: true
- probe_not_member: true
- probe_route_selected: true
- probe_config_scope: one dedicated setup-key peer /32; one isolated origin /32 UDP 4443; no member device or grant
- running_candidate_executable: false
- probe_agent_config: /etc/vpn-node/agent-run.json
- updater_log_readable: true
- probe_omission_count: 0
- invalid_probe_peer: false
- exit_code: 3
- locally_succeeded: false
- rolled_back: false
- HASH_MISMATCH: false
- recovery_required: false
- ISOLATION_BASELINE_MISSING: false
- artifact url address: false
- health_json: -
- health_attestation: -
- error_code: -
- result_last_error: -
- probe_traffic: {"attempts":431,"authenticated_udp_replies":0}
- probe_window: {"samples":78,"grant_samples":0,"source_elements":0,"destination_elements":0,"max_timeout_seconds":0,"probe_up_bytes":0,"probe_down_bytes":0,"member_bytes_before":35029,"member_bytes_after":35029,"elements_after":0}
- probe_origin_receipt: {"probe_id":"fa24d2f0-9e6e-48e8-aa24-feebec741ba7","requests":0,"source_address":""}
- probe_traversal_verified: false
- configured_probe_issued: false
- probe_gate: real candidate result does not prove §13.9; see observed grant and traversal counters; owner w2-agent-core / w1-netlab
- owner: w2-agent-core / w1-netlab: challenge exception and observed probe traversal
