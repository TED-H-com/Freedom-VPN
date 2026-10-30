# publish

Publish the enrolled exit

Status: blocked

Spec: §5.2, §14.2

- control_status: online
- safety_status: failed
- data_health: unknown
- provisioning_state: applying
- lifecycle_state: registered
- publication_intent: unpublished
- publication_observed: absent
- config_desired: 1
- config_applied: 0
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: failed
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: failed
- check_release_supported: failed
- check_source_binding: failed
- check_storage: passed
- check_tunnel_probe: not_run
- put: 422 NODE_NOT_SERVICEABLE
- put_error: NODE_NOT_SERVICEABLE node does not satisfy serving baseline requirements {"failed_checks":["provisioning_state=applying (want ready)","safety_status=failed"]}

