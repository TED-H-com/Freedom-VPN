# publish

Publish the enrolled exit

Status: pass

Run: 2454c833-ae93-48e0-96b0-00dee7cc3117

Source: 114d31df188dfa71b43c5884588e8f8f9ea958b2

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbis7pf8k4c000fupfog", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T10:33:13.499818795Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a19182-fb6c-7799-b024-5b9521904d6c"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 202
- operation_id: 4e7f6e9c-4550-407f-bbb6-4427158c3e6c
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
- check_lease: passed
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: passed
- check_release_supported: passed
- check_source_binding: passed
- check_storage: passed
- check_tunnel_probe: passed
- desired_publication_intent: published
- desired_generation: 3
- member_rate_policies: 1
- device_bindings: 1
- forward_generation: 3
- member_accept_rules: 6
- note: node.publish succeeded and publication_observed is active
