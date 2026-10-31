# publish

Publish the enrolled exit

Status: blocked

Run: fa0074cc-3b1a-403d-b8c2-3a3439fe2b7e

Source: 491a307dd505e33b14dc0e993aa7e86b28e82f68

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbijbpf8k4c0008rh8t0", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T00:27:42.469995034Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a18f5a-700a-7c96-8188-5bb197462d8e"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 202
- operation_id: 22ecf90f-847a-481d-a58c-8fcd45cb32b7
- operation_status: running
- operation_kind: node.publish
- step_agent_policy: pending NODE_NOT_SERVICEABLE member_forwarding_unconfirmed
- step_data_path_probe: pending
- step_netbird_route_readback: pending
- node_after: 200
- control_status: online
- safety_status: failed
- data_health: healthy
- provisioning_state: applying
- lifecycle_state: registered
- publication_intent: published
- publication_observed: absent
- config_desired: 3
- config_applied: 3
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: failed
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: failed
- check_release_supported: passed
- check_source_binding: failed
- check_storage: passed
- check_tunnel_probe: passed
- desired_publication_intent: published
- desired_generation: 3
- member_rate_policies: 0
- device_bindings: 0
- forward_generation: 0
- member_accept_rules: 0
- note: publication did not reach succeeded within the poll bound
- owner: w3-wire-nodes: publication operation and production peer projection
