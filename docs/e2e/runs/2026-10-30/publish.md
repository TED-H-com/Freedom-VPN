# publish

Publish the enrolled exit

Status: blocked

Run: 742a2b14-355f-462e-be89-52ffb2c1993b

Source: 05d70d63890bb181937dabfe01b9e9a680d76086

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbi2v978k4c0009qq7vg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:49:21.457400897Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 4}
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
- request: {"publication_intent":"published","group_ids":["01a18b59-c3af-7a2a-8403-ad3cdf63da80"],"reason":"e2e acceptance publish"}; If-Match="10"
- put: 202
- operation_id: f631853d-6a1a-47ab-95bf-eff4e70dacc1
- operation_status: running
- operation_kind: node.publish
- step_agent_policy: succeeded
- step_data_path_probe: failed NODE_NOT_SERVICEABLE member_forwarding_unconfirmed
- step_netbird_route_readback: succeeded
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
- forward_generation: 3
- member_accept_rules: 0
- note: publication did not succeed: agent_policy NODE_NOT_SERVICEABLE Step failed; quote the operation ID.
- owner: w3-wire-nodes: publication operation and production peer projection
