# publish

Publish the enrolled exit

Status: blocked

Run: f1a17b4e-c671-4186-9a6e-b5b7f0033d37

Source: 6e3667f57a55c2757644b2840d4a6763435f5009

Spec: §5.2, §14.2

- agent_baseline_observed: false
- stored_rows: {"node": {"peer_id": "dbiq0nn8k4c000cbromg", "data_health": "healthy", "safety_status": "failed", "provisioning_state": "applying", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "detail": "Resolver policy was not applied.", "status": "failed", "error_code": "VALIDATION_FAILED", "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T08:03:04.19501779Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "failed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 14}
- control_status: online
- safety_status: failed
- data_health: healthy
- provisioning_state: applying
- lifecycle_state: registered
- publication_intent: unpublished
- publication_observed: absent
- config_desired: 1
- config_applied: 1
- check_dns_policy: failed
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
- request: {"publication_intent":"published","group_ids":["01a190fa-4477-7ded-ab14-685c7264af20"],"reason":"e2e acceptance publish"}; If-Match="19"
- put: 422 NODE_NOT_SERVICEABLE
- put_error: NODE_NOT_SERVICEABLE node does not satisfy serving baseline requirements {"failed_checks":["provisioning_state=applying (want ready)","safety_status=failed"]}
- owner: w3-wire-nodes: publication operation and production peer projection
