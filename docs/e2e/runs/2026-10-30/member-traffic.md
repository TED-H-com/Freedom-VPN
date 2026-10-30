# member-traffic

Member device and exit traffic

Status: blocked

Run: 742a2b14-355f-462e-be89-52ffb2c1993b

Source: 05d70d63890bb181937dabfe01b9e9a680d76086

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18b59-c3c4-7302-ae26-3fc39d89212e
- plan_revision_id: 01a18b59-c3c4-7604-bb70-1d06a9e5ea12
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T05:53:40Z
- binding_valid_until: 2026-10-30T06:53:22Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- admission_counter: unavailable after 90 seconds
- heartbeat: v1
boot_id c14a667b-d336-4bc4-8119-1c74b6ee174b
mono_ns 694751037991892
wall_unix_ns 1793339711841170746
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 5
reason lease_expired
- persistent_boot_recovery_config_owner_mode: 0:600
- admin_node: 200
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: ready
- lifecycle_state: registered
- publication_intent: published
- publication_observed: error
- config_desired: 5
- config_applied: 5
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: failed
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: failed
- check_release_supported: passed
- check_source_binding: failed
- check_storage: passed
- check_tunnel_probe: passed
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 2, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "lease", "detail": "Serving lease is not in force.", "status": "failed", "error_code": "LEASE_EXPIRED", "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "qos", "detail": "Rate policy could not be applied.", "status": "failed", "error_code": "VALIDATION_FAILED", "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "source_binding", "detail": "No member binding was admitted.", "status": "failed", "error_code": "VALIDATION_FAILED", "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:54:45.887510812Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T05:54:45.887510812Z"}], "forwarding_block_reasons": ["lease_expired"], "member_forwarding_enabled": false}, "latest_lease": {"issued_at": "2026-10-30T05:48:51.111308+00:00", "expires_at": "2026-10-31T05:48:51.111308+00:00", "generation": "1", "member_count": 0}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- traffic: not claimed; no admission counter increment. Inspect lease generation/member count versus desired bindings.
- owner: w2-agent-core: lease refresh after publication/member changes
