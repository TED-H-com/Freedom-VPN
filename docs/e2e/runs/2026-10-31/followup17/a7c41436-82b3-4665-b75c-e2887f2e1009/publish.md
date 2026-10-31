# publish

Publish the enrolled exit

Status: blocked

Run: a7c41436-82b3-4665-b75c-e2887f2e1009

Source: d37720e9062b3ff4528ec0213d38c10d1b6309f3

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbiq5av8k4c000fv14mg", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:11:27.117257993Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 4}
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
- request: {"publication_intent":"published","group_ids":["01a19103-41a6-7027-9c2b-057192633643"],"reason":"e2e acceptance publish"}; If-Match="10"
- put: 202
- operation_id: b5b8f9ac-ff6b-468f-843b-17cdac557310
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
- config_applied: 1
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
