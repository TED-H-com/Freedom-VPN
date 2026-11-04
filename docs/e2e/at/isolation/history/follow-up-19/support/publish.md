# publish

Publish the enrolled exit

Status: blocked

Run: 70beb28e-4766-4004-932b-510fd01f986a

Source: 458b7a89f07c22dd4e66f839195b6549db4b883d

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-11-03T22:10:56.673939768Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "exit_group_count": 0, "peer_observe_events": 2}
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
- capacity_patch: 412 RESOURCE_VERSION_CONFLICT
- capacity_input: TEST ONLY 200 Mbps parent / 100 Mbps member caps through PATCH; required by the agent QoS model
- owner: w3-wire-nodes: publication operation and production peer projection
