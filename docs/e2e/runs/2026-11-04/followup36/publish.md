# publish

Publish the enrolled exit

Status: pass

Run: c17589fa-7e48-4dee-b574-7e29aca1effa

Source: f6edbda9909268106f7a8e08c26aab8254ae1481

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dblk4pv8k4c000fsne9g", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T14:34:47.965986969Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 23}
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
- node_write: {"started_at":"2026-11-04T14:34:48.160153346Z","finished_at":"2026-11-04T14:34:48.16814967Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a6fb-5e05-741c-ac02-cd000d4036be","if_match":"\"48\"","idempotency_key":"cfe68bd5-403d-4aee-9428-8498cbe701ee","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T14:34:48.160161019Z","finished_at":"2026-11-04T14:34:48.168140313Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a6fb-2ff8-7cea-b010-cef860080c64"],"reason":"e2e acceptance publish"}; If-Match="49"
- node_write: {"started_at":"2026-11-04T14:34:48.171447122Z","finished_at":"2026-11-04T14:34:48.178701308Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a6fb-5e05-741c-ac02-cd000d4036be/publication","if_match":"\"49\"","idempotency_key":"b09ac93c-ff05-466c-9497-fe5fa915c594","body_sha256":"acd9bdc1a4e5d4b75a3c7ddebd9dc7364210b0119d924b3bf8481f980628ebf5","started_at":"2026-11-04T14:34:48.171451135Z","finished_at":"2026-11-04T14:34:48.178696659Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 2ce86f27-6c62-40fb-8c46-9f45a078f382
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
