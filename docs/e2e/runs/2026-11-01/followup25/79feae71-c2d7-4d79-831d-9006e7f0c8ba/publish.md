# publish

Publish the enrolled exit

Status: pass

Run: 79feae71-c2d7-4d79-831d-9006e7f0c8ba

Source: 6b075f9904315b890c5510f75889a0ef099028b7

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbjhl2v8k4c000c8sb10", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:55:11.223642674Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a196bf-87d6-76ab-ab6b-7ee8d7ed7f29"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 202
- operation_id: b9a63569-a0c4-45c5-b991-899e22ff43e9
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
