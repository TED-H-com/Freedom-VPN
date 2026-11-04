# publish

Publish the enrolled exit

Status: pass

Run: 7b0db5f2-eab5-4534-824a-b54781eed3fd

Source: 0b95e69ae89c68e0fcfbf1fbbfe6774b2e1948db

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T06:03:40.546246972Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: ready
- lifecycle_state: registered
- publication_intent: unpublished
- publication_observed: absent
- config_desired: 1
- config_applied: 1
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: not_run
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: not_run
- check_release_supported: passed
- check_source_binding: not_run
- check_storage: passed
- check_tunnel_probe: passed
- node_write: {"started_at":"2026-11-04T06:03:40.959755893Z","finished_at":"2026-11-04T06:03:40.966509644Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a527-e4a0-7bd8-aeab-c050db6c950c","if_match":"\"7\"","idempotency_key":"1caed602-381f-4ae3-a58d-121c9f9c7322","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T06:03:40.959763798Z","finished_at":"2026-11-04T06:03:40.966495793Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a527-e46e-7685-b32b-291d33e0ebb4"],"reason":"e2e acceptance publish"}; If-Match="8"
- node_write: {"started_at":"2026-11-04T06:03:40.969244516Z","finished_at":"2026-11-04T06:03:40.974587982Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a527-e4a0-7bd8-aeab-c050db6c950c/publication","if_match":"\"8\"","idempotency_key":"afe90671-ffb7-4b18-a262-eb195dc56456","body_sha256":"16beb1f15ad6500f7d5594a06ed55ca5478c5e8b689fe1f307a7e2b7bb94117d","started_at":"2026-11-04T06:03:40.969249711Z","finished_at":"2026-11-04T06:03:40.97457412Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: b6d95f84-7ff1-4897-9937-659e84a6d2c8
- operation_status: succeeded
- operation_kind: node.publish
- step_agent_policy: succeeded
- step_data_path_probe: succeeded
- step_netbird_route_readback: succeeded
- node_after: 200
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: ready
- lifecycle_state: registered
- publication_intent: published
- publication_observed: active
- config_desired: 3
- config_applied: 3
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: not_run
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: not_run
- check_release_supported: passed
- check_source_binding: not_run
- check_storage: passed
- check_tunnel_probe: passed
- desired_publication_intent: published
- desired_generation: 3
- member_rate_policies: 0
- device_bindings: 0
- forward_generation: 3
- member_accept_rules: 0
- note: node.publish succeeded and publication_observed is active
