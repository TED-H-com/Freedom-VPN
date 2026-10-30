# publish

Publish the enrolled exit

Status: pass

Run: 49799a38-1980-4c0f-a527-725646230eee

Source: 4d84f0a651e60ae84f168a8a33248503a50cc94a

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbif6rv8k4c000cs14kg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T19:44:08.558728267Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a18e56-cf95-7273-8664-22b362d36fcb"],"reason":"e2e acceptance publish"}; If-Match="7"
- put: 202
- operation_id: bdfe5521-8df0-4d35-8928-6eb480d38d41
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
