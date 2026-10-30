# publish

Publish the enrolled exit

Status: blocked

Run: 5de03570-0436-477d-9847-6f6984432bd8

Source: a7a712aab8a67fcc33514a9ac76cb877a90c8143

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbi0jv78k4c000fu06bg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}, {"name": "tunnel_probe", "status": "passed", "error_code": null, "observed_at": "2026-10-30T03:08:05.184550749Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- operation_id: a3570157-7c4d-42e5-ac77-5524c3df565d
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
