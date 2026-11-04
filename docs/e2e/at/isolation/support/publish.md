# publish

Publish the enrolled exit

Status: pass

Run: 7599576e-339c-49c9-8bf5-0f710818cbda

Source: fe2c68ecbf1ef1481d744c998be95710d28be954

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T02:53:03.888823899Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- node_write: {"started_at":"2026-11-04T02:53:04.094113089Z","finished_at":"2026-11-04T02:53:04.102555727Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a479-618c-71dc-a4c9-5d8bb7a3b60e","if_match":"\"7\"","idempotency_key":"3df9abd0-be4d-4245-9267-9135b63579d1","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T02:53:04.09412103Z","finished_at":"2026-11-04T02:53:04.102532515Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a479-6155-7249-9bf8-16a0585bbb63"],"reason":"e2e acceptance publish"}; If-Match="8"
- node_write: {"started_at":"2026-11-04T02:53:04.106010971Z","finished_at":"2026-11-04T02:53:04.113206061Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a479-618c-71dc-a4c9-5d8bb7a3b60e/publication","if_match":"\"8\"","idempotency_key":"8ace4708-87d1-4468-bdc3-300e44c1a656","body_sha256":"e787a00a73d473b53af02bf2e08ee00f881a014cadfa4e1562ef551be9f2b57e","started_at":"2026-11-04T02:53:04.106018859Z","finished_at":"2026-11-04T02:53:04.113189574Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 12574b2f-ce6f-4969-a789-4386bb162902
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
