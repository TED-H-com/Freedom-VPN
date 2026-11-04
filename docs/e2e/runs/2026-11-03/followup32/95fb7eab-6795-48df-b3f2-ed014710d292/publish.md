# publish

Publish the enrolled exit

Status: pass

Run: 95fb7eab-6795-48df-b3f2-ed014710d292

Source: 59cae103757567bb32feb6db71cc8bb89dbb5d6c

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbl6uf78k4c000f1c7u0", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:33:20.533774469Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- test_only_stale_etag: capacity first attempt only
- node_write: {"started_at":"2026-11-03T23:33:22.147199018Z","finished_at":"2026-11-03T23:33:22.163596505Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648","if_match":"\"0\"","idempotency_key":"ffcf1ca2-e4c3-4a49-91f6-b204e0bfa651","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-03T23:33:22.147207301Z","finished_at":"2026-11-03T23:33:22.152607852Z","status":412,"error_code":"RESOURCE_VERSION_CONFLICT","transport_error":false,"reread":{"method":"GET","path":"/api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648","started_at":"2026-11-03T23:33:22.152628584Z","finished_at":"2026-11-03T23:33:22.156070614Z","status":200,"error_code":"","etag":"\"7\"","transport_error":false,"already_intended":false}},{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648","if_match":"\"7\"","idempotency_key":"ffcf1ca2-e4c3-4a49-91f6-b204e0bfa651","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-03T23:33:22.156129663Z","finished_at":"2026-11-03T23:33:22.163591889Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- test_only_stale_etag: publication first attempt only
- request: {"publication_intent":"published","group_ids":["01a1a3c2-5e88-7260-b390-36546cd5e14b"],"reason":"e2e acceptance publish"}; If-Match="0"
- node_write: {"started_at":"2026-11-03T23:33:22.16636968Z","finished_at":"2026-11-03T23:33:22.180291506Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648/publication","if_match":"\"0\"","idempotency_key":"adb7e726-009a-4446-87f6-b7ed11165d0f","body_sha256":"c4d9d611f9b20457ff0d19484498e9fcba5523b131a7000523cdf7105d417f01","started_at":"2026-11-03T23:33:22.16637406Z","finished_at":"2026-11-03T23:33:22.170268652Z","status":412,"error_code":"RESOURCE_VERSION_CONFLICT","transport_error":false,"reread":{"method":"GET","path":"/api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648","started_at":"2026-11-03T23:33:22.170271301Z","finished_at":"2026-11-03T23:33:22.174119545Z","status":200,"error_code":"","etag":"\"8\"","transport_error":false,"already_intended":false}},{"method":"PUT","path":"/api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648/publication","if_match":"\"8\"","idempotency_key":"adb7e726-009a-4446-87f6-b7ed11165d0f","body_sha256":"c4d9d611f9b20457ff0d19484498e9fcba5523b131a7000523cdf7105d417f01","started_at":"2026-11-03T23:33:22.17416509Z","finished_at":"2026-11-03T23:33:22.18028777Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 2e780f6d-bf98-4aab-a38c-1a74d631bbed
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
