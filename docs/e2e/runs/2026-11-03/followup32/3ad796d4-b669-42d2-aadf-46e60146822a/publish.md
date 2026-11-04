# publish

Publish the enrolled exit

Status: pass

Run: 3ad796d4-b669-42d2-aadf-46e60146822a

Source: 59cae103757567bb32feb6db71cc8bb89dbb5d6c

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbl6vff8k4c000f76ccg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:35:29.561314061Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- node_write: {"started_at":"2026-11-03T23:35:30.880262702Z","finished_at":"2026-11-03T23:35:30.977918665Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a3c4-627d-792d-9ed9-60ff88a28373","if_match":"\"7\"","idempotency_key":"0e3d8b25-8ac9-443e-9c43-bb4ecdbea20d","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-03T23:35:30.880272031Z","finished_at":"2026-11-03T23:35:30.977893599Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a3c4-5534-7dde-9388-c90497af0b50"],"reason":"e2e acceptance publish"}; If-Match="8"
- node_write: {"started_at":"2026-11-03T23:35:30.981291253Z","finished_at":"2026-11-03T23:35:30.988472198Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a3c4-627d-792d-9ed9-60ff88a28373/publication","if_match":"\"8\"","idempotency_key":"86824d24-2f79-405b-a76a-792fc7d39234","body_sha256":"9a7b8df32f6b3c6f90a4a41b5f1e8917e35d6d6607e3182f99fb402fb3e2d8b8","started_at":"2026-11-03T23:35:30.981296205Z","finished_at":"2026-11-03T23:35:30.988468327Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: eabf4f33-c2c6-42f9-9832-1b885f3e050c
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
