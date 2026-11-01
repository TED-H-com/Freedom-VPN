# publish

Publish the enrolled exit

Status: pass

Run: aedd33ae-ca2f-4c14-a118-9bd818f03fc6

Source: 6b075f9904315b890c5510f75889a0ef099028b7

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbjhehn8k4c000f6tjp0", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-01T10:41:14.396981968Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a196b2-c20c-793a-87e4-b18e636b21aa"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 202
- operation_id: cc4f8aa0-cc77-4a42-8063-f44b4475dfb4
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
