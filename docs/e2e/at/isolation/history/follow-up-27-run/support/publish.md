# publish

Publish the enrolled exit

Status: pass

Run: d69f5652-aed6-4fdd-9e7c-edface751237

Source: 37b3c25f60194e0d2261cf9aca30dfe8fc5d93be

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-05T10:13:48.5252679Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 3}
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
- node_write: {"started_at":"2026-11-05T10:13:48.791288099Z","finished_at":"2026-11-05T10:13:48.797563161Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1ab33-4234-7bd4-a7ad-685f191a3e45","if_match":"\"7\"","idempotency_key":"28398ba1-0b51-4d26-b452-fb277169bb93","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-05T10:13:48.791296871Z","finished_at":"2026-11-05T10:13:48.797546078Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1ab33-4202-7287-b0f9-89c6b151a9c1"],"reason":"e2e acceptance publish"}; If-Match="8"
- node_write: {"started_at":"2026-11-05T10:13:48.801071207Z","finished_at":"2026-11-05T10:13:48.806715177Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1ab33-4234-7bd4-a7ad-685f191a3e45/publication","if_match":"\"8\"","idempotency_key":"31475f74-35f2-4656-8d87-c2496ef5517c","body_sha256":"d9d627281765ca3d05ba1a9c472c953dbc2acd1e9a2c5772d8957e282e8186af","started_at":"2026-11-05T10:13:48.801077102Z","finished_at":"2026-11-05T10:13:48.806709702Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 166fd9c9-5919-4020-85e8-079ff441910a
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
