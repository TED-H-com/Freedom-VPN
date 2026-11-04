# publish

Publish the enrolled exit

Status: pass

Run: c52bea6c-cd91-4086-a779-656b01277dcc

Source: 037f44187aaac256c82682de01af2ef61bbb90f0

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-04T00:06:46.455532986Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 3}
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
- node_write: {"started_at":"2026-11-04T00:06:46.545489178Z","finished_at":"2026-11-04T00:06:46.55141341Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PATCH","path":"/api/v1/admin/nodes/01a1a3e1-26ff-7e1d-a31d-368b83fa2c4c","if_match":"\"7\"","idempotency_key":"1dc18c75-ae52-4792-af19-95982c752c98","body_sha256":"4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d","started_at":"2026-11-04T00:06:46.545496514Z","finished_at":"2026-11-04T00:06:46.551385535Z","status":202,"error_code":"","transport_error":false}]}
- capacity_patch: 202
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- request: {"publication_intent":"published","group_ids":["01a1a3e1-26d3-708d-a05f-4f4d492c4b73"],"reason":"e2e acceptance publish"}; If-Match="8"
- node_write: {"started_at":"2026-11-04T00:06:46.553872257Z","finished_at":"2026-11-04T00:06:46.559664515Z","max_attempts":3,"limit_seconds":10,"status":"pass","outcome":"accepted","attempts":[{"method":"PUT","path":"/api/v1/admin/nodes/01a1a3e1-26ff-7e1d-a31d-368b83fa2c4c/publication","if_match":"\"8\"","idempotency_key":"fd1f76a1-08df-4e90-a2da-8f2ef427927b","body_sha256":"23807bc8cf9ca8ae87f647e70d8b98cfb08c3807e564155d26b4bf1d1974e050","started_at":"2026-11-04T00:06:46.553877445Z","finished_at":"2026-11-04T00:06:46.55964165Z","status":202,"error_code":"","transport_error":false}]}
- put: 202
- operation_id: 0b2b06d9-d654-4836-acfa-0c64981f537c
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
