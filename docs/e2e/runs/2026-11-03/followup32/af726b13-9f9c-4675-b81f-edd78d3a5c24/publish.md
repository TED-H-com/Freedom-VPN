# publish

Publish the enrolled exit

Status: pass

Run: af726b13-9f9c-4675-b81f-edd78d3a5c24

Source: 419c065cbf93b83b0114f84071fee62981d90d99

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbl6qbf8k4c0009bnih0", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-03T23:24:33.623927896Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- node_write: {"started_at":"2026-11-03T23:24:33.694254126Z","finished_at":"2026-11-03T23:24:33.706599941Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9","if_match":"\"0\"","idempotency_key":"9bc52246-5cca-4dbb-b776-76d515b7bf48","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-03T23:24:33.694262942Z","finished_at":"2026-11-03T23:24:33.697786879Z","status":412,"error_code":"RESOURCE_VERSION_CONFLICT","transport_error":false,"reread":{"method":"GET","path":"/api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9","started_at":"2026-11-03T23:24:33.697806052Z","finished_at":"2026-11-03T23:24:33.70121898Z","status":200,"error_code":"","etag":"\"6\"","transport_error":false,"already_intended":false}},{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9","if_match":"\"6\"","idempotency_key":"9bc52246-5cca-4dbb-b776-76d515b7bf48","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-03T23:24:33.701258287Z","finished_at":"2026-11-03T23:24:33.706586484Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- test_only_stale_etag: publication first attempt only
- request: {"publication_intent":"published","group_ids":["01a1a3ba-518c-7ed9-abe1-6f4792fe242c"],"reason":"e2e acceptance publish"}; If-Match="0"
- node_write: {"started_at":"2026-11-03T23:24:33.709324982Z","finished_at":"2026-11-03T23:24:33.720383976Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9/publication","if_match":"\"0\"","idempotency_key":"d4814143-839b-459d-8cf7-f41407269b70","body_sha256":"960d69301a7fefdc7a1b62122a83becb4abbb837fb284fc26730fbc4d697c534","started_at":"2026-11-03T23:24:33.709328596Z","finished_at":"2026-11-03T23:24:33.712413024Z","status":412,"error_code":"RESOURCE_VERSION_CONFLICT","transport_error":false,"reread":{"method":"GET","path":"/api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9","started_at":"2026-11-03T23:24:33.71241594Z","finished_at":"2026-11-03T23:24:33.715408949Z","status":200,"error_code":"","etag":"\"7\"","transport_error":false,"already_intended":false}},{"method":"PUT","path":"/api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9/publication","if_match":"\"7\"","idempotency_key":"d4814143-839b-459d-8cf7-f41407269b70","body_sha256":"960d69301a7fefdc7a1b62122a83becb4abbb837fb284fc26730fbc4d697c534","started_at":"2026-11-03T23:24:33.715448105Z","finished_at":"2026-11-03T23:24:33.720380291Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: df7375b2-765c-4e9c-895d-6809c3caad59
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
