# publish

Publish the enrolled exit

Status: pass

Run: df9dbad7-049d-4eaa-ab98-5490906cc3e9

Source: 56d814113913ddc6177d47d4a5160a65c93add25

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:07:38.122107531Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- node_write: {"started_at":"2026-11-04T14:07:38.307349754Z","finished_at":"2026-11-04T14:07:38.314153243Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a6e2-f90b-7629-ba41-ef37246f9bcd","if_match":"\"7\"","idempotency_key":"0c743403-92ac-4080-8dd1-9fe5eb1d2b45","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T14:07:38.307354336Z","finished_at":"2026-11-04T14:07:38.314130152Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a6e2-f8da-72a6-a652-2c06225bc9e5"],"reason":"e2e acceptance publish"}; If-Match="8"
- node_write: {"started_at":"2026-11-04T14:07:38.31725917Z","finished_at":"2026-11-04T14:07:38.326163799Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a6e2-f90b-7629-ba41-ef37246f9bcd/publication","if_match":"\"8\"","idempotency_key":"74ea7877-b48a-4249-8ef9-fa44b207e7c7","body_sha256":"494d9077fddddd25f0efe682e907c585cac6b549a1aba5318d70522ff9c2c0ad","started_at":"2026-11-04T14:07:38.317264426Z","finished_at":"2026-11-04T14:07:38.326156657Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: e0cff9df-b3de-4234-982f-9da8ef879821
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
