# publish

Publish the enrolled exit

Status: blocked

Run: 6bf270e4-abb1-43ff-93eb-18d282fc7c2a

Source: 5477de091f449047193fdc9114cfca4ced5b36e6

Spec: §5.2, §14.2

- agent_baseline_observed: true
- stored_rows: {"node": {"peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "applying", "applied_config_generation": "1", "desired_config_generation": "1"}, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "lease", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "qos", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "source_binding", "status": "not_run", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}, {"name": "tunnel_probe", "status": "passed", "error_code": null, "observed_at": "2026-10-30T01:11:40.318735193Z"}], "peer_id": null, "data_health": "healthy", "safety_status": "passed", "provisioning_state": "ready", "applied_config_generation": "1"}, "peer_observe_events": 0}
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: applying
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
- put: 422 NODE_NOT_SERVICEABLE
- put_error: NODE_NOT_SERVICEABLE node does not satisfy serving baseline requirements {"failed_checks":["provisioning_state=applying (want ready)"]}

