# publish

Publish the enrolled exit

Status: pass

Run: c52b760c-c289-4220-a722-6c5a059819b9

Source: fff680c6c2cd539b77cc534ce9d090f8ad9a8ec3

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbl6jmv8k4c000e2ktqg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:10:24.29338113Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- node_write: {"started_at":"2026-11-03T23:10:25.871197046Z","finished_at":"2026-11-03T23:10:25.880511176Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a3ad-6aed-727b-a0f5-038b59bacde8","if_match":"\"7\"","idempotency_key":"48eabcc3-e5a7-4530-9637-3b4520f02f8c","started_at":"2026-11-03T23:10:25.871207588Z","finished_at":"2026-11-03T23:10:25.880489194Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a3ad-5a1c-7b25-966e-336d7404ba05"],"reason":"e2e acceptance publish"}; If-Match="8"
- node_write: {"started_at":"2026-11-03T23:10:25.883540387Z","finished_at":"2026-11-03T23:10:25.889830652Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a3ad-6aed-727b-a0f5-038b59bacde8/publication","if_match":"\"8\"","idempotency_key":"2dd6949c-317b-4c1d-bb01-3367c136a982","started_at":"2026-11-03T23:10:25.88354675Z","finished_at":"2026-11-03T23:10:25.889825276Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 04ec055f-e59c-4d28-9ee3-d17a8fd1a70c
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
