# publish

Publish the enrolled exit

Status: pass

Run: 4e59a2c4-f510-4e63-a6b9-ed4537e5fe35

Source: c75ad1957773af8f9b9e71036be357eeb491f943

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbiv5kn8k4c000fhcdbg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T13:53:26.485965503Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 4}
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
- request: {"publication_intent":"published","group_ids":["01a1923c-59bd-7d35-a034-ada1a6f51f07"],"reason":"e2e acceptance publish"}; If-Match="10"
- put: 202
- operation_id: 4c519420-a50d-4acd-91c6-1547eb8a20eb
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
