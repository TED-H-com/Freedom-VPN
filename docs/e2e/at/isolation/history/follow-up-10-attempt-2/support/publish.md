# publish

Publish the enrolled exit

Status: pass

Run: 5e9e4041-d942-4562-9e99-489bf735636a

Source: d9b770e06a3510880236eed57fe229465947951e

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T18:43:17.003302645Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a19345-e674-7884-b7b3-463b4b68347e"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 202
- operation_id: f7b6a874-b25d-4b8a-89af-f8a545501998
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
