# publish

Publish the enrolled exit

Status: blocked

Run: 9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04

Source: c248db966d684d58b4781349f71f8394fcb64d9b

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbkukuv8k4c0009m91h0", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-03T14:06:52.779929087Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 3}
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
- request: {"publication_intent":"published","group_ids":["01a1a1bb-e729-7046-95ff-e28bde5b696c"],"reason":"e2e acceptance publish"}; If-Match="10"
- put: 412 RESOURCE_VERSION_CONFLICT
- put_error: RESOURCE_VERSION_CONFLICT If-Match version does not match current resource_version {"current_resource_version":"11"}
- owner: w3-wire-nodes: publication operation and production peer projection
