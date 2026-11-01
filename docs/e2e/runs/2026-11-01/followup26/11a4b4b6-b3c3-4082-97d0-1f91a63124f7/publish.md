# publish

Publish the enrolled exit

Status: pass

Run: 11a4b4b6-b3c3-4082-97d0-1f91a63124f7

Source: 1528abfbaaa590038f77cc3f6a8d828b44cf9369

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbjj5h78k4c000fbpcqg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-01T12:38:33.083993544Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a1971e-29c7-7789-9ad0-5fd44df2fec0"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 202
- operation_id: b13f942f-798b-4c0b-b03d-191658f5671a
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
