# publish

Publish the enrolled exit

Status: pass

Run: af4c130b-5029-401b-bd0f-e8afaad3d37f

Source: ab0899e6799a4d992c87215c5a6705e7e2fa506f

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dblc9t78k4c0009klj70", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T05:39:36.805973876Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 23}
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
- node_write: {"started_at":"2026-11-04T05:39:37.316862528Z","finished_at":"2026-11-04T05:39:37.325233469Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a511-4190-721c-b8cf-b5151ee3e45e","if_match":"\"48\"","idempotency_key":"d319263a-28cc-4b11-bd9a-b4d10974dbe0","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T05:39:37.316872296Z","finished_at":"2026-11-04T05:39:37.325228445Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a511-1510-765f-aa8d-2b3129ad5abe"],"reason":"e2e acceptance publish"}; If-Match="49"
- node_write: {"started_at":"2026-11-04T05:39:37.329773498Z","finished_at":"2026-11-04T05:39:37.339004005Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a511-4190-721c-b8cf-b5151ee3e45e/publication","if_match":"\"49\"","idempotency_key":"ce03444f-9ee3-4fef-8503-ee9b3810a1c7","body_sha256":"f1b0a5fa7796e35c0c7903b6ec38bb405516528b237fa375c3b0ede41c32baa0","started_at":"2026-11-04T05:39:37.329778887Z","finished_at":"2026-11-04T05:39:37.338999353Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 0b54de5c-7442-4ebc-ae0c-640c1120cf4f
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
