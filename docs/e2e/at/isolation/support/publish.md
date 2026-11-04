# publish

Publish the enrolled exit

Status: pass

Run: 328ab9e0-7ff4-495a-81c1-410ae8a7d876

Source: 89accd8da1bc3cf44093d3cc0061624f3d603002

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T10:31:18.108690099Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- node_write: {"started_at":"2026-11-04T10:31:18.227728853Z","finished_at":"2026-11-04T10:31:18.234834769Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a61c-e996-7746-895d-1ae813c5020e","if_match":"\"6\"","idempotency_key":"a5fb5af8-4d83-4076-a9e6-fbff81a39880","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T10:31:18.227736354Z","finished_at":"2026-11-04T10:31:18.234817043Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a61c-e969-791a-b5c5-dfa97f61ad8f"],"reason":"e2e acceptance publish"}; If-Match="7"
- node_write: {"started_at":"2026-11-04T10:31:18.238539244Z","finished_at":"2026-11-04T10:31:18.246173117Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a61c-e996-7746-895d-1ae813c5020e/publication","if_match":"\"7\"","idempotency_key":"dbd7ea40-b913-4d97-a13d-58f29f22eee6","body_sha256":"c672d329c61bc6d93aec7b5babe8ba1e2b3816c6eacae540097626a7da08a072","started_at":"2026-11-04T10:31:18.238543261Z","finished_at":"2026-11-04T10:31:18.246167225Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 7a37bc71-f120-40f1-9f0b-98241f681f14
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
