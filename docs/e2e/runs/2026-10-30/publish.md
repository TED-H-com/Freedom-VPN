# publish

Publish the enrolled exit

Status: pass

Run: 76f0725c-2264-4021-a553-60f6c921e32b

Source: 1b2f358880d38ae643865edc2bb6f19756748456

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbid72f8k4c000992l50", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:28:38.626888397Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 4}
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
- request: {"publication_intent":"published","group_ids":["01a18dda-0ee2-7e2e-9ecb-eec6bf210c1c"],"reason":"e2e acceptance publish"}; If-Match="9"
- put: 202
- operation_id: fa59eae1-69ac-4621-9f31-1c6c7a3b5cb9
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
