# publish

Publish the enrolled exit

Status: pass

Run: b839eaa7-d437-42f8-9532-d8d74075291b

Source: e0a84fb288db42a7ea0c45ff9a8de097f94734f7

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbicuqv8k4c0009g4jv0", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:10:30.560382301Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a18dca-2474-7adc-a13f-f98ba50eb932"],"reason":"e2e acceptance publish"}; If-Match="6"
- put: 202
- operation_id: 1070766d-c405-490f-9527-eec599f93e7d
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
