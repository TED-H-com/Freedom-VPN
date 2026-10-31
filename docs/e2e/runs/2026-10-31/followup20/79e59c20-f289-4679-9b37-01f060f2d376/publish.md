# publish

Publish the enrolled exit

Status: pass

Run: 79e59c20-f289-4679-9b37-01f060f2d376

Source: c9d45bddc871db0485c86dbebe6c08d123243ab0

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": "dbj155n8k4c000fhccp0", "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-31T16:08:58.955206556Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- request: {"publication_intent":"published","group_ids":["01a192b8-7143-7740-8741-312ec635936d"],"reason":"e2e acceptance publish"}; If-Match="8"
- put: 202
- operation_id: ff5456f0-d82a-4ce0-a827-32f4aa41bd6c
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
