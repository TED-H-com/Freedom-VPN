# publish

Publish the enrolled exit

Status: blocked

Run: 8fcb95bc-a3e2-4d4e-a6c9-a86a024ee9a7

Source: e0a84fb288db42a7ea0c45ff9a8de097f94734f7

Spec: §5.2, §14.2

- agent_baseline_observed: false
- stored_rows: {"node": {"peer_id": "dbict178k4c000fegfqg", "data_health": "healthy", "safety_status": "failed", "provisioning_state": "applying", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "detail": "Resolver policy was not applied.", "status": "failed", "error_code": "VALIDATION_FAILED", "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T17:07:42.022269085Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "failed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 4}
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
- request: {"publication_intent":"published","group_ids":["01a18dc6-997e-7e24-b99e-7919efeb44a4"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 422 NODE_NOT_SERVICEABLE
- put_error: NODE_NOT_SERVICEABLE node does not satisfy serving baseline requirements {"failed_checks":["provisioning_state=applying (want ready)","safety_status=failed"]}
- owner: w3-wire-nodes: publication operation and production peer projection
