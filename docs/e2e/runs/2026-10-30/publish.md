# publish

Publish the enrolled exit

Status: blocked

Run: 5852f09a-30e8-45bc-8983-b5f212e33640

Source: 5eb716e1a36ced17b8013646c9282c92746c81da

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbhvi0n8k4c000b7i5eg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}, {"name": "tunnel_probe", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:55:39.288463922Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- put: 202
- operation_id: efb6c4aa-f32a-454b-ae06-78d7f9c2d612
- operation_status: running
- operation_kind: node.publish
- step_agent_policy: succeeded
- step_data_path_probe: succeeded
- step_netbird_route_readback: failed DEPENDENCY_UNAVAILABLE Step failed; quote the operation ID.
- node_after: 200
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: ready
- lifecycle_state: registered
- publication_intent: published
- publication_observed: error
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
- forward_generation: 0
- member_accept_rules: 0
- note: publication did not succeed: agent_policy NODE_NOT_SERVICEABLE Step failed; quote the operation ID.
- owner: w3-wire-nodes: publication operation and production peer projection

