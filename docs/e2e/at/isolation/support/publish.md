# publish

Publish the enrolled exit

Status: pass

Run: 422cdd6d-090b-4bb7-b92b-d4883259ac87

Source: 11698fc4d7205a5c902592660ccb346d7f4db750

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbid7qn8k4c000ak2250", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:29:38.621618491Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a18ddb-af9e-725a-8a0e-230130c373e2"],"reason":"e2e acceptance publish"}; If-Match="7"
- put: 202
- operation_id: 0acdfafc-786d-4fbc-b3d3-d50980867d16
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
