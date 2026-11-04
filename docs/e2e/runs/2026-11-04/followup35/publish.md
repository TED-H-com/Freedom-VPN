# publish

Publish the enrolled exit

Status: pass

Run: 8a436a87-5efe-4c18-a7ec-72be170cb69b

Source: 9fbff94c84b0d29c2d78f420e76726f7c9a706ad

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dblfre78k4c0008c2310", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T09:41:45.203352248Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 23}
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
- node_write: {"started_at":"2026-11-04T09:41:45.398255738Z","finished_at":"2026-11-04T09:41:45.40546785Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a5ee-fed7-781a-9ba1-00398dc11b55","if_match":"\"47\"","idempotency_key":"74671246-4430-48f0-9af7-5fd13891a7e8","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T09:41:45.398264977Z","finished_at":"2026-11-04T09:41:45.405461628Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a5ee-b293-739d-aaf8-56c747a0acda"],"reason":"e2e acceptance publish"}; If-Match="48"
- node_write: {"started_at":"2026-11-04T09:41:45.408830817Z","finished_at":"2026-11-04T09:41:45.414409261Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a5ee-fed7-781a-9ba1-00398dc11b55/publication","if_match":"\"48\"","idempotency_key":"3773b4a9-46e4-4295-a4e1-eccb6116ca46","body_sha256":"64983ff2668807e968be8722a29142e20f903caa8b1812655369818ff5c94012","started_at":"2026-11-04T09:41:45.408840807Z","finished_at":"2026-11-04T09:41:45.414404715Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: d9d32bbb-48c3-4cb2-90e6-180216e6f1b8
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
- timed_section: publish
