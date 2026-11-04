# publish

Publish the enrolled exit

Status: pass

Run: bd73d7a4-03f1-4b17-97c2-9c25a39ac0db

Source: 4c495c3b1516022e1de8db2f864432a8a8d171c5

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T15:49:48.91307691Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- node_write: {"started_at":"2026-11-04T15:49:49.063331895Z","finished_at":"2026-11-04T15:49:49.069756897Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a740-8552-7eb7-88ce-e72f0d8f6ccd","if_match":"\"6\"","idempotency_key":"28ace60e-c67b-4f86-bfdb-8ba27a152285","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T15:49:49.063339881Z","finished_at":"2026-11-04T15:49:49.069736914Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a740-8523-7eaa-b144-722b55359415"],"reason":"e2e acceptance publish"}; If-Match="7"
- node_write: {"started_at":"2026-11-04T15:49:49.072882661Z","finished_at":"2026-11-04T15:49:49.07968061Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a740-8552-7eb7-88ce-e72f0d8f6ccd/publication","if_match":"\"7\"","idempotency_key":"0d52b8ea-cb73-4670-a645-4d09699d7966","body_sha256":"d0595ec216b7310dd7f97e571bd884af4d4f587ac65c13514b82fac913d67148","started_at":"2026-11-04T15:49:49.072888659Z","finished_at":"2026-11-04T15:49:49.07967613Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: e980c2e9-8f94-41f7-ab33-81b04d9730df
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
